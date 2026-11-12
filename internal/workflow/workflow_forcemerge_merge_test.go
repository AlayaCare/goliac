package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/goliac-project/goliac/internal/config"
	"github.com/stretchr/testify/assert"
)

type restCall struct {
	Method     string
	Endpoint   string
	Parameters string
	Body       map[string]interface{}
}

// httpError mimics GitHubClientImpl.CallRestAPI on a non-2xx answer
func httpError(status string, body string) ([]byte, error) {
	return []byte(body), fmt.Errorf("unexpected status: %s", status)
}

type fakePR struct {
	Number  int
	Title   string
	BaseRef string
	HeadRef string
	HeadSha string
	State   string // open, closed
	Merged  bool
	// stack info (StackSize 0 when the PR is not stacked)
	StackPosition int
	StackSize     int
	StackBase     string
}

func (p *fakePR) payload() map[string]interface{} {
	m := map[string]interface{}{
		"number":   p.Number,
		"title":    p.Title,
		"html_url": fmt.Sprintf("https://github.com/myorg/goliac/pull/%d", p.Number),
		"state":    p.State,
		"merged":   p.Merged,
		"base":     map[string]interface{}{"ref": p.BaseRef},
		"head":     map[string]interface{}{"ref": p.HeadRef, "sha": p.HeadSha},
		"stack":    nil,
	}
	if p.StackSize > 0 {
		m["stack"] = map[string]interface{}{
			"position": p.StackPosition,
			"size":     p.StackSize,
			"base":     map[string]interface{}{"ref": p.StackBase},
		}
	}
	return m
}

/*
 * fakeGitHub is a small in-memory GitHub REST API for the force merge:
 * PR reads, PR lists by base/head, reviews, PR updates, issue comments.
 * Merge endpoints are answered by the test hooks.
 */
type fakeGitHub struct {
	prs   map[int]*fakePR
	calls []restCall

	syncMerge   func(call restCall) ([]byte, error)
	branchMerge func(call restCall) ([]byte, error)
	asyncMerge  func(call restCall) ([]byte, error)
	asyncPoll   func(call restCall) ([]byte, error)
	// markMergedOnBranchMerge simulates GitHub marking the PR merged once its head is in its base
	markMergedOnBranchMerge bool
}

var pullEndpoint = regexp.MustCompile(`^/repos/myorg/goliac/pulls/(\d+)$`)

func (f *fakeGitHub) CallRestAPI(ctx context.Context, endpoint, parameters, method string, body map[string]interface{}, githubToken *string) ([]byte, error) {
	call := restCall{Method: method, Endpoint: endpoint, Parameters: parameters, Body: body}
	f.calls = append(f.calls, call)

	unexpected := func() ([]byte, error) {
		return httpError("500 Internal Server Error", fmt.Sprintf(`{"message":"unexpected call %s %s"}`, method, endpoint))
	}
	hook := func(h func(call restCall) ([]byte, error)) ([]byte, error) {
		if h == nil {
			return unexpected()
		}
		return h(call)
	}

	switch {
	case method == "GET" && pullEndpoint.MatchString(endpoint):
		n, _ := strconv.Atoi(pullEndpoint.FindStringSubmatch(endpoint)[1])
		pr, ok := f.prs[n]
		if !ok {
			return httpError("404 Not Found", `{"message":"Not Found"}`)
		}
		return json.Marshal(pr.payload())

	case method == "GET" && endpoint == "/repos/myorg/goliac/pulls":
		q, _ := url.ParseQuery(parameters)
		res := []map[string]interface{}{}
		for _, n := range f.sortedNumbers() {
			pr := f.prs[n]
			if q.Get("state") == "open" && pr.State != "open" {
				continue
			}
			if b := q.Get("base"); b != "" && pr.BaseRef != b {
				continue
			}
			if h := q.Get("head"); h != "" && "myorg:"+pr.HeadRef != h {
				continue
			}
			res = append(res, pr.payload())
		}
		return json.Marshal(res)

	case method == "POST" && strings.HasSuffix(endpoint, "/reviews"):
		return []byte(`{}`), nil

	case method == "POST" && strings.HasSuffix(endpoint, "/comments"):
		return []byte(`{}`), nil

	case method == "PATCH" && pullEndpoint.MatchString(endpoint):
		n, _ := strconv.Atoi(pullEndpoint.FindStringSubmatch(endpoint)[1])
		pr := f.prs[n]
		if base, ok := body["base"].(string); ok {
			pr.BaseRef = base
		}
		if state, ok := body["state"].(string); ok {
			pr.State = state
		}
		return json.Marshal(pr.payload())

	case method == "PUT" && strings.HasSuffix(endpoint, "/merge"):
		return hook(f.syncMerge)

	case method == "POST" && endpoint == "/repos/myorg/goliac/merges":
		res, err := hook(f.branchMerge)
		if err == nil && f.markMergedOnBranchMerge {
			for _, pr := range f.prs {
				if pr.HeadSha == body["head"] {
					pr.Merged = true
					pr.State = "closed"
				}
			}
		}
		return res, err

	case method == "PUT" && strings.HasSuffix(endpoint, "/merge-async"):
		return hook(f.asyncMerge)

	case method == "GET" && strings.Contains(endpoint, "/merge-async/"):
		return hook(f.asyncPoll)
	}
	return unexpected()
}

