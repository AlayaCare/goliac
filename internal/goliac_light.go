package internal

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/engine"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

/*
 * This "version" of Goliac is here just to validate a local
 * teams directory. It is mainly used for CI purpose when we need to validate
 * a PR
 */
type GoliacLight interface {
	// Validate a local teams directory
	Validate(path string) error
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

func (g *GoliacLightImpl) Validate(path string) error {
	fs := afero.NewOsFs()
	errs, warns := g.local.LoadAndValidateLocal(fs, path)

	for _, warn := range warns {
		logrus.Warn(warn)
	}
	if len(errs) != 0 {
		for _, err := range errs {
			logrus.Error(err)
		}
		return fmt.Errorf("Not able to validate the goliac organization: see logs")
	}

	return nil
}
