package entity

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Alayacare/goliac/internal/utils"
	"github.com/go-git/go-billy/v5"
	"gopkg.in/yaml.v3"
)

type Team struct {
	Entity `yaml:",inline"`
	Spec   struct {
		Owners  []string `yaml:"owners,omitempty"`
		Members []string `yaml:"members,omitempty"`
	} `yaml:"spec"`
}

/*
 * NewTeam reads a file and returns a Team object
 * The next step is to validate the Team object using the Validate method
 */
func NewTeam(fs billy.Filesystem, filename string) (*Team, error) {
	filecontent, err := utils.ReadFile(fs, filename)
	if err != nil {
		return nil, err
	}

	team := &Team{}
	err = yaml.Unmarshal(filecontent, team)
	if err != nil {
		return nil, err
	}

	return team, nil
}

/**
 * ReadTeamDirectory reads all the files in the dirname directory and returns
 * - a map of Team objects
 * - a slice of errors that must stop the validation process
 * - a slice of warning that must not stop the validation process
 */
func ReadTeamDirectory(fs billy.Filesystem, dirname string, users map[string]*User) (map[string]*Team, []error, []Warning) {
	errors := []error{}
	warning := []Warning{}
	teams := make(map[string]*Team)

	exist, err := utils.Exists(fs, dirname)
	if err != nil {
		errors = append(errors, err)
		return teams, errors, warning
	}
	if !exist {
		return teams, errors, warning
	}

	// Parse all the teams in the dirname directory
	entries, err := fs.ReadDir(dirname)
	if err != nil {
		errors = append(errors, err)
		return nil, errors, warning
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// skipping files starting with '.'
		if e.Name()[0] == '.' {
			continue
		}
		team, err := NewTeam(fs, filepath.Join(dirname, e.Name(), "team.yaml"))
		if err != nil {
			errors = append(errors, err)
		} else {
			err, warns := team.Validate(filepath.Join(dirname, e.Name()), users)
			warning = append(warning, warns...)
			if err != nil {
				errors = append(errors, err)
			} else {
				teams[team.Name] = team
			}

		}
	}
	return teams, errors, warning
}

func (t *Team) Validate(dirname string, users map[string]*User) (error, []Warning) {
	warnings := []Warning{}

	if t.ApiVersion != "v1" {
		return fmt.Errorf("invalid apiVersion: %s for team filename %s/team.yaml", t.ApiVersion, dirname), warnings
	}

	if t.Kind != "Team" {
		return fmt.Errorf("invalid kind: %s for team filename %s/team.yaml", t.Kind, dirname), warnings
	}

	if t.Name == "" {
		return fmt.Errorf("metadata.name is empty for team filename %s", dirname), warnings
	}

	if t.Name == "everyone" {
		return fmt.Errorf("team name 'everyone' is reserved"), warnings
	}

	if strings.HasSuffix(t.Name, "-owners") {
		return fmt.Errorf("metadata.name cannot finish with '-owners' for team filename %s. It is a reserved suffix", dirname), warnings
	}

	teamname := filepath.Base(dirname)
	if t.Name != teamname {
		return fmt.Errorf("invalid metadata.name: %s for team filename %s/team.yaml", t.Name, dirname), warnings
	}

	for _, owner := range t.Spec.Owners {
		if _, ok := users[owner]; !ok {
			return fmt.Errorf("invalid owner: %s doesn't exist in team filename %s/team.yaml", owner, dirname), warnings
		}
	}

	for _, member := range t.Spec.Members {
		if _, ok := users[member]; !ok {
			return fmt.Errorf("invalid member: %s doesn't exist in team filename %s/team.yaml", member, dirname), warnings
		}
	}

	// warnings

	if len(t.Spec.Owners) < 2 {
		warnings = append(warnings, fmt.Errorf("no enough owner for team filename %s/team.yaml", dirname))
	}

	return nil, warnings
}

/**
 * AdjustTeamDirectory adjust team's defintion depending on user availability.
 * The goal is that if a user has been removed, we must update the team definition.
 * Returns:
 * - a list of (team's) file changes (to commit to Github)
 */
func ReadAndAdjustTeamDirectory(fs billy.Filesystem, dirname string, users map[string]*User) ([]string, error) {
	teamschanged := []string{}

	exist, err := utils.Exists(fs, dirname)
	if err != nil {
		return teamschanged, err
	}
	if !exist {
		return teamschanged, nil
	}

	// Parse all the teams in the dirname directory
	entries, err := fs.ReadDir(dirname)
	if err != nil {
		return teamschanged, err
	}

	for _, e := range entries {
		if e.IsDir() {
			team, err := NewTeam(fs, filepath.Join(dirname, e.Name(), "team.yaml"))
			if err != nil {
				return teamschanged, err
			} else {
				changed, err := team.Update(fs, filepath.Join(dirname, e.Name(), "team.yaml"), users)
				if err != nil {
					return teamschanged, err
				}
				if changed {
					teamschanged = append(teamschanged, filepath.Join(dirname, e.Name(), "team.yaml"))
				}
			}
		}
	}
	return teamschanged, nil
}

// Update is telling if the team needs to be adjust (and the team's definition was changed on disk),
// based on the list of (still) existing users
func (t *Team) Update(fs billy.Filesystem, filename string, users map[string]*User) (bool, error) {
	changed := false
	owners := make([]string, 0)
	for _, owner := range t.Spec.Owners {
		if _, ok := users[owner]; !ok {
			changed = true
		} else {
			owners = append(owners, owner)
		}
	}
	t.Spec.Owners = owners

	members := make([]string, 0)
	for _, member := range t.Spec.Members {
		if _, ok := users[member]; !ok {
			changed = true
		} else {
			members = append(members, member)
		}
	}
	t.Spec.Members = members

	file, err := fs.Create(filename)
	if err != nil {
		return changed, fmt.Errorf("Not able to create file %s: %v", filename, err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	err = encoder.Encode(t)
	if err != nil {
		return changed, fmt.Errorf("Not able to write file %s: %v", filename, err)
	}
	return changed, err
}
