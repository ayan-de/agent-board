package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"
	"github.com/charmbracelet/bubbles/textinput"

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
	store          *store.Store
	orchestrator   Orchestrator
	resolver       *keybinding.Resolver
	agents         []config.DetectedAgent
	activeSessions map[string]store.Session
	width          int
	height         int
	refreshed      bool
	styles         DashboardStyles
	cursor         int
	input          textinput.Model
	isInput        bool
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

func NewDashboardStyles(t *theme.Theme) DashboardStyles {
	return DashboardStyles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary),
		CardFound: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Success).
			Padding(0, 1).
			Width(30),
		CardMissing: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.TextMuted).
			Padding(0, 1).
			Width(30),
		Label: lipgloss.NewStyle().
			Foreground(t.Text),
		Value: lipgloss.NewStyle().
			Foreground(t.Text),
		Placeholder: lipgloss.NewStyle().
			Foreground(t.TextMuted),
		Footer: lipgloss.NewStyle().
			Foreground(t.TextMuted),
	}
}

func NewDashboardModel(s *store.Store, orch Orchestrator, resolver *keybinding.Resolver, agents []config.DetectedAgent, t *theme.Theme) DashboardModel {
	ti := textinput.New()
	ti.Placeholder = "Type command/answer..."
	ti.CharLimit = 156
	ti.Width = 40

	return DashboardModel{
		store:        s,
		orchestrator: orch,
		resolver:     resolver,
		agents:       agents,
		styles:       NewDashboardStyles(t),
		input:        ti,
	}
}

func (m DashboardModel) Init() tea.Cmd {
	return nil
}

func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	if m.isInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if key, ok := msg.(tea.KeyMsg); ok {
			if key.String() == "enter" {
				val := m.input.Value()
				if val != "" {
					agent := m.agents[m.cursor]
					if sess, running := m.activeSessions[agent.Binary]; running {
						_ = m.orchestrator.SendInput(sess.ID, val)
					}
					m.input.SetValue("")
				}
				m.isInput = false
				m.input.Blur()
			} else if key.String() == "esc" {
				m.isInput = false
				m.input.Blur()
			}
		}
		return m, cmd
	}

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
	case keybinding.ActionNextTicket:
		m.cursor++
		if m.cursor >= len(m.agents) {
			m.cursor = 0
		}
	case keybinding.ActionPrevTicket:
		m.cursor--
		if m.cursor < 0 {
			m.cursor = len(m.agents) - 1
		}
	case keybinding.ActionInteract:
		agent := m.agents[m.cursor]
		if _, running := m.activeSessions[agent.Binary]; running {
			m.isInput = true
			return m, m.input.Focus()
		}
	}

	return m, nil
}

func (m *DashboardModel) loadActiveSessions() {
	if m.store == nil {
		return
	}
	sessions, err := m.store.ListActiveSessions(context.Background())
	if err != nil {
		return
	}
	m.activeSessions = make(map[string]store.Session, len(sessions))
	for _, s := range sessions {
		m.activeSessions[s.Agent] = s
	}
}

func (m DashboardModel) Refresh() DashboardModel {
	m.agents = config.DetectAgents()
	m.loadActiveSessions()
	m.refreshed = true
	return m
}

func (m DashboardModel) View() string {
	if m.width == 0 {
		return ""
	}

	m.loadActiveSessions()

	sidebarWidth := 30
	contentWidth := m.width - sidebarWidth - 2
	if contentWidth < 20 {
		contentWidth = 20
	}

	sidebar := m.renderSidebar(sidebarWidth)
	content := m.renderContent(contentWidth)

	split := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)

	var b strings.Builder
	title := m.styles.Title.Render("Agent Dashboard")
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(split)
	b.WriteString("\n\n")
	footerStr := "j/k: select │ r: refresh │ Esc: back"
	if m.cursor >= 0 && m.cursor < len(m.agents) {
		agent := m.agents[m.cursor]
		if _, running := m.activeSessions[agent.Binary]; running {
			footerStr += " │ e: interact"
		}
	}
	footer := m.styles.Footer.Render(footerStr)
	b.WriteString(footer)

	return b.String()
}

func (m DashboardModel) renderSidebar(width int) string {
	var b strings.Builder
	for i, agent := range m.agents {
		prefix := "  "
		style := m.styles.Label
		if i == m.cursor {
			prefix = "▸ "
			style = m.styles.Title
		}

		statusDot := " ●"
		statusColor := "240"
		if agent.Found {
			statusColor = "42"
			if _, running := m.activeSessions[agent.Binary]; running {
				statusColor = "213"
			}
		}
		dot := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(statusDot)

		row := prefix + agent.Name + dot
		b.WriteString(style.Width(width).Render(row))
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("240")).
		Height(m.height - 8).
		Width(width).
		Render(b.String())
}

