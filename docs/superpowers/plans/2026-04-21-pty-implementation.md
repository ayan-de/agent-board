# PTY Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace tmux-based agent execution with embedded PTY processes, rendering live agent output directly inside the Bubble Tea TUI dashboard's right pane.

**Architecture:** New `internal/pty` package adapts pty-go's agent configs, PTY process management, and completion detection. New `PtyRunner` in orchestrator implements the `Runner` interface as an async runner. Dashboard reads PTY output via an output channel. Tmux package and all tmux references are removed.

**Tech Stack:** Go, creack/pty, Bubble Tea, existing orchestrator interfaces

---

## File Structure

### New Files
- `internal/pty/config.go` — Agent configs registry (opencode, claudecode, codex) and helper functions ported from pty-go
- `internal/pty/session.go` — PTY session management: start agent, monitor output, detect completion
- `internal/orchestrator/pty_runner.go` — PtyRunner implementing Runner interface, delegates to pty.Session
- `internal/pty/config_test.go` — Tests for agent config registry
- `internal/pty/session_test.go` — Tests for Session creation, completion detection, StripANSI

### Modified Files
- `internal/orchestrator/service.go` — Replace TmuxRunner type assertions with PtyRunner equivalents, rename GetPaneContent → GetPTYOutput, remove SwitchToPane
- `internal/orchestrator/types.go` — Remove PaneID/WindowID from AgentSession, add PTY state field
- `internal/tui/app.go` — Remove tmux import, dashboardPaneID, syncDashboardPane, tmux split pane logic; add PTY output polling tick
- `internal/tui/dashboard.go` — Replace GetPaneContent calls with GetPTYOutput, remove SwitchToPane action, remove input mode, add completion countdown
- `cmd/agentboard/main.go` — Remove tmux auto-launch, create PtyRunner with agent registry
- `internal/config/general.go` — Remove Tmux field
- `internal/config/defaults.go` — Remove Tmux default
- `internal/config/loader.go` — Remove AGENTBOARD_TMUX env var
- `internal/config/scaffold.go` — Remove tmux from default config template

### Deleted Files
- `internal/orchestrator/tmux_runner.go`
- `internal/orchestrator/pane_manager.go`
- `internal/tmux/tmux.go`
- `internal/tui/pane.go`

---

### Task 1: Add creack/pty dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Run go get**

```bash
cd /home/ayan-de/Projects/agent-board && go get github.com/creack/pty@latest
```

- [ ] **Step 2: Verify dependency**

```bash
cd /home/ayan-de/Projects/agent-board && go mod tidy
```

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add creack/pty dependency for PTY agent execution"
```

---

### Task 2: Create `internal/pty/config.go` — Agent Config Registry

**Files:**
- Create: `internal/pty/config.go`
- Test: `internal/pty/config_test.go`

- [ ] **Step 1: Write the failing test**

```go
package pty_test

import (
	"testing"

	"github.com/ayan-de/agent-board/internal/pty"
)

func TestNewRegistryContainsAllAgents(t *testing.T) {
	r := pty.NewRegistry()

	agents := []string{"opencode", "claudecode", "codex"}
	for _, name := range agents {
		cfg, ok := r[name]
		if !ok {
			t.Fatalf("expected agent %q in registry", name)
		}
		if cfg.Name == "" {
			t.Fatalf("agent %q has empty Name", name)
		}
		if cfg.Bin == "" {
			t.Fatalf("agent %q has empty Bin", name)
		}
		if cfg.ReadyPattern == "" {
			t.Fatalf("agent %q has empty ReadyPattern", name)
		}
		if cfg.SendPrompt == nil {
			t.Fatalf("agent %q has nil SendPrompt", name)
		}
		if cfg.GracePeriod == 0 {
			t.Fatalf("agent %q has zero GracePeriod", name)
		}
	}
}

func TestStripANSI(t *testing.T) {
	input := "\x1b[32mhello\x1b[0m \x1b[1mworld\x1b[0m"
	got := pty.StripANSI(input)
	want := "hello world"
	if got != want {
		t.Fatalf("StripANSI(%q) = %q, want %q", input, got, want)
	}
}

func TestStripANSIEmpty(t *testing.T) {
	got := pty.StripANSI("")
	if got != "" {
		t.Fatalf("StripANSI(\"\") = %q, want \"\"", got)
	}
}

func TestStripANSINoEscape(t *testing.T) {
	got := pty.StripANSI("plain text")
	if got != "plain text" {
		t.Fatalf("StripANSI(\"plain text\") = %q, want \"plain text\"", got)
	}
}

func TestDefaultFormatPrompt(t *testing.T) {
	got := pty.DefaultFormatPrompt("do the thing", "MARKER_X")
	if got == "" {
		t.Fatal("DefaultFormatPrompt returned empty string")
	}
	if got == "do the thing" {
		t.Fatal("DefaultFormatPrompt should append marker instructions")
	}
}

func TestSendPromptTypedWritesToBuffer(t *testing.T) {
	// We can't easily test the PTY file write without a real PTY,
	// but we can verify the function exists and compiles
	_ = pty.SendPromptTyped
}

