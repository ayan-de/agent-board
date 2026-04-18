package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type TmuxRunner struct {
	LookPath func(string) (string, error)
}

func NewTmuxRunner() *TmuxRunner {
	return &TmuxRunner{
		LookPath: exec.LookPath,
	}
}

func (r TmuxRunner) Start(ctx context.Context, req RunRequest) (RunHandle, error) {
	tmuxPath, err := r.LookPath("tmux")
	if err != nil {
		return RunHandle{}, fmt.Errorf("tmuxRunner.start: tmux not found: %w", err)
	}

	agentPath, err := r.LookPath(req.Agent)
	if err != nil {
		return RunHandle{}, fmt.Errorf("tmuxRunner.start: agent %s not found: %w", req.Agent, err)
	}

	sessionName := fmt.Sprintf("agentboard-%s", req.TicketID)
	// Clean up existing session
	_ = exec.Command(tmuxPath, "kill-session", "-t", sessionName).Run()

	// We'll use a temp file to capture the raw output for parsing
	logDir := filepath.Join(os.TempDir(), "agentboard")
	_ = os.MkdirAll(logDir, 0755)
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", req.SessionID))
	
	// Command to run inside tmux: run the agent and tee to log file
	// We use --format json to ensure we can parse the outcome.
	innerCmd := fmt.Sprintf("%s run --format json %q | tee %s", agentPath, req.Prompt, logFile)
	
	// Create detached session
	cmd := exec.Command(tmuxPath, "new-session", "-d", "-s", sessionName, innerCmd)
	if err := cmd.Run(); err != nil {
		return RunHandle{}, fmt.Errorf("tmuxRunner.start: failed to create tmux session: %w", err)
	}

	if req.Reporter != nil {
		req.Reporter(fmt.Sprintf("Tmux session %s created. Run 'tmux attach -t %s' to watch live.", sessionName, sessionName))
	}

	// Poll for completion and stream logs from the file
	handleChan := make(chan RunHandle)
	errChan := make(chan error)

	go func() {
		defer os.Remove(logFile)

		lastOffset := int64(0)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				_ = exec.Command(tmuxPath, "kill-session", "-t", sessionName).Run()
				errChan <- ctx.Err()
				return
			case <-ticker.C:
				// Read new lines from log file
				newContent, newOffset, _ := readNewLines(logFile, lastOffset)
				if len(newContent) > 0 {
					lines := strings.Split(strings.TrimSpace(string(newContent)), "\n")
					for _, line := range lines {
						if req.Reporter != nil {
							// Filter/Format JSON as we do in exec_runner
							// For simplicity, we can reuse parseOpencodeOutput logic later
							req.Reporter(line) 
						}
					}
					lastOffset = newOffset
				}

				// Check if tmux session still exists
				if err := exec.Command(tmuxPath, "has-session", "-t", sessionName).Run(); err != nil {
					// Session ended
					finalOutput, _ := os.ReadFile(logFile)
					handle, err := parseOpencodeOutput(bytes.NewReader(finalOutput))
					if err != nil {
						handleChan <- RunHandle{Outcome: "completed", Summary: "Session finished but could not parse result."}
					} else {
						handleChan <- handle
					}
					return
				}
			}
		}
	}()

	select {
	case handle := <-handleChan:
		return handle, nil
	case err := <-errChan:
		return RunHandle{}, err
	}
}

func readNewLines(path string, offset int64) ([]byte, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, offset, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, offset, err
	}

	if info.Size() <= offset {
		return nil, offset, nil
	}

	_, err = f.Seek(offset, 0)
	if err != nil {
		return nil, offset, err
	}

	buf := make([]byte, info.Size()-offset)
	_, err = f.Read(buf)
	if err != nil {
		return nil, offset, err
	}

	return buf, info.Size(), nil
}
