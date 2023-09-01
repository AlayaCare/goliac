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
	Data   struct {
		Writers             []string `yaml:"writers,omitempty"`
		Readers             []string `yaml:"readers,omitempty"`
		ExternalUserReaders []string `yaml:"externalUserReaders,omitempty"`
		ExternalUserWriters []string `yaml:"externalUserWriters,omitempty"`
		IsPublic            bool     `yaml:"public,omitempty"`
		IsArchived          bool     `yaml:"archived,omitempty"`
	} `yaml:"data,omitempty"`
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
			if !strings.HasSuffix(entry.Name(), ".yaml") {
				warning = append(warning, fmt.Errorf("File %s doesn't have a .yaml extension", entry.Name()))
				continue
			}
			repo, err := NewRepository(fs, filepath.Join(archivedDirname, entry.Name()))
			if err != nil {
				errors = append(errors, err)
			} else {
				if err := repo.Validate(filepath.Join(archivedDirname, entry.Name()), teams, externalUsers, true); err != nil {
					errors = append(errors, err)
				} else {
					repos[repo.Metadata.Name] = repo
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
						if err := repo.Validate(filepath.Join(teamDirname, team.Name(), sube.Name()), teams, externalUsers, false); err != nil {
							errors = append(errors, err)
						} else {
							// check if the repository doesn't already exists
							if _, exist := repos[repo.Metadata.Name]; exist {
								existing := filepath.Join(archivedDirname, repo.Metadata.Name)
								if repos[repo.Metadata.Name].Owner != nil {
									existing = filepath.Join(teamDirname, *repos[repo.Metadata.Name].Owner, repo.Metadata.Name)
								}
								errors = append(errors, fmt.Errorf("Repository %s defined in 2 places (check %s and %s)", repo.Metadata.Name, filepath.Join(teamDirname, team.Name(), sube.Name()), existing))
							} else {
								teamname := team.Name()
								repo.Owner = &teamname
								repos[repo.Metadata.Name] = repo
							}
						}
					}
				}
			}
		}
	}

	return repos, errors, warning
}

func (r *Repository) Validate(filename string, teams map[string]*Team, externalUsers map[string]*User, archived bool) error {

	if r.ApiVersion != "v1" {
		return fmt.Errorf("invalid apiVersion: %s (check repository filename %s)", r.ApiVersion, filename)
	}

	if r.Kind != "Repository" {
		return fmt.Errorf("invalid kind: %s (check repository filename %s)", r.Kind, filename)
	}

	if r.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is empty (check repository filename %s)", filename)
	}

	filename = filepath.Base(filename)
	if r.Metadata.Name != filename[:len(filename)-len(filepath.Ext(filename))] {
		return fmt.Errorf("invalid metadata.name: %s for repository filename %s", r.Metadata.Name, filename)
	}

	for _, writer := range r.Data.Writers {
		if _, ok := teams[writer]; !ok {
			return fmt.Errorf("invalid writer: %s doesn't exist (check repository filename %s)", writer, filename)
		}
	}
	for _, reader := range r.Data.Readers {
		if _, ok := teams[reader]; !ok {
			return fmt.Errorf("invalid reader: %s doesn't exist (check repository filename %s)", reader, filename)
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

	if archived != r.Data.IsArchived {
		if archived {
			return fmt.Errorf("invalid archived: %s is in the archived directory without the `archived` boolean", filename)
		} else {
			return fmt.Errorf("invalid archived: %s has `archived` set to true, but isn't in the archived directory", filename)
		}
	}
	return nil
}
