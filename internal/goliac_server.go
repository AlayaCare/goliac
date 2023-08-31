package internal

import (
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	gosync "sync"
	"syscall"
	"time"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/swagger_gen/models"
	"github.com/Alayacare/goliac/swagger_gen/restapi"
	"github.com/Alayacare/goliac/swagger_gen/restapi/operations"
	"github.com/Alayacare/goliac/swagger_gen/restapi/operations/app"
	"github.com/Alayacare/goliac/swagger_gen/restapi/operations/health"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware"
	"github.com/sirupsen/logrus"
)

/*
 * GoliacServer is here to run as a serve that
 * - sync/reconciliate periodically
 * - provide a REST API server
 */
type GoliacServer interface {
	Serve()
	GetLiveness(health.GetLivenessParams) middleware.Responder
	GetReadiness(health.GetReadinessParams) middleware.Responder
	PostFlushCache(app.PostFlushCacheParams) middleware.Responder
	PostResync(app.PostResyncParams) middleware.Responder
	GetStatus(app.GetStatusParams) middleware.Responder

	GetUsers(app.GetUsersParams) middleware.Responder
	GetUser(app.GetUserParams) middleware.Responder
	GetTeams(app.GetTeamsParams) middleware.Responder
	GetTeam(app.GetTeamParams) middleware.Responder
	GetRepositories(app.GetRepositoriesParams) middleware.Responder
	GetRepository(app.GetRepositoryParams) middleware.Responder
}

type GoliacServerImpl struct {
	goliac        Goliac
	applyMutex    gosync.Mutex
	ready         bool // when the server has finished to load the local configuration
	lastSyncTime  *time.Time
	lastSyncError error
	syncInterval  int // in seconds time remaining between 2 sync
}

func NewGoliacServer(goliac Goliac) GoliacServer {
	return &GoliacServerImpl{
		goliac: goliac,
		ready:  false,
	}
}

func (g *GoliacServerImpl) GetRepositories(app.GetRepositoriesParams) middleware.Responder {
	local := g.goliac.GetLocal()
	repositories := make(models.Repositories, 0, len(local.Repositories()))

	for _, r := range local.Repositories() {
		repo := models.Repository{
			Name:     r.Metadata.Name,
			Public:   r.Data.IsPublic,
			Archived: r.Data.IsArchived,
		}
		repositories = append(repositories, &repo)
	}

	return app.NewGetRepositoriesOK().WithPayload(repositories)
}

func (g *GoliacServerImpl) GetRepository(params app.GetRepositoryParams) middleware.Responder {
	local := g.goliac.GetLocal()

	repository, found := local.Repositories()[params.RepositoryID]
	if !found {
		message := fmt.Sprintf("Repository %s not found", params.RepositoryID)
		return app.NewGetRepositoryDefault(404).WithPayload(&models.Error{Message: &message})
	}

	readers := make([]*models.RepositoryDetailsReadersItems0, 0)
	writers := make([]*models.RepositoryDetailsWritersItems0, 0)

	for _, r := range repository.Data.Readers {
		reader := models.RepositoryDetailsReadersItems0{
			Name: r,
		}
		readers = append(readers, &reader)
	}

	if repository.Owner != nil {
		writer := models.RepositoryDetailsWritersItems0{
			Name: *repository.Owner,
		}
		writers = append(writers, &writer)
	}

	for _, w := range repository.Data.Writers {
		writer := models.RepositoryDetailsWritersItems0{
			Name: w,
		}
		writers = append(writers, &writer)
	}

	repositoryDetails := models.RepositoryDetails{
		Name:     repository.Metadata.Name,
		Public:   repository.Data.IsPublic,
		Archived: repository.Data.IsArchived,
		Readers:  readers,
		Writers:  writers,
	}

	return app.NewGetRepositoryOK().WithPayload(&repositoryDetails)
}

