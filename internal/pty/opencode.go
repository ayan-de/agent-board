package pty

import (
	"regexp"
	"time"
)

func NewOpenCode() *Config {
	return &Config{
		Name:            "opencode",
		Bin:             "opencode",
		Args:            []string{},
		ReadyPattern:    regexp.MustCompile(`Ask\s+anything`),
		SendPrompt:      SendPromptTyped,
		GracePeriod:     8 * time.Second,
		FallbackTimeout: 5 * time.Second,
		ReadyWait:       800 * time.Millisecond,
		FormatPrompt:    DefaultFormatPrompt,
		IdlePatterns:    []*regexp.Regexp{regexp.MustCompile(`Ask\s+anything`)},
	}
}