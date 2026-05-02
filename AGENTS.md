# AGENTS.md — AgentBoard

## Project Overview

AgentBoard is currently a terminal-based Kanban board for managing AI-oriented development tickets.

The implemented product today is the TUI foundation plus the first AI orchestration slice:
- Bubble Tea application shell
- Kanban board and ticket detail views
- Ticket persistence in SQLite
- Config loading and project-scoped config scaffolding
- Agent detection from `$PATH`
- Theme registry with builtin and user JSON themes
- Configurable keybindings and command palette
- Agent dashboard based on local detection state
- AI orchestration: proposal creation, approval gating, subprocess worker execution
- LangChain Go integration for coordinator and summarizer models
- MCP context carry integration via `@thisisayande/contextcarry-mcp`

The long-term product direction is full AI agent orchestration:
- tmux and embedded PTY agent execution
- session and process lifecycle management
- additional MCP integrations
- HTTP/WebSocket API
- LLM-based decomposition and assignment

---

## Status Snapshot

### Implemented

- `cmd/agentboard/main.go` starts the TUI, creates tmux session if not in one
- `internal/tui` contains the working Bubble Tea application
- `internal/store` contains the SQLite-backed ticket, session, proposal, event, and context carry persistence
- `internal/config` handles defaults, TOML loading, env overlay, project naming, config scaffolding, agent detection, and MCP server config
- `internal/theme` handles builtin theme embedding, user theme loading, parsing, and runtime selection
- `internal/keybinding` handles keymap definitions, config overrides, and action resolution
- `internal/llm` provides provider registry (openai, ollama, claude, zai) with LangChain Go behind a Client interface
- `internal/orchestrator` implements proposal creation, approval gating, session start, outcome mapping, context carry persistence, and agent execution via TmuxRunner and PtyRunner
- `internal/pty` provides agent configs (opencode, claude-code, codex, gemini) with ready patterns, prompt formatting, and PTY state machine
- `internal/prompt` central repository for all LLM prompt templates
- `internal/mcp` provides MCP manager, context carry adapter with load/save via MCP protocol
- `internal/mcpclient` wraps mcp-go stdio client for MCP server communication

### Partially Implemented

- `internal/tui/dashboard.go` shows detected agents and active sessions; connected to live agent panes via tmux split

### Placeholder / Not Yet Implemented

- `internal/api`
- `internal/decomposition`
- `internal/apitypes`

When touching one of those packages, assume the architecture is still open unless another section here says otherwise.

---

## Current Architecture

### Runtime Flow Today

```text
cmd/agentboard/main.go
  -> config.Load()
  -> store.Open()
  -> llm.NewFromConfig()
  -> mcp.NewManager() + mcp.NewContextCarryAdapter()
  -> if in tmux: NewTmuxRunner(sessionName), NewPtyRunner(sessionName)
  -> orchestrator.NewService(store, llm, runner, ctxCarry)
  -> orch.SetPtyRunner(ptyRunner)  // if available
  -> theme.Registry setup
  -> tui.NewApp(cfg, store, registry, AppDeps{Orchestrator})
  -> bubbletea.Program.Run()
```

When not already in tmux, main.go creates a new tmux session named `{project-name}` and re-executes inside it.

### Package Responsibilities

```text
agent-board/
├── cmd/agentboard/
│   └── main.go             # TUI entrypoint only
├── internal/
│   ├── tui/                # Working Bubble Tea app and views
│   ├── store/              # Working SQLite persistence
│   ├── config/             # Working config loading, defaults, detection
│   ├── theme/              # Working theme registry and JSON theme loading
│   ├── keybinding/         # Working keymap model and config overrides
│   ├── llm/                # Working LangChain Go integration with provider registry
│   ├── orchestrator/       # Working agent lifecycle layer
│   ├── prompt/             # Working central prompt repository
│   ├── mcp/                # Working MCP manager and context carry adapter
│   ├── mcpclient/          # Working mcp-go stdio client wrapper
│   ├── api/                # Planned HTTP/WebSocket API
│   ├── decomposition/      # Planned LLM-driven project decomposition
│   └── apitypes/           # Planned shared DTOs
├── docs/                   # Design notes, plans, and roadmap
└── AGENTS.md               # Operational project memory
```

### Data Flow Today

```text
TUI <-> Store
TUI <-> Config
TUI <-> Theme Registry
TUI <-> Orchestrator <-> Store
                    <-> LLM (coordinator/summarizer)
                    <-> Runner (subprocess exec)
                    <-> ContextCarryProvider (MCP)
```

---

## Working Features

### TUI

- Kanban board with four status columns
- Ticket detail view
- Ticket create and delete from the board
- Ticket editing from the ticket view
- Ticket status cycling from the ticket view
- Ticket agent assignment from the ticket view
- Agent dashboard view
- Command palette
- Help view
- Theme switching
- Non-blocking auto-dismissing notification stack at the bottom-right for short workflow feedback

