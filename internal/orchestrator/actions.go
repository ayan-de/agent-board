package orchestrator

import (
	"context"
	"fmt"
)

func (s Service) ApplyRunOutcome(ctx context.Context, input ApplyRunOutcomeInput) error {
	switch input.Outcome {
	case "completed":
		if err := s.store.SetAgentActive(ctx, input.TicketID, false); err != nil {
			return err
		}
		return s.store.MoveStatus(ctx, input.TicketID, "review")
	case "failed", "interrupted", "blocked":
		return s.store.SetAgentActive(ctx, input.TicketID, false)
	default:
		return fmt.Errorf("orchestrator.applyRunOutcome: unknown outcome %q", input.Outcome)
	}
}
