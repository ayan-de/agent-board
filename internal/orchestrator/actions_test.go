package orchestrator_test

import (
	"context"
	"testing"

	"github.com/ayan-de/agent-board/internal/orchestrator"
	"github.com/ayan-de/agent-board/internal/store"
)

func TestApplyRunOutcomeMovesTicketToReview(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{
			ID:          "AGE-01",
			Status:      "in_progress",
			Agent:       "opencode",
			AgentActive: true,
		},
	}
	svc := orchestrator.NewService(fs, nil, nil, fakeCtx{})

	err := svc.ApplyRunOutcome(context.Background(), orchestrator.ApplyRunOutcomeInput{
		TicketID: "AGE-01",
		Outcome:  "completed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if fs.lastMoveStatus != "review" {
		t.Fatalf("MoveStatus = %q, want review", fs.lastMoveStatus)
	}
	if fs.lastAgentActive != false {
		t.Fatalf("AgentActive = %v, want false", fs.lastAgentActive)
	}
}
