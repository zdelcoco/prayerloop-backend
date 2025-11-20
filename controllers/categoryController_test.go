package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PrayerLoop/models"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Test GetUserCategories
func TestGetUserCategories(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		currentUser    models.UserProfile
		hasCategories  bool
		expectedStatus int
		expectError    bool
	}{
		{
			name:   "successful fetch - user with categories",
			userID: "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
				First_Name:      "Test",
				Last_Name:       "User",
			},
			hasCategories:  true,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:   "successful fetch - user with no categories",
			userID: "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
				First_Name:      "Test",
				Last_Name:       "User",
			},
			hasCategories:  false,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:   "forbidden - accessing another user's categories",
			userID: "2",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
				First_Name:      "Test",
				Last_Name:       "User",
			},
			hasCategories:  false,
			expectedStatus: http.StatusForbidden,
			expectError:    true,
		},
		{
			name:   "bad request - invalid user ID",
			userID: "invalid",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			if tt.expectedStatus == http.StatusOK {
				if tt.hasCategories {
					now := time.Now()
					rows := sqlmock.NewRows([]string{"prayer_category_id", "category_type", "category_type_id", "category_name", "category_color", "display_sequence", "datetime_create", "datetime_update", "created_by", "updated_by"}).
						AddRow(1, "user", 1, "Family", "#FF6B6B", 0, now, now, 1, 1).
						AddRow(2, "user", 1, "Work", "#4ECDC4", 1, now, now, 1, 1)
					mock.ExpectQuery("SELECT").WillReturnRows(rows)
				} else {
					mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"prayer_category_id", "category_type", "category_type_id", "category_name", "category_color", "display_sequence", "datetime_create", "datetime_update", "created_by", "updated_by"}))
				}
			}

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, tt.currentUser, false)
			c.Params = append(c.Params, gin.Param{Key: "user_profile_id", Value: tt.userID})

			GetUserCategories(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectError && tt.hasCategories {
				var response []models.PrayerCategory
				_ = json.Unmarshal(w.Body.Bytes(), &response)
				assert.Greater(t, len(response), 0)
			}
		})
	}
}

