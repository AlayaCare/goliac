package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStrings(t *testing.T) {
	t.Run("FirstTwoWordsBeforeParenthesis", func(t *testing.T) {
		assert.Equal(t, "hello world", FirstTwoWordsBeforeParenthesis("\n hello world(this is a test)", 100))
		assert.Equal(t, "hello world", FirstTwoWordsBeforeParenthesis("hello \t world", 100))
		assert.Equal(t, "hello", FirstTwoWordsBeforeParenthesis("hello world (this is a test)", 5))
		assert.Equal(t, "hello", FirstTwoWordsBeforeParenthesis("hello world", 5))
		assert.Equal(t, "hell", FirstTwoWordsBeforeParenthesis("\nhello world (this is a test)", 5))
	})
}