func (f *fakeGitHub) sortedNumbers() []int {
	numbers := []int{}
	for n := range f.prs {
		numbers = append(numbers, n)
	}
	sort.Ints(numbers)
	return numbers
}

func (f *fakeGitHub) find(method, endpointSuffix string) []restCall {
	res := []restCall{}
	for _, c := range f.calls {
		if c.Method == method && strings.HasSuffix(c.Endpoint, endpointSuffix) {
			res = append(res, c)
		}
	}
	return res
}

func (f *fakeGitHub) reviewedPRs() []string {
	res := []string{}
	for _, c := range f.find("POST", "/reviews") {
		res = append(res, strings.TrimSuffix(strings.TrimPrefix(c.Endpoint, "/repos/myorg/goliac/pulls/"), "/reviews"))
	}
	return res
}

/*
 * fixtureStack returns the stack 1 > 2 > 3 > 4 > main:
 * PR 1 is the top (based on 2's branch), PR 4 the bottom (based on main).
 */
func fixtureStack() map[int]*fakePR {
	prs := map[int]*fakePR{}
	for n := 1; n <= 4; n++ {
		base := fmt.Sprintf("b%d", n+1)
		if n == 4 {
			base = "main"
		}
		prs[n] = &fakePR{
			Number:        n,
			Title:         fmt.Sprintf("PR %d", n),
			BaseRef:       base,
			HeadRef:       fmt.Sprintf("b%d", n),
			HeadSha:       fmt.Sprintf("sha%d", n),
			State:         "open",
			StackPosition: 5 - n,
			StackSize:     4,
			StackBase:     "main",
		}
	}
	return prs
}

func fixturePlainPR() map[int]*fakePR {
	return map[int]*fakePR{
		32: {Number: 32, Title: "Plain PR", BaseRef: "main", HeadRef: "feature", HeadSha: "aaa111", State: "open"},
	}
}

func newForcemergeForTest(t *testing.T, gh *fakeGitHub) *ForcemergeImpl {
	previousOrg := config.Config.GithubAppOrganization
	config.Config.GithubAppOrganization = "myorg"
	t.Cleanup(func() { config.Config.GithubAppOrganization = previousOrg })

	return &ForcemergeImpl{
		ws:                 NewWorkflowService("myorg", nil, nil, gh),
		stepPlugins:        map[string]StepPlugin{},
		asyncPollInterval:  time.Millisecond,
		asyncPollTimeout:   200 * time.Millisecond,
		mergedCheckTimeout: 20 * time.Millisecond,
	}
}

func prURL(n int) string {
	return fmt.Sprintf("https://github.com/myorg/goliac/pull/%d", n)
}

const (
	asyncPending = `{"status":"pending","details":{"message":"merge requested","uuid":"uuid-1","merge_method":"merge","merge_action":"direct_merge","expected_head_sha":"sha"}}`
	asyncMerged  = `{"status":"merged","details":{"message":"merged","sha":"mergecommit"}}`
)

func okJSON(s string) func(call restCall) ([]byte, error) {
	return func(call restCall) ([]byte, error) { return []byte(s), nil }
}

