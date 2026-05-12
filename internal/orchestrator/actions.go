package orchestrator

import (
	"context"
	"fmt"
)

func (s *Service) ApplyRunOutcome(ctx context.Context, input ApplyRunOutcomeInput) error {
	if input.ResumeCommand != "" {
		if err := s.store.SetResumeCommand(ctx, input.TicketID, input.ResumeCommand); err != nil {
			return fmt.Errorf("set resume command: %w", err)
		}
	}

	if err := s.store.SetAgentActive(ctx, input.TicketID, false); err != nil {
		return fmt.Errorf("set agent active: %w", err)
	}

	switch input.Outcome {
	case "completed":
		return s.store.MoveStatus(ctx, input.TicketID, "review")
	case "failed", "interrupted", "blocked":
		return nil
	default:
		return fmt.Errorf("orchestrator.applyRunOutcome: unknown outcome %q", input.Outcome)
	}
}
