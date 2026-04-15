# KanbanModel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract the 4-column Kanban board from `app.go` into a standalone `KanbanModel` in `kanban.go` with lipgloss-styled bordered columns.

**Architecture:** KanbanModel implements `tea.Model` (Init/Update/View). App holds a `KanbanModel` field and delegates window-size and key messages to it. All column state moves from App to KanbanModel. Rendering uses lipgloss bordered boxes with focused/blurred styling.

**Tech Stack:** Go 1.26, bubbletea v1.3.10, lipgloss v1.1.0, existing store/keybinding packages.

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/tui/kanban.go` | KanbanModel struct, KanbanStyles, DefaultKanbanStyles, Init/Update/View, SelectedTicket, Reload, loadColumns, renderColumn |
| `internal/tui/kanban_test.go` | New file — all kanban-specific tests |
| `internal/tui/app.go` | Remove columns/colIndex/cursors/loadColumns/renderBoard, add kanban field, delegate to KanbanModel |
| `internal/tui/app_test.go` | Remove kanban-specific tests, keep app-level tests |
| `go.mod` | Upgrade lipgloss from indirect to direct dependency |

---

### Task 1: Add lipgloss as direct dependency

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Tidy go.mod to promote lipgloss**

Run:
```bash
go get github.com/charmbracelet/lipgloss@v1.1.0
go mod tidy
```

- [ ] **Step 2: Verify lipgloss is now a direct dependency**

Run: `grep lipgloss go.mod`
Expected: line shows `github.com/charmbracelet/lipgloss v1.1.0` as a `require` (not `// indirect`)

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: promote lipgloss to direct dependency"
```

---

### Task 2: Write kanban_test.go — constructor and basic tests

**Files:**
- Create: `internal/tui/kanban_test.go`

- [ ] **Step 1: Write the test helper and first tests**

Create `internal/tui/kanban_test.go`:

```go
package tui

import (
	"path/filepath"
	"testing"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"
)

func newTestKanban(t *testing.T) KanbanModel {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.Open(dbPath, []string{"backlog", "in_progress", "review", "done"})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	km := keybinding.DefaultKeyMap()
	resolver := keybinding.NewResolver(km)
	kanban, err := NewKanbanModel(s, resolver)
	if err != nil {
		t.Fatalf("new kanban: %v", err)
	}
	return kanban
}

func TestNewKanbanModel(t *testing.T) {
	m := newTestKanban(t)

	if m.store == nil {
		t.Error("store is nil")
	}
	if m.resolver == nil {
		t.Error("resolver is nil")
	}
	if m.colIndex != 0 {
		t.Errorf("colIndex = %d, want 0", m.colIndex)
	}
	for i, col := range m.columns {
		if col == nil {
			t.Errorf("columns[%d] is nil", i)
		}
	}
}

