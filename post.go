package main

import (
	"encoding/json"
	"fmt"
	"io"
)

func post(github gitHub, clone string, number int, draftPath string, stdout io.Writer) error {
	reviewDraft, err := readDraft(draftPath)
	if err != nil {
		return err
	}
	repo, err := openReviewedRepo(clone)
	if err != nil {
		return err
	}
	pr, err := github.pullRequest(repo, number)
	if err != nil {
		return err
	}
	if pending := pr.myPendingReview; pending != nil {
		return fmt.Errorf("%s already has my pending review %s, holding %d comments", pr.url, pending.id, pending.commentCount)
	}
	// GitHub pins a review to any commitOID, even one the PR has moved past, without marking its comments outdated.
	if pr.headSha != reviewDraft.Head {
		return fmt.Errorf("%s is at %s, but the draft was written against %s", pr.url, pr.headSha, reviewDraft.Head)
	}
	touched, err := readTouchedLines(clone, pr.baseBranch, reviewDraft.Head)
	if err != nil {
		return err
	}
	if err := validateComments(repo, touched, reviewDraft.Comments); err != nil {
		return err
	}
	review, err := github.createPendingReview(pr.id, reviewDraft.Head, reviewDraft.Comments)
	if err != nil {
		return err
	}
	if len(review.commentIDs) != len(reviewDraft.Comments) {
		return fmt.Errorf("pending review %s on %s holds %d comments, but the draft has %d",
			review.id, pr.url, len(review.commentIDs), len(reviewDraft.Comments))
	}
	commentIDs := review.commentIDs
	if commentIDs == nil {
		commentIDs = []string{}
	}
	return json.NewEncoder(stdout).Encode(struct {
		ReviewID   string   `json:"reviewId"`
		URL        string   `json:"url"`
		CommentIDs []string `json:"commentIds"`
	}{review.id, pr.url, commentIDs})
}
