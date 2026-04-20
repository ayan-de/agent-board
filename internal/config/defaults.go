package config

func SetDefaults() *Config {
	return &Config{
		General: GeneralConfig{
			Log:  "info",
			Addr: ":8080",
			Mode: "tui",
		},
		Board: BoardConfig{
			Statuses: []string{"backlog", "in_progress", "review", "done"},
			Prefix:   "",
		},
		Agent: AgentConfig{
			Default: "opencode",
		},
		TUI: TUIConfig{
			Theme:  "agentboard",
			Layout: "compact",
		},
		LLM: LLMConfig{
			RequireApproval: true,
		},
		DB: DBConfig{},
		MCP: MCPConfig{
			NPMPath:  "npm",
			NodePath: "node",
			Servers: map[string]MCPServerConfig{
				"contextcarry": {
					Enabled: true,
					Command: "npx",
					Args:    []string{"-y", "@thisisayande/contextcarry-mcp"},
				},
			},
		},
	}
}
