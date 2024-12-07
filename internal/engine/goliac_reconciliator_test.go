package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gosimple/slug"
	"github.com/stretchr/testify/assert"
)

type GoliacLocalMock struct {
	users     map[string]*entity.User
	externals map[string]*entity.User
	teams     map[string]*entity.Team
	repos     map[string]*entity.Repository
	rulesets  map[string]*entity.RuleSet
}

func (m *GoliacLocalMock) Clone(fs billy.Filesystem, accesstoken, repositoryUrl, branch string) error {
	return nil
}
func (m *GoliacLocalMock) ListCommitsFromTag(tagname string) ([]*object.Commit, error) {
	return nil, fmt.Errorf("not tag %s found", tagname)
}
func (m *GoliacLocalMock) GetHeadCommit() (*object.Commit, error) {
	return nil, nil
}
func (m *GoliacLocalMock) CheckoutCommit(commit *object.Commit) error {
	return nil
}
func (m *GoliacLocalMock) PushTag(tagname string, hash plumbing.Hash, accesstoken string) error {
	return nil
}
func (m *GoliacLocalMock) LoadRepoConfig() (*config.RepositoryConfig, error) {
	return &config.RepositoryConfig{}, nil
}
func (m *GoliacLocalMock) LoadAndValidate() ([]error, []entity.Warning) {
	return nil, nil
}
func (m *GoliacLocalMock) LoadAndValidateLocal(fs billy.Filesystem) ([]error, []entity.Warning) {
	return nil, nil
}
func (m *GoliacLocalMock) Teams() map[string]*entity.Team {
	return m.teams
}
func (m *GoliacLocalMock) Repositories() map[string]*entity.Repository {
	return m.repos
}
func (m *GoliacLocalMock) Users() map[string]*entity.User {
	return m.users
}
func (m *GoliacLocalMock) ExternalUsers() map[string]*entity.User {
	return m.externals
}
func (m *GoliacLocalMock) RuleSets() map[string]*entity.RuleSet {
	return m.rulesets
}
func (m *GoliacLocalMock) UpdateAndCommitCodeOwners(repoconfig *config.RepositoryConfig, dryrun bool, accesstoken string, branch string, tagname string, githubOrganization string) error {
	return nil
}
func (m *GoliacLocalMock) ArchiveRepos(reposToArchiveList []string, accesstoken string, branch string, tagname string) error {
	return nil
}
func (m *GoliacLocalMock) SyncUsersAndTeams(repoconfig *config.RepositoryConfig, plugin UserSyncPlugin, accesstoken string, dryrun bool, force bool) (bool, error) {
	return false, nil
}
func (m *GoliacLocalMock) Close(fs billy.Filesystem) {

}

type GoliacRemoteMock struct {
	users      map[string]string
	teams      map[string]*GithubTeam // key is the slug team
	repos      map[string]*GithubRepository
	teamsrepos map[string]map[string]*GithubTeamRepo // key is the slug team
	rulesets   map[string]*GithubRuleSet
	appids     map[string]int
}

func (m *GoliacRemoteMock) Load(ctx context.Context, continueOnError bool) error {
	return nil
}
func (m *GoliacRemoteMock) IsEnterprise() bool {
	return true
}
func (m *GoliacRemoteMock) FlushCache() {
}
func (m *GoliacRemoteMock) FlushCacheUsersTeamsOnly() {
}
func (m *GoliacRemoteMock) RuleSets(ctx context.Context) map[string]*GithubRuleSet {
	return m.rulesets
}
func (m *GoliacRemoteMock) Users(ctx context.Context) map[string]string {
	return m.users
}

func (m *GoliacRemoteMock) TeamSlugByName(ctx context.Context) map[string]string {
	slugs := make(map[string]string)
	for _, v := range m.teams {
		slugs[v.Name] = slug.Make(v.Name)
	}
	return slugs
}
func (m *GoliacRemoteMock) Teams(ctx context.Context) map[string]*GithubTeam {
	return m.teams
}
func (m *GoliacRemoteMock) Repositories(ctx context.Context) map[string]*GithubRepository {
	return m.repos
}
func (m *GoliacRemoteMock) RepositoriesByRefId(ctx context.Context) map[string]*GithubRepository {
	return make(map[string]*GithubRepository)
}
func (m *GoliacRemoteMock) TeamRepositories(ctx context.Context) map[string]map[string]*GithubTeamRepo {
	return m.teamsrepos
}
func (m *GoliacRemoteMock) AppIds(ctx context.Context) map[string]int {
	return m.appids
}

type ReconciliatorListenerRecorder struct {
	UsersCreated map[string]string
	UsersRemoved map[string]string

	TeamsCreated      map[string][]string
	TeamMemberAdded   map[string][]string
	TeamMemberRemoved map[string][]string
	TeamMemberUpdated map[string][]string
	TeamParentUpdated map[string]*int
	TeamDeleted       map[string]bool

	RepositoryCreated              map[string]bool
	RepositoryTeamAdded            map[string][]string
	RepositoryTeamUpdated          map[string][]string
	RepositoryTeamRemoved          map[string][]string
	RepositoriesDeleted            map[string]bool
	RepositoriesUpdatePrivate      map[string]bool
	RepositoriesUpdateArchived     map[string]bool
	RepositoriesSetExternalUser    map[string]string
	RepositoriesRemoveExternalUser map[string]bool

	RuleSetCreated map[string]*GithubRuleSet
	RuleSetUpdated map[string]*GithubRuleSet
	RuleSetDeleted []int
}

