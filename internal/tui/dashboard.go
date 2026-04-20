package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/orchestrator"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
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
	PaneContent lipgloss.Style
}

type DashboardModel struct {
	store          *store.Store
	orchestrator   Orchestrator
	resolver       *keybinding.Resolver
	Agents         []config.DetectedAgent
	ActiveSessions map[string]store.Session
	width          int
	height         int
	refreshed      bool
	styles         DashboardStyles
	cursor         int
	input          textinput.Model
	isInput        bool

	// For pane management
	activeAgentSessions []*orchestrator.AgentSession
	selectedSessionID   string
	paneContent         string
	paneContentLoadedAt time.Time
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
		PaneContent: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("235")).
			Padding(1),
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
		PaneContent: lipgloss.NewStyle().
			Foreground(t.Text).
			Background(t.Background).
			Padding(1),
	}
}

func NewDashboardModel(s *store.Store, orch Orchestrator, resolver *keybinding.Resolver, Agents []config.DetectedAgent, t *theme.Theme) DashboardModel {
	ti := textinput.New()
	ti.Placeholder = "Type to send to agent..."
	ti.CharLimit = 156
	ti.Width = 40

	return DashboardModel{
		store:        s,
		orchestrator: orch,
		resolver:     resolver,
		Agents:       Agents,
		styles:       NewDashboardStyles(t),
		input:        ti,
	}
}

func (m DashboardModel) SelectedAgent() config.DetectedAgent {
	if m.cursor >= 0 && m.cursor < len(m.Agents) {
		return m.Agents[m.cursor]
	}
	return config.DetectedAgent{}
}

func (m DashboardModel) SelectedSession() *orchestrator.AgentSession {
	if m.selectedSessionID != "" {
		for _, sess := range m.activeAgentSessions {
			if sess.SessionID == m.selectedSessionID {
				return sess
			}
		}
	}
	if len(m.activeAgentSessions) > 0 && m.cursor < len(m.activeAgentSessions) {
		return m.activeAgentSessions[m.cursor]
	}
	return nil
}

func (m DashboardModel) visibleAgents() []config.DetectedAgent {
	agents := make([]config.DetectedAgent, 0, len(m.Agents))
	for _, agent := range m.Agents {
		if agent.Found || m.sessionForAgent(agent) != nil {
			agents = append(agents, agent)
		}
	}
	return agents
}

func (m DashboardModel) sessionForAgent(agent config.DetectedAgent) *store.Session {
	if sess, ok := m.ActiveSessions[agent.Binary]; ok {
		s := sess
		return &s
	}
	if sess, ok := m.ActiveSessions[agent.Name]; ok {
		s := sess
		return &s
	}
	return nil
}

