package services

import (
	"fmt"
	"log"
	"strconv"

	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"
	"github.com/doug-martin/goqu/v9"
)

// GetCircleMembersForNotification returns active circle members excluding specified users
// and respecting mute_notifications preferences.
func GetCircleMembersForNotification(groupID int, excludeUserIDs []int) ([]int, error) {
	var userIDs []int

	query := initializers.DB.From("user_group").
		Select("user_profile_id").
		Where(
			goqu.And(
				goqu.C("group_profile_id").Eq(groupID),
				goqu.C("is_active").IsTrue(),
				goqu.C("mute_notifications").IsFalse(),
			),
		)

	// Add exclusions if any
	if len(excludeUserIDs) > 0 {
		query = query.Where(goqu.C("user_profile_id").NotIn(excludeUserIDs))
	}

	err := query.ScanVals(&userIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get circle members for notification: %v", err)
	}

	return userIDs, nil
}

// NotifyCircleOfPrayerShared sends PRAYER_SHARED notification to circle members.
// Excludes: actor, prayer creator, and optionally the linked subject.
// actorName should be the display name (first_name or username) of the actor.
func NotifyCircleOfPrayerShared(
	groupID int,
	groupName string,
	actorID int,
	actorName string,
	prayerID int,
	prayerCreatorID int,
	linkedSubjectUserID *int,
) {
	// Build exclusion list: actor and prayer creator
	excludeIDs := []int{actorID}
	if prayerCreatorID != actorID {
		excludeIDs = append(excludeIDs, prayerCreatorID)
	}
	// Also exclude linked subject (they get a different notification)
	if linkedSubjectUserID != nil && *linkedSubjectUserID != actorID && *linkedSubjectUserID != prayerCreatorID {
		excludeIDs = append(excludeIDs, *linkedSubjectUserID)
	}

	memberIDs, err := GetCircleMembersForNotification(groupID, excludeIDs)
	if err != nil {
		log.Printf("Failed to get circle members for notification: %v", err)
		return
	}

	if len(memberIDs) == 0 {
		return
	}

	notificationMessage := fmt.Sprintf("%s shared a prayer with %s", actorName, groupName)

	// Create notification records in database for each member
	for _, memberID := range memberIDs {
		notification := models.Notification{
			User_Profile_ID:      memberID,
			Notification_Type:    models.NotificationTypePrayerShared,
			Notification_Message: notificationMessage,
			Notification_Status:  models.NotificationStatusUnread,
			Created_By:           actorID,
			Updated_By:           actorID,
		}

		insert := initializers.DB.Insert("notification").Rows(notification)
		_, err := insert.Executor().Exec()
		if err != nil {
			log.Printf("Failed to create PRAYER_SHARED notification for user %d: %v", memberID, err)
		}
	}

	// Send push notifications
	pushService := GetPushNotificationService()
	if pushService == nil {
		log.Println("Push notification service not available")
		return
	}

	payload := NotificationPayload{
		Title: groupName,
		Body:  notificationMessage,
		Data: map[string]string{
			"type":     "prayer_shared",
			"groupId":  strconv.Itoa(groupID),
			"prayerId": strconv.Itoa(prayerID),
		},
	}

	err = pushService.SendNotificationToUsers(memberIDs, payload)
	if err != nil {
		log.Printf("Failed to send PRAYER_SHARED push notifications: %v", err)
	}
}
