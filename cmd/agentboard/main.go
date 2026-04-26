package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

	if cfg.General.Tmux == "auto" || cfg.General.Tmux == "true" {
		if !tmux.IsInTmux() {
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
	}

	s, err := store.Open(cfg.DB.Path, cfg.Board.Statuses, cfg.Board.Prefix)
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

	var runner orchestrator.Runner = orchestrator.NewExecRunner()
	var ptyRunner *orchestrator.PtyRunner
	if tmux.IsInTmux() {
		sessionName := cfg.ProjectName
		if tmuxRunner, err := orchestrator.NewTmuxRunner(sessionName); err == nil {
			runner = tmuxRunner
		}
		if pr, err := orchestrator.NewPtyRunner(sessionName); err == nil {
			ptyRunner = pr
		}
	}
	mcpManager := mcp.NewManager(cfg.MCP)
	ctxCarry := mcp.NewContextCarryAdapter(mcpManager, cfg.ProjectName)
	orch := orchestrator.NewService(s, llmClient, runner, ctxCarry)
	if ptyRunner != nil {
		orch.SetPtyRunner(ptyRunner)
	}

	reg := theme.NewRegistry("dark")
	reg.LoadBuiltins()
	themesDir := filepath.Join(config.GetBaseDir(), "themes")
	reg.LoadUserThemes(themesDir)
	if err := reg.Set(cfg.TUI.Theme); err != nil {
		_ = reg.Set("agentboard")
	}

	app, err := tui.NewApp(cfg, s, reg, tui.AppDeps{
		Orchestrator: orch,
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
