package engine

import (
	"testing"

	"github.com/goliac-project/goliac/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGithubPagesComparableEqual(t *testing.T) {
	assert.True(t, githubPagesComparableEqual(nil, nil))
	a := &GithubPagesComparable{Visibility: "public", Source: "branch", Branch: "main", Path: "/"}
	b := &GithubPagesComparable{Visibility: "public", Source: "branch", Branch: "main", Path: "/"}
	assert.True(t, githubPagesComparableEqual(a, b))
	assert.False(t, githubPagesComparableEqual(a, nil))
	c := &GithubPagesComparable{Visibility: "private", Source: "branch", Branch: "main", Path: "/"}
	assert.False(t, githubPagesComparableEqual(a, c))
	d := &GithubPagesComparable{Visibility: "public", Source: "branch", Branch: "main", Path: "/", Cname: "x.com"}
	assert.False(t, githubPagesComparableEqual(a, d))
	e := &GithubPagesComparable{Visibility: "public", Source: "branch", Branch: "main", Path: "/", Cname: "x.com", HttpsEnforced: true}
	f := &GithubPagesComparable{Visibility: "public", Source: "branch", Branch: "main", Path: "/", Cname: "x.com", HttpsEnforced: true}
	assert.True(t, githubPagesComparableEqual(e, f))
}

func TestEntityGithubPagesToComparable(t *testing.T) {
	assert.Nil(t, entityGithubPagesToComparable(nil))
	gp := &entity.RepositoryGithubPages{Visibility: "private", Source: "workflow"}
	c := entityGithubPagesToComparable(gp)
	assert.Equal(t, "private", c.Visibility)
	assert.Equal(t, "workflow", c.Source)

	gp2 := &entity.RepositoryGithubPages{Visibility: "public", Source: "branch", Branch: "main", Path: "/", CustomDomain: " docs.example.com "}
	enTrue := true
	gp2.EnforceHTTPS = &enTrue
	c2 := entityGithubPagesToComparable(gp2)
	assert.Equal(t, "docs.example.com", c2.Cname)
	assert.True(t, c2.HttpsEnforced)

	gpOmit := &entity.RepositoryGithubPages{Visibility: "public", Source: "branch", Branch: "main", Path: "/"}
	c3 := entityGithubPagesToComparable(gpOmit)
	assert.False(t, c3.HttpsEnforced)
}

func TestGithubPagesRemoteToComparable(t *testing.T) {
	assert.Nil(t, githubPagesRemoteToComparable(nil))
	r := &GithubPagesRemote{BuildType: "workflow", Public: false}
	c := githubPagesRemoteToComparable(r)
	assert.Equal(t, "workflow", c.Source)
	assert.Equal(t, "private", c.Visibility)

	r2 := &GithubPagesRemote{
		BuildType: "legacy",
		Public:    true,
		Cname:     "blog.example.com",
		Source: &struct {
			Branch string `json:"branch"`
			Path   string `json:"path"`
		}{Branch: "gh-pages", Path: "/docs"},
		HttpsEnforced: true,
	}
	c2 := githubPagesRemoteToComparable(r2)
	assert.Equal(t, "branch", c2.Source)
	assert.Equal(t, "gh-pages", c2.Branch)
	assert.Equal(t, "/docs", c2.Path)
	assert.Equal(t, "public", c2.Visibility)
	assert.Equal(t, "blog.example.com", c2.Cname)
	assert.True(t, c2.HttpsEnforced)
}

func TestEntityGithubPagesFromRemote(t *testing.T) {
	assert.Nil(t, EntityGithubPagesFromRemote(nil))

	wf := &GithubPagesRemote{BuildType: "workflow", Public: true, HttpsEnforced: true}
	gp := EntityGithubPagesFromRemote(wf)
	require.NotNil(t, gp)
	assert.Equal(t, "public", gp.Visibility)
	assert.Equal(t, "workflow", gp.Source)
	assert.Equal(t, "", gp.Branch)
	assert.Equal(t, "", gp.Path)
	assert.Nil(t, gp.EnforceHTTPS)

	legacy := &GithubPagesRemote{
		BuildType:     "legacy",
		Public:        false,
		Cname:         "c.example.org",
		HttpsEnforced: true,
		Source: &struct {
			Branch string `json:"branch"`
			Path   string `json:"path"`
		}{Branch: "main", Path: "/docs"},
	}
	gp2 := EntityGithubPagesFromRemote(legacy)
	require.NotNil(t, gp2)
	assert.Equal(t, "private", gp2.Visibility)
	assert.Equal(t, "branch", gp2.Source)
	assert.Equal(t, "main", gp2.Branch)
	assert.Equal(t, "/docs", gp2.Path)
	assert.Equal(t, "c.example.org", gp2.CustomDomain)
	assert.Nil(t, gp2.EnforceHTTPS)
	assert.True(t, gp2.EnforceHTTPSEffective())

	legacyNoHTTPS := &GithubPagesRemote{
		BuildType:     "legacy",
		Public:        true,
		Cname:         "",
		HttpsEnforced: false,
		Source: &struct {
			Branch string `json:"branch"`
			Path   string `json:"path"`
		}{Branch: "main", Path: "/"},
	}
	gpNoDomain := EntityGithubPagesFromRemote(legacyNoHTTPS)
	require.NotNil(t, gpNoDomain)
	assert.Nil(t, gpNoDomain.EnforceHTTPS)

	legacyNoHTTPSCustom := &GithubPagesRemote{
		BuildType:     "legacy",
		Public:        true,
		Cname:         "old.example.org",
		HttpsEnforced: false,
		Source: &struct {
			Branch string `json:"branch"`
			Path   string `json:"path"`
		}{Branch: "main", Path: "/"},
	}
	gp3 := EntityGithubPagesFromRemote(legacyNoHTTPSCustom)
	require.NotNil(t, gp3)
	require.NotNil(t, gp3.EnforceHTTPS)
	assert.False(t, *gp3.EnforceHTTPS)

	noBranch := &GithubPagesRemote{BuildType: "legacy", Public: true, Source: nil}
	assert.Nil(t, EntityGithubPagesFromRemote(noBranch))
}

