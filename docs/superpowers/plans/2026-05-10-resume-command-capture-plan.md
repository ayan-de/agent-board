# Resume Command Capture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When an agent is stopped via Ctrl+C, capture the resume command from pane output, store it in the ticket, set `active_agent=0`, and move ticket to `review`.

**Architecture:** Add `resume_command` column to tickets table. Modify the pane monitoring callback chain to capture pane output on exit, extract resume command via regex patterns, and propagate it through `ApplyRunOutcome` to update the ticket.

**Tech Stack:** Go, SQLite (modernc.org/sqlite), tmux

---

## File Structure

| File | Responsibility |
|------|----------------|
| `internal/store/migrations.go` | Add `resume_command` column to tickets table |
| `internal/store/tickets.go` | Add `ResumeCommand` field, getter, setter |
| `internal/orchestrator/types.go` | Add `ResumeCommand` to `ApplyRunOutcomeInput`, `RunCompletion`, update `RunRequest.OnComplete` signature |
| `internal/orchestrator/pane_manager.go` | Change `onComplete` callback to include `resumeCommand`, extract command in `monitorPane()` |
| `internal/orchestrator/actions.go` | Store resume command in ticket via `SetResumeCommand` |
| `internal/orchestrator/summarizer.go` | Pass resume command through completion chain |
| `internal/orchestrator/tmux_runner.go` | Update `SendInput` to match new callback signature |
| `internal/tui/ticket.go` | Display resume command field in ticket detail view |

---

## Task 1: DB Migration

**Files:**
- Modify: `internal/store/migrations.go`

- [ ] **Step 1: Add resume_command column migration**

Find the migrations file and add the new column after `agent_active`:

```go
ALTER TABLE tickets ADD COLUMN resume_command TEXT
```

Run: `go vet ./internal/store/...`
Expected: No errors

- [ ] **Step 2: Commit**

```bash
git add internal/store/migrations.go
git commit -m "store: add resume_command column to tickets table"
```

---

## Task 2: Ticket Model

**Files:**
- Modify: `internal/store/tickets.go:11-24` (Ticket struct)
- Modify: `internal/store/tickets.go` (add getter/setter methods)

- [ ] **Step 1: Add ResumeCommand field to Ticket struct**

```go
type Ticket struct {
    ID            string
    Title         string
    Description   string
    Status        string
    Priority      string
    Agent         string
    Branch        string
    Tags          []string
    DependsOn     []string
    AgentActive   bool
    ResumeCommand string  // NEW
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

- [ ] **Step 2: Add ResumeCommand to ticketRow struct**

```go
type ticketRow struct {
    ID            string
    Title         string
    Description   string
    Status        string
    Priority      string
    Agent         string
    Branch        string
    Tags          string
    DependsOn     string
    AgentActive   bool
    ResumeCommand string  // NEW
    CreatedAt     string
    UpdatedAt     string
}
```

- [ ] **Step 3: Add SetResumeCommand and GetResumeCommand methods**

Add after `SetAgentActive`:

```go
func (s *Store) SetResumeCommand(ctx context.Context, id, cmd string) error {
    _, err := s.db.ExecContext(ctx, "UPDATE tickets SET resume_command = ?, updated_at = ? WHERE id = ?",
        cmd, time.Now().Format(time.RFC3339), id)
    if err != nil {
        return fmt.Errorf("set resume command: %w", err)
    }
    return nil
}

func (s *Store) GetResumeCommand(ctx context.Context, id string) (string, error) {
    var cmd string
    err := s.db.QueryRowContext(ctx, "SELECT resume_command FROM tickets WHERE id = ?", id).Scan(&cmd)
    if err != nil {
        if err == sql.ErrNoRows {
            return "", nil
        }
        return "", fmt.Errorf("get resume command: %w", err)
    }
    return cmd, nil
}
```

- [ ] **Step 4: Update scanTicketRow to include ResumeCommand**

Add `ResumeCommand: row.ResumeCommand` to the Ticket struct mapping in `scanTicketRow`.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/store/... -v -run Ticket`
Expected: All ticket tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/store/tickets.go
git commit -m "store: add ResumeCommand field and accessors"
```

---

## Task 3: Resume Command Extraction

**Files:**
- Create: `internal/orchestrator/resume.go`

- [ ] **Step 1: Write resume extraction function**

```go
package orchestrator

