package llm

import (
	"context"
	"fmt"

	"github.com/ayan-de/agent-board/internal/prompt"
	"github.com/tmc/langchaingo/llms"
)

type LangChainClient struct {
	Coordinator llms.Model
	Summarizer  llms.Model
}

func (c LangChainClient) GenerateProposal(ctx context.Context, in ProposalPrompt) (ProposalDraft, error) {
	p := prompt.GenerateProposal(in.TicketID, in.Title, in.Description, in.AssignedAgent, in.ContextCarry)
	text, err := llms.GenerateFromSinglePrompt(ctx, c.Coordinator, p)
	if err != nil {
		return ProposalDraft{}, fmt.Errorf("llm.generateProposal: %w", err)
	}
	return ProposalDraft{Prompt: text}, nil
}

func (c LangChainClient) SummarizeContext(ctx context.Context, in SummaryInput) (string, error) {
	p := prompt.SummarizeContext(in.TicketID, in.Outcome, in.Summary)
	text, err := llms.GenerateFromSinglePrompt(ctx, c.Summarizer, p)
	if err != nil {
		return "", fmt.Errorf("llm.summarizeContext: %w", err)
	}
	return text, nil
}
