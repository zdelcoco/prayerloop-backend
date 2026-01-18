package services

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"
	"github.com/doug-martin/goqu/v9"
)

// shouldSendDebounced checks if a notification should be sent based on debounce window.
// Uses atomic upsert to prevent race conditions. Also cleans up old records (>24h).
// Returns true if notification should be sent.
func shouldSendDebounced(notifType string, targetUserID int, entityID int, windowMinutes int) bool {
	// First, lazy cleanup of old records (older than 24 hours)
	initializers.DB.Delete("notification_debounce").
		Where(goqu.L("last_triggered_at < NOW() - INTERVAL '24 hours'")).
		Executor().Exec()

	// Check if within debounce window using atomic upsert
	// ON CONFLICT updates last_triggered_at only if outside window, returns whether update happened
	var lastTriggered time.Time
	found, err := initializers.DB.From("notification_debounce").
		Select("last_triggered_at").
		Where(
			goqu.C("notification_type").Eq(notifType),
			goqu.C("target_user_id").Eq(targetUserID),
			goqu.C("entity_id").Eq(entityID),
		).
		ScanVal(&lastTriggered)

	if err != nil {
		log.Printf("Error checking debounce: %v", err)
		return true // On error, allow notification
	}

	if found && time.Since(lastTriggered) < time.Duration(windowMinutes)*time.Minute {
		return false // Within debounce window
	}

	// Insert or update the debounce record
	_, err = initializers.DB.Insert("notification_debounce").
		Rows(goqu.Record{
			"notification_type": notifType,
			"target_user_id":    targetUserID,
			"entity_id":         entityID,
			"last_triggered_at": time.Now(),
		}).
		OnConflict(goqu.DoUpdate("notification_type, target_user_id, entity_id",
			goqu.Record{"last_triggered_at": time.Now()})).
		Executor().Exec()

	if err != nil {
		log.Printf("Error upserting debounce record: %v", err)
	}

	return true
}

// NotifySubjectOfPrayerCreated sends PRAYER_CREATED_FOR_YOU to a linked subject.
// Called when a prayer is shared to a circle and has a linked subject.
func NotifySubjectOfPrayerCreated(
	subjectUserID int,
	prayerID int,
	actorID int,
	actorName string,
	groupName string,
) {
	// Don't notify if subject is the actor (creating prayer about themselves)
	if subjectUserID == actorID {
		return
	}

	notificationMessage := fmt.Sprintf("%s created a prayer for you in %s", actorName, groupName)

	// Create notification record
	notification := models.Notification{
		User_Profile_ID:      subjectUserID,
		Notification_Type:    models.NotificationTypePrayerCreatedForYou,
		Notification_Message: notificationMessage,
		Notification_Status:  models.NotificationStatusUnread,
		Created_By:           actorID,
		Updated_By:           actorID,
	}

	insert := initializers.DB.Insert("notification").Rows(notification)
	_, err := insert.Executor().Exec()
	if err != nil {
		log.Printf("Failed to create PRAYER_CREATED_FOR_YOU notification for user %d: %v", subjectUserID, err)
	}

	// Send push notification
	pushService := GetPushNotificationService()
	if pushService == nil {
		log.Println("Push notification service not available")
		return
	}

	payload := NotificationPayload{
		Title: groupName,
		Body:  notificationMessage,
		Data: map[string]string{
			"type":     "prayer_created_for_you",
			"prayerId": strconv.Itoa(prayerID),
		},
	}

	err = pushService.SendNotificationToUser(subjectUserID, payload)
	if err != nil {
		log.Printf("Failed to send PRAYER_CREATED_FOR_YOU push notification: %v", err)
	}
}

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
				goqu.L("COALESCE(mute_notifications, FALSE) = FALSE"),
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

// NotifyCreatorOfSubjectEdit sends PRAYER_EDITED_BY_SUBJECT to the prayer creator.
// Debounced with 15-minute window to prevent notification spam from rapid edits.
func NotifyCreatorOfSubjectEdit(
	creatorID int,
	prayerID int,
	subjectUserID int,
	subjectName string,
) {
	// Check debounce - 15 minute window
	if !shouldSendDebounced(models.NotificationTypePrayerEditedBySubject, creatorID, prayerID, 15) {
		log.Printf("Debounced PRAYER_EDITED_BY_SUBJECT notification for creator %d, prayer %d", creatorID, prayerID)
		return
	}

	notificationMessage := fmt.Sprintf("%s edited a prayer about them", subjectName)

	// Create notification record
	notification := models.Notification{
		User_Profile_ID:      creatorID,
		Notification_Type:    models.NotificationTypePrayerEditedBySubject,
		Notification_Message: notificationMessage,
		Notification_Status:  models.NotificationStatusUnread,
		Created_By:           subjectUserID,
		Updated_By:           subjectUserID,
	}

	insert := initializers.DB.Insert("notification").Rows(notification)
	_, err := insert.Executor().Exec()
	if err != nil {
		log.Printf("Failed to create PRAYER_EDITED_BY_SUBJECT notification for user %d: %v", creatorID, err)
	}

	// Send push notification
	pushService := GetPushNotificationService()
	if pushService == nil {
		log.Println("Push notification service not available")
		return
	}

	payload := NotificationPayload{
		Title: "Prayer Edited",
		Body:  notificationMessage,
		Data: map[string]string{
			"type":     "prayer_edited_by_subject",
			"prayerId": strconv.Itoa(prayerID),
		},
	}

	err = pushService.SendNotificationToUser(creatorID, payload)
	if err != nil {
		log.Printf("Failed to send PRAYER_EDITED_BY_SUBJECT push notification: %v", err)
	}
}
