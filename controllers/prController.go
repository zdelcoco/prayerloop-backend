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

func GetPrayer(c *gin.Context) {
	user := c.MustGet("currentUser").(models.UserProfile)
	admin := c.MustGet("admin").(bool)

	prayerId, err := strconv.Atoi(c.Param("prayer_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prayer ID"})
		return
	}

	var userPrayers []models.UserPrayer

	query := goqu.From("prayer").
		Distinct("user_profile_id").
		Select(
			goqu.Case().
				When(goqu.I("prayer_access.access_type").Eq("user"), goqu.I("prayer_access.access_type_id")).
				When(goqu.I("prayer_access.access_type").Eq("group"), goqu.I("user_group.user_profile_id")).
				Else(nil).
				As("user_profile_id"),
			goqu.I("prayer.prayer_id"),
			goqu.I("prayer.prayer_type"),
			goqu.I("prayer.is_private"),
			goqu.I("prayer.title"),
			goqu.I("prayer.prayer_description"),
			goqu.I("prayer.is_answered"),
			goqu.I("prayer.prayer_priority"),
			goqu.I("prayer.datetime_answered"),
			goqu.I("prayer.created_by"),
			goqu.I("prayer.datetime_create"),
			goqu.I("prayer.updated_by"),
			goqu.I("prayer.datetime_update"),
			goqu.I("prayer.deleted"),
		).
		LeftJoin(goqu.T("prayer_access"), goqu.On(goqu.Ex{"prayer.prayer_id": goqu.I("prayer_access.prayer_id")})).
		LeftJoin(goqu.T("user_group"), goqu.On(
			goqu.Or(
				goqu.Ex{"prayer_access.access_type": "group", "prayer_access.access_type_id": goqu.I("user_group.group_profile_id")},
				goqu.Ex{"prayer_access.access_type": "user", "prayer_access.access_type_id": goqu.I("user_group.user_profile_id")},
			),
		)).
		Where(goqu.And(goqu.I("prayer.prayer_id").Eq(prayerId))).
		Order(goqu.I("user_profile_id").Asc(), goqu.I("prayer_access.access_type").Asc())

	sql, _, err := query.ToSQL()
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build query"})
		return
	}

	if err := initializers.DB.ScanStructsContext(c, &userPrayers, sql); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prayer record"})
		return
	}

	if len(userPrayers) > 0 {
		// return first instance of prayer regardless of user access
		if admin {
			c.JSON(http.StatusOK, userPrayers[0])
			return
		}

		// otherwise, find the first instance of the prayer that the user has access
		for _, up := range userPrayers {
			if up.User_Profile_ID == user.User_Profile_ID {
				c.JSON(http.StatusOK, up)
				return
			}
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to view this prayer record"})
		return

	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prayer record not found"})
		return
	}

}

func GetPrayers(c *gin.Context) {

	user := c.MustGet("currentUser").(models.UserProfile)

	log.Println(user.User_Profile_ID)

	var userPrayers []models.UserPrayer

	err := initializers.DB.From("prayer_access").
		Select(
			goqu.DISTINCT("user_profile_id"),
			goqu.Case().
				When(goqu.I("prayer_access.access_type").Eq("user"), goqu.I("prayer_access.access_type_id")).
				When(goqu.I("prayer_access.access_type").Eq("group"), goqu.I("user_group.user_profile_id")).
				Else(nil).
				As("user_profile_id"),
			goqu.I("prayer.prayer_id"),
			goqu.I("prayer.prayer_type"),
			goqu.I("prayer.is_private"),
			goqu.I("prayer.title"),
			goqu.I("prayer.prayer_description"),
			goqu.I("prayer.is_answered"),
			goqu.I("prayer.prayer_priority"),
			goqu.I("prayer.datetime_answered"),
			goqu.I("prayer.created_by"),
			goqu.I("prayer.datetime_create"),
			goqu.I("prayer.updated_by"),
			goqu.I("prayer.datetime_update"),
			goqu.I("prayer.deleted"),
		).
		Join(
			goqu.T("user_group"),
			goqu.On(
				goqu.Or(
					goqu.Ex{"prayer_access.access_type": "group", "prayer_access.access_type_id": goqu.I("user_group.group_profile_id")},
					goqu.Ex{"prayer_access.access_type": "user", "prayer_access.access_type_id": goqu.I("user_group.user_profile_id")},
				),
			),
		).
		Join(
			goqu.T("prayer"),
			goqu.On(goqu.Ex{"prayer_access.prayer_id": goqu.I("prayer.prayer_id")}),
		).
		Where(goqu.Ex{"user_group.user_profile_id": user.User_Profile_ID}).
		Order(goqu.I("prayer.prayer_id").Asc()).
		ScanStructsContext(c, &userPrayers)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if len(userPrayers) == 0 {
		c.JSON(200, gin.H{"message": "No prayer records found."})
		return
	}

	c.JSON(200, gin.H{
		"message": "Prayer records retrieved successfully.",
		"prayers": userPrayers,
	})
}

