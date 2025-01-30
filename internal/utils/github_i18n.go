package utils

import "regexp"

var githubAnsiStringRegexp = regexp.MustCompile("[^A-Za-z0-9_.-]")

// this function is used to convert a string to a [A-Za-z0-9_.-] string
func GithubAnsiString(str string) string {
	// convert all non [A-Za-z0-9_.-] characters to -
	return githubAnsiStringRegexp.ReplaceAllString(str, "-")
}
