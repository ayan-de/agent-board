# Design: tui/app.go — Root Bubbletea Model (Phase 1.4)

## Goal

Implement the root bubbletea model that tracks window size, resolves keybindings to actions via the existing `keybinding.Resolver`, and routes actions to navigation, CRUD, and view-state changes. The Store is wired in so ticket mutations work immediately.

## Approach

Flat model — single `App` struct implementing `bubbletea.Model`. Sub-models (kanban, ticketview) extract naturally in phases 1.5 and 1.6.

## App Struct

```go
type focusArea int
const (focusBoard focusArea = iota; focusAgentPane)

type viewMode int
const (viewBoard viewMode = iota; viewTicket; viewHelp)

type App struct {
    store    *store.Store
    resolver *keybinding.Resolver
    config   *config.Config
    width    int
    height   int

    focus      focusArea
    view       viewMode

    colIndex   int
    cursors    [4]int
    columns    [4][]store.Ticket

    activeTicket *store.Ticket
}
```

## Constructor

`NewApp(cfg *config.Config, s *store.Store) *App`

- Builds `Resolver` from `DefaultKeyMap()` + config keybinding overrides
- Calls `loadColumns()` to populate ticket data from store
- Returns initialized App ready for `tea.Program`

## bubbletea.Model Methods

### Init()

Returns `tea.EnterAltScreen`. No I/O on startup.

### Update(msg tea.Msg) (tea.Model, tea.Cmd)

**WindowSizeMsg**: Store width and height. No re-layout command needed — View() uses current dimensions.

**KeyMsg**: Convert to key string, call `resolver.Resolve(key)`, switch on resulting `Action`:

| Action | Behavior |
|--------|----------|
| ActionPrevColumn / ActionNextColumn | Move `colIndex`, clamp 0–3 |
| ActionPrevTicket / ActionNextTicket | Move cursor in current column, clamp 0–len-1 |
| ActionJumpColumn1–4 | Set `colIndex` directly |
| ActionOpenTicket | Load ticket at cursor into `activeTicket`, set `view = viewTicket` |
| ActionAddTicket | Create ticket in backlog via `store.CreateTicket`, reload columns |
| ActionDeleteTicket | Delete ticket at cursor via `store.DeleteTicket`, reload columns |
| ActionShowHelp | Toggle `viewHelp` |
| ActionQuit / ActionForceQuit | Return `tea.Quit` |
| Others (StartAgent, StopAgent, etc.) | No-op stubs for later phases |

On Escape or when `viewTicket`/`viewHelp` is active and a navigation action fires, return to `viewBoard`.

### View() string

Renders based on `viewMode`:

- **viewBoard**: Header (app name, dimensions), 4 columns side by side showing status name + ticket list (ID: Title), cursor highlight on active column/ticket.
- **viewTicket**: Ticket detail panel (ID, title, description, status, agent).
- **viewHelp**: Overlay with keybinding list from the current KeyMap.

Minimal styling for 1.4 — plain text with ANSI bold/underline for highlights. Phase 2 adds theme support.

## Data Loading

`loadColumns()` — calls `store.ListTickets` with status filter for each of the 4 statuses. Populates `columns` array. Called in constructor and after any mutation (add, delete, status change).

## New Dependency

`github.com/charmbracelet/bubbletea` — must be added to go.mod.

## Testing

All tests use table-driven patterns with a real in-memory SQLite store (temp dir, same as store tests).

### Test Cases

| Test | What it verifies |
|------|-----------------|
| TestNewApp | Constructor returns non-nil App with resolver wired |
| TestInit | Returns EnterAltScreen command |
| TestUpdateWindowSize | Stores width and height from tea.WindowSizeMsg |
| TestUpdateNavigation | Column and ticket cursor movement with bounds clamping |
| TestUpdateQuit | Returns tea.Quit |
| TestUpdateForceQuit | Returns tea.Quit |
| TestUpdateAddTicket | Ticket appears in backlog column after action |
| TestUpdateDeleteTicket | Ticket removed from column after action |
| TestUpdateOpenTicket | Switches to viewTicket, activeTicket set |
| TestUpdateShowHelp | Toggles help overlay view |
| TestViewRendersColumns | View output contains all 4 column names |
| TestViewRendersTickets | View output contains ticket IDs |

## Scope Boundaries

- 1.4 does NOT implement: agent spawning, PTY panes, slash commands, themes, HTTP API
- 1.4 does NOT use: lipgloss styling (plain text only), bubbles components
- Navigation and CRUD are inline in Update() — extracted to sub-models in 1.5/1.6

## File Changes

| File | Change |
|------|--------|
| `internal/tui/app.go` | Replace stub with full implementation |
| `internal/tui/app_test.go` | New test file |
| `go.mod` | Add `github.com/charmbracelet/bubbletea` |
