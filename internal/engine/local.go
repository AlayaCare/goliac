package engine

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/utils"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	goconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/gosimple/slug"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

/*
 * GoliacLocal
 * This interface is used to load the goliac organization from a local directory
 * and mount it in memory
 */
type GoliacLocal interface {
	GoliacLocalGit
	GoliacLocalResources
}

type GoliacLocalGit interface {
	Clone(accesstoken, repositoryUrl, branch string) error

	// Return commits from tagname to HEAD
	ListCommitsFromTag(tagname string) ([]*object.Commit, error)
	GetHeadCommit() (*object.Commit, error)
	CheckoutCommit(commit *object.Commit) error
	PushTag(tagname string, hash plumbing.Hash, accesstoken string) error

	LoadRepoConfig() (*config.RepositoryConfig, error)

	// Load and Validate from a github repository
	LoadAndValidate() ([]error, []entity.Warning)
	// whenever someone create/delete a team, we must update the github CODEOWNERS
	UpdateAndCommitCodeOwners(repoconfig *config.RepositoryConfig, dryrun bool, accesstoken string, branch string, tagname string) error
	// whenever repos are not deleted but archived
	ArchiveRepos(reposToArchiveList []string, accesstoken string, branch string, tagname string) error
	// whenever the users list is changing, reload users and teams, and commit them
	SyncUsersAndTeams(repoconfig *config.RepositoryConfig, plugin UserSyncPlugin, accesstoken string, dryrun bool, force bool) error
	Close()

	// Load and Validate from a local directory
	LoadAndValidateLocal(fs billy.Filesystem) ([]error, []entity.Warning)
}

type GoliacLocalResources interface {
	Teams() map[string]*entity.Team              // teamname, team definition
	Repositories() map[string]*entity.Repository // reponame, repo definition
	Users() map[string]*entity.User              // github username, user definition
	ExternalUsers() map[string]*entity.User
	RuleSets() map[string]*entity.RuleSet
}

type GoliacLocalImpl struct {
	teams         map[string]*entity.Team
	repositories  map[string]*entity.Repository
	users         map[string]*entity.User
	externalUsers map[string]*entity.User
	rulesets      map[string]*entity.RuleSet
	repo          *git.Repository
}

func NewGoliacLocalImpl() GoliacLocal {
	return &GoliacLocalImpl{
		teams:         map[string]*entity.Team{},
		repositories:  map[string]*entity.Repository{},
		users:         map[string]*entity.User{},
		externalUsers: map[string]*entity.User{},
		rulesets:      map[string]*entity.RuleSet{},
		repo:          nil,
	}
}

func (g *GoliacLocalImpl) Teams() map[string]*entity.Team {
	return g.teams
}

func (g *GoliacLocalImpl) Repositories() map[string]*entity.Repository {
	return g.repositories
}

func (g *GoliacLocalImpl) Users() map[string]*entity.User {
	return g.users
}

func (g *GoliacLocalImpl) ExternalUsers() map[string]*entity.User {
	return g.externalUsers
}

func (g *GoliacLocalImpl) RuleSets() map[string]*entity.RuleSet {
	return g.rulesets
}

func (g *GoliacLocalImpl) Clone(accesstoken, repositoryUrl, branch string) error {
	if g.repo != nil {
		g.Close()
	}
	// create a temp directory
	tmpDir, err := os.MkdirTemp("", "goliac")
	if err != nil {
		return err
	}

	var auth transport.AuthMethod
	if strings.HasPrefix(repositoryUrl, "https://") {
		auth = &http.BasicAuth{
			Username: "x-access-token", // This can be anything except an empty string
			Password: accesstoken,
		}
	} else {
		// ssh clone not supported yet
		return fmt.Errorf("not supported")
	}
	repo, err := git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL:  repositoryUrl,
		Auth: auth,
	})
	if err != nil {
		return err
	}
	g.repo = repo

	// checkout the branch
	w, err := g.repo.Worktree()
	if err != nil {
		return err
	}
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
	})
	if err != nil {
		return err
	}

	return err
}

func (g *GoliacLocalImpl) PushTag(tagname string, hash plumbing.Hash, accesstoken string) error {
	// Create or move the tag to the commit
	tagRefName := plumbing.ReferenceName("refs/tags/" + tagname)
	tagRef := plumbing.NewHashReference(tagRefName, hash)
	if err := g.repo.Storer.SetReference(tagRef); err != nil {
		return err
	}

	// Now push the tag to the remote repository
	auth := &http.BasicAuth{
		Username: "x-access-token", // This can be anything except an empty string
		Password: accesstoken,
	}

	// Force push with '+refs/tags/your_tag_name_here:refs/tags/your_tag_name_here'
	pushRefSpec := fmt.Sprintf("+%s:%s", tagRefName, tagRefName)
	err := g.repo.Push(&git.PushOptions{
		RefSpecs: []goconfig.RefSpec{goconfig.RefSpec(pushRefSpec)},
		Auth:     auth,
	})

	if err != nil && err.Error() == "already up-to-date" {
		return nil
	}

	return err
}

