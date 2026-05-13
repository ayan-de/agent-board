package api

import (
	"encoding/json"
	"net/http"

	"github.com/ayan-de/agent-board/internal/core"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	orch  core.Orchestrator
	store core.Store
}

func NewHandlers(orch core.Orchestrator, store core.Store) *Handlers {
	return &Handlers{orch: orch, store: store}
}

func (h *Handlers) CreateProposal(w http.ResponseWriter, r *http.Request) {
	var input core.CreateProposalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	proposal, err := h.orch.CreateProposal(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(proposal)
}

func (h *Handlers) ApproveProposal(w http.ResponseWriter, r *http.Request) {
	proposalID := chi.URLParam(r, "id")
	if proposalID == "" {
		http.Error(w, "missing proposal id", http.StatusBadRequest)
		return
	}
	if err := h.orch.ApproveProposal(r.Context(), proposalID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) StartApprovedRun(w http.ResponseWriter, r *http.Request) {
	proposalID := chi.URLParam(r, "id")
	if proposalID == "" {
		http.Error(w, "missing proposal id", http.StatusBadRequest)
		return
	}
	session, err := h.orch.StartApprovedRun(r.Context(), proposalID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(session)
}

func (h *Handlers) StartAdHocRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Agent  string `json:"agent"`
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	session, err := h.orch.StartAdHocRun(r.Context(), req.Agent, req.Prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(session)
}

func (h *Handlers) FinishRun(w http.ResponseWriter, r *http.Request) {
	var input core.FinishRunInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.orch.FinishRun(r.Context(), input); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) GetActiveSessions(w http.ResponseWriter, r *http.Request) {
	sessions := h.orch.GetActiveSessions()
	json.NewEncoder(w).Encode(sessions)
}

func (h *Handlers) GetLogs(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}
	logs := h.orch.GetLogs(sessionID)
	json.NewEncoder(w).Encode(map[string][]string{"logs": logs})
}

func (h *Handlers) SendInput(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}
	var req struct {
		Input string `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.orch.SendInput(sessionID, req.Input); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) GetPaneContent(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}
	content, err := h.orch.GetPaneContent(sessionID, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"content": content})
}

func (h *Handlers) SwitchToPane(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}
	if err := h.orch.SwitchToPane(sessionID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Ticket handlers (from store)

func (h *Handlers) ListTickets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var filters store.TicketFilters
	if err := json.NewDecoder(r.Body).Decode(&filters); err != nil {
		// If no body, list all
		filters = store.TicketFilters{}
	}
	tickets, err := h.store.ListTickets(ctx, filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tickets)
}

func (h *Handlers) GetTicket(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")
	if ticketID == "" {
		http.Error(w, "missing ticket id", http.StatusBadRequest)
		return
	}
	ticket, err := h.store.GetTicket(r.Context(), ticketID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(ticket)
}

func (h *Handlers) CreateTicket(w http.ResponseWriter, r *http.Request) {
	var ticket store.Ticket
	if err := json.NewDecoder(r.Body).Decode(&ticket); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	created, err := h.store.CreateTicket(r.Context(), ticket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(created)
}

func (h *Handlers) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")
	if ticketID == "" {
		http.Error(w, "missing ticket id", http.StatusBadRequest)
		return
	}
	var ticket store.Ticket
	if err := json.NewDecoder(r.Body).Decode(&ticket); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ticket.ID = ticketID
	updated, err := h.store.UpdateTicket(r.Context(), ticket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(updated)
}

func (h *Handlers) DeleteTicket(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")
	if ticketID == "" {
		http.Error(w, "missing ticket id", http.StatusBadRequest)
		return
	}
	if err := h.store.DeleteTicket(r.Context(), ticketID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) MoveTicketStatus(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")
	if ticketID == "" {
		http.Error(w, "missing ticket id", http.StatusBadRequest)
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.store.MoveStatus(r.Context(), ticketID, req.Status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) GetProposal(w http.ResponseWriter, r *http.Request) {
	proposalID := chi.URLParam(r, "id")
	if proposalID == "" {
		http.Error(w, "missing proposal id", http.StatusBadRequest)
		return
	}
	proposal, err := h.store.GetProposal(r.Context(), proposalID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(proposal)
}

func (h *Handlers) GetActiveSessionsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessions, err := h.store.ListActiveSessions(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(sessions)
}
