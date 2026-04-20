package tui

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/orchestrator"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var debugLog = log.New(os.Stderr, "[agentboard] ", log.Ltime)

type focusArea int

const (
	focusBoard focusArea = iota
	focusAgentPane
)

type viewMode int

const (
	viewBoard viewMode = iota
	viewTicket
	viewHelp
	viewDashboard
)

type editorFinishedMsg struct {
	err error
}

type statusChangedMsg struct {
	ticketID  string
	newStatus string
}

type proposalCreatedMsg struct {
	proposal store.Proposal
}

type proposalApprovedMsg struct {
	proposalID string
}

type runStartedMsg struct {
	proposalID string
}

type runStartFailedMsg struct {
	err error
}

type runCompletedMsg struct {
	ticketID string
}

type dashboardTickMsg time.Time

type proposalFailedMsg struct {
	ticketID string
	err      error
}

type proposalLoadedMsg struct {
	TicketID string
	proposal *store.Proposal
}

type notificationMsg struct {
	title   string
	message string
	variant NotificationVariant
}

type Orchestrator interface {
	CreateProposal(ctx context.Context, input orchestrator.CreateProposalInput) (store.Proposal, error)
	ApproveProposal(ctx context.Context, proposalID string) error
	StartApprovedRun(ctx context.Context, proposalID string) (store.Session, error)
	FinishRun(ctx context.Context, input orchestrator.FinishRunInput) error
	GetLogs(sessionID string) []string
	SendInput(sessionID, input string) error
	GetActiveSessions() []*orchestrator.AgentSession
	GetPTYOutput(sessionID string, lines int) (string, error)
	SetTerminalSize(sessionID string, rows, cols int) error
	CompletionChan() <-chan orchestrator.RunCompletion
}

type AppDeps struct {
	Orchestrator Orchestrator
}

type App struct {
	store        *store.Store
	orchestrator Orchestrator
	resolver     *keybinding.Resolver
	config       *config.Config
	registry     *theme.Registry
	width        int
	height       int

	focus        focusArea
	view         viewMode
	palette      CommandPalette
	modal        ConfirmModal
	notification NotificationStack
	quit         bool
	runCommand   tea.Cmd

	kanban       KanbanModel
	ticketView   TicketViewModel
	dashboard    DashboardModel
	activeTicket *store.Ticket

	generatingProposals map[string]bool
	completionCh        <-chan orchestrator.RunCompletion
}

func NewApp(cfg *config.Config, s *store.Store, reg *theme.Registry, deps AppDeps) (*App, error) {
	km := keybinding.DefaultKeyMap()
	if len(cfg.TUI.Keybindings) > 0 {
		keybinding.ApplyConfig(&km, cfg.TUI.Keybindings)
	}

	resolver := keybinding.NewResolver(km)

	t := reg.Active()
	kanban, err := NewKanbanModel(s, resolver, t)
	if err != nil {
		return nil, fmt.Errorf("tui.newApp: %w", err)
	}

	agents := config.DetectAgents()

	a := &App{
		store:               s,
		orchestrator:        deps.Orchestrator,
		resolver:            resolver,
		config:              cfg,
		registry:            reg,
		focus:               focusBoard,
		view:                viewBoard,
		kanban:              kanban,
		ticketView:          NewTicketViewModel(s, resolver, t, agents),
		dashboard:           NewDashboardModel(s, deps.Orchestrator, resolver, agents, t),
		generatingProposals: make(map[string]bool),
		completionCh:        deps.Orchestrator.CompletionChan(),
	}

	cr := NewCommandRegistry()
	ac := newAppCommands(a, reg, cfg)
	ac.registerAll(cr)

	a.palette = NewCommandPalette(cr, nil)
	a.palette.SetTheme(t)
	a.palette.onSelect = ac.onSelect
	a.palette.onConfirm = ac.onConfirm

	a.modal = ConfirmModal{}
	a.modal.SetTheme(t)
	a.notification = NotificationStack{}
	a.notification.SetTheme(t)

	return a, nil
}

func (a *App) applyTheme() {
	t := a.registry.Active()
	a.kanban.styles = NewKanbanStyles(t)
	a.kanban.theme = t
	a.ticketView.styles = NewTicketViewStyles(t)
	a.dashboard.styles = NewDashboardStyles(t)
	a.palette.SetTheme(t)
	a.modal.SetTheme(t)
	a.notification.SetTheme(t)
}

