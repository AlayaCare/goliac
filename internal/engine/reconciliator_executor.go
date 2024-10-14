package engine

type ReconciliatorExecutor interface {
	AddUserToOrg(dryrun bool, ghuserid string)
	RemoveUserFromOrg(dryrun bool, ghuserid string)

	CreateTeam(dryrun bool, teamname string, description string, members []string)
	UpdateTeamAddMember(dryrun bool, teamslug string, username string, role string) // role can be 'member' or 'maintainer'
	//UpdateTeamUpdateMember(dryrun bool, teamslug string, username string, role string) // role can be 'member' or 'maintainer'
	UpdateTeamRemoveMember(dryrun bool, teamslug string, username string)
	DeleteTeam(dryrun bool, teamslug string)

	CreateRepository(dryrun bool, reponame string, descrition string, writers []string, readers []string, boolProperties map[string]bool)
	UpdateRepositoryUpdateBoolProperty(dryrun bool, reponame string, propertyName string, propertyValue bool)
	UpdateRepositoryAddTeamAccess(dryrun bool, reponame string, teamslug string, permission string)    // permission can be "pull", "push", or "admin" which correspond to read, write, and admin access.
	UpdateRepositoryUpdateTeamAccess(dryrun bool, reponame string, teamslug string, permission string) // permission can be "pull", "push", or "admin" which correspond to read, write, and admin access.
	UpdateRepositoryRemoveTeamAccess(dryrun bool, reponame string, teamslug string)
	AddRuleset(dryrun bool, ruleset *GithubRuleSet)
	UpdateRuleset(dryrun bool, ruleset *GithubRuleSet)
	DeleteRuleset(dryrun bool, rulesetid int)
	UpdateRepositorySetExternalUser(dryrun bool, reponame string, githubid string, permission string) // permission can be "pull" or "push"
	UpdateRepositoryRemoveExternalUser(dryrun bool, reponame string, githubid string)
	DeleteRepository(dryrun bool, reponame string)

	Begin(dryrun bool)
	Rollback(dryrun bool, err error)
	Commit(dryrun bool) error
}
