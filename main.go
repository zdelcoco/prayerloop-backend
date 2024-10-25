package main

import (
	"github.com/gin-gonic/gin"

	"github.com/PrayerLoop/controllers"
	"github.com/PrayerLoop/initializers"
)

func init() {
	initializers.LoadEnv()
}

func main() {

	router := gin.Default()

	router.GET("/ping", controllers.Ping)

	router.Run()

	/* comment added ci/cd testing */

}
