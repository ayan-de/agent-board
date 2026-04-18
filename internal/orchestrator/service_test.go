package orchestrator_test

import (
	"context"
	"testing"

	"github.com/ayan-de/agent-board/internal/orchestrator"
)

func TestServiceCreateProposalRequiresAssignedAgent(t *testing.T) {
	svc := orchestrator.Service{}
	_, err := svc.CreateProposal(context.Background(), orchestrator.CreateProposalInput{
		TicketID: "AGT-01",
	})
	if err == nil {
		t.Fatal("expected error when no store configured")
	}
}
