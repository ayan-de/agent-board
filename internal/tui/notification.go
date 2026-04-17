package tui

import (
	"time"

	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const maxVisibleNotifications = 4

type NotificationVariant int

const (
	NotificationInfo NotificationVariant = iota
	NotificationSuccess
	NotificationWarning
	NotificationError
)

type notificationDismissMsg struct {
	id int
}

type ticketCreatedMsg struct {
	id    string
	title string
}

type agentAssignedMsg struct {
	ticketID string
	agent    string
}

type NotificationStyles struct {
	InfoBorder    lipgloss.Style
	SuccessBorder lipgloss.Style
	WarningBorder lipgloss.Style
	ErrorBorder   lipgloss.Style
	Title         lipgloss.Style
	Message       lipgloss.Style
}

type NotificationItem struct {
	id       int
	title    string
	message  string
	variant  NotificationVariant
	duration time.Duration
}

type NotificationStack struct {
	nextID int
	width  int
	height int
	items  []NotificationItem
	styles NotificationStyles
}

func DefaultNotificationStyles() NotificationStyles {
	return NotificationStyles{
		InfoBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("69")).
			Padding(0, 1),
		SuccessBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("42")).
			Padding(0, 1),
		WarningBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("214")).
			Padding(0, 1),
		ErrorBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(0, 1),
		Title: lipgloss.NewStyle().
			Bold(true),
		Message: lipgloss.NewStyle(),
	}
}

func NewNotificationStyles(t *theme.Theme) NotificationStyles {
	return NotificationStyles{
		InfoBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary).
			Padding(0, 1),
		SuccessBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Success).
			Padding(0, 1),
		WarningBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Accent).
			Padding(0, 1),
		ErrorBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(0, 1),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Text),
		Message: lipgloss.NewStyle().
			Foreground(t.Text),
	}
}

func (n *NotificationStack) SetTheme(t *theme.Theme) {
	n.styles = NewNotificationStyles(t)
}

func (n *NotificationStack) SetSize(w, h int) {
	n.width = w
	n.height = h
}

func (n NotificationStack) Active() bool {
	return len(n.items) > 0
}

func (n *NotificationStack) Show(title, message string, variant NotificationVariant, duration time.Duration) tea.Cmd {
	if duration <= 0 {
		duration = 2 * time.Second
	}

	n.nextID++
	item := NotificationItem{
		id:       n.nextID,
		title:    title,
		message:  message,
		variant:  variant,
		duration: duration,
	}
	n.items = append(n.items, item)
	if len(n.items) > maxVisibleNotifications {
		n.items = n.items[len(n.items)-maxVisibleNotifications:]
	}

	id := item.id
	return tea.Tick(duration, func(time.Time) tea.Msg {
		return notificationDismissMsg{id: id}
	})
}

func (n *NotificationStack) Close() {
	n.items = nil
}

func (n NotificationStack) HandleDismiss(msg notificationDismissMsg) NotificationStack {
	for i, item := range n.items {
		if item.id == msg.id {
			n.items = append(n.items[:i], n.items[i+1:]...)
			break
		}
	}
	return n
}

func (n NotificationStack) Update(msg tea.Msg) (NotificationStack, tea.Cmd) {
	switch msg := msg.(type) {
	case notificationDismissMsg:
		return n.HandleDismiss(msg), nil
	}
	return n, nil
}

func (n NotificationStack) View() string {
	if !n.Active() {
		return ""
	}

	blocks := make([]string, 0, len(n.items))
	for _, item := range n.items {
		blocks = append(blocks, n.viewItem(item))
	}

	return lipgloss.JoinVertical(lipgloss.Right, blocks...)
}

func (n NotificationStack) viewItem(item NotificationItem) string {
	boxWidth := 36
	if n.width > 0 {
		maxWidth := n.width / 3
		if maxWidth < 28 {
			maxWidth = 28
		}
		if maxWidth < boxWidth {
			boxWidth = maxWidth
		}
	}

	messageWidth := boxWidth - 4
	if messageWidth < 1 {
		messageWidth = 1
	}

	content := n.styles.Title.Render(item.title)
	if item.message != "" {
		content += "\n" + n.styles.Message.Width(messageWidth).Render(item.message)
	}

	return n.borderStyle(item.variant).Width(boxWidth).Render(content)
}

func (n NotificationStack) borderStyle(variant NotificationVariant) lipgloss.Style {
	switch variant {
	case NotificationSuccess:
		return n.styles.SuccessBorder
	case NotificationWarning:
		return n.styles.WarningBorder
	case NotificationError:
		return n.styles.ErrorBorder
	default:
		return n.styles.InfoBorder
	}
}
