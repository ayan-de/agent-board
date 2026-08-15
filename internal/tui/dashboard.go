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
	Selected    lipgloss.Style
}

type DashboardModel struct {
	store          *store.Store
	orchestrator   Orchestrator
	resolver       *keybinding.Resolver
	Agents         []config.DetectedAgent
	ActiveSessions map[string]store.Session // agent binary -> session
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

type agentSessionGroup struct {
	Agent    string
	Sessions []*orchestrator.AgentSession
}

func (m DashboardModel) aggregateActiveSessions() []agentSessionGroup {
	indexByAgent := make(map[string]int)
	var groups []agentSessionGroup
	for _, a := range m.Agents {
		if !a.Found {
			continue
		}
		idx := len(groups)
		indexByAgent[a.Binary] = idx
		indexByAgent[a.Name] = idx
		groups = append(groups, agentSessionGroup{Agent: a.Binary})
	}
	for _, sess := range m.activeAgentSessions {
		idx, ok := indexByAgent[sess.Agent]
		if !ok {
			idx = len(groups)
			indexByAgent[sess.Agent] = idx
			groups = append(groups, agentSessionGroup{Agent: sess.Agent})
		}
		groups[idx].Sessions = append(groups[idx].Sessions, sess)
	}
	return groups
}

func (m DashboardModel) displayNameFor(binary string) string {
	for _, a := range m.Agents {
		if a.Binary == binary {
			return a.Name
		}
	}
	return binary
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
		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("235")).
			Background(lipgloss.Color("69")),
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
		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Text).
			Background(t.Primary),
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
	groups := m.aggregateActiveSessions()
	if m.cursor < 0 || m.cursor >= len(groups) {
		return config.DetectedAgent{}
	}
	bin := groups[m.cursor].Agent
	for _, a := range m.Agents {
		if a.Binary == bin || a.Name == bin {
			return a
		}
	}
	return config.DetectedAgent{Name: m.displayNameFor(bin), Binary: bin}
}

func (m DashboardModel) SelectedSession() *orchestrator.AgentSession {
	if m.selectedSessionID != "" {
		for _, sess := range m.activeAgentSessions {
			if sess.SessionID == m.selectedSessionID {
				return sess
			}
		}
	}
	groups := m.aggregateActiveSessions()
	if m.cursor < 0 || m.cursor >= len(groups) {
		return nil
	}
	sessions := groups[m.cursor].Sessions
	if len(sessions) == 0 {
		return nil
	}
	return sessions[0]
}

func (m DashboardModel) selectedGroup() *agentSessionGroup {
	groups := m.aggregateActiveSessions()
	if m.cursor < 0 || m.cursor >= len(groups) {
		return nil
	}
	return &groups[m.cursor]
}

var jumpSessionActions = map[keybinding.Action]int{
	keybinding.ActionJumpSession1: 0,
	keybinding.ActionJumpSession2: 1,
	keybinding.ActionJumpSession3: 2,
	keybinding.ActionJumpSession4: 3,
	keybinding.ActionJumpSession5: 4,
	keybinding.ActionJumpSession6: 5,
	keybinding.ActionJumpSession7: 6,
	keybinding.ActionJumpSession8: 7,
	keybinding.ActionJumpSession9: 8,
}