import (
    "regexp"
    "strings"
)

var resumePatterns = []*regexp.Regexp{
    regexp.MustCompile(`(?i)(opencode\s+-s\s+\S+)`),
    regexp.MustCompile(`(?i)(claude\s+--resume\s+\S+)`),
    regexp.MustCompile(`(?i)(codex\s+--resume\s+\S+)`),
    regexp.MustCompile(`(?i)(gemini\s+--resume\s+\S+)`),
}

func ExtractResumeCommand(output string) string {
    if output == "" {
        return ""
    }
    lines := strings.Split(output, "\n")
    for i := len(lines) - 1; i >= 0; i-- {
        line := strings.TrimSpace(lines[i])
        for _, pattern := range resumePatterns {
            match := pattern.FindStringSubmatch(line)
            if len(match) > 1 {
                return match[1]
            }
        }
    }
    return ""
}
```

- [ ] **Step 2: Write tests**

```go
package orchestrator

import "testing"

func TestExtractResumeCommand(t *testing.T) {
    tests := []struct {
        name   string
        output string
        want   string
    }{
        {
            name:   "opencode resume",
            output: "Session   New session - 2026-05-10T07:36:46.777Z\nContinue  opencode -s ses_1ef2eca46ffeKTXRokicTzd5iI",
            want:   "opencode -s ses_1ef2eca46ffeKTXRokicTzd5iI",
        },
        {
            name:   "claude resume",
            output: "Resume this session with:\nclaude --resume 31a136eb-7bf4-496d-b00b-73c3ac8158de",
            want:   "claude --resume 31a136eb-7bf4-496d-b00b-73c3ac8158de",
        },
        {
            name:   "no resume command",
            output: "Agent completed successfully",
            want:   "",
        },
        {
            name:   "empty output",
            output: "",
            want:   "",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := ExtractResumeCommand(tt.output)
            if got != tt.want {
                t.Errorf("ExtractResumeCommand() = %q, want %q", got, tt.want)
            }
        })
    }
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/orchestrator/... -v -run TestExtractResumeCommand`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/orchestrator/resume.go
git commit -m "orchestrator: add ExtractResumeCommand function with tests"
```

---

## Task 4: Update Types

**Files:**
- Modify: `internal/orchestrator/types.go`

- [ ] **Step 1: Update ApplyRunOutcomeInput**

```go
type ApplyRunOutcomeInput struct {
    TicketID      string
    Outcome      string
    ResumeCommand string  // NEW
}
```

- [ ] **Step 2: Update RunCompletion**

```go
type RunCompletion struct {
    TicketID      string
    SessionID    string
    Outcome      string
    Summary      string
    ResumeCommand string  // NEW
}
```

- [ ] **Step 3: Update RunRequest.OnComplete callback signature**

```go
type RunRequest struct {
    TicketID   string
    SessionID  string
    Agent      string
    Prompt     string
    Reporter   func(string)
    Target     string
    OnComplete func(outcome, summary, resumeCommand string)  // CHANGED
}
```

- [ ] **Step 4: Run vet**

