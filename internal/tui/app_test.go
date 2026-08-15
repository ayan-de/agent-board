package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/orchestrator"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type fakeOrchestrator struct {
	store                      *store.Store
	lastCreateProposalTicketID string
	completionCh               chan orchestrator.RunCompletion
}

func newFakeOrchestrator(s *store.Store) *fakeOrchestrator {
	return &fakeOrchestrator{
		store:        s,
		completionCh: make(chan orchestrator.RunCompletion, 16),
	}
}

func (f *fakeOrchestrator) CreateProposal(ctx context.Context, input orchestrator.CreateProposalInput) (store.Proposal, error) {
	f.lastCreateProposalTicketID = input.TicketID
	p := store.Proposal{
		TicketID: input.TicketID,
		Status:   "pending",
		Prompt:   "Proposed work",
	}
	return f.store.CreateProposal(ctx, p)
}

func (f *fakeOrchestrator) ApproveProposal(ctx context.Context, proposalID string) error {
	return f.store.UpdateProposalStatus(ctx, proposalID, "approved")
}

func (f *fakeOrchestrator) StartApprovedRun(ctx context.Context, proposalID string) (store.Session, error) {
	p, _ := f.store.GetProposal(ctx, proposalID)
	session, err := f.store.CreateSession(ctx, store.Session{
		TicketID: p.TicketID,
		Status:   "running",
	})
	if err != nil {
		return store.Session{}, err
	}
	_ = f.store.SetAgentActive(ctx, p.TicketID, true)
	return session, nil
}

func (f *fakeOrchestrator) FinishRun(ctx context.Context, input orchestrator.FinishRunInput) error {
	return nil
}

func (f *fakeOrchestrator) GetLogs(sessionID string) []string {
	return []string{"mock log line"}
}

func (f *fakeOrchestrator) SendInput(sessionID, input string) error {
	return nil
}

func (f *fakeOrchestrator) GetActiveSessions() []*orchestrator.AgentSession {
	sessions, err := f.store.ListActiveSessions(context.Background())
	if err != nil {
		return nil
	}
	out := make([]*orchestrator.AgentSession, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, &orchestrator.AgentSession{
			SessionID: s.ID,
			TicketID:  s.TicketID,
			Agent:     s.Agent,
			StartedAt: s.StartedAt.Unix(),
			Status:    s.Status,
		})
	}
	return out
}

func (f *fakeOrchestrator) GetPaneContent(sessionID string, lines int) (string, error) {
	return "", fmt.Errorf("not implemented in fake")
}

func (f *fakeOrchestrator) SwitchToPane(sessionID string) error {
	return fmt.Errorf("not implemented in fake")
}

func (f *fakeOrchestrator) CompletionChan() <-chan orchestrator.RunCompletion {
	return f.completionCh
}

func (f *fakeOrchestrator) StartAdHocRun(ctx context.Context, ticketID, agent, prompt string) (store.Session, error) {
	session, err := f.store.CreateSession(ctx, store.Session{
		TicketID: ticketID,
		Agent:    agent,
		Status:   "running",
	})
	if err != nil {
		return store.Session{}, err
	}
	return session, nil
}

func (f *fakeOrchestrator) completeRun(ticketID, sessionID, outcome string) {
	f.store.SetAgentActive(context.Background(), ticketID, false)
	if outcome == "completed" {
		f.store.MoveStatus(context.Background(), ticketID, "review")
	}
	f.completionCh <- orchestrator.RunCompletion{
		TicketID:  ticketID,
		SessionID: sessionID,
		Outcome:   outcome,
	}
}

func newTestApp(t *testing.T) (*App, *fakeOrchestrator) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.Open(dbPath, []string{"backlog", "in_progress", "review", "done"}, "AGT-")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	cfg := config.SetDefaults()
	reg := theme.NewRegistry("dark")
	reg.Register(&theme.Theme{
		Name: "agentboard", Source: "builtin",
		Primary: lipgloss.Color("69"), Text: lipgloss.Color("15"),
		TextMuted: lipgloss.Color("240"), Background: lipgloss.Color("#000"),
		BackgroundPanel: lipgloss.Color("236"), Border: lipgloss.Color("240"),
		Success: lipgloss.Color("42"), Accent: lipgloss.Color("213"),
	})

	fo := newFakeOrchestrator(s)
	app, err := NewApp(cfg, s, reg, AppDeps{
		Orchestrator: fo,
	})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return app, fo
}

