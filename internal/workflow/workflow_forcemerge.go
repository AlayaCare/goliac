package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/goliac-project/goliac/internal/config"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ForcemergeImpl struct {
	ws          WorkflowService
	stepPlugins map[string]StepPlugin
	// bounds for the asynchronous merge poll (0 = defaults)
	asyncPollInterval time.Duration
	asyncPollTimeout  time.Duration
	// how long to wait for GitHub to mark a PR merged into its parent branch (0 = default)
	mergedCheckTimeout time.Duration
}

const (
	// PropertyMergeMode is the workflow property holding the optional merge mode of a force merge
	// ("/forcemerge:<workflow>:<stack|direct>:<explanation>")
	PropertyMergeMode = "merge_mode"
	// MergeModeDirect merges a stacked PR alone into its base branch and restacks the PRs above it (default)
	MergeModeDirect = "direct"
	// MergeModeStack merges the whole stack the PR belongs to into the stack base branch
	MergeModeStack = "stack"
)

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

	mergeMode := strings.ToLower(strings.TrimSpace(properties[PropertyMergeMode]))
	if mergeMode != "" && mergeMode != MergeModeDirect && mergeMode != MergeModeStack {
		return nil, fmt.Errorf("unknown merge mode %q (expected %q or %q)", mergeMode, MergeModeStack, MergeModeDirect)
	}

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
	err = g.mergePR(ctx, username, repo, prNumber, prPathToMerge, explanation, mergeMode)
	if err != nil {
		return nil, fmt.Errorf("error when merging the PR: %v", err)
	}

	return responses, nil
}

// pullRequestInfo is the subset of the GitHub pull request payload the force merge needs
type pullRequestInfo struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
	Base    struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Head struct {
		Ref string `json:"ref"`
		Sha string `json:"sha"`
	} `json:"head"`
	// Stack is set by GitHub when the PR belongs to a stack of pull requests (null otherwise)
	Stack *struct {
		Position int `json:"position"`
		Size     int `json:"size"`
		Base     struct {
			Ref string `json:"ref"`
		} `json:"base"`
	} `json:"stack"`
}

func (pr *pullRequestInfo) isStacked() bool {
	return pr.Stack != nil
}

// isStackBottom is true when the PR targets the base branch of its stack
// (i.e. no other open PR is below it)
func (pr *pullRequestInfo) isStackBottom() bool {
	return pr.Stack != nil && pr.Base.Ref == pr.Stack.Base.Ref
}

func (g *ForcemergeImpl) fetchPullRequest(ctx context.Context, repo, prNumber string) (*pullRequestInfo, error) {
	body, err := g.ws.CallRestAPI(
		ctx,
		fmt.Sprintf("/repos/%s/%s/pulls/%s", config.Config.GithubAppOrganization, repo, prNumber),
		"",
		"GET",
		nil,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error loading pull request %s/%s: %w", repo, prNumber, err)
	}
	var pr pullRequestInfo
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("error parsing pull request %s/%s: %w", repo, prNumber, err)
	}
	return &pr, nil
}

// fetchDownstack returns the open PRs below pr in its stack, from the closest one
// down to the bottom of the stack, by following each PR's base branch.
func (g *ForcemergeImpl) fetchDownstack(ctx context.Context, repo string, pr *pullRequestInfo) ([]*pullRequestInfo, error) {
	downstack := []*pullRequestInfo{}
	stackBase := pr.Stack.Base.Ref
	current := pr
	for current.Base.Ref != stackBase {
		if len(downstack) >= pr.Stack.Size {
			return nil, fmt.Errorf("unable to walk the stack of PR #%d down to %s: more than %d PRs found", pr.Number, stackBase, pr.Stack.Size)
		}
		body, err := g.ws.CallRestAPI(
			ctx,
			fmt.Sprintf("/repos/%s/%s/pulls", config.Config.GithubAppOrganization, repo),
			fmt.Sprintf("state=open&head=%s", url.QueryEscape(config.Config.GithubAppOrganization+":"+current.Base.Ref)),
			"GET",
			nil,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("error looking for the PR of branch %s: %v (%s)", current.Base.Ref, err, string(body))
		}
		var prs []*pullRequestInfo
		if err := json.Unmarshal(body, &prs); err != nil {
			return nil, fmt.Errorf("error parsing the PRs of branch %s: %w", current.Base.Ref, err)
		}
		if len(prs) == 0 {
			return nil, fmt.Errorf("no open PR found for branch %s below PR #%d in the stack", current.Base.Ref, current.Number)
		}
		downstack = append(downstack, prs[0])
		current = prs[0]
	}
	return downstack, nil
}

