# AGENT.md — AgentBoard

## Project Overview

AgentBoard is a terminal-based Kanban board that orchestrates AI coding agents. It manages development tickets, spawns AI agents (Claude Code, OpenCode, Cursor) as tmux panes or custom in-Go PTY panes, decomposes projects into tickets via LLM APIs, and exposes a local HTTP/WebSocket API for a future Next.js frontend.

### Architecture Summary

```
┌─────────────────────────────────────────────────────┐
│                      TUI (bubbletea)                │
│  Kanban View │ Ticket Detail │ Agent Panes │ Help    │
├─────────────────────────────────────────────────────┤
│                   Orchestrator                       │
│  tmux Session Manager │ Agent Spawner │ PTY Capture  │
├─────────────────────────────────────────────────────┤
│  Decomposition │ MCP Client │ API Server │ Storage   │
│  (LLM tickets) │ (ContextCarry, │ (chi + WS) │ (SQLite)│
│                │  SessionCarry) │             │         │
├─────────────────────────────────────────────────────────┤
│  API Types     │ MCP Client  │                            │
│  (DTOs)        │ (reusable)  │                            │
├─────────────────────────────────────────────────────┤
│                    Config                            │
│  env vars │ ~/.agentboard/config.toml │ agent detect │
└─────────────────────────────────────────────────────┘
```

Data flows down: TUI renders state from the orchestrator, which coordinates agents, storage, and MCP connections. The API server mirrors TUI actions for remote control.

---

## Folder Structure

```
agent-board/
├── cmd/agentboard/         # Entrypoint
│   └── main.go             # Wire dependencies, start TUI or API mode
├── internal/
│   ├── tui/                # Bubble Tea TUI layer
│   │   ├── app.go          # Root bubbletea.Model, window management
│   │   ├── kanban.go       # Kanban board rendering and columns
│   │   ├── keybindings.go  # Key mapping and handler dispatch
│   │   ├── ticketview.go   # Ticket detail/edit panel
│   │   └── pane.go         # Embedded agent pane (PTY-in-a-widget)
│   ├── orchestrator/       # Session and agent lifecycle
│   │   ├── session.go      # tmux session creation, layout management
│   │   ├── agent.go        # Agent representation and state tracking
│   │   ├── spawner.go      # Agent process spawning (tmux pane or PTY)
│   │   └── pty.go          # PTY allocation and I/O capture
│   ├── mcp/                # MCP server integrations
│   │   ├── client.go       # Shared MCP client bootstrap and registry
│   │   ├── contextcarry.go # ContextCarry MCP server integration
│   │   └── sessioncarry.go # SessionCarry MCP server integration
│   ├── api/                # HTTP + WebSocket API server
│   │   ├── server.go       # chi router setup, server lifecycle
│   │   ├── handlers.go     # REST handlers (tickets, sessions, agents)
│   │   ├── websocket.go    # WebSocket hub and real-time event streaming
│   │   └── middleware.go   # CORS, logging, recovery middleware
│   ├── store/              # SQLite persistence
│   │   ├── sqlite.go       # DB connection, initialization, migrations
│   │   ├── tickets.go      # Ticket CRUD operations
│   │   ├── sessions.go     # Session and agent state persistence
│   │   └── migrations.go   # Schema migration definitions
│   ├── decomposition/      # LLM-powered project breakdown
│   │   ├── decomposer.go   # Project → tickets decomposition engine
│   │   ├── assigner.go     # Auto-assign tickets to agents with reasoning
│   │   └── prompts.go      # Prompt templates for decomposition and assignment
│   ├── apitypes/           # Shared API types (DTOs)
│   │   ├── ticket.go       # Ticket DTOs
│   │   ├── session.go      # Session DTOs
│   │   └── agent.go        # Agent DTOs and status enums
│   ├── mcpclient/          # Reusable MCP client wrapper
│   │   └── client.go       # Generic MCP client for future external consumers
│   └── config/             # Configuration and environment
│       ├── config.go       # Config struct, TOML loading, env var overlay
│       ├── detection.go    # Auto-detect available agents on $PATH
│       └── defaults.go     # Default values and config scaffolding
├── go.mod
├── go.sum
└── AGENT.md                # This file — project memory
```

