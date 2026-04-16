# Ticket Card Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace single-line ticket rendering in the kanban with bordered, card-like components that show description previews, priority, agent status dots, and an animated activity indicator when agents are running.

**Architecture:** New `internal/tui/ticketcard.go` encapsulates all card rendering (compact/expanded modes, activity bar, agent dot). Kanban delegates per-ticket rendering to it. A new `internal/tui/ticketer_animation.go` holds pure animation logic. The `store.Ticket` struct gains an `AgentActive bool` field via migration. Animation ticks flow through `app.go` → `kanban.go` → card models.

**Tech Stack:** Go, bubbletea, lipgloss, existing theme system

---

### Task 1: Add `agent_active` column to database schema

**Files:**
- Modify: `internal/store/migrations.go:3-40`
- Modify: `internal/store/tickets.go:11-67`
- Test: `internal/store/store_test.go`

- [ ] **Step 1: Write the failing test for migration**

Add to `internal/store/store_test.go` after `TestOpenIdempotent`:

```go
func TestOpenAddsAgentActiveColumn(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	var val bool
	err := s.db.QueryRow("SELECT agent_active FROM tickets LIMIT 0").Scan(&val)
	if err != nil {
		t.Fatalf("agent_active column should exist: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store/ -run TestOpenAddsAgentActiveColumn -v`
Expected: FAIL — `agent_active` column does not exist

- [ ] **Step 3: Add migration for `agent_active` column**

Replace `internal/store/migrations.go` with:

```go
package store

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tickets (
		id          TEXT PRIMARY KEY,
		title       TEXT NOT NULL,
		description TEXT DEFAULT '',
		status      TEXT NOT NULL,
		priority    TEXT DEFAULT 'medium',
		agent       TEXT DEFAULT '',
		branch      TEXT DEFAULT '',
		tags        TEXT DEFAULT '[]',
		depends_on  TEXT DEFAULT '[]',
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id          TEXT PRIMARY KEY,
		ticket_id   TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
		agent       TEXT NOT NULL,
		started_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		ended_at    DATETIME,
		status      TEXT NOT NULL,
		context_key TEXT DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
	CREATE INDEX IF NOT EXISTS idx_sessions_ticket ON sessions(ticket_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("ALTER TABLE tickets ADD COLUMN agent_active INTEGER DEFAULT 0")
	if err != nil {
		return err
	}

	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/store/ -run TestOpenAddsAgentActiveColumn -v`
Expected: PASS

- [ ] **Step 5: Run all store tests to ensure nothing broke**

Run: `go test ./internal/store/ -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add internal/store/migrations.go internal/store/store_test.go
git commit -m "feat(store): add agent_active column to tickets table"
```

---

### Task 2: Add `AgentActive` field to Ticket struct and wire through CRUD

**Files:**
- Modify: `internal/store/tickets.go:11-67`
- Test: `internal/store/store_test.go`

- [ ] **Step 1: Write the failing test for AgentActive field**

Add to `internal/store/store_test.go`:

```go
func TestCreateTicketAgentActiveDefault(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	ticket, err := s.CreateTicket(context.Background(), Ticket{
		Title: "Active test", Status: "backlog",
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if ticket.AgentActive {
		t.Error("AgentActive should default to false")
	}
}

func TestSetAgentActive(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	ticket, _ := s.CreateTicket(context.Background(), Ticket{
		Title: "Active test", Status: "backlog",
	})

	err := s.SetAgentActive(context.Background(), ticket.ID, true)
	if err != nil {
		t.Fatalf("SetAgentActive: %v", err)
	}

	got, _ := s.GetTicket(context.Background(), ticket.ID)
	if !got.AgentActive {
		t.Error("AgentActive should be true after SetAgentActive(true)")
	}

	err = s.SetAgentActive(context.Background(), ticket.ID, false)
	if err != nil {
		t.Fatalf("SetAgentActive false: %v", err)
	}

	got, _ = s.GetTicket(context.Background(), ticket.ID)
	if got.AgentActive {
		t.Error("AgentActive should be false after SetAgentActive(false)")
	}
}

func TestSetAgentActiveNotFound(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	err := s.SetAgentActive(context.Background(), "AGT-99", true)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/store/ -run "TestCreateTicketAgentActive|TestSetAgentActive" -v`
Expected: FAIL — `Ticket.AgentActive` undefined, `SetAgentActive` undefined

