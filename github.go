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
	// createPendingReview posts the draft as one new pending review, pinned to its head, with each comment as a thread.
	createPendingReview(pullRequestID string, reviewDraft draft) (postedReview, error)
}

type pullRequest struct {
	id              string
	url             string
	baseBranch      string
	headSha         string
	myPendingReview *pendingReview
}

type pendingReview struct {
	id           string
	commentCount int
}

type postedReview struct {
	id         string
	commentIDs []string
}

type ghCLI struct{}

const gitHubMaxPageSize = 100

const pullRequestQuery = `query($owner: String!, $name: String!, $number: Int!, $pageSize: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      id
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
				ID          string `json:"id"`
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
	pr := pullRequest{id: found.ID, url: found.URL, baseBranch: found.BaseRefName, headSha: found.HeadRefOid}
	for _, review := range found.Reviews.Nodes {
		if review.ViewerDidAuthor {
			pr.myPendingReview = &pendingReview{id: review.ID, commentCount: review.Comments.TotalCount}
		}
	}
	return pr, nil
}

const addPullRequestReviewMutation = `mutation($input: AddPullRequestReviewInput!, $pageSize: Int!) {
  addPullRequestReview(input: $input) {
    pullRequestReview {
      id
      comments(first: $pageSize) { nodes { id } }
    }
  }
}`

type draftPullRequestReviewThread struct {
	Path      string `json:"path"`
	Body      string `json:"body"`
	Line      int    `json:"line"`
	Side      side   `json:"side"`
	StartLine *int   `json:"startLine,omitempty"`
	StartSide side   `json:"startSide,omitempty"`
}

func draftPullRequestReviewThreads(comments []comment) []draftPullRequestReviewThread {
	threads := make([]draftPullRequestReviewThread, 0, len(comments))
	for _, c := range comments {
		thread := draftPullRequestReviewThread{Path: c.Path, Body: c.Body, Line: c.Line, Side: c.Side, StartLine: c.StartLine}
		if c.StartLine != nil {
			thread.StartSide = c.Side
		}
		threads = append(threads, thread)
	}
	return threads
}

func (gh ghCLI) createPendingReview(pullRequestID string, reviewDraft draft) (postedReview, error) {
	var data struct {
		AddPullRequestReview struct {
			PullRequestReview struct {
				ID       string `json:"id"`
				Comments struct {
					Nodes []struct {
						ID string `json:"id"`
					} `json:"nodes"`
				} `json:"comments"`
			} `json:"pullRequestReview"`
		} `json:"addPullRequestReview"`
	}
	input := map[string]any{
		"pullRequestId": pullRequestID,
		"commitOID":     reviewDraft.Head,
		"threads":       draftPullRequestReviewThreads(reviewDraft.Comments),
	}
	if reviewDraft.Body != "" {
		input["body"] = reviewDraft.Body
	}
	variables := map[string]any{"input": input, "pageSize": gitHubMaxPageSize}
	if err := gh.graphql(addPullRequestReviewMutation, variables, &data); err != nil {
		return postedReview{}, err
	}
	created := data.AddPullRequestReview.PullRequestReview
	review := postedReview{id: created.ID}
	for _, posted := range created.Comments.Nodes {
		review.commentIDs = append(review.commentIDs, posted.ID)
	}
	return review, nil
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
