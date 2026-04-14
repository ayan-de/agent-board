# tui/app.go Root Bubbletea Model Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the root bubbletea model with window size tracking, keybinding action routing, ticket CRUD via store, and minimal board rendering.

**Architecture:** Flat `App` struct implementing `bubbletea.Model`. Holds `*store.Store`, `*keybinding.Resolver`, `*config.Config`, window dimensions, board navigation state, and ticket data. Actions from the keybinding Resolver are dispatched inline in `Update()`. Sub-models extract naturally in phases 1.5/1.6.

**Tech Stack:** Go, bubbletea v1.3.x, existing `keybinding` + `store` + `config` packages.

---

## File Structure

| File | Responsibility |
|------|---------------|
| `go.mod` | Add `github.com/charmbracelet/bubbletea` dependency |
| `internal/tui/app.go` | App struct, constructor, Init/Update/View, loadColumns |
| `internal/tui/app_test.go` | All tests for the root model |

---

### Task 1: Add bubbletea dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Run go get**

Run: `go get github.com/charmbracelet/bubbletea@latest`
Expected: go.mod updated, go.sum updated

- [ ] **Step 2: Verify dependency**

Run: `go list -m github.com/charmbracelet/bubbletea`
Expected: prints version like `github.com/charmbracelet/bubbletea v1.3.10`

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add bubbletea dependency"
```

---

### Task 2: App struct, constructor, and types

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Write the App struct and constructor**

Replace the contents of `internal/tui/app.go` with:

```go
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"

	tea "github.com/charmbracelet/bubbletea"
)

type focusArea int

const (
	focusBoard focusArea = iota
	focusAgentPane
)

type viewMode int

const (
	viewBoard viewMode = iota
	viewTicket
	viewHelp
)

var columnNames = [4]string{"Backlog", "In Progress", "Review", "Done"}

type App struct {
	store    *store.Store
	resolver *keybinding.Resolver
	config   *config.Config
	width    int
	height   int

	focus focusArea
	view  viewMode

	colIndex int
	cursors  [4]int
	columns  [4][]store.Ticket

	activeTicket *store.Ticket
}

func NewApp(cfg *config.Config, s *store.Store) (*App, error) {
	km := keybinding.DefaultKeyMap()
	if len(cfg.TUI.Keybindings) > 0 {
		keybinding.ApplyConfig(&km, cfg.TUI.Keybindings)
	}

	a := &App{
		store:    s,
		resolver: keybinding.NewResolver(km),
		config:   cfg,
		focus:    focusBoard,
		view:     viewBoard,
	}

	if err := a.loadColumns(); err != nil {
		return nil, fmt.Errorf("tui.newApp: %w", err)
	}

	return a, nil
}

