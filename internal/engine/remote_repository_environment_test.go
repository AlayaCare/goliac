package engine

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// LoadEnvironmentVariablesMockClient is a mock client for testing environment variables loading
type LoadEnvironmentVariablesMockClient struct {
	lastEndpoint    string
	lastParameters  string
	lastMethod      string
	lastBody        map[string]interface{}
	lastGithubToken *string
	shouldError     bool
	errorMessage    string
	responseBody    string
	accessToken     string
	accessTokenErr  error
	mutex           sync.Mutex
	callCount       int
}

func (m *LoadEnvironmentVariablesMockClient) QueryGraphQLAPI(ctx context.Context, query string, variables map[string]interface{}) ([]byte, error) {
	return nil, nil
}

func (m *LoadEnvironmentVariablesMockClient) CallRestAPI(ctx context.Context, endpoint, parameters, method string, body map[string]interface{}, githubToken *string) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.callCount++
	m.lastEndpoint = endpoint
	m.lastParameters = parameters
	m.lastMethod = method
	m.lastBody = body
	m.lastGithubToken = githubToken

	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}

	return []byte(m.responseBody), nil
}

func (m *LoadEnvironmentVariablesMockClient) GetAccessToken(ctx context.Context) (string, error) {
	if m.accessTokenErr != nil {
		return "", m.accessTokenErr
	}
	if m.accessToken == "" {
		return "mock-token", nil
	}
	return m.accessToken, nil
}

func (m *LoadEnvironmentVariablesMockClient) CreateJWT() (string, error) {
	return "mock-jwt", nil
}

func (m *LoadEnvironmentVariablesMockClient) GetAppSlug() string {
	return "mock-app"
}

