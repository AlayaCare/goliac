package usersync

import (
	"context"
	"fmt"

	"github.com/go-git/go-billy/v5"
	"github.com/goliac-project/goliac/internal/config"
	"github.com/goliac-project/goliac/internal/engine"
	"github.com/goliac-project/goliac/internal/entity"
	"github.com/goliac-project/goliac/internal/github"
	"github.com/goliac-project/goliac/internal/observability"
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
func (p *UserSyncPluginFromGithubSaml) UpdateUsers(repoconfig *config.RepositoryConfig, fs billy.Filesystem, orguserdirrectorypath string, feedback observability.RemoteObservability, logsCollector *observability.LogCollection) map[string]*entity.User {

	ctx := context.Background()
	pendingLogin, err := engine.LoadGithubLoginPendingInvitations(ctx, p.client)
	if err != nil {
		logsCollector.AddError(fmt.Errorf("not able to load pending invitations: %w", err))
		return nil
	}
	users, err := engine.LoadUsersFromGithubOrgSaml(ctx, p.client, feedback)
	if err != nil {
		logsCollector.AddError(fmt.Errorf("not able to load users from Github: %w", err))
		return nil
	}

	finalUsers := make(map[string]*entity.User)
	for name, user := range users {
		if _, ok := pendingLogin[user.Spec.GithubID]; !ok {
			finalUsers[name] = user
		}
	}

	if len(finalUsers) == 0 {
		return nil
	}

	return finalUsers
}
