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
	aliases := map[string]bool{
		"go_to_ticket_prefix": true,
	}
	for name, action := range actionNames {
		if aliases[name] {
			continue
		}
		got := action.String()
		if got != name {
			t.Errorf("actionNames[%q] = Action with String()=%q, mismatch", name, got)
		}
	}
}
