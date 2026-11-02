package main

import (
	"encoding/json"
	"strings"
	"testing"
)

type fakeGitHub struct {
	pr          pullRequest
	askedRepo   reviewedRepo
	askedNumber int
}

func (f *fakeGitHub) pullRequest(repo reviewedRepo, number int) (pullRequest, error) {
	f.askedRepo, f.askedNumber = repo, number
	return f.pr, nil
}

const fixtureNumber = 2

func fixtureGitHub(pr fixturePR) *fakeGitHub {
	return &fakeGitHub{pr: pullRequest{
		url:        "https://github.com/fixture-owner/fixture-repo/pull/2",
		baseBranch: "main",
		headSha:    pr.head,
	}}
}

func writeDraftOf(t *testing.T, d draft) string {
	t.Helper()
	content, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return writeDraft(t, string(content))
}

func TestPostAsksForThePROfTheClonesOrigin(t *testing.T) {
	pr := newFixturePR(t)
	github := fixtureGitHub(pr)
	_ = post(github, pr.clone, fixtureNumber, writeDraftOf(t, draft{Head: pr.head}))
	if github.askedRepo.owner != "fixture-owner" || github.askedRepo.name != "fixture-repo" || github.askedNumber != fixtureNumber {
		t.Errorf("want fixture-owner/fixture-repo#%d, got %s/%s#%d", fixtureNumber, github.askedRepo.owner, github.askedRepo.name, github.askedNumber)
	}
}

func TestPostRefuses(t *testing.T) {
	pr := newFixturePR(t)
	validComments := []comment{{Path: "a.txt", Line: 10, Side: right, Body: "Rename this."}}

	tests := []struct {
		name      string
		pending   *pendingReview
		head      string
		comments  []comment
		wantNamed []string
	}{
		{
			name:      "my pending review on the PR",
			pending:   &pendingReview{id: "PRR_mine", commentCount: 3},
			head:      pr.head,
			comments:  validComments,
			wantNamed: []string{"PRR_mine", "3 comments"},
		},
		{
			name:      "PR head other than the draft's",
			head:      pr.base,
			comments:  validComments,
			wantNamed: []string{pr.head, pr.base},
		},
		{
			name:      "invalid comment",
			head:      pr.head,
			comments:  append([]comment{{Path: "a.txt", Line: 6, Side: right, Body: "Outside the hunks."}}, validComments...),
			wantNamed: []string{"comment 1 (a.txt line 6)"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			github := fixtureGitHub(pr)
			github.pr.myPendingReview = tt.pending
			err := post(github, pr.clone, fixtureNumber, writeDraftOf(t, draft{Head: tt.head, Comments: tt.comments}))
			if err == nil {
				t.Fatal("want refused, got posted")
			}
			for _, want := range tt.wantNamed {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("want %q named in %q", want, err)
				}
			}
		})
	}
}
