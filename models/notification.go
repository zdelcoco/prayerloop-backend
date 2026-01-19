package models

import "time"

// Notification type constants
const (
	// NotificationTypePrayerCreatedForYou fires when a user creates a prayer for a linked subject.
	// Recipient: The linked subject (prayer_subject.user_profile_id).
	NotificationTypePrayerCreatedForYou = "PRAYER_CREATED_FOR_YOU"

	// NotificationTypePrayerEditedBySubject fires when a linked subject edits a prayer about them.
	// Recipient: The prayer creator (prayer.created_by).
	NotificationTypePrayerEditedBySubject = "PRAYER_EDITED_BY_SUBJECT"

	// NotificationTypePrayerShared fires when a prayer is shared to a circle/group.
	// Recipients: All other members of the circle/group.
	NotificationTypePrayerShared = "PRAYER_SHARED"

	// NotificationTypeGroupInvite fires when a user is invited to join a group.
	// Recipient: The invited user.
	NotificationTypeGroupInvite = "GROUP_INVITE"

	// NotificationTypeGroupMemberJoined fires when a user accepts a group invitation.
	// Recipients: All existing group members.
	NotificationTypeGroupMemberJoined = "GROUP_MEMBER_JOINED"

	// NotificationTypePrayerRemovedFromGroup fires when a linked subject removes a prayer from a group.
	// Recipient: The prayer creator.
	NotificationTypePrayerRemovedFromGroup = "PRAYER_REMOVED_FROM_GROUP"
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
	Target_Prayer_ID     *int      `json:"targetPrayerId" goqu:"skipupdate"`
	Target_Group_ID      *int      `json:"targetGroupId" goqu:"skipupdate"`
}
