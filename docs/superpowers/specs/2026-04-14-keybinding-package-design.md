# Keybinding Package Design

## Overview

A centralized keybinding system for AgentBoard. Defines user intents as `Action` constants, maps keys to actions via a `KeyMap` struct, resolves key sequences (including chords like `g3`) through a stateful `Resolver`, and allows user customization via TOML config overrides.

**Scope**: Pure logic package — no bubbletea dependency. TUI integration happens in steps 1.4/1.5.

## Architecture

### Action Enum

Every user intent is an `Action` constant. Actions are UI-context-independent — the TUI layer decides what to do with them based on which view is active.

```go
type Action int

const (
    ActionNone Action = iota
    ActionQuit
    ActionForceQuit
    ActionPrevColumn
    ActionNextColumn
    ActionPrevTicket
    ActionNextTicket
    ActionOpenTicket
    ActionAddTicket
    ActionDeleteTicket
    ActionStartAgent
    ActionStopAgent
    ActionRefresh
    ActionToggleFocus
    ActionPrevFocus
    ActionJumpColumn1
    ActionJumpColumn2
    ActionJumpColumn3
    ActionJumpColumn4
    ActionShowHelp
    ActionGoToTicket
)
```

`ActionGoToTicket` is a chord action — the Resolver tracks the `g` prefix, accumulates digits, then emits `ActionGoToTicket` with a numeric argument.

### Binding & KeyMap

```go
type Binding struct {
    Key     string
    Action  Action
    IsChord bool
}

type KeyMap struct {
    Bindings []Binding
}
```

### Default Key Bindings

| Key | Action | Chord? |
|-----|--------|--------|
| `q` | ActionQuit | no |
| `ctrl+c` | ActionForceQuit | no |
| `h` | ActionPrevColumn | no |
| `left` | ActionPrevColumn | no |
| `l` | ActionNextColumn | no |
| `right` | ActionNextColumn | no |
| `j` | ActionPrevTicket | no |
| `down` | ActionPrevTicket | no |
| `k` | ActionNextTicket | no |
| `up` | ActionNextTicket | no |
| `enter` | ActionOpenTicket | no |
| `a` | ActionAddTicket | no |
| `d` | ActionDeleteTicket | no |
| `s` | ActionStartAgent | no |
| `x` | ActionStopAgent | no |
| `r` | ActionRefresh | no |
| `tab` | ActionToggleFocus | no |
| `shift+tab` | ActionPrevFocus | no |
| `1` | ActionJumpColumn1 | no |
| `2` | ActionJumpColumn2 | no |
| `3` | ActionJumpColumn3 | no |
| `4` | ActionJumpColumn4 | no |
| `?` | ActionShowHelp | no |
| `g` | ActionGoToTicket | yes (prefix) |

Multiple keys can map to the same action (e.g., `h` and `left` both map to `ActionPrevColumn`).

### Resolver

```go
type Resolver struct {
    keyMap      KeyMap
    chordMode   bool
    chordPrefix string
    digitBuf    strings.Builder
}

func (r *Resolver) Resolve(key string) (Action, int)
```

**Logic**:

1. If in chord mode and key is a digit (`0-9`) → append to digitBuf, return `(ActionNone, 0)`.
2. If in chord mode and key is NOT a digit:
   - If digitBuf has content → emit `(ActionGoToTicket, parseInt(digitBuf))`, exit chord mode.
   - If digitBuf is empty → cancel chord mode, process key normally (step 3).
3. If not in chord mode → scan `KeyMap.Bindings` for matching key:
   - If found and `IsChord=true` → enter chord mode, set prefix, return `(ActionNone, 0)`.
   - If found and `IsChord=false` → return `(binding.Action, 0)`.
   - If not found → return `(ActionNone, 0)`.

**Reset**: `Resolver.Reset()` clears chord state. Called on view transitions or cancellations.

### Key Naming Convention

Keys use normalized string format:

- Letters: `a`-`z`
- Special keys: `enter`, `tab`, `escape`, `space`, `backspace`, `delete`, `up`, `down`, `left`, `right`, `home`, `end`, `pageup`, `pagedown`
- Modifiers: `ctrl+a`, `ctrl+c`, `shift+tab`
- Printable chars: `?`, `/`, `:`

### Config Integration

Existing `TUIConfig` gets a `Keybindings` field:

```go
type TUIConfig struct {
    Theme       string            `toml:"theme"`
    Layout      string            `toml:"layout"`
    Keybindings map[string]string `toml:"keybindings"`
}
```

Example TOML:

