package internal

import (
	"context"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/Alayacare/goliac/internal/engine"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/utils"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/client"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/stretchr/testify/assert"
)

//
// fixture
//

// create a working simple teams repository
func repoFixture1(fs billy.Filesystem) {
	fs.MkdirAll("teams", 0755)
	fs.MkdirAll("archived", 0755)

	// create users
	fs.MkdirAll("users/external", 0755)
	fs.MkdirAll("users/org", 0755)
	fs.MkdirAll("users/protected", 0755)
	utils.WriteFile(fs, "users/org/user1.yaml", []byte(`apiVersion: v1
kind: User
name: user1
spec:
  githubID: github1
`), 0644)
	utils.WriteFile(fs, "users/org/user2.yaml", []byte(`apiVersion: v1
kind: User
name: user2
spec:
  githubID: github2
`), 0644)
	utils.WriteFile(fs, "users/org/user3.yaml", []byte(`apiVersion: v1
kind: User
name: user3
spec:
  githubID: github3
`), 0644)
	utils.WriteFile(fs, "users/org/user4.yaml", []byte(`apiVersion: v1
kind: User
name: user4
spec:
  githubID: github4
`), 0644)

	// create teams
	fs.MkdirAll("teams/team1", 0755)
	utils.WriteFile(fs, "teams/team1/team.yaml", []byte(`apiVersion: v1
kind: Team
name: team1
spec:
  owners:
    - user1
    - user2
`), 0644)
	fs.MkdirAll("teams/team2", 0755)
	utils.WriteFile(fs, "teams/team2/team.yaml", []byte(`apiVersion: v1
kind: Team
name: team2
spec:
  owners:
    - user3
    - user4
`), 0644)

	// create repositories
	utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`apiVersion: v1
kind: Repository
name: repo1
`), 0644)
	utils.WriteFile(fs, "teams/team2/repo2.yaml", []byte(`apiVersion: v1
kind: Repository
name: repo2
`), 0644)

	// create goliac.yaml
	utils.WriteFile(fs, "goliac.yaml", []byte(`admin_team: admin

rulesets:
  - pattern: .*
    ruleset: default

max_changesets: 50
archive_on_delete: true

destructive_operations:
  repositories: false
  teams: false
  users: false
  rulesets: false

usersync:
  plugin: noop
`), 0644)
	// rulesets
	fs.MkdirAll("rulesets", 0755)
	utils.WriteFile(fs, "rulesets/default.yaml", []byte(`apiVersion: v1
kind: Ruleset
name: default
spec:
  enforcement: active
  bypassapps:
    - appname: goliac-project-app
      mode: always
  on:
    include: 
      - "~DEFAULT_BRANCH"

  rules:
    - ruletype: pull_request
      parameters:
        requiredApprovingReviewCount: 1
`), 0644)

	// create .github/CODEOWNERS
	utils.WriteFile(fs, ".github/CODEOWNERS", []byte(`# DO NOT MODIFY THIS FILE MANUALLY
* @goliac-project/admin
/teams/team1/* @goliac-project/team1-goliac-owners @goliac-project/admin
/teams/team2/* @goliac-project/team2-goliac-owners @goliac-project/admin
`), 0644)
}

func createTeamRepo1(src billy.Filesystem) (*git.Repository, error) {
	masterStorer := filesystem.NewStorage(src, cache.NewObjectLRUDefault())

	// Create a fake bare repository
	repo, err := git.Init(masterStorer, src)
	if err != nil {
		return nil, err
	}

	//
	// Create a new file in the working directory
	//
	repoFixture1(src)

	// commit
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	_, err = worktree.Add(".")
	if err != nil {
		return nil, err
	}
	hash, err := worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Goliac",
			Email: "goliac@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, err
	}

	// tag as v0.1.0
	_, err = repo.CreateTag("v0.1.0", hash, nil)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

/*
 */
