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

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ticketViewModeType int

const (
	ticketViewMode ticketViewModeType = iota
	ticketEditMode
	ticketAgentSelectMode
	ticketPrioritySelectMode
	ticketDependsOnSelectMode
	ticketAdHocPromptMode
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

	ticket      *store.Ticket
	fields      []ticketField
	cursor      int
	mode        ticketViewModeType
	editBuffer  string
	agents      []config.DetectedAgent
	agentCursor int

	priorities     []string
	priorityCursor int

	dependsOnTickets  []store.Ticket
	dependsOnCursor   int
	dependsOnSelected []string

	styles TicketViewStyles

	activeProposal *store.Proposal
	loading        bool

	adhocAgent        string
	adhocPrompt       string
	adhocPromptCursor int

	viewport viewport.Model
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

func NewTicketViewStyles(t *theme.Theme) TicketViewStyles {
	return TicketViewStyles{
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary),
		Label: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Text),
		Value: lipgloss.NewStyle().
			Foreground(t.Text),
		SelectedRow: lipgloss.NewStyle().
			Background(t.Primary).
			Foreground(t.Text),
		Cursor: lipgloss.NewStyle().
			Foreground(t.Primary),
		EditBox: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(t.Accent),
		Footer: lipgloss.NewStyle().
			Foreground(t.TextMuted),
		Empty: lipgloss.NewStyle().
			Foreground(t.TextMuted),
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
			editable: false,
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
			editable: true,
			set: func(t *store.Ticket, v string) {
				t.Tags = parseTags(v)
			},
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
		{
			label: "Resume",
			value: func(t *store.Ticket) string {
				if t.ResumeCommand == "" {
					return "—"
				}
				return t.ResumeCommand
			},
			editable: false,
		},
	}
}

func NewTicketViewModel(s *store.Store, resolver *keybinding.Resolver, t *theme.Theme, agents []config.DetectedAgent) TicketViewModel {
	return TicketViewModel{
		store:      s,
		resolver:   resolver,
		styles:     NewTicketViewStyles(t),
		fields:     ticketFields(),
		mode:       ticketViewMode,
		agents:     agents,
		priorities: []string{"", "low", "medium", "high", "critical"},
		viewport:   viewport.New(0, 0),
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
		vh := m.height - 7
		if vh < 0 {
			vh = 0
		}
		m.viewport.Width = m.width - 6
		m.viewport.Height = vh
		return m, nil
	case tea.KeyMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		m, nextCmd := m.handleKey(msg)
		return m, tea.Batch(cmd, nextCmd)
	}
	return m, nil
}

func (m TicketViewModel) handleKey(msg tea.KeyMsg) (TicketViewModel, tea.Cmd) {
	if m.mode == ticketEditMode {
		return m.handleEditKey(msg)
	}
	if m.mode == ticketAgentSelectMode {
		return m.handleAgentSelectKey(msg)
	}
	if m.mode == ticketPrioritySelectMode {
		return m.handlePrioritySelectKey(msg)
	}
	if m.mode == ticketDependsOnSelectMode {
		return m.handleDependsOnSelectKey(msg)
	}
	if m.mode == ticketAdHocPromptMode {
		return m.handleAdHocPromptKey(msg)
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
			return m.cycleStatus()
		}
	case "a":
		if m.ticket != nil {
			m.mode = ticketAgentSelectMode
			m.agentCursor = 0
		}
	case "p":
		if m.ticket != nil {
			m.mode = ticketPrioritySelectMode
			m.priorityCursor = 0
			for i, p := range m.priorities {
				if p == m.ticket.Priority {
					m.priorityCursor = i
					break
				}
			}
		}
	case "d":
		if m.ticket != nil {
			m.mode = ticketDependsOnSelectMode
			m.dependsOnCursor = 0
			m.dependsOnSelected = m.ticket.DependsOn
			tickets, _ := m.store.ListTickets(context.Background(), store.TicketFilters{})
			m.dependsOnTickets = tickets
			if len(m.dependsOnTickets) > 5 {
				m.dependsOnTickets = m.dependsOnTickets[:5]
			}
		}
	case "o":
		if m.activeProposal != nil && m.activeProposal.Status == "pending" {
			proposalID := m.activeProposal.ID
			return m, func() tea.Msg {
				return proposalApprovedMsg{proposalID: proposalID}
			}
		}
	case "r":
		if m.activeProposal != nil && m.activeProposal.Status == "approved" {
			return m, func() tea.Msg {
				return runStartedMsg{proposalID: m.activeProposal.ID}
			}
		}
	case "v":
		if m.activeProposal != nil {
			return m, func() tea.Msg {
				return viewProposalFullMsg{proposalID: m.activeProposal.ID}
			}
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
	case tea.KeySpace:
		m.editBuffer += " "
		return m, nil
	}

	if msg.Type == tea.KeyRunes {
		m.editBuffer += string(msg.Runes)
	}

	return m, nil
}

