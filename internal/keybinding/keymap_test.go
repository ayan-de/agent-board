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
		{"j", ActionNextTicket, false},
		{"down", ActionNextTicket, false},
		{"k", ActionPrevTicket, false},
		{"up", ActionPrevTicket, false},
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
	if !ok || action.Action != ActionNextTicket {
		t.Errorf("Lookup(%q) = (%v, %v), want (ActionNextTicket binding, true)", "j", action, ok)
	}

	_, ok = km.Lookup("zzz")
	if ok {
		t.Error("Lookup(unknown key) should return false")
	}
}
