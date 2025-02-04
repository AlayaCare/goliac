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
	"github.com/Alayacare/goliac/internal/observability"
	"github.com/Alayacare/goliac/internal/usersync"
	"github.com/go-git/go-billy/v5"
	"github.com/sirupsen/logrus"
)

const (
	GOLIAC_GIT_TAG = "goliac"
)

type GoliacObservability interface {
	SetRemoteObservability(feedback observability.RemoteObservability) error // if you want to get feedback on the loading process
}

/*
 * Goliac is the main interface of the application.
 * It is used to load and validate a goliac repository and apply it to github.
 */
type Goliac interface {
	GoliacObservability

	// will run and apply the reconciliation,
	// it returns an error if something went wrong, and a detailed list of errors and warnings
	Apply(ctx context.Context, fs billy.Filesystem, dryrun bool, repositoryUrl, branch string) (error, []error, []entity.Warning, *engine.UnmanagedResources)

	// will clone run the user-plugin to sync users, and will commit to the team repository, return true if a change was done
	UsersUpdate(ctx context.Context, fs billy.Filesystem, repositoryUrl, branch string, dryrun bool, force bool) (bool, error)

	// flush remote cache
	FlushCache()

	GetLocal() engine.GoliacLocalResources
}

type GoliacImpl struct {
	local              engine.GoliacLocal
	remote             engine.GoliacRemoteExecutor
	localGithubClient  github.GitHubClient // github client for team repository operations
	remoteGithubClient github.GitHubClient // github client for admin operations
	repoconfig         *config.RepositoryConfig
	feedback           observability.RemoteObservability
}

func NewGoliacImpl() (Goliac, error) {
	remoteGithubClient, err := github.NewGitHubClientImpl(
		config.Config.GithubServer,
		config.Config.GithubAppOrganization,
		config.Config.GithubAppID,
		config.Config.GithubAppPrivateKeyFile,
	)
	if err != nil {
		return nil, err
	}

	localGithubClient, err := github.NewGitHubClientImpl(
		config.Config.GithubServer,
		config.Config.GithubAppOrganization,
		config.Config.GithubTeamAppID,
		config.Config.GithubTeamAppPrivateKeyFile,
	)
	if err != nil {
		return nil, err
	}

	remote := engine.NewGoliacRemoteImpl(remoteGithubClient)

	usersync.InitPlugins(remoteGithubClient)

	return &GoliacImpl{
		local:              engine.NewGoliacLocalImpl(),
		remoteGithubClient: remoteGithubClient,
		localGithubClient:  localGithubClient,
		remote:             remote,
		repoconfig:         &config.RepositoryConfig{},
		feedback:           nil,
	}, nil
}

func (g *GoliacImpl) GetLocal() engine.GoliacLocalResources {
	return g.local
}

func (g *GoliacImpl) SetRemoteObservability(feedback observability.RemoteObservability) error {
	g.feedback = feedback
	g.remote.SetRemoteObservability(feedback)

	if feedback != nil {
		nb, err := g.remote.CountAssets(context.Background())
		if err != nil {
			return fmt.Errorf("error when counting assets: %v", err)
		}
		feedback.Init(nb)
	}
	return nil
}

func (g *GoliacImpl) FlushCache() {
	g.remote.FlushCache()
}

