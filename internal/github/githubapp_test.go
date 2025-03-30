package github

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetInstallations(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		httpTest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET request, got %s", r.Method)
			}
			if r.URL.Path != "/app/installations" {
				t.Fatalf("expected /app/installations, got %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"id":123, "account": {"login": "testuser"}}]`))
		}))
		defer httpTest.Close()
		gitHubServer := httpTest.URL
		jwt := "testjwt"
		client := &GitHubClientImpl{
			gitHubServer: gitHubServer,
		}

		installations, err := client.getInstallations(jwt)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(installations) == 0 {
			t.Fatalf("expected installations, got none")
		}
	})
}
