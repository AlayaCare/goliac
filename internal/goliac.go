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
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

const (
	GOLIAC_GIT_TAG = "goliac"
)

/*
 * Goliac is the main interface of the application.
 * It is used to load and validate a goliac repository and apply it to github.
 */
type Goliac interface {
	// will run and apply the reconciliation
	Apply(dryrun bool, repositoryUrl, branch string, forcesync bool) error

	// will clone run the user-plugin to sync users, and will commit to the team repository
	UsersUpdate(repositoryUrl, branch string, dryrun bool, force bool) error

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

func (g *GoliacImpl) Apply(dryrun bool, repositoryUrl, branch string, forcesync bool) error {
	err := g.loadAndValidateGoliacOrganization(repositoryUrl, branch)
	defer g.local.Close()
	if err != nil {
		return fmt.Errorf("failed to load and validate: %s", err)
	}
	u, err := url.Parse(repositoryUrl)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %v", repositoryUrl, err)
	}

	teamsreponame := strings.TrimSuffix(path.Base(u.Path), filepath.Ext(path.Base(u.Path)))

	err = g.applyToGithub(dryrun, teamsreponame, branch, forcesync)
	if err != nil {
		return err
	}
	return nil
}

func (g *GoliacImpl) loadAndValidateGoliacOrganization(repositoryUrl, branch string) error {
	var errs []error
	var warns []entity.Warning
	if strings.HasPrefix(repositoryUrl, "https://") || strings.HasPrefix(repositoryUrl, "git@") {
		accessToken, err := g.githubClient.GetAccessToken()
		if err != nil {
			return err
		}

		err = g.local.Clone(accessToken, repositoryUrl, branch)
		if err != nil {
			return fmt.Errorf("unable to clone: %v", err)
		}
		err, repoconfig := g.local.LoadRepoConfig()
		if err != nil {
			return fmt.Errorf("unable to read goliac.yaml config file: %v", err)
		}
		g.repoconfig = repoconfig

		errs, warns = g.local.LoadAndValidate()
	} else {
		// Local
		fs := afero.NewOsFs()
		errs, warns = g.local.LoadAndValidateLocal(fs, repositoryUrl)
	}

	for _, warn := range warns {
		logrus.Warn(warn)
	}
	if len(errs) != 0 {
		for _, err := range errs {
			logrus.Error(err)
		}
		return fmt.Errorf("Not able to load and validate the goliac organization: see logs")
	}

	return nil
}

/*
 * To ensure we can parse teams git logs, commit by commit (for auditing purpose),
 * we must ensure that the "squqsh and merge" option is the only option.
 * Else we may append to apply commits that are part of a PR, but wasn't the final PR commit state
 */
func (g *GoliacImpl) forceSquashMergeOnTeamsRepo(teamreponame string, branchname string) error {
	_, err := g.githubClient.CallRestAPI(fmt.Sprintf("/repos/%s/%s", config.Config.GithubAppOrganization, teamreponame), "PATCH",
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
	_, err = g.githubClient.CallRestAPI(fmt.Sprintf("/repos/%s/%s/branches/%s/protection", config.Config.GithubAppOrganization, teamreponame, branchname), "PUT",
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

func (g *GoliacImpl) applyToGithub(dryrun bool, teamreponame string, branch string, forceresync bool) error {
	err := g.remote.Load(false)
	if err != nil {
		return fmt.Errorf("Error when fetching data from Github: %v", err)
	}

	if !dryrun {
		err := g.forceSquashMergeOnTeamsRepo(teamreponame, branch)
		if err != nil {
			logrus.Errorf("Error when ensuring PR on %s repo can only be done via squash and merge: %v", teamreponame, err)
		}
	}

	reposToArchive := make(map[string]*engine.GithubRepoComparable)

	commits, err := g.local.ListCommitsFromTag(GOLIAC_GIT_TAG)
	// if we can get commits
	if err != nil {
		ga := NewGithubBatchExecutor(g.remote, g.repoconfig.MaxChangesets)
		reconciliator := engine.NewGoliacReconciliatorImpl(ga, g.repoconfig)

		ctx := context.TODO()

		err = reconciliator.Reconciliate(ctx, g.local, g.remote, teamreponame, dryrun, reposToArchive)
		if err != nil {
			return fmt.Errorf("Error when reconciliating: %v", err)
		}
		// if we resync, and dont have commits, let's resync the latest (HEAD) commit
		// or if are not in enterprise mode and cannot guarrantee that PR commits are squashed
	} else if (len(commits) == 0 && forceresync) || !g.remote.IsEnterprise() {

		ga := NewGithubBatchExecutor(g.remote, g.repoconfig.MaxChangesets)
		reconciliator := engine.NewGoliacReconciliatorImpl(ga, g.repoconfig)
		commit, err := g.local.GetHeadCommit()

		ctx := context.TODO()

		if err == nil {
			ctx = context.WithValue(context.TODO(), engine.KeyAuthor, fmt.Sprintf("%s <%s>", commit.Author.Name, commit.Author.Email))
		}

		err = reconciliator.Reconciliate(ctx, g.local, g.remote, teamreponame, dryrun, reposToArchive)
		if err != nil {
			return fmt.Errorf("Error when reconciliating: %v", err)
		}
	} else {
		// we have 1 or more commits to apply
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

				ctx := context.WithValue(context.TODO(), engine.KeyAuthor, fmt.Sprintf("%s <%s>", commit.Author.Name, commit.Author.Email))
				err = reconciliator.Reconciliate(ctx, g.local, g.remote, teamreponame, dryrun, reposToArchive)
				if err != nil {
					return fmt.Errorf("Error when reconciliating: %v", err)
				}
				if !dryrun {
					accessToken, err := g.githubClient.GetAccessToken()
					if err != nil {
						return err
					}
					g.local.PushTag(GOLIAC_GIT_TAG, commit.Hash, accessToken)
				}
			} else {
				logrus.Errorf("Not able to checkout commit %s", commit.Hash.String())
			}
		}
	}
	accessToken, err := g.githubClient.GetAccessToken()
	if err != nil {
		return err
	}

	// if we have repos to create as archived
	if len(reposToArchive) > 0 && !dryrun {
		reposToArchiveList := make([]string, 0)
		for reponame := range reposToArchive {
			reposToArchiveList = append(reposToArchiveList, reponame)
		}
		err = g.local.ArchiveRepos(reposToArchiveList, accessToken, branch, GOLIAC_GIT_TAG)
		if err != nil {
			return fmt.Errorf("Error when archiving repos: %v", err)
		}
	}
	err = g.local.UpdateAndCommitCodeOwners(g.repoconfig, dryrun, accessToken, branch, GOLIAC_GIT_TAG)
	if err != nil {
		return fmt.Errorf("Error when updating and commiting: %v", err)
	}
	return nil
}

func (g *GoliacImpl) UsersUpdate(repositoryUrl, branch string, dryrun bool, force bool) error {
	accessToken, err := g.githubClient.GetAccessToken()
	if err != nil {
		return err
	}

	err = g.local.Clone(accessToken, repositoryUrl, branch)
	if err != nil {
		return err
	}
	defer g.local.Close()

	err, repoconfig := g.local.LoadRepoConfig()
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
