package main

type side string

const (
	right side = "RIGHT"
	left  side = "LEFT"
)

type comment struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	StartLine *int   `json:"startLine,omitempty"`
	Side      side   `json:"side"`
	Body      string `json:"body"`
}
