package controllers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"

	"github.com/doug-martin/goqu/v9"
	"github.com/gin-gonic/gin"
)

func CreateGroup(c *gin.Context) {
	user := c.MustGet("currentUser").(models.UserProfile)
	admin := c.MustGet("admin").(bool)

	// only admins for now
	if !admin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Only admins can create groups"})
		return
	}

	var newGroup models.GroupCreate
	if err := c.BindJSON(&newGroup); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group := models.GroupProfile{
		Group_Name:        newGroup.Group_Name,
		Group_Description: newGroup.Group_Description,
		Is_Active:         true,
		Created_By:        user.User_Profile_ID,
		Updated_By:        user.User_Profile_ID,
		Datetime_Create:   time.Now(),
		Datetime_Update:   time.Now(),
	}

	insert := initializers.DB.Insert("group_profile").Rows(group).Returning("group_profile_id")

	var insertedID int
	_, err := insert.Executor().ScanVal(&insertedID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group", "details": err.Error()})
		return
	}

	group.Group_Profile_ID = insertedID

	c.JSON(http.StatusCreated, group)
}

func GetGroup(c *gin.Context) {
	user := c.MustGet("currentUser").(models.UserProfile)
	admin := c.MustGet("admin").(bool)

	groupID, err := strconv.Atoi(c.Param("group_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group profile ID", "details": err.Error()})
		return
	}

	var group models.GroupProfile
	found, err := initializers.DB.From("group_profile").
		Select(
			goqu.I("group_profile.group_profile_id"),
			goqu.I("group_profile.group_name"),
			goqu.I("group_profile.group_description"),
			goqu.I("group_profile.is_active"),
			goqu.I("group_profile.created_by"),
			goqu.I("group_profile.updated_by"),
			goqu.I("group_profile.datetime_create"),
			goqu.I("group_profile.datetime_update"),
		).
		Join(
			goqu.T("user_group"),
			goqu.On(goqu.Ex{"group_profile.group_profile_id": goqu.I("user_group.group_profile_id")}),
		).
		Where(
			goqu.Ex{
				"group_profile.group_profile_id": groupID,
				"user_group.user_profile_id":     user.User_Profile_ID,
			},
		).
		ScanStruct(&group)

	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group", "details": err.Error()})
		return
	}
	if !found {
		if !admin {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not authorized to view this group"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	c.JSON(http.StatusOK, group)

}

// change group schema to include is_public for searches?
func GetAllGroups(c *gin.Context) {
	admin := c.MustGet("admin").(bool)

	if !admin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin only route"})
		return
	}

	var groups []models.GroupProfile
	err := initializers.DB.From("group_profile").
		ScanStructs(&groups)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, groups)
}

func UpdateGroup(c *gin.Context) {
	user := c.MustGet("currentUser").(models.UserProfile)
	admin := c.MustGet("admin").(bool)

	if !admin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Only admins can update groups"})
		return
	}

	groupID, err := strconv.Atoi(c.Param("group_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group profile ID", "details": err.Error()})
		return
	}

	var updateGroup models.GroupUpdate
	if err := c.BindJSON(&updateGroup); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := initializers.DB.Update("group_profile").
		Set(goqu.Record{
			"group_name":        updateGroup.Group_Name,
			"group_description": updateGroup.Group_Description,
			"is_active":         updateGroup.Is_Active,
			"updated_by":        user.User_Profile_ID,
			"datetime_update":   time.Now(),
		}).
		Where(goqu.C("group_profile_id").Eq(groupID))

	result, err := update.Executor().Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group", "details": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found or no changes made"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group updated successfully"})
}

// only admins for now
// todo: allow group creator to delete group
func DeleteGroup(c *gin.Context) {
	admin := c.MustGet("admin").(bool)

	if !admin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Only admins can delete groups"})
		return
	}

	groupID, err := strconv.Atoi(c.Param("group_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group profile ID", "details": err.Error()})
		return
	}

	deleteStmt := initializers.DB.Delete("group_profile").
		Where(goqu.C("group_profile_id").Eq(groupID))

	result, err := deleteStmt.Executor().Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group", "details": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group deleted successfully"})
}

