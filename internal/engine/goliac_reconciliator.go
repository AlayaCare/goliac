package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/goliac-project/goliac/internal/config"
	"github.com/goliac-project/goliac/internal/entity"
	"github.com/goliac-project/goliac/internal/observability"
	"github.com/goliac-project/goliac/internal/utils"
	"github.com/sirupsen/logrus"
)

type UnmanagedResources struct {
	Users                  map[string]bool
	ExternallyManagedTeams map[string]bool
	Teams                  map[string]bool
	Repositories           map[string]bool
	RuleSets               map[string]bool
}

/*
 * GoliacReconciliator is here to sync the local state to the remote state
 */
type GoliacReconciliator interface {
	Reconciliate(ctx context.Context, errorCollector *observability.ErrorCollection, local GoliacReconciliatorDatasource, remote GoliacReconciliatorDatasource, isEnterprise bool, dryrun bool, manageGithubVariables bool, manageGithubAutolinks bool) (*UnmanagedResources, map[string]*GithubRepoComparable, map[string]string, error)
}

type GoliacReconciliatorImpl struct {
	executor            ReconciliatorExecutor
	reconciliatorFilter ReconciliatorFilter
	repoconfig          *config.RepositoryConfig
	unmanaged           *UnmanagedResources
}

func NewGoliacReconciliatorImpl(isEntreprise bool, executor ReconciliatorExecutor, repoconfig *config.RepositoryConfig) GoliacReconciliator {
	return &GoliacReconciliatorImpl{
		executor:            executor,
		reconciliatorFilter: NewReconciliatorFilter(isEntreprise, repoconfig),
		repoconfig:          repoconfig,
		unmanaged:           nil,
	}
}

func (r *GoliacReconciliatorImpl) Reconciliate(ctx context.Context, errorCollector *observability.ErrorCollection, local GoliacReconciliatorDatasource, remote GoliacReconciliatorDatasource, isEnterprise bool, dryrun bool, manageGithubVariables bool, manageGithubAutolinks bool) (*UnmanagedResources, map[string]*GithubRepoComparable, map[string]string, error) {
	rremote, err := NewMutableGoliacRemoteImpl(ctx, remote)
	if err != nil {
		return nil, nil, nil, err
	}
	r.Begin(ctx, dryrun)
	unmanaged := &UnmanagedResources{
		Users:                  make(map[string]bool),
		ExternallyManagedTeams: make(map[string]bool),
		Teams:                  make(map[string]bool),
		Repositories:           make(map[string]bool),
		RuleSets:               make(map[string]bool),
	}
	r.unmanaged = unmanaged

	err = r.reconciliateUsers(ctx, errorCollector, local, rremote, dryrun)
	if err != nil {
		r.Rollback(ctx, dryrun, err)
		return nil, nil, nil, err
	}

	err = r.reconciliateTeams(ctx, errorCollector, local, rremote, dryrun)
	if err != nil {
		r.Rollback(ctx, dryrun, err)
		return nil, nil, nil, err
	}

	reposToArchive, reposToRename, err := r.reconciliateRepositories(ctx, errorCollector, local, rremote, dryrun, manageGithubVariables, manageGithubAutolinks)
	if err != nil {
		r.Rollback(ctx, dryrun, err)
		return nil, nil, nil, err
	}

	if isEnterprise {
		err = r.reconciliateRulesets(ctx, errorCollector, local, rremote, r.repoconfig, dryrun)
		if err != nil {
			r.Rollback(ctx, dryrun, err)
			return nil, nil, nil, err
		}
	}

	return r.unmanaged, reposToArchive, reposToRename, r.Commit(ctx, errorCollector, dryrun)
}

/*
 * This function sync teams and team's members
 */
func (r *GoliacReconciliatorImpl) reconciliateUsers(ctx context.Context, errorCollector *observability.ErrorCollection, local GoliacReconciliatorDatasource, remote *MutableGoliacRemoteImpl, dryrun bool) error {
	rUsers := remote.Users()

	for _, lUser := range local.Users() {
		user, ok := rUsers[lUser]

		if !ok {
			// deal with non existing remote user
			r.AddUserToOrg(ctx, errorCollector, dryrun, remote, lUser)
		} else {
			delete(rUsers, user)
		}
	}

	// remaining (GH) users (aka not found locally)
	for _, rUser := range rUsers {
		// DELETE User
		r.RemoveUserFromOrg(ctx, errorCollector, dryrun, remote, rUser)
	}
	return nil
}

type GithubTeamComparable struct {
	Name              string
	Slug              string
	Members           []string
	Maintainers       []string // in Github, there are 2 types of roles in a team: maintainers and members. We will remove maintainers from the team
	ExternallyManaged bool
	ParentTeam        *string
	// not comparable
	Id int // only on remote object
}

