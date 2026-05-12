# TUI Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract business logic from TUI into `internal/board` package. TUI becomes a pure presentation layer emitting intents and rendering state.

**Architecture:** BoardService owns all board state and workflow. TUI emits intents, BoardService processes them, returns BoardViewState for rendering. API layer (future) sits alongside TUI as alternative BoardService consumer.

**Tech Stack:** Go, Bubble Tea, Lip Gloss, SQLite (store), LangChain Go (orchestrator)

---

## File Map

### New Files

| File | Purpose |
|------|---------|
| `internal/board/intents.go` | All Intent type definitions |
| `internal/board/state.go` | BoardViewState and sub-state structs |
| `internal/board/board.go` | BoardService struct, ProcessIntent, NewBoardService |
| `internal/board/kanban.go` | Kanban state machine (columns, cursors, tabs, filters) |
| `internal/board/ticket.go` | Active ticket workflow (edit, status cycle, agent assign) |
| `internal/board/proposal.go` | Proposal state machine (pending→approved→run) |
| `internal/board/dashboard.go` | Dashboard state (agents, sessions, pane management) |
| `internal/board/notify.go` | Notification state |
| `internal/tui/intents.go` | Bubble Tea intent wrappers |
| `internal/tui/renderer.go` | BoardViewState → string rendering |

### Modified Files

| File | Change |
|------|--------|
| `internal/tui/app.go` | Strip to ~80 lines, delegate to BoardService |
| `internal/tui/kanban.go` | Strip presentation only, rename to `kanban_view.go` |
| `internal/tui/ticketview.go` | Strip presentation only, rename to `ticket_view.go` |
| `internal/tui/dashboard.go` | Strip presentation only, rename to `dashboard_view.go` |
| `cmd/agentboard/main.go` | Create BoardService, pass to TUI |
| `internal/orchestrator/service.go` | No changes (already clean) |

---

## Task 1: Create `internal/board/intents.go`

**Files:**
- Create: `internal/board/intents.go`

- [ ] **Step 1: Write intents.go**

```go
package board

type Intent interface{ isIntent() }

type IntentSelectTicket struct{ TicketID string }
type IntentCreateTicket struct{ ColumnIndex int }
type IntentDeleteTicket struct{ TicketID string }
type IntentMoveTicket struct{ TicketID, NewStatus string }

type IntentEditField struct{ Field, Value string }
type IntentCycleStatus struct{}
type IntentAssignAgent struct{ AgentName string }
type IntentApproveProposal struct{}
type IntentStartRun struct{}

type IntentRefreshDashboard struct{}
type IntentStartAdHocRun struct{ Agent, Prompt string }

type IntentOpenView struct{ View ViewType }
type IntentCloseModal struct{}
type IntentConfirmModal struct{}
type IntentShowPalette struct{}
```

- [ ] **Step 2: Write state.go with ViewType and BoardViewState**

```go
package board

type ViewType int

const (
    ViewBoard ViewType = iota
    ViewTicket
    ViewDashboard
    ViewHelp
)

type BoardViewState struct {
    Kanban     KanbanViewState
    Ticket     *TicketViewState
    Dashboard  DashboardViewState
    ActiveView ViewType

    Notification *NotificationState
    Modal        *ModalState
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/board/
git commit -m "feat(board): add intents and state type definitions"
```

---

## Task 2: Create `internal/board/kanban.go`

**Files:**
- Create: `internal/board/kanban.go`

- [ ] **Step 1: Write KanbanViewState and KanbanState**

```go
package board

type KanbanViewState struct {
    Columns      []KanbanColumn
    ColIndex     int
    Cursors      []int
    ScrollOff    []int
    Tab          KanbanTab
    SearchQuery  string
    MonthOffset  int
    Theme        *theme.Theme
    Styles       KanbanStyles
}

type KanbanColumn struct {
    Def   config.Column
    Tickets []store.Ticket
}

type KanbanTab int

const (
    TabBoard KanbanTab = iota
    TabSearch
    TabDateFilter
)
```

- [ ] **Step 2: Write ProcessIntent for kanban intents**

```go
func (b *BoardService) ProcessIntent(intent Intent) BoardViewState {
    switch i := intent.(type) {
    case IntentSelectTicket:
        return b.selectTicket(i.TicketID)
    case IntentCreateTicket:
        return b.createTicket(i.ColumnIndex)
    case IntentDeleteTicket:
        return b.deleteTicket(i.TicketID)
    case IntentMoveTicket:
        return b.moveTicket(i.TicketID, i.NewStatus)
    // ... etc
    }
}
```

- [ ] **Step 3: Write kanban workflow functions**

