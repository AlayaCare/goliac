package entity

import (
	"testing"

	"github.com/Alayacare/goliac/internal/utils"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/stretchr/testify/assert"
)

func TestEntity(t *testing.T) {

	// happy path
	t.Run("happy path", func(t *testing.T) {
		fs := memfs.New()
		err := utils.WriteFile(fs, "foobar.yaml", []byte(`
apiVersion: v1
kind: FooBar
name: name
`), 0644)
		assert.Nil(t, err)
		e, err := parseEntity(fs, "foobar.yaml")
		assert.Nil(t, err)
		assert.NotNil(t, e)
		assert.Equal(t, "name", e.Name)
		assert.Equal(t, "FooBar", e.Kind)
		assert.Equal(t, "v1", e.ApiVersion)

	})

	t.Run("not happy path: file does not exist", func(t *testing.T) {
		fs := memfs.New()
		e, err := parseEntity(fs, "foobar.yaml")
		assert.NotNil(t, err)
		assert.Nil(t, e)
	})

	t.Run("not happy path: wrong yaml", func(t *testing.T) {
		fs := memfs.New()
		err := utils.WriteFile(fs, "foobar.yaml", []byte(`
apiVersion: v1
kind: FooBar
name:
name:
`), 0644)
		assert.Nil(t, err)
		_, err = parseEntity(fs, "foobar.yaml")
		assert.NotNil(t, err)
	})
}
