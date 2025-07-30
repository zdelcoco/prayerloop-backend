package controllers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"

	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"
	"github.com/doug-martin/goqu/v9"
	"golang.org/x/crypto/bcrypt"
)

func PublicUserSignup(c *gin.Context) {
	var user models.UserProfileSignup

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	var missingFields []string
	if user.Username == "" {
		missingFields = append(missingFields, "username")
	}
	if user.Password == "" {
		missingFields = append(missingFields, "password")
	}
	if user.Email == "" {
		missingFields = append(missingFields, "email")
	}
	if user.First_Name == "" {
		missingFields = append(missingFields, "firstName")
	}
	if user.Last_Name == "" {
		missingFields = append(missingFields, "lastName")
	}

	if len(missingFields) > 0 {
		var errorMsg string
		if len(missingFields) == 1 {
			errorMsg = fmt.Sprintf("The following field is required: %s", missingFields[0])
		} else {
			errorMsg = fmt.Sprintf("The following fields are required: %s", strings.Join(missingFields, ", "))
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMsg})
		return
	}

	// Check if username already exists
	userCount, err := initializers.DB.From("user_profile").Select("username").Where(goqu.C("username").Eq(user.Username)).Count()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if userCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username already exists."})
		return
	}

	// Check if email already exists
	emailCount, err := initializers.DB.From("user_profile").Select("email").Where(goqu.C("email").Eq(user.Email)).Count()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if emailCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists."})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handle phone number - convert to pointer for nullable field
	var phoneNumber *string
	if user.Phone_Number != "" {
		phoneNumber = &user.Phone_Number
	}

	newUser := models.UserProfile{
		Username:     user.Username,
		Password:     string(passwordHash),
		Email:        user.Email,
		First_Name:   user.First_Name,
		Last_Name:    user.Last_Name,
		Phone_Number: phoneNumber,
		Created_By:   1,
		Updated_By:   1,
	}

	insert := initializers.DB.Insert("user_profile").Rows(newUser).Executor()
	if _, err := insert.Exec(); err != nil {
		log.Default().Println(insert)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	} else {
		c.JSON(200, gin.H{
			"message": "User created successfully.",
			"user":    user,
		})
	}
}

func UserSignup(c *gin.Context) {
	admin := c.MustGet("admin")

	if admin != true {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "must be logged in as an admin to create a user."})
		return
	}
	var user models.UserProfileSignup

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if username already exists
	userCount, err := initializers.DB.From("user_profile").Select("username").Where(goqu.C("username").Eq(user.Username)).Count()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if userCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username already exists."})
		return
	}

	// Check if email already exists (if provided)
	if user.Email != "" {
		emailCount, err := initializers.DB.From("user_profile").Select("email").Where(goqu.C("email").Eq(user.Email)).Count()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if emailCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists."})
			return
		}
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handle phone number - convert to pointer for nullable field
	var phoneNumber *string
	if user.Phone_Number != "" {
		phoneNumber = &user.Phone_Number
	}

	newUser := models.UserProfile{
		Username:     user.Username,
		Password:     string(passwordHash),
		Email:        user.Email,
		First_Name:   user.First_Name,
		Last_Name:    user.Last_Name,
		Phone_Number: phoneNumber,
		Created_By:   1,
		Updated_By:   1,
	}

	insert := initializers.DB.Insert("user_profile").Rows(newUser).Executor()
	if _, err := insert.Exec(); err != nil {
		log.Default().Println(insert)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	} else {
		c.JSON(200, gin.H{
			"message": "User created successfully.",
			"user":    user,
		})
	}
}

func CheckUsernameAvailability(c *gin.Context) {
	username := c.Query("username")

	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username parameter is required"})
		return
	}

	userCount, err := initializers.DB.From("user_profile").Select("username").Where(goqu.C("username").Eq(username)).Count()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"username":  username,
		"available": userCount == 0,
	})
}

func UserLogin(c *gin.Context) {
	var user models.Login

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var dbUser models.UserProfile
	_, err := initializers.DB.From("user_profile").Select("*").Where(goqu.C("username").Eq(user.Username)).ScanStruct(&dbUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
	// 	return
	// }

	// update := initializers.DB.Update("user_profile").Set(goqu.Record{"password": string(passwordHash)}).Where(goqu.C("user_profile_id").Eq(dbUser.User_Profile_ID)).Executor()
	// if _, err := update.Exec(); err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
	// 	return
	// }

	err = bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(user.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password", "details": err.Error()})
		return
	}

	role := ""
	if dbUser.Admin {
		role = "admin"
	} else {
		role = "user"
	}

	generateToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":   dbUser.User_Profile_ID,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
		"role": role,
	})

	token, err := generateToken.SignedString([]byte(os.Getenv("SECRET")))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to generate token", "details": err.Error()})
	}

	c.JSON(200, gin.H{
		"message": "User logged in successfully.",
		"token":   token,
		"user":    dbUser,
	})
}

