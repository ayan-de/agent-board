# Config Package Design

## Overview

The config package (`internal/config/`) manages all configuration for AgentBoard. It uses a layered resolution system: hardcoded defaults → global TOML → project TOML → environment variables. On first run, it auto-scaffolds the `~/.agentboard/` directory structure.

## Directory Structure

```
~/.agentboard/
├── config.toml                  ← global config (theme, log level, agent defaults)
├── themes/
│   ├── catppuccin.toml
│   └── dracula.toml
└── projects/
    ├── my-web-app/              ← derived from git repo name or directory name
    │   ├── config.toml          ← project-specific (custom statuses, agents)
    │   └── board.db             ← SQLite DB for this project
    └── agent-board/
        ├── config.toml
        └── board.db
```

## Config Structs

```go
type Config struct {
    Board  BoardConfig
    Agent  AgentConfig
    TUI    TUIConfig
    LLM    LLMConfig
    DB     DBConfig
    MCP    MCPConfig
}

type BoardConfig struct {
    Statuses []string  // default: ["backlog", "in_progress", "review", "done"]
}

type AgentConfig struct {
    Default string     // default: "opencode"
}

type TUIConfig struct {
    Theme  string      // default: "default"
    Layout string      // default: "compact"
}

type LLMConfig struct {
    Provider string    // "openai", "anthropic", "ollama"
    Model    string
    APIKey   string
    BaseURL  string
}

type DBConfig struct {
    Path string        // default: ~/.agentboard/projects/<name>/board.db
}

type MCPConfig struct {
    NPMPath  string    // default: "npm"
    NodePath string    // default: "node"
}
```

## Layered Resolution

Configuration is resolved in 4 layers, each overriding the previous:

```
1. SetDefaults()         — hardcoded defaults in Go
2. LoadGlobalConfig()    — ~/.agentboard/config.toml
3. LoadProjectConfig()   — ~/.agentboard/projects/<name>/config.toml
4. ApplyEnvVars()        — AGENTBOARD_* environment variables override everything
```

Each step overwrites only fields that are explicitly set. Absent fields retain the previous layer's value.

### Environment Variables

| Variable | Overrides | Default |
|----------|-----------|---------|
| `AGENTBOARD_CONFIG` | Global config path | `~/.agentboard/config.toml` |
| `AGENTBOARD_DB` | `DB.Path` | `~/.agentboard/projects/<name>/board.db` |
| `AGENTBOARD_LOG` | Log level | `info` |
| `AGENTBOARD_ADDR` | API server bind address | `:8080` |
| `AGENTBOARD_MODE` | Startup mode | `tui` |
| `AGENTBOARD_TMUX` | tmux usage | `auto` |
| `AGENTBOARD_LLM_PROVIDER` | `LLM.Provider` | — |
| `AGENTBOARD_LLM_MODEL` | `LLM.Model` | — |
| `AGENTBOARD_LLM_API_KEY` | `LLM.APIKey` | — |
| `AGENTBOARD_LLM_BASE_URL` | `LLM.BaseURL` | — |
| `AGENTBOARD_NPM_PATH` | `MCP.NPMPath` | `npm` |
| `AGENTBOARD_NODE_PATH` | `MCP.NodePath` | `node` |
| `NO_COLOR` | Disables colored output | — |

## Project Auto-Detection

When `agentboard` starts:

1. Run `git remote get-url origin` → extract repo name (e.g., `agent-board` from `github.com/ayan-de/agent-board`)
2. Fallback: use current working directory basename
3. Slugify the name (lowercase, replace non-alphanumeric with `-`)
4. Use `~/.agentboard/projects/<slug>/` as the project directory

## Auto-Scaffold on First Run

When `agentboard` starts and `~/.agentboard/` does not exist:

1. Create `~/.agentboard/`
2. Create `~/.agentboard/themes/`
3. Create `~/.agentboard/projects/`
4. Write `~/.agentboard/config.toml` with commented defaults
5. Detect project → create `~/.agentboard/projects/<slug>/`
6. Write project `config.toml` with defaults
7. SQLite init (Phase 1.2) will create `board.db` on first query

No separate `init` command needed — the app self-scaffolds on first run.

## TOML Format

### Global config (`~/.agentboard/config.toml`)

```toml
[tui]
theme = "default"
layout = "compact"

[agent]
default = "opencode"

[llm]
provider = "openai"
model = "gpt-4o"

[mcp]
npm_path = "npm"
node_path = "node"
```

### Project config (`~/.agentboard/projects/<name>/config.toml`)

```toml
[board]
statuses = ["backlog", "in_progress", "review", "done"]

[agent]
default = "claude-code"

[tui]
theme = "catppuccin"
```

## Package Files

| File | Responsibility |
|------|---------------|
| `config.go` | `Config` struct, `Load()` function orchestrating all layers |
| `defaults.go` | `SetDefaults()` — returns `Config` with hardcoded defaults |
| `detection.go` | `DetectProjectName()` — git remote or directory basename |
| `scaffold.go` | `EnsureDirs()` — auto-scaffold `~/.agentboard/` tree |

## Testing Strategy

Each layer is tested independently:

1. **`TestSetDefaults`** — verify all fields have sensible defaults
2. **`TestLoadGlobalConfig`** — parse TOML, verify overrides (use temp file)
3. **`TestLoadProjectConfig`** — parse project TOML, verify overrides
4. **`TestApplyEnvVars`** — set env vars, verify they override TOML values
5. **`TestDetectProjectName`** — mock git remote, verify name extraction and slugification
6. **`TestEnsureDirs`** — verify directory creation (use temp dir)
7. **`TestLoadFullResolution`** — integration test: defaults → global → project → env

All tests use `t.TempDir()` for filesystem operations. No real `~/.agentboard/` touched.

## Error Handling

- Missing global config → not an error, use defaults + project config
- Missing project config → not an error, use defaults + global config
- Invalid TOML → return wrapped error: `fmt.Errorf("config.load: parsing %s: %w", path, err)`
- Missing env vars → not an error, skip that layer
- Invalid env var values → return wrapped error with the var name

## Dependencies

- `github.com/BurntSushi/toml` — TOML parsing (already in AGENTS.md)
- `os/exec` — git remote detection
- `os` — env var reading
- `path/filepath` — path manipulation
