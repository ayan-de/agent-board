package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// PaneManager manages tmux panes for running agents
type PaneManager struct {
	mu    sync.RWMutex
	panes map[string]*AgentPane // sessionID -> AgentPane
	tmux  string
}

// AgentPane represents a running agent's tmux pane
type AgentPane struct {
	SessionID  string
	TicketID   string
	Agent      string
	PaneID     string
	WindowID   string
	StartedAt  time.Time
	Status     string
	Outcome    string
	Summary    string
	cancelFunc context.CancelFunc
	promptFile string
	onComplete func(outcome, summary string)
}

// NewPaneManager creates a new pane manager
func NewPaneManager() (*PaneManager, error) {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return nil, fmt.Errorf("tmux not found: %w", err)
	}
	return &PaneManager{
		panes: make(map[string]*AgentPane),
		tmux:  tmuxPath,
	}, nil
}

// CreatePane creates a new tmux pane for an agent and starts it
func (pm *PaneManager) CreatePane(ctx context.Context, req RunRequest) (*AgentPane, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if we're in tmux
	if os.Getenv("TMUX") == "" {
		return nil, fmt.Errorf("not in tmux session - cannot create agent pane")
	}

	// Create a cancelable context for this pane
	paneCtx, cancel := context.WithCancel(ctx)

	pane := &AgentPane{
		SessionID:  req.SessionID,
		TicketID:   req.TicketID,
		Agent:      req.Agent,
		StartedAt:  time.Now(),
		Status:     "running",
		cancelFunc: cancel,
		onComplete: req.OnComplete,
	}

	// Get the main agentboard session name
	mainSession := "agentboard"

	// Find or create a window for agent panes
	windowName := fmt.Sprintf("agents-%s", req.TicketID)

	// First, check if the session exists
	checkCmd := exec.Command(pm.tmux, "has-session", "-t", mainSession)
	if checkCmd.Run() != nil {
		// Session doesn't exist, create it first
		createSessionCmd := exec.Command(pm.tmux, "new-session", "-d", "-s", mainSession)
		if err := createSessionCmd.Run(); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create tmux session %s: %w", mainSession, err)
		}
	}

	// Create a new window and capture both window_id and pane_id in one command
	// Format: "window_id:pane_id" e.g. "@1:%1"
	windowCmd := exec.Command(pm.tmux, "new-window", "-t", mainSession,
		"-n", windowName, "-d", "-P", "-F", "#{window_id}:#{pane_id}")
	output, err := windowCmd.Output()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create tmux window: %w", err)
	}

	// Parse the output "window_id:pane_id"
	parts := strings.Split(strings.TrimSpace(string(output)), ":")
	if len(parts) != 2 {
		cancel()
		return nil, fmt.Errorf("unexpected tmux output format: %s", output)
	}

	pane.WindowID = parts[0]
	pane.PaneID = parts[1]

	// Write the prompt to a persistent file (not tmp, to avoid cleanup issues)
	// Store in user's home directory under .agentboard/cache
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp"
	}
	cacheDir := fmt.Sprintf("%s/.agentboard/cache", homeDir)
	_ = os.MkdirAll(cacheDir, 0755)
	promptFile := fmt.Sprintf("%s/prompt-%s.txt", cacheDir, req.SessionID)
	if err := os.WriteFile(promptFile, []byte(req.Prompt), 0644); err != nil {
		pm.killWindow(pane.WindowID)
		cancel()
		return nil, fmt.Errorf("failed to write prompt file: %w", err)
	}
	// Store the prompt file path for cleanup later
	pane.promptFile = promptFile

	// Build the agent command using the persistent file
	// This avoids escaping issues with special characters in the prompt
	agentCmd := fmt.Sprintf("%s run \"$(cat %s)\"", req.Agent, promptFile)

	// Send the command to the pane
	sendCmd := exec.Command(pm.tmux, "send-keys", "-t", pane.PaneID, agentCmd, "Enter")
	if err := sendCmd.Run(); err != nil {
		pm.killWindow(pane.WindowID)
		cancel()
		return nil, fmt.Errorf("failed to send command to pane: %w", err)
	}

	// Start a goroutine to monitor the pane
	go pm.monitorPane(paneCtx, pane, req.Reporter)

	// Store the pane
	pm.panes[req.SessionID] = pane

	return pane, nil
}

