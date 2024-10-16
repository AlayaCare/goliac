package engine

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/utils"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/client"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
)

func createBasicStructure(fs billy.Filesystem, path string) error {
	// Create a fake repository
	err := fs.MkdirAll(path, 0755)
	if err != nil {
		return err
	}

	err = utils.WriteFile(fs, filepath.Join(path, "goliac.yaml"), []byte(`
`), 0644)
	if err != nil {
		return err
	}
	// Create users
	err = fs.MkdirAll(filepath.Join(path, "users/org"), 0755)
	if err != nil {
		return err
	}

	err = utils.WriteFile(fs, filepath.Join(path, "users/org/user1.yaml"), []byte(`
apiVersion: v1
kind: User
name: user1
spec:
  githubID: github1
`), 0644)
	if err != nil {
		return err
	}

	err = utils.WriteFile(fs, filepath.Join(path, "users/org/user2.yaml"), []byte(`
apiVersion: v1
kind: User
name: user2
spec:
  githubID: github2
`), 0644)
	if err != nil {
		return err
	}

	// Create teams
	err = fs.MkdirAll(filepath.Join(path, "teams/team1"), 0755)
	if err != nil {
		return err
	}

	err = utils.WriteFile(fs, filepath.Join(path, "teams/team1/team.yaml"), []byte(`
apiVersion: v1
kind: Team
name: team1
spec:
  owners:
  - user1
  - user2
`), 0644)
	if err != nil {
		return err
	}

	// Create repositories
	err = utils.WriteFile(fs, filepath.Join(path, "projects/repo1.yaml"), []byte(`
apiVersion: v1
kind: Repository
name: repo1
`), 0644)
	if err != nil {
		return err
	}
	return nil
}

func TestRepository(t *testing.T) {

	// happy path
	t.Run("happy path: local directory", func(t *testing.T) {
		fs := memfs.New()
		createBasicStructure(fs, "/tmp/goliac")
		g := NewGoliacLocalImpl()
		errs, warns := g.LoadAndValidateLocal(fs, "/tmp/goliac")

		assert.Equal(t, 0, len(errs))
		assert.Equal(t, 0, len(warns))
	})

	t.Run("happy path: local repository", func(t *testing.T) {
		fs := memfs.New()
		storer := memory.NewStorage()

		// Initializes a new repository
		r, err := git.Init(storer, fs)
		assert.Nil(t, err)

		createBasicStructure(fs, "/")
		w, err := r.Worktree()
		assert.Nil(t, err)
		_, err = w.Add(".")
		assert.Nil(t, err)

		_, err = w.Commit("initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "goliac",
				Email: "goliac@alayacare.com",
				When:  time.Now(),
			},
		})
		assert.Nil(t, err)

		// Verify the commit
		_, err = r.Head()
		assert.Nil(t, err)

		g := &GoliacLocalImpl{
			teams:         map[string]*entity.Team{},
			repositories:  map[string]*entity.Repository{},
			users:         map[string]*entity.User{},
			externalUsers: map[string]*entity.User{},
			repo:          r,
		}

		errs, warns := g.LoadAndValidate()

		assert.Equal(t, 0, len(errs))
		assert.Equal(t, 0, len(warns))
	})
}

type ScrambleUserSync struct {
}

func (p *ScrambleUserSync) UpdateUsers(repoconfig *config.RepositoryConfig, orguserdirrectorypath string) (map[string]*entity.User, error) {
	users := make(map[string]*entity.User)

	// added
	foobar := &entity.User{}
	foobar.ApiVersion = "v1"
	foobar.Kind = "User"
	foobar.Name = "foobar"
	foobar.Spec.GithubID = "foobar"
	users["foobar"] = foobar

	// updated
	user1 := &entity.User{}
	user1.ApiVersion = "v1"
	user1.Kind = "User"
	user1.Name = "user1"
	user1.Spec.GithubID = "user1"
	users["user1"] = foobar

	return users, nil
}

type ErroreUserSync struct {
}

func (p *ErroreUserSync) UpdateUsers(repoconfig *config.RepositoryConfig, orguserdirrectorypath string) (map[string]*entity.User, error) {
	return nil, fmt.Errorf("unknown error")
}

