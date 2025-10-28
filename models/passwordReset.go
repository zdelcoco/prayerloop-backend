package models

import "time"

type PasswordResetToken struct {
	Token_ID        int       `json:"tokenId" db:"password_reset_tokens_id" goqu:"skipinsert"`
	User_Profile_ID int       `json:"userProfileId" db:"user_profile_id"`
	Code            string    `json:"code" db:"code"`
	Expires_At      time.Time `json:"expiresAt" db:"expires_at"`
	Used            bool      `json:"used" db:"used"`
	Attempts        int       `json:"attempts" db:"attempts"`
	Created_At      time.Time `json:"createdAt" db:"created_at" goqu:"skipinsert"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type VerifyResetCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}
