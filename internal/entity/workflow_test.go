package entity

import (
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/goliac-project/goliac/internal/observability"
	"github.com/goliac-project/goliac/internal/utils"
	"github.com/stretchr/testify/assert"
)

func fixtureCreateWorkflow(t *testing.T, fs billy.Filesystem) {
	fs.MkdirAll("rulesets", 0755)
	err := utils.WriteFile(fs, "workflows/workflow1.yaml", []byte(`
apiVersion: v1
kind: Workflow
name: workflow1
spec:
  description: Geneal breaking glass PR merge workflow
  workflow_type: forcemerge
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

	err = utils.WriteFile(fs, "workflows/workflow2.yaml", []byte(`
apiVersion: v1
kind: Workflow
name: workflow2
spec:
  description: Geneal breaking glass PR merge workflow
  workflow_type: forcemerge
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

func TestWorkflow(t *testing.T) {

	// happy path
	t.Run("happy path", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateWorkflow(t, fs)

		logsCollector := observability.NewLogCollection()
		workflows := ReadWorkflowDirectory(fs, "workflows", logsCollector)
		assert.Equal(t, false, logsCollector.HasErrors())
		assert.Equal(t, false, logsCollector.HasWarns())
		assert.NotNil(t, workflows)
		assert.Equal(t, 2, len(workflows))
	})

	t.Run("happy path", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateWorkflow(t, fs)

		err := utils.WriteFile(fs, "workflows/workflow3.yaml", []byte(`
apiVersion: v1
kind: Workflow
name: workflow3
spec:
  description: Geneal breaking glass PR merge workflow
  workflow_type: forcemerge
  repositories:
    allowed:
      - ~ALL
  steps:
    - name: jira_ticket_creation
`), 0644)

		assert.Nil(t, err)

		logsCollector := observability.NewLogCollection()
		workflows := ReadWorkflowDirectory(fs, "workflows", logsCollector)
		assert.Equal(t, true, logsCollector.HasErrors()) // invalid jira_ticket_creation step
		assert.Equal(t, false, logsCollector.HasWarns())
		assert.NotNil(t, workflows)
		assert.Equal(t, 2, len(workflows))
	})

}
