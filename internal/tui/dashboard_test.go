package tui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func testDashboardTheme() *theme.Theme {
	return &theme.Theme{
		Primary: lipgloss.Color("69"), Text: lipgloss.Color("15"),
		TextMuted: lipgloss.Color("240"), Background: lipgloss.Color("#000"),
		BackgroundPanel: lipgloss.Color("236"), Border: lipgloss.Color("240"),
		Success: lipgloss.Color("42"), Accent: lipgloss.Color("213"),
	}
}

func newTestDashboard(t *testing.T) DashboardModel {
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
	agents := []config.DetectedAgent{
		{Name: "claude-code", Binary: "claude", Found: true},
		{Name: "opencode", Binary: "opencode", Found: true},
		{Name: "codex", Binary: "codex", Found: false},
		{Name: "cursor", Binary: "cursor", Found: false},
	}

	// Create a fake orchestrator for testing
	fo := &fakeOrchestrator{store: s}
	return NewDashboardModel(s, fo, resolver, agents, testDashboardTheme())
}

func TestNewDashboardModel(t *testing.T) {
	m := newTestDashboard(t)
	if m.store == nil {
		t.Error("store is nil")
	}
	if m.resolver == nil {
		t.Error("resolver is nil")
	}
	if len(m.Agents) != 4 {
		t.Errorf("Agents = %d, want 4", len(m.Agents))
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

	for _, name := range []string{"claude-code", "opencode"} {
		if !strings.Contains(view, name) {
			t.Errorf("view missing installed agent name %q", name)
		}
	}
}

func TestDashboardViewHidesNotFoundAgents(t *testing.T) {
	m := newTestDashboard(t)
	m.width = 120
	m.height = 40

	view := m.View()
	for _, name := range []string{"codex", "cursor"} {
		if strings.Contains(view, name) {
			t.Errorf("view should not show uninstalled agent %q", name)
		}
	}
}

func TestDashboardViewNoAgentsFound(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.Open(dbPath, []string{"backlog", "in_progress", "review", "done"}, "AGT-")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	km := keybinding.DefaultKeyMap()
	resolver := keybinding.NewResolver(km)
	agents := []config.DetectedAgent{
		{Name: "claude-code", Binary: "claude", Found: false},
	}

	m := NewDashboardModel(s, &fakeOrchestrator{store: s}, resolver, agents, testDashboardTheme())
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "No agents found") {
		t.Errorf("should show 'No agents found' when none installed, got: %s", view)
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
	m = m.Refresh()
	if !m.refreshed {
		t.Error("refreshed flag not set after Refresh()")
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

func TestDashboardShowsRunningWhenActiveSession(t *testing.T) {
	m := newTestDashboard(t)

	ctx := context.Background()
	ticket, err := m.store.CreateTicket(ctx, store.Ticket{Title: "Test task", Status: "in_progress"})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	_, err = m.store.CreateSession(ctx, store.Session{
		TicketID: ticket.ID,
		Agent:    "opencode",
		Status:   "running",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	m.width = 120
	m.height = 40
	view := m.View()
	plain := stripAnsi(view)

	if !strings.Contains(plain, "Running: yes") {
		t.Errorf("view should show 'Running: yes' for agent with active session, got:\n%s", plain)
	}
	if !strings.Contains(plain, ticket.ID) {
		t.Errorf("view should show ticket ID %q for running agent", ticket.ID)
	}
}

func TestDashboardShowsNotRunningWhenNoSession(t *testing.T) {
	m := newTestDashboard(t)
	m.width = 120
	m.height = 40

	view := m.View()
	plain := stripAnsi(view)
	if !strings.Contains(plain, "Running: no") {
		t.Errorf("view should show 'Running: no' for agents without active session, got:\n%s", plain)
	}
}

func TestDashboardShowsNotRunningAfterSessionEnds(t *testing.T) {
	m := newTestDashboard(t)

	ctx := context.Background()
	ticket, err := m.store.CreateTicket(ctx, store.Ticket{Title: "Done task", Status: "review"})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	sess, err := m.store.CreateSession(ctx, store.Session{
		TicketID: ticket.ID,
		Agent:    "opencode",
		Status:   "running",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	err = m.store.EndSession(ctx, sess.ID, "completed")
	if err != nil {
		t.Fatalf("end session: %v", err)
	}

	m.width = 120
	m.height = 40
	view := m.View()
	plain := stripAnsi(view)

	if !strings.Contains(plain, "Running: no") {
		t.Errorf("view should show 'Running: no' after session ends, got:\n%s", plain)
	}
}

func TestDashboardRefreshLoadsActiveSessions(t *testing.T) {
	m := newTestDashboard(t)

	ctx := context.Background()
	ticket, err := m.store.CreateTicket(ctx, store.Ticket{Title: "Active task", Status: "in_progress"})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	_, err = m.store.CreateSession(ctx, store.Session{
		TicketID: ticket.ID,
		Agent:    "claude",
		Status:   "running",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	m = m.Refresh()
	m.width = 120
	m.height = 40
	view := m.View()
	plain := stripAnsi(view)

	if !strings.Contains(plain, "Running: yes") {
		t.Errorf("view should show running after Refresh(), got:\n%s", plain)
	}
	if !strings.Contains(plain, ticket.ID) {
		t.Errorf("view should show ticket ID %q after Refresh()", ticket.ID)
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name string
		age  string
		want string
	}{
		{"30s", "30s", "30s"},
		{"90s", "90s", "1m 30s"},
		{"3700s", "3700s", "1h 1m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			since := parseDuration(t, tt.age)
			got := formatUptime(since)
			if got != tt.want {
				t.Errorf("formatUptime(%s) = %q, want %q", tt.age, got, tt.want)
			}
		})
	}
}

func parseDuration(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.ParseDuration(s)
	if err != nil {
		t.Fatalf("parse duration %q: %v", s, err)
	}
	return time.Now().Add(-d)
}
