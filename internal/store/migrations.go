package store

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tickets (
		id          TEXT PRIMARY KEY,
		title       TEXT NOT NULL,
		description TEXT DEFAULT '',
		status      TEXT NOT NULL,
		priority    TEXT DEFAULT 'medium',
		agent       TEXT DEFAULT '',
		branch      TEXT DEFAULT '',
		tags        TEXT DEFAULT '[]',
		depends_on  TEXT DEFAULT '[]',
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id          TEXT PRIMARY KEY,
		ticket_id   TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
		agent       TEXT NOT NULL,
		started_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		ended_at    DATETIME,
		status      TEXT NOT NULL,
		context_key TEXT DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
	CREATE INDEX IF NOT EXISTS idx_sessions_ticket ON sessions(ticket_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return err
	}

	return nil
}