type UserSyncPluginNoop struct {
	Fs billy.Filesystem
}

func NewUserSyncPluginNoop() UserSyncPlugin {
	return &UserSyncPluginNoop{
		Fs: osfs.New("/"),
	}
}

func (p *UserSyncPluginNoop) UpdateUsers(repoconfig *config.RepositoryConfig, orguserdirrectorypath string) (map[string]*entity.User, error) {
	users, errs, _ := entity.ReadUserDirectory(p.Fs, orguserdirrectorypath)
	if len(errs) > 0 {
		return nil, fmt.Errorf("cannot load org users (for example: %v)", errs[0])
	}

	return users, nil
}

func TestSyncUsersViaUserPlugin(t *testing.T) {

	t.Run("happy path: noop", func(t *testing.T) {
		fs := memfs.New()
		createBasicStructure(fs, "/tmp/goliac")

		removed, added, err := syncUsersViaUserPlugin(&config.RepositoryConfig{}, fs, &UserSyncPluginNoop{
			Fs: fs,
		}, "/tmp/goliac")

		assert.Nil(t, err)
		assert.Equal(t, 0, len(removed))
		assert.Equal(t, 0, len(added))
	})

	t.Run("happy path: replcae with foobar", func(t *testing.T) {
		fs := memfs.New()
		createBasicStructure(fs, "/tmp/goliac")

		removed, added, err := syncUsersViaUserPlugin(&config.RepositoryConfig{}, fs, &ScrambleUserSync{}, "/tmp/goliac")

		assert.Nil(t, err)
		assert.Equal(t, 1, len(removed))
		assert.Equal(t, 2, len(added))
		assert.Equal(t, "users/org/user1.yaml", added[0])
		assert.Equal(t, "users/org/foobar.yaml", added[1])
	})
	t.Run("not happy path: dealing with usersync error", func(t *testing.T) {
		fs := memfs.New()
		createBasicStructure(fs, "/tmp/goliac")

		_, _, err := syncUsersViaUserPlugin(&config.RepositoryConfig{}, fs, &ErroreUserSync{}, "/tmp/goliac")

		assert.NotNil(t, err)
	})
}

