package tui

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/orchestrator"
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
	fo := newFakeOrchestrator(s)
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

	sidebar := m.renderSidebar(30)
	sidebarPlain := stripAnsi(sidebar)
	if !regexp.MustCompile(`(?m)^\s*▸?\s*claude-code\s+0\b`).MatchString(sidebarPlain) {
		t.Errorf("sidebar should show installed claude-code with count 0, got:\n%s", sidebarPlain)
	}
	if !regexp.MustCompile(`(?m)^\s*▸?\s*opencode\s+0\b`).MatchString(sidebarPlain) {
		t.Errorf("sidebar should show installed opencode with count 0, got:\n%s", sidebarPlain)
	}
	if strings.Contains(sidebarPlain, "No active sessions") {
		t.Errorf("sidebar should not show 'No active sessions' when installed agents exist, got:\n%s", sidebarPlain)
	}

	plain := stripAnsi(view)
	if !strings.Contains(plain, "No active sessions") {
		t.Errorf("content pane should show 'No active sessions' for idle default selection, got:\n%s", plain)
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

	m := NewDashboardModel(s, newFakeOrchestrator(s), resolver, agents, testDashboardTheme())
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "No agents installed") {
		t.Errorf("should show 'No agents installed' when no agent is installed, got: %s", view)
	}
	if !strings.Contains(view, "No agent selected") {
		t.Errorf("should show 'No agent selected' when no agent is installed, got: %s", view)
	}
}

func TestDashboardViewRendersStatusLabels(t *testing.T) {
	m := newTestDashboard(t)
	m.width = 120
	m.height = 40

	view := m.View()
	plain := stripAnsi(view)
	if !regexp.MustCompile(`(?m)^\s*▸?\s*claude-code\s+0\b`).MatchString(plain) {
		t.Errorf("expected sidebar to show installed claude-code with count 0, got:\n%s", plain)
	}
}

