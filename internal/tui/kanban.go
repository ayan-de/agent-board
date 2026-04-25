package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type KanbanTab int

const (
	TabSearch KanbanTab = iota
	TabDateFilter
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
	TabBar         lipgloss.Style
	TabActive      lipgloss.Style
	TabInactive    lipgloss.Style
	SearchBox      lipgloss.Style
	SearchCursor   lipgloss.Style
	SearchBoxActive lipgloss.Style
}

type KanbanModel struct {
	store           *store.Store
	resolver        *keybinding.Resolver
	width           int
	height          int
	colIndex        int
	cursors         [4]int
	columns         [4][]store.Ticket
	styles          KanbanStyles
	animFrame       int
	theme           *theme.Theme
	tab             KanbanTab
	searchQuery     string
	monthOffset     int
	projectInitDate time.Time
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
		TabBar: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		TabActive: lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true),
		TabInactive: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		SearchBox: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		SearchBoxActive: lipgloss.NewStyle().
			Foreground(lipgloss.Color("69")).Bold(true),
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
		TabBar: lipgloss.NewStyle().Foreground(t.TextMuted),
		TabActive: lipgloss.NewStyle().Foreground(t.Primary).Bold(true),
		TabInactive: lipgloss.NewStyle().Foreground(t.TextMuted),
		SearchBox: lipgloss.NewStyle().
			Foreground(t.TextMuted),
		SearchBoxActive: lipgloss.NewStyle().
			Foreground(t.Primary).Bold(true),
	}
}

func NewKanbanModel(s *store.Store, resolver *keybinding.Resolver, t *theme.Theme) (KanbanModel, error) {
	m := KanbanModel{
		store:       s,
		resolver:    resolver,
		styles:      NewKanbanStyles(t),
		theme:       t,
		tab:         TabSearch,
		monthOffset: 0,
	}
	m, err := m.loadColumns()
	if err != nil {
		return m, fmt.Errorf("kanban.newKanbanModel: %w", err)
	}
	return m, nil
}

func (m KanbanModel) Init() tea.Cmd {
	if m.anyAgentActive() {
		return animationTick()
	}
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
	case tickMsg:
		m.animFrame = (m.animFrame + 1) % AnimFrames
		if m.anyAgentActive() {
			return m, animationTick()
		}
		return m, nil
	case searchResultsMsg:
		m.columns = groupByStatus(msg.tickets)
		return m, nil
	case searchQueryMsg:
		results, err := m.store.ListTickets(context.Background(), store.TicketFilters{Search: msg.query})
		if err == nil {
			m.columns = groupByStatus(results)
		}
		return m, nil
	case monthNavigateMsg:
		if msg.direction == 1 {
			m.monthOffset++
		} else if msg.direction == -1 && m.monthOffset > 0 {
			m.monthOffset--
		}
		m, _ = m.loadMonth()
		return m, nil
	case tabChangeMsg:
		m.tab = msg.tab
		return m, nil
	case deleteTicketConfirmMsg:
		return m, nil
	case deleteTicketRequestMsg:
		col := m.columns[m.colIndex]
		if len(col) > 0 {
			cursor := m.cursors[m.colIndex]
			return m, func() tea.Msg {
				return showDeleteModalMsg{ticketID: col[cursor].ID}
			}
		}
		return m, nil
	}
	return m, nil
}

