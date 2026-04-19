package orchestrator_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ayan-de/agent-board/internal/orchestrator"
)

func TestParseOpencodeOutputCompleted(t *testing.T) {
	input := `{"type":"text","part":{"text":"Hello!","type":"text"}}
{"type":"step_finish","part":{"reason":"stop"}}`
	handle, err := orchestrator.ParseOpencodeOutput(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if handle.Outcome != "completed" {
		t.Fatalf("Outcome = %q, want completed", handle.Outcome)
	}
	if handle.Summary != "Hello!" {
		t.Fatalf("Summary = %q, want Hello!", handle.Summary)
	}
}

func TestParseOpencodeOutputFailed(t *testing.T) {
	input := `{"type":"text","part":{"text":"something went wrong","type":"text"}}
{"type":"step_finish","part":{"reason":"error"}}`
	handle, err := orchestrator.ParseOpencodeOutput(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if handle.Outcome != "failed" {
		t.Fatalf("Outcome = %q, want failed", handle.Outcome)
	}
}

func TestParseOpencodeOutputMultipleTexts(t *testing.T) {
	input := `{"type":"text","part":{"text":"step 1","type":"text"}}
{"type":"text","part":{"text":"step 2","type":"text"}}
{"type":"step_finish","part":{"reason":"stop"}}`
	handle, err := orchestrator.ParseOpencodeOutput(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if handle.Summary != "step 1\nstep 2" {
		t.Fatalf("Summary = %q, want step 1\\nstep 2", handle.Summary)
	}
}

func TestParseOpencodeOutputEmpty(t *testing.T) {
	handle, err := orchestrator.ParseOpencodeOutput(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if handle.Outcome != "completed" {
		t.Fatalf("Outcome = %q, want completed", handle.Outcome)
	}
	if handle.Summary != "Agent finished its task (UI mode)." {
		t.Fatalf("Summary = %q, want default", handle.Summary)
	}
}

func TestParseOpencodeOutputNonJSON(t *testing.T) {
	handle, err := orchestrator.ParseOpencodeOutput(strings.NewReader("not json\nalso not json"))
	if err != nil {
		t.Fatal(err)
	}
	if handle.Outcome != "completed" {
		t.Fatalf("Outcome = %q, want completed for non-JSON input", handle.Outcome)
	}
}

func TestExecRunnerAgentNotFound(t *testing.T) {
	runner := orchestrator.ExecRunner{
		LookPath: func(name string) (string, error) {
			return "", fmt.Errorf("not found")
		},
	}
	_, err := runner.Start(nil, orchestrator.RunRequest{
		Agent:  "nonexistent",
		Prompt: "do work",
	})
	if err == nil {
		t.Fatal("expected error for missing agent")
	}
}
