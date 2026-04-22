package pty

import (
	"regexp"
	"time"
)

func NewClaudeCode() *Config {
	return &Config{
		Name:            "claude-code",
		Bin:             "claude",
		Args:            []string{},
		ReadyPattern:    regexp.MustCompile(`Press\s+Ctrl-C\s+again\s+to\s+exit`),
		SendPrompt:      SendPromptSingleLine,
		GracePeriod:     10 * time.Second,
		FallbackTimeout: 10 * time.Second,
		ReadyWait:       2 * time.Second,
		FormatPrompt:    ClaudeFormatPrompt,
		IdlePatterns:    []*regexp.Regexp{regexp.MustCompile(`Press\s+Ctrl-C\s+again\s+to\s+exit`)},
	}
}