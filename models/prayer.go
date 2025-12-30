package models

import "time"

type Prayer struct {
	Prayer_ID          int        `json:"prayerId" goqu:"skipinsert"`
	Prayer_Type        string     `json:"prayerType"`
	Is_Private         *bool      `json:"isPrivate"`
	Title              string     `json:"title"`
	Prayer_Description string     `json:"prayerDescription"`
	Is_Answered        *bool      `json:"isAnswered"`
	Prayer_Priority    *int       `json:"prayerPriority"`
	Prayer_Subject_ID  *int       `json:"prayerSubjectId"`
	Datetime_Answered  *time.Time `json:"datetimeAnswered"`
	Created_By         int        `json:"createdBy"`
	Datetime_Create    time.Time  `json:"datetimeCreate" goqu:"skipinsert"`
	Updated_By         int        `json:"updatedBy"`
	Datetime_Update    time.Time  `json:"datetimeUpdate" goqu:"skipinsert"`
	Deleted            bool       `json:"deleted" goqu:"skipinsert"`
}

type UserPrayer struct {
	User_Profile_ID      int        `json:"userProfileId" goqu:"skipinsert"`
	Prayer_ID            int        `json:"prayerId" goqu:"skipinsert"`
	Prayer_Access_ID     int        `json:"prayerAccessId" goqu:"skipinsert"`
	Display_Sequence     int        `json:"displaySequence" goqu:"skipinsert"`
	Prayer_Type          string     `json:"prayerType"`
	Is_Private           *bool      `json:"isPrivate"`
	Title                string     `json:"title"`
	Prayer_Description   string     `json:"prayerDescription"`
	Is_Answered          *bool      `json:"isAnswered"`
	Prayer_Priority      *int       `json:"prayerPriority"`
	Prayer_Subject_ID    *int       `json:"prayerSubjectId" goqu:"skipinsert"`
	Datetime_Answered    *time.Time `json:"datetimeAnswered"`
	Created_By           int        `json:"createdBy"`
	Datetime_Create      time.Time  `json:"datetimeCreate" goqu:"skipinsert"`
	Updated_By           int        `json:"updatedBy"`
	Datetime_Update      time.Time  `json:"datetimeUpdate" goqu:"skipinsert"`
	Deleted              bool       `json:"deleted" goqu:"skipinsert"`
	Prayer_Category_ID   *int       `json:"prayerCategoryId,omitempty" goqu:"skipinsert"`
	Category_Name        *string    `json:"categoryName,omitempty" goqu:"skipinsert"`
	Category_Color       *string    `json:"categoryColor,omitempty" goqu:"skipinsert"`
	Category_Display_Seq *int       `json:"categoryDisplaySequence,omitempty" goqu:"skipinsert"`
}

type PrayerCreate struct {
	Prayer_Type        string     `json:"prayerType"`
	Is_Private         *bool      `json:"isPrivate"`
	Title              string     `json:"title"`
	Prayer_Description string     `json:"prayerDescription"`
	Is_Answered        *bool      `json:"isAnswered"`
	Datetime_Answered  *time.Time `json:"datetimeAnswered"`
	Prayer_Priority    *int       `json:"prayerPriority"`
	Prayer_Subject_ID  *int       `json:"prayerSubjectId"`
}

type PrayerAccess struct {
	Prayer_Access_ID  int       `json:"prayerAccessId" goqu:"skipinsert"`
	Prayer_ID         int       `json:"prayerId"`
	Access_Type       string    `json:"accessType"`
	Access_Type_ID    int       `json:"accessTypeId"`
	Display_Sequence  int       `json:"displaySequence"`
	Datetime_Create   time.Time `json:"datetimeCreate" goqu:"skipinsert"`
	Datetime_Update   time.Time `json:"datetimeUpdate" goqu:"skipinsert"`
	Created_By        int       `json:"createdBy"`
	Updated_By        int       `json:"updatedBy"`
}

type PrayerAccessCreate struct {
	Access_Type    string `json:"accessType"`
	Access_Type_ID int    `json:"accessTypeId"`
}
