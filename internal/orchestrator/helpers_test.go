package orchestrator_test

import (
	"context"
	"time"

	"github.com/ayan-de/agent-board/internal/llm"
	"github.com/ayan-de/agent-board/internal/orchestrator"
	"github.com/ayan-de/agent-board/internal/store"
)

type fakeStore struct {
	ticket        store.Ticket
	proposal      store.Proposal
	contextCarry  store.ContextCarry
	activeSession bool

	lastProposal     store.Proposal
	lastMoveStatus   string
	lastAgentActive  bool
	lastContextCarry store.ContextCarry
	lastSession      store.Session
	lastEvent        store.Event
}

func (f *fakeStore) GetTicket(_ context.Context, _ string) (store.Ticket, error) {
	return f.ticket, nil
}
func (f *fakeStore) CreateProposal(_ context.Context, p store.Proposal) (store.Proposal, error) {
	p.ID = "PRO-01"
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	f.lastProposal = p
	return p, nil
}
func (f *fakeStore) GetProposal(_ context.Context, _ string) (store.Proposal, error) {
	return f.proposal, nil
}
func (f *fakeStore) UpdateProposalStatus(_ context.Context, id, status string) error {
	f.proposal.Status = status
	return nil
}
func (f *fakeStore) GetContextCarry(_ context.Context, _ string) (store.ContextCarry, error) {
	return f.contextCarry, nil
}
func (f *fakeStore) UpsertContextCarry(_ context.Context, cc store.ContextCarry) error {
	f.lastContextCarry = cc
	return nil
}
func (f *fakeStore) CreateSession(_ context.Context, s store.Session) (store.Session, error) {
	if s.ID == "" {
		s.ID = "SES-01"
	}
	f.lastSession = s
	return s, nil
}
func (f *fakeStore) EndSession(_ context.Context, _, _ string) error { return nil }
func (f *fakeStore) HasActiveSession(_ context.Context, _ string) bool {
	return f.activeSession
}
func (f *fakeStore) SetAgentActive(_ context.Context, _ string, active bool) error {
	f.lastAgentActive = active
	return nil
}
func (f *fakeStore) MoveStatus(_ context.Context, _, status string) error {
	f.lastMoveStatus = status
	return nil
}
func (f *fakeStore) SetResumeCommand(_ context.Context, _, _ string) error { return nil }
func (f *fakeStore) CreateEvent(_ context.Context, e store.Event) (store.Event, error) {
	f.lastEvent = e
	return e, nil
}
func (f *fakeStore) ListTickets(_ context.Context, _ store.TicketFilters) ([]store.Ticket, error) {
	return nil, nil
}
func (f *fakeStore) CreateTicket(_ context.Context, t store.Ticket) (store.Ticket, error) {
	return t, nil
}
func (f *fakeStore) UpdateTicket(_ context.Context, t store.Ticket) (store.Ticket, error) {
	return t, nil
}
func (f *fakeStore) DeleteTicket(_ context.Context, _ string) error {
	return nil
}
func (f *fakeStore) GetActiveProposalForTicket(_ context.Context, _ string) (store.Proposal, error) {
	return store.Proposal{}, nil
}
func (f *fakeStore) GetSession(_ context.Context, _ string) (store.Session, error) {
	return store.Session{}, nil
}
func (f *fakeStore) ListSessions(_ context.Context, _ string) ([]store.Session, error) {
	return nil, nil
}
func (f *fakeStore) ListActiveSessions(_ context.Context) ([]store.Session, error) {
	return nil, nil
}

type fakeLLMClient struct {
	proposal     llm.ProposalDraft
	summary      string
	lastProposal llm.ProposalPrompt
}

func (f *fakeLLMClient) GenerateProposal(_ context.Context, in llm.ProposalPrompt) (llm.ProposalDraft, error) {
	f.lastProposal = in
	return f.proposal, nil
}
func (f *fakeLLMClient) SummarizeContext(_ context.Context, _ llm.SummaryInput) (string, error) {
	return f.summary, nil
}

type fakeRunner struct {
	outcome string
	summary string
}

func (f fakeRunner) Start(_ context.Context, _ orchestrator.RunRequest) (orchestrator.RunHandle, error) {
	return orchestrator.RunHandle{Outcome: f.outcome, Summary: f.summary}, nil
}

type fakeAsyncRunner struct {
	onComplete func(outcome, summary, resumeCommand string)
}

func (f *fakeAsyncRunner) Start(_ context.Context, req orchestrator.RunRequest) (orchestrator.RunHandle, error) {
	f.onComplete = req.OnComplete
	return orchestrator.RunHandle{Outcome: "running", Summary: "async started"}, nil
}

type fakeTmuxRunner struct {
	outcome string
	summary string
}

func (f *fakeTmuxRunner) Start(_ context.Context, req orchestrator.RunRequest) (orchestrator.RunHandle, error) {
	if req.OnComplete != nil {
		req.OnComplete(f.outcome, f.summary, "")
	}
	return orchestrator.RunHandle{Outcome: f.outcome, Summary: f.summary}, nil
}

func (f *fakeTmuxRunner) GetPaneID(_ string) (string, bool) { return "", false }

type fakeAsyncTmuxRunner struct {
	onComplete func(outcome, summary, resumeCommand string)
}

func (f *fakeAsyncTmuxRunner) Start(_ context.Context, req orchestrator.RunRequest) (orchestrator.RunHandle, error) {
	f.onComplete = req.OnComplete
	return orchestrator.RunHandle{Outcome: "running", Summary: "async started"}, nil
}

func (f *fakeAsyncTmuxRunner) GetPaneID(_ string) (string, bool) { return "", false }

type fakeCtx struct{}

func (f fakeCtx) LoadContext(ctx context.Context, ticketID string) (string, error) { return "", nil }
func (f fakeCtx) SaveContext(ctx context.Context, ticketID, outcome string) error  { return nil }
