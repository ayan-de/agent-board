# Store Package Design

## Overview

The store package (`internal/store/`) manages SQLite persistence for AgentBoard. Each project gets its own isolated database at `~/.agentboard/projects/<name>/board.db`. The store handles ticket and session CRUD with status validation against the project's config.

## Per-Project Isolation

Each project has its own SQLite database. No shared DB, no cross-project contamination.

```
~/.agentboard/projects/
  agent-board/
    config.toml      ← BoardConfig.Statuses defines valid statuses
    board.db         ← THIS project's tickets + sessions
  my-web-app/
    config.toml
    board.db
```

When the store opens, it reads `cfg.DB.Path` (set by config package) and opens/creates that SQLite file. Statuses come from `cfg.Board.Statuses` — the store validates ticket status against this list.

## Schema

```sql
CREATE TABLE tickets (
    id          TEXT PRIMARY KEY,       -- e.g. "AGT-03", auto-generated
    title       TEXT NOT NULL,
    description TEXT DEFAULT '',
    status      TEXT NOT NULL,          -- validated against config statuses
    priority    TEXT DEFAULT 'medium',  -- low, medium, high, critical
    agent       TEXT DEFAULT '',        -- claude-code, opencode, cursor, or empty
    branch      TEXT DEFAULT '',
    tags        TEXT DEFAULT '[]',      -- JSON array of strings
    depends_on  TEXT DEFAULT '[]',      -- JSON array of ticket IDs
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,
    ticket_id   TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    agent       TEXT NOT NULL,
    started_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    ended_at    DATETIME,               -- NULL while running
    status      TEXT NOT NULL,           -- running, completed, failed, cancelled
    context_key TEXT DEFAULT ''          -- ContextCarry reference
);

CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_sessions_ticket ON sessions(ticket_id);
CREATE INDEX idx_sessions_status ON sessions(status);
```

JSON arrays for tags and depends_on instead of join tables. SQLite has no native array type, but `json_each()` allows querying when needed. Keeps the schema flat and simple for a local dev tool.

## Ticket ID Generation

Auto-incrementing prefix format: `AGT-01`, `AGT-02`, etc.

The store queries `MAX(CAST(SUBSTR(id, 5) AS INTEGER))` from tickets and increments. Prefix `AGT-` is a constant. Always zero-padded to 2 digits (or more as count grows).

## Package Structure

```
internal/store/
  store.go         ← Store struct, Open(), Close(), DB connection lifecycle
  tickets.go       ← Ticket CRUD: Create, Get, List, Update, Delete, MoveStatus
  sessions.go      ← Session CRUD: Create, Get, List, Update, EndSession
  migrations.go    ← Schema creation, version tracking
```

## Store Interface

```go
type TicketStore interface {
    CreateTicket(ctx context.Context, t Ticket) (Ticket, error)
    GetTicket(ctx context.Context, id string) (Ticket, error)
    ListTickets(ctx context.Context, filters TicketFilters) ([]Ticket, error)
    UpdateTicket(ctx context.Context, t Ticket) (Ticket, error)
    DeleteTicket(ctx context.Context, id string) error
    MoveStatus(ctx context.Context, id string, status string) error
}

type SessionStore interface {
    CreateSession(ctx context.Context, s Session) (Session, error)
    GetSession(ctx context.Context, id string) (Session, error)
    ListSessions(ctx context.Context, ticketID string) ([]Session, error)
    EndSession(ctx context.Context, id string, status string) error
}
```

Interfaces live in `store/`. The `Store` struct implements both. Consumers depend on interfaces, not concrete types.

## Go Types

```go
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

type Session struct {
    ID         string
    TicketID   string
    Agent      string
    StartedAt  time.Time
    EndedAt    *time.Time
    Status     string
    ContextKey string
}

type TicketFilters struct {
    Status   string
    Agent    string
    Priority string
    Tag      string
}
```

## Store Lifecycle

```go
func Open(dbPath string, validStatuses []string) (*Store, error)
func (s *Store) Close() error
```

`Open` creates/opens the SQLite file, runs migrations, and stores `validStatuses` for validation. `Close` closes the database connection.

## Validation

- **Status**: `CreateTicket` and `MoveStatus` validate against `validStatuses` passed at Open time. Invalid → error.
- **Priority**: only `low`, `medium`, `high`, `critical`. Invalid → error.
- **Session status**: only `running`, `completed`, `failed`, `cancelled`.
- **Required fields**: `title` must be non-empty on create.

## Error Handling

All errors wrapped with `fmt.Errorf("store.<operation>: %w", err)`.

Specific error types:
- `ErrNotFound` — ticket or session ID doesn't exist
- `ErrInvalidStatus` — status not in valid list
- `ErrInvalidPriority` — priority not in valid list

These are sentinel errors defined in the package:
```go
var ErrNotFound = errors.New("store: not found")
var ErrInvalidStatus = errors.New("store: invalid status")
var ErrInvalidPriority = errors.New("store: invalid priority")
```

## Dependencies

- `modernc.org/sqlite` — pure Go SQLite driver (no CGO, works on Termux)
- `database/sql` — stdlib SQL interface
- `encoding/json` — tags/depends_on JSON encoding
- `context` — request scoping

## Testing Strategy

- `t.TempDir()` for each test — fresh DB per test
- Test every CRUD operation for tickets
- Test every CRUD operation for sessions
- Test status validation (valid + invalid)
- Test priority validation (valid + invalid)
- Test ticket ID auto-generation sequence
- Test cascading delete (delete ticket → sessions deleted)
- Test JSON field encoding/decoding (tags, depends_on)
- Test filters (by status, agent, priority, tag)
- Test ErrNotFound on missing IDs
- Test concurrent access (SQLite WAL mode handles this)

All tests use the public interface (`Open`, `CreateTicket`, etc.) — testing through the API, not internals.