func NewReconciliatorListenerRecorder() *ReconciliatorListenerRecorder {
	r := ReconciliatorListenerRecorder{
		UsersCreated:                   make(map[string]string),
		UsersRemoved:                   make(map[string]string),
		TeamsCreated:                   make(map[string][]string),
		TeamMemberAdded:                make(map[string][]string),
		TeamMemberRemoved:              make(map[string][]string),
		TeamMemberUpdated:              make(map[string][]string),
		TeamParentUpdated:              make(map[string]*int),
		TeamDeleted:                    make(map[string]bool),
		RepositoryCreated:              make(map[string]bool),
		RepositoryTeamAdded:            make(map[string][]string),
		RepositoryTeamUpdated:          make(map[string][]string),
		RepositoryTeamRemoved:          make(map[string][]string),
		RepositoriesDeleted:            make(map[string]bool),
		RepositoriesUpdatePrivate:      make(map[string]bool),
		RepositoriesUpdateArchived:     make(map[string]bool),
		RepositoriesSetExternalUser:    make(map[string]string),
		RepositoriesRemoveExternalUser: make(map[string]bool),
		RuleSetCreated:                 make(map[string]*GithubRuleSet),
		RuleSetUpdated:                 make(map[string]*GithubRuleSet),
		RuleSetDeleted:                 make([]int, 0),
	}
	return &r
}
func (r *ReconciliatorListenerRecorder) AddUserToOrg(ctx context.Context, dryrun bool, ghuserid string) {
	r.UsersCreated[ghuserid] = ghuserid
}
func (r *ReconciliatorListenerRecorder) RemoveUserFromOrg(ctx context.Context, dryrun bool, ghuserid string) {
	r.UsersRemoved[ghuserid] = ghuserid
}
func (r *ReconciliatorListenerRecorder) CreateTeam(ctx context.Context, dryrun bool, teamname string, description string, parentTeam *int, members []string) {
	r.TeamsCreated[teamname] = append(r.TeamsCreated[teamname], members...)
}
func (r *ReconciliatorListenerRecorder) UpdateTeamAddMember(ctx context.Context, dryrun bool, teamslug string, username string, role string) {
	r.TeamMemberAdded[teamslug] = append(r.TeamMemberAdded[teamslug], username)
}
func (r *ReconciliatorListenerRecorder) UpdateTeamRemoveMember(ctx context.Context, dryrun bool, teamslug string, username string) {
	r.TeamMemberRemoved[teamslug] = append(r.TeamMemberRemoved[teamslug], username)
}
func (r *ReconciliatorListenerRecorder) UpdateTeamUpdateMember(ctx context.Context, dryrun bool, teamslug string, username string, role string) {
	r.TeamMemberUpdated[teamslug] = append(r.TeamMemberUpdated[teamslug], username)
}
func (r *ReconciliatorListenerRecorder) UpdateTeamSetParent(ctx context.Context, dryrun bool, teamslug string, parentTeam *int) {
	r.TeamParentUpdated[teamslug] = parentTeam
}
func (r *ReconciliatorListenerRecorder) DeleteTeam(ctx context.Context, dryrun bool, teamslug string) {
	r.TeamDeleted[teamslug] = true
}
func (r *ReconciliatorListenerRecorder) CreateRepository(ctx context.Context, dryrun bool, reponame string, descrition string, writers []string, readers []string, boolProperties map[string]bool) {
	r.RepositoryCreated[reponame] = true
}
func (r *ReconciliatorListenerRecorder) UpdateRepositoryAddTeamAccess(ctx context.Context, dryrun bool, reponame string, teamslug string, permission string) {
	r.RepositoryTeamAdded[reponame] = append(r.RepositoryTeamAdded[reponame], teamslug)
}
func (r *ReconciliatorListenerRecorder) UpdateRepositoryUpdateTeamAccess(ctx context.Context, dryrun bool, reponame string, teamslug string, permission string) {
	r.RepositoryTeamUpdated[reponame] = append(r.RepositoryTeamUpdated[reponame], teamslug)
}
func (r *ReconciliatorListenerRecorder) UpdateRepositoryRemoveTeamAccess(ctx context.Context, dryrun bool, reponame string, teamslug string) {
	r.RepositoryTeamRemoved[reponame] = append(r.RepositoryTeamRemoved[reponame], teamslug)
}
func (r *ReconciliatorListenerRecorder) DeleteRepository(ctx context.Context, dryrun bool, reponame string) {
	r.RepositoriesDeleted[reponame] = true
}
func (r *ReconciliatorListenerRecorder) UpdateRepositoryUpdateBoolProperty(ctx context.Context, dryrun bool, reponame string, propertyName string, propertyValue bool) {
	r.RepositoriesUpdatePrivate[reponame] = true
}
func (r *ReconciliatorListenerRecorder) UpdateRepositorySetExternalUser(ctx context.Context, dryrun bool, reponame string, githubid string, permission string) {
	r.RepositoriesSetExternalUser[githubid] = permission
}
func (r *ReconciliatorListenerRecorder) UpdateRepositoryRemoveExternalUser(ctx context.Context, dryrun bool, reponame string, githubid string) {
	r.RepositoriesRemoveExternalUser[githubid] = true
}
func (r *ReconciliatorListenerRecorder) AddRuleset(ctx context.Context, dryrun bool, ruleset *GithubRuleSet) {
	r.RuleSetCreated[ruleset.Name] = ruleset
}
func (r *ReconciliatorListenerRecorder) UpdateRuleset(ctx context.Context, dryrun bool, ruleset *GithubRuleSet) {
	r.RuleSetUpdated[ruleset.Name] = ruleset
}
func (r *ReconciliatorListenerRecorder) DeleteRuleset(ctx context.Context, dryrun bool, rulesetid int) {
	r.RuleSetDeleted = append(r.RuleSetDeleted, rulesetid)
}
func (r *ReconciliatorListenerRecorder) Begin(dryrun bool) {
}
func (r *ReconciliatorListenerRecorder) Rollback(dryrun bool, err error) {
}
func (r *ReconciliatorListenerRecorder) Commit(ctx context.Context, dryrun bool) error {
	return nil
}

