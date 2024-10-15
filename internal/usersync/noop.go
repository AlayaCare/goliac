package usersync

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/engine"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
)

type UserSyncPluginNoop struct {
	Fs billy.Filesystem
}

func NewUserSyncPluginNoop() engine.UserSyncPlugin {
	return &UserSyncPluginNoop{
		Fs: osfs.New("/"),
	}
}

func (p *UserSyncPluginNoop) UpdateUsers(repoconfig *config.RepositoryConfig, orguserdirrectorypath string) (map[string]*entity.User, error) {
	users, errs, _ := entity.ReadUserDirectory(p.Fs, orguserdirrectorypath)
	if len(errs) > 0 {
		return nil, fmt.Errorf("cannot load org users (for example: %v)", errs[0])
	}

	return users, nil
}
