# Theme System & Command Palette Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a scalable theme system with 7 built-in themes, user-customizable themes via JSON, and an extensible command palette for the AgentBoard TUI.

**Architecture:** New `internal/theme` package holds a Registry that loads embedded built-in themes and filesystem user themes. A new `CommandPalette` bubbletea model in `internal/tui/` provides a bottom-bar input with upward dropdown for command selection. All views receive colors from the active theme via constructors instead of hardcoded ANSI codes.

**Tech Stack:** Go, bubbletea, lipgloss, go:embed, encoding/json

---

## File Structure

### New files

| File | Purpose |
|---|---|
| `internal/theme/theme.go` | Theme struct, Registry struct, NewRegistry, Active, Set, All, Register |
| `internal/theme/loader.go` | parseThemeJSON, resolveColor, loadFromFS, LoadUserThemes |
| `internal/theme/builtin.go` | go:embed directive, registerBuiltins |
| `internal/theme/theme_test.go` | Tests for theme package |
| `internal/theme/themes/agentboard.json` | Default theme (ported from opencode.json) |
| `internal/theme/themes/dracula.json` | Dracula theme |
| `internal/theme/themes/gruvbox.json` | Gruvbox theme |
| `internal/theme/themes/tokyonight.json` | Tokyo Night theme |
| `internal/theme/themes/nord.json` | Nord theme |
| `internal/theme/themes/catppuccin.json` | Catppuccin theme |
| `internal/theme/themes/matrix.json` | Matrix theme |
| `internal/tui/palette.go` | CommandPalette bubbletea model |
| `internal/tui/palette_test.go` | Palette tests |
| `internal/tui/command.go` | Command struct, Item struct, command registry |
| `internal/tui/command_test.go` | Command tests |

### Modified files

| File | Change |
|---|---|
| `internal/tui/app.go` | Add *theme.Registry, CommandPalette, applyTheme, palette key routing |
| `internal/tui/kanban.go` | DefaultKanbanStyles → NewKanbanStyles(t *theme.Theme) |
| `internal/tui/ticketview.go` | DefaultTicketViewStyles → NewTicketViewStyles(t *theme.Theme) |
| `internal/tui/dashboard.go` | DefaultDashboardStyles → NewDashboardStyles(t *theme.Theme) |
| `internal/keybinding/action.go` | Add ActionOpenPalette |
| `internal/keybinding/keymap.go` | Add `:` → ActionOpenPalette binding |
| `internal/config/defaults.go` | Change default theme from "default" to "agentboard" |
| `cmd/agentboard/main.go` | Create theme.Registry, load themes, pass to NewApp |

---

## Task 1: Theme Struct & Registry

**Files:**
- Create: `internal/theme/theme.go`
- Create: `internal/theme/theme_test.go`

- [ ] **Step 1: Write the failing test for Theme struct and Registry**

Create `internal/theme/theme_test.go`:

```go
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
		Name:   "test",
		Source: "builtin",
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/theme/ -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Write minimal implementation**

Create `internal/theme/theme.go`:

```go
package theme

