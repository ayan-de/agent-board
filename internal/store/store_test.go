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
	s, err := Open(dbPath, []string{"backlog", "in_progress", "review", "done"}, "AGT-")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

func TestOpenCreatesDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	statuses := []string{"backlog", "in_progress", "review", "done"}
	s, err := Open(dbPath, statuses, "AGT-")
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
	s, err := Open(dbPath, statuses, "AGT-")
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

	s1, err := Open(dbPath, statuses, "AGT-")
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	s1.Close()

	s2, err := Open(dbPath, statuses, "AGT-")
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer s2.Close()
}

func TestOpenCreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "nested", "deep", "test.db")
	statuses := []string{"backlog", "in_progress", "review", "done"}

	s, err := Open(dbPath, statuses, "AGT-")
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

func TestCreateTicketWithCustomPrefix(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := Open(dbPath, []string{"backlog", "in_progress", "review", "done"}, "FOO-")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	t1, err := s.CreateTicket(context.Background(), Ticket{Title: "First", Status: "backlog"})
	if err != nil {
		t.Fatalf("CreateTicket 1: %v", err)
	}
	t2, err := s.CreateTicket(context.Background(), Ticket{Title: "Second", Status: "backlog"})
	if err != nil {
		t.Fatalf("CreateTicket 2: %v", err)
	}

	if t1.ID != "FOO-01" {
		t.Errorf("first ticket ID = %q, want %q", t1.ID, "FOO-01")
	}
	if t2.ID != "FOO-02" {
		t.Errorf("second ticket ID = %q, want %q", t2.ID, "FOO-02")
	}
}

