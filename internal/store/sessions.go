package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Session struct {
	ID         string
	TicketID   string
	Agent      string
	StartedAt  time.Time
	EndedAt    *time.Time
	Status     string
	ContextKey string
}

const sessionPrefix = "SES-"

func (s *Store) nextSessionID(ctx context.Context) (string, error) {
	s.db.ExecContext(ctx, "INSERT OR IGNORE INTO id_counters (prefix, next_id) VALUES (?, 1)", sessionPrefix)

	var nextID int
	err := s.db.QueryRowContext(ctx, "SELECT next_id FROM id_counters WHERE prefix = ?", sessionPrefix).Scan(&nextID)
	if err != nil {
		return "", fmt.Errorf("store.nextSessionID: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "UPDATE id_counters SET next_id = next_id + 1 WHERE prefix = ?", sessionPrefix)
	if err != nil {
		return "", fmt.Errorf("store.nextSessionID: %w", err)
	}

	return fmt.Sprintf("%s%02d", sessionPrefix, nextID), nil
}

func (s *Store) CreateSession(ctx context.Context, sess Session) (Session, error) {
	id, err := s.nextSessionID(ctx)
	if err != nil {
		return Session{}, err
	}
	sess.ID = id

	now := time.Now()
	sess.StartedAt = now

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, ticket_id, agent, started_at, status, context_key) VALUES (?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.TicketID, sess.Agent, sess.StartedAt, sess.Status, sess.ContextKey,
	)
	if err != nil {
		return Session{}, fmt.Errorf("store.createSession: %w", err)
	}

	return sess, nil
}

func (s *Store) GetSession(ctx context.Context, id string) (Session, error) {
	var sess Session
	var endedAt sql.NullTime

	err := s.db.QueryRowContext(ctx,
		"SELECT id, ticket_id, agent, started_at, ended_at, status, context_key FROM sessions WHERE id = ?",
		id,
	).Scan(&sess.ID, &sess.TicketID, &sess.Agent, &sess.StartedAt, &endedAt, &sess.Status, &sess.ContextKey)
	if err == sql.ErrNoRows {
		return Session{}, fmt.Errorf("store.getSession %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return Session{}, fmt.Errorf("store.getSession %s: %w", id, err)
	}

	if endedAt.Valid {
		sess.EndedAt = &endedAt.Time
	}

	return sess, nil
}

func (s *Store) ListSessions(ctx context.Context, ticketID string) ([]Session, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, ticket_id, agent, started_at, ended_at, status, context_key FROM sessions WHERE ticket_id = ? ORDER BY started_at ASC",
		ticketID,
	)
	if err != nil {
		return nil, fmt.Errorf("store.listSessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		var endedAt sql.NullTime
		if err := rows.Scan(&sess.ID, &sess.TicketID, &sess.Agent, &sess.StartedAt, &endedAt, &sess.Status, &sess.ContextKey); err != nil {
			return nil, fmt.Errorf("store.listSessions: %w", err)
		}
		if endedAt.Valid {
			sess.EndedAt = &endedAt.Time
		}
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

func (s *Store) EndSession(ctx context.Context, id string, status string) error {
	now := time.Now()

	result, err := s.db.ExecContext(ctx,
		"UPDATE sessions SET ended_at = ?, status = ? WHERE id = ?",
		now, status, id,
	)
	if err != nil {
		return fmt.Errorf("store.endSession: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("store.endSession %s: %w", id, ErrNotFound)
	}

	return nil
}

func (s *Store) ListActiveSessions(ctx context.Context) ([]Session, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, ticket_id, agent, started_at, ended_at, status, context_key FROM sessions WHERE ended_at IS NULL ORDER BY started_at ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("store.listActiveSessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		var endedAt sql.NullTime
		if err := rows.Scan(&sess.ID, &sess.TicketID, &sess.Agent, &sess.StartedAt, &endedAt, &sess.Status, &sess.ContextKey); err != nil {
			return nil, fmt.Errorf("store.listActiveSessions: %w", err)
		}
		if endedAt.Valid {
			sess.EndedAt = &endedAt.Time
		}
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

func (s *Store) HasActiveSession(ctx context.Context, ticketID string) bool {
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sessions WHERE ticket_id = ? AND ended_at IS NULL",
		ticketID,
	).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}
