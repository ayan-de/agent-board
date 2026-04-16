package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

	"github.com/charmbracelet/lipgloss"
)

type TicketCardModel struct {
	ticket   store.Ticket
	selected bool
	expanded bool
	width    int
	frame    int
	styles   TicketCardStyles
	theme    *theme.Theme
}

type TicketCardStyles struct {
	SelectedBorder lipgloss.Style
	NormalBorder   lipgloss.Style
	Title          lipgloss.Style
	Description    lipgloss.Style
	Metadata       lipgloss.Style
	PriorityColors map[string]lipgloss.Color
}

func DefaultTicketCardStyles() TicketCardStyles {
	return TicketCardStyles{
		SelectedBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("69")),
		NormalBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
		Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")),
		Description: lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		Metadata:    lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		PriorityColors: map[string]lipgloss.Color{
			"low":      lipgloss.Color("240"),
			"medium":   lipgloss.Color("252"),
			"high":     lipgloss.Color("214"),
			"critical": lipgloss.Color("196"),
		},
	}
}

func NewTicketCardStyles(t *theme.Theme) TicketCardStyles {
	return TicketCardStyles{
		SelectedBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary),
		NormalBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border),
		Title:       lipgloss.NewStyle().Bold(true).Foreground(t.Text),
		Description: lipgloss.NewStyle().Foreground(t.Text),
		Metadata:    lipgloss.NewStyle().Foreground(t.TextMuted),
		PriorityColors: map[string]lipgloss.Color{
			"low":      t.TextMuted,
			"medium":   t.Text,
			"high":     t.Warning,
			"critical": t.Error,
		},
	}
}

func NewTicketCardModel(ticket store.Ticket, selected bool, expanded bool, width int, frame int, t *theme.Theme) TicketCardModel {
	var styles TicketCardStyles
	if t != nil {
		styles = NewTicketCardStyles(t)
	} else {
		styles = DefaultTicketCardStyles()
	}
	return TicketCardModel{
		ticket:   ticket,
		selected: selected,
		expanded: expanded,
		width:    width,
		frame:    frame,
		styles:   styles,
		theme:    t,
	}
}

func (c TicketCardModel) CompactHeight() int {
	return 3
}

func (c TicketCardModel) ExpandedHeight() int {
	innerWidth := c.innerWidth()
	if innerWidth < 4 {
		innerWidth = 4
	}
	descLines := 1
	if c.ticket.Description != "" {
		descLines = (utf8.RuneCountInString(c.ticket.Description) + innerWidth - 1) / innerWidth
		if descLines < 1 {
			descLines = 1
		}
	}
	return 3 + descLines + 1
}

func (c TicketCardModel) Render() string {
	if c.expanded {
		return c.renderExpanded()
	}
	return c.renderCompact()
}

func (c TicketCardModel) innerWidth() int {
	if c.width < 4 {
		return 2
	}
	return c.width - 2
}

func (c TicketCardModel) renderCompact() string {
	iw := c.innerWidth()

	titleText := c.ticket.ID + " " + truncateRunes(c.ticket.Title, iw-utf8.RuneCountInString(c.ticket.ID)-1)
	titleLine := c.styles.Title.Render(titleText)

	descText := ""
	if c.ticket.Description != "" {
		descText = truncateRunes(c.ticket.Description, iw-1)
		if utf8.RuneCountInString(c.ticket.Description) > iw-1 {
			descText = descText + "…"
		}
	}
	descLine := c.styles.Description.Render(descText)

	footerLine := c.renderFooter(iw)

	content := titleLine + "\n" + descLine + "\n" + footerLine

	borderStyle := c.styles.NormalBorder
	if c.selected {
		borderStyle = c.styles.SelectedBorder
	}

	return borderStyle.Width(iw).Render(content)
}

func (c TicketCardModel) renderExpanded() string {
	iw := c.innerWidth()

	titleLine := c.styles.Title.Render(c.ticket.ID + " " + c.ticket.Title)

	sepLine := strings.Repeat("─", iw)

	var descLines string
	if c.ticket.Description != "" {
		descLines = wrapText(c.ticket.Description, iw)
	}

	footerLine := c.renderFooter(iw)

	parts := []string{titleLine, sepLine}
	if descLines != "" {
		parts = append(parts, descLines)
	}
	parts = append(parts, "", footerLine)

	content := strings.Join(parts, "\n")

	borderStyle := c.styles.NormalBorder
	if c.selected {
		borderStyle = c.styles.SelectedBorder
	}

	return borderStyle.Width(iw).Render(content)
}

func (c TicketCardModel) renderFooter(width int) string {
	priorityColor, ok := c.styles.PriorityColors[c.ticket.Priority]
	if !ok {
		priorityColor = lipgloss.Color("252")
	}
	priorityIndicator := lipgloss.NewStyle().Foreground(priorityColor).Render(fmt.Sprintf("⬥ %s", c.ticket.Priority))

	left := priorityIndicator
	right := ""

	if c.ticket.AgentActive {
		barWidth := width / 2
		if barWidth < 4 {
			barWidth = 4
		}
		bar := ActivityBar(c.frame, barWidth, c.theme)
		dot := agentDot(c.ticket.Agent, true)
		right = bar + " " + dot
	} else if c.ticket.Agent != "" {
		dot := agentDot(c.ticket.Agent, false)
		right = dot + " " + c.ticket.Agent
	}

	if right == "" {
		return left
	}

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := width - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

func truncateRunes(s string, maxLen int) string {
	if maxLen < 0 {
		maxLen = 0
	}
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen])
}

func wrapText(s string, width int) string {
	if width < 1 {
		return s
	}
	var b strings.Builder
	runes := []rune(s)
	pos := 0
	for pos < len(runes) {
		end := pos + width
		if end > len(runes) {
			end = len(runes)
		}
		b.WriteString(string(runes[pos:end]))
		if end < len(runes) {
			b.WriteString("\n")
		}
		pos = end
	}
	return b.String()
}
