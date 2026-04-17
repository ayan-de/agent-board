# Notification Overlay Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a non-blocking, auto-dismissing notification overlay for short TUI feedback such as ticket creation and agent assignment.

**Architecture:** Keep the existing blocking `ConfirmModal` unchanged and introduce a separate `NotificationModal` with its own app-level lifecycle. Emit lightweight Bubble Tea messages from ticket workflows, let `App` translate those into visible notifications, and protect against stale auto-dismiss timers with notification ids.

**Tech Stack:** Go, Bubble Tea, Lip Gloss, Go testing

---

### Task 1: Define notification behavior with tests

**Files:**
- Create: `internal/tui/notification_test.go`
- Create: `internal/tui/notification_app_test.go`
- Test: `internal/tui/notification_test.go`
- Test: `internal/tui/notification_app_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestNotificationShowSetsFieldsAndReturnsDismissCmd(t *testing.T) {
	n := newTestNotification()

	cmd := n.Show("Ticket created", "AGT-001 created", NotificationSuccess, 25*time.Millisecond)

	if !n.Active() {
		t.Fatal("notification should be active after show")
	}
	if cmd == nil {
		t.Fatal("Show should return a dismiss command")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `env GOCACHE=/tmp/agentboard-gocache go test ./internal/tui -run 'TestNotification|TestAppTicketCreate|TestAppAgentAssign'`
Expected: FAIL with undefined notification types and missing app notification fields.

- [ ] **Step 3: Expand tests to cover app flow**

```go
func TestAppTicketCreateShowsNotification(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	_, dismissCmd := app.Update(cmd())

	if dismissCmd == nil || !app.notification.Active() {
		t.Fatal("ticket created event should activate notification")
	}
}
```

- [ ] **Step 4: Run test to verify it still fails for missing implementation**

Run: `env GOCACHE=/tmp/agentboard-gocache go test ./internal/tui -run 'TestNotification|TestAppTicketCreate|TestAppAgentAssign'`
Expected: FAIL until `NotificationModal`, dismiss handling, and event hooks exist.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/notification_test.go internal/tui/notification_app_test.go
git commit -m "test: add notification overlay coverage"
```

### Task 2: Implement the notification modal and app lifecycle

**Files:**
- Create: `internal/tui/notification.go`
- Modify: `internal/tui/app.go`
- Test: `internal/tui/notification_test.go`
- Test: `internal/tui/notification_app_test.go`

- [ ] **Step 1: Write the minimal implementation**

```go
type NotificationModal struct {
	active  bool
	id      int
	title   string
	message string
	variant NotificationVariant
}

func (n *NotificationModal) Show(title, message string, variant NotificationVariant, duration time.Duration) tea.Cmd {
	n.id++
	n.active = true
	n.title = title
	n.message = message

	id := n.id
	return tea.Tick(duration, func(time.Time) tea.Msg {
		return notificationDismissMsg{id: id}
	})
}
```

- [ ] **Step 2: Wire the app to render and dismiss notifications**

```go
case notificationDismissMsg:
	a.notification = a.notification.HandleDismiss(msg)
	return a, nil
```

- [ ] **Step 3: Run focused tests to verify they pass**

Run: `env GOCACHE=/tmp/agentboard-gocache go test ./internal/tui -run 'TestNotification|TestAppTicketCreate|TestAppAgentAssign'`
Expected: PASS

- [ ] **Step 4: Refine overlay rendering**

```go
startY := 1
startX := a.width - notificationWidth - 2
finalView.WriteString(overlayLine(bgLine, notificationLine, startX))
```

- [ ] **Step 5: Commit**

```bash
git add internal/tui/notification.go internal/tui/app.go
git commit -m "feat: add notification overlay"
```

### Task 3: Emit notifications from ticket workflows and verify project-wide

**Files:**
- Modify: `internal/tui/kanban.go`
- Modify: `internal/tui/ticketview.go`
- Modify: `AGENTS.md`
- Test: `internal/tui/notification_app_test.go`

- [ ] **Step 1: Return app events from ticket creation and agent assignment**

```go
return m, func() tea.Msg {
	return ticketCreatedMsg{id: ticket.ID, title: ticket.Title}
}
```

```go
return m, func() tea.Msg {
	return agentAssignedMsg{ticketID: ticketID, agent: agent}
}
```

- [ ] **Step 2: Document the new runtime behavior**

```md
- non-blocking auto-dismissing notification overlay for ticket and agent workflow feedback
```

- [ ] **Step 3: Run the full verification suite**

Run: `env GOCACHE=/tmp/agentboard-gocache go test ./...`
Expected: PASS

Run: `env GOCACHE=/tmp/agentboard-gocache go vet ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/tui/kanban.go internal/tui/ticketview.go AGENTS.md
git commit -m "docs: record notification overlay behavior"
```
