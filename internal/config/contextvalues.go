package config

type contextKey string

const (
	// ContextKeyConfig is the key used to store the configuration in the context.
	ContextKeyStatistics contextKey = "githubStatistics"
)

type GoliacStatistics struct {
	GithubApiCalls  int
	GithubThrottled int
}
