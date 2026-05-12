package board

import (
	"context"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/orchestrator"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"
)

type Orchestrator interface {
	CreateProposal(ctx context.Context, input orchestrator.CreateProposalInput) (store.Proposal, error)
	ApproveProposal(ctx context.Context, proposalID string) error
	StartApprovedRun(ctx context.Context, proposalID string) (store.Session, error)
	StartAdHocRun(ctx context.Context, agent, prompt string) (store.Session, error)
	FinishRun(ctx context.Context, input orchestrator.FinishRunInput) error
	GetLogs(sessionID string) []string
	SendInput(sessionID, input string) error
	GetActiveSessions() []*orchestrator.AgentSession
	GetPaneContent(sessionID string, lines int) (string, error)
	SwitchToPane(sessionID string) error
	CompletionChan() <-chan orchestrator.RunCompletion
}

type BoardService struct {
	store    *store.Store
	orchestrator Orchestrator
	config   *config.Config
	registry *theme.Registry

	state *BoardViewState
}

func NewBoardService(s *store.Store, orch Orchestrator, cfg *config.Config, reg *theme.Registry) *BoardService {
	b := &BoardService{
		store:        s,
		orchestrator:  orch,
		config:        cfg,
		registry:      reg,
		state:         &BoardViewState{},
	}

	b.state.Kanban.ColumnDefs = cfg.Board.Columns
	if b.state.Kanban.ColumnDefs == nil {
		b.state.Kanban.ColumnDefs = config.DefaultColumns()
	}
	b.state.Kanban.Cursors = make([]int, len(b.state.Kanban.ColumnDefs))
	b.state.Kanban.ScrollOff = make([]int, len(b.state.Kanban.ColumnDefs))

	b.state.Dashboard.Agents = config.DetectAgents()
	b.state.Dashboard.ActiveSessions = make(map[string]store.Session)

	b.loadKanbanState()
	return b
}

func (b *BoardService) loadKanbanState() *BoardService {
	b.state.Kanban.Columns = make([]KanbanColumn, len(b.state.Kanban.ColumnDefs))
	for i, col := range b.state.Kanban.ColumnDefs {
		tickets, _ := b.store.ListTickets(context.Background(), store.TicketFilters{Status: col.Status})
		if tickets == nil {
			tickets = []store.Ticket{}
		}
		b.state.Kanban.Columns[i] = KanbanColumn{
			Def:     col,
			Tickets: tickets,
		}
	}
	return b
}

func (b *BoardService) ProcessIntent(intent Intent) BoardViewState {
	switch i := intent.(type) {
	case IntentSelectTicket:
		return KanbanSelectTicket(b, i.TicketID)
	case IntentCreateTicket:
		return KanbanCreateTicket(b, i.ColumnIndex)
	case IntentDeleteTicket:
		return KanbanDeleteTicket(b, i.TicketID)
	case IntentMoveTicket:
		return KanbanMoveTicket(b, i.TicketID, i.NewStatus)
	case IntentEditField:
		return TicketSelectField(b, i.Field)
	case IntentCycleStatus:
		return TicketCycleStatus(b)
	case IntentAssignAgent:
		return TicketAssignAgent(b, i.AgentName)
	case IntentApproveProposal:
		if b.state.Ticket != nil && b.state.Ticket.Proposal != nil {
			return ProposalApprove(b, b.state.Ticket.Proposal.ID)
		}
		return *b.state
	case IntentStartRun:
		if b.state.Ticket != nil && b.state.Ticket.Proposal != nil {
			return ProposalStartRun(b, b.state.Ticket.Proposal.ID)
		}
		return *b.state
	case IntentStartAdHocRun:
		session, err := b.orchestrator.StartAdHocRun(context.Background(), i.Agent, i.Prompt)
		if err != nil {
			b.SetNotification("Ad-hoc run failed", err.Error(), NotificationError)
		} else {
			b.SetNotification("Ad-hoc run started", "Agent: "+session.Agent, NotificationSuccess)
		}
		return *b.state
	case IntentRefreshDashboard:
		return DashboardRefresh(b)
	case IntentOpenView:
		b.state.ActiveView = i.View
		return *b.state
	case IntentShowPalette, IntentCloseModal, IntentConfirmModal:
		return *b.state
	default:
		return *b.state
	}
}

func (b *BoardService) GetState() BoardViewState {
	return *b.state
}

func (b *BoardService) SetNotification(title, message string, variant NotificationVariant) {
	b.state.Notification = &NotificationState{
		Title:   title,
		Message: message,
		Variant: variant,
	}
}

func (b *BoardService) ClearNotification() {
	b.state.Notification = nil
}

func (b *BoardService) OpenModal(title, body string, onConfirm, onCancel func()) {
	b.state.Modal = &ModalState{
		Title:    title,
		Body:     body,
		OnConfirm: onConfirm,
		OnCancel:  onCancel,
		Active:   true,
	}
}

func (b *BoardService) CloseModal() {
	b.state.Modal = nil
}

func (b *BoardService) handleRunCompletion(completion orchestrator.RunCompletion) {
	b.state.ActiveView = ViewBoard
	b.loadKanbanState()

	if completion.TicketID != "" {
		b.state.Notification = &NotificationState{
			Title:   "Run completed",
			Message: "Agent finished working on " + completion.TicketID,
			Variant: NotificationSuccess,
		}
	}
}