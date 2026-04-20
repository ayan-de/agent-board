package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ayan-de/agent-board/internal/pty"
)

type PaneManager struct {
	mu       sync.RWMutex
	panes    map[string]*AgentPane
	tmux     string
	registry map[string]*pty.AgentConfig
	chdir    string
}

type AgentPane struct {
	SessionID   string
	TicketID    string
	Agent       string
	TmuxSession string
	PaneID      string
	WindowID    string
	StartedAt   time.Time
	Status      string
	Outcome     string
	Summary     string
	cancelFunc  context.CancelFunc
	onComplete  func(outcome, summary string)
}

func NewPaneManager(registry map[string]*pty.AgentConfig, chdir string) (*PaneManager, error) {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return nil, fmt.Errorf("tmux not found: %w", err)
	}
	return &PaneManager{
		panes:    make(map[string]*AgentPane),
		tmux:     tmuxPath,
		registry: registry,
		chdir:    chdir,
	}, nil
}

func (pm *PaneManager) CreatePane(ctx context.Context, req RunRequest) (*AgentPane, error) {
	cfg, ok := pm.registry[req.Agent]
	if !ok {
		return nil, fmt.Errorf("paneManager.createPane: no config for agent %q", req.Agent)
	}

	tmuxSession := "agentboard-" + strings.ToLower(strings.ReplaceAll(req.SessionID, "_", "-"))
	paneCtx, cancel := context.WithCancel(ctx)
	pane := &AgentPane{
		SessionID:   req.SessionID,
		TicketID:    req.TicketID,
		Agent:       req.Agent,
		TmuxSession: tmuxSession,
		StartedAt:   time.Now(),
		Status:      "running",
		cancelFunc:  cancel,
		onComplete:  req.OnComplete,
	}

	cmdArgs := []string{
		"new-session", "-d", "-P",
		"-F", "#{session_name}:#{window_id}:#{pane_id}",
		"-s", tmuxSession,
		"-x", "120",
		"-y", "40",
	}
	if pm.chdir != "" {
		cmdArgs = append(cmdArgs, "-c", pm.chdir)
	}
	cmdArgs = append(cmdArgs, pm.launchCommand(cfg))

	output, err := exec.Command(pm.tmux, cmdArgs...).Output()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("paneManager.createPane: creating tmux session: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(output)), ":")
	if len(parts) != 3 {
		cancel()
		_ = pm.killSession(tmuxSession)
		return nil, fmt.Errorf("paneManager.createPane: unexpected tmux output %q", strings.TrimSpace(string(output)))
	}
	pane.TmuxSession = parts[0]
	pane.WindowID = parts[1]
	pane.PaneID = parts[2]

	pm.mu.Lock()
	pm.panes[req.SessionID] = pane
	pm.mu.Unlock()

	go pm.monitorPane(paneCtx, pane, cfg, req.Prompt, req.Reporter)

	return pane, nil
}

func (pm *PaneManager) launchCommand(cfg *pty.AgentConfig) string {
	parts := []string{shellQuote(cfg.Bin)}
	for _, arg := range cfg.Args {
		parts = append(parts, shellQuote(arg))
	}
	return "exec " + strings.Join(parts, " ")
}

func (pm *PaneManager) monitorPane(ctx context.Context, pane *AgentPane, cfg *pty.AgentConfig, prompt string, reporter func(string)) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	readyRe := regexp.MustCompile(cfg.ReadyPattern)
	doneRe := regexp.MustCompile(regexp.QuoteMeta(pty.DoneMarker))
	idleRes := make([]*regexp.Regexp, len(cfg.IdlePatterns))
	for i, pattern := range cfg.IdlePatterns {
		idleRes[i] = regexp.MustCompile(pattern)
	}

	var (
		fired             bool
		promptInjected    bool
		promptSendAt      time.Time
		completionCheckAt time.Time
		lastCapture       string
	)
	firePrompt := func() {
		if fired {
			return
		}
		fired = true
		promptSendAt = time.Now().Add(cfg.ReadyWait)
	}
	fallback := time.AfterFunc(cfg.FallbackTimeout, firePrompt)
	defer fallback.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			if fired && !promptInjected && !promptSendAt.IsZero() && !now.Before(promptSendAt) {
				if err := pm.injectPrompt(pane.PaneID, cfg, prompt); err == nil {
					promptInjected = true
					completionCheckAt = now.Add(cfg.GracePeriod)
				}
			}

			captured, captureErr := pm.capturePaneByPaneID(pane.PaneID, 200)
			if captureErr == nil && captured != "" {
				lastCapture = captured
				stripped := pty.StripANSI(captured)

				if !fired && readyRe.MatchString(stripped) {
					fallback.Stop()
					firePrompt()
				}

				if promptInjected && !completionCheckAt.IsZero() && !now.Before(completionCheckAt) {
					lines := strings.Split(stripped, "\n")
					if pty.DetectCompletionFromBuffer(lines, doneRe, idleRes) {
						pm.finishPane(pane, "completed", "Agent completed task", reporter)
						return
					}
				}
			}

			if !pm.paneExists(pane.PaneID) {
				outcome := "completed"
				summary := "Agent process exited"
				if lastCapture != "" {
					stripped := pty.StripANSI(lastCapture)
					if strings.Contains(stripped, pty.DoneMarker) {
						summary = "Agent completed task"
					}
				}
				pm.finishPane(pane, outcome, summary, reporter)
				return
			}
		}
	}
}

