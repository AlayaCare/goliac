package engine

import (
	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/entity"
)

type UserSyncPlugin interface {
	// Get the current user list directory path, returns the new user list
	UpdateUsers(repoconfig *config.RepositoryConfig, orguserdirrectorypath string) (map[string]*entity.User, error)
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
