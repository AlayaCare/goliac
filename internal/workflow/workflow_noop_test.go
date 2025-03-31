package workflow

import (
	"context"
	"testing"

	"github.com/goliac-project/goliac/internal/engine"
	"github.com/goliac-project/goliac/internal/entity"
	"github.com/stretchr/testify/assert"
)

func fixtureNoopWorkflow() *entity.Workflow {
	w := &entity.Workflow{}
	w.Name = "nooptest"
	w.ApiVersion = "v1"
	w.Kind = "Workflow"

	w.Spec.Steps = []struct {
		Name       string                 `yaml:"name"`
		Properties map[string]interface{} `yaml:"properties"`
	}{
		{
			Name: "jira_ticket_creation",
			Properties: map[string]interface{}{
				"project_key": "SRE",
				"issue_type":  "Bug",
			},
		},
	}
	w.Spec.Description = "noop test"
	w.Spec.WorkflowType = "noop"

	w.Spec.Acls = struct {
		Allowed []string `yaml:"allowed"`
		Except  []string `yaml:"except"`
	}{
		Allowed: []string{"test-team"},
		Except:  []string{},
	}

	return w
}

func TestNoop(t *testing.T) {
	t.Run("happy path: local team", func(t *testing.T) {
		lTeam := &entity.Team{}
		lTeam.Name = "test-team"
		lTeam.Spec.ExternallyManaged = false
		lTeam.Spec.Owners = []string{"test-user"}
		lTeam.Spec.Members = []string{}

		luser := entity.User{}
		luser.Name = "test-user"
		luser.Spec.GithubID = "test-user"

		local := &LocalResourceMock{
			MockWorkflows: map[string]*entity.Workflow{
				"nooptest": fixtureNoopWorkflow(),
			},
			LTeams: map[string]*entity.Team{
				"test-team": lTeam,
			},

			LUsers: map[string]*entity.User{
				"test-user": &luser,
			},
		}
		remote := &RemoteResourceMock{
			RTeams: map[string]*engine.GithubTeam{},
		}

		stepPlugins := map[string]StepPlugin{
			"jira_ticket_creation": NewStepPluginMock(),
		}

		noop := &NoopImpl{
			local:       local,
			remote:      remote,
			stepPlugins: stepPlugins,
		}

		resUrl, err := noop.ExecuteWorkflow(context.Background(), []string{"nooptest"}, "test-user", "nooptest", "explanation", map[string]string{}, false)

		assert.Nil(t, err)
		assert.Equal(t, 1, len(resUrl))
		assert.Equal(t, "mocked_url", resUrl[0])

	})
}