func (g *GoliacServerImpl) GetTeams(app.GetTeamsParams) middleware.Responder {
	teams := make(models.Teams, 0)

	local := g.goliac.GetLocal()
	for teamname, team := range local.Teams() {
		t := models.Team{
			Name:    teamname,
			Members: team.Data.Members,
			Owners:  team.Data.Owners,
		}
		teams = append(teams, &t)

	}
	return app.NewGetTeamsOK().WithPayload(teams)
}

func (g *GoliacServerImpl) GetTeam(params app.GetTeamParams) middleware.Responder {
	local := g.goliac.GetLocal()

	team, found := local.Teams()[params.TeamID]
	if !found {
		message := fmt.Sprintf("Team %s not found", params.TeamID)
		return app.NewGetTeamDefault(404).WithPayload(&models.Error{Message: &message})
	}

	repos := make(map[string]*entity.Repository)
	for reponame, repo := range local.Repositories() {
		if repo.Owner != nil && *repo.Owner == params.TeamID {
			repos[reponame] = repo
		}
		for _, r := range repo.Data.Readers {
			if r == params.TeamID {
				repos[reponame] = repo
				break
			}
		}
		for _, r := range repo.Data.Writers {
			if r == params.TeamID {
				repos[reponame] = repo
				break
			}
		}
	}

	repositories := make([]*models.Repository, 0, len(repos))
	for reponame, repo := range repos {
		r := models.Repository{
			Name:     reponame,
			Archived: repo.Data.IsArchived,
			Public:   repo.Data.IsPublic,
		}
		repositories = append(repositories, &r)
	}

	teamDetails := models.TeamDetails{
		Owners:       make([]*models.TeamDetailsOwnersItems0, len(team.Data.Owners)),
		Members:      make([]*models.TeamDetailsMembersItems0, len(team.Data.Members)),
		Name:         team.Metadata.Name,
		Repositories: repositories,
	}

	for i, u := range team.Data.Owners {
		if orgUser, ok := local.Users()[u]; ok {
			teamDetails.Owners[i] = &models.TeamDetailsOwnersItems0{
				Name:     u,
				Githubid: orgUser.Data.GithubID,
				External: false,
			}
		} else {
			extUser := local.ExternalUsers()[u]
			teamDetails.Owners[i] = &models.TeamDetailsOwnersItems0{
				Name:     u,
				Githubid: extUser.Data.GithubID,
				External: false,
			}
		}
	}

	for i, u := range team.Data.Members {
		if orgUser, ok := local.Users()[u]; ok {
			teamDetails.Members[i] = &models.TeamDetailsMembersItems0{
				Name:     u,
				Githubid: orgUser.Data.GithubID,
				External: false,
			}
		} else {
			extUser := local.ExternalUsers()[u]
			teamDetails.Members[i] = &models.TeamDetailsMembersItems0{
				Name:     u,
				Githubid: extUser.Data.GithubID,
				External: false,
			}
		}

	}

	return app.NewGetTeamOK().WithPayload(&teamDetails)
}

func (g *GoliacServerImpl) GetUsers(app.GetUsersParams) middleware.Responder {
	users := make(models.Users, 0)

	local := g.goliac.GetLocal()
	for username, user := range local.Users() {
		u := models.User{
			External: false,
			Name:     username,
			Githubid: user.Data.GithubID,
		}
		users = append(users, &u)
	}
	for username, user := range local.ExternalUsers() {
		u := models.User{
			External: true,
			Name:     username,
			Githubid: user.Data.GithubID,
		}
		users = append(users, &u)
	}
	return app.NewGetUsersOK().WithPayload(users)
}