func GetUserProfile(c *gin.Context) {

	user, _ := c.Get("currentUser")

	c.JSON(200, gin.H{
		"user":  user,
		"admin": c.MustGet("admin"),
	})
}

func GetUserGroups(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if userID != currentUser.User_Profile_ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this user's groups"})
		return
	}

	query := initializers.DB.From("user_group").
		Select(
			"group_profile.group_profile_id",
			"group_profile.group_name",
			"group_profile.group_description",
			"group_profile.is_active",
			"group_profile.datetime_create",
			"group_profile.datetime_update",
			"group_profile.created_by",
			"group_profile.updated_by",
			"group_profile.deleted",
		).
		InnerJoin(
			goqu.T("group_profile"),
			goqu.On(goqu.Ex{"user_group.group_profile_id": goqu.I("group_profile.group_profile_id")}),
		).
		Where(
			goqu.And(
				goqu.C("user_profile_id").Table("user_group").Eq(userID),
				goqu.C("is_active").Table("user_group").IsTrue(),
				goqu.C("is_active").Table("group_profile").IsTrue(),
			),
		)

	sql, args, err := query.ToSQL()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to construct query", "details": err.Error()})
		return
	}

	log.Println(sql, args)

	var groups []models.GroupProfile
	err = initializers.DB.ScanStructs(&groups, sql, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user groups", "details": err.Error()})
		return
	}

	if len(groups) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No groups found for this user"})
		return
	}

	c.JSON(http.StatusOK, groups)
}

func GetUserPrayers(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if userID != currentUser.User_Profile_ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this user's prayers"})
		return
	}

	var userPrayers []models.UserPrayer

	dbErr := initializers.DB.From("prayer_access").
		Select(
			goqu.L("?", currentUser.User_Profile_ID).As("user_profile_id"),
			goqu.I("prayer.prayer_id"),
			goqu.I("prayer_access.prayer_access_id"),
			goqu.I("prayer.prayer_type"),
			goqu.I("prayer.is_private"),
			goqu.I("prayer.title"),
			goqu.I("prayer.prayer_description"),
			goqu.I("prayer.is_answered"),
			goqu.I("prayer.prayer_priority"),
			goqu.I("prayer.datetime_answered"),
			goqu.I("prayer.created_by"),
			goqu.I("prayer.datetime_create"),
			goqu.I("prayer.updated_by"),
			goqu.I("prayer.datetime_update"),
			goqu.I("prayer.deleted"),
		).
		Join(
			goqu.T("prayer"),
			goqu.On(goqu.Ex{"prayer_access.prayer_id": goqu.I("prayer.prayer_id")}),
		).
		Where(
			goqu.And(
				goqu.Ex{"prayer_access.access_type": "user"},
				goqu.Ex{"prayer_access.access_type_id": currentUser.User_Profile_ID},
				goqu.Ex{"prayer.deleted": false},
			),
		).
		Order(goqu.I("prayer.prayer_id").Asc()).
		ScanStructsContext(c, &userPrayers)

	if dbErr != nil {
		c.JSON(500, gin.H{"error": dbErr.Error()})
		return
	}

	if len(userPrayers) == 0 {
		c.JSON(200, gin.H{"message": "No prayer records found."})
		return
	}

	c.JSON(200, gin.H{
		"message": "Prayer records retrieved successfully.",
		"prayers": userPrayers,
	})
}

