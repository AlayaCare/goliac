package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/goliac-project/goliac/internal/config"
	"github.com/goliac-project/goliac/swagger_gen/models"
	"github.com/goliac-project/goliac/swagger_gen/restapi/operations/auth"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

func (g *GoliacServerImpl) AuthGetLogin(params auth.GetAuthenticationLoginParams) middleware.Responder {
	// Get the original URL user was trying to access

	originalURL := "/auth"
	if params.Redirect != nil && *params.Redirect != "" {
		originalURL = *params.Redirect
	}

	// Store the original URL in the OAuth state parameter
	oauthState := url.QueryEscape(originalURL)
	authURL := g.oauthConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline)

	// Redirect user to GitHub OAuth login
	return auth.NewGetAuthenticationLoginFound().WithLocation(strfmt.URI(authURL))
}

type CustomResponder struct {
	responder middleware.Responder
	r         *http.Request
	session   *sessions.Session
}

func NewCustomResponder(responder middleware.Responder) *CustomResponder {
	return &CustomResponder{
		responder: responder,
	}
}

func (cr *CustomResponder) WriteResponse(rw http.ResponseWriter, p runtime.Producer) {
	if cr.session != nil && cr.r != nil {
		cr.session.Save(cr.r, rw)
	}
	cr.responder.WriteResponse(rw, p)
}

func (g *GoliacServerImpl) AuthGetCallback(params auth.GetAuthenticationCallbackParams) middleware.Responder {
	// Retrieve state (original URL)
	oauthState := params.State
	originalURL, _ := url.QueryUnescape(oauthState) // Decode original URL

	if params.Code == "" {
		message := "Missing authorization code"
		return auth.NewGetAuthenticationCallbackDefault(404).WithPayload(&models.Error{Message: &message})
	}

	// Exchange the code for an access token
	token, err := g.oauthConfig.Exchange(context.Background(), params.Code)
	if err != nil {
		message := fmt.Sprintf("Failed to get token: %s", err.Error())
		return auth.NewGetAuthenticationCallbackDefault(404).WithPayload(&models.Error{Message: &message})
	}

	session, _ := g.sessionStore.Get(params.HTTPRequest, "auth-session")
	session.Values["access_token"] = token.AccessToken
	// session.Values["user"] = user.Login

	responder := NewCustomResponder(auth.NewGetAuthenticationCallbackFound().WithLocation(strfmt.URI(originalURL)))
	responder.r = params.HTTPRequest
	responder.session = session
	return responder

	// // Fetch user info
	// user, err := fetchGitHubUser(token.AccessToken)
	// if err != nil {
	// 	http.Error(w, "Failed to get user info", http.StatusInternalServerError)
	// 	return
	// }

	// // Check organization membership
	// if !isUserInGitHubOrg(token.AccessToken, requiredOrg) {
	// 	http.Error(w, "Access denied: Not in required GitHub organization", http.StatusForbidden)
	// 	return
	// }

}

type GithubUserInfo struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

