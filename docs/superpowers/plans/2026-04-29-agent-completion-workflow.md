# Agent Completion Workflow — Corrected Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the real completion gap in the current codebase so async agent runs reliably trigger `FinishRun`, `agent_active` is cleared on completion, tickets move to `review` on successful completion, and the TUI receives completion notifications.

**Architecture:** In the current implementation, the orchestrator service owns completion for blocking runners and async runners only need to invoke the provided `OnComplete` callback. Do not add a second completion path for blocking runners. `Service.StartApprovedRun` and `Service.StartAdHocRun` already call `onComplete` inline when a runner returns a non-`"running"` handle. `TmuxRunner` already completes through `PaneManager.monitorPane`. The missing piece is the `PtyRunner` async completion path, plus the PTY start behavior must be brought back in line with the existing tests.

**Tech Stack:** Go, existing orchestrator package, table-driven tests.

---

## Current Implementation Analysis

### What Already Works

1. **Blocking completion path exists in the service**: `StartApprovedRun` and `StartAdHocRun` call `onComplete(handle.Outcome, handle.Summary)` when the runner returns a non-`"running"` handle. This already covers `ExecRunner`.
2. **Completion notifications are already wired**: the `onComplete` closure calls `FinishRun` and publishes `RunCompletion` on `completionCh`.
3. **TUI already listens for completions**: `internal/tui/app.go` waits on `CompletionChan()` after starting a run and turns the result into `runCompletedMsg`.
4. **FinishRun already applies the desired state changes**: it removes the active session, summarizes context, persists context carry, ends the session, records the event, and calls `ApplyRunOutcome`.
5. **Successful outcomes already move tickets to `review`**: `ApplyRunOutcome` sets `agent_active=false` and calls `MoveStatus(..., "review")` for `"completed"`.
6. **Tmux async completion already exists**: `PaneManager.monitorPane` calls `pane.onComplete(...)` when the pane disappears.

### What Is Actually Broken

| Component | Issue | Severity |
|-----------|-------|----------|
| **PtyRunner** (`internal/orchestrator/pty_runner.go`) | `monitorPane` is an empty stub, so PTY-based async runs never invoke `OnComplete`. | Critical |
| **PTY start behavior vs tests** | `PtyRunner.Start` does not currently match the expectations encoded in `pty_runner_test.go` for tmux session targeting and prompt injection behavior. | High |
| **Plan drift risk** | Adding `req.OnComplete(...)` inside `ExecRunner.Start` would duplicate completion handling, because the service already performs inline completion for blocking runs. | High |

### Explicit Non-Goal

- [ ] Do **not** change `ExecRunner` to call `req.OnComplete(...)` unless the service-owned blocking completion path is deliberately removed in the same change. The current service behavior already handles blocking completion and notification delivery.

---

## File Structure

```text
internal/orchestrator/
├── service.go           # Keep blocking-run completion ownership here
├── pty_runner.go        # Implement async PTY completion and align start behavior
├── pty_runner_test.go   # Add/adjust PTY completion tests
├── service_test.go      # Add explicit completion-channel assertions
├── exec_runner.go       # No completion callback change expected
├── tmux_runner.go       # Reference only; async path already exists
├── pane_manager.go      # Reference only; async path already exists
├── summarizer.go        # Already correct
├── actions.go           # Already correct
└── types.go             # Already contains OnComplete / RunCompletion
```

---

## Task 1: Lock In Current Blocking Completion Behavior

**Intent:** Document the current contract in tests so future refactors do not accidentally break it or introduce duplicate completion.

**Files:**
- Modify: `internal/orchestrator/service_test.go`

- [ ] **Step 1: Add a blocking-run completion notification test**

Add a test that starts an approved run with `fakeRunner{outcome: "completed"}` and asserts that:
- `CompletionChan()` receives exactly one completion
- the completion contains the expected ticket and outcome
- `MoveStatus("review")` and `SetAgentActive(false)` still occur

Suggested shape:

