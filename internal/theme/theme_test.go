package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRegistryNewRegistry(t *testing.T) {
	r := NewRegistry("dark")
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if r.Active() != nil {
		t.Error("Active() should be nil on empty registry")
	}
}

func TestRegistryRegisterAndActive(t *testing.T) {
	r := NewRegistry("dark")
	th := &Theme{
		Name:    "test",
		Source:  "builtin",
		Primary: lipgloss.Color("#ff0000"),
	}
	r.Register(th)

	active := r.Active()
	if active == nil {
		t.Fatal("Active() is nil after registering default theme")
	}
	if active.Name != "test" {
		t.Errorf("Active().Name = %q, want %q", active.Name, "test")
	}
}

func TestRegistrySet(t *testing.T) {
	r := NewRegistry("dark")
	r.Register(&Theme{Name: "alpha", Source: "builtin", Primary: lipgloss.Color("#111111")})
	r.Register(&Theme{Name: "beta", Source: "builtin", Primary: lipgloss.Color("#222222")})

	err := r.Set("beta")
	if err != nil {
		t.Fatalf("Set() error: %v", err)
	}
	if r.Active().Name != "beta" {
		t.Errorf("Active().Name = %q, want %q", r.Active().Name, "beta")
	}
}

func TestRegistrySetNotFound(t *testing.T) {
	r := NewRegistry("dark")
	r.Register(&Theme{Name: "alpha", Source: "builtin", Primary: lipgloss.Color("#111111")})

	err := r.Set("nonexistent")
	if err == nil {
		t.Error("Set() should return error for nonexistent theme")
	}
}

func TestRegistryAll(t *testing.T) {
	r := NewRegistry("dark")
	r.Register(&Theme{Name: "zeta", Source: "builtin", Primary: lipgloss.Color("#111111")})
	r.Register(&Theme{Name: "alpha", Source: "builtin", Primary: lipgloss.Color("#222222")})

	all := r.All()
	if len(all) != 2 {
		t.Fatalf("All() returned %d themes, want 2", len(all))
	}
	if all[0].Name != "alpha" {
		t.Errorf("All()[0].Name = %q, want %q (sorted)", all[0].Name, "alpha")
	}
	if all[1].Name != "zeta" {
		t.Errorf("All()[1].Name = %q, want %q (sorted)", all[1].Name, "zeta")
	}
}

func TestRegistryOverride(t *testing.T) {
	r := NewRegistry("dark")
	r.Register(&Theme{Name: "test", Source: "builtin", Primary: lipgloss.Color("#111111")})
	r.Register(&Theme{Name: "test", Source: "user", Primary: lipgloss.Color("#222222")})

	all := r.All()
	if len(all) != 1 {
		t.Fatalf("All() returned %d themes after override, want 1", len(all))
	}
	if all[0].Source != "user" {
		t.Errorf("overridden theme source = %q, want %q", all[0].Source, "user")
	}
}

func TestRegistryMode(t *testing.T) {
	r := NewRegistry("light")
	if r.Mode() != "light" {
		t.Errorf("Mode() = %q, want %q", r.Mode(), "light")
	}
}