func (m TicketViewModel) handleAgentSelectKey(msg tea.KeyMsg) (TicketViewModel, tea.Cmd) {
	total := len(m.agents) + 1

	switch msg.Type {
	case tea.KeyEscape:
		m.mode = ticketViewMode
		return m, nil
	case tea.KeyEnter:
		if m.ticket != nil {
			if m.agentCursor == 0 {
				m.ticket.Agent = ""
			} else {
				idx := m.agentCursor - 1
				if idx < len(m.agents) {
					m.ticket.Agent = m.agents[idx].Name
				}
			}
			_, _ = m.store.UpdateTicket(context.Background(), *m.ticket)
			agent := m.ticket.Agent
			ticketID := m.ticket.ID
			m.mode = ticketViewMode
			return m, func() tea.Msg {
				return agentAssignedMsg{ticketID: ticketID, agent: agent}
			}
		}
		m.mode = ticketViewMode
		return m, nil
	case tea.KeyUp:
		if m.agentCursor > 0 {
			m.agentCursor--
		}
		return m, nil
	case tea.KeyDown:
		if m.agentCursor < total-1 {
			m.agentCursor++
		}
		return m, nil
	}

	if msg.Type == tea.KeyRunes {
		switch string(msg.Runes) {
		case "j":
			if m.agentCursor < total-1 {
				m.agentCursor++
			}
		case "k":
			if m.agentCursor > 0 {
				m.agentCursor--
			}
		}
	}

	return m, nil
}

func (m TicketViewModel) handlePrioritySelectKey(msg tea.KeyMsg) (TicketViewModel, tea.Cmd) {
	total := len(m.priorities)

	switch msg.Type {
	case tea.KeyEscape:
		m.mode = ticketViewMode
		return m, nil
	case tea.KeyEnter:
		if m.ticket != nil && m.priorityCursor < len(m.priorities) {
			m.ticket.Priority = m.priorities[m.priorityCursor]
			_, _ = m.store.UpdateTicket(context.Background(), *m.ticket)
		}
		m.mode = ticketViewMode
		return m, nil
	case tea.KeyUp:
		if m.priorityCursor > 0 {
			m.priorityCursor--
		}
		return m, nil
	case tea.KeyDown:
		if m.priorityCursor < total-1 {
			m.priorityCursor++
		}
		return m, nil
	}

	if msg.Type == tea.KeyRunes {
		switch string(msg.Runes) {
		case "j":
			if m.priorityCursor < total-1 {
				m.priorityCursor++
			}
		case "k":
			if m.priorityCursor > 0 {
				m.priorityCursor--
			}
		}
	}

	return m, nil
}

func (m TicketViewModel) handleDependsOnSelectKey(msg tea.KeyMsg) (TicketViewModel, tea.Cmd) {
	total := len(m.dependsOnTickets)

	switch msg.Type {
	case tea.KeyEscape:
		m.mode = ticketViewMode
		return m, nil
	case tea.KeyEnter:
		if m.ticket != nil {
			m.ticket.DependsOn = m.dependsOnSelected
			_, _ = m.store.UpdateTicket(context.Background(), *m.ticket)
		}
		m.mode = ticketViewMode
		return m, nil
	case tea.KeySpace:
		if m.dependsOnCursor < len(m.dependsOnTickets) {
			id := m.dependsOnTickets[m.dependsOnCursor].ID
			if m.ticket != nil && id == m.ticket.ID {
				return m, nil
			}
			found := false
			for i, d := range m.dependsOnSelected {
				if d == id {
					m.dependsOnSelected = append(m.dependsOnSelected[:i], m.dependsOnSelected[i+1:]...)
					found = true
					break
				}
			}
			if !found {
				m.dependsOnSelected = append(m.dependsOnSelected, id)
			}
		}
		return m, nil
	case tea.KeyUp:
		if m.dependsOnCursor > 0 {
			m.dependsOnCursor--
		}
		return m, nil
	case tea.KeyDown:
		if m.dependsOnCursor < total-1 {
			m.dependsOnCursor++
		}
		return m, nil
	}

	if msg.Type == tea.KeyRunes {
		switch string(msg.Runes) {
		case "j":
			if m.dependsOnCursor < total-1 {
				m.dependsOnCursor++
			}
		case "k":
			if m.dependsOnCursor > 0 {
				m.dependsOnCursor--
			}
		}
	}

	return m, nil
}

