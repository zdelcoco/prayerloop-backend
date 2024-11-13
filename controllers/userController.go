package controllers

import (
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to generate token"})
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

	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to construct query"})
		return
	}

	log.Println(sql, args)

	var groups []models.GroupProfile
	err = initializers.DB.ScanStructs(&groups, sql, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user groups"})
		return
	}

	if len(groups) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No groups found for this user"})
		return
	}

	c.JSON(http.StatusOK, groups)
}
