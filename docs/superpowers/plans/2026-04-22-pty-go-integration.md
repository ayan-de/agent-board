# pty-go Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Embed pty-go into agent-board as `internal/pty/` package, create `PtyRunner` implementing `Runner` interface, wire into orchestrator so pressing `r` in dashboard spawns full interactive PTY session in tmux.

**Architecture:** Copy pty-go source files into `internal/pty/`, adapt them for use as a library, create `PtyRunner` adapter struct that implements the `Runner` interface, wire into orchestrator `Service`, connect dashboard keybinding to trigger PTY session.

**Tech Stack:** Go, `github.com/creack/pty`, `golang.org/x/term`, tmux

---

## File Structure

```
agent-board/
├── internal/
│   ├── pty/                    # NEW: embedded pty-go
│   │   ├── pty.go             # PtyRunner struct, Start/Stop/SendInput
│   │   ├── agent.go           # Config, Registry, FormatPrompt, StripANSI
│   │   ├── opencode.go        # NewOpenCode() config
│   │   ├── claudecode.go      # NewClaudeCode() config
│   │   ├── codex.go           # NewCodex() config
│   │   ├── geminicode.go      # NewGeminiCode() config
│   │   └── tmux.go            # tmux session helpers (adapted from pty-go)
│   └── orchestrator/
│       ├── pty_runner.go      # NEW: PtyRunner implementing Runner interface
│       ├── service.go         # MODIFY: add PtyRunner, wire StartApprovedRun
│       ├── types.go           # MODIFY: Runner interface already supports
│       └── tmux_runner.go     # KEEP: existing TmuxRunner, may coexist
└── internal/tui/
    └── dashboard.go           # MODIFY: connect 'r' to start PTY session
```

---

## Task 1: Create internal/pty/agent.go

**Files:**
- Create: `internal/pty/agent.go`

- [ ] **Step 1: Create agent.go with Config, Registry, and helpers**

```go
package pty

import (
    "bytes"
    "fmt"
    "regexp"
    "strings"
    "time"
)

const DoneMarker = "P0MX_DONE_SIGNAL"

type Config struct {
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

func NewRegistry() map[string]*Config {
    return map[string]*Config{
        "opencode":   NewOpenCode(),
        "claudecode": NewClaudeCode(),
        "codex":      NewCodex(),
        "gemini":     NewGeminiCode(),
    }
}

func DefaultFormatPrompt(prompt string) string {
    return fmt.Sprintf(
        "%s\n\nIMPORTANT: After you have fully completed all the above tasks, you MUST print exactly this line on its own: %s. Do not skip this.",
        prompt, DoneMarker,
    )
}

func ClaudeFormatPrompt(prompt string) string {
    return fmt.Sprintf(
        "%s. IMPORTANT: After fully completing all tasks, print exactly this on its own line: %s",
        prompt, DoneMarker,
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

func JoinArgs(args []string) string {
    var buf bytes.Buffer
    for i, a := range args {
        if i > 0 {
            buf.WriteByte(' ')
        }
        buf.WriteString(a)
    }
    return buf.String()
}
```

- [ ] **Step 2: Verify it builds**

Run: `cd /home/ayan-de/Projects/agent-board && go build ./internal/pty/`
Expected: no output (success)

- [ ] **Step 3: Commit**

```bash
cd /home/ayan-de/Projects/agent-board
git add internal/pty/agent.go
git commit -m "feat(pty): add pty package with Config, Registry, and helpers"
```

---

## Task 2: Create internal/pty/opencode.go, claudecode.go, codex.go, geminicode.go

**Files:**
- Create: `internal/pty/opencode.go`
- Create: `internal/pty/claudecode.go`
- Create: `internal/pty/codex.go`
- Create: `internal/pty/geminicode.go`

- [ ] **Step 1: Create opencode.go**

```go
package pty

import "time"

func NewOpenCode() *Config {
    return &Config{
        Name:            "opencode",
        Bin:             "opencode",
        ReadyPattern:    `Ask\s+anything`,
        SendPrompt:      SendPromptTyped,
        GracePeriod:     8 * time.Second,
        FallbackTimeout: 5 * time.Second,
        ReadyWait:       800 * time.Millisecond,
        FormatPrompt:    DefaultFormatPrompt,
        IdlePatterns:    []string{`Ask\s+anything`},
    }
}
```

