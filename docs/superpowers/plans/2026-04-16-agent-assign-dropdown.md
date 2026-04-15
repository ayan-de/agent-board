# Agent Assignment Dropdown + Kanban Agent Dot — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an agent selection dropdown to the ticket detail view and show a colored dot on kanban tickets that have an assigned agent.

**Architecture:** Add `AgentColor()` helper to `config` package, a new `ticketAgentSelectMode` to the ticket view, and agent-dot rendering to the kanban board. Pass detected agents from `App` to `TicketViewModel` at construction time.

**Tech Stack:** Go, bubbletea, lipgloss, SQLite (via modernc.org/sqlite)

---

### Task 1: Add `AgentColor()` helper to config package

**Files:**
- Modify: `internal/config/agent_detect.go` (add function after line 48)
- Test: `internal/config/agent_detect_test.go` (new file)

- [ ] **Step 1: Write the failing test**

Create `internal/config/agent_detect_test.go`:

```go
package config

import "testing"

func TestAgentColor(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"claude-code", "#D97757"},
		{"opencode", "#808080"},
		{"codex", "#10A37F"},
		{"cursor", "#F0DB4F"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := AgentColor(tt.name)
		if got != tt.want {
			t.Errorf("AgentColor(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestAgentColor -v`
Expected: FAIL — `AgentColor` undefined

- [ ] **Step 3: Write minimal implementation**

Append to `internal/config/agent_detect.go` after the closing brace of `DetectAgents()`:

```go
func AgentColor(name string) string {
	for _, spec := range agentSpecs {
		if spec.name == name {
			return spec.logoClr
		}
	}
	return ""
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestAgentColor -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/agent_detect.go internal/config/agent_detect_test.go
git commit -m "feat(config): add AgentColor helper for agent-to-color mapping"
```

---

### Task 2: Add `ticketAgentSelectMode` to ticket view

**Files:**
- Modify: `internal/tui/ticketview.go`
- Test: `internal/tui/ticketview_test.go`

This task adds the new mode constant, fields, constructor parameter, and key handling. It does NOT add the dropdown rendering yet (Task 3).

- [ ] **Step 1: Add mode constant and fields**

In `internal/tui/ticketview.go`, add `ticketAgentSelectMode` to the const block at line 22:

```go
const (
	ticketViewMode ticketViewModeType = iota
	ticketEditMode
	ticketAgentSelectMode
)
```

Add fields to `TicketViewModel` (after `editBuffer string` at line 55):

```go
	agents      []config.DetectedAgent
	agentCursor int
```

Add import for `"github.com/ayan-de/agent-board/internal/config"` to the import block.

- [ ] **Step 2: Update constructor to accept agents**

Change `NewTicketViewModel` signature (line 196) to accept agents:

```go
func NewTicketViewModel(s *store.Store, resolver *keybinding.Resolver, t *theme.Theme, agents []config.DetectedAgent) TicketViewModel {
	return TicketViewModel{
		store:    s,
		resolver: resolver,
		styles:   NewTicketViewStyles(t),
		fields:   ticketFields(),
		mode:     ticketViewMode,
		agents:   agents,
	}
}
```

- [ ] **Step 3: Update `handleKey` to dispatch agent select mode**

Replace the `handleKey` method (line 222):

```go
func (m TicketViewModel) handleKey(msg tea.KeyMsg) (TicketViewModel, tea.Cmd) {
	if m.mode == ticketEditMode {
		return m.handleEditKey(msg)
	}
	if m.mode == ticketAgentSelectMode {
		return m.handleAgentSelectKey(msg)
	}
	return m.handleViewKey(msg)
}
```

- [ ] **Step 4: Add `a` key trigger in `handleViewKey`**

In `handleViewKey`, add a `case "a":` to the `switch key` block (after the `case "s":` block at line 253):

```go
	case "a":
		if m.ticket != nil {
			m.mode = ticketAgentSelectMode
			m.agentCursor = 0
		}
```

- [ ] **Step 5: Add `handleAgentSelectKey` method**

