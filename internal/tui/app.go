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
)

type App struct {
	store    *store.Store
	resolver *keybinding.Resolver
	config   *config.Config
	width    int
	height   int

	focus focusArea
	view  viewMode

	kanban       KanbanModel
	ticketView   TicketViewModel
	activeTicket *store.Ticket
}

func NewApp(cfg *config.Config, s *store.Store) (*App, error) {
	km := keybinding.DefaultKeyMap()
	if len(cfg.TUI.Keybindings) > 0 {
		keybinding.ApplyConfig(&km, cfg.TUI.Keybindings)
	}

	resolver := keybinding.NewResolver(km)
	kanban, err := NewKanbanModel(s, resolver)
	if err != nil {
		return nil, fmt.Errorf("tui.newApp: %w", err)
	}

	a := &App{
		store:      s,
		resolver:   resolver,
		config:     cfg,
		focus:      focusBoard,
		view:       viewBoard,
		kanban:     kanban,
		ticketView: NewTicketViewModel(s, resolver),
	}

	return a, nil
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
		return a, nil
	case tea.KeyMsg:
		return a.handleKey(msg)
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	if a.view == viewTicket {
		a.ticketView, _ = a.ticketView.Update(msg)
		return a, nil
	}

	switch action {
	case keybinding.ActionQuit, keybinding.ActionForceQuit:
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
	default:
		a.kanban, _ = a.kanban.Update(msg)
	}

	return a, nil
}

func (a *App) View() string {
	switch a.view {
	case viewHelp:
		return a.renderHelp()
	case viewTicket:
		return a.ticketView.View()
	default:
		return a.kanban.View()
	}
}

func (a *App) renderHelp() string {
	var b strings.Builder
	helpTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")).Render("Help — Keybindings")
	fmt.Fprintf(&b, "%s\n\n", helpTitle)
	km := keybinding.DefaultKeyMap()
	for _, binding := range km.Bindings {
		fmt.Fprintf(&b, "  %-12s %s\n", binding.Key, binding.Action.String())
	}
	fmt.Fprint(&b, "\nPress ? to return")
	return b.String()
}