// fetchUpstack returns the open PRs above pr in its stack, from the closest one
// up to the top of the stack, by following the PRs based on each head branch.
func (g *ForcemergeImpl) fetchUpstack(ctx context.Context, repo string, pr *pullRequestInfo) ([]*pullRequestInfo, error) {
	upstack := []*pullRequestInfo{}
	current := pr
	for {
		prs, err := g.listOpenPRsBasedOn(ctx, repo, current.Head.Ref)
		if err != nil {
			return nil, err
		}
		if len(prs) == 0 {
			return upstack, nil
		}
		if len(prs) > 1 {
			return nil, fmt.Errorf("%d open PRs are based on branch %s (above PR #%d): the stack is ambiguous", len(prs), current.Head.Ref, current.Number)
		}
		if len(upstack) >= pr.Stack.Size {
			return nil, fmt.Errorf("unable to walk the stack of PR #%d up: more than %d PRs found", pr.Number, pr.Stack.Size)
		}
		upstack = append(upstack, prs[0])
		current = prs[0]
	}
}

// listOpenPRsBasedOn returns the open PRs whose base branch is branch
func (g *ForcemergeImpl) listOpenPRsBasedOn(ctx context.Context, repo, branch string) ([]*pullRequestInfo, error) {
	body, err := g.ws.CallRestAPI(
		ctx,
		fmt.Sprintf("/repos/%s/%s/pulls", config.Config.GithubAppOrganization, repo),
		fmt.Sprintf("state=open&base=%s", url.QueryEscape(branch)),
		"GET",
		nil,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error looking for the PRs based on branch %s: %v (%s)", branch, err, string(body))
	}
	var prs []*pullRequestInfo
	if err := json.Unmarshal(body, &prs); err != nil {
		return nil, fmt.Errorf("error parsing the PRs based on branch %s: %w", branch, err)
	}
	return prs, nil
}

func forcemergeReviewBody(username string, prNumber int, prTitle, prURL, explanation string) string {
	return fmt.Sprintf(
		"Force merge via Goliac on behalf of %s.\n\nPR #%d: %s\n%s\n\n%s",
		username, prNumber, prTitle, prURL, explanation,
	)
}

func forcemergeCommitTitle(prTitle string, prNumber int, username string) string {
	return fmt.Sprintf("%s (force merged PR %d by %s)", prTitle, prNumber, username)
}

func forcemergeCommitMessage(username, prURL, prTitle string) string {
	return fmt.Sprintf(
		"Force merge via Goliac on behalf of %s.\n\nPR: %s\nTitle: %s",
		username, prURL, prTitle,
	)
}

func (g *ForcemergeImpl) approvePR(ctx context.Context, repo string, prNumber int, reviewBody string) error {
	body, err := g.ws.CallRestAPI(
		ctx,
		fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", config.Config.GithubAppOrganization, repo, prNumber),
		"",
		"POST",
		map[string]interface{}{
			"body":  reviewBody,
			"event": "APPROVE",
		},
		nil)
	if err != nil {
		return fmt.Errorf("error approving the PR #%d: %v (%s)", prNumber, err, string(body))
	}
	return nil
}

/*
 * mergePR approves and merges the PR.
 *
 * "/squash" in the explanation asks for a squash merge.
 *
 * mergeMode only applies to a PR that is part of a GitHub stack:
 * - "" or "direct": merge this PR only into its base branch, then restack the PRs above it
 * - "stack": merge the whole stack into the stack base branch
 */
