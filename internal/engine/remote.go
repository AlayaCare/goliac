package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/github"
	"github.com/Alayacare/goliac/internal/observability"
	"github.com/gosimple/slug"
	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

const FORLOOP_STOP = 100

type GoliacRemoteResources interface {
	Teams(ctx context.Context, current bool) map[string]*GithubTeam
}

/*
 * GoliacRemote
 * This interface is used to load the goliac organization from a Github
 * and mount it in memory
 */
type GoliacRemote interface {
	// Load from a github repository. continueOnError is used for scaffolding
	Load(ctx context.Context, continueOnError bool) error

	// Flush all assets from the cache
	FlushCache()

	// Flush only the users, and teams from the cache
	FlushCacheUsersTeamsOnly()

	Users(ctx context.Context) map[string]string // key is the login, value is the role (member, admin)
	TeamSlugByName(ctx context.Context) map[string]string
	Teams(ctx context.Context, current bool) map[string]*GithubTeam             // the key is the team slug
	Repositories(ctx context.Context) map[string]*GithubRepository              // the key is the repository name
	TeamRepositories(ctx context.Context) map[string]map[string]*GithubTeamRepo // key is team slug, second key is repo name
	RuleSets(ctx context.Context) map[string]*GithubRuleSet
	AppIds(ctx context.Context) map[string]int

	IsEnterprise() bool // check if we are on an Enterprise version, or if we are on GHES 3.11+

	CountAssets(ctx context.Context) (int, error)                      // return the number of (some) assets that will be loaded (to be used with the RemoteObservability/progress bar)
	SetRemoteObservability(feedback observability.RemoteObservability) // if you want to get feedback on the loading process
}

type GoliacRemoteExecutor interface {
	GoliacRemote
	ReconciliatorExecutor
}

type GithubRepository struct {
	Name           string
	Id             int
	RefId          string
	BoolProperties map[string]bool           // archived, private, allow_auto_merge, delete_branch_on_merge, allow_update_branch
	ExternalUsers  map[string]string         // [githubid]permission
	InternalUsers  map[string]string         // [githubid]permission
	RuleSets       map[string]*GithubRuleSet // [name]ruleset
}

type GithubTeam struct {
	Name        string
	Id          int
	Slug        string
	Members     []string // user login, aka githubid
	Maintainers []string // user login (that are not in the Members array)
	ParentTeam  *int
}

type GithubTeamRepo struct {
	Name       string // repository name
	Permission string // possible values: ADMIN, MAINTAIN, WRITE, TRIAGE, READ
}

type GoliacRemoteImpl struct {
	client                github.GitHubClient
	users                 map[string]string
	repositories          map[string]*GithubRepository
	repositoriesByRefId   map[string]*GithubRepository
	teams                 map[string]*GithubTeam
	teamRepos             map[string]map[string]*GithubTeamRepo
	teamSlugByName        map[string]string
	rulesets              map[string]*GithubRuleSet
	appIds                map[string]int
	ttlExpireUsers        time.Time
	ttlExpireRepositories time.Time
	ttlExpireTeams        time.Time
	ttlExpireTeamsRepos   time.Time
	ttlExpireRulesets     time.Time
	ttlExpireAppIds       time.Time
	isEnterprise          bool
	feedback              observability.RemoteObservability
	loadTeamsMutex        sync.Mutex
}

type GHESInfo struct {
	InstalledVersion string `json:"installed_version"`
}

func getGHESVersion(ctx context.Context, client github.GitHubClient) (*GHESInfo, error) {
	body, err := client.CallRestAPI(ctx, "/api/v3", "", "GET", nil)
	if err != nil {
		return nil, err
	}

	var info GHESInfo
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, fmt.Errorf("not able to get github org information: %v", err)
	}

	return &info, nil
}

const getAssets = `
query getAssets($orgLogin: String!) {
	organization(login: $orgLogin) {
		repositories{
			totalCount
		}
		teams {
			totalCount
		}
	    membersWithRole {
    		totalCount
    	}
		samlIdentityProvider {
			externalIdentities {
				totalCount
			}
		}
	}
}
`

type GraplQLAssets struct {
	Data struct {
		Organization struct {
			Repositories struct {
				TotalCount int `json:"totalCount"`
			} `json:"repositories"`
			Teams struct {
				TotalCount int `json:"totalCount"`
			} `json:"teams"`
			MembersWithRole struct {
				TotalCount int `json:"totalCount"`
			} `json:"membersWithRole"`
			SamlIdentityProvider struct {
				ExternalIdentities struct {
					TotalCount int `json:"totalCount"`
				} `json:"externalIdentities"`
			} `json:"samlIdentityProvider"`
		} `json:"organization"`
	}
	Errors []struct {
		Path       []interface{} `json:"path"`
		Extensions struct {
			Code         string
			ErrorMessage string
		} `json:"extensions"`
		Message string
	} `json:"errors"`
}

func (g *GoliacRemoteImpl) CountAssets(ctx context.Context) (int, error) {
	variables := make(map[string]interface{})
	variables["orgLogin"] = config.Config.GithubAppOrganization

	data, err := g.client.QueryGraphQLAPI(ctx, getAssets, variables)
	if err != nil {
		return 0, err
	}
	var gResult GraplQLAssets

	// parse first page
	err = json.Unmarshal(data, &gResult)
	if err != nil {
		return 0, err
	}
	if len(gResult.Errors) > 0 {
		return 0, fmt.Errorf("graphql error on CountAssets: %v (%v)", gResult.Errors[0].Message, gResult.Errors[0].Path)
	}
	return 2*gResult.Data.Organization.Repositories.TotalCount + // we multiply by 2 because we have the repositories, and the teams per repostiory to fetch
		2*gResult.Data.Organization.Teams.TotalCount + // we multiply by 2 because we have the teams and the members per team to fetch
		gResult.Data.Organization.MembersWithRole.TotalCount +
		gResult.Data.Organization.SamlIdentityProvider.ExternalIdentities.TotalCount, nil
}

func (g *GoliacRemoteImpl) SetRemoteObservability(feedback observability.RemoteObservability) {
	g.feedback = feedback
}

type OrgInfo struct {
	TwoFactorRequirementEnabled bool `json:"two_factor_requirement_enabled"`
	Plan                        struct {
		Name string `json:"name"` // enterprise
	} `json:"plan"`
}

func getOrgInfo(ctx context.Context, orgname string, client github.GitHubClient) (*OrgInfo, error) {
	body, err := client.CallRestAPI(ctx, "/orgs/"+orgname, "", "GET", nil)
	if err != nil {
		return nil, err
	}

	var info OrgInfo
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, fmt.Errorf("not able to get github org information: %v", err)
	}

	return &info, nil
}

func isEnterprise(ctx context.Context, orgname string, client github.GitHubClient) bool {
	// are we on Github Enteprise Server
	if ghesInfo, err := getGHESVersion(ctx, client); err == nil {
		logrus.Debugf("GHES versiob: %s", ghesInfo.InstalledVersion)
		version3_11, err := version.NewVersion("3.11")
		if err != nil {
			return false
		}
		ghesVersion, err := version.NewVersion(ghesInfo.InstalledVersion)
		if err != nil {
			return false
		}
		if ghesVersion.GreaterThanOrEqual(version3_11) {
			return true
		}
	} else if info, err := getOrgInfo(ctx, orgname, client); err == nil {
		logrus.Debugf("Organization plan: %s", info.Plan.Name)
		if info.Plan.Name == "enterprise" {
			return true
		}
	}
	return false
}

func NewGoliacRemoteImpl(client github.GitHubClient) *GoliacRemoteImpl {
	ctx := context.Background()
	return &GoliacRemoteImpl{
		client:                client,
		users:                 make(map[string]string),
		repositories:          make(map[string]*GithubRepository),
		repositoriesByRefId:   make(map[string]*GithubRepository),
		teams:                 make(map[string]*GithubTeam),
		teamRepos:             make(map[string]map[string]*GithubTeamRepo),
		teamSlugByName:        make(map[string]string),
		rulesets:              make(map[string]*GithubRuleSet),
		appIds:                make(map[string]int),
		ttlExpireUsers:        time.Now(),
		ttlExpireRepositories: time.Now(),
		ttlExpireTeams:        time.Now(),
		ttlExpireTeamsRepos:   time.Now(),
		ttlExpireRulesets:     time.Now(),
		ttlExpireAppIds:       time.Now(),
		isEnterprise:          isEnterprise(ctx, config.Config.GithubAppOrganization, client),
		feedback:              nil,
	}
}

func (g *GoliacRemoteImpl) IsEnterprise() bool {
	return g.isEnterprise
}

func (g *GoliacRemoteImpl) FlushCacheUsersTeamsOnly() {
	g.ttlExpireUsers = time.Now()
	g.ttlExpireTeams = time.Now()
}

