package tui

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var statusNames = [4]string{"backlog", "in_progress", "review", "done"}

var columnNames = [4]string{"Backlog", "In Progress", "Review", "Done"}

type KanbanStyles struct {
	FocusedColumn  lipgloss.Style
	BlurredColumn  lipgloss.Style
	FocusedTitle   lipgloss.Style
	BlurredTitle   lipgloss.Style
	SelectedTicket lipgloss.Style
	Ticket         lipgloss.Style
	EmptyColumn    lipgloss.Style
}

type KanbanModel struct {
	store    *store.Store
	resolver *keybinding.Resolver
	width    int
	height   int
	colIndex int
	cursors  [4]int
	columns  [4][]store.Ticket
	styles   KanbanStyles
}

func DefaultKanbanStyles() KanbanStyles {
	return KanbanStyles{
		FocusedColumn: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("69")),
		BlurredColumn: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
		FocusedTitle: lipgloss.NewStyle().
			Background(lipgloss.Color("69")).
			Foreground(lipgloss.Color("15")).
			Bold(true),
		BlurredTitle: lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")),
		SelectedTicket: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")),
		Ticket: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		EmptyColumn: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
	}
}

func NewKanbanStyles(t *theme.Theme) KanbanStyles {
	return KanbanStyles{
		FocusedColumn: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary),
		BlurredColumn: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.TextMuted),
		FocusedTitle: lipgloss.NewStyle().
			Background(t.Primary).
			Foreground(t.Text).
			Bold(true),
		BlurredTitle: lipgloss.NewStyle().
			Background(t.BackgroundPanel).
			Foreground(t.Text),
		SelectedTicket: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Text),
		Ticket: lipgloss.NewStyle().
			Foreground(t.Text),
		EmptyColumn: lipgloss.NewStyle().
			Foreground(t.TextMuted),
	}
}

func NewKanbanModel(s *store.Store, resolver *keybinding.Resolver, t *theme.Theme) (KanbanModel, error) {
	m := KanbanModel{
		store:    s,
		resolver: resolver,
		styles:   NewKanbanStyles(t),
	}
	m, err := m.loadColumns()
	if err != nil {
		return m, fmt.Errorf("kanban.newKanbanModel: %w", err)
	}
	return m, nil
}

func (m KanbanModel) Init() tea.Cmd {
	return nil
}

func (m KanbanModel) Update(msg tea.Msg) (KanbanModel, tea.Cmd) {
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

func (m KanbanModel) handleKey(msg tea.KeyMsg) (KanbanModel, tea.Cmd) {
	key := msg.String()
	action, _ := m.resolver.Resolve(key)

	switch action {
	case keybinding.ActionPrevColumn:
		if m.colIndex > 0 {
			m.colIndex--
		}
	case keybinding.ActionNextColumn:
		if m.colIndex < 3 {
			m.colIndex++
		}
	case keybinding.ActionPrevTicket:
		if m.cursors[m.colIndex] > 0 {
			m.cursors[m.colIndex]--
		}
	case keybinding.ActionNextTicket:
		if m.cursors[m.colIndex] < len(m.columns[m.colIndex])-1 {
			m.cursors[m.colIndex]++
		}
	case keybinding.ActionJumpColumn1:
		m.colIndex = 0
	case keybinding.ActionJumpColumn2:
		m.colIndex = 1
	case keybinding.ActionJumpColumn3:
		m.colIndex = 2
	case keybinding.ActionJumpColumn4:
		m.colIndex = 3
	case keybinding.ActionAddTicket:
		_, err := m.store.CreateTicket(context.Background(), store.Ticket{
			Title:  "New Ticket",
			Status: statusNames[m.colIndex],
		})
		if err != nil {
			return m, nil
		}
		m, _ = m.loadColumns()
	case keybinding.ActionDeleteTicket:
		col := m.columns[m.colIndex]
		if len(col) > 0 {
			cursor := m.cursors[m.colIndex]
			_ = m.store.DeleteTicket(context.Background(), col[cursor].ID)
			m, _ = m.loadColumns()
		}
	}

	return m, nil
}

func (m KanbanModel) View() string {
	if m.width == 0 {
		return ""
	}

	colWidth := m.width / 4
	remainder := m.width % 4

	colInnerWidths := [4]int{}
	for i := 0; i < 4; i++ {
		w := colWidth
		if i >= 4-remainder {
			w++
		}
		colInnerWidths[i] = w - 4
		if colInnerWidths[i] < 1 {
			colInnerWidths[i] = 1
		}
	}

	availableHeight := m.height - 6
	if availableHeight < 1 {
		availableHeight = 10
	}

	cols := make([]string, 4)
	for i := 0; i < 4; i++ {
		innerWidth := colInnerWidths[i]
		var content strings.Builder

		titleStyle := m.styles.FocusedTitle
		if i != m.colIndex {
			titleStyle = m.styles.BlurredTitle
		}
		content.WriteString(titleStyle.Width(innerWidth).Render(columnNames[i]))
		content.WriteString("\n")

		tickets := m.columns[i]
		if len(tickets) == 0 {
			content.WriteString(m.styles.EmptyColumn.Render("(empty)"))
		} else {
			maxShow := availableHeight
			overflow := len(tickets) > maxShow
			if overflow {
				maxShow = availableHeight - 1
				if maxShow < 0 {
					maxShow = 0
				}
			}

			for j := 0; j < len(tickets) && j < maxShow; j++ {
				ticket := tickets[j]
				prefix := "  "
				if i == m.colIndex && j == m.cursors[i] {
					prefix = "▸ "
				}

				line := prefix + ticket.ID + " " + ticket.Title
				if utf8.RuneCountInString(line) > innerWidth {
					runes := []rune(line)
					line = string(runes[:innerWidth-1]) + "…"
				}
				if ticket.Agent != "" {
					color := config.AgentColor(ticket.Agent)
					if color != "" {
						dot := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("●")
						line = line + " " + dot
					}
				}

				if i == m.colIndex && j == m.cursors[i] {
					line = m.styles.SelectedTicket.Render(line)
				} else {
					line = m.styles.Ticket.Render(line)
				}

				content.WriteString(line)
				content.WriteString("\n")
			}

			if overflow {
				remaining := len(tickets) - maxShow
				content.WriteString(fmt.Sprintf("↓ %d more", remaining))
			}
		}

		colStyle := m.styles.FocusedColumn
		if i != m.colIndex {
			colStyle = m.styles.BlurredColumn
		}
		colStyle = colStyle.Width(innerWidth+2).Padding(0, 1)

		cols[i] = colStyle.Render(content.String())
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cols...)
}

func (m KanbanModel) SelectedTicket() *store.Ticket {
	col := m.columns[m.colIndex]
	if len(col) == 0 {
		return nil
	}
	cursor := m.cursors[m.colIndex]
	if cursor >= len(col) {
		return nil
	}
	return &col[cursor]
}

func (m KanbanModel) Reload() (KanbanModel, error) {
	return m.loadColumns()
}

func (m KanbanModel) loadColumns() (KanbanModel, error) {
	for i, status := range statusNames {
		tickets, err := m.store.ListTickets(context.Background(), store.TicketFilters{Status: status})
		if err != nil {
			return m, fmt.Errorf("kanban.loadColumns: %w", err)
		}
		if tickets == nil {
			tickets = []store.Ticket{}
		}
		m.columns[i] = tickets
	}
	for i := range m.cursors {
		if m.cursors[i] >= len(m.columns[i]) && len(m.columns[i]) > 0 {
			m.cursors[i] = len(m.columns[i]) - 1
		}
	}
	return m, nil
}