// monitorPane watches a tmux pane for completion
func (pm *PaneManager) monitorPane(ctx context.Context, pane *AgentPane, reporter func(string)) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			checkCmd := exec.Command(pm.tmux, "list-panes", "-t", pane.PaneID, "-F", "#{pane_pid}")
			output, err := checkCmd.Output()

			if err != nil || len(output) == 0 {
				outcome := "completed"
				summary := fmt.Sprintf("Agent %s finished for ticket %s", pane.Agent, pane.TicketID)

				captured, capErr := pm.capturePaneOutput(pane.PaneID, 200)
				if capErr == nil && captured != "" {
					parsed, parseErr := ParseOpencodeOutput(strings.NewReader(captured))
					if parseErr == nil {
						if parsed.Outcome != "" {
							outcome = parsed.Outcome
						}
						if parsed.Summary != "" {
							summary = parsed.Summary
						}
					}
				}

				pm.mu.Lock()
				pane.Status = outcome
				pane.Outcome = outcome
				pane.Summary = summary
				if pane.promptFile != "" {
					_ = os.Remove(pane.promptFile)
					pane.promptFile = ""
				}
				pm.mu.Unlock()

				if reporter != nil {
					reporter(summary)
				}
				if pane.onComplete != nil {
					pane.onComplete(outcome, summary)
				}
				return
			}
		}
	}
}

func (pm *PaneManager) capturePaneOutput(paneID string, lines int) (string, error) {
	captureCmd := exec.Command(pm.tmux, "capture-pane", "-t", paneID, "-p", "-e", "-J", "-C", "-P",
		"-S", fmt.Sprintf("-%d", lines))
	output, err := captureCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture pane output: %w", err)
	}
	return string(output), nil
}

// GetPane retrieves a pane by session ID
func (pm *PaneManager) GetPane(sessionID string) (*AgentPane, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	pane, ok := pm.panes[sessionID]
	return pane, ok
}

// ListPanes returns all active panes
func (pm *PaneManager) ListPanes() []*AgentPane {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	panes := make([]*AgentPane, 0, len(pm.panes))
	for _, p := range pm.panes {
		panes = append(panes, p)
	}
	return panes
}

// ListPanesByAgent returns all panes for a specific agent binary
func (pm *PaneManager) ListPanesByAgent(agent string) []*AgentPane {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	panes := make([]*AgentPane, 0)
	for _, p := range pm.panes {
		if p.Agent == agent {
			panes = append(panes, p)
		}
	}
	return panes
}

// SendInput sends input to a pane's stdin
func (pm *PaneManager) SendInput(sessionID, input string) error {
	pm.mu.RLock()
	pane, ok := pm.panes[sessionID]
	pm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("pane not found for session %s", sessionID)
	}

	cmd := exec.Command(pm.tmux, "send-keys", "-t", pane.PaneID, input, "Enter")
	return cmd.Run()
}

// RemovePane removes a pane from tracking and optionally kills it
func (pm *PaneManager) RemovePane(sessionID string, kill bool) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pane, ok := pm.panes[sessionID]
	if !ok {
		return nil
	}

	if kill {
		pm.killWindow(pane.WindowID)
		pane.cancelFunc()
	}

	// Clean up the prompt file
	if pane.promptFile != "" {
		_ = os.Remove(pane.promptFile)
		pane.promptFile = ""
	}

	delete(pm.panes, sessionID)
	return nil
}

// killWindow kills a tmux window
func (pm *PaneManager) killWindow(windowID string) {
	_ = exec.Command(pm.tmux, "kill-window", "-t", windowID).Run()
}

// SwitchToPane switches the tmux view to show a specific pane
func (pm *PaneManager) SwitchToPane(sessionID string) error {
	pm.mu.RLock()
	pane, ok := pm.panes[sessionID]
	pm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("pane not found for session %s", sessionID)
	}

	// Switch to the window containing the pane
	cmd := exec.Command(pm.tmux, "select-window", "-t", pane.WindowID)
	return cmd.Run()
}

// CapturePane captures the current content of a pane
func (pm *PaneManager) CapturePane(sessionID string, lines int) (string, error) {
	pm.mu.RLock()
	pane, ok := pm.panes[sessionID]
	pm.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("pane not found for session %s", sessionID)
	}

	// Capture pane content
	captureCmd := exec.Command(pm.tmux, "capture-pane", "-t", pane.PaneID, "-p", "-e", "-J", "-C", "-P",
		"-S", fmt.Sprintf("-%d", lines))
	output, err := captureCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture pane: %w", err)
	}

	return string(output), nil
}