func (g *GoliacLocalImpl) CheckoutCommit(commit *object.Commit) error {
	// checkout the branch
	w, err := g.repo.Worktree()
	if err != nil {
		return err
	}
	err = w.Checkout(&git.CheckoutOptions{
		Hash: commit.Hash,
	})
	if err != nil {
		return err
	}
	return nil
}

func (g *GoliacLocalImpl) GetHeadCommit() (*object.Commit, error) {
	// Get reference to the HEAD
	refHead, err := g.repo.Head()
	if err != nil {
		return nil, err
	}

	headCommit, err := g.repo.CommitObject(refHead.Hash())
	if err != nil {
		return nil, err
	}
	return headCommit, nil
}

func (g *GoliacLocalImpl) ListCommitsFromTag(tagname string) ([]*object.Commit, error) {
	if g.repo == nil {
		return nil, fmt.Errorf("git repository not cloned")
	}

	commits := make([]*object.Commit, 0)

	// Get reference to the HEAD
	refHead, err := g.repo.Head()
	if err != nil {
		return nil, err
	}

	headCommit, err := g.repo.CommitObject(refHead.Hash())
	if err != nil {
		return nil, err
	}

	// Get reference to the specific tag
	refTag, err := g.repo.Tag(tagname)
	if err != nil {
		// we can't? stop it and returns the head
		return []*object.Commit{headCommit}, nil
	}

	// Get the commits between HEAD and the specific tag
	commitLog, err := g.repo.Log(&git.LogOptions{
		From:  refHead.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, err
	}

	err = commitLog.ForEach(func(c *object.Commit) error {
		if c.Hash == refTag.Hash() {
			return errors.New("stop iteration") // This is used to stop the iteration
		}
		commits = append(commits, c)

		return nil
	})
	if err != nil && err.Error() != "stop iteration" {
		return commits, err
	}

	// let's invert the order of the commits (from tag to HEAD)
	length := len(commits)
	for i := 0; i < length/2; i++ {
		commits[i], commits[length-1-i] = commits[length-1-i], commits[i]
	}

	return commits, nil
}

func (g *GoliacLocalImpl) Close() {
	if g.repo != nil {
		w, err := g.repo.Worktree()
		if err == nil {
			os.RemoveAll(w.Filesystem.Root())
		}
	}
	g.repo = nil
}

func (g *GoliacLocalImpl) LoadRepoConfig() (*config.RepositoryConfig, error) {
	if g.repo == nil {
		return nil, fmt.Errorf("git repository not cloned")
	}
	w, err := g.repo.Worktree()
	if err != nil {
		return nil, err
	}

	var repoconfig config.RepositoryConfig

	content, err := utils.ReadFile(w.Filesystem, "goliac.yaml")
	if err != nil {
		return nil, fmt.Errorf("not able to find the /goliac.yaml configuration file: %v", err)
	}
	err = yaml.Unmarshal(content, &repoconfig)
	if err != nil {
		return nil, fmt.Errorf("not able to unmarshall the /goliac.yaml configuration file: %v", err)
	}

	return &repoconfig, nil
}

func (g *GoliacLocalImpl) codeowners_regenerate(adminteam string) string {
	codeowners := "# DO NOT MODIFY THIS FILE MANUALLY\n"
	codeowners += fmt.Sprintf("* @%s/%s\n", config.Config.GithubAppOrganization, slug.Make(adminteam))

	teamsnames := make([]string, 0)
	for _, t := range g.teams {
		teamsnames = append(teamsnames, t.Name)
	}
	sort.Strings(teamsnames)

	for _, t := range teamsnames {
		codeowners += fmt.Sprintf("/teams/%s/* @%s/%s-owners @%s/%s\n", t, config.Config.GithubAppOrganization, slug.Make(t), config.Config.GithubAppOrganization, slug.Make(adminteam))
	}

	return codeowners
}

func (g *GoliacLocalImpl) ArchiveRepos(reposToArchiveList []string, accesstoken string, branch string, tagname string) error {
	if g.repo == nil {
		return fmt.Errorf("git repository not cloned")
	}

	// Get the HEAD reference
	headRef, err := g.repo.Head()
	if err != nil {
		return err
	}

	if headRef.Name() != plumbing.NewBranchReferenceName(branch) {
		// If not on main, check out the main branch
		worktree, err := g.repo.Worktree()
		if err != nil {
			return err
		}

		err = worktree.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(branch),
			Create: false,
			Force:  true,
		})
		if err != nil {
			return err
		}
	}

	w, err := g.repo.Worktree()
	if err != nil {
		return err
	}

	err = w.Filesystem.MkdirAll("archived", 0755)
	if err != nil {
		return err
	}

	for _, reponame := range reposToArchiveList {
		repo := entity.Repository{}
		repo.ApiVersion = "v1"
		repo.Kind = "Repository"
		repo.Name = reponame

		filename := filepath.Join("archived", reponame+".yaml")
		file, err := w.Filesystem.Create(filename)
		if err != nil {
			return fmt.Errorf("not able to create file %s: %v", filename, err)
		}
		defer file.Close()

		encoder := yaml.NewEncoder(file)
		encoder.SetIndent(2)
		err = encoder.Encode(&repo)
		if err != nil {
			return fmt.Errorf("not able to write to file %s: %v", filename, err)
		}

		_, err = w.Add(filename)
		if err != nil {
			return err
		}

		// last, if the repository was not present in Goliac (it was removed from Goliac
		// but still present in Github, and we want to archive it), we must add it back
		// to the list of repositories we manage
		if _, ok := g.repositories[reponame]; !ok {
			g.repositories[reponame] = &repo
		}
	}

	_, err = w.Commit("moving deleted repositories as archived", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Goliac",
			Email: config.Config.GoliacEmail,
			When:  time.Now(),
		},
	})

	if err != nil {
		return err
	}

	err = g.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth: &http.BasicAuth{
			Username: "x-access-token", // This can be anything except an empty string
			Password: accesstoken,
		},
	})

	if err != nil {
		return fmt.Errorf("error pushing to remote: %v", err)
	}

	// push the tagname
	return g.PushTag(tagname, headRef.Hash(), accesstoken)
}