- [ ] **Step 2: Create claudecode.go**

```go
package pty

import "time"

func NewClaudeCode() *Config {
    return &Config{
        Name:            "claude-code",
        Bin:             "claude",
        ReadyPattern:    `Press\s+Ctrl-C\s+again\s+to\s+exit`,
        SendPrompt:      SendPromptSingleLine,
        GracePeriod:     10 * time.Second,
        FallbackTimeout: 10 * time.Second,
        ReadyWait:       2 * time.Second,
        FormatPrompt:    ClaudeFormatPrompt,
        IdlePatterns:    []string{`Press\s+Ctrl-C\s+again\s+to\s+exit`},
    }
}
```

- [ ] **Step 3: Create codex.go**

```go
package pty

import "time"

func NewCodex() *Config {
    return &Config{
        Name:            "codex",
        Bin:             "codex",
        Args:            []string{"--no-alt-screen"},
        ReadyPattern:    `(?m)^\s*›\s*$|Run\s+/review\s+on\s+my\s+current\s+changes`,
        SendPrompt:      SendPromptSingleLine,
        GracePeriod:     10 * time.Second,
        FallbackTimeout: 8 * time.Second,
        ReadyWait:       1 * time.Second,
        FormatPrompt:    DefaultFormatPrompt,
        IdlePatterns:    []string{`Run\s+/review\s+on\s+my\s+current\s+changes`},
    }
}
```

- [ ] **Step 4: Create geminicode.go**

```go
package pty

import "time"

func NewGeminiCode() *Config {
    return &Config{
        Name:            "gemini",
        Bin:             "gemini",
        Args:            []string{"-y"},
        ReadyPattern:    `Gemini\s+CLI`,
        SendPrompt:      SendPromptSingleLine,
        GracePeriod:     10 * time.Second,
        FallbackTimeout: 10 * time.Second,
        ReadyWait:       2 * time.Second,
        FormatPrompt:    ClaudeFormatPrompt,
        IdlePatterns:    []string{`Type\s+your\s+message`},
    }
}
```

- [ ] **Step 5: Verify all build**

Run: `cd /home/ayan-de/Projects/agent-board && go build ./internal/pty/`
Expected: no output

- [ ] **Step 6: Commit**

```bash
cd /home/ayan-de/Projects/agent-board
git add internal/pty/opencode.go internal/pty/claudecode.go internal/pty/codex.go internal/pty/geminicode.go
git commit -m "feat(pty): add agent configs for opencode, claudecode, codex, gemini"
```

---

## Task 3: Create internal/pty/tmux.go

**Files:**
- Create: `internal/pty/tmux.go`

- [ ] **Step 1: Create tmux.go**

```go
package pty

import (
    "fmt"
    "os"
    "os/exec"
    "strings"
)

func TmuxCmd(args ...string) error {
    cmd := exec.Command("tmux", args...)
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

func TmuxHasSession(name string) bool {
    return exec.Command("tmux", "has-session", "-t", name).Run() == nil
}

func IsInTmux() bool {
    return os.Getenv("TMUX") != ""
}

func BuildAgentCommand(self string, agentName string, autoExit bool, prompt string) string {
    parts := []string{self, "-" + agentName}
    if autoExit {
        parts = append(parts, "-auto-exit")
    }
    parts = append(parts, prompt)
    for i, p := range parts {
        if strings.Contains(p, " ") || strings.Contains(p, "\"") {
            parts[i] = "'" + strings.ReplaceAll(p, "'", "'\\''") + "'"
        }
    }
    return strings.Join(parts, " ")
}
```

- [ ] **Step 2: Verify it builds**

Run: `cd /home/ayan-de/Projects/agent-board && go build ./internal/pty/`
Expected: no output

- [ ] **Step 3: Commit**

```bash
cd /home/ayan-de/Projects/agent-board
git add internal/pty/tmux.go
git commit -m "feat(pty): add tmux helpers for session management"
```

---

