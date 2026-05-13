package orchestrator

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ayan-de/agent-board/internal/core"
)

type opencodeEvent struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	SessionID string `json:"sessionID"`
	Part      struct {
		Type   string `json:"type"`
		Text   string `json:"text"`
		Reason string `json:"reason"`
	} `json:"part"`
}

func parseOpencodeOutput(r io.Reader) (RunHandle, error) {
	var texts []string
	var lastReason string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var evt opencodeEvent
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}
		switch evt.Type {
		case "text":
			if evt.Part.Text != "" {
				texts = append(texts, evt.Part.Text)
			}
		case "step_finish":
			if evt.Part.Reason != "" {
				lastReason = evt.Part.Reason
			}
		}
	}

	summary := strings.Join(texts, "\n")
	if summary == "" {
		summary = "Agent finished its task."
	}

	outcome := "completed"
	if lastReason == "error" {
		outcome = "failed"
	}

	return RunHandle{Outcome: outcome, Summary: summary}, nil
}

// PaneManager manages tmux panes for running agents
type PaneManager struct {
	mu          sync.RWMutex
	panes       map[string]*paneState
	tmux        string
	sessionName string
}

// paneState represents a running agent's tmux pane (internal, includes private fields)
type paneState struct {
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
	onComplete func(outcome, summary, resumeCommand string)
}

func (p *paneState) ToCore() *core.AgentPane {
	return &core.AgentPane{
		SessionID: p.SessionID,
		TicketID:  p.TicketID,
		Agent:     p.Agent,
		PaneID:    p.PaneID,
		WindowID:  p.WindowID,
		Status:    p.Status,
		Outcome:   p.Outcome,
		Summary:   p.Summary,
	}
}

// NewPaneManager creates a new pane manager
func NewPaneManager(sessionName string) (*PaneManager, error) {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return nil, fmt.Errorf("tmux not found: %w", err)
	}
	if sessionName == "" {
		sessionName = "agentboard"
	}
	return &PaneManager{
		panes:       make(map[string]*paneState),
		tmux:        tmuxPath,
		sessionName: sessionName,
	}, nil
}

// CreatePane creates a new tmux pane for an agent and starts it
func (pm *PaneManager) CreatePane(ctx context.Context, req RunRequest) (*core.AgentPane, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if we're in tmux
	if os.Getenv("TMUX") == "" {
		return nil, fmt.Errorf("not in tmux session - cannot create agent pane")
	}

	// Create a cancelable context for this pane
	paneCtx, cancel := context.WithCancel(ctx)

	pane := &paneState{
		SessionID:  req.SessionID,
		TicketID:   req.TicketID,
		Agent:      req.Agent,
		StartedAt:  time.Now(),
		Status:     "running",
		cancelFunc: cancel,
		onComplete: req.OnComplete,
	}

	// Get the main agentboard session name
	mainSession := pm.sessionName

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

	_ = exec.Command(pm.tmux, "set-option", "-p", "-t", pane.PaneID, "remain-on-exit", "on").Run()

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

	return pane.ToCore(), nil
}

// monitorPane watches a tmux pane for completion
func (pm *PaneManager) monitorPane(ctx context.Context, pane *paneState, reporter func(string)) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastCaptured string
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Capture output continuously so we always have the latest content
			if out, err := pm.capturePaneOutput(pane.PaneID, 5000); err == nil && out != "" {
				lastCaptured = out
			}

			checkCmd := exec.Command(pm.tmux, "list-panes", "-t", pane.PaneID, "-F", "#{pane_pid}:#{pane_dead}")
			output, err := checkCmd.Output()

			outStr := strings.TrimSpace(string(output))
			isDead := strings.HasSuffix(outStr, ":1")

			if err != nil || len(outStr) == 0 || isDead {
				outcome := "completed"
				summary := fmt.Sprintf("Agent %s finished for ticket %s", pane.Agent, pane.TicketID)

				captured := captureFinalPaneOutput(lastCaptured, func() (string, error) {
					return pm.capturePaneOutput(pane.PaneID, 5000)
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

				pm.mu.Lock()
				pane.Status = outcome
				pane.Outcome = outcome
				pane.Summary = summary
				if pane.promptFile != "" {
					_ = os.Remove(pane.promptFile)
					pane.promptFile = ""
				}
				pm.mu.Unlock()

				if isDead {
					_ = exec.Command(pm.tmux, "kill-pane", "-t", pane.PaneID).Run()
				}

				if pane.onComplete != nil {
					resumeCmd := ExtractResumeCommand(captured)
					pane.onComplete(outcome, summary, resumeCmd)
				}

				if reporter != nil {
					reporter(summary)
				}
				return
			}
		}
	}
}

func (pm *PaneManager) capturePaneOutput(paneID string, lines int) (string, error) {
	output, err := captureTmuxPaneOutput(pm.tmux, paneID, lines)
	if err != nil {
		return "", fmt.Errorf("failed to capture pane output: %w", err)
	}
	return output, nil
}

// GetPane retrieves a pane by session ID
func (pm *PaneManager) GetPane(sessionID string) (*core.AgentPane, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	pane, ok := pm.panes[sessionID]
	if !ok {
		return nil, false
	}
	return pane.ToCore(), true
}

// ListPanes returns all active panes
func (pm *PaneManager) ListPanes() []*core.AgentPane {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	panes := make([]*core.AgentPane, 0, len(pm.panes))
	for _, p := range pm.panes {
		panes = append(panes, p.ToCore())
	}
	return panes
}

// ListPanesByAgent returns all panes for a specific agent binary
func (pm *PaneManager) ListPanesByAgent(agent string) []*core.AgentPane {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	panes := make([]*core.AgentPane, 0)
	for _, p := range pm.panes {
		if p.Agent == agent {
			panes = append(panes, p.ToCore())
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
