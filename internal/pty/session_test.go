package pty_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ayan-de/agent-board/internal/pty"
)

func TestSessionStateString(t *testing.T) {
	cases := map[pty.SessionState]string{
		pty.StateWaitingReady:  "waiting_ready",
		pty.StateSendingPrompt: "sending_prompt",
		pty.StateWorking:       "working",
		pty.StateDone:          "done",
	}
	for state, want := range cases {
		if state.String() != want {
			t.Fatalf("%v.String() = %q, want %q", state, state.String(), want)
		}
	}
}

func TestSessionDetectsCompletionViaDoneMarker(t *testing.T) {
	reg := pty.NewRegistry()
	cfg := reg["opencode"]

	output := "Some output\n" + pty.DoneMarker + "\nMore output"
	lines := strings.Split(output, "\n")

	detected := pty.DetectCompletion(cfg, lines, 0)
	if !detected {
		t.Fatal("expected completion detection via done marker")
	}
}

func TestSessionNoFalsePositive(t *testing.T) {
	reg := pty.NewRegistry()
	cfg := reg["opencode"]

	lines := []string{"Agent is working hard", "still going", "more work"}
	detected := pty.DetectCompletion(cfg, lines, 3*time.Second)
	if detected {
		t.Fatal("should not detect completion from normal output within grace period")
	}
}

func TestSessionDetectsCompletionViaIdlePattern(t *testing.T) {
	reg := pty.NewRegistry()
	cfg := reg["opencode"]

	lines := []string{
		"Ask anything",
		"Ask anything",
		"Ask anything",
	}

	detected := pty.DetectCompletion(cfg, lines, 5*time.Second)
	if !detected {
		t.Fatal("expected completion detection via idle pattern (3 occurrences)")
	}
}

func TestRecentOutput(t *testing.T) {
	s := &pty.Session{}
	s.AppendOutput("line1")
	s.AppendOutput("line2")
	s.AppendOutput("line3")

	recent := s.RecentOutput(2)
	if len(recent) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(recent))
	}
	if recent[0] != "line2" {
		t.Fatalf("first line = %q, want %q", recent[0], "line2")
	}
	if recent[1] != "line3" {
		t.Fatalf("second line = %q, want %q", recent[1], "line3")
	}
}

func TestRecentOutputMoreThanAvailable(t *testing.T) {
	s := &pty.Session{}
	s.AppendOutput("line1")

	recent := s.RecentOutput(5)
	if len(recent) != 1 {
		t.Fatalf("expected 1 line, got %d", len(recent))
	}
}

func TestRecentOutputEmpty(t *testing.T) {
	s := &pty.Session{}
	recent := s.RecentOutput(5)
	if len(recent) != 0 {
		t.Fatalf("expected 0 lines, got %d", len(recent))
	}
}