func (g *GoliacRemoteImpl) FlushCache() {
	g.ttlExpireUsers = time.Now()
	g.ttlExpireRepositories = time.Now()
	g.ttlExpireTeams = time.Now()
	g.ttlExpireTeamsRepos = time.Now()
	g.ttlExpireRulesets = time.Now()
	g.ttlExpireAppIds = time.Now()
}

func (g *GoliacRemoteImpl) RuleSets(ctx context.Context) map[string]*GithubRuleSet {
	if time.Now().After(g.ttlExpireRulesets) {
		rulesets, err := g.loadRulesets(ctx)
		if err == nil {
			g.rulesets = rulesets
			g.ttlExpireRulesets = time.Now().Add(time.Duration(config.Config.GithubCacheTTL) * time.Second)
		}
	}
	return g.rulesets
}

func (g *GoliacRemoteImpl) AppIds(ctx context.Context) map[string]int {
	if time.Now().After(g.ttlExpireAppIds) {
		appIds, err := g.loadAppIds(ctx)
		if err == nil {
			g.appIds = appIds
			g.ttlExpireAppIds = time.Now().Add(time.Duration(config.Config.GithubCacheTTL) * time.Second)
		}
	}
	return g.appIds
}

func (g *GoliacRemoteImpl) Users(ctx context.Context) map[string]string {
	if time.Now().After(g.ttlExpireUsers) {
		users, err := g.loadOrgUsers(ctx)
		if err == nil {
			g.users = users
			g.ttlExpireUsers = time.Now().Add(time.Duration(config.Config.GithubCacheTTL) * time.Second)
		}
	}
	return g.users
}

func (g *GoliacRemoteImpl) TeamSlugByName(ctx context.Context) map[string]string {
	if time.Now().After(g.ttlExpireTeams) {
		teams, teamSlugByName, err := g.loadTeams(ctx)
		if err == nil {
			g.teams = teams
			g.teamSlugByName = teamSlugByName
			g.ttlExpireTeams = time.Now().Add(time.Duration(config.Config.GithubCacheTTL) * time.Second)
		}
	}
	return g.teamSlugByName
}

/*
Used to get the teams (and load it if needed)
if current is true, it will return the current in memory teams without checking the TTL (useful for the UI)
*/
func (g *GoliacRemoteImpl) Teams(ctx context.Context, current bool) map[string]*GithubTeam {
	if current && len(g.teams) > 0 {
		return g.teams
	}
	g.loadTeamsMutex.Lock()
	defer g.loadTeamsMutex.Unlock()
	if time.Now().After(g.ttlExpireTeams) {
		teams, teamSlugByName, err := g.loadTeams(ctx)
		if err == nil {
			g.teams = teams
			g.teamSlugByName = teamSlugByName
			g.ttlExpireTeams = time.Now().Add(time.Duration(config.Config.GithubCacheTTL) * time.Second)
		}
	}
	return g.teams
}

func (g *GoliacRemoteImpl) Repositories(ctx context.Context) map[string]*GithubRepository {
	if time.Now().After(g.ttlExpireRepositories) {
		repositories, repositoriesByRefIds, err := g.loadRepositories(ctx)
		if err == nil {
			g.repositories = repositories
			g.repositoriesByRefId = repositoriesByRefIds
			g.ttlExpireRepositories = time.Now().Add(time.Duration(config.Config.GithubCacheTTL) * time.Second)
		}
	}
	return g.repositories
}

func (g *GoliacRemoteImpl) TeamRepositories(ctx context.Context) map[string]map[string]*GithubTeamRepo {
	if time.Now().After(g.ttlExpireTeamsRepos) {
		if config.Config.GithubConcurrentThreads <= 1 {
			teamsrepos, err := g.loadTeamReposNonConcurrently(ctx)
			if err == nil {
				g.teamRepos = teamsrepos
				g.ttlExpireTeamsRepos = time.Now().Add(time.Duration(config.Config.GithubCacheTTL) * time.Second)
			}
		} else {
			teamsrepos, err := g.loadTeamReposConcurrently(ctx, config.Config.GithubConcurrentThreads)
			if err == nil {
				g.teamRepos = teamsrepos
				g.ttlExpireTeamsRepos = time.Now().Add(time.Duration(config.Config.GithubCacheTTL) * time.Second)
			}
		}
	}
	return g.teamRepos
}

const listAllOrgMembers = `
query listAllReposInOrg($orgLogin: String!, $endCursor: String) {
    organization(login: $orgLogin) {
		membersWithRole(first: 100, after: $endCursor) {
		  edges {
            node {
              login
            }
            role
          }
          pageInfo {
            hasNextPage
            endCursor
          }
          totalCount
        }
    }
}
`

type GraplQLUsers struct {
	Data struct {
		Organization struct {
			MembersWithRole struct {
				Edges []struct {
					Node struct {
						Login string
					} `json:"node"`
					Role string `json:"role"`
				} `json:"edges"`
				PageInfo struct {
					HasNextPage bool
					EndCursor   string
				} `json:"pageInfo"`
				TotalCount int `json:"totalCount"`
			} `json:"membersWithRole"`
		}
	}
	Errors []struct {
		Path       []interface{} `json:"path"`
		Extensions struct {
			Code         string
			ErrorMessage string
		} `json:"extensions"`
		Message string
	} `json:"errors"`
}

/*
loadOrgUsers returns
map[githubid]permission (role)
role can be 'ADMIN', 'MEMBER'
*/
func (g *GoliacRemoteImpl) loadOrgUsers(ctx context.Context) (map[string]string, error) {
	logrus.Debug("loading orgUsers")
	users := make(map[string]string)

	variables := make(map[string]interface{})
	variables["orgLogin"] = config.Config.GithubAppOrganization
	variables["endCursor"] = nil

	hasNextPage := true
	count := 0
	for hasNextPage {
		data, err := g.client.QueryGraphQLAPI(ctx, listAllOrgMembers, variables)
		if err != nil {
			return users, err
		}
		var gResult GraplQLUsers

		// parse first page
		err = json.Unmarshal(data, &gResult)
		if err != nil {
			return users, err
		}
		if len(gResult.Errors) > 0 {
			return users, fmt.Errorf("graphql error on loadOrgUsers: %v (%v)", gResult.Errors[0].Message, gResult.Errors[0].Path)
		}

		for _, c := range gResult.Data.Organization.MembersWithRole.Edges {
			users[c.Node.Login] = c.Role
		}

		if g.feedback != nil {
			g.feedback.LoadingAsset("users", len(gResult.Data.Organization.MembersWithRole.Edges))
		}

		hasNextPage = gResult.Data.Organization.MembersWithRole.PageInfo.HasNextPage
		variables["endCursor"] = gResult.Data.Organization.MembersWithRole.PageInfo.EndCursor

		count++
		// sanity check to avoid loops
		if count > FORLOOP_STOP {
			break
		}
	}

	return users, nil
}

const listAllReposInOrg = `
query listAllReposInOrg($orgLogin: String!, $endCursor: String) {
    organization(login: $orgLogin) {
      repositories(first: 10, after: $endCursor) {
        nodes {
          name
		  id
		  databaseId
          isArchived
          isPrivate
		  autoMergeAllowed
          deleteBranchOnMerge
          allowUpdateBranch
          directCollaborators: collaborators(affiliation: DIRECT, first: 100) {
            edges {
              node {
                login
              }
              permission
            }
          }
          outsideCollaborators: collaborators(affiliation: OUTSIDE, first: 100) {
            edges {
              node {
                login
              }
              permission
            }
          }
          rulesets(first: 20) {
            nodes {
              databaseId
              source {
                ... on Repository {
				  name
                }
              }
              name
              target
              enforcement
              conditions {
                refName {
                  include
                  exclude
                }
              }
              rules(first:20) {
                nodes {
                  parameters {
                    ... on PullRequestParameters {
                      dismissStaleReviewsOnPush
                      requireCodeOwnerReview
                      requiredApprovingReviewCount
                      requiredReviewThreadResolution
                      requireLastPushApproval
                    }
                  }
                  type
                }
              }
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
`

type GraplQLRepositories struct {
	Data struct {
		Organization struct {
			Repositories struct {
				Nodes []struct {
					Name                string
					Id                  string
					DatabaseId          int
					IsArchived          bool
					IsPrivate           bool
					AutoMergeAllowed    bool
					DeleteBranchOnMerge bool
					AllowUpdateBranch   bool
					DirectCollaborators struct {
						Edges []struct {
							Node struct {
								Login string
							}
							Permission string
						}
					}
					OutsideCollaborators struct {
						Edges []struct {
							Node struct {
								Login string
							}
							Permission string
						}
					}
					Rulesets struct {
						Nodes []GraphQLGithubRuleSet
					}
				} `json:"nodes"`
				PageInfo struct {
					HasNextPage bool
					EndCursor   string
				} `json:"pageInfo"`
				TotalCount int `json:"totalCount"`
			} `json:"repositories"`
		}
	}
	Errors []struct {
		Path       []interface{} `json:"path"`
		Extensions struct {
			Code         string
			ErrorMessage string
		} `json:"extensions"`
		Message string
	} `json:"errors"`
}

