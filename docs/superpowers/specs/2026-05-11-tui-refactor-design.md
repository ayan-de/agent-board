# TUI Refactor: API-First Architecture

**Date:** 2026-05-11
**Status:** Approved
**Type:** Architecture Refactor

---

## Context

`internal/tui` is a "big ball of mud" — 983 lines in `app.go` handling orchestration calls, tmux management, completion channel polling, proposal lifecycle, and view switching. Sub-models (`KanbanModel`, `TicketViewModel`, `DashboardModel`) mix presentation with business logic.

The long-term product direction requires multiple UIs (TUI today, VSCode extension and web UI planned). The current architecture locks business logic into the TUI package.

## Decision

Separate presentation from business logic using a dedicated `internal/board` service. TUI becomes a pure rendering layer that emits user intents. The board service owns all state and workflow.

---

## Architecture

```
User Input → TUI (intent emission) → BoardService (workflow) → Orchestrator → Store
                ↑                                              ↓
                └───────────── State Updates ←────────────────┘
```

### Principles

- **BoardService has zero knowledge of how it's accessed.** TUI and API are both consumers.
- **TUI never calls Store or Orchestrator directly.** It emits intents and renders state.
- **State flows one direction.** Intent → BoardService → State → TUI renders.
- **No business logic in TUI.** Only UX chrome (modals, palettes, notifications).

---

## New Package: `internal/board`

```
internal/board/
├── intents.go      # Intent type definitions
├── state.go        # BoardViewState and sub-state structs
├── board.go        # BoardService — main entry point
├── kanban.go       # Kanban workflow logic (columns, cursors, tabs, filters)
├── ticket.go       # Active ticket workflow (edit, status cycle, agent assign)
├── proposal.go     # Proposal state machine (pending → approved → run)
└── dashboard.go   # Dashboard state (agent list, active sessions)
```

---

## Intent System

TUI emits structured intents. No response payloads — BoardService determines outcomes.

```go
// Board intents
type IntentSelectTicket  { TicketID string }
type IntentCreateTicket  { ColumnIndex int }
type IntentDeleteTicket  { TicketID string }
type IntentMoveTicket    { TicketID, NewStatus string }

// Ticket view intents
type IntentEditField    { Field, Value string }
type IntentCycleStatus  {}
type IntentAssignAgent   { AgentName string }
type IntentApproveProposal {}
type IntentStartRun     {}

// Dashboard intents
type IntentRefreshDashboard {}
type IntentStartAdHocRun { Agent, Prompt string }
```

---

## State System

BoardService publishes a single `BoardViewState` that contains everything TUI needs to render.

```go
type ViewType int

const (
    ViewBoard ViewType = iota
    ViewTicket
    ViewDashboard
    ViewHelp
)

type BoardViewState struct {
    Kanban     KanbanViewState
    Ticket     *TicketViewState      // nil when board is active
    Dashboard  DashboardViewState
    ActiveView ViewType

    Notification *Notification       // nil when dismissed
    Modal        *ModalState         // nil when closed
}
```

Sub-states are defined in their respective files (`kanban.go`, `ticket.go`, etc.).

---

## API Layer Design

API layer sits alongside TUI as an alternative consumer of BoardService.

```
BoardService (internal/board)
       ↑
   ┌───┴─────────────────┐
   │                     │
Local TUI          API Layer (internal/api)
                       │
              ┌─────────┴──────────┐
              │                    │
          VSCode Ext          Web UI
```

### Routes

```
GET   /board              → Get full BoardViewState
WS    /board/events       → Stream state diffs

POST  /tickets            → Create ticket
GET   /tickets            → List tickets (filterable)
PUT   /tickets/{id}       → Update ticket
DELETE /tickets/{id}      → Delete ticket

POST  /tickets/{id}/propose      → Trigger proposal creation
POST  /proposals/{id}/approve    → Approve proposal
POST  /proposals/{id}/run        → Start approved run

GET   /agents             → List detected agents
GET   /sessions          → List active sessions
```

### Event Streaming

WebSocket endpoint pushes `BoardViewState` diffs to all connected clients on every state change. TUI connects as a local WebSocket client when running in API mode (`AGENTBOARD_MODE=api`).

---

## TUI Transformation

### Responsibilities

- Renders `BoardViewState` to terminal string
- Emits intents from keypresses
- Handles TUI-specific chrome (modals, palettes, notifications)
- Animation ticks for visual feedback

### What TUI Does NOT Do

- No direct Store or Orchestrator calls
- No proposal or session logic
- No tmux management (BoardService handles this)

