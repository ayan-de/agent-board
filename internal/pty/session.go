package pty

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

type SessionState int

const (
	StateWaitingReady SessionState = iota
	StateSendingPrompt
	StateWorking
	StateDone
)

func (s SessionState) String() string {
	switch s {
	case StateWaitingReady:
		return "waiting_ready"
	case StateSendingPrompt:
		return "sending_prompt"
	case StateWorking:
		return "working"
	case StateDone:
		return "done"
	default:
		return "unknown"
	}
}

type Session struct {
	SessionID string
	Agent     string
	TicketID  string
	Chdir     string

	mu                 sync.Mutex
	state              SessionState
	output             []string
	outputMu           sync.Mutex
	cmd                *exec.Cmd
	ptmx               *os.File
	doneCh             chan struct{}
	outcome            string
	summary            string
	canCheckCompletion bool
}

func NewSession(cfg *AgentConfig, sessionID, ticketID, prompt, chdir string) (*Session, error) {
	if chdir != "" {
		abs, err := filepath.Abs(chdir)
		if err != nil {
			return nil, fmt.Errorf("pty.newSession: resolving chdir: %w", err)
		}
		chdir = abs
	}

	cmd := exec.Command(cfg.Bin, cfg.Args...)
	cmd.Env = os.Environ()
	if chdir != "" {
		cmd.Dir = chdir
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("pty.newSession: starting pty: %w", err)
	}

	ws, _ := pty.GetsizeFull(os.Stdin)
	if ws != nil {
		_ = pty.Setsize(ptmx, ws)
	} else {
		_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 40, Cols: 120})
	}

	s := &Session{
		SessionID: sessionID,
		Agent:     cfg.Name,
		TicketID:  ticketID,
		Chdir:     chdir,
		cmd:       cmd,
		ptmx:      ptmx,
		state:     StateWaitingReady,
		doneCh:    make(chan struct{}),
	}

	go s.run(cfg, prompt)

	return s, nil
}

func (s *Session) run(cfg *AgentConfig, prompt string) {
	defer close(s.doneCh)
	defer s.ptmx.Close()

	readyRe := regexp.MustCompile(cfg.ReadyPattern)
	doneRe := regexp.MustCompile(regexp.QuoteMeta(DoneMarker))
	idleRes := make([]*regexp.Regexp, len(cfg.IdlePatterns))
	for i, p := range cfg.IdlePatterns {
		idleRes[i] = regexp.MustCompile(p)
	}

	fired := false
	var firedMu sync.Mutex
	firePrompt := func(fn func()) {
		firedMu.Lock()
		defer firedMu.Unlock()
		if fired {
			return
		}
		fired = true
		fn()
	}
	fallback := time.AfterFunc(cfg.FallbackTimeout, func() {
		firePrompt(func() { s.sendPromptAndTransition(cfg, prompt) })
	})
	defer fallback.Stop()

	buf := make([]byte, 4096)
	var stripBuf []string
	var workingBuf []string

	for {
		n, err := s.ptmx.Read(buf)
		if n > 0 {
			raw := string(buf[:n])
			for _, line := range splitLines(raw) {
				s.AppendOutput(line)
				stripped := StripANSI(line)
				stripBuf = append(stripBuf, stripped)
				if len(stripBuf) > 500 {
					stripBuf = stripBuf[len(stripBuf)-500:]
				}
				workingBuf = append(workingBuf, stripped)
				if len(workingBuf) > 500 {
					workingBuf = workingBuf[len(workingBuf)-500:]
				}
			}
		}

		if err != nil {
			if s.isProcessAlive() {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			s.transitionDone("completed", "Agent process exited")
			return
		}

		s.mu.Lock()
		currentState := s.state
		s.mu.Unlock()

		switch currentState {
		case StateWaitingReady:
			for _, line := range stripBuf {
				if readyRe.MatchString(line) {
					firePrompt(func() {
						fallback.Stop()
						time.AfterFunc(cfg.ReadyWait, func() {
							s.sendPromptAndTransition(cfg, prompt)
						})
					})
					break
				}
			}

		case StateWorking:
			s.mu.Lock()
			canCheck := s.canCheckCompletion
			s.mu.Unlock()
			if !canCheck {
				continue
			}
			detected := DetectCompletionFromBuffer(workingBuf, doneRe, idleRes)
			if detected {
				s.transitionDone("completed", "Agent completed task")
				s.cleanupProcess()
				return
			}
		}
	}
}

func (s *Session) sendPromptAndTransition(cfg *AgentConfig, prompt string) {
	formatted := prompt
	if cfg.FormatPrompt != nil {
		formatted = cfg.FormatPrompt(prompt)
	}

	cfg.SendPrompt(s.ptmx, formatted)

	s.mu.Lock()
	s.state = StateWorking
	s.canCheckCompletion = false
	s.mu.Unlock()

	time.AfterFunc(cfg.GracePeriod, func() {
		s.mu.Lock()
		s.canCheckCompletion = true
		s.mu.Unlock()
	})
}

func (s *Session) transitionDone(outcome, summary string) {
	s.mu.Lock()
	s.state = StateDone
	s.outcome = outcome
	s.summary = summary
	s.mu.Unlock()
}

func (s *Session) cleanupProcess() {
	time.Sleep(500 * time.Millisecond)
	s.ptmx.Write([]byte{0x03})
	time.Sleep(300 * time.Millisecond)
	s.ptmx.Write([]byte{0x03})
	time.Sleep(300 * time.Millisecond)
	s.cmd.Process.Signal(syscall.SIGTERM)
	time.Sleep(1 * time.Second)
	s.cmd.Process.Kill()
	s.cmd.Wait()
}

func (s *Session) State() SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *Session) Outcome() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outcome
}