/*
 * UpdateAndCommitCodeOwners will collects all teams definition to update the .github/CODEOWNERS file
 * cf https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners
 */
func (g *GoliacLocalImpl) UpdateAndCommitCodeOwners(repoconfig *config.RepositoryConfig, dryrun bool, accesstoken string, branch string, tagname string) error {
	if g.repo == nil {
		return fmt.Errorf("git repository not cloned")
	}
	w, err := g.repo.Worktree()
	if err != nil {
		return err
	}

	err = w.Filesystem.MkdirAll(".github", 0755)
	if err != nil {
		return err
	}

	codeownerpath := filepath.Join(".github", "CODEOWNERS")
	var content []byte

	info, err := w.Filesystem.Stat(codeownerpath)
	if err == nil && !info.IsDir() {
		file, err := w.Filesystem.Open(codeownerpath)
		if err != nil {
			return fmt.Errorf("not able to open .github/CODEOWNERS file: %v", err)
		}
		defer file.Close()

		content, err = io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("not able to open .github/CODEOWNERS file: %v", err)
		}
	} else {
		content = []byte("")
	}

	newContent := g.codeowners_regenerate(repoconfig.AdminTeam)

	if string(content) != newContent {
		logrus.Info(".github/CODEOWNERS needs to be regenerated")
		if dryrun {
			return nil
		}

		// Get the HEAD reference
		headRef, err := g.repo.Head()
		if err != nil {
			return err
		}

		if headRef.Name() != plumbing.NewBranchReferenceName(branch) {
			// If not on main, check out the main branch
			worktree, err := g.repo.Worktree()
			if err != nil {
				return err
			}

			err = worktree.Checkout(&git.CheckoutOptions{
				Branch: plumbing.NewBranchReferenceName(branch),
				Create: false,
				Force:  true,
			})
			if err != nil {
				return err
			}
		}

		err = utils.WriteFile(w.Filesystem, codeownerpath, []byte(newContent), 0644)
		if err != nil {
			return err
		}

		_, err = w.Add(codeownerpath)
		if err != nil {
			return err
		}

		_, err = w.Commit("update CODEOWNERS", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Goliac",
				Email: config.Config.GoliacEmail,
				When:  time.Now(),
			},
		})

		if err != nil {
			return err
		}

		err = g.repo.Push(&git.PushOptions{
			RemoteName: "origin",
			Auth: &http.BasicAuth{
				Username: "x-access-token", // This can be anything except an empty string
				Password: accesstoken,
			},
		})

		if err != nil {
			return fmt.Errorf("error pushing to remote: %v", err)
		}
	}

	// push the tagname
	if !dryrun {
		// Get the HEAD reference
		headRef, err := g.repo.Head()
		if err != nil {
			return err
		}

		return g.PushTag(tagname, headRef.Hash(), accesstoken)
	}

	return nil
}

