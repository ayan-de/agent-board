package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ayan-de/agent-board/internal/pty"
	tmuxutil "github.com/ayan-de/agent-board/internal/tmux"
)

type PtyRunner struct {
	runner       *pty.PtyRunner
	tmuxMode     bool
	sessionName  string
	activePanes  map[string]string // sessionID -> paneID
	configs      map[string]*pty.Config
}

func NewPtyRunner(tmuxSession string) (*PtyRunner, error) {
	if tmuxutil.IsInTmux() {
		if currentSession, err := tmuxutil.GetCurrentSessionName(); err == nil && currentSession != "" {
			tmuxSession = currentSession
		}
	}

	runner := pty.NewPtyRunner(tmuxSession)
	return &PtyRunner{
		runner:      runner,
		tmuxMode:    pty.IsInTmux(),
		sessionName: tmuxSession,
		activePanes: make(map[string]string),
		configs: map[string]*pty.Config{
			"opencode":    pty.NewOpenCode(),
			"claudecode":  pty.NewClaudeCode(),
			"claude-code": pty.NewClaudeCode(),
			"codex":       pty.NewCodex(),
			"gemini":      pty.NewGeminiCode(),
		},
	}, nil
}

func (r *PtyRunner) Start(ctx context.Context, req RunRequest) (RunHandle, error) {
	if !r.tmuxMode {
		return RunHandle{}, fmt.Errorf("pty runner requires tmux session")
	}

	// Create a tmux window for this agent in the correct session.
	windowName := fmt.Sprintf("agent-%s", shortID(req.SessionID, 8))
	cmd := exec.Command("tmux", "new-window", "-t", r.sessionName, "-d", "-P", "-F", "#{pane_id}", "-n", windowName)
	output, err := cmd.Output()
	if err != nil {
		return RunHandle{}, fmt.Errorf("create tmux window: %w", err)
	}
	paneID := strings.TrimSpace(string(output))
	r.activePanes[req.SessionID] = paneID

	cfg, err := r.agentConfig(req.Agent)
	if err != nil {
		return RunHandle{}, err
	}

	// Write prompt to file to avoid escaping issues
	homeDir, _ := os.UserHomeDir()
	cacheDir := fmt.Sprintf("%s/.agentboard/cache", homeDir)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return RunHandle{}, fmt.Errorf("create cache dir: %w", err)
	}
	promptFile := fmt.Sprintf("%s/prompt-%s.txt", cacheDir, req.SessionID)
	formattedPrompt := req.Prompt
	if cfg.FormatPrompt != nil {
		formattedPrompt = cfg.FormatPrompt(req.Prompt)
	}
	if err := os.WriteFile(promptFile, []byte(formattedPrompt), 0644); err != nil {
		return RunHandle{}, fmt.Errorf("write prompt file: %w", err)
	}

	agentCmd := r.buildLaunchCommand(cfg)

	sendCmd := exec.Command("tmux", "send-keys", "-t", paneID, agentCmd, "Enter")
	if err := sendCmd.Run(); err != nil {
		return RunHandle{}, fmt.Errorf("send keys to pane: %w", err)
	}

	go r.injectPrompt(ctx, paneID, req.SessionID, cfg, promptFile, req.Reporter)

	if req.Reporter != nil {
		req.Reporter(fmt.Sprintf("Agent %s started in tmux pane %s", req.Agent, paneID))
	}

	// Start monitoring goroutine to detect completion and call onComplete
	go r.monitorPane(req.SessionID, paneID, req.OnComplete)

	return RunHandle{
		Outcome: "running",
		Summary: fmt.Sprintf("Agent %s in pane %s", req.Agent, paneID),
	}, nil
}

func (r *PtyRunner) agentConfig(agent string) (*pty.Config, error) {
	cfg, ok := r.configs[agent]
	if !ok {
		return nil, fmt.Errorf("unsupported PTY agent %q", agent)
	}
	return cfg, nil
}

func (r *PtyRunner) buildLaunchCommand(cfg *pty.Config) string {
	parts := append([]string{cfg.Bin}, cfg.Args...)
	for i, part := range parts {
		parts[i] = shellQuote(part)
	}
	return strings.Join(parts, " ")
}

func (r *PtyRunner) injectPrompt(ctx context.Context, paneID, sessionID string, cfg *pty.Config, promptFile string, reporter func(string)) {
	if err := r.waitForReady(ctx, paneID, cfg); err != nil {
		if reporter != nil {
			reporter(fmt.Sprintf("Failed to inject prompt into %s: %v", sessionID, err))
		}
		return
	}

	bufferName := fmt.Sprintf("agentboard-%s", sessionID)
	if err := exec.Command("tmux", "load-buffer", "-b", bufferName, promptFile).Run(); err != nil {
		if reporter != nil {
			reporter(fmt.Sprintf("Failed to stage prompt buffer for %s: %v", sessionID, err))
		}
		return
	}
	defer exec.Command("tmux", "delete-buffer", "-b", bufferName).Run()

	if err := exec.Command("tmux", "paste-buffer", "-t", paneID, "-b", bufferName).Run(); err != nil {
		if reporter != nil {
			reporter(fmt.Sprintf("Failed to paste prompt into %s: %v", sessionID, err))
		}
		return
	}
	if err := exec.Command("tmux", "send-keys", "-t", paneID, "Enter").Run(); err != nil && reporter != nil {
		reporter(fmt.Sprintf("Failed to submit prompt for %s: %v", sessionID, err))
	}
}

func (r *PtyRunner) waitForReady(ctx context.Context, paneID string, cfg *pty.Config) error {
	if cfg.ReadyPattern == nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1200 * time.Millisecond):
			return nil
		}
	}

	timeout := cfg.ReadyWait
	if timeout < 3*time.Second {
		timeout = 3 * time.Second
	}
	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		output, err := exec.Command("tmux", "capture-pane", "-t", paneID, "-p", "-J", "-S", "-200").Output()
		if err == nil && cfg.ReadyPattern.Match(output) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("agent UI did not become ready")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (r *PtyRunner) monitorPane(sessionID string, paneID string, onComplete func(outcome, summary string)) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			checkCmd := exec.Command("tmux", "list-panes", "-t", paneID, "-F", "#{pane_pid}")
			output, err := checkCmd.Output()

			if err != nil || len(output) == 0 {
				outcome := "completed"
				summary := fmt.Sprintf("Agent finished for session %s", sessionID)

				if onComplete != nil {
					onComplete(outcome, summary)
				}
				return
			}
		}
	}
}

func (r *PtyRunner) GetRunner() *pty.PtyRunner {
	return r.runner
}

func (r *PtyRunner) GetPaneID(sessionID string) (string, bool) {
	paneID, ok := r.activePanes[sessionID]
	return paneID, ok
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"\\$`()[]{}*?&;|<>!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func shortID(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}
