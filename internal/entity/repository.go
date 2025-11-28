package entity

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/goliac-project/goliac/internal/config"
	"github.com/goliac-project/goliac/internal/observability"
	"github.com/goliac-project/goliac/internal/utils"
	"gopkg.in/yaml.v3"
)

type RepositoryEnvironment struct {
	Name      string            `yaml:"name"`
	Variables map[string]string `yaml:"variables,omitempty"`
}

type RepositoryAutolink struct {
	KeyPrefix      string `yaml:"key_prefix"`
	UrlTemplate    string `yaml:"url_template"`
	IsAlphanumeric bool   `yaml:"is_alphanumeric"`
}

type Repository struct {
	Entity `yaml:",inline"`
	Spec   struct {
		Writers                    []string                     `yaml:"writers,omitempty"`
		Readers                    []string                     `yaml:"readers,omitempty"`
		ExternalUserReaders        []string                     `yaml:"externalUserReaders,omitempty"`
		ExternalUserWriters        []string                     `yaml:"externalUserWriters,omitempty"`
		Visibility                 string                       `yaml:"visibility,omitempty"`
		AllowAutoMerge             bool                         `yaml:"allow_auto_merge,omitempty"`
		AllowSquashMerge           bool                         `yaml:"allow_squash_merge,omitempty"`
		AllowRebaseMerge           bool                         `yaml:"allow_rebase_merge,omitempty"`
		AllowMergeCommit           bool                         `yaml:"allow_merge_commit,omitempty"`
		DefaultMergeCommitMessage  string                       `yaml:"default_merge_commit_message,omitempty"`
		DefaultSquashCommitMessage string                       `yaml:"default_squash_commit_message,omitempty"`
		DeleteBranchOnMerge        bool                         `yaml:"delete_branch_on_merge,omitempty"`
		AllowUpdateBranch          bool                         `yaml:"allow_update_branch,omitempty"`
		Rulesets                   []RepositoryRuleSet          `yaml:"rulesets,omitempty"`
		BranchProtections          []RepositoryBranchProtection `yaml:"branch_protections,omitempty"`
		DefaultBranchName          string                       `yaml:"default_branch,omitempty"`
		Environments               []RepositoryEnvironment      `yaml:"environments,omitempty"`
		ActionsVariables           map[string]string            `yaml:"actions_variables,omitempty"`
		Autolinks                  *[]RepositoryAutolink        `yaml:"autolinks,omitempty"`
		CustomProperties           map[string]interface{}       `yaml:"custom_properties,omitempty"`
		Topics                     []string                     `yaml:"topics,omitempty"`
	} `yaml:"spec,omitempty"`
	Archived      bool    `yaml:"archived,omitempty"` // implicit: will be set by Goliac
	Owner         *string `yaml:"-"`                  // implicit. team name owning the repo (if any)
	RenameTo      string  `yaml:"renameTo,omitempty"`
	DirectoryPath string  `yaml:"-"` // used to know where to rename the repository
	ForkFrom      string  `yaml:"forkFrom,omitempty"`
}

type RepositoryRuleSet struct {
	RuleSetDefinition `yaml:",inline"`
	Name              string `yaml:"name"`
}

