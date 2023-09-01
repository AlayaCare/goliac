package internal

import (
	"fmt"
	"testing"

	"github.com/Alayacare/goliac/internal/engine"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type ScaffoldGoliacRemoteMock struct {
	users      map[string]string
	teams      map[string]*engine.GithubTeam
	repos      map[string]*engine.GithubRepository
	teamsRepos map[string]map[string]*engine.GithubTeamRepo
}

func (s *ScaffoldGoliacRemoteMock) Load() error {
	return nil
}
func (s *ScaffoldGoliacRemoteMock) FlushCache() {
}
func (s *ScaffoldGoliacRemoteMock) Users() map[string]string {
	return s.users
}
func (s *ScaffoldGoliacRemoteMock) TeamSlugByName() map[string]string {
	return nil
}
func (s *ScaffoldGoliacRemoteMock) Teams() map[string]*engine.GithubTeam {
	return s.teams
}
func (s *ScaffoldGoliacRemoteMock) Repositories() map[string]*engine.GithubRepository {
	return s.repos
}
func (s *ScaffoldGoliacRemoteMock) TeamRepositories() map[string]map[string]*engine.GithubTeamRepo {
	return s.teamsRepos
}
func (s *ScaffoldGoliacRemoteMock) RuleSets() map[string]*engine.GithubRuleSet {
	return nil
}
func (s *ScaffoldGoliacRemoteMock) AppIds() map[string]int {
	return nil
}

func NewScaffoldGoliacRemoteMock() engine.GoliacRemote {
	users := make(map[string]string)
	teams := make(map[string]*engine.GithubTeam)
	repos := make(map[string]*engine.GithubRepository)
	teamsRepos := make(map[string]map[string]*engine.GithubTeamRepo)

	users["githubid1"] = "githubid1"
	users["githubid2"] = "githubid2"
	users["githubid3"] = "githubid3"
	users["githubid4"] = "githubid4"

	admin := engine.GithubTeam{
		Name:    "admin",
		Slug:    "admin",
		Members: []string{"githubid1", "githubid2"},
	}
	teams["admin"] = &admin

	regular := engine.GithubTeam{
		Name:    "regular",
		Slug:    "regular",
		Members: []string{"githubid2", "githubid3"},
	}
	teams["regular"] = &regular

	repo1 := engine.GithubRepository{
		Name: "repo1",
	}
	repos["repo1"] = &repo1

	repo2 := engine.GithubRepository{
		Name: "repo2",
	}
	repos["repo2"] = &repo2

	teamRepoRegular := make(map[string]*engine.GithubTeamRepo)
	teamRepoRegular["repo1"] = &engine.GithubTeamRepo{
		Name:       "repo1",
		Permission: "WRITE",
	}
	teamRepoRegular["repo2"] = &engine.GithubTeamRepo{
		Name:       "repo2",
		Permission: "READ",
	}
	teamsRepos["regular"] = teamRepoRegular

	teamRepoAdmin := make(map[string]*engine.GithubTeamRepo)
	teamRepoAdmin["repo2"] = &engine.GithubTeamRepo{
		Name:       "repo2",
		Permission: "WRITE",
	}
	teamsRepos["admin"] = teamRepoAdmin

	mock := ScaffoldGoliacRemoteMock{
		users:      users,
		teams:      teams,
		repos:      repos,
		teamsRepos: teamsRepos,
	}

	return &mock
}

func LoadGithubSamlUsersMock() (map[string]*entity.User, error) {
	users := make(map[string]*entity.User)
	user1 := &entity.User{}
	user1.ApiVersion = "v1"
	user1.Kind = "User"
	user1.Metadata.Name = "user1@company.com"
	user1.Data.GithubID = "githubid1"
	users["user1@company.com"] = user1

	user2 := &entity.User{}
	user2.ApiVersion = "v1"
	user2.Kind = "User"
	user2.Metadata.Name = "user2@company.com"
	user2.Data.GithubID = "githubid2"
	users["user2@company.com"] = user2

	user3 := &entity.User{}
	user3.ApiVersion = "v1"
	user3.Kind = "User"
	user3.Metadata.Name = "user3@company.com"
	user3.Data.GithubID = "githubid3"
	users["user3@company.com"] = user3

	return users, nil
}

func NoLoadGithubSamlUsersMock() (map[string]*entity.User, error) {
	return nil, fmt.Errorf("not able to fetch SAML data")
}

func TestScaffoldUnit(t *testing.T) {

	// happy path
	t.Run("happy path: test users no SAML", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// MockGithubClient doesn't support concurrent access

		scaffold := &Scaffold{
			remote:                     NewScaffoldGoliacRemoteMock(),
			loadUsersFromGithubOrgSaml: NoLoadGithubSamlUsersMock,
		}

		users, err := scaffold.generateUsers(fs, "/users")
		assert.Nil(t, err)
		assert.Equal(t, 4, len(users))

		found, err := afero.Exists(fs, "/users/org/githubid1.yaml")
		assert.Nil(t, err)
		assert.Equal(t, true, found)
	})

	t.Run("happy path: test users SAML", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// MockGithubClient doesn't support concurrent access

		scaffold := &Scaffold{
			remote:                     NewScaffoldGoliacRemoteMock(),
			loadUsersFromGithubOrgSaml: LoadGithubSamlUsersMock,
		}

		users, err := scaffold.generateUsers(fs, "/users")
		assert.Nil(t, err)
		assert.Equal(t, 3, len(users))

		found, err := afero.Exists(fs, "/users/org/user1@company.com.yaml")
		assert.Nil(t, err)
		assert.Equal(t, true, found)
	})

	t.Run("happy path: test rulesets", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// MockGithubClient doesn't support concurrent access

		scaffold := &Scaffold{
			remote:                     NewScaffoldGoliacRemoteMock(),
			loadUsersFromGithubOrgSaml: LoadGithubSamlUsersMock,
		}

		err := scaffold.generateRuleset(fs, "/rulesets")
		assert.Nil(t, err)

		found, err := afero.Exists(fs, "/rulesets/default.yaml")
		assert.Nil(t, err)
		assert.Equal(t, true, found)
	})

	t.Run("happy path: test goliac.conf", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// MockGithubClient doesn't support concurrent access

		scaffold := &Scaffold{
			remote:                     NewScaffoldGoliacRemoteMock(),
			loadUsersFromGithubOrgSaml: LoadGithubSamlUsersMock,
		}

		err := scaffold.generateGoliacConf(fs, "/", "admin")
		assert.Nil(t, err)

		found, err := afero.Exists(fs, "/goliac.yaml")
		assert.Nil(t, err)
		assert.Equal(t, true, found)
	})

	t.Run("happy path: test github action", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// MockGithubClient doesn't support concurrent access

		scaffold := &Scaffold{
			remote:                     NewScaffoldGoliacRemoteMock(),
			loadUsersFromGithubOrgSaml: LoadGithubSamlUsersMock,
		}

		err := scaffold.generateGithubAction(fs, "/")
		assert.Nil(t, err)

		found, err := afero.Exists(fs, "/.github/workflows/pr.yaml")
		assert.Nil(t, err)
		assert.Equal(t, true, found)
	})
}
func TestScaffoldFull(t *testing.T) {

	t.Run("happy path: test teams and repos without SAML", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// MockGithubClient doesn't support concurrent access

		scaffold := &Scaffold{
			remote:                     NewScaffoldGoliacRemoteMock(),
			loadUsersFromGithubOrgSaml: NoLoadGithubSamlUsersMock,
		}

		users, err := scaffold.generateUsers(fs, "/users")
		assert.Nil(t, err)
		assert.Equal(t, 4, len(users))

		err, foundAdmin := scaffold.generateTeams(fs, "/teams", users, "admin")
		assert.Nil(t, err)
		assert.Equal(t, true, foundAdmin)

		found, err := afero.Exists(fs, "/teams/admin/team.yaml")
		assert.Nil(t, err)
		assert.Equal(t, true, found)

		found, err = afero.Exists(fs, "/teams/regular/repo1.yaml")
		assert.Nil(t, err)
		assert.Equal(t, true, found)
	})

	t.Run("happy path: test teams and repos with SAML", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// MockGithubClient doesn't support concurrent access

		scaffold := &Scaffold{
			remote:                     NewScaffoldGoliacRemoteMock(),
			loadUsersFromGithubOrgSaml: LoadGithubSamlUsersMock,
		}

		users, err := scaffold.generateUsers(fs, "/users")
		assert.Nil(t, err)
		assert.Equal(t, 3, len(users))

		err, foundAdmin := scaffold.generateTeams(fs, "/teams", users, "admin")
		assert.Nil(t, err)
		assert.Equal(t, true, foundAdmin)

		found, err := afero.Exists(fs, "/teams/admin/team.yaml")
		assert.Nil(t, err)
		assert.Equal(t, true, found)

		found, err = afero.Exists(fs, "/teams/regular/repo1.yaml")
		assert.Nil(t, err)
		assert.Equal(t, true, found)

		regularTeam, err := afero.ReadFile(fs, "/teams/regular/team.yaml")
		assert.Nil(t, err)

		var at entity.Team
		err = yaml.Unmarshal(regularTeam, &at)
		assert.Nil(t, err)
		assert.Equal(t, "regular", at.Metadata.Name)
		assert.Equal(t, 2, len(at.Data.Owners)) // githubid2,githubid3

		repo1, err := afero.ReadFile(fs, "/teams/regular/repo1.yaml")
		assert.Nil(t, err)

		var r1 entity.Repository
		err = yaml.Unmarshal(repo1, &r1)
		assert.Nil(t, err)
		assert.Equal(t, "repo1", r1.Metadata.Name)
		assert.Equal(t, 0, len(r1.Data.Writers)) // regular -> not counted

		repo2, err := afero.ReadFile(fs, "/teams/admin/repo2.yaml")
		assert.Nil(t, err)

		var r2 entity.Repository
		err = yaml.Unmarshal(repo2, &r2)
		assert.Nil(t, err)
		assert.Equal(t, "repo2", r2.Metadata.Name)
		assert.Equal(t, 1, len(r2.Data.Readers)) // regular
	})
}
