package orchestrator

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractResumeCommand(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "opencode resume",
			output: "Session   New session - 2026-05-10T07:36:46.777Z\nContinue  opencode -s ses_1ef2eca46ffeKTXRokicTzd5iI",
			want:   "opencode -s ses_1ef2eca46ffeKTXRokicTzd5iI",
		},
		{
			name:   "claude resume",
			output: "Resume this session with:\nclaude --resume 31a136eb-7bf4-496d-b00b-73c3ac8158de",
			want:   "claude --resume 31a136eb-7bf4-496d-b00b-73c3ac8158de",
		},
		{
			name:   "no resume command",
			output: "Agent completed successfully",
			want:   "",
		},
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractResumeCommand(tt.output)
			if got != tt.want {
				t.Errorf("ExtractResumeCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCaptureFinalPaneOutputKeepsPollingAfterStaleOutput(t *testing.T) {
	captures := []string{
		"working output without resume",
		"working output without resume",
		"Session   Update README with Agent Support section\nContinue  opencode -s ses_1ed7193d1ffehIFqWvG7dgNWVx",
	}
	idx := 0
	got := captureFinalPaneOutput("previous non-empty output", func() (string, error) {
		if idx >= len(captures) {
			return "", errors.New("no more captures")
		}
		out := captures[idx]
		idx++
		return out, nil
	}, nil)

	if cmd := ExtractResumeCommand(got); cmd != "opencode -s ses_1ed7193d1ffehIFqWvG7dgNWVx" {
		t.Fatalf("resume command = %q, want opencode command; captured output:\n%s", cmd, got)
	}
	if idx != len(captures) {
		t.Fatalf("capture calls = %d, want %d", idx, len(captures))
	}
}

func TestCaptureTmuxPaneOutputIncludesAlternateScreen(t *testing.T) {
	dir := t.TempDir()
	tmuxPath := filepath.Join(dir, "tmux")
	script := `#!/bin/sh
case " $* " in
  *" -a "*) printf 'Continue  opencode -s ses_alt_screen\n' ;;
  *) printf 'regular shell output\n' ;;
esac
`
	if err := os.WriteFile(tmuxPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}

	got, err := captureTmuxPaneOutput(tmuxPath, "%42", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "regular shell output") {
		t.Fatalf("missing primary capture: %q", got)
	}
	if !strings.Contains(got, "opencode -s ses_alt_screen") {
		t.Fatalf("missing alternate screen capture: %q", got)
	}
}
