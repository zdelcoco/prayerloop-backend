package controllers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/PrayerLoop/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Test GetUserNotifications - Fetch notifications with authorization
func TestGetUserNotifications(t *testing.T) {
	tests := []struct {
		name              string
		userID            string
		currentUser       models.UserProfile
		isAdmin           bool
		hasNotifications  bool
		expectedStatus    int
		expectError       bool
	}{
		{
			name:              "successful fetch - own notifications",
			userID:            "1",
			currentUser:       MockUser(),
			isAdmin:           false,
			hasNotifications:  true,
			expectedStatus:    http.StatusOK,
			expectError:       false,
		},
		{
			name:              "successful fetch - admin views other user",
			userID:            "2",
			currentUser:       MockAdminUser(),
			isAdmin:           true,
			hasNotifications:  true,
			expectedStatus:    http.StatusOK,
			expectError:       false,
		},
		{
			name:              "no notifications found",
			userID:            "1",
			currentUser:       MockUser(),
			isAdmin:           false,
			hasNotifications:  false,
			expectedStatus:    http.StatusOK,
			expectError:       false,
		},
		{
			name:              "forbidden - view other user's notifications",
			userID:            "2",
			currentUser:       MockUser(),
			isAdmin:           false,
			hasNotifications:  false,
			expectedStatus:    http.StatusForbidden,
			expectError:       true,
		},
		{
			name:              "invalid user ID",
			userID:            "invalid",
			currentUser:       MockUser(),
			isAdmin:           false,
			hasNotifications:  false,
			expectedStatus:    http.StatusBadRequest,
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			if tt.userID != "invalid" && (tt.currentUser.User_Profile_ID == 1 || tt.isAdmin) {
				now := time.Now()

				if tt.hasNotifications {
					// Mock notifications fetch with results
					notificationRows := sqlmock.NewRows([]string{
						"notification_id", "user_profile_id", "notification_type", "notification_message",
						"notification_status", "datetime_create", "datetime_update", "created_by", "updated_by",
					}).AddRow(1, 1, "PRAYER_SHARED", "Someone shared a prayer with you", "UNREAD", now, now, 1, 1)
					mock.ExpectQuery("SELECT").WillReturnRows(notificationRows)
				} else {
					// Mock empty result
					mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{
						"notification_id", "user_profile_id", "notification_type", "notification_message",
						"notification_status", "datetime_create", "datetime_update", "created_by", "updated_by",
					}))
				}
			}

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, tt.currentUser, tt.isAdmin)
			c.Params = []gin.Param{{Key: "user_profile_id", Value: tt.userID}}
			c.Request = httptest.NewRequest("GET", "/users/"+tt.userID+"/notifications", nil)

			GetUserNotifications(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectError {
				var response map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &response)
				assert.NotNil(t, response["error"])
			} else {
				var notifications []interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &notifications)
				if tt.hasNotifications {
					assert.Greater(t, len(notifications), 0)
				} else {
					assert.Equal(t, 0, len(notifications))
				}
			}
		})
	}
}

