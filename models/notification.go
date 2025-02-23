package models

import "time"

type Notification struct {
	Notification_ID      int       `json:"notificationId" goqu:"skipinsert"`
	User_Profile_ID      int       `json:"userProfileId"`
	Notification_Type    string    `json:"notificationType"`
	Notification_Message string    `json:"notificationMessage"`
	Notification_Status  string    `json:"notificationStatus"`
	DateTime_Create      time.Time `json:"datetimeCreate" goqu:"skipinsert"`
	DateTime_Update      time.Time `json:"datetimeUpdate" goqu:"skipinsert"`
	Created_By           int       `json:"createdBy"`
	Updated_By           int       `json:"updatedBy"`
}
