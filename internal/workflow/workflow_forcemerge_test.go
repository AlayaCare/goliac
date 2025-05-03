package workflow

import (
	"context"
	"testing"

	"github.com/goliac-project/goliac/internal/engine"
	"github.com/goliac-project/goliac/internal/entity"
	"github.com/stretchr/testify/assert"
)

func fixtureForcemergeWorkflow() *entity.Workflow {
	w := &entity.Workflow{}
	w.Name = "fmtest"
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
	w.Spec.Description = "fmtest"
	w.Spec.WorkflowType = "forcemerge"
	w.Spec.Repositories = struct {
		Allowed []string `yaml:"allowed"`
		Except  []string `yaml:"except"`
	}{
		Allowed: []string{"goliac"},
	}

	// rfcWorkflow.Spec.Acls = struct {
	// 	Allowed []string "yaml:\"allowed\""
	// 	Except  []string "yaml:\"except\""
	// }{
	// 	Allowed: []string{"test-team"},
	// }

	return w
}

func TestForcemerge(t *testing.T) {
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
				"fmtest": fixtureForcemergeWorkflow(),
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

		ws := NewWorkflowService("myorg", local, remote, &GithubClientMock{})

		fc := &ForcemergeImpl{
			ws:          ws,
			stepPlugins: stepPlugins,
		}

		resUrl, err := fc.ExecuteWorkflow(context.Background(), []string{"fmtest"}, "test-user", "fmtest", "explanation", map[string]string{"pr_url": "https://github.com/goliac-project/goliac/pull/32"}, false)

		assert.Nil(t, err)
		assert.Equal(t, 1, len(resUrl))
		assert.Equal(t, "mocked_url", resUrl[0])

	})
	t.Run("happy path: remote team", func(t *testing.T) {
		lTeam := &entity.Team{}
		lTeam.Name = "test-team"
		lTeam.Spec.ExternallyManaged = true
		lTeam.Spec.Owners = []string{"test-user"}
		lTeam.Spec.Members = []string{}

		luser := entity.User{}
		luser.Name = "test-user"
		luser.Spec.GithubID = "test-user"

		local := &LocalResourceMock{
			MockWorkflows: map[string]*entity.Workflow{
				"fmtest": fixtureForcemergeWorkflow(),
			},
			LTeams: map[string]*entity.Team{
				"test-team": lTeam,
			},
			LUsers: map[string]*entity.User{
				"test-user": &luser,
			},
		}
		remote := &RemoteResourceMock{
			RTeams: map[string]*engine.GithubTeam{
				"test-team": {
					Name:    "test-team",
					Slug:    "test-team",
					Members: []string{"test-user"},
				},
			},
		}

		stepPlugins := map[string]StepPlugin{
			"jira_ticket_creation": NewStepPluginMock(),
		}

		ws := NewWorkflowService("myorg", local, remote, &GithubClientMock{})

		fc := &ForcemergeImpl{
			ws:          ws,
			stepPlugins: stepPlugins,
		}

		resUrl, err := fc.ExecuteWorkflow(context.Background(), []string{"fmtest"}, "test-user", "fmtest", "explanation", map[string]string{"pr_url": "https://github.com/goliac-project/goliac/pull/32"}, false)

		assert.Nil(t, err)
		assert.Equal(t, 1, len(resUrl))
		assert.Equal(t, "mocked_url", resUrl[0])

	})
}