// Test ToggleUserNotificationStatus - Toggle notification READ/UNREAD status
func TestToggleUserNotificationStatus(t *testing.T) {
	tests := []struct {
		name             string
		userID           string
		notificationID   string
		currentUser      models.UserProfile
		isAdmin          bool
		currentStatus    string
		notificationExists bool
		expectedStatus   int
		expectError      bool
		expectedNewStatus string
	}{
		{
			name:             "successful toggle - UNREAD to READ",
			userID:           "1",
			notificationID:   "1",
			currentUser:      MockUser(),
			isAdmin:          false,
			currentStatus:    "UNREAD",
			notificationExists: true,
			expectedStatus:   http.StatusOK,
			expectError:      false,
			expectedNewStatus: "READ",
		},
		{
			name:             "successful toggle - READ to UNREAD",
			userID:           "1",
			notificationID:   "1",
			currentUser:      MockUser(),
			isAdmin:          false,
			currentStatus:    "READ",
			notificationExists: true,
			expectedStatus:   http.StatusOK,
			expectError:      false,
			expectedNewStatus: "UNREAD",
		},
		{
			name:             "successful toggle - admin for other user",
			userID:           "2",
			notificationID:   "1",
			currentUser:      MockAdminUser(),
			isAdmin:          true,
			currentStatus:    "UNREAD",
			notificationExists: true,
			expectedStatus:   http.StatusOK,
			expectError:      false,
			expectedNewStatus: "READ",
		},
		{
			name:             "notification not found",
			userID:           "1",
			notificationID:   "999",
			currentUser:      MockUser(),
			isAdmin:          false,
			currentStatus:    "UNREAD",
			notificationExists: false,
			expectedStatus:   http.StatusNotFound,
			expectError:      true,
			expectedNewStatus: "",
		},
		{
			name:             "forbidden - modify other user's notification",
			userID:           "2",
			notificationID:   "1",
			currentUser:      MockUser(),
			isAdmin:          false,
			currentStatus:    "UNREAD",
			notificationExists: false,
			expectedStatus:   http.StatusForbidden,
			expectError:      true,
			expectedNewStatus: "",
		},
		{
			name:             "invalid user ID",
			userID:           "invalid",
			notificationID:   "1",
			currentUser:      MockUser(),
			isAdmin:          false,
			currentStatus:    "",
			notificationExists: false,
			expectedStatus:   http.StatusBadRequest,
			expectError:      true,
			expectedNewStatus: "",
		},
		{
			name:             "invalid notification ID",
			userID:           "1",
			notificationID:   "invalid",
			currentUser:      MockUser(),
			isAdmin:          false,
			currentStatus:    "",
			notificationExists: false,
			expectedStatus:   http.StatusBadRequest,
			expectError:      true,
			expectedNewStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			// Only mock database operations for valid IDs and authorized users
			if tt.userID != "invalid" && tt.notificationID != "invalid" &&
			   (tt.currentUser.User_Profile_ID == 1 || tt.isAdmin) {

				if tt.notificationExists {
					// Mock current status fetch
					statusRows := sqlmock.NewRows([]string{"notification_status"}).AddRow(tt.currentStatus)
					mock.ExpectQuery("SELECT").WillReturnRows(statusRows)

					// Mock update
					mock.ExpectExec("UPDATE \"notification\"").
						WillReturnResult(sqlmock.NewResult(0, 1))
				} else {
					// Mock current status fetch (empty)
					mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"notification_status"}))

					// Mock update returning 0 rows
					mock.ExpectExec("UPDATE \"notification\"").
						WillReturnResult(sqlmock.NewResult(0, 0))
				}
			}

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, tt.currentUser, tt.isAdmin)
			c.Params = []gin.Param{
				{Key: "user_profile_id", Value: tt.userID},
				{Key: "notification_id", Value: tt.notificationID},
			}
			c.Request = httptest.NewRequest("PATCH", "/users/"+tt.userID+"/notifications/"+tt.notificationID, nil)

			ToggleUserNotificationStatus(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectError {
				assert.NotNil(t, response["error"])
			} else {
				assert.NotNil(t, response["message"])
				assert.Contains(t, response["message"], tt.expectedNewStatus)
			}
		})
	}
}

