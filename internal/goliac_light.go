package internal

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/engine"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/sirupsen/logrus"
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
	fs := osfs.New(path)
	errs, warns := g.local.LoadAndValidateLocal(fs)

	for _, warn := range warns {
		logrus.Warn(warn)
	}
	if len(errs) != 0 {
		for _, err := range errs {
			logrus.Error(err)
		}
		return fmt.Errorf("not able to validate the goliac organization: see logs")
	}

	return nil
}
