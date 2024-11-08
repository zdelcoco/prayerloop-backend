package models

type PrayerRequest struct {
	Prayer_Request_ID int `goqu:"skipinsert"`
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
	Datetime_Create   string `goqu:"skipinsert"`
	Updated_By        int
	Datetime_Update   string `goqu:"skipinsert"`
}

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
