package utils

import (
	"unicode"
)

func FirstTwoWordsBeforeParenthesis(s string, max int) string {
	spaceCount := 0
	lastCharWasSpace := false
	result := ""
	for i, char := range s {
		if i >= max {
			return result
		}
		if char == '(' {
			return result
		}
		if unicode.IsSpace(char) || char == '\n' || char == '\t' {
			if lastCharWasSpace || len(result) == 0 {
				continue
			}
			lastCharWasSpace = true
			spaceCount++
			if spaceCount == 2 {
				return result
			}
			result += " "
		} else {
			lastCharWasSpace = false
			result += string(char)
		}
	}
	return result
}
