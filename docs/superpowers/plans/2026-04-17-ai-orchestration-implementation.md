# AI Orchestration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first end-to-end AI orchestration slice so moving an assigned ticket to `in_progress` creates an approval-gated run proposal, starts one worker agent through the orchestrator after approval, persists session/context-carry state, and moves the ticket to `review` on successful completion.

**Architecture:** AgentBoard owns orchestration, approvals, board transitions, and persistence in code. A cheap coordinator model prepares proposals and a cheap summarizer model compacts context carry, while an expensive worker CLI performs repo work through a subprocess runner. The worker reports structured outcomes; the orchestrator maps those outcomes to session and ticket transitions. All coordinator and summarizer model calls go through LangChain Go, isolated behind an `internal/llm` package.

**Tech Stack:** Go, Bubble Tea, SQLite via `modernc.org/sqlite`, LangChain Go for coordinator and summarizer model access, existing config/store/tui packages, new orchestrator and llm packages, subprocess execution via `os/exec`

---

## File Structure

Expected files to modify or create in this first slice:

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

- Use LangChain Go as the only coordinator/summarizer model integration layer in this slice.
- Still keep LangChain isolated behind `internal/llm` so the orchestrator depends on AgentBoard-owned interfaces rather than LangChain symbols spread across runtime code.
- Keep `internal/orchestrator` as the runtime owner. TUI and future API should call it, not reimplement its rules.
- Keep `store` persistence-focused. Transition rules belong in orchestrator services.

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
func TestFactoryReturnsLangChainClient(t *testing.T) {
	cfg := config.LLMConfig{
		CoordinatorProvider: "openai",
		CoordinatorModel:    "gpt-5.4-mini",
		CoordinatorAPIKey:   "test-key",
	}

	client, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected client")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/llm -run TestFactoryReturnsLangChainClient -v`
Expected: FAIL because the package and dependency do not exist yet.

- [ ] **Step 3: Write minimal LangChain-backed llm package**

```go
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

```go
type LangChainClient struct {
	coordinator llms.Model
	summarizer  llms.Model
}
```

```go
func NewFromConfig(cfg config.LLMConfig) (Client, error) {
	return &LangChainClient{}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/llm -run TestFactoryReturnsLangChainClient -v`
Expected: PASS

- [ ] **Step 5: Commit**

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

- [ ] **Step 3: Write minimal configuration implementation**

```go
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

```go
LLM: LLMConfig{
	RequireApproval: true,
},
```

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

- [ ] **Step 1: Write the failing store tests**

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store -run 'TestStore(CreateAndApproveProposal|PersistsContextCarry)' -v`
Expected: FAIL because proposal and context-carry storage do not exist yet.

- [ ] **Step 3: Write minimal migrations and store APIs**

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
```

```go
type Proposal struct {
	ID        string
	TicketID  string
	Agent     string
	Status    string
	Prompt    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Store) CreateProposal(ctx context.Context, p Proposal) (Proposal, error) { /* ... */ }
func (s *Store) GetProposal(ctx context.Context, id string) (Proposal, error) { /* ... */ }
func (s *Store) UpdateProposalStatus(ctx context.Context, id, status string) error { /* ... */ }
```

```go
type ContextCarry struct {
	TicketID   string
	Summary    string
	UpdatedAt  time.Time
}

func (s *Store) UpsertContextCarry(ctx context.Context, cc ContextCarry) error { /* ... */ }
func (s *Store) GetContextCarry(ctx context.Context, ticketID string) (ContextCarry, error) { /* ... */ }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/store -run 'TestStore(CreateAndApproveProposal|PersistsContextCarry)' -v`
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

- [ ] **Step 1: Write the failing orchestrator type test**

```go
func TestServiceCreateProposalRequiresAssignedAgent(t *testing.T) {
	svc := Service{}
	_, err := svc.CreateProposal(context.Background(), CreateProposalInput{
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

```go
type LLMClient interface {
	GenerateProposal(ctx context.Context, input llm.ProposalPrompt) (llm.ProposalDraft, error)
	SummarizeContext(ctx context.Context, input llm.SummaryInput) (string, error)
}

type Runner interface {
	Start(ctx context.Context, req RunRequest) (RunHandle, error)
}

type Store interface {
	GetTicket(ctx context.Context, id string) (store.Ticket, error)
	CreateProposal(ctx context.Context, p store.Proposal) (store.Proposal, error)
	GetContextCarry(ctx context.Context, ticketID string) (store.ContextCarry, error)
}

type Service struct {
	store      Store
	llm        LLMClient
	runner     Runner
}
```

```go
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
- Create: `internal/llm/langchain_test.go`
- Modify: `internal/llm/langchain.go`

- [ ] **Step 1: Write the failing LangChain prompting test**

```go
func TestGenerateProposalBuildsPromptFromTicketContext(t *testing.T) {
	client := LangChainClient{
		coordinator: fakeModel{response: "worker prompt"},
		summarizer:  fakeModel{response: "summary"},
	}

	got, err := client.GenerateProposal(context.Background(), ProposalPrompt{
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

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/llm -run TestGenerateProposalBuildsPromptFromTicketContext -v`
Expected: FAIL because LangChain prompt execution is not implemented.

- [ ] **Step 3: Write minimal LangChain prompt implementation**

```go
func (c LangChainClient) GenerateProposal(ctx context.Context, in ProposalPrompt) (ProposalDraft, error) {
	prompt := fmt.Sprintf(
		"Ticket ID: %s\nTitle: %s\nDescription: %s\nAssigned agent: %s\nContext carry: %s\n\nReturn only the worker prompt.",
		in.TicketID,
		in.Title,
		in.Description,
		in.AssignedAgent,
		in.ContextCarry,
	)
	text, err := llms.Call(ctx, c.coordinator, prompt)
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
	text, err := llms.Call(ctx, c.summarizer, prompt)
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

```go
func TestCreateProposalUsesTicketAndContextCarry(t *testing.T) {
	store := &fakeStore{
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
	llm := &fakeLLMClient{
		proposal: ProposalDraft{
			Prompt: "work with prior run summary",
		},
	}
	svc := Service{store: store, llm: llm}

	proposal, err := svc.CreateProposal(context.Background(), CreateProposalInput{
		TicketID: "AGE-01",
	})
	if err != nil {
		t.Fatal(err)
	}

	if proposal.TicketID != "AGE-01" {
		t.Fatalf("TicketID = %q, want AGE-01", proposal.TicketID)
	}
	if llm.lastProposal.ContextCarry != "prior run summary" {
		t.Fatalf("ContextCarry = %q, want prior run summary", llm.lastProposal.ContextCarry)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestrator -run TestCreateProposalUsesTicketAndContextCarry -v`
Expected: FAIL because proposal shaping is not implemented.

- [ ] **Step 3: Write minimal coordinator implementation**

```go
type ProposalPrompt struct {
	TicketID      string
	Title         string
	Description   string
	Agent         string
	ContextCarry  string
}

type ProposalDraft struct {
	Prompt string
}

func (s Service) CreateProposal(ctx context.Context, input CreateProposalInput) (store.Proposal, error) {
	ticket, err := s.store.GetTicket(ctx, input.TicketID)
	if err != nil {
		return store.Proposal{}, err
	}
	if ticket.Agent == "" {
		return store.Proposal{}, fmt.Errorf("orchestrator.createProposal: assigned agent is required")
	}
	cc, _ := s.store.GetContextCarry(ctx, input.TicketID)
	draft, err := s.llm.GenerateProposal(ctx, ProposalPrompt{
		TicketID:     ticket.ID,
		Title:        ticket.Title,
		Description:  ticket.Description,
		Agent:        ticket.Agent,
		ContextCarry: cc.Summary,
	})
	if err != nil {
		return store.Proposal{}, fmt.Errorf("orchestrator.createProposal: %w", err)
	}
	return s.store.CreateProposal(ctx, store.Proposal{
		TicketID: ticket.ID,
		Agent:    ticket.Agent,
		Status:   "pending",
		Prompt:   draft.Prompt,
	})
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
- Modify: `internal/orchestrator/service.go`

- [ ] **Step 1: Write the failing approval test**

```go
func TestApproveProposalRejectsStaleTicketState(t *testing.T) {
	store := &fakeStore{
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
	svc := Service{store: store}

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

```go
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
git add internal/orchestrator/approval.go internal/orchestrator/approval_test.go internal/orchestrator/service.go
git commit -m "feat: add proposal approval guards"
```

### Task 8: Implement Board Action Rules And Ticket-Scoped Permissions

**Files:**
- Create: `internal/orchestrator/actions.go`
- Create: `internal/orchestrator/actions_test.go`
- Modify: `internal/store/tickets.go`

- [ ] **Step 1: Write the failing board-action test**

```go
func TestApplyRunOutcomeMovesTicketToReview(t *testing.T) {
	store := &fakeStore{
		ticket: store.Ticket{
			ID:          "AGE-01",
			Status:      "in_progress",
			Agent:       "opencode",
			AgentActive: true,
		},
	}
	svc := Service{store: store}

	err := svc.ApplyRunOutcome(context.Background(), ApplyRunOutcomeInput{
		TicketID: "AGE-01",
		Outcome:  "completed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.lastMoveStatus != "review" {
		t.Fatalf("MoveStatus = %q, want review", store.lastMoveStatus)
	}
	if store.lastAgentActive != false {
		t.Fatalf("AgentActive = %v, want false", store.lastAgentActive)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestrator -run TestApplyRunOutcomeMovesTicketToReview -v`
Expected: FAIL because outcome mapping does not exist.

- [ ] **Step 3: Write minimal board-action implementation**

```go
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
git add internal/orchestrator/actions.go internal/orchestrator/actions_test.go internal/store/tickets.go
git commit -m "feat: add outcome-driven board transitions"
```

### Task 9: Implement Session Start And Duplicate Active Run Protection

**Files:**
- Modify: `internal/orchestrator/service.go`
- Create: `internal/orchestrator/service_test.go`
- Modify: `internal/store/sessions.go`

- [ ] **Step 1: Write the failing active-session test**

```go
func TestStartApprovedRunRejectsExistingActiveSession(t *testing.T) {
	store := &fakeStore{
		activeSession: true,
		proposal: store.Proposal{
			ID:       "PRO-01",
			TicketID: "AGE-01",
			Agent:    "opencode",
			Status:   "approved",
			Prompt:   "do work",
		},
	}
	svc := Service{store: store, runner: &fakeRunner{}}

	_, err := svc.StartApprovedRun(context.Background(), "PRO-01")
	if err == nil {
		t.Fatal("expected duplicate active session error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestrator -run TestStartApprovedRunRejectsExistingActiveSession -v`
Expected: FAIL because start flow is not implemented.

- [ ] **Step 3: Write minimal start-run implementation**

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
	_, err = s.runner.Start(ctx, RunRequest{
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
	return session, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/orchestrator -run TestStartApprovedRunRejectsExistingActiveSession -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/service.go internal/orchestrator/service_test.go internal/store/sessions.go
git commit -m "feat: start approved runs with session guards"
```

### Task 10: Implement Subprocess `exec` Runner And Structured Outcome Parsing

**Files:**
- Create: `internal/orchestrator/exec_runner.go`
- Create: `internal/orchestrator/exec_runner_test.go`

- [ ] **Step 1: Write the failing exec-runner test**

```go
func TestExecRunnerParsesStructuredOutcome(t *testing.T) {
	runner := ExecRunner{
		LookPath: func(name string) (string, error) {
			return "/bin/echo", nil
		},
		Command: func(ctx context.Context, name string, args ...string) cmdRunner {
			return fakeCmdRunner{
				stdout: `{"outcome":"completed","summary":"done"}`,
			}
		},
	}

	handle, err := runner.Start(context.Background(), RunRequest{
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

```go
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

type ExecRunner struct {
	LookPath func(string) (string, error)
	Command  func(context.Context, string, ...string) cmdRunner
}

func (r ExecRunner) Start(ctx context.Context, req RunRequest) (RunHandle, error) {
	path, err := r.LookPath(req.Agent)
	if err != nil {
		return RunHandle{}, fmt.Errorf("execRunner.start: %w", err)
	}
	cmd := r.Command(ctx, path, req.Prompt)
	out, err := cmd.Output()
	if err != nil {
		return RunHandle{}, err
	}
	var result struct {
		Outcome string `json:"outcome"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return RunHandle{Outcome: "interrupted"}, nil
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
- Test: `internal/orchestrator/service_test.go`

- [ ] **Step 1: Write the failing context-carry test**

```go
func TestFinishRunPersistsContextCarry(t *testing.T) {
	store := &fakeStore{
		ticket: store.Ticket{
			ID:     "AGE-01",
			Status: "in_progress",
		},
	}
	llm := &fakeLLMClient{summary: "short handoff summary"}
	svc := Service{store: store, llm: llm}

	err := svc.FinishRun(context.Background(), FinishRunInput{
		TicketID: "AGE-01",
		SessionID: "SES-01",
		Outcome:  "completed",
		Summary:  "raw worker summary",
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.lastContextCarry.Summary != "short handoff summary" {
		t.Fatalf("Summary = %q, want short handoff summary", store.lastContextCarry.Summary)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestrator -run TestFinishRunPersistsContextCarry -v`
Expected: FAIL because finish flow does not summarize or persist context carry.

- [ ] **Step 3: Write minimal finish-run implementation**

```go
type SummaryInput struct {
	TicketID string
	Outcome  string
	Summary  string
}

func (s Service) FinishRun(ctx context.Context, input FinishRunInput) error {
	cc, err := s.llm.SummarizeContext(ctx, SummaryInput{
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
git add internal/orchestrator/summarizer.go internal/orchestrator/service.go internal/orchestrator/service_test.go
git commit -m "feat: persist context carry from run summaries"
```

### Task 12: Add MCP Context Carry Adapter

**Files:**
- Modify: `internal/mcp/contextcarry.go`
- Create: `internal/mcp/contextcarry_test.go`

- [ ] **Step 1: Write the failing MCP adapter test**

```go
func TestContextCarryAdapterBuildsHandoffPayload(t *testing.T) {
	adapter := ContextCarryAdapter{}

	payload := adapter.Build(HandoffInput{
		TicketID:      "AGE-01",
		Title:         "Add orchestration",
		ContextCarry:  "prior summary",
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

```go
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

- [ ] **Step 1: Write the failing TUI workflow test**

```go
func TestMoveToInProgressCreatesProposalRequest(t *testing.T) {
	orchestrator := &fakeOrchestrator{}
	app := NewApp(AppDeps{
		Orchestrator: orchestrator,
	})

	app.board.selectedTicket = store.Ticket{
		ID:     "AGE-01",
		Status: "backlog",
		Agent:  "opencode",
	}

	app.moveTicketToStatus("in_progress")

	if orchestrator.lastCreateProposalTicketID != "AGE-01" {
		t.Fatalf("ticketID = %q, want AGE-01", orchestrator.lastCreateProposalTicketID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui -run TestMoveToInProgressCreatesProposalRequest -v`
Expected: FAIL because the TUI is not wired to orchestrator proposal flow.

- [ ] **Step 3: Write minimal TUI orchestration wiring**

```go
type Orchestrator interface {
	CreateProposal(ctx context.Context, input orchestrator.CreateProposalInput) (store.Proposal, error)
	ApproveProposal(ctx context.Context, proposalID string) error
	StartApprovedRun(ctx context.Context, proposalID string) (store.Session, error)
}
```

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
	orchestrator := &fakeOrchestrator{
		proposal: store.Proposal{ID: "PRO-01"},
	}
	app := NewApp(AppDeps{
		Orchestrator: orchestrator,
	})
	app.pendingProposalID = "PRO-01"

	app.approvePendingProposal()

	if orchestrator.lastApprovedProposalID != "PRO-01" {
		t.Fatalf("approved proposal = %q, want PRO-01", orchestrator.lastApprovedProposalID)
	}
	if orchestrator.lastStartedProposalID != "PRO-01" {
		t.Fatalf("started proposal = %q, want PRO-01", orchestrator.lastStartedProposalID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui -run TestApprovePendingProposalStartsRun -v`
Expected: FAIL because approval/start UI flow does not exist.

- [ ] **Step 3: Write minimal approval/start implementation**

```go
func (m *AppModel) approvePendingProposal() tea.Cmd {
	proposalID := m.pendingProposalID
	return func() tea.Msg {
		if err := m.orchestrator.ApproveProposal(context.Background(), proposalID); err != nil {
			return errMsg{err}
		}
		_, err := m.orchestrator.StartApprovedRun(context.Background(), proposalID)
		if err != nil {
			return errMsg{err}
		}
		return proposalApprovedMsg{id: proposalID}
	}
}
```

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
- Modify: `internal/orchestrator/service.go`
- Modify: `internal/store/events.go`
- Test: `internal/orchestrator/service_test.go`

- [ ] **Step 1: Write the failing event/retry test**

```go
func TestRetryFromReviewUsesStoredContextCarry(t *testing.T) {
	store := &fakeStore{
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
	llm := &fakeLLMClient{proposal: ProposalDraft{Prompt: "resume from here"}}
	svc := Service{store: store, llm: llm}

	_, err := svc.CreateProposal(context.Background(), CreateProposalInput{
		TicketID: "AGE-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if llm.lastProposal.ContextCarry != "resume from here" {
		t.Fatalf("ContextCarry = %q, want resume from here", llm.lastProposal.ContextCarry)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestrator -run TestRetryFromReviewUsesStoredContextCarry -v`
Expected: FAIL until the retry/event path is complete.

- [ ] **Step 3: Write minimal event recording implementation**

```go
func (s Service) recordEvent(ctx context.Context, kind string, ticketID string, sessionID string, payload string) error {
	return s.store.CreateEvent(ctx, store.Event{
		TicketID:  ticketID,
		SessionID: sessionID,
		Kind:      kind,
		Payload:   payload,
	})
}
```

```go
_ = s.recordEvent(ctx, "proposal.created", ticket.ID, "", draft.Prompt)
_ = s.recordEvent(ctx, "session.completed", input.TicketID, input.SessionID, input.Outcome)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/orchestrator -run TestRetryFromReviewUsesStoredContextCarry -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/service.go internal/store/events.go internal/orchestrator/service_test.go
git commit -m "feat: record orchestration events and retry context"
```

### Task 16: Full Package Verification

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: Update project memory**

```md
### Implemented

- initial orchestrator proposal and approval workflow
- subprocess runner for one worker agent
- context-carry persistence and review retry flow
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
