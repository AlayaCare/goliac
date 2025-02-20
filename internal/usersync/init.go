package usersync

import (
	"github.com/goliac-project/goliac/internal/engine"
	"github.com/goliac-project/goliac/internal/github"
)

func InitPlugins(client github.GitHubClient) {
	engine.RegisterPlugin("noop", NewUserSyncPluginNoop())
	engine.RegisterPlugin("shellscript", NewUserSyncPluginShellScript())
	engine.RegisterPlugin("fromgithubsaml", NewUserSyncPluginFromGithubSaml(client))
}
