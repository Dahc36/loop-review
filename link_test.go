package main

import (
	"strings"
	"testing"
)

func TestCheckLinks(t *testing.T) {
	pr := newFixturePR(t)
	repo := reviewedRepo{clone: pr.clone, owner: "fixture-owner", name: "fixture-repo"}
	blobLink := func(ref, rest string) string {
		return "https://github.com/fixture-owner/fixture-repo/blob/" + ref + "/" + rest
	}
	unknownSha := strings.Repeat("0", 40)

	tests := []struct {
		name  string
		body  string
		valid bool
	}{
		{"no link", "Rename this.", true},
		{"line at the head, which the clone fetches", blobLink(pr.head, "a.txt#L19"), true},
		{"range at the base", blobLink(pr.base, "a.txt#L1-L20"), true},
		{"file with no line", blobLink(pr.head, "b.txt"), true},
		{"file with another fragment", blobLink(pr.head, "b.txt#top"), true},
		{"line with a query", blobLink(pr.head, "a.txt?plain=1#L3"), true},
		{"markdown link", "See [here](" + blobLink(pr.head, "a.txt#L2") + ").", true},
		{"owner and repo in another case", strings.Replace(blobLink(pr.head, "a.txt#L2"), "fixture-owner/fixture-repo", "Fixture-Owner/Fixture-Repo", 1), true},
		{"link into another repo", "https://github.com/someone/else/blob/main/x.txt#L99", true},
		{"link into the repo outside blob", "https://github.com/fixture-owner/fixture-repo/pull/2", true},
		{"line past the end", blobLink(pr.head, "a.txt#L20"), false},
		{"line past the end, ending a sentence", "See " + blobLink(pr.head, "a.txt#L20") + ".", false},
		{"range ending past the end", blobLink(pr.head, "a.txt#L18-L20"), false},
		{"line zero", blobLink(pr.head, "a.txt#L0"), false},
		{"line past the end, owner in another case", strings.Replace(blobLink(pr.head, "a.txt#L20"), "fixture-owner", "Fixture-Owner", 1), false},
		{"file missing at the sha", blobLink(pr.head, "c.txt#L1"), false},
		{"sha with no file", blobLink(pr.head, ""), false},
		{"branch instead of a sha", blobLink("main", "a.txt#L1"), false},
		{"short sha", blobLink(pr.head[:7], "a.txt#L1"), false},
		{"sha not on origin", blobLink(unknownSha, "a.txt#L1"), false},
		{"one bad link among good ones", blobLink(pr.head, "a.txt#L1") + " and " + blobLink(pr.head, "a.txt#L20"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.checkLinks(comment{Path: "a.txt", Line: 10, Side: right, Body: tt.body})
			if tt.valid && err != nil {
				t.Errorf("want valid, got %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("want invalid, got valid")
			}
		})
	}
}
