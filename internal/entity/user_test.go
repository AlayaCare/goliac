package entity

import (
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/goliac-project/goliac/internal/observability"
	"github.com/goliac-project/goliac/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestUser(t *testing.T) {

	// happy path
	t.Run("happy path", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fs.MkdirAll("users", 0755)
		err := utils.WriteFile(fs, "users/user1.yaml", []byte(`
apiVersion: v1
kind: User
name: user1
spec:
  githubID: github1
`), 0644)
		assert.Nil(t, err)

		errorCollector := observability.NewErrorCollection()
		users := ReadUserDirectory(fs, "users", errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, users)
		assert.Equal(t, len(users), 1)
		user1 := users["user1"]
		assert.NotNil(t, user1)
		assert.Equal(t, "github1", user1.Spec.GithubID)
	})

	t.Run("happy path: with --- separator", func(t *testing.T) {
		// create a new user starting with "---"
		fs := memfs.New()
		fs.MkdirAll("users", 0755)
		err := utils.WriteFile(fs, "users/user1.yaml", []byte(`---
apiVersion: v1
kind: User
name: user1
spec:
  githubID: github1
`), 0644)
		assert.Nil(t, err)
		errorCollector := observability.NewErrorCollection()
		users := ReadUserDirectory(fs, "users", errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, users)
		assert.Equal(t, len(users), 1)
		user1 := users["user1"]
		assert.NotNil(t, user1)
		assert.Equal(t, "github1", user1.Spec.GithubID)
	})

	t.Run("not happy path: no users directory", func(t *testing.T) {
		// create a new user starting with "---"
		fs := memfs.New()
		errorCollector := observability.NewErrorCollection()
		ReadUserDirectory(fs, "users", errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
	})

	t.Run("not happy path: missing metadata", func(t *testing.T) {
		// create a new user starting with "---"
		fs := memfs.New()
		fs.MkdirAll("users", 0755)
		err := utils.WriteFile(fs, "users/user1.yaml", []byte(`---
apiVersion: v1
kind: User
spec:
  githubID: github1
`), 0644)
		assert.Nil(t, err)
		errorCollector := observability.NewErrorCollection()
		ReadUserDirectory(fs, "users", errorCollector)
		assert.Equal(t, 1, len(errorCollector.Errors))
		assert.Equal(t, false, errorCollector.HasWarns())
	})
}

func TestEqualUser(t *testing.T) {
	t.Run("happy path: same user", func(t *testing.T) {
		userA := User{}
		userA.ApiVersion = "v1"
		userA.Kind = "User"
		userA.Name = "usera"
		userA.Spec.GithubID = "githubidA"

		userB := User{}
		userB.ApiVersion = "v1"
		userB.Kind = "User"
		userB.Name = "usera"
		userB.Spec.GithubID = "githubidA"

		res := userA.Equals(&userB)

		assert.True(t, res)
	})

	t.Run("nit happy path: different user", func(t *testing.T) {
		userA := User{}
		userA.ApiVersion = "v1"
		userA.Kind = "User"
		userA.Name = "usera"
		userA.Spec.GithubID = "githubidA"

		userB := User{}
		userB.ApiVersion = "v1"
		userB.Kind = "User"
		userB.Name = "userb"
		userB.Spec.GithubID = "githubidB"

		res := userA.Equals(&userB)

		assert.False(t, res)
	})
}