func (m TicketViewModel) handleAdHocPromptKey(msg tea.KeyMsg) (TicketViewModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.mode = ticketViewMode
		m.adhocPrompt = ""
		return m, nil
	case tea.KeyEnter:
		if m.adhocPrompt != "" {
			agent := m.adhocAgent
			prompt := m.adhocPrompt
			m.mode = ticketViewMode
			m.adhocPrompt = ""
			return m, func() tea.Msg {
				return adhocRunStartedMsg{agent: agent, prompt: prompt}
			}
		}
		return m, nil
	case tea.KeyBackspace:
		if len(m.adhocPrompt) > 0 {
			runes := []rune(m.adhocPrompt)
			m.adhocPrompt = string(runes[:len(runes)-1])
		}
		return m, nil
	}

	if msg.Type == tea.KeyRunes {
		m.adhocPrompt += string(msg.Runes)
	}

	return m, nil
}

func (m TicketViewModel) cycleStatus() (TicketViewModel, tea.Cmd) {
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
	newStatus := statusCycle[nextIdx]
	ticketID := m.ticket.ID

	return m, func() tea.Msg {
		return statusChangedMsg{ticketID: ticketID, newStatus: newStatus}
	}
}

func (m TicketViewModel) buildPromptFromTicket() string {
	if m.ticket == nil {
		return ""
	}
	return fmt.Sprintf("Ticket: %s\nTitle: %s\nDescription: %s", m.ticket.ID, m.ticket.Title, m.ticket.Description)
}

func (m TicketViewModel) SetTicket(t *store.Ticket) TicketViewModel {
	m.ticket = t
	m.cursor = 0
	m.mode = ticketViewMode
	m.editBuffer = ""
	m.activeProposal = nil
	return m
}

func (m TicketViewModel) SetProposal(p *store.Proposal) TicketViewModel {
	m.activeProposal = p
	m.loading = false
	return m
}

