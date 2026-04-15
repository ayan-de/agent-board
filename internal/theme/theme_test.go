package theme

import (
	"os"
	"path/filepath"
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

func TestParseThemeJSONDirect(t *testing.T) {
	jsonData := `{
		"name": "Test Theme",
		"defs": {
			"bg": "#1a1b26",
			"fg": "#c0caf5"
		},
		"theme": {
			"primary": "#7aa2f7",
			"secondary": "#bb9af7",
			"accent": "#7dcfff",
			"error": "#f7768e",
			"warning": "#e0af68",
			"success": "#9ece6a",
			"info": "#7dcfff",
			"text": { "dark": "#c0caf5", "light": "#1a1b26" },
			"textMuted": "#565f89",
			"background": "#1a1b26",
			"backgroundPanel": "#16161e",
			"backgroundElement": "#24283b",
			"border": "#3b4261",
			"borderActive": "#7aa2f7"
		}
	}`

	th, err := parseThemeJSON([]byte(jsonData), "dark", "user")
	if err != nil {
		t.Fatalf("parseThemeJSON() error: %v", err)
	}
	if th.Name != "Test Theme" {
		t.Errorf("Name = %q, want %q", th.Name, "Test Theme")
	}
	if th.Source != "user" {
		t.Errorf("Source = %q, want %q", th.Source, "user")
	}
	if th.Primary != lipgloss.Color("#7aa2f7") {
		t.Errorf("Primary = %q, want %q", th.Primary, "#7aa2f7")
	}
	if th.Text != lipgloss.Color("#c0caf5") {
		t.Errorf("Text = %q, want %q (dark variant)", th.Text, "#c0caf5")
	}
}

func TestParseThemeJSONDefsResolution(t *testing.T) {
	jsonData := `{
		"name": "Defs Test",
		"defs": {
			"myblue": "#0000ff",
			"myred": "#ff0000"
		},
		"theme": {
			"primary": "myblue",
			"secondary": "myred",
			"accent": "#00ff00",
			"error": "#ff0000",
			"warning": "#ffff00",
			"success": "#00ff00",
			"info": "#0000ff",
			"text": "#ffffff",
			"textMuted": "#888888",
			"background": "#000000",
			"backgroundPanel": "#111111",
			"backgroundElement": "#222222",
			"border": "#333333",
			"borderActive": "myblue"
		}
	}`

	th, err := parseThemeJSON([]byte(jsonData), "dark", "builtin")
	if err != nil {
		t.Fatalf("parseThemeJSON() error: %v", err)
	}
	if th.Primary != lipgloss.Color("#0000ff") {
		t.Errorf("Primary = %q, want %q (resolved from defs)", th.Primary, "#0000ff")
	}
	if th.Secondary != lipgloss.Color("#ff0000") {
		t.Errorf("Secondary = %q, want %q (resolved from defs)", th.Secondary, "#ff0000")
	}
	if th.BorderActive != lipgloss.Color("#0000ff") {
		t.Errorf("BorderActive = %q, want %q (resolved from defs)", th.BorderActive, "#0000ff")
	}
}

func TestParseThemeJSONLightVariant(t *testing.T) {
	jsonData := `{
		"name": "Variant Test",
		"theme": {
			"primary": "#7aa2f7",
			"secondary": "#bb9af7",
			"accent": "#7dcfff",
			"error": "#f7768e",
			"warning": "#e0af68",
			"success": "#9ece6a",
			"info": "#7dcfff",
			"text": { "dark": "#c0caf5", "light": "#1a1b26" },
			"textMuted": "#565f89",
			"background": { "dark": "#1a1b26", "light": "#d5d6db" },
			"backgroundPanel": "#16161e",
			"backgroundElement": "#24283b",
			"border": "#3b4261",
			"borderActive": "#7aa2f7"
		}
	}`

	th, err := parseThemeJSON([]byte(jsonData), "light", "user")
	if err != nil {
		t.Fatalf("parseThemeJSON() error: %v", err)
	}
	if th.Text != lipgloss.Color("#1a1b26") {
		t.Errorf("Text = %q, want %q (light variant)", th.Text, "#1a1b26")
	}
	if th.Background != lipgloss.Color("#d5d6db") {
		t.Errorf("Background = %q, want %q (light variant)", th.Background, "#d5d6db")
	}
}

func TestParseThemeJSONMissingOptionalColors(t *testing.T) {
	jsonData := `{
		"name": "Minimal",
		"theme": {
			"primary": "#7aa2f7"
		}
	}`

	th, err := parseThemeJSON([]byte(jsonData), "dark", "user")
	if err != nil {
		t.Fatalf("parseThemeJSON() error: %v", err)
	}
	if th.Primary != lipgloss.Color("#7aa2f7") {
		t.Errorf("Primary = %q, want %q", th.Primary, "#7aa2f7")
	}
	if th.Secondary != lipgloss.Color("") {
		t.Errorf("Secondary = %q, want empty (missing)", th.Secondary)
	}
}

func TestLoadFromFS(t *testing.T) {
	dir := t.TempDir()
	themeJSON := `{
		"name": "FS Theme",
		"theme": {
			"primary": "#ff0000",
			"secondary": "#00ff00",
			"accent": "#0000ff",
			"error": "#ff0000",
			"warning": "#ffff00",
			"success": "#00ff00",
			"info": "#0000ff",
			"text": "#ffffff",
			"textMuted": "#888888",
			"background": "#000000",
			"backgroundPanel": "#111111",
			"backgroundElement": "#222222",
			"border": "#333333",
			"borderActive": "#ff0000"
		}
	}`
	os.WriteFile(filepath.Join(dir, "mytheme.json"), []byte(themeJSON), 0644)

	themes := loadFromFS(dir, "dark", "user")
	if len(themes) != 1 {
		t.Fatalf("loadFromFS returned %d themes, want 1", len(themes))
	}
	if themes[0].Name != "FS Theme" {
		t.Errorf("Name = %q, want %q", themes[0].Name, "FS Theme")
	}
	if themes[0].Source != "user" {
		t.Errorf("Source = %q, want %q", themes[0].Source, "user")
	}
}

func TestLoadFromFSSkipsNonJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not a theme"), 0644)
	os.WriteFile(filepath.Join(dir, ".hidden.json"), []byte(`{"name":"hidden"}`), 0644)

	themes := loadFromFS(dir, "dark", "user")
	if len(themes) != 0 {
		t.Errorf("loadFromFS returned %d themes, want 0 (non-json skipped)", len(themes))
	}
}