```go
func (b *BoardService) selectTicket(ticketID string) BoardViewState {
    ticket, err := b.store.GetTicket(context.Background(), ticketID)
    if err != nil {
        return b.state
    }

    proposal, _ := b.store.GetActiveProposalForTicket(context.Background(), ticketID)

    b.state.Ticket = &TicketViewState{
        Ticket:   &ticket,
        Proposal: &proposal,
    }
    b.state.ActiveView = ViewTicket
    return b.state
}

func (b *BoardService) createTicket(colIndex int) BoardViewState {
    if colIndex >= len(b.state.Kanban.Columns) {
        return b.state
    }
    col := b.state.Kanban.Columns[colIndex].Def
    ticket, err := b.store.CreateTicket(context.Background(), store.Ticket{
        Title:  "New Ticket",
        Status: col.Status,
    })
    if err != nil {
        return b.state
    }

    b.loadKanbanState()

    b.state.Notification = &NotificationState{
        Title:   "Ticket created",
        Message: fmt.Sprintf("%s: %s", ticket.ID, ticket.Title),
        Variant: NotificationSuccess,
    }
    return b.state
}

func (b *BoardService) deleteTicket(ticketID string) BoardViewState {
    _ = b.store.DeleteTicket(context.Background(), ticketID)
    b.loadKanbanState()
    return b.state
}

func (b *BoardService) moveTicket(ticketID, newStatus string) BoardViewState {
    _ = b.store.MoveStatus(context.Background(), ticketID, newStatus)
    b.loadKanbanState()

    if b.state.Ticket != nil && b.state.Ticket.Ticket != nil && b.state.Ticket.Ticket.ID == ticketID {
        b.state.Ticket.Ticket.Status = newStatus
    }

    if newStatus == "in_progress" && b.state.Ticket != nil {
        b.state.Ticket.Loading = true
        return b.state
    }
    return b.state
}

func (b *BoardService) loadKanbanState() {
    b.state.Kanban.Columns = make([]KanbanColumn, len(b.state.Kanban.ColumnDefs))
    for i, col := range b.state.Kanban.ColumnDefs {
        tickets, _ := b.store.ListTickets(context.Background(), store.TicketFilters{Status: col.Status})
        if tickets == nil {
            tickets = []store.Ticket{}
        }
        b.state.Kanban.Columns[i] = KanbanColumn{
            Def:     col,
            Tickets: tickets,
        }
    }
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/board/kanban.go
git commit -m "feat(board): add kanban state and workflow"
```

---

## Task 3: Create `internal/board/ticket.go`

**Files:**
- Create: `internal/board/ticket.go`

- [ ] **Step 1: Write TicketViewState**

```go
type TicketViewState struct {
    Ticket         *store.Ticket
    Fields         []TicketField
    Cursor         int
    Mode           TicketViewMode
    EditBuffer     string
    Agents         []config.DetectedAgent
    Proposal       *store.Proposal
    Loading        bool
    DependsOnTickets []store.Ticket
    DependsOnSelected []string
}

type TicketViewMode int

const (
    TicketViewMode TicketViewMode = iota
    TicketEditMode
    TicketAgentSelectMode
    TicketPrioritySelectMode
    TicketDependsOnSelectMode
)
```

- [ ] **Step 2: Write ticket workflow intents**

```go
func (b *BoardService) ProcessIntent(intent Intent) BoardViewState {
    switch i := intent.(type) {
    case IntentEditField:
        return b.editField(i.Field, i.Value)
    case IntentCycleStatus:
        return b.cycleTicketStatus()
    case IntentAssignAgent:
        return b.assignAgent(i.AgentName)
    case IntentApproveProposal:
        return b.approveProposal()
    case IntentStartRun:
        return b.startRun()
    }
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/board/ticket.go
git commit -m "feat(board): add ticket workflow intents"
```

---

## Task 4: Create `internal/board/proposal.go`

**Files:**
- Create: `internal/board/proposal.go`

- [ ] **Step 1: Write proposal workflow functions**