func (g *GoliacRemoteImpl) loadRepositories(ctx context.Context) (map[string]*GithubRepository, map[string]*GithubRepository, error) {
	logrus.Debug("loading repositories")
	repositories := make(map[string]*GithubRepository)
	repositoriesByRefId := make(map[string]*GithubRepository)

	variables := make(map[string]interface{})
	variables["orgLogin"] = config.Config.GithubAppOrganization
	variables["endCursor"] = nil

	var retErr error
	hasNextPage := true
	count := 0
	for hasNextPage {
		data, err := g.client.QueryGraphQLAPI(ctx, listAllReposInOrg, variables)
		if err != nil {
			return repositories, repositoriesByRefId, err
		}
		var gResult GraplQLRepositories

		// parse first page
		err = json.Unmarshal(data, &gResult)
		if err != nil {
			return repositories, repositoriesByRefId, err
		}
		if len(gResult.Errors) > 0 {
			retErr = fmt.Errorf("graphql error on loadRepositories: %v (%v)", gResult.Errors[0].Message, gResult.Errors[0].Path)
		}

		for _, c := range gResult.Data.Organization.Repositories.Nodes {
			repo := &GithubRepository{
				Name:  c.Name,
				Id:    c.DatabaseId,
				RefId: c.Id,
				BoolProperties: map[string]bool{
					"archived":               c.IsArchived,
					"private":                c.IsPrivate,
					"allow_auto_merge":       c.AutoMergeAllowed,
					"delete_branch_on_merge": c.DeleteBranchOnMerge,
					"allow_update_branch":    c.AllowUpdateBranch,
				},
				ExternalUsers: make(map[string]string),
				InternalUsers: make(map[string]string),
				RuleSets:      make(map[string]*GithubRuleSet),
			}
			for _, outsideCollaborator := range c.OutsideCollaborators.Edges {
				repo.ExternalUsers[outsideCollaborator.Node.Login] = outsideCollaborator.Permission
			}
			for _, internalCollaborator := range c.DirectCollaborators.Edges {
				repo.InternalUsers[internalCollaborator.Node.Login] = internalCollaborator.Permission
			}
			for _, ruleset := range c.Rulesets.Nodes {
				// if the source is the repository itself, it is not a organization ruleset
				// we add the ruleset
				if ruleset.Source.Name == c.Name {
					repo.RuleSets[ruleset.Name] = g.fromGraphQLToGithubRuleset(&ruleset)
				}
			}
			repositories[c.Name] = repo
			repositoriesByRefId[c.Id] = repo
		}

		if g.feedback != nil {
			g.feedback.LoadingAsset("repositories", len(gResult.Data.Organization.Repositories.Nodes))
		}

		hasNextPage = gResult.Data.Organization.Repositories.PageInfo.HasNextPage
		variables["endCursor"] = gResult.Data.Organization.Repositories.PageInfo.EndCursor

		count++
		// sanity check to avoid loops
		if count > FORLOOP_STOP {
			break
		}
	}

	return repositories, repositoriesByRefId, retErr
}

const listAllTeamsInOrg = `
query listAllTeamsInOrg($orgLogin: String!, $endCursor: String) {
    organization(login: $orgLogin) {
      teams(first: 100, after: $endCursor) {
        nodes {
          name
		  databaseId
          slug
		  parentTeam {
		    databaseId
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
`

type GraplQLTeams struct {
	Data struct {
		Organization struct {
			Teams struct {
				Nodes []struct {
					Name       string
					DatabaseId int `json:"databaseId"`
					Slug       string
					ParentTeam struct {
						DatabaseId int `json:"databaseId"`
					} `json:"parentTeam"`
				} `json:"nodes"`
				PageInfo struct {
					HasNextPage bool
					EndCursor   string
				} `json:"pageInfo"`
				TotalCount int `json:"totalCount"`
			} `json:"teams"`
		}
	}
	Errors []struct {
		Path       []interface{} `json:"path"`
		Extensions struct {
			Code         string
			ErrorMessage string
		} `json:"extensions"`
		Message string
	} `json:"errors"`
}

func (g *GoliacRemoteImpl) loadAppIds(ctx context.Context) (map[string]int, error) {
	logrus.Debug("loading appIds")
	appIds := map[string]int{}
	type Installation struct {
		TotalCount    int `json:"total_count"`
		Installations []struct {
			Id      int    `json:"id"`
			AppId   int    `json:"app_id"`
			Name    string `json:"name"`
			AppSlug string `json:"app_slug"`
		} `json:"installations"`
	}

	// https://docs.github.com/en/enterprise-cloud@latest/rest/orgs/orgs?apiVersion=2022-11-28#list-app-installations-for-an-organization
	body, err := g.client.CallRestAPI(ctx,
		fmt.Sprintf("/orgs/%s/installations", config.Config.GithubAppOrganization),
		"page=1&per_page=30",
		"GET",
		nil)

	if err != nil {
		return nil, fmt.Errorf("not able to list github apps: %v. %s", err, string(body))
	}

	var installations Installation
	err = json.Unmarshal(body, &installations)
	if err != nil {
		return nil, fmt.Errorf("not able to list github apps: %v", err)
	}

	for _, i := range installations.Installations {
		appIds[i.AppSlug] = i.AppId
	}

	if installations.TotalCount > 30 {
		// we need to paginate
		for i := 2; i <= (installations.TotalCount/30)+1; i++ {
			body, err := g.client.CallRestAPI(ctx,
				fmt.Sprintf("/orgs/%s/installations", config.Config.GithubAppOrganization),
				fmt.Sprintf("page=%d&per_page=30", i),
				"GET",
				nil)

			if err != nil {
				return nil, fmt.Errorf("not able to list next page github apps: %v. %s", err, string(body))
			}

			var installations Installation
			err = json.Unmarshal(body, &installations)
			if err != nil {
				return nil, fmt.Errorf("not able to list github apps: %v", err)
			}

			for _, i := range installations.Installations {
				appIds[i.AppSlug] = i.AppId
			}
		}
	}

	return appIds, nil
}

// Load from a github repository. continueOnError is used for scaffolding
func (g *GoliacRemoteImpl) Load(ctx context.Context, continueOnError bool) error {
	var retErr error

	if time.Now().After(g.ttlExpireAppIds) {
		appIds, err := g.loadAppIds(ctx)
		if err != nil {
			if !continueOnError {
				return err
			}
			logrus.Debugf("Error loading app ids: %v", err)
			retErr = fmt.Errorf("error loading app ids: %v", err)
		}
		g.appIds = appIds
		g.ttlExpireAppIds = time.Now().Add(time.Duration(config.Config.GithubCacheTTL) * time.Second)
	}

	g.loadTeamsMutex.Lock()
	if time.Now().After(g.ttlExpireTeams) {
		teams, teamSlugByName, err := g.loadTeams(ctx)
		if err != nil {
			if !continueOnError {
				g.loadTeamsMutex.Unlock()
				return err
			}
			logrus.Debugf("Error loading teams: %v", err)
			retErr = fmt.Errorf("error loading teams: %v", err)
		}
		g.teams = teams
		g.teamSlugByName = teamSlugByName
		g.ttlExpireTeams = time.Now().Add(time.Duration(config.Config.GithubCacheTTL) * time.Second)
	}
	g.loadTeamsMutex.Unlock()

	if time.Now().After(g.ttlExpireUsers) {
		users, err := g.loadOrgUsers(ctx)
		if err != nil {
			if !continueOnError {
				return err
			}
			logrus.Debugf("Error loading users: %v", err)
			retErr = fmt.Errorf("error loading users: %v", err)
		}
		g.users = users
		g.ttlExpireUsers = time.Now().Add(time.Duration(config.Config.GithubCacheTTL) * time.Second)
	}

	if time.Now().After(g.ttlExpireRepositories) {
		repositories, repositoriesByRefId, err := g.loadRepositories(ctx)
		if err != nil {
			if !continueOnError {
				return err
			}
			logrus.Debugf("Error loading repositories: %v", err)
			retErr = fmt.Errorf("error loading repositories: %v", err)
		}
		g.repositories = repositories
		g.repositoriesByRefId = repositoriesByRefId
		g.ttlExpireRepositories = time.Now().Add(time.Duration(config.Config.GithubCacheTTL) * time.Second)
	}

	// let's load the rulesets after the repositories because I need the repository refs
	if time.Now().After(g.ttlExpireRulesets) {
		rulesets, err := g.loadRulesets(ctx)
		if err != nil {
			if !continueOnError {
				return err
			}
			logrus.Debugf("Error loading rulesets: %v", err)
			retErr = fmt.Errorf("error loading rulesets: %v", err)
		}
		g.rulesets = rulesets
		g.ttlExpireRulesets = time.Now().Add(time.Duration(config.Config.GithubCacheTTL) * time.Second)
	}

	if time.Now().After(g.ttlExpireTeamsRepos) {
		if config.Config.GithubConcurrentThreads <= 1 {
			teamsrepos, err := g.loadTeamReposNonConcurrently(ctx)
			if err != nil {
				if !continueOnError {
					return err
				}
				logrus.Debugf("Error loading teams-repos: %v", err)
				retErr = fmt.Errorf("error loading teams-repos: %v", err)
			}
			g.teamRepos = teamsrepos
		} else {
			teamsrepos, err := g.loadTeamReposConcurrently(ctx, config.Config.GithubConcurrentThreads)
			if err != nil {
				if !continueOnError {
					return err
				}
				logrus.Debugf("Error loading teams-repos: %v", err)
				retErr = fmt.Errorf("error loading teams-repos: %v", err)
			}
			g.teamRepos = teamsrepos
		}
		g.ttlExpireTeamsRepos = time.Now().Add(time.Duration(config.Config.GithubCacheTTL) * time.Second)
	}

	logrus.Debugf("Nb remote users: %d", len(g.users))
	logrus.Debugf("Nb remote teams: %d", len(g.teams))
	logrus.Debugf("Nb remote repositories: %d", len(g.repositories))

	return retErr
}