func (pm *PaneManager) finishPane(pane *AgentPane, outcome, summary string, reporter func(string)) {
	pm.mu.Lock()
	current, ok := pm.panes[pane.SessionID]
	if !ok {
		pm.mu.Unlock()
		return
	}
	current.Status = outcome
	current.Outcome = outcome
	current.Summary = summary
	pm.mu.Unlock()

	if reporter != nil {
		reporter(summary)
	}
	if current.onComplete != nil {
		current.onComplete(outcome, summary)
	}
	_ = pm.killSession(current.TmuxSession)
	_ = pm.RemovePane(current.SessionID, false)
}

func (pm *PaneManager) paneExists(paneID string) bool {
	cmd := exec.Command(pm.tmux, "list-panes", "-t", paneID, "-F", "#{pane_id}")
	output, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(output)) != ""
}

func (pm *PaneManager) injectPrompt(paneID string, cfg *pty.AgentConfig, prompt string) error {
	formatted := prompt
	if cfg.FormatPrompt != nil {
		formatted = cfg.FormatPrompt(prompt)
	}

	switch cfg.Name {
	case "opencode", "codex":
		return pm.sendPromptTyped(paneID, formatted)
	case "claude-code":
		formatted = strings.ReplaceAll(formatted, "\n", " ")
		formatted = strings.ReplaceAll(formatted, "\r", " ")
		return pm.sendLiteralPrompt(paneID, formatted, false)
	default:
		tmpFile, err := pm.writePromptFile(formatted)
		if err != nil {
			return err
		}
		defer os.Remove(tmpFile)

		bufferName := "agentboard-" + uuid.NewString()
		if err := exec.Command(pm.tmux, "send-keys", "-t", paneID, "C-u").Run(); err != nil {
			return err
		}
		if err := exec.Command(pm.tmux, "load-buffer", "-b", bufferName, tmpFile).Run(); err != nil {
			return err
		}
		if err := exec.Command(pm.tmux, "paste-buffer", "-t", paneID, "-b", bufferName, "-d").Run(); err != nil {
			_ = exec.Command(pm.tmux, "delete-buffer", "-b", bufferName).Run()
			return err
		}
		return exec.Command(pm.tmux, "send-keys", "-t", paneID, "Enter").Run()
	}
}

func (pm *PaneManager) sendPromptTyped(paneID, prompt string) error {
	prompt = strings.ReplaceAll(prompt, "\r", "")
	if err := exec.Command(pm.tmux, "send-keys", "-t", paneID, "C-u", "C-w").Run(); err != nil {
		return err
	}

	lines := strings.Split(prompt, "\n")
	for i, line := range lines {
		if line != "" {
			if err := exec.Command(pm.tmux, "send-keys", "-t", paneID, "-l", line).Run(); err != nil {
				return err
			}
		}
		if i < len(lines)-1 {
			if err := exec.Command(pm.tmux, "send-keys", "-t", paneID, "C-j").Run(); err != nil {
				return err
			}
		}
	}

	time.Sleep(100 * time.Millisecond)
	if err := exec.Command(pm.tmux, "send-keys", "-t", paneID, "C-m").Run(); err != nil {
		return err
	}
	time.Sleep(150 * time.Millisecond)
	return exec.Command(pm.tmux, "send-keys", "-t", paneID, "C-m").Run()
}

