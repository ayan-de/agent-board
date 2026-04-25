# Embedding pty-go into agent-board Design

Date: 2026-04-22
Status: approved

## Context

agent-board is a Bubble Tea TUI for AI agent orchestration. Currently it supports running agents via `ExecRunner` (non-interactive, JSON output) and `TmuxRunner` (spawns agents in new tmux windows using `opencode run` command). pty-go is a standalone Go tool that provides full pseudo-terminal (PTY) support for interactive CLI agents.

The user wants to press `r` in the tmux session and get a **full interactive PTY session** — where the agent (opencode) runs with its complete interactive UI inside the tmux pane, including the ability to type, copy/paste, and see real-time terminal output as if running opencode directly in a terminal.

## Goals

1. Press `r` in agent-board → spawn full interactive opencode session in tmux pane
2. User gets complete PTY experience (interactive UI, copy/paste, real-time output)
3. Agent-board can still monitor and manage the session

## Approach

### Option A: Embed pty-go as package (RECOMMENDED)

Copy pty-go source files into `internal/pty/` in agent-board, create a `PtyRunner` struct that wraps pty-go's core logic and implements the `Runner` interface.

**Pros:**
- Single binary deployment
- Full control over PTY lifecycle
- No subprocess overhead
- Can extend/modify PTY handling for agent-board needs

**Cons:**
- Duplicates code between two repos
- Need to manually sync changes

### Option B: Call pty-go as subprocess

Keep pty-go standalone and call it as a subprocess from agent-board, parsing its output.

**Pros:**
- Zero code duplication
- pty-go stays independent

**Cons:**
- Complex IPC
- Loss of direct control over PTY lifecycle
- Harder to integrate session management

### Option C: Import pty-go as Go module

Publish pty-go as a proper Go module and import it.

**Pros:**
- Proper dependency management

**Cons:**
- Requires publishing infrastructure
- Still couples the projects

## Selected Approach: Option A — Embed pty-go as package

Embed pty-go files into `internal/pty/` and create a `PtyRunner` adapter.

## Architecture

```
agent-board/
├── internal/
│   ├── pty/                    # NEW: embedded pty-go
│   │   ├── pty.go             # Core PTY runner
│   │   ├── agent.go           # Agent config + helpers (from pty-go)
│   │   ├── opencode.go        # opencode config (from pty-go)
│   │   ├── claudecode.go      # claudecode config (from pty-go)
│   │   ├── codex.go           # codex config (from pty-go)
│   │   ├── geminicode.go      # gemini config (from pty-go)
│   │   └── tmux.go            # tmux session support (from pty-go)
│   └── orchestrator/
│       └── pty_runner.go      # NEW: PtyRunner implementing Runner interface
```

### PtyRunner

```go
type PtyRunner struct {
    tmuxSession string  // tmux session name for PTY panes
    agentConfig map[string]*AgentConfig
    activePtys  map[string]*os.File  // sessionID -> ptmx file
}
```

Implements `Runner.Start(ctx, RunRequest) (RunHandle, error)`:
1. Create tmux pane (or use existing session)
2. Start PTY process with selected agent binary
3. Wait for ready pattern
4. Inject prompt via PTY
5. Return RunHandle (non-blocking for interactive mode)

### Session Management

- `PtyRunner.StopPane(sessionID)` — send Ctrl+C, then SIGTERM, then kill
- `PtyRunner.SendInput(sessionID, input)` — write directly to PTY master
- `PtyRunner.CapturePane(sessionID, lines)` — tmux capture-pane
- `PtyRunner.ListPanes()` — list active PTY sessions

### Orchestrator Integration

The `Service` already has `GetTmuxRunner()` returning `*TmuxRunner`. The `PtyRunner` should be similarly accessible. The TUI dashboard view can call `SwitchToPane(sessionID)` to let user jump into the interactive PTY session.

## Key Implementation Details

### PTY Lifecycle

```
1. Create tmux pane (new-window or split)
2. fork/exec agent binary with ptmx as controlling terminal
3. Wait for ready pattern (regex match on PTY output)
4. Send prompt via PTY (typed or single-line depending on agent)
5. For auto-exit: monitor for done marker or idle patterns
6. For interactive: leave PTY connected, let user interact
```

### Prompt Injection

- opencode: use `SendPromptTyped` (Ctrl+U/W clearing, char-by-char)
- claudecode/codex/gemini: use `SendPromptSingleLine` (single-line paste)

### State Machine (per pty-go)

```
stateWaitingReady → stateSendingPrompt → stateWorking → stateDone
```

For interactive mode, we skip `stateDone` monitoring and let user control when to exit.

### tmux Integration

When running inside an existing tmux session (`TMUX` env var set):
- Create new window for each agent: `tmux new-window -n <ticket-id> -d`
- Run PTY agent in that window
- User can switch between windows to interact with different agents

When not in tmux:
- Use `tmux new-session -A -s agentboard` to re-launch agent-board in tmux
- Then create panes for agents

## File Structure (internal/pty/)

```
internal/pty/
├── pty.go           # PtyRunner struct, Start/Stop methods
├── agent.go         # Config struct, Registry, prompt helpers
├── opencode.go      # opencode AgentConfig
├── claudecode.go    # claudecode AgentConfig
├── codex.go         # codex AgentConfig
├── geminicode.go    # geminicode AgentConfig
└── tmux.go          # tmux session helpers
```

## Interaction Flow

1. User opens agent-board (may already be in tmux)
2. User selects a ticket with approved proposal
3. User presses `r` to start the agent run
4. orchestrator creates a session, calls `PtyRunner.Start()`
5. PtyRunner creates tmux window, starts PTY with agent binary
6. User switches to that tmux window to interact with agent
7. When agent prints `P0MX_DONE_SIGNAL` (or idle pattern), auto-exit triggers
8. Session ends, board updates with outcome

## Testing Strategy

1. Unit test `PtyRunner.Start()` with mock PTY
2. Integration test with real tmux session
3. Test prompt injection for each agent type
4. Test session cleanup (Ctrl+C → SIGTERM → kill)

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| PTY ownership / zombie processes | Careful cleanup: Ctrl+C → sleep → SIGTERM → sleep → SIGKILL |
| tmux session state | Check `TMUX` env var, fallback to creating new session |
| Agent ready pattern false positives | Use fallback timeout, don't rely solely on pattern match |
| Copy/paste with special characters | Single-line paste for most agents, proper escaping |

## Follow-up

- Implement `internal/pty/pty_runner.go` with PtyRunner struct
- Wire PtyRunner into orchestrator service
- Connect dashboard `r` key to start PTY session
- Add session list to dashboard for quick switching