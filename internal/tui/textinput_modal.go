package tui

import (
	"strings"

	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TextInputModal struct {
	active       bool
	title        string
	placeholder  string
	input        string
	onConfirm    func(string) tea.Cmd
	onCancel     func()
	width        int
	height       int
	styles       TextInputModalStyles
	theme        *theme.Theme
}

type TextInputModalStyles struct {
	Border      lipgloss.Style
	Title       lipgloss.Style
	Input       lipgloss.Style
	Placeholder lipgloss.Style
	Hint        lipgloss.Style
}

func DefaultTextInputModalStyles() TextInputModalStyles {
	return TextInputModalStyles{
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("69")).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("69")),
		Input: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		Placeholder: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		Hint: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
	}
}

func NewTextInputModalStyles(t *theme.Theme) TextInputModalStyles {
	return TextInputModalStyles{
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary),
		Input: lipgloss.NewStyle().
			Foreground(t.Text),
		Placeholder: lipgloss.NewStyle().
			Foreground(t.TextMuted),
		Hint: lipgloss.NewStyle().
			Foreground(t.TextMuted),
	}
}

func (m *TextInputModal) SetTheme(t *theme.Theme) {
	m.theme = t
	m.styles = NewTextInputModalStyles(t)
}

func (m *TextInputModal) Active() bool {
	return m.active
}

func (m *TextInputModal) Open(title, placeholder string, onConfirm func(string) tea.Cmd, onCancel func()) {
	m.active = true
	m.title = title
	m.placeholder = placeholder
	m.input = ""
	m.onConfirm = onConfirm
	m.onCancel = onCancel
}

func (m *TextInputModal) Close() {
	m.active = false
	m.title = ""
	m.placeholder = ""
	m.input = ""
	m.onConfirm = nil
	m.onCancel = nil
}

func (m TextInputModal) Update(msg tea.Msg) (TextInputModal, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m TextInputModal) handleKey(msg tea.KeyMsg) (TextInputModal, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.active = false
		if m.onCancel != nil {
			m.onCancel()
		}
		m.onConfirm = nil
		m.onCancel = nil
		return m, nil
	case tea.KeyCtrlJ:
		m.active = false
		if m.onConfirm != nil {
			cmd := m.onConfirm(m.input)
			m.onConfirm = nil
			m.onCancel = nil
			return m, cmd
		}
		return m, nil
	case tea.KeyEnter:
	default:
		runes := string(msg.Runes)
		m.input += runes
	}

	return m, nil
}

func (m TextInputModal) View() string {
	if !m.active {
		return ""
	}

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		m.ViewBox(),
	)
}

func (m TextInputModal) ViewBox() string {
	inputDisplay := m.input
	if inputDisplay == "" {
		inputDisplay = m.placeholder
	}

	lines := strings.Split(inputDisplay, "\n")
	maxLines := 10
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	var styledLines []string
	for _, line := range lines {
		if inputDisplay == m.placeholder {
			styledLines = append(styledLines, m.styles.Placeholder.Render(line))
		} else {
			styledLines = append(styledLines, m.styles.Input.Render(line))
		}
	}

	content := strings.Join(styledLines, "\n")

	boxWidth := 60
	hint := m.styles.Hint.Render("Ctrl+J to confirm  |  Esc to cancel")

	fullContent := m.styles.Title.Render(m.title) + "\n\n" +
		m.styles.Border.Width(boxWidth).Height(14).Render(content) + "\n\n" +
		hint

	return m.styles.Border.Width(boxWidth).Render(fullContent)
}

func (m *TextInputModal) SetSize(w, h int) {
	m.width = w
	m.height = h
}