package main

import (
	"os"
	"testing"
)

func TestOpenReviewedRepo(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	tests := []struct {
		origin string
		// insteadOf, when set, is where git config rewrites origin to.
		insteadOf string
		owner     string
		name      string
	}{
		{"https://github.com/someone/thing.git", "", "someone", "thing"},
		{"https://github.com/someone/thing", "", "someone", "thing"},
		{"git@github.com:someone/thing.git", "", "someone", "thing"},
		{"ssh://git@github.com/someone/thing.git", "", "someone", "thing"},
		{"https://github.com/someone/thing.git", "/some/local/path", "someone", "thing"},
		{"https://gitlab.com/someone/thing.git", "", "", ""},
		{"/some/local/path", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.origin+" "+tt.insteadOf, func(t *testing.T) {
			clone := t.TempDir()
			if _, err := git(clone, "init", "--quiet"); err != nil {
				t.Fatal(err)
			}
			if _, err := git(clone, "remote", "add", "origin", tt.origin); err != nil {
				t.Fatal(err)
			}
			if tt.insteadOf != "" {
				if _, err := git(clone, "config", "url."+tt.insteadOf+".insteadOf", tt.origin); err != nil {
					t.Fatal(err)
				}
			}
			repo, err := openReviewedRepo(clone)
			if tt.owner == "" {
				if err == nil {
					t.Errorf("want an error, got %+v", repo)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if repo.owner != tt.owner || repo.name != tt.name {
				t.Errorf("want %s/%s, got %s/%s", tt.owner, tt.name, repo.owner, repo.name)
			}
		})
	}
}
