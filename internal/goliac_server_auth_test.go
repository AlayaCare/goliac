package internal

import (
	"context"
	"net/http"
	"testing"

	"github.com/goliac-project/goliac/swagger_gen/restapi/operations/auth"
	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

type OAuth2ConfigMock struct {
	AuthCodeURLMock string
	TokenMock       string
}

func (a *OAuth2ConfigMock) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	return a.AuthCodeURLMock
}

func (a *OAuth2ConfigMock) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: a.TokenMock,
	}, nil
}

type GithubClientMock struct {
}

func (g *GithubClientMock) QueryGraphQLAPI(ctx context.Context, query string, variables map[string]interface{}) ([]byte, error) {
	return nil, nil
}
func (g *GithubClientMock) CallRestAPI(ctx context.Context, endpoint, parameters, method string, body map[string]interface{}, githubToken *string) ([]byte, error) {
	if endpoint == "/user" {
		return []byte(`{"login": "testuser", "name":"Test User"}`), nil
	}
	return nil, nil
}
func (g *GithubClientMock) GetAccessToken(ctx context.Context) (string, error) {
	return "test-token", nil
}
func (g *GithubClientMock) CreateJWT() (string, error) {
	return "test-jwt", nil
}
func (g *GithubClientMock) GetAppSlug() string {
	return "test-app-slug"
}

func TestAuthGetLogin(t *testing.T) {
	t.Run("happy path: ", func(t *testing.T) {
		localfixture, remotefixture := fixtureGoliacLocal()
		goliac := NewGoliacMock(localfixture, remotefixture)
		server := GoliacServerImpl{
			goliac: goliac,
			oauthConfig: &oauth2.Config{
				ClientID: "test-client-id",
				Scopes:   []string{"read:user"},
				Endpoint: oauth2.Endpoint{
					AuthURL:       "https://github.com/authorize",
					DeviceAuthURL: "https://github.com/authorize/device",
					TokenURL:      "https://github.com/access_token",
				},
			},
		}
		res := server.AuthGetLogin(auth.GetAuthenticationLoginParams{})
		payload := res.(*auth.GetAuthenticationLoginFound)
		assert.Equal(t, "https://github.com/authorize?access_type=offline&client_id=test-client-id&response_type=code&scope=read%3Auser&state=%252Fauth", payload.Location.String())
	})
}

func TestAuthGetCallback(t *testing.T) {
	t.Run("happy path: ", func(t *testing.T) {
		localfixture, remotefixture := fixtureGoliacLocal()
		goliac := NewGoliacMock(localfixture, remotefixture)
		server := GoliacServerImpl{
			goliac:       goliac,
			sessionStore: sessions.NewCookieStore([]byte("your-secret-key")),
			oauthConfig: &OAuth2ConfigMock{
				AuthCodeURLMock: "http://localhost:8080/auth",
				TokenMock:       "test-token",
			},
		}
		httpRequest := &http.Request{
			Method: "GET",
			URL:    nil,
			Header: http.Header{
				"Authorization": []string{"Bearer test-token"},
			},
			Body: nil,
		}
		res := server.AuthGetCallback(auth.GetAuthenticationCallbackParams{
			HTTPRequest: httpRequest,
			Code:        "test-code",
			State:       "/auth",
		})
		responder := res.(*CustomResponder)
		assert.Equal(t, "test-token", responder.session.Values["access_token"])

		payload := responder.responder.(*auth.GetAuthenticationCallbackFound)
		assert.Equal(t, "/auth", payload.Location.String())
	})

	t.Run("not happy path: ", func(t *testing.T) {
		localfixture, remotefixture := fixtureGoliacLocal()
		goliac := NewGoliacMock(localfixture, remotefixture)
		server := GoliacServerImpl{
			goliac:       goliac,
			sessionStore: sessions.NewCookieStore([]byte("your-secret-key")),
			oauthConfig: &OAuth2ConfigMock{
				AuthCodeURLMock: "http://localhost:8080/auth",
				TokenMock:       "test-token",
			},
		}
		httpRequest := &http.Request{
			Method: "GET",
			URL:    nil,
			Header: http.Header{
				"Authorization": []string{"Bearer test-token"},
			},
			Body: nil,
		}
		res := server.AuthGetCallback(auth.GetAuthenticationCallbackParams{
			HTTPRequest: httpRequest,
			Code:        "",
			State:       "/auth",
		})
		payload := res.(*auth.GetAuthenticationCallbackDefault)
		assert.Equal(t, "Missing authorization code", *payload.Payload.Message)
	})
}

