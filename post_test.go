package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
)

type fakeGitHub struct {
	pr          pullRequest
	askedRepo   reviewedRepo
	askedNumber int

	posted            bool
	postedPR          string
	postedDraft       draft
	droppedCommentIDs int
}

func (f *fakeGitHub) pullRequest(repo reviewedRepo, number int) (pullRequest, error) {
	f.askedRepo, f.askedNumber = repo, number
	return f.pr, nil
}

func (f *fakeGitHub) createPendingReview(pullRequestID string, reviewDraft draft) (postedReview, error) {
	f.posted, f.postedPR, f.postedDraft = true, pullRequestID, reviewDraft
	review := postedReview{id: "PRR_posted"}
	for i := range len(reviewDraft.Comments) - f.droppedCommentIDs {
		review.commentIDs = append(review.commentIDs, fmt.Sprintf("PRRC_%d", i+1))
	}
	return review, nil
}

const fixtureNumber = 2

func fixtureGitHub(pr fixturePR) *fakeGitHub {
	return &fakeGitHub{pr: pullRequest{
		id:         "PR_fixture",
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
	if err := post(github, pr.clone, fixtureNumber, writeDraftOf(t, draft{Head: pr.head}), io.Discard); err != nil {
		t.Fatal(err)
	}
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
			err := post(github, pr.clone, fixtureNumber, writeDraftOf(t, draft{Head: tt.head, Comments: tt.comments}), io.Discard)
			if err == nil {
				t.Fatal("want refused, got posted")
			}
			if github.posted {
				t.Error("want nothing posted, got a pending review")
			}
			for _, want := range tt.wantNamed {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("want %q named in %q", want, err)
				}
			}
		})
	}
}

func TestPostCreatesThePendingReview(t *testing.T) {
	pr := newFixturePR(t)
	reviewDraft := draft{Head: pr.head, Body: "A few renames.", Comments: []comment{
		{Path: "a.txt", Line: 10, Side: right, Body: "Rename this."},
		{Path: "a.txt", StartLine: new(8), Line: 12, Side: right, Body: "Split these."},
		{Path: "a.txt", Line: 15, Side: left, Body: "Why remove this?"},
	}}
	github := fixtureGitHub(pr)
	var stdout bytes.Buffer
	if err := post(github, pr.clone, fixtureNumber, writeDraftOf(t, reviewDraft), &stdout); err != nil {
		t.Fatal(err)
	}

	if github.postedPR != "PR_fixture" || !reflect.DeepEqual(github.postedDraft, reviewDraft) {
		t.Errorf("want %+v posted to PR_fixture, got %+v to %s", reviewDraft, github.postedDraft, github.postedPR)
	}
	var printed map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &printed); err != nil {
		t.Fatalf("want one JSON object, got %q: %v", stdout.String(), err)
	}
	want := map[string]any{
		"reviewId":   "PRR_posted",
		"url":        "https://github.com/fixture-owner/fixture-repo/pull/2",
		"commentIds": []any{"PRRC_1", "PRRC_2", "PRRC_3"},
	}
	if !reflect.DeepEqual(printed, want) {
		t.Errorf("want %v, got %v", want, printed)
	}
}

func TestPostPrintsNoCommentIdsForADraftWithoutComments(t *testing.T) {
	pr := newFixturePR(t)
	var stdout bytes.Buffer
	if err := post(fixtureGitHub(pr), pr.clone, fixtureNumber, writeDraftOf(t, draft{Head: pr.head}), &stdout); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"commentIds":[]`) {
		t.Errorf("want an empty commentIds list, got %q", stdout.String())
	}
}

func TestPostFailsOnAReviewMissingComments(t *testing.T) {
	pr := newFixturePR(t)
	comments := []comment{
		{Path: "a.txt", Line: 10, Side: right, Body: "Rename this."},
		{Path: "a.txt", Line: 15, Side: left, Body: "Why remove this?"},
	}
	github := fixtureGitHub(pr)
	github.droppedCommentIDs = 1
	var stdout bytes.Buffer
	err := post(github, pr.clone, fixtureNumber, writeDraftOf(t, draft{Head: pr.head, Comments: comments}), &stdout)
	if err == nil || !strings.Contains(err.Error(), "PRR_posted") {
		t.Errorf("want an error naming PRR_posted, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("want nothing printed, got %q", stdout.String())
	}
}
