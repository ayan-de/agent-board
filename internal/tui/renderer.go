package tui

import (
	"fmt"
	"strings"

	"github.com/ayan-de/agent-board/internal/board"
	"github.com/charmbracelet/lipgloss"
)

type Renderer struct {
	width  int
	height int
}

func NewRenderer(width, height int) *Renderer {
	return &Renderer{width: width, height: height}
}

func (r *Renderer) SetSize(width, height int) {
	r.width = width
	r.height = height
}

func (r *Renderer) Render(state board.BoardViewState) string {
	switch state.ActiveView {
	case board.ViewBoard:
		return r.renderKanban(state.Kanban)
	case board.ViewTicket:
		if state.Ticket != nil {
			return r.renderTicket(*state.Ticket)
		}
	case board.ViewDashboard:
		return r.renderDashboard(state.Dashboard)
	case board.ViewHelp:
		return r.renderHelp()
	}
	return ""
}

func (r *Renderer) renderKanban(state board.KanbanViewState) string {
	if r.width == 0 || len(state.Columns) == 0 {
		return ""
	}

	numCols := len(state.Columns)
	if numCols == 0 {
		numCols = 4
	}
	colWidth := r.width / numCols
	remainder := r.width % numCols

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

	availableHeight := r.height - 6
	if availableHeight < 1 {
		availableHeight = 10
	}

	cols := make([]string, numCols)
	for i := 0; i < numCols; i++ {
		innerWidth := colInnerWidths[i]
		var content strings.Builder

		colName := state.ColumnDefs[i].Name
		if colName == "" {
			colName = state.ColumnDefs[i].Status
		}

		titleStyle := state.Styles.FocusedTitle
		if i != state.ColIndex {
			titleStyle = state.Styles.BlurredTitle
		}
		content.WriteString(titleStyle.Width(innerWidth).Render(colName))
		content.WriteString("\n")

		tickets := state.Columns[i].Tickets
		if len(tickets) == 0 {
			content.WriteString(state.Styles.EmptyColumn.Render("(empty)"))
		} else {
			expandedIdx := -1
			if i == state.ColIndex && len(tickets) > 0 {
				expandedIdx = state.Cursors[i]
			}

			cardWidth := innerWidth
			maxShow := computeMaxVisibleKanban(len(tickets), state.ScrollOff[i], cardWidth, availableHeight, expandedIdx)
			overflow := len(tickets) > maxShow || state.ScrollOff[i] > 0

			if overflow {
				cardWidth = innerWidth - 1
				maxShow = computeMaxVisibleKanban(len(tickets), state.ScrollOff[i], cardWidth, availableHeight, expandedIdx)
			}

			var cardsContent strings.Builder

			if state.ScrollOff[i] > 0 {
				cardsContent.WriteString(state.Styles.EmptyColumn.Italic(true).Render(fmt.Sprintf("↑ %d more", state.ScrollOff[i])))
				cardsContent.WriteString("\n")
			}

			for j := state.ScrollOff[i]; j < len(tickets) && j < state.ScrollOff[i]+maxShow; j++ {
				isSelected := i == state.ColIndex && j == state.Cursors[i]
				isExpanded := j == expandedIdx

				card := NewTicketCardModel(tickets[j], isSelected, isExpanded, cardWidth, 0, state.Theme)
				cardsContent.WriteString(card.Render())

				if j < state.ScrollOff[i]+maxShow-1 || len(tickets) > state.ScrollOff[i]+maxShow {
					cardsContent.WriteString("\n")
				}
			}

			if len(tickets) > state.ScrollOff[i]+maxShow {
				remaining := len(tickets) - (state.ScrollOff[i] + maxShow)
				cardsContent.WriteString(state.Styles.EmptyColumn.Italic(true).Render(fmt.Sprintf("↓ %d more", remaining)))
			}

			if overflow {
				scrollBar := renderScrollBarKanban(state.ScrollOff[i], maxShow, len(tickets), availableHeight, state.Styles)
				content.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
					lipgloss.NewStyle().Width(cardWidth).Render(cardsContent.String()),
					scrollBar,
				))
			} else {
				content.WriteString(lipgloss.NewStyle().Width(cardWidth).Render(cardsContent.String()))
			}
		}

		colStyle := state.Styles.FocusedColumn
		if i != state.ColIndex {
			colStyle = state.Styles.BlurredColumn
		}
		colStyle = colStyle.Width(innerWidth + 2).Padding(0, 1)

		cols[i] = colStyle.Render(content.String())
	}

	tabBar := r.renderKanbanTabBar(state)
	board := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
	return lipgloss.JoinVertical(lipgloss.Top, tabBar, board)
}

