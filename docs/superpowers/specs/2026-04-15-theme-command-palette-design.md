# Phase 2.1: Theme System & Command Palette

**Date**: 2026-04-15
**Status**: Approved
**Approach**: Registry Pattern (Approach A)

---

## Overview

Add a scalable theme system and an extensible command palette to AgentBoard's TUI. Users can switch between built-in themes via a command palette, live-preview themes, and add their own custom themes as JSON files. The command palette is designed to support future commands beyond themes.

---

## 1. Theme Data Model

### Theme struct (14 core colors)

```go
type Theme struct {
    Name    string
    Source  string // "builtin", "user", "project"

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
```

### Theme JSON format (opencode-compatible)

Users create a file like `~/.config/agentboard/themes/mytheme.json`:

```json
{
  "name": "My Custom Theme",
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
}
```

- `defs` — optional reusable color aliases (referenced by name in theme values)
- `name` — displayed in the command palette; if omitted, filename stem is used
- `theme` — the 14 color values, each can be:
  - Hex string: `"#7aa2f7"`
  - Def reference: `"fg"` (resolved from `defs`)
  - Dark/light variant: `{ "dark": "#c0caf5", "light": "#1a1b26" }`

### Built-in themes (7, embedded via `go:embed`)

| Theme Name | Source | Notes |
|---|---|---|
| agentboard | Default | AgentBoard's identity theme (ported from opencode default) |
| dracula | opencode | Dracula dark theme |
| gruvbox | opencode | Gruvbox color scheme |
| tokyonight | opencode | Tokyo Night theme |
| nord | opencode | Nord polar color palette |
| catppuccin | opencode | Catppuccin Mocha |
| matrix | opencode | Matrix green-on-black |

Files: `internal/theme/themes/*.json`

### Theme loading priority (later overrides earlier)

1. Built-in embedded themes (`internal/theme/themes/*.json`)
2. `~/.config/agentboard/themes/*.json` (user-wide)
3. `.agentboard/themes/*.json` (project-local, walking up from CWD)

User/project themes with the same name as built-in themes override them.

---

## 2. Theme Package Architecture

```
internal/theme/
  theme.go       — Theme struct, Registry struct, NewRegistry(), Active(), Set(), All()
  loader.go      — loadFromFS(), parseThemeJSON(), resolveColor() — defs resolution, dark/light variants
  builtin.go     — go:embed of themes/*.json, registerBuiltins()
  themes/
    agentboard.json
    dracula.json
    gruvbox.json
    tokyonight.json
    nord.json
    catppuccin.json
    matrix.json
```

### Registry API

```go
type Registry struct {
    themes   map[string]*Theme
    active   string
    mode     string // "dark" or "light"
}

func NewRegistry(mode string) *Registry
func (r *Registry) Register(t *Theme)           // add or override a theme
func (r *Registry) Active() *Theme              // return current active theme
func (r *Registry) Set(name string) error        // switch active theme by name
func (r *Registry) All() []*Theme               // return all themes sorted by name
func (r *Registry) LoadUserThemes()             // scan ~/.config/agentboard/themes/ and .agentboard/themes/
```

The Registry holds the canonical set of themes. Views never load themes directly — they receive a `*Theme` from the registry.

---

## 3. Command Palette

### Architecture

```
internal/tui/
  palette.go     — CommandPalette model (bubbletea.Model)
  command.go     — Command struct, built-in command registry
```

### Trigger and UI

- Press `:` to open the palette
- Palette appears at the **bottom** of the screen as a 1-line input bar
- A **dropdown opens upward** above the input bar showing filtered results
- Navigate with `j/k`, `Enter` confirms, `Esc` cancels

### Visual layout