/*
This function sync teams and team's members,
*/
func (r *GoliacReconciliatorImpl) reconciliateTeams(ctx context.Context, errorCollector *observability.ErrorCollection, local GoliacReconciliatorDatasource, remote *MutableGoliacRemoteImpl, dryrun bool) error {
	lTeams, _, err := local.Teams()
	if err != nil {
		return err
	}

	rTeams := remote.Teams()

	// we need to populate -goliac-owners teams with the owners of the team
	// if the team is externally managed
	for slugteamname, team := range rTeams {
		if !strings.HasSuffix(slugteamname, config.Config.GoliacTeamOwnerSuffix) {
			// regular team
			if regularTeam, ok := lTeams[slugteamname]; ok {
				if regularTeam.ExternallyManaged {
					regularTeam.Members = append(regularTeam.Members, team.Members...)

					// let's search for the -goliac-owners team
					if ownersTeam, ok := lTeams[slugteamname+config.Config.GoliacTeamOwnerSuffix]; ok {
						ownersTeam.Members = append(ownersTeam.Members, team.Members...)
					}
				}
			}
		}
	}

	// now we compare local and remote

	compareTeam := func(teamname string, lTeam *GithubTeamComparable, rTeam *GithubTeamComparable) bool {
		if !lTeam.ExternallyManaged {
			if res, _, _ := entity.StringArrayEquivalent(lTeam.Members, rTeam.Members); !res {
				return false
			}
			if res, _, _ := entity.StringArrayEquivalent(lTeam.Maintainers, rTeam.Maintainers); !res {
				return false
			}
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
		if lTeam.ParentTeam != nil && rTeams[*lTeam.ParentTeam] != nil {
			parentTeam = &rTeams[*lTeam.ParentTeam].Id
		}
		r.CreateTeam(ctx, errorCollector, dryrun, remote, lTeam.Name, lTeam.Name, parentTeam, lTeam.Members)
	}

	onRemoved := func(key string, lTeam *GithubTeamComparable, rTeam *GithubTeamComparable) {
		// DELETE team
		r.DeleteTeam(ctx, errorCollector, dryrun, remote, rTeam.Slug)
	}

	onChanged := func(slugTeam string, lTeam *GithubTeamComparable, rTeam *GithubTeamComparable) {
		// change membership from maintainers to members
		if !lTeam.ExternallyManaged {
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
					r.UpdateTeamChangeMaintainerToMember(ctx, errorCollector, dryrun, remote, slugTeam, r_maintainer)
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
						r.UpdateTeamRemoveMember(ctx, errorCollector, dryrun, remote, slugTeam, m)
					} else {
						delete(localMembers, m)
					}
				}
				for m := range localMembers {
					// ADD team member
					r.UpdateTeamAddMember(ctx, errorCollector, dryrun, remote, slugTeam, m, "member")
				}
			}
		}

		// parent team change
		if (lTeam.ParentTeam == nil && rTeam.ParentTeam != nil) ||
			(lTeam.ParentTeam != nil && rTeam.ParentTeam == nil) ||
			(lTeam.ParentTeam != nil && rTeam.ParentTeam != nil && *lTeam.ParentTeam != *rTeam.ParentTeam) {

			var parentTeam *int
			parentTeamName := ""
			if lTeam.ParentTeam != nil && rTeams[*lTeam.ParentTeam] != nil {
				parentTeam = &rTeams[*lTeam.ParentTeam].Id
				parentTeamName = *lTeam.ParentTeam
			}

			r.UpdateTeamSetParent(ctx, errorCollector, dryrun, remote, slugTeam, parentTeam, parentTeamName)
		}
	}

	CompareEntities(lTeams, rTeams, compareTeam, onAdded, onRemoved, onChanged)

	return nil
}

type GithubRepoComparable struct {
	Visibility          string
	BoolProperties      map[string]bool
	Writers             []string
	Readers             []string
	ExternalUserReaders []string // githubids
	ExternalUserWriters []string // githubids
	InternalUsers       []string // githubids
	Rulesets            map[string]*GithubRuleSet
	BranchProtections   map[string]*GithubBranchProtection
	DefaultBranchName   string
	ActionVariables     MappedEntityLazyLoader[string]
	Environments        MappedEntityLazyLoader[*GithubEnvironment]
	Autolinks           MappedEntityLazyLoader[*GithubAutolink]
	// not comparable
	IsFork   bool
	ForkFrom string
}

type GithubEnvironment struct {
	Name      string
	Variables map[string]string
}

type GithubAutolink struct {
	Id             int
	KeyPrefix      string
	UrlTemplate    string
	IsAlphanumeric bool
}

