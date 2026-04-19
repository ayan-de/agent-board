# Completion Callback for FinishRun — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire `FinishRun` into the orchestration loop so board transitions, context carry persistence, and event recording happen automatically after agent runs complete.

**Architecture:** Add an `OnComplete` callback to `RunRequest`. The Service wires `FinishRun` as this callback. ExecRunner calls it inline after its blocking `Start()` returns. TmuxRunner's `monitorPane` calls it when the pane disappears, after capturing and parsing the pane output.

**Tech Stack:** Go, existing orchestrator package, table-driven tests.

---

### Task 1: Add OnComplete to RunRequest

**Files:**
- Modify: `internal/orchestrator/types.go:27-35`

- [ ] **Step 1: Add OnComplete field to RunRequest**

In `internal/orchestrator/types.go`, add the `OnComplete` field to the `RunRequest` struct:

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

- [ ] **Step 2: Run tests to verify nothing breaks**

Run: `go test ./internal/orchestrator/...`
Expected: all existing tests pass (OnComplete is optional, nil by default)

- [ ] **Step 3: Commit**

```bash
git add internal/orchestrator/types.go
git commit -m "feat(orchestrator): add OnComplete callback to RunRequest"
```

---

### Task 2: Wire OnComplete in Service.StartApprovedRun for ExecRunner

**Files:**
- Modify: `internal/orchestrator/service.go:100-174`

- [ ] **Step 1: Write failing test for ExecRunner calling FinishRun**

Add to `internal/orchestrator/service_test.go`:

```go
func TestStartApprovedRunCallsFinishRunForBlockingRunner(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{
			ID:     "AGE-01",
			Status: "in_progress",
		},
		proposal: store.Proposal{
			ID:       "PRO-01",
			TicketID: "AGE-01",
			Agent:    "opencode",
			Status:   "approved",
			Prompt:   "do work",
		},
	}
	fllm := &fakeLLMClient{summary: "summary of run"}
	runner := &fakeRunner{outcome: "completed", summary: "raw worker output"}
	svc := orchestrator.NewService(fs, fllm, runner, fakeCtx{})

	_, err := svc.StartApprovedRun(context.Background(), "PRO-01")
	if err != nil {
		t.Fatal(err)
	}

	if fs.lastMoveStatus != "review" {
		t.Fatalf("MoveStatus = %q, want review (FinishRun should have been called)", fs.lastMoveStatus)
	}
	if fs.lastAgentActive != false {
		t.Fatal("AgentActive should be false after FinishRun")
	}
	if fs.lastContextCarry.Summary != "summary of run" {
		t.Fatalf("ContextCarry.Summary = %q, want %q", fs.lastContextCarry.Summary, "summary of run")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/orchestrator/ -run TestStartApprovedRunCallsFinishRunForBlockingRunner -v`
Expected: FAIL — `MoveStatus = "", want review`

- [ ] **Step 3: Implement OnComplete wiring in StartApprovedRun**

Replace the body of `StartApprovedRun` in `internal/orchestrator/service.go` from line 100 through line 174. The key changes:

1. Build an `onComplete` closure that calls `s.FinishRun` in a goroutine.
2. Pass it as `req.OnComplete`.
3. After `runner.Start()` returns, if the handle outcome is not `"running"` (i.e. blocking runner), call `onComplete` inline.

Replace the section from `// Start the agent` comment through the end of the function:

```go
	onComplete := func(outcome, summary string) {
		go func() {
			_ = s.FinishRun(context.Background(), FinishRunInput{
				TicketID:  proposal.TicketID,
				SessionID: session.ID,
				Outcome:   outcome,
				Summary:   summary,
			})
		}()
	}

	handle, err := s.runner.Start(ctx, RunRequest{
		TicketID:   proposal.TicketID,
		SessionID:  session.ID,
		Agent:      proposal.Agent,
		Prompt:     proposal.Prompt,
		Reporter:   func(line string) { s.AppendLog(session.ID, line) },
		InputChan:  inputChan,
		OnComplete: onComplete,
	})

	if err != nil {
		_ = s.store.EndSession(ctx, session.ID, "failed")
		_ = s.store.SetAgentActive(ctx, proposal.TicketID, false)
		return store.Session{}, err
	}

	s.mu.Lock()
	s.activeSessions[session.ID] = &AgentSession{
		SessionID: session.ID,
		TicketID:  proposal.TicketID,
		Agent:     proposal.Agent,
		StartedAt: session.StartedAt.Unix(),
		Status:    "running",
	}
	s.mu.Unlock()

	_, _ = s.store.CreateEvent(ctx, store.Event{
		TicketID:  proposal.TicketID,
		SessionID: session.ID,
		Kind:      "run.started",
		Payload:   handle.Outcome,
	})

	if handle.Outcome != "running" {
		onComplete(handle.Outcome, handle.Summary)
	}

	return session, nil
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/orchestrator/ -run TestStartApprovedRunCallsFinishRunForBlockingRunner -v`
Expected: PASS

