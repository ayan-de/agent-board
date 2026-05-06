# AgentBoard

[![Go Version](https://img.shields.io/github/go-mod/go-version/ayan-de/agent-board)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()

**AgentBoard** is a terminal-based Kanban board for orchestrating and managing AI coding agents. It provides a visual development workflow for modern software engineering, bridging project management with automated code generation.

<img width="1863" height="450" alt="AgentBoard TUI Mockup" src="https://github-production-user-asset-6210df.s3.amazonaws.com/59247285/579415351-95cd3ac9-d3a4-4c49-91b6-dff6b6c4988a.png?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAVCODYLSA53PQK4ZA%2F20260416%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20260416T174544Z&X-Amz-Expires=300&X-Amz-Signature=a092b147f626f2c61c0bd7814d445ba3729ade629ac0876570d89c4a4657fdea&X-Amz-SignedHeaders=host&response-content-type=image%2Fpng" />

---

## Key Features

- **Modern Kanban TUI**: Terminal UI built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), featuring glassmorphism-inspired borders and smooth animations.
- **Agent Orchestration**: Spawn and manage agents like **Claude Code**, **OpenCode**, and **Cursor** via tmux panes or embedded PTY.
- **MCP Native**: Integrated Model Context Protocol support via `@thisisayande/contextcarry-mcp` for agent memory persistence.
- **Persistent Storage**: SQLite backend with automatic migrations.
- **AI Proposal Flow**: Coordinator LLM creates worker prompts, with approval gating before execution.
- **LangChain Go Integration**: Provider registry supporting OpenAI, Ollama, Claude, and Zai.

---

## 🌐 Marketing Website

For more information about AgentBoard, including feature showcases, screenshots, and project updates, visit the official website:

**https://agentboard.ayande.xyz/**

---

## Getting Started

### Prerequisites

- **Go**: 1.21 or higher
- **git**: For cloning the repository
- **tmux**: Required for tmux-mode agent spawning (Linux/macOS)
- **Node.js/npm**: Required for MCP server integrations

### Installation

#### From Binary (All Platforms)

Run the automated installer:

```bash
curl -sSL https://raw.githubusercontent.com/ayan-de/agent-board/main/install.sh | bash
```

This will detect your OS and architecture, install dependencies, and place the binary in `~/.local/bin`.

#### From Source
Ensure [Go](https://go.dev/doc/install) 1.21+ is installed.

```bash
git clone https://github.com/ayan-de/agent-board.git
cd agent-board
go build -o agentboard ./cmd/agentboard

# Add to PATH (optional)
export PATH="$HOME/.local/bin:$PATH"

# Initialize configuration
./agentboard init
```

#### Platform-Specific Notes

| Platform | Package Manager | Install Command |
|----------|-----------------|-----------------|
| Linux (apt) | `apt-get install golang git tmux npm` | Binary or source |
| macOS (brew) | `brew install go git tmux npm node` | Binary or source |
| Android (Termux) | `pkg install golang git tmux nodejs` | Binary or source |

### Basic Usage

Start the interactive Kanban board:

```bash
./agentboard
```

Or if installed via binary:
```bash
agentboard
```

### Keybindings

| Key | Action |
|-----|--------|
| `h/l` or `←/→` | Move between Kanban columns |
| `j/k` or `↑/↓` | Move between tickets |
| `Enter` | Open ticket detail view |
| `a` | Add a new ticket in the active column |
| `d` | Delete selected ticket |
| `1-4` | Jump to a specific column |
| `?` | Toggle help view |
| `i` | Toggle agent dashboard |
| `:` | Open command palette |
| `q` | Quit with confirmation |
| `Esc` | Return to board from other views |

### Ticket View

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Move between fields |
| `e` | Edit the selected editable field |
| `s` | Cycle ticket status |
| `a` | Open agent selection |
| `p` | Approve pending proposal |
| `r` | Start approved run |
| `Esc` | Cancel edit or return to board |

### Dashboard

| Key | Action |
|-----|--------|
| `r` | Re-run agent detection |
| `Esc` | Return to board |

---

## Configuration

Configuration is loaded from `~/.agentboard/config.toml` (global) or `<project>/.agentboard/config.toml` (project-level).

### Example Configuration

```toml
[general]
log = "info"
tmux = true

[db]
path = ".agentboard/agentboard.db"

[llm]
provider = "openai"
model = "gpt-4"
api_key = "${OPENAI_API_KEY}"

[llm.coordinator]
model = "gpt-4"

[llm.summarizer]
model = "gpt-4"

[tui]
theme = "nord"

[mcp.contextcarry]
command = "npx"
args = ["-y", "@thisisayande/contextcarry-mcp"]
```

### Environment Variables

Override config values:

| Variable | Config Key |
|----------|------------|
| `AGENTBOARD_LOG` | `general.log` |
| `AGENTBOARD_TMUX` | `general.tmux` |
| `AGENTBOARD_DB` | `db.path` |
| `AGENTBOARD_LLM_PROVIDER` | `llm.provider` |
| `AGENTBOARD_LLM_MODEL` | `llm.model` |
| `AGENTBOARD_LLM_API_KEY` | `llm.api_key` |
| `AGENTBOARD_LLM_BASE_URL` | `llm.base_url` |
| `AGENTBOARD_LLM_COORDINATOR_MODEL` | `llm.coordinator_model` |
| `AGENTBOARD_LLM_SUMMARIZER_MODEL` | `llm.summarizer_model` |
| `AGENTBOARD_NPM_PATH` | `mcp.npm_path` |
| `AGENTBOARD_NODE_PATH` | `mcp.node_path` |

---

## Themes

AgentBoard ships with builtin themes. User themes can be placed in `~/.agentboard/themes/` or `<project>/.agentboard/themes/` as JSON files.

### Available Themes

| Theme | Description |
|-------|-------------|
| `agentboard` | Default AgentBoard theme |
| `catppuccin` | Soft, warm pastel |
| `dracula` | Dark with vibrant colors |
| `gruvbox` | Retro groove terminal |
| `matrix` | Cyberpunk green-on-black |
| `nord` | Clean, arctic palette |
| `tokyonight` | Tokyo night city lights |

---

## Architecture

```
agent-board/
├── cmd/agentboard/
│   └── main.go             # TUI entrypoint
├── internal/
│   ├── tui/                # Bubble Tea application
│   ├── store/              # SQLite persistence
│   ├── config/             # Config loading and agent detection
│   ├── theme/              # Theme registry
│   ├── keybinding/         # Keymap and action resolution
│   ├── llm/                # LangChain Go provider registry
│   ├── orchestrator/       # Agent lifecycle management
│   ├── pty/                # PTY agent configurations
│   ├── prompt/             # LLM prompt templates
│   ├── mcp/                # MCP manager and adapters
│   └── mcpclient/          # MCP stdio client wrapper
├── docs/                   # Design notes
└── AGENTS.md               # Project state documentation
```

---

## Development

### Prerequisites

- **Go**: 1.21+
- **tmux**: Required for agent spawning
- **Node.js/npm**: Required for MCP server integrations

### Commands

```bash
# Build
go build -o agentboard ./cmd/agentboard

# Run tests
go test ./...

# Run vet
go vet ./...
```

### Debug Mode

```bash
AGENTBOARD_LOG=debug ./agentboard
```

3.  **Running Tests**:
    We follow a strict TDD discipline. Ensure all tests pass before submitting changes.
    ```bash
    go test ./...
    ```

4.  **Database Migrations**:
    Migrations are handled automatically on startup, but you can inspect the schema in `internal/store/migrations.go`.

---

## 🤝 Contributing

We welcome contributions! Please follow our workflow to keep the board clean and efficient:

1.  **Fork** the repository and create your feature branch.
2.  **Write Tests First**: We strictly follow Red-Green-Refactor. No implementation without a corresponding `_test.go` file.
3.  **Lint your code**: Run `go vet ./...` to ensure idiomatic Go.
4.  **Document**: Update `AGENT.md` if you introduce architectural changes.
5.  **Submit PR**: Ensure your commit messages follow [Conventional Commits](https://www.conventionalcommits.org/).

### Architecture Overview

- `cmd/`: Application entrypoints.
- `internal/tui/`: Bubble Tea models and components.
- `internal/orchestrator/`: Agent lifecycle and process management.
- `internal/store/`: SQLite persistence layer.
- `internal/mcp/`: Protocol clients for context management.

---

## 📜 License

Distributed under the MIT License. See `LICENSE` for more information.

---

<p align="center">
  Built with ❤️ for the AI-First Engineering community.
</p>

---

## Contributing

1. Fork and create a feature branch from `main`
2. Write tests first (TDD discipline)
3. Make tests pass
4. Refactor while keeping tests green
5. Run `go test ./...` and `go vet ./...`
6. Update `AGENTS.md` for architectural changes
7. Submit a PR with [Conventional Commits](https://www.conventionalcommits.org/)

---

## License

MIT License. See `LICENSE` for details.