package orchestrator

import (
	"regexp"
	"strings"
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