func TestKanbanInit(t *testing.T) {
	m := newTestKanban(t)
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil for KanbanModel")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run "TestNewKanbanModel|TestKanbanInit" -v`
Expected: FAIL — `NewKanbanModel` undefined

- [ ] **Step 3: Write minimal KanbanModel stub in kanban.go**

Replace contents of `internal/tui/kanban.go` with:

```go
package tui

import (
	"context"
	"fmt"

	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"

	tea "github.com/charmbracelet/bubbletea"
)

var columnNames = [4]string{"Backlog", "In Progress", "Review", "Done"}

var statusNames = [4]string{"backlog", "in_progress", "review", "done"}

type KanbanModel struct {
	store    *store.Store
	resolver *keybinding.Resolver
	width    int
	height   int

	colIndex int
	cursors  [4]int
	columns  [4][]store.Ticket

	styles KanbanStyles
}

func NewKanbanModel(s *store.Store, resolver *keybinding.Resolver) (KanbanModel, error) {
	m := KanbanModel{
		store:    s,
		resolver: resolver,
		styles:   DefaultKanbanStyles(),
	}
	if err := m.loadColumns(); err != nil {
		return KanbanModel{}, fmt.Errorf("kanban.new: %w", err)
	}
	return m, nil
}

func (m KanbanModel) Init() tea.Cmd {
	return nil
}

func (m KanbanModel) loadColumns() error {
	for i, status := range statusNames {
		tickets, err := m.store.ListTickets(context.Background(), store.TicketFilters{Status: status})
		if err != nil {
			return fmt.Errorf("kanban.loadColumns: %w", err)
		}
		if tickets == nil {
			tickets = []store.Ticket{}
		}
		m.columns[i] = tickets
	}
	for i := range m.cursors {
		if m.cursors[i] >= len(m.columns[i]) && len(m.columns[i]) > 0 {
			m.cursors[i] = len(m.columns[i]) - 1
		}
	}
	return nil
}
```

Note: `loadColumns` takes a value receiver here because `NewKanbanModel` is constructing the value. The same pattern will apply to `Reload`. The `Update` method will need a pointer receiver since it mutates state — but since this must satisfy `tea.Model`, we'll use value receivers and return the modified copy. The `loadColumns` function will need to be adjusted.

Fix — change `loadColumns` to return the modified model:

```go
func (m KanbanModel) loadColumns() (KanbanModel, error) {
	for i, status := range statusNames {
		tickets, err := m.store.ListTickets(context.Background(), store.TicketFilters{Status: status})
		if err != nil {
			return m, fmt.Errorf("kanban.loadColumns: %w", err)
		}
		if tickets == nil {
			tickets = []store.Ticket{}
		}
		m.columns[i] = tickets
	}
	for i := range m.cursors {
		if m.cursors[i] >= len(m.columns[i]) && len(m.columns[i]) > 0 {
			m.cursors[i] = len(m.columns[i]) - 1
		}
	}
	return m, nil
}
```

Update `NewKanbanModel` to match:

```go
func NewKanbanModel(s *store.Store, resolver *keybinding.Resolver) (KanbanModel, error) {
	m := KanbanModel{
		store:    s,
		resolver: resolver,
		styles:   DefaultKanbanStyles(),
	}
	var err error
	m, err = m.loadColumns()
	if err != nil {
		return KanbanModel{}, fmt.Errorf("kanban.new: %w", err)
	}
	return m, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run "TestNewKanbanModel|TestKanbanInit" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/kanban.go internal/tui/kanban_test.go
git commit -m "feat(tui): add KanbanModel struct with constructor and loadColumns"
```

---

### Task 3: Write KanbanStyles and DefaultKanbanStyles

**Files:**
- Modify: `internal/tui/kanban.go`
- Modify: `internal/tui/kanban_test.go`

- [ ] **Step 1: Write test for DefaultKanbanStyles**

Add to `internal/tui/kanban_test.go`:

```go
func TestDefaultKanbanStyles(t *testing.T) {
	s := DefaultKanbanStyles()

	empty := lipgloss.Style{}
	if s.FocusedColumn == empty {
		t.Error("FocusedColumn style is zero value")
	}
	if s.BlurredColumn == empty {
		t.Error("BlurredColumn style is zero value")
	}
	if s.FocusedTitle == empty {
		t.Error("FocusedTitle style is zero value")
	}
	if s.BlurredTitle == empty {
		t.Error("BlurredTitle style is zero value")
	}
	if s.SelectedTicket == empty {
		t.Error("SelectedTicket style is zero value")
	}
	if s.Ticket == empty {
		t.Error("Ticket style is zero value")
	}
	if s.EmptyColumn == empty {
		t.Error("EmptyColumn style is zero value")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestDefaultKanbanStyles -v`
Expected: FAIL — `DefaultKanbanStyles` undefined, `lipgloss` import missing

- [ ] **Step 3: Add lipgloss import and KanbanStyles + DefaultKanbanStyles to kanban.go**

Add to imports in `internal/tui/kanban.go`:

```go
"github.com/charmbracelet/lipgloss"
```

Add after the `statusNames` var:

```go
type KanbanStyles struct {
	FocusedColumn  lipgloss.Style
	BlurredColumn  lipgloss.Style
	FocusedTitle   lipgloss.Style
	BlurredTitle   lipgloss.Style
	SelectedTicket lipgloss.Style
	Ticket         lipgloss.Style
	EmptyColumn    lipgloss.Style
}

func DefaultKanbanStyles() KanbanStyles {
	focusedBorder := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(0, 1)

	blurredBorder := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	focusedTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("69")).
		Bold(true).
		Padding(0, 1)

	blurredTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	selected := lipgloss.NewStyle().
		Foreground(lipgloss.Color("254")).
		Bold(true)

	normal := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	empty := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	return KanbanStyles{
		FocusedColumn:  focusedBorder,
		BlurredColumn:  blurredBorder,
		FocusedTitle:   focusedTitle,
		BlurredTitle:   blurredTitle,
		SelectedTicket: selected,
		Ticket:         normal,
		EmptyColumn:    empty,
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestDefaultKanbanStyles -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/kanban.go internal/tui/kanban_test.go
git commit -m "feat(tui): add KanbanStyles with DefaultKanbanStyles"
```

---

### Task 4: Write Update method — navigation tests and implementation

**Files:**
- Modify: `internal/tui/kanban_test.go`
- Modify: `internal/tui/kanban.go`

- [ ] **Step 1: Write navigation tests**

Add to `internal/tui/kanban_test.go`:

```go
func TestKanbanWindowSize(t *testing.T) {
	m := newTestKanban(t)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	km := updated.(KanbanModel)
	if km.width != 120 {
		t.Errorf("width = %d, want 120", km.width)
	}
	if km.height != 40 {
		t.Errorf("height = %d, want 40", km.height)
	}
}

func TestKanbanColumnNavigation(t *testing.T) {
	m := newTestKanban(t)

	nextKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	for i := 0; i < 4; i++ {
		m, _ = m.Update(nextKey)
	}
	if m.(KanbanModel).colIndex != 3 {
		t.Errorf("colIndex = %d after 4 next, want 3", m.(KanbanModel).colIndex)
	}

	prevKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	km := m.(KanbanModel)
	for i := 0; i < 5; i++ {
		km, _ = km.Update(prevKey)
	}
	if km.colIndex != 0 {
		t.Errorf("colIndex = %d after 5 prev, want 0", km.colIndex)
	}
}

func TestKanbanJumpColumns(t *testing.T) {
	m := newTestKanban(t)

	tests := []struct {
		key  rune
		want int
	}{
		{'3', 2},
		{'1', 0},
		{'4', 3},
	}
	for _, tt := range tests {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tt.key}})
		km := updated.(KanbanModel)
		if km.colIndex != tt.want {
			t.Errorf("after pressing %c, colIndex = %d, want %d", tt.key, km.colIndex, tt.want)
		}
		m = km
	}
}

func TestKanbanTicketNavigation(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		m.store.CreateTicket(ctx, store.Ticket{
			Title:  fmt.Sprintf("Ticket %d", i+1),
			Status: "backlog",
		})
	}
	var err error
	m, err = m.Reload()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	nextKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ := m.Update(nextKey)
	km := updated.(KanbanModel)
	if km.cursors[0] != 1 {
		t.Errorf("cursor = %d, want 1", km.cursors[0])
	}
	m = km

	m, _ = m.Update(nextKey)
	if m.(KanbanModel).cursors[0] != 2 {
		t.Errorf("cursor = %d, want 2", m.(KanbanModel).cursors[0])
	}

	m, _ = m.Update(nextKey)
	if m.(KanbanModel).cursors[0] != 2 {
		t.Errorf("cursor = %d after clamp, want 2", m.(KanbanModel).cursors[0])
	}

	prevKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m, _ = m.Update(prevKey)
	if m.(KanbanModel).cursors[0] != 1 {
		t.Errorf("cursor = %d, want 1", m.(KanbanModel).cursors[0])
	}
	m, _ = m.Update(prevKey)
	if m.(KanbanModel).cursors[0] != 0 {
		t.Errorf("cursor = %d, want 0", m.(KanbanModel).cursors[0])
	}
	m, _ = m.Update(prevKey)
	if m.(KanbanModel).cursors[0] != 0 {
		t.Errorf("cursor = %d after clamp, want 0", m.(KanbanModel).cursors[0])
	}
}

func TestKanbanTicketNavigationEmptyColumn(t *testing.T) {
	m := newTestKanban(t)
	km := m
	km.colIndex = 1
	updated, _ := km.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if updated.(KanbanModel).cursors[1] != 0 {
		t.Errorf("cursor = %d on empty column, want 0", updated.(KanbanModel).cursors[1])
	}
}
```

Add required imports to `kanban_test.go`:

```go
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run "TestKanbanWindow|TestKanbanColumn|TestKanbanTicket|TestKanbanJump" -v`
Expected: FAIL — `Update` method doesn't exist on KanbanModel

- [ ] **Step 3: Implement Update method and Reload/SelectedTicket helpers**

Add to `internal/tui/kanban.go`:

```go
func (m KanbanModel) Update(msg tea.Msg) (KanbanModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m KanbanModel) handleKey(msg tea.KeyMsg) (KanbanModel, tea.Cmd) {
	key := msg.String()
	action, _ := m.resolver.Resolve(key)

	switch action {
	case keybinding.ActionPrevColumn:
		if m.colIndex > 0 {
			m.colIndex--
		}
	case keybinding.ActionNextColumn:
		if m.colIndex < 3 {
			m.colIndex++
		}
	case keybinding.ActionPrevTicket:
		if m.cursors[m.colIndex] > 0 {
			m.cursors[m.colIndex]--
		}
	case keybinding.ActionNextTicket:
		if m.cursors[m.colIndex] < len(m.columns[m.colIndex])-1 {
			m.cursors[m.colIndex]++
		}
	case keybinding.ActionJumpColumn1:
		m.colIndex = 0
	case keybinding.ActionJumpColumn2:
		m.colIndex = 1
	case keybinding.ActionJumpColumn3:
		m.colIndex = 2
	case keybinding.ActionJumpColumn4:
		m.colIndex = 3
	case keybinding.ActionAddTicket:
		_, err := m.store.CreateTicket(context.Background(), store.Ticket{
			Title:  "New Ticket",
			Status: "backlog",
		})
		if err != nil {
			return m, nil
		}
		m, _ = m.loadColumns()
	case keybinding.ActionDeleteTicket:
		col := m.columns[m.colIndex]
		if len(col) > 0 {
			cursor := m.cursors[m.colIndex]
			_ = m.store.DeleteTicket(context.Background(), col[cursor].ID)
			m, _ = m.loadColumns()
		}
	}

	return m, nil
}

func (m KanbanModel) SelectedTicket() *store.Ticket {
	col := m.columns[m.colIndex]
	if len(col) == 0 {
		return nil
	}
	cursor := m.cursors[m.colIndex]
	if cursor >= len(col) {
		return nil
	}
	t := col[cursor]
	return &t
}

func (m KanbanModel) Reload() (KanbanModel, error) {
	return m.loadColumns()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run "TestKanbanWindow|TestKanbanColumn|TestKanbanTicket|TestKanbanJump" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/kanban.go internal/tui/kanban_test.go
git commit -m "feat(tui): add KanbanModel Update with navigation, SelectedTicket, Reload"
```

---

### Task 5: Write add/delete/selected ticket tests

**Files:**
- Modify: `internal/tui/kanban_test.go`

- [ ] **Step 1: Write the tests**

Add to `internal/tui/kanban_test.go`:

```go
func TestKanbanAddTicket(t *testing.T) {
	m := newTestKanban(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	km := updated.(KanbanModel)

	if len(km.columns[0]) != 1 {
		t.Fatalf("backlog has %d tickets, want 1", len(km.columns[0]))
	}
	if km.columns[0][0].Title != "New Ticket" {
		t.Errorf("title = %q, want %q", km.columns[0][0].Title, "New Ticket")
	}
}

func TestKanbanDeleteTicket(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()

	m.store.CreateTicket(ctx, store.Ticket{Title: "To Delete", Status: "backlog"})
	var err error
	m, err = m.Reload()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	if len(m.columns[0]) != 1 {
		t.Fatalf("setup: backlog has %d tickets, want 1", len(m.columns[0]))
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	km := updated.(KanbanModel)
	if len(km.columns[0]) != 0 {
		t.Errorf("backlog has %d tickets after delete, want 0", len(km.columns[0]))
	}
}

func TestKanbanDeleteTicketEmptyColumn(t *testing.T) {
	m := newTestKanban(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	km := updated.(KanbanModel)
	if len(km.columns[0]) != 0 {
		t.Errorf("backlog has %d tickets, want 0", len(km.columns[0]))
	}
}

func TestKanbanSelectedTicket(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()

	ticket, _ := m.store.CreateTicket(ctx, store.Ticket{Title: "Pick Me", Status: "backlog"})
	m, _ = m.Reload()

	selected := m.SelectedTicket()
	if selected == nil {
		t.Fatal("SelectedTicket() returned nil")
	}
	if selected.ID != ticket.ID {
		t.Errorf("SelectedTicket().ID = %q, want %q", selected.ID, ticket.ID)
	}
}

func TestKanbanSelectedTicketEmptyColumn(t *testing.T) {
	m := newTestKanban(t)
	selected := m.SelectedTicket()
	if selected != nil {
		t.Errorf("SelectedTicket() = %v on empty, want nil", selected)
	}
}

func TestKanbanReload(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()

	m.store.CreateTicket(ctx, store.Ticket{Title: "External", Status: "backlog"})
	if len(m.columns[0]) != 0 {
		t.Fatalf("before reload: backlog has %d tickets, want 0", len(m.columns[0]))
	}

	m, err := m.Reload()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(m.columns[0]) != 1 {
		t.Errorf("after reload: backlog has %d tickets, want 1", len(m.columns[0]))
	}
	if m.columns[0][0].Title != "External" {
		t.Errorf("after reload: title = %q, want %q", m.columns[0][0].Title, "External")
	}
}
```

- [ ] **Step 2: Run the tests**

Run: `go test ./internal/tui/ -run "TestKanbanAdd|TestKanbanDelete|TestKanbanSelected|TestKanbanReload" -v`
Expected: PASS (implementation was added in Task 4)

- [ ] **Step 3: Commit**

```bash
git add internal/tui/kanban_test.go
git commit -m "test(tui): add kanban add/delete/selected/reload tests"
```

---

### Task 6: Implement View with lipgloss-styled columns

**Files:**
- Modify: `internal/tui/kanban.go`
- Modify: `internal/tui/kanban_test.go`

- [ ] **Step 1: Write view tests**

Add to `internal/tui/kanban_test.go`:

```go
func TestKanbanViewRendersColumns(t *testing.T) {
	m := newTestKanban(t)
	km := m
	km.width = 120
	km.height = 40

	view := km.View()
	for _, name := range columnNames {
		if !strings.Contains(view, name) {
			t.Errorf("view missing column name %q", name)
		}
	}
}

func TestKanbanViewRendersTickets(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()

	m.store.CreateTicket(ctx, store.Ticket{Title: "First Task", Status: "backlog"})
	m.store.CreateTicket(ctx, store.Ticket{Title: "Second Task", Status: "in_progress"})
	m, _ = m.Reload()
	km := m
	km.width = 120
	km.height = 40

	view := km.View()
	if !strings.Contains(view, "First Task") {
		t.Error("view missing ticket title 'First Task'")
	}
	if !strings.Contains(view, "Second Task") {
		t.Error("view missing ticket title 'Second Task'")
	}
}

func TestKanbanViewEmptyColumn(t *testing.T) {
	m := newTestKanban(t)
	km := m
	km.width = 120
	km.height = 40

	view := km.View()
	if !strings.Contains(view, "empty") {
		t.Error("view missing '(empty)' for empty columns")
	}
}

func TestKanbanViewTruncation(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()

	longTitle := strings.Repeat("A", 100)
	m.store.CreateTicket(ctx, store.Ticket{Title: longTitle, Status: "backlog"})
	m, _ = m.Reload()
	km := m
	km.width = 80
	km.height = 24

	view := km.View()
	if !strings.Contains(view, "…") {
		t.Error("view missing truncation marker '…'")
	}
}

func TestKanbanViewFocusedColumn(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()

	m.store.CreateTicket(ctx, store.Ticket{Title: "Task", Status: "backlog"})
	m, _ = m.Reload()
	km := m
	km.width = 120
	km.height = 40

	view := km.View()
	if !strings.Contains(view, "▸") {
		t.Error("view missing selection marker '▸'")
	}
}
```

Add `"strings"` to imports in `kanban_test.go`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run "TestKanbanView" -v`
Expected: FAIL — `View` method doesn't exist on KanbanModel

- [ ] **Step 3: Implement View method**

Add to `internal/tui/kanban.go`, adding these imports:

```go
"strings"
"unicode/utf8"
```

Then add the View method:

```go
func (m KanbanModel) View() string {
	if m.width == 0 {
		return "AgentBoard — Kanban"
	}

	colWidth := (m.width - 1) / 4
	innerWidth := colWidth - 4

	columns := make([]string, 4)
	for i := range columns {
		columns[i] = m.renderColumn(i, innerWidth)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

func (m KanbanModel) renderColumn(colIdx int, innerWidth int) string {
	name := columnNames[colIdx]
	isFocused := colIdx == m.colIndex

	var titleStyle lipgloss.Style
	var columnStyle lipgloss.Style

	if isFocused {
		titleStyle = m.styles.FocusedTitle
		columnStyle = m.styles.FocusedColumn
	} else {
		titleStyle = m.styles.BlurredTitle
		columnStyle = m.styles.BlurredColumn
	}

	columnStyle = columnStyle.Width(innerWidth)

	title := titleStyle.Render(name)

	tickets := m.columns[colIdx]
	cursor := m.cursors[colIdx]
	availableHeight := m.height - 6
	if availableHeight < 1 {
		availableHeight = 1
	}

	var body strings.Builder
	visibleCount := 0
	for j, ticket := range tickets {
		if visibleCount >= availableHeight {
			remaining := len(tickets) - j
			body.WriteString(m.styles.EmptyColumn.Render(fmt.Sprintf("  ↓ %d more", remaining)))
			body.WriteString("\n")
			break
		}

		line := fmt.Sprintf("  %s %s", ticket.ID, ticket.Title)
		runeCount := utf8.RuneCountInString(line)
		if runeCount > innerWidth {
			runes := []rune(line)
			line = string(runes[:innerWidth-1]) + "…"
		}

		if isFocused && j == cursor {
			line = "▸ " + line[2:]
			body.WriteString(m.styles.SelectedTicket.Render(line))
		} else {
			body.WriteString(m.styles.Ticket.Render(line))
		}
		body.WriteString("\n")
		visibleCount++
	}

	if len(tickets) == 0 {
		body.WriteString(m.styles.EmptyColumn.Render("  (empty)"))
		body.WriteString("\n")
	}

	content := lipgloss.JoinVertical(lipgloss.Top, title, body.String())
	return columnStyle.Render(content)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run "TestKanbanView" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/kanban.go internal/tui/kanban_test.go
git commit -m "feat(tui): add KanbanModel View with lipgloss-styled bordered columns"
```

---

### Task 7: Refactor App to delegate to KanbanModel

**Files:**
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/app_test.go`

This is the most delicate task. We must:
1. Remove `columns`, `colIndex`, `cursors`, `loadColumns`, `renderBoard` from App
2. Add `kanban KanbanModel` field
3. Delegate window-size and key messages to KanbanModel
4. Keep App-level concerns: quit, open ticket, show help, view routing

- [ ] **Step 1: Rewrite app.go**

Replace entire contents of `internal/tui/app.go` with:

```go
package tui

import (
	"fmt"

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

type App struct {
	store    *store.Store
	resolver *keybinding.Resolver
	config   *config.Config
	width    int
	height   int

	focus focusArea
	view  viewMode

	kanban       KanbanModel
	activeTicket *store.Ticket
}

func NewApp(cfg *config.Config, s *store.Store) (*App, error) {
	km := keybinding.DefaultKeyMap()
	if len(cfg.TUI.Keybindings) > 0 {
		keybinding.ApplyConfig(&km, cfg.TUI.Keybindings)
	}

	resolver := keybinding.NewResolver(km)
	kanban, err := NewKanbanModel(s, resolver)
	if err != nil {
		return nil, fmt.Errorf("tui.newApp: %w", err)
	}

	a := &App{
		store:    s,
		resolver: resolver,
		config:   cfg,
		focus:    focusBoard,
		view:     viewBoard,
		kanban:   kanban,
	}

	return a, nil
}

func (a *App) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.kanban, _ = a.kanban.Update(msg)
		return a, nil
	case tea.KeyMsg:
		return a.handleKey(msg)
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" && a.view != viewBoard {
		a.view = viewBoard
		a.activeTicket = nil
		return a, nil
	}

	key := msg.String()
	action, _ := a.resolver.Resolve(key)

	switch action {
	case keybinding.ActionQuit, keybinding.ActionForceQuit:
		return a, tea.Quit
	case keybinding.ActionOpenTicket:
		selected := a.kanban.SelectedTicket()
		if selected != nil {
			a.activeTicket = selected
			a.view = viewTicket
		}
	case keybinding.ActionShowHelp:
		if a.view == viewHelp {
			a.view = viewBoard
		} else {
			a.view = viewHelp
		}
	default:
		a.kanban, _ = a.kanban.Update(msg)
	}

	return a, nil
}

func (a *App) View() string {
	switch a.view {
	case viewHelp:
		return a.renderHelp()
	case viewTicket:
		return a.renderTicket()
	default:
		return a.kanban.View()
	}
}

func (a *App) renderTicket() string {
	if a.activeTicket == nil {
		return "No ticket selected"
	}
	t := a.activeTicket
	return fmt.Sprintf("Ticket: %s\nTitle:  %s\nStatus: %s\n\n%s\n\nPress Esc to return",
		t.ID, t.Title, t.Status, t.Description)
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

Wait — we removed `strings` and `fmt` imports but `renderHelp` still uses `strings.Builder` and `fmt.Sprintf`. Need to keep those imports. And `context` is no longer needed in app.go. Let me correct:

```go
import (
	"fmt"
	"strings"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"

	tea "github.com/charmbracelet/bubbletea"
)
```

- [ ] **Step 2: Rewrite app_test.go — keep only app-level tests**

Replace entire contents of `internal/tui/app_test.go` with:

```go
package tui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/store"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.Open(dbPath, []string{"backlog", "in_progress", "review", "done"})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	cfg := config.SetDefaults()
	app, err := NewApp(cfg, s)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return app
}

func TestNewApp(t *testing.T) {
	app := newTestApp(t)

	if app == nil {
		t.Fatal("app is nil")
	}
	if app.kanban.store == nil {
		t.Error("kanban store is nil")
	}
	if app.view != viewBoard {
		t.Errorf("view = %v, want viewBoard", app.view)
	}
	if app.focus != focusBoard {
		t.Errorf("focus = %v, want focusBoard", app.focus)
	}
}

func TestAppQuit(t *testing.T) {
	app := newTestApp(t)
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("cmd is nil, expected tea.Quit")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("cmd produced %T, want tea.QuitMsg", msg)
	}
}

func TestAppForceQuit(t *testing.T) {
	app := newTestApp(t)
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("cmd is nil, expected tea.Quit")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("cmd produced %T, want tea.QuitMsg", msg)
	}
}

func TestAppShowHelp(t *testing.T) {
	app := newTestApp(t)

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if app.view != viewHelp {
		t.Errorf("view = %v, want viewHelp", app.view)
	}

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if app.view != viewBoard {
		t.Errorf("view = %v, want viewBoard", app.view)
	}
}

func TestAppOpenTicket(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()

	app.store.CreateTicket(ctx, store.Ticket{Title: "Open Me", Status: "backlog"})
	var err error
	app.kanban, err = app.kanban.Reload()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	app.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if app.view != viewTicket {
		t.Errorf("view = %v, want viewTicket", app.view)
	}
	if app.activeTicket == nil || app.activeTicket.Title != "Open Me" {
		t.Errorf("activeTicket = %v, want 'Open Me'", app.activeTicket)
	}
}

func TestAppOpenTicketEmptyColumn(t *testing.T) {
	app := newTestApp(t)

	app.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if app.view != viewBoard {
		t.Errorf("view = %v, want viewBoard", app.view)
	}
}

func TestAppEscapeReturnsToBoard(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()

	app.store.CreateTicket(ctx, store.Ticket{Title: "Escape Me", Status: "backlog"})
	app.kanban, _ = app.kanban.Reload()

	app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if app.view != viewTicket {
		t.Fatalf("view = %v, want viewTicket before escape", app.view)
	}

	app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if app.view != viewBoard {
		t.Errorf("view = %v after escape, want viewBoard", app.view)
	}
	if app.activeTicket != nil {
		t.Error("activeTicket should be nil after escape")
	}
}

func TestAppViewRouting(t *testing.T) {
	app := newTestApp(t)

	view := app.View()
	if !strings.Contains(view, "Kanban") && len(view) == 0 {
		t.Error("board view is empty")
	}

	app.view = viewHelp
	view = app.View()
	if !strings.Contains(view, "Help") {
		t.Error("help view missing 'Help'")
	}

	app.view = viewTicket
	app.activeTicket = &store.Ticket{ID: "TEST-01", Title: "Routed", Status: "backlog"}
	view = app.View()
	if !strings.Contains(view, "Routed") {
		t.Error("ticket view missing title")
	}
}

func TestAppWindowResizeDelegatesToKanban(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if app.kanban.width != 120 {
		t.Errorf("kanban width = %d, want 120", app.kanban.width)
	}
	if app.kanban.height != 40 {
		t.Errorf("kanban height = %d, want 40", app.kanban.height)
	}
}

func TestAppNavigationDelegatesToKanban(t *testing.T) {
	app := newTestApp(t)

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if app.kanban.colIndex != 1 {
		t.Errorf("kanban colIndex = %d, want 1", app.kanban.colIndex)
	}

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if app.kanban.colIndex != 0 {
		t.Errorf("kanban colIndex = %d, want 0", app.kanban.colIndex)
	}
}
```

- [ ] **Step 3: Run all tests**

Run: `go test ./internal/tui/ -v`
Expected: ALL PASS

- [ ] **Step 4: Run full test suite**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "refactor(tui): delegate App to KanbanModel, extract kanban concerns"
```

---

### Task 8: Final verification

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 2: Run go vet**

Run: `go vet ./...`
Expected: no output (clean)

- [ ] **Step 3: Build the binary**

Run: `go build -o agentboard ./cmd/agentboard`
Expected: builds successfully

- [ ] **Step 4: Final commit if any fixes needed**

```bash
git add -A
git commit -m "chore: final cleanup for kanban model extraction"
```