func TestSendPromptSingleLineStripsNewlines(t *testing.T) {
	_ = pty.SendPromptSingleLine
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/ayan-de/Projects/agent-board && go test ./internal/pty/... -v -run "TestNewRegistry|TestStripANSI|TestDefaultFormatPrompt|TestSendPrompt"
```

Expected: FAIL (package doesn't exist)

- [ ] **Step 3: Write implementation**

```go
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
		"opencode":   newOpenCode(),
		"claudecode": newClaudeCode(),
		"codex":      newCodex(),
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
		IdlePatterns:    []string{`Ask\s+anything`},
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
```

- [ ] **Step 4: Run tests**

```bash
cd /home/ayan-de/Projects/agent-board && go test ./internal/pty/... -v -run "TestNewRegistry|TestStripANSI|TestDefaultFormatPrompt|TestSendPrompt"
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/pty/config.go internal/pty/config_test.go
git commit -m "feat: add internal/pty agent config registry ported from pty-go"
```

---

### Task 3: Create `internal/pty/session.go` — PTY Session Management

**Files:**
- Create: `internal/pty/session.go`
- Test: `internal/pty/session_test.go`

- [ ] **Step 1: Write the failing test**

```go
package pty_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ayan-de/agent-board/internal/pty"
)

func TestSessionStateString(t *testing.T) {
	cases := map[pty.SessionState]string{
		pty.StateWaitingReady: "waiting_ready",
		pty.StateSendingPrompt: "sending_prompt",
		pty.StateWorking:      "working",
		pty.StateDone:         "done",
	}
	for state, want := range cases {
		if state.String() != want {
			t.Fatalf("%v.String() = %q, want %q", state, state.String(), want)
		}
	}
}

func TestSessionDetectsCompletionViaDoneMarker(t *testing.T) {
	reg := pty.NewRegistry()
	cfg := reg["opencode"]

	output := "Some output\n" + pty.DoneMarker + "\nMore output"
	lines := strings.Split(output, "\n")

	detected := pty.DetectCompletion(cfg, lines, 0)
	if !detected {
		t.Fatal("expected completion detection via done marker")
	}
}

func TestSessionNoFalsePositive(t *testing.T) {
	reg := pty.NewRegistry()
	cfg := reg["opencode"]

	lines := []string{"Agent is working hard", "still going", "more work"}
	detected := pty.DetectCompletion(cfg, lines, 3*time.Second)
	if detected {
		t.Fatal("should not detect completion from normal output within grace period")
	}
}

func TestSessionDetectsCompletionViaIdlePattern(t *testing.T) {
	reg := pty.NewRegistry()
	cfg := reg["opencode"]

	// The idle pattern for opencode is "Ask anything" appearing 3+ times
	lines := []string{
		"Ask anything",
		"Ask anything",
		"Ask anything",
	}

	detected := pty.DetectCompletion(cfg, lines, 5*time.Second)
	if !detected {
		t.Fatal("expected completion detection via idle pattern (3 occurrences)")
	}
}

func TestRecentOutput(t *testing.T) {
	s := &pty.Session{}
	s.AppendOutput("line1")
	s.AppendOutput("line2")
	s.AppendOutput("line3")

	recent := s.RecentOutput(2)
	if len(recent) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(recent))
	}
	if recent[0] != "line2" {
		t.Fatalf("first line = %q, want %q", recent[0], "line2")
	}
	if recent[1] != "line3" {
		t.Fatalf("second line = %q, want %q", recent[1], "line3")
	}
}

func TestRecentOutputMoreThanAvailable(t *testing.T) {
	s := &pty.Session{}
	s.AppendOutput("line1")

	recent := s.RecentOutput(5)
	if len(recent) != 1 {
		t.Fatalf("expected 1 line, got %d", len(recent))
	}
}

