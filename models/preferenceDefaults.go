package models

import "time"

type Preference struct {
	Preference_ID   int       `json:"preferenceId" goqu:"skipinsert"`
	Preference_Key  string    `json:"preferenceKey"`
	Default_Value   string    `json:"defaultValue"`
	Description     string    `json:"description"`
	Value_Type      string    `json:"valueType"`
	Datetime_Create time.Time `json:"datetimeCreate" goqu:"skipinsert"`
	Datetime_Update time.Time `json:"datetimeUpdate" goqu:"skipinsert"`
	Created_By      int       `json:"createdBy"`
	Updated_By      int       `json:"updatedBy"`
	Is_Active       bool      `json:"isActive" goqu:"skipinsert"`
}