func (m TicketViewModel) SetLoading(loading bool) TicketViewModel {
	m.loading = loading
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

		valWidth := innerWidth - 15
		if valWidth < 10 {
			valWidth = 10
		}

		labelStr := fmt.Sprintf("%-12s", f.label)
		labelStyle := m.styles.Label
		prefixStyle := lipgloss.NewStyle()

		if i == m.cursor {
			labelStyle = m.styles.Title
			prefixStyle = m.styles.Cursor
		}

		renderedPrefix := prefixStyle.Render(prefix)
		renderedLabel := labelStyle.Render(labelStr)
		renderedVal := m.styles.Value.Copy().Width(valWidth).Render(val)

		row := lipgloss.JoinHorizontal(lipgloss.Top, renderedPrefix, renderedLabel, " ", renderedVal)

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

	if m.mode == ticketAgentSelectMode {
		b.WriteString("\n")
		b.WriteString(m.styles.Label.Render("Select Agent:"))
		b.WriteString("\n")

		items := make([]string, 0, len(m.agents)+1)
		items = append(items, "None")
		for _, ag := range m.agents {
			items = append(items, ag.Name)
		}

		for i, item := range items {
			prefix := "  "
			if i == m.agentCursor {
				prefix = "▸ "
			}

			pStyle := lipgloss.NewStyle()
			iStyle := m.styles.Value
			if i == m.agentCursor {
				pStyle = m.styles.Cursor
				iStyle = m.styles.Title
			}

			row := pStyle.Render(prefix) + iStyle.Render(item)
			b.WriteString(row)
			b.WriteString("\n")
		}
	}

	if m.mode == ticketPrioritySelectMode {
		b.WriteString("\n")
		b.WriteString(m.styles.Label.Render("Select Priority:"))
		b.WriteString("\n")

		for i, p := range m.priorities {
			display := p
			if display == "" {
				display = "None"
			}
			prefix := "  "
			if i == m.priorityCursor {
				prefix = "▸ "
			}

			pStyle := lipgloss.NewStyle()
			iStyle := m.styles.Value
			if i == m.priorityCursor {
				pStyle = m.styles.Cursor
				iStyle = m.styles.Title
			}

			row := pStyle.Render(prefix) + iStyle.Render(display)
			b.WriteString(row)
			b.WriteString("\n")
		}
	}

	if m.mode == ticketDependsOnSelectMode {
		b.WriteString("\n")
		b.WriteString(m.styles.Label.Render("Select Dependencies (space to toggle):"))
		b.WriteString("\n")

		shown := 0
		for i, t := range m.dependsOnTickets {
			if m.ticket != nil && t.ID == m.ticket.ID {
				continue
			}
			shown++
			prefix := "  "
			if i == m.dependsOnCursor {
				prefix = "▸ "
			}
			selected := ""
			for _, d := range m.dependsOnSelected {
				if d == t.ID {
					selected = " ✓"
					break
				}
			}

			pStyle := lipgloss.NewStyle()
			iStyle := m.styles.Value
			if i == m.dependsOnCursor {
				pStyle = m.styles.Cursor
				iStyle = m.styles.Title
			}

			row := pStyle.Render(prefix) + iStyle.Render(t.ID+" - "+t.Title+selected)
			b.WriteString(row)
			b.WriteString("\n")
		}
		if shown == 0 {
			b.WriteString(m.styles.Empty.Render("No tickets available"))
			b.WriteString("\n")
		}
	}

	if m.mode == ticketAdHocPromptMode {
		b.WriteString("\n")
		b.WriteString(m.styles.Title.Render(fmt.Sprintf("Run %s with prompt:", m.adhocAgent)))
		b.WriteString("\n\n")
		promptLabel := m.styles.Label.Render("Task:")
		b.WriteString(promptLabel)
		b.WriteString("\n")
		promptLine := m.adhocPrompt + "│"
		if len(promptLine) > innerWidth-4 {
			promptLine = promptLine[:innerWidth-7] + "..."
		}
		b.WriteString(m.styles.EditBox.Width(innerWidth - 4).Render(promptLine))
		b.WriteString("\n")
	}

	var footer string
	if m.mode == ticketAgentSelectMode {
		footer = "↑/k: up │ ↓/j: down │ Enter: select │ Esc: cancel"
	} else if m.mode == ticketPrioritySelectMode {
		footer = "↑/k: up │ ↓/j: down │ Enter: select │ Esc: cancel"
	} else if m.mode == ticketDependsOnSelectMode {
		footer = "↑/k: up │ ↓/j: down │ Space: toggle │ Enter: save │ Esc: cancel"
	} else if m.mode == ticketAdHocPromptMode {
		footer = "Enter: run agent │ Esc: cancel"
	} else {
		footer = "e: edit │ s: cycle status │ a: assign agent │ p: set priority │ d: set depends on │ Esc: back"
		if m.activeProposal != nil {
			footer += " │ v: view proposal"
		}
		if m.activeProposal != nil && m.activeProposal.Status == "pending" {
			footer += " │ o: approve proposal"
		} else if m.activeProposal != nil && m.activeProposal.Status == "approved" {
			footer += " │ r: start run"
		}
	}

	if m.loading || m.activeProposal != nil || m.ticket.Status == "in_progress" {
		b.WriteString("\n\n")
		b.WriteString(m.styles.Title.Render("── Active Proposal ───────────────────────"))
		b.WriteString("\n")

		if m.loading {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render("Status: generating... (please wait)"))
			b.WriteString("\n")
		} else if m.activeProposal != nil {
			statusColor := "240"
			if m.activeProposal.Status == "pending" {
				statusColor = "213"
			} else if m.activeProposal.Status == "approved" {
				statusColor = "42"
			}
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render("Status: " + m.activeProposal.Status))
			b.WriteString("\n\n")
			prompt := m.activeProposal.Prompt
			preview := prompt
			if len(preview) > 300 {
				preview = preview[:300] + "\n\n[...] Press v to view full proposal"
			}
			b.WriteString(m.styles.Value.Width(innerWidth).Render(preview))
		} else { // m.ticket.Status == "in_progress"
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("No active proposal found for this ticket."))
			b.WriteString("\n")
		}
	}

	content := b.String()
	m.viewport.SetContent(content)

	h := m.height - 6
	if h < 0 {
		h = 0
	}
	if m.viewport.Width == 0 {
		m.viewport.Width = innerWidth
	}
	if m.viewport.Height == 0 {
		m.viewport.Height = h
	}
	if m.mode != ticketViewMode {
		m.viewport.Height = strings.Count(content, "\n") + 1
	}
	footerRendered := m.styles.Footer.Render(footer)
	finalView := lipgloss.JoinVertical(lipgloss.Left, m.viewport.View(), footerRendered)
	return m.styles.Border.Render(finalView)
}

func parseTags(input string) []string {
	if input == "" {
		return nil
	}
	tags := strings.Split(input, ",")
	result := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" {
			result = append(result, t)
		}
	}
	return result
}
