package main

import "testing"

func TestCheckAnchor(t *testing.T) {
	pr := newFixturePR(t)
	touched, err := readTouchedLines(pr.clone, "main", pr.head)
	if err != nil {
		t.Fatal(err)
	}

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
