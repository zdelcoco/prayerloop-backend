package models

type PrayerRequestCreate struct {
	User_ID           int
	Group_ID          int
	Is_Private        *bool
	Title             string
	Description       string
	Is_Answered       *bool
	Is_Ongoing        *bool
	Priority          *int
	Datetime_Answered *string
	Created_By        int
	Updated_By        int
}
