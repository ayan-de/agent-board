package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"
)

func testTheme() *theme.Theme {
	return &theme.Theme{
		Primary: lipgloss.Color("69"), Text: lipgloss.Color("15"),
		TextMuted: lipgloss.Color("240"), Background: lipgloss.Color("#000"),
		BackgroundPanel: lipgloss.Color("236"), Border: lipgloss.Color("240"),
		Success: lipgloss.Color("42"), Accent: lipgloss.Color("213"),
	}
}

func newTestKanban(t *testing.T) KanbanModel {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.Open(dbPath, []string{"backlog", "in_progress", "review", "done"}, "AGT-")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	km := keybinding.DefaultKeyMap()
	resolver := keybinding.NewResolver(km)

	model, err := NewKanbanModel(s, resolver, testTheme())
	if err != nil {
		t.Fatalf("new kanban model: %v", err)
	}
	return model
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
		t.Errorf("Init() = %v, want nil", cmd)
	}
}

func TestDefaultKanbanStyles(t *testing.T) {
	s := DefaultKanbanStyles()

	styles := []struct {
		name  string
		style lipgloss.Style
	}{
		{"FocusedColumn", s.FocusedColumn},
		{"BlurredColumn", s.BlurredColumn},
		{"FocusedTitle", s.FocusedTitle},
		{"BlurredTitle", s.BlurredTitle},
		{"SelectedTicket", s.SelectedTicket},
		{"Ticket", s.Ticket},
		{"EmptyColumn", s.EmptyColumn},
	}
	for _, tt := range styles {
		rendered := tt.style.Render("test")
		if rendered == "" {
			t.Errorf("%s rendered empty string", tt.name)
		}
		if !strings.Contains(rendered, "test") {
			t.Errorf("%s render missing input text", tt.name)
		}
	}
}

func TestKanbanWindowSize(t *testing.T) {
	m := newTestKanban(t)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if updated.width != 120 {
		t.Errorf("width = %d, want 120", updated.width)
	}
	if updated.height != 40 {
		t.Errorf("height = %d, want 40", updated.height)
	}
}

func TestKanbanColumnNavigation(t *testing.T) {
	m := newTestKanban(t)

	nextKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	for i := 0; i < 4; i++ {
		m, _ = m.Update(nextKey)
	}
	if m.colIndex != 3 {
		t.Errorf("colIndex = %d after 4 next, want 3", m.colIndex)
	}

	prevKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	for i := 0; i < 5; i++ {
		m, _ = m.Update(prevKey)
	}
	if m.colIndex != 0 {
		t.Errorf("colIndex = %d after 5 prev, want 0", m.colIndex)
	}

	tests := []struct {
		key  rune
		want int
	}{
		{'3', 2},
		{'1', 0},
		{'4', 3},
	}
	for _, tt := range tests {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tt.key}})
		if m.colIndex != tt.want {
			t.Errorf("after pressing %c, colIndex = %d, want %d", tt.key, m.colIndex, tt.want)
		}
	}
}

func TestKanbanTicketNavigation(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, _ = m.store.CreateTicket(ctx, store.Ticket{
			Title:  fmt.Sprintf("Ticket %d", i+1),
			Status: "backlog",
		})
	}
	m, _ = m.Reload()

	nextKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m, _ = m.Update(nextKey)
	if m.cursors[0] != 1 {
		t.Errorf("cursor = %d, want 1", m.cursors[0])
	}
	m, _ = m.Update(nextKey)
	if m.cursors[0] != 2 {
		t.Errorf("cursor = %d, want 2", m.cursors[0])
	}
	m, _ = m.Update(nextKey)
	if m.cursors[0] != 2 {
		t.Errorf("cursor = %d after clamp, want 2", m.cursors[0])
	}

	prevKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m, _ = m.Update(prevKey)
	if m.cursors[0] != 1 {
		t.Errorf("cursor = %d, want 1", m.cursors[0])
	}
	m, _ = m.Update(prevKey)
	if m.cursors[0] != 0 {
		t.Errorf("cursor = %d, want 0", m.cursors[0])
	}
	m, _ = m.Update(prevKey)
	if m.cursors[0] != 0 {
		t.Errorf("cursor = %d after clamp, want 0", m.cursors[0])
	}
}

func TestKanbanTicketNavigationEmptyColumn(t *testing.T) {
	m := newTestKanban(t)
	m.colIndex = 1
	nextKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m, _ = m.Update(nextKey)
	if m.cursors[1] != 0 {
		t.Errorf("cursor = %d on empty column, want 0", m.cursors[1])
	}
}

