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

func TestGenerateCodeownersContent(t *testing.T) {
	t.Run("empty codeowners", func(t *testing.T) {
		repo := &Repository{}
		content := repo.GenerateCodeownersContent("myorg")
		assert.Equal(t, "", content)
	})

	t.Run("single team owner", func(t *testing.T) {
		repo := &Repository{}
		repo.Spec.Codeowners = []RepositoryCodeownersEntry{
			{Pattern: "*", Owners: []string{"sre"}},
		}
		content := repo.GenerateCodeownersContent("myorg")
		expected := "# DO NOT MODIFY THIS FILE MANUALLY\n# This file is managed by Goliac\n\n* @myorg/sre\n"
		assert.Equal(t, expected, content)
	})

	t.Run("multiple entries with teams and users", func(t *testing.T) {
		repo := &Repository{}
		repo.Spec.Codeowners = []RepositoryCodeownersEntry{
			{Pattern: "*", Owners: []string{"sre"}},
			{Pattern: "/labs/anyscale/", Owners: []string{"labs-team", "@some-user"}},
		}
		content := repo.GenerateCodeownersContent("myorg")
		expected := "# DO NOT MODIFY THIS FILE MANUALLY\n# This file is managed by Goliac\n\n* @myorg/sre\n/labs/anyscale/ @myorg/labs-team @some-user\n"
		assert.Equal(t, expected, content)
	})

	t.Run("multiple owners per pattern", func(t *testing.T) {
		repo := &Repository{}
		repo.Spec.Codeowners = []RepositoryCodeownersEntry{
			{Pattern: "*.go", Owners: []string{"backend", "devops"}},
		}
		content := repo.GenerateCodeownersContent("myorg")
		expected := "# DO NOT MODIFY THIS FILE MANUALLY\n# This file is managed by Goliac\n\n*.go @myorg/backend @myorg/devops\n"
		assert.Equal(t, expected, content)
	})

	t.Run("raw only", func(t *testing.T) {
		repo := &Repository{}
		repo.Spec.CodeownersRaw = "/vendor/ @external-user\n/docs/ @myorg/docs-team"
		content := repo.GenerateCodeownersContent("myorg")
		// sorted by pattern length: /docs/ (7) < /vendor/ (9)
		expected := "# DO NOT MODIFY THIS FILE MANUALLY\n# This file is managed by Goliac\n\n/docs/ @myorg/docs-team\n/vendor/ @external-user\n"
		assert.Equal(t, expected, content)
	})

	t.Run("hybrid structured and raw", func(t *testing.T) {
		repo := &Repository{}
		repo.Spec.Codeowners = []RepositoryCodeownersEntry{
			{Pattern: "*", Owners: []string{"sre"}},
			{Pattern: "ac-live-data/", Owners: []string{"data-team"}},
		}
		repo.Spec.CodeownersRaw = "/vendor/ @external-user"
		content := repo.GenerateCodeownersContent("myorg")
		// merged then sorted: * (1), /vendor/ (9), ac-live-data/ (13)
		expected := "# DO NOT MODIFY THIS FILE MANUALLY\n# This file is managed by Goliac\n\n* @myorg/sre\n/vendor/ @external-user\nac-live-data/ @myorg/data-team\n"
		assert.Equal(t, expected, content)
	})

	t.Run("sorts merged rules by pattern length", func(t *testing.T) {
		repo := &Repository{}
		repo.Spec.Codeowners = []RepositoryCodeownersEntry{
			{Pattern: "/zz/long/", Owners: []string{"a"}},
		}
		repo.Spec.CodeownersRaw = "* @x\n/foo/ @y"
		content := repo.GenerateCodeownersContent("myorg")
		expected := "# DO NOT MODIFY THIS FILE MANUALLY\n# This file is managed by Goliac\n\n* @x\n/foo/ @y\n/zz/long/ @myorg/a\n"
		assert.Equal(t, expected, content)
	})

	t.Run("raw comment lines stay above rules", func(t *testing.T) {
		repo := &Repository{}
		repo.Spec.CodeownersRaw = `# not a rule
# second comment
/long/path/ @a
* @b`
		content := repo.GenerateCodeownersContent("myorg")
		expected := "# DO NOT MODIFY THIS FILE MANUALLY\n# This file is managed by Goliac\n\n# not a rule\n# second comment\n\n* @b\n/long/path/ @a\n"
		assert.Equal(t, expected, content)
	})

	t.Run("empty raw whitespace is ignored", func(t *testing.T) {
		repo := &Repository{}
		repo.Spec.CodeownersRaw = "   \n  \n  "
		content := repo.GenerateCodeownersContent("myorg")
		assert.Equal(t, "", content)
	})

	t.Run("both empty returns empty", func(t *testing.T) {
		repo := &Repository{}
		repo.Spec.Codeowners = []RepositoryCodeownersEntry{}
		repo.Spec.CodeownersRaw = ""
		content := repo.GenerateCodeownersContent("myorg")
		assert.Equal(t, "", content)
	})
}

