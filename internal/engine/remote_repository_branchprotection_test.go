package engine

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/goliac-project/goliac/internal/config"
	"github.com/goliac-project/goliac/internal/observability"
	"github.com/stretchr/testify/assert"
)

// AddBranchProtectionMockClient is a dedicated mock client for AddRepositoryBranchProtection tests
type AddBranchProtectionMockClient struct {
	// Track calls to verify test behavior
	lastGraphQLQuery string
	lastVariables    map[string]interface{}
	lastGithubToken  *string

	// Configure mock responses
	shouldError  bool
	errorMessage string
	responseBody string
	callCount    int
	responses    []string
}

func (m *AddBranchProtectionMockClient) QueryGraphQLAPI(ctx context.Context, query string, variables map[string]interface{}, githubToken *string) ([]byte, error) {
	m.lastGraphQLQuery = query
	m.lastVariables = variables
	m.lastGithubToken = githubToken
	i := m.callCount
	m.callCount++

	if m.shouldError {
		return []byte(m.errorMessage), errors.New(m.errorMessage)
	}
	if len(m.responses) > 0 && i < len(m.responses) {
		return []byte(m.responses[i]), nil
	}
	if m.responseBody != "" {
		return []byte(m.responseBody), nil
	}

	// Default successful response
	return []byte(`{
		"data": {
			"createBranchProtectionRule": {
				"branchProtectionRule": {
					"id": "BP_123"
				}
			}
		}
	}`), nil
}

func (m *AddBranchProtectionMockClient) CallRestAPI(ctx context.Context, endpoint, parameters, method string, body map[string]interface{}, githubToken *string) ([]byte, error) {
	return []byte("{}"), nil
}

func (m *AddBranchProtectionMockClient) GetAccessToken(ctx context.Context) (string, error) {
	return "mock-token", nil
}

func (m *AddBranchProtectionMockClient) CreateJWT() (string, error) {
	return "mock-jwt", nil
}

func (m *AddBranchProtectionMockClient) GetAppSlug() string {
	return "mock-app"
}