func CreateUserPrayer(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if currentUser.User_Profile_ID != userID &&
		!isAdmin {
		c.JSON(http.StatusForbidden,
			gin.H{"error": fmt.Sprintf("You don't have permission to create a prayer on behalf of user %d",
				currentUser.User_Profile_ID)})
		return
	}

	var newPrayer models.PrayerCreate
	if err := c.BindJSON(&newPrayer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newPrayerEntry := models.Prayer{
		Prayer_Type:        newPrayer.Prayer_Type,
		Is_Private:         newPrayer.Is_Private,
		Title:              newPrayer.Title,
		Prayer_Description: newPrayer.Prayer_Description,
		Is_Answered:        newPrayer.Is_Answered,
		Datetime_Answered:  newPrayer.Datetime_Answered,
		Prayer_Priority:    newPrayer.Prayer_Priority,
		Created_By:         currentUser.User_Profile_ID,
		Updated_By:         currentUser.User_Profile_ID,
		Datetime_Create:    time.Now(),
		Datetime_Update:    time.Now(),
	}

	prayerInsert := initializers.DB.Insert("prayer").Rows(newPrayerEntry).Returning("prayer_id")

	var insertedPrayerID int
	_, err = prayerInsert.Executor().ScanVal(&insertedPrayerID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create prayer record", "details": err.Error()})
		return
	}

	newPrayerAccessEntry := models.PrayerAccess{
		Prayer_ID:       insertedPrayerID,
		Access_Type:     "user",
		Access_Type_ID:  userID,
		Created_By:      currentUser.User_Profile_ID,
		Updated_By:      currentUser.User_Profile_ID,
		Datetime_Create: time.Now(),
		Datetime_Update: time.Now(),
	}

	prayerAccessInsert := initializers.DB.Insert("prayer_access").Rows(newPrayerAccessEntry).Returning("prayer_access_id")

	var insertedPrayerAccessID int
	_, err = prayerAccessInsert.Executor().ScanVal(&insertedPrayerAccessID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create prayer access record", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Prayer created sucessfully!",
		"prayerId":       insertedPrayerID,
		"prayerAccessId": insertedPrayerAccessID})
}

func GetUserPreferencesWithDefaults(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if userID != currentUser.User_Profile_ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this user's preferences"})
		return
	}

	// First, get all default preferences
	var defaultPrefs []models.Preference
	dbErr := initializers.DB.From("preference").
		Select("preference_id", "preference_key", "default_value", "description", "value_type").
		Where(goqu.C("is_active").IsTrue()).
		Order(goqu.C("preference_key").Asc()).
		ScanStructsContext(c, &defaultPrefs)

	if dbErr != nil {
		c.JSON(500, gin.H{"error": "Failed to load default preferences", "details": dbErr.Error()})
		return
	}

	// Then, get user's custom preferences
	var userPrefs []models.UserPreferences
	dbErr = initializers.DB.From("user_preferences").
		Select("preference_key", "preference_value").
		Where(goqu.C("user_profile_id").Eq(userID)).
		ScanStructsContext(c, &userPrefs)

	if dbErr != nil {
		c.JSON(500, gin.H{"error": "Failed to load user preferences", "details": dbErr.Error()})
		return
	}

	// Create a map of user preferences for quick lookup
	userPrefMap := make(map[string]string)
	for _, pref := range userPrefs {
		userPrefMap[pref.Preference_Key] = pref.Preference_Value
	}

	// Build the response with defaults overridden by user preferences
	var responsePrefs []map[string]interface{}
	for _, defaultPref := range defaultPrefs {
		prefValue := defaultPref.Default_Value
		isDefault := true

		if userValue, exists := userPrefMap[defaultPref.Preference_Key]; exists {
			prefValue = userValue
			isDefault = false
		}

		responsePrefs = append(responsePrefs, map[string]interface{}{
			"preferenceId": defaultPref.Preference_ID,
			"key":          defaultPref.Preference_Key,
			"value":        prefValue,
			"description":  defaultPref.Description,
			"valueType":    defaultPref.Value_Type,
			"isDefault":    isDefault,
		})
	}

	c.JSON(200, gin.H{
		"message":     "User preferences retrieved successfully.",
		"preferences": responsePrefs,
	})
}

func GetUserPreferences(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if userID != currentUser.User_Profile_ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this user's preferences"})
		return
	}

	var preferences []models.UserPreferences

	dbErr := initializers.DB.From("user_preferences").
		Select("*").
		Where(goqu.C("user_profile_id").Eq(userID)).
		Order(goqu.C("preference_key").Asc()).
		ScanStructsContext(c, &preferences)

	if dbErr != nil {
		c.JSON(500, gin.H{"error": dbErr.Error()})
		return
	}

	if len(preferences) == 0 {
		c.JSON(200, gin.H{"message": "No preferences found for this user"})
		return
	}

	c.JSON(200, gin.H{
		"message":     "User preferences retrieved successfully.",
		"preferences": preferences,
	})
}

