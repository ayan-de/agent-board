package orchestrator

import (
	"context"
	"fmt"

	"github.com/ayan-de/agent-board/internal/pty"
)

type TmuxRunner struct {
	paneManager *PaneManager
}

func NewTmuxRunner(registry map[string]*pty.AgentConfig, chdir string) (*TmuxRunner, error) {
	pm, err := NewPaneManager(registry, chdir)
	if err != nil {
		return nil, err
	}
	return &TmuxRunner{paneManager: pm}, nil
}

func (r *TmuxRunner) Start(ctx context.Context, req RunRequest) (RunHandle, error) {
	pane, err := r.paneManager.CreatePane(ctx, req)
	if err != nil {
		return RunHandle{}, fmt.Errorf("tmuxRunner.start: %w", err)
	}

	if req.Reporter != nil {
		req.Reporter(fmt.Sprintf("Agent %s started in tmux pane %s", req.Agent, pane.PaneID))
	}

	return RunHandle{
		Outcome: "running",
		Summary: fmt.Sprintf("Agent %s started in tmux", req.Agent),
	}, nil
}

func (r *TmuxRunner) SendInput(sessionID, input string) error {
	return r.paneManager.SendInput(sessionID, input)
}

func (r *TmuxRunner) CapturePane(sessionID string, lines int) (string, error) {
	return r.paneManager.CapturePane(sessionID, lines)
}

func (r *TmuxRunner) Resize(sessionID string, rows, cols int) error {
	return r.paneManager.Resize(sessionID, rows, cols)
}

func (r *TmuxRunner) StopPane(sessionID string) error {
	return r.paneManager.RemovePane(sessionID, true)
}

func (r *TmuxRunner) GetPaneManager() *PaneManager {
	return r.paneManager
}
