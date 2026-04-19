package orchestrator

import (
	"context"

	"github.com/ayan-de/agent-board/internal/llm"
	"github.com/ayan-de/agent-board/internal/store"
)

func (s Service) FinishRun(ctx context.Context, input FinishRunInput) error {
	s.mu.Lock()
	delete(s.activeSessions, input.SessionID)
	s.mu.Unlock()

	cc, err := s.llm.SummarizeContext(ctx, llm.SummaryInput{
		TicketID: input.TicketID,
		Outcome:  input.Outcome,
		Summary:  input.Summary,
	})
	if err != nil {
		cc = input.Summary
	}

	_ = s.store.UpsertContextCarry(ctx, store.ContextCarry{
		TicketID: input.TicketID,
		Summary:  cc,
	})

	_ = s.store.EndSession(ctx, input.SessionID, input.Outcome)

	_, _ = s.store.CreateEvent(ctx, store.Event{
		TicketID:  input.TicketID,
		SessionID: input.SessionID,
		Kind:      "session." + input.Outcome,
		Payload:   input.Summary,
	})

	return s.ApplyRunOutcome(ctx, ApplyRunOutcomeInput{
		TicketID: input.TicketID,
		Outcome:  input.Outcome,
	})
}
