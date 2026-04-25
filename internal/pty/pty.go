package pty

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
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
	cmd.Env = os.Environ()

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
			ws, _ := pty.GetsizeFull(ptmx)
			if ws != nil {
				_ = pty.Setsize(ptmx, ws)
			}
		}
	}()

	injectedPrompt := prompt
	if autoExit && cfg.FormatPrompt != nil {
		injectedPrompt = cfg.FormatPrompt(prompt)
	}

	var outputBuf bytes.Buffer

	readyMarker := regexp.MustCompile(cfg.ReadyPattern.String())
	doneMarkerRe := regexp.MustCompile(regexp.QuoteMeta(DoneMarker))
	idleRes := make([]*regexp.Regexp, len(cfg.IdlePatterns))
	for i, p := range cfg.IdlePatterns {
		idleRes[i] = regexp.MustCompile(p.String())
	}

	current := stateWaitingReady
	buf := make([]byte, 4096)
	var doneOnce sync.Once
	canCheckCompletion := false

	exit := func() {
		doneOnce.Do(func() {
			os.Stderr.WriteString("\n[pty-go] task complete, closing " + cfg.Name + "...\n")
			time.Sleep(500 * time.Millisecond)
			ptmx.Write([]byte{0x03})
			time.Sleep(300 * time.Millisecond)
			ptmx.Write([]byte{0x03})
			time.Sleep(300 * time.Millisecond)
			cmd.Process.Signal(syscall.SIGTERM)
			time.Sleep(1 * time.Second)
			cmd.Process.Kill()
		})
	}

	transitionToWorking := func() {
		outputBuf.Reset()
		current = stateWorking
		time.AfterFunc(cfg.GracePeriod, func() {
			canCheckCompletion = true
		})
	}

	fallback := time.AfterFunc(cfg.FallbackTimeout, func() {
		if current == stateWaitingReady {
			current = stateSendingPrompt
			cfg.SendPrompt(ptmx, injectedPrompt)
			if autoExit {
				transitionToWorking()
			}
		}
	})
	defer fallback.Stop()

	for {
		n, err := ptmx.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		os.Stdout.Write(buf[:n])
		outputBuf.Write(buf[:n])

		switch current {
		case stateWaitingReady:
			stripped := StripANSI(outputBuf.String())
			if readyMarker.MatchString(stripped) {
				current = stateSendingPrompt
				fallback.Stop()
				time.AfterFunc(cfg.ReadyWait, func() {
					cfg.SendPrompt(ptmx, injectedPrompt)
					if autoExit {
						transitionToWorking()
					}
				})
			}

		case stateSendingPrompt:
			time.Sleep(100 * time.Millisecond)
			current = stateWorking
			if autoExit && cfg.GracePeriod > 0 {
				time.Sleep(cfg.GracePeriod)
				canCheckCompletion = true
			}

		case stateWorking:
			if !canCheckCompletion {
				continue
			}
			if outputBuf.Len() > 16384 {
				outputBuf.Next(outputBuf.Len() - 16384)
			}
			recent := StripANSI(outputBuf.String())

			if doneMarkerRe.MatchString(recent) {
				current = stateDone
				go exit()
				continue
			}

			for _, re := range idleRes {
				matches := re.FindAllStringIndex(recent, -1)
				if len(matches) >= 3 {
					current = stateDone
					go exit()
					break
				}
			}

			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				current = stateDone
				go exit()
			}
		case stateDone:
		}
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

func (p *PtyRunner) GetConfig(agentName string) *Config {
	if cfg, ok := p.registry[agentName]; ok {
		return cfg
	}
	return nil
}