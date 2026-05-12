package board

import (
	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/store"
)

type ViewType int

const (
	ViewBoard ViewType = iota
	ViewTicket
	ViewDashboard
	ViewHelp
)

type BoardViewState struct {
	Kanban       KanbanViewState
	Ticket       *TicketViewState
	Dashboard    DashboardViewState
	ActiveView   ViewType
	Notification *NotificationState
	Modal        *ModalState
	ShowPalette  bool
}

type KanbanTab int

const (
	TabBoard KanbanTab = iota
	TabSearch
	TabDateFilter
)

type KanbanColumn struct {
	Def     config.Column
	Tickets []store.Ticket
}

type KanbanViewState struct {
	Columns     []KanbanColumn
	ColIndex    int
	Cursors     []int
	ScrollOff   []int
	ColumnDefs  []config.Column
	Tab         KanbanTab
	SearchQuery string
	MonthOffset int
}

type TicketViewMode int

const (
	ModeTicketView            TicketViewMode = 0
	ModeTicketEdit            TicketViewMode = 1
	ModeTicketAgentSelect     TicketViewMode = 2
	ModeTicketPrioritySelect  TicketViewMode = 3
	ModeTicketDependsOnSelect TicketViewMode = 4
)

type TicketViewState struct {
	Ticket           *store.Ticket
	Cursor           int
	Mode             TicketViewMode
	EditBuffer       string
	Agents           []config.DetectedAgent
	Proposal         *store.Proposal
	Loading          bool
	DependsOnTickets []store.Ticket
}

type DashboardViewState struct {
	Agents         []config.DetectedAgent
	ActiveSessions map[string]store.Session
	PaneID         string
}

type NotificationVariant string

const (
	NotificationSuccess NotificationVariant = "success"
	NotificationError   NotificationVariant = "error"
	NotificationInfo    NotificationVariant = "info"
)

type NotificationState struct {
	Title   string
	Message string
	Variant NotificationVariant
}

type ModalState struct {
	Title     string
	Body      string
	OnConfirm func()
	OnCancel  func()
	Active    bool
}
