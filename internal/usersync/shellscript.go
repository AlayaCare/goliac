package usersync

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/engine"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/Alayacare/goliac/internal/observability"
	"github.com/go-git/go-billy/v5"
)

type UserSyncPluginShellScript struct{}

func NewUserSyncPluginShellScript() engine.UserSyncPlugin {
	return &UserSyncPluginShellScript{}
}

func (p *UserSyncPluginShellScript) UpdateUsers(repoconfig *config.RepositoryConfig, fs billy.Filesystem, orguserdirrectorypath string, feedback observability.RemoteLoadFeedback) (map[string]*entity.User, error) {
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