/**
 * syncusers will
 * - list the current users list
 * - call the external user sync plugin
 * - collect the difference
 * - returns deleted users, and add/updated users
 */
func syncUsersViaUserPlugin(repoconfig *config.RepositoryConfig, fs billy.Filesystem, userplugin UserSyncPlugin) ([]string, []string, error) {
	usersOrgPath := filepath.Join("users", "org")
	orgUsers, errs, _ := entity.ReadUserDirectory(fs, usersOrgPath)
	if len(errs) > 0 {
		return nil, nil, fmt.Errorf("cannot load org users (for example: %v)", errs[0])
	}

	// use usersync to update the users
	newOrgUsers, err := userplugin.UpdateUsers(repoconfig, fs, usersOrgPath)
	if err != nil {
		return nil, nil, err
	}

	// write back to disk
	deletedusers := []string{}
	updatedusers := []string{}
	for username, user := range orgUsers {
		if newuser, ok := newOrgUsers[username]; !ok {
			// deleted user
			deletedusers = append(deletedusers, filepath.Join(usersOrgPath, fmt.Sprintf("%s.yaml", username)))
			fs.Remove(filepath.Join(usersOrgPath, fmt.Sprintf("%s.yaml", username)))
		} else {
			// check if user changed
			if !newuser.Equals(user) {
				// changed user
				file, err := fs.Create(filepath.Join(usersOrgPath, fmt.Sprintf("%s.yaml", username)))
				if err != nil {
					return nil, nil, err
				}
				defer file.Close()

				encoder := yaml.NewEncoder(file)
				encoder.SetIndent(2)
				err = encoder.Encode(newuser)
				if err != nil {
					return nil, nil, err
				}
				updatedusers = append(updatedusers, filepath.Join(usersOrgPath, fmt.Sprintf("%s.yaml", username)))
			}

			delete(newOrgUsers, username)
		}
	}
	for username, user := range newOrgUsers {
		// new user
		file, err := fs.Create(filepath.Join(usersOrgPath, fmt.Sprintf("%s.yaml", username)))
		if err != nil {
			return nil, nil, err
		}
		defer file.Close()

		encoder := yaml.NewEncoder(file)
		encoder.SetIndent(2)
		err = encoder.Encode(user)
		if err != nil {
			return nil, nil, err
		}
		updatedusers = append(updatedusers, filepath.Join(usersOrgPath, fmt.Sprintf("%s.yaml", username)))
	}
	return deletedusers, updatedusers, nil
}

func (g *GoliacLocalImpl) SyncUsersAndTeams(repoconfig *config.RepositoryConfig, userplugin UserSyncPlugin, accesstoken string, dryrun bool, force bool) error {
	if g.repo == nil {
		return fmt.Errorf("git repository not cloned")
	}
	w, err := g.repo.Worktree()
	if err != nil {
		return err
	}

	// read the organization files
	rootDir := w.Filesystem.Root()

	//
	// let's update org users
	//

	// Parse all the users in the <orgDirectory>/org-users directory
	deletedusers, addedusers, err := syncUsersViaUserPlugin(repoconfig, w.Filesystem, userplugin)
	if err != nil {
		return err
	}

	//
	// let's update teams
	//

	errors, _ := g.loadUsers(w.Filesystem)
	if len(errors) > 0 {
		return fmt.Errorf("cannot read users (for example: %v)", errors[0])
	}

	teamschanged, err := entity.ReadAndAdjustTeamDirectory(w.Filesystem, filepath.Join(rootDir, "teams"), g.users)
	if err != nil {
		return err
	}

	// check if we have too many changesets
	if !force && len(teamschanged)+len(deletedusers)+len(addedusers) > repoconfig.MaxChangesets {
		return fmt.Errorf("too many changesets (%d) to commit. Please increase max_changesets in goliac.yaml", len(teamschanged)+len(deletedusers)+len(addedusers))
	}

	//
	// let's commit
	//
	if len(teamschanged) > 0 || len(deletedusers) > 0 || len(addedusers) > 0 {

		logrus.Info("some users and/or teams must be commited")

		for _, u := range deletedusers {
			logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "author": "goliac", "command": "remove_user_from_repository"}).Infof("user: %s", u)
			if !dryrun {
				_, err = w.Remove(u)
				if err != nil {
					return err
				}
			}
		}

		for _, u := range addedusers {
			logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "author": "goliac", "command": "add_user_to_repository"}).Infof("user: %s", u)
			if !dryrun {
				_, err = w.Add(u)
				if err != nil {
					return err
				}
			}
		}

		for _, t := range teamschanged {
			logrus.WithFields(map[string]interface{}{"dryrun": dryrun, "author": "goliac", "command": "update_team_to_repository"}).Infof("team: %s", t)
			if !dryrun {
				_, err = w.Add(t)
				if err != nil {
					return err
				}
			}
		}

		if dryrun {
			return nil
		}

		_, err = w.Commit("update teams and users", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Goliac",
				Email: config.Config.GoliacEmail,
				When:  time.Now(),
			},
		})

		if err != nil {
			return err
		}

		// Now push the tag to the remote repository
		auth := &http.BasicAuth{
			Username: "x-access-token", // This can be anything except an empty string
			Password: accesstoken,
		}

		err = g.repo.Push(&git.PushOptions{
			RemoteName: "origin",
			Auth:       auth,
		})

		return err
	}
	return nil
}

