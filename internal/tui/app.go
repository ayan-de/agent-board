package tui

import (
	"context"
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
	switch a.view {
	case viewHelp:
		return a.renderHelp()
	case viewTicket:
		return a.renderTicket()
	default:
		return a.renderBoard()
	}
}

func (a *App) renderBoard() string {
	var b strings.Builder

	b.WriteString("AgentBoard")
	if a.width > 0 {
		b.WriteString(fmt.Sprintf("  [%dx%d]", a.width, a.height))
	}
	b.WriteString("\n\n")

	colWidth := a.width / 4
	if colWidth < 20 {
		colWidth = 20
	}

	for i, name := range columnNames {
		if i == a.colIndex {
			b.WriteString(fmt.Sprintf("▶ %s", name))
		} else {
			b.WriteString(fmt.Sprintf("  %s", name))
		}
		if i < 3 {
			pad := colWidth - len(name) - 2
			if pad > 0 {
				b.WriteString(strings.Repeat(" ", pad))
			}
		}
	}
	b.WriteString("\n")

	for i := 0; i < 4; i++ {
		cursor := a.cursors[i]
		for j, ticket := range a.columns[i] {
			prefix := "  "
			if i == a.colIndex && j == cursor {
				prefix = "▸ "
			}
			line := fmt.Sprintf("%s%s %s", prefix, ticket.ID, ticket.Title)
			if len(line) > colWidth {
				line = line[:colWidth-1] + "…"
			}
			b.WriteString(line)
			if i < 3 {
				pad := colWidth - len(line)
				if pad > 0 {
					b.WriteString(strings.Repeat(" ", pad))
				}
			}
			b.WriteString("\n")
		}
		if len(a.columns[i]) == 0 {
			b.WriteString("  (empty)")
			if i < 3 {
				pad := colWidth - 9
				if pad > 0 {
					b.WriteString(strings.Repeat(" ", pad))
				}
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (a *App) renderTicket() string {
	if a.activeTicket == nil {
		return "No ticket selected"
	}
	t := a.activeTicket
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Ticket: %s\n", t.ID))
	b.WriteString(fmt.Sprintf("Title:  %s\n", t.Title))
	b.WriteString(fmt.Sprintf("Status: %s\n", t.Status))
	if t.Description != "" {
		b.WriteString(fmt.Sprintf("\n%s\n", t.Description))
	}
	b.WriteString("\nPress Esc to return")
	return b.String()
}

func (a *App) renderHelp() string {
	var b strings.Builder
	b.WriteString("Help — Keybindings\n\n")
	km := keybinding.DefaultKeyMap()
	for _, binding := range km.Bindings {
		b.WriteString(fmt.Sprintf("  %-12s %s\n", binding.Key, binding.Action.String()))
	}
	b.WriteString("\nPress ? to return")
	return b.String()
}
