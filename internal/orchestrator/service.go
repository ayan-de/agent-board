package orchestrator

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/ayan-de/agent-board/internal/llm"
	"github.com/ayan-de/agent-board/internal/store"
)

type Service struct {
	store  Store
	llm    LLMClient
	runner Runner
	ctx    ContextCarryProvider
	logs   map[string][]string
	inputs map[string]io.Writer
	mu     sync.RWMutex
}

func NewService(store Store, llm LLMClient, runner Runner, ctx ContextCarryProvider) *Service {
	return &Service{
		store:  store,
		llm:    llm,
		runner: runner,
		ctx:    ctx,
		logs:   make(map[string][]string),
		inputs: make(map[string]io.Writer),
	}
}

func (s Service) CreateProposal(ctx context.Context, input CreateProposalInput) (store.Proposal, error) {
	if s.store == nil {
		return store.Proposal{}, fmt.Errorf("orchestrator.createProposal: store not configured")
	}
	ticket, err := s.store.GetTicket(ctx, input.TicketID)
	if err != nil {
		return store.Proposal{}, err
	}
	if ticket.Agent == "" {
		return store.Proposal{}, fmt.Errorf("orchestrator.createProposal: assigned agent is required")
	}

	cc, _ := s.store.GetContextCarry(ctx, input.TicketID)
	externalCtx, _ := s.ctx.LoadContext(ctx, input.TicketID)

	carrySummary := cc.Summary
	if externalCtx != "" {
		carrySummary = fmt.Sprintf("%s\n\nExternal context:\n%s", carrySummary, externalCtx)
	}

	draft, err := s.llm.GenerateProposal(ctx, llm.ProposalPrompt{
		TicketID:      ticket.ID,
		Title:         ticket.Title,
		Description:   ticket.Description,
		AssignedAgent: ticket.Agent,
		ContextCarry:  carrySummary,
	})

	if err != nil {
		return store.Proposal{}, fmt.Errorf("orchestrator.createProposal: %w", err)
	}

	proposal, err := s.store.CreateProposal(ctx, store.Proposal{
		TicketID: ticket.ID,
		Agent:    ticket.Agent,
		Status:   "pending",
		Prompt:   draft.Prompt,
	})
	if err != nil {
		return store.Proposal{}, err
	}

	_, _ = s.store.CreateEvent(ctx, store.Event{
		TicketID: ticket.ID,
		Kind:     "proposal.created",
		Payload:  draft.Prompt,
	})

	return proposal, nil
}

func (s Service) StartApprovedRun(ctx context.Context, proposalID string) (store.Session, error) {
	proposal, err := s.store.GetProposal(ctx, proposalID)
	if err != nil {
		return store.Session{}, err
	}
	if proposal.Status != "approved" {
		return store.Session{}, fmt.Errorf("orchestrator.startApprovedRun: proposal is not approved")
	}
	if s.store.HasActiveSession(ctx, proposal.TicketID) {
		return store.Session{}, fmt.Errorf("orchestrator.startApprovedRun: active session exists")
	}

	session, err := s.store.CreateSession(ctx, store.Session{
		TicketID: proposal.TicketID,
		Agent:    proposal.Agent,
		Status:   "running",
	})
	if err != nil {
		return store.Session{}, err
	}

	if err := s.store.SetAgentActive(ctx, proposal.TicketID, true); err != nil {
		return store.Session{}, err
	}

	inputChan := make(chan io.Writer, 1)
	go func() {
		if w, ok := <-inputChan; ok {
			s.mu.Lock()
			s.inputs[session.ID] = w
			s.mu.Unlock()
		}
	}()

	handle, err := s.runner.Start(ctx, RunRequest{
		TicketID:  proposal.TicketID,
		SessionID: session.ID,
		Agent:     proposal.Agent,
		Prompt:    proposal.Prompt,
		Reporter:  func(line string) { s.AppendLog(session.ID, line) },
		InputChan: inputChan,
	})
	
	s.mu.Lock()
	delete(s.inputs, session.ID)
	s.mu.Unlock()
	if err != nil {
		_ = s.store.EndSession(ctx, session.ID, "failed")
		_ = s.store.SetAgentActive(ctx, proposal.TicketID, false)
		return store.Session{}, err
	}

	_, _ = s.store.CreateEvent(ctx, store.Event{
		TicketID:  proposal.TicketID,
		SessionID: session.ID,
		Kind:      "run.started",
		Payload:   handle.Outcome,
	})

	_ = s.FinishRun(ctx, FinishRunInput{
		TicketID:  proposal.TicketID,
		SessionID: session.ID,
		Outcome:   handle.Outcome,
		Summary:   handle.Summary,
	})

	return session, nil
}

func (s *Service) AppendLog(sessionID, line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs[sessionID] = append(s.logs[sessionID], line)
	if len(s.logs[sessionID]) > 1000 {
		s.logs[sessionID] = s.logs[sessionID][len(s.logs[sessionID])-1000:]
	}
}

func (s *Service) GetLogs(sessionID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.logs[sessionID]
}

func (s *Service) SendInput(sessionID, input string) error {
	s.mu.RLock()
	w, ok := s.inputs[sessionID]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("service.sendInput: session %s not found or not interactive", sessionID)
	}
	_, err := fmt.Fprintln(w, input)
	return err
}
