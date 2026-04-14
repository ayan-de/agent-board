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
