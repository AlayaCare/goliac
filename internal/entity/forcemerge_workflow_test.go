package entity

import (
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/goliac-project/goliac/internal/observability"
	"github.com/goliac-project/goliac/internal/utils"
	"github.com/stretchr/testify/assert"
)

func fixtureCreateForcemergeWorkflow(t *testing.T, fs billy.Filesystem) {
	fs.MkdirAll("rulesets", 0755)
	err := utils.WriteFile(fs, "forcemerge_workflows/workflow1.yaml", []byte(`
apiVersion: v1
kind: ForcemergeWorkflow
name: workflow1
spec:
  description: Geneal breaking glass PR merge workflow
  repositories:
    allowed:
      - .*
    except:
      - repo2
  acls:
    allowed:
      - team.*
    except:
      - team5
  steps:
    - name: jira_ticket_creation
      properties:
        project_key: SRE
    - name: slack_notification
      properties:
        channel: sre
`), 0644)
	assert.Nil(t, err)

	err = utils.WriteFile(fs, "forcemerge_workflows/workflow2.yaml", []byte(`
apiVersion: v1
kind: ForcemergeWorkflow
name: workflow2
spec:
  description: Geneal breaking glass PR merge workflow
  repositories:
    allowed:
      - repo2
  acls:
    allowed:
      - team5
  steps:
    - name: jira_ticket_creation
      properties:
        project_key: SRE
`), 0644)
	assert.Nil(t, err)
}

func TestForcemergeWorkflow(t *testing.T) {

	// happy path
	t.Run("happy path", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateForcemergeWorkflow(t, fs)

		errorCollector := observability.NewErrorCollection()
		workflows := ReadForcemergeWorkflowDirectory(fs, "forcemerge_workflows", errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, workflows)
		assert.Equal(t, 2, len(workflows))
	})

	t.Run("happy path: testing acls", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateForcemergeWorkflow(t, fs)

		errorCollector := observability.NewErrorCollection()
		workflows := ReadForcemergeWorkflowDirectory(fs, "forcemerge_workflows", errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, workflows)
		assert.Equal(t, 2, len(workflows))

		// check the acls
		assert.True(t, workflows["workflow1"].PassAcl([]string{"team1", "team2"}, "repo1"))
		assert.False(t, workflows["workflow1"].PassAcl([]string{"team1", "team2"}, "repo2"))
		assert.False(t, workflows["workflow1"].PassAcl([]string{"team5"}, "repo1"))

		assert.True(t, workflows["workflow2"].PassAcl([]string{"team1", "team5"}, "repo2"))
		assert.False(t, workflows["workflow2"].PassAcl([]string{"team1", "team2"}, "repo2"))
		assert.False(t, workflows["workflow2"].PassAcl([]string{"team5"}, "repo1"))
	})

	t.Run("happy path", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateForcemergeWorkflow(t, fs)

		err := utils.WriteFile(fs, "forcemerge_workflows/workflow3.yaml", []byte(`
apiVersion: v1
kind: ForcemergeWorkflow
name: workflow3
spec:
  description: Geneal breaking glass PR merge workflow
  repositories:
    allowed:
      - ~ALL
  steps:
    - name: jira_ticket_creation
`), 0644)

		assert.Nil(t, err)

		errorCollector := observability.NewErrorCollection()
		workflows := ReadForcemergeWorkflowDirectory(fs, "forcemerge_workflows", errorCollector)
		assert.Equal(t, true, errorCollector.HasErrors()) // invalid jira_ticket_creation step
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, workflows)
		assert.Equal(t, 2, len(workflows))
	})

}
