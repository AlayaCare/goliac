package engine

import (
	"context"

	"github.com/goliac-project/goliac/internal/entity"
	"github.com/gosimple/slug"
)

type MutableEnvironmentLazyLoader struct {
	source MappedEntityLazyLoader[*GithubEnvironment]
	entity map[string]*GithubEnvironment
}

func NewMutableEnvironmentLazyLoader(source MappedEntityLazyLoader[*GithubEnvironment]) *MutableEnvironmentLazyLoader {
	return &MutableEnvironmentLazyLoader{source: source}
}

func (l *MutableEnvironmentLazyLoader) GetEntity() map[string]*GithubEnvironment {
	if l.entity == nil {
		if l.source == nil {
			l.entity = make(map[string]*GithubEnvironment)
		} else {
			e := l.source.GetEntity()
			l.entity = make(map[string]*GithubEnvironment)
			for k, v := range e {
				env := &GithubEnvironment{
					Name:      v.Name,
					Variables: make(map[string]string),
				}
				for k2, v2 := range v.Variables {
					env.Variables[k2] = v2
				}
				l.entity[k] = env
			}
		}
	}
	return l.entity
}

type MutableRepositoryVariableLazyLoader struct {
	source MappedEntityLazyLoader[string]
	entity map[string]string
}

func NewMutableRepositoryVariableLazyLoader(source MappedEntityLazyLoader[string]) *MutableRepositoryVariableLazyLoader {
	return &MutableRepositoryVariableLazyLoader{source: source}
}

func (l *MutableRepositoryVariableLazyLoader) GetEntity() map[string]string {
	if l.entity == nil {
		if l.source == nil {
			l.entity = make(map[string]string)
		} else {
			e := l.source.GetEntity()
			l.entity = make(map[string]string)
			for k, v := range e {
				l.entity[k] = v
			}
		}
	}
	return l.entity
}

/*
MutableGoliacRemoteImpl is used by GoliacReconciliatorImpl to update
the internal status of Github representation before appyling it for real
(or running in drymode)
It is by design
- a GoliacReconciliatorDatasource (as a ReconciliatorExecutor reader)
- and a kind of ReconciliatorExecutor (as a ReconciliatorExecutor writer)
*/
type MutableGoliacRemoteImpl struct {
	users        map[string]string
	repositories map[string]*GithubRepoComparable
	teams        map[string]*GithubTeamComparable
	rulesets     map[string]*GithubRuleSet
}

