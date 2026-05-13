package core

import (
	"context"

	"github.com/ayan-de/agent-board/internal/llm"
	"github.com/ayan-de/agent-board/internal/store"
)

type LLMClient interface {
	GenerateProposal(ctx context.Context, input llm.ProposalPrompt) (llm.ProposalDraft, error)
	SummarizeContext(ctx context.Context, input llm.SummaryInput) (string, error)
}

type Runner interface {
	Start(ctx context.Context, req RunRequest) (RunHandle, error)
}

type RunRequest struct {
	TicketID   string
	SessionID  string
	Agent      string
	Prompt     string
	Reporter   func(string)
	Target     string
	OnComplete func(outcome, summary, resumeCommand string)
}

type RunHandle struct {
	Outcome string
	Summary string
}

type AgentRunner interface {
	Start(ctx context.Context, req RunRequest) (RunHandle, error)
	GetPaneID(sessionID string) (string, bool)
}

type Store interface {
	GetTicket(ctx context.Context, id string) (store.Ticket, error)
	ListTickets(ctx context.Context, filters store.TicketFilters) ([]store.Ticket, error)
	CreateTicket(ctx context.Context, t store.Ticket) (store.Ticket, error)
	UpdateTicket(ctx context.Context, t store.Ticket) (store.Ticket, error)
	DeleteTicket(ctx context.Context, id string) error
	MoveStatus(ctx context.Context, id, status string) error
	SetAgentActive(ctx context.Context, id string, active bool) error
	SetResumeCommand(ctx context.Context, id, cmd string) error
	CreateProposal(ctx context.Context, p store.Proposal) (store.Proposal, error)
	GetProposal(ctx context.Context, id string) (store.Proposal, error)
	UpdateProposalStatus(ctx context.Context, id, status string) error
	GetActiveProposalForTicket(ctx context.Context, ticketID string) (store.Proposal, error)
	GetContextCarry(ctx context.Context, ticketID string) (store.ContextCarry, error)
	UpsertContextCarry(ctx context.Context, cc store.ContextCarry) error
	CreateSession(ctx context.Context, sess store.Session) (store.Session, error)
	GetSession(ctx context.Context, id string) (store.Session, error)
	ListSessions(ctx context.Context, ticketID string) ([]store.Session, error)
	ListActiveSessions(ctx context.Context) ([]store.Session, error)
	EndSession(ctx context.Context, id, status string) error
	HasActiveSession(ctx context.Context, ticketID string) bool
	CreateEvent(ctx context.Context, e store.Event) (store.Event, error)
}

type ContextCarryProvider interface {
	LoadContext(ctx context.Context, ticketID string) (string, error)
	SaveContext(ctx context.Context, ticketID, outcome string) error
}
