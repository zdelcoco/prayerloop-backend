package controllers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

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
