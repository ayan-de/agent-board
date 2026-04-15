package tui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func newTestTicketView(t *testing.T) (TicketViewModel, *store.Store) {
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

	defaultTheme := &theme.Theme{
		Primary: lipgloss.Color("69"), Text: lipgloss.Color("15"),
		TextMuted: lipgloss.Color("240"), Background: lipgloss.Color("#000"),
		BackgroundPanel: lipgloss.Color("236"), Border: lipgloss.Color("240"),
		Success: lipgloss.Color("42"), Accent: lipgloss.Color("213"),
	}

	m := NewTicketViewModel(s, resolver, defaultTheme)
	return m, s
}

func TestNewTicketViewModel(t *testing.T) {
	m, _ := newTestTicketView(t)

	if m.store == nil {
		t.Error("store is nil")
	}
	if m.resolver == nil {
		t.Error("resolver is nil")
	}
	if m.mode != ticketViewMode {
		t.Errorf("mode = %v, want ticketViewMode", m.mode)
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestTicketViewModelInit(t *testing.T) {
	m, _ := newTestTicketView(t)
	cmd := m.Init()
	if cmd != nil {
		t.Errorf("Init() = %v, want nil", cmd)
	}
}

func TestTicketViewModelSetTicket(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:       "Test Ticket",
		Description: "A description",
		Status:      "backlog",
		Priority:    "high",
	})

	m = m.SetTicket(&ticket)
	if m.ticket == nil {
		t.Fatal("ticket is nil after SetTicket")
	}
	if m.ticket.ID != ticket.ID {
		t.Errorf("ticket.ID = %q, want %q", m.ticket.ID, ticket.ID)
	}
}

func TestTicketViewModelWindowSize(t *testing.T) {
	m, _ := newTestTicketView(t)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.width != 100 {
		t.Errorf("width = %d, want 100", m.width)
	}
	if m.height != 30 {
		t.Errorf("height = %d, want 30", m.height)
	}
}

func TestTicketViewModelFieldNavigation(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:       "Nav Test",
		Description: "Desc",
		Status:      "backlog",
	})
	m = m.SetTicket(&ticket)

	fieldCount := len(m.fields)
	if fieldCount == 0 {
		t.Fatal("no fields defined")
	}

	nextKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	for i := 0; i < fieldCount+2; i++ {
		m, _ = m.Update(nextKey)
	}
	if m.cursor != fieldCount-1 {
		t.Errorf("cursor = %d after overflow, want %d", m.cursor, fieldCount-1)
	}

	prevKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	for i := 0; i < fieldCount+2; i++ {
		m, _ = m.Update(prevKey)
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d after underflow, want 0", m.cursor)
	}
}

func TestTicketViewModelViewRendersFields(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:       "Render Me",
		Description: "Some details here",
		Status:      "in_progress",
		Priority:    "high",
		Agent:       "claude-code",
		Branch:      "feat/auth",
	})
	m = m.SetTicket(&ticket)

	view := m.View()
	if view == "" {
		t.Fatal("view is empty")
	}

	checks := []string{
		ticket.ID,
		"Render Me",
		"Some details here",
		"in_progress",
		"high",
		"claude-code",
		"feat/auth",
	}
	for _, want := range checks {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestTicketViewModelViewShowsCursor(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Cursor Test",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	view := m.View()
	if !strings.Contains(view, "▸") {
		t.Error("view missing cursor marker '▸'")
	}
}

func TestTicketViewModelCycleStatus(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Status Cycle",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	if m.ticket.Status != "in_progress" {
		t.Errorf("status = %q after one cycle, want %q", m.ticket.Status, "in_progress")
	}

	view := m.View()
	if !strings.Contains(view, "in_progress") {
		t.Error("view does not show updated status")
	}
}

func TestTicketViewModelCycleStatusWraps(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Wrap Test",
		Status: "done",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if m.ticket.Status != "backlog" {
		t.Errorf("status = %q after wrap, want %q", m.ticket.Status, "backlog")
	}
}

func TestTicketViewModelCycleStatusPersists(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Persist Test",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	loaded, err := s.GetTicket(ctx, ticket.ID)
	if err != nil {
		t.Fatalf("GetTicket: %v", err)
	}
	if loaded.Status != "in_progress" {
		t.Errorf("persisted status = %q, want %q", loaded.Status, "in_progress")
	}
}

