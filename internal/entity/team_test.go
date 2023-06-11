package entity

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func fixtureCreateUser(t *testing.T, fs afero.Fs) {
	fs.Mkdir("users", 0755)
	err := afero.WriteFile(fs, "users/user1.yaml", []byte(`
apiVersion: v1
kind: User
metadata:
  name: user1
data:
  githubID: github1
`), 0644)
	assert.Nil(t, err)

	err = afero.WriteFile(fs, "users/user2.yaml", []byte(`
apiVersion: v1
kind: User
metadata:
  name: user2
data:
  githubID: github2
`), 0644)
	assert.Nil(t, err)

	fs.Mkdir("teams", 0755)
}

func TestTeam(t *testing.T) {

	// happy path
	t.Run("happy path", func(t *testing.T) {
		// create a new user
		fs := afero.NewMemMapFs()
		fixtureCreateUser(t, fs)
		fs.Mkdir("teams/team1", 0755)

		err := afero.WriteFile(fs, "teams/team1/team.yaml", []byte(`
apiVersion: v1
kind: Team
metadata:
  name: team1
data:
  owners:
  - user1
  - user2
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
	})

	t.Run("happy path without enough owners", func(t *testing.T) {
		// create a new user
		fs := afero.NewMemMapFs()
		fixtureCreateUser(t, fs)
		fs.Mkdir("teams/team1", 0755)

		err := afero.WriteFile(fs, "teams/team1/team.yaml", []byte(`
apiVersion: v1
kind: Team
metadata:
  name: team1
data:
  owners:
  - user1
`), 0644)
		assert.Nil(t, err)
		users, errs, warns := ReadUserDirectory(fs, "users")
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, users)

		teams, errs, warns := ReadTeamDirectory(fs, "teams", users)
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 1)
		assert.NotNil(t, teams)
	})

	t.Run("not happy path: not team directory", func(t *testing.T) {
		// create a new user
		fs := afero.NewMemMapFs()

		_, errs, warns := ReadTeamDirectory(fs, "teams", map[string]*User{})
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
	})

	t.Run("not happy path: wrong username", func(t *testing.T) {
		// create a new user
		fs := afero.NewMemMapFs()
		fixtureCreateUser(t, fs)
		fs.Mkdir("teams/team1", 0755)

		err := afero.WriteFile(fs, "teams/team1/team.yaml", []byte(`
apiVersion: v1
kind: Team
metadata:
  name: team1
data:
  owners:
  - wronguser1
  - wronguser2
`), 0644)
		assert.Nil(t, err)
		users, errs, warns := ReadUserDirectory(fs, "users")
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, users)

		teams, errs, warns := ReadTeamDirectory(fs, "teams", users)
		assert.Equal(t, len(errs), 1)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, teams)
	})

	t.Run("not happy path: missing specs", func(t *testing.T) {
		// create a new user
		fs := afero.NewMemMapFs()
		fixtureCreateUser(t, fs)
		fs.Mkdir("teams/team1", 0755)

		err := afero.WriteFile(fs, "teams/team1/team.yaml", []byte(`
apiVersion: v2
kind: Foo
metadata:
  name: team2
`), 0644)
		assert.Nil(t, err)
		users, errs, warns := ReadUserDirectory(fs, "users")
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, users)

		teams, errs, warns := ReadTeamDirectory(fs, "teams", users)
		assert.Equal(t, len(errs), 1)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, teams)
	})
}
