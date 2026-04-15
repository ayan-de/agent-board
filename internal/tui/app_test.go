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
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := app.View()
	if len(view) == 0 {
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
