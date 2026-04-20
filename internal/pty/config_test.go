package pty_test

import (
	"testing"

	"github.com/ayan-de/agent-board/internal/pty"
)

func TestNewRegistryContainsAllAgents(t *testing.T) {
	r := pty.NewRegistry()

	agents := []string{"opencode", "claudecode", "codex"}
	for _, name := range agents {
		cfg, ok := r[name]
		if !ok {
			t.Fatalf("expected agent %q in registry", name)
		}
		if cfg.Name == "" {
			t.Fatalf("agent %q has empty Name", name)
		}
		if cfg.Bin == "" {
			t.Fatalf("agent %q has empty Bin", name)
		}
		if cfg.ReadyPattern == "" {
			t.Fatalf("agent %q has empty ReadyPattern", name)
		}
		if cfg.SendPrompt == nil {
			t.Fatalf("agent %q has nil SendPrompt", name)
		}
		if cfg.GracePeriod == 0 {
			t.Fatalf("agent %q has zero GracePeriod", name)
		}
	}
}

func TestStripANSI(t *testing.T) {
	input := "\x1b[32mhello\x1b[0m \x1b[1mworld\x1b[0m"
	got := pty.StripANSI(input)
	want := "hello world"
	if got != want {
		t.Fatalf("StripANSI(%q) = %q, want %q", input, got, want)
	}
}

func TestStripANSIEmpty(t *testing.T) {
	got := pty.StripANSI("")
	if got != "" {
		t.Fatalf("StripANSI(\"\") = %q, want \"\"", got)
	}
}

func TestStripANSINoEscape(t *testing.T) {
	got := pty.StripANSI("plain text")
	if got != "plain text" {
		t.Fatalf("StripANSI(\"plain text\") = %q, want \"plain text\"", got)
	}
}

func TestDefaultFormatPrompt(t *testing.T) {
	got := pty.DefaultFormatPrompt("do the thing", "MARKER_X")
	if got == "" {
		t.Fatal("DefaultFormatPrompt returned empty string")
	}
	if got == "do the thing" {
		t.Fatal("DefaultFormatPrompt should append marker instructions")
	}
}
