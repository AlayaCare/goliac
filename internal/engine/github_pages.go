package engine

import (
	"fmt"
	"strings"

	"github.com/goliac-project/goliac/internal/entity"
)

// GithubPagesComparable is the reconciled shape for GitHub Pages (see entity.RepositoryGithubPages).
type GithubPagesComparable struct {
	Visibility    string // public | private
	Source        string // branch | workflow
	Branch        string
	Path          string
	Cname         string // custom domain (GitHub cname); empty means none
	HttpsEnforced bool   // GitHub https_enforced
}

// GithubPagesRemote is the subset of GET /repos/{owner}/{repo}/pages JSON we need.
type GithubPagesRemote struct {
	HTMLURL       string `json:"html_url"`
	BuildType     string `json:"build_type"`
	Cname         string `json:"cname"`
	HttpsEnforced bool   `json:"https_enforced"`
	Source        *struct {
		Branch string `json:"branch"`
		Path   string `json:"path"`
	} `json:"source"`
	Public bool `json:"public"`
}

func githubPagesComparableEqual(a, b *GithubPagesComparable) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Visibility != b.Visibility || a.Source != b.Source || a.Branch != b.Branch || a.Path != b.Path {
		return false
	}
	if strings.TrimSpace(a.Cname) != strings.TrimSpace(b.Cname) {
		return false
	}
	if strings.TrimSpace(a.Cname) == "" {
		return true
	}
	return a.HttpsEnforced == b.HttpsEnforced
}

func cloneGithubPagesComparable(p *GithubPagesComparable) *GithubPagesComparable {
	if p == nil {
		return nil
	}
	c := *p
	return &c
}

func entityGithubPagesToComparable(p *entity.RepositoryGithubPages) *GithubPagesComparable {
	if p == nil {
		return nil
	}
	c := &GithubPagesComparable{
		Visibility: p.Visibility,
		Source:     p.Source,
		Branch:     p.Branch,
		Path:       p.Path,
		Cname:      strings.TrimSpace(p.CustomDomain),
	}
	if strings.TrimSpace(p.CustomDomain) != "" {
		c.HttpsEnforced = p.EnforceHTTPSEffective()
	}
	return c
}

func githubPagesRemoteToComparable(r *GithubPagesRemote) *GithubPagesComparable {
	if r == nil {
		return nil
	}
	c := &GithubPagesComparable{
		Cname:         strings.TrimSpace(r.Cname),
		HttpsEnforced: r.HttpsEnforced,
	}
	if r.BuildType == "workflow" {
		c.Source = "workflow"
	} else {
		c.Source = "branch"
		if r.Source != nil {
			c.Branch = r.Source.Branch
			c.Path = r.Source.Path
			if c.Path == "" {
				c.Path = "/"
			}
		}
	}
	if r.Public {
		c.Visibility = "public"
	} else {
		c.Visibility = "private"
	}
	return c
}

// EntityGithubPagesFromRemote maps GET /repos/{owner}/{repo}/pages JSON into entity.RepositoryGithubPages
// (e.g. for scaffold). Returns nil when there is no site, or when branch-based publishing would be
// invalid without a branch (would fail entity validation).
func EntityGithubPagesFromRemote(r *GithubPagesRemote) *entity.RepositoryGithubPages {
	if r == nil {
		return nil
	}
	c := githubPagesRemoteToComparable(r)
	if c.Source == "branch" && strings.TrimSpace(c.Branch) == "" {
		return nil
	}
	gp := &entity.RepositoryGithubPages{
		Visibility:   c.Visibility,
		Source:       c.Source,
		Branch:       c.Branch,
		Path:         c.Path,
		CustomDomain: c.Cname,
	}
	if strings.TrimSpace(c.Cname) != "" {
		if r.HttpsEnforced {
			gp.EnforceHTTPS = nil
		} else {
			f := false
			gp.EnforceHTTPS = &f
		}
	}
	return gp
}

// githubPagesComparableToRESTPostBody is the JSON body for POST /repos/{owner}/{repo}/pages (enable site).
func githubPagesComparableToRESTPostBody(p *GithubPagesComparable) map[string]interface{} {
	return githubPagesComparableToRESTPostBodyWithPublic(p, true)
}

// githubPagesComparableToRESTPostBodyWithPublic builds the POST body, optionally omitting the public field.
// GitHub rejects public: true on repositories where private Pages access control is unavailable.
func githubPagesComparableToRESTPostBodyWithPublic(p *GithubPagesComparable, includePublic bool) map[string]interface{} {
	body := map[string]interface{}{}
	if includePublic {
		body["public"] = p.Visibility == "public"
	}
	if p.Source == "workflow" {
		body["build_type"] = "workflow"
		return body
	}
	body["build_type"] = "legacy"
	body["source"] = map[string]interface{}{
		"branch": p.Branch,
		"path":   p.Path,
	}
	return body
}

// githubPagesComparableToRESTPutBody is the JSON body for PUT /repos/{owner}/{repo}/pages (update site).
func githubPagesComparableToRESTPutBody(p *GithubPagesComparable) map[string]interface{} {
	return githubPagesComparableToRESTPutBodyWithPublic(p, true)
}

// githubPagesComparableToRESTPutBodyWithPublic builds the PUT body, optionally omitting the public field.
func githubPagesComparableToRESTPutBodyWithPublic(p *GithubPagesComparable, includePublic bool) map[string]interface{} {
	body := githubPagesComparableToRESTPostBodyWithPublic(p, includePublic)
	if strings.TrimSpace(p.Cname) != "" {
		body["cname"] = strings.TrimSpace(p.Cname)
		body["https_enforced"] = p.HttpsEnforced
	}
	return body
}

// githubPagesPrivatePagesUnavailable reports whether GitHub rejected a Pages request because
// private Pages access control is not enabled for the repository.
func githubPagesPrivatePagesUnavailable(responseBody []byte) bool {
	return strings.Contains(string(responseBody), "Private pages is not enabled")
}

// githubPagesAPIError formats a Pages API failure, including GitHub's response body when present.
func githubPagesAPIError(err error, responseBody []byte) error {
	if err == nil {
		return nil
	}
	if len(responseBody) == 0 {
		return err
	}
	return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(responseBody)))
}

func githubPagesNeedsFollowUpPutAfterCreate(p *GithubPagesComparable) bool {
	if p == nil {
		return false
	}
	return strings.TrimSpace(p.Cname) != ""
}

// GithubPagesRemoteFromComparable builds an in-memory Pages snapshot for dry-run reconciliation.
func GithubPagesRemoteFromComparable(pages *GithubPagesComparable) *GithubPagesRemote {
	if pages == nil {
		return nil
	}
	buildType := "legacy"
	if pages.Source == "workflow" {
		buildType = "workflow"
	}
	r := &GithubPagesRemote{
		HTMLURL:       "",
		BuildType:     buildType,
		Public:        pages.Visibility == "public",
		Cname:         strings.TrimSpace(pages.Cname),
		HttpsEnforced: pages.HttpsEnforced,
	}
	if pages.Source == "branch" {
		r.Source = &struct {
			Branch string `json:"branch"`
			Path   string `json:"path"`
		}{Branch: pages.Branch, Path: pages.Path}
	}
	return r
}
