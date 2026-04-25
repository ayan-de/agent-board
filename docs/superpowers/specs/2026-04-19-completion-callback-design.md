# Completion Callback for FinishRun

## Problem

`FinishRun` is never called after agent runs complete. This breaks the entire orchestration loop: tickets stay `in_progress` forever, context carry is never persisted, no board transitions happen, and no completion events are recorded.

This affects both runners:
- **ExecRunner**: blocks until the agent exits, returns a `RunHandle` with the parsed outcome, but `StartApprovedRun` never calls `FinishRun` with it.
- **TmuxRunner**: returns immediately with `Outcome: "running"`, and `monitorPane` only sets `pane.Status = "completed"` internally without triggering any downstream processing.

A secondary problem: `monitorPane` only checks if the tmux pane still exists. It always reports `"completed"` regardless of the actual agent outcome.

## Approach

Add an `OnComplete func(outcome, summary string)` callback to `RunRequest`. The Service wires `FinishRun` as this callback. Each runner calls it when the agent finishes.

This keeps the `Runner` interface unchanged, avoids polling, and works for both blocking (ExecRunner) and non-blocking (TmuxRunner) execution models.

## Changes

### 1. RunRequest gets OnComplete

`internal/orchestrator/types.go` — add field to `RunRequest`:

```go
type RunRequest struct {
    TicketID   string
    SessionID  string
    Agent      string
    Prompt     string
    Reporter   func(string)
    InputChan  chan io.Writer
    Target     string
    OnComplete func(outcome, summary string)
}
```

### 2. Service.StartApprovedRun wires the callback

`internal/orchestrator/service.go` — in `StartApprovedRun`:

Build an `onComplete` closure that calls `s.FinishRun` in a goroutine. Pass it as `req.OnComplete`.

After `runner.Start()` returns:
- **ExecRunner** (blocking): `Start()` has already finished. Call `onComplete(handle.Outcome, handle.Summary)` immediately.
- **TmuxRunner** (non-blocking): `Start()` returns with `Outcome: "running"`. The callback will be called later by `monitorPane`.

The key behavioral difference: ExecRunner's `StartApprovedRun` now calls `onComplete` inline after `runner.Start()` returns. TmuxRunner's `StartApprovedRun` does not — the callback fires asynchronously from the monitor goroutine.

### 3. TmuxRunner passes OnComplete to PaneManager

`internal/orchestrator/tmux_runner.go` — `Start()` passes `req.OnComplete` to `pm.CreatePane`.

`internal/orchestrator/pane_manager.go` — `AgentPane` stores the callback. `CreatePane` accepts it. `monitorPane` calls it when pane disappears.

### 4. monitorPane captures output and parses outcome

`internal/orchestrator/pane_manager.go` — when the pane disappears:

1. Capture pane output one final time via `capturePaneOutput` (internal helper using existing `tmux capture-pane` logic).
2. Parse through `parseOpencodeOutput` (shared with ExecRunner) to extract outcome and summary.
3. If parsing yields no useful data, fall back to `Outcome: "completed"`, `Summary: "Agent finished (pane exited)."`.
4. Call `pane.OnComplete(outcome, summary)`.
5. Clean up prompt file.

### 5. Delete disabled test file

Delete `internal/orchestrator/exec_runner_test.go.bak`. It references a removed `Command` field on `ExecRunner` and has a `+build ignore` tag.

### 6. New tests

`internal/orchestrator/service_test.go`:
- `TestStartApprovedRunCallsFinishRunForBlockingRunner`: uses a fake blocking runner that returns a completed outcome, verifies FinishRun fires (check store state: session ended, agent active cleared, status moved).
- `TestStartApprovedRunCallsFinishRunViaOnComplete`: uses a fake runner that holds the `OnComplete` callback, simulates async completion, verifies FinishRun fires.

`internal/orchestrator/pane_manager_test.go` (new):
- Test `monitorPane` calls `OnComplete` with parsed outcome when pane exits.
- Use a fake tmux binary (shell script) that simulates pane disappearing after N seconds.

## What Does NOT Change

- `Runner` interface (`Start` only)
- `RunHandle` struct
- `FinishRun` implementation
- `ApplyRunOutcome` implementation
- TUI layer

## Data Flow After Fix

```
ExecRunner path:
  StartApprovedRun -> runner.Start() [blocks] -> returns RunHandle
    -> onComplete(handle.Outcome, handle.Summary)
    -> goroutine: FinishRun -> summarize -> persist context carry -> end session -> move status -> record event

TmuxRunner path:
  StartApprovedRun -> runner.Start() [returns immediately]
  ... agent runs in tmux pane ...
  monitorPane [1s tick] -> pane gone? -> capture output -> parse outcome
    -> pane.OnComplete(outcome, summary)
    -> goroutine: FinishRun -> summarize -> persist context carry -> end session -> move status -> record event
```