func (g *GoliacServerImpl) GetUser(params app.GetUserParams) middleware.Responder {
	local := g.goliac.GetLocal()

	user, found := local.Users()[params.UserID]
	external := false
	if !found {
		user, found = local.ExternalUsers()[params.UserID]
		if !found {
			message := fmt.Sprintf("User %s not found", params.UserID)
			return app.NewGetUserDefault(404).WithPayload(&models.Error{Message: &message})
		}
		external = true
	}

	userdetails := models.UserDetails{
		Githubid:     user.Data.GithubID,
		External:     external,
		Teams:        make([]*models.Team, 0),
		Repositories: make([]*models.Repository, 0),
	}

	// [teamname]team
	userTeams := make(map[string]*models.Team)
	for teamname, team := range local.Teams() {
		for _, owner := range team.Data.Owners {
			if owner == params.UserID {
				team := models.Team{
					Name:    teamname,
					Members: team.Data.Members,
					Owners:  team.Data.Owners,
				}
				userTeams[teamname] = &team
				break
			}
		}
		for _, member := range team.Data.Members {
			if member == params.UserID {
				team := models.Team{
					Name:    teamname,
					Members: team.Data.Members,
					Owners:  team.Data.Owners,
				}
				userTeams[teamname] = &team
				break
			}
		}
	}

	for _, t := range userTeams {
		userdetails.Teams = append(userdetails.Teams, t)
	}

	// let's sort repo per team
	teamRepo := make(map[string]map[string]*entity.Repository)
	for _, repo := range local.Repositories() {
		if repo.Owner != nil {
			if _, ok := teamRepo[*repo.Owner]; !ok {
				teamRepo[*repo.Owner] = make(map[string]*entity.Repository)
			}
			teamRepo[*repo.Owner][repo.Metadata.Name] = repo
		}
		for _, r := range repo.Data.Readers {
			if _, ok := teamRepo[r]; !ok {
				teamRepo[r] = make(map[string]*entity.Repository)
			}
			teamRepo[r][repo.Metadata.Name] = repo
		}
		for _, w := range repo.Data.Writers {
			if _, ok := teamRepo[w]; !ok {
				teamRepo[w] = make(map[string]*entity.Repository)
			}
			teamRepo[w][repo.Metadata.Name] = repo
		}
	}

	// [reponame]repo
	userRepos := make(map[string]*entity.Repository)
	for _, team := range userdetails.Teams {
		if repositories, ok := teamRepo[team.Name]; ok {
			for n, r := range repositories {
				userRepos[n] = r
			}
		}
	}

	for _, r := range userRepos {
		repo := models.Repository{
			Name:     r.Metadata.Name,
			Public:   r.Data.IsPublic,
			Archived: r.Data.IsArchived,
		}
		userdetails.Repositories = append(userdetails.Repositories, &repo)
	}

	return app.NewGetUserOK().WithPayload(&userdetails)
}

func (g *GoliacServerImpl) GetStatus(app.GetStatusParams) middleware.Responder {
	s := models.Status{
		LastSyncError:   "",
		LastSyncTime:    "N/A",
		NbRepos:         int64(len(g.goliac.GetLocal().Repositories())),
		NbTeams:         int64(len(g.goliac.GetLocal().Teams())),
		NbUsers:         int64(len(g.goliac.GetLocal().Users())),
		NbUsersExternal: int64(len(g.goliac.GetLocal().ExternalUsers())),
	}
	if g.lastSyncError != nil {
		s.LastSyncError = g.lastSyncError.Error()
	}
	if g.lastSyncTime != nil {
		s.LastSyncTime = g.lastSyncTime.UTC().Format("2006-01-02T15:04:05")
	}
	return app.NewGetStatusOK().WithPayload(&s)
}

func (g *GoliacServerImpl) GetLiveness(params health.GetLivenessParams) middleware.Responder {
	return health.NewGetLivenessOK().WithPayload(&models.Health{Status: "OK"})
}

func (g *GoliacServerImpl) GetReadiness(params health.GetReadinessParams) middleware.Responder {
	if g.ready {
		return health.NewGetLivenessOK().WithPayload(&models.Health{Status: "OK"})
	} else {
		message := "Not yet ready, loading local state"
		return health.NewGetLivenessDefault(503).WithPayload(&models.Error{Message: &message})
	}
}

func (g *GoliacServerImpl) PostFlushCache(app.PostFlushCacheParams) middleware.Responder {
	g.goliac.FlushCache()
	return app.NewPostFlushCacheOK()
}

