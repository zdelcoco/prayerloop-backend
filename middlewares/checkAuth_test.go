package middlewares

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"
	"github.com/doug-martin/goqu/v9"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

// Helper function to generate a valid JWT token
func generateValidToken(userID int, role string, expiresIn time.Duration) string {
	secret := os.Getenv("SECRET")
	if secret == "" {
		secret = "test-secret-key"
		os.Setenv("SECRET", secret)
	}

	claims := jwt.MapClaims{
		"id":   float64(userID),
		"exp":  float64(time.Now().Add(expiresIn).Unix()),
		"role": role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

// Helper function to generate a token without role claim
func generateTokenWithoutRole(userID int, expiresIn time.Duration) string {
	secret := os.Getenv("SECRET")
	if secret == "" {
		secret = "test-secret-key"
		os.Setenv("SECRET", secret)
	}

	claims := jwt.MapClaims{
		"id":  float64(userID),
		"exp": float64(time.Now().Add(expiresIn).Unix()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

// Helper function to generate an expired token
func generateExpiredToken(userID int) string {
	return generateValidToken(userID, "user", -1*time.Hour) // Expired 1 hour ago
}

// Helper function to generate a token with invalid signature
func generateInvalidSignatureToken(userID int) string {
	claims := jwt.MapClaims{
		"id":   float64(userID),
		"exp":  float64(time.Now().Add(24 * time.Hour).Unix()),
		"role": "user",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte("wrong-secret-key"))
	return tokenString
}

// Setup test database
func setupTestDB(t *testing.T) (sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	// Create goqu database instance
	goquDB := goqu.New("postgres", db)

	// Replace the global DB connection with our mock
	oldDB := initializers.DB
	initializers.DB = goquDB

	cleanup := func() {
		db.Close()
		initializers.DB = oldDB
	}

	return mock, cleanup
}

// Setup test Gin context
func setupTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	return c, w
}

// Test CheckAuth middleware
func TestCheckAuth(t *testing.T) {
	tests := []struct {
		name               string
		authHeader         string
		mockUserLookup     bool
		userExists         bool
		expectedStatus     int
		expectAbort        bool
		expectCurrentUser  bool
		expectAdmin        bool
		adminRole          bool
	}{
		{
			name:               "missing authorization header",
			authHeader:         "",
			mockUserLookup:     false,
			userExists:         false,
			expectedStatus:     http.StatusUnauthorized,
			expectAbort:        true,
			expectCurrentUser:  false,
			expectAdmin:        false,
			adminRole:          false,
		},
		{
			name:               "invalid token format - no Bearer prefix",
			authHeader:         "InvalidToken123",
			mockUserLookup:     false,
			userExists:         false,
			expectedStatus:     http.StatusUnauthorized,
			expectAbort:        true,
			expectCurrentUser:  false,
			expectAdmin:        false,
			adminRole:          false,
		},
		{
			name:               "invalid token format - wrong prefix",
			authHeader:         "Basic " + generateValidToken(1, "user", 24*time.Hour),
			mockUserLookup:     false,
			userExists:         false,
			expectedStatus:     http.StatusUnauthorized,
			expectAbort:        true,
			expectCurrentUser:  false,
			expectAdmin:        false,
			adminRole:          false,
		},
		{
			name:               "invalid JWT signature",
			authHeader:         "Bearer " + generateInvalidSignatureToken(1),
			mockUserLookup:     false,
			userExists:         false,
			expectedStatus:     http.StatusUnauthorized,
			expectAbort:        true,
			expectCurrentUser:  false,
			expectAdmin:        false,
			adminRole:          false,
		},
		{
			name:               "expired token",
			authHeader:         "Bearer " + generateExpiredToken(1),
			mockUserLookup:     false,
			userExists:         false,
			expectedStatus:     http.StatusUnauthorized,
			expectAbort:        true,
			expectCurrentUser:  false,
			expectAdmin:        false,
			adminRole:          false,
		},
		{
			name:               "valid token - user not found in database",
			authHeader:         "Bearer " + generateValidToken(999, "user", 24*time.Hour),
			mockUserLookup:     true,
			userExists:         false,
			expectedStatus:     http.StatusUnauthorized,
			expectAbort:        true,
			expectCurrentUser:  false,
			expectAdmin:        false,
			adminRole:          false,
		},
		{
			name:               "valid token - regular user",
			authHeader:         "Bearer " + generateValidToken(1, "user", 24*time.Hour),
			mockUserLookup:     true,
			userExists:         true,
			expectedStatus:     http.StatusOK,
			expectAbort:        false,
			expectCurrentUser:  true,
			expectAdmin:        true,
			adminRole:          false,
		},
		{
			name:               "valid token - admin user",
			authHeader:         "Bearer " + generateValidToken(2, "admin", 24*time.Hour),
			mockUserLookup:     true,
			userExists:         true,
			expectedStatus:     http.StatusOK,
			expectAbort:        false,
			expectCurrentUser:  true,
			expectAdmin:        true,
			adminRole:          true,
		},
		{
			name:               "valid token - no role claim (defaults to non-admin)",
			authHeader:         "Bearer " + generateTokenWithoutRole(1, 24*time.Hour),
			mockUserLookup:     true,
			userExists:         true,
			expectedStatus:     http.StatusOK,
			expectAbort:        false,
			expectCurrentUser:  true,
			expectAdmin:        true,
			adminRole:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, cleanup := setupTestDB(t)
			defer cleanup()

			// Mock database user lookup if needed
			if tt.mockUserLookup {
				now := time.Now()
				if tt.userExists {
					userRows := sqlmock.NewRows([]string{
						"user_profile_id", "email", "first_name", "last_name", "password",
						"datetime_create", "datetime_update", "created_by", "updated_by", "admin",
					})

					if tt.adminRole {
						userRows.AddRow(2, "admin@example.com", "Admin", "User", "hashedpassword", now, now, 2, 2, true)
					} else {
						userRows.AddRow(1, "test@example.com", "Test", "User", "hashedpassword", now, now, 1, 1, false)
					}

					mock.ExpectQuery("SELECT").WillReturnRows(userRows)
				} else {
					// User not found
					mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{
						"user_profile_id", "email", "first_name", "last_name", "password",
						"datetime_create", "datetime_update", "created_by", "updated_by", "admin",
					}))
				}
			}

			c, w := setupTestContext()

			// Set authorization header
			if tt.authHeader != "" {
				c.Request.Header.Set("Authorization", tt.authHeader)
			}

			CheckAuth(c)

			// Verify response status and abort status
			if tt.expectAbort {
				assert.True(t, c.IsAborted(), "Expected request to be aborted")
				assert.Equal(t, tt.expectedStatus, w.Code)
			} else {
				assert.False(t, c.IsAborted(), "Expected request not to be aborted")
			}

			// Verify currentUser was set
			if tt.expectCurrentUser {
				user, exists := c.Get("currentUser")
				assert.True(t, exists, "Expected currentUser to be set")
				assert.NotNil(t, user)

				userProfile := user.(models.UserProfile)
				if tt.adminRole {
					assert.Equal(t, 2, userProfile.User_Profile_ID)
					assert.Equal(t, "admin@example.com", userProfile.Email)
				} else {
					assert.Equal(t, 1, userProfile.User_Profile_ID)
					assert.Equal(t, "test@example.com", userProfile.Email)
				}
			} else {
				_, exists := c.Get("currentUser")
				assert.False(t, exists, "Expected currentUser not to be set")
			}

			// Verify admin flag was set
			if tt.expectAdmin {
				admin, exists := c.Get("admin")
				assert.True(t, exists, "Expected admin to be set")
				assert.Equal(t, tt.adminRole, admin.(bool))
			} else {
				_, exists := c.Get("admin")
				assert.False(t, exists, "Expected admin not to be set")
			}
		})
	}
}
