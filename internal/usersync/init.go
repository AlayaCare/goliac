package usersync

import (
	"github.com/Alayacare/goliac/internal/engine"
	"github.com/Alayacare/goliac/internal/github"
)

func InitPlugins(client github.GitHubClient) {
	engine.RegisterPlugin("noop", NewUserSyncPluginNoop())
	engine.RegisterPlugin("shellscript", NewUserSyncPluginShellScript())
	engine.RegisterPlugin("fromgithubsaml", NewUserSyncPluginFromGithubSaml(client))
}
