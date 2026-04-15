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
	if len(m.agents) != 4 {
		t.Errorf("agents = %d, want 4", len(m.agents))
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

	for _, name := range []string{"claude-code", "opencode", "codex", "cursor"} {
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
