package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"
	"github.com/doug-martin/goqu/v9"
	
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type PushNotificationService struct {
	fcmClient *messaging.Client
}

type NotificationPayload struct {
	Title    string                 `json:"title"`
	Body     string                 `json:"body"`
	Data     map[string]string      `json:"data,omitempty"`
	Sound    string                 `json:"sound,omitempty"`
	Badge    string                 `json:"badge,omitempty"`
	Priority string                 `json:"priority,omitempty"`
}

var pushService *PushNotificationService

func InitPushNotificationService() {
	pushService = &PushNotificationService{}

	// Initialize Firebase Admin SDK
	serviceAccountPath := os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH")
	
	var app *firebase.App
	var err error

	if serviceAccountPath != "" {
		// Use service account file
		opt := option.WithCredentialsFile(serviceAccountPath)
		app, err = firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			log.Printf("Failed to initialize Firebase app with service account: %v", err)
			return
		}
		log.Println("Firebase initialized with service account file")
	} else {
		// Use Application Default Credentials (ADC)
		app, err = firebase.NewApp(context.Background(), nil)
		if err != nil {
			log.Printf("Failed to initialize Firebase app with ADC: %v", err)
			return
		}
		log.Println("Firebase initialized with Application Default Credentials")
	}

	// Get messaging client
	pushService.fcmClient, err = app.Messaging(context.Background())
	if err != nil {
		log.Printf("Failed to get Firebase messaging client: %v", err)
		return
	}

	log.Println("Push notification service initialized successfully with FCM")
}

func GetPushNotificationService() *PushNotificationService {
	return pushService
}

func (s *PushNotificationService) SendNotificationToUser(userID int, payload NotificationPayload) error {
	// Get user's push tokens from database
	var tokens []models.PushToken
	query := initializers.DB.From("user_push_tokens").
		Where(goqu.C("user_profile_id").Eq(userID))

	err := query.ScanStructs(&tokens)
	if err != nil {
		return fmt.Errorf("failed to get push tokens for user %d: %v", userID, err)
	}

	if len(tokens) == 0 {
		return fmt.Errorf("no push tokens found for user %d", userID)
	}

	// Send notification to each token
	for _, token := range tokens {
		err := s.sendToToken(token, payload)
		if err != nil {
			log.Printf("Failed to send notification to token %s: %v", token.PushToken, err)
			// Continue with other tokens even if one fails
		}
	}

	return nil
}

func (s *PushNotificationService) SendNotificationToUsers(userIDs []int, payload NotificationPayload) error {
	var allErrors []error

	for _, userID := range userIDs {
		err := s.SendNotificationToUser(userID, payload)
		if err != nil {
			allErrors = append(allErrors, err)
			log.Printf("Failed to send notification to user %d: %v", userID, err)
		}
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("failed to send notifications to %d users", len(allErrors))
	}

	return nil
}

func (s *PushNotificationService) sendToToken(pushToken models.PushToken, payload NotificationPayload) error {
	// Check if this is an Expo token (for Expo Go testing)
	if strings.HasPrefix(pushToken.PushToken, "ExponentPushToken[") {
		return s.sendExpoNotification(pushToken, payload)
	}

	if s.fcmClient == nil {
		return fmt.Errorf("FCM client not initialized")
	}

	// Build the FCM message
	message := &messaging.Message{
		Token: pushToken.PushToken,
		Notification: &messaging.Notification{
			Title: payload.Title,
			Body:  payload.Body,
		},
		Data: payload.Data,
	}

	// Platform-specific configuration
	if pushToken.Platform == "ios" {
		message.APNS = &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: payload.Title,
						Body:  payload.Body,
					},
					Sound: payload.Sound,
				},
			},
		}

		// Set badge if provided
		if payload.Badge != "" {
			if badgeNum, err := strconv.Atoi(payload.Badge); err == nil {
				message.APNS.Payload.Aps.Badge = &badgeNum
			}
		}

		// Set priority
		if payload.Priority == "high" {
			message.APNS.Headers = map[string]string{
				"apns-priority": "10",
			}
		}
	} else if pushToken.Platform == "android" {
		message.Android = &messaging.AndroidConfig{
			Notification: &messaging.AndroidNotification{
				Title: payload.Title,
				Body:  payload.Body,
				Sound: payload.Sound,
			},
		}

		// Set priority
		if payload.Priority == "high" {
			message.Android.Priority = "high"
		} else {
			message.Android.Priority = "normal"
		}
	}

	// Send the message
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("Sending FCM message to token: %s", pushToken.PushToken)
	log.Printf("FCM message payload: %+v", message)
	
	response, err := s.fcmClient.Send(ctx, message)
	if err != nil {
		log.Printf("FCM send error: %v", err)
		return fmt.Errorf("failed to send FCM message: %v", err)
	}

	log.Printf("Successfully sent FCM notification. Message ID: %s", response)
	return nil
}

