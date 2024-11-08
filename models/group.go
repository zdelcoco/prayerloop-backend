package models

import "time"

type Group struct {
	Group_ID        int       `goqu:"skipinsert" json:"groupId"`
	Group_Name      string    `json:"groupName"`
	Description     string    `json:"groupDescription"`
	Is_Active       bool      `json:"isActive"`
	Datetime_Create time.Time `json:"datetimeCreate"`
	Datetime_Update time.Time `json:"datetimeUpdate"`
	Created_By      int       `json:"createdBy"`
	Updated_By      int       `json:"updatedBy"`
}

type GroupCreate struct {
	Group_Name        string `json:"groupName"`
	Group_Description string `json:"groupDescription"`
}

type GroupUpdate struct {
	Group_Name        string `json:"groupName"`
	Group_Description string `json:"groupDescription"`
	Is_Active         bool   `json:"isActive"`
}