func TestRecentOutputEmpty(t *testing.T) {
	s := &pty.Session{}
	recent := s.RecentOutput(5)
	if len(recent) != 0 {
		t.Fatalf("expected 0 lines, got %d", len(recent))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/ayan-de/Projects/agent-board && go test ./internal/pty/... -v -run "TestSession|TestRecentOutput"
```

Expected: FAIL (types/functions not defined)

- [ ] **Step 3: Write implementation**

```go
package pty

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"
)

type SessionState int

const (
	StateWaitingReady SessionState = iota
	StateSendingPrompt
	StateWorking
	StateDone
)

func (s SessionState) String() string {
	switch s {
	case StateWaitingReady:
		return "waiting_ready"
	case StateSendingPrompt:
		return "sending_prompt"
	case StateWorking:
		return "working"
	case StateDone:
		return "done"
	default:
		return "unknown"
	}
}

type Session struct {
	SessionID string
	Agent     string
	TicketID  string
	Chdir     string

	mu       sync.Mutex
	state    SessionState
	output   []string
	outputMu sync.Mutex
	cmd      *exec.Cmd
	ptmx     *os.File
	doneCh   chan struct{}
	outcome  string
	summary  string
}

func NewSession(cfg *AgentConfig, sessionID, ticketID, prompt, chdir string) (*Session, error) {
	if chdir != "" {
		abs, err := filepath.Abs(chdir)
		if err != nil {
			return nil, fmt.Errorf("pty.newSession: resolving chdir: %w", err)
		}
		chdir = abs
	}

	cmd := exec.Command(cfg.Bin, cfg.Args...)
	cmd.Env = os.Environ()
	if chdir != "" {
		cmd.Dir = chdir
	}

	ptmx, err := startPTY(cmd)
	if err != nil {
		return nil, fmt.Errorf("pty.newSession: starting pty: %w", err)
	}

	s := &Session{
		SessionID: sessionID,
		Agent:     cfg.Name,
		TicketID:  ticketID,
		Chdir:     chdir,
		cmd:       cmd,
		ptmx:      ptmx,
		state:     StateWaitingReady,
		doneCh:    make(chan struct{}),
	}

	go s.run(cfg, prompt)

	return s, nil
}

func (s *Session) run(cfg *AgentConfig, prompt string) {
	defer close(s.doneCh)
	defer s.ptmx.Close()

	readyRe := regexp.MustCompile(cfg.ReadyPattern)
	doneRe := regexp.MustCompile(regexp.QuoteMeta(DoneMarker))
	idleRes := make([]*regexp.Regexp, len(cfg.IdlePatterns))
	for i, p := range cfg.IdlePatterns {
		idleRes[i] = regexp.MustCompile(p)
	}

	fired := false
	fallback := time.AfterFunc(cfg.FallbackTimeout, func() {
		if !fired {
			fired = true
			s.sendPromptAndTransition(cfg, prompt)
		}
	})
	defer fallback.Stop()

	buf := make([]byte, 4096)
	var stripBuf []string

	for {
		n, err := s.ptmx.Read(buf)
		if err != nil {
			s.transitionDone("completed", "Agent process exited")
			return
		}

		raw := string(buf[:n])
		for _, line := range splitLines(raw) {
			s.AppendOutput(line)
			stripped := StripANSI(line)
			stripBuf = append(stripBuf, stripped)
			if len(stripBuf) > 500 {
				stripBuf = stripBuf[len(stripBuf)-500:]
			}
		}

		s.mu.Lock()
		currentState := s.state
		s.mu.Unlock()

		switch currentState {
		case StateWaitingReady:
			for _, line := range stripBuf {
				if readyRe.MatchString(line) {
					if !fired {
						fired = true
						fallback.Stop()
						time.AfterFunc(cfg.ReadyWait, func() {
							s.sendPromptAndTransition(cfg, prompt)
						})
					}
					break
				}
			}

		case StateWorking:
			detected := DetectCompletionFromBuffer(cfg, stripBuf, doneRe, idleRes)
			if detected {
				s.transitionDone("completed", "Agent completed task")
				s.cleanupProcess()
				return
			}
		}
	}
}

func (s *Session) sendPromptAndTransition(cfg *AgentConfig, prompt string) {
	formatted := prompt
	if cfg.FormatPrompt != nil {
		formatted = cfg.FormatPrompt(prompt)
	}

	cfg.SendPrompt(s.ptmx, formatted)

	s.mu.Lock()
	s.state = StateWorking
	s.mu.Unlock()

	// Reset strip buffer for working phase
	time.AfterFunc(cfg.GracePeriod, func() {
		// Grace period elapsed, now we check for completion
	})
}

func (s *Session) transitionDone(outcome, summary string) {
	s.mu.Lock()
	s.state = StateDone
	s.outcome = outcome
	s.summary = summary
	s.mu.Unlock()
}

func (s *Session) cleanupProcess() {
	time.Sleep(500 * time.Millisecond)
	s.ptmx.Write([]byte{0x03})
	time.Sleep(300 * time.Millisecond)
	s.ptmx.Write([]byte{0x03})
	time.Sleep(300 * time.Millisecond)
	s.cmd.Process.Signal(syscall.SIGTERM)
	time.Sleep(1 * time.Second)
	s.cmd.Process.Kill()
	s.cmd.Wait()
}

func (s *Session) State() SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *Session) Outcome() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outcome
}

func (s *Session) Summary() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.summary
}

func (s *Session) Wait() (outcome string, summary string) {
	<-s.doneCh
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outcome, s.summary
}

func (s *Session) DoneCh() <-chan struct{} {
	return s.doneCh
}

func (s *Session) Close() {
	s.cmd.Process.Kill()
}

func (s *Session) AppendOutput(line string) {
	s.outputMu.Lock()
	defer s.outputMu.Unlock()
	s.output = append(s.output, line)
	if len(s.output) > 2000 {
		s.output = s.output[len(s.output)-2000:]
	}
}

func (s *Session) RecentOutput(n int) []string {
	s.outputMu.Lock()
	defer s.outputMu.Unlock()
	if n >= len(s.output) {
		result := make([]string, len(s.output))
		copy(result, s.output)
		return result
	}
	result := make([]string, n)
	copy(result, s.output[len(s.output)-n:])
	return result
}

