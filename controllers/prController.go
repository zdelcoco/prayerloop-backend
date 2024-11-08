package controllers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"
	"github.com/doug-martin/goqu/v9"
)

func GetPrayerRequest(c *gin.Context) {
	user := c.MustGet("currentUser").(models.User)
	admin := c.MustGet("admin").(bool)

	prayerRequestID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prayer request ID"})
		return
	}

	query := initializers.DB.From("prayer_request").
		Where(goqu.C("prayer_request_id").Eq(prayerRequestID))

	var prayerRequest models.PrayerRequest
	found, err := query.ScanStruct(&prayerRequest)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prayer request"})
		return
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prayer request not found"})
		return
	}

	if prayerRequest.User_ID != user.User_ID && !admin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to view this prayer request"})
		return
	}

	c.JSON(http.StatusOK, prayerRequest)
}

func GetPrayerRequests(c *gin.Context) {

	user := c.MustGet("currentUser").(models.User)

	log.Println(user.User_ID)

	var prayerRequests []models.PrayerRequest
	// returns all prayer requests for the user sorted by datetime_update
	err := initializers.DB.From("prayer_request").Select("*").Where(goqu.C("user_id").Eq(user.User_ID)).Order(goqu.I("datetime_update").Desc()).ScanStructs(&prayerRequests)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if len(prayerRequests) == 0 {
		c.JSON(200, gin.H{"message": "No prayer requests found."})
		return
	}

	c.JSON(200, gin.H{
		"message":        "Prayer requests retrieved successfully.",
		"prayerRequests": prayerRequests,
	})
}

func CreatePrayerRequest(c *gin.Context) {

	user := c.MustGet("currentUser").(models.User)
	admin := c.MustGet("admin").(bool)

	var prayerRequest models.PrayerRequestCreate
	err := c.BindJSON(&prayerRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// check if user is trying to create a prayer request for someone else
	if prayerRequest.User_ID != user.User_ID &&
		!admin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to create a prayer request for this user."})
		return
	}

	prayerRequest.User_ID = user.User_ID

	insert := initializers.DB.Insert("prayer_request").Rows(prayerRequest).Executor()

	result, err := insert.Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
		return
	}
	if rowsAffected == 0 {
		log.Println("No rows were inserted")
	}

	c.JSON(200, gin.H{"message": "Prayer request created successfully."})

}

func UpdatePrayerRequest(c *gin.Context) {
	user := c.MustGet("currentUser").(models.User)
	admin := c.MustGet("admin").(bool)

	prayerRequestID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prayer request ID"})
		return
	}

	var existingPrayerRequest models.PrayerRequest
	found, err := initializers.DB.From("prayer_request").
		Where(goqu.C("prayer_request_id").Eq(prayerRequestID)).
		ScanStruct(&existingPrayerRequest)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prayer request"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prayer request not found"})
		return
	}

	if existingPrayerRequest.User_ID != user.User_ID && !admin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to update this prayer request"})
		return
	}

	var updatedPrayerRequest models.PrayerRequestCreate
	if err := c.BindJSON(&updatedPrayerRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updateRecord := goqu.Record{
		"title":           updatedPrayerRequest.Title,
		"description":     updatedPrayerRequest.Description,
		"is_private":      updatedPrayerRequest.Is_Private,
		"is_answered":     updatedPrayerRequest.Is_Answered,
		"is_ongoing":      updatedPrayerRequest.Is_Ongoing,
		"priority":        updatedPrayerRequest.Priority,
		"updated_by":      user.User_ID,
		"datetime_update": goqu.L("NOW()"),
	}

	// If the prayer request is marked as answered, add the datetime_answered field
	if updatedPrayerRequest.Is_Answered != nil && *updatedPrayerRequest.Is_Answered {
		updateRecord["datetime_answered"] = goqu.L("NOW()")
	} else {
		updateRecord["datetime_answered"] = nil
	}

	update := initializers.DB.Update("prayer_request").
		Set(updateRecord).
		Where(goqu.C("prayer_request_id").Eq(prayerRequestID))

	result, err := update.Executor().Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update prayer request"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No rows were updated"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Prayer request updated successfully"})
}

func DeletePrayerRequest(c *gin.Context) {
	user := c.MustGet("currentUser").(models.User)
	admin := c.MustGet("admin").(bool)

	prayerRequestID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prayer request ID"})
		return
	}

	var existingPrayerRequest models.PrayerRequest
	found, err := initializers.DB.From("prayer_request").
		Where(goqu.C("prayer_request_id").Eq(prayerRequestID)).
		ScanStruct(&existingPrayerRequest)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prayer request"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prayer request not found"})
		return
	}

	if existingPrayerRequest.User_ID != user.User_ID && !admin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to delete this prayer request"})
		return
	}

	deleteQuery := initializers.DB.Delete("prayer_request").
		Where(goqu.C("prayer_request_id").Eq(prayerRequestID))

	result, err := deleteQuery.Executor().Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete prayer request"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No rows were deleted"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Prayer request deleted successfully"})
}
