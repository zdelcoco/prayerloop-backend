package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/PrayerLoop/models"
	"github.com/stretchr/testify/assert"
)

// Test ForgotPassword - Initiate password reset flow by sending 6-digit code
func TestForgotPassword(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		userExists     bool
		insertFails    bool
		emailService   bool
		expectedStatus int
		expectError    bool
	}{
		{
			name: "successful request - user exists",
			requestBody: models.ForgotPasswordRequest{
				Email: "test@example.com",
			},
			userExists:     true,
			insertFails:    false,
			emailService:   false, // Email service not available in tests
			expectedStatus: http.StatusInternalServerError,
			expectError:    true, // Will fail at email service unavailable
		},
		{
			name: "user not found - returns success for security",
			requestBody: models.ForgotPasswordRequest{
				Email: "nonexistent@example.com",
			},
			userExists:     false,
			insertFails:    false,
			emailService:   false,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "invalid JSON",
			requestBody:    "{invalid json}",
			userExists:     false,
			insertFails:    false,
			emailService:   false,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "missing email",
			requestBody: map[string]interface{}{
				"notEmail": "test@example.com",
			},
			userExists:     false,
			insertFails:    false,
			emailService:   false,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "database insert fails",
			requestBody: models.ForgotPasswordRequest{
				Email: "test@example.com",
			},
			userExists:     true,
			insertFails:    true,
			emailService:   false,
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			if tt.userExists {
				// Mock user lookup
				now := time.Now()
				userRows := sqlmock.NewRows([]string{
					"user_profile_id", "email", "first_name", "last_name", "password",
					"datetime_create", "datetime_update", "created_by", "updated_by", "admin",
				}).AddRow(1, "test@example.com", "Test", "User", "hashedpassword", now, now, 1, 1, false)
				mock.ExpectQuery("SELECT").WillReturnRows(userRows)

				if tt.insertFails {
					// Mock failed insert
					mock.ExpectExec("INSERT INTO \"password_reset_tokens\"").
						WillReturnError(sqlmock.ErrCancelled)
				} else {
					// Mock successful insert
					mock.ExpectExec("INSERT INTO \"password_reset_tokens\"").
						WillReturnResult(sqlmock.NewResult(1, 1))
				}
			} else {
				// Mock user not found
				mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{
					"user_profile_id", "email", "first_name", "last_name", "password",
					"datetime_create", "datetime_update", "created_by", "updated_by", "admin",
				}))
			}

			c, w := SetupTestContext()

			var jsonData []byte
			if str, ok := tt.requestBody.(string); ok {
				jsonData = []byte(str)
			} else {
				jsonData, _ = json.Marshal(tt.requestBody)
			}

			c.Request = httptest.NewRequest("POST", "/forgot-password", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			ForgotPassword(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectError {
				assert.NotNil(t, response["error"])
			} else {
				assert.NotNil(t, response["message"])
			}
		})
	}
}

