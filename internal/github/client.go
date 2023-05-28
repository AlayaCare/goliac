package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

type GitHubClient interface {
	QueryGraphQLAPI(query string, variables map[string]interface{}) (string, error)
	CallRestAPIWithBody(endpoint, method string, body map[string]interface{}) (string, error)
	CallRestAPI(endpoint string) (string, error)
	GetAccessToken() (string, error)
}

type GitHubClientImpl struct {
	gitHubServer    string
	appID           string
	installationID  string
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
	defer t.client.mu.Unlock()

	// Refresh the access token if necessary
	if t.client.accessToken == "" || time.Until(t.client.tokenExpiration) < 5*time.Minute {
		token, err := t.client.createJWT()
		if err != nil {
			return nil, err
		}

		accessToken, expiresAt, err := t.client.getAccessTokenForInstallation(token)
		if err != nil {
			return nil, err
		}

		t.client.accessToken = accessToken
		t.client.tokenExpiration = expiresAt
	}

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
func NewGitHubClientImpl(githubServer, organizationName, appID, privateKeyFile string) (GitHubClient, error) {
	privateKey, err := ioutil.ReadFile(privateKeyFile)
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
		if installation.Account.Login == organizationName {
			client.installationID = installation.ID
			break
		}
	}

	if client.installationID == "" {
		return nil, fmt.Errorf("installation not found for organization: %s", organizationName)
	}

	transport := &AuthorizedTransport{
		client: client,
	}

	httpClient := &http.Client{Transport: transport}

	client.httpClient = httpClient

	return client, nil
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
func (client *GitHubClientImpl) QueryGraphQLAPI(query string, variables map[string]interface{}) (string, error) {
	body, err := json.Marshal(GraphQLRequest{
		Query:     query,
		Variables: variables,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", client.gitHubServer+"/graphql", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(responseBody), nil
}

/*
 * CallRestAPI
 * @param {string} endpoint
 *
 * Example:
 * responseBody, err := client.CallRestAPI("repos/my-org/my-repo")
 */
func (client *GitHubClientImpl) CallRestAPI(endpoint string) (string, error) {
	req, err := http.NewRequest("GET", client.gitHubServer+"/"+endpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(responseBody), nil
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
func (client *GitHubClientImpl) CallRestAPIWithBody(endpoint, method string, body map[string]interface{}) (string, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(method, client.gitHubServer+"/"+endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(responseBody), nil
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

func (client *GitHubClientImpl) getAccessTokenForInstallation(jwt string) (string, time.Time, error) {
	req, err := http.NewRequest("POST", client.gitHubServer+"/app/installations/"+client.installationID+"/access_tokens", nil)
	if err != nil {
		return "", time.Now(), err
	}

	req.Header.Add("Authorization", "Bearer "+jwt)
	req.Header.Add("Accept", "application/vnd.github.machine-man-preview+json")

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
func (client *GitHubClientImpl) GetAccessToken() (string, error) {

	if client.accessToken != "" && client.tokenExpiration.After(time.Now()) {
		return client.accessToken, nil
	}

	jwt, err := client.createJWT()
	if err != nil {
		return "", err
	}

	accessToken, expiration, err := client.getAccessTokenForInstallation(jwt)
	if err != nil {
		return "", err
	}

	client.accessToken = accessToken
	client.tokenExpiration = expiration

	return accessToken, nil
}
