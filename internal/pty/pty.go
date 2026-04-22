package pty

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

type state int

const (
	stateWaitingReady state = iota
	stateSendingPrompt
	stateWorking
	stateDone
)

type PtyRunner struct {
	sessionName string
	registry    map[string]*Config
	activePty   map[string]*os.File
	activeCmd   map[string]*exec.Cmd
	mu          sync.RWMutex
}

func NewPtyRunner(sessionName string) *PtyRunner {
	return &PtyRunner{
		sessionName: sessionName,
		registry:    NewRegistry(),
		activePty:   make(map[string]*os.File),
		activeCmd:   make(map[string]*exec.Cmd),
	}
}

func (p *PtyRunner) Start(sessionID, agentName, prompt string, autoExit bool) error {
	cfg, ok := p.registry[agentName]
	if !ok {
		return fmt.Errorf("unknown agent: %s", agentName)
	}

	cmd := exec.Command(cfg.Bin, cfg.Args...)
	cmd.Dir = filepath.Dir(os.Args[0])

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("pty start: %w", err)
	}

	p.mu.Lock()
	p.activePty[sessionID] = ptmx
	p.activeCmd[sessionID] = cmd
	p.mu.Unlock()

	go p.runStateMachine(sessionID, ptmx, cmd, cfg, prompt, autoExit)

	return nil
}

func (p *PtyRunner) runStateMachine(sessionID string, ptmx *os.File, cmd *exec.Cmd, cfg *Config, prompt string, autoExit bool) {
	defer p.cleanup(sessionID)

	oldState, err := term.MakeRaw(int(ptmx.Fd()))
	if err == nil {
		defer term.Restore(int(ptmx.Fd()), oldState)
	}

	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)

	go func() {
		for range sigwinch {
			cols, rows, _ := term.GetSize(int(ptmx.Fd()))
			if cols > 0 && rows > 0 {
				pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
			}
		}
	}()

	buf := make([]byte, 4096)
	var output bytes.Buffer

	s := stateWaitingReady
	readyTimeout := time.After(cfg.ReadyWait)

	for s != stateDone {
		select {
		case <-readyTimeout:
			if s == stateWaitingReady {
				return
			}
		default:
		}

		ptmx.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, err := ptmx.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		output.Write(buf[:n])

		switch s {
		case stateWaitingReady:
			if cfg.ReadyPattern != nil && cfg.ReadyPattern.Match(output.Bytes()) {
				s = stateSendingPrompt
				output.Reset()
			}
			if len(output.Bytes()) > 0 && readyTimeout != nil {
				select {
				case <-readyTimeout:
					return
				default:
				}
			}

		case stateSendingPrompt:
			time.Sleep(100 * time.Millisecond)
			formatted := cfg.FormatPrompt(prompt)
			if cfg.SendPrompt != nil {
				cfg.SendPrompt(ptmx, formatted)
			}
			s = stateWorking
			if cfg.GracePeriod > 0 {
				time.Sleep(cfg.GracePeriod)
			}

		case stateWorking:
			if bytes.Contains(output.Bytes(), []byte(DoneMarker)) {
				s = stateDone
				break
			}
			for _, idle := range cfg.IdlePatterns {
				if idle.Match(output.Bytes()) {
					time.Sleep(500 * time.Millisecond)
					s = stateDone
					break
				}
			}
			if s == stateDone {
				break
			}

			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				s = stateDone
				break
			}
		}
	}

	if autoExit {
		time.Sleep(cfg.FallbackTimeout)
	}
}

func (p *PtyRunner) Stop(sessionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cmd, ok := p.activeCmd[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	ptmx, ok := p.activePty[sessionID]
	if ok {
		ptmx.Write([]byte{0x03})
		time.Sleep(100 * time.Millisecond)
	}

	if cmd.Process != nil {
		cmd.Process.Signal(syscall.SIGTERM)
		time.Sleep(200 * time.Millisecond)
		cmd.Process.Kill()
	}

	return nil
}

func (p *PtyRunner) SendInput(sessionID, input string) error {
	p.mu.RLock()
	ptmx, ok := p.activePty[sessionID]
	p.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	_, err := ptmx.Write([]byte(input))
	return err
}

func (p *PtyRunner) GetPty(sessionID string) (*os.File, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ptmx, ok := p.activePty[sessionID]
	return ptmx, ok
}

func (p *PtyRunner) cleanup(sessionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if ptmx, ok := p.activePty[sessionID]; ok {
		ptmx.Close()
		delete(p.activePty, sessionID)
	}
	delete(p.activeCmd, sessionID)
}

func (p *PtyRunner) RegisterAgent(name string, cfg *Config) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.registry[name] = cfg
}

func (p *PtyRunner) UnregisterAgent(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.registry, name)
}