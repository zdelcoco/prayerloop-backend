package controllers

import (
	"net/http"
	"strconv"

	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"
	"github.com/PrayerLoop/services"

	"github.com/doug-martin/goqu/v9"
	"github.com/gin-gonic/gin"
)

func GetUserNotifications(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if userID != currentUser.User_Profile_ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this user's notifications"})
		return
	}

	var notifications []models.Notification

	dbErr := initializers.DB.From("notification").
		Select("notification_id",
			"user_profile_id",
			"notification_type",
			"notification_message",
			"notification_status",
			"datetime_create",
			"datetime_update",
			"created_by",
			"updated_by",
			"target_prayer_id",
			"target_group_id").
		Where(goqu.C("user_profile_id").Eq(userID)).
		Order(goqu.C("datetime_create").Desc()).
		ScanStructs(&notifications)

	if dbErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": dbErr.Error()})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

func ToggleUserNotificationStatus(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if userID != currentUser.User_Profile_ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to modify this user's notifications"})
		return
	}

	notificationID, err := strconv.Atoi(c.Param("notification_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID", "details": err.Error()})
		return
	}

	// get current notication status
	var currentStatus string
	_, dbErr := initializers.DB.From("notification").
		Select("notification_status").
		Where(goqu.C("notification_id").Eq(notificationID)).
		ScanVal(&currentStatus)

	if dbErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": dbErr.Error()})
		return
	}

	// toggle notification status
	var newStatus string
	if currentStatus == models.NotificationStatusRead {
		newStatus = models.NotificationStatusUnread
	} else {
		newStatus = models.NotificationStatusRead
	}

	update := initializers.DB.Update("notification").
		Set(goqu.Record{"notification_status": newStatus}).
		Where(goqu.C("notification_id").Eq(notificationID))

	result, err := update.Executor().Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notification", "details": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as " + newStatus})
}

func DeleteUserNotification(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if userID != currentUser.User_Profile_ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this user's notifications"})
		return
	}

	notificationID, err := strconv.Atoi(c.Param("notification_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID", "details": err.Error()})
		return
	}

	// Verify the notification belongs to the user before deleting
	var notificationUserID int
	_, dbErr := initializers.DB.From("notification").
		Select("user_profile_id").
		Where(goqu.C("notification_id").Eq(notificationID)).
		ScanVal(&notificationUserID)

	if dbErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	if notificationUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "This notification does not belong to the specified user"})
		return
	}

	// Delete the notification
	deleteQuery := initializers.DB.Delete("notification").
		Where(goqu.C("notification_id").Eq(notificationID))

	result, err := deleteQuery.Executor().Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification", "details": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted successfully"})
}

func MarkAllNotificationsAsRead(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if userID != currentUser.User_Profile_ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to modify this user's notifications"})
		return
	}

	// Update all unread notifications to read
	update := initializers.DB.Update("notification").
		Set(goqu.Record{"notification_status": models.NotificationStatusRead}).
		Where(
			goqu.C("user_profile_id").Eq(userID),
			goqu.C("notification_status").Eq(models.NotificationStatusUnread),
		)

	result, err := update.Executor().Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notifications as read", "details": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()

	c.JSON(http.StatusOK, gin.H{
		"message":       "All notifications marked as read",
		"updatedCount": rowsAffected,
	})
}

type SendNotificationRequest struct {
	UserIDs  []int             `json:"userIds" binding:"required"`
	Title    string            `json:"title" binding:"required"`
	Body     string            `json:"body" binding:"required"`
	Data     map[string]string `json:"data,omitempty"`
	Sound    string            `json:"sound,omitempty"`
	Badge    string            `json:"badge,omitempty"`
	Priority string            `json:"priority,omitempty"`
}

func SendPushNotification(c *gin.Context) {
	var request SendNotificationRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get push notification service
	pushService := services.GetPushNotificationService()
	if pushService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Push notification service not available"})
		return
	}

	// Create notification payload
	payload := services.NotificationPayload{
		Title:    request.Title,
		Body:     request.Body,
		Data:     request.Data,
		Sound:    request.Sound,
		Badge:    request.Badge,
		Priority: request.Priority,
	}

	// Send notifications to all specified users
	err := pushService.SendNotificationToUsers(request.UserIDs, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send push notifications", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Push notifications sent successfully",
		"userIds": request.UserIDs,
	})
}