/*
This function sync repositories and team's repositories permissions
It returns the list of deleted repos that must not be deleted but archived

It returns:
- reposToArchive: list of repos that must not be deleted but archived
- reposToRename: list of repos that must be renamed
- err: error if any
*/
func (r *GoliacReconciliatorImpl) reconciliateRepositories(
	ctx context.Context,
	errorCollector *observability.ErrorCollection,
	local GoliacReconciliatorDatasource,
	remote *MutableGoliacRemoteImpl,
	dryrun bool,
	manageGithubVariables bool,
	manageGithubAutolinks bool,
) (map[string]*GithubRepoComparable, map[string]string, error) {

	reposToArchive := make(map[string]*GithubRepoComparable)
	reposToRename := make(map[string]string)

	// let's start with the local cloned github-teams repo
	lRepos, toRename, err := local.Repositories()
	if err != nil {
		return reposToArchive, reposToRename, err
	}

	for reponame, renameTo := range toRename {

		r.RenameRepository(ctx, errorCollector, dryrun, remote, reponame, renameTo)

		// in the post action we have to also update the git repository
		lRepos[renameTo] = lRepos[reponame]
		delete(lRepos, reponame)
		reposToRename[reponame] = renameTo
	}

	// let's get the remote now
	rRepos := remote.Repositories()

	// now we compare local (slugTeams) and remote (rTeams)

	compareRepos := func(reponame string, lRepo *GithubRepoComparable, rRepo *GithubRepoComparable) bool {
		archived := lRepo.BoolProperties["archived"]
		if !archived {
			//
			// "nested" rulesets comparison
			//
			onRulesetAdded := func(rulename string, lRuleset *GithubRuleSet, rRuleset *GithubRuleSet) {
				// CREATE repo ruleset
				r.AddRepositoryRuleset(ctx, errorCollector, dryrun, reponame, lRuleset)
			}
			onRulesetRemoved := func(rulename string, lRuleset *GithubRuleSet, rRuleset *GithubRuleSet) {
				// DELETE repo ruleset
				r.DeleteRepositoryRuleset(ctx, errorCollector, dryrun, reponame, rRuleset)
			}
			onRulesetChange := func(rulename string, lRuleset *GithubRuleSet, rRuleset *GithubRuleSet) {
				// UPDATE ruleset
				lRuleset.Id = rRuleset.Id
				r.UpdateRepositoryRuleset(ctx, errorCollector, dryrun, reponame, lRuleset)
			}
			CompareEntities(lRepo.Rulesets, rRepo.Rulesets, compareRulesets, onRulesetAdded, onRulesetRemoved, onRulesetChange)

			//
			// "nested" branchprotections comparison
			//
			onBranchProtectionAdded := func(rulename string, lBp *GithubBranchProtection, rBp *GithubBranchProtection) {
				// CREATE repo branchprotection
				r.AddRepositoryBranchProtection(ctx, errorCollector, dryrun, reponame, lBp)
			}
			onBranchProtectionRemoved := func(rulename string, lBp *GithubBranchProtection, rBp *GithubBranchProtection) {
				// DELETE repo branchprotection
				r.DeleteRepositoryBranchProtection(ctx, errorCollector, dryrun, reponame, rBp)
			}
			onBranchProtectionChange := func(rulename string, lBp *GithubBranchProtection, rBp *GithubBranchProtection) {
				// UPDATE branchprotection
				lBp.Id = rBp.Id
				r.UpdateRepositoryBranchProtection(ctx, errorCollector, dryrun, reponame, lBp)
			}
			CompareEntities(lRepo.BranchProtections, rRepo.BranchProtections, compareBranchProtections, onBranchProtectionAdded, onBranchProtectionRemoved, onBranchProtectionChange)

			if manageGithubVariables {
				//
				// "nested" environments comparison
				//
				onEnvironmentAdded := func(environment string, lEnv *GithubEnvironment, rEnv *GithubEnvironment) {
					// CREATE repo environment
					r.AddRepositoryEnvironment(ctx, errorCollector, dryrun, remote, reponame, environment)
				}
				onEnvironmentChange := func(environment string, lEnv *GithubEnvironment, rEnv *GithubEnvironment) {
					// UPDATE repo environment

					// Check for removed or changed keys
					for name, value := range lEnv.Variables {
						if rValue, ok := rEnv.Variables[name]; !ok {
							r.AddRepositoryEnvironmentVariable(ctx, errorCollector, dryrun, remote, reponame, environment, name, value)
						} else if rValue != value {
							r.UpdateRepositoryEnvironmentVariable(ctx, errorCollector, dryrun, remote, reponame, environment, name, value)
						}
					}

					// Check for added keys
					for name := range rEnv.Variables {
						if _, ok := lEnv.Variables[name]; !ok {
							r.DeleteRepositoryEnvironmentVariable(ctx, errorCollector, dryrun, remote, reponame, environment, name)
						}
					}
				}
				onEnvironmentRemoved := func(environment string, lEnv *GithubEnvironment, rEnv *GithubEnvironment) {
					// DELETE repo environment
					r.DeleteRepositoryEnvironment(ctx, errorCollector, dryrun, remote, reponame, environment)
				}
				CompareEntities(lRepo.Environments.GetEntity(), rRepo.Environments.GetEntity(), compareEnvironments, onEnvironmentAdded, onEnvironmentRemoved, onEnvironmentChange)
			}

			if manageGithubAutolinks {
				// nested autolinks comparison IF it is defined locally
				if lRepo.Autolinks != nil {
					onAutolinkAdded := func(autolinkprefix string, lal *GithubAutolink, ral *GithubAutolink) {
						r.AddRepositoryAutolink(ctx, errorCollector, dryrun, remote, reponame, lal)
					}
					onAutolinkRemoved := func(autolinkprefix string, lal *GithubAutolink, ral *GithubAutolink) {
						r.DeleteRepositoryAutolink(ctx, errorCollector, dryrun, remote, reponame, ral.Id)
					}
					onAutolinkChange := func(autolinkname string, lal *GithubAutolink, ral *GithubAutolink) {
						r.UpdateRepositoryAutolink(ctx, errorCollector, dryrun, remote, reponame, ral)
					}
					CompareEntities(lRepo.Autolinks.GetEntity(), rRepo.Autolinks.GetEntity(), compareAutolinks, onAutolinkAdded, onAutolinkRemoved, onAutolinkChange)
				}
			}
		}

		//
		// now, comparing repo properties
		//
		for lk, lv := range lRepo.BoolProperties {
			if rv, ok := rRepo.BoolProperties[lk]; !ok || rv != lv {
				return false
			}
		}

		if lRepo.Visibility != rRepo.Visibility {
			// check back if the remote repo is a fork
			// in this case, we cannot change the visibility
			if rr, ok := rRepos[reponame]; ok {
				if !rr.IsFork {
					return false
				}
			} else {
				return false
			}
		}

		if lRepo.DefaultBranchName != "" && lRepo.DefaultBranchName != rRepo.DefaultBranchName {
			return false
		}

		if res, _, _ := entity.StringArrayEquivalent(lRepo.Readers, rRepo.Readers); !res {
			return false
		}

		if res, _, _ := entity.StringArrayEquivalent(lRepo.Writers, rRepo.Writers); !res {
			return false
		}

		if len(rRepo.InternalUsers) != 0 {
			return false
		}

		if res, _, _ := entity.StringArrayEquivalent(lRepo.ExternalUserReaders, rRepo.ExternalUserReaders); !res {
			return false
		}

		if res, _, _ := entity.StringArrayEquivalent(lRepo.ExternalUserWriters, rRepo.ExternalUserWriters); !res {
			return false
		}

		if manageGithubVariables {
			if !archived {
				if !utils.DeepEqualUnordered(lRepo.ActionVariables.GetEntity(), rRepo.ActionVariables.GetEntity()) {
					return false
				}
			}
		}

		return true
	}

	onChanged := func(reponame string, lRepo *GithubRepoComparable, rRepo *GithubRepoComparable) {
		// reconciliate repositories boolean properties
		for lk, lv := range lRepo.BoolProperties {
			if rv, ok := rRepo.BoolProperties[lk]; !ok || rv != lv {
				r.UpdateRepositoryUpdateProperty(ctx, errorCollector, dryrun, remote, reponame, lk, lv)
			}
		}
		if lRepo.Visibility != rRepo.Visibility {
			// check back if the remote repo is a fork
			// in this case, we cannot change the visibility
			if rr, ok := rRepos[reponame]; ok {
				if !rr.IsFork {
					r.UpdateRepositoryUpdateProperty(ctx, errorCollector, dryrun, remote, reponame, "visibility", lRepo.Visibility)
				}
			} else {
				r.UpdateRepositoryUpdateProperty(ctx, errorCollector, dryrun, remote, reponame, "visibility", lRepo.Visibility)
			}
		}
		if lRepo.DefaultBranchName != "" && lRepo.DefaultBranchName != rRepo.DefaultBranchName {
			r.UpdateRepositoryUpdateProperty(ctx, errorCollector, dryrun, remote, reponame, "default_branch", lRepo.DefaultBranchName)
		}

		if res, readToRemove, readToAdd := entity.StringArrayEquivalent(lRepo.Readers, rRepo.Readers); !res {
			for _, teamSlug := range readToAdd {
				r.UpdateRepositoryAddTeamAccess(ctx, errorCollector, dryrun, remote, reponame, teamSlug, "pull")
			}
			for _, teamSlug := range readToRemove {
				r.UpdateRepositoryRemoveTeamAccess(ctx, errorCollector, dryrun, remote, reponame, teamSlug)
			}
		}

		if res, writeToRemove, writeToAdd := entity.StringArrayEquivalent(lRepo.Writers, rRepo.Writers); !res {
			for _, teamSlug := range writeToAdd {
				r.UpdateRepositoryAddTeamAccess(ctx, errorCollector, dryrun, remote, reponame, teamSlug, "push")
			}
			for _, teamSlug := range writeToRemove {
				r.UpdateRepositoryRemoveTeamAccess(ctx, errorCollector, dryrun, remote, reponame, teamSlug)
			}
		}

		// internal users
		for _, internalUser := range rRepo.InternalUsers {
			r.UpdateRepositoryRemoveInternalUser(ctx, errorCollector, dryrun, remote, reponame, internalUser)
		}

		// external users
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
					r.UpdateRepositoryRemoveExternalUser(ctx, errorCollector, dryrun, remote, reponame, eReader)
				}
			}
			for _, eReader := range ereaderToAdd {
				r.UpdateRepositorySetExternalUser(ctx, errorCollector, dryrun, remote, reponame, eReader, "pull")
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
					r.UpdateRepositoryRemoveExternalUser(ctx, errorCollector, dryrun, remote, reponame, eWriter)
				}
			}
			for _, eWriter := range ewriteToAdd {
				r.UpdateRepositorySetExternalUser(ctx, errorCollector, dryrun, remote, reponame, eWriter, "push")
			}
		}

		if manageGithubVariables {
			archived := lRepo.BoolProperties["archived"]
			if !archived {
				if !utils.DeepEqualUnordered(lRepo.ActionVariables.GetEntity(), rRepo.ActionVariables.GetEntity()) {

					// Check for removed or changed keys
					for name, value := range lRepo.ActionVariables.GetEntity() {
						if rValue, ok := rRepo.ActionVariables.GetEntity()[name]; !ok {
							r.AddRepositoryVariable(ctx, errorCollector, dryrun, remote, reponame, name, value)
						} else if rValue != value {
							r.UpdateRepositoryVariable(ctx, errorCollector, dryrun, remote, reponame, name, value)
						}
					}

					// Check for added keys
					for name := range rRepo.ActionVariables.GetEntity() {
						if _, ok := lRepo.ActionVariables.GetEntity()[name]; !ok {
							r.DeleteRepositoryVariable(ctx, errorCollector, dryrun, remote, reponame, name)
						}
					}
				}
			}
		}
	}

	onAdded := func(reponame string, lRepo *GithubRepoComparable, rRepo *GithubRepoComparable) {
		// CREATE repository

		// if the repo was just archived in a previous commit and we "resume it"
		if aRepo, ok := reposToArchive[reponame]; ok {
			delete(reposToArchive, reponame)
			r.UpdateRepositoryUpdateProperty(ctx, errorCollector, dryrun, remote, reponame, "archived", false)
			// calling onChanged to update the repository permissions
			onChanged(reponame, aRepo, rRepo)
		} else {
			// check if the repo is a fork
			if lr, ok := lRepos[reponame]; ok {
				if lr.IsFork {
					r.CreateRepository(ctx, errorCollector, dryrun, remote, reponame, reponame, lRepo.Visibility, lRepo.Writers, lRepo.Readers, lRepo.BoolProperties, lRepo.DefaultBranchName, lr.ForkFrom)
				} else {
					r.CreateRepository(ctx, errorCollector, dryrun, remote, reponame, reponame, lRepo.Visibility, lRepo.Writers, lRepo.Readers, lRepo.BoolProperties, lRepo.DefaultBranchName, "")
				}
			} else {
				r.CreateRepository(ctx, errorCollector, dryrun, remote, reponame, reponame, lRepo.Visibility, lRepo.Writers, lRepo.Readers, lRepo.BoolProperties, lRepo.DefaultBranchName, "")
			}
		}
	}

	onRemoved := func(reponame string, lRepo *GithubRepoComparable, rRepo *GithubRepoComparable) {
		// here we have a repository that is not listed in the teams repository.
		// we should call DeleteRepository (that will delete if AllowDestructiveRepositories is on).
		// but if we have ArchiveOnDelete...
		if r.repoconfig.ArchiveOnDelete {
			if r.repoconfig.DestructiveOperations.AllowDestructiveRepositories {
				r.UpdateRepositoryUpdateProperty(ctx, errorCollector, dryrun, remote, reponame, "archived", true)
				reposToArchive[reponame] = rRepo
			} else {
				r.unmanaged.Repositories[reponame] = true
			}
		} else {
			r.DeleteRepository(ctx, errorCollector, dryrun, remote, reponame)
		}
	}

	CompareEntities(lRepos, rRepos, compareRepos, onAdded, onRemoved, onChanged)

	return reposToArchive, reposToRename, nil
}

