package config

import (
	"context"
	"os"

	"github.com/caarlos0/env"
	"github.com/sirupsen/logrus"
)

func init() {
	env.Parse(&Config)

	setupLogrus()
	if Config.OpenTelemetryEnabled {
		err := setupTraceProvider(context.Background())
		if err != nil {
			panic(err)
		}
	}
}

func setupLogrus() {
	l, err := logrus.ParseLevel(Config.LogrusLevel)
	if err != nil {
		logrus.WithField("err", err).Fatalf("failed to set logrus level:%s", Config.LogrusLevel)
	}
	logrus.SetLevel(l)
	logrus.SetOutput(os.Stdout)
	switch Config.LogrusFormat {
	case "text":
		logrus.SetFormatter(&logrus.TextFormatter{})
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		logrus.Warnf("unexpected logrus format: %s, should be one of: text, json", Config.LogrusFormat)
	}
}
