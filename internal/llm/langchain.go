package llm

import (
	"context"

	"github.com/tmc/langchaingo/llms"
)

type LangChainClient struct {
	Coordinator llms.Model
	Summarizer  llms.Model
}

func (c LangChainClient) GenerateProposal(_ context.Context, _ ProposalPrompt) (ProposalDraft, error) {
	return ProposalDraft{}, nil
}

func (c LangChainClient) SummarizeContext(_ context.Context, _ SummaryInput) (string, error) {
	return "", nil
}
