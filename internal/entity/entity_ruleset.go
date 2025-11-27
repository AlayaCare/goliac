package entity

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/go-git/go-billy/v5"
	"github.com/goliac-project/goliac/internal/observability"
	"github.com/goliac-project/goliac/internal/utils"
	"gopkg.in/yaml.v3"
)

type RuleSetParameters struct {
	// PullRequestParameters
	DismissStaleReviewsOnPush      bool     `yaml:"dismissStaleReviewsOnPush,omitempty"`
	RequireCodeOwnerReview         bool     `yaml:"requireCodeOwnerReview,omitempty"`
	RequiredApprovingReviewCount   int      `yaml:"requiredApprovingReviewCount,omitempty"`
	RequiredReviewThreadResolution bool     `yaml:"requiredReviewThreadResolution,omitempty"`
	RequireLastPushApproval        bool     `yaml:"requireLastPushApproval,omitempty"`
	AllowedMergeMethods            []string `yaml:"allowedMergeMethods,omitempty"`

	// RequiredStatusChecksParameters
	RequiredStatusChecks             []string `yaml:"requiredStatusChecks,omitempty"`
	StrictRequiredStatusChecksPolicy bool     `yaml:"strictRequiredStatusChecksPolicy,omitempty"`

	// BranchNamePattern / TaghNamePattern
	Name     string `yaml:"name,omitempty"`
	Negate   bool   `yaml:"negate,omitempty"`
	Operator string `yaml:"operator,omitempty"`
	Pattern  string `yaml:"pattern,omitempty"`

	// MergeQueueParameters
	CheckResponseTimeoutMinutes  int    `yaml:"checkResponseTimeoutMinutes,omitempty"`
	GroupingStrategy             string `yaml:"groupingStrategy,omitempty"` // ALLGREEN, HEADGREEN
	MaxEntriesToBuild            int    `yaml:"maxEntriesToBuild,omitempty"`
	MaxEntriesToMerge            int    `yaml:"maxEntriesToMerge,omitempty"`
	MergeMethod                  string `yaml:"mergeMethod,omitempty"` // MERGE, REBASE, SQUASH
	MinEntriesToMerge            int    `yaml:"minEntriesToMerge,omitempty"`
	MinEntriesToMergeWaitMinutes int    `yaml:"minEntriesToMergeWaitMinutes,omitempty"`
}

func CompareRulesetParameters(ruletype string, left RuleSetParameters, right RuleSetParameters) bool {
	switch ruletype {
	case "required_signatures":
		return true
	case "creation":
		return true
	case "update":
		return true
	case "deletion":
		return true
	case "non_fast_forward":
		return true
	case "required_linear_history":
		return true
	case "pull_request":
		if left.DismissStaleReviewsOnPush != right.DismissStaleReviewsOnPush {
			return false
		}
		if left.RequireCodeOwnerReview != right.RequireCodeOwnerReview {
			return false
		}
		if left.RequiredApprovingReviewCount != right.RequiredApprovingReviewCount {
			return false
		}
		if left.RequiredReviewThreadResolution != right.RequiredReviewThreadResolution {
			return false
		}
		if left.RequireLastPushApproval != right.RequireLastPushApproval {
			return false
		}
		if res, _, _ := StringArrayEquivalent(left.AllowedMergeMethods, right.AllowedMergeMethods); !res {
			return false
		}
		return true
	case "merge_queue":
		if left.CheckResponseTimeoutMinutes != right.CheckResponseTimeoutMinutes {
			return false
		}
		if left.GroupingStrategy != right.GroupingStrategy {
			return false
		}
		if left.MaxEntriesToBuild != right.MaxEntriesToBuild {
			return false
		}
		if left.MaxEntriesToMerge != right.MaxEntriesToMerge {
			return false
		}
		if left.MergeMethod != right.MergeMethod {
			return false
		}
		if left.MinEntriesToMerge != right.MinEntriesToMerge {
			return false
		}
		if left.MinEntriesToMergeWaitMinutes != right.MinEntriesToMergeWaitMinutes {
			return false
		}
		return true
	case "required_status_checks":
		if res, _, _ := StringArrayEquivalent(left.RequiredStatusChecks, right.RequiredStatusChecks); !res {
			return false
		}
		if left.StrictRequiredStatusChecksPolicy != right.StrictRequiredStatusChecksPolicy {
			return false
		}
		return true
	case "branch_name_pattern":
		if left.Name != right.Name {
			return false
		}
		if left.Negate != right.Negate {
			return false
		}
		if left.Operator != right.Operator {
			return false
		}
		if left.Pattern != right.Pattern {
			return false
		}
		return true
	case "tag_name_pattern":
		if left.Name != right.Name {
			return false
		}
		if left.Negate != right.Negate {
			return false
		}
		if left.Operator != right.Operator {
			return false
		}
		if left.Pattern != right.Pattern {
			return false
		}
		return true
	}
	return false
}