func (g *ForcemergeImpl) mergePR(ctx context.Context, username string, repo string, prNumber, prURL, explanation, mergeMode string) error {
	mergeMethod := "merge"
	if strings.Contains(explanation, "/squash") {
		mergeMethod = "squash"
	}
	// let's remove the /squash string from the explanation
	explanation = strings.ReplaceAll(explanation, "/squash", "")

	pr, err := g.fetchPullRequest(ctx, repo, prNumber)
	if err != nil {
		return err
	}
	if pr.Number == 0 {
		// the URL is authoritative when the payload has no number
		pr.Number, _ = strconv.Atoi(prNumber)
	}
	pr.HTMLURL = prURL

	if pr.isStacked() {
		if mergeMode == MergeModeStack {
			return g.mergeWholeStack(ctx, username, repo, pr, explanation, mergeMethod)
		}
		return g.mergeDirect(ctx, username, repo, pr, explanation, mergeMethod)
	}

	reviewBody := forcemergeReviewBody(username, pr.Number, pr.Title, prURL, explanation)
	if mergeMode != "" {
		reviewBody += fmt.Sprintf("\n\nMerge mode `%s` ignored: this PR is not part of a stack.", mergeMode)
	}
	commitTitle := forcemergeCommitTitle(pr.Title, pr.Number, username)
	commitMessage := forcemergeCommitMessage(username, prURL, pr.Title)

	if err := g.approvePR(ctx, repo, pr.Number, reviewBody); err != nil {
		return err
	}

	// https://docs.github.com/en/rest/pulls/pulls?apiVersion=2022-11-28#merge-a-pull-request
	body, err := g.mergePRSync(ctx, repo, prNumber, commitTitle, commitMessage, mergeMethod)
	if err != nil && mergeMethod == "merge" {
		if strings.Contains(err.Error(), "Method Not Allowed") {
			// in case of we want a squash merge
			body, err = g.mergePRSync(ctx, repo, prNumber, commitTitle, commitMessage, "squash")
		}
	}
	if err != nil && isStackedPRRefusal(body) {
		// GitHub says the PR is stacked but its payload has no stack information:
		// don't guess what part of the stack to merge
		return fmt.Errorf("error merging the PR: GitHub reports a stacked PR but returned no stack information for it, nothing was merged: %v (%s)", err, string(body))
	}
	if err != nil {
		return fmt.Errorf("error merging the PR: %v (%s)", err, string(body))
	}
	return nil
}

/*
 * mergeDirect merges a stacked PR alone into its base branch, then restacks:
 * with the stack 1 > 2 > 3 > 4 > main, a direct merge of 3 merges 3 into 4's branch
 * and retargets 2 onto 4's branch, giving 1 > 2 > 4 > main.
 * The bottom PR of a stack (base = stack base branch) is merged with the asynchronous merge API.
 */
func (g *ForcemergeImpl) mergeDirect(ctx context.Context, username, repo string, pr *pullRequestInfo, explanation, mergeMethod string) error {
	bottom := pr.isStackBottom()
	commitTitle := forcemergeCommitTitle(pr.Title, pr.Number, username)
	commitMessage := forcemergeCommitMessage(username, pr.HTMLURL, pr.Title)

	reviewBody := forcemergeReviewBody(username, pr.Number, pr.Title, pr.HTMLURL, explanation)
	reviewBody += fmt.Sprintf("\n\nStacked PR: merged alone into its base branch `%s`; the PRs based on `%s` are retargeted to `%s`.", pr.Base.Ref, pr.Head.Ref, pr.Base.Ref)
	if !bottom && mergeMethod == "squash" {
		reviewBody += " /squash is not available when merging into a parent branch: a merge commit is used instead."
	}

	if err := g.approvePR(ctx, repo, pr.Number, reviewBody); err != nil {
		return err
	}

	mergeSha := ""
	if bottom {
		// https://docs.github.com/en/rest/pulls/pulls#merge-a-pull-request-asynchronously
		// (the bottom PR has no downstack PR: only this PR is merged)
		if err := g.mergePRAsync(ctx, repo, strconv.Itoa(pr.Number), pr.Head.Sha, commitTitle, commitMessage, mergeMethod); err != nil {
			return err
		}
	} else {
		sha, err := g.mergeIntoParentBranch(ctx, repo, pr, commitTitle, commitMessage)
		if err != nil {
			return err
		}
		mergeSha = sha
	}

	restackErr := g.restack(ctx, repo, pr)

	if !bottom {
		// the asynchronous merge already reported "merged" for the bottom PR
		if err := g.ensureMerged(ctx, username, repo, pr, mergeSha); err != nil {
			if restackErr != nil {
				return fmt.Errorf("%v; %v", restackErr, err)
			}
			return err
		}
	}
	return restackErr
}

