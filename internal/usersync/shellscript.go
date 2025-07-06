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

func (p *UserSyncPluginShellScript) UpdateUsers(repoconfig *config.RepositoryConfig, fs billy.Filesystem, orguserdirrectorypath string, feedback observability.RemoteObservability, logsCollector *observability.LogCollection) map[string]*entity.User {
	cmd := exec.Command(repoconfig.UserSync.Path, filepath.Join(fs.Root(), orguserdirrectorypath))
	_, err := cmd.CombinedOutput()
	if err != nil {
		logsCollector.AddError(fmt.Errorf("not able to run the shell script: %w", err))
		return nil
	}

	users := entity.ReadUserDirectory(fs, orguserdirrectorypath, logsCollector)
	return users
}
