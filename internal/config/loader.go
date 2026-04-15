package config

import (
	"fmt"
	"os"
	"strings"

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

func SaveTheme(path, themeName string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			raw = []byte{}
		} else {
			return fmt.Errorf("config.saveTheme: reading %s: %w", path, err)
		}
	}

	var data map[string]any
	if len(raw) > 0 {
		if _, err := toml.Decode(string(raw), &data); err != nil {
			return fmt.Errorf("config.saveTheme: parsing %s: %w", path, err)
		}
	}

	if data == nil {
		data = make(map[string]any)
	}

	tui, ok := data["tui"].(map[string]any)
	if !ok {
		tui = make(map[string]any)
	}
	tui["theme"] = themeName
	data["tui"] = tui

	var buf strings.Builder
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("config.saveTheme: encoding: %w", err)
	}

	if err := os.WriteFile(path, []byte(buf.String()), 0644); err != nil {
		return fmt.Errorf("config.saveTheme: writing %s: %w", path, err)
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