func NewMutableGoliacRemoteImpl(ctx context.Context, remote GoliacReconciliatorDatasource) (*MutableGoliacRemoteImpl, error) {
	rUsers := make(map[string]string)
	for k, v := range remote.Users() {
		rUsers[k] = v
	}
	rTeams := make(map[string]*GithubTeamComparable)
	teams, _, err := remote.Teams()
	if err != nil {
		return nil, err
	}
	for k, v := range teams {
		ght := *v
		rTeams[k] = &ght
	}

	rRepositories := make(map[string]*GithubRepoComparable)
	repositories, _, err := remote.Repositories()
	if err != nil {
		return nil, err
	}
	for k, v := range repositories {
		ghr := *v
		ghrulesets := make(map[string]*GithubRuleSet)
		for k, v := range ghr.Rulesets {
			ghrulesets[k] = v
		}
		ghr.Rulesets = ghrulesets
		ghbranchprotections := make(map[string]*GithubBranchProtection)
		for k, v := range v.BranchProtections {
			ghbranchprotections[k] = v
		}
		ghr.BranchProtections = ghbranchprotections
		ghr.Environments = NewMutableEnvironmentLazyLoader(
			v.Environments,
		)
		ghr.ActionVariables = NewMutableRepositoryVariableLazyLoader(
			v.ActionVariables,
		)
		rRepositories[k] = &ghr
		ghExternalUsersReaders := make([]string, len(v.ExternalUserReaders))
		copy(ghExternalUsersReaders, v.ExternalUserReaders)
		rRepositories[k].ExternalUserReaders = ghExternalUsersReaders

		ghExternalUsersWriters := make([]string, len(v.ExternalUserWriters))
		copy(ghExternalUsersWriters, v.ExternalUserWriters)
		rRepositories[k].ExternalUserWriters = ghExternalUsersWriters

		ghInternalUsers := make([]string, len(v.InternalUsers))
		copy(ghInternalUsers, v.InternalUsers)
		rRepositories[k].InternalUsers = ghInternalUsers

		ghProperties := make(map[string]bool)
		for k, v := range v.BoolProperties {
			ghProperties[k] = v
		}
		rRepositories[k].BoolProperties = ghProperties
	}

	rrulesets := make(map[string]*GithubRuleSet)
	rulesets, err := remote.RuleSets()
	if err != nil {
		return nil, err
	}
	for k, v := range rulesets {
		ghRuleset := *v
		ghRuleset.Rules = make(map[string]entity.RuleSetParameters)
		for k, v := range v.Rules {
			ghRuleset.Rules[k] = v
		}
		ghRuleset.Repositories = make([]string, len(v.Repositories))
		copy(ghRuleset.Repositories, v.Repositories)
		ghRuleset.OnInclude = make([]string, len(v.OnInclude))
		copy(ghRuleset.OnInclude, v.OnInclude)
		ghRuleset.OnExclude = make([]string, len(v.OnExclude))
		copy(ghRuleset.OnExclude, v.OnExclude)
		ghRuleset.BypassApps = make(map[string]string)
		for k, v := range v.BypassApps {
			ghRuleset.BypassApps[k] = v
		}
		ghRuleset.BypassTeams = make(map[string]string)
		for k, v := range v.BypassTeams {
			ghRuleset.BypassTeams[k] = v
		}
		rrulesets[k] = &ghRuleset
	}

	return &MutableGoliacRemoteImpl{
		users:        rUsers,
		repositories: rRepositories,
		teams:        rTeams,
		rulesets:     rrulesets,
	}, nil
}

// githubuserid -> membership (ADMIN, MEMBER)
func (m *MutableGoliacRemoteImpl) Users() map[string]string {
	return m.users
}

func (m *MutableGoliacRemoteImpl) Teams() map[string]*GithubTeamComparable {
	return m.teams
}
func (m *MutableGoliacRemoteImpl) Repositories() map[string]*GithubRepoComparable {
	return m.repositories
}
func (m *MutableGoliacRemoteImpl) RuleSets() map[string]*GithubRuleSet {
	return m.rulesets
}

// LISTENER

func (m *MutableGoliacRemoteImpl) AddUserToOrg(ghuserid string) {
	m.users[ghuserid] = ghuserid
}

func (m *MutableGoliacRemoteImpl) RemoveUserFromOrg(ghuserid string) {
	delete(m.users, ghuserid)
}

