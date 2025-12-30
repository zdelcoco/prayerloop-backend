package models

import "time"

type PrayerSubject struct {
	Prayer_Subject_ID           int       `json:"prayerSubjectId" goqu:"skipinsert"`
	Prayer_Subject_Type         string    `json:"prayerSubjectType"`
	Prayer_Subject_Display_Name string    `json:"prayerSubjectDisplayName"`
	Notes                       *string   `json:"notes"`
	Display_Sequence            int       `json:"displaySequence"`
	Photo_S3_Key                *string   `json:"photoS3Key"`
	User_Profile_ID             *int      `json:"userProfileId"`
	Use_Linked_User_Photo       bool      `json:"useLinkedUserPhoto"`
	Link_Status                 string    `json:"linkStatus"`
	Datetime_Create             time.Time `json:"datetimeCreate" goqu:"skipinsert"`
	Datetime_Update             time.Time `json:"datetimeUpdate" goqu:"skipinsert"`
	Created_By                  int       `json:"createdBy"`
	Updated_By                  int       `json:"updatedBy"`
}

// PrayerSubjectWithPrayers is the response type that groups prayers under their subject
type PrayerSubjectWithPrayers struct {
	Prayer_Subject_ID           int           `json:"prayerSubjectId"`
	Prayer_Subject_Type         string        `json:"prayerSubjectType"`
	Prayer_Subject_Display_Name string        `json:"prayerSubjectDisplayName"`
	Notes                       *string       `json:"notes"`
	Display_Sequence            int           `json:"displaySequence"`
	Photo_S3_Key                *string       `json:"photoS3Key"`
	User_Profile_ID             *int          `json:"userProfileId"`
	Use_Linked_User_Photo       bool          `json:"useLinkedUserPhoto"`
	Link_Status                 string        `json:"linkStatus"`
	Datetime_Create             time.Time     `json:"datetimeCreate"`
	Datetime_Update             time.Time     `json:"datetimeUpdate"`
	Created_By                  int           `json:"createdBy"`
	Updated_By                  int           `json:"updatedBy"`
	Prayers                     []UserPrayer  `json:"prayers"`
}

// PrayerSubjectCreate is the input type for creating a new prayer subject
type PrayerSubjectCreate struct {
	Prayer_Subject_Type         string  `json:"prayerSubjectType"`
	Prayer_Subject_Display_Name string  `json:"prayerSubjectDisplayName"`
	Notes                       *string `json:"notes"`
	Photo_S3_Key                *string `json:"photoS3Key"`
	User_Profile_ID             *int    `json:"userProfileId"`
	Use_Linked_User_Photo       *bool   `json:"useLinkedUserPhoto"`
}

// PrayerSubjectUpdate is the input type for updating a prayer subject
type PrayerSubjectUpdate struct {
	Prayer_Subject_Display_Name *string `json:"prayerSubjectDisplayName"`
	Notes                       *string `json:"notes"`
	Photo_S3_Key                *string `json:"photoS3Key"`
	Use_Linked_User_Photo       *bool   `json:"useLinkedUserPhoto"`
}

// PrayerSubjectMembership represents a member in a family/group prayer subject
type PrayerSubjectMembership struct {
	Prayer_Subject_Membership_ID int       `json:"prayerSubjectMembershipId" goqu:"skipinsert"`
	Member_Prayer_Subject_ID     int       `json:"memberPrayerSubjectId"`
	Group_Prayer_Subject_ID      int       `json:"groupPrayerSubjectId"`
	Membership_Role              *string   `json:"membershipRole"`
	Datetime_Create              time.Time `json:"datetimeCreate" goqu:"skipinsert"`
	Created_By                   int       `json:"createdBy"`
}

// PrayerSubjectMembershipCreate is the input type for adding a member to a family/group
type PrayerSubjectMembershipCreate struct {
	Member_Prayer_Subject_ID int     `json:"memberPrayerSubjectId"`
	Membership_Role          *string `json:"membershipRole"`
}

// PrayerSubjectMemberDetail includes member details for API responses
type PrayerSubjectMemberDetail struct {
	Prayer_Subject_Membership_ID int       `json:"prayerSubjectMembershipId"`
	Member_Prayer_Subject_ID     int       `json:"memberPrayerSubjectId"`
	Membership_Role              *string   `json:"membershipRole"`
	Datetime_Create              time.Time `json:"datetimeCreate"`
	Created_By                   int       `json:"createdBy"`
	// Member details
	Member_Display_Name    string  `json:"memberDisplayName"`
	Member_Type            string  `json:"memberType"`
	Member_Photo_S3_Key    *string `json:"memberPhotoS3Key"`
	Member_User_Profile_ID *int    `json:"memberUserProfileId"`
}

// ConnectionRequest represents a request to link a prayer subject to a user
type ConnectionRequest struct {
	Request_ID         int        `json:"requestId" goqu:"skipinsert"`
	Requester_ID       int        `json:"requesterId"`
	Target_User_ID     int        `json:"targetUserId"`
	Prayer_Subject_ID  int        `json:"prayerSubjectId"`
	Status             string     `json:"status"`
	Datetime_Create    time.Time  `json:"datetimeCreate" goqu:"skipinsert"`
	Datetime_Responded *time.Time `json:"datetimeResponded"`
}

// ConnectionRequestCreate is the input for sending a connection request
type ConnectionRequestCreate struct {
	Target_User_ID    int `json:"targetUserId"`
	Prayer_Subject_ID int `json:"prayerSubjectId"`
}

// ConnectionRequestResponse is the input for accepting/declining a request
type ConnectionRequestResponse struct {
	Status string `json:"status"` // "accepted" or "declined"
}

// ConnectionRequestDetail includes full details for API responses
type ConnectionRequestDetail struct {
	Request_ID         int        `json:"requestId"`
	Requester_ID       int        `json:"requesterId"`
	Target_User_ID     int        `json:"targetUserId"`
	Prayer_Subject_ID  int        `json:"prayerSubjectId"`
	Status             string     `json:"status"`
	Datetime_Create    time.Time  `json:"datetimeCreate"`
	Datetime_Responded *time.Time `json:"datetimeResponded"`
	// Requester details
	Requester_First_Name   string  `json:"requesterFirstName"`
	Requester_Last_Name    string  `json:"requesterLastName"`
	Requester_Email        string  `json:"requesterEmail"`
	Requester_Phone_Number *string `json:"requesterPhoneNumber"`
}

// UserSearchResult is the response when searching for users
type UserSearchResult struct {
	User_Profile_ID int    `json:"userProfileId"`
	First_Name      string `json:"firstName"`
	Last_Name       string `json:"lastName"`
	Username        string `json:"username"`
}