func (a *App) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func dashboardTick() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return dashboardTickMsg(t)
	})
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.kanban, _ = a.kanban.Update(msg)
		a.ticketView, _ = a.ticketView.Update(msg)
		a.dashboard, _ = a.dashboard.Update(msg)
		a.palette, _ = a.palette.Update(msg)
		a.modal.SetSize(a.width, a.height)
		a.notification.SetSize(a.width, a.height)
		return a, nil
	case tea.KeyMsg:
		return a.handleKey(msg)
	case editorFinishedMsg:
		return a, nil
	case tickMsg:
		var cmd tea.Cmd
		a.kanban, cmd = a.kanban.Update(msg)
		a.dashboard, _ = a.dashboard.Update(msg)
		return a, cmd
	case dashboardTickMsg:
		if a.view != viewDashboard {
			return a, nil
		}
		a.dashboard = a.dashboard.Refresh()
		a.dashboard = a.dashboard.refreshPaneContent()
		return a, dashboardTick()
	case notificationDismissMsg:
		a.notification = a.notification.HandleDismiss(msg)
		return a, nil
	case ticketCreatedMsg:
		return a, a.showNotification(
			"Ticket created",
			fmt.Sprintf("%s: %s", msg.id, msg.title),
			NotificationSuccess,
		)
	case agentAssignedMsg:
		message := fmt.Sprintf("%s cleared on %s", "Agent", msg.ticketID)
		if msg.agent != "" {
			message = fmt.Sprintf("%s assigned to %s", msg.agent, msg.ticketID)
		}
		return a, a.showNotification(
			"Agent assignment updated",
			message,
			NotificationSuccess,
		)
	case statusChangedMsg:
		return a.handleStatusChanged(msg)
	case proposalCreatedMsg:
		return a, tea.Batch(
			a.showNotification(
				"Proposal created",
				fmt.Sprintf("AI proposed work for %s", msg.proposal.TicketID),
				NotificationInfo,
			),
			a.loadProposalCmd(msg.proposal.TicketID),
		)
	case proposalLoadedMsg:
		delete(a.generatingProposals, msg.TicketID)
		if a.activeTicket != nil && a.activeTicket.ID == msg.TicketID {
			a.ticketView = a.ticketView.SetProposal(msg.proposal)
		}
		return a, nil
	case proposalApprovedMsg:
		return a, a.approveProposalCmd(msg.proposalID)
	case runStartedMsg:
		return a, tea.Batch(
			a.showNotification("Run started", "Agent is working...", NotificationInfo),
			a.startRunAndListenCmd(msg.proposalID),
		)
	case runCompletedMsg:
		a.kanban, _ = a.kanban.Reload()
		a.dashboard = a.dashboard.Refresh()
		if a.activeTicket != nil && a.activeTicket.ID == msg.ticketID {
			updated, err := a.store.GetTicket(context.Background(), msg.ticketID)
			if err == nil {
				a.activeTicket = &updated
				a.ticketView = a.ticketView.SetTicket(&updated)
			}
		}
		return a, a.showNotification("Run completed", fmt.Sprintf("Agent finished working on %s", msg.ticketID), NotificationSuccess)
	case proposalFailedMsg:
		delete(a.generatingProposals, msg.ticketID)
		if a.activeTicket != nil && a.activeTicket.ID == msg.ticketID {
			a.ticketView = a.ticketView.SetLoading(false)
		}
		a.dashboard = a.dashboard.Refresh()
		return a, a.showNotification("Proposal failed", msg.err.Error(), NotificationError)
	case runStartFailedMsg:
		a.dashboard = a.dashboard.Refresh()
		return a, a.showNotification("Run failed", msg.err.Error(), NotificationError)
	case notificationMsg:
		return a, a.showNotification(msg.title, msg.message, msg.variant)
	}

	if a.kanban.NeedsTick() {
		return a, animationTick()
	}
	return a, nil
}

