package usersync

import (
	"encoding/json"
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/github"
	"github.com/Alayacare/goliac/internal/sync"
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

func NewUserSyncPluginFromGithubSaml(client github.GitHubClient) sync.UserSyncPlugin {
	return &UserSyncPluginFromGithubSaml{
		client: client,
	}
}

func (p *UserSyncPluginFromGithubSaml) UpdateUsers(repoconfig *config.RepositoryConfig, orguserdirrectorypath string) (map[string]*entity.User, error) {

	users, err := p.loadUsersFromGithubOrgSaml()

	if len(users) == 0 {
		return nil, fmt.Errorf("Not able to find any SAML identities")
	}

	return users, err
}

const listUsersFromGithubOrgSaml = `
query listSamlUsers($orgLogin: String!, $endCursor: String) {
  organization(login: $orgLogin) {
    samlIdentityProvider {
      ssoUrl
      externalIdentities(first: 100, after: $endCursor) {
        edges {
          node {
            guid
            samlIdentity {
              nameId
            }
            user {
              login
            }
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
        totalCount
      }
    }
  }
}`

type GraplQLUsersFromGithubOrgSaml struct {
	Data struct {
		Organization struct {
			SamlIdentityProvider struct {
				ExternalIdentities struct {
					Edges []struct {
						Node struct {
							Guid         string
							SamlIdentity struct {
								NameId string
							}
							User struct {
								Login string
							}
						}
					}
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					} `json:"pageInfo"`
					TotalCount int `json:"totalCount"`
				} `json:"externalIdentities"`
			}
		}
	}
	Errors []struct {
		Path []struct {
			Query string `json:"query"`
		} `json:"path"`
		Extensions struct {
			Code         string
			ErrorMessage string
		} `json:"extensions"`
		Message string
	} `json:"errors"`
}

func (p *UserSyncPluginFromGithubSaml) loadUsersFromGithubOrgSaml() (map[string]*entity.User, error) {
	users := make(map[string]*entity.User)

	variables := make(map[string]interface{})
	variables["orgLogin"] = config.Config.GithubAppOrganization
	variables["endCursor"] = nil

	hasNextPage := true
	count := 0
	for hasNextPage {
		data, err := p.client.QueryGraphQLAPI(listUsersFromGithubOrgSaml, variables)
		var gResult GraplQLUsersFromGithubOrgSaml

		// parse first page
		err = json.Unmarshal(data, &gResult)
		if err != nil {
			return users, err
		}
		if len(gResult.Errors) > 0 {
			return users, fmt.Errorf("Graphql error: %v", gResult.Errors[0].Message)
		}

		for _, c := range gResult.Data.Organization.SamlIdentityProvider.ExternalIdentities.Edges {
			user := &entity.User{}
			user.ApiVersion = "v1"
			user.Kind = "User"
			user.Metadata.Name = c.Node.SamlIdentity.NameId
			user.Data.GithubID = c.Node.User.Login

			users[c.Node.SamlIdentity.NameId] = user
		}

		hasNextPage = gResult.Data.Organization.SamlIdentityProvider.ExternalIdentities.PageInfo.HasNextPage
		variables["endCursor"] = gResult.Data.Organization.SamlIdentityProvider.ExternalIdentities.PageInfo.EndCursor

		count++
		// sanity check to avoid loops
		if count > 100 {
			break
		}
	}

	return users, nil
}