func (g *GoliacRemoteImpl) loadTeamReposNonConcurrently(ctx context.Context) (map[string]map[string]*GithubTeamRepo, error) {
	logrus.Debug("loading teamReposNonConcurrently")
	teamRepos := make(map[string]map[string]*GithubTeamRepo)

	teamsPerRepo := make(map[string]map[string]*GithubTeamRepo)
	for repository := range g.repositories {
		repos, err := g.loadTeamRepos(ctx, repository)
		if err != nil {
			return teamRepos, err
		}
		if g.feedback != nil {
			g.feedback.LoadingAsset("teams_repos", 1)
		}
		teamsPerRepo[repository] = repos
	}

	// we have all the teams per repo, now we need to invert the map
	for repository, repos := range teamsPerRepo {
		for team, repo := range repos {
			if _, ok := teamRepos[team]; ok {
				teamRepos[team][repository] = repo
			} else {
				teamRepos[team] = map[string]*GithubTeamRepo{repository: repo}
			}
		}
	}

	return teamRepos, nil
}

func (g *GoliacRemoteImpl) loadTeamReposConcurrently(ctx context.Context, maxGoroutines int64) (map[string]map[string]*GithubTeamRepo, error) {
	logrus.Debug("loading teamReposConcurrently")
	teamRepos := make(map[string]map[string]*GithubTeamRepo)

	teamsPerRepo := make(map[string]map[string]*GithubTeamRepo)

	var wg sync.WaitGroup
	var wg2 sync.WaitGroup

	// Create buffered channels
	reposChan := make(chan string, len(g.repositories))
	errChan := make(chan error, 1) // will hold the first error
	teamReposChan := make(chan struct {
		repoName string
		repos    map[string]*GithubTeamRepo
	}, len(g.repositories))

	// Create worker goroutines
	for i := int64(0); i < maxGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repoName := range reposChan {
				repos, err := g.loadTeamRepos(ctx, repoName)
				if err != nil {
					// Try to report the error
					select {
					case errChan <- err:
					default:
					}
					return
				}
				teamReposChan <- struct {
					repoName string
					repos    map[string]*GithubTeamRepo
				}{repoName, repos}
			}
		}()
	}

	// Send repositories to reposChan
	for repoName := range g.repositories {
		reposChan <- repoName
	}
	close(reposChan)

	wg2.Add(1)
	go func() {
		defer wg2.Done()
		for r := range teamReposChan {
			if g.feedback != nil {
				g.feedback.LoadingAsset("teams_repos", 1)
			}
			teamsPerRepo[r.repoName] = r.repos
		}
	}()

	// Wait for all goroutines to finish
	wg.Wait()
	close(teamReposChan)
	wg2.Wait()

	// Check if any goroutine returned an error
	select {
	case err := <-errChan:
		return teamRepos, err
	default:
		//nop
	}

	// we have all the teams per repo, now we need to invert the map
	for repository, repos := range teamsPerRepo {
		for team, repo := range repos {
			if _, ok := teamRepos[team]; ok {
				teamRepos[team][repository] = repo
			} else {
				teamRepos[team] = map[string]*GithubTeamRepo{repository: repo}
			}
		}
	}

	return teamRepos, nil
}

type TeamsRepoResponse struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
	Slug       string `json:"slug"`
}

/*
loadTeamRepos returns
map[teamSlug]repoinfo
*/
func (g *GoliacRemoteImpl) loadTeamRepos(ctx context.Context, repository string) (map[string]*GithubTeamRepo, error) {
	// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#list-repository-teams
	teamsrepo := make(map[string]*GithubTeamRepo)

	data, err := g.client.CallRestAPI(
		ctx,
		"/repos/"+config.Config.GithubAppOrganization+"/"+repository+"/teams",
		"",
		"GET",
		nil)
	if err != nil {
		return nil, fmt.Errorf("not able to list teams for repo %s: %v", repository, err)
	}

	var teams []TeamsRepoResponse
	err = json.Unmarshal(data, &teams)
	if err != nil {
		return nil, fmt.Errorf("not able to unmarshall teams for repo %s: %v", repository, err)
	}

	for _, t := range teams {
		permission := ""
		switch t.Permission {
		case "admin":
			permission = "ADMIN"
		case "push":
			permission = "WRITE"
		case "pull":
			permission = "READ"
		}
		teamsrepo[t.Slug] = &GithubTeamRepo{
			Name:       repository,
			Permission: permission,
		}
	}

	return teamsrepo, nil
}

const listAllTeamMembersInOrg = `
query listAllTeamMembersInOrg($orgLogin: String!, $teamSlug: String!, $endCursor: String) {
    organization(login: $orgLogin) {
      team(slug: $teamSlug) {
        members(first: 100, membership: IMMEDIATE, after: $endCursor) {
          edges {
            node {
              login
            }
            role
          }
          pageInfo {
            hasNextPage
            endCursor
          }
          totalCount
        }
      }
    }
  }
`

type GraplQLTeamMembers struct {
	Data struct {
		Organization struct {
			Team struct {
				Members struct {
					Edges []struct {
						Node struct {
							Login string
						}
						Role string
					} `json:"edges"`
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					} `json:"pageInfo"`
					TotalCount int `json:"totalCount"`
				} `json:"members"`
			} `json:"team"`
		}
	}
	Errors []struct {
		Path       []interface{} `json:"path"`
		Extensions struct {
			Code         string
			ErrorMessage string
		} `json:"extensions"`
		Message string
	} `json:"errors"`
}

func (g *GoliacRemoteImpl) loadTeams(ctx context.Context) (map[string]*GithubTeam, map[string]string, error) {
	logrus.Debug("loading teams")
	teams := make(map[string]*GithubTeam)
	teamSlugByName := make(map[string]string)

	variables := make(map[string]interface{})
	variables["orgLogin"] = config.Config.GithubAppOrganization
	variables["endCursor"] = nil

	hasNextPage := true
	count := 0
	for hasNextPage {
		data, err := g.client.QueryGraphQLAPI(ctx, listAllTeamsInOrg, variables)
		if err != nil {
			return teams, teamSlugByName, err
		}
		var gResult GraplQLTeams

		// parse first page
		err = json.Unmarshal(data, &gResult)
		if err != nil {
			return teams, teamSlugByName, err
		}
		if len(gResult.Errors) > 0 {
			return teams, teamSlugByName, fmt.Errorf("graphql error on loadTeams: %v (%v)", gResult.Errors[0].Message, gResult.Errors[0].Path)
		}

		for _, c := range gResult.Data.Organization.Teams.Nodes {
			team := GithubTeam{
				Name: c.Name,
				Id:   c.DatabaseId,
				Slug: c.Slug,
			}
			if c.ParentTeam.DatabaseId != 0 {
				parentId := c.ParentTeam.DatabaseId
				team.ParentTeam = &parentId
			}
			teams[c.Slug] = &team
			teamSlugByName[c.Name] = c.Slug
		}

		if g.feedback != nil {
			g.feedback.LoadingAsset("teams", len(gResult.Data.Organization.Teams.Nodes))
		}

		hasNextPage = gResult.Data.Organization.Teams.PageInfo.HasNextPage
		variables["endCursor"] = gResult.Data.Organization.Teams.PageInfo.EndCursor

		count++
		// sanity check to avoid loops
		if count > FORLOOP_STOP {
			break
		}
	}

	// load team's members
	if config.Config.GithubConcurrentThreads <= 1 {
		for _, t := range teams {
			err := g.loadTeamsMembers(ctx, t)
			if err != nil {
				return teams, teamSlugByName, err
			}
			if g.feedback != nil {
				g.feedback.LoadingAsset("teams_members", 1)
			}
		}
	} else {
		var wg sync.WaitGroup

		// Create buffered channels
		teamsChan := make(chan *GithubTeam, len(teams))
		errChan := make(chan error, 1) // will hold the first error

		// Create worker goroutines
		for i := int64(0); i < config.Config.GithubConcurrentThreads; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for t := range teamsChan {
					err := g.loadTeamsMembers(ctx, t)
					if err != nil {
						// Try to report the error
						select {
						case errChan <- err:
						default:
						}
						return
					}
					if g.feedback != nil {
						g.feedback.LoadingAsset("teams_members", 1)
					}
				}
			}()
		}

		// Send teams to teamsChan
		for _, t := range teams {
			teamsChan <- t
		}
		close(teamsChan)

		// Wait for all goroutines to finish
		wg.Wait()

		// Check if any goroutine returned an error
		select {
		case err := <-errChan:
			return teams, teamSlugByName, err
		default:
			//nop
		}
	}

	return teams, teamSlugByName, nil
}

