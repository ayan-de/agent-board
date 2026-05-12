package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ayan-de/agent-board/internal/board"
	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type KanbanTab int

const (
	TabBoard KanbanTab = iota
	TabSearch
	TabDateFilter
)

type KanbanModel struct {
	store           *store.Store
	resolver        *keybinding.Resolver
	width           int
	height          int
	colIndex        int
	cursors         []int
	scrollOffsets   []int
	columns         [][]store.Ticket
	columnDefs      []config.Column
	styles          board.KanbanStyles
	animFrame       int
	theme           *theme.Theme
	tab             KanbanTab
	searchQuery     string
	monthOffset     int
	projectInitDate time.Time
}

func DefaultKanbanStyles() board.KanbanStyles {
	return board.DefaultKanbanStyles()
}

func NewKanbanStyles(t *theme.Theme) board.KanbanStyles {
	return board.NewKanbanStyles(t)
}

func NewKanbanModel(s *store.Store, resolver *keybinding.Resolver, t *theme.Theme, columns []config.Column) (KanbanModel, error) {
	m := KanbanModel{
		store:           s,
		resolver:        resolver,
		styles:          NewKanbanStyles(t),
		theme:           t,
		tab:             TabBoard,
		monthOffset:     0,
		projectInitDate: time.Now(),
		columnDefs:  columns,
	}
	m.initDynamicState()
	m, err := m.loadColumns()
	if err != nil {
		return m, fmt.Errorf("kanban.newKanbanModel: %w", err)
	}
	return m, nil
}

func (m *KanbanModel) initDynamicState() {
	numCols := len(m.columnDefs)
	if numCols == 0 {
		numCols = 4
	}
	m.cursors = make([]int, numCols)
	m.scrollOffsets = make([]int, numCols)
	m.columns = make([][]store.Ticket, numCols)
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
		m.columns = groupByStatusDynamic(msg.tickets, m.columnDefs)
		return m, nil
	case searchQueryMsg:
		results, err := m.store.ListTickets(context.Background(), store.TicketFilters{Search: msg.query})
		if err == nil {
			m.columns = groupByStatusDynamic(results, m.columnDefs)
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
		if m.colIndex < len(m.columns) {
			col := m.columns[m.colIndex]
			if len(col) > 0 {
				cursor := m.cursors[m.colIndex]
				return m, func() tea.Msg {
					return showDeleteModalMsg{ticketID: col[cursor].ID}
				}
			}
		}
		return m, nil
	}
	return m, nil
}

