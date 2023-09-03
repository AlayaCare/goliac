package engine

type ReconciliatorExecutor interface {
	AddUserToOrg(ghuserid string)
	RemoveUserFromOrg(ghuserid string)

	CreateTeam(teamname string, description string, members []string)
	UpdateTeamAddMember(teamslug string, username string, role string) // role can be 'member' or 'maintainer'
	//UpdateTeamUpdateMember(teamslug string, username string, role string) // role can be 'member' or 'maintainer'
	UpdateTeamRemoveMember(teamslug string, username string)
	DeleteTeam(teamslug string)

	CreateRepository(reponame string, descrition string, writers []string, readers []string, public bool)
	UpdateRepositoryUpdateArchived(reponame string, archived bool)
	UpdateRepositoryUpdatePrivate(reponame string, private bool)
	UpdateRepositoryAddTeamAccess(reponame string, teamslug string, permission string)    // permission can be "pull", "push", or "admin" which correspond to read, write, and admin access.
	UpdateRepositoryUpdateTeamAccess(reponame string, teamslug string, permission string) // permission can be "pull", "push", or "admin" which correspond to read, write, and admin access.
	UpdateRepositoryRemoveTeamAccess(reponame string, teamslug string)
	AddRuleset(ruleset *GithubRuleSet)
	UpdateRuleset(ruleset *GithubRuleSet)
	DeleteRuleset(rulesetid int)
	UpdateRepositorySetExternalUser(reponame string, githubid string, permission string) // permission can be "pull" or "push"
	UpdateRepositoryRemoveExternalUser(reponame string, githubid string)
	DeleteRepository(reponame string)

	Begin()
	Rollback(error)
	Commit()
}