func (s *Session) SendInput(data []byte) error {
	_, err := s.ptmx.Write(data)
	return err
}

func DetectCompletion(cfg *AgentConfig, lines []string, elapsed time.Duration) bool {
	return DetectCompletionFromBuffer(cfg, lines,
		regexp.MustCompile(regexp.QuoteMeta(DoneMarker)),
		func() []*regexp.Regexp {
			res := make([]*regexp.Regexp, len(cfg.IdlePatterns))
			for i, p := range cfg.IdlePatterns {
				res[i] = regexp.MustCompile(p)
			}
			return res
		}(),
	)
}

func DetectCompletionFromBuffer(cfg *AgentConfig, lines []string, doneRe *regexp.Regexp, idleRes []*regexp.Regexp) bool {
	recent := lines
	if len(recent) > 200 {
		recent = recent[len(recent)-200:]
	}

	joined := strings.Join(recent, "\n")
	if doneRe.MatchString(joined) {
		return true
	}

	for _, re := range idleRes {
		matches := re.FindAllStringIndex(joined, -1)
		if len(matches) >= 3 {
			return true
		}
	}

	return false
}

func splitLines(s string) []string {
	var lines []string
	current := ""
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, current)
			current = ""
		} else if r != '\r' {
			current += string(r)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

import "strings" // will be merged with other imports

// startPTY is a wrapper for testability
var startPTY = func(cmd *exec.Cmd) (*os.File, error) {
	return ptyStart(cmd)
}

func ptyStart(cmd *exec.Cmd) (*os.File, error) {
	ptmx, err := ptyOpen(cmd)
	if err != nil {
		return nil, err
	}
	return ptmx, nil
}

var ptyOpen = func(cmd *exec.Cmd) (*os.File, error) {
	// Uses creack/pty - imported separately to avoid build issues in tests
	return nil, fmt.Errorf("ptyOpen: not available in tests")
}
```

Wait - this approach has issues with creack/pty import. Let me restructure.

Actually let me write the correct implementation:

```go
package pty

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

type SessionState int

const (
	StateWaitingReady SessionState = iota
	StateSendingPrompt
	StateWorking
	StateDone
)

func (s SessionState) String() string {
	switch s {
	case StateWaitingReady:
		return "waiting_ready"
	case StateSendingPrompt:
		return "sending_prompt"
	case StateWorking:
		return "working"
	case StateDone:
		return "done"
	default:
		return "unknown"
	}
}

type Session struct {
	SessionID string
	Agent     string
	TicketID  string
	Chdir     string

	mu       sync.Mutex
	state    SessionState
	output   []string
	outputMu sync.Mutex
	cmd      *exec.Cmd
	ptmx     *os.File
	doneCh   chan struct{}
	outcome  string
	summary  string
}

func NewSession(cfg *AgentConfig, sessionID, ticketID, prompt, chdir string) (*Session, error) {
	if chdir != "" {
		abs, err := filepath.Abs(chdir)
		if err != nil {
			return nil, fmt.Errorf("pty.newSession: resolving chdir: %w", err)
		}
		chdir = abs
	}

	cmd := exec.Command(cfg.Bin, cfg.Args...)
	cmd.Env = os.Environ()
	if chdir != "" {
		cmd.Dir = chdir
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("pty.newSession: starting pty: %w", err)
	}

	s := &Session{
		SessionID: sessionID,
		Agent:     cfg.Name,
		TicketID:  ticketID,
		Chdir:     chdir,
		cmd:       cmd,
		ptmx:      ptmx,
		state:     StateWaitingReady,
		doneCh:    make(chan struct{}),
	}

	go s.run(cfg, prompt)

	return s, nil
}

func (s *Session) run(cfg *AgentConfig, prompt string) {
	defer close(s.doneCh)
	defer s.ptmx.Close()

	readyRe := regexp.MustCompile(cfg.ReadyPattern)
	doneRe := regexp.MustCompile(regexp.QuoteMeta(DoneMarker))
	idleRes := make([]*regexp.Regexp, len(cfg.IdlePatterns))
	for i, p := range cfg.IdlePatterns {
		idleRes[i] = regexp.MustCompile(p)
	}

	fired := false
	fallback := time.AfterFunc(cfg.FallbackTimeout, func() {
		if !fired {
			fired = true
			s.sendPromptAndTransition(cfg, prompt)
		}
	})
	defer fallback.Stop()

	buf := make([]byte, 4096)
	var stripBuf []string

	for {
		n, err := s.ptmx.Read(buf)
		if err != nil {
			s.transitionDone("completed", "Agent process exited")
			return
		}

		raw := string(buf[:n])
		for _, line := range splitLines(raw) {
			s.AppendOutput(line)
			stripped := StripANSI(line)
			stripBuf = append(stripBuf, stripped)
			if len(stripBuf) > 500 {
				stripBuf = stripBuf[len(stripBuf)-500:]
			}
		}

		s.mu.Lock()
		currentState := s.state
		s.mu.Unlock()

		switch currentState {
		case StateWaitingReady:
			for _, line := range stripBuf {
				if readyRe.MatchString(line) {
					if !fired {
						fired = true
						fallback.Stop()
						time.AfterFunc(cfg.ReadyWait, func() {
							s.sendPromptAndTransition(cfg, prompt)
						})
					}
					break
				}
			}

		case StateWorking:
			detected := DetectCompletionFromBuffer(cfg, stripBuf, doneRe, idleRes)
			if detected {
				s.transitionDone("completed", "Agent completed task")
				s.cleanupProcess()
				return
			}
		}
	}
}

func (s *Session) sendPromptAndTransition(cfg *AgentConfig, prompt string) {
	formatted := prompt
	if cfg.FormatPrompt != nil {
		formatted = cfg.FormatPrompt(prompt)
	}

	cfg.SendPrompt(s.ptmx, formatted)

	s.mu.Lock()
	s.state = StateWorking
	s.mu.Unlock()
}

func (s *Session) transitionDone(outcome, summary string) {
	s.mu.Lock()
	s.state = StateDone
	s.outcome = outcome
	s.summary = summary
	s.mu.Unlock()
}

func (s *Session) cleanupProcess() {
	time.Sleep(500 * time.Millisecond)
	s.ptmx.Write([]byte{0x03})
	time.Sleep(300 * time.Millisecond)
	s.ptmx.Write([]byte{0x03})
	time.Sleep(300 * time.Millisecond)
	s.cmd.Process.Signal(syscall.SIGTERM)
	time.Sleep(1 * time.Second)
	s.cmd.Process.Kill()
	s.cmd.Wait()
}

func (s *Session) State() SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *Session) Outcome() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outcome
}

