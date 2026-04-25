package tui

import (
	"fmt"

	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ConfirmModal struct {
	active      bool
	infoModal   bool
	title       string
	message     string
	onConfirm   func() tea.Cmd
	onCancel    func()
	cursor      int
	width       int
	height      int
	styles      ConfirmModalStyles
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
	m.infoModal = false
	m.title = title
	m.message = message
	m.onConfirm = onConfirm
	m.onCancel = onCancel
	m.cursor = 1
}

func (m *ConfirmModal) OpenInfo(title, message string, onClose func()) {
	m.active = true
	m.infoModal = true
	m.title = title
	m.message = message
	m.onConfirm = func() tea.Cmd {
		m.active = false
		if onClose != nil {
			onClose()
		}
		return nil
	}
	m.onCancel = nil
	m.cursor = 0
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
			if !m.infoModal && m.cursor > 0 {
				m.cursor--
			}
		case "right", "l":
			if !m.infoModal && m.cursor < 1 {
				m.cursor++
			}
		case "enter", "esc":
			m.active = false
			if m.onConfirm != nil {
				cmd := m.onConfirm()
				m.onConfirm = nil
				m.onCancel = nil
				if cmd != nil {
					return m, cmd
				}
			}
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

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		m.ViewBox(),
	)
}

func (m ConfirmModal) ViewBox() string {
	var buttons string

	if m.infoModal {
		ok := "[ OK ]"
		ok = m.styles.Highlight.Render(ok)
		buttons = fmt.Sprintf("  %s", ok)
	} else {
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

		buttons = fmt.Sprintf("  %s    %s", yes, no)
	}

	boxWidth := 44
	messageStyle := m.styles.Message.Width(boxWidth - 4)
	wrappedMessage := messageStyle.Render(m.message)

	content := m.styles.Title.Render(m.title) + "\n\n" +
		wrappedMessage + "\n\n" +
		buttons

	return m.styles.Border.Width(boxWidth).Render(content)
}

func (m *ConfirmModal) SetSize(w, h int) {
	m.width = w
	m.height = h
}
