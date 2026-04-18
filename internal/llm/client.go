package llm

import "context"

type ProposalPrompt struct {
	TicketID      string
	Title         string
	Description   string
	AssignedAgent string
	ContextCarry  string
}

type ProposalDraft struct {
	Prompt string
}

type SummaryInput struct {
	TicketID string
	Outcome  string
	Summary  string
}

type Client interface {
	GenerateProposal(ctx context.Context, in ProposalPrompt) (ProposalDraft, error)
	SummarizeContext(ctx context.Context, in SummaryInput) (string, error)
}
