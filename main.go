package main

import (
	"github.com/gin-gonic/gin"

	"github.com/PrayerLoop/controllers"
	"github.com/PrayerLoop/initializers"
)

func init() {
	initializers.LoadEnv()
	initializers.ConnectDB()
}

func main() {

	router := gin.Default()

	router.GET("/ping", controllers.Ping)
	router.POST("/user/signup", controllers.UserSignup)

	router.Run()

}
