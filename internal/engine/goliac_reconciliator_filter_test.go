package engine

import (
	"testing"

	"github.com/goliac-project/goliac/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestReconciliatorFilterImpl_RepositoryFilter(t *testing.T) {
	t.Run("happy path: public allowed", func(t *testing.T) {
		config := &config.RepositoryConfig{}

		filter := NewReconciliatorFilter(true, config)
		repo := &GithubRepoComparable{
			Visibility: "internal",
		}
		repo = filter.RepositoryFilter("repo", repo)
		assert.Equal(t, "internal", repo.Visibility)

		repo = &GithubRepoComparable{
			Visibility: "public",
		}

		repo = filter.RepositoryFilter("repo", repo)
		assert.Equal(t, "public", repo.Visibility)
	})

	t.Run("happy path: public forbidden", func(t *testing.T) {
		config := &config.RepositoryConfig{}
		config.VisibilityRules.ForbidPublicRepositories = true

		filter := NewReconciliatorFilter(true, config)
		repo := &GithubRepoComparable{
			Visibility: "internal",
		}
		repo = filter.RepositoryFilter("repo", repo)
		assert.Equal(t, "internal", repo.Visibility)

		repo = &GithubRepoComparable{
			Visibility: "public",
		}
		repo = filter.RepositoryFilter("repo", repo)
		assert.Equal(t, "private", repo.Visibility)
	})

	t.Run("happy path: not enterprise", func(t *testing.T) {
		config := &config.RepositoryConfig{}

		filter := NewReconciliatorFilter(false, config)
		repo := &GithubRepoComparable{
			Visibility: "internal",
		}
		repo = filter.RepositoryFilter("repo", repo)
		assert.Equal(t, "private", repo.Visibility)

		repo = &GithubRepoComparable{
			Visibility: "public",
		}
		repo = filter.RepositoryFilter("repo", repo)
		assert.Equal(t, "public", repo.Visibility)
	})

	t.Run("happy path: forbidden except", func(t *testing.T) {
		config := &config.RepositoryConfig{}
		config.VisibilityRules.ForbidPublicRepositories = true
		config.VisibilityRules.ForbidPublicRepositoriesExclusions = []string{"repo"}

		filter := NewReconciliatorFilter(true, config)
		repo := &GithubRepoComparable{
			Visibility: "public",
		}
		repo = filter.RepositoryFilter("repo", repo)
		assert.Equal(t, "public", repo.Visibility)

		repo = &GithubRepoComparable{
			Visibility: "public",
		}
		repo = filter.RepositoryFilter("repo2", repo)
		assert.Equal(t, "private", repo.Visibility)
	})
}
