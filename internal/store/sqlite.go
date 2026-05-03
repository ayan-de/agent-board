package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var (
	ErrNotFound        = errors.New("store: not found")
	ErrInvalidStatus   = errors.New("store: invalid status")
	ErrInvalidPriority = errors.New("store: invalid priority")
)

type Store struct {
	db            *sql.DB
	validStatuses []string
	ticketPrefix  string
}

func Open(dbPath string, validStatuses []string, ticketPrefix string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("store.open: creating dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("store.open: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("store.open: enabling foreign keys: %w", err)
	}

	s := &Store{db: db, validStatuses: validStatuses, ticketPrefix: ticketPrefix}

	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) SetTicketPrefix(prefix string) {
	s.ticketPrefix = prefix
}
