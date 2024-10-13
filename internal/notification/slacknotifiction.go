package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type SlackNotificationService struct {
	SlackToken string
	Channel    string
}

func NewSlackNotificationService(slackToken string, channel string) NotificationService {
	return &SlackNotificationService{
		SlackToken: slackToken,
		Channel:    channel,
	}
}

type SlackMessage struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func (s *SlackNotificationService) SendNotification(message string) error {
	url := "https://slack.com/api/chat.postMessage"

	// Prepare the message payload
	msg := SlackMessage{
		Channel: s.Channel,
		Text:    message,
	}

	// Convert the payload to JSON
	jsonPayload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// Create a new HTTP POST request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create new request: %v", err)
	}

	// Set the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.SlackToken)

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response from Slack API
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response: %v", resp.Status)
	}

	return nil
}
