package workflow

import (
	"context"
	"net/url"
	"testing"

	"github.com/goliac-project/goliac/internal/engine"
	"github.com/goliac-project/goliac/internal/entity"
	"github.com/stretchr/testify/assert"
)

// strip down version of Goliac Local
type LocalResourceMock struct {
	MockWorkflows map[string]*entity.Workflow
	LTeams        map[string]*entity.Team
}

func (m *LocalResourceMock) Workflows() map[string]*entity.Workflow {
	return m.MockWorkflows
}
func (m *LocalResourceMock) Teams() map[string]*entity.Team {
	return m.LTeams
}

type RemoteResourceMock struct {
	RTeams map[string]*engine.GithubTeam
}

func (m *RemoteResourceMock) Teams(ctx context.Context, current bool) map[string]*engine.GithubTeam {
	return m.RTeams
}

type GithubClientMock struct {
	LastCallEndpoint string
	LastCallBody     map[string]interface{}
}

func (m *GithubClientMock) CallRestAPI(ctx context.Context, endpoint, parameters, method string, body map[string]interface{}, githubToken *string) ([]byte, error) {
	m.LastCallEndpoint = endpoint
	m.LastCallBody = body
	return nil, nil
}

type StepPluginMock struct{}

func (f *StepPluginMock) Execute(ctx context.Context, username, explanation string, url *url.URL, properties map[string]interface{}) (string, error) {
	// Mock implementation
	return "mocked_url", nil
}

func NewStepPluginMock() StepPlugin {
	return &StepPluginMock{}
}

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

		local := &LocalResourceMock{
			MockWorkflows: map[string]*entity.Workflow{
				"fmtest": fixtureForcemergeWorkflow(),
			},
			LTeams: map[string]*entity.Team{
				"test-team": lTeam,
			},
		}
		remote := &RemoteResourceMock{
			RTeams: map[string]*engine.GithubTeam{},
		}

		githubclient := &GithubClientMock{}
		stepPlugins := map[string]StepPlugin{
			"jira_ticket_creation": NewStepPluginMock(),
		}

		fc := &ForcemergeImpl{
			local:        local,
			remote:       remote,
			githubclient: githubclient,
			stepPlugins:  stepPlugins,
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

		local := &LocalResourceMock{
			MockWorkflows: map[string]*entity.Workflow{
				"fmtest": fixtureForcemergeWorkflow(),
			},
			LTeams: map[string]*entity.Team{
				"test-team": lTeam,
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

		githubclient := &GithubClientMock{}
		stepPlugins := map[string]StepPlugin{
			"jira_ticket_creation": NewStepPluginMock(),
		}

		fc := &ForcemergeImpl{
			local:        local,
			remote:       remote,
			githubclient: githubclient,
			stepPlugins:  stepPlugins,
		}

		resUrl, err := fc.ExecuteWorkflow(context.Background(), []string{"fmtest"}, "test-user", "fmtest", "explanation", map[string]string{"pr_url": "https://github.com/goliac-project/goliac/pull/32"}, false)

		assert.Nil(t, err)
		assert.Equal(t, 1, len(resUrl))
		assert.Equal(t, "mocked_url", resUrl[0])

	})
}
