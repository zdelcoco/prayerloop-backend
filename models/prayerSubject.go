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
