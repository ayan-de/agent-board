package orchestrator

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/ayan-de/agent-board/internal/pty"
)

var resumePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(opencode\s+-s\s+\S+)`),
	regexp.MustCompile(`(?i)(claude\s+--resume\s+\S+)`),
	regexp.MustCompile(`(?i)(codex\s+--resume\s+\S+)`),
	regexp.MustCompile(`(?i)(gemini\s+--resume\s+\S+)`),
}

func ExtractResumeCommand(output string) string {
	if output == "" {
		return ""
	}
	output = pty.StripANSI(output)
	lines := strings.Split(output, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		for _, pattern := range resumePatterns {
			match := pattern.FindStringSubmatch(line)
			if len(match) > 1 {
				return match[1]
			}
		}
	}
	return ""
}

func captureFinalPaneOutput(lastCaptured string, capture func() (string, error), delay func(time.Duration)) string {
	captured := lastCaptured
	for retry := 0; retry < 5; retry++ {
		if delay != nil {
			delay(100 * time.Millisecond)
		}
		fresh, err := capture()
		if err != nil || fresh == "" {
			continue
		}
		if captured == "" {
			captured = fresh
			continue
		}
		if !strings.Contains(captured, fresh) {
			captured += "\n" + fresh
		}
	}
	return captured
}

func captureTmuxPaneOutput(tmuxBin, paneID string, lines int) (string, error) {
	capture := func(args ...string) (string, error) {
		output, err := exec.Command(tmuxBin, args...).Output()
		if err != nil {
			return "", err
		}
		return string(output), nil
	}

	baseArgs := []string{"capture-pane", "-t", paneID, "-p", "-e", "-J", "-S", fmt.Sprintf("-%d", lines)}
	primary, primaryErr := capture(baseArgs...)

	altArgs := append([]string{}, baseArgs...)
	altArgs = append(altArgs[:1], append([]string{"-a"}, altArgs[1:]...)...)
	alternate, altErr := capture(altArgs...)

	switch {
	case primaryErr != nil && altErr != nil:
		return "", primaryErr
	case primaryErr != nil:
		return alternate, nil
	case altErr != nil || alternate == "" || strings.Contains(primary, alternate):
		return primary, nil
	case primary == "":
		return alternate, nil
	default:
		return primary + "\n" + alternate, nil
	}
}
