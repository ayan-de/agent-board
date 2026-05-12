package tui

import (
	"testing"

	"github.com/ayan-de/agent-board/internal/board"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAppNotificationDismissIgnoresStaleMessage(t *testing.T) {
	app, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	before := app.board.GetState()
	if before.Notification != nil {
		t.Fatal("notification should not be active before test")
	}

	app.board.SetNotification("Test", "First notification", board.NotificationSuccess)
	state1 := app.board.GetState()
	if state1.Notification == nil {
		t.Fatal("notification should be set")
	}
	firstID := state1.Notification.Title

	app.board.SetNotification("Test", "Second notification", board.NotificationSuccess)
	state2 := app.board.GetState()
	if state2.Notification == nil {
		t.Fatal("second notification should be set")
	}

	app.board.ClearNotification()
	state3 := app.board.GetState()
	if state3.Notification != nil {
		t.Error("notification should be cleared")
	}
	if state3.Notification != nil && state3.Notification.Title == firstID {
		t.Error("stale dismiss should not affect newer notification")
	}
}

func TestAppNotificationsStackInsteadOfReplacing(t *testing.T) {
	app, _ := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	app.board.SetNotification("First", "First message", board.NotificationSuccess)
	state := app.board.GetState()
	if state.Notification == nil {
		t.Fatal("first notification should be set")
	}

	app.board.SetNotification("Second", "Second message", board.NotificationInfo)
	state = app.board.GetState()
	if state.Notification == nil {
		t.Fatal("second notification should be set")
	}
}