package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goliac-project/goliac/internal/config"
	"github.com/sirupsen/logrus"
)

type GitHubClient interface {
	QueryGraphQLAPI(ctx context.Context, query string, variables map[string]interface{}) ([]byte, error)
	CallRestAPI(ctx context.Context, endpoint, parameters, method string, body map[string]interface{}) ([]byte, error)
	GetAccessToken(ctx context.Context) (string, error)
	GetAppSlug() string
}

type GitHubClientImpl struct {
	gitHubServer    string
	appID           int64
	installationID  int64
	appSlug         string
	privateKey      []byte
	accessToken     string
	httpClient      *http.Client
	tokenExpiration time.Time
	mu              sync.Mutex
}

type AuthorizedTransport struct {
	client *GitHubClientImpl
}

func (t *AuthorizedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.client.mu.Lock()

	// Refresh the access token if necessary
	if t.client.accessToken == "" || time.Until(t.client.tokenExpiration) < 5*time.Minute {
		token, err := t.client.createJWT()
		if err != nil {
			t.client.mu.Unlock()
			return nil, err
		}

		accessToken, expiresAt, err := t.client.getAccessTokenForInstallation(req.Context(), token)
		if err != nil {
			t.client.mu.Unlock()
			return nil, err
		}
		t.client.accessToken = accessToken
		t.client.tokenExpiration = expiresAt
	}
	t.client.mu.Unlock()

	req.Header.Add("Authorization", "Bearer "+t.client.accessToken)

	return http.DefaultTransport.RoundTrip(req)
}

/**
 * NewGitHubClient
 * @param {string} githubServer usually https://api.github.com
 * @param {string} organizationName
 * @param {string} appID
 * @param {string} privateKeyFile
 * @return {GitHubClient} client
 * @return {error} error
 *
 * Example:
 * client, err := NewGitHubClient(
 * 	"https://api.github.com",
 * 	"my-org",
 * 	"12345",
 * 	"private-key.pem",
 * )
 */
func NewGitHubClientImpl(githubServer, organizationName string, appID int64, privateKeyFile string) (GitHubClient, error) {
	privateKey, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return nil, err
	}

	client := &GitHubClientImpl{
		gitHubServer: githubServer,
		appID:        appID,
		privateKey:   privateKey,
	}

	// create JWT
	token, err := client.createJWT()
	if err != nil {
		return nil, err
	}

	// retrieve all installations for the authenticated app
	installations, err := client.getInstallations(token)
	if err != nil {
		return nil, err
	}

	// find the installation ID for the given organization
	for _, installation := range installations {
		logrus.Debugf("Found installation %s with id %d for organization: %s", installation.AppSlug, installation.ID, organizationName)
		if strings.EqualFold(installation.Account.Login, organizationName) && installation.AppId == appID {
			client.installationID = installation.ID
			client.appSlug = installation.AppSlug
			break
		}
	}

	if client.installationID == 0 {
		return nil, fmt.Errorf("installation not found for organization: %s", organizationName)
	}

	transport := &AuthorizedTransport{
		client: client,
	}

	httpClient := &http.Client{Transport: transport}

	client.httpClient = httpClient

	return client, nil
}

// waitRateLimit helps dealing with rate limits
// cf https://docs.github.com/en/rest/guides/best-practices-for-integrators?apiVersion=2022-11-28#dealing-with-rate-limits
func waitRateLimit(resetTimeStr string) error {
	if resetTimeStr == "" {
		return fmt.Errorf("X-RateLimit-Reset header not found")
	}

	logrus.Infof("Rate limit exceeded, waiting for %s", resetTimeStr)

	// Parse the reset time.
	resetTimeUnix, err := strconv.ParseInt(resetTimeStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse X-RateLimit-Reset header: %w", err)
	}

	resetTime := time.Unix(resetTimeUnix, 0)

	// Calculate how long we need to wait.
	waitDuration := time.Until(resetTime)

	// Wait until the reset time.
	time.Sleep(waitDuration)

	return nil
}

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

/*
 * QueryGraphQLAPI
 * @param {string} query
 * @param {map[string]interface{}} variables
 * @return {string} responseBody
 * @return {error} error
 *
 * Example:
 * query := `
 *	query($name: String!) {
 *		user(login: $name) {
 *			name
 *			company
 *		}
 *	}
 * `
 * variables := map[string]interface{}{
 *	"name": "octocat",
 * }
 * responseBody, err := client.QueryGraphQLAPI(query, variables)
 */
