package internal

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebhookHandler(t *testing.T) {

	t.Run("happy path: test ping webhook", func(t *testing.T) {
		callbackreceived := false
		callback := func() {
			callbackreceived = true
		}
		wh := NewGithubWebhookServerImpl("localhost", 8080, "/web", "secret", "main", callback).(*GithubWebhookServerImpl)

		body := `{
			"zen": "testing",
			"hook_id": 1234
		}`

		bodyReader := strings.NewReader(body)
		req := httptest.NewRequest("POST", "/webhook", bodyReader)
		sign := hmac.New(sha256.New, []byte("secret"))
		sign.Write([]byte(body))
		req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(sign.Sum(nil)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "ping")

		w := httptest.NewRecorder()
		wh.WebhookHandler(w, req)

		resp := w.Result()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, false, callbackreceived)
	})

	t.Run("happy path: test pull webhook", func(t *testing.T) {
		callbackreceived := false
		callback := func() {
			callbackreceived = true
		}
		wh := NewGithubWebhookServerImpl("localhost", 8080, "/web", "secret", "main", callback).(*GithubWebhookServerImpl)

		body := `{
			"ref": "refs/heads/main"
	}`

		bodyReader := strings.NewReader(body)
		req := httptest.NewRequest("POST", "/webhook", bodyReader)
		sign := hmac.New(sha256.New, []byte("secret"))
		sign.Write([]byte(body))
		req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(sign.Sum(nil)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "push")

		w := httptest.NewRecorder()
		wh.WebhookHandler(w, req)

		resp := w.Result()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, true, callbackreceived)
	})

	t.Run("not happy path: unsigned webhook", func(t *testing.T) {
		callbackreceived := false
		callback := func() {
			callbackreceived = true
		}
		wh := NewGithubWebhookServerImpl("localhost", 8080, "/web", "secret", "main", callback).(*GithubWebhookServerImpl)

		body := `{
			"zen": "testing",
			"hook_id": 1234
		}`

		bodyReader := strings.NewReader(body)
		req := httptest.NewRequest("POST", "/webhook", bodyReader)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "ping")

		w := httptest.NewRecorder()
		wh.WebhookHandler(w, req)

		resp := w.Result()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Equal(t, false, callbackreceived)
	})

}