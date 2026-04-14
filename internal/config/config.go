package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	General GeneralConfig
	Board   BoardConfig
	Agent   AgentConfig
	TUI     TUIConfig
	LLM     LLMConfig
	DB      DBConfig
	MCP     MCPConfig
}

type GeneralConfig struct {
	Log  string `toml:"log"`
	Addr string `toml:"addr"`
	Mode string `toml:"mode"`
	Tmux string `toml:"tmux"`
}

type BoardConfig struct {
	Statuses []string `toml:"statuses"`
}

type AgentConfig struct {
	Default string `toml:"default"`
}

type TUIConfig struct {
	Theme  string `toml:"theme"`
	Layout string `toml:"layout"`
}

type LLMConfig struct {
	Provider string `toml:"provider"`
	Model    string `toml:"model"`
	APIKey   string `toml:"api_key"`
	BaseURL  string `toml:"base_url"`
}

type DBConfig struct {
	Path string `toml:"path"`
}

type MCPConfig struct {
	NPMPath  string `toml:"npm_path"`
	NodePath string `toml:"node_path"`
}

func applyEnvVars(cfg *Config) {
	if v := os.Getenv("AGENTBOARD_LOG"); v != "" {
		cfg.General.Log = v
	}
	if v := os.Getenv("AGENTBOARD_ADDR"); v != "" {
		cfg.General.Addr = v
	}
	if v := os.Getenv("AGENTBOARD_MODE"); v != "" {
		cfg.General.Mode = v
	}
	if v := os.Getenv("AGENTBOARD_TMUX"); v != "" {
		cfg.General.Tmux = v
	}
	if v := os.Getenv("AGENTBOARD_DB"); v != "" {
		cfg.DB.Path = v
	}
	if v := os.Getenv("AGENTBOARD_LLM_PROVIDER"); v != "" {
		cfg.LLM.Provider = v
	}
	if v := os.Getenv("AGENTBOARD_LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("AGENTBOARD_LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("AGENTBOARD_LLM_BASE_URL"); v != "" {
		cfg.LLM.BaseURL = v
	}
	if v := os.Getenv("AGENTBOARD_NPM_PATH"); v != "" {
		cfg.MCP.NPMPath = v
	}
	if v := os.Getenv("AGENTBOARD_NODE_PATH"); v != "" {
		cfg.MCP.NodePath = v
	}
}

func loadFromFile(cfg *Config, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return fmt.Errorf("config.load: parsing %s: %w", path, err)
	}

	return nil
}