```go
func (b *BoardService) createProposal(ticketID string) BoardViewState {
    b.state.Ticket.Loading = true

    proposal, err := b.orchestrator.CreateProposal(context.Background(), orchestrator.CreateProposalInput{
        TicketID: ticketID,
    })

    if err != nil {
        b.state.Notification = &NotificationState{
            Title:   "Proposal failed",
            Message: err.Error(),
            Variant: NotificationError,
        }
        b.state.Ticket.Loading = false
        return b.state
    }

    b.state.Ticket.Proposal = &proposal
    b.state.Ticket.Loading = false
    return b.state
}

func (b *BoardService) approveProposal() BoardViewState {
    if b.state.Ticket.Proposal == nil {
        return b.state
    }

    _ = b.orchestrator.ApproveProposal(context.Background(), b.state.Ticket.Proposal.ID)
    p, _ := b.store.GetProposal(context.Background(), b.state.Ticket.Proposal.ID)
    b.state.Ticket.Proposal = &p
    return b.state
}

func (b *BoardService) startRun() BoardViewState {
    if b.state.Ticket.Proposal == nil || b.state.Ticket.Proposal.Status != "approved" {
        return b.state
    }

    session, err := b.orchestrator.StartApprovedRun(context.Background(), b.state.Ticket.Proposal.ID)
    if err != nil {
        b.state.Notification = &NotificationState{
            Title:   "Run failed",
            Message: err.Error(),
            Variant: NotificationError,
        }
    }
    return b.state
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/board/proposal.go
git commit -m "feat(board): add proposal state machine"
```

---

## Task 5: Create `internal/board/dashboard.go`

**Files:**
- Create: `internal/board/dashboard.go`

- [ ] **Step 1: Write DashboardViewState**

```go
type DashboardViewState struct {
    Agents         []config.DetectedAgent
    ActiveSessions map[string]store.Session
    PaneID         string
    Styles         DashboardStyles
    Theme          *theme.Theme
}
```

- [ ] **Step 2: Write dashboard intents**

```go
func (b *BoardService) refreshDashboard() BoardViewState {
    agents := config.DetectAgents()
    b.state.Dashboard.Agents = agents

    sessions := b.orchestrator.GetActiveSessions()
    b.state.Dashboard.ActiveSessions = make(map[string]store.Session)
    for _, s := range sessions {
        b.state.Dashboard.ActiveSessions[s.Agent] = store.Session{
            ID:        s.SessionID,
            TicketID:  s.TicketID,
            Agent:     s.Agent,
            Status:    s.Status,
            StartedAt: time.Unix(s.StartedAt, 0),
        }
    }
    return b.state
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/board/dashboard.go
git commit -m "feat(board): add dashboard state"
```

---

## Task 6: Create `internal/board/board.go`

**Files:**
- Create: `internal/board/board.go`

- [ ] **Step 1: Write BoardService struct**

```go
type BoardService struct {
    store        *store.Store
    orchestrator Orchestrator
    config       *config.Config
    registry     *theme.Registry

    state BoardViewState
}

type Orchestrator interface {
    CreateProposal(ctx context.Context, input CreateProposalInput) (store.Proposal, error)
    ApproveProposal(ctx context.Context, proposalID string) error
    StartApprovedRun(ctx context.Context, proposalID string) (store.Session, error)
    StartAdHocRun(ctx context.Context, agent, prompt string) (store.Session, error)
    GetActiveSessions() []*AgentSession
}

func NewBoardService(s *store.Store, orch Orchestrator, cfg *config.Config, reg *theme.Registry) *BoardService {
    b := &BoardService{
        store:        s,
        orchestrator:  orch,
        config:       cfg,
        registry:     reg,
    }

    // Initialize kanban state
    b.state.Kanban.ColumnDefs = cfg.Board.Columns
    if b.state.Kanban.ColumnDefs == nil {
        b.state.Kanban.ColumnDefs = config.DefaultColumns()
    }
    b.state.Kanban.Cursors = make([]int, len(b.state.Kanban.ColumnDefs))
    b.state.Kanban.ScrollOff = make([]int, len(b.state.Kanban.ColumnDefs))

    // Initialize dashboard
    b.state.Dashboard.Agents = config.DetectAgents()
    b.state.Dashboard.ActiveSessions = make(map[string]store.Session)

    b.loadKanbanState()
    return b
}
```

- [ ] **Step 2: Write ProcessIntent dispatcher**

```go
func (b *BoardService) ProcessIntent(intent Intent) BoardViewState {
    switch i := intent.(type) {
    case IntentSelectTicket:
        return b.selectTicket(i.TicketID)
    case IntentCreateTicket:
        return b.createTicket(i.ColumnIndex)
    case IntentDeleteTicket:
        return b.deleteTicket(i.TicketID)
    case IntentMoveTicket:
        return b.moveTicket(i.TicketID, i.NewStatus)
    case IntentEditField:
        return b.editField(i.Field, i.Value)
    case IntentCycleStatus:
        return b.cycleTicketStatus()
    case IntentAssignAgent:
        return b.assignAgent(i.AgentName)
    case IntentApproveProposal:
        return b.approveProposal()
    case IntentStartRun:
        return b.startRun()
    case IntentOpenView:
        b.state.ActiveView = i.View
        return b.state
    case IntentShowPalette:
        return b.state
    default:
        return b.state
    }
}
```