func (g *GoliacServerImpl) GetUserInfo(ctx context.Context, ghuToken string) (*GithubUserInfo, error) {

	// https://docs.github.com/en/rest/users/users?apiVersion=2022-11-28#get-the-authenticated-user
	body, err := g.client.CallRestAPI(ctx, "/user", "", "GET", nil, &ghuToken)

	if err != nil {
		return nil, err
	}

	var user GithubUserInfo
	err = json.Unmarshal(body, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (g *GoliacServerImpl) GetOrgMembership(ctx context.Context, ghuToken string, org string, username string) (bool, error) {
	_, err := g.client.CallRestAPI(ctx, "/orgs/"+org+"/memberships/"+username, "", "GET", nil, &ghuToken)
	if err != nil {
		return false, err
	}
	return true, nil
}

/*
helperCheckOrgMembership checks if the user is in the required GitHub organization.
If the user is not in the organization, it returns an error.
*/
func (g *GoliacServerImpl) helperCheckOrgMembership(req *http.Request) (*GithubUserInfo, int, *models.Error) {
	// Retrieve the access token from the session
	session, _ := g.sessionStore.Get(req, "auth-session")
	accessToken, ok := session.Values["access_token"].(string)
	if !ok {
		message := "Access token not found"
		return nil, 401, &models.Error{Message: &message}
	}
	userinfo, err := g.GetUserInfo(req.Context(), accessToken)
	if err != nil {
		message := "Failed to get user info"
		return nil, 401, &models.Error{Message: &message}
	}
	// Check organization membership
	isMember, err := g.GetOrgMembership(req.Context(), accessToken, config.Config.GithubAppOrganization, userinfo.Login)
	if err != nil {
		message := "Failed to check organization membership"
		return nil, 401, &models.Error{Message: &message}
	}
	if !isMember {
		message := "Access denied: Not in required GitHub organization"
		return nil, 403, &models.Error{Message: &message}
	}

	return userinfo, 200, nil
}

func (g *GoliacServerImpl) AuthGetUser(params auth.GetGithubUserParams) middleware.Responder {
	userinfo, codestatus, err := g.helperCheckOrgMembership(params.HTTPRequest)

	if err != nil {
		return auth.NewGetGithubUserDefault(codestatus).WithPayload(err)
	}
	// fmt.Println("User is in the required organization")
	// fmt.Println("User info:", userinfo)

	return auth.NewGetGithubUserOK().WithPayload(&models.Githubuser{
		GithubID: userinfo.Login,
		Name:     userinfo.Name, // usually null unfortunately
	})
}

func (g *GoliacServerImpl) AuthWorkflow(params auth.PostWorkflowParams) middleware.Responder {
	userinfo, codestatus, merr := g.helperCheckOrgMembership(params.HTTPRequest)

	if merr != nil {
		return auth.NewPostWorkflowDefault(codestatus).WithPayload(merr)
	}

	workflow, ok := g.goliac.GetLocal().Workflows()[params.WorkflowName]
	if !ok {
		message := fmt.Sprintf("Workflow %s not found", params.WorkflowName)
		return auth.NewPostWorkflowDefault(404).WithPayload(&models.Error{Message: &message})
	}

	instance := g.worflowInstances[workflow.Name]
	if instance == nil {
		message := fmt.Sprintf("Workflow instance not found: %s", workflow.Name)
		return auth.NewPostWorkflowDefault(404).WithPayload(&models.Error{Message: &message})
	}

	// check if the workflow exists and execute it
	responses, err := instance.ExecuteWorkflow(
		params.HTTPRequest.Context(),
		g.goliac.GetLocal().RepoConfig().Workflows,
		userinfo.Login,
		params.WorkflowName,
		params.Body.PrURL,
		params.Body.Explanation,
		false)
	if err != nil {
		message := fmt.Sprintf("Failed to execute workflow: %s", err.Error())
		return auth.NewPostWorkflowDefault(500).WithPayload(&models.Error{Message: &message})
	}

	return auth.NewPostWorkflowOK().WithPayload(&models.Prmerged{
		Message:      "PR merged",
		TrackingUrls: responses,
	})
}

func (g *GoliacServerImpl) AuthWorkflows(params auth.GetWorkflowsParams) middleware.Responder {
	_, codestatus, merr := g.helperCheckOrgMembership(params.HTTPRequest)

	if merr != nil {
		return auth.NewGetWorkflowsDefault(codestatus).WithPayload(merr)
	}

	workflows := models.WorkflowsPrmerged{}

	for name, workflow := range g.goliac.GetLocal().Workflows() {

		workflows = append(workflows, &models.WorkflowsPrmergedItems0{
			WorkflowName:        name,
			WorkflowDescription: workflow.Spec.Description,
		})
	}

	return auth.NewGetWorkflowsOK().WithPayload(workflows)
}
