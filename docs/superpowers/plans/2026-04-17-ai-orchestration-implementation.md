# AI Orchestration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first end-to-end AI orchestration slice so moving an assigned ticket to `in_progress` creates an approval-gated run proposal, starts one worker agent through the orchestrator after approval, persists session/context-carry state, and moves the ticket to `review` on successful completion.

**Architecture:** AgentBoard owns orchestration, approvals, board transitions, and persistence in code. A cheap coordinator model prepares proposals and a cheap summarizer model compacts context carry, while an expensive worker CLI performs repo work through a subprocess runner. The worker reports structured outcomes; the orchestrator maps those outcomes to session and ticket transitions. All coordinator and summarizer model calls go through LangChain Go, isolated behind an `internal/llm` package.

**Tech Stack:** Go, Bubble Tea, SQLite via `modernc.org/sqlite`, LangChain Go (`github.com/tmc/langchaingo`) for coordinator and summarizer model access, existing config/store/tui packages, new orchestrator and llm packages, subprocess execution via `os/exec`

---

## File Structure

Expected files to modify or create in this first slice:

- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `internal/config/llm.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/defaults.go`
- Modify: `internal/config/loader.go`
- Modify: `internal/config/scaffold.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/config/defaults_test.go`
- Modify: `internal/config/scaffold_test.go`
- Modify: `internal/store/migrations.go`
- Modify: `internal/store/sqlite.go`
- Modify: `internal/store/sessions.go`
- Modify: `internal/store/tickets.go`
- Modify: `internal/store/store_test.go`
- Create: `internal/store/proposals.go`
- Create: `internal/store/events.go`
- Create: `internal/store/contextcarry.go`
- Create: `internal/llm/client.go`
- Create: `internal/llm/langchain.go`
- Create: `internal/llm/factory.go`
- Create: `internal/llm/langchain_test.go`
- Create: `internal/orchestrator/types.go`
- Create: `internal/orchestrator/service.go`
- Create: `internal/orchestrator/approval.go`
- Create: `internal/orchestrator/coordinator.go`
- Create: `internal/orchestrator/summarizer.go`
- Create: `internal/orchestrator/actions.go`
- Create: `internal/orchestrator/exec_runner.go`
- Create: `internal/orchestrator/fake_runner_test.go`
- Create: `internal/orchestrator/service_test.go`
- Create: `internal/orchestrator/approval_test.go`
- Create: `internal/orchestrator/coordinator_test.go`
- Create: `internal/orchestrator/actions_test.go`
- Create: `internal/orchestrator/exec_runner_test.go`
- Modify: `internal/mcp/contextcarry.go`
- Create: `internal/mcp/contextcarry_test.go`
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/app_commands.go`
- Modify: `internal/tui/kanban.go`
- Modify: `internal/tui/ticketview.go`
- Modify: `internal/tui/dashboard.go`
- Modify: `internal/tui/app_test.go`
- Modify: `internal/tui/kanban_test.go`
- Modify: `internal/tui/ticketview_test.go`
- Modify: `AGENTS.md`

Notes:

- Use LangChain Go (`github.com/tmc/langchaingo`) as the only coordinator/summarizer model integration layer in this slice.
- LangChain Go is isolated behind `internal/llm` so the orchestrator depends on AgentBoard-owned interfaces rather than LangChain symbols spread across runtime code.
- The core LangChain Go type is `llms.Model` (the interface). Provider-specific constructors live in subpackages like `llms/openai`, `llms/ollama`. The convenience function `llms.GenerateFromSinglePrompt(ctx, model, prompt, ...options)` handles single-turn text generation.
- Keep `internal/orchestrator` as the runtime owner. TUI and future API should call it, not reimplement its rules.
- Keep `store` persistence-focused. Transition rules belong in orchestrator services.
- The ExecRunner blocks synchronously; the TUI wraps it in a `tea.Cmd` (a closure returning a `tea.Msg`) so it runs off the main goroutine and never freezes the Bubble Tea event loop.

### Task 1: Add LangChain Dependency And `internal/llm` Package Skeleton

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Create: `internal/llm/client.go`
- Create: `internal/llm/langchain.go`
- Create: `internal/llm/factory.go`
- Create: `internal/llm/langchain_test.go`

- [ ] **Step 1: Write the failing llm package test**

```go
package llm_test

import (
	"testing"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/llm"
)