func helperCreateAndClone1(root billy.Filesystem, src billy.Filesystem, target billy.Filesystem) (*git.Repository, *git.Repository, error) {
	repo, err := createTeamRepo1(src)
	if err != nil {
		return nil, nil, err
	}

	//
	// trying to clone it
	//

	loader := server.NewFilesystemLoader(root)
	client.InstallProtocol("inmemory", server.NewClient(loader))

	dotGit, err := target.Chroot(".git")
	if err != nil {
		return nil, nil, err
	}
	storer := filesystem.NewStorage(dotGit, cache.NewObjectLRUDefault())

	clonedRepo, err := git.Clone(storer, target, &git.CloneOptions{
		URL:      "inmemory:///src",
		Progress: nil,
	})
	if err != nil {
		return nil, nil, err
	}
	return repo, clonedRepo, nil
}

//
// GitHubClientMock
//

type GitHubClientMock struct {
}

func NewGitHubClientMock() *GitHubClientMock {
	return &GitHubClientMock{}
}

func extractQueryName(query string) string {
	queryRegex := regexp.MustCompile(`query\s+(\w+)\(.*`)
	matches := queryRegex.FindStringSubmatch(query)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func (c *GitHubClientMock) QueryGraphQLAPI(ctx context.Context, query string, variables map[string]interface{}) ([]byte, error) {
	// extract query name
	queryName := extractQueryName(query)

	if queryName == "listSamlUsers" {
		return []byte(`{
			"data": {
				"organization": {
					"samlIdentityProvider": {
						"externalIdentities": {
							"edges": [
								{
									"node": {
										"guid": "guid1",
										"samlIdentity": {
											"nameId": "user1"
										},
										"user": {
											"login": "github1"
										}
									}
								},
								{
									"node": {
										"guid": "guid2",
										"samlIdentity": {
											"nameId": "name2"
										},
										"user": {
											"login": "github2"
										}
									}
								},
								{
									"node": {
										"guid": "guid3",
										"samlIdentity": {
											"nameId": "user3"
										},
										"user": {
											"login": "github3"
										}
									}
								},
								{
									"node": {
										"guid": "guid4",
										"samlIdentity": {
											"nameId": "user4"
										},
										"user": {
											"login": "github4"
										}
									}
								}
							],
							"pageInfo": {
								"hasNextPage": false,
								"endCursor": null
							},
							"totalCount": 4
						}
					}
				}
			}
		}`), nil
	} else {
		assert.Fail(nil, "unexpected query: "+queryName)
	}
	return nil, nil
}

func (c *GitHubClientMock) CallRestAPI(ctx context.Context, endpoint, method string, body map[string]interface{}) ([]byte, error) {
	return nil, nil
}
func (c *GitHubClientMock) GetAccessToken(context.Context) (string, error) {
	return "accesstoken", nil
}
func (c *GitHubClientMock) GetAppSlug() string {
	return "goliac-project-app"
}

//
// remote mock
//

type GoliacRemoteExecutorMock struct {
	nbChanges int
}

// GoliacRemoteExecutorMock
func NewGoliacRemoteExecutorMock() engine.GoliacRemoteExecutor {
	return &GoliacRemoteExecutorMock{
		nbChanges: 0,
	}
}

func (e *GoliacRemoteExecutorMock) Load(ctx context.Context, continueOnError bool) error {
	return nil
}
func (e *GoliacRemoteExecutorMock) FlushCache() {
}
func (e *GoliacRemoteExecutorMock) FlushCacheUsersTeamsOnly() {
}
func (e *GoliacRemoteExecutorMock) Users(ctx context.Context) map[string]string {
	return map[string]string{
		"github1": "member",
		"github2": "member",
		"github3": "member",
		"github4": "member",
	}
}
func (e *GoliacRemoteExecutorMock) TeamSlugByName(ctx context.Context) map[string]string {
	return map[string]string{
		"team1":               "team1",
		"team2":               "team2",
		"team1-goliac-owners": "team1-goliac-owners",
		"team2-goliac-owners": "team2-goliac-owners",
	}
}

func (e *GoliacRemoteExecutorMock) Teams(ctx context.Context) map[string]*engine.GithubTeam {
	return map[string]*engine.GithubTeam{
		"team1": &engine.GithubTeam{
			Slug:    "team1",
			Name:    "team1",
			Members: []string{"github1", "github2"},
		},
		"team2": &engine.GithubTeam{
			Slug:    "team2",
			Name:    "team2",
			Members: []string{"github3", "github4"},
		},
		"team1-goliac-owners": &engine.GithubTeam{
			Slug:    "team1-goliac-owners",
			Name:    "team1-goliac-owners",
			Members: []string{"github1", "github2"},
		},
		"team2-goliac-owners": &engine.GithubTeam{
			Slug:    "team2-goliac-owners",
			Name:    "team2-goliac-owners",
			Members: []string{"github3", "github4"},
		},
	}
}
func (e *GoliacRemoteExecutorMock) Repositories(ctx context.Context) map[string]*engine.GithubRepository {
	return map[string]*engine.GithubRepository{
		"repo1": &engine.GithubRepository{
			Name:  "repo1",
			Id:    1,
			RefId: "MDEwOlJlcG9zaXRvcnkaMTMxNjExOQ==",
			BoolProperties: map[string]bool{
				"archived":               false,
				"private":                true,
				"allow_auto_merge":       false,
				"delete_branch_on_merge": false,
				"allow_update_branch":    false,
			},
			ExternalUsers: map[string]string{},
		},
		"repo2": &engine.GithubRepository{
			Name:  "repo2",
			Id:    2,
			RefId: "MDEwOlJlcG9zaXRvcnkaNTcwNDA4Ng==",
			BoolProperties: map[string]bool{
				"archived":               false,
				"private":                true,
				"allow_auto_merge":       false,
				"delete_branch_on_merge": false,
				"allow_update_branch":    false,
			},
			ExternalUsers: map[string]string{},
		},
	}
}
func (e *GoliacRemoteExecutorMock) TeamRepositories(ctx context.Context) map[string]map[string]*engine.GithubTeamRepo {
	return map[string]map[string]*engine.GithubTeamRepo{
		"team1": {
			"repo1": &engine.GithubTeamRepo{
				Name:       "repo1",
				Permission: "WRITE",
			},
		},
		"team2": {
			"repo2": &engine.GithubTeamRepo{
				Name:       "repo2",
				Permission: "WRITE",
			},
		},
	}
}
func (e *GoliacRemoteExecutorMock) RuleSets(ctx context.Context) map[string]*engine.GithubRuleSet {
	return map[string]*engine.GithubRuleSet{
		"default": {
			Name:        "default",
			Id:          0,
			Enforcement: "active",
			BypassApps: map[string]string{
				"goliac-project-app": "always",
			},
			OnInclude: []string{"~DEFAULT_BRANCH"},
			Rules: map[string]entity.RuleSetParameters{
				"pull_request": {
					RequiredApprovingReviewCount: 1,
				},
			},
			Repositories: []string{"repo1", "repo2"},
		},
	}
}
func (e *GoliacRemoteExecutorMock) AppIds(ctx context.Context) map[string]int {
	return map[string]int{
		"goliac-project-app": 1,
	}
}
func (e *GoliacRemoteExecutorMock) IsEnterprise() bool {
	return true
}

func (e *GoliacRemoteExecutorMock) AddUserToOrg(ctx context.Context, dryrun bool, ghuserid string) {
	e.nbChanges++
}
func (e *GoliacRemoteExecutorMock) RemoveUserFromOrg(ctx context.Context, dryrun bool, ghuserid string) {
	e.nbChanges++
}

func (e *GoliacRemoteExecutorMock) CreateTeam(ctx context.Context, dryrun bool, teamname string, description string, parentTeam *int, members []string) {
	e.nbChanges++
}
func (e *GoliacRemoteExecutorMock) UpdateTeamAddMember(ctx context.Context, dryrun bool, teamslug string, username string, role string) {
	e.nbChanges++
}
func (e *GoliacRemoteExecutorMock) UpdateTeamRemoveMember(ctx context.Context, dryrun bool, teamslug string, username string) {
	e.nbChanges++
}
func (e *GoliacRemoteExecutorMock) UpdateTeamSetParent(ctx context.Context, dryrun bool, teamslug string, parentTeam *int) {
	e.nbChanges++
}
func (e *GoliacRemoteExecutorMock) DeleteTeam(ctx context.Context, dryrun bool, teamslug string) {
	e.nbChanges++
}

func (e *GoliacRemoteExecutorMock) CreateRepository(ctx context.Context, dryrun bool, reponame string, descrition string, writers []string, readers []string, boolProperties map[string]bool) {
	e.nbChanges++
}
func (e *GoliacRemoteExecutorMock) UpdateRepositoryUpdateBoolProperty(ctx context.Context, dryrun bool, reponame string, propertyName string, propertyValue bool) {
	e.nbChanges++
}
func (e *GoliacRemoteExecutorMock) UpdateRepositoryAddTeamAccess(ctx context.Context, dryrun bool, reponame string, teamslug string, permission string) {
	e.nbChanges++
}
func (e *GoliacRemoteExecutorMock) UpdateRepositoryUpdateTeamAccess(ctx context.Context, dryrun bool, reponame string, teamslug string, permission string) {
	e.nbChanges++
}
func (e *GoliacRemoteExecutorMock) UpdateRepositoryRemoveTeamAccess(ctx context.Context, dryrun bool, reponame string, teamslug string) {
	e.nbChanges++
}
func (e *GoliacRemoteExecutorMock) AddRuleset(ctx context.Context, dryrun bool, ruleset *engine.GithubRuleSet) {
	e.nbChanges++
}
func (e *GoliacRemoteExecutorMock) UpdateRuleset(ctx context.Context, dryrun bool, ruleset *engine.GithubRuleSet) {
	e.nbChanges++
}
func (e *GoliacRemoteExecutorMock) DeleteRuleset(ctx context.Context, dryrun bool, rulesetid int) {
	e.nbChanges++
}
func (e *GoliacRemoteExecutorMock) UpdateRepositorySetExternalUser(ctx context.Context, dryrun bool, reponame string, githubid string, permission string) {
	e.nbChanges++
}
func (e *GoliacRemoteExecutorMock) UpdateRepositoryRemoveExternalUser(ctx context.Context, dryrun bool, reponame string, githubid string) {
	e.nbChanges++
}
func (e *GoliacRemoteExecutorMock) DeleteRepository(ctx context.Context, dryrun bool, reponame string) {
	e.nbChanges++
}

func (e *GoliacRemoteExecutorMock) Begin(dryrun bool) {
}
func (e *GoliacRemoteExecutorMock) Rollback(dryrun bool, err error) {
}
func (e *GoliacRemoteExecutorMock) Commit(ctx context.Context, dryrun bool) error {
	return nil
}

//
// tests
//

func TestGoliacApply(t *testing.T) {

	t.Run("happy path: get status", func(t *testing.T) {

		fs := memfs.New()
		fs.MkdirAll("src", 0755)        // create a fake bare repository
		fs.MkdirAll("teams", 0755)      // create a fake cloned repository
		fs.MkdirAll(os.TempDir(), 0755) // need a tmp folder
		srcsFs, _ := fs.Chroot("src")
		clonedFs, _ := fs.Chroot("teams")
		_, clonedRepo, err := helperCreateAndClone1(fs, srcsFs, clonedFs)
		assert.Nil(t, err)

		local := engine.NewGoliacLocalImplWithRepo(clonedRepo)
		errs, warns := local.LoadAndValidateLocal(clonedFs)
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)

		repoconfig, err := local.LoadRepoConfig()
		assert.Nil(t, err)

		remote := NewGoliacRemoteExecutorMock().(*GoliacRemoteExecutorMock)

		goliac := GoliacImpl{
			local:        local,
			remote:       remote,
			githubClient: NewGitHubClientMock(),
			repoconfig:   repoconfig,
		}
		err, errs, warns, unmanaged := goliac.Apply(context.Background(), fs, false, "inmemory:///teams", "master", false)
		assert.Nil(t, err)
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, unmanaged)
		assert.Equal(t, 0, remote.nbChanges)
	})
}
