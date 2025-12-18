package controllers

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prayer ID", "details": err.Error()})
		return
	}

	var userPrayers []models.UserPrayer

	// user_profile_id will be nil if all prayer_access records have been deleted
	// since the struct can't handle that, assign to 0
	query := goqu.From("prayer").
		Distinct("user_profile_id").
		Select(
			goqu.Case().
				When(goqu.I("prayer_access.access_type").Eq("user"), goqu.I("prayer_access.access_type_id")).
				When(goqu.I("prayer_access.access_type").Eq("group"), goqu.I("user_group.user_profile_id")).
				Else(0).
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build query", "details": err.Error()})
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
			goqu.On(
				goqu.And(
					goqu.Ex{"prayer_access.prayer_id": goqu.I("prayer.prayer_id")},
					goqu.Ex{"prayer.deleted": false},
				),
			),
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

func AddPrayerAccess(c *gin.Context) {
	userID := c.MustGet("currentUser").(models.UserProfile).User_Profile_ID
	admin := c.MustGet("admin").(bool)

	prayerId, err := strconv.Atoi(c.Param("prayer_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prayer ID", "details": err.Error()})
		return
	}

	var newPrayerAccess models.PrayerAccessCreate
	if err := c.BindJSON(&newPrayerAccess); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var existingPrayer models.Prayer
	prayerFound, err := initializers.DB.From("prayer").
		Where(goqu.C("prayer_id").Eq(prayerId), goqu.C("deleted").Eq(false)).
		ScanStruct(&existingPrayer)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Prayer record doesn't exist or is marked deleted", "details": err.Error()})
		return
	}

	if prayerFound {
		// check if access is already granted
		var existingPrayerAccess models.PrayerAccess
		accessGranted, err := initializers.DB.From("prayer_access").
			Select(
				goqu.I("prayer_access.prayer_access_id"),
				goqu.I("prayer_access.prayer_id"),
				goqu.I("prayer_access.access_type"),
				goqu.I("prayer_access.access_type_id"),
				goqu.I("prayer_access.datetime_create"),
				goqu.I("prayer_access.datetime_update"),
				goqu.I("prayer_access.created_by"),
				goqu.I("prayer_access.updated_by"),
			).
			Where(
				goqu.C("access_type").Eq(newPrayerAccess.Access_Type),
				goqu.C("access_type_id").Eq(newPrayerAccess.Access_Type_ID),
				goqu.C("prayer_id").Eq(prayerId)).
			ScanStruct(&existingPrayerAccess)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check if access is already granted", "details": err.Error()})
			return
		}

		if accessGranted {
			c.JSON(http.StatusConflict, gin.H{"error": "Access already granted"})
			return
		}

		// check to see if logged in user has permission to give access to this prayer
		var prayerAccess models.PrayerAccess
		accessFound, err := initializers.DB.From("prayer_access").
			Select(
				goqu.I("prayer_access.prayer_access_id"),
				goqu.I("prayer_access.prayer_id"),
				goqu.I("prayer_access.access_type"),
				goqu.I("prayer_access.access_type_id"),
				goqu.I("prayer_access.datetime_create"),
				goqu.I("prayer_access.datetime_update"),
				goqu.I("prayer_access.created_by"),
				goqu.I("prayer_access.updated_by"),
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
			Where(goqu.C("prayer_id").Eq(prayerId), goqu.C("user_profile_id").Eq(userID)).
			Order(goqu.I("user_profile_id").Asc(), goqu.I("prayer_access.access_type").Asc()).
			ScanStruct(&prayerAccess)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prayer access record", "details": err.Error()})
			return
		}

		if !admin && prayerAccess.Created_By != userID {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to grant access to this prayer"})
			return
		} else if accessFound || admin {

			prayerAccessInsert := models.PrayerAccess{
				Prayer_ID:      prayerId,
				Access_Type:    newPrayerAccess.Access_Type,
				Access_Type_ID: newPrayerAccess.Access_Type_ID,
				Created_By:     userID,
				Updated_By:     userID,
			}

			insert := initializers.DB.Insert("prayer_access").Rows(prayerAccessInsert).Returning("prayer_access_id")

			var insertedPrayerAccessID int
			_, err = insert.Executor().ScanVal(&insertedPrayerAccessID)
			if err != nil {
				log.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add prayer access record", "details": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Prayer access added successfully"})
		}
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prayer record not found"})
		return
	}

}

