# AGENTS.md — AgentBoard

## Project Overview

AgentBoard is currently a terminal-based Kanban board for managing AI-oriented development tickets.

The implemented product today is the TUI foundation:
- Bubble Tea application shell
- Kanban board and ticket detail views
- Ticket persistence in SQLite
- Config loading and project-scoped config scaffolding
- Agent detection from `$PATH`
- Theme registry with builtin and user JSON themes
- Configurable keybindings and command palette
- Agent dashboard based on local detection state

The long-term product direction is still AI agent orchestration:
- tmux and embedded PTY agent execution
- session and process lifecycle management
- MCP integrations
- HTTP/WebSocket API
- LLM-based decomposition and assignment

Those orchestration features are planned, but are not implemented yet.

---

## Status Snapshot

### Implemented

- `cmd/agentboard/main.go` starts the TUI
- `internal/tui` contains the working Bubble Tea application
- `internal/store` contains the SQLite-backed ticket and session persistence layer
- `internal/config` handles defaults, TOML loading, env overlay, project naming, config scaffolding, and agent detection
- `internal/theme` handles builtin theme embedding, user theme loading, parsing, and runtime selection
- `internal/keybinding` handles keymap definitions, config overrides, and action resolution

### Partially Implemented

- `internal/store/sessions.go` exists and persists session records, but there is no orchestrator using it yet
- `internal/tui/dashboard.go` shows detected agents, but it is not connected to live agent processes
- ticket assignment and status updates exist in the TUI, but agent execution does not

### Placeholder / Not Yet Implemented

- `internal/orchestrator`
- `internal/mcp`
- `internal/api`
- `internal/decomposition`
- `internal/apitypes`
- `internal/mcpclient`
- `internal/tui/pane.go` as a real embedded agent terminal

When touching one of those packages, assume the architecture is still open unless another section here says otherwise.

---

## Current Architecture

### Runtime Flow Today

```text
cmd/agentboard/main.go
  -> config.Load()
  -> store.Open()
  -> theme.Registry setup
  -> tui.NewApp()
  -> bubbletea.Program.Run()
```

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
│   ├── orchestrator/       # Planned agent lifecycle layer
│   ├── mcp/                # Planned MCP integrations
│   ├── api/                # Planned HTTP/WebSocket API
│   ├── decomposition/      # Planned LLM-driven project decomposition
│   ├── apitypes/           # Planned shared DTOs
│   └── mcpclient/          # Planned reusable MCP wrapper
├── docs/                   # Design notes, plans, and roadmap
└── AGENTS.md               # Operational project memory
```

### Data Flow Today

The actual flow today is simpler than the long-term vision:

```text
TUI <-> Store
TUI <-> Config
TUI <-> Theme Registry
TUI <-> Agent Detection
```

There is no orchestrator in the runtime path yet.

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

### Persistence

- SQLite database creation and migrations
- ticket CRUD
- ticket filtering
- session CRUD primitives
- ticket ID generation from configurable project prefix

### Config

- default config generation under `~/.agentboard`
- global and project config files
- env var overlay for several runtime fields
- project name derived from git remote or working directory
- agent detection for local CLIs on `$PATH`

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
- `agent_active` exists in the store schema already, even though there is no running orchestrator yet

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
- session persistence exists before live process orchestration
- this is acceptable, but the runtime does not use sessions yet

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
