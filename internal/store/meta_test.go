package store

import (
	"context"
	"testing"
	"time"
)

func TestProjectInitDateDefaultsToTodayForFreshStore(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()

	got, err := s.ProjectInitDate(context.Background())
	if err != nil {
		t.Fatalf("ProjectInitDate: %v", err)
	}
	want := time.Now().Format("2006-01-02")
	if got.Format("2006-01-02") != want {
		t.Errorf("ProjectInitDate = %v, want today (%s)", got, want)
	}
}

func TestProjectInitDateAnchorsToEarliestTicketOnFirstCall(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()
	ctx := context.Background()

	old := time.Date(2026, 5, 11, 1, 18, 37, 0, time.Local)
	if _, err := s.CreateTicket(ctx, Ticket{Title: "old", Status: "backlog", CreatedAt: old}); err != nil {
		t.Fatalf("create old ticket: %v", err)
	}
	newer := time.Date(2026, 8, 14, 23, 0, 0, 0, time.Local)
	if _, err := s.CreateTicket(ctx, Ticket{Title: "new", Status: "backlog", CreatedAt: newer}); err != nil {
		t.Fatalf("create new ticket: %v", err)
	}

	got, err := s.ProjectInitDate(ctx)
	if err != nil {
		t.Fatalf("ProjectInitDate: %v", err)
	}
	if got.Format("2006-01-02") != "2026-05-11" {
		t.Errorf("ProjectInitDate = %v, want 2026-05-11 (earliest ticket)", got)
	}
}

func TestProjectInitDatePersistsAcrossCalls(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()
	ctx := context.Background()

	first, err := s.ProjectInitDate(ctx)
	if err != nil {
		t.Fatalf("ProjectInitDate (first): %v", err)
	}

	// A ticket created after the anchor was already persisted must not
	// move it, even though it would otherwise be the "earliest" ticket.
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local)
	if _, err := s.CreateTicket(ctx, Ticket{Title: "backdated", Status: "backlog", CreatedAt: old}); err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	second, err := s.ProjectInitDate(ctx)
	if err != nil {
		t.Fatalf("ProjectInitDate (second): %v", err)
	}
	if !first.Equal(second) {
		t.Errorf("ProjectInitDate changed after persistence: first=%v second=%v", first, second)
	}
}

func TestProjectInitDateSurvivesLegacyTimestampFormat(t *testing.T) {
	s := openTestDB(t)
	defer s.Close()
	ctx := context.Background()

	tk, err := s.CreateTicket(ctx, Ticket{Title: "legacy", Status: "backlog"})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	// Simulate a row written in the old non-RFC3339 textual time.Time
	// format (with monotonic reading), as found in real databases.
	legacy := "2026-05-11 01:18:37.128056877 +0530 IST m=+30.029040611"
	if _, err := s.db.ExecContext(ctx, "UPDATE tickets SET created_at = ? WHERE id = ?", legacy, tk.ID); err != nil {
		t.Fatalf("simulate legacy row: %v", err)
	}

	got, err := s.ProjectInitDate(ctx)
	if err != nil {
		t.Fatalf("ProjectInitDate: %v", err)
	}
	if got.Format("2006-01-02") != "2026-05-11" {
		t.Errorf("ProjectInitDate = %v, want 2026-05-11 despite legacy timestamp format", got)
	}
}