func TestRegisterBuiltins(t *testing.T) {
	r := NewRegistry("dark")
	registerBuiltins(r)

	all := r.All()
	if len(all) < 7 {
		t.Errorf("got %d built-in themes, want at least 7", len(all))
	}

	names := make(map[string]bool)
	for _, th := range all {
		names[th.Name] = true
		if th.Source != "builtin" {
			t.Errorf("theme %q source = %q, want %q", th.Name, th.Source, "builtin")
		}
	}

	expected := []string{"agentboard", "dracula", "gruvbox", "tokyonight", "nord", "catppuccin", "matrix"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing built-in theme %q", name)
		}
	}
}

func TestBuiltinAgentboardThemeHasAllColors(t *testing.T) {
	r := NewRegistry("dark")
	registerBuiltins(r)

	err := r.Set("agentboard")
	if err != nil {
		t.Fatalf("Set(agentboard) error: %v", err)
	}

	th := r.Active()
	if th.Primary == "" {
		t.Error("Primary is empty")
	}
	if th.Text == "" {
		t.Error("Text is empty")
	}
	if th.Background == "" {
		t.Error("Background is empty")
	}
	if th.Border == "" {
		t.Error("Border is empty")
	}
}

func TestRegistryLoadUserThemes(t *testing.T) {
	r := NewRegistry("dark")
	registerBuiltins(r)

	userDir := t.TempDir()
	themeJSON := `{
		"name": "My Custom",
		"theme": {
			"primary": "#ff0000",
			"text": "#ffffff",
			"background": "#000000"
		}
	}`
	os.WriteFile(filepath.Join(userDir, "custom.json"), []byte(themeJSON), 0644)

	r.LoadUserThemes(userDir)

	err := r.Set("My Custom")
	if err != nil {
		t.Fatalf("Set(My Custom) error: %v", err)
	}
	th := r.Active()
	if th.Source != "user" {
		t.Errorf("source = %q, want %q", th.Source, "user")
	}
	if th.Primary != lipgloss.Color("#ff0000") {
		t.Errorf("Primary = %q, want %q", th.Primary, "#ff0000")
	}
}

func TestRegistryLoadUserThemesOverridesBuiltin(t *testing.T) {
	r := NewRegistry("dark")
	registerBuiltins(r)

	userDir := t.TempDir()
	themeJSON := `{
		"name": "dracula",
		"theme": {
			"primary": "#custom"
		}
	}`
	os.WriteFile(filepath.Join(userDir, "dracula.json"), []byte(themeJSON), 0644)

	r.LoadUserThemes(userDir)

	err := r.Set("dracula")
	if err != nil {
		t.Fatalf("Set(dracula) error: %v", err)
	}
	th := r.Active()
	if th.Source != "user" {
		t.Errorf("source = %q, want %q (user override)", th.Source, "user")
	}
	if th.Primary != lipgloss.Color("#custom") {
		t.Errorf("Primary = %q, want %q (user value)", th.Primary, "#custom")
	}
}
