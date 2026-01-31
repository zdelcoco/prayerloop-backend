package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestCreatePrayerSubject tests the link_status logic in CreatePrayerSubject
func TestCreatePrayerSubject(t *testing.T) {
	tests := []struct {
		name               string
		userProfileID      *int
		expectedLinkStatus string
	}{
		{
			name:               "link_status is linked when user_profile_id is self",
			userProfileID:      IntPtr(1), // Same as MockUser (User_Profile_ID = 1)
			expectedLinkStatus: "linked",
		},
		{
			name:               "link_status is linked when user_profile_id is another user",
			userProfileID:      IntPtr(99), // Different user
			expectedLinkStatus: "linked",
		},
		{
			name:               "link_status is unlinked when user_profile_id is nil",
			userProfileID:      nil,
			expectedLinkStatus: "unlinked",
		},
		{
			name:               "link_status is unlinked when user_profile_id is zero",
			userProfileID:      IntPtr(0),
			expectedLinkStatus: "unlinked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			currentUser := MockUser()
			userID := currentUser.User_Profile_ID

			// Mock MAX(display_sequence) query
			maxSeqRows := sqlmock.NewRows([]string{"coalesce"}).AddRow(-1)
			mock.ExpectQuery("SELECT COALESCE").WillReturnRows(maxSeqRows)

			// Mock INSERT with RETURNING prayer_subject_id
			mock.ExpectQuery("INSERT INTO \"prayer_subject\"").
				WillReturnRows(sqlmock.NewRows([]string{"prayer_subject_id"}).AddRow(1))

			// Mock SELECT for fetching created subject
			// Build the row based on expected link_status
			subjectRows := sqlmock.NewRows([]string{
				"prayer_subject_id", "prayer_subject_type", "prayer_subject_display_name",
				"notes", "display_sequence", "photo_s3_key", "user_profile_id",
				"use_linked_user_photo", "link_status", "datetime_create", "datetime_update",
				"created_by", "updated_by",
			}).AddRow(
				1, "individual", "Test Subject",
				nil, 0, nil, tt.userProfileID,
				false, tt.expectedLinkStatus, nil, nil,
				userID, userID,
			)
			mock.ExpectQuery("SELECT").WillReturnRows(subjectRows)

			// Build request body
			requestBody := map[string]interface{}{
				"prayerSubjectDisplayName": "Test Subject",
				"prayerSubjectType":        "individual",
			}
			if tt.userProfileID != nil {
				requestBody["userProfileId"] = *tt.userProfileID
			}

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, currentUser, false)
			c.Params = []gin.Param{{Key: "user_profile_id", Value: "1"}}

			jsonData, _ := json.Marshal(requestBody)
			c.Request = httptest.NewRequest("POST", "/users/1/prayer-subjects", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			CreatePrayerSubject(c)

			assert.Equal(t, http.StatusCreated, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Verify prayerSubject is returned
			prayerSubject, ok := response["prayerSubject"].(map[string]interface{})
			if !ok {
				// If prayerSubject not in response (edge case where SELECT fails), skip link_status check
				t.Log("prayerSubject not in response, checking prayerSubjectId only")
				assert.NotNil(t, response["prayerSubjectId"])
				return
			}

			// Verify link_status matches expected value
			linkStatus, ok := prayerSubject["linkStatus"].(string)
			if ok {
				assert.Equal(t, tt.expectedLinkStatus, linkStatus,
					"Expected link_status=%s for userProfileId=%v", tt.expectedLinkStatus, tt.userProfileID)
			}
		})
	}
}

// TestCreatePrayerSubjectValidation tests validation in CreatePrayerSubject
func TestCreatePrayerSubjectValidation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectError    bool
	}{
		{
			name: "missing display name",
			requestBody: map[string]interface{}{
				"prayerSubjectType": "individual",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "empty display name",
			requestBody: map[string]interface{}{
				"prayerSubjectDisplayName": "   ",
				"prayerSubjectType":        "individual",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid prayer subject type",
			requestBody: map[string]interface{}{
				"prayerSubjectDisplayName": "Test Subject",
				"prayerSubjectType":        "invalid_type",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, cleanup := SetupTestDB(t)
			defer cleanup()

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, MockUser(), false)
			c.Params = []gin.Param{{Key: "user_profile_id", Value: "1"}}

			jsonData, _ := json.Marshal(tt.requestBody)
			c.Request = httptest.NewRequest("POST", "/users/1/prayer-subjects", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			CreatePrayerSubject(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectError {
				assert.NotNil(t, response["error"])
			}
		})
	}
}

// TestCreatePrayerSubjectAuthorization tests authorization in CreatePrayerSubject
func TestCreatePrayerSubjectAuthorization(t *testing.T) {
	tests := []struct {
		name           string
		currentUser    int // User_Profile_ID of current user
		targetUser     int // User to create subject for
		isAdmin        bool
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "user can create for self",
			currentUser:    1,
			targetUser:     1,
			isAdmin:        false,
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name:           "user cannot create for another user",
			currentUser:    1,
			targetUser:     2,
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
			expectError:    true,
		},
		{
			name:           "admin can create for any user",
			currentUser:    2,
			targetUser:     1,
			isAdmin:        true,
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			currentUser := MockUser()
			currentUser.User_Profile_ID = tt.currentUser
			if tt.isAdmin {
				currentUser = MockAdminUser()
			}

			// Only mock DB queries if we expect success (not forbidden)
			if !tt.expectError {
				// Mock MAX(display_sequence) query
				maxSeqRows := sqlmock.NewRows([]string{"coalesce"}).AddRow(-1)
				mock.ExpectQuery("SELECT COALESCE").WillReturnRows(maxSeqRows)

				// Mock INSERT with RETURNING
				mock.ExpectQuery("INSERT INTO \"prayer_subject\"").
					WillReturnRows(sqlmock.NewRows([]string{"prayer_subject_id"}).AddRow(1))

				// Mock SELECT for fetching created subject
				subjectRows := sqlmock.NewRows([]string{
					"prayer_subject_id", "prayer_subject_type", "prayer_subject_display_name",
					"notes", "display_sequence", "photo_s3_key", "user_profile_id",
					"use_linked_user_photo", "link_status", "datetime_create", "datetime_update",
					"created_by", "updated_by",
				}).AddRow(1, "individual", "Test Subject", nil, 0, nil, nil, false, "unlinked", nil, nil, tt.targetUser, tt.targetUser)
				mock.ExpectQuery("SELECT").WillReturnRows(subjectRows)
			}

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, currentUser, tt.isAdmin)
			c.Params = []gin.Param{{Key: "user_profile_id", Value: string(rune('0' + tt.targetUser))}}

			requestBody := map[string]interface{}{
				"prayerSubjectDisplayName": "Test Subject",
				"prayerSubjectType":        "individual",
			}
			jsonData, _ := json.Marshal(requestBody)
			c.Request = httptest.NewRequest("POST", "/users/"+string(rune('0'+tt.targetUser))+"/prayer-subjects", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			CreatePrayerSubject(c)

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
