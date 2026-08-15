# Linear Integration Design

## Status

- **Created:** 2026-08-15
- **Owner:** ayan-de
- **Status:** Draft

---

## Goal

Add bidirectional Linear sync to AgentBoard so a team can move work between Linear and the local Kanban without losing the orchestrator's ownership of lifecycle, status, and agent execution. The integration is a sync, not a takeover — AgentBoard remains the source of truth for tickets, sessions, and agent runs; Linear is the source of truth for human project tracking.

The first slice ships:

- a `linear` provider package that exposes an idempotent sync engine
- webhook intake for Linear `Issue.update`, `Issue.create`, `Comment.create`, and `Attachment.create` events
- pull sync (Linear → AgentBoard) driven by a configurable interval plus on-demand `/linear pull` and `/linear sync`
- push sync (AgentBoard → Linear) on every ticket transition, with a debounce window so a flurry of LLM-driven moves does not flood Linear's GraphQL API
- per-ticket linkage so a ticket in AgentBoard can resolve to a Linear `Issue` and vice versa
- TUI surfacing of the linked Linear identifier, assignee, and sync state
- CLI mode (`agentboard linear sync`, `agentboard linear link <ticketID> <issueID>`) for scripting and CI

## Non-Goals

- a Linear OAuth web app (a personal API key is sufficient for the first slice; OAuth for multi-workspace multi-user is a follow-up)
- Linear Cycles, Projects, Initiatives, or Roadmap surfaces (issues only in slice one)
- two-way realtime synchronization at the keystroke level (debounced push + periodic pull is the contract)
- editing Linear Issues' bodies, descriptions, or comments from AgentBoard (read-and-mirror is the contract; mutations are limited to status, priority, assignee, labels)
- mobile clients (the existing API server stub is the future wire; out of scope here)

## Problem Statement

AgentBoard today is a self-hosted Kanban with a TUI and a store. Linear is the place where the rest of the team reads work, schedules releases, and reports status. People end up running two boards: the Linear cycle in the browser and the AgentBoard local view in the terminal. Every state change — moving a ticket to `review`, reassigning to a different agent, marking a ticket `done` — has to be done twice, and they drift.

The fix is a sync layer that respects the orchestrator's invariants. AgentBoard must not start an agent run on a Linear webhook alone — that path goes through the existing proposal-and-approval flow. Conversely, AgentBoard must not move a Linear ticket to `Done` directly; it must translate the local `Status` to the team's configured Linear state and let the webhook flow back if the team needs to.

## Requirements

### Functional

1. A user can add a Linear API key to `~/.agentboard/config.toml` under a `[linear]` section.
2. The user can bind a Linear `team` to the current AgentBoard project. The mapping is one-to-one for slice one.
3. The user can map AgentBoard statuses to Linear workflow states per team (e.g. `backlog → Backlog`, `in_progress → In Progress`, `review → In Review`, `done → Done`).
4. Pull sync creates AgentBoard tickets for Linear issues that have no local mapping, and updates existing mapped tickets when the Linear side changes.
5. Push sync updates the Linear issue's `state`, `priority`, `assignee`, and `labels` (not description or comments) when the local ticket changes.
6. Webhooks update local ticket state from Linear without overwriting fields AgentBoard owns (e.g. `prompt`, `branch`, `agent_active`, `resume_command`).
7. A user can manually link a local ticket to an existing Linear issue via `agentboard linear link <ticketID> <issueID>` or a TUI action.
8. A user can unlink a ticket; unlinked tickets skip push sync.
9. Conflicts where both sides changed the same field within the debounce window resolve to `last-writer-wins` with the loser recorded in `orchestration_events`.
10. The sync engine is idempotent. Running the same sync twice produces the same state.
11. The CI smoke test can run `agentboard linear sync --dry-run` and assert zero mutations.

### Non-Functional

