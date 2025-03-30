package controllers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"

	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"
	"github.com/doug-martin/goqu/v9"
	"golang.org/x/crypto/bcrypt"
)

func UserSignup(c *gin.Context) {
	admin := c.MustGet("admin")

	if admin != true {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "must be logged in as an admin to create a user."})
		return
	}

	var user models.UserProfileSignup

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userCount, err := initializers.DB.From("user_profile").Select("username").Where(goqu.C("username").Eq(user.Username)).Count()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if userCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username already exists."})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newUser := models.UserProfile{
		Username:   user.Username,
		Password:   string(passwordHash),
		Email:      user.Email,
		First_Name: user.First_Name,
		Last_Name:  user.Last_Name,
		Created_By: 1,
		Updated_By: 1,
	}

	insert := initializers.DB.Insert("user_profile").Rows(newUser).Executor()
	if _, err := insert.Exec(); err != nil {
		log.Default().Println(insert)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	} else {
		c.JSON(200, gin.H{
			"message": "User created successfully.",
			"user":    user,
		})
	}
}

func UserLogin(c *gin.Context) {
	var user models.Login

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var dbUser models.UserProfile
	_, err := initializers.DB.From("user_profile").Select("*").Where(goqu.C("username").Eq(user.Username)).ScanStruct(&dbUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
	// 	return
	// }

	// update := initializers.DB.Update("user_profile").Set(goqu.Record{"password": string(passwordHash)}).Where(goqu.C("user_profile_id").Eq(dbUser.User_Profile_ID)).Executor()
	// if _, err := update.Exec(); err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
	// 	return
	// }

	err = bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(user.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password", "details": err.Error()})
		return
	}

	role := ""
	if strings.HasPrefix(dbUser.Username, "admin") {
		role = "admin"
	} else {
		role = "user"
	}

	generateToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":   dbUser.User_Profile_ID,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
		"role": role,
	})

	token, err := generateToken.SignedString([]byte(os.Getenv("SECRET")))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to generate token", "details": err.Error()})
	}

	c.JSON(200, gin.H{
		"message": "User logged in successfully.",
		"token":   token,
		"user":    dbUser,
	})
}

func GetUserProfile(c *gin.Context) {

	user, _ := c.Get("currentUser")

	c.JSON(200, gin.H{
		"user":  user,
		"admin": c.MustGet("admin"),
	})
}

func GetUserGroups(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if userID != currentUser.User_Profile_ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this user's groups"})
		return
	}

	query := initializers.DB.From("user_group").
		Select(
			"group_profile.group_profile_id",
			"group_profile.group_name",
			"group_profile.group_description",
			"group_profile.is_active",
			"group_profile.datetime_create",
			"group_profile.datetime_update",
			"group_profile.created_by",
			"group_profile.updated_by",
			"group_profile.deleted",
		).
		InnerJoin(
			goqu.T("group_profile"),
			goqu.On(goqu.Ex{"user_group.group_profile_id": goqu.I("group_profile.group_profile_id")}),
		).
		Where(
			goqu.And(
				goqu.C("user_profile_id").Table("user_group").Eq(userID),
				goqu.C("is_active").Table("user_group").IsTrue(),
				goqu.C("is_active").Table("group_profile").IsTrue(),
			),
		)

	sql, args, err := query.ToSQL()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to construct query", "details": err.Error()})
		return
	}

	log.Println(sql, args)

	var groups []models.GroupProfile
	err = initializers.DB.ScanStructs(&groups, sql, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user groups", "details": err.Error()})
		return
	}

	if len(groups) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No groups found for this user"})
		return
	}

	c.JSON(http.StatusOK, groups)
}

func GetUserPrayers(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if userID != currentUser.User_Profile_ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this user's prayers"})
		return
	}

	var userPrayers []models.UserPrayer

	dbErr := initializers.DB.From("prayer_access").
		Select(
			goqu.DISTINCT("user_profile_id"),
			goqu.I("user_group.user_profile_id").As("user_profile_id"),
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
			goqu.T("user_group"),
			goqu.On(
				goqu.Ex{"prayer_access.access_type": "user", "prayer_access.access_type_id": goqu.I("user_group.user_profile_id")},
			),
		).
		Join(
			goqu.T("prayer"),
			goqu.On(goqu.Ex{"prayer_access.prayer_id": goqu.I("prayer.prayer_id")}),
		).
		Where(
			goqu.And(
				goqu.Ex{"user_group.user_profile_id": currentUser.User_Profile_ID},
				goqu.Ex{"prayer.deleted": false},
			),
		).
		Order(goqu.I("prayer.prayer_id").Asc()).
		ScanStructsContext(c, &userPrayers)

	if dbErr != nil {
		c.JSON(500, gin.H{"error": dbErr.Error()})
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

func CreateUserPrayer(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if currentUser.User_Profile_ID != userID &&
		!isAdmin {
		c.JSON(http.StatusForbidden,
			gin.H{"error": fmt.Sprintf("You don't have permission to create a prayer on behalf of user %d",
				currentUser.User_Profile_ID)})
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
		Access_Type:     "user",
		Access_Type_ID:  userID,
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

func GetUserPreferences(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if userID != currentUser.User_Profile_ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this user's preferences"})
		return
	}

	var preferences []models.UserPreferences

	dbErr := initializers.DB.From("user_preferences").
		Select("*").
		Where(goqu.C("user_profile_id").Eq(userID)).
		Order(goqu.C("preference_key").Asc()).
		ScanStructsContext(c, &preferences)

	if dbErr != nil {
		c.JSON(500, gin.H{"error": dbErr.Error()})
		return
	}

	if len(preferences) == 0 {
		c.JSON(200, gin.H{"message": "No preferences found for this user"})
		return
	}

	c.JSON(200, gin.H{
		"message":     "User preferences retrieved successfully.",
		"preferences": preferences,
	})
}
