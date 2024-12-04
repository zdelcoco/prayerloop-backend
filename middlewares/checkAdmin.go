package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CheckAdmin(c *gin.Context) {
	isAdmin := c.MustGet("admin").(bool)

	if !isAdmin {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
}