func TestReconciliation(t *testing.T) {

	t.Run("happy path: new team", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		newTeam := &entity.Team{}
		newTeam.Name = "new"
		newTeam.Spec.Owners = []string{"new.owner"}
		newTeam.Spec.Members = []string{"new.member"}
		local.teams["new"] = newTeam

		newOwner := entity.User{}
		newOwner.Name = "new.owner"
		newOwner.Spec.GithubID = "new_owner"
		local.users["new.owner"] = &newOwner
		newMember := entity.User{}
		newMember.Name = "new.member"
		newMember.Spec.GithubID = "new_member"
		local.users["new.member"] = &newMember

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 2 members created
		assert.Equal(t, 2, len(recorder.TeamsCreated["new"]))
		assert.Equal(t, 1, len(recorder.TeamsCreated["new"+config.Config.GoliacTeamOwnerSuffix]))
	})

	t.Run("happy path: new team with non english slug", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		newTeam := &entity.Team{}
		newTeam.Name = "nouveauté"
		newTeam.Spec.Owners = []string{"new.owner"}
		newTeam.Spec.Members = []string{"new.member"}
		local.teams["nouveauté"] = newTeam

		newOwner := entity.User{}
		newOwner.Name = "new.owner"
		newOwner.Spec.GithubID = "new_owner"
		local.users["new.owner"] = &newOwner
		newMember := entity.User{}
		newMember.Name = "new.member"
		newMember.Spec.GithubID = "new_member"
		local.users["new.member"] = &newMember

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 2 members created
		assert.Equal(t, 2, len(recorder.TeamsCreated["nouveauté"]))
		assert.Equal(t, 1, len(recorder.TeamsCreated["nouveaute"+config.Config.GoliacTeamOwnerSuffix]))
	})

	t.Run("happy path: existing team with new members", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		existingTeam := &entity.Team{}
		existingTeam.Name = "existing"
		existingTeam.Spec.Owners = []string{"existing.owner", "existing.owner2"}
		existingTeam.Spec.Members = []string{"existing.member"}
		local.teams["existing"] = existingTeam

		existing_owner := entity.User{}
		existing_owner.Name = "existing.owner"
		existing_owner.Spec.GithubID = "existing_owner"
		local.users["existing.owner"] = &existing_owner

		existing_owner2 := entity.User{}
		existing_owner2.Name = "existing.owner2"
		existing_owner2.Spec.GithubID = "existing_owner2"
		local.users["existing.owner2"] = &existing_owner2

		existing_member := entity.User{}
		existing_member.Name = "existing.member"
		existing_member.Spec.GithubID = "existing_member"
		local.users["existing.member"] = &existing_member

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing
		existingowners := &GithubTeam{
			Name:    "existing" + config.Config.GoliacTeamOwnerSuffix,
			Slug:    "existing" + config.Config.GoliacTeamOwnerSuffix,
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"+config.Config.GoliacTeamOwnerSuffix] = existingowners

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 members added
		assert.Equal(t, 0, len(recorder.TeamsCreated))
		assert.Equal(t, 1, len(recorder.TeamMemberAdded["existing"]))
	})

	t.Run("happy path: existing team with non english slug with new members", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		existingTeam := &entity.Team{}
		existingTeam.Name = "exist ing"
		existingTeam.Spec.Owners = []string{"existing.owner", "existing.owner2"}
		existingTeam.Spec.Members = []string{"existing.member"}
		local.teams["exist ing"] = existingTeam

		existing_owner := entity.User{}
		existing_owner.Name = "existing.owner"
		existing_owner.Spec.GithubID = "existing_owner"
		local.users["existing.owner"] = &existing_owner

		existing_owner2 := entity.User{}
		existing_owner2.Name = "existing.owner2"
		existing_owner2.Spec.GithubID = "existing_owner2"
		local.users["existing.owner2"] = &existing_owner2

		existing_member := entity.User{}
		existing_member.Name = "existing.member"
		existing_member.Spec.GithubID = "existing_member"
		local.users["existing.member"] = &existing_member

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		existing := &GithubTeam{
			Name:    "exist ing",
			Slug:    "exist-ing",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["exist-ing"] = existing

		existingowners := &GithubTeam{
			Name:    "exist ing" + config.Config.GoliacTeamOwnerSuffix,
			Slug:    "exist-ing" + config.Config.GoliacTeamOwnerSuffix,
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["exist-ing"+config.Config.GoliacTeamOwnerSuffix] = existingowners

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 members added
		ctx := context.TODO()
		assert.Equal(t, "exist-ing", remote.TeamSlugByName(ctx)["exist ing"])
		assert.Equal(t, 0, len(recorder.TeamsCreated))
		assert.Equal(t, 1, len(recorder.TeamMemberAdded["exist-ing"]))
	})

	t.Run("happy path: new team + adding everyone team", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{
			EveryoneTeamEnabled: true,
		}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		newTeam := &entity.Team{}
		newTeam.Name = "new"
		newTeam.Spec.Owners = []string{"new.owner"}
		newTeam.Spec.Members = []string{"new.member"}
		local.teams["new"] = newTeam

		newOwner := entity.User{}
		newOwner.Name = "new.owner"
		newOwner.Spec.GithubID = "new_owner"
		local.users["new.owner"] = &newOwner
		newMember := entity.User{}
		newMember.Name = "new.member"
		newMember.Spec.GithubID = "new_member"
		local.users["new.member"] = &newMember

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 2 members created
		assert.Equal(t, 2, len(recorder.TeamsCreated["new"]))
		assert.Equal(t, 1, len(recorder.TeamsCreated["new"+config.Config.GoliacTeamOwnerSuffix]))
		// and the everyone team
		assert.Equal(t, 2, len(recorder.TeamsCreated["everyone"]))
	})

	t.Run("happy path: removed team without destructive operation", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		removing := &GithubTeam{
			Name:    "removing",
			Slug:    "removing",
			Members: []string{"existing_owner", "existing_owner"},
		}
		remote.teams["removing"] = removing

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 team deleted
		assert.Equal(t, 0, len(recorder.TeamDeleted))
	})

	t.Run("happy path: status quo: no new parent to a team", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}

		lParentTeam := &entity.Team{}
		lParentTeam.Name = "parentTeam"
		lParentTeam.Spec.Owners = []string{"existing_owner"}
		lParentTeam.Spec.Members = []string{}
		local.teams["parentTeam"] = lParentTeam

		lChildTeam := &entity.Team{}
		lChildTeam.Name = "childTeam"
		lChildTeam.Spec.Owners = []string{"existing_owner"}
		lChildTeam.Spec.Members = []string{}
		local.teams["childTeam"] = lChildTeam

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}

		parentTeam := &GithubTeam{
			Name:    "parentTeam",
			Slug:    "parentteam",
			Members: []string{"existing_owner"},
			Id:      1,
		}

		parentTeamOwners := &GithubTeam{
			Name:    "parentteam" + config.Config.GoliacTeamOwnerSuffix,
			Slug:    "parentteam" + config.Config.GoliacTeamOwnerSuffix,
			Members: []string{"existing_owner"},
			Id:      1,
		}

		childTeam := &GithubTeam{
			Name:    "childTeam",
			Slug:    "childteam",
			Members: []string{"existing_owner"},
			Id:      2,
		}

		childTeamOwners := &GithubTeam{
			Name:    "childTeam" + config.Config.GoliacTeamOwnerSuffix,
			Slug:    "childteam" + config.Config.GoliacTeamOwnerSuffix,
			Members: []string{"existing_owner"},
			Id:      2,
		}

		remote.teams["parentteam"] = parentTeam
		remote.teams["parentteam"+config.Config.GoliacTeamOwnerSuffix] = parentTeamOwners
		remote.teams["childteam"] = childTeam
		remote.teams["childteam"+config.Config.GoliacTeamOwnerSuffix] = childTeamOwners

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 0 parent updated
		assert.Equal(t, 0, len(recorder.TeamParentUpdated))
	})

	t.Run("happy path: add parent to a team", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}

		lParentTeam := &entity.Team{}
		lParentTeam.Name = "parentTeam"
		lParentTeam.Spec.Owners = []string{"existing_owner"}
		lParentTeam.Spec.Members = []string{}
		local.teams["parentTeam"] = lParentTeam

		lChildTeam := &entity.Team{}
		lChildTeam.Name = "childTeam"
		lChildTeam.Spec.Owners = []string{"existing_owner"}
		lChildTeam.Spec.Members = []string{}
		// let's put the child under the parent
		parent := "parentTeam"
		lChildTeam.ParentTeam = &parent

		local.teams["childTeam"] = lChildTeam

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}

		parentTeam := &GithubTeam{
			Name:    "parentTeam",
			Slug:    "parentteam",
			Members: []string{"existing_owner"},
			Id:      1,
		}

		parentTeamOwners := &GithubTeam{
			Name:    "parentteam" + config.Config.GoliacTeamOwnerSuffix,
			Slug:    "parentteam" + config.Config.GoliacTeamOwnerSuffix,
			Members: []string{"existing_owner"},
			Id:      1,
		}

		childTeam := &GithubTeam{
			Name:    "childTeam",
			Slug:    "childteam",
			Members: []string{"existing_owner"},
			Id:      2,
		}

		childTeamOwners := &GithubTeam{
			Name:    "childTeam" + config.Config.GoliacTeamOwnerSuffix,
			Slug:    "childteam" + config.Config.GoliacTeamOwnerSuffix,
			Members: []string{"existing_owner"},
			Id:      2,
		}

		remote.teams["parentteam"] = parentTeam
		remote.teams["parentteam"+config.Config.GoliacTeamOwnerSuffix] = parentTeamOwners
		remote.teams["childteam"] = childTeam
		remote.teams["childteam"+config.Config.GoliacTeamOwnerSuffix] = childTeamOwners

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 team parent updated
		assert.Equal(t, 1, len(recorder.TeamParentUpdated))
	})

	t.Run("happy path: removed team", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()
		repoconfig := &config.RepositoryConfig{}
		repoconfig.DestructiveOperations.AllowDestructiveTeams = true
		r := NewGoliacReconciliatorImpl(recorder, repoconfig)
		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		removing := &GithubTeam{
			Name:    "removing",
			Slug:    "removing",
			Members: []string{"existing_owner", "existing_owner"},
		}
		remote.teams["removing"] = removing

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 team deleted
		assert.Equal(t, 1, len(recorder.TeamDeleted))
	})

	t.Run("happy path: new repo without owner", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()
		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		newRepo := &entity.Repository{}
		newRepo.Name = "new"
		newRepo.Spec.Readers = []string{}
		newRepo.Spec.Writers = []string{}
		local.repos["new"] = newRepo

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 repo created
		assert.Equal(t, 1, len(recorder.RepositoryCreated))
	})

	t.Run("happy path: new repo with owner", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		newRepo := &entity.Repository{}
		newRepo.Name = "new"
		newRepo.Spec.Readers = []string{}
		newRepo.Spec.Writers = []string{}
		owner := "existing"
		newRepo.Owner = &owner
		local.repos["new"] = newRepo

		existingTeam := &entity.Team{}
		existingTeam.Name = "existing"
		existingTeam.Spec.Owners = []string{"existing_owner"}
		existingTeam.Spec.Members = []string{"existing_member"}
		local.teams["existing"] = existingTeam

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 repo created
		assert.Equal(t, 1, len(recorder.RepositoryCreated))
	})

	t.Run("happy path: existing repo with new owner (from read to write)", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		lRepo := &entity.Repository{}
		lRepo.Name = "myrepo"
		lRepo.Spec.Readers = []string{}
		lRepo.Spec.Writers = []string{}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo

		existingTeam := &entity.Team{}
		existingTeam.Name = "existing"
		existingTeam.Spec.Owners = []string{"existing_owner"}
		existingTeam.Spec.Members = []string{"existing_member"}
		local.teams["existing"] = existingTeam

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing
		rRepo := GithubRepository{
			Name:           "myrepo",
			ExternalUsers:  map[string]string{},
			BoolProperties: map[string]bool{},
		}
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "READ",
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 team updated
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 1, len(recorder.RepositoryTeamRemoved))
		assert.Equal(t, 1, len(recorder.RepositoryTeamAdded))
		assert.Equal(t, 0, len(recorder.RepositoryTeamUpdated))
	})

	t.Run("happy path: existing repo without new owner but with everyone team", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{
			EveryoneTeamEnabled: true,
		}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		lRepo := &entity.Repository{}
		lRepo.Name = "myrepo"
		lRepo.Spec.Readers = []string{}
		lRepo.Spec.Writers = []string{}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo

		existingTeam := &entity.Team{}
		existingTeam.Name = "existing"
		existingTeam.Spec.Owners = []string{"existing_owner"}
		existingTeam.Spec.Members = []string{"existing_member"}
		local.teams["existing"] = existingTeam

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing
		rRepo := GithubRepository{
			Name:           "myrepo",
			ExternalUsers:  map[string]string{},
			BoolProperties: map[string]bool{},
		}
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "WRITE",
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 team updated
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 0, len(recorder.RepositoryTeamRemoved))
		// we have a new "everyone" team for the repository
		assert.Equal(t, 1, len(recorder.RepositoryTeamAdded))
		assert.Equal(t, 0, len(recorder.RepositoryTeamUpdated))
	})

	t.Run("happy path: add a team to an existing repo", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		lRepo := &entity.Repository{}
		lRepo.Name = "myrepo"
		lRepo.Spec.Readers = []string{"reader"}
		lRepo.Spec.Writers = []string{}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo

		existingTeam := &entity.Team{}
		existingTeam.Name = "existing"
		existingTeam.Spec.Owners = []string{"existing_owner"}
		existingTeam.Spec.Members = []string{"existing_member"}
		local.teams["existing"] = existingTeam

		readerTeam := &entity.Team{}
		readerTeam.Name = "reader"
		readerTeam.Spec.Owners = []string{"existing_owner"}
		readerTeam.Spec.Members = []string{"existing_member"}
		local.teams["reader"] = readerTeam

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		reader := &GithubTeam{
			Name:    "reader",
			Slug:    "reader",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing
		remote.teams["reader"] = reader
		rRepo := GithubRepository{
			Name:           "myrepo",
			ExternalUsers:  map[string]string{},
			BoolProperties: map[string]bool{},
		}
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "ADMIN",
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 team added
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 0, len(recorder.RepositoryTeamRemoved))
		assert.Equal(t, 1, len(recorder.RepositoryTeamAdded))
		assert.Equal(t, 0, len(recorder.RepositoryTeamUpdated))
	})

	t.Run("happy path: remove a team from an existing repo", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		lRepo := &entity.Repository{}
		lRepo.Name = "myrepo"
		lRepo.Spec.Readers = []string{}
		lRepo.Spec.Writers = []string{}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo

		existingTeam := &entity.Team{}
		existingTeam.Name = "existing"
		existingTeam.Spec.Owners = []string{"existing_owner"}
		existingTeam.Spec.Members = []string{"existing_member"}
		local.teams["existing"] = existingTeam

		readerTeam := &entity.Team{}
		readerTeam.Name = "reader"
		readerTeam.Spec.Owners = []string{"existing_owner"}
		readerTeam.Spec.Members = []string{"existing_member"}
		local.teams["reader"] = readerTeam

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		reader := &GithubTeam{
			Name:    "reader",
			Slug:    "reader",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing
		remote.teams["reader"] = reader
		rRepo := GithubRepository{
			Name:           "myrepo",
			ExternalUsers:  map[string]string{},
			BoolProperties: map[string]bool{},
		}
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "WRITE",
		}
		remote.teamsrepos["reader"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["reader"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "WRITE",
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 team removed
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 1, len(recorder.RepositoryTeamRemoved))
		assert.Equal(t, 0, len(recorder.RepositoryTeamAdded))
		assert.Equal(t, 0, len(recorder.RepositoryTeamUpdated))
	})

	t.Run("happy path: remove a team member", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		lRepo := &entity.Repository{}
		lRepo.Name = "myrepo"
		lRepo.Spec.Readers = []string{}
		lRepo.Spec.Writers = []string{}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo
		existingUser := entity.User{}
		existingUser.Spec.GithubID = "existing_member"
		local.users["existing_member"] = &existingUser
		existingOwner := entity.User{}
		existingOwner.Spec.GithubID = "existing_owner"
		local.users["existing_owner"] = &existingOwner

		existingTeam := &entity.Team{}
		existingTeam.Name = "existing"
		existingTeam.Spec.Owners = []string{"existing_owner"}
		existingTeam.Spec.Members = []string{}
		local.teams["existing"] = existingTeam

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing
		rRepo := GithubRepository{
			Name:           "myrepo",
			ExternalUsers:  map[string]string{},
			BoolProperties: map[string]bool{},
		}
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "WRITE",
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 member removed
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 0, len(recorder.RepositoryTeamRemoved))
		assert.Equal(t, 0, len(recorder.RepositoryTeamAdded))
		assert.Equal(t, 0, len(recorder.RepositoryTeamUpdated))
		assert.Equal(t, 1, len(recorder.TeamMemberRemoved))
	})

	t.Run("happy path: update a team member from maintainer to member", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		lRepo := &entity.Repository{}
		lRepo.Name = "myrepo"
		lRepo.Spec.Readers = []string{}
		lRepo.Spec.Writers = []string{}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo
		existingUser := entity.User{}
		existingUser.Spec.GithubID = "existing_member"
		local.users["existing_member"] = &existingUser
		existingOwner := entity.User{}
		existingOwner.Spec.GithubID = "existing_owner"
		local.users["existing_owner"] = &existingOwner

		existingTeam := &entity.Team{}
		existingTeam.Name = "existing"
		existingTeam.Spec.Owners = []string{"existing_owner"}
		existingTeam.Spec.Members = []string{"existing_member"}
		local.teams["existing"] = existingTeam

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		existing := &GithubTeam{
			Name:        "existing",
			Slug:        "existing",
			Members:     []string{"existing_member"},
			Maintainers: []string{"existing_owner"},
		}
		remote.teams["existing"] = existing
		rRepo := GithubRepository{
			Name:           "myrepo",
			ExternalUsers:  map[string]string{},
			BoolProperties: map[string]bool{},
		}
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "WRITE",
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 member removed
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 0, len(recorder.RepositoryTeamRemoved))
		assert.Equal(t, 0, len(recorder.RepositoryTeamAdded))
		assert.Equal(t, 0, len(recorder.RepositoryTeamUpdated))
		fmt.Println("**debug", recorder.TeamMemberRemoved)
		assert.Equal(t, 0, len(recorder.TeamMemberRemoved))
		assert.Equal(t, 1, len(recorder.TeamMemberUpdated))
	})

	t.Run("happy path: add a team AND add it to an existing repo", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()
		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		lRepo := &entity.Repository{}
		lRepo.Name = "myrepo"
		lRepo.Spec.Readers = []string{"reader"}
		lRepo.Spec.Writers = []string{}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo

		existingTeam := &entity.Team{}
		existingTeam.Name = "existing"
		existingTeam.Spec.Owners = []string{"existing_owner"}
		existingTeam.Spec.Members = []string{"existing_member"}
		local.teams["existing"] = existingTeam

		readerTeam := &entity.Team{}
		readerTeam.Name = "reader"
		readerTeam.Spec.Owners = []string{"existing_owner"}
		readerTeam.Spec.Members = []string{"existing_member"}
		local.teams["reader"] = readerTeam

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing
		rRepo := GithubRepository{
			Name:           "myrepo",
			ExternalUsers:  map[string]string{},
			BoolProperties: map[string]bool{},
		}
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "WRITE",
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 repo updated
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 0, len(recorder.RepositoryTeamRemoved))
		assert.Equal(t, 1, len(recorder.RepositoryTeamAdded))
	})

	t.Run("happy path: add a externally managed team AND add it to an existing repo", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()
		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}
		lRepo := &entity.Repository{}
		lRepo.Name = "myrepo"
		lRepo.Spec.Readers = []string{"newerTeam"}
		lRepo.Spec.Writers = []string{}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo

		existingTeam := &entity.Team{}
		existingTeam.Name = "existing"
		existingTeam.Spec.Owners = []string{"existing_owner"}
		existingTeam.Spec.Members = []string{"existing_member"}
		local.teams["existing"] = existingTeam

		newerTeam := &entity.Team{}
		newerTeam.Name = "newerTeam"
		newerTeam.Spec.ExternallyManaged = true
		local.teams["newerTeam"] = newerTeam

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner", "existing_member"},
		}
		remote.teams["existing"] = existing
		remote.teams["existing"+config.Config.GoliacTeamOwnerSuffix] = existing
		rRepo := GithubRepository{
			Name:           "myrepo",
			ExternalUsers:  map[string]string{},
			BoolProperties: map[string]bool{},
		}
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "WRITE",
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 repo updated
		assert.Equal(t, 1, len(recorder.TeamsCreated)) // the newerTeam-goliac-owners team
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 0, len(recorder.RepositoryTeamRemoved))
		assert.Equal(t, 1, len(recorder.RepositoryTeamAdded))
	})

	t.Run("happy path: existing repo with new external write collaborator", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users:     make(map[string]*entity.User),
			externals: make(map[string]*entity.User),
			teams:     make(map[string]*entity.Team),
			repos:     make(map[string]*entity.Repository),
		}
		outside1 := entity.User{}
		outside1.Name = "outside1"
		outside1.Spec.GithubID = "outside1-githubid"
		local.externals["outside1"] = &outside1

		lRepo := &entity.Repository{}
		lRepo.Name = "myrepo"
		lRepo.Spec.Readers = []string{}
		lRepo.Spec.Writers = []string{}
		lRepo.Spec.ExternalUserWriters = []string{"outside1"}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo

		existingTeam := &entity.Team{}
		existingTeam.Name = "existing"
		existingTeam.Spec.Owners = []string{"existing_owner"}
		existingTeam.Spec.Members = []string{}
		local.teams["existing"] = existingTeam

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner"},
		}
		remote.teams["existing"] = existing
		rRepo := GithubRepository{
			Name:           "myrepo",
			ExternalUsers:  make(map[string]string),
			BoolProperties: make(map[string]bool),
		}
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "WRITE",
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 team updated
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 0, len(recorder.RepositoryTeamRemoved))
		assert.Equal(t, 0, len(recorder.RepositoryTeamAdded))
		assert.Equal(t, 0, len(recorder.RepositoryTeamUpdated))
		assert.Equal(t, 1, len(recorder.RepositoriesSetExternalUser))
		assert.Equal(t, 0, len(recorder.RepositoriesRemoveExternalUser))
	})

	t.Run("happy path: existing repo with deleted external write collaborator", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users:     make(map[string]*entity.User),
			externals: make(map[string]*entity.User),
			teams:     make(map[string]*entity.Team),
			repos:     make(map[string]*entity.Repository),
		}

		lRepo := &entity.Repository{}
		lRepo.Name = "myrepo"
		lRepo.Spec.Readers = []string{}
		lRepo.Spec.Writers = []string{}
		lRepo.Spec.ExternalUserWriters = []string{}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo

		existingTeam := &entity.Team{}
		existingTeam.Name = "existing"
		existingTeam.Spec.Owners = []string{"existing_owner"}
		existingTeam.Spec.Members = []string{}
		local.teams["existing"] = existingTeam

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner"},
		}
		remote.teams["existing"] = existing
		rRepo := GithubRepository{
			Name:           "myrepo",
			ExternalUsers:  make(map[string]string),
			BoolProperties: make(map[string]bool),
		}
		rRepo.ExternalUsers["outside1-githubid"] = "WRITE"
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "WRITE",
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 team updated
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 0, len(recorder.RepositoryTeamRemoved))
		assert.Equal(t, 0, len(recorder.RepositoryTeamAdded))
		assert.Equal(t, 0, len(recorder.RepositoryTeamUpdated))
		assert.Equal(t, 0, len(recorder.RepositoriesSetExternalUser))
		assert.Equal(t, 1, len(recorder.RepositoriesRemoveExternalUser))
	})

	t.Run("happy path: existing repo with changed external write collaborator (from read to write)", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users:     make(map[string]*entity.User),
			externals: make(map[string]*entity.User),
			teams:     make(map[string]*entity.Team),
			repos:     make(map[string]*entity.Repository),
		}

		outside1 := entity.User{}
		outside1.Name = "outside1"
		outside1.Spec.GithubID = "outside1-githubid"
		local.externals["outside1"] = &outside1

		lRepo := &entity.Repository{}
		lRepo.Name = "myrepo"
		lRepo.Spec.Readers = []string{}
		lRepo.Spec.Writers = []string{}
		lRepo.Spec.ExternalUserWriters = []string{}
		lRepo.Spec.ExternalUserReaders = []string{"outside1"}
		lowner := "existing"
		lRepo.Owner = &lowner
		local.repos["myrepo"] = lRepo

		existingTeam := &entity.Team{}
		existingTeam.Name = "existing"
		existingTeam.Spec.Owners = []string{"existing_owner"}
		existingTeam.Spec.Members = []string{}
		local.teams["existing"] = existingTeam

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		existing := &GithubTeam{
			Name:    "existing",
			Slug:    "existing",
			Members: []string{"existing_owner"},
		}
		remote.teams["existing"] = existing
		rRepo := GithubRepository{
			Name:           "myrepo",
			ExternalUsers:  make(map[string]string),
			BoolProperties: make(map[string]bool),
		}
		rRepo.ExternalUsers["outside1-githubid"] = "WRITE"
		remote.repos["myrepo"] = &rRepo

		remote.teamsrepos["existing"] = make(map[string]*GithubTeamRepo)
		remote.teamsrepos["existing"]["myrepo"] = &GithubTeamRepo{
			Name:       "myrepo",
			Permission: "WRITE",
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 team updated
		assert.Equal(t, 0, len(recorder.RepositoryCreated))
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 0, len(recorder.RepositoryTeamRemoved))
		assert.Equal(t, 0, len(recorder.RepositoryTeamAdded))
		assert.Equal(t, 0, len(recorder.RepositoryTeamUpdated))
		assert.Equal(t, 1, len(recorder.RepositoriesSetExternalUser))
		assert.Equal(t, 0, len(recorder.RepositoriesRemoveExternalUser))
	})

	t.Run("happy path: removed repo without destructive operation", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		removing := &GithubRepository{
			Name: "removing",
		}
		remote.repos["removing"] = removing

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 repo deleted
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
	})

	t.Run("happy path: removed repo with archive_on_delete", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()
		repoconfig := &config.RepositoryConfig{
			ArchiveOnDelete: true,
		}
		repoconfig.DestructiveOperations.AllowDestructiveRepositories = true
		r := NewGoliacReconciliatorImpl(recorder, repoconfig)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		removing := &GithubRepository{
			Name:           "removing",
			ExternalUsers:  map[string]string{},
			BoolProperties: map[string]bool{},
		}
		remote.repos["removing"] = removing

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 repo deleted
		assert.Equal(t, 0, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 1, len(toArchive))
	})

	t.Run("happy path: removed repo withou archive_on_delete", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()
		repoconfig := &config.RepositoryConfig{
			ArchiveOnDelete: false,
		}
		repoconfig.DestructiveOperations.AllowDestructiveRepositories = true
		r := NewGoliacReconciliatorImpl(recorder, repoconfig)

		local := GoliacLocalMock{
			users: make(map[string]*entity.User),
			teams: make(map[string]*entity.Team),
			repos: make(map[string]*entity.Repository),
		}

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}
		removing := &GithubRepository{
			Name:           "removing",
			ExternalUsers:  map[string]string{},
			BoolProperties: map[string]bool{},
		}
		remote.repos["removing"] = removing

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 repo deleted
		assert.Equal(t, 1, len(recorder.RepositoriesDeleted))
		assert.Equal(t, 0, len(toArchive))
	})
}

