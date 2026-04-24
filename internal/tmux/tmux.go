package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// IsInTmux returns true if the current process is running inside tmux.
func IsInTmux() bool {
	return os.Getenv("TMUX") != ""
}

// SplitVertical splits the current pane vertically and returns the ID of the new pane.
func SplitVertical(percent int, command string) (string, error) {
	if !IsInTmux() {
		return "", fmt.Errorf("not in tmux")
	}

	args := []string{"split-window", "-h"}
	if percent > 0 {
		args = append(args, "-p", fmt.Sprintf("%d", percent))
	}
	if command != "" {
		args = append(args, command)
	}
	// Print the pane ID of the new pane
	args = append(args, "-P", "-F", "#{pane_id}")

	out, err := exec.Command("tmux", args...).Output()
	if err != nil {
		return "", fmt.Errorf("tmux split: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// KillPane kills a specific pane by ID.
func KillPane(id string) error {
	if id == "" {
		return nil
	}
	return exec.Command("tmux", "kill-pane", "-t", id).Run()
}

// RespawnPane restarts a command in an existing pane.
func RespawnPane(id, command string) error {
	if id == "" {
		return fmt.Errorf("no pane id")
	}
	return exec.Command("tmux", "respawn-pane", "-k", "-t", id, command).Run()
}

// GetCurrentPaneID returns the current pane ID.
func GetCurrentPaneID() string {
	return os.Getenv("TMUX_PANE")
}

// GetCurrentSessionName returns the active tmux session name.
func GetCurrentSessionName() (string, error) {
	if !IsInTmux() {
		return "", fmt.Errorf("not in tmux")
	}

	out, err := exec.Command("tmux", "display-message", "-p", "#S").Output()
	if err != nil {
		return "", fmt.Errorf("tmux display-message: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}