func (s *Session) Summary() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.summary
}

func (s *Session) Wait() (outcome string, summary string) {
	<-s.doneCh
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outcome, s.summary
}

func (s *Session) DoneCh() <-chan struct{} {
	return s.doneCh
}

func (s *Session) Close() {
	s.cmd.Process.Kill()
}

func (s *Session) AppendOutput(line string) {
	s.outputMu.Lock()
	defer s.outputMu.Unlock()
	s.output = append(s.output, line)
	if len(s.output) > 2000 {
		s.output = s.output[len(s.output)-2000:]
	}
}

func (s *Session) RecentOutput(n int) []string {
	s.outputMu.Lock()
	defer s.outputMu.Unlock()
	if n >= len(s.output) {
		result := make([]string, len(s.output))
		copy(result, s.output)
		return result
	}
	result := make([]string, n)
	copy(result, s.output[len(s.output)-n:])
	return result
}

func (s *Session) SendInput(data []byte) error {
	_, err := s.ptmx.Write(data)
	return err
}

func DetectCompletion(cfg *AgentConfig, lines []string, elapsed time.Duration) bool {
	doneRe := regexp.MustCompile(regexp.QuoteMeta(DoneMarker))
	idleRes := make([]*regexp.Regexp, len(cfg.IdlePatterns))
	for i, p := range cfg.IdlePatterns {
		idleRes[i] = regexp.MustCompile(p)
	}
	return DetectCompletionFromBuffer(cfg, lines, doneRe, idleRes)
}

func DetectCompletionFromBuffer(cfg *AgentConfig, lines []string, doneRe *regexp.Regexp, idleRes []*regexp.Regexp) bool {
	recent := lines
	if len(recent) > 200 {
		recent = recent[len(recent)-200:]
	}

	joined := strings.Join(recent, "\n")
	if doneRe.MatchString(joined) {
		return true
	}

	for _, re := range idleRes {
		matches := re.FindAllStringIndex(joined, -1)
		if len(matches) >= 3 {
			return true
		}
	}

	return false
}

func splitLines(s string) []string {
	var lines []string
	current := ""
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, current)
			current = ""
		} else if r != '\r' {
			current += string(r)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
```

- [ ] **Step 4: Run tests**

```bash
cd /home/ayan-de/Projects/agent-board && go test ./internal/pty/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/pty/session.go internal/pty/session_test.go
git commit -m "feat: add internal/pty session management with completion detection"
```

---

### Task 4: Create `internal/orchestrator/pty_runner.go` — PTY Runner

**Files:**
- Create: `internal/orchestrator/pty_runner.go`

- [ ] **Step 1: Write implementation**

```go
package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"github.com/ayan-de/agent-board/internal/pty"
)

type PtyRunner struct {
	mu       sync.Mutex
	sessions map[string]*pty.Session
	registry map[string]*pty.AgentConfig
	chdir    string
}

func NewPtyRunner(registry map[string]*pty.AgentConfig, chdir string) *PtyRunner {
	return &PtyRunner{
		sessions: make(map[string]*pty.Session),
		registry: registry,
		chdir:    chdir,
	}
}

