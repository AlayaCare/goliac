package config

import (
	"gopkg.in/yaml.v3"
)

type GithubCustomProperty struct {
	PropertyName     string   `yaml:"property_name" json:"property_name"`
	ValueType        string   `yaml:"value_type" json:"value_type"` // "string", "single_select", "multi_select"
	Required         bool     `yaml:"required,omitempty" json:"required,omitempty"`
	DefaultValue     string   `yaml:"default_value,omitempty" json:"default_value,omitempty"`
	Description      string   `yaml:"description,omitempty" json:"description,omitempty"`
	AllowedValues    []string `yaml:"allowed_values,omitempty" json:"allowed_values,omitempty"`
	ValuesEditableBy string   `yaml:"values_editable_by,omitempty" json:"values_editable_by,omitempty"` // "org_actors", "org_and_repo_actors"
}

type RepositoryConfig struct {
	AdminTeam           string `yaml:"admin_team"`
	EveryoneTeamEnabled bool   `yaml:"everyone_team_enabled"`

	Rulesets                []string
	MaxChangesets           int `yaml:"max_changesets"`
	GithubConcurrentThreads int `yaml:"github_concurrent_threads"`
	UserSync                struct {
		Plugin string `yaml:"plugin"`
		Path   string `yaml:"path"`
	}
	ArchiveOnDelete       bool `yaml:"archive_on_delete"`
	DestructiveOperations struct {
		AllowDestructiveRepositories bool `yaml:"repositories"`
		AllowDestructiveTeams        bool `yaml:"teams"`
		AllowDestructiveUsers        bool `yaml:"users"`
		AllowDestructiveRulesets     bool `yaml:"rulesets"`
	} `yaml:"destructive_operations"`

	VisibilityRules struct {
		ForbidPublicRepositories           bool     `yaml:"forbid_public_repositories"`
		ForbidPublicRepositoriesExclusions []string `yaml:"forbid_public_repositories_exclusions"`
	} `yaml:"visibility_rules"`

	Workflows           []string               `yaml:"workflows"`
	OrgCustomProperties []GithubCustomProperty `yaml:"org_custom_properties"`
}

// set default values
func (rc *RepositoryConfig) UnmarshalYAML(value *yaml.Node) error {
	type myStructAlias RepositoryConfig // Create a new alias type to avoid recursion
	x := &myStructAlias{}
	x.AdminTeam = "admin"
	x.MaxChangesets = 50
	x.GithubConcurrentThreads = 4
	x.UserSync.Plugin = "noop"
	x.ArchiveOnDelete = true

	if err := value.Decode(x); err != nil {
		return err
	}

	*rc = RepositoryConfig(*x)
	return nil
}