import (
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/theme/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/theme/theme.go internal/theme/theme_test.go
git commit -m "feat(theme): add Theme struct and Registry"
```

---

## Task 2: Theme JSON Loader

**Files:**
- Create: `internal/theme/loader.go`
- Modify: `internal/theme/theme_test.go` — add loader tests

- [ ] **Step 1: Write the failing test for JSON parsing**

Append to `internal/theme/theme_test.go`:

```go
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

func TestParseThemeJSONMissingNameUsesFilename(t *testing.T) {
	jsonData := `{
		"theme": {
			"primary": "#7aa2f7",
			"secondary": "#bb9af7",
			"accent": "#7dcfff",
			"error": "#f7768e",
			"warning": "#e0af68",
			"success": "#9ece6a",
			"info": "#7dcfff",
			"text": "#ffffff",
			"textMuted": "#888888",
			"background": "#000000",
			"backgroundPanel": "#111111",
			"backgroundElement": "#222222",
			"border": "#333333",
			"borderActive": "#7aa2f7"
		}
	}`

	th, err := parseThemeJSON([]byte(jsonData), "dark", "user")
	if err != nil {
		t.Fatalf("parseThemeJSON() error: %v", err)
	}
	if th.Name != "" {
		t.Errorf("Name = %q, want empty (caller sets from filename)", th.Name)
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

	themes := loadFromFS(dir, "user")
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

	themes := loadFromFS(dir, "user")
	if len(themes) != 0 {
		t.Errorf("loadFromFS returned %d themes, want 0 (non-json skipped)", len(themes))
	}
}
```

Add required imports to test file: `"os"`, `"path/filepath"`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/theme/ -v -run TestParse`
Expected: FAIL — `parseThemeJSON` and `loadFromFS` not defined

- [ ] **Step 3: Write the loader implementation**

Create `internal/theme/loader.go`:

```go
package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type themeJSON struct {
	Name  string            `json:"name"`
	Defs  map[string]string `json:"defs"`
	Theme map[string]any    `json:"theme"`
}

type variantJSON struct {
	Dark  string `json:"dark"`
	Light string `json:"light"`
}

func parseThemeJSON(data []byte, mode string, source string) (*Theme, error) {
	var tj themeJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, fmt.Errorf("theme.parseThemeJSON: %w", err)
	}

	resolved := make(map[string]string)
	for k, v := range tj.Defs {
		resolved[k] = v
	}

	th := &Theme{
		Name:   tj.Name,
		Source: source,
	}

	colorKeys := map[string]*lipgloss.Color{
		"primary":           &th.Primary,
		"secondary":         &th.Secondary,
		"accent":            &th.Accent,
		"error":             &th.Error,
		"warning":           &th.Warning,
		"success":           &th.Success,
		"info":              &th.Info,
		"text":              &th.Text,
		"textMuted":         &th.TextMuted,
		"background":        &th.Background,
		"backgroundPanel":   &th.BackgroundPanel,
		"backgroundElement": &th.BackgroundElement,
		"border":            &th.Border,
		"borderActive":      &th.BorderActive,
	}

	for key, ptr := range colorKeys {
		val, ok := tj.Theme[key]
		if !ok {
			continue
		}
		color, err := resolveColor(val, mode, resolved)
		if err != nil {
			return nil, fmt.Errorf("theme.parseThemeJSON: key %q: %w", key, err)
		}
		*ptr = lipgloss.Color(color)
	}

	return th, nil
}

func resolveColor(val any, mode string, resolved map[string]string) (string, error) {
	switch v := val.(type) {
	case string:
		if strings.HasPrefix(v, "#") {
			return v, nil
		}
		if r, ok := resolved[v]; ok {
			return r, nil
		}
		return v, nil
	case map[string]any:
		dark, _ := v["dark"].(string)
		light, _ := v["light"].(string)
		if mode == "light" && light != "" {
			return resolveColor(light, mode, resolved)
		}
		if dark != "" {
			return resolveColor(dark, mode, resolved)
		}
		return "", fmt.Errorf("variant has no color for mode %q", mode)
	default:
		return "", fmt.Errorf("unexpected color type %T", val)
	}
}

func loadFromFS(dir string, source string) []*Theme {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var themes []*Theme
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		th, err := parseThemeJSON(data, "dark", source)
		if err != nil {
			continue
		}

		if th.Name == "" {
			th.Name = strings.TrimSuffix(entry.Name(), ".json")
		}

		themes = append(themes, th)
	}
	return themes
}
```

Add `"fmt"` to imports in `theme.go` if not already present.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/theme/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/theme/loader.go internal/theme/theme_test.go
git commit -m "feat(theme): add JSON theme loader with defs and variant resolution"
```

---

## Task 3: Built-in Themes (go:embed)

**Files:**
- Create: `internal/theme/builtin.go`
- Create: `internal/theme/themes/agentboard.json`
- Create: `internal/theme/themes/dracula.json`
- Create: `internal/theme/themes/gruvbox.json`
- Create: `internal/theme/themes/tokyonight.json`
- Create: `internal/theme/themes/nord.json`
- Create: `internal/theme/themes/catppuccin.json`
- Create: `internal/theme/themes/matrix.json`
- Modify: `internal/theme/theme_test.go` — add builtin tests

- [ ] **Step 1: Create theme JSON files**

Create each JSON by copying from the opencode repo source, then trimming the `theme` section to only the 14 core keys (remove diff*, markdown*, syntax* keys). Keep the full `defs` section (unused defs are harmless).

**Source files to copy from `/home/ayan-de/Projects/githubProjects/opencode/packages/opencode/src/cli/cmd/tui/context/theme/`:**

| Target file | Source file |
|---|---|
| `internal/theme/themes/agentboard.json` | `opencode.json` — copy full file, trim theme to 14 keys |
| `internal/theme/themes/dracula.json` | `dracula.json` — copy full file, trim theme to 14 keys |
| `internal/theme/themes/gruvbox.json` | `gruvbox.json` — copy full file, trim theme to 14 keys |
| `internal/theme/themes/tokyonight.json` | `tokyonight.json` — copy full file, trim theme to 14 keys |
| `internal/theme/themes/nord.json` | `nord.json` — copy full file, trim theme to 14 keys |
| `internal/theme/themes/catppuccin.json` | `catppuccin.json` — copy full file, trim theme to 14 keys |
| `internal/theme/themes/matrix.json` | `matrix.json` — copy full file, trim theme to 14 keys |

The 14 core theme keys to keep: `primary`, `secondary`, `accent`, `error`, `warning`, `success`, `info`, `text`, `textMuted`, `background`, `backgroundPanel`, `backgroundElement`, `border`, `borderActive`.

Change `"$schema"` in each file to `"https://agentboard.dev/theme.json"`.

- [ ] **Step 2: Write the failing test for built-in themes**

Append to `internal/theme/theme_test.go`:

```go
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
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/theme/ -v -run TestRegister`
Expected: FAIL — `registerBuiltins` not defined

- [ ] **Step 4: Create builtin.go**

Create `internal/theme/builtin.go`:

```go
package theme

import "embed"

//go:embed themes/*.json
var themeFS embed.FS

func registerBuiltins(r *Registry) {
	entries, err := themeFS.ReadDir("themes")
	if err != nil {
		return
	}

	for _, entry := range entries {
		data, err := themeFS.ReadFile("themes/" + entry.Name())
		if err != nil {
			continue
		}

		th, err := parseThemeJSON(data, r.mode, "builtin")
		if err != nil {
			continue
		}

		if th.Name == "" {
			th.Name = strings.TrimSuffix(entry.Name(), ".json")
		}

		r.Register(th)
	}
}
```

Add `"strings"` to imports in `builtin.go`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/theme/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/theme/builtin.go internal/theme/themes/
git commit -m "feat(theme): add 7 built-in themes with go:embed"
```

---

## Task 4: User Theme Filesystem Loading

**Files:**
- Modify: `internal/theme/loader.go` — add `LoadUserThemes` method
- Modify: `internal/theme/theme_test.go` — add user loading tests

- [ ] **Step 1: Write the failing test**

Append to `internal/theme/theme_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/theme/ -v -run TestRegistryLoad`
Expected: FAIL — `LoadUserThemes` not defined on Registry

- [ ] **Step 3: Add LoadUserThemes to loader.go**

Add to `internal/theme/loader.go`:

```go
func (r *Registry) LoadUserThemes(dirs ...string) {
	for _, dir := range dirs {
		themes := loadFromFS(dir, "user")
		for _, th := range themes {
			if th.Name == "" {
				continue
			}
			parsed, err := parseThemeJSON(nil, r.mode, "user")
			if err != nil {
				continue
			}
			_ = parsed
			r.Register(th)
		}
	}
}
```

Wait — `loadFromFS` already handles everything including mode. But it hardcodes "dark". We need to pass mode. Let me fix: `loadFromFS` needs to accept mode, and we should update it. Actually, looking at the test, `loadFromFS` returns themes parsed with "dark" mode. The `LoadUserThemes` method should use the registry's mode. Let me update `loadFromFS` to accept mode:

Update `loadFromFS` signature to accept mode: `func loadFromFS(dir string, mode string, source string) []*Theme`. Update the `parseThemeJSON` call inside `loadFromFS` to pass `mode`. Update existing test calls to `loadFromFS` to pass `"dark"` as the mode argument.

Then add `LoadUserThemes` to `loader.go`:

```go
func (r *Registry) LoadUserThemes(dirs ...string) {
	for _, dir := range dirs {
		themes := loadFromFS(dir, r.mode, "user")
		for _, th := range themes {
			r.Register(th)
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/theme/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/theme/loader.go internal/theme/theme_test.go
git commit -m "feat(theme): add LoadUserThemes for filesystem theme loading"
```

---

## Task 5: Command & Item Structs

**Files:**
- Create: `internal/tui/command.go`
- Create: `internal/tui/command_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/command_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -v -run TestCommand`
Expected: FAIL — types not defined

- [ ] **Step 3: Create command.go**

Create `internal/tui/command.go`:

```go
package tui

import "strings"

type Item struct {
	Label       string
	Description string
	ID          string
}

type Command struct {
	Name        string
	Description string
	Prefix      string
	Items       func() []Item
}

type CommandRegistry struct {
	commands []Command
}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{}
}

func (cr *CommandRegistry) Register(cmd Command) {
	cr.commands = append(cr.commands, cmd)
}

func (cr *CommandRegistry) All() []Command {
	return cr.commands
}

func (cr *CommandRegistry) Filter(query string) []Command {
	if !strings.HasPrefix(query, "/") {
		return cr.commands
	}
	name := strings.TrimPrefix(query, "/")
	var filtered []Command
	for _, cmd := range cr.commands {
		if strings.HasPrefix(cmd.Name, name) || strings.Contains(strings.ToLower(cmd.Name), strings.ToLower(name)) {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -v -run TestCommand`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/command.go internal/tui/command_test.go
git commit -m "feat(tui): add Command and Item structs with registry"
```

---

## Task 6: CommandPalette Model

**Files:**
- Create: `internal/tui/palette.go`
- Create: `internal/tui/palette_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/palette_test.go`:

```go
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

	p.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if p.Active() {
		t.Error("palette should not be active after Esc")
	}
}

func TestPaletteFilterItems(t *testing.T) {
	p := newTestPalette()
	p.Open()

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	if len(p.filtered) == 0 {
		t.Error("filtered items should not be empty after typing /theme")
	}
}

func TestPaletteNavigation(t *testing.T) {
	p := newTestPalette()
	p.Open()

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t', 'h', 'e', 'm', 'e'}})

	if p.cursor != 0 {
		t.Errorf("cursor = %d, want 0 initially", p.cursor)
	}

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.cursor != 1 {
		t.Errorf("cursor = %d after j, want 1", p.cursor)
	}

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
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

	p := NewCommandPalette(cr, func(item Item) {
		selected = item
	})
	p.width = 120
	p.height = 40
	p.Open()

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/', 't', 'h', 'e', 'm', 'e'}})
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})

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
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/', 't', 'h', 'e', 'm', 'e'}})

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -v -run TestPalette`
Expected: FAIL — `CommandPalette` not defined

- [ ] **Step 3: Create palette.go**

Create `internal/tui/palette.go`:

```go
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
		p.input += string(msg.Runes)
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -v -run TestPalette`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/palette.go internal/tui/palette_test.go
git commit -m "feat(tui): add CommandPalette bubbletea model with upward dropdown"
```

