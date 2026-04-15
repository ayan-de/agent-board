package tui

import (
	"fmt"
	"strings"

	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CommandPalette struct {
	commands  *CommandRegistry
	input     string
	filtered  []Item
	cursor    int
	active    bool
	onSelect  func(Item)
	maxHeight int

	width  int
	height int
	theme  *theme.Theme
}

func NewCommandPalette(cr *CommandRegistry, onSelect func(Item)) CommandPalette {
	return CommandPalette{
		commands:  cr,
		onSelect:  onSelect,
		maxHeight: 8,
	}
}

func (p *CommandPalette) SetTheme(t *theme.Theme) {
	p.theme = t
}

func (p CommandPalette) Active() bool {
	return p.active
}

func (p CommandPalette) Input() string {
	return p.input
}

func (p CommandPalette) DropdownHeight() int {
	if !p.active || len(p.filtered) == 0 {
		return 0
	}
	h := len(p.filtered)
	if h > p.maxHeight {
		h = p.maxHeight
	}
	return h
}

func (p *CommandPalette) Open() {
	p.active = true
	p.input = ""
	p.cursor = 0
	p.filterItems()
}

func (p *CommandPalette) Update(msg tea.Msg) (CommandPalette, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		return *p, nil
	case tea.KeyMsg:
		return p.handleKey(msg)
	}
	return *p, nil
}

func (p *CommandPalette) handleKey(msg tea.KeyMsg) (CommandPalette, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		p.active = false
		p.input = ""
		p.filtered = nil
		return *p, nil
	case tea.KeyEnter:
		if len(p.filtered) > 0 && p.cursor < len(p.filtered) {
			if p.onSelect != nil {
				p.onSelect(p.filtered[p.cursor])
			}
		}
		p.active = false
		p.input = ""
		p.filtered = nil
		return *p, nil
	case tea.KeyBackspace:
		if len(p.input) > 0 {
			runes := []rune(p.input)
			p.input = string(runes[:len(runes)-1])
			p.filterItems()
			p.cursor = 0
		}
		return *p, nil
	case tea.KeyRunes:
		runes := string(msg.Runes)
		if len(p.filtered) > 0 {
			switch runes {
			case "j":
				if p.cursor < len(p.filtered)-1 {
					p.cursor++
					if p.onSelect != nil && p.cursor < len(p.filtered) {
						p.onSelect(p.filtered[p.cursor])
					}
				}
				return *p, nil
			case "k":
				if p.cursor > 0 {
					p.cursor--
					if p.onSelect != nil && p.cursor < len(p.filtered) {
						p.onSelect(p.filtered[p.cursor])
					}
				}
				return *p, nil
			}
		}
		p.input += runes
		p.filterItems()
		p.cursor = 0
		return *p, nil
	}

	key := msg.String()
	switch key {
	case "j", "down":
		if p.cursor < len(p.filtered)-1 {
			p.cursor++
			if p.onSelect != nil && p.cursor < len(p.filtered) {
				p.onSelect(p.filtered[p.cursor])
			}
		}
	case "k", "up":
		if p.cursor > 0 {
			p.cursor--
			if p.onSelect != nil && p.cursor < len(p.filtered) {
				p.onSelect(p.filtered[p.cursor])
			}
		}
	}

	return *p, nil
}

func (p *CommandPalette) filterItems() {
	p.filtered = nil
	if p.input == "" {
		return
	}

	for _, cmd := range p.commands.All() {
		if cmd.Items == nil {
			continue
		}
		if !strings.HasPrefix(p.input, cmd.Prefix) && cmd.Prefix != "" {
			continue
		}
		items := cmd.Items()
		query := strings.TrimPrefix(p.input, cmd.Prefix)
		for _, item := range items {
			if query == "" || strings.Contains(strings.ToLower(item.Label), strings.ToLower(query)) {
				p.filtered = append(p.filtered, item)
			}
		}
	}
}

func (p CommandPalette) View() string {
	if !p.active {
		return ""
	}

	primary := lipgloss.Color("69")
	borderColor := lipgloss.Color("240")
	if p.theme != nil {
		primary = p.theme.Primary
		borderColor = p.theme.Border
	}

	inputStyle := lipgloss.NewStyle().
		Foreground(primary).
		Width(p.width - 2)

	inputLine := inputStyle.Render(": " + p.input)

	if len(p.filtered) == 0 {
		return inputLine
	}

	var b strings.Builder
	maxShow := p.maxHeight
	if len(p.filtered) < maxShow {
		maxShow = len(p.filtered)
	}

	for i := 0; i < maxShow; i++ {
		item := p.filtered[i]
		prefix := "  "
		if i == p.cursor {
			prefix = "▸ "
		}

		label := item.Label
		desc := ""
		if item.Description != "" {
			desc = fmt.Sprintf("  %s", lipgloss.NewStyle().Foreground(borderColor).Render(item.Description))
		}

		line := prefix + label + desc
		if i == p.cursor {
			line = lipgloss.NewStyle().Foreground(primary).Bold(true).Render(prefix+label) + desc
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String() + inputLine
}
