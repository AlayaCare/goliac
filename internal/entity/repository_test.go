package entity

import (
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/goliac-project/goliac/internal/config"
	"github.com/goliac-project/goliac/internal/observability"
	"github.com/goliac-project/goliac/internal/utils"
	"github.com/stretchr/testify/assert"
)

func fixtureCreateUserTeam(t *testing.T, fs billy.Filesystem) {
	fs.MkdirAll("users", 0755)
	err := utils.WriteFile(fs, "users/user1.yaml", []byte(`
apiVersion: v1
kind: User
name: user1
spec:
  githubID: github1
`), 0644)
	assert.Nil(t, err)

	err = utils.WriteFile(fs, "users/user2.yaml", []byte(`
apiVersion: v1
kind: User
name: user2
spec:
  githubID: github2
`), 0644)
	assert.Nil(t, err)

	fs.MkdirAll("teams", 0755)
	fs.MkdirAll("teams/team1", 0755)
	err = utils.WriteFile(fs, "teams/team1/team.yaml", []byte(`
apiVersion: v1
kind: Team
name: team1
spec:
  owners:
  - user1
  - user2
`), 0644)
	assert.Nil(t, err)
}

func TestRepository(t *testing.T) {

	// happy path
	t.Run("happy path", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
`), 0644)
		assert.Nil(t, err)

		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, users)

		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, teams)

		repos := ReadRepositories(fs, "archived", "teams", teams, map[string]*User{}, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, repos)
		assert.Equal(t, 1, len(repos))
	})
	t.Run("not happy path: wrong repo name", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo2
`), 0644)
		assert.Nil(t, err)
		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, users)

		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, teams)

		ReadRepositories(fs, "archived", "teams", teams, map[string]*User{}, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.True(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
	})

	t.Run("not happy path: wrong writer team name", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  writers:
  - wrongteam
`), 0644)
		assert.Nil(t, err)
		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, users)

		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, teams)

		ReadRepositories(fs, "archived", "teams", teams, map[string]*User{}, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.True(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
	})

	t.Run("not happy path: wrong writer team name", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  readers:
  - wrongteam
`), 0644)
		assert.Nil(t, err)
		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, users)

		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, teams)

		ReadRepositories(fs, "archived", "teams", teams, map[string]*User{}, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.True(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
	})

	t.Run("happy path: archived repo in the wrong place: it doesn't matter", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  archived: true
`), 0644)
		assert.Nil(t, err)
		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, users)

		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, teams)

		repos := ReadRepositories(fs, "archived", "teams", teams, map[string]*User{}, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, repos)
		assert.Equal(t, len(repos), 1)
	})

	t.Run("happy path: archived repo", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		err := utils.WriteFile(fs, "archived/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
`), 0644)
		assert.Nil(t, err)
		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, users)

		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, teams)

		repos := ReadRepositories(fs, "archived", "teams", teams, map[string]*User{}, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, repos)
		assert.Equal(t, len(repos), 1)
	})

	t.Run("not happy path: invalid bypass_pullrequest_user", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  branch_protections:
    - pattern: main
      bypass_pullrequest_users:
        - nonexistentuser
`), 0644)
		assert.Nil(t, err)
		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, users)

		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, teams)

		ReadRepositories(fs, "archived", "teams", teams, map[string]*User{}, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.True(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
	})

	t.Run("happy path: valid bypass_pullrequest_user", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  branch_protections:
    - pattern: main
      bypass_pullrequest_users:
        - user1
`), 0644)
		assert.Nil(t, err)
		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, users)

		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, teams)

		repos := ReadRepositories(fs, "archived", "teams", teams, map[string]*User{}, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, repos)
		assert.Equal(t, 1, len(repos))
	})

	t.Run("happy path: valid bypass_pullrequest_user (external user)", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		// create external user
		fs.MkdirAll("users/external", 0755)
		err := utils.WriteFile(fs, "users/external/externaluser1.yaml", []byte(`
apiVersion: v1
kind: User
name: externaluser1
spec:
  githubID: externalgithub1
`), 0644)
		assert.Nil(t, err)

		err = utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  branch_protections:
    - pattern: main
      bypass_pullrequest_users:
        - externaluser1
`), 0644)
		assert.Nil(t, err)
		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, users)

		externalUsers := ReadUserDirectory(fs, "users/external", logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, externalUsers)

		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, teams)

		repos := ReadRepositories(fs, "archived", "teams", teams, externalUsers, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.False(t, logsCollector.HasWarns())
		assert.NotNil(t, repos)
		assert.Equal(t, 1, len(repos))
	})
}

func TestReadAndAdjustRepositories(t *testing.T) {
	t.Run("happy path: no repositories, no problem", func(t *testing.T) {
		fs := memfs.New()
		users := make(map[string]*User)
		externalUsers := make(map[string]*User)

		changed, err := ReadAndAdjustRepositories(fs, "archived", "teams", users, externalUsers)
		assert.Nil(t, err)
		assert.Equal(t, 0, len(changed))
	})

	t.Run("happy path: no change needed", func(t *testing.T) {
		fs := memfs.New()
		users := make(map[string]*User)
		user1 := &User{}
		user1.Name = "user1"
		user1.Spec.GithubID = "github1"
		users["user1"] = user1

		fs.MkdirAll("teams/team1", 0755)
		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  branch_protections:
    - pattern: main
      bypass_pullrequest_users:
        - user1
`), 0644)
		assert.Nil(t, err)

		changed, err := ReadAndAdjustRepositories(fs, "archived", "teams", users, map[string]*User{})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(changed))
	})

	t.Run("happy path: remove deleted user from bypass_pullrequest_users", func(t *testing.T) {
		fs := memfs.New()
		users := make(map[string]*User)
		// user1 exists, user2 doesn't
		user1 := &User{}
		user1.Name = "user1"
		user1.Spec.GithubID = "github1"
		users["user1"] = user1

		fs.MkdirAll("teams/team1", 0755)
		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  branch_protections:
    - pattern: main
      bypass_pullrequest_users:
        - user1
        - user2
`), 0644)
		assert.Nil(t, err)

		changed, err := ReadAndAdjustRepositories(fs, "archived", "teams", users, map[string]*User{})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(changed))

		// Verify the file was updated
		content, err := utils.ReadFile(fs, "teams/team1/repo1.yaml")
		assert.Nil(t, err)
		assert.Contains(t, string(content), "user1")
		assert.NotContains(t, string(content), "user2")
	})

	t.Run("happy path: remove deleted external user from bypass_pullrequest_users", func(t *testing.T) {
		fs := memfs.New()
		users := make(map[string]*User)
		externalUsers := make(map[string]*User)
		// externaluser1 exists, externaluser2 doesn't
		externalUser1 := &User{}
		externalUser1.Name = "externaluser1"
		externalUser1.Spec.GithubID = "externalgithub1"
		externalUsers["externaluser1"] = externalUser1

		fs.MkdirAll("teams/team1", 0755)
		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  branch_protections:
    - pattern: main
      bypass_pullrequest_users:
        - externaluser1
        - externaluser2
`), 0644)
		assert.Nil(t, err)

		changed, err := ReadAndAdjustRepositories(fs, "archived", "teams", users, externalUsers)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(changed))

		// Verify the file was updated
		content, err := utils.ReadFile(fs, "teams/team1/repo1.yaml")
		assert.Nil(t, err)
		assert.Contains(t, string(content), "externaluser1")
		assert.NotContains(t, string(content), "externaluser2")
	})

	t.Run("happy path: archived repository with deleted user", func(t *testing.T) {
		fs := memfs.New()
		users := make(map[string]*User)
		user1 := &User{}
		user1.Name = "user1"
		user1.Spec.GithubID = "github1"
		users["user1"] = user1

		fs.MkdirAll("archived", 0755)
		err := utils.WriteFile(fs, "archived/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  branch_protections:
    - pattern: main
      bypass_pullrequest_users:
        - user1
        - user2
`), 0644)
		assert.Nil(t, err)

		changed, err := ReadAndAdjustRepositories(fs, "archived", "teams", users, map[string]*User{})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(changed))

		// Verify the file was updated
		content, err := utils.ReadFile(fs, "archived/repo1.yaml")
		assert.Nil(t, err)
		assert.Contains(t, string(content), "user1")
		assert.NotContains(t, string(content), "user2")
	})
}
