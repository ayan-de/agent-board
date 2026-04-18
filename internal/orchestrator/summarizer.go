package orchestrator

import (
	"context"
	"fmt"

	"github.com/ayan-de/agent-board/internal/llm"
	"github.com/ayan-de/agent-board/internal/store"
)

func (s Service) FinishRun(ctx context.Context, input FinishRunInput) error {
	cc, err := s.llm.SummarizeContext(ctx, llm.SummaryInput{
		TicketID: input.TicketID,
		Outcome:  input.Outcome,
		Summary:  input.Summary,
	})
	if err != nil {
		return fmt.Errorf("orchestrator.finishRun: %w", err)
	}

	if err := s.store.UpsertContextCarry(ctx, store.ContextCarry{
		TicketID: input.TicketID,
		Summary:  cc,
	}); err != nil {
		return err
	}

	if err := s.store.EndSession(ctx, input.SessionID, input.Outcome); err != nil {
		return err
	}

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