func TestForcemergeMergePRNotStacked(t *testing.T) {
	stackedRefusal := `{"message":"Merging stacked PRs via this endpoint is not supported. Use the asynchronous merge endpoint instead.","status":"403"}`

	for _, mode := range []string{"", MergeModeDirect, MergeModeStack} {
		t.Run(fmt.Sprintf("synchronous merge, mode %q", mode), func(t *testing.T) {
			gh := &fakeGitHub{prs: fixturePlainPR(), syncMerge: okJSON(`{"merged":true}`)}
			fc := newForcemergeForTest(t, gh)

			err := fc.mergePR(context.Background(), "test-user", "goliac", "32", prURL(32), "because", mode)
			assert.Nil(t, err)

			reviews := gh.find("POST", "/pulls/32/reviews")
			assert.Equal(t, 1, len(reviews))
			assert.Equal(t, "APPROVE", reviews[0].Body["event"])
			expectedReview := "Force merge via Goliac on behalf of test-user.\n\nPR #32: Plain PR\nhttps://github.com/myorg/goliac/pull/32\n\nbecause"
			if mode != "" {
				expectedReview += fmt.Sprintf("\n\nMerge mode `%s` ignored: this PR is not part of a stack.", mode)
			}
			assert.Equal(t, expectedReview, reviews[0].Body["body"])

			merges := gh.find("PUT", "/pulls/32/merge")
			assert.Equal(t, 1, len(merges))
			assert.Equal(t, "merge", merges[0].Body["merge_method"])
			assert.Equal(t, "Plain PR (force merged PR 32 by test-user)", merges[0].Body["commit_title"])
			assert.Equal(t, "Force merge via Goliac on behalf of test-user.\n\nPR: https://github.com/myorg/goliac/pull/32\nTitle: Plain PR", merges[0].Body["commit_message"])
			assert.Equal(t, 0, len(gh.find("PUT", "/merge-async")))
			assert.Equal(t, 0, len(gh.find("POST", "/merges")))
			assert.Equal(t, 0, len(gh.find("PATCH", "")))
		})
	}

	t.Run("/squash", func(t *testing.T) {
		gh := &fakeGitHub{prs: fixturePlainPR(), syncMerge: okJSON(`{"merged":true}`)}
		fc := newForcemergeForTest(t, gh)

		err := fc.mergePR(context.Background(), "test-user", "goliac", "32", prURL(32), "because /squash", "")
		assert.Nil(t, err)
		merges := gh.find("PUT", "/pulls/32/merge")
		assert.Equal(t, 1, len(merges))
		assert.Equal(t, "squash", merges[0].Body["merge_method"])
		assert.NotContains(t, gh.find("POST", "/reviews")[0].Body["body"], "/squash")
	})

	t.Run("squash fallback on Method Not Allowed", func(t *testing.T) {
		gh := &fakeGitHub{prs: fixturePlainPR(), syncMerge: func(call restCall) ([]byte, error) {
			if call.Body["merge_method"] == "merge" {
				return httpError("405 Method Not Allowed", `{"message":"Merge commits are not allowed on this repository."}`)
			}
			return []byte(`{"merged":true}`), nil
		}}
		fc := newForcemergeForTest(t, gh)

		err := fc.mergePR(context.Background(), "test-user", "goliac", "32", prURL(32), "because", "")
		assert.Nil(t, err)
		merges := gh.find("PUT", "/pulls/32/merge")
		assert.Equal(t, 2, len(merges))
		assert.Equal(t, "merge", merges[0].Body["merge_method"])
		assert.Equal(t, "squash", merges[1].Body["merge_method"])
	})

	t.Run("stacked refusal without stack info merges nothing", func(t *testing.T) {
		gh := &fakeGitHub{prs: fixturePlainPR(), syncMerge: func(call restCall) ([]byte, error) {
			return httpError("403 Forbidden", stackedRefusal)
		}}
		fc := newForcemergeForTest(t, gh)

		err := fc.mergePR(context.Background(), "test-user", "goliac", "32", prURL(32), "because", "")
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "no stack information")
		assert.Equal(t, 0, len(gh.find("PUT", "/merge-async")))
		assert.Equal(t, 0, len(gh.find("POST", "/merges")))
	})
}