func (g *GoliacRemoteImpl) loadTeamsMembers(ctx context.Context, t *GithubTeam) error {
	variables := make(map[string]interface{})
	variables["orgLogin"] = config.Config.GithubAppOrganization
	variables["endCursor"] = nil
	variables["teamSlug"] = t.Slug

	hasNextPage := true
	count := 0
	for hasNextPage {
		data, err := g.client.QueryGraphQLAPI(ctx, listAllTeamMembersInOrg, variables)
		if err != nil {
			return err
		}
		var gResult GraplQLTeamMembers

		// parse first page
		err = json.Unmarshal(data, &gResult)
		if err != nil {
			return err
		}
		if len(gResult.Errors) > 0 {
			return fmt.Errorf("graphql error on loadTeams members: %v (%v)", gResult.Errors[0].Message, gResult.Errors[0].Path)
		}

		for _, c := range gResult.Data.Organization.Team.Members.Edges {
			if c.Role == "MAINTAINER" {
				t.Maintainers = append(t.Maintainers, c.Node.Login)
			} else {
				t.Members = append(t.Members, c.Node.Login)
			}
		}

		hasNextPage = gResult.Data.Organization.Team.Members.PageInfo.HasNextPage
		variables["endCursor"] = gResult.Data.Organization.Team.Members.PageInfo.EndCursor

		count++
		// sanity check to avoid loops
		if count > FORLOOP_STOP {
			break
		}
	}
	return nil
}

const listRulesets = `
query listRulesets ($orgLogin: String!) { 
	organization(login: $orgLogin) {
	  rulesets(first: 100) { 
		nodes {
		  databaseId
		  name
		  target
		  enforcement
		  bypassActors(first:100) {
			app:nodes {
			  actor {
				... on App {
					databaseId
					name
				}
			  }
			  bypassMode
			}
		  }
		  conditions {
			refName {
			  include
			  exclude
			}
			repositoryName {
			  exclude
			  include
			}
			repositoryId {
				repositoryIds
			}
		  }
		  rules(first:100) {
			nodes {
				parameters {
					... on PullRequestParameters {
						dismissStaleReviewsOnPush
						requireCodeOwnerReview
						requiredApprovingReviewCount
						requiredReviewThreadResolution
						requireLastPushApproval
					}
				}
				type
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
`

type GithubRuleSetApp struct {
	Actor struct {
		DatabaseId int
		Name       string
	}
	BypassMode string // ALWAYS, PULL_REQUEST
}

type GithubRuleSetRuleStatusCheck struct {
	Context       string
	IntegrationId int
}

type GithubRuleSetRule struct {
	Parameters struct {
		// PullRequestParameters
		DismissStaleReviewsOnPush      bool
		RequireCodeOwnerReview         bool
		RequiredApprovingReviewCount   int
		RequiredReviewThreadResolution bool
		RequireLastPushApproval        bool

		// RequiredStatusChecksParameters
		RequiredStatusChecks             []GithubRuleSetRuleStatusCheck
		StrictRequiredStatusChecksPolicy bool
	}
	ID   int
	Type string // CREATION, UPDATE, DELETION, REQUIRED_LINEAR_HISTORY, REQUIRED_DEPLOYMENTS, REQUIRED_SIGNATURES, PULL_REQUEST, REQUIRED_STATUS_CHECKS, NON_FAST_FORWARD, COMMIT_MESSAGE_PATTERN, COMMIT_AUTHOR_EMAIL_PATTERN, COMMITTER_EMAIL_PATTERN, BRANCH_NAME_PATTERN, TAG_NAME_PATTERN
}

type GraphQLGithubRuleSet struct {
	DatabaseId int
	Source     struct {
		Name string
	}
	Name         string
	Target       string // BRANCH, TAG
	Enforcement  string // DISABLED, ACTIVE, EVALUATE
	BypassActors struct {
		App []GithubRuleSetApp
	}
	Conditions struct {
		RefName struct { // target branches
			Include []string // ~DEFAULT_BRANCH, ~ALL,
			Exclude []string
		}
		RepositoryName struct { // regex
			Include   []string
			Exclude   []string
			Protected bool
		}
		RepositoryId struct { // per repo
			RepositoryIds []string
		}
	}
	Rules struct {
		Nodes []GithubRuleSetRule
	}
}

type GraplQLRuleSets struct {
	Data struct {
		Organization struct {
			Rulesets struct {
				Nodes    []GraphQLGithubRuleSet
				PageInfo struct {
					HasNextPage bool
					EndCursor   string
				} `json:"pageInfo"`
				TotalCount int `json:"totalCount"`
			} `json:"rulesets"`
		}
	}
	Errors []struct {
		Path       []string `json:"path"`
		Extensions struct {
			Code         string
			ErrorMessage string
		} `json:"extensions"`
		Message string
	} `json:"errors"`
}

type GithubRuleSet struct {
	Name        string
	Id          int               // for tracking purpose
	Enforcement string            // disabled, active, evaluate
	BypassApps  map[string]string // appname, mode (always, pull_request)

	OnInclude []string // ~DEFAULT_BRANCH, ~ALL, branch_name, ...
	OnExclude []string //  branch_name, ...

	Rules map[string]entity.RuleSetParameters

	Repositories []string // only used for organization rulesets
}

func (g *GoliacRemoteImpl) fromGraphQLToGithubRuleset(src *GraphQLGithubRuleSet) *GithubRuleSet {
	ruleset := GithubRuleSet{
		Name:         src.Name,
		Id:           src.DatabaseId,
		Enforcement:  strings.ToLower(src.Enforcement),
		BypassApps:   map[string]string{},
		OnInclude:    src.Conditions.RefName.Include,
		OnExclude:    src.Conditions.RefName.Exclude,
		Rules:        map[string]entity.RuleSetParameters{},
		Repositories: []string{},
	}
	for _, b := range src.BypassActors.App {
		ruleset.BypassApps[b.Actor.Name] = strings.ToLower(b.BypassMode)
	}

	for _, r := range src.Rules.Nodes {
		rule := entity.RuleSetParameters{
			DismissStaleReviewsOnPush:        r.Parameters.DismissStaleReviewsOnPush,
			RequireCodeOwnerReview:           r.Parameters.RequireCodeOwnerReview,
			RequiredApprovingReviewCount:     r.Parameters.RequiredApprovingReviewCount,
			RequiredReviewThreadResolution:   r.Parameters.RequiredReviewThreadResolution,
			RequireLastPushApproval:          r.Parameters.RequireLastPushApproval,
			StrictRequiredStatusChecksPolicy: r.Parameters.StrictRequiredStatusChecksPolicy,
		}
		for _, s := range r.Parameters.RequiredStatusChecks {
			rule.RequiredStatusChecks = append(rule.RequiredStatusChecks, s.Context)
		}
		ruleset.Rules[strings.ToLower(r.Type)] = rule
	}

	for _, r := range src.Conditions.RepositoryId.RepositoryIds {
		if repo, ok := g.repositoriesByRefId[r]; ok {
			ruleset.Repositories = append(ruleset.Repositories, repo.Name)
		}
	}

	return &ruleset
}