func (s *Session) Summary() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.summary
}

func (s *Session) Wait() (outcome string, summary string) {
	<-s.doneCh
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outcome, s.summary
}

func (s *Session) DoneCh() <-chan struct{} {
	return s.doneCh
}

func (s *Session) Close() {
	s.cmd.Process.Kill()
}

func (s *Session) AppendOutput(line string) {
	s.outputMu.Lock()
	defer s.outputMu.Unlock()
	s.output = append(s.output, line)
	if len(s.output) > 2000 {
		s.output = s.output[len(s.output)-2000:]
	}
}

func (s *Session) RecentOutput(n int) []string {
	s.outputMu.Lock()
	defer s.outputMu.Unlock()
	if n >= len(s.output) {
		result := make([]string, len(s.output))
		copy(result, s.output)
		return result
	}
	result := make([]string, n)
	copy(result, s.output[len(s.output)-n:])
	return result
}

func (s *Session) SendInput(data []byte) error {
	_, err := s.ptmx.Write(data)
	return err
}

func (s *Session) isProcessAlive() bool {
	if s.cmd.Process == nil {
		return false
	}
	err := s.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

func (s *Session) SetSize(rows, cols uint16) error {
	return pty.Setsize(s.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

func DetectCompletion(cfg *AgentConfig, lines []string, elapsed time.Duration) bool {
	doneRe := regexp.MustCompile(regexp.QuoteMeta(DoneMarker))
	idleRes := make([]*regexp.Regexp, len(cfg.IdlePatterns))
	for i, p := range cfg.IdlePatterns {
		idleRes[i] = regexp.MustCompile(p)
	}
	return DetectCompletionFromBuffer(lines, doneRe, idleRes)
}

func DetectCompletionFromBuffer(lines []string, doneRe *regexp.Regexp, idleRes []*regexp.Regexp) bool {
	recent := lines
	if len(recent) > 200 {
		recent = recent[len(recent)-200:]
	}

	joined := strings.Join(recent, "\n")
	if doneRe.MatchString(joined) {
		return true
	}

	for _, re := range idleRes {
		matches := re.FindAllStringIndex(joined, -1)
		if len(matches) >= 3 {
			return true
		}
	}

	return false
}

func splitLines(s string) []string {
	var lines []string
	current := ""
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, current)
			current = ""
		} else if r != '\r' {
			current += string(r)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