func TestDashboardViewRendersEmDash(t *testing.T) {
	m := newTestDashboard(t)
	m.width = 120
	m.height = 40

	view := m.View()
	plain := stripAnsi(view)
	if !regexp.MustCompile(`(?m)^\s*▸?\s*opencode\s+0\b`).MatchString(plain) {
		t.Errorf("view should show installed opencode with count 0, got:\n%s", plain)
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

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	if !m.refreshed {
		t.Error("refresh flag not set after pressing R")
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

	// Select the opencode group so the right pane shows its session.
	for i, g := range m.aggregateActiveSessions() {
		if g.Agent == "opencode" {
			m.cursor = i
			break
		}
	}

	m.width = 120
	m.height = 40
	view := m.View()
	plain := stripAnsi(view)

	// claude-code is installed but has zero sessions → shown with count 0
	if !regexp.MustCompile(`(?m)^\s*▸?\s*claude-code\s+0\b`).MatchString(plain) {
		t.Errorf("view should show installed claude-code with count 0, got:\n%s", plain)
	}
	// opencode has one session → shown with its display name and count 1
	if !regexp.MustCompile(`(?m)^\s*▸?\s*opencode\s+1\b`).MatchString(plain) {
		t.Errorf("view should show running opencode with count 1, got:\n%s", plain)
	}
	if !strings.Contains(plain, "Test task") {
		t.Errorf("view should show the ticket title, got:\n%s", plain)
	}
	if !strings.Contains(plain, ticket.ID) {
		t.Errorf("view should show ticket ID %s, got:\n%s", ticket.ID, plain)
	}
}

func TestDashboardShowsNotRunningWhenNoSession(t *testing.T) {
	m := newTestDashboard(t)
	m.width = 120
	m.height = 40

	view := m.View()
	plain := stripAnsi(view)
	if !regexp.MustCompile(`(?m)^\s*▸?\s*claude-code\s+0\b`).MatchString(plain) {
		t.Errorf("view should show installed claude-code with count 0 when no agents are running, got:\n%s", plain)
	}
	if !regexp.MustCompile(`(?m)^\s*▸?\s*opencode\s+0\b`).MatchString(plain) {
		t.Errorf("view should show installed opencode with count 0 when no agents are running, got:\n%s", plain)
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

	if !regexp.MustCompile(`(?m)^\s*▸?\s*opencode\s+0\b`).MatchString(plain) {
		t.Errorf("view should show installed opencode with count 0 after session ends, got:\n%s", plain)
	}
	if strings.Contains(plain, "Done task") {
		t.Errorf("view should not list the ended session's ticket title, got:\n%s", plain)
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

	// After Refresh with active session, the sidebar should list claude-code
	// with a count of 1.
	if !regexp.MustCompile(`(?m)^\s*▸?\s*claude-code\s+1\b`).MatchString(plain) {
		t.Errorf("view should show 'claude-code' with count 1 in sidebar, got:\n%s", plain)
	}
	if !strings.Contains(plain, "Active task") {
		t.Errorf("view should show ticket title, got:\n%s", plain)
	}
	if !strings.Contains(plain, ticket.ID) {
		t.Errorf("view should show ticket ID %s, got:\n%s", ticket.ID, plain)
	}
}

func TestAggregateActiveSessionsGroupsByAgent(t *testing.T) {
	m := newTestDashboard(t)
	m.activeAgentSessions = []*orchestrator.AgentSession{
		{SessionID: "s1", TicketID: "AGT-01", Agent: "claude"},
		{SessionID: "s2", TicketID: "AGT-02", Agent: "freecode"},
		{SessionID: "s3", TicketID: "AGT-03", Agent: "claude"},
		{SessionID: "s4", TicketID: "AGT-04", Agent: "opencode"},
		{SessionID: "s5", TicketID: "AGT-05", Agent: "freecode"},
	}

	groups := m.aggregateActiveSessions()

	want := []agentSessionGroup{
		{Agent: "claude", Sessions: []*orchestrator.AgentSession{m.activeAgentSessions[0], m.activeAgentSessions[2]}},
		{Agent: "opencode", Sessions: []*orchestrator.AgentSession{m.activeAgentSessions[3]}},
		{Agent: "freecode", Sessions: []*orchestrator.AgentSession{m.activeAgentSessions[1], m.activeAgentSessions[4]}},
	}
	if len(groups) != len(want) {
		t.Fatalf("got %d groups, want %d", len(groups), len(want))
	}
	for i, g := range groups {
		if g.Agent != want[i].Agent {
			t.Errorf("group %d agent = %q, want %q", i, g.Agent, want[i].Agent)
		}
		if len(g.Sessions) != len(want[i].Sessions) {
			t.Errorf("group %d sessions = %d, want %d", i, len(g.Sessions), len(want[i].Sessions))
			continue
		}
		for j, s := range g.Sessions {
			if s.SessionID != want[i].Sessions[j].SessionID {
				t.Errorf("group %d session %d = %q, want %q", i, j, s.SessionID, want[i].Sessions[j].SessionID)
			}
		}
	}
}

func TestAggregateActiveSessionsEmptyWhenNoAgentsInstalled(t *testing.T) {
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
		{Name: "codex", Binary: "codex", Found: false},
		{Name: "cursor", Binary: "cursor", Found: false},
	}

	m := NewDashboardModel(s, newFakeOrchestrator(s), resolver, agents, testDashboardTheme())
	groups := m.aggregateActiveSessions()
	if len(groups) != 0 {
		t.Errorf("expected 0 groups when no agents installed, got %d", len(groups))
	}
}

func TestAggregateActiveSessionsIncludesInstalledAgentsWithZeroSessions(t *testing.T) {
	m := newTestDashboard(t)
	m.activeAgentSessions = []*orchestrator.AgentSession{
		{SessionID: "s1", TicketID: "AGT-01", Agent: "opencode"},
	}
	groups := m.aggregateActiveSessions()

	byAgent := make(map[string]int, len(groups))
	for _, g := range groups {
		byAgent[g.Agent] = len(g.Sessions)
	}
	if byAgent["claude"] != 0 {
		t.Errorf("expected installed claude (0 sessions) in groups, got %d", byAgent["claude"])
	}
	if byAgent["opencode"] != 1 {
		t.Errorf("expected opencode with 1 session, got %d", byAgent["opencode"])
	}
	// Not-found agents must still be excluded.
	if _, ok := byAgent["codex"]; ok {
		t.Error("codex (not installed) should not appear in groups")
	}
	if _, ok := byAgent["cursor"]; ok {
		t.Error("cursor (not installed) should not appear in groups")
	}
}

func TestDashboardSidebarShowsInstalledAgentsEvenWhenIdle(t *testing.T) {
	m := newTestDashboard(t)
	m.width = 120
	m.height = 40

	sidebar := m.renderSidebar(30)
	plain := stripAnsi(sidebar)

	if !regexp.MustCompile(`(?m)^\s*▸?\s*claude-code\s+0\b`).MatchString(plain) {
		t.Errorf("sidebar should show installed claude-code with count 0, got:\n%s", plain)
	}
	if !regexp.MustCompile(`(?m)^\s*▸?\s*opencode\s+0\b`).MatchString(plain) {
		t.Errorf("sidebar should show installed opencode with count 0, got:\n%s", plain)
	}
	if strings.Contains(plain, "No active sessions") {
		t.Errorf("sidebar should not show 'No active sessions' placeholder when installed agents exist, got:\n%s", plain)
	}
	// codex/cursor are not installed -> still hidden.
	if strings.Contains(plain, "codex") {
		t.Errorf("sidebar should not show uninstalled codex, got:\n%s", plain)
	}
	if strings.Contains(plain, "cursor") {
		t.Errorf("sidebar should not show uninstalled cursor, got:\n%s", plain)
	}
}

func TestDashboardSidebarHidesIdleAgentsAndShowsCounts(t *testing.T) {
	m := newTestDashboard(t)
	ctx := context.Background()

	// Two claude sessions
	for _, title := range []string{"task a", "task b"} {
		tk, err := m.store.CreateTicket(ctx, store.Ticket{Title: title, Status: "in_progress"})
		if err != nil {
			t.Fatalf("create ticket: %v", err)
		}
		if _, err := m.store.CreateSession(ctx, store.Session{TicketID: tk.ID, Agent: "claude", Status: "running"}); err != nil {
			t.Fatalf("create session: %v", err)
		}
	}
	// One opencode session
	tk, err := m.store.CreateTicket(ctx, store.Ticket{Title: "task c", Status: "in_progress"})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if _, err := m.store.CreateSession(ctx, store.Session{TicketID: tk.ID, Agent: "opencode", Status: "running"}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	m.width = 120
	m.height = 40
	view := m.View()
	plain := stripAnsi(view)

	// claude should appear with count 2 on the same line
	if !regexp.MustCompile(`(?m)^\s*▸?\s*claude-code\s+2\b`).MatchString(plain) {
		t.Errorf("expected sidebar to show 'claude-code' with count 2 on its own line, got:\n%s", plain)
	}
	// opencode should appear with count 1
	if !regexp.MustCompile(`(?m)^\s*▸?\s*opencode\s+1\b`).MatchString(plain) {
		t.Errorf("expected sidebar to show 'opencode' with count 1 on its own line, got:\n%s", plain)
	}
	// codex is not found AND has no sessions → hidden entirely
	if strings.Contains(plain, "codex") {
		t.Errorf("expected 'codex' (not found, idle) to be hidden, got:\n%s", plain)
	}
	// cursor is not found AND has no sessions → hidden entirely
	if strings.Contains(plain, "cursor") {
		t.Errorf("expected 'cursor' (not found, idle) to be hidden, got:\n%s", plain)
	}
}

func TestDashboardContentListsSessionsFilteredBySelectedAgent(t *testing.T) {
	m := newTestDashboard(t)
	ctx := context.Background()

	for _, spec := range []struct{ title, agent string }{
		{"first claude job", "claude"},
		{"second claude job", "claude"},
		{"one opencode job", "opencode"},
	} {
		tk, err := m.store.CreateTicket(ctx, store.Ticket{Title: spec.title, Status: "in_progress"})
		if err != nil {
			t.Fatalf("create ticket: %v", err)
		}
		if _, err := m.store.CreateSession(ctx, store.Session{TicketID: tk.ID, Agent: spec.agent, Status: "running"}); err != nil {
			t.Fatalf("create session: %v", err)
		}
	}

	m = m.Refresh()
	m.width = 120
	m.height = 40

	// Default selection lands on the first agent group (claude).
	view := m.View()
	plain := stripAnsi(view)

	if !strings.Contains(plain, "first claude job") {
		t.Errorf("expected 'first claude job' in claude session list, got:\n%s", plain)
	}
	if !strings.Contains(plain, "second claude job") {
		t.Errorf("expected 'second claude job' in claude session list, got:\n%s", plain)
	}
	if !strings.Contains(plain, "AGT-") {
		t.Errorf("expected ticket IDs to render, got:\n%s", plain)
	}
	if strings.Contains(plain, "one opencode job") {
		t.Errorf("opencode session should be filtered out when claude selected, got:\n%s", plain)
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
