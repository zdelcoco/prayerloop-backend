package services

import (
	"fmt"
	"log"
	"strconv"

	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"
	"github.com/doug-martin/goqu/v9"
)

// shouldSendDebounced checks if a notification should be sent based on debounce window.
// Uses atomic upsert to prevent race conditions. Also cleans up old records (>24h).
// Returns true if notification should be sent.
func shouldSendDebounced(notifType string, targetUserID int, entityID int, windowMinutes int) bool {
	// Lazy cleanup of old records (older than 24 hours)
	_, cleanupErr := initializers.DB.Delete("notification_debounce").
		Where(goqu.L("last_triggered_at < NOW() - INTERVAL '24 hours'")).
		Executor().Exec()
	if cleanupErr != nil {
		log.Printf("Error cleaning up old debounce records: %v", cleanupErr)
	}

	// Atomic upsert that returns whether notification should be sent
	// Uses INSERT...ON CONFLICT DO UPDATE with a WHERE clause that only updates
	// if outside the debounce window. RETURNING tells us if update happened.
	query := `
		INSERT INTO notification_debounce (notification_type, target_user_id, entity_id, last_triggered_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (notification_type, target_user_id, entity_id)
		DO UPDATE SET last_triggered_at = NOW()
		WHERE notification_debounce.last_triggered_at < NOW() - ($4 || ' minutes')::INTERVAL
		RETURNING debounce_id
	`

	var debounceID int
	err := initializers.DB.QueryRow(query, notifType, targetUserID, entityID, windowMinutes).Scan(&debounceID)

	if err != nil {
		// No rows returned means either:
		// 1. Record exists and is within window (DO UPDATE WHERE clause failed)
		// 2. Database error
		// Check if it's a "no rows" situation vs actual error
		if err.Error() == "sql: no rows in result set" {
			return false // Within debounce window
		}
		log.Printf("Error in debounce check: %v", err)
		return true // On error, allow notification
	}

	return true // Row was inserted/updated, send notification
}

