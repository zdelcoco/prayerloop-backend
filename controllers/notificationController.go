package controllers

import (
	"net/http"
	"strconv"

	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"

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
			"updated_by").
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
	if currentStatus == "READ" {
		newStatus = "UNREAD"
	} else {
		newStatus = "READ"
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
