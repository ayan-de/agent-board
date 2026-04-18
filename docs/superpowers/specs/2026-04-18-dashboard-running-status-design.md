# Dashboard Running Status

## Problem

The agent dashboard shows static placeholder values for `Running:`, `Ticket:`, and `Uptime:` on every agent card. When an orchestrator run starts a worker subprocess for a ticket, the dashboard should reflect that the agent is actively running and show the associated ticket ID and uptime.

## Scope

- Show running state on agent dashboard cards based on active sessions (sessions where `ended_at IS NULL`)
- Refresh running state when entering the dashboard view or pressing the refresh key
- Only consider agents with active sessions as "running" — agents assigned to in-progress tickets but not yet started are not shown as running

## Out of Scope

- Real-time push updates (event-driven) while the dashboard is visible
- Showing ticket title (only ticket ID)
- Subagent count or token tracking

## Architecture

### Data flow

```
Dashboard View() / Refresh()
  -> store.ListActiveSessions(ctx)
  -> map[binaryName]Session
  -> renderCard checks map, populates Running/Ticket/Uptime
```

### Agent name matching

Sessions store `agent` as the binary name (e.g. "opencode"). `DetectedAgent.Binary` holds the same value. Matching is `DetectedAgent.Binary == session.Agent`.

## Changes

### `internal/store/sessions.go`

Add `ListActiveSessions(ctx context.Context) ([]Session, error)` that queries sessions where `ended_at IS NULL`, ordered by `started_at ASC`.

### `internal/tui/dashboard.go`

1. Add `activeSessions map[string]store.Session` field to `DashboardModel`.
2. Add `loadActiveSessions()` method that queries the store and builds the map keyed by agent binary name.
3. Call `loadActiveSessions()` in `View()` and `Refresh()`.
4. In `renderCard()`, look up the agent's active session. If found:
   - `Running:` shows "yes" (colored with agent logo color)
   - `Ticket:` shows the session's ticket ID
   - `Uptime:` shows a human-readable duration since `session.StartedAt`
5. If not found, show current defaults ("no", "—", "—").

### `internal/tui/app.go`

When entering dashboard view via `ActionShowDashboard`, trigger `dashboard.Refresh()` so session data is fresh.

## Rendering

### Card fields when running

```
Status:   installed
Running:  yes
Ticket:   AB-01
Uptime:   2m 30s
```

### Card fields when not running

```
Status:   installed
Running:  no
Ticket:   —
Uptime:   —
```

### Uptime formatting

- Under 1 minute: `<seconds>s` (e.g. `45s`)
- Under 1 hour: `<minutes>m <seconds>s` (e.g. `2m 30s`)
- Over 1 hour: `<hours>h <minutes>m` (e.g. `1h 23m`)

## Testing

- `store/sessions_test.go`: Test `ListActiveSessions` returns only active sessions
- `tui/dashboard_test.go`: Test card rendering with and without active sessions
- Verify "Running: yes" and ticket ID appear when session exists
- Verify "Running: no" when no active session for agent
- Verify uptime formatting at various durations
