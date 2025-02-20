package engine

import (
	"github.com/go-git/go-billy/v5"
	"github.com/goliac-project/goliac/internal/config"
	"github.com/goliac-project/goliac/internal/entity"
	"github.com/goliac-project/goliac/internal/observability"
)

type UserSyncPlugin interface {
	// Get the current user list directory path, returns the new user list
	UpdateUsers(repoconfig *config.RepositoryConfig, fs billy.Filesystem, orguserdirrectorypath string, feedback observability.RemoteObservability) (map[string]*entity.User, error)
}

var plugins map[string]UserSyncPlugin

func RegisterPlugin(name string, plugin UserSyncPlugin) {
	if plugins == nil {
		plugins = make(map[string]UserSyncPlugin)
	}
	plugins[name] = plugin

}

func GetUserSyncPlugin(pluginname string) (UserSyncPlugin, bool) {
	plugin, found := plugins[pluginname]
	return plugin, found
}