func compareBranchProtections(bpname string, lbp *GithubBranchProtection, rbp *GithubBranchProtection) bool {
	if lbp.Pattern != rbp.Pattern {
		return false
	}
	if lbp.RequiresApprovingReviews != rbp.RequiresApprovingReviews {
		return false
	}
	if lbp.RequiresApprovingReviews {
		if lbp.RequiredApprovingReviewCount != rbp.RequiredApprovingReviewCount {
			return false
		}
	}
	if lbp.DismissesStaleReviews != rbp.DismissesStaleReviews {
		return false
	}
	if lbp.RequiresCodeOwnerReviews != rbp.RequiresCodeOwnerReviews {
		return false
	}
	if lbp.RequireLastPushApproval != rbp.RequireLastPushApproval {
		return false
	}
	if lbp.RequiresStatusChecks != rbp.RequiresStatusChecks {
		return false
	}
	if lbp.RequiresStatusChecks {
		if lbp.RequiresStrictStatusChecks != rbp.RequiresStrictStatusChecks {
			return false
		}
		if res, _, _ := entity.StringArrayEquivalent(lbp.RequiredStatusCheckContexts, rbp.RequiredStatusCheckContexts); !res {
			return false
		}
	}
	if lbp.RequiresConversationResolution != rbp.RequiresConversationResolution {
		return false
	}
	if lbp.RequiresCommitSignatures != rbp.RequiresCommitSignatures {
		return false
	}
	if lbp.RequiresLinearHistory != rbp.RequiresLinearHistory {
		return false
	}
	if lbp.AllowsForcePushes != rbp.AllowsForcePushes {
		return false
	}
	if lbp.AllowsDeletions != rbp.AllowsDeletions {
		return false
	}
	return true
}

