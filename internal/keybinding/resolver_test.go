package keybinding

import "testing"

func TestResolverSingleKey(t *testing.T) {
	r := NewResolver(DefaultKeyMap())

	tests := []struct {
		key        string
		wantAction Action
		wantArg    int
	}{
		{"j", ActionNextTicket, 0},
		{"k", ActionPrevTicket, 0},
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

	action, _ := r.Resolve("j")
	if action != ActionNextTicket {
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
