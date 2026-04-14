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
