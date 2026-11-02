package main

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestCheckAnchor(t *testing.T) {
	touched := fixtureTouchedLines(t)

	tests := []struct {
		name    string
		comment comment
		valid   bool
	}{
		{"added line", comment{Path: "a.txt", Line: 10, Side: right}, true},
		{"context line on RIGHT", comment{Path: "a.txt", Line: 7, Side: right}, true},
		{"removed line", comment{Path: "a.txt", Line: 15, Side: left}, true},
		{"line of an added file", comment{Path: "b.txt", Line: 2, Side: right}, true},
		{"line of a deleted file", comment{Path: "c.txt", Line: 1, Side: left}, true},
		{"range of touched lines", comment{Path: "a.txt", StartLine: new(8), Line: 12, Side: right}, true},
		{"line outside the hunks", comment{Path: "a.txt", Line: 6, Side: right}, false},
		{"context line on LEFT", comment{Path: "a.txt", Line: 7, Side: left}, false},
		{"range reaching outside the hunks", comment{Path: "a.txt", StartLine: new(5), Line: 8, Side: right}, false},
		{"startLine after line", comment{Path: "a.txt", StartLine: new(12), Line: 8, Side: right}, false},
		{"unknown side", comment{Path: "a.txt", Line: 10, Side: "BOTH"}, false},
		{"file the PR does not touch", comment{Path: "untouched.txt", Line: 1, Side: right}, false},
		{"file changed on base after the PR branched", comment{Path: "d.txt", Line: 1, Side: right}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := touched.checkAnchor(tt.comment)
			if tt.valid && err != nil {
				t.Errorf("want valid, got %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("want invalid, got valid")
			}
		})
	}
}

// fixtureTouchedLines builds an origin with a PR branch and a clone of it taken before the PR,
// then reads the PR's touched lines from the clone.
//
// The PR changes line 10 and removes line 15 of a.txt's 20 lines, adds b.txt and deletes c.txt.
// After the PR branches off, main adds d.txt.
func fixtureTouchedLines(t *testing.T) touchedLines {
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

	touched, err := readTouchedLines(clone, "main", head)
	if err != nil {
		t.Fatal(err)
	}
	return touched
}
