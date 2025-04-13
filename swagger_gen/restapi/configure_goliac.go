// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"crypto/tls"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"

	"github.com/goliac-project/goliac/swagger_gen/restapi/operations"
	"github.com/goliac-project/goliac/swagger_gen/restapi/operations/app"
	"github.com/goliac-project/goliac/swagger_gen/restapi/operations/auth"
	"github.com/goliac-project/goliac/swagger_gen/restapi/operations/external"
	"github.com/goliac-project/goliac/swagger_gen/restapi/operations/health"
)

//go:generate swagger generate server --target ../../swagger_gen --name Goliac --spec ../../docs/api_docs/bundle.yaml --principal interface{}

func configureFlags(api *operations.GoliacAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.GoliacAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.UseSwaggerUI()
	// To continue using redoc as your UI, uncomment the following line
	// api.UseRedoc()

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	if api.AuthGetAuthenticationCallbackHandler == nil {
		api.AuthGetAuthenticationCallbackHandler = auth.GetAuthenticationCallbackHandlerFunc(func(params auth.GetAuthenticationCallbackParams) middleware.Responder {
			return middleware.NotImplemented("operation auth.GetAuthenticationCallback has not yet been implemented")
		})
	}
	if api.AuthGetAuthenticationLoginHandler == nil {
		api.AuthGetAuthenticationLoginHandler = auth.GetAuthenticationLoginHandlerFunc(func(params auth.GetAuthenticationLoginParams) middleware.Responder {
			return middleware.NotImplemented("operation auth.GetAuthenticationLogin has not yet been implemented")
		})
	}
	if api.AppGetCollaboratorHandler == nil {
		api.AppGetCollaboratorHandler = app.GetCollaboratorHandlerFunc(func(params app.GetCollaboratorParams) middleware.Responder {
			return middleware.NotImplemented("operation app.GetCollaborator has not yet been implemented")
		})
	}
	if api.AppGetCollaboratorsHandler == nil {
		api.AppGetCollaboratorsHandler = app.GetCollaboratorsHandlerFunc(func(params app.GetCollaboratorsParams) middleware.Responder {
			return middleware.NotImplemented("operation app.GetCollaborators has not yet been implemented")
		})
	}
	if api.AuthGetGithubUserHandler == nil {
		api.AuthGetGithubUserHandler = auth.GetGithubUserHandlerFunc(func(params auth.GetGithubUserParams) middleware.Responder {
			return middleware.NotImplemented("operation auth.GetGithubUser has not yet been implemented")
		})
	}
	if api.HealthGetLivenessHandler == nil {
		api.HealthGetLivenessHandler = health.GetLivenessHandlerFunc(func(params health.GetLivenessParams) middleware.Responder {
			return middleware.NotImplemented("operation health.GetLiveness has not yet been implemented")
		})
	}
	if api.HealthGetReadinessHandler == nil {
		api.HealthGetReadinessHandler = health.GetReadinessHandlerFunc(func(params health.GetReadinessParams) middleware.Responder {
			return middleware.NotImplemented("operation health.GetReadiness has not yet been implemented")
		})
	}
	if api.AppGetRepositoriesHandler == nil {
		api.AppGetRepositoriesHandler = app.GetRepositoriesHandlerFunc(func(params app.GetRepositoriesParams) middleware.Responder {
			return middleware.NotImplemented("operation app.GetRepositories has not yet been implemented")
		})
	}
	if api.AppGetRepositoryHandler == nil {
		api.AppGetRepositoryHandler = app.GetRepositoryHandlerFunc(func(params app.GetRepositoryParams) middleware.Responder {
			return middleware.NotImplemented("operation app.GetRepository has not yet been implemented")
		})
	}
	if api.AppGetStatiticsHandler == nil {
		api.AppGetStatiticsHandler = app.GetStatiticsHandlerFunc(func(params app.GetStatiticsParams) middleware.Responder {
			return middleware.NotImplemented("operation app.GetStatitics has not yet been implemented")
		})
	}
	if api.AppGetStatusHandler == nil {
		api.AppGetStatusHandler = app.GetStatusHandlerFunc(func(params app.GetStatusParams) middleware.Responder {
			return middleware.NotImplemented("operation app.GetStatus has not yet been implemented")
		})
	}
	if api.AppGetTeamHandler == nil {
		api.AppGetTeamHandler = app.GetTeamHandlerFunc(func(params app.GetTeamParams) middleware.Responder {
			return middleware.NotImplemented("operation app.GetTeam has not yet been implemented")
		})
	}
	if api.AppGetTeamsHandler == nil {
		api.AppGetTeamsHandler = app.GetTeamsHandlerFunc(func(params app.GetTeamsParams) middleware.Responder {
			return middleware.NotImplemented("operation app.GetTeams has not yet been implemented")
		})
	}
	if api.AppGetUnmanagedHandler == nil {
		api.AppGetUnmanagedHandler = app.GetUnmanagedHandlerFunc(func(params app.GetUnmanagedParams) middleware.Responder {
			return middleware.NotImplemented("operation app.GetUnmanaged has not yet been implemented")
		})
	}
	if api.AppGetUserHandler == nil {
		api.AppGetUserHandler = app.GetUserHandlerFunc(func(params app.GetUserParams) middleware.Responder {
			return middleware.NotImplemented("operation app.GetUser has not yet been implemented")
		})
	}
	if api.AppGetUsersHandler == nil {
		api.AppGetUsersHandler = app.GetUsersHandlerFunc(func(params app.GetUsersParams) middleware.Responder {
			return middleware.NotImplemented("operation app.GetUsers has not yet been implemented")
		})
	}
	if api.AuthGetWorkflowHandler == nil {
		api.AuthGetWorkflowHandler = auth.GetWorkflowHandlerFunc(func(params auth.GetWorkflowParams) middleware.Responder {
			return middleware.NotImplemented("operation auth.GetWorkflow has not yet been implemented")
		})
	}
	if api.AuthGetWorkflowsHandler == nil {
		api.AuthGetWorkflowsHandler = auth.GetWorkflowsHandlerFunc(func(params auth.GetWorkflowsParams) middleware.Responder {
			return middleware.NotImplemented("operation auth.GetWorkflows has not yet been implemented")
		})
	}
	if api.ExternalPostExternalCreateRepositoryHandler == nil {
		api.ExternalPostExternalCreateRepositoryHandler = external.PostExternalCreateRepositoryHandlerFunc(func(params external.PostExternalCreateRepositoryParams) middleware.Responder {
			return middleware.NotImplemented("operation external.PostExternalCreateRepository has not yet been implemented")
		})
	}
	if api.AppPostFlushCacheHandler == nil {
		api.AppPostFlushCacheHandler = app.PostFlushCacheHandlerFunc(func(params app.PostFlushCacheParams) middleware.Responder {
			return middleware.NotImplemented("operation app.PostFlushCache has not yet been implemented")
		})
	}
	if api.AppPostResyncHandler == nil {
		api.AppPostResyncHandler = app.PostResyncHandlerFunc(func(params app.PostResyncParams) middleware.Responder {
			return middleware.NotImplemented("operation app.PostResync has not yet been implemented")
		})
	}
	if api.AuthPostWorkflowHandler == nil {
		api.AuthPostWorkflowHandler = auth.PostWorkflowHandlerFunc(func(params auth.PostWorkflowParams) middleware.Responder {
			return middleware.NotImplemented("operation auth.PostWorkflow has not yet been implemented")
		})
	}

	api.PreServerShutdown = func() {}

	api.ServerShutdown = func() {}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix".
func configureServer(s *http.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation.
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics.
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}
