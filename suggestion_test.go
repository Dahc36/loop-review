package main

import (
	"strings"
	"testing"
)

func TestCheckSuggestion(t *testing.T) {
	suggestion := "```suggestion\nten\n```"
	tests := []struct {
		name  string
		side  side
		body  string
		valid bool
	}{
		{"no suggestion block", right, "Rename this.\n```go\nx := 1\n```", true},
		{"one suggestion block on RIGHT", right, "Rename this.\n" + suggestion, true},
		{"one suggestion block on LEFT", left, suggestion, false},
		{"two suggestion blocks", right, suggestion + "\n" + suggestion, false},
		{"tilde-fenced suggestion block on LEFT", left, "~~~suggestion\nten\n~~~", false},
		{"suggestion block indented by three spaces on LEFT", left, "   ```suggestion\nten\n   ```", false},
		{"two suggestion blocks with CRLF line ends", right, strings.ReplaceAll(suggestion+"\n"+suggestion, "\n", "\r\n"), false},
		{"suggestion fence quoted inside a longer fence", left, "````markdown\n" + suggestion + "\n````", true},
		{"suggestion fence indented by four spaces", left, "    ```suggestion\n    ten\n    ```", true},
		{"inline code at the start of a line, with backticks in its info string", left, "```suggestion``` is how GitHub spells it.", true},
		{"other info string starting with suggestion", left, "```suggestions\nten\n```", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkSuggestion(comment{Path: "a.txt", Line: 10, Side: tt.side, Body: tt.body})
			if tt.valid && err != nil {
				t.Errorf("want valid, got %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("want invalid, got valid")
			}
		})
	}
}
