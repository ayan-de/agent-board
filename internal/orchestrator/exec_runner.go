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

type ExecRunner struct {
	LookPath func(string) (string, error)
}

func NewExecRunner() *ExecRunner {
	return &ExecRunner{
		LookPath: exec.LookPath,
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

	cmd := exec.CommandContext(ctx, path, "run", "--format", "json", req.Prompt)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return RunHandle{}, err
	}
	if req.InputChan != nil {
		req.InputChan <- stdin
		close(req.InputChan)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return RunHandle{}, err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return RunHandle{}, err
	}

	var fullOutput bytes.Buffer
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fullOutput.WriteString(line + "\n")

		displayLine := line
		var evt opencodeEvent
		if err := json.Unmarshal([]byte(line), &evt); err == nil {
			if evt.Type == "text" && evt.Part.Text != "" {
				displayLine = evt.Part.Text
			} else if evt.Type == "step_start" {
				displayLine = fmt.Sprintf("Step: %s", evt.Part.Reason)
			} else {
				continue // Skip other JSON events from live view
			}
		}

		if req.Reporter != nil {
			req.Reporter(displayLine)
		}
	}

	if err := cmd.Wait(); err != nil {
		// Log the error but continue to parse what we have
		if req.Reporter != nil {
			req.Reporter(fmt.Sprintf("Process exited with error: %v", err))
		}
	}

	return ParseOpencodeOutput(&fullOutput)
}

func ParseOpencodeOutput(r io.Reader) (RunHandle, error) {
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
		summary = "Agent finished its task (UI mode)."
	}

	outcome := "completed"
	if lastReason == "error" {
		outcome = "failed"
	}

	return RunHandle{Outcome: outcome, Summary: summary}, nil
}