```go
func TestStartApprovedRunBlockingRunnerSendsSingleCompletion(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{ID: "AGE-01", Status: "in_progress"},
		proposal: store.Proposal{
			ID:       "PRO-01",
			TicketID: "AGE-01",
			Agent:    "opencode",
			Status:   "approved",
			Prompt:   "do work",
		},
	}
	fllm := &fakeLLMClient{summary: "summary"}
	runner := &fakeRunner{outcome: "completed", summary: "raw output"}
	svc := orchestrator.NewService(fs, fllm, runner, fakeCtx{})

	_, err := svc.StartApprovedRun(context.Background(), "PRO-01")
	if err != nil {
		t.Fatal(err)
	}

	select {
	case completion := <-svc.CompletionChan():
		if completion.TicketID != "AGE-01" {
			t.Fatalf("TicketID = %q, want AGE-01", completion.TicketID)
		}
		if completion.Outcome != "completed" {
			t.Fatalf("Outcome = %q, want completed", completion.Outcome)
		}
	default:
		t.Fatal("expected completion notification for blocking runner")
	}

	select {
	case extra := <-svc.CompletionChan():
		t.Fatalf("unexpected duplicate completion: %+v", extra)
	default:
	}

	if fs.lastMoveStatus != "review" {
		t.Fatalf("MoveStatus = %q, want review", fs.lastMoveStatus)
	}
	if fs.lastAgentActive != false {
		t.Fatal("AgentActive should be false after FinishRun")
	}
}
```

- [ ] **Step 2: Run the focused service tests**

Run:

```bash
go test ./internal/orchestrator -run 'TestStartApprovedRun(Call|BlockingRunner)' -v
```

Expected:
- existing service completion tests pass
- the new blocking-run notification test passes without modifying `ExecRunner`

- [ ] **Step 3: Do not patch `exec_runner.go`**

If this task tempts you to add `req.OnComplete(...)` inside `ExecRunner.Start`, stop. That change is incorrect unless you also remove the service inline completion path.

---

## Task 2: Fix `PtyRunner` Async Completion

**Intent:** Make PTY-backed async runs behave like `TmuxRunner`: return `"running"` immediately, then invoke `OnComplete` exactly once when the session really finishes.

**Files:**
- Modify: `internal/orchestrator/pty_runner.go`
- Modify: `internal/orchestrator/pty_runner_test.go`

- [ ] **Step 1: Add a PTY completion test that reflects the current async contract**

Write a test around `PtyRunner.Start` that:
- passes an `OnComplete` callback
- uses fake tmux behavior so the pane disappears
- waits briefly for async completion
- asserts the callback fires exactly once

The test should not assume `PtyRunner` returns a terminal outcome synchronously. It should expect `RunHandle{Outcome: "running"}` and verify completion asynchronously.

- [ ] **Step 2: Implement `monitorPane` against the real `PtyRunner` structure**

The implementation in the previous plan was invalid because it referenced fields that do not exist on `PtyRunner`. Rework `monitorPane` using the actual code:

- `PtyRunner` currently shells out to tmux and receives a `paneID`
- the monitor can poll tmux for pane existence, similar to `PaneManager.monitorPane`
- when the pane disappears, it should capture useful output if possible, derive a summary/outcome, and call `onComplete`
- the callback must be nil-safe and must fire at most once

Use the existing `PaneManager.monitorPane` behavior as the reference model, but keep the implementation local to `PtyRunner`.

- [ ] **Step 3: Keep PTY completion ownership in the runner**

`PtyRunner.Start` should continue returning:

```go
RunHandle{Outcome: "running", ...}
```

and the later completion must come from `monitorPane` through `req.OnComplete`.

- [ ] **Step 4: Verify PTY completion test passes**

Run:

```bash
go test ./internal/orchestrator -run TestPtyRunner -v
```

Expected:
- the new completion test passes
- no duplicate callback invocation

---

## Task 3: Bring PTY Start Behavior Back In Line With Existing Tests