1. The store layer remains persistence-oriented. No GraphQL client lives in `internal/store/`.
2. The orchestrator remains the owner of ticket lifecycle. The linear package calls the orchestrator service for every mutation, not the store directly.
3. The linear package must be testable with a fake GraphQL server and a fake webhook server.
4. The push path must not block the TUI on a network call. Webhook handlers and push retries go through a background worker.
5. The pull path must degrade gracefully when the Linear API is unreachable — the TUI shows a stale-sync indicator and continues.
6. The webhook endpoint must verify HMAC-SHA256 signatures using the secret from config.
7. The webhook endpoint must be idempotent on `(delivery_id, type)` so retried deliveries do not duplicate events.

## Recommended Architecture

The package layout is `internal/linear/` with a strict dependency direction:

```
internal/linear/
  client.go        — GraphQL client wrapper (one HTTP client, retries, rate limits)
  webhook.go       — HTTP server, signature verification, dispatch
  sync.go          — the pull/push engine, conflict resolution, debounce
  mapping.go       — status/state and priority/label translation
  events.go        — translates Linear webhook payloads into store.Event records
  types.go         — local structs (no dependency on Linear SDK shapes leaking)
  *_test.go        — table-driven tests, fakes for the GraphQL server
```

The package depends on `internal/store`, `internal/orchestrator`, and `internal/config`. It does **not** depend on `internal/tui`, `internal/pty`, or `internal/llm`. The TUI gets Linear state through the orchestrator, same as any other integrated data source.

### Why the Linear package goes through the orchestrator

The orchestrator already owns ticket transitions, agent approval, and session lifecycle. Push sync is a state mutation, so it must flow through the same code path that a TUI keypress or future API call uses. Doing it any other way creates a second writer that bypasses rules like "moving to `in_progress` requires an approved proposal" and "only one active session per ticket". The cost is a few extra method signatures on `orchestrator.Service`; the benefit is one rule book.

### Why status mapping is configuration, not code

Linear teams configure their own workflow states. A team might call `in_progress` "Triage" or have a separate `Blocked` column. Hardcoding `backlog → Backlog` will break the first time a team edits their workflow. The mapping lives in `~/.agentboard/projects/<name>/config.toml` under `[linear.status_map]` and is loaded with the rest of the config at startup.

### Why pull is interval-based, not webhook-only

Webhooks are best-effort. Linear documents delivery at-least-once with retries, but a missed delivery (expired endpoint, network blip, AgentBoard offline) leaves the local view stale forever. Interval-based pull is the floor. Webhooks are the ceiling. The first slice implements pull at startup and every 5 minutes thereafter, plus a `/linear pull` slash command.

### Why debounce push

A single agent run can produce several `in_progress → review → done`-style transitions in a few seconds. Pushing each one to Linear as a separate GraphQL mutation is wasteful and noisy in the team's Linear activity feed. Push coalesces changes per ticket for a 5-second window before flushing.

## Data Model

### Linear Issue model (local)

```go
package linear

type Issue struct {
    ID          string    // Linear UUID, e.g. "8a3c7e62-..."
    Identifier  string    // Human ID, e.g. "AGT-42"
    Title       string
    Description string
    State       string    // Linear workflow state name
    Priority    int       // 0=No priority, 1=Urgent, 2=High, 3=Medium, 4=Low
    AssigneeID  string
    Labels      []string
    UpdatedAt   time.Time
    CreatedAt   time.Time
    URL         string
    TeamID      string
}
```

### Store additions

Add two columns to `tickets`:

```sql
ALTER TABLE tickets ADD COLUMN linear_id TEXT;
ALTER TABLE tickets ADD COLUMN linear_synced_at DATETIME;
```

Add an index on `linear_id` for reverse lookups:

```sql
CREATE INDEX IF NOT EXISTS idx_tickets_linear_id ON tickets(linear_id);
```

Add a new table `linear_links` to record the history of mappings — useful when a user unlinks and relinks, and for the conflict log:

```sql
CREATE TABLE IF NOT EXISTS linear_links (
    id              TEXT PRIMARY KEY,
    ticket_id       TEXT NOT NULL,
    linear_id       TEXT NOT NULL,
    linear_identifier TEXT NOT NULL,
    direction       TEXT NOT NULL,  -- "pulled", "pushed", "linked", "unlinked"
    created_at      DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_linear_links_ticket ON linear_links(ticket_id);
CREATE INDEX IF NOT EXISTS idx_linear_links_linear ON linear_links(linear_id);
```

