# Keybinding Package Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a pure-logic keybinding package that defines Action constants, maps keys to actions, resolves chords, and supports TOML config overrides.

**Architecture:** Flat Action enum + KeyMap struct + stateful Resolver for chord handling. Config overrides merge onto defaults via a `map[string]string` from TOML. No TUI dependency.

**Tech Stack:** Go standard library only (`fmt`, `strings`, `strconv`). No external dependencies.

---

### Task 1: Action enum and String() method

**Files:**
- Create: `internal/keybinding/action.go`
- Test: `internal/keybinding/action_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/keybinding/action_test.go`:

```go
package keybinding

import "testing"

func TestActionString(t *testing.T) {
	tests := []struct {
		action Action
		want   string
	}{
		{ActionNone, "none"},
		{ActionQuit, "quit"},
		{ActionForceQuit, "force_quit"},
		{ActionPrevColumn, "prev_column"},
		{ActionNextColumn, "next_column"},
		{ActionPrevTicket, "prev_ticket"},
		{ActionNextTicket, "next_ticket"},
		{ActionOpenTicket, "open_ticket"},
		{ActionAddTicket, "add_ticket"},
		{ActionDeleteTicket, "delete_ticket"},
		{ActionStartAgent, "start_agent"},
		{ActionStopAgent, "stop_agent"},
		{ActionRefresh, "refresh"},
		{ActionToggleFocus, "toggle_focus"},
		{ActionPrevFocus, "prev_focus"},
		{ActionJumpColumn1, "jump_col1"},
		{ActionJumpColumn2, "jump_col2"},
		{ActionJumpColumn3, "jump_col3"},
		{ActionJumpColumn4, "jump_col4"},
		{ActionShowHelp, "show_help"},
		{ActionGoToTicket, "go_to_ticket"},
		{Action(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.action.String()
			if got != tt.want {
				t.Errorf("Action(%d).String() = %q, want %q", tt.action, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/keybinding/... -v -run TestActionString`
Expected: FAIL — package does not exist or Action type not defined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/keybinding/action.go`:

```go
package keybinding

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

