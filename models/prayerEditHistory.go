package models

import "time"

// History action type constants
const (
	// HistoryActionCreated records when a prayer is first created.
	HistoryActionCreated = "created"

	// HistoryActionEdited records when a prayer's content is modified.
	HistoryActionEdited = "edited"

	// HistoryActionAnswered records when a prayer is marked as answered (subset of edited).
	HistoryActionAnswered = "answered"

	// HistoryActionShared records when a prayer is shared to a circle/group.
	HistoryActionShared = "shared"

	// HistoryActionDeleted records when a prayer is deleted.
	HistoryActionDeleted = "deleted"
)

// PrayerEditHistory represents an entry in the prayer_edit_history table.
// Tracks who performed what action on a prayer and when.
type PrayerEditHistory struct {
	Prayer_Edit_History_ID int       `json:"prayerEditHistoryId" goqu:"skipinsert"`
	Prayer_ID              int       `json:"prayerId"`
	User_Profile_ID        int       `json:"userProfileId"`
	Action_Type            string    `json:"actionType"`
	DateTime_Create        time.Time `json:"datetimeCreate" goqu:"skipinsert"`
}
