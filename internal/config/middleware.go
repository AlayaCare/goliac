package config

import (
	"net/http"

	"github.com/phyber/negroni-gzip/gzip"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
)

// ServerShutdown is a callback function that will be called when
// we tear down the golang-skeleton server
func ServerShutdown() {
}

// SetupGlobalMiddleware setup the global middleware
func SetupGlobalMiddleware(handler http.Handler) http.Handler {
	n := negroni.New()

	if Config.MiddlewareGzipEnabled {
		n.Use(gzip.Gzip(gzip.DefaultCompression))
	}

	if Config.MiddlewareVerboseLoggerEnabled {
		middleware := NewMiddlewareFromLogger(logrus.StandardLogger(), logrus.DebugLevel, "goliac")

		for _, u := range Config.MiddlewareVerboseLoggerExcludeURLs {
			middleware.ExcludeURL(u)
		}

		n.Use(middleware)
	}

	if Config.CORSEnabled {
		n.Use(cors.New(cors.Options{
			AllowedOrigins:   Config.CORSAllowedOrigins,
			AllowedHeaders:   Config.CORSAllowedHeaders,
			ExposedHeaders:   Config.CORSExposedHeaders,
			AllowedMethods:   Config.CORSAllowedMethods,
			AllowCredentials: Config.CORSAllowCredentials,
		}))
	}

	n.Use(&negroni.Static{
		Dir:       http.Dir("./browser/goliac-ui/dist/"),
		Prefix:    Config.WebPrefix,
		IndexFile: "index.html",
	})

	n.Use(setupRecoveryMiddleware())

	n.UseHandler(handler)

	return n
}

type recoveryLogger struct{}

func (r *recoveryLogger) Printf(format string, v ...interface{}) {
	logrus.Errorf(format, v...)
}

func (r *recoveryLogger) Println(v ...interface{}) {
	logrus.Errorln(v...)
}

func setupRecoveryMiddleware() *negroni.Recovery {
	r := negroni.NewRecovery()
	r.Logger = &recoveryLogger{}
	return r
}
