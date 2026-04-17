# AI Orchestration Design

## Goal

Add the first real AI integration to AgentBoard by introducing a coordinator-driven orchestration layer that can propose, approve, start, observe, and finish external AI agent runs against Kanban tickets. The first slice should support a single coordinator backend and a single subprocess-based runner while remaining extensible to multiple model providers and multiple execution backends later.

The long-term operating model is a three-tier AI system:

- one cheap coordinator model for planning, proposal generation, and result interpretation
- one cheap summarizer/handoff model for context-carry generation and resume packages
- one expensive worker agent CLI for actual repo work when high-quality coding output is needed

## Scope

This design covers:

- user-provided LLM credentials and coordinator configuration
- approval-gated child-agent execution
- orchestrator-owned session lifecycle
- capability-based ticket and session updates from running child agents
- agent-to-agent context handoff on ticket reassignment
- orchestrator-owned board transitions after worker completion
- a three-tier AI operating model with cheap coordinator/summarizer roles and an expensive worker role
- subprocess execution as the first runner backend
- future compatibility with API/mobile control paths

This design does not cover:

- tmux or PTY execution in the first implementation
- multi-agent parallel orchestration
- embedded terminal rendering
- mobile-specific API design
- advanced decomposition workflows

## Problem Statement

AgentBoard currently detects local agent CLIs and persists tickets and sessions, but there is no live orchestration layer. The user wants AgentBoard to use a configured LLM as a coordinator that can prepare work for child agents such as `claude`, `opencode`, `cursor`, or local models in the future. The coordinator should understand ticket context, propose an execution plan, and then wait for explicit user approval before starting a child agent.

Once a run is approved and started, the child agent should be able to update board state directly, but only through constrained AgentBoard-owned actions. It must not mutate SQLite with raw SQL commands. AgentBoard must remain the single owner of lifecycle, ticket state rules, and persistence.

## Requirements

### Functional Requirements

1. The system must allow a user to configure one coordinator model provider and API key.
2. The system must generate a run proposal for a ticket before any child process starts.
3. The system must require explicit user approval before starting a child agent.
4. The system must create and persist a session when an approved run starts.
5. The system must mark the ticket as active when a session starts and clear it when the session ends.
6. The system must allow a running child agent to update its own ticket state through constrained AgentBoard actions.
7. The system must persist lifecycle and board-action events for later inspection.
8. The system must treat moving a ticket to `in_progress` as the workflow trigger for execution proposal creation when an agent is assigned.
9. A successful or user-stopped run must persist context-carry data for the ticket so a later run can resume with prior context.
10. The system must support reassignment from one child agent to another with orchestrator-owned context handoff.
11. The worker agent must report structured run outcomes, and the orchestrator must perform the resulting board transitions in code.
12. Successful worker completion must move a ticket to `review` by default, not `done`.
13. The orchestrator must reject concurrent active runs for the same ticket.
14. The first implementation must support one subprocess-based runner adapter.
15. The architecture must support future API callers using the same service layer as the TUI.

### Non-Functional Requirements

1. The store layer must remain persistence-oriented and must not become the runtime owner.
2. The TUI must remain a presentation layer that issues intents and renders orchestrator state.
3. Raw SQL must never be injected into child-agent prompts as the mechanism for board updates.
4. External runner details must remain behind narrow interfaces so tmux and PTY adapters can be added later.
5. The design must be testable with fakes for the LLM client and runner.

## Recommended Architecture

The recommended design is capability-based orchestration. AgentBoard owns the data model and exposes a narrow internal action surface that child agents can use after a run is approved and started. The child agent never talks directly to SQLite. Instead, it calls AgentBoard-defined actions such as changing ticket status, toggling `agent_active`, appending notes, or ending a session. Those actions are validated and executed by AgentBoard services.

This keeps the orchestrator as the single owner of runtime behavior, makes the design auditable, and allows the TUI and future API clients to share the same business rules.

