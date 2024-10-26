package internal

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/engine"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/github"
	"github.com/Alayacare/goliac/internal/usersync"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/sirupsen/logrus"
)

const (
	GOLIAC_GIT_TAG = "goliac"
)

/*
 * Goliac is the main interface of the application.
 * It is used to load and validate a goliac repository and apply it to github.
 */
type Goliac interface {
	// will run and apply the reconciliation,
	// it returns an error if something went wrong, and a detailed list of errors and warnings
	Apply(ctx context.Context, dryrun bool, repositoryUrl, branch string, forcesync bool) (error, []error, []entity.Warning, *engine.UnmanagedResources)

	// will clone run the user-plugin to sync users, and will commit to the team repository
	UsersUpdate(ctx context.Context, repositoryUrl, branch string, dryrun bool, force bool) error

	// flush remote cache
	FlushCache()

	GetLocal() engine.GoliacLocalResources
}

type GoliacImpl struct {
	local        engine.GoliacLocal
	remote       engine.GoliacRemoteExecutor
	githubClient github.GitHubClient
	repoconfig   *config.RepositoryConfig
}

func NewGoliacImpl() (Goliac, error) {
	githubClient, err := github.NewGitHubClientImpl(
		config.Config.GithubServer,
		config.Config.GithubAppOrganization,
		config.Config.GithubAppID,
		config.Config.GithubAppPrivateKeyFile,
	)

	if err != nil {
		return nil, err
	}

	remote := engine.NewGoliacRemoteImpl(githubClient)

	usersync.InitPlugins(githubClient)

	return &GoliacImpl{
		local:        engine.NewGoliacLocalImpl(),
		githubClient: githubClient,
		remote:       remote,
		repoconfig:   &config.RepositoryConfig{},
	}, nil
}

func (g *GoliacImpl) GetLocal() engine.GoliacLocalResources {
	return g.local
}

func (g *GoliacImpl) FlushCache() {
	g.remote.FlushCache()
}

func (g *GoliacImpl) Apply(ctx context.Context, dryrun bool, repositoryUrl, branch string, forcesync bool) (error, []error, []entity.Warning, *engine.UnmanagedResources) {
	err, errs, warns := g.loadAndValidateGoliacOrganization(ctx, repositoryUrl, branch)
	defer g.local.Close()
	if err != nil {
		return fmt.Errorf("failed to load and validate: %s", err), errs, warns, nil
	}
	u, err := url.Parse(repositoryUrl)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %v", repositoryUrl, err), errs, warns, nil
	}

	teamsreponame := strings.TrimSuffix(path.Base(u.Path), filepath.Ext(path.Base(u.Path)))

	unmanaged, err := g.applyToGithub(ctx, dryrun, teamsreponame, branch, forcesync)
	if err != nil {
		return err, errs, warns, unmanaged
	}
	return nil, errs, warns, unmanaged
}

func (g *GoliacImpl) loadAndValidateGoliacOrganization(ctx context.Context, repositoryUrl, branch string) (error, []error, []entity.Warning) {
	var errs []error
	var warns []entity.Warning
	if strings.HasPrefix(repositoryUrl, "https://") || strings.HasPrefix(repositoryUrl, "git@") {
		accessToken, err := g.githubClient.GetAccessToken(ctx)
		if err != nil {
			return err, nil, nil
		}

		err = g.local.Clone(accessToken, repositoryUrl, branch)
		if err != nil {
			return fmt.Errorf("unable to clone: %v", err), nil, nil
		}
		repoconfig, err := g.local.LoadRepoConfig()
		if err != nil {
			return fmt.Errorf("unable to read goliac.yaml config file: %v", err), nil, nil
		}
		g.repoconfig = repoconfig

		errs, warns = g.local.LoadAndValidate()
	} else {
		// Local
		fs := osfs.New(repositoryUrl)
		errs, warns = g.local.LoadAndValidateLocal(fs)
	}

	for _, warn := range warns {
		logrus.Warn(warn)
	}
	if len(errs) != 0 {
		for _, err := range errs {
			logrus.Error(err)
		}
		return fmt.Errorf("not able to load and validate the goliac organization: see logs"), errs, warns
	}

	return nil, errs, warns
}