// Test SendPushNotification - Send push notifications to users
func TestSendPushNotification(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectError    bool
	}{
		{
			name: "service unavailable - push service not initialized",
			requestBody: SendNotificationRequest{
				UserIDs:  []int{1},
				Title:    "Test Notification",
				Body:     "This is a test notification",
				Priority: "high",
			},
			expectedStatus: http.StatusServiceUnavailable, // Service not available in test environment
			expectError:    true,
		},
		{
			name: "service unavailable - multiple users",
			requestBody: SendNotificationRequest{
				UserIDs:  []int{1, 2, 3},
				Title:    "Test Notification",
				Body:     "This is a test notification",
				Priority: "normal",
			},
			expectedStatus: http.StatusServiceUnavailable, // Service not available in test environment
			expectError:    true,
		},
		{
			name: "missing required field - userIds",
			requestBody: map[string]interface{}{
				"title": "Test Notification",
				"body":  "This is a test notification",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "missing required field - title",
			requestBody: map[string]interface{}{
				"userIds": []int{1},
				"body":    "This is a test notification",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "missing required field - body",
			requestBody: map[string]interface{}{
				"userIds": []int{1},
				"title":   "Test Notification",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "invalid JSON",
			requestBody:    "{invalid json}",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := SetupTestContext()

			var jsonData []byte
			if str, ok := tt.requestBody.(string); ok {
				jsonData = []byte(str)
			} else {
				jsonData, _ = json.Marshal(tt.requestBody)
			}

			c.Request = httptest.NewRequest("POST", "/notifications/send", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			SendPushNotification(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectError {
				assert.NotNil(t, response["error"])
			} else {
				assert.NotNil(t, response["message"])
				assert.NotNil(t, response["userIds"])
			}
		})
	}
}

// Test DeleteUserNotification - Delete a notification
func TestDeleteUserNotification(t *testing.T) {
	tests := []struct {
		name                  string
		userID                string
		notificationID        string
		currentUser           models.UserProfile
		isAdmin               bool
		notificationExists    bool
		notificationBelongsToUser bool
		expectedStatus        int
		expectError           bool
	}{
		{
			name:                  "successful delete - own notification",
			userID:                "1",
			notificationID:        "1",
			currentUser:           MockUser(),
			isAdmin:               false,
			notificationExists:    true,
			notificationBelongsToUser: true,
			expectedStatus:        http.StatusOK,
			expectError:           false,
		},
		{
			name:                  "successful delete - admin deletes other user's notification",
			userID:                "2",
			notificationID:        "1",
			currentUser:           MockAdminUser(),
			isAdmin:               true,
			notificationExists:    true,
			notificationBelongsToUser: true,
			expectedStatus:        http.StatusOK,
			expectError:           false,
		},
		{
			name:                  "notification not found",
			userID:                "1",
			notificationID:        "999",
			currentUser:           MockUser(),
			isAdmin:               false,
			notificationExists:    false,
			notificationBelongsToUser: false,
			expectedStatus:        http.StatusNotFound,
			expectError:           true,
		},
		{
			name:                  "notification belongs to different user",
			userID:                "1",
			notificationID:        "1",
			currentUser:           MockUser(),
			isAdmin:               false,
			notificationExists:    true,
			notificationBelongsToUser: false,
			expectedStatus:        http.StatusForbidden,
			expectError:           true,
		},
		{
			name:                  "forbidden - delete other user's notification",
			userID:                "2",
			notificationID:        "1",
			currentUser:           MockUser(),
			isAdmin:               false,
			notificationExists:    false,
			notificationBelongsToUser: false,
			expectedStatus:        http.StatusForbidden,
			expectError:           true,
		},
		{
			name:                  "invalid user ID",
			userID:                "invalid",
			notificationID:        "1",
			currentUser:           MockUser(),
			isAdmin:               false,
			notificationExists:    false,
			notificationBelongsToUser: false,
			expectedStatus:        http.StatusBadRequest,
			expectError:           true,
		},
		{
			name:                  "invalid notification ID",
			userID:                "1",
			notificationID:        "invalid",
			currentUser:           MockUser(),
			isAdmin:               false,
			notificationExists:    false,
			notificationBelongsToUser: false,
			expectedStatus:        http.StatusBadRequest,
			expectError:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			// Only mock database operations for valid IDs and authorized users
			if tt.userID != "invalid" && tt.notificationID != "invalid" {
				// Only set up mocks if user has permission to attempt the operation
				if tt.currentUser.User_Profile_ID == 1 || tt.isAdmin {
					if tt.notificationExists {
						// Mock notification ownership check
						// Parse userID to determine which user the notification belongs to
						userIDInt := 1
						if tt.userID == "2" {
							userIDInt = 2
						}

						userIDToReturn := userIDInt
						if !tt.notificationBelongsToUser {
							userIDToReturn = 999 // Different user
						}
						ownershipRows := sqlmock.NewRows([]string{"user_profile_id"}).AddRow(userIDToReturn)
						mock.ExpectQuery("SELECT").WillReturnRows(ownershipRows)

						if tt.notificationBelongsToUser {
							// Mock delete
							mock.ExpectExec("DELETE FROM \"notification\"").
								WillReturnResult(sqlmock.NewResult(0, 1))
						}
					} else {
						// Mock ownership check (not found) - return error to simulate no rows found
						mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)
					}
				}
			}

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, tt.currentUser, tt.isAdmin)
			c.Params = []gin.Param{
				{Key: "user_profile_id", Value: tt.userID},
				{Key: "notification_id", Value: tt.notificationID},
			}
			c.Request = httptest.NewRequest("DELETE", "/users/"+tt.userID+"/notifications/"+tt.notificationID, nil)

			DeleteUserNotification(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectError {
				assert.NotNil(t, response["error"])
			} else {
				assert.NotNil(t, response["message"])
				assert.Contains(t, response["message"], "deleted successfully")
			}
		})
	}
}

// Test MarkAllNotificationsAsRead - Mark all unread notifications as read
func TestMarkAllNotificationsAsRead(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		currentUser    models.UserProfile
		isAdmin        bool
		unreadCount    int64
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "successful mark all as read - own notifications",
			userID:         "1",
			currentUser:    MockUser(),
			isAdmin:        false,
			unreadCount:    5,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "successful mark all as read - admin for other user",
			userID:         "2",
			currentUser:    MockAdminUser(),
			isAdmin:        true,
			unreadCount:    3,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "no unread notifications",
			userID:         "1",
			currentUser:    MockUser(),
			isAdmin:        false,
			unreadCount:    0,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "forbidden - mark all for other user",
			userID:         "2",
			currentUser:    MockUser(),
			isAdmin:        false,
			unreadCount:    0,
			expectedStatus: http.StatusForbidden,
			expectError:    true,
		},
		{
			name:           "invalid user ID",
			userID:         "invalid",
			currentUser:    MockUser(),
			isAdmin:        false,
			unreadCount:    0,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			// Only mock database operations for valid IDs and authorized users
			if tt.userID != "invalid" && (tt.currentUser.User_Profile_ID == 1 || tt.isAdmin) {
				// Mock update
				mock.ExpectExec("UPDATE \"notification\"").
					WillReturnResult(sqlmock.NewResult(0, tt.unreadCount))
			}

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, tt.currentUser, tt.isAdmin)
			c.Params = []gin.Param{
				{Key: "user_profile_id", Value: tt.userID},
			}
			c.Request = httptest.NewRequest("PATCH", "/users/"+tt.userID+"/notifications/mark-all-read", nil)

			MarkAllNotificationsAsRead(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectError {
				assert.NotNil(t, response["error"])
			} else {
				assert.NotNil(t, response["message"])
				assert.Contains(t, response["message"], "marked as read")
				assert.Equal(t, float64(tt.unreadCount), response["updatedCount"])
			}
		})
	}
}