### Persistence

- SQLite database creation and migrations
- ticket CRUD
- ticket filtering
- session CRUD primitives
- proposal CRUD with status tracking
- orchestration event recording
- context carry upsert with ticket-scoped keys
- ticket ID generation from configurable project prefix

### Config

- default config generation under `~/.agentboard`
- global and project config files
- env var overlay for several runtime fields
- project name derived from git remote or working directory
- agent detection for local CLIs on `$PATH`
- MCP server configuration with `[mcp.<name>]` sections in config.toml

### AI Orchestration

- proposal creation triggered by moving assigned ticket to `in_progress`
- coordinator LLM shapes worker prompt from ticket context + context carry
- approval gate with stale proposal rejection
- `TmuxRunner` (PaneManager) and `PtyRunner` for agent execution in tmux windows
- real PTY allocation with character-by-character prompt injection for interactive agents (opencode)
- tmux window management: creates `agent-{sessionID}` windows, destroys on completion
- dashboard split pane shows agent output via `tmux attach-session`
- outcome-driven board transitions (completed -> review, failed -> stays)
- context carry persistence and summarization for run continuity
- event recording for orchestration lifecycle
- MCP context carry integration via `@thisisayande/contextcarry-mcp`

### LLM Integration

- provider registry with openai, ollama, claude, zai support
- LangChain Go isolated behind `internal/llm` Client interface
- separate coordinator and summarizer model configuration
- central prompt repository in `internal/prompt`

### Themes

- builtin embedded themes
- JSON theme parsing
- user theme loading from filesystem
- runtime theme switching
- persistence of selected theme into project config

---

## Current Keybindings

These are the current default bindings implemented in code.

### Global / Board

| Key | Action |
|-----|--------|
| `h/l` or `←/→` | Move between Kanban columns |
| `j/k` or `↑/↓` | Move between tickets |
| `Enter` | Open ticket detail view |
| `a` | Add a new ticket in the active column |
| `d` | Delete the selected ticket |
| `1-4` | Jump to a specific column |
| `?` | Toggle help view |
| `i` | Toggle agent dashboard |
| `:` | Open command palette |
| `q` | Quit with confirmation |
| `Ctrl+C` | Force quit |
| `Esc` | Return to board from other views |

### Ticket View

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Move between fields |
| `e` | Edit the selected editable field |
| `s` | Cycle ticket status |
| `a` | Open agent selection |
| `p` | Approve pending proposal |
| `r` | Start approved run |
| `Enter` | Save edits / confirm selection |
| `Esc` | Cancel edit or return to board |

### Dashboard

| Key | Action |
|-----|--------|
| `r` | Re-run agent detection |
| `Esc` | Return to board |

Some keybinding actions already exist in the keybinding package but are reserved for future orchestration behavior, such as start/stop agent, focus switching, and go-to-ticket chord support.

---

## Current Data Model

### Ticket

Current stored fields:

```text
id            TEXT PRIMARY KEY
title         TEXT NOT NULL
description   TEXT
status        TEXT
priority      TEXT
agent         TEXT
branch        TEXT
tags          TEXT JSON array
depends_on    TEXT JSON array
agent_active  INTEGER
created_at    DATETIME
updated_at    DATETIME
```

Notes:
- `tags` and `depends_on` are stored as JSON arrays, not comma-separated strings
- `agent_active` is set by the orchestrator when a worker starts/stops

### Proposal

```text
id         TEXT PRIMARY KEY
ticket_id  TEXT NOT NULL
agent      TEXT NOT NULL
status     TEXT NOT NULL (pending|approved|rejected)
prompt     TEXT NOT NULL
created_at DATETIME NOT NULL
updated_at DATETIME NOT NULL
```

### Event

```text
id         TEXT PRIMARY KEY
ticket_id  TEXT NOT NULL
session_id TEXT
kind       TEXT NOT NULL
payload    TEXT NOT NULL
created_at DATETIME NOT NULL
```

### Context Carry

```text
ticket_id  TEXT PRIMARY KEY
summary    TEXT NOT NULL
updated_at DATETIME NOT NULL
```

### Session

Current stored fields:

```text
id           TEXT PRIMARY KEY
ticket_id    TEXT FK -> tickets.id
agent        TEXT
started_at   DATETIME
ended_at     DATETIME
status       TEXT
context_key  TEXT
```

Notes:
- session persistence is used by the orchestrator for active run tracking
- proposals track the approval pipeline per ticket
- events provide an append-only audit log of orchestration lifecycle

---

## Build and Run

### Commands That Work Today

```bash
# Build
go build -o agentboard ./cmd/agentboard

# Run TUI
./agentboard

# Run tests
go test ./...

# Run vet
go vet ./...
```

