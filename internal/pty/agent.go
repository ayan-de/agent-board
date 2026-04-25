package pty

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

const DoneMarker = "P0MX_DONE_SIGNAL"

type SendPromptFunc func(*os.File, string) error

type Config struct {
	Name            string
	Bin             string
	Args            []string
	ReadyPattern    *regexp.Regexp
	SendPrompt      SendPromptFunc
	FormatPrompt    func(string) string
	IdlePatterns    []*regexp.Regexp
	GracePeriod     time.Duration
	FallbackTimeout time.Duration
	ReadyWait       time.Duration
}

func DefaultFormatPrompt(prompt string) string {
	return fmt.Sprintf("%s\n%s", DoneMarker, prompt)
}

func ClaudeFormatPrompt(prompt string) string {
	return fmt.Sprintf("%s\n%s", DoneMarker, prompt)
}

func SendPromptTyped(ptmx *os.File, prompt string) error {
	controls := []byte{0x15, 0x17}
	for _, c := range controls {
		if _, err := ptmx.Write([]byte{c}); err != nil {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
	for _, c := range prompt {
		if _, err := ptmx.Write([]byte{byte(c)}); err != nil {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := ptmx.Write([]byte{0x0d}); err != nil {
		return err
	}
	return nil
}

func SendPromptSingleLine(ptmx *os.File, prompt string) error {
	single := strings.ReplaceAll(prompt, "\n", " ")
	single = strings.TrimSpace(single)
	if _, err := ptmx.Write([]byte(single)); err != nil {
		return err
	}
	if _, err := ptmx.Write([]byte{0x0d}); err != nil {
		return err
	}
	return nil
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func JoinArgs(args []string) string {
	return strings.Join(args, " ")
}

func NewOpenCode() *Config {
	return &Config{
		Name:            "opencode",
		Bin:             "opencode",
		Args:            []string{},
		ReadyPattern:    regexp.MustCompile(`Ask\s+anything`),
		SendPrompt:      SendPromptTyped,
		FormatPrompt:    DefaultFormatPrompt,
		IdlePatterns:    []*regexp.Regexp{regexp.MustCompile(`Ask\s+anything`)},
		GracePeriod:     8 * time.Second,
		FallbackTimeout: 5 * time.Second,
		ReadyWait:       800 * time.Millisecond,
	}
}

func NewClaudeCode() *Config {
	return &Config{
		Name:            "claude-code",
		Bin:             "claude",
		Args:            []string{"--no-autocomplete"},
		ReadyPattern:    regexp.MustCompile(`\?\s+for\s+shortcuts`),
		SendPrompt:      SendPromptSingleLine,
		FormatPrompt:    ClaudeFormatPrompt,
		IdlePatterns:    []*regexp.Regexp{regexp.MustCompile(`Press\s+Ctrl-C\s+again\s+to\s+exit`)},
		GracePeriod:     10 * time.Second,
		FallbackTimeout: 10 * time.Second,
		ReadyWait:       5 * time.Second,
	}
}

func NewCodex() *Config {
	return &Config{
		Name:            "codex",
		Bin:             "codex",
		Args:            []string{},
		ReadyPattern:    regexp.MustCompile(`(?m)^\s*›\s*$|Run\s+/review\s+on\s+my\s+current\s+changes`),
		SendPrompt:      SendPromptSingleLine,
		FormatPrompt:    DefaultFormatPrompt,
		IdlePatterns:    []*regexp.Regexp{regexp.MustCompile(`Run\s+/review\s+on\s+my\s+current\s+changes`)},
		GracePeriod:     8 * time.Second,
		FallbackTimeout: 120 * time.Second,
		ReadyWait:       5 * time.Second,
	}
}

func NewGeminiCode() *Config {
	return &Config{
		Name:            "gemini",
		Bin:             "gemini",
		Args:            []string{},
		ReadyPattern:    regexp.MustCompile(`Gemini\s+CLI`),
		SendPrompt:      SendPromptSingleLine,
		FormatPrompt:    DefaultFormatPrompt,
		IdlePatterns:    []*regexp.Regexp{regexp.MustCompile(`Type\s+your\s+message`)},
		GracePeriod:     8 * time.Second,
		FallbackTimeout: 120 * time.Second,
		ReadyWait:       5 * time.Second,
	}
}

func NewRegistry() map[string]*Config {
	return map[string]*Config{
		"opencode":    NewOpenCode(),
		"claudecode":  NewClaudeCode(),
		"claude-code": NewClaudeCode(),
		"codex":       NewCodex(),
		"gemini":      NewGeminiCode(),
	}
}
