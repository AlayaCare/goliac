package engine

import "context"

type ReconciliatorExecutor interface {
	AddUserToOrg(ctx context.Context, dryrun bool, ghuserid string)
	RemoveUserFromOrg(ctx context.Context, dryrun bool, ghuserid string)

	CreateTeam(ctx context.Context, dryrun bool, teamname string, description string, parentTeam *int, members []string)
	UpdateTeamAddMember(ctx context.Context, dryrun bool, teamslug string, username string, role string)    // role can be 'member' or 'maintainer'
	UpdateTeamUpdateMember(ctx context.Context, dryrun bool, teamslug string, username string, role string) // role can be 'member' or 'maintainer'
	UpdateTeamRemoveMember(ctx context.Context, dryrun bool, teamslug string, username string)
	UpdateTeamSetParent(ctx context.Context, dryrun bool, teamslug string, parentTeam *int)
	DeleteTeam(ctx context.Context, dryrun bool, teamslug string)

	CreateRepository(ctx context.Context, dryrun bool, reponame string, descrition string, writers []string, readers []string, boolProperties map[string]bool)
	UpdateRepositoryUpdateBoolProperty(ctx context.Context, dryrun bool, reponame string, propertyName string, propertyValue bool)
	UpdateRepositoryAddTeamAccess(ctx context.Context, dryrun bool, reponame string, teamslug string, permission string)    // permission can be "pull", "push", or "admin" which correspond to read, write, and admin access.
	UpdateRepositoryUpdateTeamAccess(ctx context.Context, dryrun bool, reponame string, teamslug string, permission string) // permission can be "pull", "push", or "admin" which correspond to read, write, and admin access.
	UpdateRepositoryRemoveTeamAccess(ctx context.Context, dryrun bool, reponame string, teamslug string)
	AddRuleset(ctx context.Context, dryrun bool, ruleset *GithubRuleSet)
	UpdateRuleset(ctx context.Context, dryrun bool, ruleset *GithubRuleSet)
	DeleteRuleset(ctx context.Context, dryrun bool, rulesetid int)
	UpdateRepositorySetExternalUser(ctx context.Context, dryrun bool, reponame string, githubid string, permission string) // permission can be "pull" or "push"
	UpdateRepositoryRemoveExternalUser(ctx context.Context, dryrun bool, reponame string, githubid string)
	UpdateRepositoryRemoveInternalUser(ctx context.Context, dryrun bool, reponame string, githubid string)
	DeleteRepository(ctx context.Context, dryrun bool, reponame string)
	RenameRepository(ctx context.Context, dryrun bool, reponame string, newname string)

	Begin(dryrun bool)
	Rollback(dryrun bool, err error)
	Commit(ctx context.Context, dryrun bool) error
}