Add new method after `handleEditKey` (after line 289):

```go
func (m TicketViewModel) handleAgentSelectKey(msg tea.KeyMsg) (TicketViewModel, tea.Cmd) {
	total := len(m.agents) + 1

	switch msg.Type {
	case tea.KeyEscape:
		m.mode = ticketViewMode
		return m, nil
	case tea.KeyEnter:
		if m.ticket != nil {
			if m.agentCursor == 0 {
				m.ticket.Agent = ""
			} else {
				idx := m.agentCursor - 1
				if idx < len(m.agents) {
					m.ticket.Agent = m.agents[idx].Name
				}
			}
			_, _ = m.store.UpdateTicket(context.Background(), *m.ticket)
		}
		m.mode = ticketViewMode
		return m, nil
	case tea.KeyUp:
		if m.agentCursor > 0 {
			m.agentCursor--
		}
		return m, nil
	case tea.KeyDown:
		if m.agentCursor < total-1 {
			m.agentCursor++
		}
		return m, nil
	}

	if msg.Type == tea.KeyRunes {
		switch string(msg.Runes) {
		case "j":
			if m.agentCursor < total-1 {
				m.agentCursor++
			}
		case "k":
			if m.agentCursor > 0 {
				m.agentCursor--
			}
		}
	}

	return m, nil
}
```

- [ ] **Step 6: Update `SetTicket` to reset agent mode**

In `SetTicket` (line 308), no change needed — it already resets `mode = ticketViewMode`.

- [ ] **Step 7: Update all callers of `NewTicketViewModel`**

In `internal/tui/app.go` line 81, pass agents:

```go
		ticketView: NewTicketViewModel(s, resolver, t, agents),
```

In `internal/tui/ticketview_test.go` line 37, update `newTestTicketView` to pass test agents:

```go
	testAgents := []config.DetectedAgent{
		{Name: "claude-code", LogoClr: "#D97757"},
		{Name: "opencode", LogoClr: "#808080"},
		{Name: "codex", LogoClr: "#10A37F"},
		{Name: "cursor", LogoClr: "#F0DB4F"},
	}
	m := NewTicketViewModel(s, resolver, defaultTheme, testAgents)
```

Add `"github.com/ayan-de/agent-board/internal/config"` to test file imports.

- [ ] **Step 8: Run existing tests**

Run: `go test ./internal/tui/ -v`
Expected: All existing tests pass (constructor signature updated everywhere)

- [ ] **Step 9: Write tests for agent select mode**

Add to `internal/tui/ticketview_test.go`:

```go
func TestTicketViewModelAgentSelectMode(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Agent Test",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if m.mode != ticketAgentSelectMode {
		t.Fatalf("mode = %v, want ticketAgentSelectMode", m.mode)
	}
	if m.agentCursor != 0 {
		t.Errorf("agentCursor = %d, want 0", m.agentCursor)
	}
}

func TestTicketViewModelAgentSelectNavigate(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Agent Nav",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	downKey := tea.KeyMsg{Type: tea.KeyDown}
	m, _ = m.Update(downKey)
	if m.agentCursor != 1 {
		t.Errorf("agentCursor = %d after down, want 1", m.agentCursor)
	}

	upKey := tea.KeyMsg{Type: tea.KeyUp}
	m, _ = m.Update(upKey)
	if m.agentCursor != 0 {
		t.Errorf("agentCursor = %d after up, want 0", m.agentCursor)
	}

	for i := 0; i < 10; i++ {
		m, _ = m.Update(upKey)
	}
	if m.agentCursor != 0 {
		t.Errorf("agentCursor = %d after underflow, want 0", m.agentCursor)
	}
}

func TestTicketViewModelAgentSelectAssign(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Assign Me",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.ticket.Agent != "claude-code" {
		t.Errorf("agent = %q, want %q", m.ticket.Agent, "claude-code")
	}
	if m.mode != ticketViewMode {
		t.Errorf("mode = %v after select, want ticketViewMode", m.mode)
	}

	loaded, _ := s.GetTicket(ctx, ticket.ID)
	if loaded.Agent != "claude-code" {
		t.Errorf("persisted agent = %q, want %q", loaded.Agent, "claude-code")
	}
}

func TestTicketViewModelAgentSelectNone(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Unassign Me",
		Status: "backlog",
		Agent:  "opencode",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.ticket.Agent != "" {
		t.Errorf("agent = %q after None, want empty", m.ticket.Agent)
	}

	loaded, _ := s.GetTicket(ctx, ticket.ID)
	if loaded.Agent != "" {
		t.Errorf("persisted agent = %q, want empty", loaded.Agent)
	}
}

func TestTicketViewModelAgentSelectCancel(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Cancel Agent",
		Status: "backlog",
		Agent:  "opencode",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if m.ticket.Agent != "opencode" {
		t.Errorf("agent = %q after cancel, want %q", m.ticket.Agent, "opencode")
	}
	if m.mode != ticketViewMode {
		t.Errorf("mode = %v after cancel, want ticketViewMode", m.mode)
	}
}
```

