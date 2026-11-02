package main

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// fixturePR is a made-up PR: an origin with a PR branch, and a clone of the origin taken before the PR.
type fixturePR struct {
	clone string
	// base is the commit the PR branches off, which the clone has.
	base string
	// head is the PR's head, which the clone lacks until fetched.
	head string
}

// newFixturePR builds a PR that changes line 10 and removes line 15 of a.txt's 20 lines,
// adds b.txt and deletes c.txt.
// After the PR branches off, main adds d.txt.
func newFixturePR(t *testing.T) fixturePR {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Fixture")
	t.Setenv("GIT_AUTHOR_EMAIL", "fixture@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Fixture")
	t.Setenv("GIT_COMMITTER_EMAIL", "fixture@example.com")

	origin := t.TempDir()
	clone := t.TempDir()
	mustGit := func(dir string, args ...string) string {
		t.Helper()
		out, err := git(dir, args...)
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(out)
	}
	write := func(name string, lines ...string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(origin, name), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var aLines []string
	for n := 1; n <= 20; n++ {
		aLines = append(aLines, strconv.Itoa(n))
	}
	mustGit(origin, "init", "--quiet", "--initial-branch=main")
	write("a.txt", aLines...)
	write("c.txt", "gone")
	mustGit(origin, "add", ".")
	mustGit(origin, "commit", "--quiet", "--message=base")
	base := mustGit(origin, "rev-parse", "HEAD")
	mustGit(origin, "clone", "--quiet", origin, clone)

	mustGit(origin, "switch", "--quiet", "--create", "pr")
	prLines := slices.Clone(aLines)
	prLines[10-1] = "ten"
	prLines = slices.Delete(prLines, 15-1, 15)
	write("a.txt", prLines...)
	write("b.txt", "new 1", "new 2")
	mustGit(origin, "rm", "--quiet", "c.txt")
	mustGit(origin, "add", ".")
	mustGit(origin, "commit", "--quiet", "--message=pr")
	head := mustGit(origin, "rev-parse", "HEAD")

	mustGit(origin, "switch", "--quiet", "main")
	write("d.txt", "later")
	mustGit(origin, "add", ".")
	mustGit(origin, "commit", "--quiet", "--message=later on main")

	return fixturePR{clone: clone, base: base, head: head}
}