---

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **internal/ for all private packages** | Enforces encapsulation; no external consumer can depend on internals. |
| **Everything in internal/ until needed externally** | `apitypes` and `mcpclient` stay in `internal/` — will graduate to `pkg/` when the Next.js frontend becomes a real external consumer. No premature abstraction. |
| **Interfaces defined where used** | `orchestrator` defines `AgentSpawner`; `internal/tui` defines `Renderer`. Implementations live in their own package. Prevents circular imports. |
| **modernc.org/sqlite over CGO sqlite** | Pure Go, compiles on Termux (Android ARM64) without a C toolchain. |
| **chi over stdlib mux** | Lightweight, idiomatic, compatible with `net/http` handlers. Easy middleware stacking. |
| **bubbletea + custom pane over pure tmux** | Toggle between tmux-managed panes (for existing tmux users) and embedded PTY panes (for standalone usage). User chooses at startup. |
| **TOML config + env vars** | TOML for persistent user preferences; env vars for CI/overrides. Env vars take precedence. |
| **Error wrapping with `fmt.Errorf("context: %w", err)`** | Consistent error chain for debugging. Every package wraps errors at boundary points. |
| **TDD — tests first, always** | Every feature starts with a failing test. No implementation code without a corresponding `_test.go`. This is non-negotiable. |
| **AGENT.md as project memory** | Single source of truth for architecture, conventions, and onboarding. Updated as the project evolves. |
| **MCP via mark3labs/mcp-go** | Type-safe Go MCP client. Used to connect to ContextCarry and SessionCarry npm servers via stdio transport. |

---

## Build and Run

```bash
# Build
go build -o agentboard ./cmd/agentboard

# Run (TUI mode, default)
./agentboard

# Run (API-only mode, for Next.js frontend)
./agentboard --api --addr :8080

# Initialize config
./agentboard init

# Run tests
go test ./...

# Run with verbose logging
AGENTBOARD_LOG=debug ./agentboard
```

### Dependencies

```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/bubbles
go get github.com/go-chi/chi/v5
go get github.com/mark3labs/mcp-go
go get modernc.org/sqlite
go get github.com/BurntSushi/toml
```

---

## Core Data Model

### Ticket

```
id          TEXT PRIMARY KEY  (e.g. AGT-03)
title       TEXT
description TEXT
status      TEXT  (backlog | in_progress | review | done)
agent       TEXT  (claude-code | opencode | cursor | null)
branch      TEXT
created_at  DATETIME
updated_at  DATETIME
depends_on  TEXT  (comma-separated ticket ids)
```

### Session

```
id          TEXT PRIMARY KEY
ticket_id   TEXT FK → tickets.id
agent       TEXT
started_at  DATETIME
ended_at    DATETIME
status      TEXT  (running | completed | failed | cancelled)
context_key TEXT  (ContextCarry reference)
```

---

## tmux Session Layout

When running in tmux mode, AgentBoard creates a session named `agentboard`:

```
┌──────────────────────────────────────────────────┐
│  AgentBoard TUI (bubbletea)                      │
│  ┌─────────┬──────────┬──────────┬──────────┐    │
│  │ Backlog │  In Prog │  Review  │   Done   │    │
│  │ ─────── │ ──────── │ ──────── │ ──────── │    │
│  │ AUTH-1  │ API-3    │ UI-7     │ INIT-1   │    │
│  │ DB-2    │          │          │ INIT-2   │    │
│  └─────────┴──────────┴──────────┴──────────┘    │
├──────────────────────────────────────────────────┤
│  Agent: claude-code (API-3)                      │
│  $ claude --agent ...                            │
│  > Processing ticket API-3...                    │
├──────────────────────────────────────────────────┤
│  Agent: opencode (UI-7)                          │
│  $ opencode                                      │
│  > Working on UI-7...                            │
└──────────────────────────────────────────────────┘
```

- **Top pane**: AgentBoard TUI (kanban view)
- **Bottom panes**: One per active agent, auto-created when an agent starts a ticket
- Pane layout managed by `internal/orchestrator/session.go` using tmux commands
- Agent panes are destroyed when their ticket moves to Done or is cancelled
- Pane split: horizontal splits for agents, top pane takes 60% height

---

## Agent Spawning

Agents are spawned by `internal/orchestrator/spawner.go`:

1. **Detection** (`internal/config/detection.go`): At startup, scan `$PATH` for `claude`, `opencode`, `cursor`. Store available agents.
2. **Spawning**: When a ticket is assigned to an agent:
   - **tmux mode**: Create a new pane via `tmux split-window`, run the agent CLI with appropriate flags
   - **Embedded mode**: Allocate a PTY, start the agent process, render output in a bubbletea component (`internal/tui/pane.go`)
3. **Monitoring**: Capture agent stdout/stderr. Parse status signals. Update ticket state when agent reports completion.
4. **Lifecycle**: Agents are started, paused (via SIGTSTP), resumed (SIGCONT), or killed. State persisted in SQLite.

### Agent CLI Commands

| Agent | Spawn Command |
|-------|---------------|
| Claude Code | `claude "ticket context here"` |
| OpenCode | `opencode` then `tmux send-keys -t {pane} "ticket context here" Enter` |
| Cursor | Connected via MCP (no direct CLI spawn) |