func TestGetUserInfo(t *testing.T) {
	t.Run("happy path: ", func(t *testing.T) {
		localfixture, remotefixture := fixtureGoliacLocal()
		goliac := NewGoliacMock(localfixture, remotefixture)
		server := GoliacServerImpl{
			goliac: goliac,
			oauthConfig: &OAuth2ConfigMock{
				AuthCodeURLMock: "http://localhost:8080/auth",
				TokenMock:       "test-token",
			},
			client: &GithubClientMock{},
		}
		ghInfo, err := server.GetUserInfo(context.Background(), "aToken")
		assert.Nil(t, err)
		assert.Equal(t, "testuser", ghInfo.Login)
		assert.Equal(t, "Test User", ghInfo.Name)
	})
}

func TestHhelperCheckOrgMembership(t *testing.T) {
	t.Run("happy path: ", func(t *testing.T) {
		localfixture, remotefixture := fixtureGoliacLocal()
		goliac := NewGoliacMock(localfixture, remotefixture)
		server := GoliacServerImpl{
			goliac:       goliac,
			client:       &GithubClientMock{},
			sessionStore: sessions.NewCookieStore([]byte("your-secret-key")),
		}
		httpRequest := &http.Request{
			Method: "GET",
			URL:    nil,
			Header: http.Header{
				"Authorization": []string{"Bearer test-token"},
			},
			Body: nil,
		}
		// set session
		session, _ := server.sessionStore.Get(httpRequest, "auth-session")
		session.Values["access_token"] = "my-access-token"

		userInfo, errCode, errorModel := server.helperCheckOrgMembership(httpRequest)
		assert.Nil(t, errorModel)
		assert.Equal(t, 200, errCode)
		assert.Equal(t, "testuser", userInfo.Login)
	})

	t.Run("not happy path: no auth-session ", func(t *testing.T) {
		localfixture, remotefixture := fixtureGoliacLocal()
		goliac := NewGoliacMock(localfixture, remotefixture)
		server := GoliacServerImpl{
			goliac:       goliac,
			client:       &GithubClientMock{},
			sessionStore: sessions.NewCookieStore([]byte("your-secret-key")),
		}
		httpRequest := &http.Request{
			Method: "GET",
			URL:    nil,
			Header: http.Header{
				"Authorization": []string{"Bearer test-token"},
			},
			Body: nil,
		}
		_, errCode, errorModel := server.helperCheckOrgMembership(httpRequest)
		assert.NotNil(t, errorModel)
		assert.Equal(t, 401, errCode)
	})
}

func TestAuthGetUser(t *testing.T) {
	t.Run("happy path: ", func(t *testing.T) {
		localfixture, remotefixture := fixtureGoliacLocal()
		goliac := NewGoliacMock(localfixture, remotefixture)
		server := GoliacServerImpl{
			goliac:       goliac,
			client:       &GithubClientMock{},
			sessionStore: sessions.NewCookieStore([]byte("your-secret-key")),
		}
		httpRequest := &http.Request{
			Method: "GET",
			URL:    nil,
			Header: http.Header{
				"Authorization": []string{"Bearer test-token"},
			},
			Body: nil,
		}
		// set session
		session, _ := server.sessionStore.Get(httpRequest, "auth-session")
		session.Values["access_token"] = "my-access-token"

		res := server.AuthGetUser(auth.GetGithubUserParams{
			HTTPRequest: httpRequest,
		})
		payload := res.(*auth.GetGithubUserOK)
		assert.Equal(t, "testuser", payload.Payload.GithubID)
		assert.Equal(t, "Test User", payload.Payload.Name)
	})
}

func TestAuthWorkflowsForcemerge(t *testing.T) {
	t.Run("happy path: ", func(t *testing.T) {
		localfixture, remotefixture := fixtureGoliacLocal()
		goliac := NewGoliacMock(localfixture, remotefixture)
		server := GoliacServerImpl{
			goliac:       goliac,
			client:       &GithubClientMock{},
			sessionStore: sessions.NewCookieStore([]byte("your-secret-key")),
		}
		httpRequest := &http.Request{
			Method: "GET",
			URL:    nil,
			Header: http.Header{
				"Authorization": []string{"Bearer test-token"},
			},
			Body: nil,
		}
		// set session
		session, _ := server.sessionStore.Get(httpRequest, "auth-session")
		session.Values["access_token"] = "my-access-token"

		res := server.AuthWorkflows(auth.GetWorkflowsParams{
			HTTPRequest: httpRequest,
		})
		payload := res.(*auth.GetWorkflowsOK)
		fmWorkflows := payload.Payload
		assert.Equal(t, 1, len(fmWorkflows))
		fmWorkflow := fmWorkflows[0]
		assert.Equal(t, "fmtest", fmWorkflow.WorkflowName)
	})
}