---

## Task 7: Theme-Aware View Styles

**Files:**
- Modify: `internal/tui/kanban.go` — `DefaultKanbanStyles` → `NewKanbanStyles(t *theme.Theme)`
- Modify: `internal/tui/ticketview.go` — `DefaultTicketViewStyles` → `NewTicketViewStyles(t *theme.Theme)`
- Modify: `internal/tui/dashboard.go` — `DefaultDashboardStyles` → `NewDashboardStyles(t *theme.Theme)`

- [ ] **Step 1: Update kanban.go**

Replace `DefaultKanbanStyles()` with `NewKanbanStyles(t *theme.Theme) KanbanStyles`:

```go
func NewKanbanStyles(t *theme.Theme) KanbanStyles {
	return KanbanStyles{
		FocusedColumn: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary),
		BlurredColumn: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border),
		FocusedTitle: lipgloss.NewStyle().
			Background(t.Primary).
			Foreground(t.Background).
			Bold(true),
		BlurredTitle: lipgloss.NewStyle().
			Background(t.BackgroundPanel).
			Foreground(t.Text),
		SelectedTicket: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Text),
		Ticket: lipgloss.NewStyle().
			Foreground(t.TextMuted),
		EmptyColumn: lipgloss.NewStyle().
			Foreground(t.TextMuted),
	}
}
```

