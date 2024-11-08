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
	user := c.MustGet("currentUser").(models.User)
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

	group := models.Group{
		Group_Name:      newGroup.Group_Name,
		Description:     newGroup.Group_Description,
		Is_Active:       true,
		Created_By:      user.User_ID,
		Updated_By:      user.User_ID,
		Datetime_Create: time.Now(),
		Datetime_Update: time.Now(),
	}

	insert := initializers.DB.Insert("group").Rows(group).Returning("group_id")

	var insertedID int
	_, err := insert.Executor().ScanVal(&insertedID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
		return
	}

	group.Group_ID = insertedID

	c.JSON(http.StatusCreated, group)
}

func GetGroup(c *gin.Context) {
	groupID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	var group models.Group
	found, err := initializers.DB.From("group").
		Where(goqu.C("group_id").Eq(groupID)).
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
	var groups []models.Group
	err := initializers.DB.From("group").
		ScanStructs(&groups)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
		return
	}

	c.JSON(http.StatusOK, groups)
}

func UpdateGroup(c *gin.Context) {
	user := c.MustGet("currentUser").(models.User)
	admin := c.MustGet("admin").(bool)

	if !admin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Only admins can update groups"})
		return
	}

	groupID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	var updateGroup models.GroupUpdate
	if err := c.BindJSON(&updateGroup); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := initializers.DB.Update("group").
		Set(goqu.Record{
			"group_name":      updateGroup.Group_Name,
			"description":     updateGroup.Group_Description,
			"is_active":       updateGroup.Is_Active,
			"updated_by":      user.User_ID,
			"datetime_update": time.Now(),
		}).
		Where(goqu.C("group_id").Eq(groupID))

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

func DeleteGroup(c *gin.Context) {
	admin := c.MustGet("admin").(bool)

	if !admin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Only admins can delete groups"})
		return
	}

	groupID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	deleteStmt := initializers.DB.Delete("group").
		Where(goqu.C("group_id").Eq(groupID))

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
