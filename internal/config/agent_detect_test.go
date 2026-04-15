package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectAgents(t *testing.T) {
	agents := DetectAgents()
	if len(agents) != 4 {
		t.Fatalf("DetectAgents() returned %d agents, want 4", len(agents))
	}

	names := map[string]bool{}
	for _, a := range agents {
		names[a.Name] = true
		if a.Binary == "" {
			t.Errorf("agent %q has empty Binary", a.Name)
		}
	}

	for _, want := range []string{"claude-code", "opencode", "codex", "cursor"} {
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

func TestAgentColor(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"claude-code", "#D97757"},
		{"opencode", "#808080"},
		{"codex", "#10A37F"},
		{"cursor", "#F0DB4F"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := AgentColor(tt.name)
		if got != tt.want {
			t.Errorf("AgentColor(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