func (client *GitHubClientImpl) QueryGraphQLAPI(ctx context.Context, query string, variables map[string]interface{}) ([]byte, error) {
	body, err := json.Marshal(GraphQLRequest{
		Query:     query,
		Variables: variables,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", client.gitHubServer+"/graphql", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	stats := ctx.Value(config.ContextKeyStatistics)
	if stats != nil {
		goliacStats := stats.(*config.GoliacStatistics)
		goliacStats.GithubApiCalls++
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden {
		if stats != nil {
			goliacStats := stats.(*config.GoliacStatistics)
			goliacStats.GithubThrottled++
		}

		if resp.Header.Get("X-RateLimit-Reset") != "" {
			// We're being rate limited. Get the reset time from the headers.
			if err := waitRateLimit(resp.Header.Get("X-RateLimit-Reset")); err != nil {
				return nil, err
			}
		} else if resp.Header.Get("Retry-After") != "" {
			retryAfter, err := strconv.Atoi(resp.Header.Get("Retry-After"))
			if err != nil {
				return nil, err
			}
			if retryAfter > 30 {
				retryAfter = retryAfter / 2 // ok we shouldn't be too aggressive
			}
			logrus.Debugf("2nd rate limit reached, waiting for %d seconds", retryAfter)
			time.Sleep(time.Duration(retryAfter) * time.Second)
		} else {
			return nil, fmt.Errorf("unexpected status: %s", resp.Status)
		}

		// Retry the request.
		return client.QueryGraphQLAPI(ctx, query, variables)
	} else {
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return responseBody, nil
	}
}

/*
 * CallRestAPIWithBody
 * @param {string} endpoint
 * @param {string} method
 * @param {map[string]interface{}} body
 *
 * Example:
 * body := map[string]interface{}{
 *	"name": "my-repo",
 *	"private": true,
 * }
 * responseBody, err := client.CallRestAPIWithBody("orgs/my-org/repos", "POST", body)
 */
func (client *GitHubClientImpl) CallRestAPI(ctx context.Context, endpoint, parameters, method string, body map[string]interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}
	urlpath, err := url.JoinPath(client.gitHubServer, endpoint)
	if err != nil {
		return nil, err
	}

	stats := ctx.Value(config.ContextKeyStatistics)
	if stats != nil {
		goliacStats := stats.(*config.GoliacStatistics)
		goliacStats.GithubApiCalls++
	}

	if parameters != "" {
		urlpath = urlpath + "?" + parameters
	}

	req, err := http.NewRequestWithContext(ctx, method, urlpath, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	//	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		if stats != nil {
			goliacStats := stats.(*config.GoliacStatistics)
			goliacStats.GithubThrottled++
		}

		// We're being rate limited. Get the reset time from the headers.
		if err := waitRateLimit(resp.Header.Get("X-RateLimit-Reset")); err != nil {
			return nil, err
		}

		// Retry the request.
		return client.CallRestAPI(ctx, endpoint, parameters, method, body)
	} else {
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return responseBody, fmt.Errorf("unexpected status: %s", resp.Status)
		}

		return responseBody, nil
	}
}

func (client *GitHubClientImpl) createJWT() (string, error) {
	key, err := jwt.ParseRSAPrivateKeyFromPEM(client.privateKey)
	if err != nil {
		return "", err
	}

	// create a JWT
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": int32(time.Now().Unix()),
		"exp": int32(time.Now().Add(10 * time.Minute).Unix()),
		"iss": client.appID,
	})

	// sign the JWT with the app's private key
	signedToken, err := token.SignedString(key)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

type AccessTokenResponse struct {
	Token string `json:"token"`
}

func (client *GitHubClientImpl) getAccessTokenForInstallation(ctx context.Context, jwt string) (string, time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/app/installations/%d/access_tokens", client.gitHubServer, client.installationID), nil)
	if err != nil {
		return "", time.Now(), err
	}

	req.Header.Add("Authorization", "Bearer "+jwt)
	req.Header.Add("Accept", "application/vnd.github.machine-man-preview+json")

	stats := ctx.Value(config.ContextKeyStatistics)
	if stats != nil {
		goliacStats := stats.(*config.GoliacStatistics)
		goliacStats.GithubApiCalls++
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", time.Now(), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", time.Now(), fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var accessTokenResponse AccessTokenResponse
	err = json.NewDecoder(resp.Body).Decode(&accessTokenResponse)
	if err != nil {
		return "", time.Now(), err
	}

	return accessTokenResponse.Token, time.Now().Add(1 * time.Hour), nil
}

/*
 * GetAccessToken
 * It is used mostly to get an access token to clone a private repository
 * @return {string} accessToken
 * @return {error} error
 *
 * Example:
 * accessToken, err := client.GetAccessToken()
 * if err != nil {
 *	log.Fatal(err)
 * }
 * repo, err := git.PlainClone("/path/to/clone/repository", false, &git.CloneOptions{
 *	URL: "https://github.com/owner/repo.git",
 *	Auth: &http.BasicAuth{
 *		Username: "x-access-token",
 *		Password: accessToken,
 *	},
 */
func (client *GitHubClientImpl) GetAccessToken(ctx context.Context) (string, error) {
	logrus.Debugf("GetAccessToken(): client.tokenExpiration: %v", client.tokenExpiration)

	if client.accessToken != "" && client.tokenExpiration.After(time.Now()) {
		return client.accessToken, nil
	}

	jwt, err := client.createJWT()
	if err != nil {
		return "", err
	}

	accessToken, expiration, err := client.getAccessTokenForInstallation(ctx, jwt)
	if err != nil {
		return "", err
	}

	client.accessToken = accessToken
	client.tokenExpiration = expiration

	logrus.Debugf("GetAccessToken(): client.tokenExpiration: %v", client.tokenExpiration)

	return accessToken, nil
}

func (client *GitHubClientImpl) GetAppSlug() string {
	return client.appSlug
}
