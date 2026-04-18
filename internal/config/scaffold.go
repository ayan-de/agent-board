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

[board]
# prefix = "AGB-"  # Ticket ID prefix. Default: first 3 letters of project name.

[tui]
theme = "default"
layout = "compact"

# [tui.keybindings]
# next_column = "l"
# prev_column = "h"
# next_ticket = "j"
# prev_ticket = "k"
# open_ticket = "enter"
# add_ticket = "a"
# delete_ticket = "d"
# start_agent = "s"
# stop_agent = "x"
# refresh = "r"
# show_help = "?"
# toggle_focus = "tab"
# prev_focus = "shift+tab"
# jump_col1 = "1"
# jump_col2 = "2"
# jump_col3 = "3"
# jump_col4 = "4"
# go_to_ticket_prefix = "g"

[agent]
default = "opencode"

[llm]
# provider = "openai"    # openai, ollama, claude, zai
# model = "gpt-4o-mini"
# api_key = ""
# base_url = ""
# coordinator_model = "" # defaults to model if empty
# summarizer_model = ""  # defaults to model if empty
# require_approval = true

[mcp]
npm_path = "npm"
node_path = "node"
`

	return os.WriteFile(path, []byte(defaultContent), 0644)
}