// SendMulticast sends the same notification to multiple tokens efficiently
func (s *PushNotificationService) SendMulticast(tokens []string, payload NotificationPayload) error {
	if s.fcmClient == nil {
		return fmt.Errorf("FCM client not initialized")
	}

	if len(tokens) == 0 {
		return fmt.Errorf("no tokens provided")
	}

	// Build the multicast message
	message := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: payload.Title,
			Body:  payload.Body,
		},
		Data: payload.Data,
	}

	// Send the multicast message
	ctx, cancel := context.WithTimeout(context.Background(), 10)
	defer cancel()

	response, err := s.fcmClient.SendMulticast(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send FCM multicast: %v", err)
	}

	log.Printf("Successfully sent FCM multicast. Success: %d, Failure: %d", 
		response.SuccessCount, response.FailureCount)

	// Log any failures
	if response.FailureCount > 0 {
		for i, resp := range response.Responses {
			if !resp.Success {
				log.Printf("Failed to send to token %s: %v", tokens[i], resp.Error)
			}
		}
	}

	return nil
}

// SendToTopic sends a notification to all users subscribed to a topic
func (s *PushNotificationService) SendToTopic(topic string, payload NotificationPayload) error {
	if s.fcmClient == nil {
		return fmt.Errorf("FCM client not initialized")
	}

	message := &messaging.Message{
		Topic: topic,
		Notification: &messaging.Notification{
			Title: payload.Title,
			Body:  payload.Body,
		},
		Data: payload.Data,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10)
	defer cancel()

	response, err := s.fcmClient.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send FCM topic message: %v", err)
	}

	log.Printf("Successfully sent FCM topic notification to %s. Message ID: %s", topic, response)
	return nil
}

// SubscribeToTopic subscribes tokens to a topic
func (s *PushNotificationService) SubscribeToTopic(tokens []string, topic string) error {
	if s.fcmClient == nil {
		return fmt.Errorf("FCM client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10)
	defer cancel()

	response, err := s.fcmClient.SubscribeToTopic(ctx, tokens, topic)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %v", topic, err)
	}

	log.Printf("Successfully subscribed %d tokens to topic %s. Errors: %d", 
		len(tokens)-response.FailureCount, topic, response.FailureCount)

	return nil
}

// UnsubscribeFromTopic unsubscribes tokens from a topic
func (s *PushNotificationService) UnsubscribeFromTopic(tokens []string, topic string) error {
	if s.fcmClient == nil {
		return fmt.Errorf("FCM client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10)
	defer cancel()

	response, err := s.fcmClient.UnsubscribeFromTopic(ctx, tokens, topic)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe from topic %s: %v", topic, err)
	}

	log.Printf("Successfully unsubscribed %d tokens from topic %s. Errors: %d", 
		len(tokens)-response.FailureCount, topic, response.FailureCount)

	return nil
}

// sendExpoNotification sends notification via Expo Push API (for Expo Go testing)
func (s *PushNotificationService) sendExpoNotification(pushToken models.PushToken, payload NotificationPayload) error {
	expoMessage := map[string]interface{}{
		"to":    pushToken.PushToken,
		"title": payload.Title,
		"body":  payload.Body,
		"data":  payload.Data,
	}
	
	if payload.Sound != "" {
		expoMessage["sound"] = payload.Sound
	}
	
	if payload.Priority == "high" {
		expoMessage["priority"] = "high"
	}

	jsonBody, err := json.Marshal(expoMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal Expo message: %v", err)
	}

	resp, err := http.Post("https://exp.host/--/api/v2/push/send", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to send Expo notification: %v", err)
	}
	defer resp.Body.Close()

	// Read response body to get more details
	responseBody, _ := io.ReadAll(resp.Body)
	log.Printf("Expo API response status: %d, body: %s", resp.StatusCode, string(responseBody))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Expo push API returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	log.Printf("Successfully sent Expo notification to %s", pushToken.PushToken)
	return nil
}