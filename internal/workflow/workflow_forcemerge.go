package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/goliac-project/goliac/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ForcemergeImpl struct {
	ws          WorkflowService
	stepPlugins map[string]StepPlugin
}

func NewForcemergeImpl(ws WorkflowService) Workflow {
	return &ForcemergeImpl{
		ws:          ws,
		stepPlugins: GetPlugins(),
	}
}

func (g *ForcemergeImpl) ExecuteWorkflow(ctx context.Context, repoconfigForceMergeworkflows []string, username, workflowName, explanation string, properties map[string]string, dryrun bool) ([]string, error) {
	var childSpan trace.Span
	if config.Config.OpenTelemetryEnabled {
		ctx, childSpan = otel.Tracer("goliac").Start(ctx, "ExecuteWorkflow")
		defer childSpan.End()
		childSpan.SetAttributes(
			attribute.String("workflow_name", workflowName),
			attribute.String("pr_path_to_merge", properties["pr_url"]),
		)
	}

	pr_url := properties["pr_url"]
	prPathToMerge := strings.TrimSpace(pr_url)

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
	w, err := g.ws.GetWorkflow(ctx, repoconfigForceMergeworkflows, workflowName, repo, username)
	if err != nil {
		return nil, fmt.Errorf("unable to execute the workflow: %v", err)
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
		resp, err := plugin.Execute(ctx, username, w.Spec.Description, explanation, url, step.Properties)
		if err != nil {
			return nil, fmt.Errorf("error when executing step %s: %v", step.Name, err)
		}
		responses = append(responses, resp)
	}

	// merge the PR
	err = g.mergePR(ctx, username, repo, prNumber, prPathToMerge, explanation)
	if err != nil {
		return nil, fmt.Errorf("error when merging the PR: %v", err)
	}

	return responses, nil
}

func (g *ForcemergeImpl) fetchPullRequestTitle(ctx context.Context, repo, prNumber string) (string, error) {
	body, err := g.ws.CallRestAPI(
		ctx,
		fmt.Sprintf("/repos/%s/%s/pulls/%s", config.Config.GithubAppOrganization, repo, prNumber),
		"",
		"GET",
		nil,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("error loading pull request %s/%s: %w", repo, prNumber, err)
	}
	var pr struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(body, &pr); err != nil {
		return "", fmt.Errorf("error parsing pull request %s/%s: %w", repo, prNumber, err)
	}
	return pr.Title, nil
}

func (g *ForcemergeImpl) mergePR(ctx context.Context, username string, repo string, prNumber, prURL, explanation string) error {
	mergeMethod := "merge"
	if strings.Contains(explanation, "/squash") {
		mergeMethod = "squash"
	}
	// let's remove the /squash string from the explanation
	explanation = strings.ReplaceAll(explanation, "/squash", "")

	prTitle, err := g.fetchPullRequestTitle(ctx, repo, prNumber)
	if err != nil {
		return err
	}

	reviewBody := fmt.Sprintf(
		"Force merge via Goliac on behalf of %s.\n\nPR #%s: %s\n%s\n\n%s",
		username, prNumber, prTitle, prURL, explanation,
	)
	commitTitle := fmt.Sprintf("%s (force merged PR %s by %s)", prTitle, prNumber, username)
	commitMessage := fmt.Sprintf(
		"Force merge via Goliac on behalf of %s.\n\nPR: %s\nTitle: %s",
		username, prURL, prTitle,
	)

	body, err := g.ws.CallRestAPI(
		ctx,
		fmt.Sprintf("/repos/%s/%s/pulls/%s/reviews", config.Config.GithubAppOrganization, repo, prNumber),
		"",
		"POST",
		map[string]interface{}{
			"body":  reviewBody,
			"event": "APPROVE",
		},
		nil)
	if err != nil {
		return fmt.Errorf("error approving the PR: %v (%s)", err, string(body))
	}

	// https://docs.github.com/en/rest/pulls/pulls?apiVersion=2022-11-28#merge-a-pull-request
	body, err = g.ws.CallRestAPI(
		ctx,
		fmt.Sprintf("/repos/%s/%s/pulls/%s/merge", config.Config.GithubAppOrganization, repo, prNumber),
		"",
		"PUT",
		map[string]interface{}{
			"commit_title":   commitTitle,
			"commit_message": commitMessage,
			"merge_method":   mergeMethod, // can be "merge", "squash", or "rebase"
		},
		nil)
	if err != nil && mergeMethod == "merge" {
		if strings.Contains(err.Error(), "Method Not Allowed") {
			// in case of we want a squash merge
			body, err = g.ws.CallRestAPI(
				ctx,
				fmt.Sprintf("/repos/%s/%s/pulls/%s/merge", config.Config.GithubAppOrganization, repo, prNumber),
				"",
				"PUT",
				map[string]interface{}{
					"commit_title":   commitTitle,
					"commit_message": commitMessage,
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
