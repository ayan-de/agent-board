package tui

import (
	"fmt"
	"strings"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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

type App struct {
	store    *store.Store
	resolver *keybinding.Resolver
	config   *config.Config
	registry *theme.Registry
	width    int
	height   int

	focus      focusArea
	view       viewMode
	palette    CommandPalette
	modal      ConfirmModal
	quit       bool
	runCommand tea.Cmd

	kanban       KanbanModel
	ticketView   TicketViewModel
	dashboard    DashboardModel
	activeTicket *store.Ticket
}

func NewApp(cfg *config.Config, s *store.Store, reg *theme.Registry) (*App, error) {
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
		store:      s,
		resolver:   resolver,
		config:     cfg,
		registry:   reg,
		focus:      focusBoard,
		view:       viewBoard,
		kanban:     kanban,
		ticketView: NewTicketViewModel(s, resolver, t),
		dashboard:  NewDashboardModel(s, resolver, agents, t),
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

	return a, nil
}

func (a *App) applyTheme() {
	t := a.registry.Active()
	a.kanban.styles = NewKanbanStyles(t)
	a.ticketView.styles = NewTicketViewStyles(t)
	a.dashboard.styles = NewDashboardStyles(t)
	a.palette.SetTheme(t)
	a.modal.SetTheme(t)
}

func (a *App) Init() tea.Cmd {
	return tea.EnterAltScreen
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
		return a, nil
	case tea.KeyMsg:
		return a.handleKey(msg)
	case editorFinishedMsg:
		return a, nil
	}
	return a, nil
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
		if a.view == viewTicket && a.ticketView.mode == ticketEditMode {
			a.ticketView, _ = a.ticketView.Update(msg)
			return a, nil
		}
		if a.view != viewBoard {
			a.view = viewBoard
			a.activeTicket = nil
			return a, nil
		}
	}

	if a.view == viewTicket && action != keybinding.ActionShowDashboard {
		a.ticketView, _ = a.ticketView.Update(msg)
		return a, nil
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
			a.view = viewDashboard
		}
	case keybinding.ActionOpenPalette:
		a.palette.Open()
	default:
		a.kanban, _ = a.kanban.Update(msg)
	}

	return a, nil
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
	mainView = lipgloss.NewStyle().Height(a.height).Render(mainView)

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