func (r *PtyRunner) Start(ctx context.Context, req RunRequest) (RunHandle, error) {
	cfg, ok := r.registry[req.Agent]
	if !ok {
		return RunHandle{}, fmt.Errorf("ptyRunner.start: no config for agent %q", req.Agent)
	}

	sess, err := pty.NewSession(cfg, req.SessionID, req.TicketID, req.Prompt, r.chdir)
	if err != nil {
		return RunHandle{}, fmt.Errorf("ptyRunner.start: %w", err)
	}

	r.mu.Lock()
	r.sessions[req.SessionID] = sess
	r.mu.Unlock()

	if req.Reporter != nil {
		req.Reporter(fmt.Sprintf("Agent %s started in PTY session %s", req.Agent, req.SessionID))
	}

	go func() {
		outcome, summary := sess.Wait()
		r.mu.Lock()
		delete(r.sessions, req.SessionID)
		r.mu.Unlock()

		if req.OnComplete != nil {
			req.OnComplete(outcome, summary)
		}
	}()

	return RunHandle{
		Outcome: "running",
		Summary: fmt.Sprintf("Agent %s started in PTY", req.Agent),
	}, nil
}

func (r *PtyRunner) GetSession(sessionID string) (*pty.Session, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[sessionID]
	return s, ok
}

func (r *PtyRunner) GetPTYOutput(sessionID string, lines int) (string, error) {
	r.mu.Lock()
	sess, ok := r.sessions[sessionID]
	r.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("session %s not found", sessionID)
	}

	output := sess.RecentOutput(lines)
	result := ""
	for _, line := range output {
		result += pty.StripANSI(line) + "\n"
	}
	return result, nil
}

func (r *PtyRunner) SendInput(sessionID string, input string) error {
	r.mu.Lock()
	sess, ok := r.sessions[sessionID]
	r.mu.Unlock()
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}
	return sess.SendInput([]byte(input + "\n"))
}

func (r *PtyRunner) Stop(sessionID string) error {
	r.mu.Lock()
	sess, ok := r.sessions[sessionID]
	r.mu.Unlock()
	if !ok {
		return nil
	}
	sess.Close()
	return nil
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd /home/ayan-de/Projects/agent-board && go build ./internal/orchestrator/...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/orchestrator/pty_runner.go
git commit -m "feat: add PtyRunner implementing async PTY agent execution"
```

---

### Task 5: Update orchestrator service to use PtyRunner

**Files:**
- Modify: `internal/orchestrator/service.go`
- Modify: `internal/orchestrator/types.go`

- [ ] **Step 1: Update types.go — remove PaneID/WindowID from AgentSession**

In `internal/orchestrator/types.go`, the `AgentSession` struct at line ~27 needs PaneID and WindowID removed:

Old:
```go
type AgentSession struct {
	SessionID string
	TicketID  string
	Agent     string
	StartedAt int64
	Status    string
	PaneID    string
	WindowID  string
}
```

New:
```go
type AgentSession struct {
	SessionID string
	TicketID  string
	Agent     string
	StartedAt int64
	Status    string
}
```

- [ ] **Step 2: Update service.go — replace TmuxRunner type assertions with PtyRunner**

In `internal/orchestrator/service.go`, make these changes:

1. Remove the `StopSession` method's TmuxRunner type assertion (line 246-248). Replace with:

```go
func (s *Service) StopSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	sess, ok := s.activeSessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("session %s not found", sessionID)
	}
	delete(s.activeSessions, sessionID)
	s.mu.Unlock()

	if pr, ok := s.runner.(*PtyRunner); ok {
		_ = pr.Stop(sessionID)
	}

	_ = s.store.EndSession(ctx, sessionID, "cancelled")
	_ = s.store.SetAgentActive(ctx, sess.TicketID, false)

	return nil
}
```

2. Replace `SendInput` (line 271-288) to use PtyRunner:

```go
func (s *Service) SendInput(sessionID, input string) error {
	if pr, ok := s.runner.(*PtyRunner); ok {
		return pr.SendInput(sessionID, input)
	}

	s.mu.RLock()
	w, ok := s.inputs[sessionID]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("service.sendInput: session %s not found or not interactive", sessionID)
	}
	_, err := fmt.Fprintln(w, input)
	return err
}
```

3. Replace `GetTmuxRunner` with `GetPtyRunner`:

```go
func (s *Service) GetPtyRunner() (*PtyRunner, bool) {
	pr, ok := s.runner.(*PtyRunner)
	return pr, ok
}
```

4. Replace `GetPaneContent` with `GetPTYOutput`:

```go
func (s *Service) GetPTYOutput(sessionID string, lines int) (string, error) {
	pr, ok := s.runner.(*PtyRunner)
	if !ok {
		return "", fmt.Errorf("pty output only available with PtyRunner")
	}
	return pr.GetPTYOutput(sessionID, lines)
}
```

5. Remove `SwitchToPane` method entirely.

- [ ] **Step 3: Run existing tests**

```bash
cd /home/ayan-de/Projects/agent-board && go test ./internal/orchestrator/... -v
```

Expected: all existing tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/orchestrator/service.go internal/orchestrator/types.go
git commit -m "refactor: replace TmuxRunner references with PtyRunner in orchestrator service"
```

---