func TestKanbanAddTicket(t *testing.T) {
	m := newTestKanban(t)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if len(m.columns[0]) != 1 {
		t.Fatalf("backlog has %d tickets, want 1", len(m.columns[0]))
	}
	if m.columns[0][0].Title != "New Ticket" {
		t.Errorf("title = %q, want %q", m.columns[0][0].Title, "New Ticket")
	}
}

func TestKanbanDeleteTicket(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()

	_, _ = m.store.CreateTicket(ctx, store.Ticket{Title: "To Delete", Status: "backlog"})
	m, _ = m.Reload()

	if len(m.columns[0]) != 1 {
		t.Fatalf("setup: backlog has %d tickets, want 1", len(m.columns[0]))
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Fatal("delete key should return a command")
	}
	msg := cmd()
	_, ok := msg.(deleteTicketRequestMsg)
	if !ok {
		t.Errorf("delete key should return deleteTicketRequestMsg, got %T", msg)
	}
}

func TestKanbanDeleteTicketEmptyColumn(t *testing.T) {
	m := newTestKanban(t)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd != nil {
		t.Error("delete key on empty column should not return a command")
	}
}

func TestKanbanSelectedTicket(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()

	if m.SelectedTicket() != nil {
		t.Error("SelectedTicket() should be nil on empty board")
	}

	ticket, _ := m.store.CreateTicket(ctx, store.Ticket{Title: "Selected", Status: "backlog"})
	m, _ = m.Reload()

	selected := m.SelectedTicket()
	if selected == nil {
		t.Fatal("SelectedTicket() is nil, want a ticket")
	}
	if selected.ID != ticket.ID {
		t.Errorf("SelectedTicket().ID = %q, want %q", selected.ID, ticket.ID)
	}
}

func TestKanbanReload(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()

	if len(m.columns[0]) != 0 {
		t.Fatalf("backlog has %d tickets, want 0 initially", len(m.columns[0]))
	}

	_, _ = m.store.CreateTicket(ctx, store.Ticket{Title: "External", Status: "backlog"})
	m, err := m.Reload()
	if err != nil {
		t.Fatalf("Reload() error: %v", err)
	}

	if len(m.columns[0]) != 1 {
		t.Errorf("backlog has %d tickets after reload, want 1", len(m.columns[0]))
	}
	if m.columns[0][0].Title != "External" {
		t.Errorf("title = %q, want %q", m.columns[0][0].Title, "External")
	}
}

func TestKanbanViewRendersColumns(t *testing.T) {
	m := newTestKanban(t)
	m.width = 120
	m.height = 40

	view := m.View()
	for _, name := range columnNames {
		if !strings.Contains(view, name) {
			t.Errorf("view missing column name %q", name)
		}
	}
}

func TestKanbanViewRendersTickets(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()
	m.width = 120
	m.height = 40

	_, _ = m.store.CreateTicket(ctx, store.Ticket{Title: "First Task", Status: "backlog"})
	_, _ = m.store.CreateTicket(ctx, store.Ticket{Title: "Second Task", Status: "in_progress"})
	m, _ = m.Reload()

	view := m.View()
	if !strings.Contains(view, "First Task") {
		t.Error("view missing ticket title 'First Task'")
	}
	if !strings.Contains(view, "Second Task") {
		t.Error("view missing ticket title 'Second Task'")
	}
	if !strings.Contains(view, "AGT-01") {
		t.Error("view missing ticket ID 'AGT-01'")
	}
}

func TestKanbanViewEmptyColumn(t *testing.T) {
	m := newTestKanban(t)
	m.width = 120
	m.height = 40

	view := m.View()
	if !strings.Contains(view, "(empty)") {
		t.Error("view missing '(empty)' for empty columns")
	}
}

func TestKanbanViewTruncation(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()
	m.width = 80
	m.height = 40

	longTitle := strings.Repeat("A", 100)
	_, _ = m.store.CreateTicket(ctx, store.Ticket{
		Title:  longTitle,
		Status: "in_progress",
	})
	m, _ = m.Reload()

	view := m.View()
	if !strings.Contains(view, "╭") {
		t.Error("view should contain card borders")
	}
}

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

func TestKanbanViewAgentDot(t *testing.T) {
	m := newTestKanban(t)
	ctx := context.Background()
	m.width = 120
	m.height = 40

	_, _ = m.store.CreateTicket(ctx, store.Ticket{
		Title:       "Agent Dot Test",
		Status:      "backlog",
		Agent:       "claude-code",
		AgentActive: true,
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
	if strings.Contains(view, "○") {
		t.Error("view should not contain idle dot '○' for unassigned ticket")
	}
}
