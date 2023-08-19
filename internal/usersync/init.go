package usersync

import (
	"github.com/Alayacare/goliac/internal/github"
	"github.com/Alayacare/goliac/internal/sync"
)

func InitPlugins(client github.GitHubClient) {
	sync.RegisterPlugin("noop", NewUserSyncPluginNoop())
	sync.RegisterPlugin("shellscript", NewUserSyncPluginShellScript())
	sync.RegisterPlugin("fromgithubsaml", NewUserSyncPluginFromGithubSaml(client))
}
