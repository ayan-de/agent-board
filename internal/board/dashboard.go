package board

import (
	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

type DashboardStyles struct {
	Border      lipgloss.Style
	Title       lipgloss.Style
	CardFound   lipgloss.Style
	CardMissing lipgloss.Style
	Label       lipgloss.Style
	Value       lipgloss.Style
	Footer      lipgloss.Style
	PaneContent lipgloss.Style
}

func NewDashboardStyles(t *theme.Theme) DashboardStyles {
	return DashboardStyles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary),
		CardFound: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Success).
			Padding(0, 1).
			Width(30),
		CardMissing: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.TextMuted).
			Padding(0, 1).
			Width(30),
		Label: lipgloss.NewStyle().
			Foreground(t.Text),
		Value: lipgloss.NewStyle().
			Foreground(t.Text),
		Footer: lipgloss.NewStyle().
			Foreground(t.TextMuted),
		PaneContent: lipgloss.NewStyle().
			Foreground(t.Text).
			Background(t.Background).
			Padding(1),
	}
}

func DashboardRefresh(b *BoardService) BoardViewState {
	agents := config.DetectAgents()
	b.state.Dashboard.Agents = agents

	sessions := b.orchestrator.GetActiveSessions()
	b.state.Dashboard.ActiveSessions = make(map[string]store.Session)
	for _, s := range sessions {
		b.state.Dashboard.ActiveSessions[s.Agent] = store.Session{
			ID:        s.SessionID,
			TicketID:  s.TicketID,
			Agent:     s.Agent,
			Status:    s.Status,
		}
	}
	return *b.state
}

func DashboardSyncPane(b *BoardService) BoardViewState {
	return DashboardRefresh(b)
}