type RuleSetDefinition struct {
	// Target // branch, tag
	Enforcement string // disabled, active, evaluate
	BypassApps  []struct {
		AppName string
		Mode    string // always, pull_request
	} `yaml:"bypassapps,omitempty"`
	BypassTeams []struct {
		TeamName string
		Mode     string // always, pull_request
	} `yaml:"bypassteams,omitempty"`
	Conditions struct {
		Include []string `yaml:"include,omitempty"` // ~DEFAULT_BRANCH, ~ALL, branch_name, ...
		Exclude []string `yaml:"exclude,omitempty"` //  branch_name, ...
	} `yaml:"conditions,omitempty"`

	Rules []struct {
		Ruletype   string            // required_signatures, pull_request, required_status_checks, creation, update, deletion, non_fast_forward
		Parameters RuleSetParameters `yaml:"parameters,omitempty"`
	} `yaml:"rules"`
}

/*
 * Ruleset are applied per repos based on the goliac configuration file (pattern x ruleset name)
 */
type RuleSet struct {
	Entity `yaml:",inline"`
	Spec   struct {
		Repositories struct {
			Included []string `yaml:"included"`
			Except   []string `yaml:"except"`
		} `yaml:"repositories"`
		Ruleset RuleSetDefinition `yaml:"ruleset"`
	}
}

/*
 * NewRuleSet reads a file and returns a RuleSet object
 * The next step is to validate the RuleSet object using the Validate method
 */
func NewRuleSet(fs billy.Filesystem, filename string) (*RuleSet, error) {
	filecontent, err := utils.ReadFile(fs, filename)
	if err != nil {
		return nil, err
	}

	ruleset := RuleSet{}
	err = yaml.Unmarshal(filecontent, &ruleset)
	if err != nil {
		return nil, err
	}

	// set default values for ruleset rules
	for i, rule := range ruleset.Spec.Ruleset.Rules {
		if rule.Ruletype == "pull_request" {
			if len(rule.Parameters.AllowedMergeMethods) == 0 {
				// default to MERGE, SQUASH, REBASE
				rule.Parameters.AllowedMergeMethods = []string{"MERGE", "SQUASH", "REBASE"}
				ruleset.Spec.Ruleset.Rules[i] = rule
			}
		}
		if rule.Ruletype == "merge_queue" {
			if rule.Parameters.CheckResponseTimeoutMinutes == 0 {
				// default to 10 minutes
				rule.Parameters.CheckResponseTimeoutMinutes = 10
				ruleset.Spec.Ruleset.Rules[i] = rule
			}
		}
	}

	return &ruleset, nil
}

/**
 * ReadRuleSetDirectory reads all the files in the dirname directory and returns
 * - a map of RuleSet objects
 * - a slice of errors that must stop the validation process
 * - a slice of warning that must not stop the validation process
 */