func (r *Renderer) renderKanbanTabBar(state board.KanbanViewState) string {
	w := r.width
	if w < 10 {
		return ""
	}

	boardLabel := " Board "
	if state.Tab == board.TabBoard {
		boardLabel = lipgloss.NewStyle().
			Background(state.Theme.Primary).
			Foreground(state.Theme.Text).
			Bold(true).
			Render(boardLabel)
	} else {
		boardLabel = state.Styles.TabInactive.Render(boardLabel)
	}

	searchPrefix := state.Styles.SearchBox.Render("Search: ")
	searchQuery := state.Styles.SearchBox.Render(state.SearchQuery)
	if state.Tab == board.TabSearch {
		searchPrefix = lipgloss.NewStyle().Foreground(state.Theme.Primary).Bold(true).Render("Search: ")
		searchQuery = lipgloss.NewStyle().Foreground(state.Theme.Text).Bold(true).Render(state.SearchQuery)
	}

	boardWidth := lipgloss.Width(boardLabel)
	searchWidth := lipgloss.Width(searchPrefix) + lipgloss.Width(searchQuery)
	leftPad := 2
	gap := 4

	remaining := w - boardWidth - searchWidth - leftPad - gap
	if remaining < 1 {
		remaining = 1
	}

	return strings.Repeat(" ", leftPad) +
		boardLabel + strings.Repeat(" ", gap) +
		searchPrefix + searchQuery + strings.Repeat(" ", remaining)
}

func (r *Renderer) renderTicket(state board.TicketViewState) string {
	if r.width == 0 {
		return ""
	}

	if state.Ticket == nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("No ticket selected")
	}

	innerWidth := r.width - 6
	if innerWidth < 20 {
		innerWidth = 20
	}

	agentActiveColor := lipgloss.Color("240")
	if state.Ticket.AgentActive {
		agentActiveColor = lipgloss.Color("69")
	}
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(agentActiveColor).
		Padding(1, 2).
		Width(r.width)

	var b strings.Builder

	titleLine := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")).Render(
		fmt.Sprintf("%s  %s", state.Ticket.ID, state.Ticket.Title),
	)
	b.WriteString(titleLine)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", min(innerWidth, 60)))
	b.WriteString("\n\n")

	fields := []struct {
		label string
		value string
	}{
		{"ID", state.Ticket.ID},
		{"Title", state.Ticket.Title},
		{"Description", state.Ticket.Description},
		{"Status", state.Ticket.Status},
		{"Priority", state.Ticket.Priority},
		{"Agent", state.Ticket.Agent},
		{"Branch", state.Ticket.Branch},
	}

	for i, f := range fields {
		val := f.value
		if val == "" {
			val = "—"
		}

		prefix := "  "
		if i == state.Cursor {
			prefix = "▸ "
		}

		valWidth := innerWidth - 15
		if valWidth < 10 {
			valWidth = 10
		}

		labelStr := fmt.Sprintf("%-12s", f.label)
		labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
		prefixStyle := lipgloss.NewStyle()

		if i == state.Cursor {
			labelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
			prefixStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
		}

		renderedPrefix := prefixStyle.Render(prefix)
		renderedLabel := labelStyle.Render(labelStr)
		renderedVal := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Width(valWidth).Render(val)

		row := lipgloss.JoinHorizontal(lipgloss.Top, renderedPrefix, renderedLabel, " ", renderedVal)
		b.WriteString(row)
		b.WriteString("\n")
	}

	if state.Loading {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render("Loading..."))
	}

	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("↑/k: up │ ↓/j: down │ e: edit │ s: cycle status │ a: assign agent │ Esc: back")
	final := lipgloss.JoinVertical(lipgloss.Left, b.String(), footer)

	return borderStyle.Render(final)
}

