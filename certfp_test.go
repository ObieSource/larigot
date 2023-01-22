package main

import (
	"testing"
	"time"
)

var MutedStatusCases = map[string]MutedStatus{
	"permanently muted":                  MutedStatus{true, 0},
	"temporarily muted (1 minute)":       MutedStatus{false, time.Second * 60},
	"temporarily muted (1 day)":          MutedStatus{false, time.Hour * 24},
	"temporarily muted (4 weeks 2 days)": MutedStatus{false, time.Hour * 24 * 30},
}

func TestMutedStatus(t *testing.T) {
	for expected, input := range MutedStatusCases {
		res := input.String()
		if res != expected {
			t.Errorf("For %#v, expected %q recieved %q", input, expected, res)
		}
	}
}