func (m *MutableGoliacRemoteImpl) CreateTeam(teamname string, description string, members []string, parentTeamId *int) {
	teamslug := slug.Make(teamname)
	var parentTeam *string
	if parentTeamId != nil {
		for k, v := range m.teams {
			if v.Id == *parentTeamId {
				parentTeam = &k
				break
			}
		}
	}

	t := GithubTeamComparable{
		Name:        teamname,
		Slug:        teamslug,
		Members:     members,
		Maintainers: []string{},
		ParentTeam:  parentTeam,
	}
	m.teams[teamslug] = &t
}
func (m *MutableGoliacRemoteImpl) UpdateTeamAddMember(teamslug string, username string, role string) {
	if t, ok := m.teams[teamslug]; ok {
		t.Members = append(t.Members, username)
	}
}
func (m *MutableGoliacRemoteImpl) UpdateTeamRemoveMember(teamslug string, username string) {
	if t, ok := m.teams[teamslug]; ok {
		for i, m := range t.Members {
			if m == username {
				t.Members = append(t.Members[:i], t.Members[i+1:]...)
				return
			}
		}
	}
}
func (m *MutableGoliacRemoteImpl) UpdateTeamUpdateMember(teamslug string, username string, role string) {
	if role == "maintainer" {
		if t, ok := m.teams[teamslug]; ok {
			for i, m := range t.Members {
				if m == username {
					t.Members = append(t.Members[:i], t.Members[i+1:]...)
					t.Maintainers = append(t.Maintainers, username)
					return
				}
			}
		}
	} else { // "member"
		if t, ok := m.teams[teamslug]; ok {
			for i, m := range t.Maintainers {
				if m == username {
					t.Maintainers = append(t.Maintainers[:i], t.Maintainers[i+1:]...)
					t.Members = append(t.Members, username)
					return
				}
			}
		}
	}
}
func (m *MutableGoliacRemoteImpl) UpdateTeamSetParent(ctx context.Context, dryrun bool, teamslug string, parentTeam *int) {
	if t, ok := m.teams[teamslug]; ok {
		var parentTeamName *string
		if parentTeam != nil {
			for k, v := range m.teams {
				if v.Id == *parentTeam {
					parentTeamName = &k
					break
				}
			}
		}
		t.ParentTeam = parentTeamName
	}
}
func (m *MutableGoliacRemoteImpl) DeleteTeam(teamslug string) {
	delete(m.teams, teamslug)
}
func (m *MutableGoliacRemoteImpl) CreateRepository(reponame string, descrition string, visibility string, writers []string, readers []string, boolProperties map[string]bool, defaultBranch string, forkFrom string) {
	r := GithubRepoComparable{
		Visibility:          visibility,
		BoolProperties:      boolProperties,
		Writers:             writers,
		Readers:             readers,
		ExternalUserReaders: []string{},
		ExternalUserWriters: []string{},
		InternalUsers:       []string{},
		DefaultBranchName:   defaultBranch,
		Rulesets:            make(map[string]*GithubRuleSet),
		BranchProtections:   make(map[string]*GithubBranchProtection),
		Environments:        NewMutableEnvironmentLazyLoader(nil),
		ActionVariables:     NewMutableRepositoryVariableLazyLoader(nil),
	}
	m.repositories[reponame] = &r
}
func (m *MutableGoliacRemoteImpl) UpdateRepositoryAddTeamAccess(reponame string, teamslug string, permission string) {
	if r, ok := m.repositories[reponame]; ok {
		if permission == "pull" {
			r.Readers = append(r.Readers, teamslug)
		} else if permission == "push" {
			r.Writers = append(r.Writers, teamslug)
		}
	}
}

func (m *MutableGoliacRemoteImpl) UpdateRepositoryUpdateTeamAccess(reponame string, teamslug string, permission string) {
	if r, ok := m.repositories[reponame]; ok {
		// remove the team from the readers and writers
		for i, t := range r.Writers {
			if t == teamslug {
				r.Writers = append(r.Writers[:i], r.Writers[i+1:]...)
				break
			}
		}
		for i, t := range r.Readers {
			if t == teamslug {
				r.Readers = append(r.Readers[:i], r.Readers[i+1:]...)
				break
			}
		}
		// add the team to the readers and writers
		if permission == "pull" {
			r.Readers = append(r.Readers, teamslug)
		} else if permission == "push" {
			r.Writers = append(r.Writers, teamslug)
		}
	}
}
func (m *MutableGoliacRemoteImpl) UpdateRepositoryRemoveTeamAccess(reponame string, teamslug string) {
	if r, ok := m.repositories[reponame]; ok {
		for i, t := range r.Writers {
			if t == teamslug {
				r.Writers = append(r.Writers[:i], r.Writers[i+1:]...)
				break
			}
		}
		for i, t := range r.Readers {
			if t == teamslug {
				r.Readers = append(r.Readers[:i], r.Readers[i+1:]...)
				break
			}
		}
	}
}
func (m *MutableGoliacRemoteImpl) DeleteRepository(reponame string) {
	delete(m.repositories, reponame)
}