func TestReconciliationRulesets(t *testing.T) {

	t.Run("happy path: no new ruleset in goliac conf", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()
		repoconf := config.RepositoryConfig{}

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users:    make(map[string]*entity.User),
			teams:    make(map[string]*entity.Team),
			repos:    make(map[string]*entity.Repository),
			rulesets: make(map[string]*entity.RuleSet),
		}

		newRuleset := &entity.RuleSet{}
		newRuleset.Name = "new"
		newRuleset.Spec.Enforcement = "evaluate"
		newRuleset.Spec.Rules = append(newRuleset.Spec.Rules, struct {
			Ruletype   string
			Parameters entity.RuleSetParameters
		}{
			"required_signatures", entity.RuleSetParameters{},
		})
		local.rulesets["new"] = newRuleset

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 ruleset created
		assert.Equal(t, 0, len(recorder.RuleSetCreated))
		assert.Equal(t, 0, len(recorder.RuleSetUpdated))
		assert.Equal(t, 0, len(recorder.RuleSetDeleted))
	})

	t.Run("happy path: new ruleset", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{
			Rulesets: make([]struct {
				Pattern string
				Ruleset string
			}, 0),
		}
		repoconf.Rulesets = append(repoconf.Rulesets, struct {
			Pattern string
			Ruleset string
		}{
			Pattern: ".*",
			Ruleset: "new",
		})

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users:    make(map[string]*entity.User),
			teams:    make(map[string]*entity.Team),
			repos:    make(map[string]*entity.Repository),
			rulesets: make(map[string]*entity.RuleSet),
		}

		newRuleset := &entity.RuleSet{}
		newRuleset.Name = "new"
		newRuleset.Spec.Enforcement = "evaluate"
		newRuleset.Spec.Rules = append(newRuleset.Spec.Rules, struct {
			Ruletype   string
			Parameters entity.RuleSetParameters
		}{
			"required_signatures", entity.RuleSetParameters{},
		})
		local.rulesets["new"] = newRuleset

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 ruleset created
		assert.Equal(t, 1, len(recorder.RuleSetCreated))
		assert.Equal(t, 0, len(recorder.RuleSetUpdated))
		assert.Equal(t, 0, len(recorder.RuleSetDeleted))
	})

	t.Run("happy path: update ruleset (enforcement)", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{
			Rulesets: make([]struct {
				Pattern string
				Ruleset string
			}, 0),
		}
		repoconf.Rulesets = append(repoconf.Rulesets, struct {
			Pattern string
			Ruleset string
		}{
			Pattern: ".*",
			Ruleset: "update",
		})

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users:    make(map[string]*entity.User),
			teams:    make(map[string]*entity.Team),
			repos:    make(map[string]*entity.Repository),
			rulesets: make(map[string]*entity.RuleSet),
		}

		lRuleset := &entity.RuleSet{}
		lRuleset.Name = "update"
		lRuleset.Spec.Enforcement = "evaluate"
		lRuleset.Spec.Rules = append(lRuleset.Spec.Rules, struct {
			Ruletype   string
			Parameters entity.RuleSetParameters
		}{
			"required_signatures", entity.RuleSetParameters{},
		})
		local.rulesets["update"] = lRuleset

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}

		rRuleset := &GithubRuleSet{
			Name:        "update",
			Enforcement: "active",
			Rules:       make(map[string]entity.RuleSetParameters),
		}
		rRuleset.Rules["required_signatures"] = entity.RuleSetParameters{}
		remote.rulesets["update"] = rRuleset

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 ruleset created
		assert.Equal(t, 0, len(recorder.RuleSetCreated))
		assert.Equal(t, 1, len(recorder.RuleSetUpdated))
		assert.Equal(t, 0, len(recorder.RuleSetDeleted))
	})

	t.Run("happy path: delete ruleset", func(t *testing.T) {
		recorder := NewReconciliatorListenerRecorder()

		repoconf := config.RepositoryConfig{
			Rulesets: make([]struct {
				Pattern string
				Ruleset string
			}, 0),
		}
		repoconf.DestructiveOperations.AllowDestructiveRulesets = true

		r := NewGoliacReconciliatorImpl(recorder, &repoconf)

		local := GoliacLocalMock{
			users:    make(map[string]*entity.User),
			teams:    make(map[string]*entity.Team),
			repos:    make(map[string]*entity.Repository),
			rulesets: make(map[string]*entity.RuleSet),
		}

		remote := GoliacRemoteMock{
			users:      make(map[string]string),
			teams:      make(map[string]*GithubTeam),
			repos:      make(map[string]*GithubRepository),
			teamsrepos: make(map[string]map[string]*GithubTeamRepo),
			rulesets:   make(map[string]*GithubRuleSet),
			appids:     make(map[string]int),
		}

		rRuleset := &GithubRuleSet{
			Name:        "delete",
			Enforcement: "active",
			Rules:       make(map[string]entity.RuleSetParameters),
		}
		rRuleset.Rules["required_signatures"] = entity.RuleSetParameters{}
		remote.rulesets["delete"] = rRuleset

		toArchive := make(map[string]*GithubRepoComparable)
		r.Reconciliate(context.TODO(), &local, &remote, "teams", false, toArchive)

		// 1 ruleset created
		assert.Equal(t, 0, len(recorder.RuleSetCreated))
		assert.Equal(t, 0, len(recorder.RuleSetUpdated))
		assert.Equal(t, 1, len(recorder.RuleSetDeleted))
	})
}