// restack retargets the open PRs based on the merged PR's head branch onto its base branch
func (g *ForcemergeImpl) restack(ctx context.Context, repo string, pr *pullRequestInfo) error {
	upstack, err := g.listOpenPRsBasedOn(ctx, repo, pr.Head.Ref)
	if err != nil {
		return fmt.Errorf("PR #%d merged, but restacking failed: %v", pr.Number, err)
	}
	errs := []string{}
	for _, up := range upstack {
		body, err := g.ws.CallRestAPI(
			ctx,
			fmt.Sprintf("/repos/%s/%s/pulls/%d", config.Config.GithubAppOrganization, repo, up.Number),
			"",
			"PATCH",
			map[string]interface{}{
				"base": pr.Base.Ref,
			},
			nil)
		if err != nil {
			errs = append(errs, fmt.Sprintf("PR #%d: %v (%s)", up.Number, err, string(body)))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("PR #%d merged, but retargeting onto %s failed for %s", pr.Number, pr.Base.Ref, strings.Join(errs, ", "))
	}
	return nil
}

// mergeIntoParentBranch merges a mid-stack PR into its base branch (the branch of the PR below it)
// with a merge commit, and returns the merge commit sha ("" if there was nothing to merge).
// https://docs.github.com/en/rest/branches/branches#merge-a-branch
func (g *ForcemergeImpl) mergeIntoParentBranch(ctx context.Context, repo string, pr *pullRequestInfo, commitTitle, commitMessage string) (string, error) {
	body, err := g.ws.CallRestAPI(
		ctx,
		fmt.Sprintf("/repos/%s/%s/merges", config.Config.GithubAppOrganization, repo),
		"",
		"POST",
		map[string]interface{}{
			"base":           pr.Base.Ref,
			"head":           pr.Head.Sha,
			"commit_message": commitTitle + "\n\n" + commitMessage,
		},
		nil)
	if err != nil {
		if strings.Contains(err.Error(), "409") {
			return "", fmt.Errorf("error merging the PR into its base branch %s: merge conflict, update the PR with its base branch and retry: %v (%s)", pr.Base.Ref, err, string(body))
		}
		return "", fmt.Errorf("error merging the PR into its base branch %s: %v (%s)", pr.Base.Ref, err, string(body))
	}
	var commit struct {
		Sha string `json:"sha"`
	}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &commit)
	}
	return commit.Sha, nil
}

/*
 * ensureMerged waits for GitHub to mark the PR merged once its head is part of its base branch
 * ("A pull request can be marked as merged if its head branch commits become reachable from the
 * base branch outside that pull request", GitHub docs, "About pull request merges").
 * If GitHub does not, the PR is closed with a comment explaining it was force merged.
 */
func (g *ForcemergeImpl) ensureMerged(ctx context.Context, username, repo string, pr *pullRequestInfo, mergeSha string) error {
	interval := g.asyncPollInterval
	if interval <= 0 {
		interval = defaultAsyncMergePollInterval
	}
	timeout := g.mergedCheckTimeout
	if timeout <= 0 {
		timeout = defaultMergedCheckTimeout
	}
	deadline := time.Now().Add(timeout)

	for {
		body, err := g.ws.CallRestAPI(
			ctx,
			fmt.Sprintf("/repos/%s/%s/pulls/%d", config.Config.GithubAppOrganization, repo, pr.Number),
			"",
			"GET",
			nil,
			nil)
		if err == nil {
			var state struct {
				State  string `json:"state"`
				Merged bool   `json:"merged"`
			}
			if jerr := json.Unmarshal(body, &state); jerr == nil {
				if state.Merged {
					return nil
				}
				if state.State == "closed" {
					// closed by someone else in between: nothing more to do
					return nil
				}
			}
		}
		if !time.Now().Add(interval).Before(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("PR #%d merged into %s, but checking its state failed: %v", pr.Number, pr.Base.Ref, ctx.Err())
		case <-time.After(interval):
		}
	}

	// GitHub did not mark it merged: close it, saying where it went
	comment := fmt.Sprintf("Force merged via Goliac on behalf of %s into its base branch `%s`", username, pr.Base.Ref)
	if mergeSha != "" {
		comment += fmt.Sprintf(" (merge commit %s)", mergeSha)
	}
	comment += ". GitHub did not mark this PR as merged, so Goliac closes it."
	body, err := g.ws.CallRestAPI(
		ctx,
		fmt.Sprintf("/repos/%s/%s/issues/%d/comments", config.Config.GithubAppOrganization, repo, pr.Number),
		"",
		"POST",
		map[string]interface{}{
			"body": comment,
		},
		nil)
	if err != nil {
		return fmt.Errorf("PR #%d merged into %s, but commenting before closing it failed: %v (%s)", pr.Number, pr.Base.Ref, err, string(body))
	}
	body, err = g.ws.CallRestAPI(
		ctx,
		fmt.Sprintf("/repos/%s/%s/pulls/%d", config.Config.GithubAppOrganization, repo, pr.Number),
		"",
		"PATCH",
		map[string]interface{}{
			"state": "closed",
		},
		nil)
	if err != nil {
		return fmt.Errorf("PR #%d merged into %s, but closing it failed: %v (%s)", pr.Number, pr.Base.Ref, err, string(body))
	}
	return nil
}