Update `NewKanbanModel` to accept `*theme.Theme`:

```go
func NewKanbanModel(s *store.Store, resolver *keybinding.Resolver, t *theme.Theme) (KanbanModel, error) {
	m := KanbanModel{
		store:    s,
		resolver: resolver,
		styles:   NewKanbanStyles(t),
	}
	m, err := m.loadColumns()
	if err != nil {
		return m, fmt.Errorf("kanban.newKanbanModel: %w", err)
	}
	return m, nil
}
```

Add import: `"github.com/ayan-de/agent-board/internal/theme"`

Keep `DefaultKanbanStyles()` as a wrapper for backward compat in tests:

```go
func DefaultKanbanStyles() KanbanStyles {
	return NewKanbanStyles(&theme.Theme{
		Primary: lipgloss.Color("69"), Text: lipgloss.Color("15"),
		TextMuted: lipgloss.Color("240"), Background: lipgloss.Color("#000"),
		BackgroundPanel: lipgloss.Color("236"), Border: lipgloss.Color("240"),
	})
}
```

- [ ] **Step 2: Update ticketview.go**

Replace `DefaultTicketViewStyles()` with `NewTicketViewStyles(t *theme.Theme)`:

```go
func NewTicketViewStyles(t *theme.Theme) TicketViewStyles {
	return TicketViewStyles{
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary),
		Label: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Text),
		Value: lipgloss.NewStyle().
			Foreground(t.Text),
		SelectedRow: lipgloss.NewStyle().
			Background(t.Primary).
			Foreground(t.Background),
		Cursor: lipgloss.NewStyle().
			Foreground(t.Primary),
		EditBox: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(t.Accent),
		Footer: lipgloss.NewStyle().
			Foreground(t.TextMuted),
		Empty: lipgloss.NewStyle().
			Foreground(t.TextMuted),
	}
}
```

