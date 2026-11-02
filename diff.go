package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// touchedLines holds, for each path and side, the line numbers a review comment can anchor on.
type touchedLines map[string]map[side]map[int]bool

func (t touchedLines) add(path string, s side, line int) {
	if t[path] == nil {
		t[path] = map[side]map[int]bool{}
	}
	if t[path][s] == nil {
		t[path][s] = map[int]bool{}
	}
	t[path][s][line] = true
}

// githubContextLines is how many unchanged lines GitHub shows around each change.
const githubContextLines = 3

func readTouchedLines(clone, base, head string) (touchedLines, error) {
	baseRef := "refs/remotes/origin/" + base
	if _, err := git(clone, "fetch", "--quiet", "origin", "+refs/heads/"+base+":"+baseRef, head); err != nil {
		return nil, err
	}
	// The format flags override whatever the user's git config sets.
	diff, err := git(clone, "diff", "--no-color", "--no-ext-diff", "--src-prefix=a/", "--dst-prefix=b/",
		"--unified="+strconv.Itoa(githubContextLines), baseRef+"..."+head)
	if err != nil {
		return nil, err
	}
	return parseDiff(diff)
}

func git(dir string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

var hunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

func parseDiff(diff string) (touchedLines, error) {
	touched := touchedLines{}
	var oldPath, path string
	var leftLine, rightLine, leftRemaining, rightRemaining int
	for _, line := range strings.Split(diff, "\n") {
		if leftRemaining > 0 || rightRemaining > 0 {
			switch {
			case strings.HasPrefix(line, "+"):
				touched.add(path, right, rightLine)
				rightLine++
				rightRemaining--
			case strings.HasPrefix(line, "-"):
				touched.add(path, left, leftLine)
				leftLine++
				leftRemaining--
			case strings.HasPrefix(line, " "):
				touched.add(path, right, rightLine)
				rightLine++
				rightRemaining--
				leftLine++
				leftRemaining--
			}
			continue
		}
		switch {
		case strings.HasPrefix(line, "--- "):
			p, err := diffPath(strings.TrimPrefix(line, "--- "), "a/")
			if err != nil {
				return nil, err
			}
			oldPath = p
		case strings.HasPrefix(line, "+++ "):
			p, err := diffPath(strings.TrimPrefix(line, "+++ "), "b/")
			if err != nil {
				return nil, err
			}
			path = p
			if path == "" {
				path = oldPath
			}
		case strings.HasPrefix(line, "@@ "):
			m := hunkHeader.FindStringSubmatch(line)
			if m == nil {
				return nil, fmt.Errorf("unreadable hunk header %q", line)
			}
			leftLine, leftRemaining = hunkRange(m[1], m[2])
			rightLine, rightRemaining = hunkRange(m[3], m[4])
		}
	}
	return touched, nil
}

func diffPath(header, prefix string) (string, error) {
	// git ends a path holding a space with a tab.
	header = strings.TrimSuffix(header, "\t")
	if header == "/dev/null" {
		return "", nil
	}
	if strings.HasPrefix(header, `"`) {
		unquoted, err := strconv.Unquote(header)
		if err != nil {
			return "", fmt.Errorf("unreadable path %s: %w", header, err)
		}
		header = unquoted
	}
	return strings.TrimPrefix(header, prefix), nil
}

func hunkRange(startText, lengthText string) (start, length int) {
	start, _ = strconv.Atoi(startText)
	if lengthText == "" {
		return start, 1
	}
	length, _ = strconv.Atoi(lengthText)
	return start, length
}
