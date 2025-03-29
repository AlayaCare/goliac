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
type ForcemergeLocalResourceMoc struct {
	FcWorkflow map[string]*entity.ForcemergeWorkflow
	LTeams     map[string]*entity.Team
}

func (m *ForcemergeLocalResourceMoc) ForcemergeWorkflows() map[string]*entity.ForcemergeWorkflow {
	return m.FcWorkflow
}
func (m *ForcemergeLocalResourceMoc) Teams() map[string]*entity.Team {
	return m.LTeams
}

type ForcemergeRemoteResourceMoc struct {
	RTeams map[string]*engine.GithubTeam
}

func (m *ForcemergeRemoteResourceMoc) Teams(ctx context.Context, current bool) map[string]*engine.GithubTeam {
	return m.RTeams
}

type ForcemergeGithubClientMoc struct {
	LastCallEndpoint string
	LastCallBody     map[string]interface{}
}

func (m *ForcemergeGithubClientMoc) CallRestAPI(ctx context.Context, endpoint, parameters, method string, body map[string]interface{}, githubToken *string) ([]byte, error) {
	m.LastCallEndpoint = endpoint
	m.LastCallBody = body
	return nil, nil
}

type ForcemergeStepPluginMock struct{}

func (f *ForcemergeStepPluginMock) Execute(ctx context.Context, username, explanation string, url *url.URL, properties map[string]interface{}) (string, error) {
	// Mock implementation
	return "mocked_url", nil
}

func NewForcemergeStepPluginMock() ForcemergeStepPlugin {
	return &ForcemergeStepPluginMock{}
}

func fixtureWForcemergeorkflow() *entity.ForcemergeWorkflow {
	rfcWorkflow := &entity.ForcemergeWorkflow{}
	rfcWorkflow.Name = "fmtest"
	rfcWorkflow.ApiVersion = "v1"
	rfcWorkflow.Kind = "ForcemergeWorkflow"

	rfcWorkflow.Spec.Steps = []struct {
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
	rfcWorkflow.Spec.Description = "fmtest"
	rfcWorkflow.Spec.Repositories = struct {
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

	return rfcWorkflow
}

func TestForcemerge(t *testing.T) {
	t.Run("happy path: local team", func(t *testing.T) {
		lTeam := &entity.Team{}
		lTeam.Name = "test-team"
		lTeam.Spec.ExternallyManaged = false
		lTeam.Spec.Owners = []string{"test-user"}
		lTeam.Spec.Members = []string{}

		local := &ForcemergeLocalResourceMoc{
			FcWorkflow: map[string]*entity.ForcemergeWorkflow{
				"fmtest": fixtureWForcemergeorkflow(),
			},
			LTeams: map[string]*entity.Team{
				"test-team": lTeam,
			},
		}
		remote := &ForcemergeRemoteResourceMoc{
			RTeams: map[string]*engine.GithubTeam{},
		}

		githubclient := &ForcemergeGithubClientMoc{}
		stepPlugins := map[string]ForcemergeStepPlugin{
			"jira_ticket_creation": NewForcemergeStepPluginMock(),
		}

		fc := &ForcemergeImpl{
			local:        local,
			remote:       remote,
			githubclient: githubclient,
			stepPlugins:  stepPlugins,
		}

		resUrl, err := fc.ExecuteForcemergeWorkflow(context.Background(), []string{"fmtest"}, "test-user", "fmtest", "https://github.com/goliac-project/goliac/pull/32", "explanation", false)

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

		local := &ForcemergeLocalResourceMoc{
			FcWorkflow: map[string]*entity.ForcemergeWorkflow{
				"fmtest": fixtureWForcemergeorkflow(),
			},
			LTeams: map[string]*entity.Team{
				"test-team": lTeam,
			},
		}
		remote := &ForcemergeRemoteResourceMoc{
			RTeams: map[string]*engine.GithubTeam{
				"test-team": {
					Name:    "test-team",
					Slug:    "test-team",
					Members: []string{"test-user"},
				},
			},
		}

		githubclient := &ForcemergeGithubClientMoc{}
		stepPlugins := map[string]ForcemergeStepPlugin{
			"jira_ticket_creation": NewForcemergeStepPluginMock(),
		}

		fc := &ForcemergeImpl{
			local:        local,
			remote:       remote,
			githubclient: githubclient,
			stepPlugins:  stepPlugins,
		}

		resUrl, err := fc.ExecuteForcemergeWorkflow(context.Background(), []string{"fmtest"}, "test-user", "fmtest", "https://github.com/goliac-project/goliac/pull/32", "explanation", false)

		assert.Nil(t, err)
		assert.Equal(t, 1, len(resUrl))
		assert.Equal(t, "mocked_url", resUrl[0])

	})
}
