package main

import (
	"errors"
	"fmt"
)

func post(github gitHub, clone string, number int, draftPath string) error {
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
	return errors.New("posting is not implemented yet")
}
