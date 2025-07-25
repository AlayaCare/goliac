package engine

import (
	"context"

	"github.com/goliac-project/goliac/internal/observability"
)

type ReconciliatorExecutor interface {
	AddUserToOrg(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, ghuserid string)
	RemoveUserFromOrg(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, ghuserid string)

	CreateTeam(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, teamname string, description string, parentTeam *int, members []string)
	UpdateTeamAddMember(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, teamslug string, username string, role string)    // role can be 'member' or 'maintainer'
	UpdateTeamUpdateMember(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, teamslug string, username string, role string) // role can be 'member' or 'maintainer'
	UpdateTeamRemoveMember(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, teamslug string, username string)
	UpdateTeamSetParent(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, teamslug string, parentTeam *int)
	DeleteTeam(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, teamslug string)

	// CreateRepository can have a githubToken parameter (if null it use Goliac token)
	CreateRepository(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string, descrition string, visibility string, writers []string, readers []string, boolProperties map[string]bool, defaultBranch string, githubToken *string, forkFrom string)
	UpdateRepositoryUpdateProperty(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string, propertyName string, propertyValue interface{})
	UpdateRepositoryAddTeamAccess(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string, teamslug string, permission string)    // permission can be "pull", "push", or "admin" which correspond to read, write, and admin access.
	UpdateRepositoryUpdateTeamAccess(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string, teamslug string, permission string) // permission can be "pull", "push", or "admin" which correspond to read, write, and admin access.
	UpdateRepositoryRemoveTeamAccess(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string, teamslug string)
	AddRuleset(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, ruleset *GithubRuleSet)
	UpdateRuleset(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, ruleset *GithubRuleSet)
	DeleteRuleset(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, rulesetid int)
	AddRepositoryRuleset(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string, ruleset *GithubRuleSet)
	UpdateRepositoryRuleset(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string, ruleset *GithubRuleSet)
	DeleteRepositoryRuleset(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string, rulesetid int)
	AddRepositoryBranchProtection(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string, branchprotection *GithubBranchProtection)
	UpdateRepositoryBranchProtection(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string, branchprotection *GithubBranchProtection)
	DeleteRepositoryBranchProtection(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string, branchprotection *GithubBranchProtection)
	UpdateRepositorySetExternalUser(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string, githubid string, permission string) // permission can be "pull" or "push"
	UpdateRepositoryRemoveExternalUser(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string, githubid string)
	UpdateRepositoryRemoveInternalUser(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string, githubid string)
	DeleteRepository(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string)
	RenameRepository(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, reponame string, newname string)

	// Environment management
	AddRepositoryEnvironment(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, repositoryName string, environmentName string)
	DeleteRepositoryEnvironment(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, repositoryName string, environmentName string)

	// Repository variables management
	AddRepositoryVariable(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, repositoryName string, variableName string, variableValue string)
	UpdateRepositoryVariable(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, repositoryName string, variableName string, variableValue string)
	DeleteRepositoryVariable(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, repositoryName string, variableName string)

	// Environment variables management
	AddRepositoryEnvironmentVariable(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, repositoryName string, environmentName string, variableName string, variableValue string)
	UpdateRepositoryEnvironmentVariable(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, repositoryName string, environmentName string, variableName string, variableValue string)
	DeleteRepositoryEnvironmentVariable(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, repositoryName string, environmentName string, variableName string)

	// Repository autolinks management
	AddRepositoryAutolink(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, repositoryName string, autolink *GithubAutolink)
	DeleteRepositoryAutolink(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, repositoryName string, autolinkId int)
	UpdateRepositoryAutolink(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool, repositoryName string, autolink *GithubAutolink)

	Begin(logsCollector *observability.LogCollection, dryrun bool)
	Rollback(logsCollector *observability.LogCollection, dryrun bool, err error)
	Commit(ctx context.Context, logsCollector *observability.LogCollection, dryrun bool) error
}