func RemovePrayerAccess(c *gin.Context) {
	userID := c.MustGet("currentUser").(models.UserProfile).User_Profile_ID
	admin := c.MustGet("admin").(bool)

	prayerId, err := strconv.Atoi(c.Param("prayer_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prayer ID", "details": err.Error()})
		return
	}

	var existingPrayer models.Prayer
	prayerFound, err := initializers.DB.From("prayer").
		Where(goqu.C("prayer_id").Eq(prayerId), goqu.C("deleted").Eq(false)).
		ScanStruct(&existingPrayer)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Prayer record doesn't exist or is marked deleted", "details": err.Error()})
		return
	}

	if !prayerFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prayer record not found"})
		return
	}

	accessId, err := strconv.Atoi(c.Param("prayer_access_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prayer access ID", "details": err.Error()})
		return
	}

	var existingPrayerAccess models.PrayerAccess
	accessExists, err := initializers.DB.From("prayer_access").
		Select(
			goqu.I("prayer_access.prayer_access_id"),
			goqu.I("prayer_access.prayer_id"),
			goqu.I("prayer_access.access_type"),
			goqu.I("prayer_access.access_type_id"),
			goqu.I("prayer_access.datetime_create"),
			goqu.I("prayer_access.datetime_update"),
			goqu.I("prayer_access.created_by"),
			goqu.I("prayer_access.updated_by"),
		).
		Where(goqu.C("prayer_access_id").Eq(accessId)).
		ScanStruct(&existingPrayerAccess)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prayer access record", "details": err.Error()})
		return
	}

	if !accessExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prayer access record not found"})
		return
	}

	if existingPrayerAccess.Access_Type == "group" {
		var group models.GroupProfile
		groupFound, err := initializers.DB.From("group_profile").
			Select(
				"group_profile_id",
				"group_name",
				"group_description",
				"is_active",
				"datetime_create",
				"datetime_update",
				"created_by",
				"updated_by",
				"deleted",
			).
			Where(goqu.C("group_profile_id").Eq(existingPrayerAccess.Access_Type_ID)).
			ScanStruct(&group)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group record", "details": err.Error()})
			return
		}

		if !groupFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group record not found"})
			return
		}

		if !isUserInGroup(c, group.Group_Profile_ID) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("You are not in group %d", group.Group_Profile_ID)})
			return
		}

		// allow deletion if user is admin, or if user is creator of the prayer, or if user is creator of the group
		if !admin && (group.Created_By != userID || existingPrayer.Created_By != userID) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to remove access to this prayer"})
			return
		}

	} else if existingPrayerAccess.Access_Type == "user" {

		// allow deletion if user is admin, or if user either created the prayer or is the user to whom access is granted
		if !admin && (existingPrayer.Created_By != userID || existingPrayerAccess.Access_Type_ID != userID) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to remove access to this prayer"})
			return
		}

		// When user is deleting their own prayer (access_type = "user"), delete ALL prayer_access records
		// and then delete the prayer itself
		if existingPrayerAccess.Access_Type_ID == userID && existingPrayer.Created_By == userID {
			// First, delete all prayer_access records for this prayer
			deleteAllAccessQuery := initializers.DB.Delete("prayer_access").
				Where(goqu.C("prayer_id").Eq(prayerId))

			_, err := deleteAllAccessQuery.Executor().Exec()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete all prayer access records", "details": err.Error()})
				return
			}

			// Then, mark the prayer as deleted
			deletePrayerQuery := initializers.DB.Update("prayer").
				Set(goqu.Record{
					"deleted": true,
					"updated_by": userID,
					"datetime_update": goqu.L("NOW()"),
				}).
				Where(goqu.C("prayer_id").Eq(prayerId))

			result, err := deletePrayerQuery.Executor().Exec()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete prayer", "details": err.Error()})
				return
			}

			rowsAffected, _ := result.RowsAffected()
			if rowsAffected == 0 {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "No prayer rows were deleted"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Prayer and all access records removed successfully"})
			return
		}
	}

	// Default behavior: delete only the specific prayer_access record (for group deletions or other cases)
	deleteQuery := initializers.DB.Delete("prayer_access").
		Where(goqu.C("prayer_access_id").Eq(accessId))

	result, err := deleteQuery.Executor().Exec()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete prayer access record", "details": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()

	if rowsAffected == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No rows were deleted"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Prayer access removed successfully"})

}

