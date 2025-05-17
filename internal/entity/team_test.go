package entity

import (
	"fmt"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/goliac-project/goliac/internal/observability"
	"github.com/goliac-project/goliac/internal/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func fixtureCreateUser(t *testing.T, fs billy.Filesystem) {
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
}

func TestTeam(t *testing.T) {

	// happy path
	t.Run("happy path", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUser(t, fs)
		fs.MkdirAll("teams/team1", 0755)

		err := utils.WriteFile(fs, "teams/team1/team.yaml", []byte(`
apiVersion: v1
kind: Team
name: team1
spec:
  owners:
  - user1
  - user2
`), 0644)
		assert.Nil(t, err)
		errorCollector := observability.NewErrorCollection()
		users := ReadUserDirectory(fs, "users", errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, users)

		teams := ReadTeamDirectory(fs, "teams", users, errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, teams)
	})

	t.Run("happy path without enough owners", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUser(t, fs)
		fs.MkdirAll("teams/team1", 0755)

		err := utils.WriteFile(fs, "teams/team1/team.yaml", []byte(`
apiVersion: v1
kind: Team
name: team1
spec:
  owners:
  - user1
`), 0644)
		assert.Nil(t, err)
		errorCollector := observability.NewErrorCollection()
		users := ReadUserDirectory(fs, "users", errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, users)

		teams := ReadTeamDirectory(fs, "teams", users, errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, 1, len(errorCollector.Warns))
		assert.NotNil(t, teams)
	})

	t.Run("not happy path: not team directory", func(t *testing.T) {
		// create a new user
		fs := memfs.New()

		errorCollector := observability.NewErrorCollection()
		ReadTeamDirectory(fs, "teams", map[string]*User{}, errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
	})

	t.Run("not happy path: wrong username", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUser(t, fs)
		fs.MkdirAll("teams/team1", 0755)

		err := utils.WriteFile(fs, "teams/team1/team.yaml", []byte(`
apiVersion: v1
kind: Team
name: team1
spec:
  owners:
  - wronguser1
  - wronguser2
`), 0644)
		assert.Nil(t, err)
		errorCollector := observability.NewErrorCollection()
		users := ReadUserDirectory(fs, "users", errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, users)

		teams := ReadTeamDirectory(fs, "teams", users, errorCollector)
		for _, err := range errorCollector.Errors {
			fmt.Println(err)
		}
		assert.Equal(t, 2, len(errorCollector.Errors))
		assert.Equal(t, 0, len(errorCollector.Warns))
		assert.NotNil(t, teams)
	})

	t.Run("not happy path: missing specs", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUser(t, fs)
		fs.MkdirAll("teams/team1", 0755)

		err := utils.WriteFile(fs, "teams/team1/team.yaml", []byte(`
apiVersion: v2
kind: Foo
name: team2
`), 0644)
		assert.Nil(t, err)
		errorCollector := observability.NewErrorCollection()
		users := ReadUserDirectory(fs, "users", errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, users)

		teams := ReadTeamDirectory(fs, "teams", users, errorCollector)
		for _, err := range errorCollector.Errors {
			fmt.Println(err)
		}
		assert.Equal(t, 3, len(errorCollector.Errors))
		assert.Equal(t, 1, len(errorCollector.Warns))
		assert.NotNil(t, teams)
	})

	t.Run("not happy path: not able to create a sub team without a defined parent", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUser(t, fs)
		fs.MkdirAll("teams/team1", 0755)

		err := utils.WriteFile(fs, "teams/foo/bar/team.yaml", []byte(`
apiVersion: v1
kind: Team
name: bar
spec:
  owners:
  - user1
  - user2
`), 0644)
		assert.Nil(t, err)
		assert.Nil(t, err)
		errorCollector := observability.NewErrorCollection()
		users := ReadUserDirectory(fs, "users", errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, users)

		ReadTeamDirectory(fs, "teams", users, errorCollector)
		assert.Equal(t, true, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
	})

	t.Run("not happy path: not able to create 2 times the same team", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUser(t, fs)
		fs.MkdirAll("teams/team1", 0755)

		err := utils.WriteFile(fs, "teams/foo/team.yaml", []byte(`
apiVersion: v1
kind: Team
name: foo
spec:
  owners:
  - user1
  - user2
`), 0644)
		assert.Nil(t, err)
		err = utils.WriteFile(fs, "teams/foo/bar/team.yaml", []byte(`
apiVersion: v1
kind: Team
name: bar
spec:
  owners:
  - user1
  - user2
`), 0644)
		assert.Nil(t, err)
		err = utils.WriteFile(fs, "teams/foo2/team.yaml", []byte(`
apiVersion: v1
kind: Team
name: foo2
spec:
  owners:
  - user1
  - user2
`), 0644)
		assert.Nil(t, err)
		err = utils.WriteFile(fs, "teams/foo2/bar/team.yaml", []byte(`
apiVersion: v1
kind: Team
name: bar
spec:
  owners:
  - user1
  - user2
`), 0644)
		assert.Nil(t, err)
		errorCollector := observability.NewErrorCollection()
		users := ReadUserDirectory(fs, "users", errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, users)

		ReadTeamDirectory(fs, "teams", users, errorCollector)
		assert.Equal(t, true, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
	})

	t.Run("happy path: parent and child team", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateUser(t, fs)
		fs.MkdirAll("teams/team1", 0755)

		err := utils.WriteFile(fs, "teams/team1/team.yaml", []byte(`
apiVersion: v1
kind: Team
name: team1
spec:
  owners:
  - user1
  - user2
`), 0644)
		assert.Nil(t, err)
		err = utils.WriteFile(fs, "teams/team1/subteam/team.yaml", []byte(`
apiVersion: v1
kind: Team
name: subteam
spec:
  owners:
  - user1
  - user2
`), 0644)
		assert.Nil(t, err)
		errorCollector := observability.NewErrorCollection()
		users := ReadUserDirectory(fs, "users", errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, users)

		teams := ReadTeamDirectory(fs, "teams", users, errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, teams)

		assert.Equal(t, 2, len(teams))
		team1 := teams["team1"]
		assert.NotNil(t, team1)
		subteam := teams["subteam"]
		assert.NotNil(t, subteam)
		assert.Equal(t, "team1", *subteam.ParentTeam)
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
		fs := memfs.New()
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
		fs := memfs.New()
		changed, err := team.Update(fs, "/teams/ateam/team.yaml", users)

		assert.Nil(t, err)
		assert.True(t, changed)

		f, err := utils.ReadFile(fs, "/teams/ateam/team.yaml")
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
		fs := memfs.New()
		users := make(map[string]*User)

		changed, err := ReadAndAdjustTeamDirectory(fs, "/teams", users)
		assert.Nil(t, err)
		assert.Equal(t, 0, len(changed))
	})

	t.Run("happy path: no change ", func(t *testing.T) {
		fs := memfs.New()
		users := make(map[string]*User)
		for _, username := range []string{"owner1", "owner2", "owner3", "owner3", "member1", "member2", "member3", "member4"} {
			u := User{}
			u.Name = username
			u.Spec.GithubID = username
			users[username] = &u
		}

		err := utils.WriteFile(fs, "/teams/ateam/team.yaml", []byte(`
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
	t.Run("not happy path: missing member", func(t *testing.T) {
		fs := memfs.New()
		users := make(map[string]*User)
		for _, username := range []string{"owner1", "owner2", "owner3", "owner3", "member1", "member2"} {
			u := User{}
			u.Name = username
			u.Spec.GithubID = username
			users[username] = &u
		}

		err := utils.WriteFile(fs, "/teams/ateam/team.yaml", []byte(`
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

	t.Run("happy path: do not output parentTeam", func(t *testing.T) {
		fs := memfs.New()

		parentTeam := "aparent"
		team := Team{
			ParentTeam: &parentTeam,
		}
		team.Spec.Owners = []string{"owner2", "owner3"}
		team.Name = "ateam"

		users := make(map[string]*User)
		for _, username := range []string{"owner1", "owner2"} {
			u := User{}
			u.Name = username
			u.Spec.GithubID = username
			users[username] = &u
		}
		team.Update(fs, "team.yaml", users)

		// check that the parentTeam is not output
		// and users have changed
		f, err := utils.ReadFile(fs, "team.yaml")
		assert.Nil(t, err)

		var checkTeam Team
		yaml.Unmarshal(f, &checkTeam)

		assert.Nil(t, checkTeam.ParentTeam)
		assert.Equal(t, 1, len(checkTeam.Spec.Owners))
		assert.Equal(t, "owner2", checkTeam.Spec.Owners[0])
	})
}
