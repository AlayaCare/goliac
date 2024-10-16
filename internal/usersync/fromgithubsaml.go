package usersync

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/engine"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/github"
	"github.com/go-git/go-billy/v5"
)

/*
 * UserSyncPluginFromGithubSaml: this plugin sync users from Github if the SAML IdP
 * integration has been added (to enable this feature you need a Github Entreprise subscription)
 *
 * Note: this plugin doesn't clear the Remote cache.
 */
type UserSyncPluginFromGithubSaml struct {
	client github.GitHubClient
}

func NewUserSyncPluginFromGithubSaml(client github.GitHubClient) engine.UserSyncPlugin {
	return &UserSyncPluginFromGithubSaml{
		client: client,
	}
}

func (p *UserSyncPluginFromGithubSaml) UpdateUsers(repoconfig *config.RepositoryConfig, fs billy.Filesystem, orguserdirrectorypath string) (map[string]*entity.User, error) {

	users, err := engine.LoadUsersFromGithubOrgSaml(p.client)

	if len(users) == 0 {
		return nil, fmt.Errorf("not able to find any SAML identities")
	}

	return users, err
}
