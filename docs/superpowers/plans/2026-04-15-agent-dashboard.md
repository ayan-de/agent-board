# Agent Dashboard Overlay — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a full-screen dashboard overlay showing installed CLI agents with their status, opened with `i`.

**Architecture:** New `DashboardModel` in `tui/dashboard.go` renders agent cards in a lipgloss-styled overlay. Agent detection via `config/agent_detect.go` scans `$PATH`. Integrated into `App` as a new `viewDashboard` view mode.

**Tech Stack:** Go, bubbletea, lipgloss, `os/exec.LookPath`

---

## File Structure

| File | Responsibility |
|------|----------------|
| `internal/config/agent_detect.go` | `DetectAgents()` — scans `$PATH` for claude/opencode/cursor |
| `internal/config/agent_detect_test.go` | Tests for agent detection |
| `internal/keybinding/action.go` | Add `ActionShowDashboard` constant + String() case |
| `internal/keybinding/keymap.go` | Add `{Key: "i", Action: ActionShowDashboard}` binding |
| `internal/keybinding/action_test.go` | Add dashboard action to table-driven test |
| `internal/tui/dashboard.go` | `DashboardModel` — full overlay with agent cards |
| `internal/tui/dashboard_test.go` | Tests for dashboard model |
| `internal/tui/app.go` | Add `viewDashboard`, `dashboard` field, key routing |
| `internal/tui/app_test.go` | Add dashboard integration tests |

---

### Task 1: Add `ActionShowDashboard` to keybinding package

**Files:**
- Modify: `internal/keybinding/action.go`
- Modify: `internal/keybinding/keymap.go`
- Modify: `internal/keybinding/action_test.go`

- [ ] **Step 1: Write the failing test**

Add `{ActionShowDashboard, "show_dashboard"}` to the table in `internal/keybinding/action_test.go`:

```go
{ActionShowDashboard, "show_dashboard"},
```

Insert it after the `{ActionGoToTicket, "go_to_ticket"}` entry.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/keybinding/ -run TestActionString -v`
Expected: FAIL — `undefined: ActionShowDashboard`

- [ ] **Step 3: Add the action constant and string**

In `internal/keybinding/action.go`, add `ActionShowDashboard` after `ActionGoToTicket` in the const block:

```go
ActionGoToTicket
ActionShowDashboard
```

Add the String() case before the `default:` clause:

```go
case ActionShowDashboard:
    return "show_dashboard"
```

In `internal/keybinding/keymap.go`, add the binding to `DefaultKeyMap()` after the `g` chord binding:

```go
{Key: "i", Action: ActionShowDashboard},
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/keybinding/ -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/keybinding/action.go internal/keybinding/keymap.go internal/keybinding/action_test.go
git commit -m "feat(keybinding): add ActionShowDashboard bound to 'i'"
```

---

### Task 2: Implement agent detection

**Files:**
- Create: `internal/config/agent_detect.go`
- Create: `internal/config/agent_detect_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/config/agent_detect_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectAgents(t *testing.T) {
	agents := DetectAgents()
	if len(agents) != 3 {
		t.Fatalf("DetectAgents() returned %d agents, want 3", len(agents))
	}

	names := map[string]bool{}
	for _, a := range agents {
		names[a.Name] = true
		if a.Binary == "" {
			t.Errorf("agent %q has empty Binary", a.Name)
		}
	}

	for _, want := range []string{"claude-code", "opencode", "cursor"} {
		if !names[want] {
			t.Errorf("missing agent %q", want)
		}
	}
}

func TestDetectAgentsFoundWithBinaryOnPath(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "claude")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh"), 0755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", dir)
	agents := DetectAgents()

	found := false
	for _, a := range agents {
		if a.Binary == "claude" && a.Found {
			found = true
		}
	}
	if !found {
		t.Error("claude not detected as found with fake binary on PATH")
	}
}

func TestDetectAgentsNotFoundWhenMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	agents := DetectAgents()

	for _, a := range agents {
		if a.Found {
			t.Errorf("agent %q should not be found with empty PATH", a.Name)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestDetectAgents -v`
Expected: FAIL — `undefined: DetectAgents`

- [ ] **Step 3: Implement DetectAgents**

Create `internal/config/agent_detect.go`:

```go
package config

import (
	"fmt"
	"os/exec"
)

type DetectedAgent struct {
	Name   string
	Binary string
	Path   string
	Found  bool
}

type agentSpec struct {
	name   string
	binary string
}

var agentSpecs = []agentSpec{
	{name: "claude-code", binary: "claude"},
	{name: "opencode", binary: "opencode"},
	{name: "cursor", binary: "cursor"},
}

func DetectAgents() []DetectedAgent {
	agents := make([]DetectedAgent, len(agentSpecs))
	for i, spec := range agentSpecs {
		path, err := exec.LookPath(spec.binary)
		agents[i] = DetectedAgent{
			Name:   spec.name,
			Binary: spec.binary,
			Path:   path,
			Found:  err == nil,
		}
		if err == nil {
			agents[i].Path = fmt.Sprintf("%s (%s)", spec.binary, path)
		}
	}
	return agents
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestDetectAgents -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/agent_detect.go internal/config/agent_detect_test.go
git commit -m "feat(config): add DetectAgents for PATH scanning"
```

---

### Task 3: Implement DashboardModel

**Files:**
- Create: `internal/tui/dashboard.go`
- Create: `internal/tui/dashboard_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/tui/dashboard_test.go`:

```go
package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestDashboard(t *testing.T) DashboardModel {
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
	agents := config.DetectAgents()

	return NewDashboardModel(s, resolver, agents)
}

func TestNewDashboardModel(t *testing.T) {
	m := newTestDashboard(t)
	if m.store == nil {
		t.Error("store is nil")
	}
	if m.resolver == nil {
		t.Error("resolver is nil")
	}
	if len(m.agents) != 3 {
		t.Errorf("agents = %d, want 3", len(m.agents))
	}
	if m.width != 0 {
		t.Errorf("width = %d, want 0", m.width)
	}
}

func TestDashboardInit(t *testing.T) {
	m := newTestDashboard(t)
	cmd := m.Init()
	if cmd != nil {
		t.Errorf("Init() = %v, want nil", cmd)
	}
}

func TestDashboardWindowSize(t *testing.T) {
	m := newTestDashboard(t)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.width != 100 {
		t.Errorf("width = %d, want 100", m.width)
	}
	if m.height != 30 {
		t.Errorf("height = %d, want 30", m.height)
	}
}

func TestDashboardViewRendersAgentNames(t *testing.T) {
	m := newTestDashboard(t)
	m.width = 120
	m.height = 40

	view := m.View()
	if view == "" {
		t.Fatal("view is empty")
	}

	for _, name := range []string{"claude-code", "opencode", "cursor"} {
		if !strings.Contains(view, name) {
			t.Errorf("view missing agent name %q", name)
		}
	}
}

func TestDashboardViewRendersStatusLabels(t *testing.T) {
	m := newTestDashboard(t)
	m.width = 120
	m.height = 40

	view := m.View()
	labels := []string{"Status:", "Running:", "Ticket:", "Uptime:", "Subagents:", "Tokens:"}
	for _, label := range labels {
		if !strings.Contains(view, label) {
			t.Errorf("view missing label %q", label)
		}
	}
}

func TestDashboardViewRendersEmDash(t *testing.T) {
	m := newTestDashboard(t)
	m.width = 120
	m.height = 40

	view := m.View()
	if !strings.Contains(view, "—") {
		t.Error("view should contain em-dash placeholders for Phase 3 fields")
	}
}

func TestDashboardViewRendersFooter(t *testing.T) {
	m := newTestDashboard(t)
	m.width = 120
	m.height = 40

	view := m.View()
	if !strings.Contains(view, "r: refresh") {
		t.Error("view missing refresh hint")
	}
	if !strings.Contains(view, "Esc") {
		t.Error("view missing Esc hint")
	}
}

func TestDashboardRefresh(t *testing.T) {
	m := newTestDashboard(t)
	origCount := len(m.agents)
	m = m.Refresh()
	if len(m.agents) != origCount {
		t.Errorf("agents count changed after refresh: %d vs %d", origCount, len(m.agents))
	}
}

func TestDashboardRefreshKey(t *testing.T) {
	m := newTestDashboard(t)
	m.width = 120
	m.height = 40

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if !m.refreshed {
		t.Error("refresh flag not set after pressing r")
	}
}

func TestDashboardViewNoWidth(t *testing.T) {
	m := newTestDashboard(t)
	view := m.View()
	if view != "" {
		t.Errorf("view should be empty with zero width, got: %q", view)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestNewDashboardModel -v`
Expected: FAIL — `undefined: DashboardModel`

- [ ] **Step 3: Implement DashboardModel**

Create `internal/tui/dashboard.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DashboardStyles struct {
	Border      lipgloss.Style
	Title       lipgloss.Style
	Card        lipgloss.Style
	CardFound   lipgloss.Style
	CardMissing lipgloss.Style
	Label       lipgloss.Style
	Value       lipgloss.Style
	Placeholder lipgloss.Style
	Footer      lipgloss.Style
}

type DashboardModel struct {
	store    *store.Store
	resolver *keybinding.Resolver
	agents   []config.DetectedAgent
	width    int
	height   int
	refreshed bool
	styles   DashboardStyles
}

func DefaultDashboardStyles() DashboardStyles {
	return DashboardStyles{
		Border: lipgloss.NewStyle().
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("69")),
		Card: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("69")).
			Padding(0, 1).
			Width(30),
		CardFound: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("42")).
			Padding(0, 1).
			Width(30),
		CardMissing: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Width(30),
		Label: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		Value: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		Placeholder: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		Footer: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
	}
}

func NewDashboardModel(s *store.Store, resolver *keybinding.Resolver, agents []config.DetectedAgent) DashboardModel {
	return DashboardModel{
		store:    s,
		resolver: resolver,
		agents:   agents,
		styles:   DefaultDashboardStyles(),
	}
}

func (m DashboardModel) Init() tea.Cmd {
	return nil
}

func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
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

func (m DashboardModel) handleKey(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
	key := msg.String()
	action, _ := m.resolver.Resolve(key)

	switch action {
	case keybinding.ActionRefresh:
		m = m.Refresh()
	}

	return m, nil
}

func (m DashboardModel) Refresh() DashboardModel {
	m.agents = config.DetectAgents()
	m.refreshed = true
	return m
}

func (m DashboardModel) View() string {
	if m.width == 0 {
		return ""
	}

	var b strings.Builder

	title := m.styles.Title.Render("Agent Dashboard")
	b.WriteString(title)
	b.WriteString("\n\n")

	cards := make([]string, len(m.agents))
	for i, agent := range m.agents {
		cards[i] = m.renderCard(agent)
	}

	innerWidth := m.width - 4
	cardWidth := 32
	cardsPerRow := innerWidth / cardWidth
	if cardsPerRow < 1 {
		cardsPerRow = 1
	}

	for rowStart := 0; rowStart < len(cards); rowStart += cardsPerRow {
		rowEnd := rowStart + cardsPerRow
		if rowEnd > len(cards) {
			rowEnd = len(cards)
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, cards[rowStart:rowEnd]...)
		b.WriteString(row)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	footer := m.styles.Footer.Render("r: refresh │ Esc: back")
	b.WriteString(footer)

	return b.String()
}

func (m DashboardModel) renderCard(agent config.DetectedAgent) string {
	var b strings.Builder

	name := m.styles.Title.Render(agent.Name)
	b.WriteString(name)
	b.WriteString("\n")

	statusVal := "not found"
	if agent.Found {
		statusVal = "installed"
	}

	fields := []struct {
		label string
		value string
	}{
		{"Status:", statusVal},
		{"Running:", "no"},
		{"Ticket:", "—"},
		{"Uptime:", "—"},
		{"Subagents:", "—"},
		{"Tokens:", "—"},
	}

	for _, f := range fields {
		label := m.styles.Label.Render(f.label)
		var val string
		if f.value == "—" {
			val = m.styles.Placeholder.Render(f.value)
		} else {
			val = m.styles.Value.Render(f.value)
		}
		fmt.Fprintf(&b, "%s %s\n", label, val)
	}

	style := m.styles.CardMissing
	if agent.Found {
		style = m.styles.CardFound
	}

	return style.Render(b.String())
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run "TestNewDashboardModel|TestDashboard" -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/dashboard.go internal/tui/dashboard_test.go
git commit -m "feat(tui): add DashboardModel with agent card rendering"
```

---

### Task 4: Integrate dashboard into App

**Files:**
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/app_test.go`

- [ ] **Step 1: Write the failing tests**

Add these tests to `internal/tui/app_test.go`:

```go
func TestAppShowDashboard(t *testing.T) {
	app := newTestApp(t)

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if app.view != viewDashboard {
		t.Errorf("view = %v, want viewDashboard", app.view)
	}

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if app.view != viewBoard {
		t.Errorf("view = %v after second 'i', want viewBoard", app.view)
	}
}

func TestAppEscapeFromDashboard(t *testing.T) {
	app := newTestApp(t)

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if app.view != viewDashboard {
		t.Fatalf("view = %v, want viewDashboard", app.view)
	}

	app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if app.view != viewBoard {
		t.Errorf("view = %v after esc, want viewBoard", app.view)
	}
}

func TestAppDashboardViewRenders(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	view := app.View()
	if !strings.Contains(view, "claude-code") {
		t.Error("dashboard view missing agent name")
	}
}

func TestAppWindowResizeDelegatesToDashboard(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if app.dashboard.width != 120 {
		t.Errorf("dashboard width = %d, want 120", app.dashboard.width)
	}
	if app.dashboard.height != 40 {
		t.Errorf("dashboard height = %d, want 40", app.dashboard.height)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run "TestAppShowDashboard|TestAppEscapeFromDashboard|TestAppDashboardViewRenders|TestAppWindowResizeDelegatesToDashboard" -v`
Expected: FAIL — `undefined: viewDashboard`

- [ ] **Step 3: Integrate into app.go**

Add `viewDashboard` to the viewMode const block in `internal/tui/app.go`:

```go
const (
	viewBoard viewMode = iota
	viewTicket
	viewHelp
	viewDashboard
)
```

Add `dashboard DashboardModel` field to the `App` struct (after `ticketView`):

```go
kanban       KanbanModel
ticketView   TicketViewModel
dashboard    DashboardModel
activeTicket *store.Ticket
```

Add import for config package:

```go
import (
	"fmt"
	"strings"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)
```

Initialize dashboard in `NewApp()`, after `ticketView`:

```go
agents := config.DetectAgents()
a := &App{
	store:      s,
	resolver:   resolver,
	config:     cfg,
	focus:      focusBoard,
	view:       viewBoard,
	kanban:     kanban,
	ticketView: NewTicketViewModel(s, resolver),
	dashboard:  NewDashboardModel(s, resolver, agents),
}
```

Propagate window size to dashboard in `Update()`:

```go
case tea.WindowSizeMsg:
	a.width = msg.Width
	a.height = msg.Height
	a.kanban, _ = a.kanban.Update(msg)
	a.ticketView, _ = a.ticketView.Update(msg)
	a.dashboard, _ = a.dashboard.Update(msg)
	return a, nil
```

Add dashboard key routing in `handleKey()`. Add the `ActionShowDashboard` case inside the `switch action` block (after `ActionShowHelp`):

```go
case keybinding.ActionShowDashboard:
	if a.view == viewDashboard {
		a.view = viewBoard
	} else {
		a.view = viewDashboard
	}
```

Add dashboard view routing in `handleKey()` — keys should go to dashboard when in viewDashboard mode. Add this block after the `if a.view == viewTicket` block:

```go
if a.view == viewDashboard {
	a.dashboard, _ = a.dashboard.Update(msg)
	return a, nil
}
```

Add dashboard case to `View()`:

```go
func (a *App) View() string {
	switch a.view {
	case viewHelp:
		return a.renderHelp()
	case viewTicket:
		return a.ticketView.View()
	case viewDashboard:
		return a.dashboard.View()
	default:
		return a.kanban.View()
	}
}
```

- [ ] **Step 4: Run all tests to verify they pass**

Run: `go test ./internal/tui/ -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/app_test.go
git commit -m "feat(tui): integrate agent dashboard overlay into App"
```

---

### Task 5: Final verification

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v -count=1`
Expected: All PASS

- [ ] **Step 2: Run go vet**

Run: `go vet ./...`
Expected: No output (clean)

- [ ] **Step 3: Build binary**

Run: `go build -o /tmp/agentboard ./cmd/agentboard/`
Expected: Builds without errors

- [ ] **Step 4: Manual smoke test**

Run: `go run ./cmd/agentboard`

Verify:
- `i` opens dashboard overlay with agent cards
- Agent names visible (claude-code, opencode, cursor)
- Status shows "installed" or "not found" correctly
- Phase 3 fields show "—"
- `r` refreshes detection
- `Esc` returns to board
- `i` again toggles back to board
- `?` still opens help
- `Enter` on ticket still opens ticket view
- All other keybindings still work

- [ ] **Step 5: Update keybinding reference in AGENTS.md**

In the Keybinding Reference table in `AGENTS.md`, add:

```
| `i` | Show agent dashboard |
```

Commit:

```bash
git add AGENTS.md
git commit -m "docs: add 'i' keybinding to AGENTS.md"
```
