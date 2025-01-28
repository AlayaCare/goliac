package usersync

import (
	"context"
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/engine"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/github"
	"github.com/Alayacare/goliac/internal/observability"
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

/*
Return a map of [username]*entity.User
*/
func (p *UserSyncPluginFromGithubSaml) UpdateUsers(repoconfig *config.RepositoryConfig, fs billy.Filesystem, orguserdirrectorypath string, feedback observability.RemoteObservability) (map[string]*entity.User, error) {

	ctx := context.Background()
	pendingLogin, err := engine.LoadGithubLoginPendingInvitations(ctx, p.client)
	if err != nil {
		return nil, err
	}
	users, err := engine.LoadUsersFromGithubOrgSaml(ctx, p.client, feedback)

	finalUsers := make(map[string]*entity.User)
	for name, user := range users {
		if _, ok := pendingLogin[user.Spec.GithubID]; !ok {
			finalUsers[name] = user
		}
	}

	if len(finalUsers) == 0 {
		return nil, fmt.Errorf("not able to find any SAML identities")
	}

	return finalUsers, err
}
