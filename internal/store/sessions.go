package store

import "time"

type Session struct {
	ID         string
	TicketID   string
	Agent      string
	StartedAt  time.Time
	EndedAt    *time.Time
	Status     string
	ContextKey string
}