func GetGroupUsers(c *gin.Context) {
	groupID, err := strconv.Atoi(c.Param("group_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group profile ID", "details": err.Error()})
		return
	}

	query := initializers.DB.From("user_group").
		Select(
			"user_profile.user_profile_id",
			"user_profile.username",
			"user_profile.email",
			"user_profile.first_name",
			"user_profile.last_name",
			"user_group.created_by",
			"user_group.updated_by",
		).
		InnerJoin(
			goqu.T("user_profile"),
			goqu.On(goqu.Ex{"user_group.user_profile_id": goqu.I("user_profile.user_profile_id")}),
		).
		Where(
			goqu.And(
				goqu.C("group_profile_id").Table("user_group").Eq(groupID),
				goqu.C("is_active").Table("user_group").IsTrue(),
			),
		)

	sql, args, err := query.ToSQL()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to construct query", "details": err.Error()})
		return
	}

	var users []models.UserProfile
	err = initializers.DB.ScanStructs(&users, sql, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group users", "details": err.Error()})
		return
	}

	if len(users) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No users found for this group"})
		return
	}

	c.JSON(http.StatusOK, users)
}

func AddUserToGroup(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	groupID, err := strconv.Atoi(c.Param("group_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group profile ID", "details": err.Error()})
		return
	}

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if !isAdmin && userID != currentUser.User_Profile_ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to add this user to the group"})
		return
	}

	// Check if the user is already in the group
	var existingEntry models.UserGroup
	found, err := initializers.DB.From("user_group").
		Where(
			goqu.And(
				goqu.C("user_profile_id").Eq(userID),
				goqu.C("group_profile_id").Eq(groupID),
			),
		).ScanStruct(&existingEntry)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing membership", "details": err.Error()})
		return
	}

	if found {
		c.JSON(http.StatusConflict, gin.H{"error": "User is already a member of this group"})
		return
	}

	newEntry := models.UserGroup{
		User_Profile_ID:  userID,
		Group_Profile_ID: groupID,
		Is_Active:        true,
		Created_By:       currentUser.User_Profile_ID,
		Updated_By:       currentUser.User_Profile_ID,
		Datetime_Create:  time.Now(),
		Datetime_Update:  time.Now(),
	}

	insert := initializers.DB.Insert("user_group").Rows(newEntry)

	_, err = insert.Executor().Exec()
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to group", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User added to group successfully"})
}

func RemoveUserFromGroup(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	groupID, err := strconv.Atoi(c.Param("group_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group profile ID", "details": err.Error()})
		return
	}

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if !isAdmin && userID != currentUser.User_Profile_ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to remove this user from the group"})
		return
	}

	deleteStmt := initializers.DB.Delete("user_group").
		Where(
			goqu.C("user_profile_id").Eq(userID),
			goqu.C("group_profile_id").Eq(groupID),
		)

	result, err := deleteStmt.Executor().Exec()
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove user from group", "details": err.Error()})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get rows affected", "details": err.Error()})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User is not a member of this group or already removed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User removed from group successfully"})
}

func GetGroupPrayers(c *gin.Context) {
	isAdmin := c.MustGet("admin").(bool)

	groupID, err := strconv.Atoi(c.Param("group_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group profile ID", "details": err.Error()})
		return
	}

	if !isGroupExists(groupID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group doesn't exist"})
		return
	}

	if !isUserInGroup(c, groupID) &&
		!isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view prayers for this group"})
		return
	}

	var userPrayers []models.UserPrayer

	dbErr := initializers.DB.From("prayer").
		Select(
			goqu.I("prayer.prayer_id"),
			goqu.I("prayer_access.prayer_access_id"),
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
			goqu.T("prayer_access"),
			goqu.On(goqu.Ex{"prayer.prayer_id": goqu.I("prayer_access.prayer_id")}),
		).
		Where(
			goqu.And(
				goqu.Ex{"prayer_access.access_type": "group"},
				goqu.Ex{"prayer_access.access_type_id": groupID},
			),
		).
		ScanStructsContext(c, &userPrayers)

	if dbErr != nil {
		c.JSON(500, gin.H{"error": dbErr.Error()})
		return
	}

	if len(userPrayers) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No prayer records found."})
		return
	}

	/*
		userProfileId is always 0
		the client can interpret 0 as meaning its a group prayer and not tied to one user
		todo -- consider making a separate struct for group prayers
	*/
	c.JSON(http.StatusOK, gin.H{
		"message": "Prayer records retrieved successfully.",
		"prayers": userPrayers,
	})
}

