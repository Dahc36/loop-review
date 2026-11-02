package main

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// reviewedRepo is the GitHub repo a review is posted to, read through its local clone.
type reviewedRepo struct {
	clone string
	owner string
	name  string
}

var githubURL = regexp.MustCompile(`https?://github\.com/[^\s<>()\[\]]+`)

const sentencePunctuation = ".,;:!?'\""

func (r reviewedRepo) checkLinks(c comment) error {
	for _, link := range githubURL.FindAllString(c.Body, -1) {
		link = strings.TrimRight(link, sentencePunctuation)
		if err := r.checkLink(link); err != nil {
			return fmt.Errorf("link %s: %w", link, err)
		}
	}
	return nil
}

var (
	blobPath     = regexp.MustCompile(`^/([^/]+)/([^/]+)/blob/([^/]*)/?(.*)$`)
	commitSha    = regexp.MustCompile(`^[0-9a-f]{40}$`)
	lineFragment = regexp.MustCompile(`^L(\d+)(?:-L(\d+))?$`)
)

func (r reviewedRepo) checkLink(link string) error {
	parsed, err := url.Parse(link)
	if err != nil {
		return err
	}
	m := blobPath.FindStringSubmatch(parsed.EscapedPath())
	if m == nil || !strings.EqualFold(m[1], r.owner) || !strings.EqualFold(m[2], r.name) {
		return nil
	}
	sha, escapedPath := m[3], m[4]
	if !commitSha.MatchString(sha) {
		return fmt.Errorf("ref %q is not a full commit sha", sha)
	}
	if escapedPath == "" {
		return errors.New("names no file")
	}
	path, err := url.PathUnescape(escapedPath)
	if err != nil {
		return err
	}

	if err := r.fetchCommit(sha); err != nil {
		return err
	}
	content, err := git(r.clone, "cat-file", "blob", sha+":"+path)
	if err != nil {
		return fmt.Errorf("%s is not a file at %s", path, sha)
	}

	lines := lineFragment.FindStringSubmatch(parsed.Fragment)
	if lines == nil {
		return nil
	}
	start, end := lines[1], lines[2]
	if end == "" {
		end = start
	}
	lineCount := countLines(content)
	for _, lineText := range []string{start, end} {
		line, _ := strconv.Atoi(lineText)
		if line < 1 || line > lineCount {
			return fmt.Errorf("line %d is outside %s, which has %d lines at %s", line, path, lineCount, sha)
		}
	}
	return nil
}

func (r reviewedRepo) fetchCommit(sha string) error {
	if _, err := git(r.clone, "cat-file", "-e", sha+"^{commit}"); err == nil {
		return nil
	}
	if _, err := git(r.clone, "fetch", "--quiet", "origin", sha); err != nil {
		return fmt.Errorf("sha %s is not on origin", sha)
	}
	return nil
}

func countLines(content string) int {
	count := strings.Count(content, "\n")
	if content != "" && !strings.HasSuffix(content, "\n") {
		count++
	}
	return count
}