func compareEnvironments(environment string, lEnv *GithubEnvironment, rEnv *GithubEnvironment) bool {
	if lEnv.Name != rEnv.Name {
		return false
	}
	if len(lEnv.Variables) != len(rEnv.Variables) {
		return false
	}
	for k, v := range lEnv.Variables {
		if rEnv.Variables[k] != v {
			return false
		}
	}
	return true
}

func compareAutolinks(autolinkname string, lal *GithubAutolink, ral *GithubAutolink) bool {
	if lal.KeyPrefix != ral.KeyPrefix {
		return false
	}
	if lal.UrlTemplate != ral.UrlTemplate {
		return false
	}
	if lal.IsAlphanumeric != ral.IsAlphanumeric {
		return false
	}
	return true
}

/*
used to compare org rulesets but also repo rulesets
*/
func compareRulesets(rulesetname string, lrs *GithubRuleSet, rrs *GithubRuleSet) bool {
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
	if len(lrs.BypassTeams) != len(rrs.BypassTeams) {
		return false
	}
	for k, v := range lrs.BypassTeams {
		if rrs.BypassTeams[k] != v {
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

func (r *GoliacReconciliatorImpl) reconciliateRulesets(ctx context.Context, errorCollector *observability.ErrorCollection, local GoliacReconciliatorDatasource, remote *MutableGoliacRemoteImpl, conf *config.RepositoryConfig, dryrun bool) error {
	lgrs, err := local.RuleSets()
	if err != nil {
		return err
	}

	// prepare remote comparable
	rgrs := remote.RuleSets()
	if err != nil {
		return err
	}

	// prepare the diff computation

	onAdded := func(rulesetname string, lRuleset *GithubRuleSet, rRuleset *GithubRuleSet) {
		// CREATE ruleset

		r.AddRuleset(ctx, errorCollector, dryrun, lRuleset)
	}

	onRemoved := func(rulesetname string, lRuleset *GithubRuleSet, rRuleset *GithubRuleSet) {
		// DELETE ruleset
		r.DeleteRuleset(ctx, errorCollector, dryrun, rRuleset)
	}

	onChanged := func(rulesetname string, lRuleset *GithubRuleSet, rRuleset *GithubRuleSet) {
		// UPDATE ruleset
		lRuleset.Id = rRuleset.Id
		r.UpdateRuleset(ctx, errorCollector, dryrun, lRuleset)
	}

	CompareEntities(lgrs, rgrs, compareRulesets, onAdded, onRemoved, onChanged)

	return nil
}

func (r *GoliacReconciliatorImpl) AddUserToOrg(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, ghuserid string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "add_user_to_org"}).Infof("ghuserid: %s", ghuserid)
	remote.AddUserToOrg(ghuserid)
	if r.executor != nil {
		r.executor.AddUserToOrg(ctx, errorCollector, dryrun, ghuserid)
	}
}

func (r *GoliacReconciliatorImpl) RemoveUserFromOrg(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, ghuserid string) {
	if r.repoconfig.DestructiveOperations.AllowDestructiveUsers {
		logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "remove_user_from_org"}).Infof("ghuserid: %s", ghuserid)
		remote.RemoveUserFromOrg(ghuserid)
		if r.executor != nil {
			r.executor.RemoveUserFromOrg(ctx, errorCollector, dryrun, ghuserid)
		}
	} else {
		r.unmanaged.Users[ghuserid] = true
	}
}