Within the current board model, `in_progress` remains the execution-trigger status. Moving a ticket into `in_progress` should be interpreted by AgentBoard as a request to generate an execution proposal for the assigned agent. Approval is still required before any process starts.

There should be no second execution toggle in the first slice. Status transition to `in_progress` is the single execution trigger.

The worker agent should not be responsible for Kanban management. It reports structured outcomes such as `completed`, `blocked`, `failed`, or `interrupted`, and the orchestrator maps those outcomes to ticket/session transitions in code.

## AI Roles

### Coordinator Model

The coordinator model should be inexpensive relative to the worker agent. It is responsible for:

- proposal generation
- prompt shaping
- runner/agent selection support
- interpreting worker outcomes when needed
- deciding what prior context should be included in a run

### Summarizer/Handoff Model

The summarizer model should also be inexpensive and may be the same provider/model family as the coordinator in the first implementation. It is responsible for:

- generating compact context-carry payloads
- summarizing worker output into resumable handoff context
- shrinking large execution history into a reusable ticket memory

### Worker Agent

The worker agent is typically the expensive model/runtime, such as `claude-code`, `opencode`, or another coding-oriented CLI. It is responsible for:

- doing repo work
- reading prepared ticket context
- returning structured run outcomes
- optionally providing notes or summaries for the next handoff

This split keeps expensive tokens focused on software work while low-cost models handle orchestration and memory packaging.

## Core Components

### Coordinator

The coordinator uses the configured LLM provider and model to read ticket context, project context, runner availability, and execution policy. It produces a structured `RunProposal` instead of directly starting processes.

Responsibilities:

- collect run inputs from ticket, config, and local environment
- choose an execution target and shape the child-agent prompt
- return a proposal that the user can approve or reject

The coordinator should depend on an internal `LLMClient` interface rather than a framework dependency such as LangChain in the first slice. A thin local interface is a better fit for the existing Go codebase and keeps provider integration under AgentBoard control.

### Approval Service

The approval service stores pending run proposals and enforces the policy that no child process may start before a user accepts the proposal.

Responsibilities:

- create and list pending proposals
- approve or reject proposals
- prevent duplicate or stale approvals

### Session Service

The session service owns runtime session transitions and persistence. It coordinates the creation of session rows, marks sessions as completed or failed, and ensures the ticket active flag is consistent with runtime state.

Responsibilities:

- create a session at run start
- mark a session ended on success, failure, or cancellation
- guard against duplicate active sessions on the same ticket

### Runner Interface

The runner is the transport adapter that starts and observes external agent processes.

The first backend should be an `exec` runner that launches agent CLIs as ordinary child processes with controlled arguments and environment. Future `tmux` and `pty` runners should implement the same interface.

Responsibilities:

- start a child process
- stream or collect output
- stop a running process
- report backend capabilities

### Board Action Service

The board action service is the only application-level path by which a running child agent may change ticket or session state.

Responsibilities:

- validate requested actions
- apply allowed ticket changes
- append structured events
- reject invalid or disallowed updates

Example action surface:

- `ticket.get`
- `ticket.set_status`
- `ticket.set_agent_active`
- `ticket.add_note`
- `ticket.reassign_agent`
- `ticket.write_context_carry`
- `session.finish`

### Event Log

The event log persists append-only orchestration events such as proposal creation, approval, runner start, status change requests, action rejections, and session end states.

The first slice can keep event storage simple, but the system should record enough information for TUI inspection and future API consumers.

### Context Handoff Service

The context handoff service prepares normalized ticket continuity data when work moves from one child agent to another. This service is orchestrator-owned and may use MCP adapters such as `ContextCarry` to deliver the handoff package to the next run.

Responsibilities:

- gather prior ticket, session, and event context
- produce a normalized handoff payload independent of any one runner format
- pass that payload to the next child run through an MCP adapter or equivalent injected context mechanism

The source of truth remains AgentBoard persistence. MCP is the transport and enrichment layer for handoff, not the canonical store of record.

## Data Flow

