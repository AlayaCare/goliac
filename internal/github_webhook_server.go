package internal

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type GithubWebhookServerCallback func()

/*
GithubWebhookServer is the interface for the webhook server
It will wait for a Github webhook event and call the callback function
when a merge event is received on the main branch
*/
type GithubWebhookServer interface {
	// Start the server
	Start() error
	Shutdown() error
}

type GithubWebhookServerImpl struct {
	webhookServerAddress string
	webhookServerPort    int
	webhookPath          string
	webhookSecret        string
	server               *http.Server
	mainBranch           string
	callback             GithubWebhookServerCallback
}

func NewGithubWebhookServerImpl(httpaddr string, httpport int, webhookPath string, secret string, mainBranch string, callback GithubWebhookServerCallback) GithubWebhookServer {
	return &GithubWebhookServerImpl{
		webhookServerAddress: httpaddr,
		webhookServerPort:    httpport,
		webhookPath:          webhookPath,
		webhookSecret:        secret,
		server:               nil,
		mainBranch:           mainBranch,
		callback:             callback,
	}
}

func (s *GithubWebhookServerImpl) Start() error {
	// start a new http server
	s.server = &http.Server{
		Addr: fmt.Sprintf("%s:%d", s.webhookServerAddress, s.webhookServerPort),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(s.webhookPath, s.WebhookHandler)
	s.server.Handler = mux

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil

}

func (s *GithubWebhookServerImpl) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

type PushEvent struct {
	Ref string `json:"ref"`
}

func (s *GithubWebhookServerImpl) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("Received webhook event")
	// handle the github webhook
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Invalid content type", http.StatusBadRequest)
		return
	}

	// check the secret
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		http.Error(w, "Missing signature", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	if s.webhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(s.webhookSecret))
		mac.Write(body)
		expectedSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(expectedSignature), []byte(signature)) {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Process the webhook payload
	eventType := r.Header.Get("X-GitHub-Event")

	logrus.Debugf("Received webhook event type: %s", eventType)

	// https://docs.github.com/en/webhooks/webhook-events-and-payloads
	switch eventType {
	case "ping":
		s.handlePingEvent(w)
	case "push":
		s.handlePushEvent(w, body)
	default:
		logrus.Debugf("Event type %s not supported", eventType)
		w.WriteHeader(http.StatusOK)
	}
}

func (s *GithubWebhookServerImpl) handlePingEvent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
}

func (s *GithubWebhookServerImpl) handlePushEvent(w http.ResponseWriter, body []byte) {
	var pushEvent PushEvent

	err := json.Unmarshal(body, &pushEvent)
	if err != nil {
		http.Error(w, "Failed to parse push event", http.StatusBadRequest)
		return
	}

	// Check if the push is to the main branch
	if pushEvent.Ref == fmt.Sprintf("refs/heads/%s", s.mainBranch) {
		s.callback()
	} else {
		http.Error(w, "Parse push event: wrong branch", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
