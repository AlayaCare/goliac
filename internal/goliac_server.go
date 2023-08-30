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

type GoliacServer interface {
	Serve()
	GetLiveness(health.GetLivenessParams) middleware.Responder
	GetReadiness(health.GetReadinessParams) middleware.Responder
	PostFlushCache(app.PostFlushCacheParams) middleware.Responder
	PostResync(app.PostResyncParams) middleware.Responder
	GetStatus(app.GetStatusParams) middleware.Responder

	GetUsers(app.GetUsersParams) middleware.Responder
	GetUser(app.GetUserParams) middleware.Responder
}

type GoliacServerImpl struct {
	goliac     Goliac
	applyMutex gosync.Mutex
	// when the server has finished to load the local configuration
	ready         bool
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

func (g *GoliacServerImpl) GetUsers(app.GetUsersParams) middleware.Responder {
	users := models.Users{
		Users: make([]*models.User, 0),
	}
	local := g.goliac.GetLocal()
	for username := range local.Users() {
		u := models.User{
			External: false,
			Name:     username,
		}
		users.Users = append(users.Users, &u)
	}
	for username := range local.ExternalUsers() {
		u := models.User{
			External: true,
			Name:     username,
		}
		users.Users = append(users.Users, &u)
	}
	return app.NewGetUsersOK().WithPayload(&users)
}

func (g *GoliacServerImpl) GetUser(params app.GetUserParams) middleware.Responder {
	userdetails := models.UserDetails{
		Teams:        make([]*models.Team, 0),
		Repositories: make([]*models.Repository, 0),
	}
	local := g.goliac.GetLocal()
	for teamname, team := range local.Teams() {
		for _, owner := range team.Data.Owners {
			if owner == params.UserID {
				team := models.Team{
					Name:    teamname,
					Members: team.Data.Members,
					Owners:  team.Data.Owners,
				}
				userdetails.Teams = append(userdetails.Teams, &team)
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
				userdetails.Teams = append(userdetails.Teams, &team)
				break
			}
		}
	}

	teamRepo := make(map[string][]*entity.Repository)
	for _, repo := range local.Repositories() {
		if repo.Owner != nil {
			teamRepo[*repo.Owner] = append(teamRepo[*repo.Owner], repo)
		}
		for _, r := range repo.Data.Readers {
			teamRepo[r] = append(teamRepo[*repo.Owner], repo)
		}
		for _, w := range repo.Data.Writers {
			teamRepo[w] = append(teamRepo[*repo.Owner], repo)
		}
	}
	fmt.Println(local.Repositories())
	fmt.Println(teamRepo)

	for _, team := range userdetails.Teams {
		if repositories, ok := teamRepo[team.Name]; ok {
			for _, r := range repositories {
				repo := models.Repository{
					Name:     r.Metadata.Name,
					Public:   r.Data.IsPublic,
					Archived: r.Data.IsArchived,
				}
				userdetails.Repositories = append(userdetails.Repositories, &repo)
			}
		}
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
