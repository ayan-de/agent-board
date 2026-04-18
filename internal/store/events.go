package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID        string
	TicketID  string
	SessionID string
	Kind      string
	Payload   string
	CreatedAt time.Time
}

func (s *Store) CreateEvent(ctx context.Context, e Event) (Event, error) {
	e.ID = uuid.New().String()
	e.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO orchestration_events (id, ticket_id, session_id, kind, payload, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		e.ID, e.TicketID, e.SessionID, e.Kind, e.Payload, e.CreatedAt,
	)
	if err != nil {
		return Event{}, fmt.Errorf("store.createEvent: %w", err)
	}

	return e, nil
}
