package orchestrator

import (
	"context"
	"fmt"

	"github.com/ayan-de/agent-board/internal/pty"
)

type PtyRunner struct {
	runner   *pty.PtyRunner
	tmuxMode bool
}

func NewPtyRunner(tmuxSession string) (*PtyRunner, error) {
	runner := pty.NewPtyRunner(tmuxSession)
	return &PtyRunner{
		runner:   runner,
		tmuxMode: pty.IsInTmux(),
	}, nil
}

func (r *PtyRunner) Start(ctx context.Context, req RunRequest) (RunHandle, error) {
	autoExit := true
	if err := r.runner.Start(req.SessionID, req.Agent, req.Prompt, autoExit); err != nil {
		return RunHandle{}, fmt.Errorf("ptyRunner.start: %w", err)
	}
	return RunHandle{}, nil
}

func (r *PtyRunner) GetRunner() *pty.PtyRunner {
	return r.runner
}