func UpdatePrayer(c *gin.Context) {
	userID := c.MustGet("currentUser").(models.UserProfile).User_Profile_ID
	admin := c.MustGet("admin").(bool)

	prayerId, err := strconv.Atoi(c.Param("prayer_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prayer ID", "details": err.Error()})
		return
	}

	var existingPrayer models.Prayer
	prayerFound, err := initializers.DB.From("prayer").
		Where(goqu.C("prayer_id").Eq(prayerId), goqu.C("deleted").Eq(false)).
		ScanStruct(&existingPrayer)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Prayer record doesn't exist or is marked deleted", "details": err.Error()})
		return
	}

	if !prayerFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prayer record not found"})
		return
	}

	var updatedPrayer models.PrayerCreate
	if err := c.BindJSON(&updatedPrayer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if !admin && existingPrayer.Created_By != userID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to update this prayer"})
		return
	}

	// if any incoming field is nil, retain the existing value
	// pass updatedPrayer by reference to modify the original (underlying) struct
	existingPrayerValue := reflect.ValueOf(existingPrayer)
	updatedPrayerValue := reflect.ValueOf(&updatedPrayer).Elem()

	// if any field in updatedPrayer is zero value, get the corresponding field value
	// from the existingPrayer struct
	for i := 0; i < updatedPrayerValue.NumField(); i++ {
		field := updatedPrayerValue.Field(i)
		if field.IsZero() {
			existingField := existingPrayerValue.Field(i)
			if field.Type().AssignableTo(existingField.Type()) {
				field.Set(existingField)
			}
		}
	}

	updateQuery := initializers.DB.Update("prayer").
		Set(goqu.Record{
			"prayer_type":        updatedPrayer.Prayer_Type,
			"is_private":         updatedPrayer.Is_Private,
			"title":              updatedPrayer.Title,
			"prayer_description": updatedPrayer.Prayer_Description,
			"is_answered":        updatedPrayer.Is_Answered,
			"datetime_answered":  updatedPrayer.Datetime_Answered,
			"prayer_priority":    updatedPrayer.Prayer_Priority,
			"updated_by":         userID,
			"datetime_update":    goqu.L("NOW()"),
		}).
		Where(goqu.C("prayer_id").Eq(prayerId))

	result, err := updateQuery.Executor().Exec()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update prayer record", "details": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()

	if rowsAffected == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No rows were updated"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Prayer record updated successfully"})

}

func DeletePrayer(c *gin.Context) {
	userID := c.MustGet("currentUser").(models.UserProfile).User_Profile_ID
	admin := c.MustGet("admin").(bool)

	prayerId, err := strconv.Atoi(c.Param("prayer_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prayer ID", "details": err.Error()})
		return
	}

	var existingPrayer models.Prayer
	prayerFound, err := initializers.DB.From("prayer").
		Where(goqu.C("prayer_id").Eq(prayerId), goqu.C("deleted").Eq(false)).
		ScanStruct(&existingPrayer)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Prayer record doesn't exist or is already marked deleted", "details": err.Error()})
		return
	}

	if !prayerFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prayer record not found"})
		return
	}

	if !admin && existingPrayer.Created_By != userID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to delete this prayer"})
		return
	}

	// find any assoicated prayer_access records.  if any exist, cannot delete prayer
	var prayerAccessCount int
	_, err = initializers.DB.From("prayer_access").
		Select(goqu.COUNT("*")).
		Where(goqu.C("prayer_id").Eq(prayerId)).
		ScanVal(&prayerAccessCount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for related prayer access records", "details": err.Error()})
		return
	}

	if prayerAccessCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete prayer record while related access record(s) exist"})
		return
	}

	updateQuery := initializers.DB.Update("prayer").
		Set(goqu.Record{
			"deleted": true,
		}).
		Where(goqu.C("prayer_id").Eq(prayerId))

	result, err := updateQuery.Executor().Exec()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark prayer record as deleted", "details": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()

	if rowsAffected == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No rows were marked as deleted"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Prayer record marked as deleted successfully"})
}
