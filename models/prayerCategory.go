package models

import "time"

type PrayerCategory struct {
	Prayer_Category_ID   int       `json:"prayerCategoryId" goqu:"skipinsert"`
	Category_Type        string    `json:"categoryType"`
	Category_Type_ID     int       `json:"categoryTypeId"`
	Category_Name        string    `json:"categoryName"`
	Category_Color       string    `json:"categoryColor"`
	Display_Sequence     int       `json:"displaySequence"`
	Datetime_Create      time.Time `json:"datetimeCreate" goqu:"skipinsert,skipupdate"`
	Datetime_Update      time.Time `json:"datetimeUpdate" goqu:"skipinsert,skipupdate"`
	Created_By           int       `json:"createdBy"`
	Updated_By           int       `json:"updatedBy"`
}

type PrayerCategoryCreate struct {
	Category_Name  string `json:"categoryName" binding:"required"`
	Category_Color string `json:"categoryColor"`
}

type PrayerCategoryUpdate struct {
	Category_Name  string `json:"categoryName"`
	Category_Color string `json:"categoryColor"`
}

type PrayerCategoryReorder struct {
	Category_IDs []int `json:"categoryIds" binding:"required"`
}

type PrayerCategoryItem struct {
	Prayer_Category_Item_ID int       `json:"prayerCategoryItemId" goqu:"skipinsert"`
	Prayer_Category_ID      int       `json:"prayerCategoryId"`
	Prayer_Access_ID        int       `json:"prayerAccessId"`
	Datetime_Create         time.Time `json:"datetimeCreate" goqu:"skipinsert,skipupdate"`
	Created_By              int       `json:"createdBy"`
}