// NotifySubjectOfPrayerCreated sends PRAYER_CREATED_FOR_YOU to a linked subject.
// Called when a prayer is shared to a circle and has a linked subject.
func NotifySubjectOfPrayerCreated(
	subjectUserID int,
	prayerID int,
	groupID int,
	actorID int,
	actorName string,
	groupName string,
) {
	// Don't notify if subject is the actor (creating prayer about themselves)
	if subjectUserID == actorID {
		return
	}

	// CRITICAL: Don't notify if subject is not a member of the circle
	// This prevents privacy leaks where subjects learn about circles they're not in
	var memberCount int
	checkQuery := initializers.DB.From("user_group").
		Select(goqu.COUNT("*")).
		Where(
			goqu.And(
				goqu.C("group_profile_id").Eq(groupID),
				goqu.C("user_profile_id").Eq(subjectUserID),
				goqu.C("is_active").IsTrue(),
			),
		)

	_, memberCheckErr := checkQuery.Executor().ScanVal(&memberCount)
	if memberCheckErr != nil || memberCount == 0 {
		// Subject is not a member of this circle - don't notify them
		// This maintains privacy: subjects shouldn't know about circles they're not in
		return
	}

	notificationMessage := fmt.Sprintf("%s created a prayer for you in %s", actorName, groupName)

	// Create notification record with target for navigation
	notification := models.Notification{
		User_Profile_ID:      subjectUserID,
		Notification_Type:    models.NotificationTypePrayerCreatedForYou,
		Notification_Message: notificationMessage,
		Notification_Status:  models.NotificationStatusUnread,
		Created_By:           actorID,
		Updated_By:           actorID,
		Target_Prayer_ID:     &prayerID,
		Target_Group_ID:      &groupID,
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
			"groupId":  strconv.Itoa(groupID),
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

	// Create notification records in database for each member with navigation targets
	for _, memberID := range memberIDs {
		notification := models.Notification{
			User_Profile_ID:      memberID,
			Notification_Type:    models.NotificationTypePrayerShared,
			Notification_Message: notificationMessage,
			Notification_Status:  models.NotificationStatusUnread,
			Created_By:           actorID,
			Updated_By:           actorID,
			Target_Prayer_ID:     &prayerID,
			Target_Group_ID:      &groupID,
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

// NotifyCreatorOfPrayerRemovedFromGroup sends PRAYER_REMOVED_FROM_GROUP to the prayer creator.
// Called when a linked subject removes a prayer from a group they didn't create.
func NotifyCreatorOfPrayerRemovedFromGroup(
	creatorID int,
	prayerID int,
	groupID int,
	subjectUserID int,
	subjectName string,
	groupName string,
) {
	// Don't notify if creator is the one removing (they already know)
	if creatorID == subjectUserID {
		return
	}

	notificationMessage := fmt.Sprintf("%s removed a prayer you made for them from %s", subjectName, groupName)

	// Create notification record with target for navigation
	notification := models.Notification{
		User_Profile_ID:      creatorID,
		Notification_Type:    models.NotificationTypePrayerRemovedFromGroup,
		Notification_Message: notificationMessage,
		Notification_Status:  models.NotificationStatusUnread,
		Created_By:           subjectUserID,
		Updated_By:           subjectUserID,
		Target_Prayer_ID:     &prayerID,
		Target_Group_ID:      &groupID,
	}

	insert := initializers.DB.Insert("notification").Rows(notification)
	_, err := insert.Executor().Exec()
	if err != nil {
		log.Printf("Failed to create PRAYER_REMOVED_FROM_GROUP notification for user %d: %v", creatorID, err)
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
			"type":     "prayer_removed_from_group",
			"prayerId": strconv.Itoa(prayerID),
			"groupId":  strconv.Itoa(groupID),
		},
	}

	err = pushService.SendNotificationToUser(creatorID, payload)
	if err != nil {
		log.Printf("Failed to send PRAYER_REMOVED_FROM_GROUP push notification: %v", err)
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

	// Find a shared group for better navigation context
	sharedGroupID := getSharedGroupForCommentNotification(prayerID, subjectUserID, creatorID)

	notificationMessage := fmt.Sprintf("%s edited a prayer about them", subjectName)

	// Create notification record with target for navigation
	notification := models.Notification{
		User_Profile_ID:      creatorID,
		Notification_Type:    models.NotificationTypePrayerEditedBySubject,
		Notification_Message: notificationMessage,
		Notification_Status:  models.NotificationStatusUnread,
		Created_By:           subjectUserID,
		Updated_By:           subjectUserID,
		Target_Prayer_ID:     &prayerID,
		Target_Group_ID:      sharedGroupID,
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

	// Include groupId in push notification if we found a shared group
	if sharedGroupID != nil {
		payload.Data["groupId"] = strconv.Itoa(*sharedGroupID)
	}

	err = pushService.SendNotificationToUser(creatorID, payload)
	if err != nil {
		log.Printf("Failed to send PRAYER_EDITED_BY_SUBJECT push notification: %v", err)
	}
}

// getSharedGroupForCommentNotification finds a group where both the commenter and recipient are members
// and where the prayer is shared. Returns the first matching group ID, or nil if none found.
func getSharedGroupForCommentNotification(prayerID int, commenterID int, recipientID int) *int {
	// Query to find groups where:
	// 1. The prayer is shared (via prayer_access with access_type='group')
	// 2. Both commenter and recipient are active members
	query := `
		SELECT DISTINCT pa.access_type_id AS group_id
		FROM prayer_access pa
		JOIN user_group ug1 ON pa.access_type_id = ug1.group_profile_id
		JOIN user_group ug2 ON pa.access_type_id = ug2.group_profile_id
		WHERE pa.prayer_id = $1
		  AND pa.access_type = 'group'
		  AND ug1.user_profile_id = $2
		  AND ug1.is_active = TRUE
		  AND ug2.user_profile_id = $3
		  AND ug2.is_active = TRUE
		LIMIT 1
	`

	var groupID int
	err := initializers.DB.QueryRow(query, prayerID, commenterID, recipientID).Scan(&groupID)
	if err != nil {
		// No shared group found, or database error
		return nil
	}

	return &groupID
}

// NotifyUsersOfNewComment notifies prayer creator, subject, and previous commenters of new comment.
// Debounced with 15-minute window to prevent notification spam from rapid comments.
func NotifyUsersOfNewComment(prayerID int, commentID int, commenterID int) {
	// Get commenter name for notification message
	var commenterName string
	_, _ = initializers.DB.From("user_profile").
		Select("first_name").
		Where(goqu.C("user_profile_id").Eq(commenterID)).
		ScanVal(&commenterName)

	if commenterName == "" {
		commenterName = "Someone"
	}

	// 1. Get prayer creator
	var prayer models.Prayer
	_, err := initializers.DB.From("prayer").
		Where(goqu.C("prayer_id").Eq(prayerID)).
		ScanStruct(&prayer)
	if err != nil {
		log.Printf("Failed to fetch prayer for comment notification: %v", err)
		return
	}

	recipientIDs := []int{}

	// 2. Add creator if not the commenter
	if prayer.Created_By != commenterID {
		recipientIDs = append(recipientIDs, prayer.Created_By)
	}

	// 3. Add linked subject user if exists and not the commenter
	if prayer.Prayer_Subject_ID != nil {
		var subjectUserID *int
		_, _ = initializers.DB.From("prayer_subject").
			Select("user_profile_id").
			Where(goqu.C("prayer_subject_id").Eq(*prayer.Prayer_Subject_ID)).
			ScanVal(&subjectUserID)

		if subjectUserID != nil && *subjectUserID != commenterID {
			recipientIDs = append(recipientIDs, *subjectUserID)
		}
	}

	// 4. Add previous commenters (excluding current commenter and already-added users)
	var previousCommenters []int
	_ = initializers.DB.From("prayer_comment").
		Select(goqu.DISTINCT("user_profile_id")).
		Where(
			goqu.And(
				goqu.C("prayer_id").Eq(prayerID),
				goqu.C("comment_id").Neq(commentID), // Exclude this comment
				goqu.C("user_profile_id").Neq(commenterID), // Exclude commenter
			),
		).
		ScanVals(&previousCommenters)

	// Deduplicate: only add if not already in recipientIDs
	for _, prevCommenter := range previousCommenters {
		alreadyAdded := false
		for _, existing := range recipientIDs {
			if existing == prevCommenter {
				alreadyAdded = true
				break
			}
		}
		if !alreadyAdded {
			recipientIDs = append(recipientIDs, prevCommenter)
		}
	}

	// 5. For each recipient, check debounce and create notification
	for _, recipientID := range recipientIDs {
		// Check 15-minute debounce window
		if !shouldSendDebounced(models.NotificationTypePrayerCommentAdded, recipientID, prayerID, 15) {
			log.Printf("Debounced comment notification for user %d, prayer %d", recipientID, prayerID)
			continue
		}

		// Find a shared group for better navigation context
		sharedGroupID := getSharedGroupForCommentNotification(prayerID, commenterID, recipientID)

		notificationMessage := fmt.Sprintf("%s commented on a prayer", commenterName)

		notification := models.Notification{
			User_Profile_ID:      recipientID,
			Notification_Type:    models.NotificationTypePrayerCommentAdded,
			Notification_Message: notificationMessage,
			Notification_Status:  models.NotificationStatusUnread,
			Target_Prayer_ID:     &prayerID,
			Target_Comment_ID:    &commentID,
			Target_Group_ID:      sharedGroupID, // Include group context if found
			Created_By:           commenterID,
			Updated_By:           commenterID,
		}

		insert := initializers.DB.Insert("notification").Rows(notification)
		_, insertErr := insert.Executor().Exec()
		if insertErr != nil {
			log.Printf("Failed to create comment notification: %v", insertErr)
		} else {
			// Successfully created notification, send push
			pushService := GetPushNotificationService()
			if pushService != nil {
				payload := NotificationPayload{
					Title: "New Comment",
					Body:  notificationMessage,
					Data: map[string]string{
						"type":      models.NotificationTypePrayerCommentAdded,
						"prayerId":  strconv.Itoa(prayerID),
						"commentId": strconv.Itoa(commentID),
					},
				}

				// Include groupId in push notification if we found a shared group
				if sharedGroupID != nil {
					payload.Data["groupId"] = strconv.Itoa(*sharedGroupID)
				}

				err = pushService.SendNotificationToUser(recipientID, payload)
				if err != nil {
					log.Printf("Failed to send comment push notification: %v", err)
				}
			}
		}
	}
}
