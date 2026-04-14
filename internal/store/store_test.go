package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func openTestDB(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := Open(dbPath, []string{"backlog", "in_progress", "review", "done"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

func TestOpenCreatesDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	statuses := []string{"backlog", "in_progress", "review", "done"}
	s, err := Open(dbPath, statuses)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("Open should create the database file")
	}
}

func TestOpenRunsMigrations(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	statuses := []string{"backlog", "in_progress", "review", "done"}
	s, err := Open(dbPath, statuses)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	var name string
	err = s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='tickets'").Scan(&name)
	if err != nil {
		t.Fatalf("tickets table not found: %v", err)
	}
	if name != "tickets" {
		t.Errorf("table name = %q, want %q", name, "tickets")
	}

	err = s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='sessions'").Scan(&name)
	if err != nil {
		t.Fatalf("sessions table not found: %v", err)
	}
	if name != "sessions" {
		t.Errorf("table name = %q, want %q", name, "sessions")
	}
}

func TestOpenIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	statuses := []string{"backlog", "in_progress", "review", "done"}

	s1, err := Open(dbPath, statuses)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	s1.Close()

	s2, err := Open(dbPath, statuses)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer s2.Close()
}

func TestClose(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	statuses := []string{"backlog", "in_progress", "review", "done"}

	s, err := Open(dbPath, statuses)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestOpenCreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "nested", "deep", "test.db")
	statuses := []string{"backlog", "in_progress", "review", "done"}

	s, err := Open(dbPath, statuses)
	if err != nil {
		t.Fatalf("Open with nested path: %v", err)
	}
	defer s.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("Open should create parent directories")
	}
}