func CreateGroupPrayer(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	groupID, err := strconv.Atoi(c.Param("group_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group profile ID", "details": err.Error()})
		return
	}

	if !isGroupExists(groupID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group doesn't exist"})
		return
	}

	if !isUserInGroup(c, groupID) &&
		!isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to create prayers for this group"})
		return
	}

	var newPrayer models.PrayerCreate
	if err := c.BindJSON(&newPrayer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newPrayerEntry := models.Prayer{
		Prayer_Type:        newPrayer.Prayer_Type,
		Is_Private:         newPrayer.Is_Private,
		Title:              newPrayer.Title,
		Prayer_Description: newPrayer.Prayer_Description,
		Is_Answered:        newPrayer.Is_Answered,
		Datetime_Answered:  newPrayer.Datetime_Answered,
		Prayer_Priority:    newPrayer.Prayer_Priority,
		Created_By:         currentUser.User_Profile_ID,
		Updated_By:         currentUser.User_Profile_ID,
		Datetime_Create:    time.Now(),
		Datetime_Update:    time.Now(),
	}

	prayerInsert := initializers.DB.Insert("prayer").Rows(newPrayerEntry).Returning("prayer_id")

	var insertedPrayerID int
	_, err = prayerInsert.Executor().ScanVal(&insertedPrayerID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create prayer record", "details": err.Error()})
		return
	}

	newPrayerAccessEntry := models.PrayerAccess{
		Prayer_ID:       insertedPrayerID,
		Access_Type:     "group",
		Access_Type_ID:  groupID,
		Created_By:      currentUser.User_Profile_ID,
		Updated_By:      currentUser.User_Profile_ID,
		Datetime_Create: time.Now(),
		Datetime_Update: time.Now(),
	}

	prayerAccessInsert := initializers.DB.Insert("prayer_access").Rows(newPrayerAccessEntry).Returning("prayer_access_id")

	var insertedPrayerAccessID int
	_, err = prayerAccessInsert.Executor().ScanVal(&insertedPrayerAccessID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create prayer access record", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Prayer created sucessfully!",
		"prayerId":       insertedPrayerID,
		"prayerAccessId": insertedPrayerAccessID})
}

func isUserInGroup(c *gin.Context, groupID int) bool {
	currentUser := c.MustGet("currentUser").(models.UserProfile)

	var numRows int
	_, err := initializers.DB.From("user_group").
		Select(goqu.COUNT("user_group_id")).
		Where(
			goqu.Ex{
				"user_group.group_profile_id": groupID,
				"user_group.user_profile_id":  currentUser.User_Profile_ID,
			},
		).ScanVal(&numRows)

	if err != nil {
		panic(fmt.Sprintf("error checking if user is in group: %s", err))
	}

	if numRows == 1 {
		return true
	}

	return false
}

func isGroupExists(groupID int) bool {
	var numRows int
	_, err := initializers.DB.From("group_profile").
		Select(goqu.COUNT("group_profile_id")).
		Where(
			goqu.Ex{
				"group_profile.group_profile_id": groupID,
			},
		).ScanVal(&numRows)

	if err != nil {
		panic(fmt.Sprintf("error checking if group exists: %s", err))
	}

	if numRows == 1 {
		return true
	}

	return false

}
