package utils

import (
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/stretchr/testify/assert"
)

func TestFs(t *testing.T) {
	t.Run("happy path: simple file", func(t *testing.T) {
		fs := memfs.New()
		exists, err := Exists(fs, "test")
		assert.Nil(t, err)
		assert.False(t, exists)
	})
	t.Run("happy path: simple directory", func(t *testing.T) {
		fs := memfs.New()

		// let's create a directory
		err := fs.MkdirAll("/dir", 0755)
		assert.Nil(t, err)

		err = WriteFile(fs, "/dir/test", []byte("test"), 0644)
		assert.Nil(t, err)

		exists, err := Exists(fs, "/dir")
		assert.Nil(t, err)
		assert.True(t, exists)
	})
}
