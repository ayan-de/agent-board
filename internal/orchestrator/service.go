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

	// For tracking active sessions
	activeSessions map[string]*AgentSession
}

// AgentSession tracks an active agent session
type AgentSession struct {
	SessionID string
	TicketID  string
	Agent     string
	StartedAt int64
	Status    string
	PaneID    string
	WindowID  string
}

func NewService(store Store, llm LLMClient, runner Runner, ctx ContextCarryProvider) *Service {
	return &Service{
		store:          store,
		llm:            llm,
		runner:         runner,
		ctx:            ctx,
		logs:           make(map[string][]string),
		inputs:         make(map[string]io.Writer),
		activeSessions: make(map[string]*AgentSession),
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

func (s *Service) StartApprovedRun(ctx context.Context, proposalID string) (store.Session, error) {
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

	// For TmuxRunner, we don't need stdin pipe since it's a pane
	// For ExecRunner, we might still need it
	inputChan := make(chan io.Writer, 1)
	go func() {
		if w, ok := <-inputChan; ok {
			s.mu.Lock()
			s.inputs[session.ID] = w
			s.mu.Unlock()
		}
	}()

	onComplete := func(outcome, summary string) {
		_ = s.FinishRun(context.Background(), FinishRunInput{
			TicketID:  proposal.TicketID,
			SessionID: session.ID,
			Outcome:   outcome,
			Summary:   summary,
		})
	}

	handle, err := s.runner.Start(ctx, RunRequest{
		TicketID:   proposal.TicketID,
		SessionID:  session.ID,
		Agent:      proposal.Agent,
		Prompt:     proposal.Prompt,
		Reporter:   func(line string) { s.AppendLog(session.ID, line) },
		InputChan:  inputChan,
		OnComplete: onComplete,
	})

	if err != nil {
		_ = s.store.EndSession(ctx, session.ID, "failed")
		_ = s.store.SetAgentActive(ctx, proposal.TicketID, false)
		return store.Session{}, err
	}

	s.mu.Lock()
	s.activeSessions[session.ID] = &AgentSession{
		SessionID: session.ID,
		TicketID:  proposal.TicketID,
		Agent:     proposal.Agent,
		StartedAt: session.StartedAt.Unix(),
		Status:    "running",
	}
	s.mu.Unlock()

	_, _ = s.store.CreateEvent(ctx, store.Event{
		TicketID:  proposal.TicketID,
		SessionID: session.ID,
		Kind:      "run.started",
		Payload:   handle.Outcome,
	})

	if handle.Outcome != "running" {
		onComplete(handle.Outcome, handle.Summary)
	}

	return session, nil
}

// GetActiveSessions returns all active agent sessions
func (s *Service) GetActiveSessions() []*AgentSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*AgentSession, 0, len(s.activeSessions))
	for _, sess := range s.activeSessions {
		sessions = append(sessions, sess)
	}
	return sessions
}

// GetActiveSessionByTicket returns the active session for a ticket
func (s *Service) GetActiveSessionByTicket(ticketID string) (*AgentSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, sess := range s.activeSessions {
		if sess.TicketID == ticketID {
			return sess, true
		}
	}
	return nil, false
}

// GetActiveSessionByAgent returns active sessions for a specific agent
func (s *Service) GetActiveSessionByAgent(agent string) *AgentSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, sess := range s.activeSessions {
		if sess.Agent == agent {
			return sess
		}
	}
	return nil
}

// StopSession stops an active agent session
func (s *Service) StopSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	sess, ok := s.activeSessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("session %s not found", sessionID)
	}
	delete(s.activeSessions, sessionID)
	s.mu.Unlock()

	// Try to use TmuxRunner's StopPane if available
	if tmuxRunner, ok := s.runner.(*TmuxRunner); ok {
		_ = tmuxRunner.StopPane(sessionID)
	}

	_ = s.store.EndSession(ctx, sessionID, "cancelled")
	_ = s.store.SetAgentActive(ctx, sess.TicketID, false)

	return nil
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
	// First try the TmuxRunner's SendInput
	if tmuxRunner, ok := s.runner.(*TmuxRunner); ok {
		if err := tmuxRunner.SendInput(sessionID, input); err == nil {
			return nil
		}
	}

	// Fall back to stdin pipe for ExecRunner
	s.mu.RLock()
	w, ok := s.inputs[sessionID]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("service.sendInput: session %s not found or not interactive", sessionID)
	}
	_, err := fmt.Fprintln(w, input)
	return err
}

// GetTmuxRunner returns the runner as a TmuxRunner if it is one
func (s *Service) GetTmuxRunner() (*TmuxRunner, bool) {
	tmuxRunner, ok := s.runner.(*TmuxRunner)
	return tmuxRunner, ok
}

// GetPaneContent returns the current content of a pane
func (s *Service) GetPaneContent(sessionID string, lines int) (string, error) {
	tmuxRunner, ok := s.runner.(*TmuxRunner)
	if !ok {
		return "", fmt.Errorf("pane content only available with TmuxRunner")
	}
	return tmuxRunner.CapturePane(sessionID, lines)
}

// SwitchToPane switches the tmux view to a specific pane
func (s *Service) SwitchToPane(sessionID string) error {
	tmuxRunner, ok := s.runner.(*TmuxRunner)
	if !ok {
		return fmt.Errorf("pane switching only available with TmuxRunner")
	}
	pm := tmuxRunner.GetPaneManager()
	return pm.SwitchToPane(sessionID)
}
