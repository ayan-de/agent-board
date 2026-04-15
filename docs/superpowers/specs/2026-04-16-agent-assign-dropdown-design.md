# Agent Assignment Dropdown + Kanban Agent Dot

## Summary

Add an agent selection dropdown to the ticket detail view and show a colored dot on kanban board tickets that have an assigned agent.

## Motivation

Currently the Agent field in ticket view is a free-text field edited via `e`. Users must type the exact agent name. A dropdown showing detected agents is faster and prevents typos. On the kanban board, there's no visual indicator of which agent is assigned to which ticket.

## Design

### 1. Agent Select Mode in Ticket View

**File:** `internal/tui/ticketview.go`

Add a new mode `ticketAgentSelectMode` alongside existing `ticketViewMode` and `ticketEditMode`.

New fields on `TicketViewModel`:
- `agentCursor int` — cursor position in agent list
- `agents []config.DetectedAgent` — list of detected agents passed in during construction

**Trigger:** Pressing `a` while in view mode enters agent select mode. This mirrors how `e` enters edit mode.

**Dropdown rendering:** Below the Agent field row, a bordered list appears:
- First item: `[None]` (clears agent assignment)
- Remaining items: colored bullet + agent name (e.g. `● claude-code`)
- Selected item highlighted with background color
- `j/k` navigate, `Enter` selects, `Esc` cancels

**On selection:** Updates `ticket.Agent` and persists via `store.UpdateTicket()`, then returns to view mode.

### 2. Agent Color Dot on Kanban Board

**File:** `internal/tui/kanban.go`

After rendering `ID Title` for each ticket, if `ticket.Agent != ""`, append a colored circle character.

Format: `▸ AGE-01 Some Title ●` where `●` is colored per agent's defined color.

Agent colors (from `config/agent_detect.go`):
- claude-code: `#D97757`
- opencode: `#808080`
- codex: `#10A37F`
- cursor: `#F0DB4F`

### 3. Agent Color Helper

**File:** `internal/config/agent_detect.go` (add helper)

Extract a function `AgentColor(name string) string` that maps agent name to hex color. Returns empty string for unknown agents. Used by both ticket view dropdown and kanban rendering.

### 4. Keybinding

- In ticket view: `a` enters agent select mode (new behavior, no conflict — `a` currently does nothing in ticket view)
- `s` continues to cycle status (unchanged)

### 5. Data Flow

```
Ticket view (browse) → press 'a' → agent select mode
  → j/k navigate agents
  → Enter → store.UpdateTicket() → view mode
  → Esc → view mode (no change)
```

## Files Changed

| File | Change |
|------|--------|
| `internal/config/agent_detect.go` | Add `AgentColor()` helper |
| `internal/tui/ticketview.go` | Add `ticketAgentSelectMode`, dropdown rendering, key handling |
| `internal/tui/kanban.go` | Add colored dot after ticket title |
| `internal/tui/app.go` | Pass detected agents to `TicketViewModel` |

## Testing

- `ticketview_test.go`: Test mode transitions, agent selection, cancel, None option
- `kanban_test.go`: Test colored dot rendering for assigned/unassigned tickets
- `agent_detect_test.go`: Test `AgentColor()` for known and unknown agents
