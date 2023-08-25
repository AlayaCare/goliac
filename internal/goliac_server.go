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
	"github.com/Alayacare/goliac/internal/swagger_gen/models"
	"github.com/Alayacare/goliac/internal/swagger_gen/restapi"
	"github.com/Alayacare/goliac/internal/swagger_gen/restapi/operations"
	"github.com/Alayacare/goliac/internal/swagger_gen/restapi/operations/health"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware"
	"github.com/sirupsen/logrus"
)

type GoliacServer interface {
	Serve()
	GetLiveness(health.GetLivenessParams) middleware.Responder
	GetReadiness(health.GetReadinessParams) middleware.Responder
}

type GoliacServerImpl struct {
	goliac     Goliac
	applyMutex gosync.Mutex
}

func NewGoliacServer(goliac Goliac) GoliacServer {
	return &GoliacServerImpl{
		goliac: goliac,
	}
}

func (c *GoliacServerImpl) GetLiveness(params health.GetLivenessParams) middleware.Responder {
	return health.NewGetLivenessOK().WithPayload(&models.Health{Status: "OK"})
}

func (c *GoliacServerImpl) GetReadiness(params health.GetReadinessParams) middleware.Responder {
	return health.NewGetLivenessOK().WithPayload(&models.Health{Status: "OK"})
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
		interval := 0
		for {
			select {
			case <-stopCh:
				restserver.Shutdown()
				return
			default:
				interval--
				time.Sleep(1 * time.Second)
				if interval <= 0 {
					// Do some work here
					err := g.serveApply()
					if err != nil {
						logrus.Error(err)
					}
					interval = config.Config.ServerApplyInterval
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
