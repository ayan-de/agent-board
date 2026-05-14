package keybinding

import (
	"fmt"
	"strings"
)

type Binding struct {
	Key         string
	Action      Action
	IsChord     bool
	Description string
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

func (km *KeyMap) GetByAction(action Action) (key string) {
	for _, b := range km.Bindings {
		if b.Action == action {
			return b.Key
		}
	}
	return ""
}

// TicketViewHelp returns the help footer text for ticket view
func (km *KeyMap) TicketViewHelp() string {
	actions := []Action{
		ActionStartAgent,       // s: cycle status
		ActionGenerateProposal, // c: generate proposal
		ActionAssignAgent,      // A: assign agent
		ActionSetPriority,      // p: set priority
		ActionStartRun,         // r: start run
	}
	var parts []string
	for _, a := range actions {
		for _, b := range km.Bindings {
			if b.Action == a {
				parts = append(parts, fmt.Sprintf("%s: %s", b.Key, b.Description))
				break
			}
		}
	}
	return strings.Join(parts, " │ ")
}

// HeaderHelp returns the help text for the header bar
func (km *KeyMap) HeaderHelp() string {
	actions := []Action{
		ActionAddTicket,
		ActionDeleteTicket,
		ActionShowHelp,
		ActionRefresh,
	}
	var parts []string
	for _, a := range actions {
		for _, b := range km.Bindings {
			if b.Action == a {
				parts = append(parts, fmt.Sprintf("%s: %s", b.Key, b.Description))
				break
			}
		}
	}
	return strings.Join(parts, "  │  ")
}

// BindingsForActions returns formatted key: desc pairs for given actions
func (km *KeyMap) BindingsForActions(actions ...Action) string {
	var parts []string
	for _, a := range actions {
		for _, b := range km.Bindings {
			if b.Action == a {
				parts = append(parts, fmt.Sprintf("%s: %s", b.Key, b.Description))
				break
			}
		}
	}
	return strings.Join(parts, "  │  ")
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Bindings: []Binding{
			{Key: "q", Action: ActionQuit, Description: "quit"},
			{Key: "ctrl+c", Action: ActionForceQuit, Description: "force quit"},
			{Key: "h", Action: ActionPrevColumn, Description: "prev column"},
			{Key: "left", Action: ActionPrevColumn, Description: "prev column"},
			{Key: "l", Action: ActionNextColumn, Description: "next column"},
			{Key: "right", Action: ActionNextColumn, Description: "next column"},
			{Key: "j", Action: ActionNextTicket, Description: "next ticket"},
			{Key: "down", Action: ActionNextTicket, Description: "next ticket"},
			{Key: "k", Action: ActionPrevTicket, Description: "prev ticket"},
			{Key: "up", Action: ActionPrevTicket, Description: "prev ticket"},
			{Key: "enter", Action: ActionOpenTicket, Description: "open ticket"},
			{Key: "a", Action: ActionAddTicket, Description: "add ticket"},
			{Key: "d", Action: ActionDeleteTicket, Description: "delete ticket"},
			{Key: "s", Action: ActionCycleStatus, Description: "cycle status"},
			{Key: "x", Action: ActionStopAgent, Description: "stop agent"},
			{Key: "r", Action: ActionStartRun, Description: "start run"},
			{Key: "R", Action: ActionRefresh, Description: "refresh"},
			{Key: "tab", Action: ActionToggleFocus, Description: "toggle focus"},
			{Key: "shift+tab", Action: ActionPrevFocus, Description: "prev focus"},
			{Key: "c", Action: ActionGenerateProposal, Description: "generate proposal"},
			{Key: "A", Action: ActionAssignAgent, Description: "assign agent"},
			{Key: "p", Action: ActionSetPriority, Description: "set priority"},
			{Key: "o", Action: ActionApproveProposal, Description: "approve proposal"},
			{Key: "v", Action: ActionViewProposal, Description: "view proposal"},
			{Key: "d", Action: ActionSetDependsOn, Description: "set depends on"},
			{Key: "1", Action: ActionJumpColumn1, Description: "jump col 1"},
			{Key: "2", Action: ActionJumpColumn2, Description: "jump col 2"},
			{Key: "3", Action: ActionJumpColumn3, Description: "jump col 3"},
			{Key: "4", Action: ActionJumpColumn4, Description: "jump col 4"},
			{Key: "?", Action: ActionShowHelp, Description: "show help"},
			{Key: "g", Action: ActionGoToTicket, IsChord: true, Description: "go to ticket"},
			{Key: "i", Action: ActionShowDashboard, Description: "show dashboard"},
			{Key: "e", Action: ActionInteract, Description: "edit"},
			{Key: "v", Action: ActionSwitchToPane, Description: "switch to pane"},
			{Key: ":", Action: ActionOpenPalette, Description: "open palette"},
		},
	}
}
