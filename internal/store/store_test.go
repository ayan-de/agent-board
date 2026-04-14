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