func TestForcemergeMergePRDirect(t *testing.T) {
	for _, mode := range []string{"", MergeModeDirect} {
		t.Run(fmt.Sprintf("mid-stack PR 3, mode %q: merged into 4's branch, PR 2 restacked onto it", mode), func(t *testing.T) {
			gh := &fakeGitHub{prs: fixtureStack(), branchMerge: okJSON(`{"sha":"mergecommit"}`), markMergedOnBranchMerge: true}
			fc := newForcemergeForTest(t, gh)

			err := fc.mergePR(context.Background(), "test-user", "goliac", "3", prURL(3), "because", mode)
			assert.Nil(t, err)

			merges := gh.find("POST", "/repos/myorg/goliac/merges")
			assert.Equal(t, 1, len(merges))
			assert.Equal(t, "b4", merges[0].Body["base"])
			assert.Equal(t, "sha3", merges[0].Body["head"])
			assert.Equal(t, "PR 3 (force merged PR 3 by test-user)\n\nForce merge via Goliac on behalf of test-user.\n\nPR: https://github.com/myorg/goliac/pull/3\nTitle: PR 3", merges[0].Body["commit_message"])

			assert.Equal(t, []string{"3"}, gh.reviewedPRs())
			review := gh.find("POST", "/reviews")[0].Body["body"].(string)
			assert.True(t, strings.HasPrefix(review, "Force merge via Goliac on behalf of test-user.\n\nPR #3: PR 3\nhttps://github.com/myorg/goliac/pull/3\n\nbecause"))
			assert.Contains(t, review, "merged alone into its base branch `b4`; the PRs based on `b3` are retargeted to `b4`.")

			// restack: 1 > 2 > 4 > main
			patches := gh.find("PATCH", "/pulls/2")
			assert.Equal(t, 1, len(patches))
			assert.Equal(t, "b4", patches[0].Body["base"])
			assert.Equal(t, "b4", gh.prs[2].BaseRef)
			assert.Equal(t, "b2", gh.prs[1].BaseRef)
			assert.Equal(t, "main", gh.prs[4].BaseRef)

			// GitHub marked it merged: no close, no comment
			assert.True(t, gh.prs[3].Merged)
			assert.Equal(t, 0, len(gh.find("PATCH", "/pulls/3")))
			assert.Equal(t, 0, len(gh.find("POST", "/comments")))
			assert.Equal(t, 0, len(gh.find("PUT", "/merge-async")))
			assert.Equal(t, 0, len(gh.find("PUT", "/merge")))
		})
	}

	t.Run("mid-stack PR 3 not marked merged by GitHub: closed with a comment", func(t *testing.T) {
		gh := &fakeGitHub{prs: fixtureStack(), branchMerge: okJSON(`{"sha":"mergecommit"}`)}
		fc := newForcemergeForTest(t, gh)

		err := fc.mergePR(context.Background(), "test-user", "goliac", "3", prURL(3), "because", MergeModeDirect)
		assert.Nil(t, err)

		comments := gh.find("POST", "/issues/3/comments")
		assert.Equal(t, 1, len(comments))
		assert.Equal(t, "Force merged via Goliac on behalf of test-user into its base branch `b4` (merge commit mergecommit). GitHub did not mark this PR as merged, so Goliac closes it.", comments[0].Body["body"])
		closes := gh.find("PATCH", "/pulls/3")
		assert.Equal(t, 1, len(closes))
		assert.Equal(t, "closed", closes[0].Body["state"])
		assert.Equal(t, "closed", gh.prs[3].State)
		// restack still done
		assert.Equal(t, "b4", gh.prs[2].BaseRef)
	})

	t.Run("mid-stack PR 3 with /squash: merge commit, noted in the review", func(t *testing.T) {
		gh := &fakeGitHub{prs: fixtureStack(), branchMerge: okJSON(`{"sha":"mergecommit"}`), markMergedOnBranchMerge: true}
		fc := newForcemergeForTest(t, gh)

		err := fc.mergePR(context.Background(), "test-user", "goliac", "3", prURL(3), "because /squash", "")
		assert.Nil(t, err)
		assert.Equal(t, 1, len(gh.find("POST", "/repos/myorg/goliac/merges")))
		review := gh.find("POST", "/reviews")[0].Body["body"].(string)
		assert.Contains(t, review, "/squash is not available when merging into a parent branch: a merge commit is used instead.")
		assert.NotContains(t, review, "because /squash")
	})

	t.Run("bottom PR 4: merged alone with merge-async, PR 3 restacked onto main", func(t *testing.T) {
		polls := 0
		gh := &fakeGitHub{
			prs:        fixtureStack(),
			asyncMerge: okJSON(asyncPending),
			asyncPoll: func(call restCall) ([]byte, error) {
				polls++
				if polls < 3 {
					return []byte(asyncPending), nil
				}
				return []byte(asyncMerged), nil
			},
		}
		fc := newForcemergeForTest(t, gh)

		err := fc.mergePR(context.Background(), "test-user", "goliac", "4", prURL(4), "because /squash", MergeModeDirect)
		assert.Nil(t, err)
		assert.Equal(t, 3, polls)
		assert.Equal(t, 3, len(gh.find("GET", "/pulls/4/merge-async/uuid-1")))

		asyncCalls := gh.find("PUT", "/pulls/4/merge-async")
		assert.Equal(t, 1, len(asyncCalls))
		assert.Equal(t, "squash", asyncCalls[0].Body["merge_method"])
		assert.Equal(t, "direct_merge", asyncCalls[0].Body["merge_action"])
		assert.Equal(t, true, asyncCalls[0].Body["bypass_rules"])
		assert.Equal(t, "sha4", asyncCalls[0].Body["sha"])
		assert.Equal(t, "PR 4 (force merged PR 4 by test-user)", asyncCalls[0].Body["commit_title"])
		assert.Equal(t, "Force merge via Goliac on behalf of test-user.\n\nPR: https://github.com/myorg/goliac/pull/4\nTitle: PR 4", asyncCalls[0].Body["commit_message"])

		assert.Equal(t, []string{"4"}, gh.reviewedPRs())
		assert.NotContains(t, gh.find("POST", "/reviews")[0].Body["body"], "merge commit is used instead")
		assert.Equal(t, "main", gh.prs[3].BaseRef)
		assert.Equal(t, 1, len(gh.find("PATCH", "/pulls/3")))
		assert.Equal(t, 0, len(gh.find("POST", "/merges")))
		assert.Equal(t, 0, len(gh.find("PUT", "/pulls/4/merge")))
		assert.Equal(t, 0, len(gh.find("POST", "/comments")))
	})

	t.Run("top PR 1: merged into 2's branch, nothing to restack", func(t *testing.T) {
		gh := &fakeGitHub{prs: fixtureStack(), branchMerge: okJSON(`{"sha":"mergecommit"}`), markMergedOnBranchMerge: true}
		fc := newForcemergeForTest(t, gh)

		err := fc.mergePR(context.Background(), "test-user", "goliac", "1", prURL(1), "because", "")
		assert.Nil(t, err)
		merges := gh.find("POST", "/repos/myorg/goliac/merges")
		assert.Equal(t, 1, len(merges))
		assert.Equal(t, "b2", merges[0].Body["base"])
		assert.Equal(t, "sha1", merges[0].Body["head"])
		assert.Equal(t, 0, len(gh.find("PATCH", "")))
		assert.Equal(t, "b3", gh.prs[2].BaseRef)
	})

	t.Run("mid-stack PR 3: merge conflict with its base branch, nothing restacked", func(t *testing.T) {
		gh := &fakeGitHub{prs: fixtureStack(), branchMerge: func(call restCall) ([]byte, error) {
			return httpError("409 Conflict", `{"message":"Merge conflict"}`)
		}}
		fc := newForcemergeForTest(t, gh)

		err := fc.mergePR(context.Background(), "test-user", "goliac", "3", prURL(3), "because", "")
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "merge conflict")
		assert.Contains(t, err.Error(), "b4")
		assert.Equal(t, 0, len(gh.find("PATCH", "")))
		assert.Equal(t, "b3", gh.prs[2].BaseRef)
	})

	t.Run("bottom PR 4: asynchronous merge fails, nothing restacked", func(t *testing.T) {
		gh := &fakeGitHub{
			prs:        fixtureStack(),
			asyncMerge: okJSON(asyncPending),
			asyncPoll:  okJSON(`{"status":"failed","details":{"message":"Merge conflict between base and head"}}`),
		}
		fc := newForcemergeForTest(t, gh)

		err := fc.mergePR(context.Background(), "test-user", "goliac", "4", prURL(4), "because", "")
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "GitHub refused the asynchronous merge (request uuid-1): Merge conflict between base and head")
		assert.Equal(t, 0, len(gh.find("PATCH", "")))
	})
}

