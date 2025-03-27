package internal

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	"github.com/goliac-project/goliac/internal/config"
	"github.com/goliac-project/goliac/internal/entity"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (g *GoliacImpl) GetForcemergeWorkflows() map[string]*entity.ForcemergeWorkflow {
	workflows := make(map[string]*entity.ForcemergeWorkflow)
	lWorkflows := g.local.ForcemergeWorkflows()
	if lWorkflows == nil {
		return workflows
	}
	for _, name := range g.repoconfig.ForceMergeworkflows {
		w := g.local.ForcemergeWorkflows()
		if w != nil {
			workflows[name] = lWorkflows[name]
		}
	}
	return workflows
}

func (g *GoliacImpl) ExecuteForcemergeWorkflow(ctx context.Context, username string, workflowName string, prPathToMerge string, dryrun bool) ([]string, error) {
	var childSpan trace.Span
	if config.Config.OpenTelemetryEnabled {
		ctx, childSpan = otel.Tracer("goliac").Start(ctx, "ExecuteWorkflow")
		defer childSpan.End()
		childSpan.SetAttributes(
			attribute.String("workflow_name", workflowName),
			attribute.String("pr_path_to_merge", prPathToMerge),
		)
	}

	// check if the workflow is enabled
	workflows := g.repoconfig.ForceMergeworkflows
	if workflows == nil {
		return nil, fmt.Errorf("workflows not found")
	}
	found := false
	for _, w := range workflows {
		if w == workflowName {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("workflow not found")
	}

	// load the workflow
	forcemergeworkflows := g.local.ForcemergeWorkflows()
	if forcemergeworkflows == nil {
		return nil, fmt.Errorf("workflows not found")
	}

	w, ok := forcemergeworkflows[workflowName]
	if !ok {
		return nil, fmt.Errorf("workflows not found")
	}

	// check workflow acl
	teams := []string{}
	rTeams := g.remote.Teams(ctx, true)

	for _, lTeam := range g.local.Teams() {
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

	if !w.PassAcl(teams, prPathToMerge) {
		return nil, fmt.Errorf("access denied")
	}

	// check the prToMerge
	if prPathToMerge == "" {
		return nil, fmt.Errorf("prPathToMerge is empty")
	}
	url, err := url.Parse(prPathToMerge)
	if err != nil {
		return nil, fmt.Errorf("prPathToMerge is not a valid URL")
	}
	// let's extract the PR number and the repo
	prExtract := regexp.MustCompile(`.*/([^/]*)/pull/(\d+)`)
	prMatch := prExtract.FindStringSubmatch(url.Path)
	if len(prMatch) != 3 {
		return nil, fmt.Errorf("prPathToMerge is not a valid PR URL")
	}
	repo := prMatch[1]
	prNumber := prMatch[2]

	// execute the workflow
	if dryrun {
		return nil, nil
	}

	responses := []string{}
	for _, step := range w.Spec.Steps {
		if step.Name == "jira" {
			// execute
			resp, err := g.executeWorkflowStepJira(ctx, w, username, url)
			if err != nil {
				return nil, err
			}
			responses = append(responses, resp)
		}
	}

	// merge the PR
	err = g.mergePR(ctx, username, repo, prNumber)
	if err != nil {
		return nil, err
	}

	return responses, nil
}

func (g *GoliacImpl) executeWorkflowStepJira(ctx context.Context, w *entity.ForcemergeWorkflow, username string, url *url.URL) (string, error) {
	// TBD
	return "", nil
}

func (g *GoliacImpl) mergePR(ctx context.Context, username string, repo string, prNumber string) error {

	_, err := g.remoteGithubClient.CallRestAPI(
		ctx,
		fmt.Sprintf("/repos/%s/pulls/%s/merge", repo, prNumber),
		"",
		"PUT",
		map[string]interface{}{
			"commit_title":   fmt.Sprintf("force merge PR %s for %s", prNumber, username),
			"commit_message": fmt.Sprintf("force merge PR %s via Goliac on behalf of %s", prNumber, username),
			"merge_method":   "merge", // can be "merge", "squash", or "rebase"
		},
		nil)
	if err != nil {
		return err
	}
	return nil
}
