package orchestrator

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type CmdOutputRunner interface {
	Output() ([]byte, error)
}

type ExecRunner struct {
	LookPath func(string) (string, error)
	Command  func(context.Context, string, ...string) CmdOutputRunner
}

func NewExecRunner() *ExecRunner {
	return &ExecRunner{
		LookPath: exec.LookPath,
		Command: func(ctx context.Context, name string, args ...string) CmdOutputRunner {
			return exec.CommandContext(ctx, name, args...)
		},
	}
}

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

func (r ExecRunner) Start(ctx context.Context, req RunRequest) (RunHandle, error) {
	path, err := r.LookPath(req.Agent)
	if err != nil {
		return RunHandle{}, fmt.Errorf("execRunner.start: %w", err)
	}

	out, err := r.Command(ctx, path, "run", "--format", "json", req.Prompt).Output()
	if err != nil {
		return RunHandle{Outcome: "failed", Summary: err.Error()}, nil
	}

	return parseOpencodeOutput(bytes.NewReader(out))
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
		summary = "agent completed with no text output"
	}

	outcome := "completed"
	if lastReason == "error" {
		outcome = "failed"
	}

	return RunHandle{Outcome: outcome, Summary: summary}, nil
}
