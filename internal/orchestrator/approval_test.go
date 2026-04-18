package orchestrator_test

import (
	"context"
	"testing"
	"time"

	"github.com/ayan-de/agent-board/internal/orchestrator"
	"github.com/ayan-de/agent-board/internal/store"
)

func TestApproveProposalSucceeds(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{
			ID:        "AGT-01",
			UpdatedAt: time.Unix(10, 0),
		},
		proposal: store.Proposal{
			ID:        "PRO-01",
			TicketID:  "AGT-01",
			Status:    "pending",
			CreatedAt: time.Unix(20, 0),
		},
	}
	svc := orchestrator.NewService(fs, nil, nil)

	err := svc.ApproveProposal(context.Background(), "PRO-01")
	if err != nil {
		t.Fatal(err)
	}
	if fs.proposal.Status != "approved" {
		t.Fatalf("Status = %q, want approved", fs.proposal.Status)
	}
}

func TestApproveProposalRejectsStaleTicketState(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{
			ID:        "AGT-01",
			UpdatedAt: time.Unix(20, 0),
		},
		proposal: store.Proposal{
			ID:        "PRO-01",
			TicketID:  "AGT-01",
			Status:    "pending",
			CreatedAt: time.Unix(10, 0),
		},
	}
	svc := orchestrator.NewService(fs, nil, nil)

	err := svc.ApproveProposal(context.Background(), "PRO-01")
	if err == nil {
		t.Fatal("expected stale proposal error")
	}
}

func TestApproveProposalRejectsNonPending(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{
			ID:        "AGT-01",
			UpdatedAt: time.Unix(10, 0),
		},
		proposal: store.Proposal{
			ID:        "PRO-01",
			TicketID:  "AGT-01",
			Status:    "approved",
			CreatedAt: time.Unix(20, 0),
		},
	}
	svc := orchestrator.NewService(fs, nil, nil)

	err := svc.ApproveProposal(context.Background(), "PRO-01")
	if err == nil {
		t.Fatal("expected error for already approved proposal")
	}
}
