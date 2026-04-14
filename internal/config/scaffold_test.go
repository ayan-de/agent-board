package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestEnsureDirs(t *testing.T) {
	baseDir := t.TempDir()
	projectName := "test-project"

	err := EnsureDirs(baseDir, projectName)
	if err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}

	expectedDirs := []string{
		baseDir,
		filepath.Join(baseDir, "themes"),
		filepath.Join(baseDir, "projects"),
		filepath.Join(baseDir, "projects", projectName),
	}

	for _, dir := range expectedDirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("dir %q does not exist: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q is not a directory", dir)
		}
	}
}

func TestEnsureDirsIdempotent(t *testing.T) {
	baseDir := t.TempDir()

	err := EnsureDirs(baseDir, "proj")
	if err != nil {
		t.Fatalf("first EnsureDirs: %v", err)
	}
	err = EnsureDirs(baseDir, "proj")
	if err != nil {
		t.Fatalf("second EnsureDirs: %v", err)
	}
}

func TestWriteDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	err := WriteDefaultConfig(cfgPath)
	if err != nil {
		t.Fatalf("WriteDefaultConfig: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	content := string(data)
	if len(content) == 0 {
		t.Fatal("config file is empty")
	}

	cfg := SetDefaults()
	if _, err := toml.Decode(string(data), cfg); err != nil {
		t.Fatalf("decode default config: %v", err)
	}
	if cfg.TUI.Theme != "default" {
		t.Errorf("decoded TUI.Theme = %q, want %q", cfg.TUI.Theme, "default")
	}
}

func TestWriteDefaultConfigDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	existing := []byte(`[tui]
theme = "catppuccin"
`)
	if err := os.WriteFile(cfgPath, existing, 0644); err != nil {
		t.Fatalf("write existing: %v", err)
	}

	err := WriteDefaultConfig(cfgPath)
	if err != nil {
		t.Fatalf("WriteDefaultConfig: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	if string(data) != string(existing) {
		t.Error("WriteDefaultConfig should not overwrite existing config")
	}
}