type RepositoryBranchProtection struct {
	Pattern                        string   `yaml:"pattern"` // branch name pattern like "master" or "release/*"
	RequiresApprovingReviews       bool     `yaml:"requires_approving_reviews,omitempty"`
	RequiredApprovingReviewCount   int      `yaml:"required_approving_review_count,omitempty"`
	DismissesStaleReviews          bool     `yaml:"dismisses_stale_reviews,omitempty"`
	RequiresCodeOwnerReviews       bool     `yaml:"requires_code_owner_reviews,omitempty"`
	RequireLastPushApproval        bool     `yaml:"require_last_push_approval,omitempty"`
	RequiresStatusChecks           bool     `yaml:"requires_status_checks,omitempty"`
	RequiresStrictStatusChecks     bool     `yaml:"requires_strict_status_checks,omitempty"`  // if true, only the status checks in required_status_check_contexts will be considered
	RequiredStatusCheckContexts    []string `yaml:"required_status_check_contexts,omitempty"` // list of status check contexts like "continuous-integration/travis-ci"
	RequiresConversationResolution bool     `yaml:"requires_conversation_resolution,omitempty"`
	RequiresCommitSignatures       bool     `yaml:"requires_commit_signatures,omitempty"`
	RequiresLinearHistory          bool     `yaml:"requires_linear_history,omitempty"`
	AllowsForcePushes              bool     `yaml:"allows_force_pushes,omitempty"`
	AllowsDeletions                bool     `yaml:"allows_deletions,omitempty"`
	BypassPullRequestUsers         []string `yaml:"bypass_pullrequest_users,omitempty"`
	BypassPullRequestTeams         []string `yaml:"bypass_pullrequest_teams,omitempty"`
	BypassPullRequestApps          []string `yaml:"bypass_pullrequest_apps,omitempty"`
}

/*
 * NewRepository reads a file and returns a Repository object
 * The next step is to validate the Repository object using the Validate method
 */
func NewRepository(fs billy.Filesystem, filename string) (*Repository, error) {
	filecontent, err := utils.ReadFile(fs, filename)
	if err != nil {
		return nil, err
	}

	repository := &Repository{}
	repository.Spec.Visibility = "private"                         // default visibility
	repository.Spec.DefaultBranchName = ""                         // default branch name
	repository.Spec.AllowAutoMerge = false                         // default allow auto merge
	repository.Spec.AllowSquashMerge = true                        // default allow squash merge
	repository.Spec.AllowRebaseMerge = true                        // default allow rebase merge
	repository.Spec.AllowMergeCommit = true                        // default allow merge commit
	repository.Spec.DefaultMergeCommitMessage = "Default message"  // default merge commit message template
	repository.Spec.DefaultSquashCommitMessage = "Default message" // default squash commit message template
	err = yaml.Unmarshal(filecontent, repository)
	if err != nil {
		return nil, err
	}
	repository.DirectoryPath = filepath.Dir(filename)

	// Normalize custom properties: convert int values to strings (especially for Tier)
	if repository.Spec.CustomProperties != nil {
		normalized := make(map[string]interface{})
		for k, v := range repository.Spec.CustomProperties {
			// Convert int to string, especially for Tier
			switch val := v.(type) {
			case int:
				normalized[k] = fmt.Sprintf("%d", val)
			default:
				normalized[k] = v
			}
		}
		repository.Spec.CustomProperties = normalized
	}

	// set default values for ruleset rules
	for i, ruleset := range repository.Spec.Rulesets {
		for j, rule := range ruleset.RuleSetDefinition.Rules {
			if rule.Ruletype == "pull_request" {
				if len(rule.Parameters.AllowedMergeMethods) == 0 {
					// default to MERGE, SQUASH, REBASE
					rule.Parameters.AllowedMergeMethods = []string{"MERGE", "SQUASH", "REBASE"}
					ruleset.RuleSetDefinition.Rules[j] = rule
				}
			}
			if rule.Ruletype == "merge_queue" {
				if rule.Parameters.CheckResponseTimeoutMinutes == 0 {
					// default to 10 minutes
					rule.Parameters.CheckResponseTimeoutMinutes = 10
					ruleset.RuleSetDefinition.Rules[j] = rule
				}
			}
		}
		repository.Spec.Rulesets[i] = ruleset
	}
	return repository, nil
}

/**
 * ReadRepositories reads all the files in the dirname directory and
 * add them to the owner's team and returns
 * - a map of Repository objects
 * - a slice of errors that must stop the validation process
 * - a slice of warning that must not stop the validation process
 */