// Test VerifyResetCode - Verify 6-digit code and return temporary token
func TestVerifyResetCode(t *testing.T) {
	tests := []struct {
		name            string
		requestBody     interface{}
		userExists      bool
		tokenExists     bool
		tokenExpired    bool
		tokenUsed       bool
		attempts        int
		updateFails     bool
		expectedStatus  int
		expectError     bool
		expectTempToken bool
	}{
		{
			name: "successful verification",
			requestBody: models.VerifyResetCodeRequest{
				Email: "test@example.com",
				Code:  "123456",
			},
			userExists:      true,
			tokenExists:     true,
			tokenExpired:    false,
			tokenUsed:       false,
			attempts:        0,
			updateFails:     false,
			expectedStatus:  http.StatusOK,
			expectError:     false,
			expectTempToken: true,
		},
		{
			name: "user not found",
			requestBody: models.VerifyResetCodeRequest{
				Email: "nonexistent@example.com",
				Code:  "123456",
			},
			userExists:      false,
			tokenExists:     false,
			tokenExpired:    false,
			tokenUsed:       false,
			attempts:        0,
			updateFails:     false,
			expectedStatus:  http.StatusUnauthorized,
			expectError:     true,
			expectTempToken: false,
		},
		{
			name: "token not found",
			requestBody: models.VerifyResetCodeRequest{
				Email: "test@example.com",
				Code:  "999999",
			},
			userExists:      true,
			tokenExists:     false,
			tokenExpired:    false,
			tokenUsed:       false,
			attempts:        0,
			updateFails:     false,
			expectedStatus:  http.StatusUnauthorized,
			expectError:     true,
			expectTempToken: false,
		},
		{
			name: "token expired",
			requestBody: models.VerifyResetCodeRequest{
				Email: "test@example.com",
				Code:  "123456",
			},
			userExists:      true,
			tokenExists:     true,
			tokenExpired:    true,
			tokenUsed:       false,
			attempts:        0,
			updateFails:     false,
			expectedStatus:  http.StatusUnauthorized,
			expectError:     true,
			expectTempToken: false,
		},
		{
			name: "token already used",
			requestBody: models.VerifyResetCodeRequest{
				Email: "test@example.com",
				Code:  "123456",
			},
			userExists:      true,
			tokenExists:     true,
			tokenExpired:    false,
			tokenUsed:       true,
			attempts:        0,
			updateFails:     false,
			expectedStatus:  http.StatusUnauthorized,
			expectError:     true,
			expectTempToken: false,
		},
		{
			name: "max attempts exceeded",
			requestBody: models.VerifyResetCodeRequest{
				Email: "test@example.com",
				Code:  "123456",
			},
			userExists:      true,
			tokenExists:     true,
			tokenExpired:    false,
			tokenUsed:       false,
			attempts:        3,
			updateFails:     false,
			expectedStatus:  http.StatusUnauthorized,
			expectError:     true,
			expectTempToken: false,
		},
		{
			name:            "invalid JSON",
			requestBody:     "{invalid json}",
			userExists:      false,
			tokenExists:     false,
			tokenExpired:    false,
			tokenUsed:       false,
			attempts:        0,
			updateFails:     false,
			expectedStatus:  http.StatusBadRequest,
			expectError:     true,
			expectTempToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			if tt.userExists {
				// Mock user lookup
				now := time.Now()
				userRows := sqlmock.NewRows([]string{
					"user_profile_id", "email", "first_name", "last_name", "password",
					"datetime_create", "datetime_update", "created_by", "updated_by", "admin",
				}).AddRow(1, "test@example.com", "Test", "User", "hashedpassword", now, now, 1, 1, false)
				mock.ExpectQuery("SELECT").WillReturnRows(userRows)

				if tt.tokenExists && !tt.tokenExpired && !tt.tokenUsed {
					// Mock valid token lookup
					expiresAt := time.Now().Add(10 * time.Minute) // Not expired
					tokenRows := sqlmock.NewRows([]string{
						"password_reset_tokens_id", "user_profile_id", "code", "expires_at",
						"used", "attempts", "created_at",
					}).AddRow(1, 1, "123456", expiresAt, false, tt.attempts, now)
					mock.ExpectQuery("SELECT").WillReturnRows(tokenRows)

					if tt.attempts < 3 {
						// Mock attempt count update
						if tt.updateFails {
							mock.ExpectExec("UPDATE \"password_reset_tokens\"").
								WillReturnError(sqlmock.ErrCancelled)
						} else {
							mock.ExpectExec("UPDATE \"password_reset_tokens\"").
								WillReturnResult(sqlmock.NewResult(0, 1))
						}
					}
				} else {
					// Mock token not found, expired, or used
					// This query won't return results due to WHERE conditions
					mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{
						"password_reset_tokens_id", "user_profile_id", "code", "expires_at",
						"used", "attempts", "created_at",
					}))
				}
			} else {
				// Mock user not found
				mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{
					"user_profile_id", "email", "first_name", "last_name", "password",
					"datetime_create", "datetime_update", "created_by", "updated_by", "admin",
				}))
			}

			c, w := SetupTestContext()

			var jsonData []byte
			if str, ok := tt.requestBody.(string); ok {
				jsonData = []byte(str)
			} else {
				jsonData, _ = json.Marshal(tt.requestBody)
			}

			c.Request = httptest.NewRequest("POST", "/verify-reset-code", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			VerifyResetCode(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectError {
				assert.NotNil(t, response["error"])
			} else {
				assert.NotNil(t, response["message"])
				if tt.expectTempToken {
					assert.NotNil(t, response["token"])
					assert.NotNil(t, response["userId"])
				}
			}
		})
	}
}

