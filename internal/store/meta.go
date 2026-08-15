package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const metaKeyProjectInitDate = "project_init_date"

// ProjectInitDate returns the project's persisted start date, computing and
// storing it on first call so it can no longer drift if config files or
// directory timestamps are ever reset. New stores anchor to today; stores
// that already contain tickets anchor to the earliest ticket's creation
// date (by day, tolerant of legacy non-RFC3339 timestamp text) so upgrades
// don't lose history.
func (s *Store) ProjectInitDate(ctx context.Context) (time.Time, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM meta WHERE key = ?", metaKeyProjectInitDate).Scan(&raw)
	if err == nil {
		return time.Parse("2006-01-02", raw)
	}
	if err != sql.ErrNoRows {
		return time.Time{}, fmt.Errorf("store.ProjectInitDate: %w", err)
	}

	dateStr := time.Now().Format("2006-01-02")
	var earliest sql.NullString
	if err := s.db.QueryRowContext(ctx, "SELECT MIN(SUBSTR(created_at, 1, 10)) FROM tickets").Scan(&earliest); err == nil && earliest.Valid && earliest.String != "" {
		dateStr = earliest.String
	}

	if _, err := s.db.ExecContext(ctx,
		"INSERT INTO meta (key, value) VALUES (?, ?)", metaKeyProjectInitDate, dateStr,
	); err != nil {
		return time.Time{}, fmt.Errorf("store.ProjectInitDate: %w", err)
	}
	return time.Parse("2006-01-02", dateStr)
}
