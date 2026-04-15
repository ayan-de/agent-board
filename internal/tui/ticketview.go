package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ticketViewModeType int

const (
	ticketViewMode ticketViewModeType = iota
	ticketEditMode
)

var statusCycle = [4]string{"backlog", "in_progress", "review", "done"}

type ticketField struct {
	label    string
	value    func(t *store.Ticket) string
	editable bool
	set      func(t *store.Ticket, v string)
}

type TicketViewStyles struct {
	Border      lipgloss.Style
	Title       lipgloss.Style
	Label       lipgloss.Style
	Value       lipgloss.Style
	SelectedRow lipgloss.Style
	Cursor      lipgloss.Style
	EditBox     lipgloss.Style
	Footer      lipgloss.Style
	Empty       lipgloss.Style
}

type TicketViewModel struct {
	store    *store.Store
	resolver *keybinding.Resolver
	width    int
	height   int

	ticket     *store.Ticket
	fields     []ticketField
	cursor     int
	mode       ticketViewModeType
	editBuffer string

	styles TicketViewStyles
}

func DefaultTicketViewStyles() TicketViewStyles {
	return TicketViewStyles{
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("69")).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("69")),
		Label: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252")),
		Value: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		SelectedRow: lipgloss.NewStyle().
			Background(lipgloss.Color("69")).
			Foreground(lipgloss.Color("15")),
		Cursor: lipgloss.NewStyle().
			Foreground(lipgloss.Color("69")),
		EditBox: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("213")),
		Footer: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		Empty: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
	}
}

func ticketFields() []ticketField {
	return []ticketField{
		{
			label:    "ID",
			value:    func(t *store.Ticket) string { return t.ID },
			editable: false,
		},
		{
			label:    "Title",
			value:    func(t *store.Ticket) string { return t.Title },
			editable: true,
			set:      func(t *store.Ticket, v string) { t.Title = v },
		},
		{
			label:    "Description",
			value:    func(t *store.Ticket) string { return t.Description },
			editable: true,
			set:      func(t *store.Ticket, v string) { t.Description = v },
		},
		{
			label:    "Status",
			value:    func(t *store.Ticket) string { return t.Status },
			editable: false,
		},
		{
			label:    "Priority",
			value:    func(t *store.Ticket) string { return t.Priority },
			editable: false,
		},
		{
			label:    "Agent",
			value:    func(t *store.Ticket) string { return t.Agent },
			editable: true,
			set:      func(t *store.Ticket, v string) { t.Agent = v },
		},
		{
			label:    "Branch",
			value:    func(t *store.Ticket) string { return t.Branch },
			editable: true,
			set:      func(t *store.Ticket, v string) { t.Branch = v },
		},
		{
			label: "Tags",
			value: func(t *store.Ticket) string {
				if len(t.Tags) == 0 {
					return ""
				}
				return strings.Join(t.Tags, ", ")
			},
			editable: false,
		},
		{
			label: "Depends On",
			value: func(t *store.Ticket) string {
				if len(t.DependsOn) == 0 {
					return ""
				}
				return strings.Join(t.DependsOn, ", ")
			},
			editable: false,
		},
		{
			label: "Created",
			value: func(t *store.Ticket) string {
				return t.CreatedAt.Format(time.DateTime)
			},
			editable: false,
		},
		{
			label: "Updated",
			value: func(t *store.Ticket) string {
				return t.UpdatedAt.Format(time.DateTime)
			},
			editable: false,
		},
	}
}

func NewTicketViewModel(s *store.Store, resolver *keybinding.Resolver) TicketViewModel {
	return TicketViewModel{
		store:    s,
		resolver: resolver,
		styles:   DefaultTicketViewStyles(),
		fields:   ticketFields(),
		mode:     ticketViewMode,
	}
}

func (m TicketViewModel) Init() tea.Cmd {
	return nil
}

func (m TicketViewModel) Update(msg tea.Msg) (TicketViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m TicketViewModel) handleKey(msg tea.KeyMsg) (TicketViewModel, tea.Cmd) {
	if m.mode == ticketEditMode {
		return m.handleEditKey(msg)
	}
	return m.handleViewKey(msg)
}