func (g *GoliacServerImpl) PostResync(app.PostResyncParams) middleware.Responder {
	go func() {
		err := g.serveApply()
		now := time.Now()
		g.lastSyncTime = &now
		g.lastSyncError = err
		if err != nil {
			logrus.Error(err)
		}
		g.syncInterval = config.Config.ServerApplyInterval
	}()
	return app.NewPostResyncOK()
}

func (g *GoliacServerImpl) Serve() {
	var wg gosync.WaitGroup
	stopCh := make(chan struct{})

	restserver, err := g.StartRESTApi()
	if err != nil {
		logrus.Fatal(err)
	}

	// start the REST server
	go func() {
		if err := restserver.Serve(); err != nil {
			logrus.Error(err)
			close(stopCh)
		}
	}()

	logrus.Info("Server started")
	// Start the goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		g.syncInterval = 0
		for {
			select {
			case <-stopCh:
				restserver.Shutdown()
				return
			default:
				g.syncInterval--
				time.Sleep(1 * time.Second)
				if g.syncInterval <= 0 {
					// Do some work here
					err := g.serveApply()
					now := time.Now()
					g.lastSyncTime = &now
					g.lastSyncError = err
					if err != nil {
						logrus.Error(err)
					}
					g.syncInterval = config.Config.ServerApplyInterval
				}
			}
		}
	}()

	// Handle OS signals
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	<-signalCh
	fmt.Println("Received OS signal, stopping Goliac...")

	close(stopCh)
	wg.Wait()
}

func (g *GoliacServerImpl) StartRESTApi() (*restapi.Server, error) {
	swaggerSpec, err := loads.Embedded(restapi.SwaggerJSON, restapi.FlatSwaggerJSON)
	if err != nil {
		return nil, err
	}

	api := operations.NewGoliacAPI(swaggerSpec)

	// configure API

	// healthcheck
	api.HealthGetLivenessHandler = health.GetLivenessHandlerFunc(g.GetLiveness)
	api.HealthGetReadinessHandler = health.GetReadinessHandlerFunc(g.GetReadiness)

	api.AppPostFlushCacheHandler = app.PostFlushCacheHandlerFunc(g.PostFlushCache)
	api.AppPostResyncHandler = app.PostResyncHandlerFunc(g.PostResync)
	api.AppGetStatusHandler = app.GetStatusHandlerFunc(g.GetStatus)

	api.AppGetUsersHandler = app.GetUsersHandlerFunc(g.GetUsers)
	api.AppGetUserHandler = app.GetUserHandlerFunc(g.GetUser)
	api.AppGetTeamsHandler = app.GetTeamsHandlerFunc(g.GetTeams)
	api.AppGetTeamHandler = app.GetTeamHandlerFunc(g.GetTeam)
	api.AppGetRepositoriesHandler = app.GetRepositoriesHandlerFunc(g.GetRepositories)
	api.AppGetRepositoryHandler = app.GetRepositoryHandlerFunc(g.GetRepository)

	server := restapi.NewServer(api)

	server.Host = config.Config.SwaggerHost
	server.Port = config.Config.SwaggerPort

	server.ConfigureAPI()

	return server, nil
}

func (g *GoliacServerImpl) serveApply() error {
	if !g.applyMutex.TryLock() {
		// already locked: we are already appyling
		return nil
	}
	defer g.applyMutex.Unlock()
	repo := config.Config.ServerGitRepository
	branch := config.Config.ServerGitBranch
	err := g.goliac.LoadAndValidateGoliacOrganization(repo, branch)
	defer g.goliac.Close()
	if err != nil {
		return fmt.Errorf("failed to load and validate: %s", err)
	}

	// we are ready (to give local state, and to sync with remote)
	g.ready = true

	u, err := url.Parse(repo)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %v", repo, err)
	}
	teamsreponame := strings.TrimSuffix(path.Base(u.Path), filepath.Ext(path.Base(u.Path)))

	err = g.goliac.ApplyToGithub(false, teamsreponame, branch)
	if err != nil {
		return fmt.Errorf("failed to apply on branch %s: %s", branch, err)
	}
	return nil
}
