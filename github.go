package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// gitHub is everything post needs from GitHub, so that how GitHub is reached stays out of what gets posted.
type gitHub interface {
	pullRequest(repo reviewedRepo, number int) (pullRequest, error)
}

type pullRequest struct {
	url             string
	baseBranch      string
	headSha         string
	myPendingReview *pendingReview
}

type pendingReview struct {
	id           string
	commentCount int
}

type ghCLI struct{}

const gitHubMaxPageSize = 100

const pullRequestQuery = `query($owner: String!, $name: String!, $number: Int!, $pageSize: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      url
      baseRefName
      headRefOid
      reviews(states: PENDING, first: $pageSize) {
        nodes { id viewerDidAuthor comments { totalCount } }
      }
    }
  }
}`

func (gh ghCLI) pullRequest(repo reviewedRepo, number int) (pullRequest, error) {
	var data struct {
		Repository struct {
			PullRequest struct {
				URL         string `json:"url"`
				BaseRefName string `json:"baseRefName"`
				HeadRefOid  string `json:"headRefOid"`
				Reviews     struct {
					Nodes []struct {
						ID              string `json:"id"`
						ViewerDidAuthor bool   `json:"viewerDidAuthor"`
						Comments        struct {
							TotalCount int `json:"totalCount"`
						} `json:"comments"`
					} `json:"nodes"`
				} `json:"reviews"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}
	variables := map[string]any{"owner": repo.owner, "name": repo.name, "number": number, "pageSize": gitHubMaxPageSize}
	if err := gh.graphql(pullRequestQuery, variables, &data); err != nil {
		return pullRequest{}, err
	}
	found := data.Repository.PullRequest
	pr := pullRequest{url: found.URL, baseBranch: found.BaseRefName, headSha: found.HeadRefOid}
	for _, review := range found.Reviews.Nodes {
		if review.ViewerDidAuthor {
			pr.myPendingReview = &pendingReview{id: review.ID, commentCount: review.Comments.TotalCount}
		}
	}
	return pr, nil
}

func (ghCLI) graphql(query string, variables map[string]any, data any) error {
	request, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		return err
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gh", "api", "graphql", "--input", "-")
	cmd.Stdin = bytes.NewReader(request)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh api graphql: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var response struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return fmt.Errorf("gh api graphql: %w", err)
	}
	return json.Unmarshal(response.Data, data)
}
