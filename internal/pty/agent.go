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

func NewRegistry() map[string]*Config {
	return map[string]*Config{
		"opencode": {
			Name:            "opencode",
			Bin:             "opencode",
			Args:            []string{},
			FormatPrompt:    DefaultFormatPrompt,
			SendPrompt:      SendPromptSingleLine,
			ReadyWait:       5 * time.Second,
			FallbackTimeout: 120 * time.Second,
		},
		"claudecode": {
			Name:            "claudecode",
			Bin:             "claude",
			Args:            []string{},
			FormatPrompt:    ClaudeFormatPrompt,
			SendPrompt:      SendPromptSingleLine,
			ReadyWait:       5 * time.Second,
			FallbackTimeout: 120 * time.Second,
		},
		"codex": {
			Name:            "codex",
			Bin:             "codex",
			Args:            []string{},
			FormatPrompt:    DefaultFormatPrompt,
			SendPrompt:      SendPromptSingleLine,
			ReadyWait:       5 * time.Second,
			FallbackTimeout: 120 * time.Second,
		},
		"gemini": {
			Name:            "gemini",
			Bin:             "gemini",
			Args:            []string{},
			FormatPrompt:    DefaultFormatPrompt,
			SendPrompt:      SendPromptSingleLine,
			ReadyWait:       5 * time.Second,
			FallbackTimeout: 120 * time.Second,
		},
	}
}
