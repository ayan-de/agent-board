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
	ActionShowDashboard
	ActionOpenPalette
	ActionInteract
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
	case ActionShowDashboard:
		return "show_dashboard"
	case ActionOpenPalette:
		return "open_palette"
	case ActionInteract:
		return "interact"
	default:
		return "unknown"
	}
}
