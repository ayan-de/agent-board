package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestPalette() CommandPalette {
	cr := NewCommandRegistry()
	cr.Register(Command{
		Name:        "theme",
		Description: "Change color theme",
		Prefix:      "/",
		Items: func() []Item {
			return []Item{
				{Label: "agentboard", Description: "Default", ID: "agentboard"},
				{Label: "dracula", Description: "Dracula", ID: "dracula"},
				{Label: "gruvbox", Description: "Gruvbox", ID: "gruvbox"},
			}
		},
	})

	var selected Item
	p := NewCommandPalette(cr, func(item Item) {
		selected = item
	})
	p.width = 120
	p.height = 40
	_ = selected
	return p
}

func TestPaletteOpenClose(t *testing.T) {
	p := newTestPalette()

	if p.Active() {
		t.Error("palette should not be active initially")
	}

	p.Open()
	if !p.Active() {
		t.Error("palette should be active after Open()")
	}

	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if p.Active() {
		t.Error("palette should not be active after Esc")
	}
}

func TestPaletteFilterItems(t *testing.T) {
	p := newTestPalette()
	p.Open()

	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d', 'r'}})

	if len(p.filtered) == 0 {
		t.Error("filtered items should not be empty after typing /dr")
	}
	for _, item := range p.filtered {
		if item.Label != "dracula" {
			t.Errorf("expected dracula, got %q", item.Label)
		}
	}
}

func TestPaletteNavigation(t *testing.T) {
	p := newTestPalette()
	p.Open()

	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	if len(p.filtered) != 3 {
		t.Fatalf("expected 3 items with '/' prefix, got %d", len(p.filtered))
	}
	if p.cursor != 0 {
		t.Errorf("cursor = %d, want 0 initially", p.cursor)
	}

	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.cursor != 1 {
		t.Errorf("cursor = %d after j, want 1", p.cursor)
	}

	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.cursor != 0 {
		t.Errorf("cursor = %d after k, want 0", p.cursor)
	}
}

func TestPaletteSelection(t *testing.T) {
	var selected Item
	cr := NewCommandRegistry()
	cr.Register(Command{
		Name:   "theme",
		Prefix: "/",
		Items: func() []Item {
			return []Item{
				{Label: "dracula", Description: "Dracula", ID: "dracula"},
			}
		},
	})

	p := NewCommandPalette(cr, nil)
	p.onConfirm = func(item Item) {
		selected = item
	}
	p.width = 120
	p.height = 40
	p.Open()

	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if p.Active() {
		t.Error("palette should close after Enter")
	}
	if selected.ID != "dracula" {
		t.Errorf("selected.ID = %q, want %q", selected.ID, "dracula")
	}
}

func TestPaletteViewRenders(t *testing.T) {
	p := newTestPalette()
	p.Open()
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	view := p.View()
	if len(view) == 0 {
		t.Error("View() returned empty string")
	}
}

func TestPaletteHeight(t *testing.T) {
	p := newTestPalette()
	p.Open()

	h := p.DropdownHeight()
	if h < 0 {
		t.Errorf("DropdownHeight() = %d, want >= 0", h)
	}
}

func TestPaletteInput(t *testing.T) {
	p := newTestPalette()
	p.Open()

	if p.Input() != "" {
		t.Errorf("Input() = %q, want empty initially", p.Input())
	}
}
