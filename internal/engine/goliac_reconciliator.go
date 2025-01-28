package engine

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/gosimple/slug"
	"github.com/sirupsen/logrus"
)

type UnmanagedResources struct {
	Users                  map[string]bool
	ExternallyManagedTeams map[string]bool
	Teams                  map[string]bool
	Repositories           map[string]bool
	RuleSets               map[int]bool
}

/*
 * GoliacReconciliator is here to sync the local state to the remote state
 */
type GoliacReconciliator interface {
	Reconciliate(ctx context.Context, local GoliacLocal, remote GoliacRemote, teamreponame string, dryrun bool, goliacAdminSlug string, reposToArchive map[string]*GithubRepoComparable) (*UnmanagedResources, error)
}

type GoliacReconciliatorImpl struct {
	executor   ReconciliatorExecutor
	repoconfig *config.RepositoryConfig
	unmanaged  *UnmanagedResources
}

func NewGoliacReconciliatorImpl(executor ReconciliatorExecutor, repoconfig *config.RepositoryConfig) GoliacReconciliator {
	return &GoliacReconciliatorImpl{
		executor:   executor,
		repoconfig: repoconfig,
		unmanaged:  nil,
	}
}

func (r *GoliacReconciliatorImpl) Reconciliate(ctx context.Context, local GoliacLocal, remote GoliacRemote, teamsreponame string, dryrun bool, goliacAdminSlug string, reposToArchive map[string]*GithubRepoComparable) (*UnmanagedResources, error) {
	rremote := NewMutableGoliacRemoteImpl(ctx, remote)
	r.Begin(ctx, dryrun)
	unmanaged := &UnmanagedResources{
		Users:                  make(map[string]bool),
		ExternallyManagedTeams: make(map[string]bool),
		Teams:                  make(map[string]bool),
		Repositories:           make(map[string]bool),
		RuleSets:               make(map[int]bool),
	}
	r.unmanaged = unmanaged

	err := r.reconciliateUsers(ctx, local, rremote, dryrun, unmanaged)
	if err != nil {
		r.Rollback(ctx, dryrun, err)
		return nil, err
	}

	allOwners, err := r.reconciliateTeams(ctx, local, rremote, dryrun)
	if err != nil {
		r.Rollback(ctx, dryrun, err)
		return nil, err
	}

	// the teams repos must have the goliacAdminSlug and all owners as writer
	// the writing operation will be controlled by the CODEOWNERS file
	repos := local.Repositories()
	teamsRepo := &entity.Repository{}
	teamsRepo.Spec.Writers = append(allOwners, goliacAdminSlug)
	teamsRepo.Spec.IsPublic = false
	repos[teamsreponame] = teamsRepo

	err = r.reconciliateRepositories(ctx, local, rremote, teamsreponame, dryrun, reposToArchive)
	if err != nil {
		r.Rollback(ctx, dryrun, err)
		return nil, err
	}

	if remote.IsEnterprise() {
		err = r.reconciliateRulesets(ctx, local, rremote, r.repoconfig, dryrun)
		if err != nil {
			r.Rollback(ctx, dryrun, err)
			return nil, err
		}
	}

	return r.unmanaged, r.Commit(ctx, dryrun)
}

/*
 * This function sync teams and team's members
 */
func (r *GoliacReconciliatorImpl) reconciliateUsers(ctx context.Context, local GoliacLocal, remote *MutableGoliacRemoteImpl, dryrun bool, unmanaged *UnmanagedResources) error {
	ghUsers := remote.Users()

	rUsers := make(map[string]string)
	for u := range ghUsers {
		rUsers[u] = u
	}

	for _, lUser := range local.Users() {
		user, ok := rUsers[lUser.Spec.GithubID]

		if !ok {
			// deal with non existing remote user
			r.AddUserToOrg(ctx, dryrun, remote, lUser.Spec.GithubID)
		} else {
			delete(rUsers, user)
		}
	}

	// remaining (GH) users (aka not found locally)
	for _, rUser := range rUsers {
		// DELETE User
		r.RemoveUserFromOrg(ctx, dryrun, remote, rUser)
	}
	return nil
}

type GithubTeamComparable struct {
	Name        string
	Slug        string
	Members     []string
	Maintainers []string
	ParentTeam  *string
}

/*
 * This function sync teams and team's members
 */
