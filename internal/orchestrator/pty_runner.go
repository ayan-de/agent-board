package orchestrator

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "strings"

    "github.com/ayan-de/agent-board/internal/pty"
)

type PtyRunner struct {
    runner       *pty.PtyRunner
    tmuxMode     bool
    sessionName  string
    activePanes  map[string]string // sessionID -> paneID
}

func NewPtyRunner(tmuxSession string) (*PtyRunner, error) {
    runner := pty.NewPtyRunner(tmuxSession)
    return &PtyRunner{
        runner:      runner,
        tmuxMode:    pty.IsInTmux(),
        sessionName: tmuxSession,
        activePanes: make(map[string]string),
    }, nil
}

func (r *PtyRunner) Start(ctx context.Context, req RunRequest) (RunHandle, error) {
    if !r.tmuxMode {
        return RunHandle{}, fmt.Errorf("pty runner requires tmux session")
    }

    // Create a tmux window for this agent
    windowName := fmt.Sprintf("agent-%s", req.SessionID[:8])
    cmd := exec.Command("tmux", "new-window", "-d", "-P", "-F", "#{pane_id}", "-n", windowName)
    output, err := cmd.Output()
    if err != nil {
        return RunHandle{}, fmt.Errorf("create tmux window: %w", err)
    }
    paneID := strings.TrimSpace(string(output))
    r.activePanes[req.SessionID] = paneID

    // Write prompt to file to avoid escaping issues
    homeDir, _ := os.UserHomeDir()
    cacheDir := fmt.Sprintf("%s/.agentboard/cache", homeDir)
    os.MkdirAll(cacheDir, 0755)
    promptFile := fmt.Sprintf("%s/prompt-%s.txt", cacheDir, req.SessionID)
    os.WriteFile(promptFile, []byte(req.Prompt), 0644)

    // Get agent binary
    bin := req.Agent
    if bin == "opencode" || bin == "claude-code" {
        bin = "claude" // map display names to binaries
    }

    // Build command: run agent with prompt from file
    agentCmd := fmt.Sprintf("%s run \"$(cat %s)\"", bin, promptFile)

    // Send command to the tmux pane
    sendCmd := exec.Command("tmux", "send-keys", "-t", paneID, agentCmd, "Enter")
    if err := sendCmd.Run(); err != nil {
        return RunHandle{}, fmt.Errorf("send keys to pane: %w", err)
    }

    if req.Reporter != nil {
        req.Reporter(fmt.Sprintf("Agent %s started in tmux pane %s", req.Agent, paneID))
    }

    return RunHandle{
        Outcome: "running",
        Summary: fmt.Sprintf("Agent %s in pane %s", req.Agent, paneID),
    }, nil
}

func (r *PtyRunner) GetRunner() *pty.PtyRunner {
    return r.runner
}

func (r *PtyRunner) GetPaneID(sessionID string) (string, bool) {
    paneID, ok := r.activePanes[sessionID]
    return paneID, ok
}