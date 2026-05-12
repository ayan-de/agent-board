package board

type Intent interface{ isIntent() }

type IntentSelectTicket struct{ TicketID string }
type IntentCreateTicket struct{ ColumnIndex int }
type IntentDeleteTicket struct{ TicketID string }
type IntentMoveTicket struct{ TicketID, NewStatus string }

type IntentEditField struct{ Field, Value string }
type IntentCycleStatus struct{}
type IntentAssignAgent struct{ AgentName string }
type IntentApproveProposal struct{}
type IntentStartRun struct{}

type IntentRefreshDashboard struct{}
type IntentStartAdHocRun struct{ Agent, Prompt string }

type IntentOpenView struct{ View ViewType }
type IntentCloseModal struct{}
type IntentConfirmModal struct{}
type IntentShowPalette struct{}

func (IntentSelectTicket) isIntent()     {}
func (IntentCreateTicket) isIntent()    {}
func (IntentDeleteTicket) isIntent()    {}
func (IntentMoveTicket) isIntent()      {}
func (IntentEditField) isIntent()       {}
func (IntentCycleStatus) isIntent()     {}
func (IntentAssignAgent) isIntent()     {}
func (IntentApproveProposal) isIntent() {}
func (IntentStartRun) isIntent()        {}
func (IntentRefreshDashboard) isIntent() {}
func (IntentStartAdHocRun) isIntent()   {}
func (IntentOpenView) isIntent()        {}
func (IntentCloseModal) isIntent()      {}
func (IntentConfirmModal) isIntent()    {}
func (IntentShowPalette) isIntent()     {}