func (a *App) loadColumns() error {
	statuses := [4]string{"backlog", "in_progress", "review", "done"}
	for i, status := range statuses {
		tickets, err := a.store.ListTickets(context.Background(), store.TicketFilters{Status: status})
		if err != nil {
			return fmt.Errorf("tui.loadColumns: %w", err)
		}
		a.columns[i] = tickets
	}
	for i := range a.cursors {
		if a.cursors[i] >= len(a.columns[i]) && len(a.columns[i]) > 0 {
			a.cursors[i] = len(a.columns[i]) - 1
		}
	}
	return nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/tui/`
Expected: success, no errors

- [ ] **Step 3: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat(tui): add App struct, constructor, and loadColumns"
```

---

### Task 3: Tests for constructor

**Files:**
- Create: `internal/tui/app_test.go`

- [ ] **Step 1: Write the test helper and constructor tests**

Create `internal/tui/app_test.go`:

```go
package tui

import (
	"path/filepath"
	"testing"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/store"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.Open(dbPath, []string{"backlog", "in_progress", "review", "done"})
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	cfg := config.SetDefaults()
	app, err := NewApp(cfg, s)
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	return app
}

func TestNewApp(t *testing.T) {
	app := newTestApp(t)

	if app == nil {
		t.Fatal("NewApp should return non-nil")
	}
	if app.resolver == nil {
		t.Fatal("resolver should be initialized")
	}
	if app.store == nil {
		t.Fatal("store should be set")
	}
	if app.view != viewBoard {
		t.Errorf("view = %d, want %d", app.view, viewBoard)
	}
	if app.focus != focusBoard {
		t.Errorf("focus = %d, want %d", app.focus, focusBoard)
	}
	if app.colIndex != 0 {
		t.Errorf("colIndex = %d, want 0", app.colIndex)
	}
}

func TestNewAppWithKeybindingOverrides(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.Open(dbPath, []string{"backlog", "in_progress", "review", "done"})
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer s.Close()

	cfg := config.SetDefaults()
	cfg.TUI.Keybindings = map[string]string{
		"quit": "Q",
	}

	app, err := NewApp(cfg, s)
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	action, _ := app.resolver.Resolve("Q")
	if action != keybinding.ActionQuit {
		t.Errorf("overridden key 'Q' resolved to %v, want ActionQuit", action)
	}
}

func TestNewAppLoadsColumns(t *testing.T) {
	app := newTestApp(t)

	for i, col := range app.columns {
		if col == nil {
			t.Errorf("columns[%d] is nil, want empty slice", i)
		}
	}
}
```

- [ ] **Step 2: Add missing import in app_test.go**

The test file uses `keybinding` package — add the import:

```go
import (
	"path/filepath"
	"testing"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"
)
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v -run TestNewApp`
Expected: all 3 tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/tui/app_test.go
git commit -m "test(tui): add constructor tests for App"
```

---

### Task 4: Init method

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/tui/app_test.go`:

```go
func TestInit(t *testing.T) {
	app := newTestApp(t)
	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init should return a non-nil command")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -v -run TestInit`
Expected: FAIL — `Init` not defined on `App`

- [ ] **Step 3: Write minimal implementation**

Add to `internal/tui/app.go`:

```go
func (a *App) Init() tea.Cmd {
	return tea.EnterAltScreen
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -v -run TestInit`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "feat(tui): add Init method"
```

---

### Task 5: Update — WindowSizeMsg

**Files:**
- Modify: `internal/tui/app.go`, `internal/tui/app_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/tui/app_test.go`:

```go
func TestUpdateWindowSize(t *testing.T) {
	app := newTestApp(t)

	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	u := updated.(*App)

	if u.width != 120 {
		t.Errorf("width = %d, want 120", u.width)
	}
	if u.height != 40 {
		t.Errorf("height = %d, want 40", u.height)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -v -run TestUpdateWindowSize`
Expected: FAIL — `Update` not defined

- [ ] **Step 3: Write minimal implementation**

Add to `internal/tui/app.go`:

```go
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	}
	return a, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -v -run TestUpdateWindowSize`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "feat(tui): handle WindowSizeMsg in Update"
```

---

### Task 6: Update — Quit and ForceQuit

**Files:**
- Modify: `internal/tui/app.go`, `internal/tui/app_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/tui/app_test.go`:

```go
func TestUpdateQuit(t *testing.T) {
	app := newTestApp(t)

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("quit should return a command")
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("quit command should produce tea.QuitMsg, got %T", msg)
	}
}

func TestUpdateForceQuit(t *testing.T) {
	app := newTestApp(t)

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("force quit should return a command")
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("force quit command should produce tea.QuitMsg, got %T", msg)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -v -run "TestUpdateQuit|TestUpdateForceQuit"`
Expected: FAIL — quit actions not handled

- [ ] **Step 3: Write implementation**

Replace the `Update` method in `internal/tui/app.go` with:

```go
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	case tea.KeyMsg:
		return a.handleKey(msg)
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	action, _ := a.resolver.Resolve(key)

	switch action {
	case keybinding.ActionQuit, keybinding.ActionForceQuit:
		return a, tea.Quit
	}

	return a, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v -run "TestUpdateQuit|TestUpdateForceQuit"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "feat(tui): handle quit and force quit key actions"
```

---

### Task 7: Update — Navigation actions

**Files:**
- Modify: `internal/tui/app.go`, `internal/tui/app_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/tui/app_test.go`:

```go
func TestUpdateNavigationPrevNextColumn(t *testing.T) {
	app := newTestApp(t)

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if app.colIndex != 1 {
		t.Errorf("after 'l', colIndex = %d, want 1", app.colIndex)
	}

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if app.colIndex != 2 {
		t.Errorf("after second 'l', colIndex = %d, want 2", app.colIndex)
	}

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if app.colIndex != 3 {
		t.Errorf("colIndex should clamp at 3, got %d", app.colIndex)
	}

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if app.colIndex != 2 {
		t.Errorf("after 'h', colIndex = %d, want 2", app.colIndex)
	}

	for i := 0; i < 5; i++ {
		_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	}
	if app.colIndex != 0 {
		t.Errorf("colIndex should clamp at 0, got %d", app.colIndex)
	}
}

func TestUpdateNavigationJumpColumns(t *testing.T) {
	app := newTestApp(t)

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	if app.colIndex != 2 {
		t.Errorf("after '3', colIndex = %d, want 2", app.colIndex)
	}

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if app.colIndex != 0 {
		t.Errorf("after '1', colIndex = %d, want 0", app.colIndex)
	}

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	if app.colIndex != 3 {
		t.Errorf("after '4', colIndex = %d, want 3", app.colIndex)
	}
}

func TestUpdateNavigationPrevNextTicket(t *testing.T) {
	app := newTestApp(t)

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, _ = app.store.CreateTicket(ctx, store.Ticket{
			Title:  fmt.Sprintf("Ticket %d", i+1),
			Status: "backlog",
		})
	}
	_ = app.loadColumns()

	if len(app.columns[0]) != 3 {
		t.Fatalf("backlog column has %d tickets, want 3", len(app.columns[0]))
	}

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if app.cursors[0] != 1 {
		t.Errorf("after 'j', cursor = %d, want 1", app.cursors[0])
	}

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if app.cursors[0] != 2 {
		t.Errorf("after second 'j', cursor = %d, want 2", app.cursors[0])
	}

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if app.cursors[0] != 2 {
		t.Errorf("cursor should clamp at 2, got %d", app.cursors[0])
	}

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if app.cursors[0] != 1 {
		t.Errorf("after 'k', cursor = %d, want 1", app.cursors[0])
	}

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if app.cursors[0] != 0 {
		t.Errorf("cursor should clamp at 0, got %d", app.cursors[0])
	}
}

func TestUpdateNavigationEmptyColumn(t *testing.T) {
	app := newTestApp(t)

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if app.cursors[1] != 0 {
		t.Errorf("cursor in empty column should be 0, got %d", app.cursors[1])
	}

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if app.cursors[1] != 0 {
		t.Errorf("cursor should stay 0 in empty column, got %d", app.cursors[1])
	}
}
```

Add these imports to the test file (merge with existing imports):
```go
"context"
"fmt"
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -v -run "TestUpdateNavigation"`
Expected: FAIL — navigation actions not handled

- [ ] **Step 3: Write implementation**

Replace `handleKey` in `internal/tui/app.go` with:

```go
func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	action, _ := a.resolver.Resolve(key)

	switch action {
	case keybinding.ActionQuit, keybinding.ActionForceQuit:
		return a, tea.Quit
	case keybinding.ActionPrevColumn:
		if a.colIndex > 0 {
			a.colIndex--
		}
	case keybinding.ActionNextColumn:
		if a.colIndex < 3 {
			a.colIndex++
		}
	case keybinding.ActionPrevTicket:
		col := a.columns[a.colIndex]
		if len(col) > 0 && a.cursors[a.colIndex] > 0 {
			a.cursors[a.colIndex]--
		}
	case keybinding.ActionNextTicket:
		col := a.columns[a.colIndex]
		if len(col) > 0 && a.cursors[a.colIndex] < len(col)-1 {
			a.cursors[a.colIndex]++
		}
	case keybinding.ActionJumpColumn1:
		a.colIndex = 0
	case keybinding.ActionJumpColumn2:
		a.colIndex = 1
	case keybinding.ActionJumpColumn3:
		a.colIndex = 2
	case keybinding.ActionJumpColumn4:
		a.colIndex = 3
	}

	return a, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v -run "TestUpdateNavigation"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "feat(tui): handle column and ticket navigation actions"
```

---

### Task 8: Update — AddTicket and DeleteTicket

**Files:**
- Modify: `internal/tui/app.go`, `internal/tui/app_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/tui/app_test.go`:

```go
func TestUpdateAddTicket(t *testing.T) {
	app := newTestApp(t)

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if len(app.columns[0]) != 1 {
		t.Fatalf("backlog should have 1 ticket after add, got %d", len(app.columns[0]))
	}

	created := app.columns[0][0]
	if created.Title != "New Ticket" {
		t.Errorf("title = %q, want %q", created.Title, "New Ticket")
	}
	if created.Status != "backlog" {
		t.Errorf("status = %q, want %q", created.Status, "backlog")
	}
}

func TestUpdateDeleteTicket(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()

	_, _ = app.store.CreateTicket(ctx, store.Ticket{Title: "To delete", Status: "backlog"})
	_ = app.loadColumns()

	if len(app.columns[0]) != 1 {
		t.Fatalf("precondition: backlog should have 1 ticket, got %d", len(app.columns[0]))
	}

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	if len(app.columns[0]) != 0 {
		t.Errorf("backlog should be empty after delete, got %d tickets", len(app.columns[0]))
	}
}

func TestUpdateDeleteTicketEmptyColumn(t *testing.T) {
	app := newTestApp(t)

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	if len(app.columns[0]) != 0 {
		t.Error("delete on empty column should be a no-op")
	}
}
```

Add `"context"` import if not already present.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -v -run "TestUpdateAddTicket|TestUpdateDeleteTicket"`
Expected: FAIL — add/delete not handled

- [ ] **Step 3: Write implementation**

Add to the `handleKey` switch in `internal/tui/app.go`, before the closing `}` of the switch:

```go
	case keybinding.ActionAddTicket:
		_, err := a.store.CreateTicket(context.Background(), store.Ticket{
			Title:  "New Ticket",
			Status: "backlog",
		})
		if err != nil {
			return a, nil
		}
		_ = a.loadColumns()
	case keybinding.ActionDeleteTicket:
		col := a.columns[a.colIndex]
		if len(col) > 0 {
			cursor := a.cursors[a.colIndex]
			_ = a.store.DeleteTicket(context.Background(), col[cursor].ID)
			_ = a.loadColumns()
		}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v -run "TestUpdateAddTicket|TestUpdateDeleteTicket"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "feat(tui): handle add and delete ticket actions"
```

---

### Task 9: Update — OpenTicket and ShowHelp

**Files:**
- Modify: `internal/tui/app.go`, `internal/tui/app_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/tui/app_test.go`:

```go
func TestUpdateOpenTicket(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()

	created, _ := app.store.CreateTicket(ctx, store.Ticket{
		Title:       "View me",
		Description: "Some details",
		Status:      "backlog",
	})
	_ = app.loadColumns()

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if app.view != viewTicket {
		t.Errorf("view = %d, want viewTicket", app.view)
	}
	if app.activeTicket == nil {
		t.Fatal("activeTicket should be set")
	}
	if app.activeTicket.ID != created.ID {
		t.Errorf("activeTicket.ID = %q, want %q", app.activeTicket.ID, created.ID)
	}
}

func TestUpdateOpenTicketEmptyColumn(t *testing.T) {
	app := newTestApp(t)

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if app.view != viewBoard {
		t.Error("open ticket on empty column should stay in board view")
	}
}

func TestUpdateShowHelp(t *testing.T) {
	app := newTestApp(t)

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if app.view != viewHelp {
		t.Errorf("view = %d, want viewHelp after '?'", app.view)
	}

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if app.view != viewBoard {
		t.Errorf("view = %d, want viewBoard after second '?'", app.view)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -v -run "TestUpdateOpenTicket|TestUpdateShowHelp"`
Expected: FAIL — open/help not handled

- [ ] **Step 3: Write implementation**

Add to the `handleKey` switch in `internal/tui/app.go`:

```go
	case keybinding.ActionOpenTicket:
		col := a.columns[a.colIndex]
		if len(col) > 0 {
			ticket := col[a.cursors[a.colIndex]]
			a.activeTicket = &ticket
			a.view = viewTicket
		}
	case keybinding.ActionShowHelp:
		if a.view == viewHelp {
			a.view = viewBoard
		} else {
			a.view = viewHelp
		}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v -run "TestUpdateOpenTicket|TestUpdateShowHelp"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "feat(tui): handle open ticket and help toggle actions"
```

---

### Task 10: Escape key to return to board view

**Files:**
- Modify: `internal/tui/app.go`, `internal/tui/app_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/tui/app_test.go`:

```go
func TestUpdateEscapeReturnsToBoard(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()

	_, _ = app.store.CreateTicket(ctx, store.Ticket{Title: "T", Status: "backlog"})
	_ = app.loadColumns()

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if app.view != viewTicket {
		t.Fatal("precondition: should be in ticket view")
	}

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if app.view != viewBoard {
		t.Errorf("after Escape, view = %d, want viewBoard", app.view)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -v -run TestUpdateEscapeReturnsToBoard`
Expected: FAIL — escape not handled

- [ ] **Step 3: Write implementation**

Add an `esc` key check at the top of `handleKey`, before the resolver:

```go
func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" && a.view != viewBoard {
		a.view = viewBoard
		a.activeTicket = nil
		return a, nil
	}

	key := msg.String()
	action, _ := a.resolver.Resolve(key)
	// ... rest of switch
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -v -run TestUpdateEscapeReturnsToBoard`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "feat(tui): handle escape to return to board view"
```

---

### Task 11: View — Board rendering

**Files:**
- Modify: `internal/tui/app.go`, `internal/tui/app_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/tui/app_test.go`:

```go
func TestViewBoardRendersColumnNames(t *testing.T) {
	app := newTestApp(t)
	app.width = 120
	app.height = 40

	out := app.View()

	for _, name := range columnNames {
		if !strings.Contains(out, name) {
			t.Errorf("View output should contain %q", name)
		}
	}
}

func TestViewBoardRendersTickets(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()

	_, _ = app.store.CreateTicket(ctx, store.Ticket{Title: "Auth system", Status: "backlog"})
	_, _ = app.store.CreateTicket(ctx, store.Ticket{Title: "API layer", Status: "in_progress"})
	_ = app.loadColumns()
	app.width = 120
	app.height = 40

	out := app.View()

	if !strings.Contains(out, "Auth system") {
		t.Error("View output should contain ticket title 'Auth system'")
	}
	if !strings.Contains(out, "API layer") {
		t.Error("View output should contain ticket title 'API layer'")
	}
	if !strings.Contains(out, "AGT-01") {
		t.Error("View output should contain ticket ID 'AGT-01'")
	}
}

func TestViewTicketDetail(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()

	_, _ = app.store.CreateTicket(ctx, store.Ticket{
		Title:       "Detail view test",
		Description: "Full description here",
		Status:      "backlog",
	})
	_ = app.loadColumns()
	app.width = 120
	app.height = 40

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	out := app.View()

	if !strings.Contains(out, "Detail view test") {
		t.Error("ticket view should contain ticket title")
	}
	if !strings.Contains(out, "Full description here") {
		t.Error("ticket view should contain description")
	}
}

func TestViewHelp(t *testing.T) {
	app := newTestApp(t)
	app.width = 120
	app.height = 40
	app.view = viewHelp

	out := app.View()

	if !strings.Contains(out, "Help") {
		t.Error("help view should contain 'Help'")
	}
	if !strings.Contains(out, "quit") {
		t.Error("help view should list keybinding 'quit'")
	}
}
```

Add `"strings"` import if not already present in test file.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -v -run "TestViewBoard|TestViewTicket|TestViewHelp"`
Expected: FAIL — `View` not defined

- [ ] **Step 3: Write implementation**

Add to `internal/tui/app.go`:

```go
func (a *App) View() string {
	switch a.view {
	case viewHelp:
		return a.renderHelp()
	case viewTicket:
		return a.renderTicket()
	default:
		return a.renderBoard()
	}
}

func (a *App) renderBoard() string {
	var b strings.Builder

	b.WriteString("AgentBoard")
	if a.width > 0 {
		b.WriteString(fmt.Sprintf("  [%dx%d]", a.width, a.height))
	}
	b.WriteString("\n\n")

	colWidth := a.width / 4
	if colWidth < 20 {
		colWidth = 20
	}

	for i, name := range columnNames {
		if i == a.colIndex {
			b.WriteString(fmt.Sprintf("▶ %s", name))
		} else {
			b.WriteString(fmt.Sprintf("  %s", name))
		}
		if i < 3 {
			pad := colWidth - len(name) - 2
			if pad > 0 {
				b.WriteString(strings.Repeat(" ", pad))
			}
		}
	}
	b.WriteString("\n")

	for i := 0; i < 4; i++ {
		cursor := a.cursors[i]
		for j, ticket := range a.columns[i] {
			prefix := "  "
			if i == a.colIndex && j == cursor {
				prefix = "▸ "
			}
			line := fmt.Sprintf("%s%s %s", prefix, ticket.ID, ticket.Title)
			if len(line) > colWidth {
				line = line[:colWidth-1] + "…"
			}
			b.WriteString(line)
			if i < 3 {
				pad := colWidth - len(line)
				if pad > 0 {
					b.WriteString(strings.Repeat(" ", pad))
				}
			}
			b.WriteString("\n")
		}
		if len(a.columns[i]) == 0 {
			b.WriteString("  (empty)")
			if i < 3 {
				pad := colWidth - 9
				if pad > 0 {
					b.WriteString(strings.Repeat(" ", pad))
				}
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (a *App) renderTicket() string {
	if a.activeTicket == nil {
		return "No ticket selected"
	}
	t := a.activeTicket
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Ticket: %s\n", t.ID))
	b.WriteString(fmt.Sprintf("Title:  %s\n", t.Title))
	b.WriteString(fmt.Sprintf("Status: %s\n", t.Status))
	if t.Description != "" {
		b.WriteString(fmt.Sprintf("\n%s\n", t.Description))
	}
	b.WriteString("\nPress Esc to return")
	return b.String()
}

func (a *App) renderHelp() string {
	var b strings.Builder
	b.WriteString("Help — Keybindings\n\n")
	km := keybinding.DefaultKeyMap()
	for _, binding := range km.Bindings {
		b.WriteString(fmt.Sprintf("  %-12s %s\n", binding.Key, binding.Action.String()))
	}
	b.WriteString("\nPress ? to return")
	return b.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v -run "TestViewBoard|TestViewTicket|TestViewHelp"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "feat(tui): implement View with board, ticket detail, and help rendering"
```

---

### Task 12: Run full test suite and vet

**Files:** None — verification only

- [ ] **Step 1: Run all tests**

Run: `go test ./... -v`
Expected: all packages PASS

- [ ] **Step 2: Run go vet**

Run: `go vet ./...`
Expected: no issues

- [ ] **Step 3: Commit if any fixes needed, otherwise done**

If all clean, Phase 1.4 is complete.
