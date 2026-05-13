package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ayan-de/agent-board/internal/core"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	router   *chi.Mux
	handlers *Handlers
	hubMgr   *HubManager
	srv      *http.Server
}

func NewServer(orch core.Orchestrator, store core.Store) *Server {
	r := Routes()
	h := NewHandlers(orch, store)

	hubMgr := NewHubManager(orch.CompletionChan())

	r.Get("/health", h.Health)

	r.Route("/api", func(r chi.Router) {
		// Orchestrator endpoints
		r.Post("/proposals", h.CreateProposal)
		r.Post("/proposals/{id}/approve", h.ApproveProposal)
		r.Post("/runs/{id}/start", h.StartApprovedRun)
		r.Post("/runs/adhoc", h.StartAdHocRun)
		r.Post("/runs/{id}/finish", h.FinishRun)
		r.Get("/sessions", h.GetActiveSessions)
		r.Get("/sessions/{id}/logs", h.GetLogs)
		r.Post("/sessions/{id}/input", h.SendInput)
		r.Get("/sessions/{id}/pane", h.GetPaneContent)
		r.Post("/sessions/{id}/switch", h.SwitchToPane)

		// Ticket endpoints
		r.Get("/tickets", h.ListTickets)
		r.Post("/tickets", h.CreateTicket)
		r.Get("/tickets/{id}", h.GetTicket)
		r.Put("/tickets/{id}", h.UpdateTicket)
		r.Delete("/tickets/{id}", h.DeleteTicket)
		r.Post("/tickets/{id}/status", h.MoveTicketStatus)

		// Proposal endpoints
		r.Get("/proposals/{id}", h.GetProposal)

		// Session endpoints
		r.Get("/sessions/list", h.GetActiveSessionsList)

		// WebSocket endpoint
		r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
			sessionID := r.URL.Query().Get("session")
			if sessionID == "" {
				sessionID = "global"
			}
			hub := hubMgr.GetHub(sessionID)
			hub.ServeWs(w, r, sessionID)
		})
	})

	return &Server{
		router:   r,
		handlers: h,
		hubMgr:   hubMgr,
	}
}

func (s *Server) Start(addr string) error {
	s.srv = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Printf("api server listening on %s", addr)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("api server failed: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}