Add a new table `linear_pending_push` for the debounce queue:

```sql
CREATE TABLE IF NOT EXISTS linear_pending_push (
    ticket_id       TEXT PRIMARY KEY,
    fields          TEXT NOT NULL,  -- JSON map of fields to push
    enqueued_at     DATETIME NOT NULL,
    attempt         INTEGER NOT NULL DEFAULT 0
);
```

### Ticket struct updates

```go
type Ticket struct {
    ID             string
    Title          string
    Prompt         string
    Status         string
    Priority       string
    Agent          string
    Branch         string
    Tags           []string
    DependsOn      []string
    AgentActive    bool
    ResumeCommand  string
    LinearID       string    // NEW
    LinearSyncedAt time.Time // NEW
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

### Store methods

```go
func (s *Store) SetLinearID(ctx context.Context, ticketID, linearID, identifier string) error
func (s *Store) ClearLinearID(ctx context.Context, ticketID string) error
func (s *Store) GetTicketByLinearID(ctx context.Context, linearID string) (Ticket, error)
func (s *Store) ListTicketsWithLinearID(ctx context.Context) ([]Ticket, error)
func (s *Store) SetLinearSyncedAt(ctx context.Context, ticketID string) error
func (s *Store) EnqueuePush(ctx context.Context, ticketID string, fields map[string]any) error
func (s *Store) DequeuePush(ctx context.Context, ticketID string) (PushJob, error)
func (s *Store) ListPendingPushes(ctx context.Context) ([]PushJob, error)
func (s *Store) RecordLinearLink(ctx context.Context, link LinearLink) error
```

## Sync Engine

### Pull (Linear → AgentBoard)

```
PullSync(ctx context.Context, teamID string) (PullResult, error)
```

Steps:

1. Resolve the Linear team from config. If unset, return `ErrNoTeam`.
2. Fetch issues updated after `last_pull_at` (default: now on first run, persisted in `meta`).
3. For each issue:
   - If `linear_id` is already mapped to a local ticket, diff and update. Skip fields AgentBoard owns.
   - If unmapped, create a new AgentBoard ticket with `Status = mapped(issue.State)`, `Priority = mapped(issue.Priority)`, `Agent = ""`, `Prompt = ""`, then record the link.
4. Update `meta.last_linear_pull_at`.
5. Return a `PullResult{Created, Updated, Skipped, Errors}`.

The diff for an existing mapped ticket is restricted to fields Linear owns:

| Field             | Source of truth |
|-------------------|-----------------|
| `Title`           | Linear (Linear is the human-facing source) |
| `Priority`        | Linear (mutually editable; LWW) |
| `Tags` / labels   | Linear (Linear owns labels) |
| `Status`          | Both (translates via `status_map`) |
| `Prompt`          | AgentBoard |
| `Branch`          | AgentBoard |
| `Agent`           | AgentBoard |
| `AgentActive`     | AgentBoard |
| `ResumeCommand`   | AgentBoard |

### Push (AgentBoard → Linear)

Push is fan-out from the orchestrator's `OnTicketChanged` hook. The hook receives a `store.Ticket` and a `ChangeSet` (the fields that mutated). The hook:

1. If `ticket.LinearID == ""`, skip.
2. Build a `PushJob{TicketID, Fields: ChangeSet}` and `store.EnqueuePush(...)`.
3. Return. The worker handles the rest.

The worker loop runs every 2 seconds:

```
for {
    select {
    case <-ctx.Done(): return
    case <-ticker.C:
        jobs := store.ListPendingPushes(ctx)
        for _, job := range jobs {
            // Coalesce: if job is < 5s old, defer.
            if time.Since(job.EnqueuedAt) < 5*time.Second {
                continue
            }
            if err := pushOne(ctx, job); err != nil {
                job.Attempt++
                if job.Attempt >= 5 {
                    store.DeleteJob(ctx, job.TicketID)
                    notify(ticketID, "linear push failed permanently")
                }
                continue
            }
            store.DeleteJob(ctx, job.TicketID)
            store.SetLinearSyncedAt(ctx, ticketID)
        }
    }
}
```

`pushOne` builds a GraphQL `issueUpdate` mutation:

```graphql
mutation UpdateIssue($id: String!, $input: IssueUpdateInput!) {
    issueUpdate(id: $id, input: $input) {
        success
        issue { id updatedAt }
    }
}
```

The `input` only includes fields Linear owns. To map a `Status` change to a Linear state, the engine looks up `status_map[localStatus]` and resolves that name to a Linear state ID via a cached `states` table (refreshed on each pull).

### Webhook intake

The webhook server listens on `127.0.0.1:<linear.webhook_port>` (default `:8081`). The endpoint is `/linear/webhook`. Handlers:

1. Read raw body, verify `Linear-Signature` HMAC-SHA256 against `cfg.Linear.WebhookSecret`. Reject with 401 on mismatch.
2. Look up `(delivery_id, event_type)` in a small in-memory dedup table. Reject with 200 if duplicate.
3. Parse the JSON envelope. The `action` field is `create`, `update`, or `remove`.
4. Dispatch to `linear.events.go`:
   - `Issue.create` → schedule a pull for that issue
   - `Issue.update` → apply if we have a mapping, ignore `agent_active`/`resume_command`/`prompt` field names
   - `Issue.remove` → if mapped, mark the local ticket's `linear_id = ""` and append to `linear_links` as `unlinked`
   - `Comment.create` → log to `orchestration_events` for the audit trail; not surfaced in the TUI in slice one
   - `Attachment.create` → ignore (out of scope)
5. Acknowledge with 200. Webhook processing itself returns no business error to Linear — failures are logged and the local audit row records what happened.

The webhook server is a thin `http.Handler` that calls `linear.NewSyncService(...).HandleWebhook(...)`. It does not live inside the orchestrator's HTTP server because the API server stub is not yet implemented. Putting it under `/linear/webhook` on its own port means the path is stable once the API server arrives.

## Orchestrator integration

The orchestrator needs three additions:

1. `OnTicketChanged(ctx, ticket, changeSet)` callback that the store emits after every write. The linear package subscribes to this callback at startup.
2. `Service.LinkToLinear(ctx, ticketID, linearID)` and `Service.UnlinkFromLinear(ctx, ticketID)` for the manual link commands. These go through the same confirmation flow as any other state change.
3. `Service.PullLinear(ctx)` and `Service.SyncLinear(ctx)` so the TUI and CLI can request a sync without reaching into the linear package directly.

The orchestrator must not import the linear package. Instead, the wiring in `cmd/agentboard/run.go` does:

```go
syncSvc := linear.NewService(cfg.Linear, s, llmClient)
orch.SetChangeListener(syncSvc.OnTicketChanged)
syncSvc.SetOrchestrator(orch)
```

This keeps the dependency arrow out of the orchestrator and means the linear package is replaceable with a Jira or GitHub Issues package later without touching the orchestrator.

## TUI

The ticket detail view gains a `Linear` section:

```
┌─ Ticket: AGT-03 ─────────────────────────────────────┐
│ Title:    Implement webhook signature verification   │
│ Status:   in_progress                              │
│ Priority: high                                      │
│ Agent:    claude-code                               │
│ Linear:   AGT-12 (synced 2m ago)                   │
│                                                     │
│ Description:                                        │
│ ...
```

A new palette command `/linear pull` triggers a sync and shows a toast with the result. A new palette command `/linear link` opens a prompt for an issue ID, calls `orch.LinkToLinear(...)`, and re-renders. A new palette command `/linear unlink` does the reverse.

A new keybinding `L` on the ticket detail view opens the linked Linear issue in the user's browser:

```go
open.Run(ticket.LinearURL)
```

## CLI

The existing `internal/cli` dispatcher is a flat registry of top-level commands (`agentboard update`, etc.) with no nested subcommand parsing. The linear integration adds one top-level `linear` command that switches on its first positional argument:

```
agentboard linear sync [--dry-run] [--pull-only|--push-only]
agentboard linear link <ticketID> <issueID>
agentboard linear unlink <ticketID>
agentboard linear serve [--port N]    # webhook listener only, headless mode
```

`--dry-run` is the test-friendly flag. It runs the pull and push pipelines but does not write to the store or call the Linear mutate API. The CI smoke test uses this to assert zero mutations on a populated fixture board.

`--pull-only` and `--push-only` are escape hatches for the rare case the user wants to operate one direction manually.

`linear serve` runs the webhook listener without launching the TUI. This is the right entry point for CI environments and remote dev boxes where the TUI is not running but the team still wants inbound Linear updates. The TUI mode starts the same listener in-process, so users do not need to run `linear serve` separately on their workstation.

## Config

```toml
# ~/.agentboard/config.toml
[linear]
api_key = "lin_api_..."            # required
team_id = "..."                     # Linear team UUID; required for sync
webhook_port = 8081                 # local webhook listener port
webhook_secret = "..."              # HMAC secret for signature verification
sync_interval = "5m"                # pull interval (zero = manual only)
push_debounce = "5s"                # how long to coalesce changes per ticket
status_map = { backlog = "Backlog", in_progress = "In Progress", review = "In Review", done = "Done" }
priority_map = { urgent = 1, high = 2, medium = 3, low = 4 }
```

Env vars override (matching the existing convention):

| Env var                            | Field                       |
|------------------------------------|-----------------------------|
| `AGENTBOARD_LINEAR_API_KEY`        | `api_key`                   |
| `AGENTBOARD_LINEAR_TEAM_ID`        | `team_id`                   |
| `AGENTBOARD_LINEAR_WEBHOOK_SECRET` | `webhook_secret`            |
| `AGENTBOARD_LINEAR_WEBHOOK_PORT`   | `webhook_port`              |
| `AGENTBOARD_LINEAR_SYNC_INTERVAL`  | `sync_interval` (Go duration) |

## Errors and Observability

The linear package defines a sentinel error set:

```go
var (
    ErrNoTeam         = errors.New("linear: no team configured")
    ErrUnauthenticated = errors.New("linear: invalid or missing API key")
    ErrNotFound       = errors.New("linear: issue not found")
    ErrRateLimited    = errors.New("linear: rate limited")
    ErrConflict       = errors.New("linear: conflict, retry")
)
```

The sync service renders these as user-visible toasts in the TUI and as `orchestration_events` rows with `kind = "linear.error"` and `payload = err.Error()`. Every push and pull writes a `kind = "linear.pull"` or `kind = "linear.push"` event with the count of created/updated/skipped entries.

A new `meta` key `linear_last_pull_at` stores the last successful pull. The TUI dashboard reads this and shows a small `↻ synced 3m ago` indicator next to the Linear section. If the last pull failed, the indicator turns red and surfaces the error.

## Files to Modify

| File | Change |
|------|--------|
| `internal/store/migrations.go` | Add `linear_id`, `linear_synced_at` columns; new `linear_links`, `linear_pending_push` tables |
| `internal/store/tickets.go` | Add `LinearID`, `LinearSyncedAt` fields; new getters/setters |
| `internal/store/linear_store.go` | New file: `linear_links`, `linear_pending_push` CRUD |
| `internal/orchestrator/service.go` | Add `OnTicketChanged`, `LinkToLinear`, `UnlinkFromLinear`, `PullLinear`, `SyncLinear` |
| `internal/orchestrator/service.go` | Wire listener pattern so the linear package can subscribe |
| `internal/linear/client.go` | New file: GraphQL client with retries, rate-limit handling |
| `internal/linear/webhook.go` | New file: HTTP server, HMAC verification, dedup |
| `internal/linear/sync.go` | New file: pull/push engine, debounce worker |
| `internal/linear/mapping.go` | New file: status/priority translation |
| `internal/linear/events.go` | New file: webhook payload → store events |
| `internal/linear/types.go` | New file: local structs |
| `internal/config/config.go` | Add `Linear LinearConfig` field |
| `internal/config/defaults.go` | Default values for the linear block |
| `internal/config/config.go` | Add env var overlay |
| `cmd/agentboard/run.go` | Wire linear service into the orchestrator |
| `cmd/agentboard/run.go` | Start the webhook server and sync worker |
| `internal/tui/ticketview.go` | Render Linear section in the detail view |
| `internal/tui/palette.go` | Add `/linear pull`, `/linear link`, `/linear unlink` commands |
| `internal/tui/app.go` | Add `L` keybinding on ticket detail to open Linear URL |
| `internal/cli/cli.go` | Register the `linear` top-level command |
| `cmd/agentboard/linear.go` | New file: `linear sync` / `link` / `unlink` / `serve` handlers, dispatching on the first positional arg |
| `AGENTS.md` | Document the linear package and config block |

## Testing

1. Given a populated Linear fixture, `linear.sync.Pull` creates the expected AgentBoard tickets with status mapped from `status_map`.
2. Given a local ticket whose `Status` changes, `pushOne` enqueues a job and the worker eventually sends an `issueUpdate` mutation with the mapped state.
3. Two pushes inside the debounce window collapse to one mutation when the worker flushes.
4. A webhook delivery with a bad HMAC signature returns 401 and does not write to the store.
5. A duplicate webhook delivery (same `delivery_id`) is acknowledged 200 but does not re-apply the change.
6. A webhook delivery for `Issue.update` that touches `agent_active` does not overwrite the local value.
7. `agentboard linear sync --dry-run` on a populated fixture board returns `Created=0`, `Updated=0`, `Pushed=0` and writes nothing to the store.
8. A failed push (network error) increments `attempt` and is retried until `attempt >= 5`, then it's dropped and an `orchestration_events` row records the failure.
9. The manual link command `agentboard linear link AGT-01 AGT-12` writes a `linear_links` row with `direction = "linked"` and sets `tickets.linear_id`.
10. The TUI renders the Linear section only when `linear_id` is set.

## Rollout

1. Land the store migrations and the orchestrator listener hook first, behind no wires. New code, no new behavior.
2. Land `internal/linear` with a fake GraphQL server and full unit tests. The package compiles and tests pass without a real Linear API key.
3. Land the CLI subcommands and the config block. The user can run `linear sync --dry-run` and see what would change.
4. Land the webhook server and the real GraphQL client. Wire the sync worker into `run.go`.
5. Land the TUI changes.
6. Smoke test with a real Linear workspace on a throwaway team.

## Open Questions

- Should we expose the Linear cycle as an AgentBoard tag, or as a separate field? Slice one ignores cycles. Add a follow-up spec if anyone needs it.
- Should push sync also send comments? The instinct is no — comments are a chat channel and the agent's prompts are not chat. Revisit if a user asks.
- Should we hold `unlinked` tickets in a "ghost" state so they can be re-linked later, or hard-delete the mapping? Slice one soft-keeps the `linear_links` history row and clears `tickets.linear_id`.
- Does Linear's GraphQL play nicely with a 5s debounce from the worker's perspective? The push is a single mutation per ticket, so 5s is fine. If we ever push many tickets at once, the worker must back off on 429s. The client already handles `Retry-After` for that.

## Notes

- The orchestrator's invariant "moving to `in_progress` requires an approved proposal" must hold for webhook-triggered moves too. A webhook that says `state = "In Progress"` is interpreted as a status change, not a request to start an agent. The orchestrator maps the state change and creates a proposal if the agent is assigned, identical to the manual path.
- The webhook endpoint listens on `127.0.0.1` only in slice one. Publishing it to the public internet requires a tunnel (ngrok, Cloudflare Tunnel) and is out of scope. Document the tunnel requirement in the user-facing README.
- The store remains the source of truth for mapping. The linear package caches the `Issue` struct in memory but re-reads `tickets.linear_id` on every sync. This keeps the package stateless across restarts.
- The 5-second debounce is a default. Power users can set `push_debounce = "0s"` to flush every change immediately, at the cost of more GraphQL calls and more Linear activity. Mention this in the config comments.
- The `linear_id` column is TEXT, not structured. Linear UUIDs are opaque. Don't try to parse them.
- The integrations marketplace for Linear is OAuth-only. The personal API key path is fine for slice one but blocks a future "install from Linear" flow. Note this in the open-questions section of the rollout PR.