func TestNewApp(t *testing.T) {
	app, _ := newTestApp(t)

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

func TestNewAppShowsTicketsCreatedBeforeToday(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.Open(dbPath, []string{"backlog", "in_progress", "review", "done"}, "AGT-")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	old := time.Now().AddDate(0, 0, -7)
	if _, err := s.CreateTicket(context.Background(), store.Ticket{
		Title: "old ticket", Status: "backlog", CreatedAt: old,
	}); err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	cfg := config.SetDefaults()
	reg := theme.NewRegistry("dark")
	reg.Register(&theme.Theme{
		Name: "agentboard", Source: "builtin",
		Primary: lipgloss.Color("69"), Text: lipgloss.Color("15"),
		TextMuted: lipgloss.Color("240"), Background: lipgloss.Color("#000"),
		BackgroundPanel: lipgloss.Color("236"), Border: lipgloss.Color("240"),
		Success: lipgloss.Color("42"), Accent: lipgloss.Color("213"),
	})

	fo := newFakeOrchestrator(s)
	app, err := NewApp(cfg, s, reg, AppDeps{Orchestrator: fo})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	found := false
	for _, col := range app.kanban.columns {
		for _, tk := range col {
			if tk.Title == "old ticket" {
				found = true
			}
		}
	}
	if !found {
		t.Error("ticket created before app start should be visible immediately, without a manual refresh")
	}
}

func TestMoveToInProgressCreatesProposalRequest(t *testing.T) {
	app, fo := newTestApp(t)
	ctx := context.Background()

	ticket, _ := app.store.CreateTicket(ctx, store.Ticket{Status: "backlog", Agent: "opencode", Title: "Test Ticket"})
	app.activeTicket = &ticket

	// Directly send generateProposalMsg to Update
	_, cmd := app.Update(generateProposalMsg{
		ticketID: ticket.ID,
	})

	if cmd == nil {
		t.Fatal("expected command for proposal creation")
	}

	execCmd(app, cmd)

	if fo.lastCreateProposalTicketID != ticket.ID {
		t.Fatalf("ticketID = %q, want %q", fo.lastCreateProposalTicketID, ticket.ID)
	}
}

func updateLoop(app *App, msg tea.Msg) {
	_, cmd := app.Update(msg)
	if cmd == nil {
		return
	}
	execCmd(app, cmd)
}

func execCmd(app *App, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	done := make(chan tea.Msg, 1)
	go func() {
		done <- cmd()
	}()

	select {
	case msg := <-done:
		if msg == nil {
			return
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batch {
				execCmd(app, c)
			}
		} else {
			updateLoop(app, msg)
		}
	case <-time.After(20 * time.Millisecond):
		// Skip timers
	}
}

func TestAppQuitShowsConfirmation(t *testing.T) {
	app, _ := newTestApp(t)

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Fatal("q should not quit directly, should open modal")
	}
	if !app.modal.Active() {
		t.Fatal("modal should be active after pressing q")
	}
}

func TestAppQuitConfirmYes(t *testing.T) {
	app, _ := newTestApp(t)

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !app.modal.Active() {
		t.Fatal("modal should be active")
	}

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("cmd is nil, expected tea.Quit after confirming")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("cmd produced %T, want tea.QuitMsg", msg)
	}
}

func TestAppQuitConfirmCancel(t *testing.T) {
	app, _ := newTestApp(t)

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !app.modal.Active() {
		t.Fatal("modal should be active")
	}

	app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if app.modal.Active() {
		t.Error("modal should be closed after escape")
	}
}

func TestAppForceQuit(t *testing.T) {
	app, _ := newTestApp(t)
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
	app, _ := newTestApp(t)

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
	app, _ := newTestApp(t)
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
	app, _ := newTestApp(t)

	app.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if app.view != viewBoard {
		t.Errorf("view = %v, want viewBoard", app.view)
	}
}

func TestAppEscapeReturnsToBoard(t *testing.T) {
	app, _ := newTestApp(t)
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
	app, _ := newTestApp(t)
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
	app.ticketView = app.ticketView.SetTicket(app.activeTicket)
	view = app.View()
	if !strings.Contains(view, "Routed") {
		t.Error("ticket view missing title")
	}
}

func TestAppWindowResizeDelegatesToKanban(t *testing.T) {
	app, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if app.kanban.width != 120 {
		t.Errorf("kanban width = %d, want 120", app.kanban.width)
	}
	if app.kanban.height != 40 {
		t.Errorf("kanban height = %d, want 40", app.kanban.height)
	}
}

func TestAppNavigationDelegatesToKanban(t *testing.T) {
	app, _ := newTestApp(t)

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if app.kanban.colIndex != 1 {
		t.Errorf("kanban colIndex = %d, want 1", app.kanban.colIndex)
	}

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if app.kanban.colIndex != 0 {
		t.Errorf("kanban colIndex = %d, want 0", app.kanban.colIndex)
	}
}