func (r *GoliacReconciliatorImpl) reconciliateTeams(ctx context.Context, local GoliacLocal, remote *MutableGoliacRemoteImpl, dryrun bool) ([]string, error) {
	ghTeams := remote.Teams()
	rUsers := remote.Users()
	allOwners := []string{}

	ghTeamsPerId := make(map[int]*GithubTeam)
	for _, v := range ghTeams {
		ghTeamsPerId[v.Id] = v
	}

	rTeams := make(map[string]*GithubTeamComparable)
	for k, v := range ghTeams {
		members := make([]string, len(v.Members))
		copy(members, v.Members)
		maintainers := []string{}

		// let's filter admin from the maintainers
		for _, m := range v.Maintainers {
			if rUsers[m] == "ADMIN" {
				members = append(members, m)
			} else {
				maintainers = append(maintainers, m)
			}
		}

		team := &GithubTeamComparable{
			Name:        v.Name,
			Slug:        v.Slug,
			Members:     members,
			Maintainers: maintainers,
			ParentTeam:  nil,
		}
		if v.ParentTeam != nil {
			if parent, ok := ghTeamsPerId[*v.ParentTeam]; ok {
				team.ParentTeam = &parent.Name
			}
		}

		// key is the team's slug
		rTeams[k] = team
	}

	// prepare the teams we want (regular and "-goliac-owners"/config.Config.GoliacTeamOwnerSuffix)
	slugTeams := make(map[string]*GithubTeamComparable)
	lTeams := local.Teams()
	lUsers := local.Users()

	for teamname, teamvalue := range lTeams {
		teamslug := slug.Make(teamname)
		allOwners = append(allOwners, teamslug+config.Config.GoliacTeamOwnerSuffix)

		// if the team is externally managed, we don't want to touch it
		// we just remove it from the list
		if teamvalue.Spec.ExternallyManaged {
			// let's add it to the special -goliac-owners
			membersOwners := []string{}
			membersMaintainers := []string{}
			if rt, ok := rTeams[teamslug]; ok {
				membersOwners = append(membersOwners, rt.Members...)
				membersMaintainers = append(membersMaintainers, rt.Maintainers...)
			}
			team := &GithubTeamComparable{
				Name:        teamslug + config.Config.GoliacTeamOwnerSuffix,
				Slug:        teamslug + config.Config.GoliacTeamOwnerSuffix,
				Members:     membersOwners,
				Maintainers: membersMaintainers,
			}
			slugTeams[teamslug+config.Config.GoliacTeamOwnerSuffix] = team

			r.unmanaged.ExternallyManagedTeams[teamslug] = true
			delete(rTeams, teamslug)
			continue
		}

		members := []string{}
		membersOwners := []string{}
		// teamvalue.Spec.Members are not github id
		for _, m := range teamvalue.Spec.Members {
			if u, ok := lUsers[m]; ok {
				members = append(members, u.Spec.GithubID)
			}
		}
		for _, m := range teamvalue.Spec.Owners {
			if u, ok := lUsers[m]; ok {
				members = append(members, u.Spec.GithubID)
				membersOwners = append(membersOwners, u.Spec.GithubID)
			}
		}

		team := &GithubTeamComparable{
			Name:    teamname,
			Slug:    teamslug,
			Members: members,
		}
		if teamvalue.ParentTeam != nil {
			parentTeam := slug.Make(*teamvalue.ParentTeam)
			team.ParentTeam = &parentTeam
		}
		slugTeams[teamslug] = team

		// owners
		team = &GithubTeamComparable{
			Name:        teamslug + config.Config.GoliacTeamOwnerSuffix,
			Slug:        teamslug + config.Config.GoliacTeamOwnerSuffix,
			Members:     membersOwners,
			Maintainers: []string{},
		}
		slugTeams[teamslug+config.Config.GoliacTeamOwnerSuffix] = team
	}

	// adding the "everyone" team
	if r.repoconfig.EveryoneTeamEnabled {
		everyone := GithubTeamComparable{
			Name:    "everyone",
			Slug:    "everyone",
			Members: []string{},
		}
		for u := range local.Users() {
			everyone.Members = append(everyone.Members, u)
		}
		slugTeams["everyone"] = &everyone
	}

	// now we compare local (slugTeams) and remote (rTeams)

	compareTeam := func(lTeam *GithubTeamComparable, rTeam *GithubTeamComparable) bool {
		if res, _, _ := entity.StringArrayEquivalent(lTeam.Members, rTeam.Members); !res {
			return false
		}
		if res, _, _ := entity.StringArrayEquivalent(lTeam.Maintainers, rTeam.Maintainers); !res {
			return false
		}
		if (lTeam.ParentTeam == nil && rTeam.ParentTeam != nil) ||
			(lTeam.ParentTeam != nil && rTeam.ParentTeam == nil) ||
			(lTeam.ParentTeam != nil && rTeam.ParentTeam != nil && *lTeam.ParentTeam != *rTeam.ParentTeam) {
			return false
		}

		return true
	}

	onAdded := func(key string, lTeam *GithubTeamComparable, rTeam *GithubTeamComparable) {
		// CREATE team

		// it is possible that parent team will be added in 2 pass
		var parentTeam *int
		if lTeam.ParentTeam != nil && ghTeams[*lTeam.ParentTeam] != nil {
			parentTeam = &ghTeams[*lTeam.ParentTeam].Id
		}
		r.CreateTeam(ctx, dryrun, remote, lTeam.Name, lTeam.Name, parentTeam, lTeam.Members)
	}

	onRemoved := func(key string, lTeam *GithubTeamComparable, rTeam *GithubTeamComparable) {
		// DELETE team
		r.DeleteTeam(ctx, dryrun, remote, rTeam.Slug)
	}

	onChanged := func(slugTeam string, lTeam *GithubTeamComparable, rTeam *GithubTeamComparable) {
		// change membership from maintainers to members

		rmaintainers := make([]string, len(rTeam.Maintainers))
		copy(rmaintainers, rTeam.Maintainers)

		for _, r_maintainer := range rmaintainers {
			found := false
			for _, l_maintainer := range lTeam.Maintainers {
				if r_maintainer == l_maintainer {
					found = true
					break
				}
			}
			if !found {
				// let's downgrade the maintainer to member
				r.UpdateTeamChangeMaintainerToMember(ctx, dryrun, remote, slugTeam, r_maintainer)
				for i, m := range rTeam.Maintainers {
					if m == r_maintainer {
						rTeam.Maintainers = append(rTeam.Maintainers[:i], rTeam.Maintainers[i+1:]...)
						break
					}
				}
				rTeam.Members = append(rTeam.Members, r_maintainer)
			}
		}

		// membership change
		if res, _, _ := entity.StringArrayEquivalent(lTeam.Members, rTeam.Members); !res {
			localMembers := make(map[string]bool)
			for _, m := range lTeam.Members {
				localMembers[m] = true
			}

			for _, m := range rTeam.Members {
				if _, ok := localMembers[m]; !ok {
					// REMOVE team member
					r.UpdateTeamRemoveMember(ctx, dryrun, remote, slugTeam, m)
				} else {
					delete(localMembers, m)
				}
			}
			for m := range localMembers {
				// ADD team member
				r.UpdateTeamAddMember(ctx, dryrun, remote, slugTeam, m, "member")
			}
		}

		// parent team change
		if (lTeam.ParentTeam == nil && rTeam.ParentTeam != nil) ||
			(lTeam.ParentTeam != nil && rTeam.ParentTeam == nil) ||
			(lTeam.ParentTeam != nil && rTeam.ParentTeam != nil && *lTeam.ParentTeam != *rTeam.ParentTeam) {

			var parentTeam *int
			parentTeamName := ""
			if lTeam.ParentTeam != nil && ghTeams[*lTeam.ParentTeam] != nil {
				parentTeam = &ghTeams[*lTeam.ParentTeam].Id
				parentTeamName = *lTeam.ParentTeam
			}

			r.UpdateTeamSetParent(ctx, dryrun, remote, slugTeam, parentTeam, parentTeamName)
		}
	}

	CompareEntities(slugTeams, rTeams, compareTeam, onAdded, onRemoved, onChanged)

	return allOwners, nil
}