Update `NewTicketViewModel` to accept `*theme.Theme`:

```go
func NewTicketViewModel(s *store.Store, resolver *keybinding.Resolver, t *theme.Theme) TicketViewModel {
	return TicketViewModel{
		store:    s,
		resolver: resolver,
		styles:   NewTicketViewStyles(t),
		fields:   ticketFields(),
		mode:     ticketViewMode,
	}
}
```

Add import: `"github.com/ayan-de/agent-board/internal/theme"`

- [ ] **Step 3: Update dashboard.go**

Replace `DefaultDashboardStyles()` with `NewDashboardStyles(t *theme.Theme)`:

```go
func NewDashboardStyles(t *theme.Theme) DashboardStyles {
	return DashboardStyles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary),
		CardFound: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Success).
			Padding(0, 1).
			Width(30),
		CardMissing: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(0, 1).
			Width(30),
		Label: lipgloss.NewStyle().
			Foreground(t.Text),
		Value: lipgloss.NewStyle().
			Foreground(t.Text),
		Placeholder: lipgloss.NewStyle().
			Foreground(t.TextMuted),
		Footer: lipgloss.NewStyle().
			Foreground(t.TextMuted),
	}
}
```

Update `NewDashboardModel` to accept `*theme.Theme`:

```go
func NewDashboardModel(s *store.Store, resolver *keybinding.Resolver, agents []config.DetectedAgent, t *theme.Theme) DashboardModel {
	return DashboardModel{
		store:    s,
		resolver: resolver,
		agents:   agents,
		styles:   NewDashboardStyles(t),
	}
}
```

Add import: `"github.com/ayan-de/agent-board/internal/theme"`

- [ ] **Step 4: Fix all tests that call the old constructors**

Update `internal/tui/kanban_test.go`: in `newTestKanban`, pass a default theme:

```go
func newTestKanban(t *testing.T) KanbanModel {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.Open(dbPath, []string{"backlog", "in_progress", "review", "done"})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	km := keybinding.DefaultKeyMap()
	resolver := keybinding.NewResolver(km)

	defaultTheme := &theme.Theme{
		Primary: lipgloss.Color("69"), Text: lipgloss.Color("15"),
		TextMuted: lipgloss.Color("240"), Background: lipgloss.Color("#000"),
		BackgroundPanel: lipgloss.Color("236"), Border: lipgloss.Color("240"),
		Success: lipgloss.Color("42"), Accent: lipgloss.Color("213"),
	}

	model, err := NewKanbanModel(s, resolver, defaultTheme)
	if err != nil {
		t.Fatalf("new kanban model: %v", err)
	}
	return model
}
```

Add import: `"github.com/ayan-de/agent-board/internal/theme"`

Similarly update `app_test.go`'s `newTestApp` — but that's handled in Task 8 when we wire the registry.

For now, add a helper `testTheme()` in the test files:

```go
func testTheme() *theme.Theme {
	return &theme.Theme{
		Primary: lipgloss.Color("69"), Text: lipgloss.Color("15"),
		TextMuted: lipgloss.Color("240"), Background: lipgloss.Color("#000"),
		BackgroundPanel: lipgloss.Color("236"), Border: lipgloss.Color("240"),
		Success: lipgloss.Color("42"), Accent: lipgloss.Color("213"),
	}
}
```

Use this in `newTestKanban`, `newTestApp` (update to pass `testTheme()` to all constructors).

- [ ] **Step 5: Run all tests**