func (m *DashboardModel) JumpToSession(index int) {
	groups := m.aggregateActiveSessions()
	sessionCount := 0
	for i, g := range groups {
		if index < sessionCount+len(g.Sessions) {
			m.cursor = i
			if len(g.Sessions) > 0 {
				m.selectedSessionID = g.Sessions[index-sessionCount].SessionID
			}
			return
		}
		sessionCount += len(g.Sessions)
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

	if index, ok := jumpSessionActions[action]; ok {
		m.JumpToSession(index)
		return m, nil
	}

	switch action {
	case keybinding.ActionRefresh:
		if m.selectedSessionID != "" {
			_ = m.orchestrator.SwitchToPane(m.selectedSessionID)
		} else {
			m = m.Refresh()
		}
	case keybinding.ActionNextTicket:
		m.cursor++
		groups := m.aggregateActiveSessions()
		if len(groups) == 0 {
			m.cursor = 0
		} else if m.cursor >= len(groups) {
			m.cursor = 0
		}
		if g := m.selectedGroup(); g != nil && len(g.Sessions) > 0 {
			m.selectedSessionID = g.Sessions[0].SessionID
		}
	case keybinding.ActionPrevTicket:
		m.cursor--
		groups := m.aggregateActiveSessions()
		if len(groups) == 0 {
			m.cursor = 0
		} else if m.cursor < 0 {
			m.cursor = len(groups) - 1
		}
		if g := m.selectedGroup(); g != nil && len(g.Sessions) > 0 {
			m.selectedSessionID = g.Sessions[0].SessionID
		}
	case keybinding.ActionInteract:
		sess := m.SelectedSession()
		if sess != nil {
			m.isInput = true
			return m, m.input.Focus()
		}
	case keybinding.ActionSwitchToPane:
		sess := m.SelectedSession()
		if sess != nil {
			// Switch tmux to show the agent's pane
			_ = m.orchestrator.SwitchToPane(sess.SessionID)
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
}

func (m DashboardModel) Refresh() DashboardModel {
	m.Agents = config.DetectAgents()
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
	sess := m.SelectedSession()
	if sess != nil {
		footerStr += " │ e: send input │ w: switch to pane"
	}
	footer := m.styles.Footer.Render(footerStr)
	b.WriteString(footer)

	return b.String()
}

// renderSidebar renders one entry per agent spec: the full ASCII logo on
// top, followed by a single line with the agent name and its active session
// count. All 5 providers are shown in fixed spec order, regardless of
// whether their binary is installed; uninstalled entries are dimmed. The
// cursor-selected provider is highlighted with a theme background color
// spanning the whole entry (logo lines and the name/count line).
func (m DashboardModel) renderSidebar(width int) string {
	var b strings.Builder

	groups := m.aggregateActiveSessions()
	indexByBinary := make(map[string]int, len(groups))
	for i, g := range groups {
		indexByBinary[g.Agent] = i
	}

	foundByBinary := make(map[string]bool, len(m.Agents))
	for _, a := range m.Agents {
		foundByBinary[a.Binary] = a.Found
	}

	for _, spec := range config.Specs() {
		logoColor := lipgloss.Color(spec.LogoClr)
		nameStyle := m.styles.Label
		prefix := "  "
		count := 0
		selected := false
		if idx, ok := indexByBinary[spec.Binary]; ok {
			count = len(groups[idx].Sessions)
			if idx == m.cursor {
				selected = true
				nameStyle = m.styles.Title
				prefix = "▸ "
			}
		}
		if !foundByBinary[spec.Binary] {
			logoColor = lipgloss.Color("240")
			nameStyle = m.styles.Placeholder
		}

		logoStyle := lipgloss.NewStyle().Foreground(logoColor).Width(width)
		if selected {
			logoStyle = logoStyle.Background(m.styles.Selected.GetBackground())
			nameStyle = m.styles.Selected
		}
		for _, l := range strings.Split(spec.Logo, "\n") {
			b.WriteString(logoStyle.Render(l))
			b.WriteString("\n")
		}
		row := fmt.Sprintf("%s%-12s  %d", prefix, spec.Name, count)
		b.WriteString(nameStyle.Width(width).Render(row))
		b.WriteString("\n")
	}

	if len(groups) == 0 {
		b.WriteString(m.styles.Placeholder.Render("No agents installed"))
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
	group := m.selectedGroup()
	if group == nil {
		return m.styles.Placeholder.Width(width).Render("No agent selected")
	}

	var b strings.Builder

	displayName := m.displayNameFor(group.Agent)
	b.WriteString(m.styles.Title.Render(displayName))
	b.WriteString(m.styles.Label.Render(fmt.Sprintf("  (%d sessions)", len(group.Sessions))))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", width))
	b.WriteString("\n\n")

	for i, sess := range group.Sessions {
		title := m.sessionTitle(sess.TicketID)
		row := fmt.Sprintf("  %2d  %-30s  %s", i+1, truncate(title, 30), sess.TicketID)
		b.WriteString(m.styles.Value.Render(row))
		b.WriteString("\n")
	}

	if len(group.Sessions) > 0 {
		b.WriteString("\n")
	}

	// Live pane content for the first session of the selected group
	if len(group.Sessions) == 0 {
		b.WriteString(m.styles.Placeholder.Render("  No active sessions"))
		b.WriteString("\n")
	} else {
		sess := group.Sessions[0]
		paneContent := m.paneContent
		if time.Since(m.paneContentLoadedAt) > 500*time.Millisecond {
			if content, err := m.orchestrator.GetPaneContent(sess.SessionID, 30); err == nil {
				m.paneContent = content
				m.paneContentLoadedAt = time.Now()
				paneContent = content
			}
		}

		if paneContent == "" {
			if content, err := m.orchestrator.GetPaneContent(sess.SessionID, 30); err == nil {
				m.paneContent = content
				m.paneContentLoadedAt = time.Now()
				paneContent = content
			}
		}

		if paneContent != "" {
			b.WriteString(m.styles.Title.Render("Live Agent Output"))
			b.WriteString("\n")
			lines := strings.Split(paneContent, "\n")
			maxLines := m.height - 16
			if maxLines < 5 {
				maxLines = 5
			}
			if len(lines) > maxLines {
				lines = lines[len(lines)-maxLines:]
			}
			for _, line := range lines {
				if len(line) > width-6 {
					line = line[:width-9] + "..."
				}
				b.WriteString(m.styles.Value.Render("  " + line))
				b.WriteString("\n")
			}
		} else {
			b.WriteString(m.styles.Placeholder.Render("  Starting agent..."))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	if m.isInput {
		b.WriteString(m.styles.Label.Render("Send to agent: "))
		b.WriteString(m.input.View())
		b.WriteString("\n")
	} else {
		b.WriteString(m.styles.Placeholder.Render("Press 'e' to send input to agent"))
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Padding(0, 2).
		Width(width).
		Height(m.height - 8).
		Render(b.String())
}

func (m DashboardModel) sessionTitle(ticketID string) string {
	if m.store == nil {
		return ""
	}
	tk, err := m.store.GetTicket(context.Background(), ticketID)
	if err != nil {
		return ""
	}
	return tk.Title
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
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
