package orchestrator

import (
	"context"
	"io"

	"github.com/ayan-de/agent-board/internal/llm"
	"github.com/ayan-de/agent-board/internal/store"
)

type CreateProposalInput struct {
	TicketID string
}

type ApplyRunOutcomeInput struct {
	TicketID string
	Outcome  string
}

type FinishRunInput struct {
	TicketID  string
	SessionID string
	Outcome   string
	Summary   string
}

type RunRequest struct {
	TicketID  string
	SessionID string
	Agent     string
	Prompt    string
	Reporter  func(string)
	InputChan chan io.Writer
}


type RunHandle struct {
	Outcome string
	Summary string
}

type LLMClient interface {
	GenerateProposal(ctx context.Context, input llm.ProposalPrompt) (llm.ProposalDraft, error)
	SummarizeContext(ctx context.Context, input llm.SummaryInput) (string, error)
}

type Runner interface {
	Start(ctx context.Context, req RunRequest) (RunHandle, error)
}

type Store interface {
	GetTicket(ctx context.Context, id string) (store.Ticket, error)
	CreateProposal(ctx context.Context, p store.Proposal) (store.Proposal, error)
	GetProposal(ctx context.Context, id string) (store.Proposal, error)
	UpdateProposalStatus(ctx context.Context, id, status string) error
	GetContextCarry(ctx context.Context, ticketID string) (store.ContextCarry, error)
	UpsertContextCarry(ctx context.Context, cc store.ContextCarry) error
	CreateSession(ctx context.Context, sess store.Session) (store.Session, error)
	EndSession(ctx context.Context, id, status string) error
	HasActiveSession(ctx context.Context, ticketID string) bool
	SetAgentActive(ctx context.Context, id string, active bool) error
	MoveStatus(ctx context.Context, id, status string) error
	CreateEvent(ctx context.Context, e store.Event) (store.Event, error)
}
type ContextCarryProvider interface {
	LoadContext(ctx context.Context, ticketID string) (string, error)
	SaveContext(ctx context.Context, ticketID, outcome string) error
}