func ReadRuleSetDirectory(fs billy.Filesystem, dirname string, LogCollection *observability.LogCollection) map[string]*RuleSet {
	rulesets := make(map[string]*RuleSet)

	exist, err := utils.Exists(fs, dirname)
	if err != nil {
		LogCollection.AddError(err)
		return rulesets
	}
	if !exist {
		return rulesets
	}

	// Parse all the rulesets in the dirname directory
	entries, err := fs.ReadDir(dirname)
	if err != nil {
		LogCollection.AddError(err)
		return rulesets
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		// skipping files starting with '.'
		if e.Name()[0] == '.' {
			continue
		}
		ruleset, err := NewRuleSet(fs, filepath.Join(dirname, e.Name()))
		if err != nil {
			LogCollection.AddError(err)
		} else {
			err := ruleset.Validate(filepath.Join(dirname, e.Name()))
			if err != nil {
				LogCollection.AddError(err)
			} else {
				rulesets[ruleset.Name] = ruleset
			}

		}
	}
	return rulesets
}

func ValidateRulesetDefinition(r *RuleSetDefinition, filename string) error {
	for _, rule := range r.Rules {
		if rule.Ruletype != "required_signatures" &&
			rule.Ruletype != "pull_request" &&
			rule.Ruletype != "required_status_checks" &&
			rule.Ruletype != "creation" &&
			rule.Ruletype != "update" &&
			rule.Ruletype != "deletion" &&
			rule.Ruletype != "non_fast_forward" &&
			rule.Ruletype != "required_linear_history" &&
			rule.Ruletype != "branch_name_pattern" &&
			rule.Ruletype != "tag_name_pattern" &&
			rule.Ruletype != "merge_queue" {
			return fmt.Errorf("invalid ruletype: %s for ruleset filename %s", rule.Ruletype, filename)
		}

		if rule.Ruletype == "branch_name_pattern" ||
			rule.Ruletype == "tag_name_pattern" {
			if rule.Parameters.Operator != "starts_with" &&
				rule.Parameters.Operator != "ends_with" &&
				rule.Parameters.Operator != "contains" &&
				rule.Parameters.Operator != "regex" {
				return fmt.Errorf("invalid ruletype: %s for ruleset filename %s: operator must be 'starts_with','ends_with','contains' or 'regex' ", rule.Ruletype, filename)
			}
			if rule.Parameters.Pattern == "" {
				return fmt.Errorf("invalid ruletype: %s for ruleset filename %s: pattern must not be empty ", rule.Ruletype, filename)
			}
		}
		if rule.Ruletype == "pull_request" {
			if len(rule.Parameters.AllowedMergeMethods) == 0 {
				return fmt.Errorf("invalid ruletype: %s for ruleset filename %s: allowed_merge_methods must not be empty ", rule.Ruletype, filename)
			}
			for _, mergeMethod := range rule.Parameters.AllowedMergeMethods {
				if mergeMethod != "MERGE" && mergeMethod != "SQUASH" && mergeMethod != "REBASE" {
					return fmt.Errorf("invalid ruletype: %s for ruleset filename %s: allowed_merge_methods must be 'MERGE', 'SQUASH' or 'REBASE' ", rule.Ruletype, filename)
				}
			}
		}
		if rule.Ruletype == "merge_queue" {
			if rule.Parameters.CheckResponseTimeoutMinutes == 0 {
				return fmt.Errorf("invalid ruletype: %s for ruleset filename %s: checkResponseTimeoutMinutes must not be empty ", rule.Ruletype, filename)
			}
			if rule.Parameters.GroupingStrategy != "ALLGREEN" && rule.Parameters.GroupingStrategy != "HEADGREEN" {
				return fmt.Errorf("invalid ruletype: %s for ruleset filename %s: groupingStrategy must be 'ALLGREEN' or 'HEADGREEN' ", rule.Ruletype, filename)
			}
		}
	}

	if r.Enforcement != "disabled" && r.Enforcement != "active" && r.Enforcement != "evaluate" {
		return fmt.Errorf("invalid enforcement: %s for ruleset filename %s", r.Enforcement, filename)
	}

	return nil
}

