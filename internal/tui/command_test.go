package tui

import (
	"testing"
)

func TestCommandRegistry(t *testing.T) {
	cr := NewCommandRegistry()

	cr.Register(Command{
		Name:        "theme",
		Description: "Change color theme",
		Prefix:      "/",
		Items: func() []Item {
			return []Item{
				{Label: "dracula", Description: "Dracula theme", ID: "dracula"},
			}
		},
	})

	cmds := cr.All()
	if len(cmds) != 1 {
		t.Fatalf("All() returned %d commands, want 1", len(cmds))
	}
	if cmds[0].Name != "theme" {
		t.Errorf("Name = %q, want %q", cmds[0].Name, "theme")
	}

	items := cmds[0].Items()
	if len(items) != 1 {
		t.Fatalf("Items() returned %d items, want 1", len(items))
	}
	if items[0].Label != "dracula" {
		t.Errorf("Label = %q, want %q", items[0].Label, "dracula")
	}
}

func TestCommandRegistryFilterByPrefix(t *testing.T) {
	cr := NewCommandRegistry()
	cr.Register(Command{Name: "theme", Description: "Theme", Prefix: "/", Items: func() []Item { return nil }})
	cr.Register(Command{Name: "keybindings", Description: "Keys", Prefix: "/", Items: func() []Item { return nil }})

	filtered := cr.Filter("/theme")
	if len(filtered) != 1 {
		t.Fatalf("Filter(/theme) returned %d, want 1", len(filtered))
	}
	if filtered[0].Name != "theme" {
		t.Errorf("Filter result Name = %q, want %q", filtered[0].Name, "theme")
	}
}
