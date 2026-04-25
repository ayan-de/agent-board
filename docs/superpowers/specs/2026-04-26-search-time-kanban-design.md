# Search & Time-Span Kanban Design

## Overview

Add a Search tab and Date Filter tab to the Kanban board, replacing the current single Kanban view. The Search tab provides full-text search across ticket titles and descriptions. The Date Filter tab lets users navigate through monthly time spans anchored to the project's initialization date.

---

## Project Start Date

**Source**: `os.Stat()` on `.agentboard/<project>/` directory creation time, called on first project open.

**Config field**: `board.project_init_date` written automatically to `config.toml` on first project initialization. Field is read-only with an inline comment:

```toml
# Do not change — or you will lose progress
board.project_init_date = "2025-01-15"
```

**Month span definition**: always `15th of month X → 14th of month X+1`.

- Example: `Jan 15 - Feb 14 2025`
- Example: `Feb 15 - Mar 14 2025`

The first visible month starts at the project init date's 15th (or the 1st of that month if the 15th has passed). The last visible month ends at today + 14 days.

---

## Tab Bar

Below the header and above the Kanban board, a new tab bar with two tabs:

| Tab | Content |
|-----|---------|
| `Search` | Search bar + results Kanban (default active) |
| `Date Filter` | Time-spaned Kanban with month navigation |

**Keys**:
- `h/l` or `←/→`: move between tabs
- `Enter` or `Esc` (when not editing): activate selected tab content
- `j/k`: navigate cards within columns (same as Kanban)

---

## Search Tab

### Search Bar
- Always visible input field below the tab bar
- Placeholder text: `Search by title or description...`
- Debounce: 400ms after last keystroke before executing search
- Searches `title` AND `description` columns in SQLite

### Results View
- Same 4-column Kanban layout (Backlog / In Progress / Review / Done)
- Cards shown flat within columns — no month grouping
- When search query is empty: show centered empty state text `"Start typing to search tickets"`
- Month grouping is **not** shown in search results — tickets from all matching months appear in their status columns

### Empty State
- No matches: `"No tickets match your search"`
- Query empty: `"Start typing to search tickets"`

---

## Date Filter Tab

### Month Header
Format: `Jan 15 - Feb 14 2025 (3 cards)`

- Left-padded to column width
- Date span of the current month window
- Card count in parentheses — number of tickets with `created_at` in that month
- Empty months show `(0 cards)`

### Column Content
- Same 4-column Kanban layout
- Only tickets with `created_at` within the current month span are shown
- Each column shows "(empty)" when no tickets exist for that status in that month

### Month Navigation
- `←` arrow key: move to previous month (older)
- `→` arrow key: move to next month (newer)
- `←` at the oldest accessible month with no tickets: no-op
- `→` can advance indefinitely into future months (showing 0 cards)
- Month window is computed relative to project init date

---

## Architecture

### New Types

**`KanbanTab`** enum:
```go
type KanbanTab int
const (
    TabSearch KanbanTab = iota
    TabDateFilter
)
```

**`TimeFilterModel`** (new model in `kanban.go`):
- `tab KanbanTab` — active tab
- `searchQuery string` — current search string
- `monthOffset int` — months relative to project init date (0 = first month)
- `projectInitDate time.Time` — set once from config

**`TicketFilters`** — extended:
```go
type TicketFilters struct {
    Status   string
    Agent    string
    Priority string
    Tag      string
    From     *time.Time // NEW
    To       *time.Time // NEW
}
```

**Store `ListTickets`** — extended SQL with optional `WHERE created_at BETWEEN ? AND ?`.

### Config Changes
`internal/config/config.go`:
- Add `ProjectInitDate time.Time` to `BoardConfig`
- Write automatically on first project load
- Load from config on subsequent loads

### Message Types
- `searchQueryMsg{query string}` — debounced search input
- `monthNavigateMsg{direction int}` — `←` = -1, `→` = +1
- `tabChangeMsg{tab KanbanTab}`

---

## Implementation Notes

- Search debounce uses a ticker/timer approach in `Update()` — cancel previous timer on new input
- Month window calculation is a pure function: `MonthWindow(projectInit, offset) (from, to time.Time)`
- Date filtering happens server-side in SQLite — `created_at` column is already indexed
- When switching tabs, preserve state (search query persists, month offset persists)
- The "Search" and "Date Filter" tabs are part of the Kanban view — not separate views — to share state and reduce complexity
