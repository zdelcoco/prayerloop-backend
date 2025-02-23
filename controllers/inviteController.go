package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"

	"github.com/gin-gonic/gin"
)

func CreateGroupInviteCode(c *gin.Context) {
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
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to generate an invite code for this group"})
		return
	}

	inviteCode := generateInviteCode(groupID)

	groupInvite := models.GroupInvite{
		Group_Profile_ID: groupID,
		Invite_Code:      inviteCode,
		Datetime_Create:  time.Now(),
		Datetime_Update:  time.Now(),
		Created_By:       currentUser.User_Profile_ID,
		Updated_By:       currentUser.User_Profile_ID,
		Datetime_Expires: time.Now().AddDate(0, 0, 7),
		Is_Active:        true,
	}

	insert := initializers.DB.Insert("group_invite").Rows(groupInvite).Returning("invite_code")

	var insertedInviteCode string
	_, insertErr := insert.Executor().ScanVal(&insertedInviteCode)
	if insertErr != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate invite code", "details": insertErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"inviteCode": insertedInviteCode, "expiresAt": groupInvite.Datetime_Expires})

}

func generateInviteCode(id int) string {
	randomBytes := make([]byte, 2)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}

	randomString := hex.EncodeToString(randomBytes)

	return strings.ToUpper(fmt.Sprintf("%04d-%s", id, randomString))
}
