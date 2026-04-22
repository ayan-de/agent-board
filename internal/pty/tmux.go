package pty

import (
	"os"
	"os/exec"
	"strings"
)

func TmuxCmd(args ...string) error {
	cmd := exec.Command("tmux", args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func TmuxHasSession(name string) bool {
	return exec.Command("tmux", "has-session", "-t", name).Run() == nil
}

func IsInTmux() bool {
	return os.Getenv("TMUX") != ""
}

func BuildAgentCommand(self string, agentName string, autoExit bool, prompt string) string {
	parts := []string{self, "-" + agentName}
	if autoExit {
		parts = append(parts, "-auto-exit")
	}
	parts = append(parts, prompt)
	for i, p := range parts {
		if strings.Contains(p, " ") || strings.Contains(p, "\"") {
			parts[i] = "'" + strings.ReplaceAll(p, "'", "'\\''") + "'"
		}
	}
	return strings.Join(parts, " ")
}
