package orchestrator_test

import (
	"context"
	"testing"

	"github.com/ayan-de/agent-board/internal/orchestrator"
	"github.com/ayan-de/agent-board/internal/store"
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

func TestStartApprovedRunRejectsExistingActiveSession(t *testing.T) {
	fs := &fakeStore{
		activeSession: true,
		proposal: store.Proposal{
			ID:       "PRO-01",
			TicketID: "AGE-01",
			Agent:    "opencode",
			Status:   "approved",
			Prompt:   "do work",
		},
	}
	svc := orchestrator.NewService(fs, nil, nil, fakeCtx{})

	_, err := svc.StartApprovedRun(context.Background(), "PRO-01")
	if err == nil {
		t.Fatal("expected duplicate active session error")
	}
}

func TestFinishRunPersistsContextCarry(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{
			ID:     "AGE-01",
			Status: "in_progress",
		},
	}
	fllm := &fakeLLMClient{summary: "short handoff summary"}
	svc := orchestrator.NewService(fs, fllm, nil, fakeCtx{})

	err := svc.FinishRun(context.Background(), orchestrator.FinishRunInput{
		TicketID:  "AGE-01",
		SessionID: "SES-01",
		Outcome:   "completed",
		Summary:   "raw worker summary",
	})
	if err != nil {
		t.Fatal(err)
	}
	if fs.lastContextCarry.Summary != "short handoff summary" {
		t.Fatalf("Summary = %q, want short handoff summary", fs.lastContextCarry.Summary)
	}
}
