package models

import "time"

type GroupInvite struct {
	Group_Invite_ID  int       `json:"groupInviteId" goqu:"skipinsert"`
	Group_Profile_ID int       `json:"groupProfileId"`
	Invite_Code      string    `json:"inviteCode"`
	Datetime_Create  time.Time `json:"datetimeCreate"`
	Datetime_Update  time.Time `json:"datetimeUpdate"`
	Created_By       int       `json:"createdBy"`
	Updated_By       int       `json:"updatedBy"`
	Datetime_Expires time.Time `json:"datetimeExpires"`
	Is_Active        bool      `json:"isActive"`
}
