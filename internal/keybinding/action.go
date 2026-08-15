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
	ActionNextTab
	ActionPrevTab
	ActionJumpColumn1
	ActionJumpColumn2
	ActionJumpColumn3
	ActionJumpColumn4
	ActionShowHelp
	ActionGenerateProposal
	ActionGoToTicket
	ActionShowDashboard
	ActionOpenPalette
	ActionInteract
	ActionSwitchToPane
	ActionStartRun
	ActionAssignAgent
	ActionSetPriority
	ActionCycleStatus
	ActionApproveProposal
	ActionViewProposal
	ActionSetDependsOn
	ActionJumpSession1
	ActionJumpSession2
	ActionJumpSession3
	ActionJumpSession4
	ActionJumpSession5
	ActionJumpSession6
	ActionJumpSession7
	ActionJumpSession8
	ActionJumpSession9
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
	case ActionNextTab:
		return "next_tab"
	case ActionPrevTab:
		return "prev_tab"
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
	case ActionSwitchToPane:
		return "switch_to_pane"
	case ActionStartRun:
		return "start_run"
	case ActionAssignAgent:
		return "assign_agent"
	case ActionSetPriority:
		return "set_priority"
	case ActionCycleStatus:
		return "cycle_status"
	case ActionApproveProposal:
		return "approve_proposal"
	case ActionViewProposal:
		return "view_proposal"
	case ActionSetDependsOn:
		return "set_depends_on"
	case ActionJumpSession1:
		return "jump_session_1"
	case ActionJumpSession2:
		return "jump_session_2"
	case ActionJumpSession3:
		return "jump_session_3"
	case ActionJumpSession4:
		return "jump_session_4"
	case ActionJumpSession5:
		return "jump_session_5"
	case ActionJumpSession6:
		return "jump_session_6"
	case ActionJumpSession7:
		return "jump_session_7"
	case ActionJumpSession8:
		return "jump_session_8"
	case ActionJumpSession9:
		return "jump_session_9"
	default:
		return "unknown"
	}
}