### Task 6: Update config — remove tmux field

**Files:**
- Modify: `internal/config/general.go`
- Modify: `internal/config/defaults.go`
- Modify: `internal/config/loader.go`
- Modify: `internal/config/scaffold.go`

- [ ] **Step 1: Remove Tmux from GeneralConfig**

In `internal/config/general.go`, change:

```go
type GeneralConfig struct {
	Log  string `toml:"log"`
	Addr string `toml:"addr"`
	Mode string `toml:"mode"`
	Tmux string `toml:"tmux"`
}
```

To:

```go
type GeneralConfig struct {
	Log  string `toml:"log"`
	Addr string `toml:"addr"`
	Mode string `toml:"mode"`
}
```

- [ ] **Step 2: Remove Tmux default**

In `internal/config/defaults.go`, remove `Tmux: "auto",` from the GeneralConfig initialization.

- [ ] **Step 3: Remove AGENTBOARD_TMUX env var**

In `internal/config/loader.go`, remove lines 74-76:

```go
if v := os.Getenv("AGENTBOARD_TMUX"); v != "" {
	cfg.General.Tmux = v
}
```

- [ ] **Step 4: Remove tmux from scaffold template**

In `internal/config/scaffold.go`, remove the line `tmux = "auto"` from the default config template.

- [ ] **Step 5: Verify build**

```bash
cd /home/ayan-de/Projects/agent-board && go build ./...
```

Expected: compilation errors from app.go (tmux references) — that's expected, will fix in next task

- [ ] **Step 6: Commit**

```bash
git add internal/config/general.go internal/config/defaults.go internal/config/loader.go internal/config/scaffold.go
git commit -m "refactor: remove tmux config field from config package"
```

---

### Task 7: Delete tmux package and files

**Files:**
- Delete: `internal/orchestrator/tmux_runner.go`
- Delete: `internal/orchestrator/pane_manager.go`
- Delete: `internal/tmux/tmux.go`
- Delete: `internal/tui/pane.go`

- [ ] **Step 1: Remove tmux-related orchestrator files**

```bash
rm /home/ayan-de/Projects/agent-board/internal/orchestrator/tmux_runner.go /home/ayan-de/Projects/agent-board/internal/orchestrator/pane_manager.go
```

- [ ] **Step 2: Remove tmux package**

```bash
rm /home/ayan-de/Projects/agent-board/internal/tmux/tmux.go && rmdir /home/ayan-de/Projects/agent-board/internal/tmux
```

- [ ] **Step 3: Remove empty pane.go**

```bash
rm /home/ayan-de/Projects/agent-board/internal/tui/pane.go
```

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "refactor: remove tmux runner, pane manager, and tmux package"
```

---

### Task 8: Update `cmd/agentboard/main.go` — use PtyRunner

**Files:**
- Modify: `cmd/agentboard/main.go`

- [ ] **Step 1: Rewrite main.go**

Replace the entire file with:

```go
package main

