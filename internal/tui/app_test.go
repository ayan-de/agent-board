package tui

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/orchestrator"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

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
	return []*orchestrator.AgentSession{}
}

func (f *fakeOrchestrator) GetPaneContent(sessionID string, lines int) (string, error) {
	return "", nil
}

func (f *fakeOrchestrator) SwitchToPane(sessionID string) error {
	return nil
}

func (f *fakeOrchestrator) CompletionChan() <-chan orchestrator.RunCompletion {
	return f.completionCh
}

func (f *fakeOrchestrator) StartAdHocRun(ctx context.Context, agent, prompt string) (store.Session, error) {
	session, err := f.store.CreateSession(ctx, store.Session{
		TicketID: "",
		Agent:    agent,
		Status:   "running",
	})
	if err != nil {
		return store.Session{}, err
	}
	return session, nil
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
	if app.board == nil {
		t.Error("board service is nil")
	}
}