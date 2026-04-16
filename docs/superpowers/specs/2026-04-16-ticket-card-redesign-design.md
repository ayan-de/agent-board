# Ticket Card Redesign — Descriptive Cards with Activity Indicator

**Date**: 2026-04-16
**Status**: Approved

## Problem

Kanban tickets are rendered as single-line items (`▸ AGT-01 Ticket Title ●`), providing minimal information at a glance. Users cannot see descriptions, priority, or whether an agent is actively working without entering the ticket detail view. There is no visual signal for agent activity.

## Design

Replace single-line ticket rendering with bordered, card-like ticket components that adapt between compact (collapsed) and expanded (selected) modes. Add an animated **Agent Activity Indicator** — a scrolling block bar that pulses when an agent is actively working on a ticket. Place an agent status dot at the bottom-right corner of each card.

### Activity Indicator Behavior

| Agent State | Indicator | Bottom-Right Dot |
|-------------|-----------|------------------|
| No agent assigned | No indicator | Empty |
| Agent assigned, idle | No indicator | Muted dot `○` |
| Agent running | Animated scrolling blocks | Colored pulsing dot `●` |
| Agent done/failed | No indicator | Green `●` / Red `●` |

The indicator is NOT a progress bar — there is no percentage. It simply signals: "the agent is working, the system is not idle."

### Compact Card (3 lines, default)

**Agent idle / no agent:**
```
╭─────────────────────╮
│ AGT-01 Ticket Title │
│ A brief description…│
│               medium○│
╰─────────────────────╯
```

**Agent actively running:**
```
╭─────────────────────╮
│ AGT-01 Ticket Title │
│ A brief description…│
│ ▓▓▓░░░░▓▓░░   ●●claude│
╰─────────────────────╯
```

- Line 1: ID + Title
- Line 2: Description preview (1 line, truncated with `…`)
- Line 3: Activity indicator (animated scrolling blocks when running) + bottom-right agent dot + agent name

### Expanded Card (7-8 lines, selected ticket)

**Agent idle / no agent:**
```
╭─────────────────────────────╮
│ AGT-01 Full Ticket Title    │
│ ─────────────────────────── │
│ Full description text that  │
│ wraps across multiple lines │
│ showing the complete detail. │
│                             │
│ ⬥ medium              ○    │
╰─────────────────────────────╯
```

**Agent actively running:**
```
╭─────────────────────────────╮
│ AGT-01 Full Ticket Title    │
│ ─────────────────────────── │
│ Full description text that  │
│ wraps across multiple lines │
│ showing the complete detail. │
│                             │
│ ▓▓▓▓░░░░▓▓▓░░░  ● claude  │
╰─────────────────────────────╯
```

- Line 1: ID + Title
- Line 2: Separator
- Lines 3-N: Full description (word-wrapped)
- Metadata line: Priority indicator + activity indicator bar + agent status dot + agent name (right-aligned)

### Adaptive Behavior

- All cards render in compact mode by default (3 lines)
- The selected ticket (current cursor position in focused column) expands to show full details
- Kanban layout dynamically calculates how many cards fit based on expanded card height

## Data Model

### Ticket.AgentActive field

Add an `AgentActive bool` field to the `store.Ticket` struct. Stored in SQLite via a new migration:

```sql
ALTER TABLE tickets ADD COLUMN agent_active INTEGER DEFAULT 0;
```

This is set to `true` by the orchestrator when an agent starts working on a ticket, and `false` when it stops (completes, fails, is cancelled). Until the orchestrator exists, it defaults to `false`.

### Deriving agent state

The card determines the display state from a combination of existing fields:

| Condition | Display State |
|-----------|---------------|
| `ticket.Agent == ""` | No agent — no indicator, no dot |
| `ticket.Agent != "" && !ticket.AgentActive` | Idle — no indicator, muted dot |
| `ticket.Agent != "" && ticket.AgentActive` | Running — animated indicator, colored dot |

## Component Architecture

### New file: `internal/tui/ticketcard.go`

```go
type TicketCardModel struct {
    ticket     store.Ticket
    selected   bool
    expanded   bool
    width      int
    animFrame  int
    styles     TicketCardStyles
}

type TicketCardStyles struct {
    SelectedBorder    lipgloss.Style
    NormalBorder      lipgloss.Style
    Title             lipgloss.Style
    Description       lipgloss.Style
    Metadata          lipgloss.Style
    ActivityFilled    lipgloss.Color
    ActivityEmpty     lipgloss.Color
    AgentDotActive    lipgloss.Color
    AgentDotIdle      lipgloss.Color
    AgentDotDone      lipgloss.Color
    AgentDotFailed    lipgloss.Color
    PriorityColors    map[string]lipgloss.Color
}
```

**Methods:**

