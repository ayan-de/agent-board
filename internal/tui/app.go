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

type App struct {
	store    *store.Store
	resolver *keybinding.Resolver
	config   *config.Config
	registry *theme.Registry
	width    int
	height   int

	focus   focusArea
	view    viewMode
	palette CommandPalette

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

	cr := NewCommandRegistry()
	cr.Register(Command{
		Name:        "theme",
		Description: "Change color theme",
		Prefix:      "/",
		Items: func() []Item {
			themes := reg.All()
			items := make([]Item, len(themes))
			for i, th := range themes {
				items[i] = Item{
					Label:       th.Name,
					Description: th.Source,
					ID:          th.Name,
				}
			}
			return items
		},
	})

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

	a.palette = NewCommandPalette(cr, nil)
	a.palette.SetTheme(t)
	a.palette.onSelect = func(item Item) {
		a.registry.Set(item.ID)
		a.applyTheme()
	}

	return a, nil
}

func (a *App) applyTheme() {
	t := a.registry.Active()
	a.kanban.styles = NewKanbanStyles(t)
	a.ticketView.styles = NewTicketViewStyles(t)
	a.dashboard.styles = NewDashboardStyles(t)
	a.palette.SetTheme(t)
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
		return a, nil
	case tea.KeyMsg:
		return a.handleKey(msg)
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.palette.Active() {
		a.palette, _ = a.palette.Update(msg)
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
	paletteView := a.palette.View()
	paletteLines := 0
	if a.palette.Active() {
		paletteLines = a.palette.DropdownHeight() + 1
	}

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

	if paletteLines > 0 {
		return lipgloss.JoinVertical(lipgloss.Bottom,
			lipgloss.NewStyle().Height(a.height-paletteLines).Render(mainView),
			paletteView,
		)
	}

	return lipgloss.NewStyle().Height(a.height).Render(mainView)
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
