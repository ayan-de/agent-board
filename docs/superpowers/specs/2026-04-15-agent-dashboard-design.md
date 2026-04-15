# Agent Dashboard Overlay — Design Spec

## Summary

Full-screen overlay showing installed CLI agents, their status, and operational metrics. Opened with `i`, closed with `Esc`, refreshed with `r`.

## Keybinding

| Key | Action |
|-----|--------|
| `i` | Open/close dashboard overlay |
| `r` | Refresh agent detection |
| `Esc` | Close dashboard, return to previous view |

New action: `ActionShowDashboard` added to `keybinding/action.go`.

## Layout

```
╭──────────────────────────────────────────────────────────────╮
│  Agent Dashboard                                    r: refresh │
│──────────────────────────────────────────────────────────────│
│                                                              │
│  ┌─────────────────────┐  ┌─────────────────────┐            │
│  │  claude-code        │  │  opencode            │            │
│  │  Status: installed  │  │  Status: installed   │            │
│  │  Running: no        │  │  Running: no         │            │
│  │  Ticket: —          │  │  Ticket: —           │            │
│  │  Uptime: —          │  │  Uptime: —           │            │
│  │  Subagents: —       │  │  Subagents: —        │            │
│  │  Tokens: —          │  │  Tokens: —           │            │
│  └─────────────────────┘  └─────────────────────┘            │
│                                                              │
│  ┌─────────────────────┐                                     │
│  │  cursor             │                                     │
│  │  Status: not found  │                                     │
│  │  ...                │                                     │
│  └─────────────────────┘                                     │
│                                                              │
│  Press Esc to return                                         │
╰──────────────────────────────────────────────────────────────╯
```

Agent cards arranged horizontally, wrapping to next row when width exceeded.

## Components

### 1. `internal/config/detection.go` — Agent Detection

Scans `$PATH` for agent binaries at startup and on refresh.

```go
type DetectedAgent struct {
    Name     string    // e.g. "claude-code", "opencode", "cursor"
    Binary   string    // e.g. "claude", "opencode", "cursor"
    Path     string    // full path if found, "" otherwise
    Found    bool      // true if binary exists on $PATH
}
```

Function: `DetectAgents() []DetectedAgent` — checks for `claude`, `opencode`, `cursor` on `$PATH` using `exec.LookPath`.

### 2. `internal/tui/dashboard.go` — DashboardModel

Follows the `KanbanModel`/`TicketViewModel` pattern: value-receiver `Update`, lipgloss styles, string builder rendering.

```go
type DashboardModel struct {
    store    *store.Store
    resolver *keybinding.Resolver
    agents   []DetectedAgent
    width    int
    height   int
    styles   DashboardStyles
}
```

Fields per card:
- **Status**: `installed` (found) or `not found`
- **Running**: `yes` / `no` — always `no` until Phase 3
- **Ticket**: assigned ticket ID or `—`
- **Uptime**: `—` placeholder until Phase 3
- **Subagents**: `—` placeholder until Phase 3
- **Tokens**: `—` placeholder until Phase 3

Methods:
- `NewDashboardModel(s, resolver, agents) DashboardModel`
- `Init() tea.Cmd`
- `Update(msg) (DashboardModel, tea.Cmd)` — handles WindowSizeMsg, refresh
- `View() string` — renders full overlay
- `Refresh() DashboardModel` — re-runs DetectAgents

### 3. `internal/tui/app.go` — Integration

- Add `viewDashboard` to `viewMode` enum
- Add `dashboard DashboardModel` field to `App`
- Construct in `NewApp()` with detected agents
- Route `i` key to toggle dashboard view
- Route keys to `dashboard.Update()` when in `viewDashboard`
- Add window size propagation to dashboard

### 4. `internal/keybinding/action.go` — New Action

Add `ActionShowDashboard` constant. Map to key `i` in `DefaultKeyMap()`.

## File Changes

| File | Change |
|------|--------|
| `internal/config/detection.go` | New file — `DetectAgents()` function |
| `internal/config/detection_test.go` | New file — tests for agent detection |
| `internal/tui/dashboard.go` | New file — `DashboardModel` |
| `internal/tui/dashboard_test.go` | New file — tests for dashboard |
| `internal/tui/app.go` | Add `viewDashboard`, `dashboard` field, key routing |
| `internal/tui/app_test.go` | Add dashboard view tests |
| `internal/keybinding/action.go` | Add `ActionShowDashboard` |
| `internal/keybinding/keymap.go` | Map `i` to `ActionShowDashboard` |
| `internal/keybinding/action_test.go` | Add test for new action string |

## Phase 3 Extension Points

The dashboard is designed to accept live data when Phase 3 lands:
- `DetectedAgent` gains `Running bool`, `StartedAt time.Time`, `TicketID string`, `SubagentCount int`, `TokenUsage int64`
- Card rendering switches from `—` to live values
- `r` refresh becomes more meaningful — re-polls agent process state

## Test Plan

1. Agent detection: test with known/missing binaries, verify `Found` flag
2. Dashboard construction: verify model fields initialized
3. View rendering: verify agent names, status labels appear in output
4. Key routing: verify `i` toggles dashboard view in App
5. Refresh: verify `r` triggers re-detection
6. Integration: verify `Esc` returns to board from dashboard
