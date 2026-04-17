package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/ayan-de/agent-board/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func TestAppTicketCreateShowsNotification(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatal("ticket create should return ticket created message command")
	}

	_, dismissCmd := app.Update(cmd())
	if dismissCmd == nil {
		t.Fatal("ticket created message should return notification dismiss command")
	}
	if !app.notification.Active() {
		t.Fatal("notification should be active after ticket create")
	}
	if len(app.notification.items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(app.notification.items))
	}
	if app.notification.items[0].title != "Ticket created" {
		t.Fatalf("title = %q, want %q", app.notification.items[0].title, "Ticket created")
	}
	if !strings.Contains(app.notification.items[0].message, "New Ticket") {
		t.Fatalf("message = %q, want to mention new ticket", app.notification.items[0].message)
	}
}

func TestAppTicketCreateNotificationDoesNotBlockNavigation(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatal("ticket create should return ticket created message command")
	}
	app.Update(cmd())
	if !app.notification.Active() {
		t.Fatal("notification should be active after ticket create")
	}

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if app.kanban.colIndex != 1 {
		t.Fatalf("colIndex = %d, want 1", app.kanban.colIndex)
	}
}

func TestAppNotificationDismissIgnoresStaleMessage(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	_, firstCmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if firstCmd == nil {
		t.Fatal("first ticket create should return ticket created message command")
	}
	_, dismissCmd := app.Update(firstCmd())
	if dismissCmd == nil {
		t.Fatal("ticket created message should return dismiss command")
	}
	firstID := app.notification.items[0].id

	_, secondCmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if secondCmd == nil {
		t.Fatal("second ticket create should return ticket created message command")
	}
	app.Update(secondCmd())
	secondID := app.notification.items[1].id

	app.Update(notificationDismissMsg{id: firstID})

	if !app.notification.Active() {
		t.Fatal("stale dismiss should not clear newer notification")
	}
	if len(app.notification.items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(app.notification.items))
	}
	if app.notification.items[0].id != secondID {
		t.Fatalf("notification id = %d, want %d", app.notification.items[0].id, secondID)
	}
}

func TestAppAgentAssignShowsNotification(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	ctx := context.Background()

	ticket, err := app.store.CreateTicket(ctx, store.Ticket{Title: "Assign Me", Status: "backlog"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	app.kanban, _ = app.kanban.Reload()
	app.activeTicket = &ticket
	app.ticketView = app.ticketView.SetTicket(&ticket)
	app.view = viewTicket

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("agent assign should return agent assigned message command")
	}
	_, dismissCmd := app.Update(cmd())
	if dismissCmd == nil {
		t.Fatal("agent assigned message should return notification dismiss command")
	}
	if !app.notification.Active() {
		t.Fatal("notification should be active after agent assign")
	}
	if len(app.notification.items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(app.notification.items))
	}
	if app.notification.items[0].title != "Agent assignment updated" {
		t.Fatalf("title = %q, want %q", app.notification.items[0].title, "Agent assignment updated")
	}
	if !strings.Contains(app.notification.items[0].message, ticket.ID) {
		t.Fatalf("message = %q, want to mention ticket id", app.notification.items[0].message)
	}
}

func TestAppNotificationsStackInsteadOfReplacing(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	_, firstCmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if firstCmd == nil {
		t.Fatal("first ticket create should return ticket created message command")
	}
	app.Update(firstCmd())

	_, secondCmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if secondCmd == nil {
		t.Fatal("second ticket create should return ticket created message command")
	}
	app.Update(secondCmd())

	if len(app.notification.items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(app.notification.items))
	}
	if app.notification.items[0].id == app.notification.items[1].id {
		t.Fatal("stacked notifications should have distinct ids")
	}
}