### New Structure

```
internal/tui/
├── app.go           # Thin shell: Update() routes intents, View() renders
├── renderer.go      # BoardViewState → string
├── intents.go      # Intent message wrappers for Bubble Tea
└── components/     # Pure presentation
    ├── kanban.go    # Kanban board rendering
    ├── ticket.go    # Ticket detail rendering
    ├── dashboard.go # Agent dashboard rendering
    ├── modal.go     # Modal rendering
    ├── palette.go   # Command palette
    └── notify.go    # Notification stack
```

### App.go Before/After

**Before (983 lines):**
```go
type App struct {
    store        *store.Store
    orchestrator Orchestrator
    kanban       KanbanModel
    ticketView   TicketViewModel
    dashboard    DashboardModel
    // ... business logic scattered across Update() switch
}
```

**After (~80 lines):**
```go
type App struct {
    board    *BoardService
    renderer *Renderer
    state    BoardViewState
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        a.renderer.SetSize(msg.Width, msg.Height)
        return a, nil
    case tea.KeyMsg:
        intent := a.resolveIntent(msg)
        if intent != nil {
            a.state = a.board.ProcessIntent(intent)
        }
        return a, nil
    case BoardUpdatedMsg:
        a.state = msg.State
        return a, nil
    }
    return a, nil
}

func (a *App) View() string {
    return a.renderer.Render(a.state)
}
```

**TUI imports:**
- `internal/board` (state + intents)
- `internal/theme`
- Bubble Tea + Lip Gloss (presentation primitives only)

---

## Migration Plan

### Phase 1: Create `internal/board`

1. Define intent types in `intents.go`
2. Define `BoardViewState` and sub-state structs in `state.go`
3. Implement `BoardService` in `board.go`
4. Extract logic from `app.go`, `kanban.go`, `ticketview.go`, `dashboard.go`
5. Keep orchestrator calls unchanged (they're already well-separated)
6. Write tests alongside each extraction

### Phase 2: Migrate TUI

1. Create `internal/tui/renderer.go`
2. Create `internal/tui/intents.go`
3. Refactor `app.go` to use `BoardService`
4. Move sub-models to `components/` (pure presentation only)

### Phase 3: Wire Everything

1. Update `main.go` to create `BoardService`
2. Remove direct orchestrator/store calls from TUI
3. Run full test suite, verify all features

### Files to Create

| File | Purpose |
|------|---------|
| `internal/board/intents.go` | Intent type definitions |
| `internal/board/state.go` | `BoardViewState` and sub-state structs |
| `internal/board/board.go` | `BoardService` entry point |
| `internal/board/kanban.go` | Kanban workflow logic |
| `internal/board/ticket.go` | Active ticket workflow |
| `internal/board/proposal.go` | Proposal state machine |
| `internal/tui/renderer.go` | State → string rendering |
| `internal/tui/intents.go` | Intent wrappers for Bubble Tea |

### Files to Modify

| File | Change |
|------|--------|
| `internal/tui/app.go` | Strip to ~80 lines, use BoardService |
| `internal/tui/kanban.go` | Strip to presentation only |
| `internal/tui/ticketview.go` | Strip to presentation only |
| `internal/tui/dashboard.go` | Strip to presentation only |
| `cmd/agentboard/main.go` | Create BoardService, pass to TUI |

### Files to Delete

None until Phase 3 is complete and verified.

---

## Consequences

**Benefits:**
- TUI becomes replaceable (VSCode, web UI, CLI client)
- Business logic is in one place, testable without UI
- State changes are explicit and traceable
- New features only require intent + handler, not TUI modifications

**Trade-offs:**
- Three-phase migration takes time
- BoardService must stay in sync with current behavior
- Temporary code duplication during migration

**Risks:**
- Performance impact from state serialization for API layer (mitigate: push diffs, not full state)
- BoardService becomes a god object (mitigate: keep it as a facade delegating to kanban/ticket/proposal)

---

## Alternatives Considered

### Option A: Keep sub-models, thin TUI internally

Move business logic from `app.go` into sub-models but keep them in TUI package. Rejected because it doesn't enable API-first — business logic remains coupled to TUI.

### Option B: API first, TUI as API client

Skip local TUI refactor, build HTTP API, make TUI a client. Rejected because TUI is the primary UI today and users won't accept a refactor that blocks current work.

### Option C: Event sourcing

Use event sourcing for state management. Rejected because it adds complexity disproportionate to current scale. BoardService publishes snapshots, not events.