func (r *RuleSet) Validate(filename string) error {

	if r.ApiVersion != "v1" {
		return fmt.Errorf("invalid apiVersion: %s for ruleset filename %s", r.ApiVersion, filename)
	}

	if r.Kind != "Ruleset" {
		return fmt.Errorf("invalid kind: %s for ruleset filename %s", r.Kind, filename)
	}

	if r.Name == "" {
		return fmt.Errorf("metadata.name is empty for ruleset filename %s", filename)
	}

	filename = filepath.Base(filename)
	if r.Name != filename[:len(filename)-len(filepath.Ext(filename))] {
		return fmt.Errorf("invalid metadata.name: %s for ruleset filename %s", r.Name, filename)
	}

	err := ValidateRulesetDefinition(&r.Spec.Ruleset, filename)
	if err != nil {
		return err
	}

	for _, ba := range r.Spec.Ruleset.BypassApps {
		if ba.AppName == "" {
			return fmt.Errorf("invalid appname: %s for bypassapp in ruleset filename %s", ba.AppName, filename)
		}
		if ba.Mode != "always" && ba.Mode != "pull_request" {
			return fmt.Errorf("invalid mode: %s for bypassapp %s in ruleset filename %s", ba.Mode, ba.AppName, filename)
		}
	}
	for _, bt := range r.Spec.Ruleset.BypassTeams {
		if bt.TeamName == "" {
			return fmt.Errorf("invalid teamname: %s for bypassteam in ruleset filename %s", bt.TeamName, filename)
		}
		if bt.Mode != "always" && bt.Mode != "pull_request" {
			return fmt.Errorf("invalid mode: %s for bypassteam %s in ruleset filename %s", bt.Mode, bt.TeamName, filename)
		}
	}
	for _, include := range r.Spec.Ruleset.Conditions.Include {
		if include[0] == '~' && (include != "~DEFAULT_BRANCH" && include != "~ALL") {
			return fmt.Errorf("invalid include: %s in ruleset filename %s", include, filename)
		}
	}
	for _, exclude := range r.Spec.Ruleset.Conditions.Exclude {
		if exclude[0] == '~' && (exclude != "~DEFAULT_BRANCH" && exclude != "~ALL") {
			return fmt.Errorf("invalid exclude: %s in ruleset filename %s", exclude, filename)
		}
	}

	// validate Repositories regex
	for _, repo := range r.Spec.Repositories.Included {
		if repo == "~ALL" {
			continue
		}
		_, err := regexp.Compile(fmt.Sprintf("^%s$", repo))
		if err != nil {
			return fmt.Errorf("error compiling regex %s: %v", repo, err)
		}
	}

	for _, repo := range r.Spec.Repositories.Except {
		_, err := regexp.Compile(fmt.Sprintf("^%s$", repo))
		if err != nil {
			return fmt.Errorf("error compiling regex %s: %v", repo, err)
		}
	}

	return nil
}

// this function will return repositories impacted by this Ruleset
func (r *RuleSet) BuildRepositoriesList(repos []string) ([]string, error) {
	repositoriesList := make([]string, 0)
	for _, repository := range repos {
		repoMatch := false
		if len(r.Spec.Repositories.Included) == 0 {
			repoMatch = true
		} else {
			for _, repo := range r.Spec.Repositories.Included {
				if repo == "~ALL" {
					repoMatch = true
					break
				}
				repoRegex, err := regexp.Match(fmt.Sprintf("^%s$", repo), []byte(repository))
				if err != nil {
					return nil, err
				}
				if repoRegex {
					repoMatch = true
					break
				}
			}
		}

		for _, repo := range r.Spec.Repositories.Except {
			repoRegex, err := regexp.Match(fmt.Sprintf("^%s$", repo), []byte(repository))
			if err != nil {
				return nil, err
			}
			if repoRegex {
				repoMatch = false
				break
			}
		}

		if repoMatch {
			repositoriesList = append(repositoriesList, repository)
		}
	}
	return repositoriesList, nil
}