func (a Action) String() string {
	switch a {
	case ActionNone:
		return "none"
	case ActionQuit:
		return "quit"
	case ActionForceQuit:
		return "force_quit"
	case ActionPrevColumn:
		return "prev_column"
	case ActionNextColumn:
		return "next_column"
	case ActionPrevTicket:
		return "prev_ticket"
	case ActionNextTicket:
		return "next_ticket"
	case ActionOpenTicket:
		return "open_ticket"
	case ActionAddTicket:
		return "add_ticket"
	case ActionDeleteTicket:
		return "delete_ticket"
	case ActionStartAgent:
		return "start_agent"
	case ActionStopAgent:
		return "stop_agent"
	case ActionRefresh:
		return "refresh"
	case ActionToggleFocus:
		return "toggle_focus"
	case ActionPrevFocus:
		return "prev_focus"
	case ActionJumpColumn1:
		return "jump_col1"
	case ActionJumpColumn2:
		return "jump_col2"
	case ActionJumpColumn3:
		return "jump_col3"
	case ActionJumpColumn4:
		return "jump_col4"
	case ActionShowHelp:
		return "show_help"
	case ActionGoToTicket:
		return "go_to_ticket"
	default:
		return "unknown"
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/keybinding/... -v -run TestActionString`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/keybinding/action.go internal/keybinding/action_test.go
git commit -m "feat(keybinding): add Action enum and String method"
```

---

### Task 2: Binding, KeyMap, and DefaultKeyMap

**Files:**
- Create: `internal/keybinding/keymap.go`
- Test: `internal/keybinding/keymap_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/keybinding/keymap_test.go`:

```go
package keybinding

import "testing"

func TestDefaultKeyMapContainsAllActions(t *testing.T) {
	km := DefaultKeyMap()

	actionsSeen := map[Action]bool{}
	for _, b := range km.Bindings {
		actionsSeen[b.Action] = true
	}

	requiredActions := []Action{
		ActionQuit,
		ActionForceQuit,
		ActionPrevColumn,
		ActionNextColumn,
		ActionPrevTicket,
		ActionNextTicket,
		ActionOpenTicket,
		ActionAddTicket,
		ActionDeleteTicket,
		ActionStartAgent,
		ActionStopAgent,
		ActionRefresh,
		ActionToggleFocus,
		ActionPrevFocus,
		ActionJumpColumn1,
		ActionJumpColumn2,
		ActionJumpColumn3,
		ActionJumpColumn4,
		ActionShowHelp,
		ActionGoToTicket,
	}

	for _, a := range requiredActions {
		if !actionsSeen[a] {
			t.Errorf("DefaultKeyMap missing binding for action %s", a)
		}
	}
}

func TestDefaultKeyMapNoDuplicateKeys(t *testing.T) {
	km := DefaultKeyMap()
	seen := map[string]bool{}

	for _, b := range km.Bindings {
		if seen[b.Key] {
			t.Errorf("duplicate key %q in DefaultKeyMap", b.Key)
		}
		seen[b.Key] = true
	}
}

func TestDefaultKeyMapReturnsFreshCopy(t *testing.T) {
	km1 := DefaultKeyMap()
	km2 := DefaultKeyMap()

	km1.Bindings[0].Key = "MUTATED"

	if km2.Bindings[0].Key == "MUTATED" {
		t.Error("DefaultKeyMap returns shared reference, mutations leak across calls")
	}
}

func TestDefaultKeyMapExpectedBindings(t *testing.T) {
	km := DefaultKeyMap()

	tests := []struct {
		key    string
		action Action
		chord  bool
	}{
		{"q", ActionQuit, false},
		{"ctrl+c", ActionForceQuit, false},
		{"h", ActionPrevColumn, false},
		{"left", ActionPrevColumn, false},
		{"l", ActionNextColumn, false},
		{"right", ActionNextColumn, false},
		{"j", ActionPrevTicket, false},
		{"down", ActionPrevTicket, false},
		{"k", ActionNextTicket, false},
		{"up", ActionNextTicket, false},
		{"enter", ActionOpenTicket, false},
		{"a", ActionAddTicket, false},
		{"d", ActionDeleteTicket, false},
		{"s", ActionStartAgent, false},
		{"x", ActionStopAgent, false},
		{"r", ActionRefresh, false},
		{"tab", ActionToggleFocus, false},
		{"shift+tab", ActionPrevFocus, false},
		{"1", ActionJumpColumn1, false},
		{"2", ActionJumpColumn2, false},
		{"3", ActionJumpColumn3, false},
		{"4", ActionJumpColumn4, false},
		{"?", ActionShowHelp, false},
		{"g", ActionGoToTicket, true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			found := false
			for _, b := range km.Bindings {
				if b.Key == tt.key {
					found = true
					if b.Action != tt.action {
						t.Errorf("key %q maps to action %s, want %s", tt.key, b.Action, tt.action)
					}
					if b.IsChord != tt.chord {
						t.Errorf("key %q IsChord=%v, want %v", tt.key, b.IsChord, tt.chord)
					}
					break
				}
			}
			if !found {
				t.Errorf("key %q not found in DefaultKeyMap", tt.key)
			}
		})
	}
}

func TestKeyMapLookup(t *testing.T) {
	km := DefaultKeyMap()

	action, ok := km.Lookup("j")
	if !ok || action.Action != ActionPrevTicket {
		t.Errorf("Lookup(%q) = (%v, %v), want (ActionPrevTicket binding, true)", "j", action, ok)
	}

	_, ok = km.Lookup("zzz")
	if ok {
		t.Error("Lookup(unknown key) should return false")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/keybinding/... -v -run TestDefault`
Expected: FAIL — `DefaultKeyMap` not defined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/keybinding/keymap.go`:

```go
package keybinding

type Binding struct {
	Key     string
	Action  Action
	IsChord bool
}

type KeyMap struct {
	Bindings []Binding
}

func (km *KeyMap) Lookup(key string) (Binding, bool) {
	for _, b := range km.Bindings {
		if b.Key == key {
			return b, true
		}
	}
	return Binding{}, false
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Bindings: []Binding{
			{Key: "q", Action: ActionQuit},
			{Key: "ctrl+c", Action: ActionForceQuit},
			{Key: "h", Action: ActionPrevColumn},
			{Key: "left", Action: ActionPrevColumn},
			{Key: "l", Action: ActionNextColumn},
			{Key: "right", Action: ActionNextColumn},
			{Key: "j", Action: ActionPrevTicket},
			{Key: "down", Action: ActionPrevTicket},
			{Key: "k", Action: ActionNextTicket},
			{Key: "up", Action: ActionNextTicket},
			{Key: "enter", Action: ActionOpenTicket},
			{Key: "a", Action: ActionAddTicket},
			{Key: "d", Action: ActionDeleteTicket},
			{Key: "s", Action: ActionStartAgent},
			{Key: "x", Action: ActionStopAgent},
			{Key: "r", Action: ActionRefresh},
			{Key: "tab", Action: ActionToggleFocus},
			{Key: "shift+tab", Action: ActionPrevFocus},
			{Key: "1", Action: ActionJumpColumn1},
			{Key: "2", Action: ActionJumpColumn2},
			{Key: "3", Action: ActionJumpColumn3},
			{Key: "4", Action: ActionJumpColumn4},
			{Key: "?", Action: ActionShowHelp},
			{Key: "g", Action: ActionGoToTicket, IsChord: true},
		},
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/keybinding/... -v -run TestDefault`
Expected: PASS

- [ ] **Step 5: Run all keybinding tests**

Run: `go test ./internal/keybinding/... -v`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/keybinding/keymap.go internal/keybinding/keymap_test.go
git commit -m "feat(keybinding): add Binding, KeyMap, DefaultKeyMap, and Lookup"
```

---

### Task 3: Resolver with chord support

**Files:**
- Create: `internal/keybinding/resolver.go`
- Test: `internal/keybinding/resolver_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/keybinding/resolver_test.go`:

```go
package keybinding

import "testing"

func TestResolverSingleKey(t *testing.T) {
	r := NewResolver(DefaultKeyMap())

	tests := []struct {
		key          string
		wantAction   Action
		wantArg      int
	}{
		{"j", ActionPrevTicket, 0},
		{"k", ActionNextTicket, 0},
		{"h", ActionPrevColumn, 0},
		{"l", ActionNextColumn, 0},
		{"q", ActionQuit, 0},
		{"enter", ActionOpenTicket, 0},
		{"a", ActionAddTicket, 0},
		{"d", ActionDeleteTicket, 0},
		{"s", ActionStartAgent, 0},
		{"x", ActionStopAgent, 0},
		{"r", ActionRefresh, 0},
		{"tab", ActionToggleFocus, 0},
		{"shift+tab", ActionPrevFocus, 0},
		{"1", ActionJumpColumn1, 0},
		{"2", ActionJumpColumn2, 0},
		{"3", ActionJumpColumn3, 0},
		{"4", ActionJumpColumn4, 0},
		{"?", ActionShowHelp, 0},
		{"z", ActionNone, 0},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			r.Reset()
			action, arg := r.Resolve(tt.key)
			if action != tt.wantAction || arg != tt.wantArg {
				t.Errorf("Resolve(%q) = (%s, %d), want (%s, %d)", tt.key, action, arg, tt.wantAction, tt.wantArg)
			}
		})
	}
}

func TestResolverChordGoToTicket(t *testing.T) {
	r := NewResolver(DefaultKeyMap())

	action, arg := r.Resolve("g")
	if action != ActionNone || arg != 0 {
		t.Fatalf("first 'g': got (%s, %d), want (none, 0)", action, arg)
	}

	action, arg = r.Resolve("3")
	if action != ActionGoToTicket || arg != 3 {
		t.Fatalf("after 'g3': got (%s, %d), want (go_to_ticket, 3)", action, arg)
	}
}

func TestResolverChordMultiDigit(t *testing.T) {
	r := NewResolver(DefaultKeyMap())

	r.Resolve("g")
	r.Resolve("1")
	action, arg := r.Resolve("2")

	if action != ActionGoToTicket || arg != 12 {
		t.Errorf("after 'g12': got (%s, %d), want (go_to_ticket, 12)", action, arg)
	}
}

func TestResolverChordCancelledByNonDigit(t *testing.T) {
	r := NewResolver(DefaultKeyMap())

	r.Resolve("g")
	action, arg := r.Resolve("q")

	if action != ActionQuit || arg != 0 {
		t.Errorf("after 'gq': got (%s, %d), want (quit, 0)", action, arg)
	}
}

func TestResolverChordCancelledByRepeatPrefix(t *testing.T) {
	r := NewResolver(DefaultKeyMap())

	r.Resolve("g")
	action, arg := r.Resolve("g")

	if action != ActionNone || arg != 0 {
		t.Errorf("after 'gg': second 'g' should start new chord, got (%s, %d)", action, arg)
	}

	if !r.InChordMode() {
		t.Error("after 'gg': resolver should be in chord mode from second 'g'")
	}
}

func TestResolverReset(t *testing.T) {
	r := NewResolver(DefaultKeyMap())

	r.Resolve("g")
	if !r.InChordMode() {
		t.Fatal("expected chord mode after 'g'")
	}

	r.Reset()
	if r.InChordMode() {
		t.Error("expected chord mode cleared after Reset()")
	}

	action, arg := r.Resolve("j")
	if action != ActionPrevTicket {
		t.Errorf("after reset, 'j' should work normally, got %s", action)
	}
}

func TestResolverCtrlC(t *testing.T) {
	r := NewResolver(DefaultKeyMap())

	action, arg := r.Resolve("ctrl+c")
	if action != ActionForceQuit || arg != 0 {
		t.Errorf("Resolve(ctrl+c) = (%s, %d), want (force_quit, 0)", action, arg)
	}
}

func TestResolverUnknownKeyInChordMode(t *testing.T) {
	r := NewResolver(DefaultKeyMap())

	r.Resolve("g")
	action, arg := r.Resolve("z")

	if action != ActionNone || arg != 0 {
		t.Errorf("after 'gz': got (%s, %d), want (none, 0) — chord cancelled, unknown key", action, arg)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/keybinding/... -v -run TestResolver`
Expected: FAIL — `NewResolver` not defined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/keybinding/resolver.go`:

```go
package keybinding

import (
	"strconv"
	"strings"
	"unicode"
)

type Resolver struct {
	keyMap      KeyMap
	chordMode   bool
	chordPrefix string
	digitBuf    strings.Builder
}

func NewResolver(km KeyMap) *Resolver {
	return &Resolver{keyMap: km}
}

func (r *Resolver) Resolve(key string) (Action, int) {
	if r.chordMode {
		if len(key) == 1 && unicode.IsDigit(rune(key[0])) {
			r.digitBuf.WriteString(key)
			return ActionNone, 0
		}

		if r.digitBuf.Len() > 0 {
			n, _ := strconv.Atoi(r.digitBuf.String())
			r.chordMode = false
			r.chordPrefix = ""
			r.digitBuf.Reset()
			return ActionGoToTicket, n
		}

		r.chordMode = false
		r.chordPrefix = ""
		r.digitBuf.Reset()
	}

	binding, ok := r.keyMap.Lookup(key)
	if !ok {
		return ActionNone, 0
	}

	if binding.IsChord {
		r.chordMode = true
		r.chordPrefix = key
		return ActionNone, 0
	}

	return binding.Action, 0
}

func (r *Resolver) Reset() {
	r.chordMode = false
	r.chordPrefix = ""
	r.digitBuf.Reset()
}

func (r *Resolver) InChordMode() bool {
	return r.chordMode
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/keybinding/... -v -run TestResolver`
Expected: ALL PASS

- [ ] **Step 5: Run all keybinding tests**

Run: `go test ./internal/keybinding/... -v`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/keybinding/resolver.go internal/keybinding/resolver_test.go
git commit -m "feat(keybinding): add Resolver with chord support for go-to-ticket"
```

---

### Task 4: Config override support (ApplyConfig)

**Files:**
- Create: `internal/keybinding/config.go`
- Test: `internal/keybinding/config_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/keybinding/config_test.go`:

```go
package keybinding

import "testing"

func TestApplyConfigSingleOverride(t *testing.T) {
	km := DefaultKeyMap()
	overrides := map[string]string{
		"next_column": "L",
	}

	ApplyConfig(&km, overrides)

	for _, b := range km.Bindings {
		if b.Action == ActionNextColumn && b.Key == "l" {
			t.Error("old key 'l' still present for next_column after override")
		}
	}

	found := false
	for _, b := range km.Bindings {
		if b.Action == ActionNextColumn && b.Key == "L" {
			found = true
		}
	}
	if !found {
		t.Error("new key 'L' not found for next_column after override")
	}
}

func TestApplyConfigEmptyMap(t *testing.T) {
	km1 := DefaultKeyMap()
	km2 := DefaultKeyMap()
	ApplyConfig(&km2, map[string]string{})

	if len(km1.Bindings) != len(km2.Bindings) {
		t.Error("empty config changed number of bindings")
	}
}

func TestApplyConfigUnknownActionIgnored(t *testing.T) {
	km := DefaultKeyMap()
	originalLen := len(km.Bindings)

	ApplyConfig(&km, map[string]string{
		"nonexistent_action": "X",
	})

	if len(km.Bindings) != originalLen {
		t.Error("unknown action should be ignored, binding count changed")
	}
}

func TestApplyConfigGoToTicketPrefix(t *testing.T) {
	km := DefaultKeyMap()
	ApplyConfig(&km, map[string]string{
		"go_to_ticket_prefix": "G",
	})

	for _, b := range km.Bindings {
		if b.Action == ActionGoToTicket && b.Key == "g" {
			t.Error("old prefix 'g' still present after override")
		}
	}

	found := false
	for _, b := range km.Bindings {
		if b.Action == ActionGoToTicket && b.Key == "G" && b.IsChord {
			found = true
		}
	}
	if !found {
		t.Error("new prefix 'G' not found with IsChord=true")
	}
}

func TestApplyConfigReplacesAllBindingsForAction(t *testing.T) {
	km := DefaultKeyMap()
	ApplyConfig(&km, map[string]string{
		"prev_column": "H",
	})

	for _, b := range km.Bindings {
		if b.Action == ActionPrevColumn {
			if b.Key != "H" {
				t.Errorf("expected all prev_column bindings to be 'H', got %q", b.Key)
			}
		}
	}
}

func TestActionNamesMap(t *testing.T) {
	for name, action := range actionNames {
		got := action.String()
		if got != name {
			t.Errorf("actionNames[%q] = Action with String()=%q, mismatch", name, got)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/keybinding/... -v -run TestApplyConfig`
Expected: FAIL — `ApplyConfig` not defined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/keybinding/config.go`:

```go
package keybinding

var actionNames = map[string]Action{
	"quit":                ActionQuit,
	"force_quit":          ActionForceQuit,
	"prev_column":         ActionPrevColumn,
	"next_column":         ActionNextColumn,
	"prev_ticket":         ActionPrevTicket,
	"next_ticket":         ActionNextTicket,
	"open_ticket":         ActionOpenTicket,
	"add_ticket":          ActionAddTicket,
	"delete_ticket":       ActionDeleteTicket,
	"start_agent":         ActionStartAgent,
	"stop_agent":          ActionStopAgent,
	"refresh":             ActionRefresh,
	"toggle_focus":        ActionToggleFocus,
	"prev_focus":          ActionPrevFocus,
	"jump_col1":           ActionJumpColumn1,
	"jump_col2":           ActionJumpColumn2,
	"jump_col3":           ActionJumpColumn3,
	"jump_col4":           ActionJumpColumn4,
	"show_help":           ActionShowHelp,
	"go_to_ticket_prefix": ActionGoToTicket,
}

func ApplyConfig(km *KeyMap, overrides map[string]string) {
	for name, newKey := range overrides {
		action, ok := actionNames[name]
		if !ok {
			continue
		}

		filtered := km.Bindings[:0]
		for _, b := range km.Bindings {
			if b.Action != action {
				filtered = append(filtered, b)
			}
		}

		isChord := action == ActionGoToTicket
		filtered = append(filtered, Binding{
			Key:     newKey,
			Action:  action,
			IsChord: isChord,
		})
		km.Bindings = filtered
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/keybinding/... -v -run TestApplyConfig`
Expected: ALL PASS

- [ ] **Step 5: Run all keybinding tests**

Run: `go test ./internal/keybinding/... -v`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/keybinding/config.go internal/keybinding/config_test.go
git commit -m "feat(keybinding): add ApplyConfig for TOML keybinding overrides"
```

---

### Task 5: Wire Keybindings into TUIConfig

**Files:**
- Modify: `internal/config/tui.go`
- Modify: `internal/config/defaults.go`
- Modify: `internal/config/scaffold.go`
- Test: `internal/config/config_test.go` (verify keybindings load from TOML)

- [ ] **Step 1: Write the failing test**

Add to `internal/config/config_test.go` (read existing file first to find insertion point):

```go
func TestLoadWithKeybindings(t *testing.T) {
	dir := t.TempDir()
	projectDir := dir + "/projects/test-project"
	os.MkdirAll(projectDir, 0755)

	tomlContent := `
[general]
log = "debug"

[tui]
theme = "dracula"
layout = "comfortable"

[tui.keybindings]
next_column = "L"
prev_column = "H"
`
	os.WriteFile(dir+"/config.toml", []byte(tomlContent), 0644)

	cfg, err := config.LoadFromDir(dir, "test-project")
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}

	if cfg.TUI.Keybindings["next_column"] != "L" {
		t.Errorf("keybindings next_column = %q, want %q", cfg.TUI.Keybindings["next_column"], "L")
	}
	if cfg.TUI.Keybindings["prev_column"] != "H" {
		t.Errorf("keybindings prev_column = %q, want %q", cfg.TUI.Keybindings["prev_column"], "H")
	}
	if cfg.TUI.Theme != "dracula" {
		t.Errorf("theme = %q, want %q", cfg.TUI.Theme, "dracula")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/... -v -run TestLoadWithKeybindings`
Expected: FAIL — `TUIConfig.Keybindings` field doesn't exist yet.

- [ ] **Step 3: Modify TUIConfig to add Keybindings field**

Edit `internal/config/tui.go` to:

```go
package config

type TUIConfig struct {
	Theme       string            `toml:"theme"`
	Layout      string            `toml:"layout"`
	Keybindings map[string]string `toml:"keybindings"`
}
```

- [ ] **Step 4: Update scaffold to include keybindings section**

Edit `internal/config/scaffold.go` default config content to include:

```toml
[tui]
theme = "default"
layout = "compact"

# [tui.keybindings]
# next_column = "l"
# prev_column = "h"
# next_ticket = "j"
# prev_ticket = "k"
# open_ticket = "enter"
# add_ticket = "a"
# delete_ticket = "d"
# start_agent = "s"
# stop_agent = "x"
# refresh = "r"
# show_help = "?"
# toggle_focus = "tab"
# prev_focus = "shift+tab"
# jump_col1 = "1"
# jump_col2 = "2"
# jump_col3 = "3"
# jump_col4 = "4"
# go_to_ticket_prefix = "g"
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/config/... -v -run TestLoadWithKeybindings`
Expected: PASS

- [ ] **Step 6: Run all config and keybinding tests**

Run: `go test ./internal/config/... ./internal/keybinding/... -v`
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add internal/config/tui.go internal/config/scaffold.go internal/config/config_test.go
git commit -m "feat(config): add Keybindings field to TUIConfig with TOML support"
```

---

### Task 6: Run go vet and full test suite

- [ ] **Step 1: Run go vet on all packages**

Run: `go vet ./...`
Expected: No issues.

- [ ] **Step 2: Run full test suite**

Run: `go test ./... -v`
Expected: ALL PASS — config, store, and keybinding packages all green.

- [ ] **Step 3: Commit if any fixes were needed**

```bash
git add -A
git commit -m "fix(keybinding): address vet and test issues"
```
