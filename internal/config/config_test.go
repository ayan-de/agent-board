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
	if cfg.Board.Prefix != "" {
		t.Errorf("Board.Prefix should be empty when not set in config, got %q", cfg.Board.Prefix)
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

func TestApplyEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		check   func(*Config, *testing.T)
	}{
		{
			name: "AGENTBOARD_LOG overrides General.Log",
			envVars: map[string]string{
				"AGENTBOARD_LOG": "debug",
			},
			check: func(cfg *Config, t *testing.T) {
				if cfg.General.Log != "debug" {
					t.Errorf("General.Log = %q, want %q", cfg.General.Log, "debug")
				}
			},
		},
		{
			name: "AGENTBOARD_ADDR overrides General.Addr",
			envVars: map[string]string{
				"AGENTBOARD_ADDR": ":3000",
			},
			check: func(cfg *Config, t *testing.T) {
				if cfg.General.Addr != ":3000" {
					t.Errorf("General.Addr = %q, want %q", cfg.General.Addr, ":3000")
				}
			},
		},
		{
			name: "AGENTBOARD_MODE overrides General.Mode",
			envVars: map[string]string{
				"AGENTBOARD_MODE": "api",
			},
			check: func(cfg *Config, t *testing.T) {
				if cfg.General.Mode != "api" {
					t.Errorf("General.Mode = %q, want %q", cfg.General.Mode, "api")
				}
			},
		},
		{
			name: "AGENTBOARD_TMUX overrides General.Tmux",
			envVars: map[string]string{
				"AGENTBOARD_TMUX": "always",
			},
			check: func(cfg *Config, t *testing.T) {
				if cfg.General.Tmux != "always" {
					t.Errorf("General.Tmux = %q, want %q", cfg.General.Tmux, "always")
				}
			},
		},
		{
			name: "AGENTBOARD_DB overrides DB.Path",
			envVars: map[string]string{
				"AGENTBOARD_DB": "/tmp/custom.db",
			},
			check: func(cfg *Config, t *testing.T) {
				if cfg.DB.Path != "/tmp/custom.db" {
					t.Errorf("DB.Path = %q, want %q", cfg.DB.Path, "/tmp/custom.db")
				}
			},
		},
		{
			name: "AGENTBOARD_LLM_PROVIDER overrides LLM.Provider",
			envVars: map[string]string{
				"AGENTBOARD_LLM_PROVIDER": "ollama",
			},
			check: func(cfg *Config, t *testing.T) {
				if cfg.LLM.Provider != "ollama" {
					t.Errorf("LLM.Provider = %q, want %q", cfg.LLM.Provider, "ollama")
				}
			},
		},
		{
			name: "AGENTBOARD_LLM_MODEL overrides LLM.Model",
			envVars: map[string]string{
				"AGENTBOARD_LLM_MODEL": "llama3",
			},
			check: func(cfg *Config, t *testing.T) {
				if cfg.LLM.Model != "llama3" {
					t.Errorf("LLM.Model = %q, want %q", cfg.LLM.Model, "llama3")
				}
			},
		},
		{
			name: "AGENTBOARD_LLM_API_KEY overrides LLM.APIKey",
			envVars: map[string]string{
				"AGENTBOARD_LLM_API_KEY": "sk-test-key",
			},
			check: func(cfg *Config, t *testing.T) {
				if cfg.LLM.APIKey != "sk-test-key" {
					t.Errorf("LLM.APIKey = %q, want %q", cfg.LLM.APIKey, "sk-test-key")
				}
			},
		},
		{
			name: "AGENTBOARD_LLM_BASE_URL overrides LLM.BaseURL",
			envVars: map[string]string{
				"AGENTBOARD_LLM_BASE_URL": "http://localhost:11434",
			},
			check: func(cfg *Config, t *testing.T) {
				if cfg.LLM.BaseURL != "http://localhost:11434" {
					t.Errorf("LLM.BaseURL = %q, want %q", cfg.LLM.BaseURL, "http://localhost:11434")
				}
			},
		},
		{
			name: "AGENTBOARD_NPM_PATH overrides MCP.NPMPath",
			envVars: map[string]string{
				"AGENTBOARD_NPM_PATH": "/custom/npm",
			},
			check: func(cfg *Config, t *testing.T) {
				if cfg.MCP.NPMPath != "/custom/npm" {
					t.Errorf("MCP.NPMPath = %q, want %q", cfg.MCP.NPMPath, "/custom/npm")
				}
			},
		},
		{
			name: "AGENTBOARD_NODE_PATH overrides MCP.NodePath",
			envVars: map[string]string{
				"AGENTBOARD_NODE_PATH": "/custom/node",
			},
			check: func(cfg *Config, t *testing.T) {
				if cfg.MCP.NodePath != "/custom/node" {
					t.Errorf("MCP.NodePath = %q, want %q", cfg.MCP.NodePath, "/custom/node")
				}
			},
		},
		{
			name:    "no env vars preserves defaults",
			envVars: map[string]string{},
			check: func(cfg *Config, t *testing.T) {
				if cfg.General.Log != "info" {
					t.Errorf("General.Log = %q, want default %q", cfg.General.Log, "info")
				}
				if cfg.Agent.Default != "opencode" {
					t.Errorf("Agent.Default = %q, want default %q", cfg.Agent.Default, "opencode")
				}
			},
		},
		{
			name: "multiple env vars override simultaneously",
			envVars: map[string]string{
				"AGENTBOARD_LOG":          "warn",
				"AGENTBOARD_LLM_PROVIDER": "openai",
				"AGENTBOARD_LLM_MODEL":    "gpt-4o",
				"AGENTBOARD_LLM_API_KEY":  "sk-multi",
				"AGENTBOARD_NPM_PATH":     "/opt/npm",
			},
			check: func(cfg *Config, t *testing.T) {
				if cfg.General.Log != "warn" {
					t.Errorf("General.Log = %q, want %q", cfg.General.Log, "warn")
				}
				if cfg.LLM.Provider != "openai" {
					t.Errorf("LLM.Provider = %q, want %q", cfg.LLM.Provider, "openai")
				}
				if cfg.LLM.Model != "gpt-4o" {
					t.Errorf("LLM.Model = %q, want %q", cfg.LLM.Model, "gpt-4o")
				}
				if cfg.MCP.NPMPath != "/opt/npm" {
					t.Errorf("MCP.NPMPath = %q, want %q", cfg.MCP.NPMPath, "/opt/npm")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg := SetDefaults()
			applyEnvVars(cfg)
			tt.check(cfg, t)
		})
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

func TestLoadFromDir(t *testing.T) {
	baseDir := t.TempDir()
	projectName := "my-app"

	globalCfg := []byte(`
[general]
log = "debug"

[tui]
theme = "catppuccin"

[agent]
default = "claude-code"
`)
	globalPath := filepath.Join(baseDir, "config.toml")
	if err := os.WriteFile(globalPath, globalCfg, 0644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	projDir := filepath.Join(baseDir, "projects", projectName)
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	projCfg := []byte(`
[board]
statuses = ["todo", "doing", "done"]

[tui]
theme = "dracula"
`)
	projPath := filepath.Join(projDir, "config.toml")
	if err := os.WriteFile(projPath, projCfg, 0644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	t.Setenv("AGENTBOARD_LLM_API_KEY", "sk-from-env")
	t.Setenv("AGENTBOARD_LOG", "error")

	cfg, err := LoadFromDir(baseDir, projectName)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}

	if cfg.General.Log != "error" {
		t.Errorf("General.Log = %q, env should override to %q", cfg.General.Log, "error")
	}
	if cfg.General.Addr != ":8080" {
		t.Errorf("General.Addr = %q, want default %q", cfg.General.Addr, ":8080")
	}
	if cfg.TUI.Theme != "dracula" {
		t.Errorf("TUI.Theme = %q, project should override to %q", cfg.TUI.Theme, "dracula")
	}
	if cfg.Agent.Default != "claude-code" {
		t.Errorf("Agent.Default = %q, want global %q", cfg.Agent.Default, "claude-code")
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

	if cfg.LLM.APIKey != "sk-from-env" {
		t.Errorf("LLM.APIKey = %q, want env %q", cfg.LLM.APIKey, "sk-from-env")
	}
	if cfg.DB.Path != filepath.Join(baseDir, "projects", projectName, "board.db") {
		t.Errorf("DB.Path = %q, unexpected", cfg.DB.Path)
	}
}

func TestLoadFromDirNoConfigsExist(t *testing.T) {
	baseDir := t.TempDir()

	cfg, err := LoadFromDir(baseDir, "new-project")
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}

	if cfg.General.Log != "info" {
		t.Errorf("General.Log = %q, want default %q", cfg.General.Log, "info")
	}
	if cfg.Agent.Default != "opencode" {
		t.Errorf("Agent.Default = %q, want default %q", cfg.Agent.Default, "opencode")
	}
	if cfg.Board.Prefix != "NEW-" {
		t.Errorf("Board.Prefix = %q, want default %q", cfg.Board.Prefix, "NEW-")
	}
}

func TestLoadFromDirWithCustomPrefix(t *testing.T) {
	baseDir := t.TempDir()
	projectName := "my-app"

	projDir := filepath.Join(baseDir, "projects", projectName)
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	projCfg := []byte(`
[board]
prefix = "CUSTOM-"
`)
	projPath := filepath.Join(projDir, "config.toml")
	if err := os.WriteFile(projPath, projCfg, 0644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	cfg, err := LoadFromDir(baseDir, projectName)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}

	if cfg.Board.Prefix != "CUSTOM-" {
		t.Errorf("Board.Prefix = %q, want %q", cfg.Board.Prefix, "CUSTOM-")
	}
}

func TestLoadWithKeybindings(t *testing.T) {
	dir := t.TempDir()
	projectDir := dir + "/projects/test-project"
	os.MkdirAll(projectDir, 0755)

	tomlContent := `
[general]
log = "debug"

[tui]
theme = "dracula"
layout = "comfortable"

[tui.keybindings]
next_column = "L"
prev_column = "H"
`
	os.WriteFile(filepath.Join(projectDir, "config.toml"), []byte(tomlContent), 0644)

	cfg, err := LoadFromDir(dir, "test-project")
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}

	if cfg.TUI.Keybindings["next_column"] != "L" {
		t.Errorf("keybindings next_column = %q, want %q", cfg.TUI.Keybindings["next_column"], "L")
	}
	if cfg.TUI.Keybindings["prev_column"] != "H" {
		t.Errorf("keybindings prev_column = %q, want %q", cfg.TUI.Keybindings["prev_column"], "H")
	}
	if cfg.TUI.Theme != "dracula" {
		t.Errorf("theme = %q, want %q", cfg.TUI.Theme, "dracula")
	}
}

func TestSetDefaultsRequireApprovalTrue(t *testing.T) {
	cfg := SetDefaults()
	if !cfg.LLM.RequireApproval {
		t.Fatal("LLM.RequireApproval should default to true")
	}
}

func TestLoadFromDirReadsOrchestrationFields(t *testing.T) {
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "projects", "agent-board")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(projectDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
[llm]
provider = "ollama"
model = "qwen2.5-coder"
base_url = "http://127.0.0.1:11434"
coordinator_model = "qwen2.5-coder"
summarizer_model = "qwen2.5:7b"
require_approval = true
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromDir(baseDir, "agent-board")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.LLM.Provider != "ollama" {
		t.Fatalf("Provider = %q, want ollama", cfg.LLM.Provider)
	}
	if cfg.LLM.CoordinatorModel != "qwen2.5-coder" {
		t.Fatalf("CoordinatorModel = %q, want qwen2.5-coder", cfg.LLM.CoordinatorModel)
	}
	if cfg.LLM.SummarizerModel != "qwen2.5:7b" {
		t.Fatalf("SummarizerModel = %q, want qwen2.5:7b", cfg.LLM.SummarizerModel)
	}
	if !cfg.LLM.RequireApproval {
		t.Fatal("RequireApproval should be true")
	}
}

func TestApplyEnvVarsCoordinatorModel(t *testing.T) {
	t.Setenv("AGENTBOARD_LLM_COORDINATOR_MODEL", "gpt-4o")
	t.Setenv("AGENTBOARD_LLM_SUMMARIZER_MODEL", "gpt-4o-mini")

	cfg := SetDefaults()
	applyEnvVars(cfg)

	if cfg.LLM.CoordinatorModel != "gpt-4o" {
		t.Errorf("CoordinatorModel = %q, want gpt-4o", cfg.LLM.CoordinatorModel)
	}
	if cfg.LLM.SummarizerModel != "gpt-4o-mini" {
		t.Errorf("SummarizerModel = %q, want gpt-4o-mini", cfg.LLM.SummarizerModel)
	}
}

func TestGetGitRemote(t *testing.T) {
	remote := getGitRemote()
	_ = remote
}

func TestGetProjectInitDate(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "projects", "testproj")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}
	date, err := GetProjectInitDate(dir, "testproj")
	if err != nil {
		t.Fatalf("GetProjectInitDate error: %v", err)
	}
	if date.IsZero() {
		t.Error("date should not be zero")
	}
}
