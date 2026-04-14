# Store Package Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the store package with SQLite persistence, ticket/session CRUD, status validation, and per-project isolation.

**Architecture:** Each project gets its own SQLite DB. `Open()` takes a path + valid statuses. Types defined in their own files. `migrations.go` handles schema. `tickets.go` and `sessions.go` implement CRUD. All tests use temp dirs with fresh DBs.

**Tech Stack:** Go 1.26, modernc.org/sqlite, database/sql, encoding/json

---

### Task 1: Add SQLite Dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Get the dependency**

```bash
go get modernc.org/sqlite
```

- [ ] **Step 2: Verify**

```bash
go mod tidy && cat go.mod
```

Expected: `modernc.org/sqlite` appears in `go.mod`

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum && git commit -m "chore: add modernc.org/sqlite dependency"
```

---

### Task 2: Store Types + Sentinel Errors + Open/Close + Migrations

**Files:**
- Modify: `internal/store/store.go`
- Create: `internal/store/ticket.go`
- Create: `internal/store/session.go`
- Create: `internal/store/store_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/store/store_test.go`:

```go
package store

import (
	"os"
	"path/filepath"
	"testing"
)

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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/store/ -run "TestOpen|TestClose" -v
```

Expected: FAIL — `Open` undefined

- [ ] **Step 3: Implement types, errors, Open, Close, migrations**

Replace `internal/store/store.go`:

```go
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
}

func Open(dbPath string, validStatuses []string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("store.open: creating dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("store.open: %w", err)
	}

	db.SetMaxOpenConns(1)

	s := &Store{db: db, validStatuses: validStatuses}

	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
```

Replace `internal/store/migrations.go`:

```go
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
```

Create `internal/store/ticket.go`:

```go
package store

import (
	"encoding/json"
	"time"
)

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

type TicketFilters struct {
	Status   string
	Agent    string
	Priority string
	Tag      string
}

type ticketRow struct {
	ID          string
	Title       string
	Description string
	Status      string
	Priority    string
	Agent       string
	Branch      string
	Tags        string
	DependsOn   string
	CreatedAt   string
	UpdatedAt   string
}

func (r ticketRow) toTicket() (Ticket, error) {
	var tags []string
	if err := json.Unmarshal([]byte(r.Tags), &tags); err != nil {
		return Ticket{}, err
	}
	var dependsOn []string
	if err := json.Unmarshal([]byte(r.DependsOn), &dependsOn); err != nil {
		return Ticket{}, err
	}

	return Ticket{
		ID:          r.ID,
		Title:       r.Title,
		Description: r.Description,
		Status:      r.Status,
		Priority:    r.Priority,
		Agent:       r.Agent,
		Branch:      r.Branch,
		Tags:        tags,
		DependsOn:   dependsOn,
	}, nil
}
```

Create `internal/store/session.go`:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/store/ -run "TestOpen|TestClose" -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store/ && git commit -m "feat(store): add Open, Close, migrations, types, and sentinel errors"
```

---

### Task 3: Ticket Create + ID Generation

**Files:**
- Modify: `internal/store/tickets.go`
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/store/store_test.go`:

```go
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
```

Also add the helper at the top of the test file (after imports):

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/store/ -run "TestCreate" -v
```

Expected: FAIL — `CreateTicket` undefined

- [ ] **Step 3: Implement CreateTicket**

Append to `internal/store/tickets.go`:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const ticketPrefix = "AGT-"

var validPriorities = map[string]bool{
	"low": true, "medium": true, "high": true, "critical": true,
}

func (s *Store) isValidStatus(status string) bool {
	for _, v := range s.validStatuses {
		if v == status {
			return true
		}
	}
	return false
}

func (s *Store) nextTicketID(ctx context.Context) (string, error) {
	var maxID int
	err := s.db.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(CAST(SUBSTR(id, 5) AS INTEGER)), 0) FROM tickets",
	).Scan(&maxID)
	if err != nil {
		return "", fmt.Errorf("store.nextTicketID: %w", err)
	}
	return fmt.Sprintf("%s%02d", ticketPrefix, maxID+1), nil
}

