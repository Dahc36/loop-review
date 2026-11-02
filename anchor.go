package main

import "fmt"

func (t touchedLines) checkAnchor(c comment) error {
	if c.Side != right && c.Side != left {
		return fmt.Errorf("side %q is neither %s nor %s", c.Side, right, left)
	}
	first := c.Line
	if c.StartLine != nil {
		first = *c.StartLine
	}
	if first > c.Line {
		return fmt.Errorf("startLine %d is after line %d", first, c.Line)
	}
	for line := first; line <= c.Line; line++ {
		if !t[c.Path][c.Side][line] {
			return fmt.Errorf("line %d on %s is not a touched line", line, c.Side)
		}
	}
	return nil
}