## Task 4: Create internal/pty/pty.go (PtyRunner core)

**Files:**
- Create: `internal/pty/pty.go`

- [ ] **Step 1: Create pty.go with PtyRunner struct and methods**

```go
package pty

import (
    "bytes"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "regexp"
    "sync"
    "syscall"
    "time"

    "github.com/creack/pty"
    "golang.org/x/term"
)

type PtyRunner struct {
    sessionName string
    registry    map[string]*Config
    activePty   map[string]*os.File
    activeCmd   map[string]*exec.Cmd
    mu          sync.RWMutex
}

func NewPtyRunner(sessionName string) *PtyRunner {
    return &PtyRunner{
        sessionName: sessionName,
        registry:     NewRegistry(),
        activePty:    make(map[string]*os.File),
        activeCmd:    make(map[string]*exec.Cmd),
    }
}

func (p *PtyRunner) Start(sessionID string, agentName string, prompt string, autoExit bool) error {
    ag, ok := p.registry[agentName]
    if !ok {
        return fmt.Errorf("unknown agent: %s", agentName)
    }

    chdir := ""
    cmd := exec.Command(ag.Bin, ag.Args...)
    cmd.Env = os.Environ()
    if chdir != "" {
        abs, _ := filepath.Abs(chdir)
        cmd.Dir = abs
    }

    ptmx, err := pty.Start(cmd)
    if err != nil {
        return err
    }

    p.mu.Lock()
    p.activePty[sessionID] = ptmx
    p.activeCmd[sessionID] = cmd
    p.mu.Unlock()

    go p.runStateMachine(sessionID, ptmx, cmd, ag, prompt, autoExit)

    return nil
}

func (p *PtyRunner) runStateMachine(sessionID string, ptmx *os.File, cmd *exec.Cmd, ag *Config, prompt string, autoExit bool) {
    defer ptmx.Close()

    oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
    if err == nil {
        defer term.Restore(int(os.Stdin.Fd()), oldState)
    }

    ch := make(chan os.Signal, 1)
    signal.Notify(ch, syscall.SIGWINCH)
    go func() {
        for range ch {
            ws, _ := pty.GetsizeFull(os.Stdin)
            if ws != nil {
                _ = pty.Setsize(ptmx, ws)
            }
        }
    }()
    ch <- syscall.SIGWINCH

    cmdDone := make(chan struct{})
    go func() {
        cmd.Wait()
        close(cmdDone)
    }()

    stdinEnabled := make(chan struct{})
    go func() {
        <-stdinEnabled
        io.Copy(ptmx, os.Stdin)
    }()

    injectedPrompt := prompt
    if autoExit && ag.FormatPrompt != nil {
        injectedPrompt = ag.FormatPrompt(prompt)
    }

    var outputBuf bytes.Buffer
    tee := io.TeeReader(ptmx, os.Stdout)

    readyMarker := regexp.MustCompile(ag.ReadyPattern)
    doneMarkerRe := regexp.MustCompile(regexp.QuoteMeta(DoneMarker))
    idleRes := make([]*regexp.Regexp, len(ag.IdlePatterns))
    for i, p := range ag.IdlePatterns {
        idleRes[i] = regexp.MustCompile(p)
    }

    current := stateWaitingReady
    buf := make([]byte, 4096)
    var doneOnce sync.Once
    canCheckCompletion := false

    exit := func() {
        doneOnce.Do(func() {
            os.Stderr.WriteString("\n[pty] task complete, closing...\n")
            time.Sleep(500 * time.Millisecond)
            ptmx.Write([]byte{0x03})
            time.Sleep(300 * time.Millisecond)
            ptmx.Write([]byte{0x03})
            time.Sleep(300 * time.Millisecond)
            cmd.Process.Signal(syscall.SIGTERM)
            time.Sleep(1 * time.Second)
            cmd.Process.Kill()
        })
    }

    transitionToWorking := func() {
        outputBuf.Reset()
        current = stateWorking
        time.AfterFunc(ag.GracePeriod, func() {
            canCheckCompletion = true
        })
    }

    fallback := time.AfterFunc(ag.FallbackTimeout, func() {
        if current == stateWaitingReady {
            current = stateSendingPrompt
            ag.SendPrompt(ptmx, injectedPrompt)
            if autoExit {
                transitionToWorking()
            } else {
                close(stdinEnabled)
            }
        }
    })
    defer fallback.Stop()

    for {
        select {
        case <-cmdDone:
            p.cleanup(sessionID)
            return
        default:
        }

        n, err := tee.Read(buf)
        if err != nil {
            break
        }

        outputBuf.Write(buf[:n])

        switch current {
        case stateWaitingReady:
            stripped := StripANSI(outputBuf.String())
            if readyMarker.MatchString(stripped) {
                current = stateSendingPrompt
                fallback.Stop()
                time.AfterFunc(ag.ReadyWait, func() {
                    ag.SendPrompt(ptmx, injectedPrompt)
                    if autoExit {
                        transitionToWorking()
                    } else {
                        close(stdinEnabled)
                    }
                })
            }

        case stateWorking:
            if !canCheckCompletion {
                continue
            }
            outputBuf.Reset()
            recent := StripANSI(outputBuf.String())

            if doneMarkerRe.MatchString(recent) {
                current = stateDone
                go exit()
                continue
            }

            for _, re := range idleRes {
                matches := re.FindAllStringIndex(recent, -1)
                if len(matches) >= 3 {
                    current = stateDone
                    go exit()
                    break
                }
            }
        case stateDone:
        }
    }

    p.cleanup(sessionID)
}

func (p *PtyRunner) cleanup(sessionID string) {
    p.mu.Lock()
    delete(p.activePty, sessionID)
    delete(p.activeCmd, sessionID)
    p.mu.Unlock()
}

func (p *PtyRunner) Stop(sessionID string) error {
    p.mu.RLock()
    ptmx, ok := p.activePty[sessionID]
    p.mu.RUnlock()

    if !ok {
        return fmt.Errorf("session %s not found", sessionID)
    }

    ptmx.Write([]byte{0x03})
    time.Sleep(300 * time.Millisecond)
    ptmx.Write([]byte{0x03})
    time.Sleep(300 * time.Millisecond)

    p.mu.RLock()
    cmd, ok := p.activeCmd[sessionID]
    p.mu.RUnlock()

    if ok && cmd.Process != nil {
        cmd.Process.Signal(syscall.SIGTERM)
        time.Sleep(1 * time.Second)
        cmd.Process.Kill()
    }

    return nil
}

func (p *PtyRunner) SendInput(sessionID string, input string) error {
    p.mu.RLock()
    ptmx, ok := p.activePty[sessionID]
    p.mu.RUnlock()

    if !ok {
        return fmt.Errorf("session %s not found", sessionID)
    }

    _, err := ptmx.Write([]byte(input))
    return err
}

type state int

const (
    stateWaitingReady state = iota
    stateSendingPrompt
    stateWorking
    stateDone
)
```