func (m TicketViewModel) handleViewKey(msg tea.KeyMsg) (TicketViewModel, tea.Cmd) {
	key := msg.String()
	action, _ := m.resolver.Resolve(key)

	switch action {
	case keybinding.ActionNextTicket:
		if m.cursor < len(m.fields)-1 {
			m.cursor++
		}
	case keybinding.ActionPrevTicket:
		if m.cursor > 0 {
			m.cursor--
		}
	}

	switch key {
	case "e":
		if m.ticket != nil && m.cursor < len(m.fields) && m.fields[m.cursor].editable {
			m.editBuffer = m.fields[m.cursor].value(m.ticket)
			m.mode = ticketEditMode
		}
	case "s":
		if m.ticket != nil {
			m = m.cycleStatus()
		}
	}

	return m, nil
}

func (m TicketViewModel) handleEditKey(msg tea.KeyMsg) (TicketViewModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if m.ticket != nil && m.cursor < len(m.fields) {
			f := m.fields[m.cursor]
			if f.set != nil {
				f.set(m.ticket, m.editBuffer)
				_, _ = m.store.UpdateTicket(context.Background(), *m.ticket)
			}
		}
		m.mode = ticketViewMode
		m.editBuffer = ""
		return m, nil
	case tea.KeyEscape:
		m.mode = ticketViewMode
		m.editBuffer = ""
		return m, nil
	case tea.KeyBackspace:
		if len(m.editBuffer) > 0 {
			runes := []rune(m.editBuffer)
			m.editBuffer = string(runes[:len(runes)-1])
		}
		return m, nil
	}

	if msg.Type == tea.KeyRunes {
		m.editBuffer += string(msg.Runes)
	}

	return m, nil
}

func (m TicketViewModel) cycleStatus() TicketViewModel {
	currentIdx := -1
	for i, s := range statusCycle {
		if s == m.ticket.Status {
			currentIdx = i
			break
		}
	}
	if currentIdx == -1 {
		currentIdx = 0
	}
	nextIdx := (currentIdx + 1) % len(statusCycle)
	m.ticket.Status = statusCycle[nextIdx]
	_ = m.store.MoveStatus(context.Background(), m.ticket.ID, m.ticket.Status)
	return m
}

func (m TicketViewModel) SetTicket(t *store.Ticket) TicketViewModel {
	m.ticket = t
	m.cursor = 0
	m.mode = ticketViewMode
	m.editBuffer = ""
	return m
}

func (m TicketViewModel) View() string {
	if m.ticket == nil {
		return m.styles.Empty.Render("No ticket selected")
	}

	if m.width == 0 {
		return ""
	}

	innerWidth := m.width - 6
	if innerWidth < 20 {
		innerWidth = 20
	}

	var b strings.Builder

	titleLine := m.styles.Title.Render(fmt.Sprintf("%s  %s", m.ticket.ID, m.ticket.Title))
	b.WriteString(titleLine)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", min(innerWidth, 60)))
	b.WriteString("\n\n")

	for i, f := range m.fields {
		val := f.value(m.ticket)
		if val == "" {
			val = "—"
		}

		prefix := "  "
		if i == m.cursor {
			prefix = "▸ "
		}

		label := m.styles.Label.Render(fmt.Sprintf("%-12s", f.label))

		row := prefix + label + " " + val

		if i == m.cursor {
			row = m.styles.SelectedRow.Width(innerWidth - 2).Render(row)
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	if m.mode == ticketEditMode {
		b.WriteString("\n")
		editLabel := fmt.Sprintf("Edit %s:", m.fields[m.cursor].label)
		b.WriteString(m.styles.Label.Render(editLabel))
		b.WriteString("\n")
		editLine := m.editBuffer + "│"
		b.WriteString(m.styles.EditBox.Width(innerWidth - 4).Render(editLine))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	footer := "e: edit │ s: cycle status │ Esc: back"
	b.WriteString(m.styles.Footer.Render(footer))

	return m.styles.Border.Render(b.String())
}
