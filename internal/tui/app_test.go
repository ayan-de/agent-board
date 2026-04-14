package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
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
	if app.resolver == nil {
		t.Error("resolver is nil")
	}
	if app.store == nil {
		t.Error("store is nil")
	}
	if app.view != viewBoard {
		t.Errorf("view = %v, want viewBoard", app.view)
	}
	if app.focus != focusBoard {
		t.Errorf("focus = %v, want focusBoard", app.focus)
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
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	cfg := config.SetDefaults()
	cfg.TUI.Keybindings = map[string]string{"quit": "Q"}

	app, err := NewApp(cfg, s)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	action, _ := app.resolver.Resolve("Q")
	if action != keybinding.ActionQuit {
		t.Errorf("resolve Q = %v, want ActionQuit", action)
	}
}

func TestNewAppLoadsColumns(t *testing.T) {
	app := newTestApp(t)

	for i, col := range app.columns {
		if col == nil {
			t.Errorf("columns[%d] is nil", i)
		}
	}
}

func TestUpdateWindowSize(t *testing.T) {
	app := newTestApp(t)
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(*App)
	if model.width != 120 {
		t.Errorf("width = %d, want 120", model.width)
	}
	if model.height != 40 {
		t.Errorf("height = %d, want 40", model.height)
	}
}

func TestInit(t *testing.T) {
	app := newTestApp(t)
	cmd := app.Init()
	if cmd == nil {
		t.Error("Init() returned nil cmd")
	}
}

func TestUpdateQuit(t *testing.T) {
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

func TestUpdateForceQuit(t *testing.T) {
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

func TestUpdateNavigationPrevNextColumn(t *testing.T) {
	app := newTestApp(t)

	nextKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	for i := 0; i < 4; i++ {
		app.Update(nextKey)
	}
	if app.colIndex != 3 {
		t.Errorf("colIndex = %d after 4 next, want 3", app.colIndex)
	}

	prevKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	for i := 0; i < 5; i++ {
		app.Update(prevKey)
	}
	if app.colIndex != 0 {
		t.Errorf("colIndex = %d after 5 prev, want 0", app.colIndex)
	}
}

func TestUpdateNavigationJumpColumns(t *testing.T) {
	app := newTestApp(t)

	tests := []struct {
		key  rune
		want int
	}{
		{'3', 2},
		{'1', 0},
		{'4', 3},
	}
	for _, tt := range tests {
		app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tt.key}})
		if app.colIndex != tt.want {
			t.Errorf("after pressing %c, colIndex = %d, want %d", tt.key, app.colIndex, tt.want)
		}
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

	nextKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	app.Update(nextKey)
	if app.cursors[0] != 1 {
		t.Errorf("cursor = %d, want 1", app.cursors[0])
	}
	app.Update(nextKey)
	if app.cursors[0] != 2 {
		t.Errorf("cursor = %d, want 2", app.cursors[0])
	}
	app.Update(nextKey)
	if app.cursors[0] != 2 {
		t.Errorf("cursor = %d after clamp, want 2", app.cursors[0])
	}

	prevKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	app.Update(prevKey)
	if app.cursors[0] != 1 {
		t.Errorf("cursor = %d, want 1", app.cursors[0])
	}
	app.Update(prevKey)
	if app.cursors[0] != 0 {
		t.Errorf("cursor = %d, want 0", app.cursors[0])
	}
	app.Update(prevKey)
	if app.cursors[0] != 0 {
		t.Errorf("cursor = %d after clamp, want 0", app.cursors[0])
	}
}

func TestUpdateNavigationEmptyColumn(t *testing.T) {
	app := newTestApp(t)

	app.colIndex = 1
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if app.cursors[1] != 0 {
		t.Errorf("cursor = %d on empty column, want 0", app.cursors[1])
	}
}

func TestUpdateAddTicket(t *testing.T) {
	app := newTestApp(t)

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if len(app.columns[0]) != 1 {
		t.Fatalf("backlog has %d tickets, want 1", len(app.columns[0]))
	}
	if app.columns[0][0].Title != "New Ticket" {
		t.Errorf("title = %q, want %q", app.columns[0][0].Title, "New Ticket")
	}
}

func TestUpdateDeleteTicket(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()

	_, _ = app.store.CreateTicket(ctx, store.Ticket{Title: "To Delete", Status: "backlog"})
	_ = app.loadColumns()

	if len(app.columns[0]) != 1 {
		t.Fatalf("setup: backlog has %d tickets, want 1", len(app.columns[0]))
	}

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	if len(app.columns[0]) != 0 {
		t.Errorf("backlog has %d tickets after delete, want 0", len(app.columns[0]))
	}
}

func TestUpdateDeleteTicketEmptyColumn(t *testing.T) {
	app := newTestApp(t)

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	if len(app.columns[0]) != 0 {
		t.Errorf("backlog has %d tickets, want 0", len(app.columns[0]))
	}
}

func TestUpdateOpenTicket(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()

	ticket, _ := app.store.CreateTicket(ctx, store.Ticket{Title: "Open Me", Status: "backlog"})
	_ = app.loadColumns()

	app.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if app.view != viewTicket {
		t.Errorf("view = %v, want viewTicket", app.view)
	}
	if app.activeTicket == nil || app.activeTicket.ID != ticket.ID {
		t.Errorf("activeTicket.ID = %v, want %s", app.activeTicket, ticket.ID)
	}
}

func TestUpdateOpenTicketEmptyColumn(t *testing.T) {
	app := newTestApp(t)

	app.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if app.view != viewBoard {
		t.Errorf("view = %v, want viewBoard", app.view)
	}
}

func TestUpdateShowHelp(t *testing.T) {
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
