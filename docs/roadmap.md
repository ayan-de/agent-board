# AgentBoard Roadmap

## Phase 1 — Core TUI Shell (start here)

> Get something on screen. Learn Go + TDD by building the visible layer first.

| Step | Package | What | TDD Approach |
|------|---------|------|-------------|
| 1.1 | `config/` | Config struct + TOML loading + env overlay | Test: load config, env overrides, defaults |
| 1.2 | `store/` | SQLite init, migrations, ticket CRUD | Test: create/read/update/delete tickets |
| 1.3 | `keybinding/` | Centralized key mapping, Action constants, KeyMap struct, user-configurable via TOML | Test: key → action lookup, custom bindings, default bindings |
| 1.4 | `tui/app.go` | Bubbletea root model, window size tracking, routes actions from keybinding package | Test: Init/Update/View returns expected |
| 1.5 | `tui/kanban.go` | 4-column board: Backlog → In Progress → Review → Done | Test: column navigation, ticket list |
| 1.6 | `tui/ticketview.go` | Ticket detail panel (title, desc, status) | Test: view/edit ticket fields |
| 1.7 | `cmd/agentboard/main.go` | Wire config → store → keybinding → TUI, start program | Manual: `go run ./cmd/agentboard` |

**Milestone**: `./agentboard` opens a working Kanban board you can add/move/delete tickets on.

---

## Phase 2 — Theme System & Slash Commands

> Customizable look and feel. `/theme`, `/board`, `/quit` etc.

| Step | Package | What |
|------|---------|------|
| 2.1 | `tui/theme/` | Theme struct (colors, borders, spacing, font styles). Built-in themes: `default`, `dracula`, `solarized`, `catppuccin`, `gruvbox` |
| 2.2 | `tui/command.go` | Slash command parser — `/theme dracula`, `/board compact`, `/quit` |
| 2.3 | `tui/commandbar.go` | Input bar at bottom (`:` or `/` triggers it) |
| 2.4 | `config/themes/` | TOML theme files in `~/.agentboard/themes/` for user-defined themes |
| 2.5 | `tui/layout.go` | Layout engine: `compact`, `comfortable`, `spacious` — configurable gaps/padding |

**Milestone**: `/theme catppuccin` instantly restyles the board. Users can create custom themes via TOML.

---

## Phase 3 — Agent Spawning (Panes & Floating)

> Run opencode/claude/codex inside the TUI as embedded panes or floating windows.

| Step | Package | What |
|------|---------|------|
| 3.1 | `config/detection.go` | Detect `claude`, `opencode`, `codex` on `$PATH` |
| 3.2 | `orchestrator/agent.go` | Agent struct: name, type, status, ticket assignment |
| 3.3 | `orchestrator/pty.go` | PTY allocation — allocate pseudo-terminal, capture I/O |
| 3.4 | `orchestrator/spawner.go` | Start agent process, attach to PTY |
| 3.5 | `tui/pane.go` | Embedded pane widget — renders PTY output inside bubbletea |
| 3.6 | `tui/floating.go` | Floating pane overlay — agent pane as a modal/popup over the board |
| 3.7 | `tui/layout_manager.go` | Pane layout: `split-below`, `split-right`, `floating`, `tabbed` |
| 3.8 | `orchestrator/session.go` | tmux session management (optional mode for tmux users) |

**Pane styles:**

```
split-below:          split-right:         floating:         tabbed:
┌──────────────┐      ┌────┬─────────┐    ┌──────────────┐  [Board|Agent1|Agent2]
│   Kanban     │      │ K  │ Agent   │    │ Kanban       │  ┌──────────────────┐
├──────────────┤      │ a  │ Pane    │    │ ┌──────────┐ │  │ Active tab       │
│ Agent Pane   │      │ n  │         │    │ │ Float    │ │  │ content          │
└──────────────┘      │ b  │         │    │ │ Agent    │ │  └──────────────────┘
                      │ a  │         │    │ │          │ │
                      │ n  │         │    │ └──────────┘ │
                      └────┴─────────┘    └──────────────┘
```

**Milestone**: Press `s` on a ticket → agent spawns in a pane/floating window. Switch layouts with `/layout floating`.

---

## Phase 4 — AI Decomposition (LangChain Go)

> Break a project spec into tickets using LLM. LangChain Go for structured AI workflows.

| Step | Package | What |
|------|---------|------|
| 4.1 | `decomposition/llm.go` | LLM provider interface (OpenAI, Anthropic, Ollama) via LangChain Go |
| 4.2 | `decomposition/decomposer.go` | Project description → structured ticket list |
| 4.3 | `decomposition/assigner.go` | Auto-assign tickets to best agent with reasoning |
| 4.4 | `decomposition/prompts.go` | Prompt templates (decomposition, assignment, re-estimation) |
| 4.5 | `tui/decompose_view.go` | UI: paste project spec → review generated tickets → accept/reject/edit |
| 4.6 | `/decompose` command | Slash command to trigger decomposition on current project |

**Milestone**: `/decompose "Build auth system with JWT"` → board fills with tickets, auto-assigned to agents.

---

## Phase 5 — MCP Integration (Context Carry)

> Persist agent context across sessions.

| Step | Package | What |
|------|---------|------|
| 5.1 | `mcp/client.go` | Generic MCP client bootstrap (stdio transport) |
| 5.2 | `mcp/contextcarry.go` | Save/load agent context per ticket |
| 5.3 | `mcp/sessioncarry.go` | Cross-session state tracking |
| 5.4 | `mcpclient/client.go` | Reusable client wrapper for external consumers |

**Milestone**: Agent picks up a ticket it worked on yesterday → loads prior context automatically.

---

## Phase 6 — HTTP/WebSocket API

> Remote control for Next.js frontend.

| Step | Package | What |
|------|---------|------|
| 6.1 | `api/server.go` | Chi router, CORS, middleware |
| 6.2 | `api/handlers.go` | REST: tickets, sessions, agents CRUD |
| 6.3 | `api/websocket.go` | Real-time event streaming (board updates) |
| 6.4 | `apitypes/` | Shared DTOs |
| 6.5 | `cmd/agentboard/main.go` | `--api` mode flag |

**Milestone**: `curl localhost:8080/api/tickets` returns board state. WebSocket streams live updates.

---

## Phase 7 — Polish & Distribution

| Step | What |
|------|------|
| 7.1 | Plugin system for custom slash commands |
| 7.2 | Board persistence — auto-save on every mutation |
| 7.3 | Multi-project support (switch boards) |
| 7.4 | `Makefile` + `goreleaser` for cross-platform binaries |
| 7.5 | Termux (Android ARM64) support — pure Go, no CGO |

---

## Build Order

```
Phase 1 (Core TUI)     ← START HERE, learn Go fundamentals
    ↓
Phase 2 (Themes/Cmds)  ← Visual polish, slash commands
    ↓
Phase 3 (Agent Panes)  ← The "superpower" — embedded agents
    ↓
Phase 4 (LangChain)    ← AI decomposition
    ↓
Phase 5 (MCP)          ← Context persistence
    ↓
Phase 6 (API)          ← Remote control
    ↓
Phase 7 (Polish)       ← Distribution
```