### Notes

- the entrypoint currently launches TUI mode only
- there is no implemented `--api` mode yet
- there is no implemented `init` subcommand yet
- in restricted sandboxes, `go test ./...` may require a writable `GOCACHE`

---

## Environment Variables

### Implemented

| Variable | Description |
|----------|-------------|
| `AGENTBOARD_LOG` | overrides `general.log` |
| `AGENTBOARD_ADDR` | overrides `general.addr` |
| `AGENTBOARD_MODE` | overrides `general.mode` |
| `AGENTBOARD_TMUX` | overrides `general.tmux` |
| `AGENTBOARD_DB` | overrides `db.path` |
| `AGENTBOARD_LLM_PROVIDER` | overrides `llm.provider` |
| `AGENTBOARD_LLM_MODEL` | overrides `llm.model` |
| `AGENTBOARD_LLM_API_KEY` | overrides `llm.api_key` |
| `AGENTBOARD_LLM_BASE_URL` | overrides `llm.base_url` |
| `AGENTBOARD_LLM_COORDINATOR_MODEL` | overrides `llm.coordinator_model` |
| `AGENTBOARD_LLM_SUMMARIZER_MODEL` | overrides `llm.summarizer_model` |
| `AGENTBOARD_NPM_PATH` | overrides `mcp.npm_path` |
| `AGENTBOARD_NODE_PATH` | overrides `mcp.node_path` |

### Not Implemented Yet

- `AGENTBOARD_CONFIG` is documented elsewhere but is not currently honored by `config.Load()`

---

## Design Decisions

| Decision | Current Assessment |
|----------|--------------------|
| **internal/ for all private packages** | Good. This keeps the surface small while the architecture is still moving. |
| **Everything internal until proven external** | Good. `apitypes` and `mcpclient` should stay internal until there is a real external consumer. |
| **Config/store/TUI split** | Good. These boundaries are clear and are the strongest part of the codebase today. |
| **modernc.org/sqlite** | Good choice for pure-Go portability, especially Termux. |
| **Bubble Tea + Lip Gloss** | Good fit for the product. |
| **TDD-first discipline** | Good rule. It is followed well in the implemented packages and should continue for orchestration work. |
| **Orchestrator as a separate layer** | Good direction, but it must become the single owner of process lifecycle once implemented. Do not let TUI start owning process logic. |

---

## Architecture Guidance

### What Is Good

- package boundaries are simple and readable
- the TUI depends on stable services like config, store, theme, and keybinding rather than global state
- persistence is isolated behind the store package
- theme and keybinding systems are independent, testable modules
- the repo has useful tests in the implemented areas

### Main Risks

- placeholder packages can create false confidence if docs or future code assume they already define stable contracts
- `internal/tui` currently owns some workflow logic directly; if that expands into orchestration behavior, the UI layer will become too heavy
- `store` currently mixes durable domain state with some future runtime state like `agent_active`; keep a clear distinction once live agents exist
- `config` is doing several jobs today: defaults, path resolution, env overlay, project naming, and agent detection. This is still manageable, but avoid turning it into a catch-all package

### Guidance For Upcoming Orchestration Work

- make `internal/orchestrator` the only package that starts, stops, and observes agent processes
- keep Bubble Tea as a presentation layer that issues intents and renders state
- introduce interfaces at the consumer boundary when real orchestration dependencies arrive
- do not let `api` call process code separately from TUI; both TUI and API should talk to the same orchestration/service layer
- treat MCP and tmux integrations as adapters behind small interfaces
- keep session persistence and live runtime state separate in the model

---

## Suggested Next Build Order

1. Define the orchestrator domain model and interfaces before wiring tmux or PTY process execution.
2. Implement a minimal in-process orchestrator service for start, stop, list, and observe agent sessions.
3. Hook the TUI to that orchestrator for one agent flow end to end.
4. Add persistence for orchestration events and session transitions where needed.
5. Add tmux and PTY adapters behind the orchestrator.
6. Add API handlers only after the orchestrator API is stable enough for both TUI and HTTP consumers.
7. Add MCP and decomposition after the local orchestration loop is solid.

---

## Contributing

### Workflow

1. Create a feature branch from `main`
2. Write the test first
3. Make the test pass with the smallest correct implementation
4. Refactor while keeping tests green
5. Run `go test ./...` and `go vet ./...`
6. Update `AGENTS.md` when architecture, runtime behavior, or developer contracts change
7. Submit a PR linked to the relevant ticket

### Code Conventions

- wrap errors with `fmt.Errorf("context: %w", err)`
- avoid circular imports
- avoid global mutable state
- keep comments sparse and high-signal
- prefer table-driven tests
- keep the UI layer thin as orchestration arrives

### Platform Support

- Linux: primary target
- Termux / Android ARM64: supported target, avoid CGO
- macOS: best effort
