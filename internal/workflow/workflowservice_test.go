package workflow

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/goliac-project/goliac/internal/entity"
	"github.com/stretchr/testify/assert"
)

func fixtureGenericWorkflow(teamsAllowed string) *entity.Workflow {
	w := &entity.Workflow{}
	w.Name = "genericworkflow"
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
	w.Spec.Description = "generic workflow"
	w.Spec.WorkflowType = "generic"

	w.Spec.Repositories = struct {
		Allowed []string `yaml:"allowed"`
		Except  []string `yaml:"except"`
	}{
		Allowed: []string{"~ALL"},
		Except:  []string{},
	}

	w.Spec.Acls = struct {
		Allowed []string `yaml:"allowed"`
		Except  []string `yaml:"except"`
	}{
		Allowed: []string{teamsAllowed},
		Except:  []string{},
	}

	return w
}

type GithubClientApproversMock struct {
}

func (m *GithubClientApproversMock) CallRestAPI(ctx context.Context, endpoint, parameters, method string, body map[string]interface{}, githubToken *string) ([]byte, error) {
	var fileResponse struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	fileResponse.Content = base64.StdEncoding.EncodeToString([]byte(`
teams:
- test-team
users:
- test-user
`))
	fileResponse.Encoding = "base64"

	marshaled, err := json.Marshal(fileResponse)
	if err != nil {
		return nil, err
	}

	return marshaled, nil
}

func TestWorkflowService(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		local := &LocalResourceMock{
			MockWorkflows: map[string]*entity.Workflow{
				"genericworkflow": fixtureGenericWorkflow("test-team"),
			},
			LTeams: map[string]*entity.Team{},
			LUsers: map[string]*entity.User{},
		}
		testTeam := &entity.Team{}
		testTeam.Entity.Name = "test-team"
		testTeam.Spec.ExternallyManaged = false
		testTeam.Spec.Owners = []string{"test-user"}
		local.LTeams["test-team"] = testTeam

		testUser := &entity.User{}
		testUser.Entity.Name = "test-user"
		testUser.Spec.GithubID = "testUserGHID"
		local.LUsers["test-user"] = testUser

		remote := &RemoteResourceMock{}

		ws := NewWorkflowService("myorg", local, remote, &GithubClientMock{})

		w, err := ws.GetWorkflow(context.Background(), []string{"genericworkflow"}, "genericworkflow", "repo1", "testUserGHID")
		assert.Nil(t, err)
		assert.NotNil(t, w)
		assert.Equal(t, "genericworkflow", w.Name)
		assert.Equal(t, "generic", w.Spec.WorkflowType)
		assert.Equal(t, "generic workflow", w.Spec.Description)
		assert.Equal(t, []string{"test-team"}, w.Spec.Acls.Allowed)
		assert.Equal(t, []string{}, w.Spec.Acls.Except)
	})

	t.Run("happy path with repo approvers", func(t *testing.T) {
		local := &LocalResourceMock{
			MockWorkflows: map[string]*entity.Workflow{
				"genericworkflow": fixtureGenericWorkflow("~GOLIAC_REPOSITORY_APPROVERS"),
			},
			LTeams: map[string]*entity.Team{},
			LUsers: map[string]*entity.User{},
		}
		testTeam := &entity.Team{}
		testTeam.Entity.Name = "test-team"
		testTeam.Spec.ExternallyManaged = false
		testTeam.Spec.Owners = []string{"test-user"}
		local.LTeams["test-team"] = testTeam

		testUser := &entity.User{}
		testUser.Entity.Name = "test-user"
		testUser.Spec.GithubID = "testUserGHID"
		local.LUsers["test-user"] = testUser

		remote := &RemoteResourceMock{}

		ws := NewWorkflowService("myorg", local, remote, &GithubClientApproversMock{})

		w, err := ws.GetWorkflow(context.Background(), []string{"genericworkflow"}, "genericworkflow", "repo1", "testUserGHID")
		assert.Nil(t, err)
		assert.NotNil(t, w)
		assert.Equal(t, "genericworkflow", w.Name)
		assert.Equal(t, "generic", w.Spec.WorkflowType)
		assert.Equal(t, "generic workflow", w.Spec.Description)
		assert.Equal(t, []string{"~GOLIAC_REPOSITORY_APPROVERS"}, w.Spec.Acls.Allowed)
		assert.Equal(t, []string{}, w.Spec.Acls.Except)
	})

	t.Run("happy path with repo approvers: just the test-team", func(t *testing.T) {
		local := &LocalResourceMock{
			MockWorkflows: map[string]*entity.Workflow{
				"genericworkflow": fixtureGenericWorkflow("~GOLIAC_REPOSITORY_APPROVERS"),
			},
			LTeams: map[string]*entity.Team{},
			LUsers: map[string]*entity.User{},
		}
		testTeam := &entity.Team{}
		testTeam.Entity.Name = "test-team"
		testTeam.Spec.ExternallyManaged = false
		testTeam.Spec.Owners = []string{"another-user"}
		local.LTeams["test-team"] = testTeam

		testUser := &entity.User{}
		testUser.Entity.Name = "another-user"
		testUser.Spec.GithubID = "anotherUserGHID"
		local.LUsers["another-user"] = testUser

		remote := &RemoteResourceMock{}

		ws := NewWorkflowService("myorg", local, remote, &GithubClientApproversMock{})

		w, err := ws.GetWorkflow(context.Background(), []string{"genericworkflow"}, "genericworkflow", "repo1", "anotherUserGHID")
		assert.Nil(t, err)
		assert.NotNil(t, w)
	})

	t.Run("not happy path with repo approvers: neither matching team, nor user", func(t *testing.T) {
		local := &LocalResourceMock{
			MockWorkflows: map[string]*entity.Workflow{
				"genericworkflow": fixtureGenericWorkflow("~GOLIAC_REPOSITORY_APPROVERS"),
			},
			LTeams: map[string]*entity.Team{},
			LUsers: map[string]*entity.User{},
		}
		testTeam := &entity.Team{}
		testTeam.Entity.Name = "another-team"
		testTeam.Spec.ExternallyManaged = false
		testTeam.Spec.Owners = []string{"another-user"}
		local.LTeams["another-team"] = testTeam

		testUser := &entity.User{}
		testUser.Entity.Name = "another-user"
		testUser.Spec.GithubID = "anotherUserGHID"
		local.LUsers["another-user"] = testUser

		remote := &RemoteResourceMock{}

		ws := NewWorkflowService("myorg", local, remote, &GithubClientApproversMock{})

		w, err := ws.GetWorkflow(context.Background(), []string{"genericworkflow"}, "genericworkflow", "repo1", "anotherUserGHID")
		assert.NotNil(t, err)
		assert.Nil(t, w)
	})
}
