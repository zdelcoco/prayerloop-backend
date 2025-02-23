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
		auth.GET("/users/:user_profile_id/groups", controllers.GetUserGroups)

		auth.GET("/users/:user_profile_id/prayers", controllers.GetUserPrayers)
		auth.POST("/users/:user_profile_id/prayers", controllers.CreateUserPrayer)

		// notification routes
		auth.GET("/users/:user_profile_id/notifications", controllers.GetUserNotifications)
		auth.PATCH("/users/:user_profile_id/notifications/:notification_id", controllers.ToggleUserNotificationStatus)

		// group routes
		auth.GET("/groups", controllers.GetAllGroups)
		auth.POST("/groups", controllers.CreateGroup)
		auth.GET("/groups/:group_profile_id", controllers.GetGroup)
		auth.PUT("/groups/:group_profile_id", controllers.UpdateGroup)
		auth.DELETE("/groups/:group_profile_id", controllers.DeleteGroup)

		auth.GET("/groups/:group_profile_id/prayers", controllers.GetGroupPrayers)
		auth.POST("/groups/:group_profile_id/prayers", controllers.CreateGroupPrayer)

		auth.GET("/groups/:group_profile_id/users", controllers.GetGroupUsers)
		auth.POST("/groups/:group_profile_id/users/:user_profile_id", controllers.AddUserToGroup)
		auth.DELETE("/groups/:group_profile_id/users/:user_profile_id", controllers.RemoveUserFromGroup)

		// group invite
		auth.POST("/groups/:group_profile_id/invite", controllers.CreateGroupInviteCode)

		// prayer routes
		auth.PUT("/prayers/:prayer_id", controllers.UpdatePrayer)
		auth.DELETE("/prayers/:prayer_id", controllers.DeletePrayer)
		auth.POST("/prayers/:prayer_id/access", controllers.AddPrayerAccess)
		auth.DELETE("/prayers/:prayer_id/access/:prayer_access_id", controllers.RemovePrayerAccess)

		//admin only routes
		admin := auth.Group("/")
		admin.Use(middlewares.CheckAdmin)
		{
			admin.GET("/prayers", controllers.GetPrayers)
			admin.GET("/prayers/:prayer_id", controllers.GetPrayer)
		}
	}

	router.Run()
}
