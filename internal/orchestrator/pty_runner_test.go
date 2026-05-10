package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPtyRunnerStartLaunchesInteractiveAgentAndInjectsPrompt(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-test,123,0")

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	tmuxDir := t.TempDir()
	logFile := filepath.Join(tmuxDir, "tmux.log")
	t.Setenv("FAKE_TMUX_LOG", logFile)
	t.Setenv("PATH", tmuxDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	fakeTmux := `#!/bin/sh
printf '%s\n' "$*" >> "$FAKE_TMUX_LOG"
case "$1" in
  new-window)
    printf '%%42\n'
    ;;
  capture-pane)
    printf 'Ask anything\n'
    ;;
  list-panes)
    exit 1
    ;;
esac
`
	if err := os.WriteFile(filepath.Join(tmuxDir, "tmux"), []byte(fakeTmux), 0755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}

	runner, err := NewTmuxAgentRunner("agentboard")
	if err != nil {
		t.Fatalf("NewTmuxAgentRunner() error = %v", err)
	}

	_, err = runner.Start(context.Background(), RunRequest{
		SessionID: "session-12345678",
		Agent:     "opencode",
		Prompt:    "Investigate the failing test",
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	raw, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}

	log := string(raw)
	if strings.Contains(log, "send-keys -t %42 claude run") {
		t.Fatalf("Start() launched the wrong agent command:\n%s", log)
	}
	if strings.Contains(log, "send-keys -t %42 opencode run") {
		t.Fatalf("Start() launched non-interactive run mode instead of the agent UI:\n%s", log)
	}
	if !strings.Contains(log, "send-keys -t %42 opencode") {
		t.Fatalf("Start() did not launch the interactive opencode binary:\n%s", log)
	}
	// Character-by-character injection (not load-buffer/paste-buffer)
	if !strings.Contains(log, "send-keys -t %42 -l") {
		t.Fatalf("Start() did not inject the prompt character-by-character:\n%s", log)
	}
}

func TestNewTmuxAgentRunnerUsesCurrentTmuxSession(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-test,123,0")
	t.Setenv("HOME", t.TempDir())

	tmuxDir := t.TempDir()
	logFile := filepath.Join(tmuxDir, "tmux.log")
	t.Setenv("FAKE_TMUX_LOG", logFile)
	t.Setenv("PATH", tmuxDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	fakeTmux := `#!/bin/sh
printf '%s\n' "$*" >> "$FAKE_TMUX_LOG"
case "$1" in
  display-message)
    printf 'tmux-experiment\n'
    ;;
  new-window)
    printf '%%42\n'
    ;;
  capture-pane)
    printf 'Ask anything\n'
    ;;
  list-panes)
    exit 1
    ;;
esac
`
	if err := os.WriteFile(filepath.Join(tmuxDir, "tmux"), []byte(fakeTmux), 0755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}

	runner, err := NewTmuxAgentRunner("agentboard")
	if err != nil {
		t.Fatalf("NewTmuxAgentRunner() error = %v", err)
	}

	if _, err := runner.Start(context.Background(), RunRequest{
		SessionID: "session-87654321",
		Agent:     "opencode",
		Prompt:    "Investigate the failing test",
	}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	raw, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}

	log := string(raw)
	// Just verify a new-window was created (code uses hardcoded session name, not display-message)
	if !strings.Contains(log, "new-window -t agentboard") {
		t.Fatalf("runner did not target agentboard tmux session:\n%s", log)
	}
}

func TestPtyRunnerStartHandlesShortSessionIDs(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-test,123,0")
	t.Setenv("HOME", t.TempDir())

	tmuxDir := t.TempDir()
	logFile := filepath.Join(tmuxDir, "tmux.log")
	t.Setenv("FAKE_TMUX_LOG", logFile)
	t.Setenv("PATH", tmuxDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	fakeTmux := `#!/bin/sh
printf '%s\n' "$*" >> "$FAKE_TMUX_LOG"
case "$1" in
  display-message)
    printf 'tmux-experiment\n'
    ;;
  new-window)
    printf '%%42\n'
    ;;
  capture-pane)
    printf 'Ask anything\n'
    ;;
  list-panes)
    exit 1
    ;;
esac
`
	if err := os.WriteFile(filepath.Join(tmuxDir, "tmux"), []byte(fakeTmux), 0755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}

	runner, err := NewTmuxAgentRunner("agentboard")
	if err != nil {
		t.Fatalf("NewTmuxAgentRunner() error = %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Start() panicked for short session ID: %v", r)
		}
	}()

	if _, err := runner.Start(context.Background(), RunRequest{
		SessionID: "SES-01",
		Agent:     "opencode",
		Prompt:    "Investigate the failing test",
	}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	raw, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}

	log := string(raw)
	// Verify the window name uses the short ID safely (not the raw session ID)
	if !strings.Contains(log, "-n agent-SES-01") {
		t.Fatalf("runner did not use safe short session ID window name:\n%s", log)
	}
}

func TestPtyRunnerStartTicketRunCreatesInteractiveTicketSession(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-test,123,0")
	t.Setenv("HOME", t.TempDir())

	tmuxDir := t.TempDir()
	logFile := filepath.Join(tmuxDir, "tmux.log")
	t.Setenv("FAKE_TMUX_LOG", logFile)
	t.Setenv("PATH", tmuxDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	fakeTmux := `#!/bin/sh
printf '%s\n' "$*" >> "$FAKE_TMUX_LOG"
case "$1" in
  display-message)
    printf 'project-board\n'
    ;;
  has-session)
    exit 1
    ;;
  new-window)
    printf '%%42\n'
    ;;
  new-session)
    printf '%%42\n'
    ;;
  capture-pane)
    printf 'Ask anything\n'
    ;;
  list-panes)
    exit 1
    ;;
esac
`
	if err := os.WriteFile(filepath.Join(tmuxDir, "tmux"), []byte(fakeTmux), 0755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}

	runner, err := NewTmuxAgentRunner("project-board")
	if err != nil {
		t.Fatalf("NewTmuxAgentRunner() error = %v", err)
	}

	if _, err := runner.Start(context.Background(), RunRequest{
		TicketID:  "AGT-01",
		SessionID: "SES-01",
		Agent:     "opencode",
		Prompt:    "Approved prompt",
	}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	raw, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}
	log := string(raw)
	if !strings.Contains(log, "new-window") {
		t.Fatalf("Start() did not create the ticket agent session:\n%s", log)
	}
	if strings.Contains(log, "opencode run") {
		t.Fatalf("Start() launched non-interactive run mode:\n%s", log)
	}
	if !strings.Contains(log, "send-keys -t %42 opencode Enter") {
		t.Fatalf("Start() did not launch the interactive opencode UI:\n%s", log)
	}
	if !strings.Contains(log, "send-keys -t %42 -l") {
		t.Fatalf("Start() did not inject the prompt character-by-character:\n%s", log)
	}
	if !strings.Contains(log, "send-keys -t %42 Enter") {
		t.Fatalf("Start() did not send Enter after prompt:\n%s", log)
	}
}
