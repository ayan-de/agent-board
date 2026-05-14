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
		ticket_id   TEXT,
		agent       TEXT NOT NULL,
		started_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		ended_at    DATETIME,
		status      TEXT NOT NULL,
		context_key TEXT DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
	CREATE INDEX IF NOT EXISTS idx_sessions_ticket ON sessions(ticket_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);

	CREATE TABLE IF NOT EXISTS proposals (
		id TEXT PRIMARY KEY,
		ticket_id TEXT NOT NULL,
		agent TEXT NOT NULL,
		status TEXT NOT NULL,
		prompt TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS orchestration_events (
		id TEXT PRIMARY KEY,
		ticket_id TEXT NOT NULL,
		session_id TEXT,
		kind TEXT NOT NULL,
		payload TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS context_carry (
		ticket_id TEXT PRIMARY KEY,
		summary TEXT NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_proposals_ticket ON proposals(ticket_id);
	CREATE INDEX IF NOT EXISTS idx_proposals_status ON proposals(status);
	CREATE INDEX IF NOT EXISTS idx_events_ticket ON orchestration_events(ticket_id);

	CREATE TABLE IF NOT EXISTS id_counters (
		prefix TEXT PRIMARY KEY,
		next_id INTEGER NOT NULL DEFAULT 1
	);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return err
	}

	var hasCol bool
	err = s.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('tickets') WHERE name='agent_active'").Scan(&hasCol)
	if err == nil && !hasCol {
		_, err = s.db.Exec("ALTER TABLE tickets ADD COLUMN agent_active INTEGER DEFAULT 0")
		if err != nil {
			return err
		}
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('tickets') WHERE name='resume_command'").Scan(&hasCol)
	if err == nil && !hasCol {
		_, err = s.db.Exec("ALTER TABLE tickets ADD COLUMN resume_command TEXT")
		if err != nil {
			return err
		}
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name='ticket_id' AND notnull=1").Scan(&hasCol)
	if err == nil && hasCol {
		var hasRows bool
		err = s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM sessions WHERE ticket_id IS NOT NULL AND ticket_id != '')").Scan(&hasRows)
		if err == nil && !hasRows {
			_, err = s.db.Exec("ALTER TABLE sessions RENAME TO sessions_old")
			if err != nil {
				return err
			}
			_, err = s.db.Exec(`
				CREATE TABLE sessions (
					id          TEXT PRIMARY KEY,
					ticket_id   TEXT,
					agent       TEXT NOT NULL,
					started_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
					ended_at    DATETIME,
					status      TEXT NOT NULL,
					context_key TEXT DEFAULT ''
				)
			`)
			if err != nil {
				return err
			}
			_, err = s.db.Exec("INSERT INTO sessions (id, ticket_id, agent, started_at, ended_at, status, context_key) SELECT id, ticket_id, agent, started_at, ended_at, status, context_key FROM sessions_old WHERE ticket_id IS NOT NULL AND ticket_id != ''")
			if err != nil {
				return err
			}
			_, err = s.db.Exec("DROP TABLE sessions_old")
			if err != nil {
				return err
			}
		}
	}

	return nil
}