func ReadRepositories(fs billy.Filesystem, archivedDirname string, teamDirname string, teams map[string]*Team, externalUsers map[string]*User, users map[string]*User, customProperties []*config.GithubCustomProperty, LogCollection *observability.LogCollection) map[string]*Repository {
	repos := make(map[string]*Repository)

	// archived dir
	exist, err := utils.Exists(fs, archivedDirname)
	if err != nil {
		LogCollection.AddError(err)
		return repos
	}
	if exist {
		entries, err := fs.ReadDir(archivedDirname)
		if err != nil {
			LogCollection.AddError(err)
			return nil
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			// skipping files starting with '.'
			if entry.Name()[0] == '.' {
				continue
			}
			if !strings.HasSuffix(entry.Name(), ".yaml") {
				LogCollection.AddWarn(fmt.Errorf("file %s doesn't have a .yaml extension", entry.Name()))
				continue
			}
			repo, err := NewRepository(fs, filepath.Join(archivedDirname, entry.Name()))
			if err != nil {
				LogCollection.AddError(err)
			} else {
				if err := repo.Validate(filepath.Join(archivedDirname, entry.Name()), teams, externalUsers, users, customProperties); err != nil {
					LogCollection.AddError(err)
				} else {
					repo.Archived = true
					repos[repo.Name] = repo
				}
			}
		}
	}
	// regular teams dir
	exist, err = utils.Exists(fs, teamDirname)
	if err != nil {
		LogCollection.AddError(err)
		return repos
	}
	if !exist {
		return repos
	}

	// Parse all the repositories in the teamDirname directory
	entries, err := fs.ReadDir(teamDirname)
	if err != nil {
		LogCollection.AddError(err)
		return nil
	}

	for _, team := range entries {
		if team.IsDir() {
			recursiveReadRepositories(fs, archivedDirname, filepath.Join(teamDirname, team.Name()), team.Name(), repos, teams, externalUsers, users, customProperties, LogCollection)
		}
	}

	return repos
}

func recursiveReadRepositories(fs billy.Filesystem, archivedDirPath string, teamDirPath string, teamName string, repos map[string]*Repository, teams map[string]*Team, externalUsers map[string]*User, users map[string]*User, customProperties []*config.GithubCustomProperty, LogCollection *observability.LogCollection) {

	subentries, err := fs.ReadDir(teamDirPath)
	if err != nil {
		LogCollection.AddError(err)
		return
	}
	for _, sube := range subentries {
		if sube.IsDir() && sube.Name()[0] != '.' {
			recursiveReadRepositories(fs, archivedDirPath, filepath.Join(teamDirPath, sube.Name()), sube.Name(), repos, teams, externalUsers, users, customProperties, LogCollection)
		}
		if !sube.IsDir() && sube.Name() != "team.yaml" {
			if filepath.Ext(sube.Name()) != ".yaml" {
				LogCollection.AddError(fmt.Errorf("file %s doesn't have a .yaml extension", sube.Name()))
				continue
			}
			repo, err := NewRepository(fs, filepath.Join(teamDirPath, sube.Name()))
			if err != nil {
				LogCollection.AddError(err)
			} else {
				if err := repo.Validate(filepath.Join(teamDirPath, sube.Name()), teams, externalUsers, users, customProperties); err != nil {
					LogCollection.AddError(err)
				} else {
					// check if the repository doesn't already exists
					if _, exist := repos[repo.Name]; exist {
						existing := filepath.Join(archivedDirPath, repo.Name)
						if repos[repo.Name].Owner != nil {
							existing = filepath.Join(*repos[repo.Name].Owner, repo.Name)
						}
						LogCollection.AddError(fmt.Errorf("Repository %s defined in 2 places (check %s and %s)", repo.Name, filepath.Join(teamDirPath, sube.Name()), existing))
					} else {
						teamname := teamName
						repo.Owner = &teamname
						repo.Archived = false
						repos[repo.Name] = repo
					}
				}
			}
		}
	}
}

