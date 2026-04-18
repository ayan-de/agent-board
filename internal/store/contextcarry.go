package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type ContextCarry struct {
	TicketID  string
	Summary   string
	UpdatedAt time.Time
}

func (s *Store) UpsertContextCarry(ctx context.Context, cc ContextCarry) error {
	cc.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO context_carry (ticket_id, summary, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(ticket_id) DO UPDATE SET summary = excluded.summary, updated_at = excluded.updated_at`,
		cc.TicketID, cc.Summary, cc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("store.upsertContextCarry: %w", err)
	}
	return nil
}

func (s *Store) GetContextCarry(ctx context.Context, ticketID string) (ContextCarry, error) {
	var cc ContextCarry
	err := s.db.QueryRowContext(ctx,
		"SELECT ticket_id, summary, updated_at FROM context_carry WHERE ticket_id = ?",
		ticketID,
	).Scan(&cc.TicketID, &cc.Summary, &cc.UpdatedAt)
	if err == sql.ErrNoRows {
		return ContextCarry{}, fmt.Errorf("store.getContextCarry %s: %w", ticketID, ErrNotFound)
	}
	if err != nil {
		return ContextCarry{}, fmt.Errorf("store.getContextCarry %s: %w", ticketID, err)
	}
	return cc, nil
}