/*
 * mergeWholeStack merges every PR of the stack into the stack base branch:
 * with the stack 1 > 2 > 3 > 4 > main, a stack merge requested on any of them merges 1 to 4 into main.
 * Every PR of the stack is approved, then the top PR is merged with the asynchronous merge API,
 * which includes all its open downstack PRs.
 */
func (g *ForcemergeImpl) mergeWholeStack(ctx context.Context, username, repo string, pr *pullRequestInfo, explanation, mergeMethod string) error {
	upstack, err := g.fetchUpstack(ctx, repo, pr)
	if err != nil {
		return fmt.Errorf("nothing merged: %v", err)
	}
	downstack, err := g.fetchDownstack(ctx, repo, pr)
	if err != nil {
		return fmt.Errorf("nothing merged: %v", err)
	}

	// top to bottom
	stack := []*pullRequestInfo{}
	for i := len(upstack) - 1; i >= 0; i-- {
		stack = append(stack, upstack[i])
	}
	stack = append(stack, pr)
	stack = append(stack, downstack...)

	if len(stack) != pr.Stack.Size {
		numbers := []string{}
		for _, p := range stack {
			numbers = append(numbers, fmt.Sprintf("#%d", p.Number))
		}
		return fmt.Errorf("nothing merged: GitHub reports a stack of %d PRs but only %d open PRs were found (%s): a PR in the stack is missing or not open", pr.Stack.Size, len(stack), strings.Join(numbers, " > "))
	}
	top := stack[0]
	bottom := stack[len(stack)-1]

	for _, p := range stack {
		reviewBody := forcemergeReviewBody(username, p.Number, p.Title, p.HTMLURL, explanation) +
			fmt.Sprintf("\n\nMerging the whole stack (%d PRs, #%d to #%d) into `%s`, requested on PR #%d.", len(stack), top.Number, bottom.Number, pr.Stack.Base.Ref, pr.Number)
		if err := g.approvePR(ctx, repo, p.Number, reviewBody); err != nil {
			return err
		}
	}

	// https://docs.github.com/en/rest/pulls/pulls#merge-a-pull-request-asynchronously
	// "For a stacked pull request, the operation includes all open downstack pull requests."
	return g.mergePRAsync(
		ctx,
		repo,
		strconv.Itoa(top.Number),
		top.Head.Sha,
		forcemergeCommitTitle(top.Title, top.Number, username),
		forcemergeCommitMessage(username, top.HTMLURL, top.Title),
		mergeMethod,
	)
}