func createEmptyTeamRepo(src billy.Filesystem) (*git.Repository, error) {
	masterStorer := filesystem.NewStorage(src, cache.NewObjectLRUDefault())

	// Create a fake bare repository
	repo, err := git.Init(masterStorer, src)
	if err != nil {
		return nil, err
	}

	//
	// Create a new file in the working directory
	//

	// goliac.yaml
	err = util.WriteFile(src, "goliac.yaml", []byte(`
admin_team: github-admins

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
`), 0644)
	if err != nil {
		return nil, err
	}

	// Create users
	err = src.MkdirAll("users/org", 0755)
	if err != nil {
		return nil, err
	}
	err = util.WriteFile(src, "users/org/admin.yaml", []byte(`
apiVersion: v1
kind: User
name: admin
spec:
  githubID: admin
`), 0644)
	if err != nil {
		return nil, err
	}

	// Create teams
	err = src.MkdirAll("teams/github-admins", 0755)
	if err != nil {
		return nil, err
	}
	err = util.WriteFile(src, "teams/github-admins/team.yaml", []byte(`
apiVersion: v1
kind: Team
name: github-admins
spec:
  owners:
  - admin
`), 0644)
	if err != nil {
		return nil, err
	}

	// Create repositories
	err = util.WriteFile(src, "teams/github-admins/repo1.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo1
`), 0644)
	if err != nil {
		return nil, err
	}

	// commit
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	_, err = worktree.Add(".")
	if err != nil {
		return nil, err
	}
	hash, err := worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Goliac",
			Email: "goliac@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, err
	}

	// tag as v0.1.0
	_, err = repo.CreateTag("v0.1.0", hash, nil)
	if err != nil {
		return nil, err
	}

	// let's add a new commit after
	err = util.WriteFile(src, "teams/github-admins/repo2.yaml", []byte(`
apiVersion: v1
kind: Repository
name: repo2
`), 0644)
	if err != nil {
		return nil, err
	}

	// commit
	worktree, err = repo.Worktree()
	if err != nil {
		return nil, err
	}
	_, err = worktree.Add(".")
	if err != nil {
		return nil, err
	}
	_, err = worktree.Commit("add another repo", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Goliac",
			Email: "goliac@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func helperCreateAndClone(root billy.Filesystem, src billy.Filesystem, target billy.Filesystem) (*git.Repository, *git.Repository, error) {
	repo, err := createEmptyTeamRepo(src)
	if err != nil {
		return nil, nil, err
	}

	//
	// trying to clone it
	//

	loader := server.NewFilesystemLoader(root)
	client.InstallProtocol("inmemory", server.NewClient(loader))

	dotGit, err := target.Chroot(".git")
	if err != nil {
		return nil, nil, err
	}
	storer := filesystem.NewStorage(dotGit, cache.NewObjectLRUDefault())

	clonedRepo, err := git.Clone(storer, target, &git.CloneOptions{
		URL:      "inmemory:///src",
		Progress: nil,
	})
	if err != nil {
		return nil, nil, err
	}
	return repo, clonedRepo, nil
}

func TestPushTag(t *testing.T) {
	t.Run("push a tag into an upstream git repository", func(t *testing.T) {
		rootfs := memfs.New()
		src, _ := rootfs.Chroot("/src")
		target, _ := src.Chroot("/target")

		repo, clonedRepo, err := helperCreateAndClone(rootfs, src, target)
		assert.Nil(t, err)
		assert.NotNil(t, repo)
		assert.NotNil(t, clonedRepo)

		//
		// push tag
		//
		g := GoliacLocalImpl{
			teams:         map[string]*entity.Team{},
			repositories:  map[string]*entity.Repository{},
			users:         map[string]*entity.User{},
			externalUsers: map[string]*entity.User{},
			rulesets:      map[string]*entity.RuleSet{},
			repo:          clonedRepo,
		}

		// create a commit
		utils.WriteFile(target, "test.txt", []byte(`test`), 0644)
		w, err := clonedRepo.Worktree()
		assert.Nil(t, err)
		_, err = w.Add(".")
		assert.Nil(t, err)

		hash, err := w.Commit("new commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Goliac",
				Email: config.Config.GoliacEmail,
				When:  time.Now(),
			},
		})
		assert.Nil(t, err)

		err = g.PushTag("v1.0.0", hash, "none")
		assert.Nil(t, err)

		// check the tag
		tag, err := clonedRepo.Tag("v1.0.0")
		assert.Nil(t, err)
		assert.NotNil(t, tag)

		// read tag on the master repository
		tag, err = repo.Tag("v1.0.0")
		assert.Nil(t, err)
		assert.NotNil(t, tag)
	})
}

func TestBasicGitops(t *testing.T) {
	t.Run("clone", func(t *testing.T) {
		rootfs := memfs.New()
		src, _ := rootfs.Chroot("/src")
		target, _ := src.Chroot("/target")

		repo, clonedRepo, err := helperCreateAndClone(rootfs, src, target)
		assert.Nil(t, err)
		assert.NotNil(t, repo)
		assert.NotNil(t, clonedRepo)

	})

	t.Run("CheckoutCommit", func(t *testing.T) {
		rootfs := memfs.New()
		src, _ := rootfs.Chroot("/src")
		target, _ := src.Chroot("/target")

		repo, clonedRepo, err := helperCreateAndClone(rootfs, src, target)
		assert.Nil(t, err)
		assert.NotNil(t, repo)
		assert.NotNil(t, clonedRepo)

		// get commits
		g := GoliacLocalImpl{
			teams:         map[string]*entity.Team{},
			repositories:  map[string]*entity.Repository{},
			users:         map[string]*entity.User{},
			externalUsers: map[string]*entity.User{},
			rulesets:      map[string]*entity.RuleSet{},
			repo:          clonedRepo,
		}

		commits, err := g.ListCommitsFromTag("v0.1.0")
		assert.Nil(t, err)
		assert.Equal(t, 1, len(commits))

		//
		// checkout the commit
		//
		err = g.CheckoutCommit(commits[0])
		assert.Nil(t, err)

		// check the number of files in the 'teams/github-admins' directory
		files, err := target.ReadDir("teams/github-admins")
		assert.Nil(t, err)
		assert.Equal(t, 3, len(files))

		//
		// checkout v0.1.0
		//
		tagRef, err := clonedRepo.Tag("v0.1.0")
		assert.Nil(t, err)
		tagObject, err := repo.TagObject(tagRef.Hash())
		var commit *object.Commit
		if err == plumbing.ErrObjectNotFound {
			// If the tag is lightweight, the reference points directly to the commit
			commit1, err := repo.CommitObject(tagRef.Hash())
			assert.Nil(t, err)
			fmt.Println("commit1", commit1)
			commit = commit1
		} else {
			// If the tag is annotated, resolve the commit it points to
			commit2, err := tagObject.Commit()
			assert.Nil(t, err)
			fmt.Println("commit2", commit2)
			commit = commit2
		}

		err = g.CheckoutCommit(commit)
		assert.Nil(t, err)
		files, err = target.ReadDir("teams/github-admins")
		assert.Nil(t, err)
		assert.Equal(t, 2, len(files))

		//
		// checkout again the latest commit
		//
		err = g.CheckoutCommit(commits[0])
		assert.Nil(t, err)

		// check the number of files in the 'teams/github-admins' directory
		files, err = target.ReadDir("teams/github-admins")
		assert.Nil(t, err)
		assert.Equal(t, 3, len(files))
	})

	t.Run("CheckoutCommit", func(t *testing.T) {
		rootfs := memfs.New()
		src, _ := rootfs.Chroot("/src")
		target, _ := src.Chroot("/target")

		repo, clonedRepo, err := helperCreateAndClone(rootfs, src, target)
		assert.Nil(t, err)
		assert.NotNil(t, repo)
		assert.NotNil(t, clonedRepo)

		// get commits
		g := GoliacLocalImpl{
			teams:         map[string]*entity.Team{},
			repositories:  map[string]*entity.Repository{},
			users:         map[string]*entity.User{},
			externalUsers: map[string]*entity.User{},
			rulesets:      map[string]*entity.RuleSet{},
			repo:          clonedRepo,
		}

		head, err := g.GetHeadCommit()
		assert.Nil(t, err)

		err = g.CheckoutCommit(head)
		assert.Nil(t, err)
		files, err := target.ReadDir("teams/github-admins")
		assert.Nil(t, err)
		assert.Equal(t, 3, len(files)) // it should be 3 because we have 3 files in the 'teams/github-admins' directory
	})

	t.Run("LoadRepoConfig", func(t *testing.T) {
		rootfs := memfs.New()
		src, _ := rootfs.Chroot("/src")
		target, _ := src.Chroot("/target")

		repo, clonedRepo, err := helperCreateAndClone(rootfs, src, target)
		assert.Nil(t, err)
		assert.NotNil(t, repo)
		assert.NotNil(t, clonedRepo)

		// get commits
		g := GoliacLocalImpl{
			teams:         map[string]*entity.Team{},
			repositories:  map[string]*entity.Repository{},
			users:         map[string]*entity.User{},
			externalUsers: map[string]*entity.User{},
			rulesets:      map[string]*entity.RuleSet{},
			repo:          clonedRepo,
		}

		files, err := target.ReadDir("/")
		assert.Nil(t, err)
		for _, file := range files {
			fmt.Println(file.Name())
		}

		err, goliacConfig := g.LoadRepoConfig()
		assert.Nil(t, err)
		assert.NotNil(t, goliacConfig)
		assert.Equal(t, "github-admins", goliacConfig.AdminTeam)
		assert.Equal(t, 50, goliacConfig.MaxChangesets)
		assert.Equal(t, true, goliacConfig.ArchiveOnDelete)
		assert.Equal(t, false, goliacConfig.DestructiveOperations.AllowDestructiveRepositories)
		assert.Equal(t, false, goliacConfig.DestructiveOperations.AllowDestructiveTeams)
		assert.Equal(t, false, goliacConfig.DestructiveOperations.AllowDestructiveUsers)
		assert.Equal(t, false, goliacConfig.DestructiveOperations.AllowDestructiveRulesets)
		assert.Equal(t, "noop", goliacConfig.UserSync.Plugin)
	})

	t.Run("codeowners_regenerate", func(t *testing.T) {
		rootfs := memfs.New()
		src, _ := rootfs.Chroot("/src")
		target, _ := src.Chroot("/target")

		repo, clonedRepo, err := helperCreateAndClone(rootfs, src, target)
		assert.Nil(t, err)
		assert.NotNil(t, repo)
		assert.NotNil(t, clonedRepo)

		// get commits
		adminTeam := entity.Team{}
		adminTeam.ApiVersion = "v1"
		adminTeam.Kind = "Team"
		adminTeam.Name = "github-admins"
		adminTeam.Spec.Owners = []string{"admin"}

		g := GoliacLocalImpl{
			teams: map[string]*entity.Team{
				"github-admins": &adminTeam,
			},
			repositories:  map[string]*entity.Repository{},
			users:         map[string]*entity.User{},
			externalUsers: map[string]*entity.User{},
			rulesets:      map[string]*entity.RuleSet{},
			repo:          clonedRepo,
		}

		content := g.codeowners_regenerate("github-admins")

		// check the content of the CODEOWNERS file
		assert.Equal(t, "# DO NOT MODIFY THIS FILE MANUALLY\n* @/github-admins\n/teams/github-admins/* @/github-admins-owners @/github-admins\n", content)
	})

	t.Run("ArchiveRepos", func(t *testing.T) {
		rootfs := memfs.New()
		src, _ := rootfs.Chroot("/src")
		target, _ := src.Chroot("/target")

		repo, clonedRepo, err := helperCreateAndClone(rootfs, src, target)
		assert.Nil(t, err)
		assert.NotNil(t, repo)
		assert.NotNil(t, clonedRepo)

		// get commits
		g := GoliacLocalImpl{
			teams:         map[string]*entity.Team{},
			repositories:  map[string]*entity.Repository{},
			users:         map[string]*entity.User{},
			externalUsers: map[string]*entity.User{},
			rulesets:      map[string]*entity.RuleSet{},
			repo:          clonedRepo,
		}

		// archive the repository 'repo1'
		err = g.ArchiveRepos([]string{"repo1"}, "none", "main", "foobar")
		assert.Nil(t, err)

		// check the content of the 'archived/repo1.yaml' file
		content, err := utils.ReadFile(target, "archived/repo1.yaml")
		assert.Nil(t, err)
		assert.Equal(t, "apiVersion: v1\nkind: Repository\nname: repo1\n", string(content))
	})

	t.Run("UpdateAndCommitCodeOwners", func(t *testing.T) {
		rootfs := memfs.New()
		src, _ := rootfs.Chroot("/src")
		target, _ := src.Chroot("/target")

		repo, clonedRepo, err := helperCreateAndClone(rootfs, src, target)
		assert.Nil(t, err)
		assert.NotNil(t, repo)
		assert.NotNil(t, clonedRepo)

		// get commits
		adminTeam := entity.Team{}
		adminTeam.ApiVersion = "v1"
		adminTeam.Kind = "Team"
		adminTeam.Name = "github-admins"
		adminTeam.Spec.Owners = []string{"admin"}

		g := GoliacLocalImpl{
			teams: map[string]*entity.Team{
				"github-admins": &adminTeam,
			},
			repositories:  map[string]*entity.Repository{},
			users:         map[string]*entity.User{},
			externalUsers: map[string]*entity.User{},
			rulesets:      map[string]*entity.RuleSet{},
			repo:          clonedRepo,
		}

		err, goliacConfig := g.LoadRepoConfig()
		assert.Nil(t, err)
		assert.NotNil(t, goliacConfig)

		// update and commit the CODEOWNERS file
		err = g.UpdateAndCommitCodeOwners(goliacConfig, false, "none", "main", "foobar")
		assert.Nil(t, err)

		// check the content of the CODEOWNERS file
		content, err := utils.ReadFile(target, ".github/CODEOWNERS")
		assert.Nil(t, err)
		assert.Equal(t, "# DO NOT MODIFY THIS FILE MANUALLY\n* @/github-admins\n/teams/github-admins/* @/github-admins-owners @/github-admins\n", string(content))
	})
}
