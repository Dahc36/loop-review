package main

import (
	"fmt"
	"regexp"
	"strings"
)

// checkSuggestion leaves coverage to checkAnchor: on RIGHT, the touched lines are exactly
// the added and context lines at head, which GitHub can apply a suggestion to.
func checkSuggestion(c comment) error {
	switch n := countSuggestionBlocks(c.Body); {
	case n > 1:
		return fmt.Errorf("%d suggestion blocks, but GitHub applies only one per comment", n)
	case n == 1 && c.Side == left:
		return fmt.Errorf("suggestion block on %s, where GitHub cannot apply it", left)
	}
	return nil
}

var codeFence = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})(.*)$")

func countSuggestionBlocks(body string) int {
	count := 0
	var openFence string
	for _, line := range strings.Split(body, "\n") {
		m := codeFence.FindStringSubmatch(strings.TrimSuffix(line, "\r"))
		if m == nil {
			continue
		}
		fence, rest := m[1], m[2]
		if openFence != "" {
			if fence[0] == openFence[0] && len(fence) >= len(openFence) && strings.TrimSpace(rest) == "" {
				openFence = ""
			}
			continue
		}
		if fence[0] == '`' && strings.Contains(rest, "`") {
			continue
		}
		openFence = fence
		if info := strings.Fields(rest); len(info) > 0 && info[0] == "suggestion" {
			count++
		}
	}
	return count
}