func (pm *PaneManager) sendLiteralPrompt(paneID, prompt string, clearWord bool) error {
	args := []string{"send-keys", "-t", paneID, "C-u"}
	if clearWord {
		args = append(args, "C-w")
	}
	if err := exec.Command(pm.tmux, args...).Run(); err != nil {
		return err
	}
	if err := exec.Command(pm.tmux, "send-keys", "-t", paneID, "-l", prompt).Run(); err != nil {
		return err
	}
	return exec.Command(pm.tmux, "send-keys", "-t", paneID, "Enter").Run()
}

func (pm *PaneManager) writePromptFile(prompt string) (string, error) {
	dir := filepath.Join(os.TempDir(), "agentboard")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("paneManager.writePromptFile: mkdir: %w", err)
	}
	path := filepath.Join(dir, "prompt-"+uuid.NewString()+".txt")
	if err := os.WriteFile(path, []byte(prompt), 0o600); err != nil {
		return "", fmt.Errorf("paneManager.writePromptFile: write: %w", err)
	}
	return path, nil
}

func (pm *PaneManager) GetPane(sessionID string) (*AgentPane, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	pane, ok := pm.panes[sessionID]
	return pane, ok
}

func (pm *PaneManager) ListPanes() []*AgentPane {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	panes := make([]*AgentPane, 0, len(pm.panes))
	for _, pane := range pm.panes {
		panes = append(panes, pane)
	}
	return panes
}

func (pm *PaneManager) SendInput(sessionID, input string) error {
	pm.mu.RLock()
	pane, ok := pm.panes[sessionID]
	pm.mu.RUnlock()
	if !ok {
		return fmt.Errorf("paneManager.sendInput: session %s not found", sessionID)
	}
	if err := exec.Command(pm.tmux, "send-keys", "-t", pane.PaneID, "-l", input).Run(); err != nil {
		return err
	}
	return exec.Command(pm.tmux, "send-keys", "-t", pane.PaneID, "Enter").Run()
}

func (pm *PaneManager) RemovePane(sessionID string, kill bool) error {
	pm.mu.Lock()
	pane, ok := pm.panes[sessionID]
	if ok {
		delete(pm.panes, sessionID)
	}
	pm.mu.Unlock()
	if !ok {
		return nil
	}

	if pane.cancelFunc != nil {
		pane.cancelFunc()
	}
	if kill {
		return pm.killSession(pane.TmuxSession)
	}
	return nil
}

func (pm *PaneManager) Resize(sessionID string, rows, cols int) error {
	pm.mu.RLock()
	pane, ok := pm.panes[sessionID]
	pm.mu.RUnlock()
	if !ok {
		return fmt.Errorf("paneManager.resize: session %s not found", sessionID)
	}
	cmd := exec.Command(pm.tmux, "resize-window", "-t", pane.TmuxSession, "-x", fmt.Sprintf("%d", cols), "-y", fmt.Sprintf("%d", rows))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("paneManager.resize: %w", err)
	}
	return nil
}

func (pm *PaneManager) CapturePane(sessionID string, lines int) (string, error) {
	pm.mu.RLock()
	pane, ok := pm.panes[sessionID]
	pm.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("paneManager.capturePane: session %s not found", sessionID)
	}
	return pm.capturePaneByPaneID(pane.PaneID, lines)
}

func (pm *PaneManager) capturePaneByPaneID(paneID string, lines int) (string, error) {
	cmd := exec.Command(pm.tmux, "capture-pane", "-t", paneID, "-p", "-e", "-J", "-S", fmt.Sprintf("-%d", lines))
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("paneManager.capturePaneByPaneID: %w", err)
	}
	return string(output), nil
}

func (pm *PaneManager) killSession(sessionName string) error {
	return exec.Command(pm.tmux, "kill-session", "-t", sessionName).Run()
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