// Test ResetPassword - Complete password reset using temporary token
func TestResetPassword(t *testing.T) {
	// Create a valid temp token for testing
	validToken, _ := createTempToken(1)
	// Manually create an expired token (>5 minutes old)
	// base64 of "1:0" where timestamp 0 = Jan 1, 1970
	expiredToken := "MTow"

	tests := []struct {
		name           string
		requestBody    interface{}
		token          string
		userExists     bool
		updateFails    bool
		expectedStatus int
		expectError    bool
	}{
		{
			name: "successful password reset",
			requestBody: models.ResetPasswordRequest{
				Token:       validToken,
				NewPassword: "newpassword123",
			},
			token:          validToken,
			userExists:     true,
			updateFails:    false,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "password too short",
			requestBody: models.ResetPasswordRequest{
				Token:       validToken,
				NewPassword: "short",
			},
			token:          validToken,
			userExists:     true,
			updateFails:    false,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid token format",
			requestBody: models.ResetPasswordRequest{
				Token:       "invalid-token",
				NewPassword: "newpassword123",
			},
			token:          "invalid-token",
			userExists:     false,
			updateFails:    false,
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name: "expired token",
			requestBody: models.ResetPasswordRequest{
				Token:       expiredToken,
				NewPassword: "newpassword123",
			},
			token:          expiredToken,
			userExists:     false,
			updateFails:    false,
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name: "user not found",
			requestBody: models.ResetPasswordRequest{
				Token:       validToken,
				NewPassword: "newpassword123",
			},
			token:          validToken,
			userExists:     false,
			updateFails:    false,
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name: "database update fails",
			requestBody: models.ResetPasswordRequest{
				Token:       validToken,
				NewPassword: "newpassword123",
			},
			token:          validToken,
			userExists:     true,
			updateFails:    true,
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name:           "invalid JSON",
			requestBody:    "{invalid json}",
			token:          "",
			userExists:     false,
			updateFails:    false,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "missing token",
			requestBody: map[string]interface{}{
				"newPassword": "newpassword123",
			},
			token:          "",
			userExists:     false,
			updateFails:    false,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			// Only mock database operations for valid tokens
			if tt.token == validToken && tt.userExists {
				// Mock user lookup
				now := time.Now()
				userRows := sqlmock.NewRows([]string{
					"user_profile_id", "email", "first_name", "last_name", "password",
					"datetime_create", "datetime_update", "created_by", "updated_by", "admin",
				}).AddRow(1, "test@example.com", "Test", "User", "hashedpassword", now, now, 1, 1, false)
				mock.ExpectQuery("SELECT").WillReturnRows(userRows)

				if tt.updateFails {
					// Mock failed password update
					mock.ExpectExec("UPDATE \"user_profile\"").
						WillReturnError(sqlmock.ErrCancelled)
				} else {
					// Mock successful password update
					mock.ExpectExec("UPDATE \"user_profile\"").
						WillReturnResult(sqlmock.NewResult(0, 1))

					// Mock marking tokens as used
					mock.ExpectExec("UPDATE \"password_reset_tokens\"").
						WillReturnResult(sqlmock.NewResult(0, 1))
				}
			} else if tt.token == validToken && !tt.userExists {
				// Mock user not found
				mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{
					"user_profile_id", "email", "first_name", "last_name", "password",
					"datetime_create", "datetime_update", "created_by", "updated_by", "admin",
				}))
			}

			c, w := SetupTestContext()

			var jsonData []byte
			if str, ok := tt.requestBody.(string); ok {
				jsonData = []byte(str)
			} else {
				jsonData, _ = json.Marshal(tt.requestBody)
			}

			c.Request = httptest.NewRequest("POST", "/reset-password", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			ResetPassword(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectError {
				assert.NotNil(t, response["error"])
			} else {
				assert.NotNil(t, response["message"])
			}
		})
	}
}
