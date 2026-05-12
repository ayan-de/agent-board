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
type IntentPrevColumn struct{}
type IntentNextColumn struct{}
type IntentPrevTicket struct{}
type IntentNextTicket struct{}
type IntentJumpColumn struct{ Index int }
type IntentCommitField struct{ Field, Value string }
type IntentCancelEdit struct{}
type IntentOpenAgentSelect struct{}
type IntentOpenPrioritySelect struct{}
type IntentOpenDependsOnSelect struct{}
type IntentToggleDependsOn struct{ DependsOnID string }
type IntentMoveCursor struct{ Direction int }
type IntentSelectAgentAtCursor struct{}
type IntentReturnToBoard struct{}
type IntentViewProposal struct{}

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
func (IntentConfirmModal) isIntent()     {}
func (IntentShowPalette) isIntent()      {}
func (IntentPrevColumn) isIntent()      {}
func (IntentNextColumn) isIntent()      {}
func (IntentPrevTicket) isIntent()      {}
func (IntentNextTicket) isIntent()       {}
func (IntentJumpColumn) isIntent()       {}
func (IntentCommitField) isIntent()      {}
func (IntentCancelEdit) isIntent()       {}
func (IntentOpenAgentSelect) isIntent()  {}
func (IntentOpenPrioritySelect) isIntent(){}
func (IntentOpenDependsOnSelect) isIntent(){}
func (IntentToggleDependsOn) isIntent() {}
func (IntentMoveCursor) isIntent()      {}
func (IntentSelectAgentAtCursor) isIntent() {}
func (IntentReturnToBoard) isIntent()   {}
func (IntentViewProposal) isIntent()    {}