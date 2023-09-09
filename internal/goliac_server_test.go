package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Alayacare/goliac/internal/engine"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/swagger_gen/restapi/operations/app"
)

type GoliacLocalMock struct {
	teams         map[string]*entity.Team
	repositories  map[string]*entity.Repository
	users         map[string]*entity.User
	externalUsers map[string]*entity.User
	rulesets      map[string]*entity.RuleSet
}

func (g *GoliacLocalMock) Teams() map[string]*entity.Team {
	return g.teams
}
func (g *GoliacLocalMock) Repositories() map[string]*entity.Repository {
	return g.repositories
}
func (g *GoliacLocalMock) Users() map[string]*entity.User {
	return g.users
}
func (g *GoliacLocalMock) ExternalUsers() map[string]*entity.User {
	return g.externalUsers
}
func (g *GoliacLocalMock) RuleSets() map[string]*entity.RuleSet {
	return g.rulesets
}

func fixtureGoliacLocal() *GoliacLocalMock {
	l := GoliacLocalMock{
		teams:         make(map[string]*entity.Team),
		repositories:  make(map[string]*entity.Repository),
		users:         make(map[string]*entity.User),
		externalUsers: make(map[string]*entity.User),
		rulesets:      make(map[string]*entity.RuleSet),
	}

	// users
	user1 := entity.User{}
	user1.Name = "user1"
	user1.Spec.GithubID = "github1"

	user2 := entity.User{}
	user2.Name = "user2"
	user2.Spec.GithubID = "github2"

	user3 := entity.User{}
	user3.Name = "user3"
	user3.Spec.GithubID = "github3"

	l.users["user1"] = &user1
	l.users["user2"] = &user2
	l.users["user3"] = &user3

	// external users
	userE1 := entity.User{}
	userE1.Name = "userE1"
	userE1.Spec.GithubID = "githubE1"

	l.externalUsers["userE1"] = &userE1

	// teams
	ateam := entity.Team{}
	ateam.Name = "ateam"
	ateam.Spec.Owners = []string{"user1"}
	ateam.Spec.Members = []string{"user3"}

	mixteam := entity.Team{}
	mixteam.Name = "mixteam"
	mixteam.Spec.Owners = []string{"user1"}
	mixteam.Spec.Members = []string{"userE1"}

	l.teams["ateam"] = &ateam
	l.teams["mixteam"] = &mixteam

	// repositories
	repoA := entity.Repository{}
	repoA.Name = "repoA"
	ownerA := "ateam"
	repoA.Owner = &ownerA
	repoA.Spec.Readers = []string{}
	repoA.Spec.Writers = []string{}

	repoB := entity.Repository{}
	repoB.Name = "repoB"
	ownerB := "mixteam"
	repoB.Owner = &ownerB
	repoB.Spec.Readers = []string{"ateam"}
	repoB.Spec.Writers = []string{}

	l.repositories["repoA"] = &repoA
	l.repositories["repoB"] = &repoB

	return &l
}

type GoliacMock struct {
	local engine.GoliacLocalResources
}

func (g *GoliacMock) Apply(dryrun bool, repo string, branch string, forceresync bool) error {
	return nil
}
func (g *GoliacMock) UsersUpdate(repositoryUrl, branch string) error {
	return nil
}
func (g *GoliacMock) FlushCache() {
}

func (g *GoliacMock) GetLocal() engine.GoliacLocalResources {
	return g.local
}
func NewGoliacMock(local engine.GoliacLocalResources) Goliac {
	mock := GoliacMock{
		local: local,
	}
	return &mock
}

func TestAppGetUsers(t *testing.T) {
	fixture := fixtureGoliacLocal()
	goliac := NewGoliacMock(fixture)
	now := time.Now()
	server := GoliacServerImpl{
		goliac:        goliac,
		ready:         true,
		lastSyncTime:  &now,
		lastSyncError: nil,
	}

	t.Run("happy path: get status", func(t *testing.T) {
		res := server.GetStatus(app.GetStatusParams{})
		payload := res.(*app.GetStatusOK)
		assert.Equal(t, int64(2), payload.Payload.NbRepos)
		assert.Equal(t, int64(2), payload.Payload.NbTeams)
		assert.Equal(t, int64(3), payload.Payload.NbUsers)
		assert.Equal(t, int64(1), payload.Payload.NbUsersExternal)
	})

	t.Run("happy path: list users", func(t *testing.T) {
		res := server.GetUsers(app.GetUsersParams{})
		payload := res.(*app.GetUsersOK)
		assert.Equal(t, 3, len(payload.Payload)) // 3 users + 1 external
	})

	t.Run("happy path: get user1", func(t *testing.T) {
		res := server.GetUser(app.GetUserParams{UserID: "user1"})
		payload := res.(*app.GetUserOK)
		assert.Equal(t, 2, len(payload.Payload.Teams))
		assert.Equal(t, 2, len(payload.Payload.Teams))
		assert.Equal(t, 2, len(payload.Payload.Repositories))
	})
}
func TestAppGetTeams(t *testing.T) {
	fixture := fixtureGoliacLocal()
	goliac := NewGoliacMock(fixture)
	now := time.Now()
	server := GoliacServerImpl{
		goliac:        goliac,
		ready:         true,
		lastSyncTime:  &now,
		lastSyncError: nil,
	}

	t.Run("happy path: get teams", func(t *testing.T) {
		res := server.GetTeams(app.GetTeamsParams{})
		payload := res.(*app.GetTeamsOK)
		assert.Equal(t, 2, len(payload.Payload))
	})

	t.Run("happy path: get team ", func(t *testing.T) {
		res := server.GetTeam(app.GetTeamParams{TeamID: "ateam"})
		payload := res.(*app.GetTeamOK)
		assert.Equal(t, "ateam", payload.Payload.Name)
		assert.Equal(t, 1, len(payload.Payload.Owners))
		assert.Equal(t, 1, len(payload.Payload.Members))
	})
	t.Run("not happy path: team not found", func(t *testing.T) {
		res := server.GetTeam(app.GetTeamParams{TeamID: "unknown"})
		assert.NotZero(t, res.(*app.GetTeamDefault))
	})
}

func TestAppGetRepositories(t *testing.T) {
	fixture := fixtureGoliacLocal()
	goliac := NewGoliacMock(fixture)
	now := time.Now()
	server := GoliacServerImpl{
		goliac:        goliac,
		ready:         true,
		lastSyncTime:  &now,
		lastSyncError: nil,
	}

	t.Run("happy path: get repositories", func(t *testing.T) {
		res := server.GetRepositories(app.GetRepositoriesParams{})
		payload := res.(*app.GetRepositoriesOK)
		assert.Equal(t, 2, len(payload.Payload))
	})

	t.Run("happy path: get repository ", func(t *testing.T) {
		res := server.GetRepository(app.GetRepositoryParams{RepositoryID: "repoB"})
		payload := res.(*app.GetRepositoryOK)
		assert.Equal(t, "repoB", payload.Payload.Name)
		assert.Equal(t, 2, len(payload.Payload.Teams))
	})

	t.Run("not happy path: repository not found", func(t *testing.T) {
		res := server.GetRepository(app.GetRepositoryParams{RepositoryID: "repoC"})
		assert.NotZero(t, res.(*app.GetRepositoryDefault))
	})
}
