package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Installation struct {
	ID      int64  `json:"id"`
	AppId   int64  `json:"app_id"`
	AppSlug string `json:"app_slug"`
	Account struct {
		Login string `json:"login"`
	} `json:"account"`
}

func (client *GitHubClientImpl) getInstallations(jwt string) ([]Installation, error) {
	req, err := http.NewRequest("GET", client.gitHubServer+"/app/installations", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+jwt)
	req.Header.Add("Accept", "application/vnd.github.machine-man-preview+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var installations []Installation
	// read body into a string
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read body when getting Github installation id: %v", err)
	}

	err = json.Unmarshal(body, &installations)
	if err != nil {
		return nil, fmt.Errorf("when trying to get Github installation id: unable to decode %s: %v", body, err)
	}

	return installations, nil
}