type GithubRepoComparable struct {
	BoolProperties      map[string]bool
	Writers             []string
	Readers             []string
	ExternalUserReaders []string // githubids
	ExternalUserWriters []string // githubids
}

/*
 * This function sync repositories and team's repositories permissions
 * It returns the list of deleted repos that must not be deleted but archived
 */
func (r *GoliacReconciliatorImpl) reconciliateRepositories(ctx context.Context, local GoliacLocal, remote *MutableGoliacRemoteImpl, teamsreponame string, dryrun bool, toArchive map[string]*GithubRepoComparable) error {
	ghRepos := remote.Repositories()
	rRepos := make(map[string]*GithubRepoComparable)
	for k, v := range ghRepos {
		repo := &GithubRepoComparable{
			BoolProperties:      map[string]bool{},
			Writers:             []string{},
			Readers:             []string{},
			ExternalUserReaders: []string{},
			ExternalUserWriters: []string{},
		}
		for pk, pv := range v.BoolProperties {
			repo.BoolProperties[pk] = pv
		}

		for cGithubid, cPermission := range v.ExternalUsers {
			if cPermission == "WRITE" {
				repo.ExternalUserWriters = append(repo.ExternalUserWriters, cGithubid)
			} else {
				repo.ExternalUserReaders = append(repo.ExternalUserReaders, cGithubid)
			}
		}

		// rRepos[slug.Make(k)] = repo
		rRepos[k] = repo
	}

	// on the remote object, I have teams->repos, and I need repos->teams
	for t, repos := range remote.TeamRepositories() {
		for r, p := range repos {
			if rr, ok := rRepos[r]; ok {
				if p.Permission == "ADMIN" || p.Permission == "WRITE" {
					rr.Writers = append(rr.Writers, t)
				} else {
					rr.Readers = append(rr.Readers, t)
				}
			}
		}
	}

	lRepos := make(map[string]*GithubRepoComparable)
	for reponame, lRepo := range local.Repositories() {
		writers := make([]string, 0)
		for _, w := range lRepo.Spec.Writers {
			writers = append(writers, slug.Make(w))
		}
		// add the team owner's name ;-)
		if lRepo.Owner != nil {
			writers = append(writers, slug.Make(*lRepo.Owner))
		}
		readers := make([]string, 0)
		for _, r := range lRepo.Spec.Readers {
			readers = append(readers, slug.Make(r))
		}

		// special case for the Goliac "teams" repo
		if reponame == teamsreponame {
			for teamname := range local.Teams() {
				writers = append(writers, slug.Make(teamname)+config.Config.GoliacTeamOwnerSuffix)
			}
		}

		// adding the "everyone" team to each repository
		if r.repoconfig.EveryoneTeamEnabled {
			readers = append(readers, "everyone")
		}

		// adding exernal reader/writer
		eReaders := make([]string, 0)
		for _, r := range lRepo.Spec.ExternalUserReaders {
			if user, ok := local.ExternalUsers()[r]; ok {
				eReaders = append(eReaders, user.Spec.GithubID)
			}
		}

		eWriters := make([]string, 0)
		for _, w := range lRepo.Spec.ExternalUserWriters {
			if user, ok := local.ExternalUsers()[w]; ok {
				eWriters = append(eWriters, user.Spec.GithubID)
			}
		}

		// lRepos[slug.Make(reponame)] = &GithubRepoComparable{
		lRepos[reponame] = &GithubRepoComparable{
			BoolProperties: map[string]bool{
				"private":                !lRepo.Spec.IsPublic,
				"archived":               lRepo.Archived,
				"allow_auto_merge":       lRepo.Spec.AllowAutoMerge,
				"delete_branch_on_merge": lRepo.Spec.DeleteBranchOnMerge,
				"allow_update_branch":    lRepo.Spec.AllowUpdateBranch,
			},
			Readers:             readers,
			Writers:             writers,
			ExternalUserReaders: eReaders,
			ExternalUserWriters: eWriters,
		}
	}

	// now we compare local (slugTeams) and remote (rTeams)

	compareRepos := func(lRepo *GithubRepoComparable, rRepo *GithubRepoComparable) bool {
		for lk, lv := range lRepo.BoolProperties {
			if rv, ok := rRepo.BoolProperties[lk]; !ok || rv != lv {
				return false
			}
		}

		if res, _, _ := entity.StringArrayEquivalent(lRepo.Readers, rRepo.Readers); !res {
			return false
		}

		if res, _, _ := entity.StringArrayEquivalent(lRepo.Writers, rRepo.Writers); !res {
			return false
		}

		if res, _, _ := entity.StringArrayEquivalent(lRepo.ExternalUserReaders, rRepo.ExternalUserReaders); !res {
			return false
		}

		if res, _, _ := entity.StringArrayEquivalent(lRepo.ExternalUserWriters, rRepo.ExternalUserWriters); !res {
			return false
		}

		return true
	}

	onChanged := func(reponame string, lRepo *GithubRepoComparable, rRepo *GithubRepoComparable) {
		// reconciliate repositories boolean properties
		for lk, lv := range lRepo.BoolProperties {
			if rv, ok := rRepo.BoolProperties[lk]; !ok || rv != lv {
				r.UpdateRepositoryUpdateBoolProperty(ctx, dryrun, remote, reponame, lk, lv)
			}
		}

		if res, readToRemove, readToAdd := entity.StringArrayEquivalent(lRepo.Readers, rRepo.Readers); !res {
			for _, teamSlug := range readToAdd {
				r.UpdateRepositoryAddTeamAccess(ctx, dryrun, remote, reponame, teamSlug, "pull")
			}
			for _, teamSlug := range readToRemove {
				r.UpdateRepositoryRemoveTeamAccess(ctx, dryrun, remote, reponame, teamSlug)
			}
		}

		if res, writeToRemove, writeToAdd := entity.StringArrayEquivalent(lRepo.Writers, rRepo.Writers); !res {
			for _, teamSlug := range writeToAdd {
				r.UpdateRepositoryAddTeamAccess(ctx, dryrun, remote, reponame, teamSlug, "push")
			}
			for _, teamSlug := range writeToRemove {
				r.UpdateRepositoryRemoveTeamAccess(ctx, dryrun, remote, reponame, teamSlug)
			}
		}

		resEreader, ereaderToRemove, ereaderToAdd := entity.StringArrayEquivalent(lRepo.ExternalUserReaders, rRepo.ExternalUserReaders)
		resEWriter, ewriteToRemove, ewriteToAdd := entity.StringArrayEquivalent(lRepo.ExternalUserWriters, rRepo.ExternalUserWriters)

		if !resEreader {
			for _, eReader := range ereaderToRemove {
				// check if it is added in the writers
				found := false
				for _, eWriter := range ewriteToAdd {
					if eWriter == eReader {
						found = true
						break
					}
				}
				if !found {
					r.UpdateRepositoryRemoveExternalUser(ctx, dryrun, remote, reponame, eReader)
				}
			}
			for _, eReader := range ereaderToAdd {
				r.UpdateRepositorySetExternalUser(ctx, dryrun, remote, reponame, eReader, "pull")
			}
		}

		if !resEWriter {
			for _, eWriter := range ewriteToRemove {
				// check if it is added in the writers
				found := false
				for _, eReader := range ereaderToAdd {
					if eReader == eWriter {
						found = true
						break
					}
				}
				if !found {
					r.UpdateRepositoryRemoveExternalUser(ctx, dryrun, remote, reponame, eWriter)
				}
			}
			for _, eWriter := range ewriteToAdd {
				r.UpdateRepositorySetExternalUser(ctx, dryrun, remote, reponame, eWriter, "push")
			}
		}

	}

	onAdded := func(reponame string, lRepo *GithubRepoComparable, rRepo *GithubRepoComparable) {
		// CREATE repository

		// if the repo was just archived in a previous commit and we "resume it"
		if aRepo, ok := toArchive[reponame]; ok {
			delete(toArchive, reponame)
			r.UpdateRepositoryUpdateBoolProperty(ctx, dryrun, remote, reponame, "archived", false)
			// calling onChanged to update the repository permissions
			onChanged(reponame, aRepo, rRepo)
		} else {
			r.CreateRepository(ctx, dryrun, remote, reponame, reponame, lRepo.Writers, lRepo.Readers, lRepo.BoolProperties)
		}
	}

	onRemoved := func(reponame string, lRepo *GithubRepoComparable, rRepo *GithubRepoComparable) {
		// here we have a repository that is not listed in the teams repository.
		// we should call DeleteRepository (that will delete if AllowDestructiveRepositories is on).
		// but if we have ArchiveOnDelete...
		if r.repoconfig.ArchiveOnDelete {
			if r.repoconfig.DestructiveOperations.AllowDestructiveRepositories {
				r.UpdateRepositoryUpdateBoolProperty(ctx, dryrun, remote, reponame, "archived", true)
				toArchive[reponame] = rRepo
			} else {
				r.unmanaged.Repositories[reponame] = true
			}
		} else {
			r.DeleteRepository(ctx, dryrun, remote, reponame)
		}
	}

	CompareEntities(lRepos, rRepos, compareRepos, onAdded, onRemoved, onChanged)

	return nil
}

