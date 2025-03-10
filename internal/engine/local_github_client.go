package engine

import (
	"context"
	"fmt"

	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
)

type LocalGithubClient interface {
	CreatePullRequest(ctx context.Context, orgname, reponame, baseBranch, branch, title string) (*github.PullRequest, error)
	MergePullRequest(ctx context.Context, pr *github.PullRequest, mainBranch string) error
}

type LocalGithubClientImpl struct {
	client *github.Client
}

func NewLocalGithubClientImpl(ctx context.Context, accesstoken string) *LocalGithubClientImpl {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accesstoken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &LocalGithubClientImpl{
		client: client,
	}
}

// Function to create a PR using GitHub API
func (l *LocalGithubClientImpl) CreatePullRequest(ctx context.Context, orgname, reponame, baseBranch, branch, title string) (*github.PullRequest, error) {

	newPR := &github.NewPullRequest{
		Title: github.String(title),
		Head:  github.String(branch),
		Base:  github.String(baseBranch),
		//		Body:                github.String(prBody),
		MaintainerCanModify: github.Bool(true),
	}

	pr, _, err := l.client.PullRequests.Create(ctx, orgname, reponame, newPR)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

func (l *LocalGithubClientImpl) MergePullRequest(ctx context.Context, pr *github.PullRequest, mainBranch string) error {

	// Create review request
	review := &github.PullRequestReviewRequest{
		Body:  github.String("Approving this PR via GitHub App."),
		Event: github.String("APPROVE"),
	}

	// Approve the PR
	_, _, err := l.client.PullRequests.CreateReview(ctx, pr.GetBase().GetRepo().GetOwner().GetLogin(), pr.GetBase().GetRepo().GetName(), pr.GetNumber(), review)
	if err != nil {
		return fmt.Errorf("failed to approve PR: %v", err)
	}

	// Merge request options
	mergeOpts := &github.PullRequestOptions{
		MergeMethod: "squash", // Options: "merge", "squash", "rebase"
	}

	_, _, err = l.client.PullRequests.Merge(ctx, pr.GetBase().GetRepo().GetOwner().GetLogin(), pr.GetBase().GetRepo().GetName(), pr.GetNumber(), "Merging PR", mergeOpts)

	if err != nil {
		return err
	}
	return nil
}