func (g *ForcemergeImpl) mergePRSync(ctx context.Context, repo, prNumber, commitTitle, commitMessage, mergeMethod string) ([]byte, error) {
	return g.ws.CallRestAPI(
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
}

// isStackedPRRefusal detects the 403 GitHub returns when a stacked PR is merged
// through the synchronous merge endpoint:
// "Merging stacked PRs via this endpoint is not supported. Use the asynchronous merge endpoint instead."
func isStackedPRRefusal(body []byte) bool {
	msg := strings.ToLower(string(body))
	return strings.Contains(msg, "stacked") && strings.Contains(msg, "asynchronous merge endpoint")
}

const (
	defaultAsyncMergePollInterval = 2 * time.Second
	defaultAsyncMergePollTimeout  = 2 * time.Minute
	defaultMergedCheckTimeout     = 30 * time.Second
)

// asyncMergeResult is the payload of
// PUT /repos/{owner}/{repo}/pulls/{pull_number}/merge-async and
// GET /repos/{owner}/{repo}/pulls/{pull_number}/merge-async/{uuid}
type asyncMergeResult struct {
	Status  string `json:"status"` // pending, merged, enqueued, failed
	Details struct {
		Message string `json:"message"`
		UUID    string `json:"uuid"`
		Sha     string `json:"sha"`
	} `json:"details"`
}

func (r *asyncMergeResult) describe() string {
	if r.Details.Message != "" {
		return r.Details.Message
	}
	return "no message from GitHub"
}

// mergePRAsync merges a PR through the asynchronous merge endpoint and waits
// (bounded) for the outcome.
// https://docs.github.com/en/rest/pulls/pulls#merge-a-pull-request-asynchronously
func (g *ForcemergeImpl) mergePRAsync(ctx context.Context, repo, prNumber, headSha, commitTitle, commitMessage, mergeMethod string) error {
	payload := map[string]interface{}{
		"commit_title":   commitTitle,
		"commit_message": commitMessage,
		"merge_method":   mergeMethod,
		// a force merge merges now: never park the PR in a merge queue
		"merge_action": "direct_merge",
		// same as the synchronous endpoint: let Goliac use its bypass permissions
		"bypass_rules": true,
	}
	if headSha != "" {
		// only merge the head that was approved; GitHub cancels the merge if the PR is pushed meanwhile
		payload["sha"] = headSha
	}

	body, err := g.ws.CallRestAPI(
		ctx,
		fmt.Sprintf("/repos/%s/%s/pulls/%s/merge-async", config.Config.GithubAppOrganization, repo, prNumber),
		"",
		"PUT",
		payload,
		nil)
	if err != nil {
		// 409: an asynchronous merge is already pending for this PR, follow that one
		if strings.Contains(err.Error(), "409") {
			var pending asyncMergeResult
			if jerr := json.Unmarshal(body, &pending); jerr == nil && pending.Details.UUID != "" {
				return g.waitAsyncMerge(ctx, repo, prNumber, &pending)
			}
		}
		if strings.Contains(err.Error(), "400") {
			return fmt.Errorf("error merging the PR asynchronously: the PR is not ready to be merged (closed or draft?): %v (%s)", err, string(body))
		}
		return fmt.Errorf("error merging the PR asynchronously: %v (%s)", err, string(body))
	}

	var result asyncMergeResult
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("error parsing the asynchronous merge response: %v (%s)", err, string(body))
	}
	return g.waitAsyncMerge(ctx, repo, prNumber, &result)
}

// waitAsyncMerge polls the asynchronous merge request until it is final or the timeout is reached
func (g *ForcemergeImpl) waitAsyncMerge(ctx context.Context, repo, prNumber string, result *asyncMergeResult) error {
	interval := g.asyncPollInterval
	if interval <= 0 {
		interval = defaultAsyncMergePollInterval
	}
	timeout := g.asyncPollTimeout
	if timeout <= 0 {
		timeout = defaultAsyncMergePollTimeout
	}
	deadline := time.Now().Add(timeout)
	uuid := result.Details.UUID

	for {
		switch result.Status {
		case "merged":
			return nil
		case "enqueued":
			// we asked for a direct merge, so GitHub should not queue it; the PR will merge with the queue
			logrus.Warnf("PR %s/%s was added to a merge queue instead of being merged directly", repo, prNumber)
			return nil
		case "failed":
			return fmt.Errorf("GitHub refused the asynchronous merge (request %s): %s", uuid, result.describe())
		case "pending":
			if uuid == "" {
				return fmt.Errorf("asynchronous merge is pending but GitHub returned no request id: %s", result.describe())
			}
		default:
			return fmt.Errorf("unexpected asynchronous merge status %q (request %s): %s", result.Status, uuid, result.describe())
		}

		if !time.Now().Add(interval).Before(deadline) {
			return fmt.Errorf("timeout after %s waiting for the asynchronous merge (request %s) to complete: still pending, check the PR before retrying", timeout, uuid)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("asynchronous merge (request %s) still pending: %v", uuid, ctx.Err())
		case <-time.After(interval):
		}

		body, err := g.ws.CallRestAPI(
			ctx,
			fmt.Sprintf("/repos/%s/%s/pulls/%s/merge-async/%s", config.Config.GithubAppOrganization, repo, prNumber, uuid),
			"",
			"GET",
			nil,
			nil)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return fmt.Errorf("asynchronous merge request %s not found (expired?): %v (%s)", uuid, err, string(body))
			}
			// transient error: keep polling until the deadline
			logrus.Warnf("error polling asynchronous merge %s of PR %s/%s: %v (%s)", uuid, repo, prNumber, err, string(body))
			continue
		}
		var next asyncMergeResult
		if err := json.Unmarshal(body, &next); err != nil {
			return fmt.Errorf("error parsing the asynchronous merge status: %v (%s)", err, string(body))
		}
		if next.Details.UUID == "" {
			next.Details.UUID = uuid
		}
		result = &next
	}
}
