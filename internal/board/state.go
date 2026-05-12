package board

import (
	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

type ViewType int

const (
	ViewBoard ViewType = iota
	ViewTicket
	ViewDashboard
	ViewHelp
)

type BoardViewState struct {
	Kanban      KanbanViewState
	Ticket      *TicketViewState
	Dashboard   DashboardViewState
	ActiveView  ViewType
	Notification *NotificationState
	Modal        *ModalState
}

type KanbanTab int

const (
	TabBoard KanbanTab = iota
	TabSearch
	TabDateFilter
)

type KanbanColumn struct {
	Def      config.Column
	Tickets  []store.Ticket
}

type KanbanViewState struct {
	Columns      []KanbanColumn
	ColIndex     int
	Cursors      []int
	ScrollOff    []int
	ColumnDefs   []config.Column
	Tab          KanbanTab
	SearchQuery  string
	MonthOffset  int
	Theme        *theme.Theme
	Styles       KanbanStyles
}

type KanbanStyles struct {
	FocusedColumn   lipgloss.Style
	BlurredColumn   lipgloss.Style
	FocusedTitle    lipgloss.Style
	BlurredTitle    lipgloss.Style
	SelectedTicket  lipgloss.Style
	Ticket          lipgloss.Style
	EmptyColumn     lipgloss.Style
	TabBar          lipgloss.Style
	TabActive       lipgloss.Style
	TabInactive     lipgloss.Style
	SearchBox       lipgloss.Style
	SearchCursor    lipgloss.Style
	SearchBoxActive lipgloss.Style
}

// DefaultKanbanStyles returns styles with hardcoded default colors.
func DefaultKanbanStyles() KanbanStyles {
	return KanbanStyles{
		FocusedColumn: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("69")),
		BlurredColumn: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
		FocusedTitle: lipgloss.NewStyle().
			Background(lipgloss.Color("69")).
			Foreground(lipgloss.Color("15")).
			Bold(true),
		BlurredTitle: lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")),
		SelectedTicket: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")),
		Ticket: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		EmptyColumn: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		TabBar:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		TabActive:   lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true),
		TabInactive: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		SearchBox: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		SearchBoxActive: lipgloss.NewStyle().
			Foreground(lipgloss.Color("211")).Bold(true),
	}
}

// NewKanbanStyles returns styles themed from the given theme.
func NewKanbanStyles(t *theme.Theme) KanbanStyles {
	return KanbanStyles{
		FocusedColumn: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary),
		BlurredColumn: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.TextMuted),
		FocusedTitle: lipgloss.NewStyle().
			Background(t.Primary).
			Foreground(t.Text).
			Bold(true),
		BlurredTitle: lipgloss.NewStyle().
			Background(t.BackgroundPanel).
			Foreground(t.Text),
		SelectedTicket: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Text).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderActive),
		Ticket: lipgloss.NewStyle().
			Foreground(t.Text),
		EmptyColumn: lipgloss.NewStyle().
			Foreground(t.TextMuted),
		TabBar:      lipgloss.NewStyle().Foreground(t.TextMuted),
		TabActive:   lipgloss.NewStyle().Foreground(t.Primary).Bold(true),
		TabInactive: lipgloss.NewStyle().Foreground(t.TextMuted),
		SearchBox: lipgloss.NewStyle().
			Foreground(t.TextMuted),
		SearchBoxActive: lipgloss.NewStyle().
			Foreground(t.Secondary).Bold(true),
	}
}

type TicketViewMode int

const (
	ModeTicketView TicketViewMode = 0
	ModeTicketEdit TicketViewMode = 1
	ModeTicketAgentSelect TicketViewMode = 2
	ModeTicketPrioritySelect TicketViewMode = 3
	ModeTicketDependsOnSelect TicketViewMode = 4
)

type TicketViewState struct {
	Ticket      *store.Ticket
	Cursor      int
	Mode        TicketViewMode
	EditBuffer  string
	Agents      []config.DetectedAgent
	Proposal    *store.Proposal
	Loading     bool
	DependsOnTickets []store.Ticket
}

type DashboardViewState struct {
	Agents        []config.DetectedAgent
	ActiveSessions map[string]store.Session
	PaneID        string
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
	Title    string
	Body     string
	OnConfirm func()
	OnCancel func()
	Active   bool
}