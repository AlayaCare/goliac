package github

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type MockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func TestQueryGraphQLAPI(t *testing.T) {
	// Create a test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"user": {"name": "octocat"}}}`))
	}))
	defer testServer.Close()

	// Replace the httpClient with a mock
	client := &GitHubClientImpl{
		httpClient: &http.Client{
			Transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					return http.Get(testServer.URL) // Use the test server's URL
				},
			},
		},
	}

	// Call the function and check the result
	query := `query { user(login: "octocat") { name } }`
	ctx := context.TODO()
	result, err := client.QueryGraphQLAPI(ctx, query, nil, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !strings.Contains(string(result), "octocat") {
		t.Errorf("expected 'octocat' in the result, got %s", result)
	}
}

func TestCallRestAPI(t *testing.T) {
	t.Run("happy path: GET", func(t *testing.T) {
		// Create a test server
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"login": "octocat"}`))
		}))
		defer testServer.Close()

		// Replace the httpClient with a mock
		client := &GitHubClientImpl{
			gitHubServer: testServer.URL,
			httpClient:   &http.Client{},
		}

		// Call the function and check the result
		ctx := context.TODO()
		result, err := client.CallRestAPI(ctx, "/octocat", "", "GET", nil, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !strings.Contains(string(result), "octocat") {
			t.Errorf("expected 'octocat' in the result, got %s", result)
		}
	})

	t.Run("happy path: POST", func(t *testing.T) {
		// Create a test server
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/octocat" {
				// Read the request body
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("failed to read request body: %v", err)
				}
				if string(body) != `{"param":"value"}` {
					t.Errorf("expected request body to be {'param':'value'}, got %s", string(body))
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"login": "octocat"}`))
				return
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer testServer.Close()

		// Replace the httpClient with a mock
		client := &GitHubClientImpl{
			gitHubServer: testServer.URL,
			httpClient:   &http.Client{},
		}

		// Call the function and check the result
		ctx := context.TODO()
		result, err := client.CallRestAPI(ctx, "/octocat", "", "POST", map[string]interface{}{"param": "value"}, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !strings.Contains(string(result), "octocat") {
			t.Errorf("expected 'octocat' in the result, got %s", result)
		}
	})
}

func TestGetHeaderCaseInsensitive(t *testing.T) {
	headers := http.Header{
		"X-Ratelimit-Reset":     {"1763908461"},
		"Retry-After":           {"30"},
		"X-Ratelimit-Remaining": {"0"},
	}
	header := getHeaderCaseInsensitive(headers, "X-RateLimit-Reset")
	if header != "1763908461" {
		t.Errorf("expected '1763908461', got %s", header)
	}
	header = getHeaderCaseInsensitive(headers, "Retry-After")
	if header != "30" {
		t.Errorf("expected '30', got %s", header)
	}
	header = getHeaderCaseInsensitive(headers, "X-RateLimit-Remaining")
	if header != "0" {
		t.Errorf("expected '0', got %s", header)
	}
}
