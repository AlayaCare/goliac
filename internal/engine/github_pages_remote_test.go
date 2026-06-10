package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/goliac-project/goliac/internal/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type GithubPagesMockClient struct {
	callCount            int
	pagesWriteCallCount  int
	lastMethod           string
	lastBody             map[string]interface{}
	lastPagesWriteMethod string
	lastPagesWriteBody   map[string]interface{}
	failFirstCall        bool
	failAllCalls         bool
	failWhenPublicField  bool
	pagesGetBody         []byte
}

func (m *GithubPagesMockClient) QueryGraphQLAPI(ctx context.Context, query string, variables map[string]interface{}, githubToken *string) ([]byte, error) {
	return []byte("{}"), nil
}

func (m *GithubPagesMockClient) CallRestAPI(ctx context.Context, endpoint, parameters, method string, body map[string]interface{}, githubToken *string) ([]byte, error) {
	m.callCount++
	m.lastMethod = method
	m.lastBody = body

	if method == "GET" && strings.HasSuffix(endpoint, "/pages") {
		if m.pagesGetBody != nil {
			return m.pagesGetBody, nil
		}
		return []byte(`{"build_type":"workflow","public":true,"html_url":"https://goliac-project.github.io/goliac/"}`), nil
	}

	if method != "POST" && method != "PUT" {
		return []byte("{}"), nil
	}

	if !strings.HasSuffix(endpoint, "/pages") {
		return []byte("{}"), nil
	}

	m.pagesWriteCallCount++
	m.lastPagesWriteMethod = method
	m.lastPagesWriteBody = body

	if m.failAllCalls {
		responseBody := []byte(`{"message":"Private pages is not enabled for this repository. All Pages will be public.","status":"400"}`)
		return responseBody, fmt.Errorf("unexpected status: 400 Bad Request")
	}

	if m.failWhenPublicField {
		_, hasPublic := body["public"]
		if hasPublic {
			responseBody := []byte(`{"message":"Private pages is not enabled for this repository. All Pages will be public.","status":"400"}`)
			return responseBody, fmt.Errorf("unexpected status: 400 Bad Request")
		}
	}

	if m.failFirstCall && m.pagesWriteCallCount == 1 {
		responseBody := []byte(`{"message":"Private pages is not enabled for this repository. All Pages will be public.","status":"400"}`)
		return responseBody, fmt.Errorf("unexpected status: 400 Bad Request")
	}

	return []byte("{}"), nil
}

func (m *GithubPagesMockClient) GetAccessToken(ctx context.Context) (string, error) {
	return "mock-token", nil
}

func (m *GithubPagesMockClient) CreateJWT() (string, error) {
	return "mock-jwt", nil
}

func (m *GithubPagesMockClient) GetAppSlug() string {
	return "mock-app"
}

func TestUpdateRepositoryGithubPagesRetriesWithoutPublic(t *testing.T) {
	mockClient := &GithubPagesMockClient{failFirstCall: true}
	remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)
	ctx := context.TODO()
	remoteImpl.repositories["goliac"] = &GithubRepository{Name: "goliac"}
	logs := observability.NewLogCollection()

	pages := &GithubPagesComparable{Visibility: "public", Source: "workflow"}
	remoteImpl.UpdateRepositoryGithubPages(ctx, logs, false, "goliac", pages)

	assert.False(t, logs.HasErrors())
	assert.Equal(t, 2, mockClient.pagesWriteCallCount)
	assert.Equal(t, "PUT", mockClient.lastPagesWriteMethod)
	_, hasPublic := mockClient.lastPagesWriteBody["public"]
	assert.False(t, hasPublic)
	assert.Equal(t, "workflow", mockClient.lastPagesWriteBody["build_type"])
}

func TestCreateRepositoryGithubPagesRetriesWithoutPublic(t *testing.T) {
	mockClient := &GithubPagesMockClient{failFirstCall: true}
	remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)
	ctx := context.TODO()
	remoteImpl.repositories["goliac"] = &GithubRepository{Name: "goliac"}
	logs := observability.NewLogCollection()

	pages := &GithubPagesComparable{Visibility: "public", Source: "workflow"}
	remoteImpl.CreateRepositoryGithubPages(ctx, logs, false, "goliac", pages)

	assert.False(t, logs.HasErrors())
	assert.Equal(t, 2, mockClient.pagesWriteCallCount)
	assert.Equal(t, "POST", mockClient.lastPagesWriteMethod)
	_, hasPublic := mockClient.lastPagesWriteBody["public"]
	assert.False(t, hasPublic)
}

