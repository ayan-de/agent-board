package tui

import (
	"fmt"
	"path/filepath"
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
	cr.Register(Command{
		Name:        "import theme",
		Description: "Import a theme from JSON",
		Prefix:      "",
		Items: func() []Item {
			return []Item{
				{Label: "import theme", Description: "Paste theme JSON to import", ID: "ACTION:import_theme"},
			}
		},
	})
	cr.Register(Command{
		Name:        "orchestrator",
		Description: "View orchestrator configuration",
		Prefix:      "",
		Items: func() []Item {
			return []Item{
				{Label: "orchestrator", Description: "View LLM and agent settings", ID: "ACTION:orchestrator"},
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
	ac.app.propagateTheme()
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
		ac.app.openConfigEditor()
	case "import_theme":
		themesDir := filepath.Join(config.GetBaseDir(), "themes")
		ac.app.modal.Open(
			"Import Theme",
			fmt.Sprintf("Generate your theme at ayande.xyz and save the JSON file to:\n%s", themesDir),
			func() tea.Cmd {
				ac.registry.LoadUserThemes(themesDir)
				themes := ac.registry.All()
				var userThemes []string
				for _, t := range themes {
					if t.Source == "user" {
						userThemes = append(userThemes, t.Name)
					}
				}
				if len(userThemes) == 0 {
					return func() tea.Msg {
						return notificationMsg{
							title:   "No Themes Found",
							message: "No user themes found in " + themesDir,
							variant: NotificationWarning,
						}
					}
				}
				return func() tea.Msg {
					return notificationMsg{
						title:   "Theme Loaded",
						message: fmt.Sprintf("Found %d theme(s). Use /theme to select one.", len(userThemes)),
						variant: NotificationSuccess,
					}
				}
			},
			nil,
		)
	case "orchestrator":
		llm := ac.config.LLM
		msg := fmt.Sprintf(`[llm]
provider = %q
model = %q
base_url = %q
coordinator_model = %q
summarizer_model = %q
require_approval = %v`, llm.Provider, llm.Model, llm.BaseURL, llm.CoordinatorModel, llm.SummarizerModel, llm.RequireApproval)
		ac.app.modal.OpenInfo("Orchestrator Configuration", msg, nil)
	default:
		ac.registry.Set(id)
		ac.app.propagateTheme()
		config.SaveTheme(ac.config.ProjectConfigPath, id)
	}
}