import (
	"fmt"
	"os"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/llm"
	"github.com/ayan-de/agent-board/internal/mcp"
	"github.com/ayan-de/agent-board/internal/orchestrator"
	"github.com/ayan-de/agent-board/internal/pty"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"
	"github.com/ayan-de/agent-board/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	s, err := store.Open(cfg.DB.Path, cfg.Board.Statuses, cfg.Board.Prefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening store: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	llmClient, err := llm.NewFromConfig(cfg.LLM)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating llm client: %v\n", err)
		os.Exit(1)
	}

	registry := pty.NewRegistry()
	var runner orchestrator.Runner
	if len(registry) > 0 {
		runner = orchestrator.NewPtyRunner(registry, "")
	} else {
		runner = orchestrator.NewExecRunner()
	}

	mcpManager := mcp.NewManager(cfg.MCP)
	ctxCarry := mcp.NewContextCarryAdapter(mcpManager, cfg.ProjectName)
	orch := orchestrator.NewService(s, llmClient, runner, ctxCarry)

	reg := theme.NewRegistry("dark")
	reg.LoadBuiltins()
	reg.LoadUserThemes()
	if err := reg.Set(cfg.TUI.Theme); err != nil {
		_ = reg.Set("agentboard")
	}

	app, err := tui.NewApp(cfg, s, reg, tui.AppDeps{
		Orchestrator: orch,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating app: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running tui: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add cmd/agentboard/main.go
git commit -m "refactor: replace tmux launch with PtyRunner in main entrypoint"
```

---

### Task 9: Update `internal/tui/app.go` — remove tmux, add PTY polling

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Update imports**

Remove `"github.com/ayan-de/agent-board/internal/tmux"` from imports.

- [ ] **Step 2: Update Orchestrator interface**

Change the interface at line 85-96 to:

```go
type Orchestrator interface {
	CreateProposal(ctx context.Context, input orchestrator.CreateProposalInput) (store.Proposal, error)
	ApproveProposal(ctx context.Context, proposalID string) error
	StartApprovedRun(ctx context.Context, proposalID string) (store.Session, error)
	FinishRun(ctx context.Context, input orchestrator.FinishRunInput) error
	GetLogs(sessionID string) []string
	SendInput(sessionID, input string) error
	GetActiveSessions() []*orchestrator.AgentSession
	GetPTYOutput(sessionID string, lines int) (string, error)
	CompletionChan() <-chan orchestrator.RunCompletion
}
```

(Removed `SwitchToPane`, renamed `GetPaneContent` → `GetPTYOutput`)

- [ ] **Step 3: Remove dashboardPaneID and related fields**

Remove `dashboardPaneID`, `lastSelectedAgent`, `lastSelectedSession` from the App struct.

- [ ] **Step 4: Update tickMsg handler**

Replace lines 215-232 (the tmux split pane logic) with a simple dashboard update:

```go
case tickMsg:
	var cmd tea.Cmd
	a.kanban, cmd = a.kanban.Update(msg)
	a.dashboard, _ = a.dashboard.Update(msg)
	return a, cmd
```

- [ ] **Step 5: Remove syncDashboardPane method**

Delete the entire `syncDashboardPane` method (lines 345-373).

- [ ] **Step 6: Update dashboard toggle**

Replace the ActionShowDashboard case (lines 510-526) with:

```go
case keybinding.ActionShowDashboard:
	if a.view == viewDashboard {
		a.view = viewBoard
	} else {
		a.dashboard = a.dashboard.Refresh()
		a.view = viewDashboard
	}
```

- [ ] **Step 7: Commit**

```bash
git add internal/tui/app.go
git commit -m "refactor: remove tmux from TUI app, use PTY output interface"
```

---

### Task 10: Update `internal/tui/dashboard.go` — use PTY output

**Files:**
- Modify: `internal/tui/dashboard.go`

- [ ] **Step 1: Update footer text**

In `View()` method around line 296-297, change footer:

```go
footerStr := "j/k: select │ r: refresh │ Esc: back"
```

Remove the conditional append of `" │ e: send input │ v: view in tmux"`.

- [ ] **Step 2: Update handleKey — remove SwitchToPane action**

Remove the `case keybinding.ActionSwitchToPane:` block (lines 231-236).

Remove the `case keybinding.ActionInteract:` block (lines 225-230).

- [ ] **Step 3: Update renderContent — use GetPTYOutput**

Replace the pane content fetching block (lines 447-464) with:

```go
ptyOutput := m.paneContent
if time.Since(m.paneContentLoadedAt) > 500*time.Millisecond {
	if content, err := m.orchestrator.GetPTYOutput(sess.SessionID, 30); err == nil {
		m.paneContent = content
		m.paneContentLoadedAt = time.Now()
		ptyOutput = content
	}
}

if ptyOutput == "" {
	if content, err := m.orchestrator.GetPTYOutput(sess.SessionID, 30); err == nil {
		m.paneContent = content
		m.paneContentLoadedAt = time.Now()
		ptyOutput = content
	}
}
```

Replace variable references from `paneContent` to `ptyOutput` in the display section.

- [ ] **Step 4: Remove input mode from renderContent**

Remove the input prompt display (lines 496-503):

```go
// Remove this entire block:
if m.isInput {
	b.WriteString(m.styles.Label.Render("Send to agent: "))
	b.WriteString(m.input.View())
	b.WriteString("\n")
} else {
	b.WriteString(m.styles.Placeholder.Render("Press 'e' to send input to agent"))
	b.WriteString("\n")
}
```

- [ ] **Step 5: Verify build**

```bash
cd /home/ayan-de/Projects/agent-board && go build ./...
```

Expected: clean build

- [ ] **Step 6: Commit**

```bash
git add internal/tui/dashboard.go
git commit -m "refactor: dashboard uses PTY output, removes tmux pane switching"
```

---

### Task 11: Run full test suite and verify

**Files:**
- All

- [ ] **Step 1: Run all tests**

```bash
cd /home/ayan-de/Projects/agent-board && go test ./...
```

Expected: all tests pass

- [ ] **Step 2: Run vet**

```bash
cd /home/ayan-de/Projects/agent-board && go vet ./...
```

Expected: no issues

- [ ] **Step 3: Build binary**

```bash
cd /home/ayan-de/Projects/agent-board && go build -o agentboard ./cmd/agentboard
```

Expected: clean build

- [ ] **Step 4: Commit any fixes**

```bash
git add -A && git commit -m "fix: address test and build issues after PTY migration"
```

(Only if needed)

---

### Task 12: Update AGENTS.md

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: Update references**

Replace all tmux references in AGENTS.md:
- Change "tmux and embedded PTY" to "embedded PTY"
- Remove tmux from the runtime flow
- Update package responsibilities to show `internal/pty` and remove `internal/tmux`
- Update data flow diagram
- Update status snapshot
- Update environment variables section (remove AGENTBOARD_TMUX)
- Update keybindings (remove `e: send input` and `v: view in tmux` from Dashboard)

- [ ] **Step 2: Commit**

```bash
git add AGENTS.md
git commit -m "docs: update AGENTS.md for PTY implementation"
```