func TestAppShowDashboard(t *testing.T) {
	app, _ := newTestApp(t)

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
	app, _ := newTestApp(t)

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if app.view != viewDashboard {
		t.Fatalf("view = %v, want viewDashboard", app.view)
	}

	app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if app.view != viewBoard {
		t.Errorf("view = %v after esc, want viewBoard", app.view)
	}
}

func TestAppJumpSessionEntersDashboardAndSelectsSession(t *testing.T) {
	app, _ := newTestApp(t)
	ctx := context.Background()

	titles := []string{"task a", "task b", "task c"}
	agents := []string{"claude", "claude", "opencode"}
	for i := range titles {
		tk, err := app.store.CreateTicket(ctx, store.Ticket{Title: titles[i], Status: "in_progress"})
		if err != nil {
			t.Fatalf("create ticket: %v", err)
		}
		if _, err := app.store.CreateSession(ctx, store.Session{TicketID: tk.ID, Agent: agents[i], Status: "running"}); err != nil {
			t.Fatalf("create session: %v", err)
		}
	}

	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Alt: true, Runes: []rune{'1'}})
	app = m.(*App)
	if app.view != viewDashboard {
		t.Fatalf("alt+1 should switch to dashboard view, got %v", app.view)
	}
	first := app.dashboard.SelectedSession()
	if first == nil || first.Agent != "claude" {
		t.Fatalf("alt+1 should select the first session (claude), got %+v", first)
	}

	// alt+3 while already on the dashboard must jump too, not be swallowed.
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Alt: true, Runes: []rune{'3'}})
	app = m.(*App)
	third := app.dashboard.SelectedSession()
	if third == nil || third.Agent != "opencode" {
		t.Errorf("alt+3 while already on dashboard should select the 3rd session (opencode), got %+v", third)
	}
}

func TestAppDashboardViewRenders(t *testing.T) {
	app, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	view := app.View()
	plain := stripAnsi(view)
	if !strings.Contains(view, "Agent Dashboard") {
		t.Error("dashboard view missing title")
	}
	// Sidebar must show installed agents with count 0 when no sessions are active.
	hasZeroAgent := false
	for _, name := range []string{"claude-code", "opencode", "codex", "cursor", "freecode"} {
		if regexp.MustCompile(`(?m)^\s*▸?\s*` + regexp.QuoteMeta(name) + `\s+0\b`).MatchString(plain) {
			hasZeroAgent = true
			break
		}
	}
	if !hasZeroAgent {
		t.Errorf("dashboard view should show at least one installed agent with count 0, got:\n%s", plain)
	}
}

func TestAppWindowResizeDelegatesToDashboard(t *testing.T) {
	app, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if app.dashboard.width != 120 {
		t.Errorf("dashboard width = %d, want 120", app.dashboard.width)
	}
	if app.dashboard.height != 40 {
		t.Errorf("dashboard height = %d, want 40", app.dashboard.height)
	}
}

func TestProposalApprovalAndRunWorkflow(t *testing.T) {
	app, fo := newTestApp(t)
	ctx := context.Background()

	ticket, _ := app.store.CreateTicket(ctx, store.Ticket{Status: "backlog", Agent: "claude-code", Title: "Work Ticket"})
	app.activeTicket = &ticket

	// 1. Move to in_progress
	_, cmd := app.Update(statusChangedMsg{
		ticketID:  ticket.ID,
		newStatus: "in_progress",
	})
	execCmd(app, cmd)

	// 2. Create proposal explicitly via generateProposalMsg
	_, cmd = app.Update(generateProposalMsg{
		ticketID: ticket.ID,
	})
	execCmd(app, cmd)

	if fo.lastCreateProposalTicketID != ticket.ID {
		t.Fatalf("expected proposal for %s", ticket.ID)
	}

	// 3. Mock proposal loading (simulate what happens in App after creation)
	p, _ := app.store.GetActiveProposalForTicket(ctx, ticket.ID)
	app.Update(proposalLoadedMsg{TicketID: ticket.ID, proposal: &p})

	if app.ticketView.activeProposal == nil {
		t.Fatal("activeProposal should not be nil")
	}

	// 4. Approve proposal
	_, cmd = app.Update(proposalApprovedMsg{proposalID: p.ID})
	execCmd(app, cmd)

	// Verify status updated in store
	updatedP, _ := app.store.GetProposal(ctx, p.ID)
	if updatedP.Status != "approved" {
		t.Errorf("proposal status = %s, want approved", updatedP.Status)
	}

	// 5. Start run
	_, cmd = app.Update(runStartedMsg{proposalID: p.ID})
	execCmd(app, cmd)

	// Verify session created and agent active
	if !app.store.HasActiveSession(ctx, ticket.ID) {
		t.Error("expected active session")
	}
	updatedT, _ := app.store.GetTicket(ctx, ticket.ID)
	if !updatedT.AgentActive {
		t.Error("expected ticket AgentActive to be true")
	}
}
