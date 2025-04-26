package workflow

import (
	"context"
	"fmt"
	"net/url"

	"github.com/goliac-project/goliac/internal/engine"
	"github.com/goliac-project/goliac/internal/entity"
)

// strip down version of Goliac Local
type WorkflowLocalResource interface {
	Workflows() map[string]*entity.Workflow
	Teams() map[string]*entity.Team
	Users() map[string]*entity.User // github username, user definition
}

// strip down version of Goliac Remote (if we have an externally managed team)
type WorkflowRemoteResource interface {
	Teams(ctx context.Context, current bool) map[string]*engine.GithubTeam
}

type StepPlugin interface {
	Execute(ctx context.Context, username, workflowDescription, explanation string, url *url.URL, properties map[string]interface{}) (string, error)
}

type Workflow interface {
	ExecuteWorkflow(ctx context.Context, repoconfigForceMergeworkflows []string, username, workflowName, explanation string, properties map[string]string, dryrun bool) ([]string, error)
}

// returns the corresponding workflow
func GetWorkflow(ctx context.Context, local WorkflowLocalResource, remote WorkflowRemoteResource, repoconfigForceMergeworkflows []string, workflowName, repo, githubId string) (*entity.Workflow, error) {
	// check if the workflow is enabled
	if repoconfigForceMergeworkflows == nil {
		return nil, fmt.Errorf("workflows not found")
	}
	found := false
	for _, w := range repoconfigForceMergeworkflows {
		if w == workflowName {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("workflow not found")
	}

	// load the workflow
	forcemergeworkflows := local.Workflows()
	if forcemergeworkflows == nil {
		return nil, fmt.Errorf("workflows not found")
	}

	w, ok := forcemergeworkflows[workflowName]
	if !ok {
		return nil, fmt.Errorf("workflows not found")
	}

	// fmt.Println("githubId", githubId)
	// fmt.Println("users", local.Users())

	// get the username
	username := ""
	if githubId != "" {
		users := local.Users()
		for name, user := range users {
			if user.Spec.GithubID == githubId {
				username = name
				break
			}
		}
	}
	// fmt.Println("username", username)

	// check workflow acl
	teams := []string{}
	rTeams := remote.Teams(ctx, true)

	for _, lTeam := range local.Teams() {
		if lTeam.Spec.ExternallyManaged {
			// get team definition from github
			rTeam := rTeams[lTeam.Name]
			if rTeam == nil {
				continue
			}
			for _, owner := range rTeam.Maintainers {
				if owner == username {
					teams = append(teams, lTeam.Name)
				}
			}
			for _, member := range rTeam.Members {
				if member == username {
					teams = append(teams, lTeam.Name)
				}
			}
		} else {
			// get team definition from local
			for _, owner := range lTeam.Spec.Owners {
				if owner == username {
					teams = append(teams, lTeam.Name)
				}
			}
			for _, member := range lTeam.Spec.Members {
				if member == username {
					teams = append(teams, lTeam.Name)
				}
			}
		}
	}

	// check the ACL
	if !w.PassAcl(teams, repo) {
		return nil, fmt.Errorf("access denied")
	}
	return w, nil
}