/*
 * To ensure we can parse teams git logs, commit by commit (for auditing purpose),
 * we must ensure that the "squqsh and merge" option is the only option.
 * Else we may append to apply commits that are part of a PR, but wasn't the final PR commit state
 */
func (g *GoliacImpl) forceSquashMergeOnTeamsRepo(ctx context.Context, teamreponame string, branchname string) error {
	_, err := g.githubClient.CallRestAPI(ctx, fmt.Sprintf("/repos/%s/%s", config.Config.GithubAppOrganization, teamreponame), "PATCH",
		map[string]interface{}{
			"allow_merge_commit": false,
			"allow_rebase_merge": false,
			"allow_squash_merge": true,
		})
	if err != nil {
		return err
	}

	// add an extra branch protection
	contexts := []string{}

	if config.Config.ServerGitBranchProtectionRequiredCheck != "" {
		contexts = append(contexts, config.Config.ServerGitBranchProtectionRequiredCheck)
	}
	_, err = g.githubClient.CallRestAPI(ctx, fmt.Sprintf("/repos/%s/%s/branches/%s/protection", config.Config.GithubAppOrganization, teamreponame, branchname), "PUT",
		map[string]interface{}{
			"required_status_checks": map[string]interface{}{
				"strict":   true,     // // This ensures branches are up to date before merging
				"contexts": contexts, // Status checks to enforce, see scaffold.go for the job name
			},
			"enforce_admins":                nil,
			"required_pull_request_reviews": nil,
			// required_pull_request_reviews could have been
			//{
			// "dismiss_stale_reviews": true,   // Optional: Whether or not approved reviews are dismissed when a new commit is pushed.
			//"require_code_owner_reviews": false,  // Optional: If set, only code owners can approve the PR.
			//"required_approving_review_count": 1   // Number of approvals required. Set this to 1 for one review.
			//},
			"restrictions": nil,
		})
	return err
}