func (a *App) loadProposalCmd(ticketID string) tea.Cmd {
	return func() tea.Msg {
		p, err := a.store.GetActiveProposalForTicket(context.Background(), ticketID)
		if err != nil {
			return proposalLoadedMsg{proposal: nil, TicketID: ticketID}
		}
		return proposalLoadedMsg{proposal: &p, TicketID: ticketID}
	}
}

func (a *App) approveProposalCmd(proposalID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		err := a.orchestrator.ApproveProposal(ctx, proposalID)
		if err != nil {
			return notificationMsg{title: "Error", message: err.Error(), variant: NotificationError}
		}
		// Reload proposal
		p, _ := a.store.GetProposal(context.Background(), proposalID)
		return proposalLoadedMsg{proposal: &p, TicketID: p.TicketID}
	}
}

func (a *App) startRunAndListenCmd(proposalID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		_, err := a.orchestrator.StartApprovedRun(ctx, proposalID)
		if err != nil {
			return runStartFailedMsg{err: err}
		}

		completion := <-a.completionCh
		return runCompletedMsg{ticketID: completion.TicketID}
	}
}

func (a *App) handleStatusChanged(msg statusChangedMsg) (tea.Model, tea.Cmd) {
	// First update the store
	err := a.store.MoveStatus(context.Background(), msg.ticketID, msg.newStatus)
	if err != nil {
		return a, a.showNotification("Error", err.Error(), NotificationError)
	}

	// Update local state if it's the active ticket
	if a.activeTicket != nil && a.activeTicket.ID == msg.ticketID {
		a.activeTicket.Status = msg.newStatus
		a.ticketView = a.ticketView.SetTicket(a.activeTicket)
	}

	var cmds []tea.Cmd
	cmds = append(cmds, a.showNotification("Status updated", fmt.Sprintf("%s moved to %s", msg.ticketID, msg.newStatus), NotificationSuccess))
	if a.activeTicket != nil && a.activeTicket.ID == msg.ticketID {
		a.ticketView = a.ticketView.SetLoading(msg.newStatus == "in_progress")
	}

	// If moved to in_progress, trigger orchestration
	if msg.newStatus == "in_progress" {
		a.generatingProposals[msg.ticketID] = true
		cmds = append(cmds, a.createProposalCmd(msg.ticketID))
	} else {
		delete(a.generatingProposals, msg.ticketID)
	}

	return a, tea.Batch(cmds...)
}

func (a *App) createProposalCmd(ticketID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		debugLog.Printf("createProposal: ticket=%s", ticketID)
		proposal, err := a.orchestrator.CreateProposal(ctx, orchestrator.CreateProposalInput{
			TicketID: ticketID,
		})
		if err != nil {
			debugLog.Printf("createProposal FAILED: %v", err)
			return proposalFailedMsg{ticketID: ticketID, err: err}
		}
		debugLog.Printf("createProposal OK: id=%s", proposal.ID)
		return proposalCreatedMsg{proposal: proposal}
	}
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.modal.Active() {
		var cmd tea.Cmd
		a.modal, cmd = a.modal.Update(msg)
		return a, cmd
	}

	if a.palette.Active() {
		a.palette, _ = a.palette.Update(msg)
		if !a.palette.Active() {
			if a.quit {
				return a, tea.Quit
			}
			if a.runCommand != nil {
				cmd := a.runCommand
				a.runCommand = nil
				return a, cmd
			}
		}
		return a, nil
	}

	key := msg.String()
	action, _ := a.resolver.Resolve(key)

	if key == "esc" {
		if a.view == viewTicket && (a.ticketView.mode == ticketEditMode || a.ticketView.mode == ticketAgentSelectMode) {
			var cmd tea.Cmd
			a.ticketView, cmd = a.ticketView.Update(msg)
			return a, cmd
		}
		if a.view == viewTicket {
			a.kanban, _ = a.kanban.Reload()
		}
		if a.view != viewBoard {
			a.view = viewBoard
			a.activeTicket = nil
			return a, nil
		}
	}

	if a.view == viewTicket && a.ticketView.mode == ticketEditMode {
		var cmd tea.Cmd
		a.ticketView, cmd = a.ticketView.Update(msg)
		return a, cmd
	}

	if a.view == viewTicket && action != keybinding.ActionShowDashboard {
		var cmd tea.Cmd
		a.ticketView, cmd = a.ticketView.Update(msg)
		return a, cmd
	}

	if a.view == viewDashboard && action != keybinding.ActionShowDashboard {
		a.dashboard, _ = a.dashboard.Update(msg)
		return a, nil
	}

	switch action {
	case keybinding.ActionQuit:
		a.modal.Open(
			"Quit AgentBoard",
			"Are you sure you want to quit?",
			func() tea.Cmd { return tea.Quit },
			nil,
		)
	case keybinding.ActionForceQuit:
		return a, tea.Quit
	case keybinding.ActionOpenTicket:
		selected := a.kanban.SelectedTicket()
		if selected != nil {
			a.activeTicket = selected
			a.ticketView = a.ticketView.SetTicket(selected)
			a.view = viewTicket
			a.ticketView = a.ticketView.SetLoading(a.generatingProposals[selected.ID])
			p, _ := a.store.GetActiveProposalForTicket(context.Background(), selected.ID)
			if p.ID != "" {
				a.ticketView = a.ticketView.SetProposal(&p)
			}
			return a, nil
		}
	case keybinding.ActionShowHelp:
		if a.view == viewHelp {
			a.view = viewBoard
		} else {
			a.view = viewHelp
		}
	case keybinding.ActionShowDashboard:
		if a.view == viewDashboard {
			a.view = viewBoard
		} else {
			a.dashboard = a.dashboard.Refresh()
			a.dashboard = a.dashboard.refreshPaneContent()
			a.view = viewDashboard
			return a, dashboardTick()
		}
	case keybinding.ActionOpenPalette:
		a.palette.Open()
	case keybinding.ActionRefresh:
		a.kanban, _ = a.kanban.Reload()
	default:
		var cmd tea.Cmd
		a.kanban, cmd = a.kanban.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a *App) showNotification(title, message string, variant NotificationVariant) tea.Cmd {
	dur := 2 * time.Second
	if variant == NotificationError {
		dur = 5 * time.Second
	}
	return a.notification.Show(title, message, variant, dur)
}