// func CreatePrayerRequest(c *gin.Context) {

// 	user := c.MustGet("currentUser").(models.User)
// 	admin := c.MustGet("admin").(bool)

// 	var prayerRequest models.PrayerRequestCreate
// 	err := c.BindJSON(&prayerRequest)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	// check if user is trying to create a prayer request for someone else
// 	if prayerRequest.User_ID != user.User_ID &&
// 		!admin {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to create a prayer request for this user."})
// 		return
// 	}

// 	prayerRequest.User_ID = user.User_ID

// 	insert := initializers.DB.Insert("prayer_request").Rows(prayerRequest).Executor()

// 	result, err := insert.Exec()
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	rowsAffected, err := result.RowsAffected()
// 	if err != nil {
// 		log.Printf("Error getting rows affected: %v", err)
// 		return
// 	}
// 	if rowsAffected == 0 {
// 		log.Println("No rows were inserted")
// 	}

// 	c.JSON(200, gin.H{"message": "Prayer request created successfully."})

// }

// func UpdatePrayerRequest(c *gin.Context) {
// 	user := c.MustGet("currentUser").(models.User)
// 	admin := c.MustGet("admin").(bool)

// 	prayerId, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prayer request ID"})
// 		return
// 	}

// 	var existingPrayerRequest models.PrayerRequest
// 	found, err := initializers.DB.From("prayer_request").
// 		Where(goqu.C("prayer_request_id").Eq(prayerId)).
// 		ScanStruct(&existingPrayerRequest)

// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prayer request"})
// 		return
// 	}
// 	if !found {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Prayer request not found"})
// 		return
// 	}

// 	if existingPrayerRequest.User_ID != user.User_ID && !admin {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to update this prayer request"})
// 		return
// 	}

// 	var updatedPrayerRequest models.PrayerRequestCreate
// 	if err := c.BindJSON(&updatedPrayerRequest); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	updateRecord := goqu.Record{
// 		"title":           updatedPrayerRequest.Title,
// 		"description":     updatedPrayerRequest.Description,
// 		"is_private":      updatedPrayerRequest.Is_Private,
// 		"is_answered":     updatedPrayerRequest.Is_Answered,
// 		"is_ongoing":      updatedPrayerRequest.Is_Ongoing,
// 		"priority":        updatedPrayerRequest.Priority,
// 		"updated_by":      user.User_ID,
// 		"datetime_update": goqu.L("NOW()"),
// 	}

// 	// If the prayer request is marked as answered, add the datetime_answered field
// 	if updatedPrayerRequest.Is_Answered != nil && *updatedPrayerRequest.Is_Answered {
// 		updateRecord["datetime_answered"] = goqu.L("NOW()")
// 	} else {
// 		updateRecord["datetime_answered"] = nil
// 	}

// 	update := initializers.DB.Update("prayer_request").
// 		Set(updateRecord).
// 		Where(goqu.C("prayer_request_id").Eq(prayerId))

// 	result, err := update.Executor().Exec()
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update prayer request"})
// 		return
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "No rows were updated"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"message": "Prayer request updated successfully"})
// }

// func DeletePrayerRequest(c *gin.Context) {
// 	user := c.MustGet("currentUser").(models.User)
// 	admin := c.MustGet("admin").(bool)

// 	prayerId, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prayer request ID"})
// 		return
// 	}

// 	var existingPrayerRequest models.PrayerRequest
// 	found, err := initializers.DB.From("prayer_request").
// 		Where(goqu.C("prayer_request_id").Eq(prayerId)).
// 		ScanStruct(&existingPrayerRequest)

// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prayer request"})
// 		return
// 	}
// 	if !found {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Prayer request not found"})
// 		return
// 	}

// 	if existingPrayerRequest.User_ID != user.User_ID && !admin {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to delete this prayer request"})
// 		return
// 	}

// 	deleteQuery := initializers.DB.Delete("prayer_request").
// 		Where(goqu.C("prayer_request_id").Eq(prayerId))

// 	result, err := deleteQuery.Executor().Exec()
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete prayer request"})
// 		return
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "No rows were deleted"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"message": "Prayer request deleted successfully"})
// }