func TestCreateTicketPrefixSwitchPreservesOld(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s1, err := Open(dbPath, []string{"backlog", "in_progress", "review", "done"}, "AAA-")
	if err != nil {
		t.Fatalf("Open 1: %v", err)
	}
	t1, _ := s1.CreateTicket(context.Background(), Ticket{Title: "First", Status: "backlog"})
	t2, _ := s1.CreateTicket(context.Background(), Ticket{Title: "Second", Status: "backlog"})
	s1.Close()

	if t1.ID != "AAA-01" || t2.ID != "AAA-02" {
		t.Fatalf("prefix AAA-: got %q, %q", t1.ID, t2.ID)
	}

	s2, err := Open(dbPath, []string{"backlog", "in_progress", "review", "done"}, "BBB-")
	if err != nil {
		t.Fatalf("Open 2: %v", err)
	}
	defer s2.Close()

	t3, err := s2.CreateTicket(context.Background(), Ticket{Title: "Third", Status: "backlog"})
	if err != nil {
		t.Fatalf("CreateTicket 3: %v", err)
	}

	if t3.ID != "BBB-01" {
		t.Errorf("after prefix switch, ticket ID = %q, want %q", t3.ID, "BBB-01")
	}

	old, err := s2.GetTicket(context.Background(), "AAA-01")
	if err != nil {
		t.Fatalf("old ticket AAA-01 should still exist: %v", err)
	}
	if old.Title != "First" {
		t.Errorf("old ticket title = %q, want %q", old.Title, "First")
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

func TestCreateSession(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	ticket, err := s.CreateTicket(context.Background(), Ticket{Title: "Session test", Status: "in_progress"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	session, err := s.CreateSession(context.Background(), Session{
		TicketID: ticket.ID,
		Agent:    "claude-code",
		Status:   "running",
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if session.ID == "" {
		t.Fatal("session ID should be auto-generated")
	}
	if session.TicketID != ticket.ID {
		t.Errorf("TicketID = %q, want %q", session.TicketID, ticket.ID)
	}
	if session.Agent != "claude-code" {
		t.Errorf("Agent = %q, want %q", session.Agent, "claude-code")
	}
	if session.StartedAt.IsZero() {
		t.Fatal("StartedAt should be set")
	}
	if session.EndedAt != nil {
		t.Fatal("EndedAt should be nil for running session")
	}
}

func TestGetSession(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	ticket, _ := s.CreateTicket(context.Background(), Ticket{Title: "T", Status: "backlog"})
	created, _ := s.CreateSession(context.Background(), Session{
		TicketID: ticket.ID, Agent: "opencode", Status: "running",
	})

	got, err := s.GetSession(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.Agent != "opencode" {
		t.Errorf("Agent = %q, want %q", got.Agent, "opencode")
	}
}

func TestGetSessionNotFound(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	_, err := s.GetSession(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestListSessionsByTicket(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	t1, _ := s.CreateTicket(context.Background(), Ticket{Title: "T1", Status: "backlog"})
	t2, _ := s.CreateTicket(context.Background(), Ticket{Title: "T2", Status: "backlog"})

	s.CreateSession(context.Background(), Session{TicketID: t1.ID, Agent: "claude-code", Status: "completed"})
	s.CreateSession(context.Background(), Session{TicketID: t1.ID, Agent: "opencode", Status: "running"})
	s.CreateSession(context.Background(), Session{TicketID: t2.ID, Agent: "cursor", Status: "running"})

	sessions, err := s.ListSessions(context.Background(), t1.ID)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("got %d sessions, want 2", len(sessions))
	}
}

func TestListSessionsEmpty(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	ticket, _ := s.CreateTicket(context.Background(), Ticket{Title: "T", Status: "backlog"})

	sessions, err := s.ListSessions(context.Background(), ticket.ID)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("got %d sessions, want 0", len(sessions))
	}
}

func TestEndSession(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	ticket, _ := s.CreateTicket(context.Background(), Ticket{Title: "T", Status: "backlog"})
	created, _ := s.CreateSession(context.Background(), Session{
		TicketID: ticket.ID, Agent: "claude-code", Status: "running",
	})

	err := s.EndSession(context.Background(), created.ID, "completed")
	if err != nil {
		t.Fatalf("EndSession: %v", err)
	}

	got, err := s.GetSession(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("Status = %q, want %q", got.Status, "completed")
	}
	if got.EndedAt == nil {
		t.Fatal("EndedAt should be set after EndSession")
	}
}

func TestEndSessionNotFound(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	err := s.EndSession(context.Background(), "nonexistent", "completed")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestOpenAddsAgentActiveColumn(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('tickets') WHERE name='agent_active'").Scan(&count)
	if err != nil {
		t.Fatalf("pragma_table_info query failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("agent_active column should exist in tickets table, got count=%d", count)
	}
}

func TestCreateTicketAgentActiveDefault(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	ticket, err := s.CreateTicket(context.Background(), Ticket{
		Title: "Active test", Status: "backlog",
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if ticket.AgentActive {
		t.Error("AgentActive should default to false")
	}
}

func TestSetAgentActive(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	ticket, _ := s.CreateTicket(context.Background(), Ticket{
		Title: "Active test", Status: "backlog",
	})

	err := s.SetAgentActive(context.Background(), ticket.ID, true)
	if err != nil {
		t.Fatalf("SetAgentActive: %v", err)
	}

	got, _ := s.GetTicket(context.Background(), ticket.ID)
	if !got.AgentActive {
		t.Error("AgentActive should be true after SetAgentActive(true)")
	}

	err = s.SetAgentActive(context.Background(), ticket.ID, false)
	if err != nil {
		t.Fatalf("SetAgentActive false: %v", err)
	}

	got, _ = s.GetTicket(context.Background(), ticket.ID)
	if got.AgentActive {
		t.Error("AgentActive should be false after SetAgentActive(false)")
	}
}

func TestSetAgentActiveNotFound(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	err := s.SetAgentActive(context.Background(), "AGT-99", true)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestDeleteTicketCascadesSessions(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	ticket, err := s.CreateTicket(context.Background(), Ticket{Title: "With session", Status: "in_progress"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	session, err := s.CreateSession(context.Background(), Session{
		TicketID: ticket.ID,
		Agent:    "claude-code",
		Status:   "running",
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	err = s.DeleteTicket(context.Background(), ticket.ID)
	if err != nil {
		t.Fatalf("DeleteTicket: %v", err)
	}

	_, err = s.GetSession(context.Background(), session.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("session should be deleted with ticket, error = %v, want ErrNotFound", err)
	}
}

func TestCreateAndApproveProposal(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	proposal, err := s.CreateProposal(context.Background(), Proposal{
		TicketID: "AGT-01",
		Agent:    "opencode",
		Status:   "pending",
		Prompt:   "do the work",
	})
	if err != nil {
		t.Fatal(err)
	}
	if proposal.ID == "" {
		t.Fatal("expected proposal ID")
	}
	if proposal.Status != "pending" {
		t.Fatalf("Status = %q, want pending", proposal.Status)
	}

	err = s.UpdateProposalStatus(context.Background(), proposal.ID, "approved")
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.GetProposal(context.Background(), proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "approved" {
		t.Fatalf("Status = %q, want approved", got.Status)
	}
}

func TestGetProposalNotFound(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	_, err := s.GetProposal(context.Background(), "PRO-99")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestUpsertAndGetContextCarry(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	err := s.UpsertContextCarry(context.Background(), ContextCarry{
		TicketID: "AGT-01",
		Summary:  "previous run summary",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.GetContextCarry(context.Background(), "AGT-01")
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary != "previous run summary" {
		t.Fatalf("Summary = %q, want previous run summary", got.Summary)
	}
}

func TestUpsertContextCarryOverwrites(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	s.UpsertContextCarry(context.Background(), ContextCarry{
		TicketID: "AGT-01",
		Summary:  "first",
	})
	s.UpsertContextCarry(context.Background(), ContextCarry{
		TicketID: "AGT-01",
		Summary:  "second",
	})

	got, err := s.GetContextCarry(context.Background(), "AGT-01")
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary != "second" {
		t.Fatalf("Summary = %q, want second", got.Summary)
	}
}

func TestGetContextCarryNotFound(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	_, err := s.GetContextCarry(context.Background(), "AGT-99")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestCreateAndRetrieveEvent(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	event, err := s.CreateEvent(context.Background(), Event{
		TicketID:  "AGT-01",
		SessionID: "SES-01",
		Kind:      "proposal.created",
		Payload:   "test payload",
	})
	if err != nil {
		t.Fatal(err)
	}
	if event.ID == "" {
		t.Fatal("expected event ID")
	}
	if event.TicketID != "AGT-01" {
		t.Fatalf("TicketID = %q, want AGT-01", event.TicketID)
	}
	if event.Kind != "proposal.created" {
		t.Fatalf("Kind = %q, want proposal.created", event.Kind)
	}
	if event.CreatedAt.IsZero() {
		t.Fatal("CreatedAt should be set")
	}
}

func TestHasActiveSession(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	ticket, _ := s.CreateTicket(context.Background(), Ticket{Title: "T", Status: "in_progress"})

	if s.HasActiveSession(context.Background(), ticket.ID) {
		t.Fatal("should have no active session initially")
	}

	s.CreateSession(context.Background(), Session{
		TicketID: ticket.ID,
		Agent:    "opencode",
		Status:   "running",
	})

	if !s.HasActiveSession(context.Background(), ticket.ID) {
		t.Fatal("should have active session after creation")
	}
}

func TestHasActiveSessionFalseAfterEnd(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	ticket, _ := s.CreateTicket(context.Background(), Ticket{Title: "T", Status: "in_progress"})
	session, _ := s.CreateSession(context.Background(), Session{
		TicketID: ticket.ID,
		Agent:    "opencode",
		Status:   "running",
	})

	s.EndSession(context.Background(), session.ID, "completed")

	if s.HasActiveSession(context.Background(), ticket.ID) {
		t.Fatal("should have no active session after end")
	}
}

func TestListActiveSessions(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	sessions, err := s.ListActiveSessions(context.Background())
	if err != nil {
		t.Fatalf("ListActiveSessions on empty: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected 0 active sessions, got %d", len(sessions))
	}

	t1, _ := s.CreateTicket(context.Background(), Ticket{Title: "T1", Status: "in_progress"})
	t2, _ := s.CreateTicket(context.Background(), Ticket{Title: "T2", Status: "in_progress"})

	s1, _ := s.CreateSession(context.Background(), Session{
		TicketID: t1.ID,
		Agent:    "opencode",
		Status:   "running",
	})
	s2, _ := s.CreateSession(context.Background(), Session{
		TicketID: t2.ID,
		Agent:    "claude",
		Status:   "running",
	})

	sessions, err = s.ListActiveSessions(context.Background())
	if err != nil {
		t.Fatalf("ListActiveSessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 active sessions, got %d", len(sessions))
	}

	got := map[string]string{}
	for _, sess := range sessions {
		got[sess.ID] = sess.Agent
	}
	if got[s1.ID] != "opencode" {
		t.Errorf("session %s agent = %q, want opencode", s1.ID, got[s1.ID])
	}
	if got[s2.ID] != "claude" {
		t.Errorf("session %s agent = %q, want claude", s2.ID, got[s2.ID])
	}

	s.EndSession(context.Background(), s1.ID, "completed")

	sessions, _ = s.ListActiveSessions(context.Background())
	if len(sessions) != 1 {
		t.Fatalf("expected 1 active session after ending one, got %d", len(sessions))
	}
	if sessions[0].ID != s2.ID {
		t.Errorf("remaining session = %s, want %s", sessions[0].ID, s2.ID)
	}
}

func TestDeleteTicketCascadesProposals(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	ticket, err := s.CreateTicket(context.Background(), Ticket{Title: "With proposal", Status: "in_progress"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	proposal, err := s.CreateProposal(context.Background(), Proposal{
		TicketID: ticket.ID,
		Agent:    "opencode",
		Status:   "pending",
		Prompt:   "do the work",
	})
	if err != nil {
		t.Fatalf("CreateProposal: %v", err)
	}

	err = s.DeleteTicket(context.Background(), ticket.ID)
	if err != nil {
		t.Fatalf("DeleteTicket: %v", err)
	}

	_, err = s.GetProposal(context.Background(), proposal.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("proposal should be deleted with ticket, error = %v, want ErrNotFound", err)
	}
}

func TestDeleteTicketCascadesEvents(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	ticket, err := s.CreateTicket(context.Background(), Ticket{Title: "With event", Status: "in_progress"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	_, err = s.CreateEvent(context.Background(), Event{
		TicketID: ticket.ID,
		Kind:     "test",
		Payload:  "data",
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}

	err = s.DeleteTicket(context.Background(), ticket.ID)
	if err != nil {
		t.Fatalf("DeleteTicket: %v", err)
	}

	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM orchestration_events WHERE ticket_id = ?", ticket.ID).Scan(&count)
	if count != 0 {
		t.Errorf("events should be deleted with ticket, found %d", count)
	}
}

func TestTicketIDDoesNotReuseAfterDelete(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	t1, _ := s.CreateTicket(context.Background(), Ticket{Title: "First", Status: "backlog"})
	s.DeleteTicket(context.Background(), t1.ID)

	t2, _ := s.CreateTicket(context.Background(), Ticket{Title: "Second", Status: "backlog"})

	if t2.ID == t1.ID {
		t.Errorf("new ticket ID %q reused deleted ID %q", t2.ID, t1.ID)
	}
	if t2.ID != "AGT-02" {
		t.Errorf("new ticket ID = %q, want AGT-02", t2.ID)
	}
}