func TestAddRepositoryBranchProtection(t *testing.T) {
	t.Run("happy path: add branch protection", func(t *testing.T) {
		// Setup mock client
		mockClient := &AddBranchProtectionMockClient{}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Add repository to the remote impl
		repo := &GithubRepository{
			Name:              "test-repo",
			Id:                123,
			RefId:             "R_123",
			BranchProtections: make(map[string]*GithubBranchProtection),
		}
		remoteImpl.repositories = map[string]*GithubRepository{
			"test-repo": repo,
		}

		// Setup test data
		branchProtection := &GithubBranchProtection{
			Pattern:                        "main",
			RequiresApprovingReviews:       true,
			RequiredApprovingReviewCount:   2,
			DismissesStaleReviews:          true,
			RequiresCodeOwnerReviews:       true,
			RequireLastPushApproval:        true,
			RequiresStatusChecks:           true,
			RequiresStrictStatusChecks:     true,
			RequiredStatusCheckContexts:    []string{"test-check"},
			RequiresConversationResolution: true,
			RequiresCommitSignatures:       true,
			RequiresLinearHistory:          true,
			AllowsForcePushes:              false,
			AllowsDeletions:                false,
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call AddRepositoryBranchProtection
		remoteImpl.AddRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", branchProtection)

		// Verify no errors occurred
		assert.False(t, logsCollector.HasErrors())

		// Verify the GraphQL query was made with correct variables
		expectedVariables := map[string]interface{}{
			"repositoryId":                   "R_123",
			"pattern":                        "main",
			"requiresApprovingReviews":       true,
			"requiredApprovingReviewCount":   2,
			"dismissesStaleReviews":          true,
			"requiresCodeOwnerReviews":       true,
			"requireLastPushApproval":        true,
			"requiresStatusChecks":           true,
			"requiresStrictStatusChecks":     true,
			"requiredStatusCheckContexts":    []string{"test-check"},
			"requiresConversationResolution": true,
			"requiresCommitSignatures":       true,
			"requiresLinearHistory":          true,
			"allowsForcePushes":              false,
			"allowsDeletions":                false,
			"bypassPullRequestActorIds":      []string{},
		}
		assert.Equal(t, expectedVariables, mockClient.lastVariables)

		// Verify the branch protection was added to the cache
		assert.Contains(t, repo.BranchProtections, "main")
		assert.Equal(t, "BP_123", repo.BranchProtections["main"].Id)
	})

	t.Run("app bypass actor uses GraphQL node id from appIds", func(t *testing.T) {
		mockClient := &AddBranchProtectionMockClient{}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		repo := &GithubRepository{
			Name:              "test-repo",
			Id:                123,
			RefId:             "R_123",
			BranchProtections: make(map[string]*GithubBranchProtection),
		}
		remoteImpl.repositories = map[string]*GithubRepository{
			"test-repo": repo,
		}
		remoteImpl.appIds = map[string]*GithubApp{
			"my-app": {
				Slug:      "my-app",
				GraphqlId: "A_app_node_id",
			},
		}

		branchProtection := &GithubBranchProtection{
			Pattern:                        "main",
			RequiresApprovingReviews:       true,
			RequiredApprovingReviewCount:   2,
			DismissesStaleReviews:          true,
			RequiresCodeOwnerReviews:       true,
			RequireLastPushApproval:        true,
			RequiresStatusChecks:           true,
			RequiresStrictStatusChecks:     true,
			RequiredStatusCheckContexts:    []string{"test-check"},
			RequiresConversationResolution: true,
			RequiresCommitSignatures:       true,
			RequiresLinearHistory:          true,
			AllowsForcePushes:              false,
			AllowsDeletions:                false,
		}
		var bypassNode BypassPullRequestAllowanceNode
		bypassNode.Actor.AppSlug = "my-app"
		branchProtection.BypassPullRequestAllowances.Nodes = []BypassPullRequestAllowanceNode{bypassNode}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		remoteImpl.AddRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", branchProtection)

		assert.False(t, logsCollector.HasErrors())
		ids, ok := mockClient.lastVariables["bypassPullRequestActorIds"].([]string)
		assert.True(t, ok)
		assert.Equal(t, []string{"A_app_node_id"}, ids)
	})

	t.Run("error path: repository not found", func(t *testing.T) {
		// Setup mock client
		mockClient := &AddBranchProtectionMockClient{}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Setup test data
		branchProtection := &GithubBranchProtection{
			Pattern: "main",
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call AddRepositoryBranchProtection with non-existent repository
		remoteImpl.AddRepositoryBranchProtection(ctx, logsCollector, false, "non-existent-repo", branchProtection)

		// Verify error was collected
		assert.True(t, logsCollector.HasErrors())
		assert.Contains(t, logsCollector.Errors[0].Error(), "repository non-existent-repo not found")

		// Verify no GraphQL query was made
		assert.Empty(t, mockClient.lastGraphQLQuery)
	})

	t.Run("error path: API error", func(t *testing.T) {
		// Setup mock client with error
		mockClient := &AddBranchProtectionMockClient{
			shouldError:  true,
			errorMessage: "failed to create branch protection",
		}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Add repository to the remote impl
		repo := &GithubRepository{
			Name:              "test-repo",
			Id:                123,
			RefId:             "R_123",
			BranchProtections: make(map[string]*GithubBranchProtection),
		}
		remoteImpl.repositories = map[string]*GithubRepository{
			"test-repo": repo,
		}

		// Setup test data
		branchProtection := &GithubBranchProtection{
			Pattern: "main",
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call AddRepositoryBranchProtection
		remoteImpl.AddRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", branchProtection)

		// Verify error was collected
		assert.True(t, logsCollector.HasErrors())
		assert.Contains(t, logsCollector.Errors[0].Error(), "failed to add branch protection to repository")

		// Verify the branch protection was not added to the cache
		assert.NotContains(t, repo.BranchProtections, "main")
	})

	t.Run("happy path: dry run", func(t *testing.T) {
		// Setup mock client
		mockClient := &AddBranchProtectionMockClient{}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Add repository to the remote impl
		repo := &GithubRepository{
			Name:              "test-repo",
			Id:                123,
			RefId:             "R_123",
			BranchProtections: make(map[string]*GithubBranchProtection),
		}
		remoteImpl.repositories = map[string]*GithubRepository{
			"test-repo": repo,
		}

		// Setup test data
		branchProtection := &GithubBranchProtection{
			Pattern: "main",
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call AddRepositoryBranchProtection in dry run mode
		remoteImpl.AddRepositoryBranchProtection(ctx, logsCollector, true, "test-repo", branchProtection)

		// Verify no errors occurred
		assert.False(t, logsCollector.HasErrors())

		// Verify no GraphQL query was made
		assert.Empty(t, mockClient.lastGraphQLQuery)

		// Verify the branch protection was added to the cache
		assert.Contains(t, repo.BranchProtections, "main")
	})

	t.Run("error path: GraphQL response error", func(t *testing.T) {
		// Setup mock client with error response
		mockClient := &AddBranchProtectionMockClient{
			responseBody: `{
				"errors": [
					{
						"message": "Invalid branch protection configuration",
						"path": ["createBranchProtectionRule"]
					}
				]
			}`,
		}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Add repository to the remote impl
		repo := &GithubRepository{
			Name:              "test-repo",
			Id:                123,
			RefId:             "R_123",
			BranchProtections: make(map[string]*GithubBranchProtection),
		}
		remoteImpl.repositories = map[string]*GithubRepository{
			"test-repo": repo,
		}

		// Setup test data
		branchProtection := &GithubBranchProtection{
			Pattern: "main",
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call AddRepositoryBranchProtection
		remoteImpl.AddRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", branchProtection)

		// Verify error was collected
		assert.True(t, logsCollector.HasErrors())
		assert.Contains(t, logsCollector.Errors[0].Error(), "graphql error on AddRepositoryBranchProtection")

		// Verify the branch protection was not added to the cache
		assert.NotContains(t, repo.BranchProtections, "main")
	})
}

// DeleteBranchProtectionMockClient is a dedicated mock client for DeleteRepositoryBranchProtection tests
type DeleteBranchProtectionMockClient struct {
	// Track calls to verify test behavior
	lastGraphQLQuery string
	lastVariables    map[string]interface{}
	lastGithubToken  *string

	// Configure mock responses
	shouldError  bool
	errorMessage string
	responseBody string
	callCount    int
	responses    []string
}

func (m *DeleteBranchProtectionMockClient) QueryGraphQLAPI(ctx context.Context, query string, variables map[string]interface{}, githubToken *string) ([]byte, error) {
	m.lastGraphQLQuery = query
	m.lastVariables = variables
	m.lastGithubToken = githubToken
	i := m.callCount
	m.callCount++

	if m.shouldError {
		return []byte(m.errorMessage), errors.New(m.errorMessage)
	}
	if len(m.responses) > 0 && i < len(m.responses) {
		return []byte(m.responses[i]), nil
	}
	if m.responseBody != "" {
		return []byte(m.responseBody), nil
	}

	// Default successful response
	return []byte(`{
		"data": {
			"deleteBranchProtectionRule": {
				"clientMutationId": "test-mutation"
			}
		}
	}`), nil
}

func (m *DeleteBranchProtectionMockClient) CallRestAPI(ctx context.Context, endpoint, parameters, method string, body map[string]interface{}, githubToken *string) ([]byte, error) {
	return []byte("{}"), nil
}

func (m *DeleteBranchProtectionMockClient) GetAccessToken(ctx context.Context) (string, error) {
	return "mock-token", nil
}

func (m *DeleteBranchProtectionMockClient) CreateJWT() (string, error) {
	return "mock-jwt", nil
}

func (m *DeleteBranchProtectionMockClient) GetAppSlug() string {
	return "mock-app"
}

func TestDeleteRepositoryBranchProtection(t *testing.T) {
	t.Run("happy path: delete branch protection", func(t *testing.T) {
		// Setup mock client
		mockClient := &DeleteBranchProtectionMockClient{}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Add repository with branch protection to the remote impl
		branchProtection := &GithubBranchProtection{
			Id:      "BP_123",
			Pattern: "main",
		}
		repo := &GithubRepository{
			Name:  "test-repo",
			Id:    123,
			RefId: "R_123",
			BranchProtections: map[string]*GithubBranchProtection{
				"main": branchProtection,
			},
		}
		remoteImpl.repositories = map[string]*GithubRepository{
			"test-repo": repo,
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call DeleteRepositoryBranchProtection
		remoteImpl.DeleteRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", branchProtection)

		// Verify no errors occurred
		assert.False(t, logsCollector.HasErrors())

		// Verify the GraphQL query was made with correct variables
		expectedVariables := map[string]interface{}{
			"branchProtectionRuleId": "BP_123",
		}
		assert.Equal(t, expectedVariables, mockClient.lastVariables)
		assert.Contains(t, mockClient.lastGraphQLQuery, "deleteBranchProtectionRule")

		// Verify the branch protection was removed from the cache
		assert.NotContains(t, repo.BranchProtections, "main")
	})

	t.Run("error path: repository not found", func(t *testing.T) {
		// Setup mock client
		mockClient := &DeleteBranchProtectionMockClient{}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Setup test data
		branchProtection := &GithubBranchProtection{
			Id:      "BP_123",
			Pattern: "main",
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call DeleteRepositoryBranchProtection with non-existent repository
		remoteImpl.DeleteRepositoryBranchProtection(ctx, logsCollector, false, "non-existent-repo", branchProtection)

		// Verify error was collected
		assert.True(t, logsCollector.HasErrors())
		assert.Contains(t, logsCollector.Errors[0].Error(), "repository non-existent-repo not found")

		// Verify no GraphQL query was made
		assert.Empty(t, mockClient.lastGraphQLQuery)
	})

	t.Run("error path: API error", func(t *testing.T) {
		// Setup mock client with error
		mockClient := &DeleteBranchProtectionMockClient{
			shouldError:  true,
			errorMessage: "failed to delete branch protection",
		}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Add repository with branch protection to the remote impl
		branchProtection := &GithubBranchProtection{
			Id:      "BP_123",
			Pattern: "main",
		}
		repo := &GithubRepository{
			Name:  "test-repo",
			Id:    123,
			RefId: "R_123",
			BranchProtections: map[string]*GithubBranchProtection{
				"main": branchProtection,
			},
		}
		remoteImpl.repositories = map[string]*GithubRepository{
			"test-repo": repo,
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call DeleteRepositoryBranchProtection
		remoteImpl.DeleteRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", branchProtection)

		// Verify error was collected
		assert.True(t, logsCollector.HasErrors())
		assert.Contains(t, logsCollector.Errors[0].Error(), "failed to delete branch protection for repository")

		// Verify the branch protection remains in the cache
		assert.Contains(t, repo.BranchProtections, "main")
	})

	t.Run("happy path: dry run", func(t *testing.T) {
		// Setup mock client
		mockClient := &DeleteBranchProtectionMockClient{}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Add repository with branch protection to the remote impl
		branchProtection := &GithubBranchProtection{
			Id:      "BP_123",
			Pattern: "main",
		}
		repo := &GithubRepository{
			Name:  "test-repo",
			Id:    123,
			RefId: "R_123",
			BranchProtections: map[string]*GithubBranchProtection{
				"main": branchProtection,
			},
		}
		remoteImpl.repositories = map[string]*GithubRepository{
			"test-repo": repo,
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call DeleteRepositoryBranchProtection in dry run mode
		remoteImpl.DeleteRepositoryBranchProtection(ctx, logsCollector, true, "test-repo", branchProtection)

		// Verify no errors occurred
		assert.False(t, logsCollector.HasErrors())

		// Verify no GraphQL query was made
		assert.Empty(t, mockClient.lastGraphQLQuery)

		// Verify the branch protection was removed from the cache
		assert.NotContains(t, repo.BranchProtections, "main")
	})

	t.Run("error path: GraphQL response error", func(t *testing.T) {
		// Setup mock client with error response
		mockClient := &DeleteBranchProtectionMockClient{
			responseBody: `{
				"errors": [
					{
						"message": "Branch protection rule not found",
						"path": ["deleteBranchProtectionRule"]
					}
				]
			}`,
		}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Add repository with branch protection to the remote impl
		branchProtection := &GithubBranchProtection{
			Id:      "BP_123",
			Pattern: "main",
		}
		repo := &GithubRepository{
			Name:  "test-repo",
			Id:    123,
			RefId: "R_123",
			BranchProtections: map[string]*GithubBranchProtection{
				"main": branchProtection,
			},
		}
		remoteImpl.repositories = map[string]*GithubRepository{
			"test-repo": repo,
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call DeleteRepositoryBranchProtection
		remoteImpl.DeleteRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", branchProtection)

		// Verify error was collected
		assert.True(t, logsCollector.HasErrors())
		assert.Contains(t, logsCollector.Errors[0].Error(), "graphql error on DeleteRepositoryBranchProtection")

		// Verify the branch protection remains in the cache
		assert.Contains(t, repo.BranchProtections, "main")
	})

	t.Run("happy path: delete non-existent branch protection", func(t *testing.T) {
		// Setup mock client
		mockClient := &DeleteBranchProtectionMockClient{}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Add repository without branch protection to the remote impl
		repo := &GithubRepository{
			Name:              "test-repo",
			Id:                123,
			RefId:             "R_123",
			BranchProtections: map[string]*GithubBranchProtection{},
		}
		remoteImpl.repositories = map[string]*GithubRepository{
			"test-repo": repo,
		}

		// Setup test data for non-existent branch protection
		branchProtection := &GithubBranchProtection{
			Id:      "BP_123",
			Pattern: "main",
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call DeleteRepositoryBranchProtection
		remoteImpl.DeleteRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", branchProtection)

		// Verify no errors occurred
		assert.True(t, logsCollector.HasErrors())
		assert.Contains(t, logsCollector.Errors[0].Error(), "branch protection for repository test-repo not found")

		// Verify the cache remains empty
		assert.Empty(t, repo.BranchProtections)
	})
}

// UpdateBranchProtectionMockClient is a dedicated mock client for UpdateRepositoryBranchProtection tests
type UpdateBranchProtectionMockClient struct {
	// Track calls to verify test behavior
	lastGraphQLQuery string
	lastVariables    map[string]interface{}
	lastGithubToken  *string

	// Configure mock responses
	shouldError  bool
	errorMessage string
	responseBody string
	callCount    int
	responses    []string
}

func (m *UpdateBranchProtectionMockClient) QueryGraphQLAPI(ctx context.Context, query string, variables map[string]interface{}, githubToken *string) ([]byte, error) {
	m.lastGraphQLQuery = query
	m.lastVariables = variables
	m.lastGithubToken = githubToken
	i := m.callCount
	m.callCount++

	if m.shouldError {
		return []byte(m.errorMessage), errors.New(m.errorMessage)
	}
	if len(m.responses) > 0 && i < len(m.responses) {
		return []byte(m.responses[i]), nil
	}
	if m.responseBody != "" {
		return []byte(m.responseBody), nil
	}

	// Default successful response
	return []byte(`{
		"data": {
			"updateBranchProtectionRule": {
				"branchProtectionRule": {
					"id": "BP_123"
				}
			}
		}
	}`), nil
}

func (m *UpdateBranchProtectionMockClient) CallRestAPI(ctx context.Context, endpoint, parameters, method string, body map[string]interface{}, githubToken *string) ([]byte, error) {
	return []byte("{}"), nil
}

func (m *UpdateBranchProtectionMockClient) GetAccessToken(ctx context.Context) (string, error) {
	return "mock-token", nil
}

func (m *UpdateBranchProtectionMockClient) CreateJWT() (string, error) {
	return "mock-jwt", nil
}

func (m *UpdateBranchProtectionMockClient) GetAppSlug() string {
	return "mock-app"
}

func TestUpdateRepositoryBranchProtection(t *testing.T) {
	t.Run("happy path: update branch protection", func(t *testing.T) {
		// Setup mock client
		mockClient := &UpdateBranchProtectionMockClient{}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Add repository with existing branch protection to the remote impl
		existingBranchProtection := &GithubBranchProtection{
			Id:                             "BP_123",
			Pattern:                        "main",
			RequiresApprovingReviews:       false,
			RequiredApprovingReviewCount:   1,
			DismissesStaleReviews:          false,
			RequiresCodeOwnerReviews:       false,
			RequireLastPushApproval:        false,
			RequiresStatusChecks:           false,
			RequiresStrictStatusChecks:     false,
			RequiredStatusCheckContexts:    []string{},
			RequiresConversationResolution: false,
			RequiresCommitSignatures:       false,
			RequiresLinearHistory:          false,
			AllowsForcePushes:              true,
			AllowsDeletions:                true,
		}
		repo := &GithubRepository{
			Name:  "test-repo",
			Id:    123,
			RefId: "R_123",
			BranchProtections: map[string]*GithubBranchProtection{
				"main": existingBranchProtection,
			},
		}
		remoteImpl.repositories = map[string]*GithubRepository{
			"test-repo": repo,
		}

		// Setup updated branch protection
		updatedBranchProtection := &GithubBranchProtection{
			Id:                             "BP_123",
			Pattern:                        "main",
			RequiresApprovingReviews:       true,
			RequiredApprovingReviewCount:   2,
			DismissesStaleReviews:          true,
			RequiresCodeOwnerReviews:       true,
			RequireLastPushApproval:        true,
			RequiresStatusChecks:           true,
			RequiresStrictStatusChecks:     true,
			RequiredStatusCheckContexts:    []string{"test-check"},
			RequiresConversationResolution: true,
			RequiresCommitSignatures:       true,
			RequiresLinearHistory:          true,
			AllowsForcePushes:              false,
			AllowsDeletions:                false,
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call UpdateRepositoryBranchProtection
		remoteImpl.UpdateRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", updatedBranchProtection)

		// Verify no errors occurred
		assert.False(t, logsCollector.HasErrors())

		// Verify the GraphQL query was made with correct variables
		expectedVariables := map[string]interface{}{
			"branchProtectionRuleId":         "BP_123",
			"pattern":                        "main",
			"requiresApprovingReviews":       true,
			"requiredApprovingReviewCount":   2,
			"dismissesStaleReviews":          true,
			"requiresCodeOwnerReviews":       true,
			"requireLastPushApproval":        true,
			"requiresStatusChecks":           true,
			"requiresStrictStatusChecks":     true,
			"requiredStatusCheckContexts":    []string{"test-check"},
			"requiresConversationResolution": true,
			"requiresCommitSignatures":       true,
			"requiresLinearHistory":          true,
			"allowsForcePushes":              false,
			"allowsDeletions":                false,
			"bypassPullRequestActorIds":      []string{},
		}
		assert.Equal(t, expectedVariables, mockClient.lastVariables)
		assert.Contains(t, mockClient.lastGraphQLQuery, "updateBranchProtectionRule")

		// Verify the branch protection was updated in the cache
		assert.Equal(t, updatedBranchProtection, repo.BranchProtections["main"])
	})

	t.Run("error path: repository not found", func(t *testing.T) {
		// Setup mock client
		mockClient := &UpdateBranchProtectionMockClient{}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Setup test data
		branchProtection := &GithubBranchProtection{
			Id:      "BP_123",
			Pattern: "main",
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call UpdateRepositoryBranchProtection with non-existent repository
		remoteImpl.UpdateRepositoryBranchProtection(ctx, logsCollector, false, "non-existent-repo", branchProtection)

		// Verify error was collected
		assert.True(t, logsCollector.HasErrors())
		assert.Contains(t, logsCollector.Errors[0].Error(), "repository non-existent-repo not found")

		// Verify no GraphQL query was made
		assert.Empty(t, mockClient.lastGraphQLQuery)
	})

	t.Run("error path: API error", func(t *testing.T) {
		// Setup mock client with error
		mockClient := &UpdateBranchProtectionMockClient{
			shouldError:  true,
			errorMessage: "failed to update branch protection",
		}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Add repository with branch protection to the remote impl
		originalBranchProtection := &GithubBranchProtection{
			Id:      "BP_123",
			Pattern: "main",
		}
		repo := &GithubRepository{
			Name:  "test-repo",
			Id:    123,
			RefId: "R_123",
			BranchProtections: map[string]*GithubBranchProtection{
				"main": originalBranchProtection,
			},
		}
		remoteImpl.repositories = map[string]*GithubRepository{
			"test-repo": repo,
		}

		// Setup updated branch protection
		updatedBranchProtection := &GithubBranchProtection{
			Id:                       "BP_123",
			Pattern:                  "main",
			RequiresApprovingReviews: true,
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call UpdateRepositoryBranchProtection
		remoteImpl.UpdateRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", updatedBranchProtection)

		// Verify error was collected
		assert.True(t, logsCollector.HasErrors())
		assert.Contains(t, logsCollector.Errors[0].Error(), "failed to update branch protection for repository")

		// Verify the branch protection remains unchanged in the cache
		assert.Equal(t, originalBranchProtection, repo.BranchProtections["main"])
	})

	t.Run("happy path: dry run", func(t *testing.T) {
		// Setup mock client
		mockClient := &UpdateBranchProtectionMockClient{}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Add repository with branch protection to the remote impl
		originalBranchProtection := &GithubBranchProtection{
			Id:      "BP_123",
			Pattern: "main",
		}
		repo := &GithubRepository{
			Name:  "test-repo",
			Id:    123,
			RefId: "R_123",
			BranchProtections: map[string]*GithubBranchProtection{
				"main": originalBranchProtection,
			},
		}
		remoteImpl.repositories = map[string]*GithubRepository{
			"test-repo": repo,
		}

		// Setup updated branch protection
		updatedBranchProtection := &GithubBranchProtection{
			Id:                       "BP_123",
			Pattern:                  "main",
			RequiresApprovingReviews: true,
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call UpdateRepositoryBranchProtection in dry run mode
		remoteImpl.UpdateRepositoryBranchProtection(ctx, logsCollector, true, "test-repo", updatedBranchProtection)

		// Verify no errors occurred
		assert.False(t, logsCollector.HasErrors())

		// Verify no GraphQL query was made
		assert.Empty(t, mockClient.lastGraphQLQuery)

		// Verify the branch protection was updated in the cache
		assert.Equal(t, updatedBranchProtection, repo.BranchProtections["main"])
	})

	t.Run("error path: GraphQL response error", func(t *testing.T) {
		// Setup mock client with error response
		mockClient := &UpdateBranchProtectionMockClient{
			responseBody: `{
				"errors": [
					{
						"message": "Invalid branch protection configuration",
						"path": ["updateBranchProtectionRule"]
					}
				]
			}`,
		}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)

		// Add repository with branch protection to the remote impl
		originalBranchProtection := &GithubBranchProtection{
			Id:      "BP_123",
			Pattern: "main",
		}
		repo := &GithubRepository{
			Name:  "test-repo",
			Id:    123,
			RefId: "R_123",
			BranchProtections: map[string]*GithubBranchProtection{
				"main": originalBranchProtection,
			},
		}
		remoteImpl.repositories = map[string]*GithubRepository{
			"test-repo": repo,
		}

		// Setup updated branch protection
		updatedBranchProtection := &GithubBranchProtection{
			Id:                       "BP_123",
			Pattern:                  "main",
			RequiresApprovingReviews: true,
		}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()

		// Call UpdateRepositoryBranchProtection
		remoteImpl.UpdateRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", updatedBranchProtection)

		// Verify error was collected
		assert.True(t, logsCollector.HasErrors())
		assert.Contains(t, logsCollector.Errors[0].Error(), "graphql error on UpdateRepositoryBranchProtection")

		// Verify the branch protection remains unchanged in the cache
		assert.Equal(t, originalBranchProtection, repo.BranchProtections["main"])
	})
}

func TestQueryGraphQLBranchProtectionMutationPATRetry(t *testing.T) {
	pat := "test-admin-pat"
	savePAT := config.Config.GithubPersonalAccessToken
	t.Cleanup(func() { config.Config.GithubPersonalAccessToken = savePAT })
	config.Config.GithubPersonalAccessToken = pat

	mockClient := &AddBranchProtectionMockClient{
		responses: []string{
			`{"errors":[{"message":"Resource not accessible by integration ([createBranchProtectionRule])","path":["createBranchProtectionRule"]}]}`,
			`{"data":{"createBranchProtectionRule":{"branchProtectionRule":{"id":"BP_456"}}}}`,
		},
	}
	remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)
	ctx := context.TODO()
	_, res, err := queryGraphQLBranchProtectionMutationWithPATRetry[GraphqlBranchProtectionRuleCreateMutationResponse](remoteImpl, ctx, createBranchProtectionRule, map[string]interface{}{
		"repositoryId": "R_1",
	})
	assert.NoError(t, err)
	assert.Empty(t, res.Errors)
	assert.Equal(t, "BP_456", res.Data.CreateBranchProtectionRule.BranchProtectionRule.Id)
	assert.Equal(t, 2, mockClient.callCount)
	assert.NotNil(t, mockClient.lastGithubToken)
	assert.Equal(t, pat, *mockClient.lastGithubToken)
}

func TestBranchProtectionGraphQLPATFallbackRetry(t *testing.T) {
	pat := "test-admin-pat"
	savePAT := config.Config.GithubPersonalAccessToken
	t.Cleanup(func() { config.Config.GithubPersonalAccessToken = savePAT })

	t.Run("Add retry with PAT after integration error", func(t *testing.T) {
		config.Config.GithubPersonalAccessToken = pat
		mockClient := &AddBranchProtectionMockClient{
			responses: []string{
				`{"errors":[{"message":"Resource not accessible by integration ([createBranchProtectionRule])","path":["createBranchProtectionRule"]}]}`,
				`{"data":{"createBranchProtectionRule":{"branchProtectionRule":{"id":"BP_456"}}}}`,
			},
		}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)
		repo := &GithubRepository{
			Name:              "test-repo",
			Id:                123,
			RefId:             "R_123",
			BranchProtections: make(map[string]*GithubBranchProtection),
		}
		remoteImpl.repositories = map[string]*GithubRepository{"test-repo": repo}
		bp := &GithubBranchProtection{Pattern: "main"}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()
		remoteImpl.AddRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", bp)

		assert.False(t, logsCollector.HasErrors())
		assert.Equal(t, 2, mockClient.callCount)
		assert.NotNil(t, mockClient.lastGithubToken)
		assert.Equal(t, pat, *mockClient.lastGithubToken)
		assert.Equal(t, "BP_456", repo.BranchProtections["main"].Id)
	})

	t.Run("Update retry with PAT after integration error", func(t *testing.T) {
		config.Config.GithubPersonalAccessToken = pat
		mockClient := &UpdateBranchProtectionMockClient{
			responses: []string{
				`{"errors":[{"message":"Resource not accessible by integration ([updateBranchProtectionRule])","path":["updateBranchProtectionRule"]}]}`,
				`{"data":{"updateBranchProtectionRule":{"branchProtectionRule":{"id":"BP_123"}}}}`,
			},
		}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)
		existing := &GithubBranchProtection{Id: "BP_123", Pattern: "main"}
		repo := &GithubRepository{
			Name:              "test-repo",
			Id:                123,
			RefId:             "R_123",
			BranchProtections: map[string]*GithubBranchProtection{"main": existing},
		}
		remoteImpl.repositories = map[string]*GithubRepository{"test-repo": repo}
		updated := &GithubBranchProtection{Id: "BP_123", Pattern: "main", RequiresApprovingReviews: true}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()
		remoteImpl.UpdateRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", updated)

		assert.False(t, logsCollector.HasErrors())
		assert.Equal(t, 2, mockClient.callCount)
		assert.NotNil(t, mockClient.lastGithubToken)
		assert.Equal(t, pat, *mockClient.lastGithubToken)
	})

	t.Run("Delete retry with PAT after integration error", func(t *testing.T) {
		config.Config.GithubPersonalAccessToken = pat
		mockClient := &DeleteBranchProtectionMockClient{
			responses: []string{
				`{"errors":[{"message":"Resource not accessible by integration ([deleteBranchProtectionRule])","path":["deleteBranchProtectionRule"]}]}`,
				`{"data":{"deleteBranchProtectionRule":{"clientMutationId":"x"}}}`,
			},
		}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)
		bp := &GithubBranchProtection{Id: "BP_123", Pattern: "main"}
		repo := &GithubRepository{
			Name:              "test-repo",
			Id:                123,
			RefId:             "R_123",
			BranchProtections: map[string]*GithubBranchProtection{"main": bp},
		}
		remoteImpl.repositories = map[string]*GithubRepository{"test-repo": repo}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()
		remoteImpl.DeleteRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", bp)

		assert.False(t, logsCollector.HasErrors())
		assert.Equal(t, 2, mockClient.callCount)
		assert.NotNil(t, mockClient.lastGithubToken)
		assert.Equal(t, pat, *mockClient.lastGithubToken)
		assert.NotContains(t, repo.BranchProtections, "main")
	})

	t.Run("integration error without PAT does not retry", func(t *testing.T) {
		config.Config.GithubPersonalAccessToken = ""
		mockClient := &UpdateBranchProtectionMockClient{
			responses: []string{
				`{"errors":[{"message":"Resource not accessible by integration ([updateBranchProtectionRule])","path":["updateBranchProtectionRule"]}]}`,
			},
		}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)
		existing := &GithubBranchProtection{Id: "BP_123", Pattern: "main"}
		repo := &GithubRepository{
			Name:              "test-repo",
			Id:                123,
			RefId:             "R_123",
			BranchProtections: map[string]*GithubBranchProtection{"main": existing},
		}
		remoteImpl.repositories = map[string]*GithubRepository{"test-repo": repo}
		updated := &GithubBranchProtection{Id: "BP_123", Pattern: "main", RequiresApprovingReviews: true}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()
		remoteImpl.UpdateRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", updated)

		assert.True(t, logsCollector.HasErrors())
		assert.Equal(t, 1, mockClient.callCount)
		assert.Nil(t, mockClient.lastGithubToken)
		assert.Equal(t, existing, repo.BranchProtections["main"])
	})

	t.Run("non-integration GraphQL error does not retry with PAT", func(t *testing.T) {
		config.Config.GithubPersonalAccessToken = pat
		mockClient := &UpdateBranchProtectionMockClient{
			responses: []string{
				`{"errors":[{"message":"Invalid branch protection configuration","path":["updateBranchProtectionRule"]}]}`,
			},
		}
		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true, true, true)
		existing := &GithubBranchProtection{Id: "BP_123", Pattern: "main"}
		repo := &GithubRepository{
			Name:              "test-repo",
			Id:                123,
			RefId:             "R_123",
			BranchProtections: map[string]*GithubBranchProtection{"main": existing},
		}
		remoteImpl.repositories = map[string]*GithubRepository{"test-repo": repo}
		updated := &GithubBranchProtection{Id: "BP_123", Pattern: "main", RequiresApprovingReviews: true}

		ctx := context.TODO()
		logsCollector := observability.NewLogCollection()
		remoteImpl.UpdateRepositoryBranchProtection(ctx, logsCollector, false, "test-repo", updated)

		assert.True(t, logsCollector.HasErrors())
		assert.Equal(t, 1, mockClient.callCount)
		assert.Nil(t, mockClient.lastGithubToken)
	})
}

func TestGraphqlBranchProtectionRuleMutationResponseUnmarshal(t *testing.T) {
	t.Run("create success", func(t *testing.T) {
		var res GraphqlBranchProtectionRuleCreateMutationResponse
		err := json.Unmarshal([]byte(`{"data":{"createBranchProtectionRule":{"branchProtectionRule":{"id":"BP_1"}}}}`), &res)
		assert.NoError(t, err)
		assert.Equal(t, "BP_1", res.Data.CreateBranchProtectionRule.BranchProtectionRule.Id)
	})
	t.Run("update success", func(t *testing.T) {
		var res GraphqlBranchProtectionRuleUpdateMutationResponse
		err := json.Unmarshal([]byte(`{"data":{"updateBranchProtectionRule":{"branchProtectionRule":{"id":"BP_2"}}}}`), &res)
		assert.NoError(t, err)
		assert.Equal(t, "BP_2", res.Data.UpdateBranchProtectionRule.BranchProtectionRule.Id)
	})
	t.Run("delete success", func(t *testing.T) {
		var res GraphqlBranchProtectionRuleDeleteMutationResponse
		err := json.Unmarshal([]byte(`{"data":{"deleteBranchProtectionRule":{"clientMutationId":"x"}}}`), &res)
		assert.NoError(t, err)
		assert.NotNil(t, res.Data.DeleteBranchProtectionRule.ClientMutationId)
		assert.Equal(t, "x", *res.Data.DeleteBranchProtectionRule.ClientMutationId)
	})
	t.Run("graphql error", func(t *testing.T) {
		var res GraphqlBranchProtectionRuleUpdateMutationResponse
		err := json.Unmarshal([]byte(`{"errors":[{"message":"oops","path":["updateBranchProtectionRule"]}]}`), &res)
		assert.NoError(t, err)
		assert.Len(t, res.Errors, 1)
		assert.Equal(t, "oops", res.Errors[0].Message)
	})
}
