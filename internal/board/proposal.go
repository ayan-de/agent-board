package board

import (
	"context"
	"time"

	"github.com/ayan-de/agent-board/internal/orchestrator"
)

func ProposalCreate(b *BoardService, ticketID string) BoardViewState {
	b.state.Ticket.Loading = true

	proposal, err := b.orchestrator.CreateProposal(context.Background(), orchestrator.CreateProposalInput{
		TicketID: ticketID,
	})

	if err != nil {
		b.state.Notification = &NotificationState{
			Title:   "Proposal failed",
			Message: err.Error(),
			Variant: NotificationError,
		}
		b.state.Ticket.Loading = false
		return *b.state
	}

	b.state.Ticket.Proposal = &proposal
	b.state.Ticket.Loading = false
	b.state.Notification = &NotificationState{
		Title:   "Proposal created",
		Message: "AI proposed work for " + ticketID,
		Variant: NotificationInfo,
	}
	return *b.state
}

func ProposalApprove(b *BoardService, proposalID string) BoardViewState {
	if b.state.Ticket == nil || b.state.Ticket.Proposal == nil {
		return *b.state
	}

	err := b.orchestrator.ApproveProposal(context.Background(), proposalID)
	if err != nil {
		b.state.Notification = &NotificationState{
			Title:   "Error",
			Message: err.Error(),
			Variant: NotificationError,
		}
		return *b.state
	}

	p, _ := b.store.GetProposal(context.Background(), proposalID)
	b.state.Ticket.Proposal = &p
	return *b.state
}

func ProposalStartRun(b *BoardService, proposalID string) BoardViewState {
	if b.state.Ticket == nil || b.state.Ticket.Proposal == nil {
		return *b.state
	}

	_, err := b.orchestrator.StartApprovedRun(context.Background(), proposalID)
	if err != nil {
		b.state.Notification = &NotificationState{
			Title:   "Run failed",
			Message: err.Error(),
			Variant: NotificationError,
		}
		return *b.state
	}

	b.state.Notification = &NotificationState{
		Title:   "Run started",
		Message: "Agent is working...",
		Variant: NotificationInfo,
	}

	go func() {
		select {
		case completion := <-b.orchestrator.CompletionChan():
			b.handleRunCompletion(completion)
		case <-time.After(60 * time.Second):
			b.handleRunCompletion(orchestrator.RunCompletion{TicketID: ""})
		}
	}()

	return *b.state
}