func (r *Renderer) renderDashboard(state board.DashboardViewState) string {
	if r.width == 0 {
		return ""
	}

	sidebarWidth := 30
	contentWidth := r.width - sidebarWidth - 2
	if contentWidth < 20 {
		contentWidth = 20
	}

	sidebar := r.renderDashboardSidebar(state, sidebarWidth)
	content := r.renderDashboardContent(state, contentWidth)

	split := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)

	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")).Render("Agent Dashboard")
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(split)
	b.WriteString("\n\n")
	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("j/k: select │ r: refresh │ Esc: back")
	b.WriteString(footer)

	return b.String()
}

func (r *Renderer) renderDashboardSidebar(state board.DashboardViewState, width int) string {
	var b strings.Builder

	for i, agent := range state.Agents {
		if !agent.Found {
			continue
		}
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		if i == 0 {
			prefix = "▸ "
			style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
		}

		statusDot := " ●"
		statusColor := "240"
		if agent.Found {
			statusColor = "42"
		}
		dot := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(statusDot)

		row := prefix + agent.Name + dot
		b.WriteString(style.Width(width).Render(row))
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("240")).
		Height(r.height - 8).
		Width(width).
		Render(b.String())
}

func (r *Renderer) renderDashboardContent(state board.DashboardViewState, width int) string {
	if len(state.Agents) == 0 {
		return lipgloss.NewStyle().
			Width(width).
			Height(r.height - 8).
			Foreground(lipgloss.Color("240")).
			Render("No agents detected")
	}

	agent := state.Agents[0]

	var b strings.Builder
	logoColor := lipgloss.Color(agent.LogoClr)
	if !agent.Found {
		logoColor = lipgloss.Color("240")
	}
	logoStyle := lipgloss.NewStyle().Foreground(logoColor)
	b.WriteString(logoStyle.Render(agent.Logo))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")).Render(agent.Name))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", width))
	b.WriteString("\n\n")

	statusVal := "NOT INSTALLED"
	if agent.Found {
		statusVal = "READY"
	}

	fields := []struct {
		label string
		value string
	}{
		{"Status:", statusVal},
		{"Binary:", agent.Binary},
	}

	for _, f := range fields {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(fmt.Sprintf("%-12s", f.label)))
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(f.value))
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Padding(0, 2).
		Width(width).
		Height(r.height - 8).
		Render(b.String())
}

func (r *Renderer) renderHelp() string {
	if r.width == 0 {
		return ""
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2)

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")).Render("Help"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", r.width-4))
	b.WriteString("\n\n")

	shortcuts := []struct {
		key     string
		action  string
	}{
		{"h/l or ←/→", "Move between columns"},
		{"j/k or ↑/↓", "Move between tickets"},
		{"Enter", "Open ticket detail"},
		{"a", "Add new ticket"},
		{"d", "Delete ticket"},
		{"1-4", "Jump to column"},
		{"?", "Toggle help"},
		{"i", "Toggle dashboard"},
		{":", "Command palette"},
		{"q", "Quit"},
		{"Esc", "Return to board"},
	}

	for _, s := range shortcuts {
		keyStr := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")).Render(s.key)
		actionStr := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(s.action)
		b.WriteString(keyStr)
		b.WriteString("  ")
		b.WriteString(actionStr)
		b.WriteString("\n")
	}

	return borderStyle.Width(r.width).Render(b.String())
}

func computeMaxVisibleKanban(total int, scrollOff int, width int, height int, expandedIdx int) int {
	if total == 0 {
		return 0
	}

	hRemaining := height
	if scrollOff > 0 {
		hRemaining--
	}

	count := 0
	lines := 0
	for i := scrollOff; i < total; i++ {
		h := 4
		if i == expandedIdx {
			h = 6
		}

		hWithSpacer := h
		if i > scrollOff {
			hWithSpacer++
		}

		if lines+hWithSpacer > hRemaining {
			break
		}

		if i+1 < total && lines+hWithSpacer+1 > hRemaining {
			break
		}

		lines += hWithSpacer
		count++
	}

	if count < 1 && total > 0 {
		count = 1
	}
	return count
}

func renderScrollBarKanban(scrollOff int, maxVisible int, total int, height int, styles board.KanbanStyles) string {
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
			sb.WriteString(styles.TabActive.Render("┃"))
		} else {
			sb.WriteString(styles.TabInactive.Render("│"))
		}
		if i < height-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}