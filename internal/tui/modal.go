package tui

import (
	"fmt"

	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ConfirmModal struct {
	active    bool
	title     string
	message   string
	onConfirm func() tea.Cmd
	onCancel  func()
	cursor    int
	width     int
	height    int
	styles    ConfirmModalStyles
}

type ConfirmModalStyles struct {
	Border    lipgloss.Style
	Title     lipgloss.Style
	Message   lipgloss.Style
	Confirm   lipgloss.Style
	Cancel    lipgloss.Style
	Highlight lipgloss.Style
}

func DefaultConfirmModalStyles() ConfirmModalStyles {
	return ConfirmModalStyles{
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("69")).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("69")),
		Message:   lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		Highlight: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226")),
	}
}

func NewConfirmModalStyles(t *theme.Theme) ConfirmModalStyles {
	return ConfirmModalStyles{
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary),
		Message:   lipgloss.NewStyle().Foreground(t.Text),
		Highlight: lipgloss.NewStyle().Bold(true).Foreground(t.Accent),
	}
}

func (m *ConfirmModal) SetTheme(t *theme.Theme) {
	m.styles = NewConfirmModalStyles(t)
}

func (m *ConfirmModal) Active() bool {
	return m.active
}

func (m *ConfirmModal) Open(title, message string, onConfirm func() tea.Cmd, onCancel func()) {
	m.active = true
	m.title = title
	m.message = message
	m.onConfirm = onConfirm
	m.onCancel = onCancel
	m.cursor = 1
}

func (m *ConfirmModal) Close() {
	m.active = false
	m.title = ""
	m.message = ""
	m.onConfirm = nil
	m.onCancel = nil
}

func (m ConfirmModal) Update(msg tea.Msg) (ConfirmModal, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if m.cursor > 0 {
				m.cursor--
			}
		case "right", "l":
			if m.cursor < 1 {
				m.cursor++
			}
		case "enter":
			m.active = false
			if m.cursor == 0 && m.onConfirm != nil {
				cmd := m.onConfirm()
				m.onConfirm = nil
				m.onCancel = nil
				return m, cmd
			}
			if m.onCancel != nil {
				m.onCancel()
			}
			m.onConfirm = nil
			m.onCancel = nil
		case "esc":
			m.active = false
			if m.onCancel != nil {
				m.onCancel()
			}
			m.onConfirm = nil
			m.onCancel = nil
		}
	}

	return m, nil
}

func (m ConfirmModal) View() string {
	if !m.active {
		return ""
	}

	yes := "[ Yes ]"
	no := "[ No ]"

	if m.cursor == 0 {
		yes = m.styles.Highlight.Render(yes)
	} else {
		yes = m.styles.Confirm.Render(yes)
	}

	if m.cursor == 1 {
		no = m.styles.Highlight.Render(no)
	} else {
		no = m.styles.Cancel.Render(no)
	}

	buttons := fmt.Sprintf("  %s    %s", yes, no)

	content := m.styles.Title.Render(m.title) + "\n\n" +
		m.styles.Message.Render(m.message) + "\n\n" +
		buttons

	boxWidth := 44
	content = m.styles.Border.Width(boxWidth).Render(content)

	verticalPad := max((m.height-lipgloss.Height(content))/2, 0)
	horizPad := max((m.width-boxWidth-4)/2, 0)

	padded := lipgloss.NewStyle().
		Padding(verticalPad, horizPad).
		Render(content)

	return padded
}

func (m ConfirmModal) SetSize(w, h int) {
	m.width = w
	m.height = h
}