func TestLoadEnvironmentVariables(t *testing.T) {
	t.Run("happy path: load environment variables for multiple repositories", func(t *testing.T) {
		// Setup mock client
		mockClient := &LoadEnvironmentVariablesMockClient{
			responseBody: `{
				"total_count": 2,
				"variables": [
					{
						"name": "VAR1",
						"value": "value1",
						"created_at": "2023-01-01T00:00:00Z",
						"updated_at": "2023-01-01T00:00:00Z"
					},
					{
						"name": "VAR2",
						"value": "value2",
						"created_at": "2023-01-01T00:00:00Z",
						"updated_at": "2023-01-01T00:00:00Z"
					}
				]
			}`,
		}

		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true)
		repositories := map[string]*GithubRepository{
			"repo1": {
				Name: "repo1",
				Id:   123,
				Environments: map[string]*GithubEnvironment{
					"production": {
						Name:      "production",
						Variables: map[string]string{},
					},
				},
			},
			"repo2": {
				Name: "repo2",
				Id:   456,
				Environments: map[string]*GithubEnvironment{
					"staging": {
						Name:      "staging",
						Variables: map[string]string{},
					},
				},
			},
		}

		ctx := context.TODO()

		// Call loadEnvironmentVariables
		envVars, err := remoteImpl.loadEnvironmentVariables(ctx, 2, repositories)

		// Verify no errors occurred
		assert.NoError(t, err)
		assert.NotNil(t, envVars)
		assert.Equal(t, 2, len(envVars))

		// Verify environment variables were loaded correctly for repo1
		repo1Vars := envVars["repo1"]
		assert.NotNil(t, repo1Vars)
		assert.Equal(t, 1, len(repo1Vars))
		prodVars := repo1Vars["production"]
		assert.NotNil(t, prodVars)
		assert.Equal(t, 2, len(prodVars))
		assert.Equal(t, "value1", prodVars["VAR1"].Value)
		assert.Equal(t, "value2", prodVars["VAR2"].Value)

		// Verify environment variables were loaded correctly for repo2
		repo2Vars := envVars["repo2"]
		assert.NotNil(t, repo2Vars)
		assert.Equal(t, 1, len(repo2Vars))
		stagingVars := repo2Vars["staging"]
		assert.NotNil(t, stagingVars)
		assert.Equal(t, 2, len(stagingVars))
		assert.Equal(t, "value1", stagingVars["VAR1"].Value)
		assert.Equal(t, "value2", stagingVars["VAR2"].Value)

		// Verify API calls were made correctly
		assert.Equal(t, 3, mockClient.callCount) // 2 calls per repository (1 per environment)
	})

	t.Run("error path: API error", func(t *testing.T) {
		mockClient := &LoadEnvironmentVariablesMockClient{
			shouldError:  true,
			errorMessage: "API error",
		}

		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true)
		repositories := map[string]*GithubRepository{
			"repo1": {
				Name: "repo1",
				Id:   123,
				Environments: map[string]*GithubEnvironment{
					"production": {
						Name:      "production",
						Variables: map[string]string{},
					},
				},
			},
		}

		ctx := context.TODO()

		// Call loadEnvironmentVariables
		_, err := remoteImpl.loadEnvironmentVariables(ctx, 2, repositories)

		// Verify error was returned
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API error")
	})

	t.Run("happy path: empty repositories", func(t *testing.T) {
		mockClient := &LoadEnvironmentVariablesMockClient{}

		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true)
		repositories := map[string]*GithubRepository{}

		ctx := context.TODO()

		// Call loadEnvironmentVariables
		envVars, err := remoteImpl.loadEnvironmentVariables(ctx, 2, repositories)

		// Verify no errors and empty map returned
		assert.NoError(t, err)
		assert.NotNil(t, envVars)
		assert.Empty(t, envVars)
	})

	t.Run("happy path: repositories with no environments", func(t *testing.T) {
		mockClient := &LoadEnvironmentVariablesMockClient{}

		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true)
		repositories := map[string]*GithubRepository{
			"repo1": {
				Name:         "repo1",
				Id:           123,
				Environments: map[string]*GithubEnvironment{},
			},
		}

		ctx := context.TODO()

		// Call loadEnvironmentVariables
		envVars, err := remoteImpl.loadEnvironmentVariables(ctx, 2, repositories)

		// Verify no errors and empty map returned
		assert.NoError(t, err)
		assert.NotNil(t, envVars)
		assert.Equal(t, 1, len(envVars))
		assert.Empty(t, envVars["repo1"])
	})

	t.Run("error path: invalid JSON response", func(t *testing.T) {
		mockClient := &LoadEnvironmentVariablesMockClient{
			responseBody: `invalid json`,
		}

		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true)
		repositories := map[string]*GithubRepository{
			"repo1": {
				Name: "repo1",
				Id:   123,
				Environments: map[string]*GithubEnvironment{
					"production": {
						Name:      "production",
						Variables: map[string]string{},
					},
				},
			},
		}

		ctx := context.TODO()

		// Call loadEnvironmentVariables
		_, err := remoteImpl.loadEnvironmentVariables(ctx, 2, repositories)

		// Verify error was returned
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("happy path: empty variables list", func(t *testing.T) {
		mockClient := &LoadEnvironmentVariablesMockClient{
			responseBody: `{
				"total_count": 0,
				"variables": []
			}`,
		}

		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true)
		repositories := map[string]*GithubRepository{
			"repo1": {
				Name: "repo1",
				Id:   123,
				Environments: map[string]*GithubEnvironment{
					"production": {
						Name:      "production",
						Variables: map[string]string{},
					},
				},
			},
		}

		ctx := context.TODO()

		// Call loadEnvironmentVariables
		envVars, err := remoteImpl.loadEnvironmentVariables(ctx, 2, repositories)

		// Verify no errors and empty map returned
		assert.NoError(t, err)
		assert.NotNil(t, envVars)
		assert.Equal(t, 1, len(envVars))
		assert.Equal(t, 1, len(envVars["repo1"]))
		assert.Empty(t, envVars["repo1"]["production"])
	})

	t.Run("happy path: concurrent loading with rate limiting", func(t *testing.T) {
		mockClient := &LoadEnvironmentVariablesMockClient{
			responseBody: `{
				"total_count": 1,
				"variables": [
					{
						"name": "VAR1",
						"value": "value1",
						"created_at": "2023-01-01T00:00:00Z",
						"updated_at": "2023-01-01T00:00:00Z"
					}
				]
			}`,
		}

		remoteImpl := NewGoliacRemoteImpl(mockClient, "myorg", true)
		repositories := map[string]*GithubRepository{
			"repo1": {
				Name: "repo1",
				Id:   123,
				Environments: map[string]*GithubEnvironment{
					"production": {
						Name:      "production",
						Variables: map[string]string{},
					},
				},
			},
			"repo2": {
				Name: "repo2",
				Id:   456,
				Environments: map[string]*GithubEnvironment{
					"staging": {
						Name:      "staging",
						Variables: map[string]string{},
					},
				},
			},
			"repo3": {
				Name: "repo3",
				Id:   789,
				Environments: map[string]*GithubEnvironment{
					"development": {
						Name:      "development",
						Variables: map[string]string{},
					},
				},
			},
		}

		ctx := context.TODO()

		// Call loadEnvironmentVariables with maxGoroutines = 2
		envVars, err := remoteImpl.loadEnvironmentVariables(ctx, 2, repositories)

		// Verify no errors occurred
		assert.NoError(t, err)
		assert.NotNil(t, envVars)
		assert.Equal(t, 3, len(envVars))

		// Verify all repositories were processed
		assert.NotNil(t, envVars["repo1"])
		assert.NotNil(t, envVars["repo2"])
		assert.NotNil(t, envVars["repo3"])

		// Verify rate limiting was applied (max 2 concurrent goroutines)
		assert.LessOrEqual(t, mockClient.callCount, 6) // 3 repositories * 2 environments
	})
}