- [ ] **Step 5: Run all orchestrator tests**

Run: `go test ./internal/orchestrator/ -v`
Expected: all tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/orchestrator/service.go internal/orchestrator/service_test.go
git commit -m "feat(orchestrator): wire FinishRun via OnComplete callback for blocking runners"
```

---

### Task 3: Write test for async OnComplete (non-blocking runner)

**Files:**
- Modify: `internal/orchestrator/helpers_test.go`
- Modify: `internal/orchestrator/service_test.go`

- [ ] **Step 1: Add fakeAsyncRunner to helpers_test.go**

Add to `internal/orchestrator/helpers_test.go` after the `fakeRunner` definition:

```go
type fakeAsyncRunner struct {
	onComplete func(outcome, summary string)
}

func (f *fakeAsyncRunner) Start(_ context.Context, req orchestrator.RunRequest) (orchestrator.RunHandle, error) {
	f.onComplete = req.OnComplete
	return orchestrator.RunHandle{Outcome: "running", Summary: "async started"}, nil
}
```

- [ ] **Step 2: Write failing test for async OnComplete**

Add to `internal/orchestrator/service_test.go`:

```go
func TestStartApprovedRunCallsFinishRunViaAsyncOnComplete(t *testing.T) {
	fs := &fakeStore{
		ticket: store.Ticket{
			ID:     "AGE-01",
			Status: "in_progress",
		},
		proposal: store.Proposal{
			ID:       "PRO-01",
			TicketID: "AGE-01",
			Agent:    "opencode",
			Status:   "approved",
			Prompt:   "do work",
		},
	}
	fllm := &fakeLLMClient{summary: "summary of async run"}
	runner := &fakeAsyncRunner{}
	svc := orchestrator.NewService(fs, fllm, runner, fakeCtx{})

	_, err := svc.StartApprovedRun(context.Background(), "PRO-01")
	if err != nil {
		t.Fatal(err)
	}

	if fs.lastMoveStatus != "" {
		t.Fatal("FinishRun should NOT have been called yet for non-blocking runner")
	}

	runner.onComplete("completed", "async worker output")

	if fs.lastMoveStatus != "review" {
		t.Fatalf("MoveStatus = %q, want review after OnComplete fires", fs.lastMoveStatus)
	}
	if fs.lastContextCarry.Summary != "summary of async run" {
		t.Fatalf("ContextCarry.Summary = %q, want %q", fs.lastContextCarry.Summary, "summary of async run")
	}
}
```

- [ ] **Step 3: Run test to verify it passes**

Run: `go test ./internal/orchestrator/ -run TestStartApprovedRunCallsFinishRunViaAsyncOnComplete -v`
Expected: PASS (the wiring from Task 2 already handles this — OnComplete is stored and called later)

- [ ] **Step 4: Commit**

```bash
git add internal/orchestrator/helpers_test.go internal/orchestrator/service_test.go
git commit -m "test(orchestrator): add async runner FinishRun test via OnComplete callback"
```

---

### Task 4: Wire OnComplete through TmuxRunner to PaneManager

**Files:**
- Modify: `internal/orchestrator/tmux_runner.go:26-43`
- Modify: `internal/orchestrator/pane_manager.go:20-33` (AgentPane struct)
- Modify: `internal/orchestrator/pane_manager.go:48-141` (CreatePane)
- Modify: `internal/orchestrator/pane_manager.go:144-181` (monitorPane)

- [ ] **Step 1: Add OnComplete to AgentPane and CreatePane**

In `internal/orchestrator/pane_manager.go`, add `onComplete` field to `AgentPane`:

```go
type AgentPane struct {
	SessionID   string
	TicketID    string
	Agent       string
	PaneID      string
	WindowID    string
	StartedAt   time.Time
	Status      string
	Outcome     string
	Summary     string
	cancelFunc  context.CancelFunc
	promptFile  string
	onComplete  func(outcome, summary string)
}
```

In `CreatePane`, store the callback from the request. After `pane.Agent = req.Agent` (around line 64), it's already set. We need to store the callback — add after `pane` initialization:

```go
	pane.onComplete = req.OnComplete