1. The user assigns a ticket to an external agent and requests execution from the TUI.
2. The user moves the ticket to `in_progress`, and the TUI calls the orchestrator to create a proposal for that ticket.
3. The coordinator reads ticket state, configuration, detected agents, and local context, then returns a `RunProposal`.
4. The approval service persists the proposal as pending.
5. The user reviews the proposal in the TUI and approves it.
6. The session service creates a runtime session.
7. The board action service marks the ticket as active.
8. The runner starts the selected external CLI process with injected context and the constrained AgentBoard action contract.
9. During execution, the child agent calls AgentBoard actions to report progress or update its ticket state.
10. The worker returns a structured outcome such as `completed`, `blocked`, `failed`, or `interrupted`.
11. The orchestrator validates the outcome, persists any events or summaries, and performs the board transition in code.
12. On successful completion, the orchestrator persists context-carry data for the ticket, clears `agent_active`, and moves the ticket to `review`.
13. If the user wants more work after review, the ticket can be moved back to `in_progress`, a new proposal is generated, and the next run starts with the stored context-carry data after approval.
14. On failure or cancellation, the session service ends the session and clears `agent_active`, while preserving whatever resumable context is available.

## End-To-End User Flow

1. The user installs AgentBoard.
2. On startup in a project root, AgentBoard loads config, derives the project name, opens or creates the project database, loads theme and keybindings, and detects available local agent CLIs.
3. The user configures the coordinator model, base URL if needed, and API key or local endpoint.
4. The user opens AgentBoard inside the target repository.
5. The user creates a ticket in `backlog`.
6. The user assigns a worker agent to the ticket.
7. The user moves the ticket to `in_progress`.
8. AgentBoard interprets that move as an execution request and gathers ticket title, description, assignment, dependencies, and any stored context-carry data.
9. The coordinator generates a run proposal.
10. The user approves the proposal.
11. The orchestrator creates a session, sets `agent_active=1`, and starts the worker runner.
12. The worker performs the repo task and reports a structured outcome.
13. The summarizer/handoff stage produces or updates the ticket context-carry payload.
14. The orchestrator performs session and board updates in code.
15. On success, the ticket moves to `review`.
16. If the user wants another iteration, they move the ticket back to `in_progress`, approve a new run, and the next worker starts with the stored context-carry data.
17. The user moves the ticket to `done` only after human review and acceptance.

### Reassignment Handoff Flow

1. The user changes the assigned agent on a ticket that already has prior session history.
2. The orchestrator records the reassignment request.
3. The context handoff service gathers ticket data, prior session metadata, persisted events, and selected execution context.
4. The service builds a normalized handoff payload for the next agent.
5. The orchestrator passes that payload through the `ContextCarry` adapter when preparing the next run.
6. The coordinator generates a new run proposal using both current ticket state and carried context.
7. The user approves the new run.
8. The new runner starts with the prior context available from the beginning of execution.

## Package Responsibilities

### `internal/orchestrator`

This package should become the runtime owner.

Recommended sub-responsibilities:

- coordinator logic
- proposal and approval management
- session lifecycle orchestration
- runner interfaces and adapters
- action dispatch and validation
- runtime event publishing

### `internal/store`

This package should remain persistence-only. It can gain CRUD methods needed by orchestration, but it should not become the place where process lifecycle or approval policy is implemented.

### `internal/tui`

The TUI should trigger intents such as propose run, approve run, reject run, stop run, and inspect events. It should not start processes directly and should not embed orchestration rules.

### `internal/api`

This package should later expose the same orchestrator services for remote control, including future mobile clients. It must call into the same orchestration layer as the TUI rather than implementing a second execution path.

### `internal/mcp`

This package should contain adapters such as `ContextCarry` that the orchestrator can use to transport handoff context between runs. It should not own orchestration policy or lifecycle decisions.

### `internal/config`

This package should hold:

- coordinator provider
- summarizer provider if separated later
- model
- base URL
- API key or API key reference
- runner defaults
- approval policy defaults if needed later

