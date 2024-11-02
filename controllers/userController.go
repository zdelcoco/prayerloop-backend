package controllers

import (
	"log"
	"net/http"

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
	var user models.UserSignup

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userCount, err := initializers.DB.From("user").Select("username").Where(goqu.C("username").Eq(user.Username)).Count()
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

	newUser := models.User{
		Username:   user.Username,
		Password:   string(passwordHash),
		Email:      user.Email,
		First_Name: user.FirstName,
		Last_Name:  user.LastName,
		Created_By: 1,
		Updated_By: 1,
	}

	insert := initializers.DB.Insert("user").Rows(newUser).Executor()
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

	var dbUser models.User
	_, err := initializers.DB.From("user").Select("*").Where(goqu.C("username").Eq(user.Username)).ScanStruct(&dbUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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
		"id":   dbUser.User_ID,
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
