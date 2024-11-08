package controllers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"
	"github.com/doug-martin/goqu/v9"
)

func GetPrayerRequestsForUser(c *gin.Context) {

	user, _ := c.Get("currentUser")

	log.Println(user.(models.User).User_ID)

	var prayerRequests []models.PrayerRequest
	// returns all prayer requests for the user sorted by datetime_update
	err := initializers.DB.From("prayer_request").Select("*").Where(goqu.C("user_id").Eq(user.(models.User).User_ID)).Order(goqu.I("datetime_update").Desc()).ScanStructs(&prayerRequests)
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

	user, _ := c.Get("currentUser")
	admin := c.MustGet("admin")

	var prayerRequest models.PrayerRequestCreate
	err := c.BindJSON(&prayerRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// check if user is trying to create a prayer request for someone else
	if prayerRequest.User_ID != user.(models.User).User_ID &&
		!admin.(bool) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to create a prayer request for this user."})
		return
	}

	prayerRequest.User_ID = user.(models.User).User_ID

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
