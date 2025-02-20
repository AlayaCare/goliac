package usersync

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/goliac-project/goliac/internal/config"
	"github.com/goliac-project/goliac/internal/engine"
	"github.com/goliac-project/goliac/internal/entity"
	"github.com/goliac-project/goliac/internal/observability"
)

type UserSyncPluginShellScript struct{}

func NewUserSyncPluginShellScript() engine.UserSyncPlugin {
	return &UserSyncPluginShellScript{}
}

func (p *UserSyncPluginShellScript) UpdateUsers(repoconfig *config.RepositoryConfig, fs billy.Filesystem, orguserdirrectorypath string, feedback observability.RemoteObservability) (map[string]*entity.User, error) {
	cmd := exec.Command(repoconfig.UserSync.Path, filepath.Join(fs.Root(), orguserdirrectorypath))
	_, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	users, errs, _ := entity.ReadUserDirectory(fs, orguserdirrectorypath)
	if len(errs) > 0 {
		return nil, fmt.Errorf("cannot load org users (for example: %v)", errs[0])
	}

	return users, nil
}