- [ ] **Step 3: Add `AgentActive` to Ticket struct and ticketRow**

In `internal/store/tickets.go`, update the `Ticket` struct to add the field:

```go
type Ticket struct {
	ID          string
	Title       string
	Description string
	Status      string
	Priority    string
	Agent       string
	Branch      string
	Tags        []string
	DependsOn   []string
	AgentActive bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

Update `ticketRow` struct:

```go
type ticketRow struct {
	ID          string
	Title       string
	Description string
	Status      string
	Priority    string
	Agent       string
	Branch      string
	Tags        string
	DependsOn   string
	AgentActive bool
	CreatedAt   string
	UpdatedAt   string
}
```

Update `toTicket()`:

```go
func (r ticketRow) toTicket() (Ticket, error) {
	var tags []string
	if err := json.Unmarshal([]byte(r.Tags), &tags); err != nil {
		return Ticket{}, err
	}
	var dependsOn []string
	if err := json.Unmarshal([]byte(r.DependsOn), &dependsOn); err != nil {
		return Ticket{}, err
	}

	return Ticket{
		ID:          r.ID,
		Title:       r.Title,
		Description: r.Description,
		Status:      r.Status,
		Priority:    r.Priority,
		Agent:       r.Agent,
		Branch:      r.Branch,
		Tags:        tags,
		DependsOn:   dependsOn,
		AgentActive: r.AgentActive,
	}, nil
}
```

Update `CreateTicket` to include `agent_active` in the INSERT:

```go
_, err = s.db.ExecContext(ctx,
	`INSERT INTO tickets (id, title, description, status, priority, agent, branch, tags, depends_on, agent_active, created_at, updated_at)
	 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	t.ID, t.Title, t.Description, t.Status, t.Priority, t.Agent, t.Branch, string(tags), string(deps), t.AgentActive, t.CreatedAt, t.UpdatedAt,
)
```

Update `GetTicket` query and Scan to include `agent_active`:

```go
func (s *Store) GetTicket(ctx context.Context, id string) (Ticket, error) {
	var r ticketRow
	err := s.db.QueryRowContext(ctx,
		"SELECT id, title, description, status, priority, agent, branch, tags, depends_on, agent_active, created_at, updated_at FROM tickets WHERE id = ?",
		id,
	).Scan(&r.ID, &r.Title, &r.Description, &r.Status, &r.Priority, &r.Agent, &r.Branch, &r.Tags, &r.DependsOn, &r.AgentActive, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return Ticket{}, fmt.Errorf("store.getTicket %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return Ticket{}, fmt.Errorf("store.getTicket %s: %w", id, err)
	}

	ticket, err := r.toTicket()
	if err != nil {
		return Ticket{}, fmt.Errorf("store.getTicket %s: %w", id, err)
	}

	return ticket, nil
}
```

Update `ListTickets` query and Scan:

```go
func (s *Store) ListTickets(ctx context.Context, filters TicketFilters) ([]Ticket, error) {
	query := "SELECT id, title, description, status, priority, agent, branch, tags, depends_on, agent_active, created_at, updated_at FROM tickets WHERE 1=1"
	var args []interface{}

	if filters.Status != "" {
		query += " AND status = ?"
		args = append(args, filters.Status)
	}
	if filters.Agent != "" {
		query += " AND agent = ?"
		args = append(args, filters.Agent)
	}
	if filters.Priority != "" {
		query += " AND priority = ?"
		args = append(args, filters.Priority)
	}
	if filters.Tag != "" {
		query += " AND tags LIKE ?"
		args = append(args, `%"`+filters.Tag+`"%`)
	}

	query += " ORDER BY created_at ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store.listTickets: %w", err)
	}
	defer rows.Close()

	var tickets []Ticket
	for rows.Next() {
		var r ticketRow
		if err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.Status, &r.Priority, &r.Agent, &r.Branch, &r.Tags, &r.DependsOn, &r.AgentActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store.listTickets: %w", err)
		}
		ticket, err := r.toTicket()
		if err != nil {
			return nil, fmt.Errorf("store.listTickets: %w", err)
		}
		tickets = append(tickets, ticket)
	}

	return tickets, nil
}
```

Add `SetAgentActive` method:

```go
func (s *Store) SetAgentActive(ctx context.Context, id string, active bool) error {
	result, err := s.db.ExecContext(ctx,
		"UPDATE tickets SET agent_active = ?, updated_at = ? WHERE id = ?",
		active, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("store.setAgentActive: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("store.setAgentActive %s: %w", id, ErrNotFound)
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/store/ -run "TestCreateTicketAgentActive|TestSetAgentActive" -v`
Expected: All PASS

- [ ] **Step 5: Run all store tests**

Run: `go test ./internal/store/ -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add internal/store/tickets.go internal/store/store_test.go
git commit -m "feat(store): add AgentActive field to Ticket with SetAgentActive method"
```

---

### Task 3: Create animation logic for activity indicator

**Files:**
- Create: `internal/tui/ticketer_animation.go`
- Create: `internal/tui/ticketer_animation_test.go`

- [ ] **Step 1: Write the failing test for ActivityBar**

Create `internal/tui/ticketer_animation_test.go`:

```go
package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestActivityBarWidth(t *testing.T) {
	bar := ActivityBar(0, 20, nil)
	visualWidth := lipgloss.Width(bar)
	if visualWidth != 20 {
		t.Errorf("ActivityBar width = %d, want 20", visualWidth)
	}
}

func TestActivityBarAllFrames(t *testing.T) {
	for frame := 0; frame < AnimFrames; frame++ {
		bar := ActivityBar(frame, 20, nil)
		if bar == "" {
			t.Errorf("frame %d: bar is empty", frame)
		}
	}
}

func TestActivityBarContainsGradientBlocks(t *testing.T) {
	bar := ActivityBar(0, 20, nil)
	if !strings.Contains(bar, "█") {
		t.Error("bar should contain peak blocks '█'")
	}
	if !strings.Contains(bar, "░") {
		t.Error("bar should contain empty blocks '░'")
	}
}

func TestActivityBarScrolling(t *testing.T) {
	bar0 := ActivityBar(0, 20, nil)
	bar1 := ActivityBar(1, 20, nil)
	if bar0 == bar1 {
		t.Error("consecutive frames should produce different bars")
	}
}

func TestActivityBarMinimumWidth(t *testing.T) {
	bar := ActivityBar(0, 4, nil)
	visualWidth := lipgloss.Width(bar)
	if visualWidth != 4 {
		t.Errorf("minimum width bar = %d, want 4", visualWidth)
	}
}

func TestActivityBarWrapsTo8Frames(t *testing.T) {
	bar8 := ActivityBar(8, 20, nil)
	bar0 := ActivityBar(0, 20, nil)
	if bar8 != bar0 {
		t.Error("frame 8 should wrap back to frame 0 pattern")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run TestActivityBar -v`
Expected: FAIL — `ActivityBar` undefined

- [ ] **Step 3: Implement ActivityBar**

Create `internal/tui/ticketer_animation.go`:

```go
package tui

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const AnimFrames = 8

var animPatterns = [AnimFrames]string{
	"░░▒▒▓▓██▓▓▒▒░░",
	"░▒▒▓▓██▓▓▒▒░░░",
	"▒▒▓▓██▓▓▒▒░░░░",
	"▒▓▓██▓▓▒▒░░░░▒",
	"▓▓██▓▓▒▒░░░░▒▒",
	"▓██▓▓▒▒░░░░▒▒▓",
	"██▓▓▒▒░░░░▒▒▓▓",
	"▓▓▒▒░░░░▒▒▓▓██",
}

type tickMsg struct{}

func ActivityBar(frame int, width int, t *theme.Theme) string {
	if width < 4 {
		width = 4
	}

	pattern := animPatterns[frame%AnimFrames]
	patternRunes := []rune(pattern)
	patternLen := utf8.RuneCountInString(pattern)

	filledColor := lipgloss.Color("213")
	emptyColor := lipgloss.Color("240")
	if t != nil {
		filledColor = t.Accent
		emptyColor = t.TextMuted
	}

	var b strings.Builder
	for i := 0; i < width; i++ {
		r := patternRunes[i%patternLen]
		switch r {
		case '█', '▓', '▒':
			b.WriteString(lipgloss.NewStyle().Foreground(filledColor).Render(string(r)))
		default:
			b.WriteString(lipgloss.NewStyle().Foreground(emptyColor).Render(string(r)))
		}
	}

	return b.String()
}

func agentDot(agent string, active bool) string {
	if agent == "" {
		return ""
	}

	if active {
		color := config.AgentColor(agent)
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("●")
	}

	return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("○")
}

func animationTick() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run TestActivityBar -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/ticketer_animation.go internal/tui/ticketer_animation_test.go
git commit -m "feat(tui): add ActivityBar animation for agent activity indicator"
```

---

### Task 4: Create TicketCardModel component

**Files:**
- Create: `internal/tui/ticketcard.go`
- Create: `internal/tui/ticketcard_test.go`

- [ ] **Step 1: Write the failing tests for TicketCardModel**

Create `internal/tui/ticketcard_test.go`:

```go
package tui

import (
	"strings"
	"testing"

	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

	"github.com/charmbracelet/lipgloss"
)

func testCardTheme() *theme.Theme {
	return &theme.Theme{
		Primary: lipgloss.Color("69"), Text: lipgloss.Color("15"),
		TextMuted: lipgloss.Color("240"), Background: lipgloss.Color("#000"),
		BackgroundPanel: lipgloss.Color("236"), Border: lipgloss.Color("240"),
		BorderActive: lipgloss.Color("69"), Accent: lipgloss.Color("213"),
		Success: lipgloss.Color("42"), Error: lipgloss.Color("196"),
		Warning: lipgloss.Color("214"),
	}
}

func testTicket() store.Ticket {
	return store.Ticket{
		ID:          "AGT-01",
		Title:       "Implement Auth",
		Description: "Add JWT authentication flow",
		Status:      "in_progress",
		Priority:    "high",
		Agent:       "claude-code",
		AgentActive: false,
	}
}

func TestTicketCardCompactRenders(t *testing.T) {
	ticket := testTicket()
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "AGT-01") {
		t.Error("compact card missing ticket ID")
	}
	if !strings.Contains(output, "Implement Auth") {
		t.Error("compact card missing title")
	}
	if !strings.Contains(output, "Add JWT authentication flow") {
		t.Error("compact card missing description")
	}
}

func TestTicketCardCompactHasBorder(t *testing.T) {
	ticket := testTicket()
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "╭") && !strings.Contains(output, "╰") {
		t.Error("compact card missing rounded border corners")
	}
}

func TestTicketCardCompactHeight(t *testing.T) {
	ticket := testTicket()
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	if card.CompactHeight() != 3 {
		t.Errorf("CompactHeight = %d, want 3", card.CompactHeight())
	}
}

func TestTicketCardExpandedRenders(t *testing.T) {
	ticket := testTicket()
	card := NewTicketCardModel(ticket, true, true, 40, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "AGT-01") {
		t.Error("expanded card missing ticket ID")
	}
	if !strings.Contains(output, "Implement Auth") {
		t.Error("expanded card missing title")
	}
	if !strings.Contains(output, "Add JWT authentication flow") {
		t.Error("expanded card missing description")
	}
}

func TestTicketCardExpandedHasBorder(t *testing.T) {
	ticket := testTicket()
	card := NewTicketCardModel(ticket, true, true, 40, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "╭") && !strings.Contains(output, "╰") {
		t.Error("expanded card missing rounded border corners")
	}
}

func TestTicketCardAgentDotIdle(t *testing.T) {
	ticket := testTicket()
	ticket.AgentActive = false
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "○") {
		t.Error("idle agent card should show muted dot '○'")
	}
}

func TestTicketCardAgentDotActive(t *testing.T) {
	ticket := testTicket()
	ticket.AgentActive = true
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "●") {
		t.Error("active agent card should show colored dot '●'")
	}
}

func TestTicketCardNoAgentNoDot(t *testing.T) {
	ticket := testTicket()
	ticket.Agent = ""
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if strings.Contains(output, "○") || strings.Contains(output, "●") {
		t.Error("no-agent card should not show any dot")
	}
}

func TestTicketCardActivityBarWhenActive(t *testing.T) {
	ticket := testTicket()
	ticket.AgentActive = true
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "▓") {
		t.Error("active card should show activity bar with filled blocks")
	}
}

func TestTicketCardNoActivityBarWhenIdle(t *testing.T) {
	ticket := testTicket()
	ticket.AgentActive = false
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if strings.Contains(output, "▓") {
		t.Error("idle card should not show activity bar")
	}
}

func TestTicketCardSelectedBorder(t *testing.T) {
	ticket := testTicket()
	card := NewTicketCardModel(ticket, true, false, 30, 0, testCardTheme())
	output := card.Render()

	if output == "" {
		t.Fatal("selected card rendered empty")
	}
}

func TestTicketCardLongTitleTruncation(t *testing.T) {
	ticket := testTicket()
	ticket.Title = strings.Repeat("X", 200)
	card := NewTicketCardModel(ticket, false, false, 20, 0, testCardTheme())
	output := card.Render()

	if strings.Contains(output, ticket.Title) {
		t.Error("long title should be truncated, not shown in full")
	}
}

func TestTicketCardLongDescriptionTruncation(t *testing.T) {
	ticket := testTicket()
	ticket.Description = strings.Repeat("D", 200)
	card := NewTicketCardModel(ticket, false, false, 20, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "…") {
		t.Error("long description should be truncated with ellipsis")
	}
}

func TestTicketCardEmptyDescription(t *testing.T) {
	ticket := testTicket()
	ticket.Description = ""
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if output == "" {
		t.Fatal("card with empty description should still render")
	}
}

func TestDefaultTicketCardStyles(t *testing.T) {
	styles := DefaultTicketCardStyles()
	if styles.SelectedBorder.GetBorderStyle() == lipgloss.NewStyle().GetBorderStyle() {
		t.Error("SelectedBorder should have a border set")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run TestTicketCard -v`
Expected: FAIL — `TicketCardModel` undefined, `NewTicketCardModel` undefined

- [ ] **Step 3: Implement TicketCardModel**

Create `internal/tui/ticketcard.go`:

```go
package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

	"github.com/charmbracelet/lipgloss"
)

type TicketCardModel struct {
	ticket   store.Ticket
	selected bool
	expanded bool
	width    int
	frame    int
	styles   TicketCardStyles
	theme    *theme.Theme
}

type TicketCardStyles struct {
	SelectedBorder lipgloss.Style
	NormalBorder   lipgloss.Style
	Title          lipgloss.Style
	Description    lipgloss.Style
	Metadata       lipgloss.Style
	PriorityColors map[string]lipgloss.Color
}

func DefaultTicketCardStyles() TicketCardStyles {
	return TicketCardStyles{
		SelectedBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("69")),
		NormalBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
		Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")),
		Description: lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		Metadata:    lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		PriorityColors: map[string]lipgloss.Color{
			"low":      lipgloss.Color("240"),
			"medium":   lipgloss.Color("252"),
			"high":     lipgloss.Color("214"),
			"critical": lipgloss.Color("196"),
		},
	}
}

func NewTicketCardStyles(t *theme.Theme) TicketCardStyles {
	return TicketCardStyles{
		SelectedBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary),
		NormalBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border),
		Title:       lipgloss.NewStyle().Bold(true).Foreground(t.Text),
		Description: lipgloss.NewStyle().Foreground(t.Text),
		Metadata:    lipgloss.NewStyle().Foreground(t.TextMuted),
		PriorityColors: map[string]lipgloss.Color{
			"low":      t.TextMuted,
			"medium":   t.Text,
			"high":     t.Warning,
			"critical": t.Error,
		},
	}
}

func NewTicketCardModel(ticket store.Ticket, selected bool, expanded bool, width int, frame int, t *theme.Theme) TicketCardModel {
	var styles TicketCardStyles
	if t != nil {
		styles = NewTicketCardStyles(t)
	} else {
		styles = DefaultTicketCardStyles()
	}
	return TicketCardModel{
		ticket:   ticket,
		selected: selected,
		expanded: expanded,
		width:    width,
		frame:    frame,
		styles:   styles,
		theme:    t,
	}
}

func (c TicketCardModel) CompactHeight() int {
	return 3
}

func (c TicketCardModel) ExpandedHeight() int {
	innerWidth := c.innerWidth()
	if innerWidth < 4 {
		innerWidth = 4
	}
	descLines := 1
	if c.ticket.Description != "" {
		descLines = (utf8.RuneCountInString(c.ticket.Description) + innerWidth - 1) / innerWidth
		if descLines < 1 {
			descLines = 1
		}
	}
	return 3 + descLines + 1
}

func (c TicketCardModel) Render() string {
	if c.expanded {
		return c.renderExpanded()
	}
	return c.renderCompact()
}

func (c TicketCardModel) innerWidth() int {
	if c.width < 4 {
		return 2
	}
	return c.width - 4
}

func (c TicketCardModel) renderCompact() string {
	iw := c.innerWidth()

	titleLine := c.ticket.ID + " " + truncateRunes(c.ticket.Title, iw-utf8.RuneCountInString(c.ticket.ID)-1)
	titleLine = c.styles.Title.Render(titleLine)

	descText := truncateRunes(c.ticket.Description, iw-1)
	if c.ticket.Description != "" && utf8.RuneCountInString(c.ticket.Description) > iw-1 {
		descText = descText + "…"
	} else if c.ticket.Description == "" {
		descText = ""
	}
	descLine := c.styles.Description.Render(descText)

	footerLine := c.renderFooter(iw)

	content := titleLine + "\n" + descLine + "\n" + footerLine

	borderStyle := c.styles.NormalBorder
	if c.selected {
		borderStyle = c.styles.SelectedBorder
	}

	return borderStyle.Width(iw).Render(content)
}

func (c TicketCardModel) renderExpanded() string {
	iw := c.innerWidth()

	titleLine := c.ticket.ID + " " + c.ticket.Title
	titleLine = c.styles.Title.Render(titleLine)

	sepLine := strings.Repeat("─", iw)

	var descLines string
	if c.ticket.Description == "" {
		descLines = ""
	} else {
		descLines = wrapText(c.ticket.Description, iw)
	}

	footerLine := c.renderFooter(iw)

	parts := []string{titleLine, sepLine}
	if descLines != "" {
		parts = append(parts, descLines)
	}
	parts = append(parts, "", footerLine)

	content := strings.Join(parts, "\n")

	borderStyle := c.styles.NormalBorder
	if c.selected {
		borderStyle = c.styles.SelectedBorder
	}

	return borderStyle.Width(iw).Render(content)
}

func (c TicketCardModel) renderFooter(width int) string {
	priorityColor, ok := c.styles.PriorityColors[c.ticket.Priority]
	if !ok {
		priorityColor = lipgloss.Color("252")
	}
	priorityIndicator := lipgloss.NewStyle().Foreground(priorityColor).Render(fmt.Sprintf("⬥ %s", c.ticket.Priority))

	left := priorityIndicator
	right := ""

	if c.ticket.AgentActive {
		barWidth := width / 2
		if barWidth < 4 {
			barWidth = 4
		}
		bar := ActivityBar(c.frame, barWidth, c.theme)
		dot := agentDot(c.ticket.Agent, true)
		right = bar + " " + dot
	} else if c.ticket.Agent != "" {
		dot := agentDot(c.ticket.Agent, false)
		right = dot + " " + c.ticket.Agent
	}

	if right == "" {
		return left
	}

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := width - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

func truncateRunes(s string, maxLen int) string {
	if maxLen < 0 {
		maxLen = 0
	}
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen])
}

func wrapText(s string, width int) string {
	if width < 1 {
		return s
	}
	var b strings.Builder
	runes := []rune(s)
	pos := 0
	for pos < len(runes) {
		end := pos + width
		if end > len(runes) {
			end = len(runes)
		}
		b.WriteString(string(runes[pos:end]))
		if end < len(runes) {
			b.WriteString("\n")
		}
		pos = end
	}
	return b.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run TestTicketCard -v`
Expected: All PASS

- [ ] **Step 5: Run all tui tests**

Run: `go test ./internal/tui/ -v`
Expected: All PASS (existing tests may need updates if they assert on exact View output)

- [ ] **Step 6: Commit**

```bash
git add internal/tui/ticketcard.go internal/tui/ticketcard_test.go
git commit -m "feat(tui): add TicketCardModel with compact/expanded card rendering"
```

---

### Task 5: Integrate TicketCardModel into KanbanModel

**Files:**
- Modify: `internal/tui/kanban.go:32-41,110-120,172-268`
- Test: `internal/tui/kanban_test.go`

- [ ] **Step 1: Update KanbanModel to use cards and track animation**

In `internal/tui/kanban.go`, add `animFrame int` and `theme *theme.Theme` fields to the `KanbanModel` struct. The full updated struct is in the View() code block below.

Update `Update()` to handle `tickMsg` and advance animation:

```go
func (m KanbanModel) Update(msg tea.Msg) (KanbanModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tickMsg:
		m.animFrame = (m.animFrame + 1) % AnimFrames
		if m.anyAgentActive() {
			return m, animationTick()
		}
		return m, nil
	}
	return m, nil
}
```

Add helper method:

```go
func (m KanbanModel) anyAgentActive() bool {
	for _, col := range m.columns {
		for _, t := range col {
			if t.AgentActive {
				return true
			}
		}
	}
	return false
}
```

Note: `animationTick()` is already defined in `ticketer_animation.go`.

Update `Init()` to start animation if agents are active:

```go
func (m KanbanModel) Init() tea.Cmd {
	if m.anyAgentActive() {
		return animationTick()
	}
	return nil
}
```

Update `loadColumns()` to trigger animation after reload:

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

Add a `NeedsTick()` method for app.go to check:

```go
func (m KanbanModel) NeedsTick() bool {
	return m.anyAgentActive()
}
```

- [ ] **Step 2: Replace the ticket rendering loop in View()**

Replace the entire ticket rendering section in `View()` (the `for j := 0; j < len(tickets) && j < maxShow; j++` block) with card-based rendering:

```go
func (m KanbanModel) View() string {
	if m.width == 0 {
		return ""
	}

	colWidth := m.width / 4
	remainder := m.width % 4

	colInnerWidths := [4]int{}
	for i := 0; i < 4; i++ {
		w := colWidth
		if i >= 4-remainder {
			w++
		}
		colInnerWidths[i] = w - 4
		if colInnerWidths[i] < 1 {
			colInnerWidths[i] = 1
		}
	}

	availableHeight := m.height - 6
	if availableHeight < 1 {
		availableHeight = 10
	}

	cols := make([]string, 4)
	for i := 0; i < 4; i++ {
		innerWidth := colInnerWidths[i]
		var content strings.Builder

		titleStyle := m.styles.FocusedTitle
		if i != m.colIndex {
			titleStyle = m.styles.BlurredTitle
		}
		content.WriteString(titleStyle.Width(innerWidth).Render(columnNames[i]))
		content.WriteString("\n")

		tickets := m.columns[i]
		if len(tickets) == 0 {
			content.WriteString(m.styles.EmptyColumn.Render("(empty)"))
		} else {
			cardWidth := innerWidth
			lineHeight := 4
			expandedIdx := -1
			if i == m.colIndex && len(tickets) > 0 {
				expandedIdx = m.cursors[i]
			}

			maxShow := 0
			usedLines := 0
			for j := 0; j < len(tickets); j++ {
				h := lineHeight
				if j == expandedIdx {
					card := NewTicketCardModel(tickets[j], true, true, cardWidth, m.animFrame, m.theme)
					h = card.ExpandedHeight() + 1
				}
				if usedLines+h > availableHeight {
					break
				}
				usedLines += h
				maxShow = j + 1
			}

			overflow := len(tickets) > maxShow

			for j := 0; j < len(tickets) && j < maxShow; j++ {
				isSelected := i == m.colIndex && j == m.cursors[i]
				isExpanded := j == expandedIdx

				card := NewTicketCardModel(tickets[j], isSelected, isExpanded, cardWidth, m.animFrame, m.theme)
				content.WriteString(card.Render())
				content.WriteString("\n")
			}

			if overflow {
				remaining := len(tickets) - maxShow
				content.WriteString(fmt.Sprintf("↓ %d more", remaining))
			}
		}

		colStyle := m.styles.FocusedColumn
		if i != m.colIndex {
			colStyle = m.styles.BlurredColumn
		}
		colStyle = colStyle.Width(innerWidth+2).Padding(0, 1)

		cols[i] = colStyle.Render(content.String())
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cols...)
}
```

Also add `theme *theme.Theme` field to `KanbanModel` and store it in `NewKanbanModel`:

```go
type KanbanModel struct {
	store     *store.Store
	resolver  *keybinding.Resolver
	width     int
	height    int
	colIndex  int
	cursors   [4]int
	columns   [4][]store.Ticket
	styles    KanbanStyles
	animFrame int
	theme     *theme.Theme
}

func NewKanbanModel(s *store.Store, resolver *keybinding.Resolver, t *theme.Theme) (KanbanModel, error) {
	m := KanbanModel{
		store:    s,
		resolver: resolver,
		styles:   NewKanbanStyles(t),
		theme:    t,
	}
	m, err := m.loadColumns()
	if err != nil {
		return m, fmt.Errorf("kanban.newKanbanModel: %w", err)
	}
	return m, nil
}
```

- [ ] **Step 3: Run all kanban tests**

Run: `go test ./internal/tui/ -run TestKanban -v`
Expected: Some existing tests may fail because View output changed. Fix them.

The tests `TestKanbanViewRendersTickets`, `TestKanbanViewAgentDot`, `TestKanbanViewFocusedColumn`, `TestKanbanViewTruncation` assert on the old single-line format. Update assertions to check card content instead:

- `TestKanbanViewRendersTickets`: Still passes — checks for title text which is in cards
- `TestKanbanViewAgentDot`: May need update — the dot `●` is still in the card
- `TestKanbanViewFocusedColumn`: The `▸` marker is gone (replaced by selected border). Update to check for border instead.
- `TestKanbanViewTruncation`: Still passes — `…` is used in card description truncation
- `TestKanbanViewNoAgentDot`: Needs update — no `●` should appear for unassigned

Update `TestKanbanViewFocusedColumn`:

```go
func TestKanbanViewFocusedColumn(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()
	m.width = 120
	m.height = 40

	_, _ = m.store.CreateTicket(ctx, store.Ticket{Title: "Focused", Status: "backlog"})
	m, _ = m.Reload()

	view := m.View()
	if !strings.Contains(view, "Focused") {
		t.Error("view missing ticket title 'Focused' for selected ticket")
	}
	if !strings.Contains(view, "╭") {
		t.Error("view should have bordered cards")
	}
}
```

Update `TestKanbanViewNoAgentDot`:

```go
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
	if strings.Contains(view, "○") {
		t.Error("view should not contain idle dot '○' for unassigned ticket")
	}
}
```

- [ ] **Step 4: Run all tui tests after fixes**

Run: `go test ./internal/tui/ -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/kanban.go internal/tui/kanban_test.go
git commit -m "feat(tui): integrate TicketCardModel into kanban with adaptive card layout"
```

---

### Task 6: Wire animation tick through app.go

**Files:**
- Modify: `internal/tui/app.go:100-130`
- Test: `internal/tui/app_test.go`

- [ ] **Step 1: Update app.go Update() to handle tickMsg**

In `internal/tui/app.go`, add handling for `tickMsg` in the `Update()` method, right after the `editorFinishedMsg` case:

```go
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.kanban, _ = a.kanban.Update(msg)
		a.ticketView, _ = a.ticketView.Update(msg)
		a.dashboard, _ = a.dashboard.Update(msg)
		a.palette, _ = a.palette.Update(msg)
		a.modal.SetSize(a.width, a.height)
		return a, nil
	case tea.KeyMsg:
		return a.handleKey(msg)
	case editorFinishedMsg:
		return a, nil
	case tickMsg:
		if a.view == viewBoard {
			var cmd tea.Cmd
			a.kanban, cmd = a.kanban.Update(msg)
			return a, cmd
		}
		return a, nil
	}

	if a.kanban.NeedsTick() && a.view == viewBoard {
		return a, animationTick()
	}

	return a, nil
}
```

Also update `applyTheme()` to propagate theme to the kanban (it already does via styles, but the new `theme` field needs updating):

```go
func (a *App) applyTheme() {
	t := a.registry.Active()
	a.kanban.styles = NewKanbanStyles(t)
	a.kanban.theme = t
	a.ticketView.styles = NewTicketViewStyles(t)
	a.dashboard.styles = NewDashboardStyles(t)
	a.palette.SetTheme(t)
	a.modal.SetTheme(t)
}
```

- [ ] **Step 2: Run all tests**

Run: `go test ./internal/tui/ -v`
Expected: All PASS

- [ ] **Step 3: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat(tui): wire animation tick through app update loop"
```

---

### Task 7: Final verification and cleanup

**Files:**
- All modified files

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: All PASS

- [ ] **Step 2: Run go vet**

Run: `go vet ./...`
Expected: No issues

- [ ] **Step 3: Build the binary**

Run: `go build -o agentboard ./cmd/agentboard`
Expected: Build succeeds

- [ ] **Step 4: Final commit if any cleanup needed**

```bash
git add -A
git commit -m "chore: cleanup after ticket card redesign"
```
