package entity

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type Repository struct {
	Entity `yaml:",inline"`
	Spec   struct {
		Writers             []string `yaml:"writers,omitempty"`
		Readers             []string `yaml:"readers,omitempty"`
		ExternalUserReaders []string `yaml:"externalUserReaders,omitempty"`
		ExternalUserWriters []string `yaml:"externalUserWriters,omitempty"`
		IsPublic            bool     `yaml:"public,omitempty"`
		IsArchived          bool     `yaml:"archived,omitempty"` // implicit: will be set by Goliac
	} `yaml:"spec,omitempty"`
	Owner *string `yaml:"owner,omitempty"` // implicit. team name owning the repo (if any)
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
 * ReadRepositories reads all the files in the dirname directory and
 * add them to the owner's team and returns
 * - a map of Repository objects
 * - a slice of errors that must stop the validation process
 * - a slice of warning that must not stop the validation process
 */
func ReadRepositories(fs afero.Fs, archivedDirname string, teamDirname string, teams map[string]*Team, externalUsers map[string]*User) (map[string]*Repository, []error, []Warning) {
	errors := []error{}
	warning := []Warning{}
	repos := make(map[string]*Repository)

	// archived dir
	exist, err := afero.Exists(fs, archivedDirname)
	if err != nil {
		errors = append(errors, err)
		return repos, errors, warning
	}
	if exist {
		entries, err := afero.ReadDir(fs, archivedDirname)
		if err != nil {
			errors = append(errors, err)
			return nil, errors, warning
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
				warning = append(warning, fmt.Errorf("File %s doesn't have a .yaml extension", entry.Name()))
				continue
			}
			repo, err := NewRepository(fs, filepath.Join(archivedDirname, entry.Name()))
			if err != nil {
				errors = append(errors, err)
			} else {
				if err := repo.Validate(filepath.Join(archivedDirname, entry.Name()), teams, externalUsers); err != nil {
					errors = append(errors, err)
				} else {
					repo.Spec.IsArchived = true
					repos[repo.Name] = repo
				}
			}
		}
	}
	// regular teams dir
	exist, err = afero.Exists(fs, teamDirname)
	if err != nil {
		errors = append(errors, err)
		return repos, errors, warning
	}
	if !exist {
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
							// check if the repository doesn't already exists
							if _, exist := repos[repo.Name]; exist {
								existing := filepath.Join(archivedDirname, repo.Name)
								if repos[repo.Name].Owner != nil {
									existing = filepath.Join(teamDirname, *repos[repo.Name].Owner, repo.Name)
								}
								errors = append(errors, fmt.Errorf("Repository %s defined in 2 places (check %s and %s)", repo.Name, filepath.Join(teamDirname, team.Name(), sube.Name()), existing))
							} else {
								teamname := team.Name()
								repo.Owner = &teamname
								repo.Spec.IsArchived = false
								repos[repo.Name] = repo
							}
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

	return nil
}