**Note**: OpenCode is interactive-first — it has no prompt flag. Spawn the process, then inject context via `tmux send-keys` (tmux mode) or PTY write (embedded mode).

---

## MCP Integration

### ContextCarry

- **Package**: `contextcarry` npm package
- **Purpose**: Persists AI session context across conversations. AgentBoard uses it to restore agent memory when reassigning tickets.
- **Integration**: `internal/mcp/contextcarry.go` starts ContextCarry as a stdio subprocess, connects via mcp-go client.
- **Usage**: Before spawning an agent on a ticket, load prior context from ContextCarry. After agent finishes, save context.

### SessionCarry

- **Package**: `sessioncarry` npm package
- **Purpose**: Manages session state across multiple AI agent instances.
- **Integration**: `internal/mcp/sessioncarry.go` mirrors ContextCarry pattern.
- **Usage**: Track which agent worked on which ticket, store cross-session learnings.

### MCP Client Lifecycle

```
1. Detect npm + node on $PATH
2. npx contextcarry serve (stdio transport)
3. Connect mcp-go client
4. Call tools: save_context, load_context, list_sessions
5. Graceful shutdown on exit
```

---

## Keybinding Reference

| Key | Action |
|-----|--------|
| `h/l` or `←/→` | Move between Kanban columns |
| `j/k` or `↑/↓` | Move between tickets in column |
| `Enter` | Open ticket detail view |
| `a` | Add new ticket |
| `d` | Delete ticket (with confirmation) |
| `s` | Start agent on selected ticket |
| `x` | Stop running agent |
| `r` | Refresh board state |
| `Tab` | Toggle focus between board and agent pane |
| `1-4` | Jump to column (Backlog, In Progress, Review, Done) |
| `?` | Show help overlay |
| `i` | Show agent dashboard overlay |
| `q` | Quit |
| `Ctrl+C` | Force quit |

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENTBOARD_CONFIG` | `~/.agentboard/config.toml` | Path to config file |
| `AGENTBOARD_DB` | `~/.agentboard/board.db` | SQLite database path |
| `AGENTBOARD_LOG` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `AGENTBOARD_ADDR` | `:8080` | API server bind address |
| `AGENTBOARD_MODE` | `tui` | Startup mode: `tui`, `api`, `both` |
| `AGENTBOARD_TMUX` | `auto` | tmux usage: `auto`, `always`, `never` |
| `AGENTBOARD_LLM_PROVIDER` | — | LLM provider for decomposition: `openai`, `anthropic`, `ollama` |
| `AGENTBOARD_LLM_MODEL` | — | Model name (e.g., `gpt-4o`, `claude-sonnet-4-20250514`) |
| `AGENTBOARD_LLM_API_KEY` | — | API key for LLM provider |
| `AGENTBOARD_LLM_BASE_URL` | — | Custom API base URL (for Ollama, etc.) |
| `AGENTBOARD_NPM_PATH` | `npm` | Path to npm binary (for MCP servers) |
| `AGENTBOARD_NODE_PATH` | `node` | Path to node binary (for MCP servers) |
| `NO_COLOR` | — | Disable colored output (respects standard env var) |

---

## Contributing

### Workflow

1. Create a feature branch from `main`
2. **Write the test first** — every feature, bugfix, or refactor starts with a failing test in the relevant `_test.go` file
3. Make the test pass with minimal implementation
4. Refactor if needed, keeping tests green
5. Run `go test ./...` and `go vet ./...`
6. Update AGENT.md if architecture changes
7. Submit PR with description linking to relevant ticket

### TDD Discipline

- **Red → Green → Refactor**. No implementation code without a test.
- Each `internal/` package has a `_test.go` file. Table-driven tests preferred.
- Tests must be independent and repeatable — no test ordering assumptions.
- Use interfaces and dependency injection to make packages testable in isolation.
- Mock external dependencies (tmux, MCP servers, LLM APIs) with test doubles.

### Code Conventions

- **Error wrapping**: Always wrap with `fmt.Errorf("package.context: %w", err)`
- **No circular imports**: `internal/` packages must not import each other circularly. Use interfaces defined at the consumer side.
- **No global state**: Pass dependencies explicitly. Use dependency injection in `cmd/agentboard/main.go`.
- **No comments unless asked**: Code should be self-documenting through naming.
- **Tests**: Each `internal/` package has a `_test.go` file. Table-driven tests preferred.
- **Go vet clean**: All code must pass `go vet`.

### Platform Support

- **Linux (x86_64, ARM64)**: Primary target
- **Termux (Android ARM64)**: Supported. No CGO dependencies. All pure Go.
- **macOS**: Best-effort. tmux integration works; PTY may differ.

### Git Conventions

- Conventional commits: `feat(tui): add kanban column reordering`
- Squash merge PRs
- Keep main green