func (a *App) renderHeader() string {
	t := a.registry.Active()
	fg := lipgloss.Color("252")
	if t != nil {
		fg = t.Text
	}
	muted := lipgloss.Color("240")
	if t != nil {
		muted = t.TextMuted
	}

	name := a.config.ProjectName
	if name == "" {
		name = "AgentBoard"
	}
	labelStyle := lipgloss.NewStyle().Foreground(muted)
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(fg)
	hintStyle := lipgloss.NewStyle().Foreground(muted)

	projectPart := labelStyle.Render("Project: ") + nameStyle.Render(name)
	hintPart := hintStyle.Render("?: help │ r: refresh")

	projectWidth := lipgloss.Width(projectPart)
	hintWidth := lipgloss.Width(hintPart)
	available := a.width - hintWidth - 1
	if available < projectWidth {
		available = projectWidth
	}

	leftPad := (available - projectWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	left := strings.Repeat(" ", leftPad) + projectPart
	right := hintPart

	totalLeft := lipgloss.Width(left)
	gap := a.width - totalLeft - hintWidth
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

func (a *App) View() string {
	var mainView string
	switch a.view {
	case viewHelp:
		mainView = a.renderHelp()
	case viewTicket:
		mainView = a.ticketView.View()
	case viewDashboard:
		mainView = a.dashboard.View()
	default:
		mainView = a.kanban.View()
	}
	header := a.renderHeader()
	mainView = lipgloss.NewStyle().Height(a.height - 1).Render(mainView)
	mainView = header + "\n" + mainView

	if a.palette.Active() {
		paletteView := a.palette.View()
		paletteLines := strings.Split(paletteView, "\n")
		paletteHeight := len(paletteLines)
		bgLines := strings.Split(mainView, "\n")

		for len(bgLines) < a.height {
			bgLines = append(bgLines, "")
		}

		startY := a.height - paletteHeight
		var finalView strings.Builder
		for i := 0; i < a.height; i++ {
			bgLine := ""
			if i < len(bgLines) {
				bgLine = bgLines[i]
			}

			if i >= startY && i < a.height {
				row := i - startY
				paletteLine := paletteLines[row]
				// Palette is docked at the left (x=0)
				finalView.WriteString(overlayLine(bgLine, paletteLine, 0))
			} else {
				finalView.WriteString(bgLine)
			}
			if i < a.height-1 {
				finalView.WriteString("\n")
			}
		}
		mainView = finalView.String()
	}

	if a.modal.Active() {
		modalBox := a.modal.ViewBox()
		modalHeight := lipgloss.Height(modalBox)
		modalWidth := lipgloss.Width(modalBox)
		bgLines := strings.Split(mainView, "\n")

		for len(bgLines) < a.height {
			bgLines = append(bgLines, "")
		}

		startY := (a.height - modalHeight) / 2
		startX := (a.width - modalWidth) / 2

		var finalView strings.Builder
		modalLines := strings.Split(modalBox, "\n")

		for i := 0; i < a.height; i++ {
			bgLine := ""
			if i < len(bgLines) {
				bgLine = bgLines[i]
			}

			if i >= startY && i < startY+modalHeight {
				row := i - startY
				modalLine := modalLines[row]
				finalView.WriteString(overlayLine(bgLine, modalLine, startX))
			} else {
				finalView.WriteString(bgLine)
			}
			if i < a.height-1 {
				finalView.WriteString("\n")
			}
		}
		return finalView.String()
	}

	if a.notification.Active() {
		notificationStack := a.notification.View()
		notificationHeight := lipgloss.Height(notificationStack)
		notificationWidth := lipgloss.Width(notificationStack)
		bgLines := strings.Split(mainView, "\n")

		for len(bgLines) < a.height {
			bgLines = append(bgLines, "")
		}

		startY := a.height - notificationHeight - 1
		if startY < 1 {
			startY = 1
		}
		startX := a.width - notificationWidth - 2
		if startX < 0 {
			startX = 0
		}

		var finalView strings.Builder
		notificationLines := strings.Split(notificationStack, "\n")

		for i := 0; i < a.height; i++ {
			bgLine := ""
			if i < len(bgLines) {
				bgLine = bgLines[i]
			}

			if i >= startY && i < startY+notificationHeight {
				row := i - startY
				notificationLine := notificationLines[row]
				finalView.WriteString(overlayLine(bgLine, notificationLine, startX))
			} else {
				finalView.WriteString(bgLine)
			}
			if i < a.height-1 {
				finalView.WriteString("\n")
			}
		}
		return finalView.String()
	}

	return mainView
}

// overlayLine places fg over bg at the given x offset, preserving bg on both sides.
func overlayLine(bg, fg string, x int) string {
	bgWidth := lipgloss.Width(bg)
	fgWidth := lipgloss.Width(fg)

	if x < 0 {
		x = 0
	}
	if x+fgWidth > bgWidth {
		x = (bgWidth - fgWidth) / 2
	}

	left := ansiTruncate(bg, x)
	right := ansiSkip(bg, x+fgWidth)

	return left + fg + right
}

// ansiTruncate returns the part of the string up to the given visual width.
func ansiTruncate(s string, limit int) string {
	var (
		visualPos int
		inEscape  bool
	)

	for i, r := range s {
		if visualPos >= limit && !inEscape {
			return s[:i]
		}

		if r == '\x1b' {
			inEscape = true
			continue
		}

		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}

		visualPos++
	}
	return s
}

// ansiSkip returns the part of the string starting after the given visual width.
func ansiSkip(s string, skip int) string {
	var (
		visualPos int
		inEscape  bool
	)

	for i, r := range s {
		if visualPos >= skip && !inEscape {
			return s[i:]
		}

		if r == '\x1b' {
			inEscape = true
			continue
		}

		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}

		// Simplified width check (most TUI chars are width 1)
		visualPos++
	}
	return ""
}

func (a *App) renderHelp() string {
	t := a.registry.Active()
	primary := lipgloss.Color("69")
	if t != nil {
		primary = t.Primary
	}

	var b strings.Builder
	helpTitle := lipgloss.NewStyle().Bold(true).Foreground(primary).Render("Help — Keybindings")
	fmt.Fprintf(&b, "%s\n\n", helpTitle)
	km := keybinding.DefaultKeyMap()
	for _, binding := range km.Bindings {
		fmt.Fprintf(&b, "  %-12s %s\n", binding.Key, binding.Action.String())
	}
	fmt.Fprint(&b, "\nPress ? to return")
	return b.String()
}
