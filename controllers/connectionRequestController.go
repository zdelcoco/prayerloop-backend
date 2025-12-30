package controllers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"
	"github.com/PrayerLoop/services"
	"github.com/doug-martin/goqu/v9"
)

// SearchUserByEmail searches for a user by their email address
func SearchUserByEmail(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)

	email := strings.TrimSpace(c.Query("email"))
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email parameter is required"})
		return
	}

	// Basic email validation
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email format"})
		return
	}

	// Search for user by exact email match
	var user models.UserSearchResult
	found, err := initializers.DB.From("user_profile").
		Select(
			"user_profile_id",
			"first_name",
			"last_name",
			"username",
		).
		Where(
			goqu.And(
				goqu.C("email").Eq(email),
				goqu.C("user_profile_id").Neq(currentUser.User_Profile_ID), // Don't return self
			),
		).
		ScanStruct(&user)

	if err != nil {
		log.Println("Error searching for user:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search for user", "details": err.Error()})
		return
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "No user found with this email address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User found",
		"user":    user,
	})
}

// SendConnectionRequest sends a request to link a prayer subject to a user
func SendConnectionRequest(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)

	var requestData models.ConnectionRequestCreate
	if err := c.BindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify the prayer subject exists and belongs to current user
	var prayerSubject models.PrayerSubject
	found, err := initializers.DB.From("prayer_subject").
		Select("*").
		Where(goqu.C("prayer_subject_id").Eq(requestData.Prayer_Subject_ID)).
		ScanStruct(&prayerSubject)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prayer subject", "details": err.Error()})
		return
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prayer subject not found"})
		return
	}

	if prayerSubject.Created_By != currentUser.User_Profile_ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only send connection requests for your own prayer subjects"})
		return
	}

	// Check if the prayer subject is already linked
	if prayerSubject.User_Profile_ID != nil && prayerSubject.Link_Status == "linked" {
		c.JSON(http.StatusConflict, gin.H{"error": "This prayer subject is already linked to a user"})
		return
	}

	// Verify the target user exists
	var targetUserExists bool
	targetUserExists, err = initializers.DB.From("user_profile").
		Select(goqu.L("1")).
		Where(goqu.C("user_profile_id").Eq(requestData.Target_User_ID)).
		ScanVal(new(int))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify target user", "details": err.Error()})
		return
	}

	if !targetUserExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Target user not found"})
		return
	}

	// Cannot send request to yourself
	if requestData.Target_User_ID == currentUser.User_Profile_ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot send a connection request to yourself"})
		return
	}

	// Check for existing pending request
	var existingCount int
	_, err = initializers.DB.From("prayer_connection_request").
		Select(goqu.COUNT("request_id")).
		Where(
			goqu.C("requester_id").Eq(currentUser.User_Profile_ID),
			goqu.C("target_user_id").Eq(requestData.Target_User_ID),
			goqu.C("prayer_subject_id").Eq(requestData.Prayer_Subject_ID),
			goqu.C("status").Eq("pending"),
		).
		ScanVal(&existingCount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing requests", "details": err.Error()})
		return
	}

	if existingCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "A pending connection request already exists for this prayer subject and user"})
		return
	}

	// Create the connection request
	newRequest := models.ConnectionRequest{
		Requester_ID:      currentUser.User_Profile_ID,
		Target_User_ID:    requestData.Target_User_ID,
		Prayer_Subject_ID: requestData.Prayer_Subject_ID,
		Status:            "pending",
	}

	insert := initializers.DB.Insert("prayer_connection_request").Rows(newRequest).Returning("request_id")

	var insertedID int
	_, err = insert.Executor().ScanVal(&insertedID)
	if err != nil {
		log.Println("Failed to create connection request:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send connection request", "details": err.Error()})
		return
	}

	// Update prayer subject link_status to pending
	_, err = initializers.DB.Update("prayer_subject").
		Set(goqu.Record{
			"user_profile_id": requestData.Target_User_ID,
			"link_status":     "pending",
			"updated_by":      currentUser.User_Profile_ID,
			"datetime_update": time.Now(),
		}).
		Where(goqu.C("prayer_subject_id").Eq(requestData.Prayer_Subject_ID)).
		Executor().Exec()

	if err != nil {
		log.Printf("Warning: Failed to update prayer subject link status: %v", err)
		// Don't fail the request, the connection request was created
	}

	// Send push notification to target user
	go func() {
		pushService := services.GetPushNotificationService()
		if pushService == nil {
			return
		}

		displayName := currentUser.First_Name
		if displayName == "" {
			displayName = currentUser.Username
		}

		payload := services.NotificationPayload{
			Title: "Prayer Connection Request",
			Body:  fmt.Sprintf("%s wants to connect with you for prayer", displayName),
			Data: map[string]string{
				"type":      "connection_request",
				"requestId": strconv.Itoa(insertedID),
			},
		}

		err := pushService.SendNotificationToUsers([]int{requestData.Target_User_ID}, payload)
		if err != nil {
			log.Printf("Failed to send connection request notification: %v", err)
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Connection request sent successfully",
		"requestId": insertedID,
	})
}