### `internal/decomposition`

This package may later help shape prompts or task decomposition, but it should not own execution, session lifecycle, or board mutations.

## Configuration Strategy

The first slice should reuse the existing `LLMConfig` shape where possible, expanding it only as needed. The user can provide a model API key through configuration, and AgentBoard uses that key for the coordinator only. Child agent CLIs remain local tools discovered on the host.

For the first implementation, storing the API key in config is acceptable if it follows the existing config storage conventions. If secret management becomes broader later, configuration can evolve to support environment references or external secret stores without changing orchestrator contracts.

## Why Not Direct SQL From Child Agents

Direct SQL updates from child agents are not acceptable as the primary control mechanism because they:

- bypass validation rules
- make auditability weak
- couple prompts to storage schema
- create unsafe mutation paths
- force future API clients to duplicate state logic

Using AgentBoard actions instead keeps the storage schema private and allows the runtime contract to evolve independently from SQLite details.

## Status-Driven Execution

The current board statuses remain `backlog`, `in_progress`, `review`, and `done`. In this design, moving a ticket into `in_progress` is the workflow signal that execution should be proposed for the currently assigned agent.

This does not remove the approval gate. The intended sequence is:

1. The user assigns an agent to a ticket.
2. The user moves the ticket to `in_progress`.
3. AgentBoard generates a run proposal from the ticket title, description, dependencies, and any stored context-carry payload.
4. The user approves the run.
5. The orchestrator starts the session.

There is no additional start toggle in the first implementation. Entering `in_progress` is the single execution request action.

When a run completes successfully, the orchestrator should store context-carry data for the ticket and move it to `review`. If the work is not good enough, the user can move the ticket back to `in_progress`, approve another run, and the next session resumes with the carried context.

The worker does not move the ticket to `done`. `done` remains a user acceptance state.

## Agent Reassignment And Context Carry

When a ticket is reassigned from one child agent to another, AgentBoard should preserve continuity through an orchestrator-owned handoff flow. For example, if a ticket was previously worked by `claude-code` and is then reassigned to `opencode`, the next run should receive prior context through a normalized handoff package rather than through raw prompt copy-paste or direct agent-to-agent coordination.

The handoff package should be assembled from AgentBoard-owned sources such as:

- current ticket fields
- prior session metadata
- lifecycle and board-action events
- persisted notes or summaries
- stored context-carry payload from the last completed or interrupted run
- selected output excerpts where available

The orchestrator may then pass that package through an MCP adapter such as `ContextCarry` when starting the next run. This keeps context transfer deterministic, auditable, and independent from any single runner's native prompt format.

## Error Handling

### Proposal Creation Failure

No session is created. The error is returned to the caller and optionally logged as a proposal failure event.

### Approval Rejection

The proposal remains non-runnable and no process is started.

### Duplicate Active Run

The orchestrator rejects the start attempt before spawning a runner.

### Runner Startup Failure

If session creation has already occurred, the session is transitioned to a failed startup state and the ticket active flag is cleared.

### Invalid Child Action

The board action service rejects the action, records an event, and keeps the session running unless a stricter policy is later introduced.

### Child Process Crash or Exit

The session is ended with an appropriate terminal status and the ticket active flag is cleared.

### Review Retry

If a ticket in `review` is moved back to `in_progress`, the orchestrator creates a new approval-gated proposal using the stored context-carry data from the prior run so the next session can continue rather than restarting from nothing.

### Missing Assignment

If a ticket is moved to `in_progress` without an assigned worker agent, the orchestrator must reject execution request creation and return the ticket to a non-running state with a clear user-visible error.

### Missing Runner Binary

If the ticket is assigned to a worker agent whose CLI is not currently detected on the machine, proposal creation or approval must fail before session start.

### Invalid Coordinator Or Summarizer Configuration

If the configured provider, model, base URL, or API key is invalid, unreachable, or missing, the orchestrator must fail proposal generation before any session is created.

### Duplicate Active Run Request

