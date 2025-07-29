package models

import "time"

type UserProfile struct {
	User_Profile_ID      int       `json:"userProfileId" goqu:"skipinsert"`
	Username             string    `json:"username"`
	Password             string    `json:"-"`
	Email                string    `json:"email"`
	First_Name           string    `json:"firstName"`
	Last_Name            string    `json:"lastName"`
	Phone_Number         string    `json:"phoneNumber"`
	Email_Verified       bool      `json:"emailVerified" goqu:"skipinsert"`
	Phone_Verified       bool      `json:"phoneVerified" goqu:"skipinsert"`
	Verification_Token   string    `json:"-"`
	Admin                bool      `json:"admin" goqu:"skipinsert"`
	Created_By           int       `json:"createdBy"`
	Datetime_Create      time.Time `json:"datetimeCreate" goqu:"skipinsert"`
	Updated_By           int       `json:"updatedBy"`
	Datetime_Update      time.Time `json:"datetimeUpdate" goqu:"skipinsert"`
	Deleted              bool      `json:"deleted" goqu:"skipinsert"`
}

type UserProfileSignup struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	Email        string `json:"email"`
	First_Name   string `json:"firstName"`
	Last_Name    string `json:"lastName"`
	Phone_Number string `json:"phoneNumber"`
}
