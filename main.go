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
	router.POST("/login", controllers.UserLogin)

	auth := router.Group("/")
	auth.Use(middlewares.CheckAuth)
	{
		// User signup route is currently only available to admins
		auth.POST("/users", controllers.UserSignup)

		auth.GET("/users/me", controllers.GetUserProfile)
		auth.GET("/users/:id/groups", controllers.GetUserGroups)

		// prayer-request routes
		auth.POST("/prayer-requests", controllers.CreatePrayerRequest)
		auth.GET("/prayer-requests/:id", controllers.GetPrayerRequest)
		auth.GET("/prayer-requests", controllers.GetPrayerRequests)
		auth.PUT("/prayer-requests/:id", controllers.UpdatePrayerRequest)
		auth.DELETE("/prayer-requests/:id", controllers.DeletePrayerRequest)

		// group routes
		auth.POST("/groups", controllers.CreateGroup)
		auth.GET("/groups/:group_id", controllers.GetGroup)
		auth.GET("/groups", controllers.GetAllGroups)
		auth.PUT("/groups/:group_id", controllers.UpdateGroup)
		auth.DELETE("/groups/:group_id", controllers.DeleteGroup)

		auth.GET("/groups/:group_id/users", controllers.GetGroupUsers)
		auth.POST("/groups/:group_id/users/:user_id", controllers.AddUserToGroup)
		auth.DELETE("/groups/:group_id/users/:user_id", controllers.RemoveUserFromGroup)

	}

	router.Run()
}
