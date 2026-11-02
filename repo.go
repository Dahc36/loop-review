package main

import (
	"fmt"
	"regexp"
	"strings"
)

// reviewedRepo is the GitHub repo a review is posted to, read through its local clone.
type reviewedRepo struct {
	clone string
	owner string
	name  string
}

var gitHubRemote = regexp.MustCompile(`^(?:https://github\.com/|git@github\.com:|ssh://git@github\.com/)([^/]+)/([^/]+?)(?:\.git)?/?$`)

func openReviewedRepo(clone string) (reviewedRepo, error) {
	origin, err := git(clone, "config", "--get", "remote.origin.url")
	if err != nil {
		return reviewedRepo{}, err
	}
	origin = strings.TrimSpace(origin)
	m := gitHubRemote.FindStringSubmatch(origin)
	if m == nil {
		return reviewedRepo{}, fmt.Errorf("origin %s of %s is not a GitHub repo", origin, clone)
	}
	return reviewedRepo{clone: clone, owner: m[1], name: m[2]}, nil
}
