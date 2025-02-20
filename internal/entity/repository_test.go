package entity

import (
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
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
		users, errs, warns := ReadUserDirectory(fs, "users")
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, users)

		teams, errs, warns := ReadTeamDirectory(fs, "teams", users)
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, teams)

		repos, errs, warns := ReadRepositories(fs, "archived", "teams", teams, map[string]*User{})
		assert.Equal(t, 0, len(errs))
		assert.Equal(t, 0, len(warns))
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
		users, errs, warns := ReadUserDirectory(fs, "users")
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, users)

		teams, errs, warns := ReadTeamDirectory(fs, "teams", users)
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, teams)

		_, errs, warns = ReadRepositories(fs, "archived", "teams", teams, map[string]*User{})
		assert.Equal(t, len(errs), 1)
		assert.Equal(t, len(warns), 0)
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
		users, errs, warns := ReadUserDirectory(fs, "users")
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, users)

		teams, errs, warns := ReadTeamDirectory(fs, "teams", users)
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, teams)

		_, errs, warns = ReadRepositories(fs, "archived", "teams", teams, map[string]*User{})
		assert.Equal(t, len(errs), 1)
		assert.Equal(t, len(warns), 0)
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
		users, errs, warns := ReadUserDirectory(fs, "users")
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, users)

		teams, errs, warns := ReadTeamDirectory(fs, "teams", users)
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, teams)

		_, errs, warns = ReadRepositories(fs, "archived", "teams", teams, map[string]*User{})
		assert.Equal(t, len(errs), 1)
		assert.Equal(t, len(warns), 0)
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
		users, errs, warns := ReadUserDirectory(fs, "users")
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, users)

		teams, errs, warns := ReadTeamDirectory(fs, "teams", users)
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, teams)

		repos, errs, warns := ReadRepositories(fs, "archived", "teams", teams, map[string]*User{})
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
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
		users, errs, warns := ReadUserDirectory(fs, "users")
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, users)

		teams, errs, warns := ReadTeamDirectory(fs, "teams", users)
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, teams)

		repos, errs, warns := ReadRepositories(fs, "archived", "teams", teams, map[string]*User{})
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, repos)
		assert.Equal(t, len(repos), 1)
	})
}
