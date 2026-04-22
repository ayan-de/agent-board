package pty

import (
	"regexp"
	"time"
)

func NewCodex() *Config {
	return &Config{
		Name:            "codex",
		Bin:             "codex",
		Args:            []string{"--no-alt-screen"},
		ReadyPattern:    regexp.MustCompile(`(?m)^\s*›\s*$|Run\s+/review\s+on\s+my\s+current\s+changes`),
		SendPrompt:      SendPromptSingleLine,
		GracePeriod:     10 * time.Second,
		FallbackTimeout: 8 * time.Second,
		ReadyWait:       1 * time.Second,
		FormatPrompt:    DefaultFormatPrompt,
		IdlePatterns:    []*regexp.Regexp{regexp.MustCompile(`Run\s+/review\s+on\s+my\s+current\s+changes`)},
	}
}