func TestForcemergeMergePRStack(t *testing.T) {
	for _, from := range []int{3, 1, 4} {
		t.Run(fmt.Sprintf("stack requested on PR %d: every PR approved, merge-async on the top PR 1", from), func(t *testing.T) {
			gh := &fakeGitHub{prs: fixtureStack(), asyncMerge: okJSON(asyncMerged)}
			fc := newForcemergeForTest(t, gh)

			err := fc.mergePR(context.Background(), "test-user", "goliac", strconv.Itoa(from), prURL(from), "ship it /squash", MergeModeStack)
			assert.Nil(t, err)

			assert.Equal(t, []string{"1", "2", "3", "4"}, gh.reviewedPRs())
			for i, r := range gh.find("POST", "/reviews") {
				n := i + 1
				assert.Equal(t, "APPROVE", r.Body["event"])
				assert.Equal(t,
					fmt.Sprintf("Force merge via Goliac on behalf of test-user.\n\nPR #%d: PR %d\nhttps://github.com/myorg/goliac/pull/%d\n\nship it \n\nMerging the whole stack (4 PRs, #1 to #4) into `main`, requested on PR #%d.", n, n, n, from),
					r.Body["body"])
			}

			asyncCalls := gh.find("PUT", "/merge-async")
			assert.Equal(t, 1, len(asyncCalls))
			assert.Equal(t, "/repos/myorg/goliac/pulls/1/merge-async", asyncCalls[0].Endpoint)
			assert.Equal(t, "sha1", asyncCalls[0].Body["sha"])
			assert.Equal(t, "squash", asyncCalls[0].Body["merge_method"])
			assert.Equal(t, "direct_merge", asyncCalls[0].Body["merge_action"])
			assert.Equal(t, true, asyncCalls[0].Body["bypass_rules"])
			assert.Equal(t, "PR 1 (force merged PR 1 by test-user)", asyncCalls[0].Body["commit_title"])

			assert.Equal(t, 0, len(gh.find("POST", "/merges")))
			assert.Equal(t, 0, len(gh.find("PATCH", "")))
			assert.Equal(t, 0, len(gh.find("PUT", "/merge")))
		})
	}

	t.Run("a closed PR in the stack (above): nothing approved or merged", func(t *testing.T) {
		prs := fixtureStack()
		prs[2].State = "closed"
		gh := &fakeGitHub{prs: prs, asyncMerge: okJSON(asyncMerged)}
		fc := newForcemergeForTest(t, gh)

		err := fc.mergePR(context.Background(), "test-user", "goliac", "3", prURL(3), "ship it", MergeModeStack)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "nothing merged: GitHub reports a stack of 4 PRs but only 2 open PRs were found (#3 > #4)")
		assert.Equal(t, 0, len(gh.find("POST", "/reviews")))
		assert.Equal(t, 0, len(gh.find("PUT", "/merge-async")))
	})

	t.Run("a missing PR in the stack (below): nothing approved or merged", func(t *testing.T) {
		prs := fixtureStack()
		delete(prs, 4)
		gh := &fakeGitHub{prs: prs, asyncMerge: okJSON(asyncMerged)}
		fc := newForcemergeForTest(t, gh)

		err := fc.mergePR(context.Background(), "test-user", "goliac", "2", prURL(2), "ship it", MergeModeStack)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "nothing merged: no open PR found for branch b4 below PR #3")
		assert.Equal(t, 0, len(gh.find("POST", "/reviews")))
		assert.Equal(t, 0, len(gh.find("PUT", "/merge-async")))
	})

	t.Run("asynchronous merge times out", func(t *testing.T) {
		gh := &fakeGitHub{prs: fixtureStack(), asyncMerge: okJSON(asyncPending), asyncPoll: okJSON(asyncPending)}
		fc := newForcemergeForTest(t, gh)
		fc.asyncPollTimeout = 20 * time.Millisecond

		err := fc.mergePR(context.Background(), "test-user", "goliac", "3", prURL(3), "ship it", MergeModeStack)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "timeout after 20ms waiting for the asynchronous merge (request uuid-1)")
	})

	t.Run("asynchronous merge fails", func(t *testing.T) {
		gh := &fakeGitHub{
			prs:        fixtureStack(),
			asyncMerge: okJSON(asyncPending),
			asyncPoll:  okJSON(`{"status":"failed","details":{"message":"Required status check is failing"}}`),
		}
		fc := newForcemergeForTest(t, gh)

		err := fc.mergePR(context.Background(), "test-user", "goliac", "3", prURL(3), "ship it", MergeModeStack)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "GitHub refused the asynchronous merge (request uuid-1): Required status check is failing")
	})
}