func TestTicketViewModelEditTitle(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Old Title",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	if m.mode != ticketEditMode {
		t.Errorf("mode = %v, want ticketEditMode", m.mode)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})

	view := m.View()
	if !strings.Contains(view, "Old TitleX") {
		t.Errorf("edit buffer not rendered, got: %s", view)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.mode != ticketViewMode {
		t.Errorf("mode = %v after enter, want ticketViewMode", m.mode)
	}
	if m.ticket.Title != "Old TitleX" {
		t.Errorf("title = %q, want %q", m.ticket.Title, "Old TitleX")
	}

	loaded, _ := s.GetTicket(ctx, ticket.ID)
	if loaded.Title != "Old TitleX" {
		t.Errorf("persisted title = %q, want %q", loaded.Title, "Old TitleX")
	}
}

func TestTicketViewModelEditCancel(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Keep This",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Z'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if m.mode != ticketViewMode {
		t.Errorf("mode = %v after cancel, want ticketViewMode", m.mode)
	}
	if m.ticket.Title != "Keep This" {
		t.Errorf("title = %q after cancel, want %q", m.ticket.Title, "Keep This")
	}
}

func TestTicketViewModelEditBackspace(t *testing.T) {
	m, _ := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := m.store.CreateTicket(ctx, store.Ticket{
		Title:  "Hello",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})

	if m.editBuffer != "Hell" {
		t.Errorf("editBuffer = %q after backspace, want %q", m.editBuffer, "Hell")
	}
}

func TestTicketViewModelEditDescriptionField(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:       "Desc Edit",
		Description: "Old desc",
		Status:      "backlog",
	})
	m = m.SetTicket(&ticket)

	descIdx := -1
	for i, f := range m.fields {
		if f.label == "Description" {
			descIdx = i
			break
		}
	}
	if descIdx < 0 {
		t.Fatal("Description field not found")
	}

	m.cursor = descIdx
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	if m.mode != ticketEditMode {
		t.Errorf("mode = %v, want ticketEditMode", m.mode)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.ticket.Description != "Old desc!" {
		t.Errorf("description = %q, want %q", m.ticket.Description, "Old desc!")
	}
}

func TestTicketViewModelViewNoTicket(t *testing.T) {
	m, _ := newTestTicketView(t)
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "No ticket") {
		t.Errorf("view without ticket should say 'No ticket', got: %q", view)
	}
}

func TestTicketViewModelEscReturnsToViewMode(t *testing.T) {
	m, _ := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := m.store.CreateTicket(ctx, store.Ticket{
		Title:  "Esc Test",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m.mode != ticketEditMode {
		t.Fatalf("mode = %v, want ticketEditMode", m.mode)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.mode != ticketViewMode {
		t.Errorf("mode = %v after esc in edit, want ticketViewMode", m.mode)
	}
}

func TestTicketViewModelTagsDisplayed(t *testing.T) {
	m, _ := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := m.store.CreateTicket(ctx, store.Ticket{
		Title:  "Tagged",
		Status: "backlog",
		Tags:   []string{"auth", "api"},
	})
	m = m.SetTicket(&ticket)

	view := m.View()
	if !strings.Contains(view, "auth") || !strings.Contains(view, "api") {
		t.Errorf("view missing tags, got: %s", view)
	}
}

func TestTicketViewModelDependsOnDisplayed(t *testing.T) {
	m, _ := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := m.store.CreateTicket(ctx, store.Ticket{
		Title:     "Dep Test",
		Status:    "backlog",
		DependsOn: []string{"AGT-01", "AGT-02"},
	})
	m = m.SetTicket(&ticket)

	view := m.View()
	if !strings.Contains(view, "AGT-01") || !strings.Contains(view, "AGT-02") {
		t.Errorf("view missing depends_on, got: %s", view)
	}
}

func TestTicketViewModelTimestampsDisplayed(t *testing.T) {
	m, _ := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := m.store.CreateTicket(ctx, store.Ticket{
		Title:  "Time Test",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	view := m.View()
	if !strings.Contains(view, "Created") || !strings.Contains(view, "Updated") {
		t.Error("view missing timestamp labels")
	}
}

func TestTicketViewModelEditNonEditableField(t *testing.T) {
	m, _ := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := m.store.CreateTicket(ctx, store.Ticket{
		Title:  "No Edit ID",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	for i, f := range m.fields {
		if f.label == "ID" && !f.editable {
			m.cursor = i
			m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
			if m.mode == ticketEditMode {
				t.Error("should not enter edit mode for non-editable field")
			}
			return
		}
	}
}