func (g *GoliacImpl) applyToGithub(ctx context.Context, dryrun bool, teamreponame string, branch string, forceresync bool) (*engine.UnmanagedResources, error) {
	err := g.remote.Load(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("error when fetching data from Github: %v", err)
	}

	if !dryrun {
		err := g.forceSquashMergeOnTeamsRepo(ctx, teamreponame, branch)
		if err != nil {
			logrus.Errorf("Error when ensuring PR on %s, repo can only be done via squash and merge: %v", teamreponame, err)
		}
	}

	// if the repo was just archived in a previous commit and we "resume it"
	// so we keep a track of all repos that we want to archive until the end of the process
	reposToArchive := make(map[string]*engine.GithubRepoComparable)
	var unmanaged *engine.UnmanagedResources

	commits, err := g.local.ListCommitsFromTag(GOLIAC_GIT_TAG)
	// if we can get commits
	if err != nil {
		ga := NewGithubBatchExecutor(g.remote, g.repoconfig.MaxChangesets)
		reconciliator := engine.NewGoliacReconciliatorImpl(ga, g.repoconfig)

		unmanaged, err = reconciliator.Reconciliate(ctx, g.local, g.remote, teamreponame, dryrun, reposToArchive)
		if err != nil {
			return unmanaged, fmt.Errorf("error when reconciliating: %v", err)
		}
		// if we resync, and dont have commits, let's resync the latest (HEAD) commit
		// or if are not in enterprise mode and cannot guarrantee that PR commits are squashed
	} else if (len(commits) == 0 && forceresync) || !g.remote.IsEnterprise() {

		ga := NewGithubBatchExecutor(g.remote, g.repoconfig.MaxChangesets)
		reconciliator := engine.NewGoliacReconciliatorImpl(ga, g.repoconfig)
		commit, err := g.local.GetHeadCommit()

		if err == nil {
			ctx = context.WithValue(ctx, engine.KeyAuthor, fmt.Sprintf("%s <%s>", commit.Author.Name, commit.Author.Email))
		}

		unmanaged, err = reconciliator.Reconciliate(ctx, g.local, g.remote, teamreponame, dryrun, reposToArchive)
		if err != nil {
			return unmanaged, fmt.Errorf("error when reconciliating: %v", err)
		}
	} else {
		// we have 1 or more commits to apply
		var lastErr error
		for _, commit := range commits {
			if err := g.local.CheckoutCommit(commit); err == nil {
				errs, _ := g.local.LoadAndValidate()
				if len(errs) > 0 {
					for _, err := range errs {
						logrus.Error(err)
					}
					continue
				}
				ga := NewGithubBatchExecutor(g.remote, g.repoconfig.MaxChangesets)
				reconciliator := engine.NewGoliacReconciliatorImpl(ga, g.repoconfig)

				ctx := context.WithValue(ctx, engine.KeyAuthor, fmt.Sprintf("%s <%s>", commit.Author.Name, commit.Author.Email))
				unmanaged, err = reconciliator.Reconciliate(ctx, g.local, g.remote, teamreponame, dryrun, reposToArchive)
				if err != nil {
					// we keep the last error and continue
					// to see if the next commit can be applied without error
					// (like if we reached the max changesets, but the next commit will fix it)
					lastErr = fmt.Errorf("error when reconciliating: %v", err)
				} else {
					lastErr = nil
				}
				if !dryrun && err == nil {
					accessToken, err := g.githubClient.GetAccessToken(ctx)
					if err != nil {
						return unmanaged, err
					}
					g.local.PushTag(GOLIAC_GIT_TAG, commit.Hash, accessToken)
				}
			} else {
				logrus.Errorf("Not able to checkout commit %s", commit.Hash.String())
			}
		}
		if lastErr != nil {
			return unmanaged, lastErr
		}
	}
	accessToken, err := g.githubClient.GetAccessToken(ctx)
	if err != nil {
		return unmanaged, err
	}

	// if we have repos to create as archived
	if len(reposToArchive) > 0 && !dryrun {
		reposToArchiveList := make([]string, 0)
		for reponame := range reposToArchive {
			reposToArchiveList = append(reposToArchiveList, reponame)
		}
		err = g.local.ArchiveRepos(reposToArchiveList, accessToken, branch, GOLIAC_GIT_TAG)
		if err != nil {
			return unmanaged, fmt.Errorf("error when archiving repos: %v", err)
		}
	}
	err = g.local.UpdateAndCommitCodeOwners(g.repoconfig, dryrun, accessToken, branch, GOLIAC_GIT_TAG)
	if err != nil {
		return unmanaged, fmt.Errorf("error when updating and commiting: %v", err)
	}
	return unmanaged, nil
}

func (g *GoliacImpl) UsersUpdate(ctx context.Context, repositoryUrl, branch string, dryrun bool, force bool) error {
	accessToken, err := g.githubClient.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	err = g.local.Clone(accessToken, repositoryUrl, branch)
	if err != nil {
		return err
	}
	defer g.local.Close()

	repoconfig, err := g.local.LoadRepoConfig()
	if err != nil {
		return fmt.Errorf("unable to read goliac.yaml config file: %v", err)
	}

	userplugin, found := engine.GetUserSyncPlugin(repoconfig.UserSync.Plugin)
	if !found {
		return fmt.Errorf("User Sync Plugin %s not found", repoconfig.UserSync.Plugin)
	}

	err = g.local.SyncUsersAndTeams(repoconfig, userplugin, accessToken, dryrun, force)
	return err
}
