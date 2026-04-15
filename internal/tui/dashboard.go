package tui

import (
	"fmt"
	"strings"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DashboardStyles struct {
	Border      lipgloss.Style
	Title       lipgloss.Style
	CardFound   lipgloss.Style
	CardMissing lipgloss.Style
	Label       lipgloss.Style
	Value       lipgloss.Style
	Placeholder lipgloss.Style
	Footer      lipgloss.Style
}

type DashboardModel struct {
	store     *store.Store
	resolver  *keybinding.Resolver
	agents    []config.DetectedAgent
	width     int
	height    int
	refreshed bool
	styles    DashboardStyles
}

func DefaultDashboardStyles() DashboardStyles {
	return DashboardStyles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("69")),
		CardFound: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("42")).
			Padding(0, 1).
			Width(30),
		CardMissing: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Width(30),
		Label: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		Value: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		Placeholder: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		Footer: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
	}
}

func NewDashboardModel(s *store.Store, resolver *keybinding.Resolver, agents []config.DetectedAgent) DashboardModel {
	return DashboardModel{
		store:    s,
		resolver: resolver,
		agents:   agents,
		styles:   DefaultDashboardStyles(),
	}
}

func (m DashboardModel) Init() tea.Cmd {
	return nil
}

func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
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

func (m DashboardModel) handleKey(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
	key := msg.String()
	action, _ := m.resolver.Resolve(key)

	switch action {
	case keybinding.ActionRefresh:
		m = m.Refresh()
	}

	return m, nil
}

func (m DashboardModel) Refresh() DashboardModel {
	m.agents = config.DetectAgents()
	m.refreshed = true
	return m
}

func (m DashboardModel) View() string {
	if m.width == 0 {
		return ""
	}

	var b strings.Builder

	title := m.styles.Title.Render("Agent Dashboard")
	b.WriteString(title)
	b.WriteString("\n\n")

	var found []config.DetectedAgent
	for _, a := range m.agents {
		if a.Found {
			found = append(found, a)
		}
	}

	if len(found) == 0 {
		b.WriteString(m.styles.Placeholder.Render("No agents found on $PATH"))
		b.WriteString("\n\n")
		footer := m.styles.Footer.Render("r: refresh | Esc: back")
		b.WriteString(footer)
		return b.String()
	}

	cards := make([]string, len(found))
	for i, agent := range found {
		cards[i] = m.renderCard(agent)
	}

	innerWidth := m.width - 4
	cardWidth := 32
	cardsPerRow := innerWidth / cardWidth
	if cardsPerRow < 1 {
		cardsPerRow = 1
	}

	for rowStart := 0; rowStart < len(cards); rowStart += cardsPerRow {
		rowEnd := rowStart + cardsPerRow
		if rowEnd > len(cards) {
			rowEnd = len(cards)
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, cards[rowStart:rowEnd]...)
		b.WriteString(row)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	footer := m.styles.Footer.Render("r: refresh | Esc: back")
	b.WriteString(footer)

	return b.String()
}

func (m DashboardModel) renderCard(agent config.DetectedAgent) string {
	var b strings.Builder

	name := m.styles.Title.Render(agent.Name)
	b.WriteString(name)
	b.WriteString("\n")

	statusVal := "not found"
	if agent.Found {
		statusVal = "installed"
	}

	fields := []struct {
		label string
		value string
	}{
		{"Status:", statusVal},
		{"Running:", "no"},
		{"Ticket:", "—"},
		{"Uptime:", "—"},
		{"Subagents:", "—"},
		{"Tokens:", "—"},
	}

	for _, f := range fields {
		label := m.styles.Label.Render(f.label)
		var val string
		if f.value == "—" {
			val = m.styles.Placeholder.Render(f.value)
		} else {
			val = m.styles.Value.Render(f.value)
		}
		fmt.Fprintf(&b, "%s %s\n", label, val)
	}

	style := m.styles.CardMissing
	if agent.Found {
		style = m.styles.CardFound
	}

	return style.Render(b.String())
}
