package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ayan-de/agent-board/internal/pty"
)

type PtyRunner struct {
	runner      *pty.PtyRunner
	sessionName string
}

func NewPtyRunner(tmuxSession string) (*PtyRunner, error) {
	runner := pty.NewPtyRunner(tmuxSession)
	return &PtyRunner{
		runner:      runner,
		sessionName: tmuxSession,
	}, nil
}

func (r *PtyRunner) Start(ctx context.Context, req RunRequest) (RunHandle, error) {
	sessionName := r.sessionName
	if sessionName == "" {
		sessionName = "agentboard-agents"
	}

	windowName := fmt.Sprintf("agent-%s", shortID(req.SessionID, 8))

	cmd := exec.Command("tmux", "new-window", "-t", sessionName, "-d", "-P", "-F", "#{pane_id}", "-n", windowName)
	output, err := cmd.Output()
	if err != nil {
		return RunHandle{}, fmt.Errorf("create tmux window: %w", err)
	}
	paneID := strings.TrimSpace(string(output))

	cfg := r.runner.GetConfig(req.Agent)
	formattedPrompt := req.Prompt
	if cfg != nil && cfg.FormatPrompt != nil {
		formattedPrompt = cfg.FormatPrompt(req.Prompt)
	}

	homeDir, _ := os.UserHomeDir()
	cacheDir := fmt.Sprintf("%s/.agentboard/cache", homeDir)
	_ = os.MkdirAll(cacheDir, 0755)
	promptFile := fmt.Sprintf("%s/prompt-%s.txt", cacheDir, req.SessionID)
	if err := os.WriteFile(promptFile, []byte(formattedPrompt), 0644); err != nil {
		return RunHandle{}, fmt.Errorf("write prompt file: %w", err)
	}

	if err := exec.Command("tmux", "send-keys", "-t", paneID, cfg.Bin, "Enter").Run(); err != nil {
		return RunHandle{}, fmt.Errorf("send bin to pane: %w", err)
	}

	go r.injectPrompt(paneID, req.SessionID, cfg, promptFile)

	if req.Reporter != nil {
		req.Reporter(fmt.Sprintf("Agent %s started in tmux pane %s", req.Agent, paneID))
	}

	go r.monitorPane(req.SessionID, paneID, req.OnComplete)

	return RunHandle{
		Outcome: "running",
		Summary: fmt.Sprintf("Agent %s in pane %s", req.Agent, paneID),
	}, nil
}

func (r *PtyRunner) injectPrompt(paneID, sessionID string, cfg *pty.Config, promptFile string) {
	if err := waitForReady(paneID, cfg); err != nil {
		return
	}

	promptBytes, err := os.ReadFile(promptFile)
	if err != nil {
		return
	}
	prompt := string(promptBytes)

	for _, c := range prompt {
		var key string
		if c == '\n' {
			key = "C-m"
		} else {
			key = fmt.Sprintf("%c", c)
		}
		exec.Command("tmux", "send-keys", "-t", paneID, "-l", key).Run()
		time.Sleep(10 * time.Millisecond)
	}
	exec.Command("tmux", "send-keys", "-t", paneID, "Enter").Run()
}

func waitForReady(paneID string, cfg *pty.Config) error {
	timeout := cfg.ReadyWait
	if timeout < 3*time.Second {
		timeout = 3 * time.Second
	}
	deadline := time.Now().Add(timeout)

	for {
		output, err := exec.Command("tmux", "capture-pane", "-t", paneID, "-p", "-J", "-S", "-200").Output()
		if err == nil {
			stripped := pty.StripANSI(string(output))
			if cfg.ReadyPattern != nil && cfg.ReadyPattern.MatchString(stripped) {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("agent did not become ready")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (r *PtyRunner) monitorPane(sessionID string, paneID string, onComplete func(outcome, summary string)) {
	_ = sessionID
	_ = paneID
	_ = onComplete
}

func (r *PtyRunner) GetRunner() *pty.PtyRunner {
	return r.runner
}

func (r *PtyRunner) GetPaneID(sessionID string) (string, bool) {
	return "", false
}

func shortID(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}

func (r *PtyRunner) RunAgent(sessionID, agentName, prompt string, autoExit bool) error {
	return r.runner.Start(sessionID, agentName, prompt, autoExit)
}