func UpdateUserPreferences(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	isAdmin := c.MustGet("admin").(bool)

	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user profile ID", "details": err.Error()})
		return
	}

	if userID != currentUser.User_Profile_ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this user's preferences"})
		return
	}

	preferenceID, err := strconv.Atoi(c.Param("preference_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid preference ID", "details": err.Error()})
		return
	}

	var updatedPreference models.UserPreferencesUpdate
	if err := c.BindJSON(&updatedPreference); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// First, verify the preference_id exists in the preference table
	var preference []models.Preference
	dbErr := initializers.DB.From("preference").
		Select("*").
		Where(goqu.C("preference_id").Eq(preferenceID)).
		ScanStructsContext(c, &preference)

	if dbErr != nil {
		c.JSON(500, gin.H{"error": "Failed to verify preference", "details": dbErr.Error()})
		return
	}

	if len(preference) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Preference not found"})
		return
	}

	basePreference := preference[0]

	// Validate that the preference key matches
	if basePreference.Preference_Key != updatedPreference.Preference_Key {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Preference key mismatch: expected '%s', but received '%s'",
				basePreference.Preference_Key,
				updatedPreference.Preference_Key),
		})
		return
	}

	// Check if user already has a custom preference for this preference_id
	var existingUserPrefs []models.UserPreferences
	dbErr = initializers.DB.From("user_preferences").
		Select("*").
		Where(goqu.And(
			goqu.C("user_profile_id").Eq(userID),
			goqu.C("preference_key").Eq(basePreference.Preference_Key),
		)).
		ScanStructsContext(c, &existingUserPrefs)

	if dbErr != nil {
		c.JSON(500, gin.H{"error": "Failed to check existing user preferences", "details": dbErr.Error()})
		return
	}

	// Perform validation based on preference type
	if basePreference.Value_Type == "boolean" &&
		updatedPreference.Preference_Value != "true" &&
		updatedPreference.Preference_Value != "false" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid value for boolean preference '%s'. Allowed values are 'true' or 'false', but received '%s'",
				basePreference.Preference_Key,
				updatedPreference.Preference_Value),
		})
		return
	}

	if updatedPreference.Preference_Key == "theme" &&
		updatedPreference.Preference_Value != "light" &&
		updatedPreference.Preference_Value != "dark" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid value for preference key 'theme'. Allowed values are 'light' or 'dark', but received '%s'",
				updatedPreference.Preference_Value),
		})
		return
	}

	// Check if this would be a no-op (same value)
	if len(existingUserPrefs) > 0 {
		existing := existingUserPrefs[0]
		if existing.Preference_Value == updatedPreference.Preference_Value &&
			existing.Is_Active == updatedPreference.Is_Active {
			c.JSON(http.StatusOK, gin.H{
				"message": "No changes detected in the user preferences. No update performed.",
				"preference": map[string]interface{}{
					"preferenceId": basePreference.Preference_ID,
					"key":          basePreference.Preference_Key,
					"value":        existing.Preference_Value,
					"description":  basePreference.Description,
					"valueType":    basePreference.Value_Type,
					"isDefault":    false,
				},
			})
			return
		}

		// Update existing user preference
		update := initializers.DB.Update("user_preferences").
			Set(goqu.Record{
				"preference_value": updatedPreference.Preference_Value,
				"is_active":        updatedPreference.Is_Active,
				"datetime_update":  time.Now(),
			}).
			Where(goqu.C("user_preferences_id").Eq(existing.User_Preferences_ID))

		_, err := update.Executor().Exec()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user preferences", "details": err.Error()})
			return
		}
	} else {
		// Insert new user preference
		newUserPref := models.UserPreferences{
			User_Profile_ID:  userID,
			Preference_Key:   basePreference.Preference_Key,
			Preference_Value: updatedPreference.Preference_Value,
			Is_Active:        updatedPreference.Is_Active,
			Datetime_Create:  time.Now(),
			Datetime_Update:  time.Now(),
		}

		insert := initializers.DB.Insert("user_preferences").Rows(newUserPref).Executor()
		if _, err := insert.Exec(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user preference", "details": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User preferences updated successfully.",
		"preference": map[string]interface{}{
			"preferenceId": basePreference.Preference_ID,
			"key":          basePreference.Preference_Key,
			"value":        updatedPreference.Preference_Value,
			"description":  basePreference.Description,
			"valueType":    basePreference.Value_Type,
			"isDefault":    false,
		},
	})
}
