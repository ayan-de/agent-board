package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Config struct {
	General GeneralConfig
	Board   BoardConfig
	Agent   AgentConfig
	TUI     TUIConfig
	LLM     LLMConfig
	DB      DBConfig
	MCP     MCPConfig

	ConfigPath        string
	ProjectConfigPath string
	ProjectName       string
}

func LoadFromDir(baseDir, projectName string) (*Config, error) {
	cfg := SetDefaults()
	cfg.ProjectName = projectName

	if err := EnsureDirs(baseDir, projectName); err != nil {
		return nil, err
	}

	globalPath := filepath.Join(baseDir, "config.toml")
	_ = WriteDefaultConfig(globalPath)
	if err := loadFromFile(cfg, globalPath); err != nil {
		return nil, err
	}

	projPath := filepath.Join(baseDir, "projects", projectName, "config.toml")
	_ = WriteDefaultConfig(projPath)
	if err := loadFromFile(cfg, projPath); err != nil {
		return nil, err
	}

	if cfg.DB.Path == "" {
		cfg.DB.Path = filepath.Join(baseDir, "projects", projectName, "board.db")
	}

	if cfg.Board.Prefix == "" {
		cfg.Board.Prefix = DefaultPrefix(projectName)
	}

	cfg.ConfigPath = globalPath
	cfg.ProjectConfigPath = projPath

	applyEnvVars(cfg)

	return cfg, nil
}

func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("config.load: home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".agentboard")

	remote := getGitRemote()
	fallback := filepath.Base(getWorkingDir())
	projectName := ExtractProjectName(remote, fallback)

	return LoadFromDir(baseDir, projectName)
}

func getGitRemote() string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func getWorkingDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return dir
}

func GetBaseDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".agentboard"
	}
	return filepath.Join(homeDir, ".agentboard")
}
