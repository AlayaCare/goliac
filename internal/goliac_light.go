package internal

import (
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/goliac-project/goliac/internal/config"
	"github.com/goliac-project/goliac/internal/engine"
	"github.com/goliac-project/goliac/internal/observability"
	"github.com/sirupsen/logrus"
)

/*
 * This "version" of Goliac is here just to validate a local
 * teams directory. It is mainly used for CI purpose when we need to validate
 * a PR
 */
type GoliacLight interface {
	// Validate a local teams directory
	Validate(path string, errorCollector *observability.ErrorCollection)
}

type GoliacLightImpl struct {
	local      engine.GoliacLocal
	repoconfig *config.RepositoryConfig
}

func NewGoliacLightImpl() (GoliacLight, error) {
	return &GoliacLightImpl{
		local:      engine.NewGoliacLocalImpl(),
		repoconfig: &config.RepositoryConfig{},
	}, nil
}

func (g *GoliacLightImpl) Validate(path string, errorCollector *observability.ErrorCollection) {
	fs := osfs.New(path)
	g.local.LoadAndValidateLocal(fs, errorCollector)

	for _, warn := range errorCollector.Warns {
		logrus.Warn(warn)
	}
	if errorCollector.HasErrors() {
		for _, err := range errorCollector.Errors {
			logrus.Error(err)
		}
	}
}