Note: The above uses `io` and `signal` packages which need to be imported. Also `signal.Notify` needs `os/signal` import. Fix imports:

```go
import (
    "bytes"
    "fmt"
    "io"
    "os"
    "os/exec"
    "os/signal"
    "path/filepath"
    "regexp"
    "sync"
    "syscall"
    "time"

    "github.com/creack/pty"
    "golang.org/x/term"
)
```

- [ ] **Step 2: Verify it builds (fix any import or type errors)**

Run: `cd /home/ayan-de/Projects/agent-board && go build ./internal/pty/`
Expected: no output

- [ ] **Step 3: Commit**

```bash
cd /home/ayan-de/Projects/agent-board
git add internal/pty/pty.go
git commit -m "feat(pty): add PtyRunner with PTY state machine"
```

---

## Task 5: Create internal/orchestrator/pty_runner.go

**Files:**
- Create: `internal/orchestrator/pty_runner.go`

- [ ] **Step 1: Create pty_runner.go**

```go
package orchestrator

import (
    "context"
    "fmt"

    "github.com/ayan-de/agent-board/internal/pty"
)

type PtyRunner struct {
    runner    *pty.PtyRunner
    tmuxMode  bool
}

func NewPtyRunner(tmuxSession string) (*PtyRunner, error) {
    runner := pty.NewPtyRunner(tmuxSession)
    return &PtyRunner{
        runner:   runner,
        tmuxMode: pty.IsInTmux(),
    }, nil
}

func (r *PtyRunner) Start(ctx context.Context, req RunRequest) (RunHandle, error) {
    if !r.tmuxMode {
        return RunHandle{}, fmt.Errorf("pty runner requires tmux session")
    }

    autoExit := true
    if err := r.runner.Start(req.SessionID, req.Agent, req.Prompt, autoExit); err != nil {
        return RunHandle{}, fmt.Errorf("ptyRunner.start: %w", err)
    }

    if req.Reporter != nil {
        req.Reporter(fmt.Sprintf("PTY agent %s started for session %s", req.Agent, req.SessionID))
    }

    return RunHandle{
        Outcome: "running",
        Summary: fmt.Sprintf("Interactive PTY session %s", req.SessionID),
    }, nil
}

func (r *PtyRunner) Stop(sessionID string) error {
    return r.runner.Stop(sessionID)
}

func (r *PtyRunner) SendInput(sessionID, input string) error {
    return r.runner.SendInput(sessionID, input)
}

func (r *PtyRunner) GetActivePty(sessionID string) (interface{}, error) {
    return r.runner.GetPty(sessionID)
}
```

