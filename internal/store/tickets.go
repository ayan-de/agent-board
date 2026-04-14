package store

import (
	"context"
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
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TicketFilters struct {
	Status   string
	Agent    string
	Priority string
	Tag      string
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
	}, nil
}

const ticketPrefix = "AGT-"

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
	var maxID int
	err := s.db.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(CAST(SUBSTR(id, 5) AS INTEGER)), 0) FROM tickets",
	).Scan(&maxID)
	if err != nil {
		return "", fmt.Errorf("store.nextTicketID: %w", err)
	}
	return fmt.Sprintf("%s%02d", ticketPrefix, maxID+1), nil
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
	t.CreatedAt = now
	t.UpdatedAt = now

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO tickets (id, title, description, status, priority, agent, branch, tags, depends_on, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Title, t.Description, t.Status, t.Priority, t.Agent, t.Branch, string(tags), string(deps), t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return Ticket{}, fmt.Errorf("store.createTicket: %w", err)
	}

	return t, nil
}