- [ ] **Step 3: Write buildState helper**

```go
func (b *BoardService) buildState() BoardViewState {
    b.state.Kanban.Theme = b.registry.Active()
    b.state.Kanban.Styles = NewKanbanStyles(b.registry.Active())
    return b.state
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/board/board.go
git commit -m "feat(board): add BoardService and ProcessIntent dispatcher"
```

---

## Task 7: Create `internal/tui/intents.go`

**Files:**
- Create: `internal/tui/intents.go`

- [ ] **Step 1: Write Bubble Tea intent wrappers**

```go
package tui

import "github.com/ayan-de/agent-board/internal/board"

type boardIntentMsg struct {
    intent board.Intent
}

func BoardIntent(i board.Intent) tea.Msg {
    return boardIntentMsg{intent: i}
}

func extractIntent(msg tea.Msg) board.Intent {
    switch m := msg.(type) {
    case boardIntentMsg:
        return m.intent
    }
    return nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/tui/intents.go
git commit -m "feat(tui): add Bubble Tea intent wrappers"
```

---

## Task 8: Create `internal/tui/renderer.go`

**Files:**
- Create: `internal/tui/renderer.go`

- [ ] **Step 1: Write Renderer struct**

```go
type Renderer struct {
    width  int
    height int
}

func NewRenderer(width, height int) *Renderer {
    return &Renderer{width: width, height: height}
}

func (r *Renderer) SetSize(width, height int) {
    r.width = width
    r.height = height
}
```

- [ ] **Step 2: Write Render method**

```go
func (r *Renderer) Render(state board.BoardViewState) string {
    switch state.ActiveView {
    case board.ViewBoard:
        return r.renderKanban(state.Kanban)
    case board.ViewTicket:
        return r.renderTicket(state.Ticket)
    case board.ViewDashboard:
        return r.renderDashboard(state.Dashboard)
    case board.ViewHelp:
        return r.renderHelp()
    }
    return ""
}
```

- [ ] **Step 3: Write sub-renderers using existing kanban/ticket/dashboard View() methods**

```go
func (r *Renderer) renderKanban(state board.KanbanViewState) string {
    model := NewKanbanModelFromState(state)
    return model.View()
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/tui/renderer.go
git commit -m "feat(tui): add state renderer"
```

---

## Task 9: Refactor `internal/tui/app.go`

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Strip App struct to minimal form**

```go
type App struct {
    board    *board.BoardService
    renderer *Renderer
    state    board.BoardViewState
    palette  CommandPalette
    modal    ConfirmModal
    textInput TextInputModal
    notification NotificationStack

    quit bool
}
```

- [ ] **Step 2: Replace Update() with intent routing**

```go
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
    case notificationDismissMsg:
        a.state.Notification = nil
        return a, nil
    }
    return a, nil
}
```

- [ ] **Step 3: Replace View() with renderer call**

```go
func (a *App) View() string {
    return a.renderer.Render(a.state)
}
```

- [ ] **Step 4: Remove orchestrator, store, kanban, ticketView, dashboard fields**
- [ ] **Step 5: Remove all proposal/run/completion channel logic**
- [ ] **Step 6: Remove tmux split management (move to board)**

- [ ] **Step 7: Commit**

```bash
git add internal/tui/app.go
git commit -m "refactor(tui): strip App to intent router and renderer"
```

---

## Task 10: Update `cmd/agentboard/main.go`

**Files:**
- Modify: `cmd/agentboard/main.go`

- [ ] **Step 1: Create BoardService in main()**

```go
boardSvc := board.NewBoardService(store, orch, cfg, registry)

app, err := tui.NewApp(cfg, store, registry, tui.AppDeps{
    Board: boardSvc,
})
```

- [ ] **Step 2: Commit**

```bash
git add cmd/agentboard/main.go
git commit -m "chore(main): wire BoardService to TUI"
```

---

## Task 11: Integration and verification

**Files:**
- All modified files

- [ ] **Step 1: Run full test suite**

```bash
go test ./...
go vet ./...
```

- [ ] **Step 2: Run TUI manually, verify all features work**

```bash
./agentboard
```

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "chore: complete TUI refactor - business logic in board service"
```

---

## Self-Review Checklist

- [ ] All intent types defined in `intents.go` have a handler in `board.go`
- [ ] `BoardViewState` covers all fields needed by renderer
- [ ] No direct store/orchestrator calls in TUI package
- [ ] TUI imports `internal/board`, not `internal/orchestrator` or `internal/store`
- [ ] All existing features verified after migration
- [ ] Tests pass for modified packages