func (s *Store) CreateTicket(ctx context.Context, t Ticket) (Ticket, error) {
	if t.Title == "" {
		return Ticket{}, fmt.Errorf("store.createTicket: title is required")
	}
	if !s.isValidStatus(t.Status) {
		return Ticket{}, fmt.Errorf("store.createTicket: %q: %w", t.Status, ErrInvalidStatus)
	}
	if t.Priority != "" && !validPriorities[t.Priority] {
		return Ticket{}, fmt.Errorf("store.createTicket: %q: %w", t.Priority, ErrInvalidPriority)
	}
	if t.Priority == "" {
		t.Priority = "medium"
	}

	id, err := s.nextTicketID(ctx)
	if err != nil {
		return Ticket{}, err
	}
	t.ID = id

	tags, err := json.Marshal(t.Tags)
	if err != nil {
		return Ticket{}, fmt.Errorf("store.createTicket: encoding tags: %w", err)
	}
	deps, err := json.Marshal(t.DependsOn)
	if err != nil {
		return Ticket{}, fmt.Errorf("store.createTicket: encoding depends_on: %w", err)
	}

	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO tickets (id, title, description, status, priority, agent, branch, tags, depends_on, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Title, t.Description, t.Status, t.Priority, t.Agent, t.Branch, string(tags), string(deps), t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return Ticket{}, fmt.Errorf("store.createTicket: %w", err)
	}

	return t, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/store/ -run "TestCreate" -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store/ && git commit -m "feat(store): add CreateTicket with ID generation and validation"
```

---

### Task 4: Ticket Get, List, Update, Delete, MoveStatus

**Files:**
- Modify: `internal/store/tickets.go`
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/store/store_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/store/ -run "TestGet|TestList|TestUpdate|TestDelete|TestMove" -v
```

Expected: FAIL — `GetTicket` undefined

- [ ] **Step 3: Implement GetTicket, ListTickets, UpdateTicket, DeleteTicket, MoveStatus**

Append to `internal/store/tickets.go`:

```go
func (s *Store) GetTicket(ctx context.Context, id string) (Ticket, error) {
	var r ticketRow
	err := s.db.QueryRowContext(ctx,
		"SELECT id, title, description, status, priority, agent, branch, tags, depends_on, created_at, updated_at FROM tickets WHERE id = ?",
		id,
	).Scan(&r.ID, &r.Title, &r.Description, &r.Status, &r.Priority, &r.Agent, &r.Branch, &r.Tags, &r.DependsOn, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return Ticket{}, fmt.Errorf("store.getTicket %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return Ticket{}, fmt.Errorf("store.getTicket %s: %w", id, err)
	}

	ticket, err := r.toTicket()
	if err != nil {
		return Ticket{}, fmt.Errorf("store.getTicket %s: %w", id, err)
	}

	return ticket, nil
}

func (s *Store) ListTickets(ctx context.Context, filters TicketFilters) ([]Ticket, error) {
	query := "SELECT id, title, description, status, priority, agent, branch, tags, depends_on, created_at, updated_at FROM tickets WHERE 1=1"
	var args []interface{}

	if filters.Status != "" {
		query += " AND status = ?"
		args = append(args, filters.Status)
	}
	if filters.Agent != "" {
		query += " AND agent = ?"
		args = append(args, filters.Agent)
	}
	if filters.Priority != "" {
		query += " AND priority = ?"
		args = append(args, filters.Priority)
	}
	if filters.Tag != "" {
		query += " AND tags LIKE ?"
		args = append(args, `%"`+filters.Tag+`"%`)
	}

	query += " ORDER BY created_at ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store.listTickets: %w", err)
	}
	defer rows.Close()

	var tickets []Ticket
	for rows.Next() {
		var r ticketRow
		if err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.Status, &r.Priority, &r.Agent, &r.Branch, &r.Tags, &r.DependsOn, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store.listTickets: %w", err)
		}
		ticket, err := r.toTicket()
		if err != nil {
			return nil, fmt.Errorf("store.listTickets: %w", err)
		}
		tickets = append(tickets, ticket)
	}

	return tickets, nil
}

func (s *Store) UpdateTicket(ctx context.Context, t Ticket) (Ticket, error) {
	if !s.isValidStatus(t.Status) {
		return Ticket{}, fmt.Errorf("store.updateTicket: %q: %w", t.Status, ErrInvalidStatus)
	}
	if t.Priority != "" && !validPriorities[t.Priority] {
		return Ticket{}, fmt.Errorf("store.updateTicket: %q: %w", t.Priority, ErrInvalidPriority)
	}

	tags, err := json.Marshal(t.Tags)
	if err != nil {
		return Ticket{}, fmt.Errorf("store.updateTicket: encoding tags: %w", err)
	}
	deps, err := json.Marshal(t.DependsOn)
	if err != nil {
		return Ticket{}, fmt.Errorf("store.updateTicket: encoding depends_on: %w", err)
	}

	t.UpdatedAt = time.Now()

	result, err := s.db.ExecContext(ctx,
		`UPDATE tickets SET title=?, description=?, status=?, priority=?, agent=?, branch=?, tags=?, depends_on=?, updated_at=? WHERE id=?`,
		t.Title, t.Description, t.Status, t.Priority, t.Agent, t.Branch, string(tags), string(deps), t.UpdatedAt, t.ID,
	)
	if err != nil {
		return Ticket{}, fmt.Errorf("store.updateTicket: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return Ticket{}, fmt.Errorf("store.updateTicket %s: %w", t.ID, ErrNotFound)
	}

	return t, nil
}

func (s *Store) DeleteTicket(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM tickets WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("store.deleteTicket: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("store.deleteTicket %s: %w", id, ErrNotFound)
	}

	return nil
}

func (s *Store) MoveStatus(ctx context.Context, id string, status string) error {
	if !s.isValidStatus(status) {
		return fmt.Errorf("store.moveStatus: %q: %w", status, ErrInvalidStatus)
	}

	result, err := s.db.ExecContext(ctx,
		"UPDATE tickets SET status = ?, updated_at = ? WHERE id = ?",
		status, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("store.moveStatus: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("store.moveStatus %s: %w", id, ErrNotFound)
	}

	return nil
}
```

Also add `"database/sql"` to the imports in `tickets.go`.

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/store/ -run "TestGet|TestList|TestUpdate|TestDelete|TestMove" -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store/ && git commit -m "feat(store): add ticket Get, List, Update, Delete, MoveStatus"
```

---

### Task 5: Session CRUD

**Files:**
- Modify: `internal/store/sessions.go`
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/store/store_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/store/ -run "TestCreateSession|TestGetSession|TestListSession|TestEndSession" -v
```

Expected: FAIL — `CreateSession` undefined

- [ ] **Step 3: Implement session CRUD**

Replace `internal/store/sessions.go`:

```go
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const sessionPrefix = "SES-"

func (s *Store) nextSessionID(ctx context.Context) (string, error) {
	var maxID int
	err := s.db.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(CAST(SUBSTR(id, 5) AS INTEGER)), 0) FROM sessions",
	).Scan(&maxID)
	if err != nil {
		return "", fmt.Errorf("store.nextSessionID: %w", err)
	}
	return fmt.Sprintf("%s%02d", sessionPrefix, maxID+1), nil
}

func (s *Store) CreateSession(ctx context.Context, sess Session) (Session, error) {
	id, err := s.nextSessionID(ctx)
	if err != nil {
		return Session{}, err
	}
	sess.ID = id

	now := time.Now()
	sess.StartedAt = now

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, ticket_id, agent, started_at, status, context_key) VALUES (?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.TicketID, sess.Agent, sess.StartedAt, sess.Status, sess.ContextKey,
	)
	if err != nil {
		return Session{}, fmt.Errorf("store.createSession: %w", err)
	}

	return sess, nil
}

func (s *Store) GetSession(ctx context.Context, id string) (Session, error) {
	var sess Session
	var endedAt sql.NullTime

	err := s.db.QueryRowContext(ctx,
		"SELECT id, ticket_id, agent, started_at, ended_at, status, context_key FROM sessions WHERE id = ?",
		id,
	).Scan(&sess.ID, &sess.TicketID, &sess.Agent, &sess.StartedAt, &endedAt, &sess.Status, &sess.ContextKey)
	if err == sql.ErrNoRows {
		return Session{}, fmt.Errorf("store.getSession %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return Session{}, fmt.Errorf("store.getSession %s: %w", id, err)
	}

	if endedAt.Valid {
		sess.EndedAt = &endedAt.Time
	}

	return sess, nil
}

func (s *Store) ListSessions(ctx context.Context, ticketID string) ([]Session, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, ticket_id, agent, started_at, ended_at, status, context_key FROM sessions WHERE ticket_id = ? ORDER BY started_at ASC",
		ticketID,
	)
	if err != nil {
		return nil, fmt.Errorf("store.listSessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		var endedAt sql.NullTime
		if err := rows.Scan(&sess.ID, &sess.TicketID, &sess.Agent, &sess.StartedAt, &endedAt, &sess.Status, &sess.ContextKey); err != nil {
			return nil, fmt.Errorf("store.listSessions: %w", err)
		}
		if endedAt.Valid {
			sess.EndedAt = &endedAt.Time
		}
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

func (s *Store) EndSession(ctx context.Context, id string, status string) error {
	now := time.Now()

	result, err := s.db.ExecContext(ctx,
		"UPDATE sessions SET ended_at = ?, status = ? WHERE id = ?",
		now, status, id,
	)
	if err != nil {
		return fmt.Errorf("store.endSession: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("store.endSession %s: %w", id, ErrNotFound)
	}

	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/store/ -run "TestCreateSession|TestGetSession|TestListSession|TestEndSession" -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store/ && git commit -m "feat(store): add session Create, Get, List, EndSession"
```

---

### Task 6: Final Verification

- [ ] **Step 1: Run all store tests**

```bash
go test ./internal/store/ -v
```

Expected: ALL PASS

- [ ] **Step 2: Run go vet**

```bash
go vet ./internal/store/
```

Expected: No issues

- [ ] **Step 3: Run full test suite**

```bash
go test ./...
```

Expected: PASS

- [ ] **Step 4: Run go build**

```bash
go build ./...
```

Expected: Builds successfully