func (r *GoliacReconciliatorImpl) reconciliateRulesets(ctx context.Context, local GoliacLocal, remote *MutableGoliacRemoteImpl, conf *config.RepositoryConfig, dryrun bool) error {
	repositories := local.Repositories()

	lgrs := map[string]*GithubRuleSet{}
	// prepare local comparable
	for _, confrs := range conf.Rulesets {
		match, err := regexp.Compile(confrs.Pattern)
		if err != nil {
			return fmt.Errorf("not able to parse ruleset regular expression %s: %v", confrs.Pattern, err)
		}
		rs, ok := local.RuleSets()[confrs.Ruleset]
		if !ok {
			return fmt.Errorf("not able to find ruleset %s definition", confrs.Ruleset)
		}

		grs := GithubRuleSet{
			Name:        rs.Name,
			Enforcement: rs.Spec.Enforcement,
			BypassApps:  map[string]string{},
			OnInclude:   rs.Spec.On.Include,
			OnExclude:   rs.Spec.On.Exclude,
			Rules:       map[string]entity.RuleSetParameters{},
		}
		for _, b := range rs.Spec.BypassApps {
			grs.BypassApps[b.AppName] = b.Mode
		}
		for _, r := range rs.Spec.Rules {
			grs.Rules[r.Ruletype] = r.Parameters
		}
		for reponame := range repositories {
			if match.Match([]byte(slug.Make(reponame))) {
				grs.Repositories = append(grs.Repositories, slug.Make(reponame))
			}
		}
		lgrs[rs.Name] = &grs
	}

	// prepare remote comparable
	rgrs := remote.RuleSets()

	// prepare the diff computation

	compareRulesets := func(lrs *GithubRuleSet, rrs *GithubRuleSet) bool {
		if lrs.Enforcement != rrs.Enforcement {
			return false
		}
		if len(lrs.BypassApps) != len(rrs.BypassApps) {
			return false
		}
		for k, v := range lrs.BypassApps {
			if rrs.BypassApps[k] != v {
				return false
			}
		}
		if res, _, _ := entity.StringArrayEquivalent(lrs.OnInclude, rrs.OnInclude); !res {
			return false
		}
		if res, _, _ := entity.StringArrayEquivalent(lrs.OnExclude, rrs.OnExclude); !res {
			return false
		}
		if len(lrs.Rules) != len(rrs.Rules) {
			return false
		}
		for k, v := range lrs.Rules {
			if !entity.CompareRulesetParameters(k, v, rrs.Rules[k]) {
				return false
			}
		}
		if res, _, _ := entity.StringArrayEquivalent(lrs.Repositories, rrs.Repositories); !res {
			return false
		}

		return true
	}

	onAdded := func(rulesetname string, lRuleset *GithubRuleSet, rRuleset *GithubRuleSet) {
		// CREATE ruleset

		r.AddRuleset(ctx, dryrun, lRuleset)
	}

	onRemoved := func(rulesetname string, lRuleset *GithubRuleSet, rRuleset *GithubRuleSet) {
		// DELETE ruleset
		r.DeleteRuleset(ctx, dryrun, rRuleset.Id)
	}

	onChanged := func(rulesetname string, lRuleset *GithubRuleSet, rRuleset *GithubRuleSet) {
		// UPDATE ruleset
		lRuleset.Id = rRuleset.Id
		r.UpdateRuleset(ctx, dryrun, lRuleset)
	}

	CompareEntities(lgrs, rgrs, compareRulesets, onAdded, onRemoved, onChanged)

	return nil
}