/*
 * Load the goliac organization from Github (after the repository has been cloned)
 * - read the organization files
 * - validate the organization
 */
func (g *GoliacLocalImpl) LoadAndValidate() ([]error, []entity.Warning) {
	if g.repo == nil {
		return []error{fmt.Errorf("the repository has not been cloned. Did you called .Clone()?")}, []entity.Warning{}
	}

	// read the organization files

	w, err := g.repo.Worktree()
	if err != nil {
		return []error{err}, []entity.Warning{}
	}
	errs, warns := g.LoadAndValidateLocal(w.Filesystem)

	return errs, warns
}

func (g *GoliacLocalImpl) loadUsers(fs billy.Filesystem) ([]error, []entity.Warning) {
	errors := []error{}
	warnings := []entity.Warning{}

	// Parse all the users in the <orgDirectory>/protected-users directory
	protectedUsers, errs, warns := entity.ReadUserDirectory(fs, filepath.Join("users", "protected"))
	errors = append(errors, errs...)
	warnings = append(warnings, warns...)
	g.users = protectedUsers

	// Parse all the users in the <orgDirectory>/org-users directory
	orgUsers, errs, warns := entity.ReadUserDirectory(fs, filepath.Join("users", "org"))
	errors = append(errors, errs...)
	warnings = append(warnings, warns...)

	// not users? not good
	if orgUsers == nil {
		return errors, warnings
	}

	for k, v := range orgUsers {
		g.users[k] = v
	}

	// Parse all the users in the <orgDirectory>/external-users directory
	externalUsers, errs, warns := entity.ReadUserDirectory(fs, filepath.Join("users", "external"))
	errors = append(errors, errs...)
	warnings = append(warnings, warns...)
	g.externalUsers = externalUsers

	rulesets, errs, warns := entity.ReadRuleSetDirectory(fs, filepath.Join("rulesets"))
	errors = append(errors, errs...)
	warnings = append(warnings, warns...)
	g.rulesets = rulesets

	return errors, warnings
}

/**
 * readOrganization reads all the organization files and returns
 * - a slice of errors that must stop the vlidation process
 * - a slice of warning that must not stop the validation process
 */
func (g *GoliacLocalImpl) LoadAndValidateLocal(fs billy.Filesystem) ([]error, []entity.Warning) {
	errors, warnings := g.loadUsers(fs)

	if len(errors) > 0 {
		return errors, warnings
	}

	// Parse all the teams in the <orgDirectory>/teams directory
	teams, errs, warns := entity.ReadTeamDirectory(fs, "teams", g.users)
	errors = append(errors, errs...)
	warnings = append(warnings, warns...)
	g.teams = teams

	// Parse all repositories in the <orgDirectory>/teams/<teamname> directories
	repos, errs, warns := entity.ReadRepositories(fs, "archived", "teams", g.teams, g.externalUsers)
	errors = append(errors, errs...)
	warnings = append(warnings, warns...)
	g.repositories = repos

	rulesets, errs, warns := entity.ReadRuleSetDirectory(fs, "rulesets")
	errors = append(errors, errs...)
	warnings = append(warnings, warns...)
	g.rulesets = rulesets

	logrus.Debugf("Nb local users: %d", len(g.users))
	logrus.Debugf("Nb local external users: %d", len(g.externalUsers))
	logrus.Debugf("Nb local teams: %d", len(g.teams))
	logrus.Debugf("Nb local repositories: %d", len(g.repositories))

	return errors, warnings
}