```

This line goes right after `cancelFunc: cancel,` in the pane struct literal.

- [ ] **Step 2: Update monitorPane to capture output, parse outcome, and call OnComplete**

Replace the `monitorPane` method in `internal/orchestrator/pane_manager.go`:

```go
func (pm *PaneManager) monitorPane(ctx context.Context, pane *AgentPane, reporter func(string)) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			checkCmd := exec.Command(pm.tmux, "list-panes", "-t", pane.PaneID, "-F", "#{pane_pid}")
			output, err := checkCmd.Output()

			if err != nil || len(output) == 0 {
				outcome := "completed"
				summary := fmt.Sprintf("Agent %s finished for ticket %s", pane.Agent, pane.TicketID)

				captured, capErr := pm.capturePaneOutput(pane.PaneID, 200)
				if capErr == nil && captured != "" {
					parsed, parseErr := parseOpencodeOutput(strings.NewReader(captured))
					if parseErr == nil {
						if parsed.Outcome != "" {
							outcome = parsed.Outcome
						}
						if parsed.Summary != "" {
							summary = parsed.Summary
						}
					}
				}

				pm.mu.Lock()
				pane.Status = outcome
				pane.Outcome = outcome
				pane.Summary = summary
				if pane.promptFile != "" {
					_ = os.Remove(pane.promptFile)
					pane.promptFile = ""
				}
				pm.mu.Unlock()

				if reporter != nil {
					reporter(summary)
				}
				if pane.onComplete != nil {
					pane.onComplete(outcome, summary)
				}
				return
			}
		}
	}
}
```

Add the `capturePaneOutput` helper method to `PaneManager`:

```go
func (pm *PaneManager) capturePaneOutput(paneID string, lines int) (string, error) {
	captureCmd := exec.Command(pm.tmux, "capture-pane", "-t", paneID, "-p", "-e", "-J", "-C", "-P",
		"-S", fmt.Sprintf("-%d", lines))
	output, err := captureCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture pane output: %w", err)
	}
	return string(output), nil
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/orchestrator/ -v`
Expected: all tests pass (tmux code is not exercised by unit tests, so no regressions)

- [ ] **Step 4: Run vet**

Run: `go vet ./internal/orchestrator/`
Expected: no issues

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/tmux_runner.go internal/orchestrator/pane_manager.go
git commit -m "feat(orchestrator): wire OnComplete through TmuxRunner to PaneManager monitor"
```

---

### Task 5: Delete disabled test file and add parseOpencodeOutput tests

**Files:**
- Delete: `internal/orchestrator/exec_runner_test.go.bak`
- Create: `internal/orchestrator/exec_runner_test.go`

- [ ] **Step 1: Delete the .bak file**

```bash
rm internal/orchestrator/exec_runner_test.go.bak
```

- [ ] **Step 2: Create new exec_runner tests**