func (r *GoliacReconciliatorImpl) AddUserToOrg(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, ghuserid string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "add_user_to_org"}).Infof("ghuserid: %s", ghuserid)
	remote.AddUserToOrg(ghuserid)
	if r.executor != nil {
		r.executor.AddUserToOrg(ctx, dryrun, ghuserid)
	}
}

func (r *GoliacReconciliatorImpl) RemoveUserFromOrg(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, ghuserid string) {
	if r.repoconfig.DestructiveOperations.AllowDestructiveUsers {
		logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "remove_user_from_org"}).Infof("ghuserid: %s", ghuserid)
		remote.RemoveUserFromOrg(ghuserid)
		if r.executor != nil {
			r.executor.RemoveUserFromOrg(ctx, dryrun, ghuserid)
		}
	} else {
		r.unmanaged.Users[ghuserid] = true
	}
}

func (r *GoliacReconciliatorImpl) CreateTeam(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, teamname string, description string, parentTeam *int, members []string) {
	parenTeamId := "nil"
	if parentTeam != nil {
		parenTeamId = fmt.Sprintf("%d", *parentTeam)
	}

	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "create_team"}).Infof("teamname: %s, parentTeam : %s, members: %s", teamname, parenTeamId, strings.Join(members, ","))
	remote.CreateTeam(teamname, description, members)
	if r.executor != nil {
		r.executor.CreateTeam(ctx, dryrun, teamname, description, parentTeam, members)
	}
}
func (r *GoliacReconciliatorImpl) UpdateTeamAddMember(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string, ghuserid string, role string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_team_add_member"}).Infof("teamslug: %s, ghuserid: %s, role: %s", teamslug, ghuserid, role)
	remote.UpdateTeamAddMember(teamslug, ghuserid, "member")
	if r.executor != nil {
		r.executor.UpdateTeamAddMember(ctx, dryrun, teamslug, ghuserid, "member")
	}
}
func (r *GoliacReconciliatorImpl) UpdateTeamRemoveMember(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string, ghuserid string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_team_remove_member"}).Infof("teamslug: %s, ghuserid: %s", teamslug, ghuserid)
	remote.UpdateTeamRemoveMember(teamslug, ghuserid)
	if r.executor != nil {
		r.executor.UpdateTeamRemoveMember(ctx, dryrun, teamslug, ghuserid)
	}
}
func (r *GoliacReconciliatorImpl) UpdateTeamChangeMaintainerToMember(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string, ghuserid string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_team_change_maintainer_to_member"}).Infof("teamslug: %s, ghuserid: %s", teamslug, ghuserid)
	remote.UpdateTeamUpdateMember(teamslug, ghuserid, "member")
	if r.executor != nil {
		r.executor.UpdateTeamUpdateMember(ctx, dryrun, teamslug, ghuserid, "member")
	}
}
func (r *GoliacReconciliatorImpl) UpdateTeamSetParent(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string, parentTeam *int, parentTeamName string) {
	parenTeamId := "nil"
	if parentTeam != nil {
		parenTeamId = fmt.Sprintf("%d", *parentTeam)
	}

	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_team_parentteam"}).Infof("teamslug: %s, parentteam: %s (%s)", teamslug, parenTeamId, parentTeamName)
	remote.UpdateTeamSetParent(ctx, dryrun, teamslug, parentTeam)
	if r.executor != nil {
		r.executor.UpdateTeamSetParent(ctx, dryrun, teamslug, parentTeam)
	}
}
func (r *GoliacReconciliatorImpl) DeleteTeam(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string) {
	if r.repoconfig.DestructiveOperations.AllowDestructiveTeams {
		logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_team"}).Infof("teamslug: %s", teamslug)
		remote.DeleteTeam(teamslug)
		if r.executor != nil {
			r.executor.DeleteTeam(ctx, dryrun, teamslug)
		}
	} else {
		r.unmanaged.Teams[teamslug] = true
	}
}
func (r *GoliacReconciliatorImpl) CreateRepository(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, descrition string, writers []string, readers []string, boolProperties map[string]bool) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "create_repository"}).Infof("repositoryname: %s, readers: %s, writers: %s, boolProperties: %v", reponame, strings.Join(readers, ","), strings.Join(writers, ","), boolProperties)
	remote.CreateRepository(reponame, reponame, writers, readers, boolProperties)
	if r.executor != nil {
		r.executor.CreateRepository(ctx, dryrun, reponame, reponame, writers, readers, boolProperties)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryAddTeamAccess(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, teamslug string, permission string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_add_team"}).Infof("repositoryname: %s, teamslug: %s, permission: %s", reponame, teamslug, permission)
	remote.UpdateRepositoryAddTeamAccess(reponame, teamslug, permission)
	if r.executor != nil {
		r.executor.UpdateRepositoryAddTeamAccess(ctx, dryrun, reponame, teamslug, permission)
	}
}

func (r *GoliacReconciliatorImpl) UpdateRepositoryUpdateTeamAccess(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, teamslug string, permission string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_update_team"}).Infof("repositoryname: %s, teamslug:%s, permission: %s", reponame, teamslug, permission)
	remote.UpdateRepositoryUpdateTeamAccess(reponame, teamslug, permission)
	if r.executor != nil {
		r.executor.UpdateRepositoryUpdateTeamAccess(ctx, dryrun, reponame, teamslug, permission)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryRemoveTeamAccess(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, teamslug string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_remove_team"}).Infof("repositoryname: %s, teamslug:%s", reponame, teamslug)
	remote.UpdateRepositoryRemoveTeamAccess(reponame, teamslug)
	if r.executor != nil {
		r.executor.UpdateRepositoryRemoveTeamAccess(ctx, dryrun, reponame, teamslug)
	}
}

func (r *GoliacReconciliatorImpl) DeleteRepository(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string) {
	if r.repoconfig.DestructiveOperations.AllowDestructiveRepositories {
		logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_repository"}).Infof("repositoryname: %s", reponame)
		remote.DeleteRepository(reponame)
		if r.executor != nil {
			r.executor.DeleteRepository(ctx, dryrun, reponame)
		}
	} else {
		r.unmanaged.Repositories[reponame] = true
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryUpdateBoolProperty(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, propertyName string, propertyValue bool) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_update_bool_property"}).Infof("repositoryname: %s %s:%v", reponame, propertyName, propertyValue)
	remote.UpdateRepositoryUpdateBoolProperty(reponame, propertyName, propertyValue)
	if r.executor != nil {
		r.executor.UpdateRepositoryUpdateBoolProperty(ctx, dryrun, reponame, propertyName, propertyValue)
	}
}
func (r *GoliacReconciliatorImpl) AddRuleset(ctx context.Context, dryrun bool, ruleset *GithubRuleSet) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "add_ruleset"}).Infof("ruleset: %s (id: %d) enforcement: %s", ruleset.Name, ruleset.Id, ruleset.Enforcement)
	if r.executor != nil {
		r.executor.AddRuleset(ctx, dryrun, ruleset)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRuleset(ctx context.Context, dryrun bool, ruleset *GithubRuleSet) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_ruleset"}).Infof("ruleset: %s (id: %d) enforcement: %s", ruleset.Name, ruleset.Id, ruleset.Enforcement)
	if r.executor != nil {
		r.executor.UpdateRuleset(ctx, dryrun, ruleset)
	}
}
func (r *GoliacReconciliatorImpl) DeleteRuleset(ctx context.Context, dryrun bool, rulesetid int) {
	if r.repoconfig.DestructiveOperations.AllowDestructiveRulesets {
		logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_ruleset"}).Infof("ruleset id:%d", rulesetid)
		if r.executor != nil {
			r.executor.DeleteRuleset(ctx, dryrun, rulesetid)
		}
	} else {
		r.unmanaged.RuleSets[rulesetid] = true
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositorySetExternalUser(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, collaboatorGithubId string, permission string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_set_external_user"}).Infof("repositoryname: %s collaborator:%s permission:%s", reponame, collaboatorGithubId, permission)
	remote.UpdateRepositorySetExternalUser(reponame, collaboatorGithubId, permission)
	if r.executor != nil {
		r.executor.UpdateRepositorySetExternalUser(ctx, dryrun, reponame, collaboatorGithubId, permission)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryRemoveExternalUser(ctx context.Context, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, collaboatorGithubId string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_remove_external_user"}).Infof("repositoryname: %s collaborator:%s", reponame, collaboatorGithubId)
	remote.UpdateRepositoryRemoveExternalUser(reponame, collaboatorGithubId)
	if r.executor != nil {
		r.executor.UpdateRepositoryRemoveExternalUser(ctx, dryrun, reponame, collaboatorGithubId)
	}
}
func (r *GoliacReconciliatorImpl) Begin(ctx context.Context, dryrun bool) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun}).Debugf("reconciliation begin")
	if r.executor != nil {
		r.executor.Begin(dryrun)
	}
}
func (r *GoliacReconciliatorImpl) Rollback(ctx context.Context, dryrun bool, err error) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun}).Debugf("reconciliation rollback")
	if r.executor != nil {
		r.executor.Rollback(dryrun, err)
	}
}
func (r *GoliacReconciliatorImpl) Commit(ctx context.Context, dryrun bool) error {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun}).Debugf("reconciliation commit")
	if r.executor != nil {
		return r.executor.Commit(ctx, dryrun)
	}
	return nil
}
