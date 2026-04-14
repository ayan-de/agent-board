package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func EnsureDirs(baseDir, projectName string) error {
	dirs := []string{
		baseDir,
		filepath.Join(baseDir, "themes"),
		filepath.Join(baseDir, "projects"),
		filepath.Join(baseDir, "projects", projectName),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("config.scaffold: creating %s: %w", dir, err)
		}
	}

	return nil
}

func WriteDefaultConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("config.scaffold: creating %s: %w", dir, err)
	}

	defaultContent := `# AgentBoard Configuration
# Edit this file to customize your setup.

[general]
log = "info"
addr = ":8080"
mode = "tui"
tmux = "auto"

[tui]
theme = "default"
layout = "compact"

[agent]
default = "opencode"

[llm]
# provider = "openai"
# model = "gpt-4o"
# api_key = ""
# base_url = ""

[mcp]
npm_path = "npm"
node_path = "node"
`

	return os.WriteFile(path, []byte(defaultContent), 0644)
}
