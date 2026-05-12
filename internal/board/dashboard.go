package board

import (
	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/store"
)

func DashboardRefresh(b *BoardService) BoardViewState {
	agents := config.DetectAgents()
	b.state.Dashboard.Agents = agents

	sessions := b.orchestrator.GetActiveSessions()
	b.state.Dashboard.ActiveSessions = make(map[string]store.Session)
	for _, s := range sessions {
		b.state.Dashboard.ActiveSessions[s.Agent] = store.Session{
			ID:       s.SessionID,
			TicketID: s.TicketID,
			Agent:    s.Agent,
			Status:   s.Status,
		}
	}
	return *b.state
}

func DashboardSyncPane(b *BoardService) BoardViewState {
	return DashboardRefresh(b)
}
