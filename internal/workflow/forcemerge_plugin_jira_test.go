package workflow

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"encoding/json"

	"github.com/stretchr/testify/assert"
)

func TestJiraPluginWorkflow(t *testing.T) {

	t.Run("happy path: jira creation", func(t *testing.T) {
		httptest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			fmt.Println(string(body))
			var jBody JiraIssue
			err := json.Unmarshal(body, &jBody)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"errorMessages":["Project key is required"]}`))
				return
			}

			if jBody.Fields.Project.Key == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"errorMessages":["Project key is required"]}`))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id":"1234","key":"SRE-123","self":"https://mycompany.atlassian.net/rest/api/2/issue/1234"}`))
			}
		}))
		defer httptest.Close()

		plugin := ForcemergeStepPluginJira{
			AtlassianUrlDomain: httptest.URL,
			ProjectKey:         "",
			Email:              "serviceaccount@company.com",
			ApiToken:           "123456",
			IssueType:          "Bug",
		}

		prurl, err := url.Parse("https://github.com/mycompany/myrepo/pull/123")
		assert.Nil(t, err)

		returl, err := plugin.Execute(context.Background(), "foo", "explanation", prurl, map[string]interface{}{
			"project_key": "SRE",
		})

		assert.Nil(t, err)
		assert.Equal(t, fmt.Sprintf("%s/browse/SRE-123", httptest.URL), returl)
	})

	t.Run("not happy path: missing properties", func(t *testing.T) {
		httptest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var jBody JiraIssue
			err := json.Unmarshal(body, &jBody)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"errorMessages":["Project key is required"]}`))
				return
			}

			if jBody.Fields.Project.Key == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"errorMessages":["Project key is required"]}`))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id":"1234","key":"SRE-123","self":"https://mycompany.atlassian.net/rest/api/2/issue/1234"}`))
			}
		}))
		defer httptest.Close()

		plugin := ForcemergeStepPluginJira{
			AtlassianUrlDomain: httptest.URL,
			ProjectKey:         "",
			Email:              "serviceaccount@company.com",
			ApiToken:           "123456",
			IssueType:          "Bug",
		}

		prurl, err := url.Parse("https://github.com/mycompany/myrepo/pull/123")
		assert.Nil(t, err)

		returl, err := plugin.Execute(context.Background(), "foo", "explanation", prurl, map[string]interface{}{
			"project": "SRE",
		})

		assert.NotNil(t, err)
		assert.Equal(t, "", returl)
	})
}
