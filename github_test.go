package main

import (
	"reflect"
	"testing"
)

func TestDraftPullRequestReviewThreads(t *testing.T) {
	comments := []comment{
		{Path: "a.txt", Line: 10, Side: right, Body: "Rename this."},
		{Path: "a.txt", StartLine: new(15), Line: 16, Side: left, Body: "Why remove these?"},
	}
	want := []draftPullRequestReviewThread{
		{Path: "a.txt", Line: 10, Side: right, Body: "Rename this."},
		{Path: "a.txt", StartLine: new(15), StartSide: left, Line: 16, Side: left, Body: "Why remove these?"},
	}
	if got := draftPullRequestReviewThreads(comments); !reflect.DeepEqual(got, want) {
		t.Errorf("want %+v, got %+v", want, got)
	}
}
