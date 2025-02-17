package entity

import (
	"fmt"
	"path/filepath"

	"github.com/Alayacare/goliac/internal/utils"
	"github.com/go-git/go-billy/v5"
	"gopkg.in/yaml.v3"
)

type RuleSetParameters struct {
	// PullRequestParameters
	DismissStaleReviewsOnPush      bool `yaml:"dismissStaleReviewsOnPush,omitempty"`
	RequireCodeOwnerReview         bool `yaml:"requireCodeOwnerReview,omitempty"`
	RequiredApprovingReviewCount   int  `yaml:"requiredApprovingReviewCount,omitempty"`
	RequiredReviewThreadResolution bool `yaml:"requiredReviewThreadResolution,omitempty"`
	RequireLastPushApproval        bool `yaml:"requireLastPushApproval,omitempty"`

	// RequiredStatusChecksParameters
	RequiredStatusChecks             []string `yaml:"requiredStatusChecks,omitempty"`
	StrictRequiredStatusChecksPolicy bool     `yaml:"strictRequiredStatusChecksPolicy,omitempty"`
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
		return true
	case "required_status_checks":
		if res, _, _ := StringArrayEquivalent(left.RequiredStatusChecks, right.RequiredStatusChecks); !res {
			return false
		}
		if left.StrictRequiredStatusChecksPolicy != right.StrictRequiredStatusChecksPolicy {
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
	Spec   RuleSetDefinition `yaml:"spec"`
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

	return &ruleset, nil
}

/**
 * ReadRuleSetDirectory reads all the files in the dirname directory and returns
 * - a map of RuleSet objects
 * - a slice of errors that must stop the validation process
 * - a slice of warning that must not stop the validation process
 */
func ReadRuleSetDirectory(fs billy.Filesystem, dirname string) (map[string]*RuleSet, []error, []Warning) {
	errors := []error{}
	warning := []Warning{}
	rulesets := make(map[string]*RuleSet)

	exist, err := utils.Exists(fs, dirname)
	if err != nil {
		errors = append(errors, err)
		return rulesets, errors, warning
	}
	if !exist {
		return rulesets, errors, warning
	}

	// Parse all the rulesets in the dirname directory
	entries, err := fs.ReadDir(dirname)
	if err != nil {
		errors = append(errors, err)
		return rulesets, errors, warning
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
			errors = append(errors, err)
		} else {
			err := ruleset.Validate(filepath.Join(dirname, e.Name()))
			if err != nil {
				errors = append(errors, err)
			} else {
				rulesets[ruleset.Name] = ruleset
			}

		}
	}
	return rulesets, errors, warning
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

	for _, rule := range r.Spec.Rules {
		if rule.Ruletype != "required_signatures" &&
			rule.Ruletype != "pull_request" &&
			rule.Ruletype != "required_status_checks" &&
			rule.Ruletype != "creation" &&
			rule.Ruletype != "update" &&
			rule.Ruletype != "deletion" &&
			rule.Ruletype != "non_fast_forward" {
			return fmt.Errorf("invalid rulettype: %s for ruleset filename %s", rule.Ruletype, filename)
		}
	}

	if r.Spec.Enforcement != "disable" && r.Spec.Enforcement != "active" && r.Spec.Enforcement != "evaluate" {
		return fmt.Errorf("invalid enforcement: %s for ruleset filename %s", r.Spec.Enforcement, filename)
	}

	for _, ba := range r.Spec.BypassApps {
		if ba.Mode != "always" && ba.Mode != "pull_request" {
			return fmt.Errorf("invalid mode: %s for bypassapp %s in ruleset filename %s", ba.Mode, ba.AppName, filename)
		}
	}
	for _, include := range r.Spec.Conditions.Include {
		if include[0] == '~' && (include != "~DEFAULT_BRANCH" && include != "~ALL") {
			return fmt.Errorf("invalid include: %s in ruleset filename %s", include, filename)
		}
	}
	for _, exclude := range r.Spec.Conditions.Exclude {
		if exclude[0] == '~' && (exclude != "~DEFAULT_BRANCH" && exclude != "~ALL") {
			return fmt.Errorf("invalid exclude: %s in ruleset filename %s", exclude, filename)
		}
	}

	return nil
}
