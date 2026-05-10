# Resume Command Capture Design

## Status

- **Created:** 2026-05-10
- **Owner:** ayan-de
- **Staus:** Draft

---

## Background

When an AI agent (opencode, claude-code) is stopped via Ctrl+C, it outputs a resume command before exiting. For example:

```
opencode -s ses_1ef2eca46ffeKTXRokicTzd5iI
claude --resume 31a136eb-7bf4-496d-b00b-73c3ac8158de
```

This resume command is ticket-specific — it resumes that specific agent session for that ticket. The user needs to store this command on the ticket so they can resume the session later.

---

## Workflow

1. User starts an agent run for a ticket (ticket moves to `in_progress`, `active_agent=1`)
2. Agent runs in a tmux pane within the AgentBoard tmux session
3. User presses **Ctrl+C** inside the tmux pane
4. Agent stops, outputs resume command to the pane
5. **System captures** the resume command from pane output, stores in ticket
6. **System sets** `active_agent=0`, moves ticket to `review`
7. User presses **Ctrl+D** to close the dead tmux window (no system action needed)

---

## Design

### 1. Database Migration

**File:** `internal/store/migrations.go`

Add `resume_command TEXT` column to the `tickets` table:

```go
ALTER TABLE tickets ADD COLUMN resume_command TEXT
```

### 2. Ticket Model

**File:** `internal/store/tickets.go`

Add `ResumeCommand` field to the `Ticket` struct:

```go
type Ticket struct {
    // ... existing fields ...
    ResumeCommand string
}
```

Add getter/setter methods:

```go
func (s *Store) SetResumeCommand(ctx context.Context, id, cmd string) error
func (s *Store) GetResumeCommand(ctx context.Context, id string) (string, error)
```

### 3. Orchestrator — Capture Resume Command

**File:** `internal/orchestrator/pane_manager.go`

Modify `monitorPane()` to capture pane output on exit:

```go
func (pm *PaneManager) monitorPane(ctx context.Context, pane *pane) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            // Check if pane still exists
            if !pm.tmux.SessionExists(pane.sessionID) {
                // Capture final output before cleaning up
                output, _ := pm.tmux.CapturePane(pane.windowID, pane.paneID, -50, -1)
                pane.onComplete(output, "interrupted")
                pm.RemovePane(pane.sessionID)
                return
            }
            time.Sleep(500 * time.Millisecond)
        }
    }
}
```

The `onComplete()` callback signature needs to accept output:

```go
type PaneCallback func(output string, outcome string)
```

### 4. Resume Command Extraction

**File:** `internal/orchestrator/actions.go` (or new file)

Create a function to extract resume command from pane output:

```go
var resumePatterns = []*regexp.Regexp{
    regexp.MustCompile(`opencode\s+-s\s+(\S+)`),
    regexp.MustCompile(`claude\s+--resume\s+(\S+)`),
    regexp.MustCompile(`codex\s+--resume\s+(\S+)`),
    regexp.MustCompile(`gemini\s+--resume\s+(\S+)`),
}

func ExtractResumeCommand(output string) string {
    for _, pattern := range resumePatterns {
        match := pattern.FindStringSubmatch(output)
        if len(match) > 1 {
            return match[0] // Return full command
        }
    }
    return ""
}
```

### 5. Orchestrator Actions

**File:** `internal/orchestrator/actions.go`

Modify `ApplyRunOutcome()` to also capture and store resume command:

```go
func (s *Service) ApplyRunOutcome(ctx context.Context, input ApplyRunOutcomeInput) error {
    ticketID := input.TicketID

    // Capture resume command if provided
    if input.ResumeCommand != "" {
        if err := s.store.SetResumeCommand(ctx, ticketID, input.ResumeCommand); err != nil {
            return fmt.Errorf("set resume command: %w", err)
        }
    }

    // Set agent inactive
    if err := s.store.SetAgentActive(ctx, ticketID, false); err != nil {
        return fmt.Errorf("set agent active: %w", err)
    }

    // Move to review if completed
    if input.Outcome == "completed" {
        return s.store.MoveStatus(ctx, ticketID, "review")
    }
    return nil
}
```

Update the `RunCompletion` struct to include `ResumeCommand`:

```go
type RunCompletion struct {
    TicketID      string
    Outcome       string // completed, failed, interrupted
    ResumeCommand string
}
```

### 6. TmuxRunner Integration

**File:** `internal/orchestrator/tmux_runner.go`

Modify `SendInput()` to send Ctrl+C to the pane when user presses Ctrl+C in tmux. The runner needs to detect when the pane process exits and trigger capture.

Actually, the current architecture already handles this — `monitorPane()` detects pane exit via `tmux SessionExists()`. The issue is capturing the output before cleanup. We need to capture output before calling `RemovePane()`.

### 7. TUI — Show Resume Command

**File:** `internal/tui/ticket.go`

Add a read-only `Resume Command` field to the ticket detail view:

```
┌─ Ticket: DEV-042 ───────────────────────────────────┐
│ Title:    Implement user authentication              │
│ Status:   in_review                                │
│ Priority: high                                      │
│ Agent:    claude-code                               │
│ Branch:   feature/auth                              │
│ Tags:     [security auth]                           │
│ Resume:   claude --resume 31a136eb-7bf4-...        │
│                                                   │
│ Description:                                        │
│ ...
```

The field should display the resume command if set, otherwise show "—" or be hidden.

### 8. Signal Handling Clarification

- **Ctrl+C** inside tmux pane → sends SIGINT to agent process → agent outputs resume command and exits → `monitorPane()` detects exit → captures output
- **Ctrl+D** inside tmux pane → closes tmux pane/window (tmux session itself is already dead) → no system action needed, no event fired

---

## Files to Modify

| File | Change |
|------|--------|
| `internal/store/migrations.go` | Add `resume_command` column |
| `internal/store/tickets.go` | Add field, getter, setter |
| `internal/orchestrator/pane_manager.go` | Capture output on pane exit |
| `internal/orchestrator/types.go` | Update `RunCompletion` with `ResumeCommand` |
| `internal/orchestrator/actions.go` | Store resume command in `ApplyRunOutcome()` |
| `internal/orchestrator/summarizer.go` | Pass resume command through |
| `internal/tui/ticket.go` | Display resume command field |

---

## Testing

1. Start agent run for ticket → ticket moves to `in_progress`, `active_agent=1`
2. Press Ctrl+C in agent pane → agent outputs resume command and exits
3. Verify ticket has `resume_command` set, `active_agent=0`, status=`review`
4. Verify resume command is displayed in ticket detail view
5. Press Ctrl+D to close dead pane → no error

---

## Notes

- The resume command is per-ticket, not per-session (if user resumes multiple times, the last command overwrites)
- Resume command is currently read-only from TUI; user edits are out of scope for now
- Pattern matching is simple regex; assumes command appears in last ~50 lines of pane output
- Different agents (opencode, claude-code, codex, gemini) have different resume command formats — all patterns covered