// GetIncomingConnectionRequests returns pending connection requests for the current user
func GetIncomingConnectionRequests(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if userID != currentUser.User_Profile_ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this user's connection requests"})
		return
	}

	// Get status filter, default to pending
	status := c.DefaultQuery("status", "pending")
	if status != "pending" && status != "accepted" && status != "declined" && status != "all" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status filter. Must be 'pending', 'accepted', 'declined', or 'all'"})
		return
	}

	query := initializers.DB.From("prayer_connection_request").
		Select(
			goqu.I("prayer_connection_request.request_id"),
			goqu.I("prayer_connection_request.requester_id"),
			goqu.I("prayer_connection_request.target_user_id"),
			goqu.I("prayer_connection_request.prayer_subject_id"),
			goqu.I("prayer_connection_request.status"),
			goqu.I("prayer_connection_request.datetime_create"),
			goqu.I("prayer_connection_request.datetime_responded"),
			goqu.I("user_profile.first_name").As("requester_first_name"),
			goqu.I("user_profile.last_name").As("requester_last_name"),
			goqu.I("user_profile.email").As("requester_email"),
			goqu.I("user_profile.phone_number").As("requester_phone_number"),
		).
		Join(
			goqu.T("user_profile"),
			goqu.On(goqu.Ex{"prayer_connection_request.requester_id": goqu.I("user_profile.user_profile_id")}),
		).
		Where(goqu.C("target_user_id").Eq(userID))

	if status != "all" {
		query = query.Where(goqu.C("status").Table("prayer_connection_request").Eq(status))
	}

	query = query.Order(goqu.I("prayer_connection_request.datetime_create").Desc())

	var requests []models.ConnectionRequestDetail
	err = query.ScanStructs(&requests)

	if err != nil {
		log.Println("Failed to fetch incoming connection requests:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch connection requests", "details": err.Error()})
		return
	}

	if requests == nil {
		requests = []models.ConnectionRequestDetail{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Connection requests retrieved successfully",
		"requests": requests,
	})
}

// GetOutgoingConnectionRequests returns connection requests sent by the current user
func GetOutgoingConnectionRequests(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if userID != currentUser.User_Profile_ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this user's connection requests"})
		return
	}

	// Get status filter, default to all
	status := c.DefaultQuery("status", "all")
	if status != "pending" && status != "accepted" && status != "declined" && status != "all" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status filter. Must be 'pending', 'accepted', 'declined', or 'all'"})
		return
	}

	query := initializers.DB.From("prayer_connection_request").
		Select(
			goqu.I("prayer_connection_request.request_id"),
			goqu.I("prayer_connection_request.requester_id"),
			goqu.I("prayer_connection_request.target_user_id"),
			goqu.I("prayer_connection_request.prayer_subject_id"),
			goqu.I("prayer_connection_request.status"),
			goqu.I("prayer_connection_request.datetime_create"),
			goqu.I("prayer_connection_request.datetime_responded"),
			goqu.I("user_profile.first_name").As("requester_first_name"),
			goqu.I("user_profile.last_name").As("requester_last_name"),
			goqu.I("user_profile.email").As("requester_email"),
			goqu.I("user_profile.phone_number").As("requester_phone_number"),
		).
		Join(
			goqu.T("user_profile"),
			goqu.On(goqu.Ex{"prayer_connection_request.requester_id": goqu.I("user_profile.user_profile_id")}),
		).
		Where(goqu.C("requester_id").Eq(userID))

	if status != "all" {
		query = query.Where(goqu.C("status").Table("prayer_connection_request").Eq(status))
	}

	query = query.Order(goqu.I("prayer_connection_request.datetime_create").Desc())

	var requests []models.ConnectionRequestDetail
	err = query.ScanStructs(&requests)

	if err != nil {
		log.Println("Failed to fetch outgoing connection requests:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch connection requests", "details": err.Error()})
		return
	}

	if requests == nil {
		requests = []models.ConnectionRequestDetail{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Connection requests retrieved successfully",
		"requests": requests,
	})
}