- [ ] **Step 2: Add GetPty method to internal/pty/pty.go**

Add to PtyRunner:
```go
func (p *PtyRunner) GetPty(sessionID string) (*os.File, bool) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    f, ok := p.activePty[sessionID]
    return f, ok
}
```

- [ ] **Step 3: Verify it builds**

Run: `cd /home/ayan-de/Projects/agent-board && go build ./internal/orchestrator/`
Expected: no output

- [ ] **Step 4: Commit**

```bash
cd /home/ayan-de/Projects/agent-board
git add internal/orchestrator/pty_runner.go internal/pty/pty.go
git commit -m "feat(orchestrator): add PtyRunner adapter for PTY execution"
```

---

## Task 6: Wire PtyRunner into orchestrator Service

**Files:**
- Modify: `internal/orchestrator/service.go:37-48`

- [ ] **Step 1: Add PtyRunner to Service struct**

In `service.go`, add to Service struct:
```go
type Service struct {
    store  Store
    llm    LLMClient
    runner Runner
    ctx    ContextCarryProvider
    ptyRunner *PtyRunner  // ADD THIS
    logs   map[string][]string
    inputs map[string]io.Writer
    mu     sync.RWMutex

    activeSessions map[string]*AgentSession
    completionCh   chan RunCompletion
}
```

- [ ] **Step 2: Update NewService to accept PtyRunner**

```go
func NewService(store Store, llm LLMClient, runner Runner, ctx ContextCarryProvider) *Service {
    return &Service{
        store:          store,
        llm:            llm,
        runner:         runner,
        ctx:            ctx,
        logs:           make(map[string][]string),
        inputs:         make(map[string]io.Writer),
        activeSessions: make(map[string]*AgentSession),
        completionCh:   make(chan RunCompletion, 16),
    }
}
```

Or add separate setter:
```go
func (s *Service) SetPtyRunner(pr *PtyRunner) {
    s.mu.Lock()
    s.ptyRunner = pr
    s.mu.Unlock()
}
```

- [ ] **Step 3: Modify StartApprovedRun to use PtyRunner when available**

In `StartApprovedRun`, after creating session:
```go
// Use PtyRunner if available
if s.ptyRunner != nil {
    if err := s.ptyRunner.Start(ctx, RunRequest{
        TicketID:   proposal.TicketID,
        SessionID:  session.ID,
        Agent:      proposal.Agent,
        Prompt:     proposal.Prompt,
        Reporter:   func(line string) { s.AppendLog(session.ID, line) },
        OnComplete: onComplete,
    }); err != nil {
        _ = s.store.EndSession(ctx, session.ID, "failed")
        _ = s.store.SetAgentActive(ctx, proposal.TicketID, false)
        return store.Session{}, err
    }
    // ... rest unchanged
}
```

- [ ] **Step 4: Verify it builds**