func (r *Repository) Validate(filename string, teams map[string]*Team, externalUsers map[string]*User, users map[string]*User, customProperties []*config.GithubCustomProperty) error {

	if r.ApiVersion != "v1" {
		return fmt.Errorf("invalid apiVersion: %s (check repository filename %s)", r.ApiVersion, filename)
	}

	if r.Kind != "Repository" {
		return fmt.Errorf("invalid kind: %s (check repository filename %s)", r.Kind, filename)
	}

	if r.Name == "" {
		return fmt.Errorf("name is empty (check repository filename %s)", filename)
	}

	filename = filepath.Base(filename)
	if r.Name != filename[:len(filename)-len(filepath.Ext(filename))] {
		return fmt.Errorf("invalid name: %s for repository filename %s", r.Name, filename)
	}

	visibility := r.Spec.Visibility
	if visibility != "public" && visibility != "private" && visibility != "internal" {
		return fmt.Errorf("invalid visibility: %s for repository filename %s", visibility, filename)
	}

	for _, writer := range r.Spec.Writers {
		if _, ok := teams[writer]; !ok {
			return fmt.Errorf("invalid writer: %s doesn't exist (check repository filename %s)", writer, filename)
		}
	}
	for _, reader := range r.Spec.Readers {
		if _, ok := teams[reader]; !ok {
			return fmt.Errorf("invalid reader: %s doesn't exist (check repository filename %s)", reader, filename)
		}
	}

	for _, externalUserReader := range r.Spec.ExternalUserReaders {
		if _, ok := externalUsers[externalUserReader]; !ok {
			return fmt.Errorf("invalid externalUserReader: %s doesn't exist in repository filename %s", externalUserReader, filename)
		}
	}

	for _, externalUserWriter := range r.Spec.ExternalUserWriters {
		if _, ok := externalUsers[externalUserWriter]; !ok {
			return fmt.Errorf("invalid externalUserWriter: %s doesn't exist in repository filename %s", externalUserWriter, filename)
		}
	}

	// Validate branch protection bypass_pullrequest_users
	for _, bp := range r.Spec.BranchProtections {
		for _, bypassUser := range bp.BypassPullRequestUsers {
			// Check if user exists in regular users or external users
			if users != nil {
				if _, ok := users[bypassUser]; !ok {
					if _, ok := externalUsers[bypassUser]; !ok {
						return fmt.Errorf("invalid bypass_pullrequest_user: %s doesn't exist in repository filename %s (branch protection pattern: %s)", bypassUser, filename, bp.Pattern)
					}
				}
			} else {
				// Fallback: check external users only if users map is not provided
				if _, ok := externalUsers[bypassUser]; !ok {
					return fmt.Errorf("invalid bypass_pullrequest_user: %s doesn't exist in repository filename %s (branch protection pattern: %s)", bypassUser, filename, bp.Pattern)
				}
			}
		}
	}

	rulesetname := make(map[string]bool)
	for _, ruleset := range r.Spec.Rulesets {
		if ruleset.Name == "" {
			return fmt.Errorf("invalid ruleset: each ruleset must have a name")
		}
		err := ValidateRulesetDefinition(&ruleset.RuleSetDefinition, filename)
		if err != nil {
			return err
		}
		if _, ok := rulesetname[ruleset.Name]; ok {
			return fmt.Errorf("invalid ruleset: each ruleset must have a uniq name, found 2 times %s", ruleset.Name)
		}

		rulesetname[ruleset.Name] = true

		// Validate mergeMethod compatibility with repository merge settings
		// and groupingStrategy validity
		for _, rule := range ruleset.RuleSetDefinition.Rules {
			if rule.Ruletype == "merge_queue" {
				// Validate groupingStrategy
				if rule.Parameters.GroupingStrategy != "" && rule.Parameters.GroupingStrategy != "ALLGREEN" && rule.Parameters.GroupingStrategy != "HEADGREEN" {
					return fmt.Errorf("invalid groupingStrategy: '%s' in merge_queue rule must be 'ALLGREEN' or 'HEADGREEN' for repository %s (ruleset: %s)", rule.Parameters.GroupingStrategy, r.Name, ruleset.Name)
				}
				// Validate mergeMethod compatibility
				if rule.Parameters.MergeMethod != "" {
					switch rule.Parameters.MergeMethod {
					case "SQUASH":
						if !r.Spec.AllowSquashMerge {
							return fmt.Errorf("invalid mergeMethod: 'SQUASH' in merge_queue rule requires allow_squash_merge to be true for repository %s (ruleset: %s)", r.Name, ruleset.Name)
						}
					case "REBASE":
						if !r.Spec.AllowRebaseMerge {
							return fmt.Errorf("invalid mergeMethod: 'REBASE' in merge_queue rule requires allow_rebase_merge to be true for repository %s (ruleset: %s)", r.Name, ruleset.Name)
						}
					case "MERGE":
						if !r.Spec.AllowMergeCommit {
							return fmt.Errorf("invalid mergeMethod: 'MERGE' in merge_queue rule requires allow_merge_commit to be true for repository %s (ruleset: %s)", r.Name, ruleset.Name)
						}
					}
				}
			}
		}
	}

	if utils.GithubAnsiString(r.Name) != r.Name {
		return fmt.Errorf("invalid name: %s will be changed to %s (check repository filename %s)", r.Name, utils.GithubAnsiString(r.Name), filename)
	}

	if r.ForkFrom != "" {
		// specific to github
		r.ForkFrom = strings.TrimPrefix(r.ForkFrom, "https://github.com/")
		r.ForkFrom = strings.TrimSuffix(r.ForkFrom, ".git")
		// formFrom must be "organization/repository"
		var forkFromPattern = regexp.MustCompile(`^[^/]+/[^/]+$`)

		if !forkFromPattern.MatchString(r.ForkFrom) {
			return fmt.Errorf("%s: invalid fork format: %s - must be in the format 'organization/repository'", r.Name, r.ForkFrom)
		}
	}

	if r.Spec.AllowMergeCommit {
		if r.Spec.DefaultMergeCommitMessage != "Default message" &&
			r.Spec.DefaultMergeCommitMessage != "Pull request title" &&
			r.Spec.DefaultMergeCommitMessage != "Pull request title and description" {
			return fmt.Errorf("%s: invalid default merge commit message: %s (it must be 'Default message', 'Pull request title', or 'Pull request title and description')", r.Name, r.Spec.DefaultMergeCommitMessage)
		}
	}

	if r.Spec.AllowSquashMerge {
		if r.Spec.DefaultSquashCommitMessage != "Default message" &&
			r.Spec.DefaultSquashCommitMessage != "Pull request title" &&
			r.Spec.DefaultSquashCommitMessage != "Pull request title and commit details" &&
			r.Spec.DefaultSquashCommitMessage != "Pull request title and description" {
			return fmt.Errorf("%s: invalid default squash commit message: %s (it must be 'Default message', 'Pull request title', 'Pull request title and commit details', or 'Pull request title and description')", r.Name, r.Spec.DefaultSquashCommitMessage)
		}
	}

	if r.Spec.CustomProperties != nil {
		for propName := range r.Spec.CustomProperties {
			found := false
			for _, customProperty := range customProperties {
				if customProperty.PropertyName == propName {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("invalid custom property: %s is not defined in the organization custom properties", propName)
			}
		}
	}

	return nil
}

/**
 * ReadAndAdjustRepositories adjusts repository definitions depending on user availability.
 * The goal is that if a user has been removed, we must update the repository definition
 * by removing them from branch_protections bypass_pullrequest_users.
 * Returns:
 * - a list of (repository's) file changes (to commit to Github)
 */
func ReadAndAdjustRepositories(fs billy.Filesystem, archivedDirname string, teamDirname string, users map[string]*User, externalUsers map[string]*User) ([]string, error) {
	reposChanged := []string{}

	// Combine regular users and external users for checking
	allUsers := make(map[string]*User)
	for k, v := range users {
		allUsers[k] = v
	}
	for k, v := range externalUsers {
		allUsers[k] = v
	}

	// archived dir
	exist, err := utils.Exists(fs, archivedDirname)
	if err != nil {
		return reposChanged, err
	}
	if exist {
		entries, err := fs.ReadDir(archivedDirname)
		if err != nil {
			return reposChanged, err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			// skipping files starting with '.'
			if entry.Name()[0] == '.' {
				continue
			}
			if !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}
			repo, err := NewRepository(fs, filepath.Join(archivedDirname, entry.Name()))
			if err != nil {
				continue
			}
			changed, err := repo.Update(fs, filepath.Join(archivedDirname, entry.Name()), allUsers)
			if err != nil {
				return reposChanged, err
			}
			if changed {
				reposChanged = append(reposChanged, filepath.Join(archivedDirname, entry.Name()))
			}
		}
	}

	// regular teams dir
	exist, err = utils.Exists(fs, teamDirname)
	if err != nil {
		return reposChanged, err
	}
	if !exist {
		return reposChanged, nil
	}

	// Parse all the repositories in the teamDirname directory
	entries, err := fs.ReadDir(teamDirname)
	if err != nil {
		return reposChanged, err
	}

	for _, team := range entries {
		if team.IsDir() {
			err := recursiveReadAndAdjustRepositories(fs, archivedDirname, filepath.Join(teamDirname, team.Name()), allUsers, &reposChanged)
			if err != nil {
				return reposChanged, err
			}
		}
	}

	return reposChanged, nil
}

