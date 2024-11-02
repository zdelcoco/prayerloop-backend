package main

import (
	"github.com/gin-gonic/gin"

	"github.com/PrayerLoop/controllers"
	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/middlewares"
)

func init() {
	initializers.LoadEnv()
	initializers.ConnectDB()
}

func main() {

	router := gin.Default()

	router.GET("/ping", controllers.Ping)

	router.GET("/user/profile", middlewares.CheckAuth, controllers.GetUserProfile)
	router.POST("/user/signup", middlewares.CheckAuth, controllers.UserSignup)
	router.POST("/user/login", controllers.UserLogin)

	router.Run()

}
