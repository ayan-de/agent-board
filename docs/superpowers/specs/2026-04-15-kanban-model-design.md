# KanbanModel Design — Step 1.5

## Summary

Extract the 4-column Kanban board from `app.go` into a standalone `KanbanModel` in `kanban.go`. Replace the plain-text `renderBoard()` with lipgloss-styled bordered columns following the nap project's focus/blur pattern.

## Approach

**Approach 1: Standalone `tea.Model`** — KanbanModel implements `tea.Model` (Init/Update/View). App holds a `KanbanModel` field and delegates relevant messages to it. This follows the composition pattern used in the nap project where each pane is its own model.

## Data Model

```go
type KanbanModel struct {
    store    *store.Store
    resolver *keybinding.Resolver
    width    int
    height   int

    colIndex int
    cursors  [4]int
    columns  [4][]store.Ticket

    styles   KanbanStyles
}

type KanbanStyles struct {
    FocusedColumn  lipgloss.Style
    BlurredColumn  lipgloss.Style
    FocusedTitle   lipgloss.Style
    BlurredTitle   lipgloss.Style
    SelectedTicket lipgloss.Style
    Ticket         lipgloss.Style
    EmptyColumn    lipgloss.Style
}
```

All column state (`columns`, `colIndex`, `cursors`) moves from `App` to `KanbanModel`. `KanbanStyles` follows the nap focused/blurred pattern — focused column gets a bright border, blurred columns get a dim border.

`DefaultKanbanStyles()` returns the built-in style set. The theme system (Phase 2) will override these later.

## API

```go
func NewKanbanModel(store *store.Store, resolver *keybinding.Resolver) (KanbanModel, error)
func (m KanbanModel) Init() tea.Cmd
func (m KanbanModel) Update(msg tea.Msg) (KanbanModel, tea.Cmd)
func (m KanbanModel) View() string
func (m KanbanModel) SelectedTicket() *store.Ticket
func (m KanbanModel) Reload() error
```

### Update handles:

- `tea.WindowSizeMsg` — stores width/height, recalculates column widths
- `tea.KeyMsg` — resolves key via resolver, handles: `PrevColumn`, `NextColumn`, `PrevTicket`, `NextTicket`, `JumpColumn1–4`, `AddTicket`, `DeleteTicket`
- Returns unhandled messages unchanged (App handles `Quit`, `OpenTicket`, `ShowHelp`)

### App changes:

- `App.kanban KanbanModel` replaces `columns`, `colIndex`, `cursors` fields
- `App.Update()` forwards `WindowSizeMsg` and key messages to `kanban.Update(msg)`
- `App.renderBoard()` becomes `m.kanban.View()`
- `App.OpenTicket` reads `m.kanban.SelectedTicket()` to get the active ticket
- `App` keeps: `view`, `focus`, `activeTicket`, resolver, config
- `AddTicket` and `DeleteTicket` move entirely into KanbanModel

## Rendering

```
┌─ Backlog ──────┐  ┌─ In Progress ─┐  ┌─── Review ────┐  ┌──── Done ─────┐
│ ▸ AGT-01 First │  │                │  │                │  │                │
│   AGT-02 Second│  │   (empty)      │  │   (empty)      │  │   (empty)      │
│                │  │                │  │                │  │                │
└────────────────┘  └────────────────┘  └────────────────┘  └────────────────┘
```

### Layout algorithm:

1. `colWidth = (terminalWidth - 1) / 4` — distribute evenly
2. Each column is a `lipgloss.Style` with `Border(true, false, true, false)` (top and bottom borders), `Width(colWidth - 2)` (subtract border chars)
3. Focused column: border uses bright color (e.g., `lipgloss.Color("69")` blue)
4. Blurred columns: border uses dim color (e.g., `lipgloss.Color("240")` gray)
5. Column title rendered inside the top border via styled border title

### Ticket rendering per column:

- Selected ticket in focused column: `▸ AGT-01 Title` with bold foreground
- Unselected tickets: `  AGT-02 Title` with normal foreground
- Empty column: `  (empty)` in dim color
- Long titles truncated with `…` at column width

### Height management:

- `availableHeight = height - 4` (subtract title bar + help hint at bottom)
- Each column body shows up to `availableHeight` tickets
- If more tickets than fit, show a scroll indicator `↓ N more` at the bottom

## Testing Strategy

### `kanban_test.go` — kanban-specific tests:

| Test | What it covers |
|------|---------------|
| `TestNewKanbanModel` | Constructor, resolver set, columns loaded |
| `TestKanbanWindowSize` | Width/height stored |
| `TestKanbanColumnNavigation` | PrevColumn, NextColumn, JumpColumn1–4, clamping |
| `TestKanbanTicketNavigation` | PrevTicket, NextTicket, clamp at bounds, empty column |
| `TestKanbanAddTicket` | Creates ticket in backlog, reloads columns |
| `TestKanbanDeleteTicket` | Deletes selected ticket, reloads, empty column no-op |
| `TestKanbanSelectedTicket` | Returns correct ticket, nil on empty column |
| `TestKanbanReload` | Manual reload picks up external store changes |
| `TestKanbanViewRendersColumns` | View output contains all 4 column names |
| `TestKanbanViewRendersTickets` | View output contains ticket IDs and titles |
| `TestKanbanViewFocusedColumn` | Focused column has different styling from blurred |
| `TestKanbanViewTruncation` | Long titles get truncated with `…` |
| `TestKanbanViewEmptyColumn` | Empty columns show `(empty)` |

### `app_test.go` — app-level tests (remaining after extraction):

| Test | What it covers |
|------|---------------|
| `TestNewApp` | Constructor, kanban wired, resolver set |
| `TestAppQuit` | Quit/force quit |
| `TestAppShowHelp` | Help toggle |
| `TestAppOpenTicket` | Delegates to kanban.SelectedTicket() |
| `TestAppEscapeReturnsToBoard` | View switching |
| `TestAppViewRouting` | Board/ticket/help view dispatch |

### Test helper:

`newTestKanban(t)` creates a KanbanModel with temp DB, similar to existing `newTestApp(t)`.

## Files Modified

| File | Change |
|------|--------|
| `internal/tui/kanban.go` | KanbanModel, KanbanStyles, DefaultKanbanStyles, full implementation |
| `internal/tui/kanban_test.go` | New file — all kanban-specific tests |
| `internal/tui/app.go` | Remove columns/colIndex/cursors, add kanban field, delegate to KanbanModel |
| `internal/tui/app_test.go` | Remove kanban-specific tests, keep app-level tests |
| `go.mod` | Add `github.com/charmbracelet/lipgloss` as direct dependency |