func TestFactoryReturnsClient(t *testing.T) {
	cfg := config.LLMConfig{
		CoordinatorProvider: "openai",
		CoordinatorModel:    "gpt-4o-mini",
		CoordinatorAPIKey:   "test-key",
	}

	client, err := llm.NewFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected client")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/llm -run TestFactoryReturnsClient -v`
Expected: FAIL because the package and dependency do not exist yet.

- [ ] **Step 3: Add LangChain Go dependency**

Run: `go get github.com/tmc/langchaingo@v0.1.14`

- [ ] **Step 4: Write the `internal/llm` package types and factory**

`internal/llm/client.go`:

```go
package llm

import "context"

type ProposalPrompt struct {
	TicketID      string
	Title         string
	Description   string
	AssignedAgent string
	ContextCarry  string
}

type ProposalDraft struct {
	Prompt string
}

type SummaryInput struct {
	TicketID string
	Outcome  string
	Summary  string
}

type Client interface {
	GenerateProposal(ctx context.Context, in ProposalPrompt) (ProposalDraft, error)
	SummarizeContext(ctx context.Context, in SummaryInput) (string, error)
}
```

`internal/llm/langchain.go`:

```go
package llm

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
)

type LangChainClient struct {
	coordinator llms.Model
	summarizer  llms.Model
}
```

`internal/llm/factory.go`:

```go
package llm

import (
	"fmt"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/tmc/langchaingo/llms/openai"
)

func NewFromConfig(cfg config.LLMConfig) (Client, error) {
	coordinator, err := newProviderModel(
		cfg.CoordinatorProvider,
		cfg.CoordinatorModel,
		cfg.CoordinatorAPIKey,
		cfg.CoordinatorBaseURL,
	)
	if err != nil {
		return nil, fmt.Errorf("llm.newFromConfig.coordinator: %w", err)
	}

	summarizer, err := newProviderModel(
		cfg.SummarizerProvider,
		cfg.SummarizerModel,
		cfg.SummarizerAPIKey,
		cfg.SummarizerBaseURL,
	)
	if err != nil {
		return nil, fmt.Errorf("llm.newFromConfig.summarizer: %w", err)
	}

	return &LangChainClient{
		coordinator: coordinator,
		summarizer:  summarizer,
	}, nil
}

func newProviderModel(provider, model, apiKey, baseURL string) (llms.Model, error) {
	switch provider {
	case "openai", "":
		opts := []openai.Option{}
		if apiKey != "" {
			opts = append(opts, openai.WithToken(apiKey))
		}
		if baseURL != "" {
			opts = append(opts, openai.WithBaseURL(baseURL))
		}
		if model != "" {
			opts = append(opts, openai.WithModel(model))
		}
		return openai.New(opts...)
	default:
		return nil, fmt.Errorf("llm.newProviderModel: unsupported provider %q", provider)
	}
}
```

Note: Only the `openai` provider is wired in slice 1. `ollama` and other providers will be added in later slices using the same factory pattern with their respective langchaingo subpackages (`llms/ollama`, etc.).

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/llm -run TestFactoryReturnsClient -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/llm/client.go internal/llm/langchain.go internal/llm/factory.go internal/llm/langchain_test.go
git commit -m "feat: add langchain llm integration layer"
```

### Task 2: Expand Config For Coordinator And Summarizer

**Files:**
- Modify: `internal/config/llm.go`
- Modify: `internal/config/defaults.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/loader.go`
- Modify: `internal/config/scaffold.go`
- Test: `internal/config/config_test.go`
- Test: `internal/config/defaults_test.go`
- Test: `internal/config/scaffold_test.go`

- [ ] **Step 1: Write the failing config tests**

Add to the appropriate test files in `internal/config/`:

```go
func TestSetDefaultsProvidesOrchestrationLLMDefaults(t *testing.T) {
	cfg := SetDefaults()

	if cfg.LLM.CoordinatorProvider != "" {
		t.Fatalf("CoordinatorProvider = %q, want empty", cfg.LLM.CoordinatorProvider)
	}
	if cfg.LLM.SummarizerProvider != "" {
		t.Fatalf("SummarizerProvider = %q, want empty", cfg.LLM.SummarizerProvider)
	}
	if cfg.LLM.RequireApproval != true {
		t.Fatalf("RequireApproval = %v, want true", cfg.LLM.RequireApproval)
	}
}

func TestLoadFromDirReadsCoordinatorFields(t *testing.T) {
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "projects", "agent-board")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(projectDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
[llm]
coordinator_provider = "ollama"
coordinator_model = "qwen2.5-coder"
coordinator_base_url = "http://127.0.0.1:11434"
summarizer_provider = "ollama"
summarizer_model = "qwen2.5:7b"
require_approval = true
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromDir(baseDir, "agent-board")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.LLM.CoordinatorProvider != "ollama" {
		t.Fatalf("CoordinatorProvider = %q, want ollama", cfg.LLM.CoordinatorProvider)
	}
	if cfg.LLM.SummarizerModel != "qwen2.5:7b" {
		t.Fatalf("SummarizerModel = %q, want qwen2.5:7b", cfg.LLM.SummarizerModel)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run 'Test(SetDefaultsProvidesOrchestrationLLMDefaults|LoadFromDirReadsCoordinatorFields)' -v`
Expected: FAIL because `LLMConfig` does not yet expose orchestration fields.

- [ ] **Step 3: Expand `LLMConfig` and loader**

`internal/config/llm.go`:

```go
package config

type LLMConfig struct {
	Provider string `toml:"provider"`
	Model    string `toml:"model"`
	APIKey   string `toml:"api_key"`
	BaseURL  string `toml:"base_url"`

	CoordinatorProvider string `toml:"coordinator_provider"`
	CoordinatorModel    string `toml:"coordinator_model"`
	CoordinatorAPIKey   string `toml:"coordinator_api_key"`
	CoordinatorBaseURL  string `toml:"coordinator_base_url"`

	SummarizerProvider string `toml:"summarizer_provider"`
	SummarizerModel    string `toml:"summarizer_model"`
	SummarizerAPIKey   string `toml:"summarizer_api_key"`
	SummarizerBaseURL  string `toml:"summarizer_base_url"`

	RequireApproval bool `toml:"require_approval"`
}
```

In `defaults.go`, add `RequireApproval: true` to the LLM default block.

In `loader.go`, add fallback logic after loading TOML:

```go
if cfg.LLM.CoordinatorProvider == "" {
	cfg.LLM.CoordinatorProvider = cfg.LLM.Provider
	cfg.LLM.CoordinatorModel = cfg.LLM.Model
	cfg.LLM.CoordinatorAPIKey = cfg.LLM.APIKey
	cfg.LLM.CoordinatorBaseURL = cfg.LLM.BaseURL
}
if cfg.LLM.SummarizerProvider == "" {
	cfg.LLM.SummarizerProvider = cfg.LLM.CoordinatorProvider
	cfg.LLM.SummarizerModel = cfg.LLM.CoordinatorModel
	cfg.LLM.SummarizerAPIKey = cfg.LLM.CoordinatorAPIKey
	cfg.LLM.SummarizerBaseURL = cfg.LLM.CoordinatorBaseURL
}
```

In `scaffold.go`, add the new fields to the scaffolded config template if it writes one.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config -run 'Test(SetDefaultsProvidesOrchestrationLLMDefaults|LoadFromDirReadsCoordinatorFields)' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/llm.go internal/config/defaults.go internal/config/config.go internal/config/loader.go internal/config/scaffold.go internal/config/config_test.go internal/config/defaults_test.go internal/config/scaffold_test.go
git commit -m "feat: add orchestration llm config"
```

### Task 3: Add Store Tables And CRUD For Proposals, Events, And Context Carry

**Files:**
- Modify: `internal/store/migrations.go`
- Modify: `internal/store/sqlite.go`
- Modify: `internal/store/sessions.go`
- Modify: `internal/store/tickets.go`
- Create: `internal/store/proposals.go`
- Create: `internal/store/events.go`
- Create: `internal/store/contextcarry.go`
- Test: `internal/store/store_test.go`

Note: The existing `Ticket` struct already has `AgentActive bool` and the store already has `SetAgentActive()` and `MoveStatus()` methods. This task adds the new tables and accessors.

- [ ] **Step 1: Write the failing store tests**

Add to `internal/store/store_test.go`:

```go
func TestStoreCreateAndApproveProposal(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	proposal, err := s.CreateProposal(ctx, Proposal{
		TicketID: "AGE-01",
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

	if err := s.UpdateProposalStatus(ctx, proposal.ID, "approved"); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetProposal(ctx, proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "approved" {
		t.Fatalf("Status = %q, want approved", got.Status)
	}
}

func TestStorePersistsContextCarry(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	if err := s.UpsertContextCarry(ctx, ContextCarry{
		TicketID: "AGE-01",
		Summary:  "previous run summary",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetContextCarry(ctx, "AGE-01")
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary != "previous run summary" {
		t.Fatalf("Summary = %q, want previous run summary", got.Summary)
	}
}

func TestStoreCreateAndRetrieveEvent(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	event, err := s.CreateEvent(ctx, Event{
		TicketID: "AGE-01",
		Kind:     "proposal.created",
		Payload:  "test payload",
	})
	if err != nil {
		t.Fatal(err)
	}

	if event.ID == "" {
		t.Fatal("expected event ID")
	}
	if event.TicketID != "AGE-01" {
		t.Fatalf("TicketID = %q, want AGE-01", event.TicketID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store -run 'TestStore(CreateAndApproveProposal|PersistsContextCarry|CreateAndRetrieveEvent)' -v`
Expected: FAIL because proposal, context-carry, and event storage do not exist yet.

- [ ] **Step 3: Write minimal migrations and store APIs**

Add to `internal/store/migrations.go` (append new tables to the existing `migrate()` method):

```sql
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
```

`internal/store/proposals.go`:

```go
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Proposal struct {
	ID        string
	TicketID  string
	Agent     string
	Status    string
	Prompt    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

const proposalPrefix = "PRO-"

func (s *Store) nextProposalID(ctx context.Context) (string, error) {
	var maxID int
	err := s.db.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(CAST(SUBSTR(id, 5) AS INTEGER)), 0) FROM proposals",
	).Scan(&maxID)
	if err != nil {
		return "", fmt.Errorf("store.nextProposalID: %w", err)
	}
	return fmt.Sprintf("%s%02d", proposalPrefix, maxID+1), nil
}

func (s *Store) CreateProposal(ctx context.Context, p Proposal) (Proposal, error) {
	id, err := s.nextProposalID(ctx)
	if err != nil {
		return Proposal{}, err
	}
	p.ID = id

	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO proposals (id, ticket_id, agent, status, prompt, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.TicketID, p.Agent, p.Status, p.Prompt, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return Proposal{}, fmt.Errorf("store.createProposal: %w", err)
	}

	return p, nil
}

func (s *Store) GetProposal(ctx context.Context, id string) (Proposal, error) {
	var p Proposal
	err := s.db.QueryRowContext(ctx,
		"SELECT id, ticket_id, agent, status, prompt, created_at, updated_at FROM proposals WHERE id = ?",
		id,
	).Scan(&p.ID, &p.TicketID, &p.Agent, &p.Status, &p.Prompt, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return Proposal{}, fmt.Errorf("store.getProposal %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return Proposal{}, fmt.Errorf("store.getProposal %s: %w", id, err)
	}
	return p, nil
}

func (s *Store) UpdateProposalStatus(ctx context.Context, id, status string) error {
	result, err := s.db.ExecContext(ctx,
		"UPDATE proposals SET status = ?, updated_at = ? WHERE id = ?",
		status, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("store.updateProposalStatus: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("store.updateProposalStatus %s: %w", id, ErrNotFound)
	}
	return nil
}
```

`internal/store/events.go`:

```go
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID        string
	TicketID  string
	SessionID string
	Kind      string
	Payload   string
	CreatedAt time.Time
}

func (s *Store) CreateEvent(ctx context.Context, e Event) (Event, error) {
	e.ID = uuid.New().String()
	e.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO orchestration_events (id, ticket_id, session_id, kind, payload, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		e.ID, e.TicketID, e.SessionID, e.Kind, e.Payload, e.CreatedAt,
	)
	if err != nil {
		return Event{}, fmt.Errorf("store.createEvent: %w", err)
	}

	return e, nil
}
```

`internal/store/contextcarry.go`:

```go
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type ContextCarry struct {
	TicketID  string
	Summary   string
	UpdatedAt time.Time
}

func (s *Store) UpsertContextCarry(ctx context.Context, cc ContextCarry) error {
	cc.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO context_carry (ticket_id, summary, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(ticket_id) DO UPDATE SET summary = excluded.summary, updated_at = excluded.updated_at`,
		cc.TicketID, cc.Summary, cc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("store.upsertContextCarry: %w", err)
	}
	return nil
}

func (s *Store) GetContextCarry(ctx context.Context, ticketID string) (ContextCarry, error) {
	var cc ContextCarry
	err := s.db.QueryRowContext(ctx,
		"SELECT ticket_id, summary, updated_at FROM context_carry WHERE ticket_id = ?",
		ticketID,
	).Scan(&cc.TicketID, &cc.Summary, &cc.UpdatedAt)
	if err == sql.ErrNoRows {
		return ContextCarry{}, fmt.Errorf("store.getContextCarry %s: %w", ticketID, ErrNotFound)
	}
	if err != nil {
		return ContextCarry{}, fmt.Errorf("store.getContextCarry %s: %w", ticketID, err)
	}
	return cc, nil
}
```

Also add `HasActiveSession` to `internal/store/sessions.go`:

```go
func (s *Store) HasActiveSession(ctx context.Context, ticketID string) bool {
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sessions WHERE ticket_id = ? AND ended_at IS NULL",
		ticketID,
	).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/store -run 'TestStore(CreateAndApproveProposal|PersistsContextCarry|CreateAndRetrieveEvent)' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store/migrations.go internal/store/sqlite.go internal/store/sessions.go internal/store/tickets.go internal/store/proposals.go internal/store/events.go internal/store/contextcarry.go internal/store/store_test.go
git commit -m "feat: add orchestration persistence"
```

### Task 4: Define Orchestrator Domain Types And Local Interfaces

**Files:**
- Create: `internal/orchestrator/types.go`
- Create: `internal/orchestrator/service.go`
- Test: `internal/orchestrator/service_test.go`

Note: Remove existing empty stub files (`agent.go`, `session.go`, `spawner.go`, `pty.go`) if they conflict.

- [ ] **Step 1: Write the failing orchestrator type test**

```go
package orchestrator_test

import (
	"context"
	"testing"

	"github.com/ayan-de/agent-board/internal/orchestrator"
)

func TestServiceCreateProposalRequiresAssignedAgent(t *testing.T) {
	svc := orchestrator.Service{}
	_, err := svc.CreateProposal(context.Background(), orchestrator.CreateProposalInput{
		TicketID: "AGE-01",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestrator -run TestServiceCreateProposalRequiresAssignedAgent -v`
Expected: FAIL because the service types do not exist yet.

- [ ] **Step 3: Write minimal orchestrator types**

`internal/orchestrator/types.go`:

```go
package orchestrator

import (
	"context"

	"github.com/ayan-de/agent-board/internal/llm"
	"github.com/ayan-de/agent-board/internal/store"
)

type CreateProposalInput struct {
	TicketID string
}

type ApplyRunOutcomeInput struct {
	TicketID string
	Outcome  string
}

type FinishRunInput struct {
	TicketID  string
	SessionID string
	Outcome   string
	Summary   string
}

type LLMClient interface {
	GenerateProposal(ctx context.Context, input llm.ProposalPrompt) (llm.ProposalDraft, error)
	SummarizeContext(ctx context.Context, input llm.SummaryInput) (string, error)
}

type Runner interface {
	Start(ctx context.Context, req RunRequest) (RunHandle, error)
}

type RunRequest struct {
	TicketID  string
	SessionID string
	Agent     string
	Prompt    string
}

type RunHandle struct {
	Outcome string
	Summary string
}

type Store interface {
	GetTicket(ctx context.Context, id string) (store.Ticket, error)
	CreateProposal(ctx context.Context, p store.Proposal) (store.Proposal, error)
	GetProposal(ctx context.Context, id string) (store.Proposal, error)
	UpdateProposalStatus(ctx context.Context, id, status string) error
	GetContextCarry(ctx context.Context, ticketID string) (store.ContextCarry, error)
	UpsertContextCarry(ctx context.Context, cc store.ContextCarry) error
	CreateSession(ctx context.Context, sess store.Session) (store.Session, error)
	EndSession(ctx context.Context, id, status string) error
	HasActiveSession(ctx context.Context, ticketID string) bool
	SetAgentActive(ctx context.Context, id string, active bool) error
	MoveStatus(ctx context.Context, id, status string) error
	CreateEvent(ctx context.Context, e store.Event) (store.Event, error)
}
```

`internal/orchestrator/service.go`:

```go
package orchestrator

import (
	"context"
	"fmt"
)

type Service struct {
	store  Store
	llm    LLMClient
	runner Runner
}

func NewService(store Store, llm LLMClient, runner Runner) *Service {
	return &Service{store: store, llm: llm, runner: runner}
}

func (s Service) CreateProposal(ctx context.Context, input CreateProposalInput) (store.Proposal, error) {
	ticket, err := s.store.GetTicket(ctx, input.TicketID)
	if err != nil {
		return store.Proposal{}, err
	}
	if ticket.Agent == "" {
		return store.Proposal{}, fmt.Errorf("orchestrator.createProposal: assigned agent is required")
	}
	return store.Proposal{}, fmt.Errorf("orchestrator.createProposal: not implemented")
}
```

Note: This imports `store` directly in the return type. The full import of `"github.com/ayan-de/agent-board/internal/store"` is needed in `service.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/orchestrator -run TestServiceCreateProposalRequiresAssignedAgent -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/types.go internal/orchestrator/service.go internal/orchestrator/service_test.go
git commit -m "feat: define orchestrator service types"
```

### Task 5: Implement LangChain Prompting For Proposal Creation And Summarization

**Files:**
- Modify: `internal/llm/langchain.go`
- Modify: `internal/llm/langchain_test.go`

- [ ] **Step 1: Write the failing LangChain prompting test**

```go
package llm_test

import (
	"context"
	"testing"

	"github.com/ayan-de/agent-board/internal/llm"
)

type fakeModel struct {
	response string
}

func (f fakeModel) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{Content: f.response},
		},
	}, nil
}

func (f fakeModel) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return f.response, nil
}

