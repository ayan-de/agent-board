package store

import (
	"encoding/json"
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
