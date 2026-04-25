package orchestrator

import (
	"context"
	"fmt"
)

// TmuxRunner manages agents in tmux panes using a PaneManager
type TmuxRunner struct {
	paneManager *PaneManager
}

// NewTmuxRunner creates a new TmuxRunner with a pane manager
func NewTmuxRunner() (*TmuxRunner, error) {
	pm, err := NewPaneManager()
	if err != nil {
		return nil, err
	}
	return &TmuxRunner{
		paneManager: pm,
	}, nil
}

// Start creates a new tmux pane for the agent and starts it non-blocking
// The agent runs in the background and can be monitored via the pane manager
func (r *TmuxRunner) Start(ctx context.Context, req RunRequest) (RunHandle, error) {
	// Create a new pane for this agent session
	pane, err := r.paneManager.CreatePane(ctx, req)
	if err != nil {
		return RunHandle{}, fmt.Errorf("tmuxRunner.start: %w", err)
	}

	if req.Reporter != nil {
		req.Reporter(fmt.Sprintf("Agent %s started in tmux window %s. Press 'i' in dashboard to view.",
			req.Agent, pane.WindowID))
	}

	// Return immediately - agent is running in background
	// The pane manager will monitor and update status when it completes
	return RunHandle{
		Outcome: "running",
		Summary: fmt.Sprintf("Agent started in pane %s", pane.WindowID),
	}, nil
}

// GetPaneManager returns the underlying pane manager
func (r *TmuxRunner) GetPaneManager() *PaneManager {
	return r.paneManager
}

// SendInput sends input to a specific agent session
func (r *TmuxRunner) SendInput(sessionID, input string) error {
	return r.paneManager.SendInput(sessionID, input)
}

// CapturePane captures the current content of a pane
func (r *TmuxRunner) CapturePane(sessionID string, lines int) (string, error) {
	return r.paneManager.CapturePane(sessionID, lines)
}

// ListPanes returns all active agent panes
func (r *TmuxRunner) ListPanes() []*AgentPane {
	return r.paneManager.ListPanes()
}

// StopPane stops a specific agent pane
func (r *TmuxRunner) StopPane(sessionID string) error {
	return r.paneManager.RemovePane(sessionID, true)
}