Create `internal/orchestrator/exec_runner_test.go` with tests for `parseOpencodeOutput` (the pure function that doesn't require a subprocess):

```go
package orchestrator_test

import (
	"strings"
	"testing"

	"github.com/ayan-de/agent-board/internal/orchestrator"
)

func TestParseOpencodeOutputCompleted(t *testing.T) {
	input := `{"type":"text","part":{"text":"Hello!","type":"text"}}
{"type":"step_finish","part":{"reason":"stop"}}`

	handle, err := orchestrator.ParseOpencodeOutput(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if handle.Outcome != "completed" {
		t.Fatalf("Outcome = %q, want completed", handle.Outcome)
	}
	if handle.Summary != "Hello!" {
		t.Fatalf("Summary = %q, want Hello!", handle.Summary)
	}
}

func TestParseOpencodeOutputFailed(t *testing.T) {
	input := `{"type":"text","part":{"text":"something went wrong","type":"text"}}
{"type":"step_finish","part":{"reason":"error"}}`

	handle, err := orchestrator.ParseOpencodeOutput(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if handle.Outcome != "failed" {
		t.Fatalf("Outcome = %q, want failed", handle.Outcome)
	}
}

func TestParseOpencodeOutputMultipleTexts(t *testing.T) {
	input := `{"type":"text","part":{"text":"step 1","type":"text"}}
{"type":"text","part":{"text":"step 2","type":"text"}}
{"type":"step_finish","part":{"reason":"stop"}}`

	handle, err := orchestrator.ParseOpencodeOutput(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if handle.Summary != "step 1\nstep 2" {
		t.Fatalf("Summary = %q, want step 1\\nstep 2", handle.Summary)
	}
}

func TestParseOpencodeOutputEmpty(t *testing.T) {
	handle, err := orchestrator.ParseOpencodeOutput(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if handle.Outcome != "completed" {
		t.Fatalf("Outcome = %q, want completed", handle.Outcome)
	}
	if handle.Summary != "Agent finished its task (UI mode)." {
		t.Fatalf("Summary = %q, want default", handle.Summary)
	}
}

func TestParseOpencodeOutputNonJSON(t *testing.T) {
	input := "not json at all\nalso not json"

	handle, err := orchestrator.ParseOpencodeOutput(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if handle.Outcome != "completed" {
		t.Fatalf("Outcome = %q, want completed for non-JSON input", handle.Outcome)
	}
}

func TestExecRunnerAgentNotFound(t *testing.T) {
	runner := orchestrator.ExecRunner{
		LookPath: func(name string) (string, error) {
			return "", strings.NewReader("").Close()
		},
	}
	_ = runner
}
```

Wait — the `LookPath` fake needs to return an error properly. Let me fix:

```go
package orchestrator_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ayan-de/agent-board/internal/orchestrator"
)

func TestParseOpencodeOutputCompleted(t *testing.T) {
	input := `{"type":"text","part":{"text":"Hello!","type":"text"}}
{"type":"step_finish","part":{"reason":"stop"}}`
	handle, err := orchestrator.ParseOpencodeOutput(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if handle.Outcome != "completed" {
		t.Fatalf("Outcome = %q, want completed", handle.Outcome)
	}
	if handle.Summary != "Hello!" {
		t.Fatalf("Summary = %q, want Hello!", handle.Summary)
	}
}

func TestParseOpencodeOutputFailed(t *testing.T) {
	input := `{"type":"text","part":{"text":"something went wrong","type":"text"}}
{"type":"step_finish","part":{"reason":"error"}}`
	handle, err := orchestrator.ParseOpencodeOutput(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if handle.Outcome != "failed" {
		t.Fatalf("Outcome = %q, want failed", handle.Outcome)
	}
}

func TestParseOpencodeOutputMultipleTexts(t *testing.T) {
	input := `{"type":"text","part":{"text":"step 1","type":"text"}}
{"type":"text","part":{"text":"step 2","type":"text"}}
{"type":"step_finish","part":{"reason":"stop"}}`
	handle, err := orchestrator.ParseOpencodeOutput(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if handle.Summary != "step 1\nstep 2" {
		t.Fatalf("Summary = %q, want step 1\\nstep 2", handle.Summary)
	}
}

func TestParseOpencodeOutputEmpty(t *testing.T) {
	handle, err := orchestrator.ParseOpencodeOutput(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if handle.Outcome != "completed" {
		t.Fatalf("Outcome = %q, want completed", handle.Outcome)
	}
	if handle.Summary != "Agent finished its task (UI mode)." {
		t.Fatalf("Summary = %q, want default", handle.Summary)
	}
}

func TestParseOpencodeOutputNonJSON(t *testing.T) {
	handle, err := orchestrator.ParseOpencodeOutput(strings.NewReader("not json\nalso not json"))
	if err != nil {
		t.Fatal(err)
	}
	if handle.Outcome != "completed" {
		t.Fatalf("Outcome = %q, want completed for non-JSON input", handle.Outcome)
	}
}

func TestExecRunnerAgentNotFound(t *testing.T) {
	runner := orchestrator.ExecRunner{
		LookPath: func(name string) (string, error) {
			return "", fmt.Errorf("not found")
		},
	}
	_, err := runner.Start(nil, orchestrator.RunRequest{
		Agent:  "nonexistent",
		Prompt: "do work",
	})
	if err == nil {
		t.Fatal("expected error for missing agent")
	}
}
```

- [ ] **Step 3: Export parseOpencodeOutput**

The tests call `orchestrator.ParseOpencodeOutput` but the current function is lowercase (unexported). In `internal/orchestrator/exec_runner.go`, rename `parseOpencodeOutput` to `ParseOpencodeOutput`:

Change the function signature from:
```go
func parseOpencodeOutput(r io.Reader) (RunHandle, error) {
```
to:
```go
func ParseOpencodeOutput(r io.Reader) (RunHandle, error) {
```

Also update the call site in `Start()`:
```go
	return ParseOpencodeOutput(&fullOutput)
```

- [ ] **Step 4: Run all tests**

Run: `go test ./internal/orchestrator/ -v`
Expected: all tests pass

- [ ] **Step 5: Commit**

```bash
git add -u internal/orchestrator/exec_runner_test.go.bak
git add internal/orchestrator/exec_runner.go internal/orchestrator/exec_runner_test.go
git commit -m "test(orchestrator): replace disabled .bak tests with parseOpencodeOutput tests"
```

---

### Task 6: Run full test suite and vet

- [ ] **Step 1: Run go vet**

Run: `go vet ./...`
Expected: no issues

- [ ] **Step 2: Run all tests**

Run: `go test ./...`
Expected: all tests pass

- [ ] **Step 3: Build**

Run: `go build -o agentboard ./cmd/agentboard`
Expected: builds successfully

- [ ] **Step 4: Final commit (if any fixes needed)**

Only if the previous steps revealed issues that needed fixing.