func TestCreateTicket(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	ticket, err := s.CreateTicket(context.Background(), Ticket{
		Title:       "Implement auth",
		Description: "Add JWT authentication",
		Status:      "backlog",
		Priority:    "high",
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	if ticket.ID == "" {
		t.Fatal("ticket ID should be auto-generated")
	}
	if ticket.Title != "Implement auth" {
		t.Errorf("Title = %q, want %q", ticket.Title, "Implement auth")
	}
	if ticket.Status != "backlog" {
		t.Errorf("Status = %q, want %q", ticket.Status, "backlog")
	}
	if ticket.Priority != "high" {
		t.Errorf("Priority = %q, want %q", ticket.Priority, "high")
	}
	if ticket.CreatedAt.IsZero() {
		t.Fatal("CreatedAt should be set")
	}
}

func TestCreateTicketAutoIncrementsID(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	t1, err := s.CreateTicket(context.Background(), Ticket{Title: "First", Status: "backlog"})
	if err != nil {
		t.Fatalf("CreateTicket 1: %v", err)
	}
	t2, err := s.CreateTicket(context.Background(), Ticket{Title: "Second", Status: "backlog"})
	if err != nil {
		t.Fatalf("CreateTicket 2: %v", err)
	}

	if t1.ID != "AGT-01" {
		t.Errorf("first ticket ID = %q, want %q", t1.ID, "AGT-01")
	}
	if t2.ID != "AGT-02" {
		t.Errorf("second ticket ID = %q, want %q", t2.ID, "AGT-02")
	}
}

func TestCreateTicketInvalidStatus(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	_, err := s.CreateTicket(context.Background(), Ticket{
		Title:  "Bad status",
		Status: "nonexistent",
	})
	if err == nil {
		t.Fatal("should reject invalid status")
	}
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("error = %v, want ErrInvalidStatus", err)
	}
}

func TestCreateTicketInvalidPriority(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	_, err := s.CreateTicket(context.Background(), Ticket{
		Title:    "Bad priority",
		Status:   "backlog",
		Priority: "urgent",
	})
	if err == nil {
		t.Fatal("should reject invalid priority")
	}
	if !errors.Is(err, ErrInvalidPriority) {
		t.Errorf("error = %v, want ErrInvalidPriority", err)
	}
}

func TestCreateTicketEmptyTitle(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	_, err := s.CreateTicket(context.Background(), Ticket{
		Status: "backlog",
	})
	if err == nil {
		t.Fatal("should reject empty title")
	}
}

func TestCreateTicketWithTagsAndDeps(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	ticket, err := s.CreateTicket(context.Background(), Ticket{
		Title:     "Tagged ticket",
		Status:    "backlog",
		Tags:      []string{"auth", "backend"},
		DependsOn: []string{"AGT-01"},
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	if len(ticket.Tags) != 2 {
		t.Fatalf("Tags len = %d, want 2", len(ticket.Tags))
	}
	if ticket.Tags[0] != "auth" || ticket.Tags[1] != "backend" {
		t.Errorf("Tags = %v, want [auth backend]", ticket.Tags)
	}
	if len(ticket.DependsOn) != 1 || ticket.DependsOn[0] != "AGT-01" {
		t.Errorf("DependsOn = %v, want [AGT-01]", ticket.DependsOn)
	}
}

func TestGetTicket(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	created, err := s.CreateTicket(context.Background(), Ticket{
		Title:    "Get me",
		Status:   "backlog",
		Priority: "high",
		Tags:     []string{"test"},
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	got, err := s.GetTicket(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetTicket: %v", err)
	}
	if got.Title != "Get me" {
		t.Errorf("Title = %q, want %q", got.Title, "Get me")
	}
	if got.Priority != "high" {
		t.Errorf("Priority = %q, want %q", got.Priority, "high")
	}
	if len(got.Tags) != 1 || got.Tags[0] != "test" {
		t.Errorf("Tags = %v, want [test]", got.Tags)
	}
}

func TestGetTicketNotFound(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	_, err := s.GetTicket(context.Background(), "AGT-99")
	if err == nil {
		t.Fatal("should return error for missing ticket")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestListTicketsAll(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	s.CreateTicket(context.Background(), Ticket{Title: "A", Status: "backlog"})
	s.CreateTicket(context.Background(), Ticket{Title: "B", Status: "in_progress"})
	s.CreateTicket(context.Background(), Ticket{Title: "C", Status: "done"})

	tickets, err := s.ListTickets(context.Background(), TicketFilters{})
	if err != nil {
		t.Fatalf("ListTickets: %v", err)
	}
	if len(tickets) != 3 {
		t.Errorf("got %d tickets, want 3", len(tickets))
	}
}

func TestListTicketsByStatus(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	s.CreateTicket(context.Background(), Ticket{Title: "A", Status: "backlog"})
	s.CreateTicket(context.Background(), Ticket{Title: "B", Status: "in_progress"})
	s.CreateTicket(context.Background(), Ticket{Title: "C", Status: "backlog"})

	tickets, err := s.ListTickets(context.Background(), TicketFilters{Status: "backlog"})
	if err != nil {
		t.Fatalf("ListTickets: %v", err)
	}
	if len(tickets) != 2 {
		t.Errorf("got %d tickets, want 2", len(tickets))
	}
}

func TestListTicketsByAgent(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	s.CreateTicket(context.Background(), Ticket{Title: "A", Status: "backlog", Agent: "claude-code"})
	s.CreateTicket(context.Background(), Ticket{Title: "B", Status: "backlog", Agent: "opencode"})

	tickets, err := s.ListTickets(context.Background(), TicketFilters{Agent: "claude-code"})
	if err != nil {
		t.Fatalf("ListTickets: %v", err)
	}
	if len(tickets) != 1 {
		t.Errorf("got %d tickets, want 1", len(tickets))
	}
	if tickets[0].Agent != "claude-code" {
		t.Errorf("Agent = %q, want %q", tickets[0].Agent, "claude-code")
	}
}

func TestListTicketsByPriority(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	s.CreateTicket(context.Background(), Ticket{Title: "A", Status: "backlog", Priority: "high"})
	s.CreateTicket(context.Background(), Ticket{Title: "B", Status: "backlog", Priority: "low"})

	tickets, err := s.ListTickets(context.Background(), TicketFilters{Priority: "high"})
	if err != nil {
		t.Fatalf("ListTickets: %v", err)
	}
	if len(tickets) != 1 {
		t.Errorf("got %d tickets, want 1", len(tickets))
	}
}

func TestListTicketsByTag(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	s.CreateTicket(context.Background(), Ticket{Title: "A", Status: "backlog", Tags: []string{"auth", "backend"}})
	s.CreateTicket(context.Background(), Ticket{Title: "B", Status: "backlog", Tags: []string{"frontend"}})

	tickets, err := s.ListTickets(context.Background(), TicketFilters{Tag: "auth"})
	if err != nil {
		t.Fatalf("ListTickets: %v", err)
	}
	if len(tickets) != 1 {
		t.Errorf("got %d tickets, want 1", len(tickets))
	}
}

func TestUpdateTicket(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	created, err := s.CreateTicket(context.Background(), Ticket{
		Title:    "Original",
		Status:   "backlog",
		Priority: "medium",
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	created.Title = "Updated"
	created.Description = "New description"
	created.Tags = []string{"updated"}

	updated, err := s.UpdateTicket(context.Background(), created)
	if err != nil {
		t.Fatalf("UpdateTicket: %v", err)
	}
	if updated.Title != "Updated" {
		t.Errorf("Title = %q, want %q", updated.Title, "Updated")
	}
	if updated.Description != "New description" {
		t.Errorf("Description = %q, want %q", updated.Description, "New description")
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "updated" {
		t.Errorf("Tags = %v, want [updated]", updated.Tags)
	}

	got, err := s.GetTicket(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetTicket: %v", err)
	}
	if got.Title != "Updated" {
		t.Errorf("persisted Title = %q, want %q", got.Title, "Updated")
	}
}

func TestUpdateTicketNotFound(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	_, err := s.UpdateTicket(context.Background(), Ticket{ID: "AGT-99", Title: "Ghost", Status: "backlog"})
	if err == nil {
		t.Fatal("should return error for missing ticket")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestDeleteTicket(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	created, err := s.CreateTicket(context.Background(), Ticket{Title: "Delete me", Status: "backlog"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	err = s.DeleteTicket(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("DeleteTicket: %v", err)
	}

	_, err = s.GetTicket(context.Background(), created.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete, GetTicket error = %v, want ErrNotFound", err)
	}
}

func TestDeleteTicketNotFound(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	err := s.DeleteTicket(context.Background(), "AGT-99")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("DeleteTicket error = %v, want ErrNotFound", err)
	}
}

func TestMoveStatus(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	created, err := s.CreateTicket(context.Background(), Ticket{Title: "Move me", Status: "backlog"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	err = s.MoveStatus(context.Background(), created.ID, "in_progress")
	if err != nil {
		t.Fatalf("MoveStatus: %v", err)
	}

	got, err := s.GetTicket(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetTicket: %v", err)
	}
	if got.Status != "in_progress" {
		t.Errorf("Status = %q, want %q", got.Status, "in_progress")
	}
}

func TestMoveStatusInvalid(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	created, err := s.CreateTicket(context.Background(), Ticket{Title: "Move me", Status: "backlog"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	err = s.MoveStatus(context.Background(), created.ID, "nonexistent")
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("MoveStatus error = %v, want ErrInvalidStatus", err)
	}
}

func TestMoveStatusNotFound(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	err := s.MoveStatus(context.Background(), "AGT-99", "backlog")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("MoveStatus error = %v, want ErrNotFound", err)
	}
}
