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
			{Key: "j", Action: ActionNextTicket},
			{Key: "down", Action: ActionNextTicket},
			{Key: "k", Action: ActionPrevTicket},
			{Key: "up", Action: ActionPrevTicket},
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
			{Key: "i", Action: ActionShowDashboard},
			{Key: "e", Action: ActionInteract},
			{Key: ":", Action: ActionOpenPalette},
		},
	}
}