```toml
[tui]
theme = "default"
layout = "compact"

[tui.keybindings]
next_column = "l"
prev_column = "h"
next_ticket = "j"
prev_ticket = "k"
open_ticket = "enter"
add_ticket = "a"
delete_ticket = "d"
start_agent = "s"
stop_agent = "x"
refresh = "r"
show_help = "?"
toggle_focus = "tab"
prev_focus = "shift+tab"
jump_col1 = "1"
jump_col2 = "2"
jump_col3 = "3"
jump_col4 = "4"
go_to_ticket_prefix = "g"
```

**Config action name → Action mapping**:

```go
var actionNames = map[string]Action{
    "quit":                 ActionQuit,
    "force_quit":           ActionForceQuit,
    "prev_column":          ActionPrevColumn,
    "next_column":          ActionNextColumn,
    "prev_ticket":          ActionPrevTicket,
    "next_ticket":          ActionNextTicket,
    "open_ticket":          ActionOpenTicket,
    "add_ticket":           ActionAddTicket,
    "delete_ticket":        ActionDeleteTicket,
    "start_agent":          ActionStartAgent,
    "stop_agent":           ActionStopAgent,
    "refresh":              ActionRefresh,
    "toggle_focus":         ActionToggleFocus,
    "prev_focus":           ActionPrevFocus,
    "jump_col1":            ActionJumpColumn1,
    "jump_col2":            ActionJumpColumn2,
    "jump_col3":            ActionJumpColumn3,
    "jump_col4":            ActionJumpColumn4,
    "show_help":            ActionShowHelp,
    "go_to_ticket_prefix":  ActionGoToTicket,
}
```

**ApplyConfig**: Takes `map[string]string`, merges onto `DefaultKeyMap()`:
- For each config entry, look up action name in `actionNames`.
- If found, replace ALL bindings for that action with the new key.
- If not found, skip (silently — user may have typos, don't crash).
- `go_to_ticket_prefix` sets the chord prefix key for `ActionGoToTicket`.

## File Layout

```
internal/keybinding/
    action.go         — Action enum, String(), actionNames map
    keymap.go         — Binding, KeyMap, DefaultKeyMap()
    resolver.go       — Resolver struct, Resolve(), Reset()
    config.go         — ApplyConfig() to merge TOML overrides into KeyMap
    action_test.go    — Action String() tests
    keymap_test.go    — DefaultKeyMap coverage, no duplicate keys
    resolver_test.go  — Table-driven: key sequences → (Action, arg)
    config_test.go    — TOML override merging, unknown keys, empty config
```

## Testing Strategy

All pure logic, no TUI framework dependency.

### Resolver Tests (table-driven)

| Input sequence | Expected result |
|---|---|
| `"j"` | `(ActionNextTicket, 0)` |
| `"g", "3"` | `(ActionNone, 0)` then `(ActionGoToTicket, 3)` |
| `"g", "1", "2"` | `(ActionNone, 0)` then `(ActionNone, 0)` then `(ActionGoToTicket, 12)` |
| `"g", "q"` | `(ActionNone, 0)` then `(ActionQuit, 0)` — chord cancelled |
| `"g", "g"` | `(ActionNone, 0)` then `(ActionNone, 0)` — chord cancelled, `g` in chord starts new chord |
| Unknown key `"z"` | `(ActionNone, 0)` |

### KeyMap Tests

- `DefaultKeyMap()` contains all expected bindings (one test per action).
- No duplicate keys (each key appears at most once).
- `DefaultKeyMap()` is safe to mutate without affecting future calls (returns fresh copy).

### Config Tests

- Override single action: `{"next_column": "L"}` replaces `l` with `L`.
- Unknown action name silently ignored.
- Empty config map returns unmodified KeyMap.
- `go_to_ticket_prefix` override changes the chord trigger key.
- Override replaces all bindings for an action (e.g., `h` and `left` both for `ActionPrevColumn`).

### Action Tests

- `Action.String()` returns expected name for each constant.
- `ActionNone.String()` returns `"none"`.

## Scalability

- **New actions**: Add const to `Action` enum, entry to `actionNames`, binding to `DefaultKeyMap()`.
- **New chord prefixes**: Add `Binding` with `IsChord=true`, extend Resolver with prefix-specific handlers.
- **Context-scoped keymaps**: Future step — wrap `KeyMap` per UI context (kanban, ticket detail, agent pane). Current flat design doesn't block this.
- **Multiple bindings per action**: `KeyMap.Bindings` is a slice — same action can have multiple keys.
