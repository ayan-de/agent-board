package config

import (
	"testing"
)

func TestSetDefaults(t *testing.T) {
	cfg := SetDefaults()

	if cfg.General.Log != "info" {
		t.Errorf("General.Log = %q, want %q", cfg.General.Log, "info")
	}
	if cfg.General.Addr != ":8080" {
		t.Errorf("General.Addr = %q, want %q", cfg.General.Addr, ":8080")
	}
	if cfg.General.Mode != "tui" {
		t.Errorf("General.Mode = %q, want %q", cfg.General.Mode, "tui")
	}
	if cfg.General.Tmux != "auto" {
		t.Errorf("General.Tmux = %q, want %q", cfg.General.Tmux, "auto")
	}

	wantStatuses := []string{"backlog", "in_progress", "review", "done"}
	if len(cfg.Board.Statuses) != len(wantStatuses) {
		t.Fatalf("Board.Statuses len = %d, want %d", len(cfg.Board.Statuses), len(wantStatuses))
	}
	for i, s := range cfg.Board.Statuses {
		if s != wantStatuses[i] {
			t.Errorf("Board.Statuses[%d] = %q, want %q", i, s, wantStatuses[i])
		}
	}

	if cfg.Agent.Default != "opencode" {
		t.Errorf("Agent.Default = %q, want %q", cfg.Agent.Default, "opencode")
	}

	if cfg.TUI.Theme != "default" {
		t.Errorf("TUI.Theme = %q, want %q", cfg.TUI.Theme, "default")
	}
	if cfg.TUI.Layout != "compact" {
		t.Errorf("TUI.Layout = %q, want %q", cfg.TUI.Layout, "compact")
	}

	if cfg.LLM.Provider != "" {
		t.Errorf("LLM.Provider = %q, want empty", cfg.LLM.Provider)
	}
	if cfg.LLM.APIKey != "" {
		t.Errorf("LLM.APIKey = %q, want empty", cfg.LLM.APIKey)
	}

	if cfg.MCP.NPMPath != "npm" {
		t.Errorf("MCP.NPMPath = %q, want %q", cfg.MCP.NPMPath, "npm")
	}
	if cfg.MCP.NodePath != "node" {
		t.Errorf("MCP.NodePath = %q, want %q", cfg.MCP.NodePath, "node")
	}
}
