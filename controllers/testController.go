package controllers

import (
	"net/http"

	"github.com/PrayerLoop/services"
	"github.com/gin-gonic/gin"
)

// TestEmailService sends a test password reset email
// This is for development/testing purposes only
func TestEmailService(c *gin.Context) {
	type TestEmailRequest struct {
		Email     string `json:"email" binding:"required,email"`
		FirstName string `json:"firstName"`
	}

	var req TestEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Valid email is required", "details": err.Error()})
		return
	}

	// Use default first name if not provided
	if req.FirstName == "" {
		req.FirstName = "Test User"
	}

	// Get email service
	emailService := services.GetEmailService()
	if emailService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Email service is not initialized. Check RESEND_API_KEY in .env",
		})
		return
	}

	// Send test email with a sample 6-digit code
	testCode := "123456"
	err := emailService.SendPasswordResetEmail(req.Email, testCode, req.FirstName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send test email",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Test email sent successfully!",
		"email":     req.Email,
		"code":      testCode,
		"firstName": req.FirstName,
		"note":      "Check your inbox for the password reset email with code: " + testCode,
	})
}
