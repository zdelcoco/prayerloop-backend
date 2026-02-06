package models

import "time"

type PrayerAnalytics struct {
	Prayer_Analytics_ID  int        `json:"prayerAnalyticsId" db:"prayer_analytics_id"`
	Prayer_ID            int        `json:"prayerId" db:"prayer_id"`
	Total_Prayers        int        `json:"totalPrayers" db:"total_prayers"`
	Datetime_Last_Prayed *time.Time `json:"datetimeLastPrayed" db:"datetime_last_prayed"`
	Last_Prayed_By       *int       `json:"lastPrayedBy" db:"last_prayed_by"`
	Num_Unique_Users     int        `json:"numUniqueUsers" db:"num_unique_users"`
	Num_Shares           int        `json:"numShares" db:"num_shares"`
}

// PrayerAnalyticsResponse is the response type for GET endpoint (subset of fields)
type PrayerAnalyticsResponse struct {
	TotalPrayers   int `json:"totalPrayers"`
	NumUniqueUsers int `json:"numUniqueUsers"`
}
