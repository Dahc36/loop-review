package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type draft struct {
	// Head is the sha the review was written against.
	Head string `json:"head"`
	// Body, when set, is the review's overall comment.
	Body     string    `json:"body,omitempty"`
	Comments []comment `json:"comments"`
}

func readDraft(path string) (draft, error) {
	file, err := os.Open(path)
	if err != nil {
		return draft{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var parsed draft
	if err := decoder.Decode(&parsed); err != nil {
		return draft{}, fmt.Errorf("draft %s: %w", path, err)
	}
	if parsed.Head == "" {
		return draft{}, fmt.Errorf("draft %s has no head", path)
	}
	return parsed, nil
}

func validateComments(repo reviewedRepo, touched touchedLines, comments []comment) error {
	var invalid []error
	for i, c := range comments {
		if err := validateComment(repo, touched, c); err != nil {
			invalid = append(invalid, fmt.Errorf("comment %d (%s line %d): %w", i+1, c.Path, c.Line, err))
		}
	}
	return errors.Join(invalid...)
}

func validateComment(repo reviewedRepo, touched touchedLines, c comment) error {
	if err := touched.checkAnchor(c); err != nil {
		return err
	}
	if err := checkSuggestion(c); err != nil {
		return err
	}
	return repo.checkLinks(c)
}
