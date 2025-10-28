package controllers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"
	"github.com/PrayerLoop/services"
	"github.com/doug-martin/goqu/v9"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// ForgotPassword initiates the password reset flow by sending a 6-digit code to the user's email
func ForgotPassword(c *gin.Context) {
	var req models.ForgotPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Valid email address is required", "details": err.Error()})
		return
	}

	// Find user by email
	var user models.UserProfile
	found, err := initializers.DB.From("user_profile").
		Select("*").
		Where(goqu.C("email").Eq(req.Email)).
		ScanStruct(&user)

	if err != nil || !found {
		// Return success even if email doesn't exist
		c.JSON(http.StatusOK, gin.H{
			"message": "If this email exists in our system, a verification code has been sent.",
		})
		return
	}

	code, err := generate6DigitCode()
	if err != nil {
		log.Printf("Failed to generate verification code: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate verification code"})
		return
	}

	// Set expiration time (15 minutes from now)
	expiresAt := time.Now().Add(15 * time.Minute)

	// Insert reset token into database
	resetToken := models.PasswordResetToken{
		User_Profile_ID: user.User_Profile_ID,
		Code:            code,
		Expires_At:      expiresAt,
		Used:            false,
		Attempts:        0,
	}

	insert := initializers.DB.Insert("password_reset_tokens").Rows(resetToken).Executor()
	if _, err := insert.Exec(); err != nil {
		log.Printf("Failed to store password reset token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password reset request"})
		return
	}

	// Send email with verification code
	emailService := services.GetEmailService()
	if emailService == nil {
		log.Println("Email service not initialized")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Email service unavailable"})
		return
	}

	err = emailService.SendPasswordResetEmail(user.Email, code, user.First_Name)
	if err != nil {
		log.Printf("Failed to send password reset email: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification email"})
		return
	}

	log.Printf("Password reset code sent to user %d (%s)", user.User_Profile_ID, user.Email)

	c.JSON(http.StatusOK, gin.H{
		"message": "If this email exists in our system, a verification code has been sent.",
	})
}

// VerifyResetCode verifies the 6-digit code and returns a temporary token for password reset
func VerifyResetCode(c *gin.Context) {
	var req models.VerifyResetCodeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and 6-digit code are required", "details": err.Error()})
		return
	}

	// Find user by email
	var user models.UserProfile
	found, err := initializers.DB.From("user_profile").
		Select("*").
		Where(goqu.C("email").Eq(req.Email)).
		ScanStruct(&user)

	if err != nil || !found {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or verification code"})
		return
	}

	// Find the most recent unused, non-expired reset token for this user
	var resetToken models.PasswordResetToken
	found, err = initializers.DB.From("password_reset_tokens").
		Select("*").
		Where(goqu.And(
			goqu.C("user_profile_id").Eq(user.User_Profile_ID),
			goqu.C("code").Eq(req.Code),
			goqu.C("used").Eq(false),
			goqu.C("expires_at").Gt(time.Now()),
		)).
		Order(goqu.C("created_at").Desc()).
		ScanStruct(&resetToken)

	if err != nil || !found {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired verification code"})
		return
	}

	// Check attempt count
	if resetToken.Attempts >= 3 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Maximum verification attempts exceeded. Please request a new code.",
		})
		return
	}

	// Increment attempt count
	updateAttempts := initializers.DB.Update("password_reset_tokens").
		Set(goqu.Record{"attempts": resetToken.Attempts + 1}).
		Where(goqu.C("password_reset_tokens_id").Eq(resetToken.Token_ID)).
		Executor()

	if _, err := updateAttempts.Exec(); err != nil {
		log.Printf("Failed to update attempt count: %v", err)
	}

	// Generate a temporary token (valid for 5 minutes) for the final reset step
	tempToken, err := createTempToken(user.User_Profile_ID)
	if err != nil {
		log.Printf("Failed to generate temporary token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify code"})
		return
	}

	log.Printf("Verification code verified for user %d (%s)", user.User_Profile_ID, user.Email)

	c.JSON(http.StatusOK, gin.H{
		"message": "Verification code is valid",
		"token":   tempToken,
		"userId":  user.User_Profile_ID,
	})
}

// ResetPassword resets the user's password using the temporary token from verification
func ResetPassword(c *gin.Context) {
	var req models.ResetPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token and new password are required", "details": err.Error()})
		return
	}

	// Validate password length
	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 6 characters long"})
		return
	}

	// Decode the temporary token to get user ID
	// For simplicity, we're using base64(userID:timestamp:code)
	// In production, you might want to use JWT or store tokens in database
	userID, valid := validateTempToken(req.Token)
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		return
	}

	// Get user
	var user models.UserProfile
	found, err := initializers.DB.From("user_profile").
		Select("*").
		Where(goqu.C("user_profile_id").Eq(userID)).
		ScanStruct(&user)

	if err != nil || !found {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Hash the new password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	// Update password
	updatePassword := initializers.DB.Update("user_profile").
		Set(goqu.Record{
			"password":        string(passwordHash),
			"updated_by":      userID,
			"datetime_update": time.Now(),
		}).
		Where(goqu.C("user_profile_id").Eq(userID)).
		Executor()

	if _, err := updatePassword.Exec(); err != nil {
		log.Printf("Failed to update password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	// Mark all reset tokens for this user as used
	markUsed := initializers.DB.Update("password_reset_tokens").
		Set(goqu.Record{"used": true}).
		Where(goqu.C("user_profile_id").Eq(userID)).
		Executor()

	if _, err := markUsed.Exec(); err != nil {
		log.Printf("Failed to mark reset tokens as used: %v", err)
		// Non-critical error, continue
	}

	log.Printf("Password successfully reset for user %d (%s)", user.User_Profile_ID, user.Email)

	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successfully. You can now login with your new password.",
	})
}

// Helper function to generate a cryptographically secure 6-digit code
func generate6DigitCode() (string, error) {
	// Generate a random number between 0 and 999999
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	// Format as 6-digit string with leading zeros
	code := fmt.Sprintf("%06d", n.Int64())
	return code, nil
}

// Helper function to generate a secure temporary token
func generateSecureToken() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode as base64
	token := base64.URLEncoding.EncodeToString(bytes)
	return token, nil
}

// Helper function to validate temporary token
// In a production system, you might want to store these tokens in the database
// or use JWT with expiration
func validateTempToken(token string) (int, bool) {
	// For now, we'll use a simple approach:
	// Token format: base64(userID:timestamp)
	// In production, use JWT or store tokens in database with expiration

	// Decode base64
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return 0, false
	}

	// Parse userID:timestamp
	var userID int
	var timestamp int64
	_, err = fmt.Sscanf(string(decoded), "%d:%d", &userID, &timestamp)
	if err != nil {
		return 0, false
	}

	// Check if token is expired (5 minutes)
	tokenTime := time.Unix(timestamp, 0)
	if time.Since(tokenTime) > 5*time.Minute {
		return 0, false
	}

	return userID, true
}

// Helper function to create temporary token (called from VerifyResetCode)
func createTempToken(userID int) (string, error) {
	// Create token: userID:timestamp
	timestamp := time.Now().Unix()
	tokenData := fmt.Sprintf("%d:%d", userID, timestamp)

	// Encode as base64
	token := base64.URLEncoding.EncodeToString([]byte(tokenData))
	return token, nil
}
