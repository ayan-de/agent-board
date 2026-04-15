package tui

import (
	"fmt"
	"strings"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"

	tea "github.com/charmbracelet/bubbletea"
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
		store:    s,
		resolver: resolver,
		config:   cfg,
		focus:    focusBoard,
		view:     viewBoard,
		kanban:   kanban,
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
		return a, nil
	case tea.KeyMsg:
		return a.handleKey(msg)
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" && a.view != viewBoard {
		a.view = viewBoard
		a.activeTicket = nil
		return a, nil
	}

	key := msg.String()
	action, _ := a.resolver.Resolve(key)

	switch action {
	case keybinding.ActionQuit, keybinding.ActionForceQuit:
		return a, tea.Quit
	case keybinding.ActionOpenTicket:
		selected := a.kanban.SelectedTicket()
		if selected != nil {
			a.activeTicket = selected
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
		return a.renderTicket()
	default:
		return a.kanban.View()
	}
}

func (a *App) renderTicket() string {
	if a.activeTicket == nil {
		return "No ticket selected"
	}
	t := a.activeTicket
	var b strings.Builder
	fmt.Fprintf(&b, "Ticket: %s\n", t.ID)
	fmt.Fprintf(&b, "Title:  %s\n", t.Title)
	fmt.Fprintf(&b, "Status: %s\n", t.Status)
	if t.Description != "" {
		fmt.Fprintf(&b, "\n%s\n", t.Description)
	}
	fmt.Fprint(&b, "\nPress Esc to return")
	return b.String()
}

func (a *App) renderHelp() string {
	var b strings.Builder
	fmt.Fprint(&b, "Help — Keybindings\n\n")
	km := keybinding.DefaultKeyMap()
	for _, binding := range km.Bindings {
		fmt.Fprintf(&b, "  %-12s %s\n", binding.Key, binding.Action.String())
	}
	fmt.Fprint(&b, "\nPress ? to return")
	return b.String()
}