```
┌─────────────────────────────────────────────────────────────────┐
│  ┌─────────┬──────────┬──────────┬──────────┐                  │
│  │ Backlog │  In Prog │  Review  │   Done   │                  │
│  │ ─────── │ ──────── │ ──────── │ ──────── │                  │
│  │ AUTH-1  │ API-3    │ UI-7     │ INIT-1   │                  │
│  │ DB-2    │          │          │ INIT-2   │                  │
│  └─────────┴──────────┴──────────┴──────────┘                  │
│                                                                 │
│  ▸ agentboard    AgentBoard default theme                       │
│    dracula       Dracula dark theme                             │
│    gruvbox       Gruvbox color scheme                           │
│    matrix        Matrix green-on-black                          │
│    nord          Nord polar color palette                       │
│    tokyonight    Tokyo Night theme                              │
│─────────────────────────────────────────────────────────────────│
│ : /theme                                                        │
└─────────────────────────────────────────────────────────────────┘
```

The main view shrinks by the dropdown + input bar height so nothing overlaps.

### Command struct (extensible)

```go
type Command struct {
    Name        string
    Description string
    Category    string
    Prefix      string            // e.g. "/" — commands are "/theme", "/help"
    Execute     func(arg string)
    Items       func() []Item
}

type Item struct {
    Label       string
    Description string
    ID          string
}
```

Future commands register themselves using the same struct: `/theme`, `/keybindings`, `/agents`, `/layout`, etc.

### Palette model

```go
type CommandPalette struct {
    commands    []Command
    input       string
    filtered    []Item
    cursor      int
    active      bool
    onSelect    func(Item)
    maxHeight   int               // max dropdown items visible (default 8)
}
```

- Filtering: typed text matched against `Item.Label` (case-insensitive prefix match)
- Live preview: on cursor move, the onSelect callback fires immediately
- No z-layering: `App` subtracts palette height from main view's available height

### Theme switching key flow

1. User presses `:` → palette opens, input empty
2. User types `/theme` → filtered to themes list, all themes shown
3. User presses `j/k` → cursor moves, theme live-previews via `onSelect` callback
4. User presses `Enter` → theme confirmed, palette closes, theme persisted to config
5. User presses `Esc` → palette closes, theme reverted to pre-palette value

---

## 4. View Integration & Style System

### Replacing hardcoded styles

Each view's `Default*Styles()` function becomes a constructor accepting `*theme.Theme`:

```go
// Before
func DefaultKanbanStyles() KanbanStyles { ... }

// After
func NewKanbanStyles(t *theme.Theme) KanbanStyles { ... }
func NewTicketViewStyles(t *theme.Theme) TicketViewStyles { ... }
func NewDashboardStyles(t *theme.Theme) DashboardStyles { ... }
```

### Style mapping (old → new)

| Old hardcoded color | Theme variable | Usage |
|---|---|---|
| `"69"` (purple-blue) | `Primary` | Focused borders, titles, cursors |
| `"15"` (white) | `Text` | Foreground on colored backgrounds |
| `"240"` (gray) | `TextMuted` | Muted borders, empty text, footers |
| `"236"` (dark gray) | `BackgroundPanel` | Blurred title background |
| `"252"` (light gray) | `Text` | General text, labels, values |
| `"42"` (green) | `Success` | Dashboard card borders |
| `"213"` (pink) | `Accent` | Edit box border |

### Theme change flow

1. User selects theme in palette → `palette.onSelect(item)` fires
2. `App` calls `themeRegistry.Set(item.ID)`
3. `App` calls `applyTheme()` which re-creates all sub-model styles:
   - `kanban.styles = NewKanbanStyles(registry.Active())`
   - `ticketView.styles = NewTicketViewStyles(registry.Active())`
   - `dashboard.styles = NewDashboardStyles(registry.Active())`
4. Bubbletea re-renders — everything reflects new colors immediately

### Persistence

Selected theme name saved to `~/.agentboard/config.toml` under `[tui]` theme key. On startup, `config.TUI.Theme` is read and `registry.Set(cfg.TUI.Theme)` called before rendering.

---

## 5. Package Dependency & Wiring

### New packages

