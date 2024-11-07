package controllers

import (
	"log"

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
