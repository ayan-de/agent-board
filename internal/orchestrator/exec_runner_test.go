package orchestrator_test

import (
	"context"
	"testing"

	"github.com/ayan-de/agent-board/internal/orchestrator"
)

type fakeCmdRunner struct {
	stdout   string
	stderr   string
	runError error
}

func (f fakeCmdRunner) Output() ([]byte, error) {
	if f.runError != nil {
		return nil, f.runError
	}
	return []byte(f.stdout), nil
}

func TestExecRunnerParsesStructuredOutcome(t *testing.T) {
	runner := orchestrator.ExecRunner{
		LookPath: func(name string) (string, error) {
			return "/bin/echo", nil
		},
		Command: func(ctx context.Context, name string, args ...string) orchestrator.CmdRunner {
			return fakeCmdRunner{
				stdout: `{"outcome":"completed","summary":"done"}`,
			}
		},
	}

	handle, err := runner.Start(context.Background(), orchestrator.RunRequest{
		TicketID: "AGE-01",
		Agent:    "opencode",
		Prompt:   "do work",
	})
	if err != nil {
		t.Fatal(err)
	}
	if handle.Outcome != "completed" {
		t.Fatalf("Outcome = %q, want completed", handle.Outcome)
	}
}
