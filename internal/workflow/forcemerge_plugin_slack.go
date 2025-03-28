package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/goliac-project/goliac/internal/config"
)

type ForcemergeStepPluginSlack struct {
	SlackToken string
	Channel    string
}

func NewForcemergeStepPluginSlack() ForcemergeStepPlugin {
	return &ForcemergeStepPluginSlack{
		SlackToken: config.Config.SlackToken,
		Channel:    config.Config.SlackChannel,
	}
}

type SlackMessage struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func (f *ForcemergeStepPluginSlack) Execute(ctx context.Context, username, explanation string, url *url.URL, properties map[string]interface{}) (string, error) {
	channel := f.Channel
	if properties["channel"] != nil {
		channel = properties["channel"].(string)
	}

	slackUrl := "https://slack.com/api/chat.postMessage"

	message := fmt.Sprintf("PR %s has been merged by %s with explanation: %s", url.Path, username, explanation)
	// Prepare the message payload
	msg := SlackMessage{
		Channel: channel,
		Text:    message,
	}

	// Convert the payload to JSON
	jsonPayload, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// Create a new HTTP POST request
	req, err := http.NewRequest("POST", slackUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create new request: %v", err)
	}

	// Set the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.SlackToken)

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response from Slack API
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-200 response: %v", resp.Status)
	}

	return "", nil
}