// Test GetGroupCategories
func TestGetGroupCategories(t *testing.T) {
	tests := []struct {
		name           string
		groupID        string
		currentUser    models.UserProfile
		isMember       bool
		hasCategories  bool
		expectedStatus int
		expectError    bool
	}{
		{
			name:    "successful fetch - member with categories",
			groupID: "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
			},
			isMember:       true,
			hasCategories:  true,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:    "forbidden - not a member",
			groupID: "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
			},
			isMember:       false,
			expectedStatus: http.StatusForbidden,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			if tt.expectedStatus == http.StatusOK || tt.isMember {
				// Mock membership check
				membershipRows := sqlmock.NewRows([]string{"user_group_id", "user_profile_id", "group_profile_id"}).
					AddRow(1, 1, 1)
				mock.ExpectQuery("SELECT").WillReturnRows(membershipRows)

				if tt.hasCategories {
					now := time.Now()
					categoryRows := sqlmock.NewRows([]string{"prayer_category_id", "category_type", "category_type_id", "category_name", "category_color", "display_sequence", "datetime_create", "datetime_update", "created_by", "updated_by"}).
						AddRow(1, "group", 1, "Mission", "#6A4C93", 0, now, now, 1, 1)
					mock.ExpectQuery("SELECT").WillReturnRows(categoryRows)
				} else {
					mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"prayer_category_id", "category_type", "category_type_id", "category_name", "category_color", "display_sequence", "datetime_create", "datetime_update", "created_by", "updated_by"}))
				}
			} else {
				// Not a member
				mock.ExpectQuery("SELECT").WillReturnError(sqlmock.ErrCancelled)
			}

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, tt.currentUser, false)
			c.Params = append(c.Params, gin.Param{Key: "group_profile_id", Value: tt.groupID})

			GetGroupCategories(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// Test CreateUserCategory
func TestCreateUserCategory(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		currentUser    models.UserProfile
		body           map[string]interface{}
		expectedStatus int
		expectError    bool
	}{
		{
			name:   "successful creation",
			userID: "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
			},
			body: map[string]interface{}{
				"categoryName":  "Family",
				"categoryColor": "#FF6B6B",
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name:   "forbidden - creating for another user",
			userID: "2",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
			},
			body: map[string]interface{}{
				"categoryName": "Family",
			},
			expectedStatus: http.StatusForbidden,
			expectError:    true,
		},
		{
			name:   "bad request - missing required field",
			userID: "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
			},
			body:           map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			if tt.expectedStatus == http.StatusCreated {
				// Mock max sequence query
				mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(0))
				// Mock insert
				mock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
			}

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, tt.currentUser, false)
			c.Params = append(c.Params, gin.Param{Key: "user_profile_id", Value: tt.userID})

			bodyBytes, _ := json.Marshal(tt.body)
			c.Request = httptest.NewRequest("POST", "/users/"+tt.userID+"/categories", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			CreateUserCategory(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// Test CreateGroupCategory
func TestCreateGroupCategory(t *testing.T) {
	tests := []struct {
		name           string
		groupID        string
		currentUser    models.UserProfile
		isMember       bool
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name:    "successful creation - member creates category",
			groupID: "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
			},
			isMember: true,
			body: map[string]interface{}{
				"categoryName":  "Mission",
				"categoryColor": "#6A4C93",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:    "forbidden - non-member tries to create",
			groupID: "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
			},
			isMember: false,
			body: map[string]interface{}{
				"categoryName": "Mission",
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			if tt.isMember {
				// Mock membership check
				membershipRows := sqlmock.NewRows([]string{"user_group_id", "user_profile_id", "group_profile_id"}).
					AddRow(1, 1, 1)
				mock.ExpectQuery("SELECT").WillReturnRows(membershipRows)

				if tt.expectedStatus == http.StatusCreated {
					// Mock max sequence query
					mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(0))
					// Mock insert
					mock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
				}
			} else {
				mock.ExpectQuery("SELECT").WillReturnError(sqlmock.ErrCancelled)
			}

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, tt.currentUser, false)
			c.Params = append(c.Params, gin.Param{Key: "group_profile_id", Value: tt.groupID})

			bodyBytes, _ := json.Marshal(tt.body)
			c.Request = httptest.NewRequest("POST", "/groups/"+tt.groupID+"/categories", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			CreateGroupCategory(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// Test UpdateCategory
func TestUpdateCategory(t *testing.T) {
	tests := []struct {
		name           string
		categoryID     string
		currentUser    models.UserProfile
		isOwner        bool
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name:       "successful update - user owns category",
			categoryID: "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
			},
			isOwner: true,
			body: map[string]interface{}{
				"categoryName":  "Updated Family",
				"categoryColor": "#FF0000",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "forbidden - user doesn't own category",
			categoryID: "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 2,
			},
			isOwner: false,
			body: map[string]interface{}{
				"categoryName": "Updated Family",
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			now := time.Now()
			ownerID := 1
			if !tt.isOwner {
				ownerID = 3
			}

			// Mock get category
			categoryRows := sqlmock.NewRows([]string{"prayer_category_id", "category_type", "category_type_id", "category_name", "category_color", "display_sequence", "datetime_create", "datetime_update", "created_by", "updated_by"}).
				AddRow(1, "user", ownerID, "Family", "#FF6B6B", 0, now, now, ownerID, ownerID)
			mock.ExpectQuery("SELECT").WillReturnRows(categoryRows)

			if tt.expectedStatus == http.StatusOK {
				// Mock update
				mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
				// Mock fetch updated category
				updatedRows := sqlmock.NewRows([]string{"prayer_category_id", "category_type", "category_type_id", "category_name", "category_color", "display_sequence", "datetime_create", "datetime_update", "created_by", "updated_by"}).
					AddRow(1, "user", ownerID, "Updated Family", "#FF0000", 0, now, now, ownerID, ownerID)
				mock.ExpectQuery("SELECT").WillReturnRows(updatedRows)
			}

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, tt.currentUser, false)
			c.Params = append(c.Params, gin.Param{Key: "prayer_category_id", Value: tt.categoryID})

			bodyBytes, _ := json.Marshal(tt.body)
			c.Request = httptest.NewRequest("PUT", "/categories/"+tt.categoryID, bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			UpdateCategory(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// Test DeleteCategory
func TestDeleteCategory(t *testing.T) {
	tests := []struct {
		name           string
		categoryID     string
		currentUser    models.UserProfile
		isOwner        bool
		expectedStatus int
	}{
		{
			name:       "successful deletion - user owns category",
			categoryID: "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
			},
			isOwner:        true,
			expectedStatus: http.StatusOK,
		},
		{
			name:       "forbidden - user doesn't own category",
			categoryID: "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 2,
			},
			isOwner:        false,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			now := time.Now()
			ownerID := 1
			if !tt.isOwner {
				ownerID = 3
			}

			// Mock get category
			categoryRows := sqlmock.NewRows([]string{"prayer_category_id", "category_type", "category_type_id", "category_name", "category_color", "display_sequence", "datetime_create", "datetime_update", "created_by", "updated_by"}).
				AddRow(1, "user", ownerID, "Family", "#FF6B6B", 0, now, now, ownerID, ownerID)
			mock.ExpectQuery("SELECT").WillReturnRows(categoryRows)

			if tt.expectedStatus == http.StatusOK {
				// Mock delete
				mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(0, 1))
			}

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, tt.currentUser, false)
			c.Params = append(c.Params, gin.Param{Key: "prayer_category_id", Value: tt.categoryID})
			c.Request = httptest.NewRequest("DELETE", "/categories/"+tt.categoryID, nil)

			DeleteCategory(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// Test ReorderUserCategories
func TestReorderUserCategories(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		currentUser    models.UserProfile
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name:   "successful reorder",
			userID: "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
			},
			body: map[string]interface{}{
				"categoryIds": []int{2, 1, 3},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "forbidden - reordering another user's categories",
			userID: "2",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
			},
			body: map[string]interface{}{
				"categoryIds": []int{2, 1},
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			if tt.expectedStatus == http.StatusOK {
				// Mock updates for each category
				mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
			}

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, tt.currentUser, false)
			c.Params = append(c.Params, gin.Param{Key: "user_profile_id", Value: tt.userID})

			bodyBytes, _ := json.Marshal(tt.body)
			c.Request = httptest.NewRequest("PATCH", "/users/"+tt.userID+"/categories/reorder", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			ReorderUserCategories(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// Test AddPrayerToCategory
func TestAddPrayerToCategory(t *testing.T) {
	tests := []struct {
		name           string
		categoryID     string
		prayerID       string
		currentUser    models.UserProfile
		categoryType   string
		expectedStatus int
	}{
		{
			name:       "successful add - user prayer to user category",
			categoryID: "1",
			prayerID:   "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
			},
			categoryType:   "user",
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			now := time.Now()

			// Mock get category
			categoryRows := sqlmock.NewRows([]string{"prayer_category_id", "category_type", "category_type_id", "category_name", "category_color", "display_sequence", "datetime_create", "datetime_update", "created_by", "updated_by"}).
				AddRow(1, "user", 1, "Family", "#FF6B6B", 0, now, now, 1, 1)
			mock.ExpectQuery("SELECT").WillReturnRows(categoryRows)

			// Mock get prayer access
			prayerAccessRows := sqlmock.NewRows([]string{"prayer_access_id", "prayer_id", "access_type", "access_type_id"}).
				AddRow(1, 1, "user", 1)
			mock.ExpectQuery("SELECT").WillReturnRows(prayerAccessRows)

			// Mock check existing
			mock.ExpectQuery("SELECT").WillReturnError(sqlmock.ErrCancelled)

			// Mock insert
			mock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, tt.currentUser, false)
			c.Params = append(c.Params, gin.Param{Key: "prayer_category_id", Value: tt.categoryID})
			c.Params = append(c.Params, gin.Param{Key: "prayer_access_id", Value: tt.prayerID})
			c.Request = httptest.NewRequest("POST", "/categories/"+tt.categoryID+"/prayers/"+tt.prayerID, nil)

			AddPrayerToCategory(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// Test RemovePrayerFromCategory
func TestRemovePrayerFromCategory(t *testing.T) {
	tests := []struct {
		name           string
		prayerID       string
		currentUser    models.UserProfile
		hasCategory    bool
		expectedStatus int
	}{
		{
			name:     "successful removal",
			prayerID: "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
			},
			hasCategory:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:     "not found - prayer not in any category",
			prayerID: "1",
			currentUser: models.UserProfile{
				User_Profile_ID: 1,
			},
			hasCategory:    false,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, cleanup := SetupTestDB(t)
			defer cleanup()

			if tt.hasCategory {
				now := time.Now()
				// Mock get category item
				itemRows := sqlmock.NewRows([]string{"prayer_category_item_id", "prayer_category_id", "prayer_access_id", "datetime_create", "created_by"}).
					AddRow(1, 1, 1, now, 1)
				mock.ExpectQuery("SELECT").WillReturnRows(itemRows)

				// Mock get category
				categoryRows := sqlmock.NewRows([]string{"prayer_category_id", "category_type", "category_type_id", "category_name", "category_color", "display_sequence", "datetime_create", "datetime_update", "created_by", "updated_by"}).
					AddRow(1, "user", 1, "Family", "#FF6B6B", 0, now, now, 1, 1)
				mock.ExpectQuery("SELECT").WillReturnRows(categoryRows)

				// Mock delete
				mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(0, 1))
			} else {
				mock.ExpectQuery("SELECT").WillReturnError(sqlmock.ErrCancelled)
			}

			c, w := SetupTestContext()
			SetAuthenticatedUser(c, tt.currentUser, false)
			c.Params = append(c.Params, gin.Param{Key: "prayer_access_id", Value: tt.prayerID})
			c.Request = httptest.NewRequest("DELETE", "/categories/1/prayers/"+tt.prayerID, nil)

			RemovePrayerFromCategory(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
