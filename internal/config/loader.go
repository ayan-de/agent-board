package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

func loadFromFile(cfg *Config, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return fmt.Errorf("config.load: parsing %s: %w", path, err)
	}

	return nil
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
