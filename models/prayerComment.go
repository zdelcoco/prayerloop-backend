package models

import "time"

// Comment represents a comment on a prayer
type Comment struct {
	Comment_ID      int       `json:"commentId" db:"comment_id" goqu:"skipinsert"`
	Prayer_ID       int       `json:"prayerId" db:"prayer_id"`
	User_Profile_ID int       `json:"userProfileId" db:"user_profile_id"`
	Comment_Text    string    `json:"commentText" db:"comment_text"`
	Is_Private      bool      `json:"isPrivate" db:"is_private"`
	Is_Hidden       bool      `json:"isHidden" db:"is_hidden"`
	DateTime_Create time.Time `json:"datetimeCreate" db:"datetime_create" goqu:"skipinsert"`
	DateTime_Update time.Time `json:"datetimeUpdate" db:"datetime_update" goqu:"skipinsert"`
	Created_By      int       `json:"createdBy" db:"created_by"`
	Updated_By      int       `json:"updatedBy" db:"updated_by"`
}

// CommentCreate represents the request body for creating a comment
type CommentCreate struct {
	Comment_Text string `json:"commentText"`
	Is_Private   bool   `json:"isPrivate"`
}

// CommentWithUser includes commenter information for display purposes
type CommentWithUser struct {
	Comment
	Commenter_Name   string  `json:"commenterName" db:"commenter_name"`
	Commenter_Avatar *string `json:"commenterAvatar,omitempty" db:"commenter_avatar"`
}