func TestGenerateProposalBuildsPromptFromTicketContext(t *testing.T) {
	client := llm.LangChainClient{
		Coordinator: fakeModel{response: "worker prompt"},
		Summarizer:  fakeModel{response: "summary"},
	}

	got, err := client.GenerateProposal(context.Background(), llm.ProposalPrompt{
		TicketID:      "AGE-01",
		Title:         "Add orchestrator",
		Description:   "Build service layer",
		AssignedAgent: "opencode",
		ContextCarry:  "prior run summary",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Prompt != "worker prompt" {
		t.Fatalf("Prompt = %q, want worker prompt", got.Prompt)
	}
}
```

Note: `LangChainClient` fields are exported so tests in the external test package can construct them. The `llms.Model` interface requires `GenerateContent`. The test creates a `fakeModel` that satisfies `llms.Model`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/llm -run TestGenerateProposalBuildsPromptFromTicketContext -v`
Expected: FAIL because LangChain prompt execution is not implemented.

- [ ] **Step 3: Write minimal LangChain prompt implementation**

Update `internal/llm/langchain.go`:

```go
package llm

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
)

type LangChainClient struct {
	Coordinator llms.Model
	Summarizer  llms.Model
}

func (c LangChainClient) GenerateProposal(ctx context.Context, in ProposalPrompt) (ProposalDraft, error) {
	prompt := fmt.Sprintf(
		"Ticket ID: %s\nTitle: %s\nDescription: %s\nAssigned agent: %s\nContext carry: %s\n\nReturn only the worker prompt.",
		in.TicketID,
		in.Title,
		in.Description,
		in.AssignedAgent,
		in.ContextCarry,
	)
	text, err := llms.GenerateFromSinglePrompt(ctx, c.Coordinator, prompt)
	if err != nil {
		return ProposalDraft{}, fmt.Errorf("llm.generateProposal: %w", err)
	}
	return ProposalDraft{Prompt: text}, nil
}

func (c LangChainClient) SummarizeContext(ctx context.Context, in SummaryInput) (string, error) {
	prompt := fmt.Sprintf(
		"Ticket ID: %s\nOutcome: %s\nWorker summary: %s\n\nReturn a compact resumable context summary.",
		in.TicketID,
		in.Outcome,
		in.Summary,
	)
	text, err := llms.GenerateFromSinglePrompt(ctx, c.Summarizer, prompt)
	if err != nil {
		return "", fmt.Errorf("llm.summarizeContext: %w", err)
	}
	return text, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/llm -run TestGenerateProposalBuildsPromptFromTicketContext -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/llm/langchain.go internal/llm/langchain_test.go
git commit -m "feat: add langchain proposal and summary prompts"
```

### Task 6: Implement Proposal Creation For `in_progress` Execution Requests

**Files:**
- Create: `internal/orchestrator/coordinator.go`
- Create: `internal/orchestrator/coordinator_test.go`
- Modify: `internal/orchestrator/service.go`

- [ ] **Step 1: Write the failing proposal-generation test**

Create `internal/orchestrator/coordinator_test.go` with test helpers and the test:

```go
package orchestrator_test

import (
	"context"
	"testing"
	"time"

	"github.com/ayan-de/agent-board/internal/llm"
	"github.com/ayan-de/agent-board/internal/orchestrator"
	"github.com/ayan-de/agent-board/internal/store"
)

type fakeStore struct {
	ticket       store.Ticket
	proposal     store.Proposal
	contextCarry store.ContextCarry

	lastProposal store.Proposal
}

func (f *fakeStore) GetTicket(_ context.Context, _ string) (store.Ticket, error) {
	return f.ticket, nil
}
func (f *fakeStore) CreateProposal(_ context.Context, p store.Proposal) (store.Proposal, error) {
	p.ID = "PRO-01"
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	f.lastProposal = p
	return p, nil
}
func (f *fakeStore) GetProposal(_ context.Context, _ string) (store.Proposal, error) {
	return f.proposal, nil
}
func (f *fakeStore) UpdateProposalStatus(_ context.Context, _, _ string) error { return nil }
func (f *fakeStore) GetContextCarry(_ context.Context, _ string) (store.ContextCarry, error) {
	return f.contextCarry, nil
}
func (f *fakeStore) UpsertContextCarry(_ context.Context, _ store.ContextCarry) error {
	return nil
}
func (f *fakeStore) CreateSession(_ context.Context, s store.Session) (store.Session, error) {
	return s, nil
}
func (f *fakeStore) EndSession(_ context.Context, _, _ string) error     { return nil }
func (f *fakeStore) HasActiveSession(_ context.Context, _ string) bool    { return false }
func (f *fakeStore) SetAgentActive(_ context.Context, _ string, _ bool) error { return nil }
func (f *fakeStore) MoveStatus(_ context.Context, _, _ string) error      { return nil }
func (f *fakeStore) CreateEvent(_ context.Context, e store.Event) (store.Event, error) {
	return e, nil
}

type fakeLLMClient struct {
	proposal     llm.ProposalDraft
	summary      string
	lastProposal llm.ProposalPrompt
}

func (f *fakeLLMClient) GenerateProposal(_ context.Context, in llm.ProposalPrompt) (llm.ProposalDraft, error) {
	f.lastProposal = in
	return f.proposal, nil
}
func (f *fakeLLMClient) SummarizeContext(_ context.Context, _ llm.SummaryInput) (string, error) {
	return f.summary, nil
}

func TestCreateProposalUsesTicketAndContextCarry(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{
			ID:          "AGE-01",
			Title:       "Add orchestrator",
			Description: "Build orchestration flow",
			Status:      "in_progress",
			Agent:       "opencode",
		},
		contextCarry: store.ContextCarry{
			TicketID: "AGE-01",
			Summary:  "prior run summary",
		},
	}
	fllm := &fakeLLMClient{
		proposal: llm.ProposalDraft{
			Prompt: "work with prior run summary",
		},
	}
	svc := orchestrator.NewService(fs, fllm, nil)

	proposal, err := svc.CreateProposal(context.Background(), orchestrator.CreateProposalInput{
		TicketID: "AGE-01",
	})
	if err != nil {
		t.Fatal(err)
	}

	if proposal.TicketID != "AGE-01" {
		t.Fatalf("TicketID = %q, want AGE-01", proposal.TicketID)
	}
	if fllm.lastProposal.ContextCarry != "prior run summary" {
		t.Fatalf("ContextCarry = %q, want prior run summary", fllm.lastProposal.ContextCarry)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestrator -run TestCreateProposalUsesTicketAndContextCarry -v`
Expected: FAIL because proposal shaping is not implemented.

- [ ] **Step 3: Write minimal coordinator implementation**

Update `internal/orchestrator/service.go` to replace the stub `CreateProposal`:

```go
func (s Service) CreateProposal(ctx context.Context, input CreateProposalInput) (store.Proposal, error) {
	ticket, err := s.store.GetTicket(ctx, input.TicketID)
	if err != nil {
		return store.Proposal{}, err
	}
	if ticket.Agent == "" {
		return store.Proposal{}, fmt.Errorf("orchestrator.createProposal: assigned agent is required")
	}

	cc, _ := s.store.GetContextCarry(ctx, input.TicketID)

	draft, err := s.llm.GenerateProposal(ctx, llm.ProposalPrompt{
		TicketID:      ticket.ID,
		Title:         ticket.Title,
		Description:   ticket.Description,
		AssignedAgent: ticket.Agent,
		ContextCarry:  cc.Summary,
	})
	if err != nil {
		return store.Proposal{}, fmt.Errorf("orchestrator.createProposal: %w", err)
	}

	proposal, err := s.store.CreateProposal(ctx, store.Proposal{
		TicketID: ticket.ID,
		Agent:    ticket.Agent,
		Status:   "pending",
		Prompt:   draft.Prompt,
	})
	if err != nil {
		return store.Proposal{}, err
	}

	_, _ = s.store.CreateEvent(ctx, store.Event{
		TicketID: ticket.ID,
		Kind:     "proposal.created",
		Payload:  draft.Prompt,
	})

	return proposal, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/orchestrator -run TestCreateProposalUsesTicketAndContextCarry -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/coordinator.go internal/orchestrator/coordinator_test.go internal/orchestrator/service.go
git commit -m "feat: create execution proposals from ticket context"
```

### Task 7: Implement Approval Service And Stale Proposal Guard

**Files:**
- Create: `internal/orchestrator/approval.go`
- Create: `internal/orchestrator/approval_test.go`

- [ ] **Step 1: Write the failing approval test**

```go
func TestApproveProposalRejectsStaleTicketState(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{
			ID:        "AGE-01",
			Title:     "Updated title",
			Agent:     "opencode",
			Status:    "in_progress",
			UpdatedAt: time.Unix(20, 0),
		},
		proposal: store.Proposal{
			ID:        "PRO-01",
			TicketID:  "AGE-01",
			Agent:     "opencode",
			Status:    "pending",
			CreatedAt: time.Unix(10, 0),
		},
	}
	svc := orchestrator.NewService(fs, nil, nil)

	err := svc.ApproveProposal(context.Background(), "PRO-01")
	if err == nil {
		t.Fatal("expected stale proposal error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestrator -run TestApproveProposalRejectsStaleTicketState -v`
Expected: FAIL because approval rules are not implemented.

- [ ] **Step 3: Write minimal approval implementation**

`internal/orchestrator/approval.go`:

```go
package orchestrator

import (
	"context"
	"fmt"
)

func (s Service) ApproveProposal(ctx context.Context, proposalID string) error {
	proposal, err := s.store.GetProposal(ctx, proposalID)
	if err != nil {
		return err
	}
	ticket, err := s.store.GetTicket(ctx, proposal.TicketID)
	if err != nil {
		return err
	}
	if ticket.UpdatedAt.After(proposal.CreatedAt) {
		return fmt.Errorf("orchestrator.approveProposal: proposal is stale")
	}
	return s.store.UpdateProposalStatus(ctx, proposalID, "approved")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/orchestrator -run TestApproveProposalRejectsStaleTicketState -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/approval.go internal/orchestrator/approval_test.go
git commit -m "feat: add proposal approval guards"
```

### Task 8: Implement Board Action Rules And Ticket-Scoped Permissions

**Files:**
- Create: `internal/orchestrator/actions.go`
- Create: `internal/orchestrator/actions_test.go`

Note: `store.SetAgentActive()` and `store.MoveStatus()` already exist. No changes to `internal/store/tickets.go` are needed.

- [ ] **Step 1: Write the failing board-action test**

```go
func TestApplyRunOutcomeMovesTicketToReview(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{
			ID:          "AGE-01",
			Status:      "in_progress",
			Agent:       "opencode",
			AgentActive: true,
		},
	}
	svc := orchestrator.NewService(fs, nil, nil)

	err := svc.ApplyRunOutcome(context.Background(), orchestrator.ApplyRunOutcomeInput{
		TicketID: "AGE-01",
		Outcome:  "completed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if fs.lastMoveStatus != "review" {
		t.Fatalf("MoveStatus = %q, want review", fs.lastMoveStatus)
	}
	if fs.lastAgentActive != false {
		t.Fatalf("AgentActive = %v, want false", fs.lastAgentActive)
	}
}
```

Note: Add `lastMoveStatus string` and `lastAgentActive bool` tracking fields to `fakeStore` in the test helper, and update the `MoveStatus` and `SetAgentActive` fake implementations to record the last values.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestrator -run TestApplyRunOutcomeMovesTicketToReview -v`
Expected: FAIL because outcome mapping does not exist.

- [ ] **Step 3: Write minimal board-action implementation**

`internal/orchestrator/actions.go`:

```go
package orchestrator

import (
	"context"
	"fmt"
)

func (s Service) ApplyRunOutcome(ctx context.Context, input ApplyRunOutcomeInput) error {
	switch input.Outcome {
	case "completed":
		if err := s.store.SetAgentActive(ctx, input.TicketID, false); err != nil {
			return err
		}
		return s.store.MoveStatus(ctx, input.TicketID, "review")
	case "failed", "interrupted", "blocked":
		return s.store.SetAgentActive(ctx, input.TicketID, false)
	default:
		return fmt.Errorf("orchestrator.applyRunOutcome: unknown outcome %q", input.Outcome)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/orchestrator -run TestApplyRunOutcomeMovesTicketToReview -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/actions.go internal/orchestrator/actions_test.go
git commit -m "feat: add outcome-driven board transitions"
```

### Task 9: Implement Session Start And Duplicate Active Run Protection

**Files:**
- Modify: `internal/orchestrator/service.go`
- Modify: `internal/orchestrator/service_test.go`

- [ ] **Step 1: Write the failing active-session test**

```go
func TestStartApprovedRunRejectsExistingActiveSession(t *testing.T) {
	fs := &fakeStore{
		activeSession: true,
		proposal: store.Proposal{
			ID:       "PRO-01",
			TicketID: "AGE-01",
			Agent:    "opencode",
			Status:   "approved",
			Prompt:   "do work",
		},
	}
	svc := orchestrator.NewService(fs, nil, &fakeRunner{})

	_, err := svc.StartApprovedRun(context.Background(), "PRO-01")
	if err == nil {
		t.Fatal("expected duplicate active session error")
	}
}
```

Note: Add `activeSession bool` and `fakeRunner` to the test helper file. `fakeRunner` is a minimal `Runner` implementation.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestrator -run TestStartApprovedRunRejectsExistingActiveSession -v`
Expected: FAIL because start flow is not implemented.

- [ ] **Step 3: Write minimal start-run implementation**

Add to `internal/orchestrator/service.go`:

```go
func (s Service) StartApprovedRun(ctx context.Context, proposalID string) (store.Session, error) {
	proposal, err := s.store.GetProposal(ctx, proposalID)
	if err != nil {
		return store.Session{}, err
	}
	if proposal.Status != "approved" {
		return store.Session{}, fmt.Errorf("orchestrator.startApprovedRun: proposal is not approved")
	}
	if s.store.HasActiveSession(ctx, proposal.TicketID) {
		return store.Session{}, fmt.Errorf("orchestrator.startApprovedRun: active session exists")
	}

	session, err := s.store.CreateSession(ctx, store.Session{
		TicketID: proposal.TicketID,
		Agent:    proposal.Agent,
		Status:   "running",
	})
	if err != nil {
		return store.Session{}, err
	}

	if err := s.store.SetAgentActive(ctx, proposal.TicketID, true); err != nil {
		return store.Session{}, err
	}

	handle, err := s.runner.Start(ctx, RunRequest{
		TicketID:  proposal.TicketID,
		SessionID: session.ID,
		Agent:     proposal.Agent,
		Prompt:    proposal.Prompt,
	})
	if err != nil {
		_ = s.store.EndSession(ctx, session.ID, "failed")
		_ = s.store.SetAgentActive(ctx, proposal.TicketID, false)
		return store.Session{}, err
	}

	_, _ = s.store.CreateEvent(ctx, store.Event{
		TicketID:  proposal.TicketID,
		SessionID: session.ID,
		Kind:      "run.started",
		Payload:   handle.Outcome,
	})

	return session, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/orchestrator -run TestStartApprovedRunRejectsExistingActiveSession -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/service.go internal/orchestrator/service_test.go
git commit -m "feat: start approved runs with session guards"
```

### Task 10: Implement Subprocess `exec` Runner And Structured Outcome Parsing

**Files:**
- Create: `internal/orchestrator/exec_runner.go`
- Create: `internal/orchestrator/exec_runner_test.go`

Note: The `ExecRunner.Start()` method blocks on `cmd.Output()`. This is intentional -- the TUI wraps it in a `tea.Cmd` closure (Task 13) so it runs off the Bubble Tea main goroutine and never freezes the UI.

- [ ] **Step 1: Write the failing exec-runner test**

```go
package orchestrator_test

import (
	"context"
	"testing"

	"github.com/ayan-de/agent-board/internal/orchestrator"
)

type fakeCmdRunner struct {
	stdout   string
	stderr   string
	runError error
}

func (f fakeCmdRunner) Output() ([]byte, error) {
	if f.runError != nil {
		return nil, f.runError
	}
	return []byte(f.stdout), nil
}

func TestExecRunnerParsesStructuredOutcome(t *testing.T) {
	runner := orchestrator.ExecRunner{
		LookPath: func(name string) (string, error) {
			return "/bin/echo", nil
		},
		Command: func(ctx context.Context, name string, args ...string) orchestrator.CmdRunner {
			return fakeCmdRunner{
				stdout: `{"outcome":"completed","summary":"done"}`,
			}
		},
	}

	handle, err := runner.Start(context.Background(), orchestrator.RunRequest{
		TicketID: "AGE-01",
		Agent:    "opencode",
		Prompt:   "do work",
	})
	if err != nil {
		t.Fatal(err)
	}
	if handle.Outcome != "completed" {
		t.Fatalf("Outcome = %q, want completed", handle.Outcome)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestrator -run TestExecRunnerParsesStructuredOutcome -v`
Expected: FAIL because the runner does not exist.

- [ ] **Step 3: Write minimal exec-runner implementation**

`internal/orchestrator/exec_runner.go`:

```go
package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type CmdRunner interface {
	Output() ([]byte, error)
}

type ExecRunner struct {
	LookPath func(string) (string, error)
	Command  func(context.Context, string, ...string) CmdRunner
}

func NewExecRunner() *ExecRunner {
	return &ExecRunner{
		LookPath: exec.LookPath,
		Command: func(ctx context.Context, name string, args ...string) CmdRunner {
			return exec.CommandContext(ctx, name, args...)
		},
	}
}

func (r ExecRunner) Start(ctx context.Context, req RunRequest) (RunHandle, error) {
	path, err := r.LookPath(req.Agent)
	if err != nil {
		return RunHandle{}, fmt.Errorf("execRunner.start: %w", err)
	}
	cmd := r.Command(ctx, path, req.Prompt)
	out, err := cmd.Output()
	if err != nil {
		return RunHandle{Outcome: "failed", Summary: err.Error()}, nil
	}
	var result struct {
		Outcome string `json:"outcome"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return RunHandle{Outcome: "interrupted", Summary: string(out)}, nil
	}
	return RunHandle{Outcome: result.Outcome, Summary: result.Summary}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/orchestrator -run TestExecRunnerParsesStructuredOutcome -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/exec_runner.go internal/orchestrator/exec_runner_test.go
git commit -m "feat: add subprocess exec runner"
```

### Task 11: Persist Context Carry From Worker Outcomes

**Files:**
- Create: `internal/orchestrator/summarizer.go`
- Modify: `internal/orchestrator/service.go`
- Modify: `internal/orchestrator/service_test.go`

- [ ] **Step 1: Write the failing context-carry test**

```go
func TestFinishRunPersistsContextCarry(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{
			ID:     "AGE-01",
			Status: "in_progress",
		},
	}
	fllm := &fakeLLMClient{summary: "short handoff summary"}
	svc := orchestrator.NewService(fs, fllm, nil)

	err := svc.FinishRun(context.Background(), orchestrator.FinishRunInput{
		TicketID:  "AGE-01",
		SessionID: "SES-01",
		Outcome:   "completed",
		Summary:   "raw worker summary",
	})
	if err != nil {
		t.Fatal(err)
	}
	if fs.lastContextCarry.Summary != "short handoff summary" {
		t.Fatalf("Summary = %q, want short handoff summary", fs.lastContextCarry.Summary)
	}
}
```

Note: Add `lastContextCarry store.ContextCarry` to `fakeStore` and update `UpsertContextCarry` to record it.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestrator -run TestFinishRunPersistsContextCarry -v`
Expected: FAIL because finish flow does not summarize or persist context carry.

- [ ] **Step 3: Write minimal finish-run implementation**

`internal/orchestrator/summarizer.go`:

```go
package orchestrator

import (
	"context"
	"fmt"

	"github.com/ayan-de/agent-board/internal/llm"
	"github.com/ayan-de/agent-board/internal/store"
)

func (s Service) FinishRun(ctx context.Context, input FinishRunInput) error {
	cc, err := s.llm.SummarizeContext(ctx, llm.SummaryInput{
		TicketID: input.TicketID,
		Outcome:  input.Outcome,
		Summary:  input.Summary,
	})
	if err != nil {
		return fmt.Errorf("orchestrator.finishRun: %w", err)
	}

	if err := s.store.UpsertContextCarry(ctx, store.ContextCarry{
		TicketID: input.TicketID,
		Summary:  cc,
	}); err != nil {
		return err
	}

	if err := s.store.EndSession(ctx, input.SessionID, input.Outcome); err != nil {
		return err
	}

	_, _ = s.store.CreateEvent(ctx, store.Event{
		TicketID:  input.TicketID,
		SessionID: input.SessionID,
		Kind:      "session." + input.Outcome,
		Payload:   input.Summary,
	})

	return s.ApplyRunOutcome(ctx, ApplyRunOutcomeInput{
		TicketID: input.TicketID,
		Outcome:  input.Outcome,
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/orchestrator -run TestFinishRunPersistsContextCarry -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/summarizer.go internal/orchestrator/service_test.go
git commit -m "feat: persist context carry from run summaries"
```

### Task 12: Add MCP Context Carry Adapter

**Files:**
- Modify: `internal/mcp/contextcarry.go`
- Create: `internal/mcp/contextcarry_test.go`

- [ ] **Step 1: Write the failing MCP adapter test**

```go
package mcp_test

import (
	"strings"
	"testing"

	"github.com/ayan-de/agent-board/internal/mcp"
)

func TestContextCarryAdapterBuildsHandoffPayload(t *testing.T) {
	adapter := mcp.ContextCarryAdapter{}

	payload := adapter.Build(mcp.HandoffInput{
		TicketID:     "AGE-01",
		Title:        "Add orchestration",
		ContextCarry: "prior summary",
	})

	if !strings.Contains(payload, "prior summary") {
		t.Fatalf("payload %q does not include prior summary", payload)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mcp -run TestContextCarryAdapterBuildsHandoffPayload -v`
Expected: FAIL because the adapter is still a placeholder.

- [ ] **Step 3: Write minimal MCP adapter implementation**

`internal/mcp/contextcarry.go`:

```go
package mcp

import "fmt"

type HandoffInput struct {
	TicketID     string
	Title        string
	Description  string
	ContextCarry string
}

type ContextCarryAdapter struct{}

func (ContextCarryAdapter) Build(input HandoffInput) string {
	return fmt.Sprintf(
		"ticket=%s\ntitle=%s\ncontext_carry=%s\n",
		input.TicketID,
		input.Title,
		input.ContextCarry,
	)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mcp -run TestContextCarryAdapterBuildsHandoffPayload -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/contextcarry.go internal/mcp/contextcarry_test.go
git commit -m "feat: add context carry handoff adapter"
```

### Task 13: Wire TUI Status Change To Proposal And Approval Flow

**Files:**
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/app_commands.go`
- Modify: `internal/tui/kanban.go`
- Modify: `internal/tui/ticketview.go`
- Test: `internal/tui/app_test.go`
- Test: `internal/tui/kanban_test.go`
- Test: `internal/tui/ticketview_test.go`

Note: The TUI wraps the orchestrator calls in `tea.Cmd` closures so blocking operations run off the Bubble Tea event loop. The `ExecRunner` blocks on subprocess completion; wrapping it in a `tea.Cmd` ensures the TUI stays responsive.

- [ ] **Step 1: Write the failing TUI workflow test**

```go
func TestMoveToInProgressCreatesProposalRequest(t *testing.T) {
	fo := &fakeOrchestrator{}
	app := NewApp(AppDeps{
		Orchestrator: fo,
	})

	app.board.selectedTicket = store.Ticket{
		ID:     "AGE-01",
		Status: "backlog",
		Agent:  "opencode",
	}

	app.moveTicketToStatus("in_progress")

	if fo.lastCreateProposalTicketID != "AGE-01" {
		t.Fatalf("ticketID = %q, want AGE-01", fo.lastCreateProposalTicketID)
	}
}
```

Note: `AppDeps` is a dependency injection struct added to `AppModel`. `fakeOrchestrator` implements the TUI's `Orchestrator` interface. `moveTicketToStatus` is extracted from the existing status-cycling logic.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui -run TestMoveToInProgressCreatesProposalRequest -v`
Expected: FAIL because the TUI is not wired to orchestrator proposal flow.

- [ ] **Step 3: Write minimal TUI orchestration wiring**

Add an `Orchestrator` interface to the TUI package in `internal/tui/app.go`:

```go
type Orchestrator interface {
	CreateProposal(ctx context.Context, input orchestrator.CreateProposalInput) (store.Proposal, error)
	ApproveProposal(ctx context.Context, proposalID string) error
	StartApprovedRun(ctx context.Context, proposalID string) (store.Session, error)
	FinishRun(ctx context.Context, input orchestrator.FinishRunInput) error
}
```

Add `AppDeps` struct:

```go
type AppDeps struct {
	Orchestrator Orchestrator
}
```

In the status-change handler (wherever status cycling currently happens), when the new status is `in_progress` and the ticket has an agent:

```go
if newStatus == "in_progress" && ticket.Agent != "" {
	proposal, err := m.orchestrator.CreateProposal(ctx, orchestrator.CreateProposalInput{
		TicketID: ticket.ID,
	})
	if err != nil {
		return m, errMsg{err}
	}
	m.pendingProposalID = proposal.ID
	return m, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui -run TestMoveToInProgressCreatesProposalRequest -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/app_commands.go internal/tui/kanban.go internal/tui/ticketview.go internal/tui/app_test.go internal/tui/kanban_test.go internal/tui/ticketview_test.go
git commit -m "feat: trigger proposals from in-progress status changes"
```

### Task 14: Add Approval UI And Start Run Workflow

**Files:**
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/dashboard.go`
- Test: `internal/tui/app_test.go`

- [ ] **Step 1: Write the failing approval UI test**

```go
func TestApprovePendingProposalStartsRun(t *testing.T) {
	fo := &fakeOrchestrator{
		proposal: store.Proposal{ID: "PRO-01"},
	}
	app := NewApp(AppDeps{
		Orchestrator: fo,
	})
	app.pendingProposalID = "PRO-01"

	cmd := app.approvePendingProposal()

	msg := cmd()
	if msg == nil {
		t.Fatal("expected command message")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui -run TestApprovePendingProposalStartsRun -v`
Expected: FAIL because approval/start UI flow does not exist.

- [ ] **Step 3: Write minimal approval/start implementation**

```go
type proposalApprovedMsg struct {
	id store.Proposal
}

type runStartedMsg struct {
	session store.Session
}

func (m *AppModel) approvePendingProposal() tea.Cmd {
	proposalID := m.pendingProposalID
	return func() tea.Msg {
		if err := m.orchestrator.ApproveProposal(context.Background(), proposalID); err != nil {
			return errMsg{err}
		}
		session, err := m.orchestrator.StartApprovedRun(context.Background(), proposalID)
		if err != nil {
			return errMsg{err}
		}

		handle, err := m.runner.Start(context.Background(), orchestrator.RunRequest{
			TicketID:  session.TicketID,
			SessionID: session.ID,
			Agent:     session.Agent,
			Prompt:    "", // filled from proposal
		})
		if err != nil {
			return errMsg{err}
		}

		err = m.orchestrator.FinishRun(context.Background(), orchestrator.FinishRunInput{
			TicketID:  session.TicketID,
			SessionID: session.ID,
			Outcome:   handle.Outcome,
			Summary:   handle.Summary,
		})
		if err != nil {
			return errMsg{err}
		}

		return runStartedMsg{session: session}
	}
}
```

Note: The `runner.Start` and `orchestrator.FinishRun` calls happen inside the `tea.Cmd` closure, which runs on a background goroutine. This prevents the synchronous `ExecRunner` from blocking the TUI.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui -run TestApprovePendingProposalStartsRun -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/dashboard.go internal/tui/app_test.go
git commit -m "feat: approve and start pending proposals"
```

### Task 15: Add Event Recording And Review Retry Tests

**Files:**
- Modify: `internal/orchestrator/service_test.go`

- [ ] **Step 1: Write the failing event/retry test**

```go
func TestRetryFromReviewUsesStoredContextCarry(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{
			ID:     "AGE-01",
			Status: "review",
			Agent:  "opencode",
		},
		contextCarry: store.ContextCarry{
			TicketID: "AGE-01",
			Summary:  "resume from here",
		},
	}
	fllm := &fakeLLMClient{proposal: llm.ProposalDraft{Prompt: "resume from here"}}
	svc := orchestrator.NewService(fs, fllm, nil)

	_, err := svc.CreateProposal(context.Background(), orchestrator.CreateProposalInput{
		TicketID: "AGE-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if fllm.lastProposal.ContextCarry != "resume from here" {
		t.Fatalf("ContextCarry = %q, want resume from here", fllm.lastProposal.ContextCarry)
	}
}
```

Note: This test reuses the `CreateProposal` implementation from Task 6. It verifies that creating a proposal for a ticket in `review` with stored context carry passes that context to the LLM. No additional implementation is needed -- the existing `CreateProposal` already reads `GetContextCarry` and passes it to the LLM.

- [ ] **Step 2: Run test to verify it passes**

Run: `go test ./internal/orchestrator -run TestRetryFromReviewUsesStoredContextCarry -v`
Expected: PASS (the existing CreateProposal implementation already handles this case).

- [ ] **Step 3: Commit**

```bash
git add internal/orchestrator/service_test.go
git commit -m "test: verify review retry uses stored context carry"
```

### Task 16: Full Package Verification

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: Update project memory**

Add to the "Implemented" section in `AGENTS.md`:

```md
- initial orchestrator proposal and approval workflow
- subprocess runner for one worker agent
- context-carry persistence and review retry flow
- LangChain Go integration for coordinator and summarizer models
- approval-gated execution triggered by moving tickets to in_progress
- event recording for orchestration lifecycle
```

- [ ] **Step 2: Run focused package tests**

Run: `go test ./internal/llm ./internal/config ./internal/store ./internal/orchestrator ./internal/mcp ./internal/tui`
Expected: PASS

- [ ] **Step 3: Run full test suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 4: Run vet**

Run: `go vet ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add AGENTS.md
git commit -m "docs: update project memory for ai orchestration slice"
```
