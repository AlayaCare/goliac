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

type GithubWebhookServerIssueCommentCallback func(repository, prUrl, githubIdCaller, comment string, comment_id int)

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
	organization         string
	repository           string
	mainBranch           string
	callback             GithubWebhookServerCallback
	issueCommentCallback GithubWebhookServerIssueCommentCallback
}

func NewGithubWebhookServerImpl(httpaddr string, httpport int, webhookPath string, secret string, organization, repository, mainBranch string, callback GithubWebhookServerCallback, issueCommentCallback GithubWebhookServerIssueCommentCallback) GithubWebhookServer {
	return &GithubWebhookServerImpl{
		webhookServerAddress: httpaddr,
		webhookServerPort:    httpport,
		webhookPath:          webhookPath,
		webhookSecret:        secret,
		server:               nil,
		organization:         organization,
		repository:           repository,
		mainBranch:           mainBranch,
		callback:             callback,
		issueCommentCallback: issueCommentCallback,
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
	Ref        string `json:"ref"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
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
	case "issue_comment":
		s.handleIssueCommentEvent(w, body)
	default:
		logrus.Debugf("Event type %s not supported", eventType)
		w.WriteHeader(http.StatusOK)
	}
}

func (s *GithubWebhookServerImpl) handlePingEvent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
}

// the goal of this function is to trigger the callback when a push event is received on the main branch
// and only if the repository is the teams-repo
func (s *GithubWebhookServerImpl) handlePushEvent(w http.ResponseWriter, body []byte) {
	var pushEvent PushEvent

	err := json.Unmarshal(body, &pushEvent)
	if err != nil {
		http.Error(w, "Failed to parse push event", http.StatusBadRequest)
		return
	}

	// we are only interested in the teams-repo
	if pushEvent.Repository.FullName != fmt.Sprintf("%s/%s", s.organization, s.repository) {
		w.WriteHeader(http.StatusOK)
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

// the goal of this function is to trigger the issueCommentCallback when a comment is added to a PR
func (s *GithubWebhookServerImpl) handleIssueCommentEvent(w http.ResponseWriter, body []byte) {
	var issueCommentEvent IssueCommentEvent

	err := json.Unmarshal(body, &issueCommentEvent)
	if err != nil {
		http.Error(w, "Failed to parse pull request event", http.StatusBadRequest)
		return
	}
	prUrl := fmt.Sprintf("https://github.com/%s/pull/%d", issueCommentEvent.Repository.FullName, issueCommentEvent.Issue.Number)

	s.issueCommentCallback(
		issueCommentEvent.Repository.FullName,
		prUrl,
		issueCommentEvent.Sender.Login,
		issueCommentEvent.Comment.Body,
		issueCommentEvent.Comment.ID,
	)

	w.WriteHeader(http.StatusOK)
}

type IssueCommentEvent struct {
	Action  string `json:"action"`
	Comment struct {
		ID   int    `json:"id"`
		Body string `json:"body"`
	} `json:"comment"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
	Issue struct {
		Number int `json:"number"`
	} `json:"issue"`
}
