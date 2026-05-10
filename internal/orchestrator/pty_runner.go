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

type tmuxAgentRunner struct {
	sessionName string
}

func NewTmuxAgentRunner(tmuxSession string) (*tmuxAgentRunner, error) {
	return &tmuxAgentRunner{
		sessionName: tmuxSession,
	}, nil
}

func (r *tmuxAgentRunner) Start(ctx context.Context, req RunRequest) (RunHandle, error) {
	sessionName := r.sessionName
	if sessionName == "" {
		sessionName = "agentboard-agents"
	}

	windowName := fmt.Sprintf("agent-%s", shortID(req.SessionID, 8))

	cmd := exec.Command("tmux", "new-window", "-t", sessionName, "-d", "-a", "-P", "-F", "#{window_id}:#{pane_id}", "-n", windowName)
	output, err := cmd.Output()
	if err != nil {
		return RunHandle{}, fmt.Errorf("create tmux window: %w", err)
	}
	parts := strings.Split(strings.TrimSpace(string(output)), ":")
	paneID := parts[len(parts)-1]

	_ = exec.Command("tmux", "set-option", "-p", "-t", paneID, "remain-on-exit", "on").Run()

	cfg := pty.GetConfig(req.Agent)
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

func (r *tmuxAgentRunner) injectPrompt(paneID, sessionID string, cfg *pty.Config, promptFile string) {
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

func (r *tmuxAgentRunner) monitorPane(sessionID string, paneID string, onComplete func(outcome, summary, resumeCommand string)) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastCaptured string
	for {
		select {
		case <-context.Background().Done():
			return
		case <-ticker.C:
			// Capture output continuously so we always have the latest content
			if out, err := r.capturePaneOutput(paneID, 5000); err == nil && out != "" {
				lastCaptured = out
			}

			checkCmd := exec.Command("tmux", "list-panes", "-t", paneID, "-F", "#{pane_pid}:#{pane_dead}")
			output, err := checkCmd.Output()

			outStr := strings.TrimSpace(string(output))
			isDead := strings.HasSuffix(outStr, ":1")

			if err != nil || len(outStr) == 0 || isDead {
				outcome := "completed"
				summary := fmt.Sprintf("Agent session %s finished", sessionID)

				captured := captureFinalPaneOutput(lastCaptured, func() (string, error) {
					return r.capturePaneOutput(paneID, 5000)
				}, time.Sleep)

				if captured != "" {
					parsed, parseErr := parseOpencodeOutput(strings.NewReader(captured))
					if parseErr == nil {
						if parsed.Outcome != "" {
							outcome = parsed.Outcome
						}
						if parsed.Summary != "" {
							summary = parsed.Summary
						}
					}
				}

				if isDead {
					_ = exec.Command("tmux", "kill-pane", "-t", paneID).Run()
				}

				if onComplete != nil {
					resumeCmd := ExtractResumeCommand(captured)
					onComplete(outcome, summary, resumeCmd)
				}
				return
			}
		}
	}
}

func (r *tmuxAgentRunner) capturePaneOutput(paneID string, lines int) (string, error) {
	output, err := captureTmuxPaneOutput("tmux", paneID, lines)
	if err != nil {
		return "", fmt.Errorf("failed to capture pane output: %w", err)
	}
	return output, nil
}

func (r *tmuxAgentRunner) GetPaneID(sessionID string) (string, bool) {
	return "", false
}

func shortID(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}
