package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Proposal struct {
	ID        string
	TicketID  string
	Agent     string
	Status    string
	Prompt    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

const proposalPrefix = "PRO-"

func (s *Store) nextProposalID(ctx context.Context) (string, error) {
	s.db.ExecContext(ctx, "INSERT OR IGNORE INTO id_counters (prefix, next_id) VALUES (?, 1)", proposalPrefix)

	var nextID int
	err := s.db.QueryRowContext(ctx, "SELECT next_id FROM id_counters WHERE prefix = ?", proposalPrefix).Scan(&nextID)
	if err != nil {
		return "", fmt.Errorf("store.nextProposalID: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "UPDATE id_counters SET next_id = next_id + 1 WHERE prefix = ?", proposalPrefix)
	if err != nil {
		return "", fmt.Errorf("store.nextProposalID: %w", err)
	}

	return fmt.Sprintf("%s%02d", proposalPrefix, nextID), nil
}

func (s *Store) CreateProposal(ctx context.Context, p Proposal) (Proposal, error) {
	id, err := s.nextProposalID(ctx)
	if err != nil {
		return Proposal{}, err
	}
	p.ID = id

	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO proposals (id, ticket_id, agent, status, prompt, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.TicketID, p.Agent, p.Status, p.Prompt, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return Proposal{}, fmt.Errorf("store.createProposal: %w", err)
	}

	return p, nil
}

func (s *Store) GetProposal(ctx context.Context, id string) (Proposal, error) {
	var p Proposal
	err := s.db.QueryRowContext(ctx,
		"SELECT id, ticket_id, agent, status, prompt, created_at, updated_at FROM proposals WHERE id = ?",
		id,
	).Scan(&p.ID, &p.TicketID, &p.Agent, &p.Status, &p.Prompt, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return Proposal{}, fmt.Errorf("store.getProposal %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return Proposal{}, fmt.Errorf("store.getProposal %s: %w", id, err)
	}
	return p, nil
}

func (s *Store) UpdateProposalStatus(ctx context.Context, id, status string) error {
	result, err := s.db.ExecContext(ctx,
		"UPDATE proposals SET status = ?, updated_at = ? WHERE id = ?",
		status, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("store.updateProposalStatus: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("store.updateProposalStatus %s: %w", id, ErrNotFound)
	}
	return nil
}

func (s *Store) GetActiveProposalForTicket(ctx context.Context, ticketID string) (Proposal, error) {
	var p Proposal
	err := s.db.QueryRowContext(ctx,
		"SELECT id, ticket_id, agent, status, prompt, created_at, updated_at FROM proposals WHERE ticket_id = ? AND status IN ('pending', 'approved') ORDER BY created_at DESC LIMIT 1",
		ticketID,
	).Scan(&p.ID, &p.TicketID, &p.Agent, &p.Status, &p.Prompt, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return Proposal{}, fmt.Errorf("store.getActiveProposalForTicket %s: %w", ticketID, ErrNotFound)
	}
	if err != nil {
		return Proposal{}, fmt.Errorf("store.getActiveProposalForTicket %s: %w", ticketID, err)
	}
	return p, nil
}