func (m KanbanModel) handleKey(msg tea.KeyMsg) (KanbanModel, tea.Cmd) {
	if m.tab == TabSearch && msg.Type == tea.KeyRunes {
		r := msg.Runes[0]
		if r >= 32 && r < 127 {
			action, _ := m.resolver.Resolve(msg.String())
			if action == keybinding.ActionNone {
				m.searchQuery += string(r)
				return m, m.debouncedSearch()
			}
		}
	}
	if m.tab == TabSearch && (msg.Type == tea.KeyBackspace || msg.String() == "backspace") {
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			return m, m.debouncedSearch()
		}
	}

	key := msg.String()
	action, _ := m.resolver.Resolve(key)

	switch action {
	case keybinding.ActionNextTab:
		if m.tab == TabSearch {
			m.tab = TabDateFilter
		} else {
			m.tab = TabSearch
		}
		return m, nil
	case keybinding.ActionPrevTab:
		if m.tab == TabSearch {
			m.tab = TabDateFilter
		} else {
			m.tab = TabSearch
		}
		return m, nil
	case keybinding.ActionPrevColumn:
		if m.tab == TabDateFilter {
			if m.monthOffset > 0 {
				m.monthOffset--
				m, _ = m.loadMonth()
				return m, nil
			}
			return m, nil
		}
		if m.colIndex > 0 {
			m.colIndex--
		}
	case keybinding.ActionNextColumn:
		if m.tab == TabDateFilter {
			m.monthOffset++
			m, _ = m.loadMonth()
			return m, nil
		}
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
		if m.colIndex != 0 {
			return m, func() tea.Msg {
				return notificationMsg{title: "Cannot create ticket", message: "Tickets can only be created in Backlog", variant: NotificationError}
			}
		}
		ticket, err := m.store.CreateTicket(context.Background(), store.Ticket{
			Title:  "New Ticket",
			Status: statusNames[0],
		})
		if err != nil {
			return m, nil
		}
		m, _ = m.loadColumns()
		return m, func() tea.Msg {
			return ticketCreatedMsg{id: ticket.ID, title: ticket.Title}
		}
	case keybinding.ActionDeleteTicket:
		col := m.columns[m.colIndex]
		if len(col) > 0 {
			cursor := m.cursors[m.colIndex]
			return m, func() tea.Msg {
				return deleteTicketRequestMsg{ticketID: col[cursor].ID}
			}
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

	availableHeight := m.height - 7
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
			cardWidth := innerWidth
			compactRendered := 2 + 2 + 1
			expandedIdx := -1
			if i == m.colIndex && len(tickets) > 0 {
				expandedIdx = m.cursors[i]
			}

			maxShow := 0
			usedLines := 0
			for j := 0; j < len(tickets); j++ {
				h := compactRendered
				if j == expandedIdx {
					card := NewTicketCardModel(tickets[j], true, true, cardWidth, m.animFrame, m.theme)
					h = card.ExpandedHeight() + 1
				}
				if usedLines+h > availableHeight {
					break
				}
				usedLines += h
				maxShow = j + 1
			}

			overflow := len(tickets) > maxShow

			for j := 0; j < len(tickets) && j < maxShow; j++ {
				isSelected := i == m.colIndex && j == m.cursors[i]
				isExpanded := j == expandedIdx

				card := NewTicketCardModel(tickets[j], isSelected, isExpanded, cardWidth, m.animFrame, m.theme)
				content.WriteString(card.Render())
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

	tabBar := m.renderTabBar()
	board := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
	return lipgloss.JoinVertical(lipgloss.Top, tabBar, board)
}

func (m KanbanModel) renderMonthHeader() string {
	from, to := MonthWindow(m.projectInitDate, m.monthOffset)
	count := len(m.columns[0]) + len(m.columns[1]) + len(m.columns[2]) + len(m.columns[3])
	return from.Format("Jan 15") + " - " + to.Format("Feb 14 2006") + " (" + strconv.Itoa(count) + " cards)"
}

func (m KanbanModel) renderSearchBar() string {
	prompt := "Search: " + m.searchQuery
	if m.tab == TabSearch {
		return m.styles.SearchBoxActive.Render(prompt)
	}
	return m.styles.SearchBox.Render(prompt)
}

func (m KanbanModel) renderTabBar() string {
	w := m.width

	searchBar := m.renderSearchBar()
	monthHeader := m.renderMonthHeader()
	if m.tab == TabDateFilter {
		monthHeader = m.styles.TabActive.Render(monthHeader)
	} else {
		monthHeader = m.styles.TabInactive.Render(monthHeader)
	}

	searchWidth := m.width/2 - 4
	if searchWidth < 20 {
		searchWidth = 20
	}
	actualSearchWidth := searchWidth + 4

	gap := 3
	totalWidth := actualSearchWidth + gap + lipgloss.Width(monthHeader)
	leftPad := w - totalWidth
	if leftPad < 1 {
		leftPad = 1
	}

	return searchBar + strings.Repeat(" ", leftPad) + monthHeader
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

func (m KanbanModel) Column() []store.Ticket {
	return m.columns[m.colIndex]
}

func (m KanbanModel) Cursor() int {
	return m.cursors[m.colIndex]
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

func (m KanbanModel) anyAgentActive() bool {
	for _, col := range m.columns {
		for _, t := range col {
			if t.AgentActive {
				return true
			}
		}
	}
	return false
}

func (m KanbanModel) NeedsTick() bool {
	return m.anyAgentActive()
}

func (m KanbanModel) debouncedSearch() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(400 * time.Millisecond)
		return searchQueryMsg{query: m.searchQuery}
	}
}

func groupByStatus(tickets []store.Ticket) [4][]store.Ticket {
	cols := [4][]store.Ticket{}
	statuses := [4]string{"backlog", "in_progress", "review", "done"}
	for _, t := range tickets {
		for i, s := range statuses {
			if t.Status == s {
				cols[i] = append(cols[i], t)
			}
		}
	}
	return cols
}

func MonthWindow(initDate time.Time, offset int) (from, to time.Time) {
	base := time.Date(initDate.Year(), initDate.Month(), 15, 0, 0, 0, 0, initDate.Location())
	from = base.AddDate(0, offset, 0)
	to = time.Date(from.Year(), from.Month()+1, 14, 23, 59, 59, 0, from.Location())
	return from, to
}

func (m KanbanModel) loadMonth() (KanbanModel, error) {
	from, to := MonthWindow(m.projectInitDate, m.monthOffset)
	fromPtr := &from
	toPtr := &to
	tickets, err := m.store.ListTickets(context.Background(), store.TicketFilters{From: fromPtr, To: toPtr})
	if err != nil {
		return m, err
	}
	m.columns = groupByStatus(tickets)
	return m, nil
}
