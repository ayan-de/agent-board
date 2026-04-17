package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func testNotificationStyles() NotificationStyles {
	return NotificationStyles{
		InfoBorder:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
		SuccessBorder: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
		WarningBorder: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
		ErrorBorder:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
		Title:         lipgloss.NewStyle().Bold(true),
		Message:       lipgloss.NewStyle(),
	}
}

func newTestNotification() NotificationStack {
	n := NotificationStack{}
	n.SetSize(100, 30)
	n.styles = testNotificationStyles()
	return n
}

func TestNotificationInactiveByDefault(t *testing.T) {
	n := newTestNotification()
	if n.Active() {
		t.Fatal("notification should be inactive by default")
	}
}

func TestNotificationShowAppendsItemAndReturnsDismissCmd(t *testing.T) {
	n := newTestNotification()

	cmd := n.Show("Ticket created", "AGT-001 created", NotificationSuccess, 25*time.Millisecond)

	if !n.Active() {
		t.Fatal("notification should be active after show")
	}
	if len(n.items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(n.items))
	}
	if n.items[0].title != "Ticket created" {
		t.Fatalf("title = %q, want %q", n.items[0].title, "Ticket created")
	}
	if n.items[0].message != "AGT-001 created" {
		t.Fatalf("message = %q, want %q", n.items[0].message, "AGT-001 created")
	}
	if n.items[0].variant != NotificationSuccess {
		t.Fatalf("variant = %v, want %v", n.items[0].variant, NotificationSuccess)
	}
	if cmd == nil {
		t.Fatal("Show should return a dismiss command")
	}

	msg := cmd()
	dismiss, ok := msg.(notificationDismissMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want notificationDismissMsg", msg)
	}
	if dismiss.id != n.items[0].id {
		t.Fatalf("dismiss id = %d, want %d", dismiss.id, n.items[0].id)
	}
}

func TestNotificationDismissMessageRemovesMatchingNotification(t *testing.T) {
	n := newTestNotification()
	_ = n.Show("Saved", "Ticket updated", NotificationInfo, time.Second)
	_ = n.Show("Assigned", "Agent updated", NotificationSuccess, time.Second)
	firstID := n.items[0].id
	secondID := n.items[1].id

	n = n.HandleDismiss(notificationDismissMsg{id: firstID})

	if !n.Active() {
		t.Fatal("stack should remain active when one of multiple notifications is dismissed")
	}
	if len(n.items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(n.items))
	}
	if n.items[0].id != secondID {
		t.Fatalf("remaining id = %d, want %d", n.items[0].id, secondID)
	}
}

func TestNotificationDismissIgnoresStaleMessage(t *testing.T) {
	n := newTestNotification()
	_ = n.Show("First", "Old message", NotificationInfo, time.Second)
	staleID := n.items[0].id
	_ = n.Show("Second", "New message", NotificationSuccess, time.Second)

	n = n.HandleDismiss(notificationDismissMsg{id: staleID})

	if !n.Active() {
		t.Fatal("stack should remain active after one notification is dismissed")
	}
	if len(n.items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(n.items))
	}
	if n.items[0].title != "Second" {
		t.Fatalf("title = %q, want %q", n.items[0].title, "Second")
	}
}

func TestNotificationViewContainsAllVisibleItems(t *testing.T) {
	n := newTestNotification()
	_ = n.Show("Agent assigned", "codex assigned to AGT-001", NotificationSuccess, time.Second)
	_ = n.Show("Ticket created", "AGT-002 created", NotificationInfo, time.Second)

	view := n.View()
	if !strings.Contains(view, "Agent assigned") {
		t.Fatal("view missing first title")
	}
	if !strings.Contains(view, "codex assigned to AGT-001") {
		t.Fatal("view missing first message")
	}
	if !strings.Contains(view, "Ticket created") {
		t.Fatal("view missing second title")
	}
}

func TestNotificationViewEmptyWhenInactive(t *testing.T) {
	n := newTestNotification()
	if n.View() != "" {
		t.Fatal("inactive notification view should be empty")
	}
}

func TestNotificationDoesNotHandleKeyboardInput(t *testing.T) {
	n := newTestNotification()
	_ = n.Show("Saved", "Ticket updated", NotificationInfo, time.Second)

	n, cmd := n.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !n.Active() {
		t.Fatal("notification should ignore keyboard input")
	}
	if cmd != nil {
		t.Fatal("notification should not return commands for keyboard input")
	}
}

func TestNotificationShowKeepsNewestFourItems(t *testing.T) {
	n := newTestNotification()

	for i := 0; i < 5; i++ {
		_ = n.Show(
			string(rune('A'+i)),
			"message",
			NotificationInfo,
			time.Second,
		)
	}

	if len(n.items) != 4 {
		t.Fatalf("len(items) = %d, want 4", len(n.items))
	}
	if n.items[0].title != "B" {
		t.Fatalf("oldest title = %q, want %q", n.items[0].title, "B")
	}
	if n.items[3].title != "E" {
		t.Fatalf("newest title = %q, want %q", n.items[3].title, "E")
	}
}