func (m DashboardModel) refreshPaneContent() DashboardModel {
	sess := m.SelectedSession()
	if sess == nil {
		m.paneContent = ""
		m.paneContentLoadedAt = time.Time{}
		return m
	}

	rows := m.height - 12
	if rows < 8 {
		rows = 8
	}
	cols := m.width - 36
	if cols < 40 {
		cols = 40
	}
	_ = m.orchestrator.SetTerminalSize(sess.SessionID, rows, cols)

	content, err := m.orchestrator.GetPTYOutput(sess.SessionID, rows)
	if err != nil {
		return m
	}

	m.paneContent = content
	m.paneContentLoadedAt = time.Now()
	return m
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
					sess := m.SelectedSession()
					if sess != nil {
						_ = m.orchestrator.SendInput(sess.SessionID, val)
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
		if m.activeAgentSessions == nil || len(m.activeAgentSessions) == 0 {
			if m.cursor >= len(m.Agents) {
				m.cursor = 0
			}
		} else {
			if m.cursor >= len(m.activeAgentSessions) {
				m.cursor = 0
			}
			// Update selected session
			if m.cursor < len(m.activeAgentSessions) {
				m.selectedSessionID = m.activeAgentSessions[m.cursor].SessionID
			}
		}
	case keybinding.ActionPrevTicket:
		m.cursor--
		if m.activeAgentSessions == nil || len(m.activeAgentSessions) == 0 {
			if m.cursor < 0 {
				m.cursor = len(m.Agents) - 1
			}
		} else {
			if m.cursor < 0 {
				m.cursor = len(m.activeAgentSessions) - 1
			}
			if m.cursor >= 0 && m.cursor < len(m.activeAgentSessions) {
				m.selectedSessionID = m.activeAgentSessions[m.cursor].SessionID
			}
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
	m.ActiveSessions = make(map[string]store.Session, len(sessions))
	for _, s := range sessions {
		m.ActiveSessions[s.Agent] = s
	}

	// Also get active agent sessions from orchestrator (for TmuxRunner)
	m.activeAgentSessions = m.orchestrator.GetActiveSessions()
	if len(m.activeAgentSessions) > 0 && m.selectedSessionID == "" {
		m.selectedSessionID = m.activeAgentSessions[0].SessionID
	}
	if m.selectedSessionID != "" {
		found := false
		for _, sess := range m.activeAgentSessions {
			if sess.SessionID == m.selectedSessionID {
				found = true
				break
			}
		}
		if !found {
			m.selectedSessionID = ""
		}
	}
	if len(m.activeAgentSessions) == 0 && len(m.ActiveSessions) > 0 {
		currentRunning := false
		if m.cursor >= 0 && m.cursor < len(m.Agents) {
			currentRunning = m.sessionForAgent(m.Agents[m.cursor]) != nil
		}
		if !currentRunning {
			for i, agent := range m.Agents {
				if m.sessionForAgent(agent) != nil {
					m.cursor = i
					break
				}
			}
		}
	}
}

func (m DashboardModel) Refresh() DashboardModel {
	detected := config.DetectAgents()
	if len(m.Agents) == 0 {
		m.Agents = detected
	} else {
		merged := make([]config.DetectedAgent, len(m.Agents))
		copy(merged, m.Agents)
		byBinary := make(map[string]config.DetectedAgent, len(detected))
		byName := make(map[string]config.DetectedAgent, len(detected))
		for _, agent := range detected {
			byBinary[agent.Binary] = agent
			byName[agent.Name] = agent
		}
		for i, agent := range merged {
			if updated, ok := byBinary[agent.Binary]; ok {
				merged[i].Found = updated.Found
				merged[i].Path = updated.Path
				if updated.Logo != "" {
					merged[i].Logo = updated.Logo
				}
				if updated.LogoClr != "" {
					merged[i].LogoClr = updated.LogoClr
				}
				continue
			}
			if updated, ok := byName[agent.Name]; ok {
				merged[i].Found = updated.Found
				merged[i].Path = updated.Path
				if updated.Logo != "" {
					merged[i].Logo = updated.Logo
				}
				if updated.LogoClr != "" {
					merged[i].LogoClr = updated.LogoClr
				}
			}
		}
		m.Agents = merged
	}
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
	footer := m.styles.Footer.Render(footerStr)
	b.WriteString(footer)

	return b.String()
}

func (m DashboardModel) renderSidebar(width int) string {
	var b strings.Builder

	// If we have active agent sessions, show those instead of detected agents
	sessions := m.activeAgentSessions
	if len(sessions) == 0 {
		agents := m.visibleAgents()
		if len(agents) == 0 {
			b.WriteString(m.styles.Placeholder.Render("No agents found"))
			return lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, true, false, false).
				BorderForeground(lipgloss.Color("240")).
				Height(m.height - 8).
				Width(width).
				Render(b.String())
		}

		if m.cursor >= len(agents) {
			m.cursor = len(agents) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}

		for i, agent := range agents {
			prefix := "  "
			style := m.styles.Label
			if i == m.cursor && len(sessions) == 0 {
				prefix = "▸ "
				style = m.styles.Title
			}

			statusDot := " ●"
			statusColor := "240"
			if agent.Found {
				statusColor = "42"
				if m.sessionForAgent(agent) != nil {
					statusColor = "213"
				}
			}
			dot := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(statusDot)

			row := prefix + agent.Name + dot
			b.WriteString(style.Width(width).Render(row))
			b.WriteString("\n")
		}
	} else {
		// Show active agent sessions
		for i, sess := range sessions {
			prefix := "  "
			style := m.styles.Label
			if i == m.cursor {
				prefix = "▸ "
				style = m.styles.Title
			}

			// Map agent name to display name
			agentName := sess.Agent
			agentLogo := "●"
			agentColor := "213" // running color

			for _, a := range m.Agents {
				if a.Binary == sess.Agent || a.Name == sess.Agent {
					agentName = a.Name
					agentLogo = a.Logo
					agentColor = "42"
					break
				}
			}

			dot := lipgloss.NewStyle().Foreground(lipgloss.Color(agentColor)).Render(agentLogo)
			row := prefix + agentName + " (" + sess.TicketID + ")" + dot
			b.WriteString(style.Width(width).Render(row))
			b.WriteString("\n")
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("240")).
		Height(m.height - 8).
		Width(width).
		Render(b.String())
}

func (m DashboardModel) renderContent(width int) string {
	sess := m.SelectedSession()
	if sess == nil {
		agents := m.visibleAgents()
		if len(agents) == 0 {
			return m.styles.Placeholder.Width(width).Render("No agents found")
		}
		if m.cursor >= len(agents) {
			m.cursor = len(agents) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}

		agent := agents[m.cursor]
		if agent.Binary == "" {
			return m.styles.Placeholder.Width(width).Render("No agent selected")
		}

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

		runningVal := "no"
		ticketVal := "—"
		uptimeVal := "—"
		if runningSess := m.sessionForAgent(agent); runningSess != nil {
			statusVal = "RUNNING"
			runningVal = "yes"
			ticketVal = runningSess.TicketID
			uptimeVal = formatUptime(runningSess.StartedAt)
		}

		fields := []struct {
			label string
			value string
		}{
			{"Status:", statusVal},
			{"Binary:", agent.Binary},
			{"Running:", runningVal},
			{"Ticket:", ticketVal},
			{"Uptime:", uptimeVal},
			{"Tmux:", "—"},
			{"Subagents:", "—"},
			{"Tokens:", "—"},
		}
		for _, f := range fields {
			b.WriteString(m.styles.Label.Render(f.label + " "))
			if f.value == "—" {
				b.WriteString(m.styles.Placeholder.Render(f.value))
			} else {
				b.WriteString(m.styles.Value.Render(f.value))
			}
			b.WriteString("\n")
		}

		return lipgloss.NewStyle().
			Padding(0, 2).
			Width(width).
			Height(m.height - 8).
			Render(b.String())
	}

	// We have an active session - show its pane content
	var b strings.Builder

	// Session info
	b.WriteString(m.styles.Title.Render(fmt.Sprintf("%s - %s", sess.Agent, sess.TicketID)))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", width))
	b.WriteString("\n\n")

	if sess.Target != "" {
		b.WriteString(m.styles.Label.Render("Tmux: "))
		b.WriteString(m.styles.Value.Render(sess.Target))
		b.WriteString("\n")
		b.WriteString(m.styles.Label.Render("Attach: "))
		b.WriteString(m.styles.Value.Render("tmux attach -t " + sess.Target))
		b.WriteString("\n\n")
	}

	if sess.Prompt != "" {
		b.WriteString(m.styles.Label.Render("Prompt:"))
		b.WriteString("\n")
		promptLines := strings.Split(sess.Prompt, "\n")
		for _, line := range promptLines {
			b.WriteString(m.styles.Value.Render("  " + line))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Get pane content
	paneContent := m.paneContent
	if time.Since(m.paneContentLoadedAt) > 500*time.Millisecond {
		// Refresh pane content
		if content, err := m.orchestrator.GetPTYOutput(sess.SessionID, 30); err == nil {
			m.paneContent = content
			m.paneContentLoadedAt = time.Now()
			paneContent = content
		}
	}

	if paneContent == "" {
		// Try to get it now
		if content, err := m.orchestrator.GetPTYOutput(sess.SessionID, 30); err == nil {
			m.paneContent = content
			m.paneContentLoadedAt = time.Now()
			paneContent = content
		}
	}

	if paneContent != "" {
		b.WriteString(m.styles.Title.Render("Live Agent Output"))
		b.WriteString("\n")
		b.WriteString(paneContent)
		if !strings.HasSuffix(paneContent, "\n") {
			b.WriteString("\n")
		}
	} else {
		b.WriteString(m.styles.Placeholder.Render("  Starting agent..."))
		b.WriteString("\n")
	}

	b.WriteString("\n")

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

	if sess, ok := m.ActiveSessions[agent.Binary]; ok {
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
		{"SubAgents:", "—"},
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
