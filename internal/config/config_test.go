package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFileGlobal(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	content := []byte(`
[general]
log = "debug"
addr = ":9090"

[tui]
theme = "catppuccin"
layout = "spacious"

[agent]
default = "claude-code"

[llm]
provider = "anthropic"
model = "claude-sonnet-4-20250514"

[mcp]
npm_path = "/usr/local/bin/npm"
`)
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := SetDefaults()
	err := loadFromFile(cfg, cfgPath)
	if err != nil {
		t.Fatalf("loadFromFile: %v", err)
	}

	if cfg.General.Log != "debug" {
		t.Errorf("General.Log = %q, want %q", cfg.General.Log, "debug")
	}
	if cfg.General.Addr != ":9090" {
		t.Errorf("General.Addr = %q, want %q", cfg.General.Addr, ":9090")
	}
	if cfg.TUI.Theme != "catppuccin" {
		t.Errorf("TUI.Theme = %q, want %q", cfg.TUI.Theme, "catppuccin")
	}
	if cfg.TUI.Layout != "spacious" {
		t.Errorf("TUI.Layout = %q, want %q", cfg.TUI.Layout, "spacious")
	}
	if cfg.Agent.Default != "claude-code" {
		t.Errorf("Agent.Default = %q, want %q", cfg.Agent.Default, "claude-code")
	}
	if cfg.LLM.Provider != "anthropic" {
		t.Errorf("LLM.Provider = %q, want %q", cfg.LLM.Provider, "anthropic")
	}
	if cfg.LLM.Model != "claude-sonnet-4-20250514" {
		t.Errorf("LLM.Model = %q, want %q", cfg.LLM.Model, "claude-sonnet-4-20250514")
	}
	if cfg.MCP.NPMPath != "/usr/local/bin/npm" {
		t.Errorf("MCP.NPMPath = %q, want %q", cfg.MCP.NPMPath, "/usr/local/bin/npm")
	}

	if cfg.General.Mode != "tui" {
		t.Errorf("General.Mode should retain default %q, got %q", "tui", cfg.General.Mode)
	}
}

func TestLoadFromFileProject(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	content := []byte(`
[board]
statuses = ["todo", "doing", "done"]

[tui]
theme = "dracula"
`)
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := SetDefaults()
	err := loadFromFile(cfg, cfgPath)
	if err != nil {
		t.Fatalf("loadFromFile: %v", err)
	}

	wantStatuses := []string{"todo", "doing", "done"}
	if len(cfg.Board.Statuses) != len(wantStatuses) {
		t.Fatalf("Board.Statuses len = %d, want %d", len(cfg.Board.Statuses), len(wantStatuses))
	}
	for i, s := range cfg.Board.Statuses {
		if s != wantStatuses[i] {
			t.Errorf("Board.Statuses[%d] = %q, want %q", i, s, wantStatuses[i])
		}
	}
	if cfg.TUI.Theme != "dracula" {
		t.Errorf("TUI.Theme = %q, want %q", cfg.TUI.Theme, "dracula")
	}
	if cfg.Agent.Default != "opencode" {
		t.Errorf("Agent.Default should retain default %q, got %q", "opencode", cfg.Agent.Default)
	}
}

func TestLoadFromFileMissing(t *testing.T) {
	cfg := SetDefaults()
	err := loadFromFile(cfg, "/nonexistent/path/config.toml")
	if err != nil {
		t.Errorf("loadFromFile on missing file should not error, got: %v", err)
	}
	if cfg.Agent.Default != "opencode" {
		t.Errorf("defaults should be preserved when file missing")
	}
}

func TestLoadFromFileInvalidTOML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(cfgPath, []byte(`[invalid toml {{{`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := SetDefaults()
	err := loadFromFile(cfg, cfgPath)
	if err == nil {
		t.Fatal("loadFromFile with invalid TOML should return error")
	}
}
