package controllers

import (
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
		return
	}

	group.Group_Profile_ID = insertedID

	c.JSON(http.StatusCreated, group)
}

func GetGroup(c *gin.Context) {
	groupID, err := strconv.Atoi(c.Param("group_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group profile ID"})
		return
	}

	var group models.GroupProfile
	found, err := initializers.DB.From("group_profile").
		Where(goqu.C("group_profile_id").Eq(groupID)).
		ScanStruct(&group)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	c.JSON(http.StatusOK, group)
}

// probably make this an admin function later
// or change group schema to include is_public for searches
func GetAllGroups(c *gin.Context) {
	var groups []models.GroupProfile
	err := initializers.DB.From("group_profile").
		ScanStructs(&groups)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group profile ID"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group profile ID"})
		return
	}

	deleteStmt := initializers.DB.Delete("group_profile").
		Where(goqu.C("group_profile_id").Eq(groupID))

	result, err := deleteStmt.Executor().Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group profile ID"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to construct query"})
		return
	}

	var users []models.UserProfile
	err = initializers.DB.ScanStructs(&users, sql, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group users"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group profile ID"})
		return
	}

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing membership"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User added to group successfully"})
}

func RemoveUserFromGroup(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	groupID, err := strconv.Atoi(c.Param("group_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group profile ID"})
		return
	}

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove user from group"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get rows affected"})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User is not a member of this group or already removed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User removed from group successfully"})
}
