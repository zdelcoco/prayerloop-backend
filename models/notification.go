package models

import "time"

// Notification type constants
const (
	NotificationTypePrayerCreatedForYou  = "PRAYER_CREATED_FOR_YOU"
	NotificationTypePrayerEditedBySubject = "PRAYER_EDITED_BY_SUBJECT"
	NotificationTypePrayerShared         = "PRAYER_SHARED"
	NotificationTypeGroupInvite          = "GROUP_INVITE"
	NotificationTypeGroupMemberJoined    = "GROUP_MEMBER_JOINED"
)

// Notification status constants
const (
	NotificationStatusRead   = "READ"
	NotificationStatusUnread = "UNREAD"
)

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
