package orchestrator_test

import (
	"context"
	"testing"

	"github.com/ayan-de/agent-board/internal/llm"
	"github.com/ayan-de/agent-board/internal/orchestrator"
	"github.com/ayan-de/agent-board/internal/store"
)

func TestCreateProposalUsesTicketAndContextCarry(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{
			ID:          "AGT-01",
			Title:       "Add orchestrator",
			Description: "Build orchestration flow",
			Status:      "in_progress",
			Agent:       "opencode",
		},
		contextCarry: store.ContextCarry{
			TicketID: "AGT-01",
			Summary:  "prior run summary",
		},
	}
	fllm := &fakeLLMClient{
		proposal: llm.ProposalDraft{
			Prompt: "work with prior run summary",
		},
	}
	svc := orchestrator.NewService(fs, fllm, nil, fakeCtx{})

	proposal, err := svc.CreateProposal(context.Background(), orchestrator.CreateProposalInput{
		TicketID: "AGT-01",
	})
	if err != nil {
		t.Fatal(err)
	}

	if proposal.TicketID != "AGT-01" {
		t.Fatalf("TicketID = %q, want AGT-01", proposal.TicketID)
	}
	if proposal.Agent != "opencode" {
		t.Fatalf("Agent = %q, want opencode", proposal.Agent)
	}
	if proposal.Status != "pending" {
		t.Fatalf("Status = %q, want pending", proposal.Status)
	}
	if proposal.Prompt != "work with prior run summary" {
		t.Fatalf("Prompt = %q, want work with prior run summary", proposal.Prompt)
	}
	if fllm.lastProposal.ContextCarry != "prior run summary" {
		t.Fatalf("ContextCarry passed to LLM = %q, want prior run summary", fllm.lastProposal.ContextCarry)
	}
	if fllm.lastProposal.Title != "Add orchestrator" {
		t.Fatalf("Title passed to LLM = %q, want Add orchestrator", fllm.lastProposal.Title)
	}
	if fllm.lastProposal.AssignedAgent != "opencode" {
		t.Fatalf("Agent passed to LLM = %q, want opencode", fllm.lastProposal.AssignedAgent)
	}
}

func TestCreateProposalRecordsEvent(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{
			ID:    "AGT-01",
			Title: "Test",
			Agent: "opencode",
		},
	}
	fllm := &fakeLLMClient{
		proposal: llm.ProposalDraft{Prompt: "do work"},
	}
	svc := orchestrator.NewService(fs, fllm, nil, fakeCtx{})

	_, err := svc.CreateProposal(context.Background(), orchestrator.CreateProposalInput{
		TicketID: "AGT-01",
	})
	if err != nil {
		t.Fatal(err)
	}

	if fs.lastEvent.Kind != "proposal.created" {
		t.Fatalf("Event Kind = %q, want proposal.created", fs.lastEvent.Kind)
	}
	if fs.lastEvent.TicketID != "AGT-01" {
		t.Fatalf("Event TicketID = %q, want AGT-01", fs.lastEvent.TicketID)
	}
}

func TestCreateProposalRejectsUnassignedTicket(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{
			ID:    "AGT-01",
			Title: "No agent",
			Agent: "",
		},
	}
	svc := orchestrator.NewService(fs, nil, nil, fakeCtx{})

	_, err := svc.CreateProposal(context.Background(), orchestrator.CreateProposalInput{
		TicketID: "AGT-01",
	})
	if err == nil {
		t.Fatal("expected error for unassigned ticket")
	}
}