func TestUpdateRepositoryGithubPagesPrivateDoesNotRetry(t *testing.T) {
	mockClient := &GithubPagesMockClient{failAllCalls: true}
	remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)
	ctx := context.TODO()
	remoteImpl.repositories["goliac"] = &GithubRepository{Name: "goliac"}
	logs := observability.NewLogCollection()

	pages := &GithubPagesComparable{Visibility: "private", Source: "workflow"}
	remoteImpl.UpdateRepositoryGithubPages(ctx, logs, false, "goliac", pages)

	assert.True(t, logs.HasErrors())
	assert.Equal(t, 1, mockClient.pagesWriteCallCount)
	require.NotNil(t, mockClient.lastPagesWriteBody)
	assert.Equal(t, false, mockClient.lastPagesWriteBody["public"])
}

func TestCallRepositoryGithubPagesAPIIncludesResponseBodyOnError(t *testing.T) {
	mockClient := &GithubPagesMockClient{failAllCalls: true}
	remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)
	ctx := context.TODO()
	pages := &GithubPagesComparable{Visibility: "public", Source: "workflow"}

	_, err := remoteImpl.callRepositoryGithubPagesAPI(ctx, "goliac", "PUT", pages, true)
	require.Error(t, err)
	assert.ErrorContains(t, err, "Private pages is not enabled")
}

func TestUpdateRepositoryGithubPagesPreservesCustomDomainOnRetry(t *testing.T) {
	mockClient := &GithubPagesMockClient{failFirstCall: true}
	remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)
	ctx := context.TODO()
	remoteImpl.repositories["goliac"] = &GithubRepository{Name: "goliac"}
	logs := observability.NewLogCollection()

	pages := &GithubPagesComparable{
		Visibility:    "public",
		Source:        "branch",
		Branch:        "main",
		Path:          "/docs",
		Cname:         "docs.example.com",
		HttpsEnforced: true,
	}
	remoteImpl.UpdateRepositoryGithubPages(ctx, logs, false, "goliac", pages)

	assert.False(t, logs.HasErrors())
	assert.Equal(t, 2, mockClient.pagesWriteCallCount)
	assert.Equal(t, "docs.example.com", mockClient.lastPagesWriteBody["cname"])
	assert.Equal(t, true, mockClient.lastPagesWriteBody["https_enforced"])
	_, hasPublic := mockClient.lastPagesWriteBody["public"]
	assert.False(t, hasPublic)
	src, ok := mockClient.lastPagesWriteBody["source"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "main", src["branch"])
	assert.Equal(t, "/docs", src["path"])
}

func TestCreateRepositoryGithubPagesFollowUpPutRetriesWithoutPublic(t *testing.T) {
	mockClient := &GithubPagesMockClient{failWhenPublicField: true}
	remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)
	ctx := context.TODO()
	remoteImpl.repositories["goliac"] = &GithubRepository{Name: "goliac"}
	logs := observability.NewLogCollection()

	pages := &GithubPagesComparable{
		Visibility:    "public",
		Source:        "branch",
		Branch:        "main",
		Path:          "/",
		Cname:         "docs.example.com",
		HttpsEnforced: true,
	}
	remoteImpl.CreateRepositoryGithubPages(ctx, logs, false, "goliac", pages)

	assert.False(t, logs.HasErrors())
	assert.Equal(t, 4, mockClient.pagesWriteCallCount)
	assert.Equal(t, "PUT", mockClient.lastPagesWriteMethod)
	_, hasPublic := mockClient.lastPagesWriteBody["public"]
	assert.False(t, hasPublic)
	assert.Equal(t, "docs.example.com", mockClient.lastPagesWriteBody["cname"])
}

func TestGithubPagesMockClientResponseBodyShape(t *testing.T) {
	responseBody := []byte(`{"message":"Private pages is not enabled for this repository. All Pages will be public.","status":"400"}`)
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(responseBody, &parsed))
	assert.Equal(t, "400", parsed["status"])
}