func (g *GoliacImpl) Apply(ctx context.Context, fs billy.Filesystem, dryrun bool, repositoryUrl, branch string) (error, []error, []entity.Warning, *engine.UnmanagedResources) {
	err, errs, warns := g.loadAndValidateGoliacOrganization(ctx, fs, repositoryUrl, branch)
	defer g.local.Close(fs)
	if err != nil {
		return fmt.Errorf("failed to load and validate: %s", err), errs, warns, nil
	}
	if !strings.HasPrefix(repositoryUrl, "https://") &&
		!strings.HasPrefix(repositoryUrl, "inmemory:///") { // <- only for testing purposes
		return fmt.Errorf("local mode is not supported for plan/apply, you must specify the https url of the remote team git repository. Check the documentation"), errs, warns, nil
	}

	u, err := url.Parse(repositoryUrl)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %v", repositoryUrl, err), errs, warns, nil
	}

	teamreponame := strings.TrimSuffix(path.Base(u.Path), filepath.Ext(path.Base(u.Path)))

	// ensure that the team repo is configured to only allow squash and merge
	if !dryrun {
		err := g.forceSquashMergeOnTeamsRepo(ctx, teamreponame, branch)
		if err != nil {
			return fmt.Errorf("error when ensuring PR on %s, repo can only be done via squash and merge: %v", teamreponame, err), errs, warns, nil
		}
	}

	unmanaged, err := g.applyToGithub(ctx, dryrun, config.Config.GithubAppOrganization, teamreponame, branch, config.Config.SyncUsersBeforeApply)
	for _, warn := range warns {
		logrus.Warn(warn)
	}
	if err != nil {
		return err, errs, warns, unmanaged
	}

	return nil, errs, warns, unmanaged
}

