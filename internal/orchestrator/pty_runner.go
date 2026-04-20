package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"github.com/ayan-de/agent-board/internal/pty"
)

type PtyRunner struct {
	mu       sync.Mutex
	sessions map[string]*pty.Session
	registry map[string]*pty.AgentConfig
	chdir    string
}

func NewPtyRunner(registry map[string]*pty.AgentConfig, chdir string) *PtyRunner {
	return &PtyRunner{
		sessions: make(map[string]*pty.Session),
		registry: registry,
		chdir:    chdir,
	}
}

func (r *PtyRunner) Start(ctx context.Context, req RunRequest) (RunHandle, error) {
	cfg, ok := r.registry[req.Agent]
	if !ok {
		return RunHandle{}, fmt.Errorf("ptyRunner.start: no config for agent %q", req.Agent)
	}

	sess, err := pty.NewSession(cfg, req.SessionID, req.TicketID, req.Prompt, r.chdir)
	if err != nil {
		return RunHandle{}, fmt.Errorf("ptyRunner.start: %w", err)
	}

	r.mu.Lock()
	r.sessions[req.SessionID] = sess
	r.mu.Unlock()

	if req.Reporter != nil {
		req.Reporter(fmt.Sprintf("Agent %s started in PTY session %s", req.Agent, req.SessionID))
	}

	go func() {
		outcome, summary := sess.Wait()
		r.mu.Lock()
		delete(r.sessions, req.SessionID)
		r.mu.Unlock()

		if req.OnComplete != nil {
			req.OnComplete(outcome, summary)
		}
	}()

	return RunHandle{
		Outcome: "running",
		Summary: fmt.Sprintf("Agent %s started in PTY", req.Agent),
	}, nil
}

func (r *PtyRunner) GetSession(sessionID string) (*pty.Session, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[sessionID]
	return s, ok
}

func (r *PtyRunner) GetPTYOutput(sessionID string, lines int) (string, error) {
	r.mu.Lock()
	sess, ok := r.sessions[sessionID]
	r.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("session %s not found", sessionID)
	}

	output := sess.RecentOutput(lines)
	result := ""
	for _, line := range output {
		result += pty.StripANSI(line) + "\n"
	}
	return result, nil
}

func (r *PtyRunner) SendInput(sessionID string, input string) error {
	r.mu.Lock()
	sess, ok := r.sessions[sessionID]
	r.mu.Unlock()
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}
	return sess.SendInput([]byte(input + "\n"))
}

func (r *PtyRunner) Stop(sessionID string) error {
	r.mu.Lock()
	sess, ok := r.sessions[sessionID]
	r.mu.Unlock()
	if !ok {
		return nil
	}
	sess.Close()
	return nil
}
