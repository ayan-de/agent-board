package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type CmdRunner interface {
	Output() ([]byte, error)
}

type ExecRunner struct {
	LookPath func(string) (string, error)
	Command  func(context.Context, string, ...string) CmdRunner
}

func NewExecRunner() *ExecRunner {
	return &ExecRunner{
		LookPath: exec.LookPath,
		Command: func(ctx context.Context, name string, args ...string) CmdRunner {
			return exec.CommandContext(ctx, name, args...)
		},
	}
}

func (r ExecRunner) Start(ctx context.Context, req RunRequest) (RunHandle, error) {
	path, err := r.LookPath(req.Agent)
	if err != nil {
		return RunHandle{}, fmt.Errorf("execRunner.start: %w", err)
	}
	cmd := r.Command(ctx, path, req.Prompt)
	out, err := cmd.Output()
	if err != nil {
		return RunHandle{Outcome: "failed", Summary: err.Error()}, nil
	}
	var result struct {
		Outcome string `json:"outcome"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return RunHandle{Outcome: "interrupted", Summary: string(out)}, nil
	}
	return RunHandle{Outcome: result.Outcome, Summary: result.Summary}, nil
}
