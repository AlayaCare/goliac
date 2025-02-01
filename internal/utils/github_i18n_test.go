package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGithubAnsiString(t *testing.T) {
	t.Run("happy path: no change", func(t *testing.T) {
		str := "test"
		assert.Equal(t, "test", GithubAnsiString(str))
	})
	t.Run("happy path: with number", func(t *testing.T) {
		str := "Repo1"
		assert.Equal(t, "Repo1", GithubAnsiString(str))
	})

	t.Run("not happy path: replace", func(t *testing.T) {
		str := "test@123"
		assert.Equal(t, "test-123", GithubAnsiString(str))
	})
	t.Run("not happy path: accent", func(t *testing.T) {
		str := "été"
		assert.Equal(t, "-t-", GithubAnsiString(str))
	})
}