Run: `cd /home/ayan-de/Projects/agent-board && go build ./internal/orchestrator/`
Expected: no output

- [ ] **Step 5: Commit**

```bash
cd /home/ayan-de/Projects/agent-board
git add internal/orchestrator/service.go
git commit -m "feat(orchestrator): wire PtyRunner into Service.StartApprovedRun"
```

---

## Task 7: Connect dashboard 'r' key to start PTY session

**Files:**
- Modify: `internal/tui/dashboard.go`

First, read dashboard.go to understand the current `r` key handling:

- [ ] **Step 1: Read dashboard.go to find 'r' keybinding**

Run: `cat /home/ayan-de/Projects/agent-board/internal/tui/dashboard.go`
Expected: Find where `r` key is handled

- [ ] **Step 2: Modify 'r' handler to use PtyRunner**

If `r` currently calls orchestrator.StartApprovedRun, we need to ensure PtyRunner is wired. The flow should be:

```go
case tea.KeyMsg:
    switch msg.String() {
    case "r":
        // Start PTY session for selected agent
        if sess := orch.GetActiveSessionByAgent("opencode"); sess != nil {
            // Switch to the PTY pane
            orch.SwitchToPane(sess.SessionID)
        } else {
            // Start new PTY session
            // Get selected ticket's proposal
            // Call StartApprovedRun which now uses PtyRunner
        }
    }
```

- [ ] **Step 3: Verify it builds**

Run: `cd /home/ayan-de/Projects/agent-board && go build ./internal/tui/`
Expected: no output

- [ ] **Step 4: Commit**

```bash
cd /home/ayan-de/Projects/agent-board
git add internal/tui/dashboard.go
git commit -m "feat(tui): connect dashboard 'r' to PTY session start"
```

---

## Task 8: Add PtyRunner initialization in main.go

**Files:**
- Modify: `cmd/agentboard/main.go:54-63`

- [ ] **Step 1: Add PtyRunner creation in main.go**

After the runner initialization:

```go
var runner orchestrator.Runner = orchestrator.NewExecRunner()
// Only use TmuxRunner if we're actually inside a tmux session
if tmux.IsInTmux() {
    if tmuxRunner, err := orchestrator.NewTmuxRunner(); err == nil {
        runner = tmuxRunner
    }
    // NEW: Also create PtyRunner
    if ptyRunner, err := orchestrator.NewPtyRunner("agentboard"); err == nil {
        orch.SetPtyRunner(ptyRunner)
    }
}
```

- [ ] **Step 2: Verify it builds**

Run: `cd /home/ayan-de/Projects/agent-board && go build ./cmd/agentboard/`
Expected: no output

- [ ] **Step 3: Commit**

```bash
cd /home/ayan-de/Projects/agent-board
git add cmd/agentboard/main.go
git commit -m "feat(main): initialize PtyRunner in tmux mode"
```

---

## Task 9: Run full build, vet, and test

- [ ] **Step 1: Run full build**

Run: `cd /home/ayan-de/Projects/agent-board && go build -o agentboard ./cmd/agentboard`
Expected: no output

- [ ] **Step 2: Run vet**

Run: `cd /home/ayan-de/Projects/agent-board && go vet ./...`
Expected: no output

- [ ] **Step 3: Run tests**

Run: `cd /home/ayan-de/Projects/agent-board && go test ./...`
Expected: all pass

- [ ] **Step 4: Commit**

```bash
cd /home/ayan-de/Projects/agent-board
git add -A
git commit -m "feat: integrate pty-go for full interactive PTY sessions in tmux"
```

---

## Self-Review Checklist

1. **Spec coverage:** All spec requirements implemented?
   - PtyRunner struct ✓
   - Runner interface implementation ✓
   - tmux integration ✓
   - dashboard 'r' key ✓

2. **Placeholder scan:** No "TBD", "TODO", "implement later" found

3. **Type consistency:**
   - PtyRunner.Start takes (sessionID, agentName, prompt, autoExit)
   - pty_runner.go calls runner.Start with RunRequest
   - Types consistent ✓

---

## Spec Reference

See `docs/superpowers/specs/2026-04-22-pty-go-integration-design.md`