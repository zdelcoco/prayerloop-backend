package models

import "time"

type PushToken struct {
	UserPushTokenID int       `json:"userPushTokenId" db:"user_push_tokens_id"`
	UserProfileID   int       `json:"userProfileId" db:"user_profile_id"`
	PushToken       string    `json:"pushToken" db:"push_token"`
	Platform        string    `json:"platform" db:"platform"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time `json:"updatedAt" db:"updated_at"`
}

type PushTokenRequest struct {
	PushToken string `json:"pushToken" binding:"required"`
	Platform  string `json:"platform" binding:"required,oneof=ios android"`
}
