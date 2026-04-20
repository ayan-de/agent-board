package pty

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

const DoneMarker = "P0MX_DONE_SIGNAL"

type AgentConfig struct {
	Name            string
	Bin             string
	Args            []string
	ReadyPattern    string
	SendPrompt      func(ptmx *os.File, prompt string)
	FormatPrompt    func(prompt string) string
	IdlePatterns    []string
	GracePeriod     time.Duration
	FallbackTimeout time.Duration
	ReadyWait       time.Duration
}

func NewRegistry() map[string]*AgentConfig {
	return map[string]*AgentConfig{
		"opencode":    newOpenCode(),
		"claude-code": newClaudeCode(),
		"codex":       newCodex(),
	}
}

func newOpenCode() *AgentConfig {
	return &AgentConfig{
		Name:            "opencode",
		Bin:             "opencode",
		ReadyPattern:    `Ask\s+anything`,
		SendPrompt:      SendPromptTyped,
		GracePeriod:     8 * time.Second,
		FallbackTimeout: 5 * time.Second,
		ReadyWait:       800 * time.Millisecond,
		FormatPrompt:    func(p string) string { return DefaultFormatPrompt(p, DoneMarker) },
		IdlePatterns:    []string{}, // Rely on DoneMarker for completion
	}
}

func newClaudeCode() *AgentConfig {
	return &AgentConfig{
		Name:            "claude-code",
		Bin:             "claude",
		ReadyPattern:    `Press\s+Ctrl-C\s+again\s+to\s+exit`,
		SendPrompt:      SendPromptSingleLine,
		GracePeriod:     10 * time.Second,
		FallbackTimeout: 10 * time.Second,
		ReadyWait:       2 * time.Second,
		FormatPrompt:    func(p string) string { return ClaudeFormatPrompt(p, DoneMarker) },
		IdlePatterns:    []string{`Press\s+Ctrl-C\s+again\s+to\s+exit`},
	}
}

func newCodex() *AgentConfig {
	return &AgentConfig{
		Name:            "codex",
		Bin:             "codex",
		Args:            []string{"--no-alt-screen"},
		ReadyPattern:    `OpenAI\s+Codex|Run\s+/review\s+on\s+my\s+current\s+changes`,
		SendPrompt:      SendPromptTyped,
		GracePeriod:     10 * time.Second,
		FallbackTimeout: 8 * time.Second,
		ReadyWait:       1 * time.Second,
		FormatPrompt:    func(p string) string { return DefaultFormatPrompt(p, DoneMarker) },
		IdlePatterns:    []string{`Run\s+/review\s+on\s+my\s+current\s+changes`},
	}
}

func DefaultFormatPrompt(prompt, doneMarker string) string {
	return fmt.Sprintf(
		"%s\n\nIMPORTANT: After you have fully completed all the above tasks, you MUST print exactly this line on its own: %s. Do not skip this.",
		prompt, doneMarker,
	)
}

func ClaudeFormatPrompt(prompt, doneMarker string) string {
	return fmt.Sprintf(
		"%s. IMPORTANT: After fully completing all tasks, print exactly this on its own line: %s",
		prompt, doneMarker,
	)
}

func SendPromptTyped(ptmx *os.File, prompt string) {
	ptmx.Write([]byte{0x15})
	time.Sleep(50 * time.Millisecond)
	ptmx.Write([]byte{0x17})
	time.Sleep(50 * time.Millisecond)
	ptmx.Write([]byte(prompt))
	time.Sleep(100 * time.Millisecond)
	ptmx.Write([]byte{0x0d})
}

func SendPromptSingleLine(ptmx *os.File, prompt string) {
	singleLine := strings.ReplaceAll(prompt, "\n", " ")
	singleLine = strings.ReplaceAll(singleLine, "\r", " ")
	ptmx.Write([]byte(singleLine))
	time.Sleep(300 * time.Millisecond)
	ptmx.Write([]byte{0x0d})
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?\x07|\x1b\[.*?m`)

func StripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}