Run: `go test ./internal/tui/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/tui/kanban.go internal/tui/ticketview.go internal/tui/dashboard.go internal/tui/kanban_test.go internal/tui/app_test.go
git commit -m "feat(tui): convert hardcoded styles to theme-aware constructors"
```

---

## Task 8: Wire Theme Registry & Palette into App

**Files:**
- Modify: `internal/tui/app.go` — add registry, palette, applyTheme, palette routing
- Modify: `internal/tui/app_test.go` — update tests for new signature
- Modify: `internal/keybinding/action.go` — add `ActionOpenPalette`
- Modify: `internal/keybinding/keymap.go` — add `:` binding
- Modify: `internal/config/defaults.go` — change default theme to "agentboard"
- Modify: `cmd/agentboard/main.go` — wire registry

- [ ] **Step 1: Add ActionOpenPalette to keybinding**

In `internal/keybinding/action.go`, add after `ActionShowDashboard`:

```go
ActionOpenPalette
```

Add to `String()` method:

```go
case ActionOpenPalette:
	return "open_palette"
```

In `internal/keybinding/keymap.go`, add to `DefaultKeyMap()` bindings:

```go
{Key: ":", Action: ActionOpenPalette},
```

- [ ] **Step 2: Update config default**

In `internal/config/defaults.go`, change `Theme: "default"` to `Theme: "agentboard"`.

- [ ] **Step 3: Update app.go**

Add `theme` import and fields:

```go
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
```

Update `NewApp`:

```go
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

	paletteTheme := t
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
		palette:    NewCommandPalette(cr, func(item Item) {}),
	}

	a.palette.SetTheme(paletteTheme)
	a.palette.onSelect = func(item Item) {
		a.registry.Set(item.ID)
		a.applyTheme()
	}

	return a, nil
}
```

Add `applyTheme` method:

```go
func (a *App) applyTheme() {
	t := a.registry.Active()
	a.kanban.styles = NewKanbanStyles(t)
	a.ticketView.styles = NewTicketViewStyles(t)
	a.dashboard.styles = NewDashboardStyles(t)
	a.palette.SetTheme(t)
}
```

Update `handleKey` to intercept palette keys:

```go
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
```

Update `View` to account for palette:

```go
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
		return mainView + "\n" + paletteView
	}
	return mainView
}
```

Update `renderHelp` to use theme:

```go
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
```

- [ ] **Step 4: Update main.go**

```go
package main

import (
	"fmt"
	"os"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"
	"github.com/ayan-de/agent-board/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	s, err := store.Open(cfg.DB.Path, cfg.Board.Statuses)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening store: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	reg := theme.NewRegistry("dark")
	reg.LoadUserThemes()
	if err := reg.Set(cfg.TUI.Theme); err != nil {
		_ = reg.Set("agentboard")
	}

	app, err := tui.NewApp(cfg, s, reg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating app: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running tui: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Update app_test.go**

Update `newTestApp`:

```go
func newTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.Open(dbPath, []string{"backlog", "in_progress", "review", "done"})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	cfg := config.SetDefaults()
	reg := theme.NewRegistry("dark")
	reg.Register(&theme.Theme{
		Name:   "agentboard",
		Source: "builtin",
		Primary: lipgloss.Color("69"), Text: lipgloss.Color("15"),
		TextMuted: lipgloss.Color("240"), Background: lipgloss.Color("#000"),
		BackgroundPanel: lipgloss.Color("236"), Border: lipgloss.Color("240"),
		Success: lipgloss.Color("42"), Accent: lipgloss.Color("213"),
	})

	app, err := NewApp(cfg, s, reg)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return app
}
```

Add imports: `"github.com/ayan-de/agent-board/internal/theme"`, `"github.com/charmbracelet/lipgloss"`.

- [ ] **Step 6: Run all tests**

Run: `go test ./... -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/tui/app.go internal/tui/app_test.go internal/keybinding/action.go internal/keybinding/keymap.go internal/config/defaults.go cmd/agentboard/main.go
git commit -m "feat: wire theme registry and command palette into App"
```

---

## Task 9: Final Verification

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: All tests PASS

- [ ] **Step 2: Run go vet**

Run: `go vet ./...`
Expected: No issues

- [ ] **Step 3: Build the binary**

Run: `go build -o agentboard ./cmd/agentboard`
Expected: Build succeeds

- [ ] **Step 4: Final commit (if any fixes needed)**

```bash
git add -A
git commit -m "fix: address final test and build issues"
```