func (g *GoliacRemoteImpl) loadRulesets(ctx context.Context) (map[string]*GithubRuleSet, error) {
	logrus.Debug("loading rulesets")
	variables := make(map[string]interface{})
	variables["orgLogin"] = config.Config.GithubAppOrganization
	variables["endCursor"] = nil

	rulesets := make(map[string]*GithubRuleSet)

	hasNextPage := true
	count := 0
	for hasNextPage {
		data, err := g.client.QueryGraphQLAPI(ctx, listRulesets, variables)
		if err != nil {
			return rulesets, err
		}
		var gResult GraplQLRuleSets

		// parse first page
		err = json.Unmarshal(data, &gResult)
		if err != nil {
			return rulesets, err
		}
		if len(gResult.Errors) > 0 {
			return rulesets, fmt.Errorf("graphql error on loadRulesets: %v (%v)", gResult.Errors[0].Message, gResult.Errors[0].Path)
		}

		for _, c := range gResult.Data.Organization.Rulesets.Nodes {
			rulesets[c.Name] = g.fromGraphQLToGithubRuleset(&c)
		}

		hasNextPage = gResult.Data.Organization.Rulesets.PageInfo.HasNextPage
		variables["endCursor"] = gResult.Data.Organization.Rulesets.PageInfo.EndCursor

		count++
		// sanity check to avoid loops
		if count > FORLOOP_STOP {
			break
		}
	}

	return rulesets, nil
}

func (g *GoliacRemoteImpl) prepareRuleset(ruleset *GithubRuleSet) map[string]interface{} {
	bypassActors := make([]map[string]interface{}, 0)

	for appname, mode := range ruleset.BypassApps {
		// let's find the app id based on the app slug name
		if appId, ok := g.appIds[appname]; ok {
			bypassActor := map[string]interface{}{
				"actor_id":    appId,
				"actor_type":  "Integration",
				"bypass_mode": mode,
			}
			bypassActors = append(bypassActors, bypassActor)
		}
	}

	repoIds := []int{}
	for _, r := range ruleset.Repositories {
		if rid, ok := g.repositories[r]; ok {
			repoIds = append(repoIds, rid.Id)
		}
	}
	include := ruleset.OnInclude
	if include == nil {
		include = []string{}
	}
	exclude := ruleset.OnExclude
	if exclude == nil {
		exclude = []string{}
	}
	conditions := map[string]interface{}{
		"ref_name": map[string]interface{}{
			"include": include,
			"exclude": exclude,
		},
		"repository_id": map[string]interface{}{
			"repository_ids": repoIds,
		},
	}

	rules := make([]map[string]interface{}, 0)
	for ruletype, rule := range ruleset.Rules {
		switch ruletype {
		case "required_signatures":
			rules = append(rules, map[string]interface{}{
				"type": "required_signatures",
			})
		case "creation":
			rules = append(rules, map[string]interface{}{
				"type": "creation",
			})
		case "update":
			rules = append(rules, map[string]interface{}{
				"type": "update",
			})
		case "deletion":
			rules = append(rules, map[string]interface{}{
				"type": "deletion",
			})
		case "pull_request":
			rules = append(rules, map[string]interface{}{
				"type": "pull_request",
				"parameters": map[string]interface{}{
					"dismiss_stale_reviews_on_push":     rule.DismissStaleReviewsOnPush,
					"require_code_owner_review":         rule.RequireCodeOwnerReview,
					"required_approving_review_count":   rule.RequiredApprovingReviewCount,
					"required_review_thread_resolution": rule.RequiredReviewThreadResolution,
					"require_last_push_approval":        rule.RequireLastPushApproval,
				},
			})
		case "required_status_checks":
			rules = append(rules, map[string]interface{}{
				"type": "required_status_checks",
				"parameters": map[string]interface{}{
					"required_status_checks":               rule.RequiredStatusChecks,
					"strict_required_status_checks_policy": rule.StrictRequiredStatusChecksPolicy,
				},
			})
		}
	}

	payload := map[string]interface{}{
		"name":          ruleset.Name,
		"target":        "branch",
		"enforcement":   ruleset.Enforcement,
		"bypass_actors": bypassActors,
		"conditions":    conditions,
		"rules":         rules,
	}
	return payload
}

func (g *GoliacRemoteImpl) AddRuleset(ctx context.Context, dryrun bool, ruleset *GithubRuleSet) {
	// add ruleset
	// https://docs.github.com/en/enterprise-cloud@latest/rest/orgs/rules?apiVersion=2022-11-28#create-an-organization-repository-ruleset

	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/rulesets", config.Config.GithubAppOrganization),
			"",
			"POST",
			g.prepareRuleset(ruleset),
		)
		if err != nil {
			logrus.Errorf("failed to add ruleset to org: %v. %s", err, string(body))
		}
	}

	g.rulesets[ruleset.Name] = ruleset
}

func (g *GoliacRemoteImpl) UpdateRuleset(ctx context.Context, dryrun bool, ruleset *GithubRuleSet) {
	// update ruleset
	// https://docs.github.com/en/enterprise-cloud@latest/rest/orgs/rules?apiVersion=2022-11-28#update-an-organization-repository-ruleset

	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/rulesets/%d", config.Config.GithubAppOrganization, ruleset.Id),
			"",
			"PUT",
			g.prepareRuleset(ruleset),
		)
		if err != nil {
			logrus.Errorf("failed to update ruleset %d to org: %v. %s", ruleset.Id, err, string(body))
		}
	}

	g.rulesets[ruleset.Name] = ruleset
}

func (g *GoliacRemoteImpl) DeleteRuleset(ctx context.Context, dryrun bool, rulesetid int) {
	// remove ruleset
	// https://docs.github.com/en/enterprise-cloud@latest/rest/orgs/rules?apiVersion=2022-11-28#delete-an-organization-repository-ruleset

	if !dryrun {
		_, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/rulesets/%d", config.Config.GithubAppOrganization, rulesetid),
			"",
			"DELETE",
			nil,
		)
		if err != nil {
			logrus.Errorf("failed to remove ruleset from org: %v", err)
		}
	}

	for _, r := range g.rulesets {
		if r.Id == rulesetid {
			delete(g.rulesets, r.Name)
			break
		}
	}
}

func (g *GoliacRemoteImpl) AddRepositoryRuleset(ctx context.Context, dryrun bool, reponame string, ruleset *GithubRuleSet) {
	// add repository ruleset
	// https://docs.github.com/en/rest/repos/rules?apiVersion=2022-11-28#create-a-repository-ruleset

	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/%s/rulesets", config.Config.GithubAppOrganization, reponame),
			"",
			"POST",
			g.prepareRuleset(ruleset),
		)
		if err != nil {
			logrus.Errorf("failed to add ruleset to repository: %v. %s", err, string(body))
		}
	}
	repo := g.repositories[reponame]
	if repo != nil {
		repo.RuleSets[ruleset.Name] = ruleset
	}
}

func (g *GoliacRemoteImpl) UpdateRepositoryRuleset(ctx context.Context, dryrun bool, reponame string, ruleset *GithubRuleSet) {
	// update repository ruleset
	// https://docs.github.com/en/rest/repos/rules?apiVersion=2022-11-28#update-a-repository-ruleset

	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/%s/rulesets/%d", config.Config.GithubAppOrganization, reponame, ruleset.Id),
			"",
			"PUT",
			g.prepareRuleset(ruleset),
		)
		if err != nil {
			logrus.Errorf("failed to update ruleset %d to repository: %v. %s", ruleset.Id, err, string(body))
		}
	}
	repo := g.repositories[reponame]
	if repo != nil {
		repo.RuleSets[ruleset.Name] = ruleset
	}
}

func (g *GoliacRemoteImpl) DeleteRepositoryRuleset(ctx context.Context, dryrun bool, reponame string, rulesetid int) {
	// remove repository ruleset
	// https://docs.github.com/en/rest/repos/rules?apiVersion=2022-11-28#delete-a-repository-ruleset

	if !dryrun {
		_, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/%s/rulesets/%d", config.Config.GithubAppOrganization, reponame, rulesetid),
			"",
			"DELETE",
			nil,
		)
		if err != nil {
			logrus.Errorf("failed to remove ruleset from repository: %v", err)
		}
	}

	repo := g.repositories[reponame]
	if repo != nil {
		for _, r := range repo.RuleSets {
			if r.Id == rulesetid {
				delete(repo.RuleSets, r.Name)
				break
			}
		}
	}
}

func (g *GoliacRemoteImpl) AddUserToOrg(ctx context.Context, dryrun bool, ghuserid string) {
	// add member
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#create-a-team
	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/memberships/%s", config.Config.GithubAppOrganization, ghuserid),
			"",
			"PUT",
			map[string]interface{}{"role": "member"},
		)
		if err != nil {
			logrus.Errorf("failed to add user to org: %v. %s", err, string(body))
		}
	}

	g.users[ghuserid] = ghuserid
}

