package tui

import (
	"os/exec"
	"strings"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
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
	cr.Register(Command{
		Name:        "config",
		Description: "Edit project config",
		Prefix:      "",
		Items: func() []Item {
			return []Item{
				{Label: "config", Description: "Open config.toml in editor", ID: "ACTION:config"},
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
	case "config":
		ac.app.runCommand = tea.ExecProcess(
			exec.Command("nvim", ac.config.ProjectConfigPath),
			func(err error) tea.Msg { return editorFinishedMsg{err: err} },
		)
	default:
		ac.registry.Set(id)
		ac.app.applyTheme()
		config.SaveTheme(ac.config.ProjectConfigPath, id)
	}
}
