package theme

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Name   string
	Source string

	Primary           lipgloss.Color
	Secondary         lipgloss.Color
	Accent            lipgloss.Color
	Error             lipgloss.Color
	Warning           lipgloss.Color
	Success           lipgloss.Color
	Info              lipgloss.Color
	Text              lipgloss.Color
	TextMuted         lipgloss.Color
	Background        lipgloss.Color
	BackgroundPanel   lipgloss.Color
	BackgroundElement lipgloss.Color
	Border            lipgloss.Color
	BorderActive      lipgloss.Color
}

type Registry struct {
	themes map[string]*Theme
	active string
	mode   string
}

func NewRegistry(mode string) *Registry {
	return &Registry{
		themes: make(map[string]*Theme),
		mode:   mode,
	}
}

func (r *Registry) Register(t *Theme) {
	r.themes[t.Name] = t
	if r.active == "" {
		r.active = t.Name
	}
}

func (r *Registry) Active() *Theme {
	return r.themes[r.active]
}

func (r *Registry) Set(name string) error {
	if _, ok := r.themes[name]; !ok {
		return fmt.Errorf("theme.registry: theme %q not found", name)
	}
	r.active = name
	return nil
}

func (r *Registry) All() []*Theme {
	all := make([]*Theme, 0, len(r.themes))
	for _, t := range r.themes {
		all = append(all, t)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})
	return all
}

func (r *Registry) Mode() string {
	return r.mode
}