**Intent:** The PTY package already has failing tests unrelated to the new completion test. This plan must include them because `go test ./internal/orchestrator` is not green today.

**Files:**
- Modify: `internal/orchestrator/pty_runner.go`
- Modify: `internal/orchestrator/pty_runner_test.go` only if the test expectations are truly obsolete and you can justify that with current product requirements

- [ ] **Step 1: Reproduce the current failures**

Run:

```bash
go test ./internal/orchestrator/... 
```

Current expected failures include:
- prompt injection behavior mismatch
- current tmux session targeting mismatch
- tmux window creation argument mismatch

- [ ] **Step 2: Fix behavior, not just tests**

Align `PtyRunner.Start` with the existing tests unless there is a documented product decision that invalidates them.

The current tests expect at least these behaviors:
- interactive agent launch, not `opencode run`
- prompt injection via tmux buffer/paste flow rather than character-by-character `send-keys`
- targeting the active tmux session when appropriate
- safe handling for short session IDs

- [ ] **Step 3: Re-run PTY-focused tests**

Run:

```bash
go test ./internal/orchestrator -run TestPtyRunner -v
```

Expected:
- the existing PTY start tests pass
- the new PTY completion test also passes

---

## Task 4: Verify Full Orchestrator Completion Flow

**Intent:** Make the completion workflow explicit for both blocking and async paths.

**Files:**
- Modify: `internal/orchestrator/service_test.go`

- [ ] **Step 1: Keep or extend the existing async service test**

`TestStartApprovedRunCallsFinishRunViaAsyncOnComplete` already proves the service reacts correctly once an async runner invokes the callback. Extend it only if needed to assert:
- a completion notification is published
- the notification is published exactly once

- [ ] **Step 2: Add an ad-hoc run completion assertion if coverage is missing**

`StartAdHocRun` uses the same completion channel pattern. Add a focused test only if current coverage does not already protect that path.

- [ ] **Step 3: Run orchestrator tests**

Run:

```bash
go test ./internal/orchestrator/... -v
```

Expected:
- all orchestrator tests pass

---

## Task 5: Repository Verification

- [ ] **Step 1: Run `go vet`**

```bash
go vet ./...
```

- [ ] **Step 2: Run all tests**

```bash
go test ./...
```

- [ ] **Step 3: Build**

```bash
go build -o agentboard ./cmd/agentboard
```

---

## Testing Recommendations

1. **Service-level blocking test**: verify one completion event and no duplicate completion for blocking runners.
2. **Service-level async test**: verify `FinishRun` does not happen before callback, then does happen after callback.
3. **PTY runner test**: verify `OnComplete` fires asynchronously exactly once.
4. **Regression tests for PTY start**: keep the existing tmux session targeting and prompt injection checks green.
5. **Manual validation**: run a real PTY-backed agent and confirm the ticket moves to `review` and the TUI refreshes after completion.

---

## Summary of Expected Code Changes

| File | Change |
|------|--------|
| `internal/orchestrator/service_test.go` | Add explicit assertions for blocking-run notification delivery and duplicate protection |
| `internal/orchestrator/pty_runner.go` | Implement real async completion monitoring and align PTY start behavior with tests |
| `internal/orchestrator/pty_runner_test.go` | Add PTY async completion test and keep existing PTY start tests passing |
| `internal/orchestrator/exec_runner.go` | No completion callback change expected |

---

## Dependencies

- No new dependencies expected
- Existing `OnComplete func(outcome, summary string)` on `RunRequest`
- Existing `completionCh chan RunCompletion` in `Service`
- Existing `FinishRun` and `ApplyRunOutcome`
- Existing `PaneManager.monitorPane` as the model for async completion behavior

---

## Notes For Implementers

- The previous version of this plan was incorrect about `ExecRunner`. Do not reintroduce that mistake.
- The real async gap is PTY completion, not blocking-run notification delivery.
- The PTY area already has red tests. Treat completion work and PTY-start regression fixes as one implementation slice, not two unrelated tasks.
