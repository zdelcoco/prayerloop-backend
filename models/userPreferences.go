package models

import "time"

type UserPreferences struct {
	User_Preferences_ID int       `json:"userPreferenceId" goqu:"skipinsert"`
	User_Profile_ID     int       `json:"userId"`
	Preference_Key      string    `json:"preferenceKey"`
	Preference_Value    string    `json:"preferenceValue"`
	Is_Active           bool      `json:"isActive"`
	Datetime_Create     time.Time `json:"datetimeCreate"`
	Datetime_Update     time.Time `json:"datetimeUpdate"`
}

type UserPreferencesUpdate struct {
	Preference_Key   string `json:"preferenceKey"`
	Preference_Value string `json:"preferenceValue"`
	Is_Active        bool   `json:"isActive"`
}