Run: `go vet ./internal/orchestrator/...`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/types.go
git commit -m "orchestrator: add ResumeCommand to ApplyRunOutcomeInput, RunCompletion, and RunRequest"
```

---

## Task 5: Update PaneManager Callback

**Files:**
- Modify: `internal/orchestrator/pane_manager.go`

- [ ] **Step 1: Update AgentPane.onComplete callback signature**

```go
type AgentPane struct {
    SessionID  string
    TicketID   string
    Agent      string
    PaneID     string
    WindowID   string
    StartedAt  time.Time
    Status     string
    Outcome    string
    Summary    string
    cancelFunc context.CancelFunc
    promptFile string
    onComplete func(outcome, summary, resumeCommand string)  // CHANGED
}
```

- [ ] **Step 2: Update monitorPane to extract and pass resume command**

In `monitorPane()` around line 246-248, change:
```go
if pane.onComplete != nil {
    pane.onComplete(outcome, summary)
}
```

To:
```go
if pane.onComplete != nil {
    resumeCmd := ExtractResumeCommand(captured)
    pane.onComplete(outcome, summary, resumeCmd)
}
```

- [ ] **Step 3: Run vet**

Run: `go vet ./internal/orchestrator/...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/orchestrator/pane_manager.go
git commit -m "orchestrator: update onComplete callback to include resumeCommand"
```

---

## Task 6: Update Orchestrator Actions

**Files:**
- Modify: `internal/orchestrator/actions.go`

- [ ] **Step 1: Update ApplyRunOutcome to store resume command**

```go
func (s Service) ApplyRunOutcome(ctx context.Context, input ApplyRunOutcomeInput) error {
    if input.ResumeCommand != "" {
        if err := s.store.SetResumeCommand(ctx, input.TicketID, input.ResumeCommand); err != nil {
            return fmt.Errorf("set resume command: %w", err)
        }
    }

    if err := s.store.SetAgentActive(ctx, input.TicketID, false); err != nil {
        return fmt.Errorf("set agent active: %w", err)
    }

    switch input.Outcome {
    case "completed":
        return s.store.MoveStatus(ctx, input.TicketID, "review")
    case "failed", "interrupted", "blocked":
        return nil
    default:
        return fmt.Errorf("orchestrator.applyRunOutcome: unknown outcome %q", input.Outcome)
    }
}
```

- [ ] **Step 2: Run vet**

Run: `go vet ./internal/orchestrator/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/orchestrator/actions.go
git commit -m "orchestrator: store resume command in ApplyRunOutcome"
```

---

## Task 7: Update TmuxRunner

**Files:**
- Modify: `internal/orchestrator/tmux_runner.go`

- [ ] **Step 1: Update Start method's onComplete callback usage**

Find where `RunRequest.OnComplete` is called and update the signature. The callback is invoked around lines where `pane.onComplete(outcome, summary)` is called.

- [ ] **Step 2: Update SendInput**

Check if `SendInput` references the old callback signature and update if needed.

- [ ] **Step 3: Run vet**

Run: `go vet ./internal/orchestrator/...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/orchestrator/tmux_runner.go
git commit -m "orchestrator: update TmuxRunner callback signature"
```

---

## Task 8: Update Summarizer

**Files:**
- Modify: `internal/orchestrator/summarizer.go`

- [ ] **Step 1: Update FinishRun to pass resume command to ApplyRunOutcome**

The `FinishRun` function calls `ApplyRunOutcome`. Update `FinishRunInput` to include `ResumeCommand` if not already, and pass it through.

- [ ] **Step 2: Run vet**

Run: `go vet ./internal/orchestrator/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/orchestrator/summarizer.go
git commit -m "orchestrator: pass resume command through FinishRun"
```

---

## Task 9: TUI Ticket View

**Files:**
- Modify: `internal/tui/ticket.go` (or wherever ticket detail rendering happens)

- [ ] **Step 1: Add resume command display to ticket detail view**

Find the ticket detail view rendering and add a line showing the resume command if set:

```go
if ticket.ResumeCommand != "" {
    fmt.Fprintf(w, "Resume:   %s\n", ticket.ResumeCommand)
} else {
    fmt.Fprintf(w, "Resume:   —\n")
}
```

- [ ] **Step 2: Run build**

Run: `go build ./cmd/agentboard`
Expected: Compiles successfully

- [ ] **Step 3: Commit**

```bash
git add internal/tui/
git commit -m "tui: display resume command in ticket detail view"
```

---

## Task 10: End-to-End Test

- [ ] **Step 1: Test the full flow**

1. Start agentboard TUI
2. Create a ticket and assign an agent
3. Move ticket to in_progress to trigger agent run
4. Press Ctrl+C in the agent pane
5. Verify resume command appears in pane output
6. Verify ticket now has `active_agent=0` and status=`review`
7. Verify resume command is displayed in ticket detail view
8. Press Ctrl+D to close dead pane

---

## Spec Coverage Check

| Spec Section | Task |
|-------------|------|
| DB migration for resume_command column | Task 1 |
| Ticket model with ResumeCommand field | Task 2 |
| Extract resume command from pane output | Task 3 |
| Store resume command in ApplyRunOutcome | Task 6 |
| Display in ticket view | Task 9 |
| Ctrl+C capture workflow | Tasks 5, 6 |
| Ticket moves to review | Task 6 |
| active_agent=0 on stop | Task 6 |

All spec sections are covered. No placeholders found.