func TestRepositoryCodeownersValidation(t *testing.T) {
	t.Run("happy path: valid codeowners with team reference", func(t *testing.T) {
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  codeowners:
    - pattern: "*"
      owners:
        - team1
`), 0644)
		assert.Nil(t, err)

		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		repos := ReadRepositories(fs, "archived", "teams", teams, map[string]*User{}, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.Equal(t, 1, len(repos))
	})

	t.Run("happy path: codeowners with direct user reference", func(t *testing.T) {
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  codeowners:
    - pattern: "*"
      owners:
        - "@some-github-user"
`), 0644)
		assert.Nil(t, err)

		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		repos := ReadRepositories(fs, "archived", "teams", teams, map[string]*User{}, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.Equal(t, 1, len(repos))
	})

	t.Run("not happy path: codeowners with invalid team", func(t *testing.T) {
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  codeowners:
    - pattern: "*"
      owners:
        - nonexistent-team
`), 0644)
		assert.Nil(t, err)

		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		ReadRepositories(fs, "archived", "teams", teams, map[string]*User{}, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.True(t, logsCollector.HasErrors())
	})

	t.Run("not happy path: codeowners with empty pattern", func(t *testing.T) {
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  codeowners:
    - pattern: ""
      owners:
        - team1
`), 0644)
		assert.Nil(t, err)

		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		ReadRepositories(fs, "archived", "teams", teams, map[string]*User{}, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.True(t, logsCollector.HasErrors())
	})

	t.Run("happy path: codeowners_raw only", func(t *testing.T) {
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  codeowners_raw: |
    * @AlayaCare/team-badwolf
    ac-live-data/ @AlayaCare/team-sphinx
`), 0644)
		assert.Nil(t, err)

		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		repos := ReadRepositories(fs, "archived", "teams", teams, map[string]*User{}, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.Equal(t, 1, len(repos))
	})

	t.Run("happy path: hybrid codeowners and codeowners_raw", func(t *testing.T) {
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  codeowners:
    - pattern: "*"
      owners:
        - team1
  codeowners_raw: |
    /vendor/ @external-contributor
`), 0644)
		assert.Nil(t, err)

		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		repos := ReadRepositories(fs, "archived", "teams", teams, map[string]*User{}, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.False(t, logsCollector.HasErrors())
		assert.Equal(t, 1, len(repos))
	})

	t.Run("not happy path: codeowners with empty owners", func(t *testing.T) {
		fs := memfs.New()
		fixtureCreateUserTeam(t, fs)

		err := utils.WriteFile(fs, "teams/team1/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
spec:
  codeowners:
    - pattern: "*"
      owners: []
`), 0644)
		assert.Nil(t, err)

		logsCollector := observability.NewLogCollection()
		users := ReadUserDirectory(fs, "users", logsCollector)
		teams := ReadTeamDirectory(fs, "teams", users, logsCollector)
		ReadRepositories(fs, "archived", "teams", teams, map[string]*User{}, users, []*config.GithubCustomProperty{}, logsCollector)
		assert.True(t, logsCollector.HasErrors())
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
