package models

import "time"

type UserGroup struct {
	User_Group_ID    int       `json:"userGroupId" goqu:"skipinsert"`
	User_Profile_ID  int       `json:"userId"`
	Group_Profile_ID int       `json:"groupId"`
	Is_Active        bool      `json:"isActive"`
	Created_By       int       `json:"createdBy"`
	Updated_By       int       `json:"updatedBy"`
	Datetime_Create  time.Time `json:"datetimeCreate"`
	Datetime_Update  time.Time `json:"datetimeUpdate"`
}
