package entity

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type User struct {
	Entity `yaml:",inline"`
	Data   struct {
		GithubID string `yaml:"githubID"`
	} `yaml:"data"`
}

/*
 * NewUser reads a file and returns a User object
 * The next step is to validate the User object using the Validate method
 */
func NewUser(fs afero.Fs, filename string) (*User, error) {
	filecontent, err := afero.ReadFile(fs, filename)
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
func ReadUserDirectory(fs afero.Fs, dirname string) (map[string]*User, []error, []error) {
	errors := []error{}
	warning := []error{}
	users := make(map[string]*User)

	exist, err := afero.Exists(fs, dirname)
	if err != nil {
		errors = append(errors, err)
		return users, errors, warning
	}
	if exist == false {
		return users, errors, warning
	}

	// Parse all the users in the dirname directory
	entries, err := afero.ReadDir(fs, dirname)
	if err != nil {
		errors = append(errors, err)
		return users, errors, warning
	}

	for _, e := range entries {
		if !e.IsDir() {
			user, err := NewUser(fs, filepath.Join(dirname, e.Name()))
			if err != nil {
				errors = append(errors, err)
			} else {
				err = user.Validate(filepath.Join(dirname, e.Name()))
				if err != nil {
					errors = append(errors, err)
				} else {
					users[user.Metadata.Name] = user
				}
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

	if u.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is empty for user filename %s", filename)
	}

	filename = filepath.Base(filename)
	if u.Metadata.Name != filename[:len(filename)-len(filepath.Ext(filename))] {
		return fmt.Errorf("invalid metadata.name: %s for user filename %s", u.Metadata.Name, filename)
	}

	if u.Data.GithubID == "" {
		return fmt.Errorf("data.githubID is empty for user filename %s", filename)
	}

	return nil
}