func (g *GoliacRemoteImpl) RemoveUserFromOrg(ctx context.Context, dryrun bool, ghuserid string) {
	// remove member
	// https://docs.github.com/en/rest/orgs/members?apiVersion=2022-11-28#remove-organization-membership-for-a-user
	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/memberships/%s", config.Config.GithubAppOrganization, ghuserid),
			"",
			"DELETE",
			nil,
		)
		if err != nil {
			logrus.Errorf("failed to remove user from org: %v. %s", err, string(body))
		}
	}

	delete(g.users, ghuserid)
}

type CreateTeamResponse struct {
	Name string
	Slug string
}

func (g *GoliacRemoteImpl) CreateTeam(ctx context.Context, dryrun bool, teamname string, description string, parentTeam *int, members []string) {
	slugname := slug.Make(teamname)
	// create team
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#create-a-team
	if !dryrun {
		params := map[string]interface{}{
			"name":        teamname,
			"description": description,
			"privacy":     "closed",
		}
		if parentTeam != nil {
			params["parent_team_id"] = parentTeam
		}
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/teams", config.Config.GithubAppOrganization),
			"",
			"POST",
			params,
		)
		if err != nil {
			logrus.Errorf("failed to create team: %v. %s", err, string(body))
			return
		}
		var res CreateTeamResponse
		err = json.Unmarshal(body, &res)
		if err != nil {
			logrus.Errorf("failed to create team: %v", err)
			return
		}

		// add members
		for _, member := range members {
			// https://docs.github.com/en/rest/teams/members?apiVersion=2022-11-28#add-or-update-team-membership-for-a-user
			body, err := g.client.CallRestAPI(
				ctx,
				fmt.Sprintf("orgs/%s/teams/%s/memberships/%s", config.Config.GithubAppOrganization, res.Slug, member),
				"",
				"PUT",
				map[string]interface{}{"role": "member"},
			)
			if err != nil {
				logrus.Errorf("failed to create team: %v. %s", err, string(body))
				return
			}
		}
		slugname = res.Slug
	}

	g.teams[slugname] = &GithubTeam{
		Name:        teamname,
		Slug:        slugname,
		Members:     members,
		Maintainers: []string{},
	}
	g.teamSlugByName[teamname] = slugname
}

// role = member or maintainer (usually we use member)
func (g *GoliacRemoteImpl) UpdateTeamAddMember(ctx context.Context, dryrun bool, teamslug string, username string, role string) {
	// https://docs.github.com/en/rest/teams/members?apiVersion=2022-11-28#add-or-update-team-membership-for-a-user
	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/teams/%s/memberships/%s", config.Config.GithubAppOrganization, teamslug, username),
			"",
			"PUT",
			map[string]interface{}{"role": role},
		)
		if err != nil {
			logrus.Errorf("failed to add team member: %v. %s", err, string(body))
		}
	}

	if role == "maintainer" {
		if team, ok := g.teams[teamslug]; ok {
			// searching for maintainers
			found := false
			for _, m := range team.Maintainers {
				if m == username {
					found = true
					break
				}
			}
			if !found {
				g.teams[teamslug].Maintainers = append(g.teams[teamslug].Maintainers, username)
			}
		}
	} else {
		if team, ok := g.teams[teamslug]; ok {
			// searching for members
			found := false
			for _, m := range team.Members {
				if m == username {
					found = true
					break
				}
			}
			if !found {
				g.teams[teamslug].Members = append(g.teams[teamslug].Members, username)
			}
		}
	}
}

// role = member or maintainer (usually we use member)
func (g *GoliacRemoteImpl) UpdateTeamUpdateMember(ctx context.Context, dryrun bool, teamslug string, username string, role string) {
	// https://docs.github.com/en/rest/teams/members?apiVersion=2022-11-28#add-or-update-team-membership-for-a-user
	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/teams/%s/memberships/%s", config.Config.GithubAppOrganization, teamslug, username),
			"",
			"PUT",
			map[string]interface{}{"role": role},
		)
		if err != nil {
			logrus.Errorf("failed to update team member: %v. %s", err, string(body))
		}
	}

	if role == "maintainer" {
		if team, ok := g.teams[teamslug]; ok {
			// searching for maintainers
			found := false
			for _, m := range team.Maintainers {
				if m == username {
					found = true
					break
				}
			}
			if !found {
				g.teams[teamslug].Maintainers = append(g.teams[teamslug].Maintainers, username)
			}
			// searching for members
			for i, m := range team.Members {
				if m == username {
					g.teams[teamslug].Members = append(g.teams[teamslug].Members[:i], g.teams[teamslug].Members[i+1:]...)
					break
				}
			}
		}
	} else {
		if team, ok := g.teams[teamslug]; ok {
			// searching for members
			found := false
			for _, m := range team.Members {
				if m == username {
					found = true
					break
				}
			}
			if !found {
				g.teams[teamslug].Members = append(g.teams[teamslug].Members, username)
			}
			// searching for maintainers
			for i, m := range team.Maintainers {
				if m == username {
					g.teams[teamslug].Maintainers = append(g.teams[teamslug].Maintainers[:i], g.teams[teamslug].Maintainers[i+1:]...)
					break
				}
			}
		}
	}
}

func (g *GoliacRemoteImpl) UpdateTeamRemoveMember(ctx context.Context, dryrun bool, teamslug string, username string) {
	// https://docs.github.com/en/rest/teams/members?apiVersion=2022-11-28#add-or-update-team-membership-for-a-user
	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("orgs/%s/teams/%s/memberships/%s", config.Config.GithubAppOrganization, teamslug, username),
			"",
			"DELETE",
			nil,
		)
		if err != nil {
			logrus.Errorf("failed to remove team member: %v. %s", err, string(body))
		}
	}

	if team, ok := g.teams[teamslug]; ok {
		members := team.Members
		found := false
		for i, m := range members {
			if m == username {
				found = true
				members = append(members[:i], members[i+1:]...)
			}
		}
		if found {
			g.teams[teamslug].Members = members
		}
	}
}

func (g *GoliacRemoteImpl) UpdateTeamSetParent(ctx context.Context, dryrun bool, teamslug string, parentTeam *int) {
	// set parent's team
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#update-a-team
	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/teams/%s", config.Config.GithubAppOrganization, teamslug),
			"",
			"PATCH",
			map[string]interface{}{"parent_team_id": parentTeam},
		)
		if err != nil {
			logrus.Errorf("failed to delete a team: %v. %s", err, string(body))
		}
	}
}

func (g *GoliacRemoteImpl) DeleteTeam(ctx context.Context, dryrun bool, teamslug string) {
	// delete team
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#delete-a-team
	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/teams/%s", config.Config.GithubAppOrganization, teamslug),
			"",
			"DELETE",
			nil,
		)
		if err != nil {
			logrus.Errorf("failed to delete a team: %v. %s", err, string(body))
		}
	}

	delete(g.teams, teamslug)
	for name, slug := range g.teamSlugByName {
		if slug == teamslug {
			delete(g.teamSlugByName, name)
		}
	}
}

type CreateRepositoryResponse struct {
	Id     int    `json:"id"`
	NodeId string `json:"node_id"`
}

/*
boolProperties are:
- private
- archived
- allow_auto_merge
- delete_branch_on_merge
- allow_update_branch
- ...
*/
func (g *GoliacRemoteImpl) CreateRepository(ctx context.Context, dryrun bool, reponame string, description string, writers []string, readers []string, boolProperties map[string]bool) {
	repoId := 0
	repoRefId := reponame
	// create repository
	// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#create-an-organization-repository
	if !dryrun {
		props := map[string]interface{}{
			"name":        reponame,
			"description": description,
		}
		for k, v := range boolProperties {
			props[k] = v
		}

		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/repos", config.Config.GithubAppOrganization),
			"",
			"POST",
			props,
		)
		if err != nil {
			logrus.Errorf("failed to create repository: %v. %s", err, string(body))
			return
		}

		// get the repo id
		var resp CreateRepositoryResponse
		err = json.Unmarshal(body, &resp)
		if err != nil {
			logrus.Errorf("failed to read the create repository action response: %v", err)
			return
		}
		repoId = resp.Id
		repoRefId = resp.NodeId
	}

	// update the repositories list
	newRepo := &GithubRepository{
		Name:           reponame,
		Id:             repoId,
		RefId:          repoRefId,
		BoolProperties: boolProperties,
	}
	g.repositories[reponame] = newRepo
	g.repositoriesByRefId[repoRefId] = newRepo

	// add members
	for _, reader := range readers {
		// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#add-or-update-team-repository-permissions
		if !dryrun {
			body, err := g.client.CallRestAPI(
				ctx,
				fmt.Sprintf("orgs/%s/teams/%s/repos/%s/%s", config.Config.GithubAppOrganization, reader, config.Config.GithubAppOrganization, reponame),
				"",
				"PUT",
				map[string]interface{}{"permission": "pull"},
			)
			if err != nil {
				logrus.Errorf("failed to create repository (and add members): %v. %s", err, string(body))
				return
			}
		}

		teamsRepos := g.teamRepos[reader]
		if teamsRepos == nil {
			teamsRepos = make(map[string]*GithubTeamRepo)
		}
		teamsRepos[reponame] = &GithubTeamRepo{
			Name:       reponame,
			Permission: "READ",
		}
		g.teamRepos[reader] = teamsRepos
	}
	for _, writer := range writers {
		// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#add-or-update-team-repository-permissions
		if !dryrun {
			body, err := g.client.CallRestAPI(
				ctx,
				fmt.Sprintf("orgs/%s/teams/%s/repos/%s/%s", config.Config.GithubAppOrganization, writer, config.Config.GithubAppOrganization, reponame),
				"",
				"PUT",
				map[string]interface{}{"permission": "push"},
			)
			if err != nil {
				logrus.Errorf("failed to create repository (and add members): %v. %s", err, string(body))
			}
		}

		teamsRepos := g.teamRepos[writer]
		if teamsRepos == nil {
			teamsRepos = make(map[string]*GithubTeamRepo)
		}
		teamsRepos[reponame] = &GithubTeamRepo{
			Name:       reponame,
			Permission: "WRITE",
		}
		g.teamRepos[writer] = teamsRepos
	}
}