func TestGithubPagesComparableToRESTPostBody(t *testing.T) {
	wf := &GithubPagesComparable{Visibility: "private", Source: "workflow"}
	body := githubPagesComparableToRESTPostBody(wf)
	assert.Equal(t, "workflow", body["build_type"])
	assert.Equal(t, false, body["public"])
	_, hasCname := body["cname"]
	assert.False(t, hasCname)

	br := &GithubPagesComparable{Visibility: "public", Source: "branch", Branch: "main", Path: "/"}
	body2 := githubPagesComparableToRESTPostBody(br)
	assert.Equal(t, "legacy", body2["build_type"])
	assert.Equal(t, true, body2["public"])
	src := body2["source"].(map[string]interface{})
	assert.Equal(t, "main", src["branch"])
	assert.Equal(t, "/", src["path"])
}

func TestGithubPagesComparableToRESTPostBodyWithoutPublic(t *testing.T) {
	wf := &GithubPagesComparable{Visibility: "public", Source: "workflow"}
	body := githubPagesComparableToRESTPostBodyWithPublic(wf, false)
	assert.Equal(t, "workflow", body["build_type"])
	_, hasPublic := body["public"]
	assert.False(t, hasPublic)

	br := &GithubPagesComparable{Visibility: "public", Source: "branch", Branch: "main", Path: "/docs"}
	body2 := githubPagesComparableToRESTPutBodyWithPublic(br, false)
	assert.Equal(t, "legacy", body2["build_type"])
	_, hasPublic2 := body2["public"]
	assert.False(t, hasPublic2)
	src := body2["source"].(map[string]interface{})
	assert.Equal(t, "main", src["branch"])
	assert.Equal(t, "/docs", src["path"])
}

func TestGithubPagesPrivatePagesUnavailable(t *testing.T) {
	responseBody := []byte(`{"message":"Private pages is not enabled for this repository. All Pages will be public.","status":"400"}`)
	assert.True(t, githubPagesPrivatePagesUnavailable(responseBody))
	assert.False(t, githubPagesPrivatePagesUnavailable([]byte(`{"message":"some other error"}`)))
}

func TestGithubPagesAPIError(t *testing.T) {
	baseErr := assert.AnError
	assert.Nil(t, githubPagesAPIError(nil, nil))
	assert.Equal(t, baseErr, githubPagesAPIError(baseErr, nil))
	wrapped := githubPagesAPIError(baseErr, []byte(`{"message":"Private pages is not enabled"}`))
	assert.ErrorContains(t, wrapped, "Private pages is not enabled")
}

func TestGithubPagesComparableToRESTPutBody(t *testing.T) {
	br := &GithubPagesComparable{Visibility: "public", Source: "branch", Branch: "main", Path: "/", Cname: "docs.example.com", HttpsEnforced: true}
	body := githubPagesComparableToRESTPutBody(br)
	assert.Equal(t, "docs.example.com", body["cname"])
	assert.Equal(t, true, body["https_enforced"])

	noDomain := &GithubPagesComparable{Visibility: "public", Source: "branch", Branch: "main", Path: "/"}
	body2 := githubPagesComparableToRESTPutBody(noDomain)
	_, hasCname := body2["cname"]
	assert.False(t, hasCname)
	_, hasHTTPS := body2["https_enforced"]
	assert.False(t, hasHTTPS)
}

func TestGithubPagesNeedsFollowUpPutAfterCreate(t *testing.T) {
	assert.False(t, githubPagesNeedsFollowUpPutAfterCreate(nil))
	assert.False(t, githubPagesNeedsFollowUpPutAfterCreate(&GithubPagesComparable{Source: "branch", Branch: "main", Path: "/"}))
	assert.True(t, githubPagesNeedsFollowUpPutAfterCreate(&GithubPagesComparable{Source: "branch", Branch: "main", Path: "/", Cname: "x.com"}))
	assert.False(t, githubPagesNeedsFollowUpPutAfterCreate(&GithubPagesComparable{Source: "workflow", HttpsEnforced: true}))
}

func TestGithubPagesRemoteFromComparable(t *testing.T) {
	assert.Nil(t, GithubPagesRemoteFromComparable(nil))
	r := GithubPagesRemoteFromComparable(&GithubPagesComparable{
		Visibility: "private", Source: "branch", Branch: "dev", Path: "/docs",
		Cname: "c.dev", HttpsEnforced: true,
	})
	require.NotNil(t, r)
	assert.Equal(t, "legacy", r.BuildType)
	assert.False(t, r.Public)
	assert.Equal(t, "c.dev", r.Cname)
	assert.True(t, r.HttpsEnforced)
	require.NotNil(t, r.Source)
	assert.Equal(t, "dev", r.Source.Branch)
	assert.Equal(t, "/docs", r.Source.Path)
}