func (m KanbanModel) handleKey(msg tea.KeyMsg) (KanbanModel, tea.Cmd) {
	if msg.Type == tea.KeyTab {
		m.tab = (m.tab + 1) % 3
		return m, nil
	}
	if msg.Type == tea.KeyShiftTab {
		if m.tab == TabBoard {
			m.tab = TabDateFilter
		} else {
			m.tab--
		}
		return m, nil
	}

	// If in search mode, intercept keys for search and disable most board keybindings
	if m.tab == TabSearch {
		// Handle backspace
		if msg.Type == tea.KeyBackspace || msg.String() == "backspace" {
			if len(m.searchQuery) > 0 {
				m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				return m, m.debouncedSearch()
			}
			return m, nil
		}

		// Handle enter or escape to exit search mode?
		// For now, just let it be.

		// Handle typing
		if msg.Type == tea.KeyRunes {
			m.searchQuery += string(msg.Runes)
			return m, m.debouncedSearch()
		}

		// Block global actions like 'q', 'a', 'd' while in search mode
		// except for specific ones we might want?
		key := msg.String()
		action, _ := m.resolver.Resolve(key)
		if action != keybinding.ActionToggleFocus && action != keybinding.ActionPrevFocus && action != keybinding.ActionForceQuit {
			return m, nil
		}
	}

	key := msg.String()
	action, _ := m.resolver.Resolve(key)

	switch action {
	case keybinding.ActionToggleFocus:
		m.tab = (m.tab + 1) % 3
		return m, nil
	case keybinding.ActionPrevFocus:
		if m.tab == TabBoard {
			m.tab = TabDateFilter
		} else {
			m.tab--
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
		if m.colIndex < len(m.columnDefs)-1 {
			m.colIndex++
		}
	case keybinding.ActionPrevTicket:
		if m.cursors[m.colIndex] > 0 {
			m.cursors[m.colIndex]--
			if m.cursors[m.colIndex] < m.scrollOffsets[m.colIndex] {
				m.scrollOffsets[m.colIndex] = m.cursors[m.colIndex]
			}
		}
	case keybinding.ActionNextTicket:
		if m.colIndex < len(m.columns) && m.cursors[m.colIndex] < len(m.columns[m.colIndex])-1 {
			m.cursors[m.colIndex]++
			availH := m.height - 6
			if availH < 1 {
				availH = 10
			}
			numCols := len(m.columnDefs)
			if numCols == 0 {
				numCols = 4
			}
			colWidth := m.width / numCols
			innerWidth := colWidth - 4
			if innerWidth < 1 {
				innerWidth = 1
			}
			maxVisible := m.computeMaxVisible(m.colIndex, m.scrollOffsets[m.colIndex], innerWidth, availH)
			if m.cursors[m.colIndex] >= m.scrollOffsets[m.colIndex]+maxVisible {
				m.scrollOffsets[m.colIndex] = m.cursors[m.colIndex] - maxVisible + 1
			}
		}
	case keybinding.ActionJumpColumn1:
		if len(m.columnDefs) > 0 {
			m.colIndex = 0
		}
	case keybinding.ActionJumpColumn2:
		if len(m.columnDefs) > 1 {
			m.colIndex = 1
		}
	case keybinding.ActionJumpColumn3:
		if len(m.columnDefs) > 2 {
			m.colIndex = 2
		}
	case keybinding.ActionJumpColumn4:
		if len(m.columnDefs) > 3 {
			m.colIndex = 3
		}
	case keybinding.ActionAddTicket:
		if len(m.columnDefs) == 0 || m.colIndex != 0 {
			return m, func() tea.Msg {
				return notificationMsg{title: "Cannot create ticket", message: "Tickets can only be created in Backlog", variant: NotificationError}
			}
		}
		ticket, err := m.store.CreateTicket(context.Background(), store.Ticket{
			Title:  "New Ticket",
			Status: m.columnDefs[0].Status,
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

	numCols := len(m.columnDefs)
	if numCols == 0 {
		numCols = 4
	}
	colWidth := m.width / numCols
	remainder := m.width % numCols

	colInnerWidths := make([]int, numCols)
	for i := 0; i < numCols; i++ {
		w := colWidth
		if i >= numCols-remainder {
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

	cols := make([]string, numCols)
	for i := 0; i < numCols; i++ {
		innerWidth := colInnerWidths[i]
		var content strings.Builder

		colName := m.columnDefs[i].Name
		if colName == "" {
			colName = m.columnDefs[i].Status
		}

		titleStyle := m.styles.FocusedTitle
		if i != m.colIndex {
			titleStyle = m.styles.BlurredTitle
		}
		content.WriteString(titleStyle.Width(innerWidth).Render(colName))
		content.WriteString("\n")

		tickets := m.columns[i]
		if len(tickets) == 0 {
			content.WriteString(m.styles.EmptyColumn.Render("(empty)"))
		} else {
			expandedIdx := -1
			if i == m.colIndex && len(tickets) > 0 {
				expandedIdx = m.cursors[i]
			}

			// Determine if we need a scrollbar
			cardWidth := innerWidth
			maxShow := m.computeMaxVisible(i, m.scrollOffsets[i], cardWidth, availableHeight)
			overflow := len(tickets) > maxShow || m.scrollOffsets[i] > 0

			if overflow {
				cardWidth = innerWidth - 1
				// Recalculate with narrower width
				maxShow = m.computeMaxVisible(i, m.scrollOffsets[i], cardWidth, availableHeight)
			}

			var cardsContent strings.Builder

			// Top indicator
			if m.scrollOffsets[i] > 0 {
				cardsContent.WriteString(m.styles.EmptyColumn.Italic(true).Render(fmt.Sprintf("↑ %d more", m.scrollOffsets[i])))
				cardsContent.WriteString("\n")
			}

			for j := m.scrollOffsets[i]; j < len(tickets) && j < m.scrollOffsets[i]+maxShow; j++ {
				isSelected := i == m.colIndex && j == m.cursors[i]
				isExpanded := j == expandedIdx

				card := NewTicketCardModel(tickets[j], isSelected, isExpanded, cardWidth, m.animFrame, m.theme)
				cardsContent.WriteString(card.Render())

				// Add spacer if not the last card and there's more content below (either a card or an indicator)
				if j < m.scrollOffsets[i]+maxShow-1 || len(tickets) > m.scrollOffsets[i]+maxShow {
					cardsContent.WriteString("\n")
				}
			}

			// Bottom indicator
			if len(tickets) > m.scrollOffsets[i]+maxShow {
				remaining := len(tickets) - (m.scrollOffsets[i] + maxShow)
				cardsContent.WriteString(m.styles.EmptyColumn.Italic(true).Render(fmt.Sprintf("↓ %d more", remaining)))
			}

			if overflow {
				scrollBar := m.renderScrollBar(m.scrollOffsets[i], maxShow, len(tickets), availableHeight)
				content.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
					lipgloss.NewStyle().Width(cardWidth).Render(cardsContent.String()),
					scrollBar,
				))
			} else {
				content.WriteString(lipgloss.NewStyle().Width(cardWidth).Render(cardsContent.String()))
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
	count := 0
	for _, col := range m.columns {
		count += len(col)
	}
	return from.Format("Jan 02") + " - " + to.Format("Jan 02 2006") + " (" + strconv.Itoa(count) + " cards)"
}

func (m KanbanModel) renderSearchBar() string {
	prefix := "Search: "
	query := m.searchQuery

	var prefixStyle, queryStyle lipgloss.Style
	if m.tab == TabSearch {
		prefixStyle = lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true)
		queryStyle = lipgloss.NewStyle().Foreground(m.theme.Text).Bold(true)
	} else {
		prefixStyle = m.styles.SearchBox
		queryStyle = m.styles.SearchBox
	}

	return prefixStyle.Render(prefix) + queryStyle.Render(query)
}

func (m KanbanModel) renderTabBar() string {
	w := m.width
	if w < 10 {
		return ""
	}

	boardLabel := " Board "
	if m.tab == TabBoard {
		boardLabel = lipgloss.NewStyle().
			Background(m.theme.Primary).
			Foreground(m.theme.Text).
			Bold(true).
			Render(boardLabel)
	} else {
		boardLabel = m.styles.TabInactive.Render(boardLabel)
	}

	searchBar := m.renderSearchBar()
	monthHeader := m.renderMonthHeader()
	if m.tab == TabDateFilter {
		monthHeader = m.styles.TabActive.Render(monthHeader)
	} else {
		monthHeader = m.styles.TabInactive.Render(monthHeader)
	}

	boardWidth := lipgloss.Width(boardLabel)
	searchWidth := lipgloss.Width(searchBar)
	monthWidth := lipgloss.Width(monthHeader)

	leftPad := 2
	rightPad := 2

	// Spacing between elements
	gap1 := 4 // Between Board and Search
	gap2 := w - boardWidth - searchWidth - monthWidth - leftPad - rightPad - gap1

	if gap2 < 1 {
		gap2 = 1
	}

	return strings.Repeat(" ", leftPad) +
		boardLabel + strings.Repeat(" ", gap1) +
		searchBar + strings.Repeat(" ", gap2) +
		monthHeader + strings.Repeat(" ", rightPad)
}

func (m KanbanModel) IsSearchActive() bool {
	return m.tab == TabSearch
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
	if m.colIndex >= len(m.columns) {
		return []store.Ticket{}
	}
	return m.columns[m.colIndex]
}

func (m KanbanModel) Cursor() int {
	return m.cursors[m.colIndex]
}

func (m KanbanModel) Reload() (KanbanModel, error) {
	return m.loadColumns()
}

func (m KanbanModel) UpdateColumnDefs(columns []config.Column) (KanbanModel, error) {
	m.columnDefs = columns
	numCols := len(m.columnDefs)
	if numCols == 0 {
		numCols = 4
	}
	m.cursors = make([]int, numCols)
	m.scrollOffsets = make([]int, numCols)
	m.columns = make([][]store.Ticket, numCols)
	return m.loadColumns()
}

func (m KanbanModel) loadColumns() (KanbanModel, error) {
	for i, col := range m.columnDefs {
		tickets, err := m.store.ListTickets(context.Background(), store.TicketFilters{Status: col.Status})
		if err != nil {
			return m, fmt.Errorf("kanban.loadColumns: %w", err)
		}
		if tickets == nil {
			tickets = []store.Ticket{}
		}
		m.columns[i] = tickets
	}
	for i := range m.cursors {
		if i < len(m.columns) && m.cursors[i] >= len(m.columns[i]) && len(m.columns[i]) > 0 {
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

func groupByStatusDynamic(tickets []store.Ticket, columnDefs []config.Column) [][]store.Ticket {
	if len(columnDefs) == 0 {
		columnDefs = config.DefaultColumns()
	}
	cols := make([][]store.Ticket, len(columnDefs))
	statusMap := make(map[string]int)
	for i, col := range columnDefs {
		statusMap[col.Status] = i
	}
	for _, t := range tickets {
		if idx, ok := statusMap[t.Status]; ok {
			cols[idx] = append(cols[idx], t)
		}
	}
	return cols
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
	// Start exactly from the init date, no anchor to the 15th
	from = initDate.AddDate(0, offset, 0)
	// Set 'from' to the beginning of that day
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())

	// 'to' is one month later minus one day, set to the end of that day
	to = from.AddDate(0, 1, -1)
	to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, to.Location())

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
	m.columns = groupByStatusDynamic(tickets, m.columnDefs)
	return m, nil
}

func (m KanbanModel) computeMaxVisible(colIndex int, startIdx int, width int, availableHeight int) int {
	tickets := m.columns[colIndex]
	if len(tickets) == 0 {
		return 0
	}
	expandedIdx := -1
	if colIndex == m.colIndex {
		expandedIdx = m.cursors[colIndex]
	}

	hRemaining := availableHeight
	if startIdx > 0 {
		hRemaining-- // Reserve for "↑ X more"
	}

	count := 0
	lines := 0
	for i := startIdx; i < len(tickets); i++ {
		h := 4 // Default compact height (2 border + 2 content)
		if i == expandedIdx {
			card := NewTicketCardModel(tickets[i], false, true, width, 0, m.theme)
			h = card.ExpandedHeight() + 2 // 2 border + content
		}

		// Add spacer if not the first card
		hWithSpacer := h
		if i > startIdx {
			hWithSpacer++
		}

		// Check if this card fits
		if lines+hWithSpacer > hRemaining {
			break
		}

		// If this is NOT the last ticket overall, we must ensure there's room
		// for at least the "↓ X more" indicator if we stop after this card.
		if i+1 < len(tickets) && lines+hWithSpacer+1 > hRemaining {
			break
		}

		lines += hWithSpacer
		count++
	}
	if count < 1 && len(tickets) > 0 {
		count = 1
	}
	return count
}

func (m KanbanModel) renderScrollBar(scrollOff int, maxVisible int, total int, height int) string {
	if total <= maxVisible || height <= 0 {
		return ""
	}

	thumbLen := (maxVisible * height) / total
	if thumbLen < 1 {
		thumbLen = 1
	}

	thumbPos := (scrollOff * height) / total
	if thumbPos+thumbLen > height {
		thumbPos = height - thumbLen
	}

	var sb strings.Builder
	for i := 0; i < height; i++ {
		if i >= thumbPos && i < thumbPos+thumbLen {
			sb.WriteString(m.styles.TabActive.Render("┃"))
		} else {
			sb.WriteString(m.styles.TabInactive.Render("│"))
		}
		if i < height-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
