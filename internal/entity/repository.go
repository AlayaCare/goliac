package entity

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type Repository struct {
	Entity `yaml:",inline"`
	Data   struct {
		Writers             []string `yaml:"writers"`
		Readers             []string `yaml:"readers"`
		ExternalUserReaders []string `yaml:"externalUserReaders"`
		ExternalUserWriters []string `yaml:"externalUserWriters"`
	} `yaml:"data"`
}

/*
 * NewRepository reads a file and returns a Repository object
 * The next step is to validate the Repository object using the Validate method
 */
func NewRepository(fs afero.Fs, filename string) (*Repository, error) {
	filecontent, err := afero.ReadFile(fs, filename)
	if err != nil {
		return nil, err
	}

	repository := &Repository{}
	err = yaml.Unmarshal(filecontent, repository)
	if err != nil {
		return nil, err
	}

	return repository, nil
}

/**
 * ReadRepositories reads all the files in the dirname directory and returns
 * - a map of Repository objects
 * - a slice of errors that must stop the vlidation process
 * - a slice of warning that must not stop the validation process
 */
func ReadRepositories(fs afero.Fs, teamDirname string, teams map[string]*Team, externalUsers map[string]*User) (map[string]*Repository, []error, []error) {
	errors := []error{}
	warning := []error{}
	repos := make(map[string]*Repository)

	exist, err := afero.Exists(fs, teamDirname)
	if err != nil {
		errors = append(errors, err)
		return repos, errors, warning
	}
	if exist == false {
		return repos, errors, warning
	}

	// Parse all the repositories in the teamDirname directory
	entries, err := afero.ReadDir(fs, teamDirname)
	if err != nil {
		errors = append(errors, err)
		return nil, errors, warning
	}

	for _, team := range entries {
		if team.IsDir() {
			subentries, err := afero.ReadDir(fs, filepath.Join(teamDirname, team.Name()))
			if err != nil {
				errors = append(errors, err)
				continue
			}
			for _, sube := range subentries {
				if !sube.IsDir() && filepath.Ext(sube.Name()) == ".yaml" && sube.Name() != "team.yaml" {
					repo, err := NewRepository(fs, filepath.Join(teamDirname, team.Name(), sube.Name()))
					if err != nil {
						errors = append(errors, err)
					} else {
						if err := repo.Validate(filepath.Join(teamDirname, team.Name(), sube.Name()), teams, externalUsers); err != nil {
							errors = append(errors, err)
						} else {
							repo.Data.Writers = append(repo.Data.Writers, team.Name())
							repos[repo.Metadata.Name] = repo
						}
					}
				}
			}
		}
	}

	return repos, errors, warning
}

func (r *Repository) Validate(filename string, teams map[string]*Team, externalUsers map[string]*User) error {

	if r.ApiVersion != "v1" {
		return fmt.Errorf("invalid apiVersion: %s for repository filename %s", r.ApiVersion, filename)
	}

	if r.Kind != "Repository" {
		return fmt.Errorf("invalid kind: %s for repository filename %s", r.Kind, filename)
	}

	if r.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is empty for repository filename %s", filename)
	}

	filename = filepath.Base(filename)
	if r.Metadata.Name != filename[:len(filename)-len(filepath.Ext(filename))] {
		return fmt.Errorf("invalid metadata.name: %s for repository filename %s", r.Metadata.Name, filename)
	}

	for _, writer := range r.Data.Writers {
		if _, ok := teams[writer]; !ok {
			return fmt.Errorf("invalid writer: %s doesn't exist in repository filename %s", writer, filename)
		}
	}
	for _, reader := range r.Data.Readers {
		if _, ok := teams[reader]; !ok {
			return fmt.Errorf("invalid reader: %s doesn't exist in repository filename %s", reader, filename)
		}
	}

	for _, externalUserReader := range r.Data.ExternalUserReaders {
		if _, ok := externalUsers[externalUserReader]; !ok {
			return fmt.Errorf("invalid externalUserReader: %s doesn't exist in repository filename %s", externalUserReader, filename)
		}
	}

	for _, externalUserWriter := range r.Data.ExternalUserWriters {
		if _, ok := externalUsers[externalUserWriter]; !ok {
			return fmt.Errorf("invalid externalUserWriter: %s doesn't exist in repository filename %s", externalUserWriter, filename)
		}
	}

	return nil
}
