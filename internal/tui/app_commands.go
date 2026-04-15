package tui

import (
	"strings"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/theme"
)

type appCommands struct {
	app      *App
	registry *theme.Registry
	config   *config.Config
}

func newAppCommands(a *App, reg *theme.Registry, cfg *config.Config) *appCommands {
	return &appCommands{app: a, registry: reg, config: cfg}
}

func (ac *appCommands) registerAll(cr *CommandRegistry) {
	cr.Register(Command{
		Name:        "theme",
		Description: "Change color theme",
		Prefix:      "/",
		Items:       ac.themeItems,
	})
	cr.Register(Command{
		Name:        "quit",
		Description: "Quit AgentBoard",
		Prefix:      "",
		Items: func() []Item {
			return []Item{
				{Label: "quit", Description: "Exit the application", ID: "ACTION:quit"},
			}
		},
	})
}

func (ac *appCommands) themeItems() []Item {
	themes := ac.registry.All()
	items := make([]Item, len(themes))
	for i, th := range themes {
		items[i] = Item{
			Label:       th.Name,
			Description: th.Source,
			ID:          th.Name,
		}
	}
	return items
}

func (ac *appCommands) onSelect(item Item) {
	ac.registry.Set(item.ID)
	ac.app.applyTheme()
}

func (ac *appCommands) onConfirm(item Item) {
	id := item.ID
	if strings.HasPrefix(id, "ACTION:") {
		id = strings.TrimPrefix(id, "ACTION:")
	}
	switch id {
	case "quit":
		ac.app.quit = true
	default:
		ac.registry.Set(id)
		ac.app.applyTheme()
		config.SaveTheme(ac.config.ProjectConfigPath, id)
	}
}
