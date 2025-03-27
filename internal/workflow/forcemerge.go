package workflow

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/goliac-project/goliac/internal/config"
	"github.com/goliac-project/goliac/internal/engine"
	"github.com/goliac-project/goliac/internal/entity"
	"github.com/goliac-project/goliac/internal/github"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Forcemerge interface {
	ExecuteForcemergeWorkflow(ctx context.Context, repoconfigForceMergeworkflows []string, username, workflowName, prPathToMerge, explanation string, dryrun bool) ([]string, error)
}

type ForcemergeLocalResource interface {
	ForcemergeWorkflows() map[string]*entity.ForcemergeWorkflow
	Teams() map[string]*entity.Team
}

type ForcemergeRemoteResource interface {
	Teams(ctx context.Context, current bool) map[string]*engine.GithubTeam
}

type ForcemergeImpl struct {
	local        ForcemergeLocalResource
	remote       ForcemergeRemoteResource
	githubclient github.GitHubClient
	stepPlugins  map[string]ForcemergeStepPlugin
}

func NewForcemergeImpl(local ForcemergeLocalResource, remote ForcemergeRemoteResource, githubclient github.GitHubClient) Forcemerge {
	stepPlugins := map[string]ForcemergeStepPlugin{
		"jira": NewForcemergeStepPlugJira(),
	}
	return &ForcemergeImpl{
		local:        local,
		remote:       remote,
		githubclient: githubclient,
		stepPlugins:  stepPlugins,
	}
}

func (g *ForcemergeImpl) ExecuteForcemergeWorkflow(ctx context.Context, repoconfigForceMergeworkflows []string, username, workflowName, prPathToMerge, explanation string, dryrun bool) ([]string, error) {
	var childSpan trace.Span
	if config.Config.OpenTelemetryEnabled {
		ctx, childSpan = otel.Tracer("goliac").Start(ctx, "ExecuteWorkflow")
		defer childSpan.End()
		childSpan.SetAttributes(
			attribute.String("workflow_name", workflowName),
			attribute.String("pr_path_to_merge", prPathToMerge),
		)
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

	// check workflow and acl
	w, err := g.getWorkflow(ctx, repoconfigForceMergeworkflows, workflowName, prPathToMerge, username)
	if err != nil {
		return nil, fmt.Errorf("unable to load the workflow: %v", err)
	}

	// execute the workflow
	if dryrun {
		return nil, nil
	}

	responses := []string{}
	for _, step := range w.Spec.Steps {
		plugin := g.stepPlugins[step.Name]
		if plugin == nil {
			return nil, fmt.Errorf("plugin %s not found", step.Name)
		}
		resp, err := plugin.Execute(ctx, username, explanation, url, step.Properties)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}

	// merge the PR
	err = g.mergePR(ctx, username, repo, prNumber, explanation)
	if err != nil {
		return nil, err
	}

	return responses, nil
}

// returns the corresponding workflow
func (g *ForcemergeImpl) getWorkflow(ctx context.Context, repoconfigForceMergeworkflows []string, workflowName, repo, username string) (*entity.ForcemergeWorkflow, error) {
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

	// check the ACL
	if !w.PassAcl(teams, repo) {
		return nil, fmt.Errorf("access denied")
	}
	return w, nil
}

func (g *ForcemergeImpl) mergePR(ctx context.Context, username string, repo string, prNumber, explanation string) error {
	body, err := g.githubclient.CallRestAPI(
		ctx,
		fmt.Sprintf("/repos/%s/%s/pulls/%s/reviews", config.Config.GithubAppOrganization, repo, prNumber),
		"",
		"POST",
		map[string]interface{}{
			"body":  fmt.Sprintf("force merge PR %s via Goliac on behalf of %s.\n%s", prNumber, username, explanation),
			"event": "APPROVE",
		},
		nil)
	if err != nil {
		return fmt.Errorf("error approving the PR: %v (%s)", err, string(body))
	}

	// https://docs.github.com/en/rest/pulls/pulls?apiVersion=2022-11-28#merge-a-pull-request
	body, err = g.githubclient.CallRestAPI(
		ctx,
		fmt.Sprintf("/repos/%s/%s/pulls/%s/merge", config.Config.GithubAppOrganization, repo, prNumber),
		"",
		"PUT",
		map[string]interface{}{
			"commit_title":   fmt.Sprintf("force merge PR %s for %s", prNumber, username),
			"commit_message": fmt.Sprintf("force merge PR %s via Goliac on behalf of %s", prNumber, username),
			"merge_method":   "merge", // can be "merge", "squash", or "rebase"
		},
		nil)
	if err != nil {
		if strings.Contains(err.Error(), "Method Not Allowed") {
			// in case of we want a squash merge
			body, err = g.githubclient.CallRestAPI(
				ctx,
				fmt.Sprintf("/repos/%s/%s/pulls/%s/merge", config.Config.GithubAppOrganization, repo, prNumber),
				"",
				"PUT",
				map[string]interface{}{
					"commit_title":   fmt.Sprintf("force merge PR %s for %s", prNumber, username),
					"commit_message": fmt.Sprintf("force merge PR %s via Goliac on behalf of %s", prNumber, username),
					"merge_method":   "squash", // can be "merge", "squash", or "rebase"
				},
				nil)
		}
	}
	if err != nil {
		return fmt.Errorf("error merging the PR: %v (%s)", err, string(body))
	}
	return nil
}
