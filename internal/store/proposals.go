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
	var maxID int
	err := s.db.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(CAST(SUBSTR(id, 5) AS INTEGER)), 0) FROM proposals",
	).Scan(&maxID)
	if err != nil {
		return "", fmt.Errorf("store.nextProposalID: %w", err)
	}
	return fmt.Sprintf("%s%02d", proposalPrefix, maxID+1), nil
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
