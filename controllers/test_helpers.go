package controllers

import (
	"database/sql"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"
	"github.com/doug-martin/goqu/v9"
	"github.com/gin-gonic/gin"
)

// SetupTestDB creates a mock database and sets it as the global DB for testing
func SetupTestDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}

	// Create goqu database instance
	goquDB := goqu.New("postgres", db)

	// Store original DB to restore after test
	originalDB := initializers.DB
	initializers.DB = goquDB

	// Return cleanup function
	cleanup := func() {
		// Small delay to allow goroutines (like push notifications) to complete
		time.Sleep(10 * time.Millisecond)
		db.Close()
		initializers.DB = originalDB
	}

	return db, mock, cleanup
}

// SetupTestContext creates a test Gin context with a response recorder
func SetupTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

// SetAuthenticatedUser sets the currentUser and admin values in the Gin context
// This simulates what the CheckAuth middleware does
func SetAuthenticatedUser(c *gin.Context, user models.UserProfile, isAdmin bool) {
	c.Set("currentUser", user)
	c.Set("admin", isAdmin)
}