func (r *GoliacReconciliatorImpl) CreateTeam(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, teamname string, description string, parentTeam *int, members []string) {
	parentTeamId := "nil"
	if parentTeam != nil {
		parentTeamId = fmt.Sprintf("%d", *parentTeam)
	}

	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "create_team"}).Infof("teamname: %s, parentTeam: %s, members: %s", teamname, parentTeamId, strings.Join(members, ","))
	remote.CreateTeam(teamname, description, members, parentTeam)
	if r.executor != nil {
		r.executor.CreateTeam(ctx, errorCollector, dryrun, teamname, description, parentTeam, members)
	}
}
func (r *GoliacReconciliatorImpl) UpdateTeamAddMember(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string, ghuserid string, role string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_team_add_member"}).Infof("teamslug: %s, ghuserid: %s, role: %s", teamslug, ghuserid, role)
	remote.UpdateTeamAddMember(teamslug, ghuserid, "member")
	if r.executor != nil {
		r.executor.UpdateTeamAddMember(ctx, errorCollector, dryrun, teamslug, ghuserid, "member")
	}
}
func (r *GoliacReconciliatorImpl) UpdateTeamRemoveMember(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string, ghuserid string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_team_remove_member"}).Infof("teamslug: %s, ghuserid: %s", teamslug, ghuserid)
	remote.UpdateTeamRemoveMember(teamslug, ghuserid)
	if r.executor != nil {
		r.executor.UpdateTeamRemoveMember(ctx, errorCollector, dryrun, teamslug, ghuserid)
	}
}
func (r *GoliacReconciliatorImpl) UpdateTeamChangeMaintainerToMember(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string, ghuserid string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_team_change_maintainer_to_member"}).Infof("teamslug: %s, ghuserid: %s", teamslug, ghuserid)
	remote.UpdateTeamUpdateMember(teamslug, ghuserid, "member")
	if r.executor != nil {
		r.executor.UpdateTeamUpdateMember(ctx, errorCollector, dryrun, teamslug, ghuserid, "member")
	}
}
func (r *GoliacReconciliatorImpl) UpdateTeamSetParent(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string, parentTeam *int, parentTeamName string) {
	parenTeamId := "nil"
	if parentTeam != nil {
		parenTeamId = fmt.Sprintf("%d", *parentTeam)
	}

	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_team_parentteam"}).Infof("teamslug: %s, parentteam: %s (%s)", teamslug, parenTeamId, parentTeamName)
	remote.UpdateTeamSetParent(ctx, dryrun, teamslug, parentTeam)
	if r.executor != nil {
		r.executor.UpdateTeamSetParent(ctx, errorCollector, dryrun, teamslug, parentTeam)
	}
}
func (r *GoliacReconciliatorImpl) DeleteTeam(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, teamslug string) {
	if r.repoconfig.DestructiveOperations.AllowDestructiveTeams {
		logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_team"}).Infof("teamslug: %s", teamslug)
		remote.DeleteTeam(teamslug)
		if r.executor != nil {
			r.executor.DeleteTeam(ctx, errorCollector, dryrun, teamslug)
		}
	} else {
		r.unmanaged.Teams[teamslug] = true
	}
}
func (r *GoliacReconciliatorImpl) CreateRepository(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, descrition string, visibility string, writers []string, readers []string, boolProperties map[string]bool, defaultBranch string, forkFrom string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "create_repository"}).Infof("repositoryname: %s, readers: %s, writers: %s, boolProperties: %v", reponame, strings.Join(readers, ","), strings.Join(writers, ","), boolProperties)
	remote.CreateRepository(reponame, reponame, visibility, writers, readers, boolProperties, defaultBranch, forkFrom)
	if r.executor != nil {
		r.executor.CreateRepository(ctx, errorCollector, dryrun, reponame, reponame, visibility, writers, readers, boolProperties, defaultBranch, nil, forkFrom)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryAddTeamAccess(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, teamslug string, permission string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_add_team"}).Infof("repositoryname: %s, teamslug: %s, permission: %s", reponame, teamslug, permission)
	remote.UpdateRepositoryAddTeamAccess(reponame, teamslug, permission)
	if r.executor != nil {
		r.executor.UpdateRepositoryAddTeamAccess(ctx, errorCollector, dryrun, reponame, teamslug, permission)
	}
}

func (r *GoliacReconciliatorImpl) UpdateRepositoryUpdateTeamAccess(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, teamslug string, permission string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_update_team"}).Infof("repositoryname: %s, teamslug:%s, permission: %s", reponame, teamslug, permission)
	remote.UpdateRepositoryUpdateTeamAccess(reponame, teamslug, permission)
	if r.executor != nil {
		r.executor.UpdateRepositoryUpdateTeamAccess(ctx, errorCollector, dryrun, reponame, teamslug, permission)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryRemoveTeamAccess(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, teamslug string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_remove_team"}).Infof("repositoryname: %s, teamslug:%s", reponame, teamslug)
	remote.UpdateRepositoryRemoveTeamAccess(reponame, teamslug)
	if r.executor != nil {
		r.executor.UpdateRepositoryRemoveTeamAccess(ctx, errorCollector, dryrun, reponame, teamslug)
	}
}

func (r *GoliacReconciliatorImpl) DeleteRepository(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string) {
	if r.repoconfig.DestructiveOperations.AllowDestructiveRepositories {
		logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_repository"}).Infof("repositoryname: %s", reponame)
		remote.DeleteRepository(reponame)
		if r.executor != nil {
			r.executor.DeleteRepository(ctx, errorCollector, dryrun, reponame)
		}
	} else {
		r.unmanaged.Repositories[reponame] = true
	}
}

func (r *GoliacReconciliatorImpl) RenameRepository(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, newname string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "rename_repository"}).Infof("repositoryname: %s newname: %s", reponame, newname)
	remote.RenameRepository(reponame, newname)
	if r.executor != nil {
		r.executor.RenameRepository(ctx, errorCollector, dryrun, reponame, newname)
	}
}

func (r *GoliacReconciliatorImpl) UpdateRepositoryUpdateProperty(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, propertyName string, propertyValue interface{}) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_update_property"}).Infof("repositoryname: %s %s:%v", reponame, propertyName, propertyValue)
	remote.UpdateRepositoryUpdateProperty(reponame, propertyName, propertyValue)
	if r.executor != nil {
		r.executor.UpdateRepositoryUpdateProperty(ctx, errorCollector, dryrun, reponame, propertyName, propertyValue)
	}
}
func (r *GoliacReconciliatorImpl) AddRuleset(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, ruleset *GithubRuleSet) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "add_ruleset"}).Infof("ruleset: %s (id: %d) enforcement: %s", ruleset.Name, ruleset.Id, ruleset.Enforcement)
	if r.executor != nil {
		r.executor.AddRuleset(ctx, errorCollector, dryrun, ruleset)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRuleset(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, ruleset *GithubRuleSet) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_ruleset"}).Infof("ruleset: %s (id: %d) enforcement: %s", ruleset.Name, ruleset.Id, ruleset.Enforcement)
	if r.executor != nil {
		r.executor.UpdateRuleset(ctx, errorCollector, dryrun, ruleset)
	}
}
func (r *GoliacReconciliatorImpl) DeleteRuleset(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, ruleset *GithubRuleSet) {
	if r.repoconfig.DestructiveOperations.AllowDestructiveRulesets {
		logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_ruleset"}).Infof("ruleset id:%d", ruleset.Id)
		if r.executor != nil {
			r.executor.DeleteRuleset(ctx, errorCollector, dryrun, ruleset.Id)
		}
	} else {
		r.unmanaged.RuleSets[ruleset.Name] = true
	}
}
func (r *GoliacReconciliatorImpl) AddRepositoryRuleset(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, reponame string, ruleset *GithubRuleSet) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "add_repository_ruleset"}).Infof("repository: %s, ruleset: %s (id: %d) enforcement: %s", reponame, ruleset.Name, ruleset.Id, ruleset.Enforcement)
	if r.executor != nil {
		r.executor.AddRepositoryRuleset(ctx, errorCollector, dryrun, reponame, ruleset)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryRuleset(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, reponame string, ruleset *GithubRuleSet) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_ruleset"}).Infof("repository: %s, ruleset: %s (id: %d) enforcement: %s", reponame, ruleset.Name, ruleset.Id, ruleset.Enforcement)
	if r.executor != nil {
		r.executor.UpdateRepositoryRuleset(ctx, errorCollector, dryrun, reponame, ruleset)
	}
}
func (r *GoliacReconciliatorImpl) DeleteRepositoryRuleset(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, reponame string, ruleset *GithubRuleSet) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_repository_ruleset"}).Infof("repository: %s, ruleset id:%d", reponame, ruleset.Id)
	if r.executor != nil {
		r.executor.DeleteRepositoryRuleset(ctx, errorCollector, dryrun, reponame, ruleset.Id)
	}
}
func (r *GoliacReconciliatorImpl) AddRepositoryBranchProtection(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, reponame string, branchprotection *GithubBranchProtection) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "add_repository_branchprotection"}).Infof("repository: %s, branchprotection: %s", reponame, branchprotection.Pattern)
	if r.executor != nil {
		r.executor.AddRepositoryBranchProtection(ctx, errorCollector, dryrun, reponame, branchprotection)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryBranchProtection(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, reponame string, branchprotection *GithubBranchProtection) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_branchprotection"}).Infof("repository: %s, branchprotection: %s", reponame, branchprotection.Pattern)
	if r.executor != nil {
		r.executor.UpdateRepositoryBranchProtection(ctx, errorCollector, dryrun, reponame, branchprotection)
	}
}
func (r *GoliacReconciliatorImpl) DeleteRepositoryBranchProtection(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, reponame string, branchprotection *GithubBranchProtection) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_repository_branchprotection"}).Infof("repository: %s, branchprotection: %s", reponame, branchprotection.Pattern)
	if r.executor != nil {
		r.executor.DeleteRepositoryBranchProtection(ctx, errorCollector, dryrun, reponame, branchprotection)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositorySetExternalUser(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, collaboatorGithubId string, permission string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_set_external_user"}).Infof("repositoryname: %s collaborator:%s permission:%s", reponame, collaboatorGithubId, permission)
	remote.UpdateRepositorySetExternalUser(reponame, collaboatorGithubId, permission)
	if r.executor != nil {
		r.executor.UpdateRepositorySetExternalUser(ctx, errorCollector, dryrun, reponame, collaboatorGithubId, permission)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryRemoveInternalUser(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, collaboatorGithubId string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_remove_internal_user"}).Infof("repositoryname: %s collaborator:%s", reponame, collaboatorGithubId)
	remote.UpdateRepositoryRemoveInternalUser(reponame, collaboatorGithubId)
	if r.executor != nil {
		r.executor.UpdateRepositoryRemoveInternalUser(ctx, errorCollector, dryrun, reponame, collaboatorGithubId)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryRemoveExternalUser(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, collaboatorGithubId string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_remove_external_user"}).Infof("repositoryname: %s collaborator:%s", reponame, collaboatorGithubId)
	remote.UpdateRepositoryRemoveExternalUser(reponame, collaboatorGithubId)
	if r.executor != nil {
		r.executor.UpdateRepositoryRemoveExternalUser(ctx, errorCollector, dryrun, reponame, collaboatorGithubId)
	}
}
func (r *GoliacReconciliatorImpl) AddRepositoryEnvironment(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, environment string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "add_repository_environment"}).Infof("repository: %s, environment: %s", reponame, environment)
	remote.AddRepositoryEnvironment(reponame, environment)
	if r.executor != nil {
		r.executor.AddRepositoryEnvironment(ctx, errorCollector, dryrun, reponame, environment)
	}
}
func (r *GoliacReconciliatorImpl) DeleteRepositoryEnvironment(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, environment string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_repository_environment"}).Infof("repository: %s, environment: %s", reponame, environment)
	remote.DeleteRepositoryEnvironment(reponame, environment)
	if r.executor != nil {
		r.executor.DeleteRepositoryEnvironment(ctx, errorCollector, dryrun, reponame, environment)
	}
}
func (r *GoliacReconciliatorImpl) AddRepositoryVariable(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, variable string, value string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "add_repository_variable"}).Infof("repository: %s, variable: %s, value: %s", reponame, variable, value)
	remote.AddRepositoryVariable(reponame, variable, value)
	if r.executor != nil {
		r.executor.AddRepositoryVariable(ctx, errorCollector, dryrun, reponame, variable, value)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryVariable(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, variable string, value string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_variable"}).Infof("repository: %s, variable: %s, value: %s", reponame, variable, value)
	remote.UpdateRepositoryVariable(reponame, variable, value)
	if r.executor != nil {
		r.executor.UpdateRepositoryVariable(ctx, errorCollector, dryrun, reponame, variable, value)
	}
}
func (r *GoliacReconciliatorImpl) DeleteRepositoryVariable(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, variable string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_repository_variable"}).Infof("repository: %s, variable: %s", reponame, variable)
	remote.DeleteRepositoryVariable(reponame, variable)
	if r.executor != nil {
		r.executor.DeleteRepositoryVariable(ctx, errorCollector, dryrun, reponame, variable)
	}
}
func (r *GoliacReconciliatorImpl) AddRepositoryEnvironmentVariable(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, environment string, variable string, value string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "add_repository_environment_variable"}).Infof("repository: %s, environment: %s, variable: %s, value: %s", reponame, environment, variable, value)
	remote.AddRepositoryEnvironmentVariable(reponame, environment, variable, value)
	if r.executor != nil {
		r.executor.AddRepositoryEnvironmentVariable(ctx, errorCollector, dryrun, reponame, environment, variable, value)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryEnvironmentVariable(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, environment string, variable string, value string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_environment_variable"}).Infof("repository: %s, environment: %s, variable: %s, value: %s", reponame, environment, variable, value)
	remote.UpdateRepositoryEnvironmentVariable(reponame, environment, variable, value)
	if r.executor != nil {
		r.executor.UpdateRepositoryEnvironmentVariable(ctx, errorCollector, dryrun, reponame, environment, variable, value)
	}
}
func (r *GoliacReconciliatorImpl) DeleteRepositoryEnvironmentVariable(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, environment string, variable string) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_repository_environment_variable"}).Infof("repository: %s, environment: %s, variable: %s", reponame, environment, variable)
	remote.DeleteRepositoryEnvironmentVariable(reponame, environment, variable)
	if r.executor != nil {
		r.executor.DeleteRepositoryEnvironmentVariable(ctx, errorCollector, dryrun, reponame, environment, variable)
	}
}
func (r *GoliacReconciliatorImpl) AddRepositoryAutolink(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, autolink *GithubAutolink) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "add_repository_autolink"}).Infof("repository: %s, autolink: %s", reponame, autolink.KeyPrefix)
	remote.AddRepositoryAutolink(reponame, autolink)
	if r.executor != nil {
		r.executor.AddRepositoryAutolink(ctx, errorCollector, dryrun, reponame, autolink)
	}
}
func (r *GoliacReconciliatorImpl) DeleteRepositoryAutolink(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, autolinkId int) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "delete_repository_autolink"}).Infof("repository: %s, autolink id: %d", reponame, autolinkId)
	remote.DeleteRepositoryAutolink(reponame, autolinkId)
	if r.executor != nil {
		r.executor.DeleteRepositoryAutolink(ctx, errorCollector, dryrun, reponame, autolinkId)
	}
}
func (r *GoliacReconciliatorImpl) UpdateRepositoryAutolink(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool, remote *MutableGoliacRemoteImpl, reponame string, autolink *GithubAutolink) {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "command": "update_repository_autolink"}).Infof("repository: %s, autolink: %s", reponame, autolink.KeyPrefix)
	remote.UpdateRepositoryAutolink(reponame, autolink)
	if r.executor != nil {
		r.executor.UpdateRepositoryAutolink(ctx, errorCollector, dryrun, reponame, autolink)
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
func (r *GoliacReconciliatorImpl) Commit(ctx context.Context, errorCollector *observability.ErrorCollection, dryrun bool) error {
	logrus.WithFields(map[string]interface{}{"dryrun": dryrun}).Debugf("reconciliation commit")
	if r.executor != nil {
		return r.executor.Commit(ctx, errorCollector, dryrun)
	}
	return nil
}
