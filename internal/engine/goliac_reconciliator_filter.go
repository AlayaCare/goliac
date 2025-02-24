package engine

import (
	"regexp"

	"github.com/goliac-project/goliac/internal/config"
)

type ReconciliatorFilter interface {
	RepositoryFilter(reponame string, repo *GithubRepoComparable) *GithubRepoComparable
}

type ReconciliatorFilterImpl struct {
	isEnterprise                   bool
	ForbidPublicRepositories       bool
	ForbidPulicRepostitoriesExcept []*regexp.Regexp
}

func NewReconciliatorFilter(isEnterprise bool, config *config.RepositoryConfig) *ReconciliatorFilterImpl {
	exclude := []*regexp.Regexp{}
	if config != nil {
		for _, pattern := range config.VisibilityRules.ForbidPublicRepositoriesExclusions {
			exclude = append(exclude, regexp.MustCompile("^"+pattern+"$"))
		}
	}

	return &ReconciliatorFilterImpl{
		isEnterprise:                   isEnterprise,
		ForbidPublicRepositories:       config.VisibilityRules.ForbidPublicRepositories,
		ForbidPulicRepostitoriesExcept: exclude,
	}
}

func (r *ReconciliatorFilterImpl) RepositoryFilter(reponame string, repo *GithubRepoComparable) *GithubRepoComparable {
	if !r.isEnterprise {
		if repo.Visibility == "internal" {
			repo.Visibility = "private"
		}
	}

	if r.ForbidPublicRepositories {
		for _, exclude := range r.ForbidPulicRepostitoriesExcept {
			if exclude.MatchString(reponame) {
				return repo
			}
		}
		if repo.Visibility == "public" {
			repo.Visibility = "private"
		}
	}
	return repo
}
