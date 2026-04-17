# Notification Overlay Design

## Goal

Add a simple non-blocking notification overlay to the TUI for short success/info feedback such as ticket creation and agent assignment.

## Decision

Keep the existing blocking `ConfirmModal` for destructive confirmation flows and introduce a separate notification-specific component for auto-dismiss behavior.

## Behavior

- top-right overlay
- non-blocking while visible
- no keyboard interaction
- auto-dismiss only
- reopening replaces the current notification
- stale dismiss timers must not close a newer notification

## Integration

- `KanbanModel` emits a ticket-created message after successful ticket creation
- `TicketViewModel` emits an agent-assigned message after successful selection
- `App` translates those events into notification visibility and renders the overlay above the active view

## Testing

- notification lifecycle unit tests
- stale-dismiss protection tests
- app integration tests for ticket creation and agent assignment