// RespondToConnectionRequest accepts or declines a connection request
func RespondToConnectionRequest(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)

	requestID, err := strconv.Atoi(c.Param("request_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID", "details": err.Error()})
		return
	}

	// Fetch the connection request
	var request models.ConnectionRequest
	found, err := initializers.DB.From("prayer_connection_request").
		Select("*").
		Where(goqu.C("request_id").Eq(requestID)).
		ScanStruct(&request)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch connection request", "details": err.Error()})
		return
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Connection request not found"})
		return
	}

	// Only the target user can respond
	if request.Target_User_ID != currentUser.User_Profile_ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only respond to connection requests sent to you"})
		return
	}

	// Can only respond to pending requests
	if request.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("This request has already been %s", request.Status)})
		return
	}

	var responseData models.ConnectionRequestResponse
	if err := c.BindJSON(&responseData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status
	if responseData.Status != "accepted" && responseData.Status != "declined" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be 'accepted' or 'declined'"})
		return
	}

	now := time.Now()

	// Update the connection request
	_, err = initializers.DB.Update("prayer_connection_request").
		Set(goqu.Record{
			"status":             responseData.Status,
			"datetime_responded": now,
		}).
		Where(goqu.C("request_id").Eq(requestID)).
		Executor().Exec()

	if err != nil {
		log.Println("Failed to update connection request:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to respond to connection request", "details": err.Error()})
		return
	}

	// Update prayer subject based on response
	var linkStatus string
	var userProfileID interface{}
	if responseData.Status == "accepted" {
		linkStatus = "linked"
		userProfileID = currentUser.User_Profile_ID
	} else {
		linkStatus = "declined"
		userProfileID = nil // Clear the pending link
	}

	_, err = initializers.DB.Update("prayer_subject").
		Set(goqu.Record{
			"link_status":     linkStatus,
			"user_profile_id": userProfileID,
			"datetime_update": now,
		}).
		Where(goqu.C("prayer_subject_id").Eq(request.Prayer_Subject_ID)).
		Executor().Exec()

	if err != nil {
		log.Printf("Warning: Failed to update prayer subject link status: %v", err)
		// Don't fail - the request response was recorded
	}

	// Send notification to requester
	go func() {
		pushService := services.GetPushNotificationService()
		if pushService == nil {
			return
		}

		displayName := currentUser.First_Name
		if displayName == "" {
			displayName = currentUser.Username
		}

		var body string
		if responseData.Status == "accepted" {
			body = fmt.Sprintf("%s accepted your prayer connection request", displayName)
		} else {
			body = fmt.Sprintf("%s declined your prayer connection request", displayName)
		}

		payload := services.NotificationPayload{
			Title: "Connection Request Update",
			Body:  body,
			Data: map[string]string{
				"type":      "connection_response",
				"requestId": strconv.Itoa(requestID),
				"status":    responseData.Status,
			},
		}

		err := pushService.SendNotificationToUsers([]int{request.Requester_ID}, payload)
		if err != nil {
			log.Printf("Failed to send connection response notification: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Connection request %s successfully", responseData.Status),
		"status":  responseData.Status,
	})
}

// RemovePrayerSubjectLink removes the link between a prayer subject and a user
func RemovePrayerSubjectLink(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	subjectID, err := strconv.Atoi(c.Param("prayer_subject_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prayer subject ID", "details": err.Error()})
		return
	}

	// Fetch the prayer subject
	var subject models.PrayerSubject
	found, err := initializers.DB.From("prayer_subject").
		Select("*").
		Where(goqu.C("prayer_subject_id").Eq(subjectID)).
		ScanStruct(&subject)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prayer subject", "details": err.Error()})
		return
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prayer subject not found"})
		return
	}

	// Check permissions:
	// - Creator can remove link from their own subjects
	// - Linked user can remove link from themselves
	isCreator := subject.Created_By == currentUser.User_Profile_ID
	isLinkedUser := subject.User_Profile_ID != nil && *subject.User_Profile_ID == currentUser.User_Profile_ID

	if !isCreator && !isLinkedUser && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to remove this link"})
		return
	}

	// Check if there's actually a link to remove
	if subject.Link_Status == "unlinked" || subject.User_Profile_ID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This prayer subject is not linked to any user"})
		return
	}

	// Cannot unlink a self-subject (where created_by == user_profile_id)
	if subject.User_Profile_ID != nil && *subject.User_Profile_ID == subject.Created_By {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot unlink a personal 'self' prayer subject"})
		return
	}

	// Remove the link
	_, err = initializers.DB.Update("prayer_subject").
		Set(goqu.Record{
			"user_profile_id": nil,
			"link_status":     "unlinked",
			"updated_by":      currentUser.User_Profile_ID,
			"datetime_update": time.Now(),
		}).
		Where(goqu.C("prayer_subject_id").Eq(subjectID)).
		Executor().Exec()

	if err != nil {
		log.Println("Failed to remove prayer subject link:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove link", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Link removed successfully"})
}

// GetPendingConnectionRequestCount returns the count of pending incoming requests
func GetPendingConnectionRequestCount(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if userID != currentUser.User_Profile_ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this user's connection requests"})
		return
	}

	var count int
	_, err = initializers.DB.From("prayer_connection_request").
		Select(goqu.COUNT("request_id")).
		Where(
			goqu.C("target_user_id").Eq(userID),
			goqu.C("status").Eq("pending"),
		).
		ScanVal(&count)

	if err != nil {
		log.Println("Failed to count pending requests:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count pending requests", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}
