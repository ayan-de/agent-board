package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectAgents(t *testing.T) {
	agents := DetectAgents()
	if len(agents) != 3 {
		t.Fatalf("DetectAgents() returned %d agents, want 3", len(agents))
	}

	names := map[string]bool{}
	for _, a := range agents {
		names[a.Name] = true
		if a.Binary == "" {
			t.Errorf("agent %q has empty Binary", a.Name)
		}
	}

	for _, want := range []string{"claude-code", "opencode", "cursor"} {
		if !names[want] {
			t.Errorf("missing agent %q", want)
		}
	}
}

func TestDetectAgentsFoundWithBinaryOnPath(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "claude")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh"), 0755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", dir)
	agents := DetectAgents()

	found := false
	for _, a := range agents {
		if a.Binary == "claude" && a.Found {
			found = true
		}
	}
	if !found {
		t.Error("claude not detected as found with fake binary on PATH")
	}
}

func TestDetectAgentsNotFoundWhenMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	agents := DetectAgents()

	for _, a := range agents {
		if a.Found {
			t.Errorf("agent %q should not be found with empty PATH", a.Name)
		}
	}
}