- `internal/theme/` — zero TUI dependency. Only imports `lipgloss` for `lipgloss.Color`.

### Updated packages

| Package | Change |
|---|---|
| `internal/tui/app.go` | Hold `*theme.Registry`, pass theme to sub-models, wire palette |
| `internal/tui/kanban.go` | Accept `*theme.Theme` for initial styles |
| `internal/tui/ticketview.go` | Same pattern |
| `internal/tui/dashboard.go` | Same pattern |
| `internal/tui/palette.go` | New — CommandPalette bubbletea model |
| `internal/tui/command.go` | New — Command struct and registry |
| `internal/config/config.go` | `TUI.Theme` field already exists, now consumed on startup |
| `cmd/agentboard/main.go` | Create `theme.NewRegistry()`, load builtins + user themes, pass to TUI |

### Dependency graph (no cycles)

```
cmd/agentboard
  ├── internal/config
  ├── internal/store
  ├── internal/theme   ← new
  └── internal/tui
        ├── internal/theme
        ├── internal/store
        └── internal/keybinding
```

### Config TOML

```toml
[tui]
theme = "agentboard"   # default, user can change via palette
```

No new env vars — theme is set via command palette and persisted to TOML.

---

## 6. Keybindings Affected

| Key | Action | Change |
|-----|--------|--------|
| `:` | Open command palette | **New** — triggers `CommandPalette` |
| `Esc` | Close palette / revert | **New** — closes palette, reverts theme if previewing |
| `Enter` | Confirm selection | **New** — confirms theme/command, persists, closes palette |
| `j/k` | Navigate dropdown | **Extended** — works in palette dropdown when palette is active |

The `:` key does not conflict with any existing binding.

---

## 7. Testing Strategy

| Package | Tests |
|---|---|
| `internal/theme` | Table-driven tests for: JSON parsing, defs resolution, dark/light variant selection, override priority, filesystem loading, missing color defaults |
| `internal/tui/palette.go` | Tests for: open/close, filtering, cursor navigation, selection callback, Esc revert |
| `internal/tui/command.go` | Tests for: command registration, item listing, prefix matching |
| Existing view tests | Updated: `New*Styles(theme)` constructors tested with mock themes |

---

## 8. File List Summary

### New files

| File | Purpose |
|---|---|
| `internal/theme/theme.go` | Theme struct, Registry struct, methods |
| `internal/theme/loader.go` | JSON parsing, defs resolution, filesystem scanning |
| `internal/theme/builtin.go` | go:embed directive, built-in registration |
| `internal/theme/theme_test.go` | Theme package tests |
| `internal/theme/themes/agentboard.json` | Default theme |
| `internal/theme/themes/dracula.json` | Dracula theme |
| `internal/theme/themes/gruvbox.json` | Gruvbox theme |
| `internal/theme/themes/tokyonight.json` | Tokyo Night theme |
| `internal/theme/themes/nord.json` | Nord theme |
| `internal/theme/themes/catppuccin.json` | Catppuccin theme |
| `internal/theme/themes/matrix.json` | Matrix theme |
| `internal/tui/palette.go` | CommandPalette bubbletea model |
| `internal/tui/palette_test.go` | Palette tests |
| `internal/tui/command.go` | Command struct and registry |
| `internal/tui/command_test.go` | Command tests |

### Modified files

| File | Change |
|---|---|
| `cmd/agentboard/main.go` | Wire theme registry |
| `internal/tui/app.go` | Hold registry, palette, style rebuild on theme change |
| `internal/tui/kanban.go` | `DefaultKanbanStyles` → `NewKanbanStyles(t *theme.Theme)` |
| `internal/tui/ticketview.go` | `DefaultTicketViewStyles` → `NewTicketViewStyles(t *theme.Theme)` |
| `internal/tui/dashboard.go` | `DefaultDashboardStyles` → `NewDashboardStyles(t *theme.Theme)` |
| `internal/config/config.go` | Consume `TUI.Theme` on startup |