func (m *MutableGoliacRemoteImpl) RenameRepository(reponame string, newname string) {
	r := m.repositories[reponame]

	// it is not supposed to be nil
	if r == nil {
		return
	}
	delete(m.repositories, reponame)
	m.repositories[newname] = r
}

/*
UpdateRepositoryUpdateBoolProperty is used for
- visibility (string)
- archived (bool)
- allow_auto_merge (bool)
- delete_branch_on_merge (bool)
- allow_update_branch (bool)
*/
func (m *MutableGoliacRemoteImpl) UpdateRepositoryUpdateProperty(reponame string, propertyName string, propertyValue interface{}) {
	if r, ok := m.repositories[reponame]; ok {
		if propertyName == "visibility" {
			r.Visibility = propertyValue.(string)
		} else if propertyName == "default_branch" {
			r.DefaultBranchName = propertyValue.(string)
		} else {
			r.BoolProperties[propertyName] = propertyValue.(bool)
		}
	}
}
func (m *MutableGoliacRemoteImpl) UpdateRepositorySetExternalUser(reponame string, collaboatorGithubId string, permission string) {
	if r, ok := m.repositories[reponame]; ok {
		if permission == "pull" {
			r.ExternalUserReaders = append(r.ExternalUserReaders, collaboatorGithubId)
		} else if permission == "push" {
			r.ExternalUserWriters = append(r.ExternalUserWriters, collaboatorGithubId)
		}
	}
}
func (m *MutableGoliacRemoteImpl) UpdateRepositoryRemoveExternalUser(reponame string, collaboatorGithubId string) {
	if r, ok := m.repositories[reponame]; ok {
		for i, t := range r.ExternalUserReaders {
			if t == collaboatorGithubId {
				r.ExternalUserReaders = append(r.ExternalUserReaders[:i], r.ExternalUserReaders[i+1:]...)
				break
			}
		}
		for i, t := range r.ExternalUserWriters {
			if t == collaboatorGithubId {
				r.ExternalUserWriters = append(r.ExternalUserWriters[:i], r.ExternalUserWriters[i+1:]...)
				break
			}
		}
	}
}
func (m *MutableGoliacRemoteImpl) UpdateRepositoryRemoveInternalUser(reponame string, collaboatorGithubId string) {
	if r, ok := m.repositories[reponame]; ok {
		for i, t := range r.InternalUsers {
			if t == collaboatorGithubId {
				r.InternalUsers = append(r.InternalUsers[:i], r.InternalUsers[i+1:]...)
				break
			}
		}
	}
}
func (m *MutableGoliacRemoteImpl) AddRepositoryRuleset(reponame string, ruleset *GithubRuleSet) {
	if r, ok := m.repositories[reponame]; ok {
		r.Rulesets[ruleset.Name] = ruleset
	}
}
func (m *MutableGoliacRemoteImpl) UpdateRepositoryRuleset(reponame string, ruleset *GithubRuleSet) {
	if r, ok := m.repositories[reponame]; ok {
		r.Rulesets[ruleset.Name] = ruleset
	}
}
func (m *MutableGoliacRemoteImpl) DeleteRepositoryRuleset(reponame string, rulesetid int) {
	if r, ok := m.repositories[reponame]; ok {
		for _, rs := range r.Rulesets {
			if rs.Id == rulesetid {
				delete(r.Rulesets, rs.Name)
				break
			}
		}
	}
}

func (m *MutableGoliacRemoteImpl) AddRuleset(ruleset *GithubRuleSet) {
	m.rulesets[ruleset.Name] = ruleset
}
func (m *MutableGoliacRemoteImpl) UpdateRuleset(ruleset *GithubRuleSet) {
	m.rulesets[ruleset.Name] = ruleset
}
func (m *MutableGoliacRemoteImpl) DeleteRuleset(rulesetid int) {
	for _, rs := range m.rulesets {
		if rs.Id == rulesetid {
			delete(m.rulesets, rs.Name)
			break
		}
	}
}

