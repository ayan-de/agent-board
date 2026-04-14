package config

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