func recursiveReadAndAdjustRepositories(fs billy.Filesystem, archivedDirPath string, teamDirPath string, allUsers map[string]*User, reposChanged *[]string) error {
	subentries, err := fs.ReadDir(teamDirPath)
	if err != nil {
		return err
	}
	for _, sube := range subentries {
		if sube.IsDir() && sube.Name()[0] != '.' {
			err := recursiveReadAndAdjustRepositories(fs, archivedDirPath, filepath.Join(teamDirPath, sube.Name()), allUsers, reposChanged)
			if err != nil {
				return err
			}
		}
		if !sube.IsDir() && sube.Name() != "team.yaml" {
			if filepath.Ext(sube.Name()) != ".yaml" {
				continue
			}
			repo, err := NewRepository(fs, filepath.Join(teamDirPath, sube.Name()))
			if err != nil {
				continue
			}
			changed, err := repo.Update(fs, filepath.Join(teamDirPath, sube.Name()), allUsers)
			if err != nil {
				return err
			}
			if changed {
				*reposChanged = append(*reposChanged, filepath.Join(teamDirPath, sube.Name()))
			}
		}
	}
	return nil
}

// Update is telling if the repository needs to be adjusted (and the repository's definition was changed on disk),
// based on the list of (still) existing users. It removes users from branch_protections bypass_pullrequest_users
// if they don't exist in the users map.
func (r *Repository) Update(fs billy.Filesystem, filename string, allUsers map[string]*User) (bool, error) {
	changed := false

	// Update branch protections bypass_pullrequest_users
	for i := range r.Spec.BranchProtections {
		bp := &r.Spec.BranchProtections[i]
		validUsers := make([]string, 0)
		for _, bypassUser := range bp.BypassPullRequestUsers {
			if _, ok := allUsers[bypassUser]; ok {
				validUsers = append(validUsers, bypassUser)
			} else {
				changed = true
			}
		}
		bp.BypassPullRequestUsers = validUsers
	}

	if !changed {
		return false, nil
	}

	file, err := fs.Create(filename)
	if err != nil {
		return changed, fmt.Errorf("not able to create file %s: %v", filename, err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	err = encoder.Encode(r)
	if err != nil {
		return changed, fmt.Errorf("not able to write file %s: %v", filename, err)
	}
	return changed, err
}