If a ticket already has an active session, moving it again to `in_progress` or re-approving a proposal must not create a second concurrent run for the same ticket.

### Active Session Reassignment

If the user changes the assigned agent while a session is already active, the first implementation should reject the reassignment until the active run is stopped or finished.

### Stale Proposal

If a ticket changes materially after a proposal is generated but before it is approved, the proposal should be treated as stale and regenerated.

### Unstructured Worker Exit

If the worker exits without a valid structured outcome, the orchestrator should mark the run as failed or interrupted, preserve any useful partial context, and avoid moving the ticket to `done`.

### Worker Reports Success Without Useful Output

If the worker reports `completed` but provides no meaningful repo changes, notes, or resumable context, the orchestrator may still move the ticket to `review`, but it must record the weak completion signal in events rather than silently treating it as a high-confidence success.

### Oversized Context Carry

If stored context-carry data becomes too large for the next run, the summarizer must compact it into a bounded payload before proposal generation or runner startup. AgentBoard should persist the compact summary rather than passing unbounded raw history.

### Partial Context On Failure

If a run fails after producing useful intermediate output, the summarizer should persist a partial context-carry payload so a retry from `review` or `in_progress` can continue from the latest known state.

### Ticket Edited During Active Run

If the user edits title, description, dependencies, or assignment while a run is active, the first implementation should treat those edits as affecting only future runs. The active worker session continues with the context it started with unless explicitly stopped and restarted.

### Project Isolation

Sessions, events, context-carry data, proposals, and runner actions must remain scoped to the current project database. A worker session in one project must never be allowed to mutate tickets in another project.

### Ticket-Scoped Permissions

Every worker session must be bound to exactly one ticket. Worker-issued actions must be limited to that ticket and its own session, preventing accidental updates to unrelated tickets.

## Testing Strategy

The first implementation should be test-driven and use fakes aggressively.

Priority coverage:

- coordinator proposal shaping with a fake `LLMClient`
- approval lifecycle behavior
- session lifecycle behavior with a fake runner
- board action validation and legal transition rules
- duplicate active run protection
- subprocess runner behavior where practical

An integration-style fake runner should be introduced before relying on real external CLIs in tests.

## Phased Delivery

### Phase 1

- define orchestrator interfaces and domain types
- add proposal and approval workflow
- implement session lifecycle orchestration
- implement the `exec` runner
- implement structured worker outcome parsing and orchestrator-owned board transitions
- implement constrained board actions
- persist per-ticket context-carry data after a run
- implement reassignment handoff package generation
- persist session and event transitions
- connect the TUI so moving a ticket to `in_progress` creates and approves one child run

### Phase 2

- add richer output streaming and inspection
- add more child actions if required
- expose orchestration services through the API package

### Phase 3

- add tmux and PTY runners
- add more coordinator backends and richer decomposition flows
- support multi-session and multi-agent orchestration

## Open Decisions Resolved In This Design

- Initial execution backend: subprocess `exec` runner
- Start policy: explicit user approval required before each run
- Execution trigger: moving a ticket to `in_progress` creates a run proposal for the assigned agent
- Post-start mutation policy: running child agent may update its own board state through constrained AgentBoard actions
- Post-run policy: successful runs persist context-carry data and move the ticket to `review`
- Acceptance policy: only user review moves a ticket from `review` to `done`
- AI tiering: cheap coordinator plus cheap summarizer, with expensive worker agent used for repo execution
- Extensibility target: one coordinator and one runner first, but architecture remains provider- and backend-extensible

## Success Criteria

The first orchestration slice is successful when a user can:

1. configure one coordinator model
2. assign a ticket to one supported local child agent
3. request a run and receive a proposal
4. approve the proposal in the TUI
5. start the child process through the orchestrator
6. see the ticket become active
7. observe the worker report a structured result and the orchestrator perform the resulting board transition
8. observe the session complete, context-carry data persist, and the ticket move to `review`
9. move the ticket back to `in_progress` and start a follow-up run with the carried context
