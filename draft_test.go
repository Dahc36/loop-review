package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeDraft(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "draft.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadDraft(t *testing.T) {
	t.Run("head and comments, one a range", func(t *testing.T) {
		path := writeDraft(t, `{"head": "abc", "comments": [
			{"path": "a.txt", "line": 10, "side": "RIGHT", "body": "Rename this."},
			{"path": "a.txt", "startLine": 15, "line": 16, "side": "LEFT", "body": "Why remove these?"}
		]}`)
		got, err := readDraft(path)
		if err != nil {
			t.Fatal(err)
		}
		want := draft{Head: "abc", Comments: []comment{
			{Path: "a.txt", Line: 10, Side: right, Body: "Rename this."},
			{Path: "a.txt", StartLine: new(15), Line: 16, Side: left, Body: "Why remove these?"},
		}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("want %+v, got %+v", want, got)
		}
	})

	invalid := []struct {
		name    string
		content string
	}{
		{"no head", `{"comments": []}`},
		{"unknown field", `{"head": "abc", "comments": [{"path": "a.txt", "start_line": 8, "line": 10, "side": "RIGHT", "body": "Rename these."}]}`},
		{"not JSON", `head: abc`},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := readDraft(writeDraft(t, tt.content)); err == nil {
				t.Error("want invalid, got valid")
			}
		})
	}
}

func TestValidateComments(t *testing.T) {
	pr := newFixturePR(t)
	touched, err := readTouchedLines(pr.clone, "main", pr.head)
	if err != nil {
		t.Fatal(err)
	}
	repo := reviewedRepo{clone: pr.clone, owner: "fixture-owner", name: "fixture-repo"}
	suggestion := "```suggestion\nten\n```"

	t.Run("every comment valid", func(t *testing.T) {
		comments := []comment{
			{Path: "a.txt", Line: 10, Side: right, Body: suggestion},
			{Path: "a.txt", Line: 15, Side: left, Body: "Why remove this?"},
		}
		if err := validateComments(repo, touched, comments); err != nil {
			t.Errorf("want valid, got %v", err)
		}
	})

	t.Run("one line per invalid comment, in draft order", func(t *testing.T) {
		comments := []comment{
			{Path: "a.txt", Line: 6, Side: right, Body: "Outside the hunks."},
			{Path: "a.txt", Line: 10, Side: right, Body: "Fine."},
			{Path: "a.txt", Line: 15, Side: left, Body: suggestion},
			{Path: "b.txt", Line: 1, Side: right, Body: "https://github.com/fixture-owner/fixture-repo/blob/main/a.txt#L1"},
		}
		err := validateComments(repo, touched, comments)
		if err == nil {
			t.Fatal("want invalid, got valid")
		}
		lines := strings.Split(err.Error(), "\n")
		wantPrefixes := []string{"comment 1 (a.txt line 6): ", "comment 3 (a.txt line 15): ", "comment 4 (b.txt line 1): "}
		if len(lines) != len(wantPrefixes) {
			t.Fatalf("want %d lines, got %q", len(wantPrefixes), lines)
		}
		for i, want := range wantPrefixes {
			if !strings.HasPrefix(lines[i], want) {
				t.Errorf("line %d: want prefix %q, got %q", i+1, want, lines[i])
			}
		}
	})
}
