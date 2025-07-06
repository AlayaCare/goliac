package usersync

import (
	"github.com/go-git/go-billy/v5"
	"github.com/goliac-project/goliac/internal/config"
	"github.com/goliac-project/goliac/internal/engine"
	"github.com/goliac-project/goliac/internal/entity"
	"github.com/goliac-project/goliac/internal/observability"
)

type UserSyncPluginNoop struct {
	Fs billy.Filesystem
}

func NewUserSyncPluginNoop() engine.UserSyncPlugin {
	return &UserSyncPluginNoop{}
}

func (p *UserSyncPluginNoop) UpdateUsers(repoconfig *config.RepositoryConfig, fs billy.Filesystem, orguserdirrectorypath string, feedback observability.RemoteObservability, logsCollector *observability.LogCollection) map[string]*entity.User {

	users := entity.ReadUserDirectory(fs, orguserdirrectorypath, logsCollector)
	return users
}