func (m DashboardModel) renderContent(width int) string {
	if m.cursor < 0 || m.cursor >= len(m.agents) {
		return m.styles.Placeholder.Width(width).Render("No agent selected")
	}

	agent := m.agents[m.cursor]
	var b strings.Builder

	logoColor := lipgloss.Color(agent.LogoClr)
	if !agent.Found {
		logoColor = lipgloss.Color("240")
	}
	logoStyle := lipgloss.NewStyle().Foreground(logoColor)
	b.WriteString(logoStyle.Render(agent.Logo))
	b.WriteString("\n\n")

	b.WriteString(m.styles.Title.Render(agent.Name))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", width))
	b.WriteString("\n\n")

	statusVal := "NOT INSTALLED"
	if agent.Found {
		statusVal = "READY"
	}

	sess, running := m.activeSessions[agent.Binary]
	if running {
		statusVal = "RUNNING"
	}

	fields := []struct {
		label string
		value string
	}{
		{"Status:", statusVal},
		{"Binary:", agent.Binary},
	}

	if running {
		fields = append(fields, []struct {
			label string
			value string
		}{
			{"Ticket:", sess.TicketID},
			{"Uptime:", formatUptime(sess.StartedAt)},
		}...)
	}

	for _, f := range fields {
		b.WriteString(m.styles.Label.Render(fmt.Sprintf("%-12s", f.label)))
		b.WriteString(m.styles.Value.Render(f.value))
		b.WriteString("\n")
	}

	if running {
		b.WriteString("\n")
		b.WriteString(m.styles.Title.Render("Live Output"))
		b.WriteString("\n")
		logs := m.orchestrator.GetLogs(sess.ID)
		if len(logs) == 0 {
			b.WriteString(m.styles.Placeholder.Render("  (Waiting for output...)"))
		} else {
			start := 0
			if len(logs) > 20 {
				start = len(logs) - 20
			}
			for _, line := range logs[start:] {
				// Truncate line if too long
				displayLine := line
				if len(displayLine) > width-4 {
					displayLine = displayLine[:width-7] + "..."
				}
				b.WriteString(m.styles.Value.Render("  " + displayLine))
				b.WriteString("\n")
			}
		}

		b.WriteString("\n")
		b.WriteString(m.styles.Label.Render("Command: "))
		b.WriteString(m.input.View())
		b.WriteString("\n")
	} else if agent.Found {
		b.WriteString("\n")
		b.WriteString(m.styles.Placeholder.Render("Agent is idle. Assign it to a ticket and set status to 'in_progress' to start."))
	}

	return lipgloss.NewStyle().
		Padding(0, 2).
		Width(width).
		Height(m.height - 8).
		Render(b.String())
}

func formatUptime(since time.Time) string {
	d := time.Since(since)
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

func (m DashboardModel) renderCard(agent config.DetectedAgent) string {
	logoColor := lipgloss.Color(agent.LogoClr)
	if !agent.Found {
		logoColor = lipgloss.Color("240")
	}

	logoStyle := lipgloss.NewStyle().Foreground(logoColor)
	logoBlock := logoStyle.Render(agent.Logo)

	name := m.styles.Title.Render(agent.Name)
	dot := lipgloss.NewStyle().Foreground(logoColor).Render(" ●")
	var infoBuilder strings.Builder
	infoBuilder.WriteString(name)
	infoBuilder.WriteString(dot)
	infoBuilder.WriteString("\n")

	statusVal := "not found"
	if agent.Found {
		statusVal = "installed"
	}

	runningVal := "no"
	ticketVal := "—"
	uptimeVal := "—"

	if sess, ok := m.activeSessions[agent.Binary]; ok {
		runningVal = "yes"
		ticketVal = sess.TicketID
		uptimeVal = formatUptime(sess.StartedAt)
	}

	fields := []struct {
		label string
		value string
	}{
		{"Status:", statusVal},
		{"Running:", runningVal},
		{"Ticket:", ticketVal},
		{"Uptime:", uptimeVal},
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
		fmt.Fprintf(&infoBuilder, "%s %s\n", label, val)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, logoBlock, "  ", infoBuilder.String())

	style := m.styles.CardMissing
	if agent.Found {
		style = m.styles.CardFound
	}

	return style.Width(38).Render(row)
}
