package tui

import (
	"context"
	"fmt"

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

var columnNames = [4]string{"Backlog", "In Progress", "Review", "Done"}

type App struct {
	store    *store.Store
	resolver *keybinding.Resolver
	config   *config.Config
	width    int
	height   int

	focus focusArea
	view  viewMode

	colIndex int
	cursors  [4]int
	columns  [4][]store.Ticket

	activeTicket *store.Ticket
}

func NewApp(cfg *config.Config, s *store.Store) (*App, error) {
	km := keybinding.DefaultKeyMap()
	if len(cfg.TUI.Keybindings) > 0 {
		keybinding.ApplyConfig(&km, cfg.TUI.Keybindings)
	}

	a := &App{
		store:    s,
		resolver: keybinding.NewResolver(km),
		config:   cfg,
		focus:    focusBoard,
		view:     viewBoard,
	}

	if err := a.loadColumns(); err != nil {
		return nil, fmt.Errorf("tui.newApp: %w", err)
	}

	return a, nil
}

func (a *App) loadColumns() error {
	statuses := [4]string{"backlog", "in_progress", "review", "done"}
	for i, status := range statuses {
		tickets, err := a.store.ListTickets(context.Background(), store.TicketFilters{Status: status})
		if err != nil {
			return fmt.Errorf("tui.loadColumns: %w", err)
		}
		if tickets == nil {
			tickets = []store.Ticket{}
		}
		a.columns[i] = tickets
	}
	for i := range a.cursors {
		if a.cursors[i] >= len(a.columns[i]) && len(a.columns[i]) > 0 {
			a.cursors[i] = len(a.columns[i]) - 1
		}
	}
	return nil
}

func (a *App) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	case tea.KeyMsg:
		return a.handleKey(msg)
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	action, _ := a.resolver.Resolve(key)

	switch action {
	case keybinding.ActionQuit, keybinding.ActionForceQuit:
		return a, tea.Quit
	case keybinding.ActionPrevColumn:
		if a.colIndex > 0 {
			a.colIndex--
		}
	case keybinding.ActionNextColumn:
		if a.colIndex < 3 {
			a.colIndex++
		}
	case keybinding.ActionPrevTicket:
		if a.cursors[a.colIndex] > 0 {
			a.cursors[a.colIndex]--
		}
	case keybinding.ActionNextTicket:
		if a.cursors[a.colIndex] < len(a.columns[a.colIndex])-1 {
			a.cursors[a.colIndex]++
		}
	case keybinding.ActionJumpColumn1:
		a.colIndex = 0
	case keybinding.ActionJumpColumn2:
		a.colIndex = 1
	case keybinding.ActionJumpColumn3:
		a.colIndex = 2
	case keybinding.ActionJumpColumn4:
		a.colIndex = 3
	case keybinding.ActionAddTicket:
		_, err := a.store.CreateTicket(context.Background(), store.Ticket{
			Title:  "New Ticket",
			Status: "backlog",
		})
		if err != nil {
			return a, nil
		}
		_ = a.loadColumns()
	case keybinding.ActionDeleteTicket:
		col := a.columns[a.colIndex]
		if len(col) > 0 {
			cursor := a.cursors[a.colIndex]
			_ = a.store.DeleteTicket(context.Background(), col[cursor].ID)
			_ = a.loadColumns()
		}
	case keybinding.ActionOpenTicket:
		col := a.columns[a.colIndex]
		if len(col) > 0 {
			ticket := col[a.cursors[a.colIndex]]
			a.activeTicket = &ticket
			a.view = viewTicket
		}
	case keybinding.ActionShowHelp:
		if a.view == viewHelp {
			a.view = viewBoard
		} else {
			a.view = viewHelp
		}
	}

	return a, nil
}

func (a *App) View() string {
	return ""
}
