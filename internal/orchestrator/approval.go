package orchestrator

import (
	"context"
	"fmt"

	"github.com/ayan-de/agent-board/internal/store"
)

func (s *Service) ApproveProposal(ctx context.Context, proposalID string) error {
	proposal, err := s.store.GetProposal(ctx, proposalID)
	if err != nil {
		return err
	}
	if proposal.Status != "pending" {
		return fmt.Errorf("orchestrator.approveProposal: proposal is not pending")
	}

	ticket, err := s.store.GetTicket(ctx, proposal.TicketID)
	if err != nil {
		return err
	}
	if ticket.UpdatedAt.After(proposal.CreatedAt) {
		return fmt.Errorf("orchestrator.approveProposal: proposal is stale (ticket updated after proposal)")
	}

	if err := s.store.UpdateProposalStatus(ctx, proposalID, "approved"); err != nil {
		return err
	}

	_, _ = s.store.CreateEvent(ctx, store.Event{
		TicketID: proposal.TicketID,
		Kind:     "proposal.approved",
		Payload:  proposalID,
	})

	return nil
}
