package engine

import (
	"context"
	"testing"

	"github.com/google/go-github/v55/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLocalGithubClient is a mock implementation of the LocalGithubClient interface
type MockLocalGithubClient struct {
	mock.Mock
}

func (m *MockLocalGithubClient) CreatePullRequest(ctx context.Context, orgname, reponame, baseBranch, branch, title string) (*github.PullRequest, error) {
	args := m.Called(ctx, orgname, reponame, baseBranch, branch, title)
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

func (m *MockLocalGithubClient) MergePullRequest(ctx context.Context, pr *github.PullRequest, mainBranch string) error {
	args := m.Called(ctx, pr, mainBranch)
	return args.Error(0)
}

func TestCreatePullRequest(t *testing.T) {
	mockClient := new(MockLocalGithubClient)
	ctx := context.Background()
	org := "test-org"
	repo := "test-repo"
	baseBranch := "main"
	branch := "feature-branch"
	title := "Test PR"

	expectedPR := &github.PullRequest{Number: github.Int(123)}

	// Set up expectations
	mockClient.On("CreatePullRequest", ctx, org, repo, baseBranch, branch, title).Return(expectedPR, nil)

	// Call the function
	pr, err := mockClient.CreatePullRequest(ctx, org, repo, baseBranch, branch, title)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, expectedPR, pr)

	// Verify expectations
	mockClient.AssertExpectations(t)
}

func TestMergePullRequest(t *testing.T) {
	mockClient := new(MockLocalGithubClient)
	ctx := context.Background()
	pr := &github.PullRequest{Number: github.Int(123)}
	mainBranch := "main"

	// Set up expectations
	mockClient.On("MergePullRequest", ctx, pr, mainBranch).Return(nil)

	// Call the function
	err := mockClient.MergePullRequest(ctx, pr, mainBranch)

	// Assertions
	assert.NoError(t, err)

	// Verify expectations
	mockClient.AssertExpectations(t)
}
