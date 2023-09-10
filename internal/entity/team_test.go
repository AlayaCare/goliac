package entity

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func fixtureCreateUser(t *testing.T, fs afero.Fs) {
	fs.Mkdir("users", 0755)
	err := afero.WriteFile(fs, "users/user1.yaml", []byte(`
apiVersion: v1
kind: User
name: user1
spec:
  githubID: github1
`), 0644)
	assert.Nil(t, err)

	err = afero.WriteFile(fs, "users/user2.yaml", []byte(`
apiVersion: v1
kind: User
name: user2
spec:
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
name: team1
spec:
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
name: team1
spec:
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
name: team1
spec:
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

func TestAdjustTeam(t *testing.T) {
	t.Run("happy path: no change ", func(t *testing.T) {
		team := Team{}
		team.Spec.Owners = []string{"owner2", "owner3"}
		team.Spec.Members = []string{"member2", "member3"}
		users := make(map[string]*User)
		for _, username := range []string{"owner1", "owner2", "owner3", "owner3", "member1", "member2", "member3", "member4"} {
			u := User{}
			u.Name = username
			u.Spec.GithubID = username
			users[username] = &u
		}
		fs := afero.NewMemMapFs()
		changed, err := team.Update(fs, "/teams/ateam/team.yaml", users)

		assert.Nil(t, err)
		assert.False(t, changed)
	})
	t.Run("not happy path: missing member ", func(t *testing.T) {
		team := Team{}
		team.Spec.Owners = []string{"owner2", "owner3"}
		team.Spec.Members = []string{"member2", "member3"}
		users := make(map[string]*User)
		for _, username := range []string{"owner1", "owner2", "owner3", "owner3", "member1", "member2"} {
			u := User{}
			u.Name = username
			u.Spec.GithubID = username
			users[username] = &u
		}
		fs := afero.NewMemMapFs()
		changed, err := team.Update(fs, "/teams/ateam/team.yaml", users)

		assert.Nil(t, err)
		assert.True(t, changed)

		f, err := afero.ReadFile(fs, "/teams/ateam/team.yaml")
		assert.Nil(t, err)

		var checkTeam Team
		yaml.Unmarshal(f, &checkTeam)

		assert.Equal(t, 1, len(checkTeam.Spec.Members))
		assert.Equal(t, 2, len(checkTeam.Spec.Owners))
		assert.Equal(t, "member2", checkTeam.Spec.Members[0])
	})
}

func TestReadAndAdjustTeam(t *testing.T) {
	t.Run("happy path: no team, no problem", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		users := make(map[string]*User)

		changed, err := ReadAndAdjustTeamDirectory(fs, "/teams", users)
		assert.Nil(t, err)
		assert.Equal(t, 0, len(changed))
	})

	t.Run("happy path: no change ", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		users := make(map[string]*User)
		for _, username := range []string{"owner1", "owner2", "owner3", "owner3", "member1", "member2", "member3", "member4"} {
			u := User{}
			u.Name = username
			u.Spec.GithubID = username
			users[username] = &u
		}

		err := afero.WriteFile(fs, "/teams/ateam/team.yaml", []byte(`
apiVersion: v1
kind: Team
name: ateam
spec:
  owners:
    - owner2
    - owner3
  members:
    - member2
    - member3
`), 0644)
		assert.Nil(t, err)
		changed, err := ReadAndAdjustTeamDirectory(fs, "/teams", users)
		assert.Nil(t, err)
		assert.Equal(t, 0, len(changed))
	})
	t.Run("not happy path: missing member ", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		users := make(map[string]*User)
		for _, username := range []string{"owner1", "owner2", "owner3", "owner3", "member1", "member2"} {
			u := User{}
			u.Name = username
			u.Spec.GithubID = username
			users[username] = &u
		}

		err := afero.WriteFile(fs, "/teams/ateam/team.yaml", []byte(`
apiVersion: v1
kind: Team
name: ateam
spec:
  owners:
    - owner2
    - owner3
  members:
    - member2
    - member3
`), 0644)
		assert.Nil(t, err)
		changed, err := ReadAndAdjustTeamDirectory(fs, "/teams", users)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(changed))
	})
}
