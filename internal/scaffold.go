package internal

import (
	"fmt"
	"path"
	"strings"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/engine"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/github"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type LoadGithubSamlUsers func() (map[string]*entity.User, error)

type Scaffold struct {
	remote                     engine.GoliacRemote
	loadUsersFromGithubOrgSaml LoadGithubSamlUsers
}

func NewScaffold() (*Scaffold, error) {
	githubClient, err := github.NewGitHubClientImpl(
		config.Config.GithubServer,
		config.Config.GithubAppOrganization,
		config.Config.GithubAppID,
		config.Config.GithubAppPrivateKeyFile,
	)

	if err != nil {
		return nil, err
	}

	remote := engine.NewGoliacRemoteImpl(githubClient)

	return &Scaffold{
		remote: remote,
		loadUsersFromGithubOrgSaml: func() (map[string]*entity.User, error) {
			return engine.LoadUsersFromGithubOrgSaml(githubClient)
		},
	}, nil
}

/*
 * Generate will generate a full teams directory structure compatible with Goliac
 */
func (s *Scaffold) Generate(rootpath string, adminteam string) error {
	fs := afero.NewOsFs()
	if err := s.remote.Load(true); err != nil {
		logrus.Warnf("Not able to load all information from Github: %v, but I will try to continue", err)
	}
	return s.generate(fs, rootpath, adminteam)
}

func (s *Scaffold) generate(fs afero.Fs, rootpath string, adminteam string) error {
	fs.RemoveAll(path.Join(rootpath, "users"))
	fs.RemoveAll(path.Join(rootpath, "teams"))
	fs.RemoveAll(path.Join(rootpath, "rulesets"))
	fs.RemoveAll(path.Join(rootpath, "archived"))

	fs.MkdirAll(path.Join(rootpath, "archived"), 0755)
	fs.MkdirAll(path.Join(rootpath, "rulesets"), 0755)
	fs.MkdirAll(path.Join(rootpath, "teams"), 0755)

	usermap, err := s.generateUsers(fs, path.Join(rootpath, "users"))
	if err != nil {
		return fmt.Errorf("Error creaing the users directory: %v", err)
	}

	err, foundAdmin := s.generateTeams(fs, path.Join(rootpath, "teams"), usermap, adminteam)
	if err != nil {
		return fmt.Errorf("Error creating the teams directory: %v", err)
	}

	if err := s.generateRuleset(fs, path.Join(rootpath, "rulesets")); err != nil {
		return fmt.Errorf("Error creating the rulesets directory: %v", err)
	}

	if err := s.generateGoliacConf(fs, rootpath, adminteam); err != nil {
		return fmt.Errorf("Error creating the goliac.yaml file: %v", err)
	}

	if err := s.generateGithubAction(fs, rootpath); err != nil {
		return fmt.Errorf("Error creating the .github/workflows/pr.yaml file: %v", err)
	}

	if err := s.generateReadme(fs, rootpath); err != nil {
		return fmt.Errorf("Error creating the README.md file: %v", err)
	}

	if !foundAdmin {
		return fmt.Errorf("The admin team '%s' was not found", adminteam)
	}

	return nil
}

func (s *Scaffold) generateTeams(fs afero.Fs, teamspath string, usermap map[string]string, adminteam string) (error, bool) {
	adminTeamFound := false

	teamsRepositories := s.remote.TeamRepositories()
	teams := s.remote.Teams()

	// to ensure only one owner
	repoAdmin := make(map[string]string)
	teamsRepos := make(map[string][]string)
	// to get all teams access per repo
	repoWrite := make(map[string][]string)
	repoRead := make(map[string][]string)

	// searching for ADMIN first
	for team, tr := range teamsRepositories {
		for reponame, repo := range tr {
			if repo.Permission == "ADMIN" {
				// if there is no admin attached yet to this repo
				if _, ok := repoAdmin[reponame]; !ok {
					repoAdmin[reponame] = team
					teamsRepos[team] = append(teamsRepos[team], reponame)
				}
				repoWrite[reponame] = append(repoWrite[reponame], team)
			}
		}
	}
	// searching for WRITE second
	for team, tr := range teamsRepositories {
		for reponame, repo := range tr {
			if repo.Permission == "WRITE" {
				// if there is no admin attached yet to this repo
				if _, ok := repoAdmin[reponame]; !ok {
					repoAdmin[reponame] = team
					teamsRepos[team] = append(teamsRepos[team], reponame)
				}
				repoWrite[reponame] = append(repoWrite[reponame], team)
			}
			if repo.Permission != "ADMIN" && repo.Permission != "WRITE" {
				repoRead[reponame] = append(repoRead[reponame], team)
			}
		}
	}

	countOrphaned := 0
	// orphan repos should go to the admin team
	for repo := range s.remote.Repositories() {
		logrus.Debugf("repo %s is orphaned, attaching it to the admin (%s) team", repo, adminteam)
		if _, ok := repoAdmin[repo]; !ok {
			repoAdmin[repo] = adminteam
			teamsRepos[adminteam] = append(teamsRepos[adminteam], repo)
			countOrphaned++
		}
	}
	logrus.Infof("%d orphaned repositories have been added to the admin %s team", countOrphaned, adminteam)

	for team, repos := range teamsRepos {
		// write the team dir
		if t := teams[team]; t != nil {
			if strings.HasSuffix(team, "-owners") {
				continue
			}
			lTeam := entity.Team{}
			lTeam.ApiVersion = "v1"
			lTeam.Kind = "Team"
			lTeam.Name = team
			for _, m := range t.Members {
				// put the right user name instead of the github id
				lTeam.Spec.Owners = append(lTeam.Spec.Owners, usermap[m])
			}
			fs.MkdirAll(path.Join(teamspath, team), 0755)
			if err := writeYamlFile(path.Join(teamspath, team, "team.yaml"), &lTeam, fs); err != nil {
				logrus.Errorf("not able to write team file %s: %v", team, err)
			}

			// write repos
			for _, r := range repos {
				lRepo := entity.Repository{}
				lRepo.ApiVersion = "v1"
				lRepo.Kind = "Repository"
				lRepo.Name = r
				lRepo.Spec.Writers = repoWrite[r]
				lRepo.Spec.Readers = repoRead[r]

				// removing team name from writer
				for i, t := range lRepo.Spec.Writers {
					if t == team {
						lRepo.Spec.Writers = append(lRepo.Spec.Writers[:i], lRepo.Spec.Writers[i+1:]...)
						break
					}
				}
				// removing team owner (especially for the special case teams repo)
				for i, t := range lRepo.Spec.Writers {
					if strings.HasSuffix(t, "-owners") {
						lRepo.Spec.Writers = append(lRepo.Spec.Writers[:i], lRepo.Spec.Writers[i+1:]...)
						break
					}
				}
				if err := writeYamlFile(path.Join(teamspath, team, r+".yaml"), &lRepo, fs); err != nil {
					logrus.Errorf("not able to write repo file %s/%s.yaml: %v", team, r, err)
				}
			}
		}
	}

	for team, t := range teams {
		if strings.HasSuffix(team, "-owners") {
			continue
		}

		if team == adminteam {
			adminTeamFound = true
		}

		// searching for loney team (ie without repos)
		if _, ok := teamsRepos[team]; !ok {
			lTeam := entity.Team{}
			lTeam.ApiVersion = "v1"
			lTeam.Kind = "Team"
			lTeam.Name = team
			for _, m := range t.Members {
				// put the right user name instead of the github id
				lTeam.Spec.Owners = append(lTeam.Spec.Owners, usermap[m])
			}
			fs.MkdirAll(path.Join(teamspath, team), 0755)
			if err := writeYamlFile(path.Join(teamspath, team, "team.yaml"), &lTeam, fs); err != nil {
				logrus.Errorf("not able to write team file %s/team.yaml: %v", team, err)
			}

		}
	}

	return nil, adminTeamFound
}

/*
 * Returns a map[<githubid>]<username>
 */
func (s *Scaffold) generateUsers(fs afero.Fs, userspath string) (map[string]string, error) {
	fs.MkdirAll(path.Join(userspath, "protected"), 0755)
	fs.MkdirAll(path.Join(userspath, "org"), 0755)
	fs.MkdirAll(path.Join(userspath, "external"), 0755)

	usermap := make(map[string]string)
	// test SAML integration
	users, err := s.loadUsersFromGithubOrgSaml()

	if len(users) > 0 && err == nil {
		logrus.Debug("SAML integration enabled")
		for username, user := range users {
			usermap[user.Spec.GithubID] = username
			if err := writeYamlFile(path.Join(userspath, "org", username+".yaml"), &user, fs); err != nil {
				logrus.Errorf("Not able to write user file org/%s.yaml: %v", username, err)
			}
		}
	} else {
		// fail back on github id
		logrus.Debug("SAML integration disabled")
		for githubid := range s.remote.Users() {
			usermap[githubid] = githubid
			user := entity.User{}
			user.ApiVersion = "v1"
			user.Kind = "User"
			user.Name = githubid
			user.Spec.GithubID = githubid

			if err := writeYamlFile(path.Join(userspath, "org", githubid+".yaml"), user, fs); err != nil {
				logrus.Errorf("Not able to write user file org/%s.yaml: %v", githubid, err)
			}
		}
	}

	return usermap, nil
}

func (s *Scaffold) generateRuleset(fs afero.Fs, rulesetspath string) error {
	ruleset := `apiVersion: v1
kind: Ruleset
name: default
spec:
  enforcement: active
  bypassapps:
    - appname: goliac-project-app
      mode: always
  on:
    include: 
    - "~DEFAULT_BRANCH"

  rules:
    - ruletype: pull_request
      parameters:
        requiredApprovingReviewCount: 1
`
	if err := writeFile(path.Join(rulesetspath, "default.yaml"), []byte(ruleset), fs); err != nil {
		return err
	}
	return nil

}

func (s *Scaffold) generateGoliacConf(fs afero.Fs, rootpath string, adminteam string) error {
	conf := fmt.Sprintf(`
admin_team: %s

rulesets:
  - pattern: .*
    ruleset: default

max_changesets: 50
archive_on_delete: true

destructive_operations:
  repositories: false
  teams: false
  users: false
  rulesets: false

usersync:
  plugin: noop
`, adminteam)
	if err := writeFile(path.Join(rootpath, "goliac.yaml"), []byte(conf), fs); err != nil {
		return err
	}
	return nil
}

func (s *Scaffold) generateGithubAction(fs afero.Fs, rootpath string) error {
	fs.MkdirAll(path.Join(rootpath, ".github", "workflows"), 0755)

	workflow := `
name: Validate structure

on: [pull_request]

jobs:
	build:
		name: validate
		runs-on: ubuntu-latest
		steps:
			- uses: actions/checkout@v3

			- name: Verify
			uses: addnab/docker-run-action@v3
			with:
				image: ghcr.io/nzin/goliac
				options: -v ${{ github.workspace }}:/work 
				run: /app/goliac verify /work
`
	if err := writeFile(path.Join(rootpath, ".github", "workflows", "pr.yaml"), []byte(workflow), fs); err != nil {
		return err
	}
	return nil
}

func (s *Scaffold) generateReadme(fs afero.Fs, rootpath string) error {
	readme := `
# teams

This repository manage the Github organization through [Goliac](https://github.com/alayacare/goliac) application

## Create a repository

On a given team subdirectory you can create a repository definition via a yaml file (like ` + "`" + `/teams/foobar/awesome-repository.yaml` + "`" + `):

` + "```" + `
apiVersion: v1
kind: Repository
name: awesome-repository
` + "```" + `

This will create a ` + "`" + `awesome-repository` + "`" + ` repository under your organization, that will be 
- private by default
- writable by all owners/members of this team (in our example ` + "`" + `foobar` + "`" + `)

You can of course tweak that:

` + "```" + `
apiVersion: v1
kind: Repository
name: awesome-repository
spec:
  public: true
  writers:
  - anotherteamA
  - anotherteamB
  readers:
  - anotherteamC
  - anotherteamD
` + "```" + `

In this last example:
- the repository is now publci
- other teams have write (` + "`" + `notherteamA` + "`" + `, ` + "`" + `anotherteamB` + "`" + `) or read (` + "`" + `anotherteamC` + "`" + `, ` + "`" + `anotherteamD` + "`" + `) access

### Create a new team

If you want to create a new team (like ` + "`" + `foobar` + "`" + `), you need to create a PR with a ` + "`" + `/teams/foobar/team.yaml` + "`" + ` file:

` + "```" + `
apiVersion: v1
kind: Team
name: foobar
spec:
  owners:
    - user1
    - user2
  members:
    - user3
    - user4
` + "```" + `

The users defined there are in 2 different categories
- members: are part of the team (and will be writer on all repositories of the team)
- owners: are part of the team (and will be writer on all repositories of the team) AMD can approve PR in the ` + "`" + `foobar` + "`" + ` teams repository (when you want to change a team definition, or when you want to create/update a repository definition)

The users name used are the one defined in the ` + "`" + `/users` + "`" + ` sub directories (like ` + "`" + `alice` + "`" + `)

### Archive a repository

You can archive a repository, by a PR that move the yaml repository file into the ` + "`" + `/archived` + "`" + ` directory

`
	if err := writeFile(path.Join(rootpath, "README.md"), []byte(readme), fs); err != nil {
		return err
	}
	return nil
}

// helper function to write a yaml file (with 2 spaces indentation)
func writeYamlFile(filename string, in interface{}, fs afero.Fs) error {
	file, err := fs.Create(filename)
	if err != nil {
		return fmt.Errorf("Not able to create file %s: %v", filename, err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	err = encoder.Encode(in)
	if err != nil {
		return fmt.Errorf("Not able to write to file %s: %v", filename, err)
	}
	return nil
}

// helper function to write a file
func writeFile(filename string, content []byte, fs afero.Fs) error {
	file, err := fs.Create(filename)
	if err != nil {
		return fmt.Errorf("Not able to create file %s: %v", filename, err)
	}
	defer file.Close()

	_, err = file.Write(content)
	if err != nil {
		return fmt.Errorf("Not able to write to file %s: %v", filename, err)
	}
	return nil
}
