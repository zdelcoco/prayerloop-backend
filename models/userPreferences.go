package models

import "time"

type UserPreferences struct {
	User_Preferences_ID int       `json:"userPreferenceId" goqu:"skipinsert"`
	User_Profile_ID     int       `json:"userId"`
	Preference_Key      string    `json:"preferenceKey"`   // e.g., "theme", "notifications"
	Preference_Value    string    `json:"preferenceValue"` // e.g., "dark", "true"
	Is_Active           bool      `json:"isActive"`        // Indicates if the preference is active
	Datetime_Create     time.Time `json:"datetimeCreate"`  // Timestamp when the preference was created
	Datetime_Update     time.Time `json:"datetimeUpdate"`  // Timestamp when the preference was last updated
}
