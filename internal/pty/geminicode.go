package pty

import (
	"regexp"
	"time"
)

func NewGeminiCode() *Config {
	return &Config{
		Name:            "gemini",
		Bin:             "gemini",
		Args:            []string{"-y"},
		ReadyPattern:    regexp.MustCompile(`Gemini\s+CLI`),
		SendPrompt:      SendPromptSingleLine,
		GracePeriod:     10 * time.Second,
		FallbackTimeout: 10 * time.Second,
		ReadyWait:       2 * time.Second,
		FormatPrompt:    ClaudeFormatPrompt,
		IdlePatterns:    []*regexp.Regexp{regexp.MustCompile(`Type\s+your\s+message`)},
	}
}