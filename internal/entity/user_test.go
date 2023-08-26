package entity

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestUser(t *testing.T) {

	// happy path
	t.Run("happy path", func(t *testing.T) {
		// create a new user
		fs := afero.NewMemMapFs()
		fs.Mkdir("users", 0755)
		err := afero.WriteFile(fs, "users/user1.yaml", []byte(`
apiVersion: v1
kind: User
metadata:
  name: user1
data:
  githubID: github1
`), 0644)
		assert.Nil(t, err)
		users, errs, warns := ReadUserDirectory(fs, "users")
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, users)
		assert.Equal(t, len(users), 1)
		user1 := users["user1"]
		assert.NotNil(t, user1)
		assert.Equal(t, "github1", user1.Data.GithubID)
	})

	t.Run("happy path: with --- separator", func(t *testing.T) {
		// create a new user starting with "---"
		fs := afero.NewMemMapFs()
		fs.Mkdir("users", 0755)
		err := afero.WriteFile(fs, "users/user1.yaml", []byte(`---
apiVersion: v1
kind: User
metadata:
  name: user1
data:
  githubID: github1
`), 0644)
		assert.Nil(t, err)
		users, errs, warns := ReadUserDirectory(fs, "users")
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
		assert.NotNil(t, users)
		assert.Equal(t, len(users), 1)
		user1 := users["user1"]
		assert.NotNil(t, user1)
		assert.Equal(t, "github1", user1.Data.GithubID)
	})

	t.Run("not happy path: no users directory", func(t *testing.T) {
		// create a new user starting with "---"
		fs := afero.NewMemMapFs()
		_, errs, warns := ReadUserDirectory(fs, "users")
		assert.Equal(t, len(errs), 0)
		assert.Equal(t, len(warns), 0)
	})

	t.Run("not happy path: missing metadata", func(t *testing.T) {
		// create a new user starting with "---"
		fs := afero.NewMemMapFs()
		fs.Mkdir("users", 0755)
		err := afero.WriteFile(fs, "users/user1.yaml", []byte(`---
apiVersion: v1
kind: User
data:
  githubID: github1
`), 0644)
		assert.Nil(t, err)
		_, errs, warns := ReadUserDirectory(fs, "users")
		assert.Equal(t, len(errs), 1)
		assert.Equal(t, len(warns), 0)
	})
}
