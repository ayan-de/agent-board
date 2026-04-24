package llm_test

import (
	"context"
	"testing"

	"github.com/ayan-de/agent-board/internal/llm"
	"github.com/tmc/langchaingo/llms"
)

type stubModel struct {
	called bool
}

func (m *stubModel) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	m.called = true
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{{Content: "generated worker prompt"}},
	}, nil
}

func (m *stubModel) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	m.called = true
	return "generated response", nil
}

func TestGenerateProposalBuildsPromptFromTicketContext(t *testing.T) {
	coordinator := &stubModel{}
	summarizer := &stubModel{}

	client := &llm.LangChainClient{
		Coordinator: coordinator,
		Summarizer:  summarizer,
	}

	got, err := client.GenerateProposal(context.Background(), llm.ProposalPrompt{
		TicketID:      "AGT-01",
		Title:         "Add orchestrator",
		Description:   "Build service layer",
		AssignedAgent: "opencode",
		ContextCarry:  "prior run summary",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !coordinator.called {
		t.Fatal("expected coordinator to be called")
	}
}

func TestSummarizeContextReturnsSummary(t *testing.T) {
	coordinator := &stubModel{}
	summarizer := &stubModel{}

	client := &llm.LangChainClient{
		Coordinator: coordinator,
		Summarizer:  summarizer,
	}

	got, err := client.SummarizeContext(context.Background(), llm.SummaryInput{
		TicketID: "AGT-01",
		Outcome:  "completed",
		Summary:  "raw worker output",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected non-empty summary")
	}
	if !summarizer.called {
		t.Fatal("expected summarizer to be called")
	}
}

func TestGenerateProposalStripsThinkBlocks(t *testing.T) {
	client := &llm.LangChainClient{
		Coordinator: &stubModel{},
	}

	client.Coordinator = llmModelFunc(func(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
		return "<think>internal reasoning</think>\nWorker prompt body", nil
	})

	got, err := client.GenerateProposal(context.Background(), llm.ProposalPrompt{
		TicketID:      "AGT-01",
		Title:         "Add orchestrator",
		Description:   "Build service layer",
		AssignedAgent: "opencode",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Prompt != "Worker prompt body" {
		t.Fatalf("Prompt = %q, want sanitized worker prompt", got.Prompt)
	}
}

type llmModelFunc func(ctx context.Context, prompt string, options ...llms.CallOption) (string, error)

func (f llmModelFunc) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return f(ctx, prompt, options...)
}

func (f llmModelFunc) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	text, err := f(ctx, "", options...)
	if err != nil {
		return nil, err
	}
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{{Content: text}},
	}, nil
}