func (m *MutableGoliacRemoteImpl) AddRepositoryEnvironment(repositoryName string, environmentName string) {
	if r, ok := m.repositories[repositoryName]; ok {
		r.Environments.GetEntity()[environmentName] = &GithubEnvironment{
			Name:      environmentName,
			Variables: map[string]string{},
		}
	}
}
func (m *MutableGoliacRemoteImpl) DeleteRepositoryEnvironment(repositoryName string, environmentName string) {
	delete(m.repositories[repositoryName].Environments.GetEntity(), environmentName)
}

// Repository variables management
func (m *MutableGoliacRemoteImpl) AddRepositoryVariable(repositoryName string, variableName string, variableValue string) {
	if r, ok := m.repositories[repositoryName]; ok {
		r.ActionVariables.GetEntity()[variableName] = variableValue
	}
}
func (m *MutableGoliacRemoteImpl) UpdateRepositoryVariable(repositoryName string, variableName string, variableValue string) {
	if r, ok := m.repositories[repositoryName]; ok {
		r.ActionVariables.GetEntity()[variableName] = variableValue
	}
}

func (m *MutableGoliacRemoteImpl) DeleteRepositoryVariable(repositoryName string, variableName string) {
	delete(m.repositories[repositoryName].ActionVariables.GetEntity(), variableName)
}

// Environment variables management
func (m *MutableGoliacRemoteImpl) AddRepositoryEnvironmentVariable(repositoryName string, environmentName string, variableName string, variableValue string) {
	if r, ok := m.repositories[repositoryName]; ok {
		if r.Environments.GetEntity()[environmentName] == nil {
			r.Environments.GetEntity()[environmentName] = &GithubEnvironment{
				Name:      environmentName,
				Variables: map[string]string{},
			}
		}
		r.Environments.GetEntity()[environmentName].Variables[variableName] = variableValue
	}
}
func (m *MutableGoliacRemoteImpl) UpdateRepositoryEnvironmentVariable(repositoryName string, environmentName string, variableName string, variableValue string) {
	if r, ok := m.repositories[repositoryName]; ok {
		if r.Environments.GetEntity()[environmentName] == nil {
			r.Environments.GetEntity()[environmentName] = &GithubEnvironment{
				Name:      environmentName,
				Variables: map[string]string{},
			}
		}
		r.Environments.GetEntity()[environmentName].Variables[variableName] = variableValue
	}
}
func (m *MutableGoliacRemoteImpl) DeleteRepositoryEnvironmentVariable(repositoryName string, environmentName string, variableName string) {
	if r, ok := m.repositories[repositoryName]; ok {
		if r.Environments.GetEntity()[environmentName] != nil {
			delete(r.Environments.GetEntity()[environmentName].Variables, variableName)
		}
	}
}
func (m *MutableGoliacRemoteImpl) AddRepositoryAutolink(repositoryName string, autolink *GithubAutolink) {
	if r, ok := m.repositories[repositoryName]; ok {
		r.Autolinks.GetEntity()[autolink.KeyPrefix] = autolink
	}
}
func (m *MutableGoliacRemoteImpl) DeleteRepositoryAutolink(repositoryName string, autolinkId int) {
	for key, autolink := range m.repositories[repositoryName].Autolinks.GetEntity() {
		if autolink.Id == autolinkId {
			delete(m.repositories[repositoryName].Autolinks.GetEntity(), key)
			break
		}
	}
}
func (m *MutableGoliacRemoteImpl) UpdateRepositoryAutolink(repositoryName string, previousAutolinkId int, autolink *GithubAutolink) {
	if r, ok := m.repositories[repositoryName]; ok {
		r.Autolinks.GetEntity()[autolink.KeyPrefix] = autolink
	}
}
