package usersync

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/engine"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/observability"
	"github.com/go-git/go-billy/v5"
)

type UserSyncPluginNoop struct {
	Fs billy.Filesystem
}

func NewUserSyncPluginNoop() engine.UserSyncPlugin {
	return &UserSyncPluginNoop{}
}

func (p *UserSyncPluginNoop) UpdateUsers(repoconfig *config.RepositoryConfig, fs billy.Filesystem, orguserdirrectorypath string, feedback observability.RemoteLoadFeedback) (map[string]*entity.User, error) {

	users, errs, _ := entity.ReadUserDirectory(fs, orguserdirrectorypath)
	if len(errs) > 0 {
		return nil, fmt.Errorf("cannot load org users (for example: %v)", errs[0])
	}

	return users, nil
}
