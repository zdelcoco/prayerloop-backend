package models

import "time"

type GroupProfile struct {
	Group_Profile_ID     int       `json:"groupId" goqu:"skipinsert" `
	Group_Name           string    `json:"groupName"`
	Group_Description    string    `json:"groupDescription"`
	Is_Active            bool      `json:"isActive"`
	Datetime_Create      time.Time `json:"datetimeCreate"`
	Datetime_Update      time.Time `json:"datetimeUpdate"`
	Created_By           int       `json:"createdBy"`
	Updated_By           int       `json:"updatedBy"`
	Deleted              bool      `json:"deleted" goqu:"skipinsert"`
}

type GroupCreate struct {
	Group_Name        string `json:"groupName"`
	Group_Description string `json:"groupDescription"`
}

type GroupUpdate struct {
	Group_Name        string `json:"groupName"`
	Group_Description string `json:"groupDescription"`
	Is_Active         bool   `json:"isActive"`
	Deleted           bool   `json:"deleted"`
}