func (g *GoliacRemoteImpl) UpdateRepositoryAddTeamAccess(ctx context.Context, dryrun bool, reponame string, teamslug string, permission string) {
	// update member
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#add-or-update-team-repository-permissions
	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/teams/%s/repos/%s/%s", config.Config.GithubAppOrganization, teamslug, config.Config.GithubAppOrganization, reponame),
			"",
			"PUT",
			map[string]interface{}{"permission": permission},
		)
		if err != nil {
			logrus.Errorf("failed to add team access: %v. %s", err, string(body))
		}
	}

	teamsRepos := g.teamRepos[teamslug]
	if teamsRepos == nil {
		teamsRepos = make(map[string]*GithubTeamRepo)
	}
	rPermission := "READ"
	if permission == "push" {
		rPermission = "WRITE"
	}
	teamsRepos[reponame] = &GithubTeamRepo{
		Name:       reponame,
		Permission: rPermission,
	}
	g.teamRepos[teamslug] = teamsRepos
}

func (g *GoliacRemoteImpl) UpdateRepositoryUpdateTeamAccess(ctx context.Context, dryrun bool, reponame string, teamslug string, permission string) {
	// update member
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#add-or-update-team-repository-permissions
	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/orgs/%s/teams/%s/repos/%s/%s", config.Config.GithubAppOrganization, teamslug, config.Config.GithubAppOrganization, reponame),
			"",
			"PUT",
			map[string]interface{}{"permission": permission},
		)
		if err != nil {
			logrus.Errorf("failed to add team access: %v. %s", err, string(body))
		}
	}

	teamsRepos := g.teamRepos[teamslug]
	if teamsRepos == nil {
		teamsRepos = make(map[string]*GithubTeamRepo)
	}
	rPermission := "READ"
	if permission == "push" {
		rPermission = "WRITE"
	}
	teamsRepos[reponame] = &GithubTeamRepo{
		Name:       reponame,
		Permission: rPermission,
	}
	g.teamRepos[teamslug] = teamsRepos
}

func (g *GoliacRemoteImpl) UpdateRepositoryRemoveTeamAccess(ctx context.Context, dryrun bool, reponame string, teamslug string) {
	// delete member
	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#remove-a-repository-from-a-team
	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("orgs/%s/teams/%s/repos/%s/%s", config.Config.GithubAppOrganization, teamslug, config.Config.GithubAppOrganization, reponame),
			"",
			"DELETE",
			nil,
		)
		if err != nil {
			logrus.Errorf("failed to remove team access: %. %s", err, string(body))
		}
	}

	teamsRepos := g.teamRepos[teamslug]
	if teamsRepos != nil {
		delete(g.teamRepos[teamslug], reponame)
	}
}

/*
Used for
- private
- allow_auto_merge
- delete_branch_on_merge
- allow_update_branch
- archived
*/
func (g *GoliacRemoteImpl) UpdateRepositoryUpdateBoolProperty(ctx context.Context, dryrun bool, reponame string, propertyName string, propertyValue bool) {
	// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#update-a-repository
	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("repos/%s/%s", config.Config.GithubAppOrganization, reponame),
			"",
			"PATCH",
			map[string]interface{}{propertyName: propertyValue},
		)
		if err != nil {
			logrus.Errorf("failed to update repository %s setting: %v. %s", propertyName, err, string(body))
		}
	}

	if repo, ok := g.repositories[reponame]; ok {
		repo.BoolProperties[propertyName] = propertyValue
	}
}

func (g *GoliacRemoteImpl) UpdateRepositorySetExternalUser(ctx context.Context, dryrun bool, reponame string, githubid string, permission string) {
	// https://docs.github.com/en/rest/collaborators/collaborators?apiVersion=2022-11-28#add-a-repository-collaborator
	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("repos/%s/%s/collaborators/%s", config.Config.GithubAppOrganization, reponame, githubid),
			"",
			"PUT",
			map[string]interface{}{"permission": permission},
		)
		if err != nil {
			logrus.Errorf("failed to set repository collaborator: %v. %s", err, string(body))
		}
	}

	if repo, ok := g.repositories[reponame]; ok {
		if permission == "push" {
			repo.ExternalUsers[githubid] = "WRITE"
		} else {
			repo.ExternalUsers[githubid] = "READ"
		}
	}
}

func (g *GoliacRemoteImpl) updateRepositoryRemoveUser(ctx context.Context, dryrun bool, reponame string, githubid string) {
	// https://docs.github.com/en/rest/collaborators/collaborators?apiVersion=2022-11-28#remove-a-repository-collaborator
	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("repos/%s/%s/collaborators/%s", config.Config.GithubAppOrganization, reponame, githubid),
			"",
			"DELETE",
			nil,
		)
		if err != nil {
			logrus.Errorf("failed to remove repository collaborator: %v. %s", err, string(body))
		}
	}

	if repo, ok := g.repositories[reponame]; ok {
		delete(repo.ExternalUsers, githubid)
	}
}

func (g *GoliacRemoteImpl) UpdateRepositoryRemoveExternalUser(ctx context.Context, dryrun bool, reponame string, githubid string) {
	g.updateRepositoryRemoveUser(ctx, dryrun, reponame, githubid)
}

func (g *GoliacRemoteImpl) UpdateRepositoryRemoveInternalUser(ctx context.Context, dryrun bool, reponame string, githubid string) {
	g.updateRepositoryRemoveUser(ctx, dryrun, reponame, githubid)
}

func (g *GoliacRemoteImpl) DeleteRepository(ctx context.Context, dryrun bool, reponame string) {
	// delete repo
	// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#delete-a-repository
	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/repos/%s/%s", config.Config.GithubAppOrganization, reponame),
			"",
			"DELETE",
			nil,
		)
		if err != nil {
			logrus.Errorf("failed to delete repository: %v. %s", err, string(body))
		}
	}

	// update the repositories list
	if r, ok := g.repositories[reponame]; ok {
		delete(g.repositoriesByRefId, r.RefId)
		delete(g.repositories, reponame)
	}

}
func (g *GoliacRemoteImpl) RenameRepository(ctx context.Context, dryrun bool, reponame string, newname string) {
	// update repository
	// https://docs.github.com/fr/rest/repos/repos?apiVersion=2022-11-28#update-a-repository
	if !dryrun {
		body, err := g.client.CallRestAPI(
			ctx,
			fmt.Sprintf("/repos/%s/%s", config.Config.GithubAppOrganization, reponame),
			"",
			"PATCH",
			map[string]interface{}{"name": newname},
		)
		if err != nil {
			logrus.Errorf("failed to rename the repository %s (to %s): %v. %s", reponame, newname, err, string(body))
		}

		// update the repositories list
		if r, ok := g.repositories[reponame]; ok {
			delete(g.repositoriesByRefId, r.RefId)
			delete(g.repositories, reponame)
			r.Name = newname
			g.repositories[newname] = r
			g.repositoriesByRefId[r.RefId] = r

			for _, tr := range g.teamRepos {
				for rname, r := range tr {
					if rname == reponame {
						delete(tr, rname)
						r.Name = newname
						tr[newname] = r
					}
				}
			}
		}
	}
}
func (g *GoliacRemoteImpl) Begin(dryrun bool) {
}
func (g *GoliacRemoteImpl) Rollback(dryrun bool, err error) {
}
func (g *GoliacRemoteImpl) Commit(ctx context.Context, dryrun bool) error {
	return nil
}
