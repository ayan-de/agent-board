package tui

import (
	"path/filepath"
	"testing"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"
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
