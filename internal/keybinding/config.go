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
	"next_tab":            ActionNextTab,
	"prev_tab":            ActionPrevTab,
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
