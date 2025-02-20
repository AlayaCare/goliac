package entity

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/goliac-project/goliac/internal/utils"
	"gopkg.in/yaml.v3"
)

type User struct {
	Entity `yaml:",inline"`
	Spec   struct {
		GithubID string `yaml:"githubID"`
	} `yaml:"spec"`
}

/*
 * NewUser reads a file and returns a User object
 * The next step is to validate the User object using the Validate method
 */
func NewUser(fs billy.Filesystem, filename string) (*User, error) {
	filecontent, err := utils.ReadFile(fs, filename)
	if err != nil {
		return nil, err
	}

	user := &User{}
	err = yaml.Unmarshal(filecontent, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

/**
 * ReadUserDirectory reads all the files in the dirname directory and returns
 * - a map of User objects
 * - a slice of errors that must stop the vlidation process
 * - a slice of warning that must not stop the validation process
 */
func ReadUserDirectory(fs billy.Filesystem, dirname string) (map[string]*User, []error, []Warning) {
	errors := []error{}
	warning := []Warning{}
	users := make(map[string]*User)

	exist, err := utils.Exists(fs, dirname)
	if err != nil {
		errors = append(errors, err)
		return users, errors, warning
	}
	if !exist {
		return users, errors, warning
	}

	// Parse all the users in the dirname directory
	entries, err := fs.ReadDir(dirname)
	if err != nil {
		errors = append(errors, err)
		return users, errors, warning
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		// skipping files starting with '.'
		if e.Name()[0] == '.' {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		user, err := NewUser(fs, filepath.Join(dirname, e.Name()))
		if err != nil {
			errors = append(errors, err)
		} else {
			err = user.Validate(filepath.Join(dirname, e.Name()))
			if err != nil {
				errors = append(errors, err)
			} else {
				users[user.Name] = user
			}
		}

	}
	return users, errors, warning
}

func (u *User) Validate(filename string) error {

	if u.ApiVersion != "v1" {
		return fmt.Errorf("invalid apiVersion: %s for user filename %s", u.ApiVersion, filename)
	}

	if u.Kind != "User" {
		return fmt.Errorf("invalid kind: %s for user filename %s", u.Kind, filename)
	}

	if u.Name == "" {
		return fmt.Errorf("metadata.name is empty for user filename %s", filename)
	}

	filename = filepath.Base(filename)
	if u.Name != filename[:len(filename)-len(filepath.Ext(filename))] {
		return fmt.Errorf("invalid metadata.name: %s for user filename %s", u.Name, filename)
	}

	if u.Spec.GithubID == "" {
		return fmt.Errorf("spec.githubID is empty for user filename %s", filename)
	}

	return nil
}

func (u *User) Equals(a *User) bool {
	if u.ApiVersion != a.ApiVersion {
		return false
	}
	if u.Kind != a.Kind {
		return false
	}
	if u.Name != a.Name {
		return false
	}
	if u.Spec.GithubID != a.Spec.GithubID {
		return false
	}

	return true
}
