package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ayan-de/agent-board/internal/board"
	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/llm"
	"github.com/ayan-de/agent-board/internal/mcp"
	"github.com/ayan-de/agent-board/internal/orchestrator"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"
	"github.com/ayan-de/agent-board/internal/tmux"
	"github.com/ayan-de/agent-board/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	if !tmux.IsInTmux() && cfg.General.Tmux != "never" {
		sessionName := cfg.ProjectName
		cmd := exec.Command("tmux", "new-session", "-A", "-s", sessionName, os.Args[0])
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error launching tmux: %v\n", err)
			os.Exit(1)
		}
		return
	}

	sessionName := cfg.ProjectName
	if actualSession, err := tmux.GetCurrentSessionName(); err == nil {
		sessionName = actualSession
	}

	statuses := make([]string, len(cfg.Board.Columns))
	for i, col := range cfg.Board.Columns {
		statuses[i] = col.Status
	}
	s, err := store.Open(cfg.DB.Path, statuses, cfg.Board.Prefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening store: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	llmClient, err := llm.NewFromConfig(cfg.LLM)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating llm client: %v\n", err)
		os.Exit(1)
	}

	var runner orchestrator.Runner
	var agentRunner orchestrator.AgentRunner
	tmuxRunner, err := orchestrator.NewTmuxRunner(sessionName)
	if err == nil {
		runner = tmuxRunner
	}
	if tr, err := orchestrator.NewTmuxAgentRunner(sessionName); err == nil {
		agentRunner = tr
	}
	mcpManager := mcp.NewManager(cfg.MCP)
	ctxCarry := mcp.NewContextCarryAdapter(mcpManager, cfg.ProjectName)
	orch := orchestrator.NewService(s, llmClient, runner, ctxCarry)
	if agentRunner != nil {
		orch.SetAgentRunner(agentRunner)
	}

	reg := theme.NewRegistry("dark")
	reg.LoadBuiltins()
	themesDir := filepath.Join(config.GetBaseDir(), "themes")
	reg.LoadUserThemes(themesDir)
	if err := reg.Set(cfg.TUI.Theme); err != nil {
		_ = reg.Set("agentboard")
	}

	boardSvc := board.NewBoardService(s, orch, cfg)

	app, err := tui.NewApp(cfg, s, reg, tui.AppDeps{
		Board: boardSvc,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating app: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running tui: %v\n", err)
		os.Exit(1)
	}
}