func (g *GoliacImpl) loadAndValidateGoliacOrganization(ctx context.Context, fs billy.Filesystem, repositoryUrl, branch string) (error, []error, []entity.Warning) {
	var errs []error
	var warns []entity.Warning
	if strings.HasPrefix(repositoryUrl, "https://") || strings.HasPrefix(repositoryUrl, "git@") || strings.HasPrefix(repositoryUrl, "inmemory:///") {
		accessToken := ""
		var err error
		if strings.HasPrefix(repositoryUrl, "https://") {
			accessToken, err = g.localGithubClient.GetAccessToken(ctx)
			if err != nil {
				return err, nil, nil
			}
		}

		err = g.local.Clone(fs, accessToken, repositoryUrl, branch)
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
		subfs, err := fs.Chroot(repositoryUrl)
		if err != nil {
			return fmt.Errorf("unable to chroot to %s: %v", repositoryUrl, err), nil, nil
		}
		errs, warns = g.local.LoadAndValidateLocal(subfs)
	}

	for _, warn := range warns {
		logrus.Debug(warn)
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
	// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#update-a-repository
	_, err := g.remoteGithubClient.CallRestAPI(ctx,
		fmt.Sprintf("/repos/%s/%s", config.Config.GithubAppOrganization, teamreponame),
		"",
		"PATCH",
		map[string]interface{}{
			"allow_merge_commit":     false, // allow merging pull requests with a merge commit
			"allow_rebase_merge":     false, // allow rebase-merging pull requests
			"allow_squash_merge":     true,  // allow squash-merging pull requests
			"delete_branch_on_merge": true,  // automatically deleting head branches when pull requests are merged
		})
	if err != nil {
		return err
	}

	// add an extra branch protection
	contexts := []string{}

	if config.Config.ServerGitBranchProtectionRequiredCheck != "" {
		contexts = append(contexts, config.Config.ServerGitBranchProtectionRequiredCheck)
	}
	// https://docs.github.com/en/rest/branches/branch-protection?apiVersion=2022-11-28#update-branch-protection
	_, err = g.remoteGithubClient.CallRestAPI(ctx,
		fmt.Sprintf("/repos/%s/%s/branches/%s/protection", config.Config.GithubAppOrganization, teamreponame, branchname),
		"",
		"PUT",
		map[string]interface{}{
			"required_status_checks": map[string]interface{}{
				"strict":   true,     // // This ensures branches are up to date before merging
				"contexts": contexts, // Status checks to enforce, see scaffold.go for the job name
			},
			"enforce_admins": nil,
			"required_pull_request_reviews": map[string]interface{}{
				"require_last_push_approval": true, // Optional: Require approval from the author of the latest commit.
			},
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

/*
Apply the changes to the github team repository:
  - load the data from github
  - sync users if needed (from external sources)
  - apply the changes
  - update the codeowners file
*/
func (g *GoliacImpl) applyToGithub(ctx context.Context, dryrun bool, githubOrganization string, teamreponame string, branch string, syncusersbeforeapply bool) (*engine.UnmanagedResources, error) {
	err := g.remote.Load(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("error when fetching data from Github: %v", err)
	}

	//
	// prelude
	//

	// we try to sync users before applying the changes
	if syncusersbeforeapply {
		userplugin, found := engine.GetUserSyncPlugin(g.repoconfig.UserSync.Plugin)
		if !found {
			logrus.Warnf("user sync plugin %s not found", g.repoconfig.UserSync.Plugin)
		} else {
			accessToken, err := g.localGithubClient.GetAccessToken(ctx)
			if err != nil {
				return nil, err
			}
			change, err := g.local.SyncUsersAndTeams(g.repoconfig, userplugin, accessToken, dryrun, false, g.feedback)
			if err != nil {
				return nil, err
			}
			if change {
				g.remote.FlushCacheUsersTeamsOnly()
			}
		}
	}

	//
	// main
	//

	// we apply the changes to the github team repository
	unmanaged, err := g.applyCommitsToGithub(ctx, dryrun, teamreponame, branch)
	if err != nil {
		return unmanaged, fmt.Errorf("error when applying to github: %v", err)
	}

	//
	// post
	//

	// we update the codeowners file
	if !dryrun {
		accessToken, err := g.localGithubClient.GetAccessToken(ctx)
		if err != nil {
			return unmanaged, err
		}
		err = g.local.UpdateAndCommitCodeOwners(g.repoconfig, dryrun, accessToken, branch, GOLIAC_GIT_TAG, githubOrganization)
		if err != nil {
			return unmanaged, fmt.Errorf("error when updating and commiting: %v", err)
		}
	}

	return unmanaged, nil
}

func (g *GoliacImpl) applyCommitsToGithub(ctx context.Context, dryrun bool, teamreponame string, branch string) (*engine.UnmanagedResources, error) {

	// if the repo was just archived in a previous commit and we "resume it"
	// so we keep a track of all repos that we want to archive until the end of the process
	reposToArchive := make(map[string]*engine.GithubRepoComparable)
	var unmanaged *engine.UnmanagedResources

	ga := NewGithubBatchExecutor(g.remote, g.repoconfig.MaxChangesets)
	reconciliator := engine.NewGoliacReconciliatorImpl(ga, g.repoconfig)

	commit, err := g.local.GetHeadCommit()
	if err != nil {
		return unmanaged, fmt.Errorf("error when getting head commit: %v", err)
	}

	// the repo has already been cloned (to HEAD) and validated (see loadAndValidateGoliacOrganization)
	// we can now apply the changes to the github team repository
	unmanaged, err = reconciliator.Reconciliate(ctx, g.local, g.remote, teamreponame, dryrun, g.repoconfig.AdminTeam, reposToArchive)
	if err != nil {
		return unmanaged, fmt.Errorf("error when reconciliating: %v", err)
	}

	if !dryrun {
		accessToken, err := g.localGithubClient.GetAccessToken(ctx)
		if err != nil {
			return unmanaged, err
		}
		g.local.PushTag(GOLIAC_GIT_TAG, commit.Hash, accessToken)
	}

	accessToken, err := g.localGithubClient.GetAccessToken(ctx)
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
	return unmanaged, nil
}

func (g *GoliacImpl) UsersUpdate(ctx context.Context, fs billy.Filesystem, repositoryUrl, branch string, dryrun bool, force bool) (bool, error) {
	accessToken, err := g.localGithubClient.GetAccessToken(ctx)
	if err != nil {
		return false, err
	}

	err = g.local.Clone(fs, accessToken, repositoryUrl, branch)
	if err != nil {
		return false, err
	}
	defer g.local.Close(fs)

	repoconfig, err := g.local.LoadRepoConfig()
	if err != nil {
		return false, fmt.Errorf("unable to read goliac.yaml config file: %v", err)
	}

	userplugin, found := engine.GetUserSyncPlugin(repoconfig.UserSync.Plugin)
	if !found {
		return false, fmt.Errorf("user sync Plugin %s not found", repoconfig.UserSync.Plugin)
	}

	return g.local.SyncUsersAndTeams(repoconfig, userplugin, accessToken, dryrun, force, g.feedback)
}