- [ ] **Step 10: Run all ticket view tests**

Run: `go test ./internal/tui/ -run TestTicketView -v`
Expected: All PASS

- [ ] **Step 11: Commit**

```bash
git add internal/tui/ticketview.go internal/tui/ticketview_test.go internal/tui/app.go
git commit -m "feat(tui): add agent select mode to ticket view with dropdown logic"
```

---

### Task 3: Render agent dropdown in ticket view

**Files:**
- Modify: `internal/tui/ticketview.go` (View method, line 316-376)
- Test: `internal/tui/ticketview_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/tui/ticketview_test.go`:

```go
func TestTicketViewModelAgentSelectDropdownRenders(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Dropdown Render",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	view := m.View()
	if !strings.Contains(view, "None") {
		t.Error("dropdown missing 'None' option")
	}
	if !strings.Contains(view, "claude-code") {
		t.Error("dropdown missing 'claude-code' agent")
	}
	if !strings.Contains(view, "opencode") {
		t.Error("dropdown missing 'opencode' agent")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestTicketViewModelAgentSelectDropdownRenders -v`
Expected: FAIL — dropdown not rendered yet (view won't contain "None" or agent names in select mode)

- [ ] **Step 3: Add dropdown rendering to View method**

In `internal/tui/ticketview.go`, inside the `View()` method, add a block after the `ticketEditMode` block (after line 369, before `b.WriteString("\n")` and footer):

```go
	if m.mode == ticketAgentSelectMode {
		b.WriteString("\n")
		b.WriteString(m.styles.Label.Render("Select Agent:"))
		b.WriteString("\n")

		items := make([]string, 0, len(m.agents)+1)
		items = append(items, "None")
		for _, ag := range m.agents {
			items = append(items, ag.Name)
		}

		for i, item := range items {
			prefix := "  "
			if i == m.agentCursor {
				prefix = "▸ "
			}

			row := prefix + item
			if i == m.agentCursor {
				row = m.styles.SelectedRow.Width(innerWidth - 2).Render(row)
			}
			b.WriteString(row)
			b.WriteString("\n")
		}
	}
```

- [ ] **Step 4: Update footer to show context-sensitive hints**

Change the footer line (currently `"e: edit │ s: cycle status │ Esc: back"`) to be mode-aware. Replace lines 372-373:

```go
	var footer string
	if m.mode == ticketAgentSelectMode {
		footer = "↑/k: up │ ↓/j: down │ Enter: select │ Esc: cancel"
	} else {
		footer = "e: edit │ s: cycle status │ a: assign agent │ Esc: back"
	}
	b.WriteString(m.styles.Footer.Render(footer))
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/tui/ -run TestTicketView -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add internal/tui/ticketview.go internal/tui/ticketview_test.go
git commit -m "feat(tui): render agent selection dropdown in ticket view"
```

---

### Task 4: Show agent color dot on kanban tickets

**Files:**
- Modify: `internal/tui/kanban.go` (ticket rendering, line 228)
- Test: `internal/tui/kanban_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/tui/kanban_test.go`:

```go
func TestKanbanViewAgentDot(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()
	m.width = 120
	m.height = 40

	_, _ = m.store.CreateTicket(ctx, store.Ticket{
		Title:  "Agent Dot Test",
		Status: "backlog",
		Agent:  "claude-code",
	})
	m, _ = m.Reload()

	view := m.View()
	if !strings.Contains(view, "Agent Dot Test") {
		t.Fatal("view missing ticket title")
	}
	if !strings.Contains(view, "●") {
		t.Error("view missing agent dot '●' for assigned ticket")
	}
}

func TestKanbanViewNoAgentDot(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()
	m.width = 120
	m.height = 40

	_, _ = m.store.CreateTicket(ctx, store.Ticket{
		Title:  "No Agent",
		Status: "backlog",
	})
	m, _ = m.Reload()

	view := m.View()
	if strings.Contains(view, "●") {
		t.Error("view should not contain agent dot '●' for unassigned ticket")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestKanbanViewAgent -v`
Expected: FAIL — no `●` in view

- [ ] **Step 3: Add agent dot rendering to kanban**

In `internal/tui/kanban.go`, add import for `"github.com/ayan-de/agent-board/internal/config"`.

In the `View()` method, change line 228 from:

```go
			line := prefix + ticket.ID + " " + ticket.Title
```

to:

```go
			line := prefix + ticket.ID + " " + ticket.Title
			if ticket.Agent != "" {
				color := config.AgentColor(ticket.Agent)
				if color != "" {
					dot := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("●")
					line = line + " " + dot
				}
			}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/tui/ -run TestKanbanViewAgent -v`
Expected: PASS

- [ ] **Step 5: Run full test suite**

Run: `go test ./...`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add internal/tui/kanban.go internal/tui/kanban_test.go
git commit -m "feat(tui): show colored agent dot on kanban tickets"
```

---

### Task 5: Handle escape from agent select mode in app.go

**Files:**
- Modify: `internal/tui/app.go` (line 157-166)

- [ ] **Step 1: Write the failing test**

Add to `internal/tui/app_test.go`:

```go
func TestAppEscapeFromAgentSelectReturnsToTicketView(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()

	app.store.CreateTicket(ctx, store.Ticket{Title: "Agent Esc", Status: "backlog"})
	app.kanban, _ = app.kanban.Reload()

	app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if app.view != viewTicket {
		t.Fatalf("view = %v, want viewTicket", app.view)
	}

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if app.ticketView.mode != ticketAgentSelectMode {
		t.Fatalf("mode = %v, want ticketAgentSelectMode", app.ticketView.mode)
	}

	app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if app.view != viewTicket {
		t.Errorf("view = %v after first esc, want viewTicket", app.view)
	}
	if app.ticketView.mode != ticketViewMode {
		t.Errorf("mode = %v after first esc, want ticketViewMode", app.ticketView.mode)
	}

	app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if app.view != viewBoard {
		t.Errorf("view = %v after second esc, want viewBoard", app.view)
	}
}
```

- [ ] **Step 2: Update escape handling in app.go**

In `internal/tui/app.go`, update the `key == "esc"` block (lines 157-167) to also handle agent select mode:

```go
	if key == "esc" {
		if a.view == viewTicket && (a.ticketView.mode == ticketEditMode || a.ticketView.mode == ticketAgentSelectMode) {
			a.ticketView, _ = a.ticketView.Update(msg)
			return a, nil
		}
		if a.view != viewBoard {
			a.view = viewBoard
			a.activeTicket = nil
			return a, nil
		}
	}
```

- [ ] **Step 3: Run all tests**

Run: `go test ./... -v`
Expected: All PASS

- [ ] **Step 4: Run go vet**

Run: `go vet ./...`
Expected: No issues

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "feat(tui): handle escape from agent select mode in app routing"
```