| Method | Purpose |
|--------|---------|
| `NewTicketCardModel(ticket, selected, expanded, width, frame, theme)` | Constructor |
| `Render() string` | Returns rendered card string (compact or expanded) |
| `RenderActivity(frame, width) string` | Builds animated activity indicator |
| `RenderAgentDot() string` | Returns colored dot based on agent state |
| `CompactHeight() int` | Returns 3 (constant) |
| `ExpandedHeight() int` | Calculated from description length + width |

**Style constructors** (matching existing pattern):

- `DefaultTicketCardStyles()` — hardcoded ANSI fallback
- `NewTicketCardStyles(t *theme.Theme)` — theme-aware colors

### Animation: `internal/tui/ticketer_animation.go`

Separate file for animation logic to keep concerns clean:

```go
const AnimFrames = 8

var animPatterns = [AnimFrames]string{
    "░░▒▒▓▓██▓▓▒▒░░",
    "░▒▒▓▓██▓▓▒▒░░░",
    "▒▒▓▓██▓▓▒▒░░░░",
    "▒▓▓██▓▓▒▒░░░░▒",
    "▓▓██▓▓▒▒░░░░▒▒",
    "▓██▓▓▒▒░░░░▒▒▓",
    "██▓▓▒▒░░░░▒▒▓▓",
    "▓▓▒▒░░░░▒▒▓▓██",
}

func ActivityBar(frame int, width int, theme *theme.Theme) string
```

The animation is a scrolling gradient pattern with 8 predefined frames. Each frame's 14-character pattern is tiled/truncated to fit the target width. `frame` cycles 0→7 (advanced by `tea.Tick`). Colors come from the active theme — `Accent` for filled blocks, `TextMuted` for empty blocks.

The bar only renders when `ticket.AgentActive == true`. Otherwise the line shows just the priority and agent dot.

### Changes to `internal/tui/kanban.go`

1. **Card rendering**: Replace single-line ticket rendering with `TicketCardModel.Render()` calls
2. **Layout calculation**: Compute visible cards based on card line heights (compact = 3 lines + 1 gap, expanded = dynamic + 1 gap)
3. **Overflow indicator**: Keep `"↓ N more"` but adjusted for multi-line cards
4. **Animation tick**: Add `tea.Tick` command to advance animation frames when any ticket has `AgentActive == true`. When no agents are active, no tick is sent (zero overhead).
5. **Agent state**: Pass `ticket.AgentActive` to card models

### Changes to `internal/tui/app.go`

1. Wire the animation tick through the update loop: kanban returns a `tea.Tick(120ms)` cmd when agents are active, advancing `animFrame` on each tick.

### Changes to `internal/store/tickets.go`

1. Add `AgentActive bool` to `Ticket` struct
2. Update `CreateTicket` and `UpdateTicket` to handle `agent_active` field
3. Add `SetAgentActive(ticketID string, active bool) error` method

### Changes to `internal/store/migrations.go`

Add migration: `ALTER TABLE tickets ADD COLUMN agent_active INTEGER DEFAULT 0`.

### Changes to `internal/theme/themes/*.json`

No theme file changes needed. Activity indicator uses existing `Accent` and `TextMuted` theme colors. Agent dots use existing `Success`, `Error`, `TextMuted` colors.

## Data Flow

```
Orchestrator (future) → sets Ticket.AgentActive = true/false in SQLite
                              ↓
KanbanModel.loadColumns() → reads tickets from store (includes AgentActive)
                              ↓
KanbanModel.Update() → if any AgentActive, tick animFrame
KanbanModel.View() → passes ticket + animFrame to TicketCardModel
                              ↓
TicketCardModel.Render() → draws bordered card with activity indicator + agent dot
```

Until the orchestrator exists, `AgentActive` defaults to `false` for all tickets. Cards render without the indicator.

## Scalability & Maintainability

1. **Theme-aware**: All styles use theme colors — works with all 7 built-in themes
2. **Width-adaptive**: Card content wraps/truncates to available width
3. **Testable**: `ActivityBar()` is a pure function (frame + width → string). `TicketCardModel.Render()` is deterministic given struct fields + frame. Table-driven tests for both.
4. **No coupling**: `ticketcard.go` imports only `store`, `lipgloss`, `theme` — no dependency on `kanban.go`, `app.go`, or orchestrator
5. **Extensible**: Adding new card fields (tags, dependencies) means adding lines to render methods
6. **Simple data flow**: Activity state comes from ticket field — no extra interfaces until orchestrator needs them
7. **Zero overhead when idle**: No tick commands sent when no agents are active

## Testing Strategy

- `ticketcard_test.go`: Table-driven tests for `ActivityBar()`, `RenderAgentDot()`, compact `Render()`, expanded `Render()`, edge cases (no agent, active agent, empty description, long title)
- `ticket_animation_test.go`: Test all 4 animation frames, width edge cases
- `kanban_test.go`: Update existing tests for new card-based rendering, test overflow calculation with multi-line cards, test animation tick logic
- `tickets_test.go`: Test `SetAgentActive`, verify migration adds column
