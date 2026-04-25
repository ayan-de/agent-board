package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Ticket struct {
	ID          string
	Title       string
	Description string
	Status      string
	Priority    string
	Agent       string
	Branch      string
	Tags        []string
	DependsOn   []string
	AgentActive bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TicketFilters struct {
	Status   string
	Agent    string
	Priority string
	Tag      string
	From     *time.Time
	To       *time.Time
	Search   string
}

type ticketRow struct {
	ID          string
	Title       string
	Description string
	Status      string
	Priority    string
	Agent       string
	Branch      string
	Tags        string
	DependsOn   string
	AgentActive bool
	CreatedAt   string
	UpdatedAt   string
}

func (r ticketRow) toTicket() (Ticket, error) {
	var tags []string
	if err := json.Unmarshal([]byte(r.Tags), &tags); err != nil {
		return Ticket{}, err
	}
	var dependsOn []string
	if err := json.Unmarshal([]byte(r.DependsOn), &dependsOn); err != nil {
		return Ticket{}, err
	}

	return Ticket{
		ID:          r.ID,
		Title:       r.Title,
		Description: r.Description,
		Status:      r.Status,
		Priority:    r.Priority,
		Agent:       r.Agent,
		Branch:      r.Branch,
		Tags:        tags,
		DependsOn:   dependsOn,
		AgentActive: r.AgentActive,
	}, nil
}

var validPriorities = map[string]bool{
	"low": true, "medium": true, "high": true, "critical": true,
}

func (s *Store) isValidStatus(status string) bool {
	for _, v := range s.validStatuses {
		if v == status {
			return true
		}
	}
	return false
}

func (s *Store) nextTicketID(ctx context.Context) (string, error) {
	prefix := s.ticketPrefix

	s.db.ExecContext(ctx, "INSERT OR IGNORE INTO id_counters (prefix, next_id) VALUES (?, 1)", prefix)

	var nextID int
	err := s.db.QueryRowContext(ctx, "SELECT next_id FROM id_counters WHERE prefix = ?", prefix).Scan(&nextID)
	if err != nil {
		return "", fmt.Errorf("store.nextTicketID: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "UPDATE id_counters SET next_id = next_id + 1 WHERE prefix = ?", prefix)
	if err != nil {
		return "", fmt.Errorf("store.nextTicketID: %w", err)
	}

	return fmt.Sprintf("%s%02d", prefix, nextID), nil
}

func (s *Store) CreateTicket(ctx context.Context, t Ticket) (Ticket, error) {
	if t.Title == "" {
		return Ticket{}, fmt.Errorf("store.createTicket: title is required")
	}
	if !s.isValidStatus(t.Status) {
		return Ticket{}, fmt.Errorf("store.createTicket: %q: %w", t.Status, ErrInvalidStatus)
	}
	if t.Priority != "" && !validPriorities[t.Priority] {
		return Ticket{}, fmt.Errorf("store.createTicket: %q: %w", t.Priority, ErrInvalidPriority)
	}
	if t.Priority == "" {
		t.Priority = "medium"
	}

	id, err := s.nextTicketID(ctx)
	if err != nil {
		return Ticket{}, err
	}
	t.ID = id

	tags, err := json.Marshal(t.Tags)
	if err != nil {
		return Ticket{}, fmt.Errorf("store.createTicket: encoding tags: %w", err)
	}
	deps, err := json.Marshal(t.DependsOn)
	if err != nil {
		return Ticket{}, fmt.Errorf("store.createTicket: encoding depends_on: %w", err)
	}

	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO tickets (id, title, description, status, priority, agent, branch, tags, depends_on, agent_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Title, t.Description, t.Status, t.Priority, t.Agent, t.Branch, string(tags), string(deps), t.AgentActive, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return Ticket{}, fmt.Errorf("store.createTicket: %w", err)
	}

	return t, nil
}

func (s *Store) GetTicket(ctx context.Context, id string) (Ticket, error) {
	var r ticketRow
	err := s.db.QueryRowContext(ctx,
		"SELECT id, title, description, status, priority, agent, branch, tags, depends_on, agent_active, created_at, updated_at FROM tickets WHERE id = ?",
		id,
	).Scan(&r.ID, &r.Title, &r.Description, &r.Status, &r.Priority, &r.Agent, &r.Branch, &r.Tags, &r.DependsOn, &r.AgentActive, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return Ticket{}, fmt.Errorf("store.getTicket %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return Ticket{}, fmt.Errorf("store.getTicket %s: %w", id, err)
	}

	ticket, err := r.toTicket()
	if err != nil {
		return Ticket{}, fmt.Errorf("store.getTicket %s: %w", id, err)
	}

	return ticket, nil
}

func (s *Store) ListTickets(ctx context.Context, filters TicketFilters) ([]Ticket, error) {
	query := "SELECT id, title, description, status, priority, agent, branch, tags, depends_on, agent_active, created_at, updated_at FROM tickets WHERE 1=1"
	var args []interface{}

	if filters.Status != "" {
		query += " AND status = ?"
		args = append(args, filters.Status)
	}
	if filters.Agent != "" {
		query += " AND agent = ?"
		args = append(args, filters.Agent)
	}
	if filters.Priority != "" {
		query += " AND priority = ?"
		args = append(args, filters.Priority)
	}
	if filters.Tag != "" {
		query += " AND tags LIKE ?"
		args = append(args, `%"`+filters.Tag+`"%`)
	}
	if filters.From != nil {
		query += " AND created_at >= ?"
		args = append(args, *filters.From)
	}
	if filters.To != nil {
		query += " AND created_at <= ?"
		args = append(args, *filters.To)
	}
	if filters.Search != "" {
		query += " AND (title LIKE ? OR description LIKE ?)"
		pattern := "%" + filters.Search + "%"
		args = append(args, pattern, pattern)
	}

	query += " ORDER BY created_at ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store.listTickets: %w", err)
	}
	defer rows.Close()

	var tickets []Ticket
	for rows.Next() {
		var r ticketRow
		if err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.Status, &r.Priority, &r.Agent, &r.Branch, &r.Tags, &r.DependsOn, &r.AgentActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store.listTickets: %w", err)
		}
		ticket, err := r.toTicket()
		if err != nil {
			return nil, fmt.Errorf("store.listTickets: %w", err)
		}
		tickets = append(tickets, ticket)
	}

	return tickets, nil
}

func (s *Store) UpdateTicket(ctx context.Context, t Ticket) (Ticket, error) {
	if !s.isValidStatus(t.Status) {
		return Ticket{}, fmt.Errorf("store.updateTicket: %q: %w", t.Status, ErrInvalidStatus)
	}
	if t.Priority != "" && !validPriorities[t.Priority] {
		return Ticket{}, fmt.Errorf("store.updateTicket: %q: %w", t.Priority, ErrInvalidPriority)
	}

	tags, err := json.Marshal(t.Tags)
	if err != nil {
		return Ticket{}, fmt.Errorf("store.updateTicket: encoding tags: %w", err)
	}
	deps, err := json.Marshal(t.DependsOn)
	if err != nil {
		return Ticket{}, fmt.Errorf("store.updateTicket: encoding depends_on: %w", err)
	}

	t.UpdatedAt = time.Now()

	result, err := s.db.ExecContext(ctx,
		`UPDATE tickets SET title=?, description=?, status=?, priority=?, agent=?, branch=?, tags=?, depends_on=?, updated_at=? WHERE id=?`,
		t.Title, t.Description, t.Status, t.Priority, t.Agent, t.Branch, string(tags), string(deps), t.UpdatedAt, t.ID,
	)
	if err != nil {
		return Ticket{}, fmt.Errorf("store.updateTicket: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return Ticket{}, fmt.Errorf("store.updateTicket %s: %w", t.ID, ErrNotFound)
	}

	return t, nil
}

func (s *Store) DeleteTicket(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM context_carry WHERE ticket_id = ?", id)
	if err != nil {
		return fmt.Errorf("store.deleteTicket: context_carry: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM orchestration_events WHERE ticket_id = ?", id)
	if err != nil {
		return fmt.Errorf("store.deleteTicket: events: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM proposals WHERE ticket_id = ?", id)
	if err != nil {
		return fmt.Errorf("store.deleteTicket: proposals: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM sessions WHERE ticket_id = ?", id)
	if err != nil {
		return fmt.Errorf("store.deleteTicket: sessions: %w", err)
	}

	result, err := s.db.ExecContext(ctx, "DELETE FROM tickets WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("store.deleteTicket: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("store.deleteTicket %s: %w", id, ErrNotFound)
	}

	return nil
}

func (s *Store) SetAgentActive(ctx context.Context, id string, active bool) error {
	result, err := s.db.ExecContext(ctx,
		"UPDATE tickets SET agent_active = ?, updated_at = ? WHERE id = ?",
		active, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("store.setAgentActive: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("store.setAgentActive %s: %w", id, ErrNotFound)
	}

	return nil
}

func (s *Store) MoveStatus(ctx context.Context, id string, status string) error {
	if !s.isValidStatus(status) {
		return fmt.Errorf("store.moveStatus: %q: %w", status, ErrInvalidStatus)
	}

	result, err := s.db.ExecContext(ctx,
		"UPDATE tickets SET status = ?, updated_at = ? WHERE id = ?",
		status, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("store.moveStatus: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("store.moveStatus %s: %w", id, ErrNotFound)
	}

	return nil
}