func TestForcemergeAsyncMerge(t *testing.T) {
	// asynchronous merge of the bottom PR 4 of the stack
	run := func(t *testing.T, gh *fakeGitHub) error {
		gh.prs = fixtureStack()
		fc := newForcemergeForTest(t, gh)
		return fc.mergePR(context.Background(), "test-user", "goliac", "4", prURL(4), "because", "")
	}

	t.Run("timeout while still pending", func(t *testing.T) {
		gh := &fakeGitHub{prs: fixtureStack(), asyncMerge: okJSON(asyncPending), asyncPoll: okJSON(asyncPending)}
		fc := newForcemergeForTest(t, gh)
		fc.asyncPollTimeout = 20 * time.Millisecond
		err := fc.mergePR(context.Background(), "test-user", "goliac", "4", prURL(4), "because", "")
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "timeout after 20ms waiting for the asynchronous merge (request uuid-1)")
	})

	t.Run("PR not ready (400)", func(t *testing.T) {
		err := run(t, &fakeGitHub{asyncMerge: func(call restCall) ([]byte, error) {
			return httpError("400 Bad Request", `{"message":"Pull request is in draft state"}`)
		}})
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "not ready to be merged")
		assert.Contains(t, err.Error(), "draft")
	})

	t.Run("an existing request (409) is followed", func(t *testing.T) {
		err := run(t, &fakeGitHub{
			asyncMerge: func(call restCall) ([]byte, error) { return httpError("409 Conflict", asyncPending) },
			asyncPoll:  okJSON(asyncMerged),
		})
		assert.Nil(t, err)
	})

	t.Run("already merged (200 merged) needs no poll", func(t *testing.T) {
		gh := &fakeGitHub{asyncMerge: okJSON(asyncMerged)}
		err := run(t, gh)
		assert.Nil(t, err)
		assert.Equal(t, 0, len(gh.find("GET", "/merge-async/uuid-1")))
	})

	t.Run("transient poll error then merged", func(t *testing.T) {
		polls := 0
		err := run(t, &fakeGitHub{
			asyncMerge: okJSON(asyncPending),
			asyncPoll: func(call restCall) ([]byte, error) {
				polls++
				if polls == 1 {
					return httpError("502 Bad Gateway", `{}`)
				}
				return []byte(asyncMerged), nil
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 2, polls)
	})

	t.Run("expired request (404)", func(t *testing.T) {
		err := run(t, &fakeGitHub{
			asyncMerge: okJSON(asyncPending),
			asyncPoll:  func(call restCall) ([]byte, error) { return httpError("404 Not Found", `{"message":"Not Found"}`) },
		})
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "not found (expired?)")
	})
}

func TestForcemergeExecuteWorkflowMergeMode(t *testing.T) {
	t.Run("unknown merge mode is refused before any step", func(t *testing.T) {
		gh := &fakeGitHub{prs: fixturePlainPR()}
		fc := newForcemergeForTest(t, gh)

		_, err := fc.ExecuteWorkflow(context.Background(), []string{"fmtest"}, "test-user", "fmtest", "because",
			map[string]string{"pr_url": prURL(32), PropertyMergeMode: "sideways"}, false)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), `unknown merge mode "sideways"`)
		assert.Equal(t, 0, len(gh.calls))
	})
}
