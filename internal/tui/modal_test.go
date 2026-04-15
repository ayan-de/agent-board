package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func testModalTheme() *ConfirmModalStyles {
	return &ConfirmModalStyles{
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2),
		Title:     lipgloss.NewStyle().Bold(true),
		Message:   lipgloss.NewStyle(),
		Confirm:   lipgloss.NewStyle(),
		Cancel:    lipgloss.NewStyle(),
		Highlight: lipgloss.NewStyle().Bold(true),
	}
}

func newTestModal() ConfirmModal {
	m := ConfirmModal{}
	m.SetSize(80, 24)
	m.styles = *testModalTheme()
	return m
}

func TestConfirmModalNotActiveByDefault(t *testing.T) {
	m := newTestModal()
	if m.Active() {
		t.Error("modal should not be active by default")
	}
}

func TestConfirmModalOpenSetsActive(t *testing.T) {
	m := newTestModal()
	m.Open("Title", "Message", func() tea.Cmd { return tea.Quit }, nil)
	if !m.Active() {
		t.Error("modal should be active after Open")
	}
	if m.title != "Title" {
		t.Errorf("title = %q, want %q", m.title, "Title")
	}
	if m.message != "Message" {
		t.Errorf("message = %q, want %q", m.message, "Message")
	}
}

func TestConfirmModalDefaultsToNo(t *testing.T) {
	m := newTestModal()
	m.Open("T", "M", func() tea.Cmd { return tea.Quit }, nil)
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (No)", m.cursor)
	}
}

func TestConfirmModalEnterNoCancels(t *testing.T) {
	m := newTestModal()
	cancelled := false
	m.Open("T", "M", func() tea.Cmd { return tea.Quit }, func() { cancelled = true })

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("should not return command when pressing No")
	}
	if m.Active() {
		t.Error("modal should be closed after enter")
	}
	if !cancelled {
		t.Error("onCancel should have been called")
	}
}

func TestConfirmModalEnterYesConfirms(t *testing.T) {
	m := newTestModal()
	m.Open("T", "M", func() tea.Cmd { return tea.Quit }, nil)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("should return tea.Quit command when pressing Yes")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("cmd produced %T, want tea.QuitMsg", msg)
	}
	if m.Active() {
		t.Error("modal should be closed after confirm")
	}
}

func TestConfirmModalEscapeCancels(t *testing.T) {
	m := newTestModal()
	cancelled := false
	m.Open("T", "M", func() tea.Cmd { return tea.Quit }, func() { cancelled = true })

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd != nil {
		t.Error("escape should not produce command")
	}
	if m.Active() {
		t.Error("modal should be closed after escape")
	}
	if !cancelled {
		t.Error("onCancel should have been called")
	}
}

func TestConfirmModalNavigateLeftRight(t *testing.T) {
	m := newTestModal()
	m.Open("T", "M", nil, nil)

	if m.cursor != 1 {
		t.Fatalf("cursor = %d, want 1 (No)", m.cursor)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.cursor != 0 {
		t.Errorf("after left: cursor = %d, want 0 (Yes)", m.cursor)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.cursor != 1 {
		t.Errorf("after right: cursor = %d, want 1 (No)", m.cursor)
	}
}

func TestConfirmModalClampsCursor(t *testing.T) {
	m := newTestModal()
	m.Open("T", "M", nil, nil)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.cursor != 1 {
		t.Errorf("right at max: cursor = %d, want 1", m.cursor)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.cursor != 0 {
		t.Errorf("left to min: cursor = %d, want 0", m.cursor)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.cursor != 0 {
		t.Errorf("left past min: cursor = %d, want 0", m.cursor)
	}
}

func TestConfirmModalViewEmptyWhenInactive(t *testing.T) {
	m := newTestModal()
	if m.View() != "" {
		t.Error("inactive modal View should be empty")
	}
}

func TestConfirmModalViewContainsTitle(t *testing.T) {
	m := newTestModal()
	m.Open("Delete Ticket?", "This cannot be undone.", nil, nil)
	view := m.View()
	if !strings.Contains(view, "Delete Ticket?") {
		t.Error("view should contain title")
	}
	if !strings.Contains(view, "This cannot be undone.") {
		t.Error("view should contain message")
	}
	if !strings.Contains(view, "Yes") {
		t.Error("view should contain Yes button")
	}
	if !strings.Contains(view, "No") {
		t.Error("view should contain No button")
	}
}

func TestConfirmModalCloseResetsState(t *testing.T) {
	m := newTestModal()
	m.Open("T", "M", func() tea.Cmd { return tea.Quit }, func() {})
	m.Close()

	if m.Active() {
		t.Error("modal should be inactive after Close")
	}
	if m.title != "" {
		t.Errorf("title = %q, want empty", m.title)
	}
	if m.onConfirm != nil {
		t.Error("onConfirm should be nil after Close")
	}
}

func TestConfirmModalIgnoresKeysWhenInactive(t *testing.T) {
	m := newTestModal()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("inactive modal should not produce commands")
	}
}
