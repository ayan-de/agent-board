# 🤖 AgentBoard

[![Go Version](https://img.shields.io/github/go-mod/go-version/ayan-de/agent-board)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()

**AgentBoard** is a premium, terminal-based Kanban board designed to orchestrate and manage AI coding agents. It provides a visual development workflow for modern software engineering, bridging the gap between project management and automated code generation.

<img width="1863" height="450" alt="AgentBoard TUI Mockup" src="https://github-production-user-asset-6210df.s3.amazonaws.com/59247285/579415351-95cd3ac9-d3a4-4c49-91b6-dff6b6c4988a.png?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAVCODYLSA53PQK4ZA%2F20260416%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20260416T174544Z&X-Amz-Expires=300&X-Amz-Signature=a092b147f626f2c61c0bd7814d445ba3729ade629ac0876570d89c4a4657fdea&X-Amz-SignedHeaders=host&response-content-type=image%2Fpng" />

---

## ✨ Key Features

- **📊 Modern Kanban TUI**: A sleek Terminal User Interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), featuring glassmorphism-inspired borders and smooth animations.
- **🤖 Agent Orchestration**: Seamlessly spawn and manage agents like **Claude Code**, **OpenCode**, and **Cursor**.
- **🪟 tmux Integration**: Run agents in their own tmux panes or embedded PTY views for maximum flexibility.
- **🔌 MCP Native**: Integrated support for Model Context Protocol (MCP) servers like `ContextCarry` and `SessionCarry` to preserve agent memory.
- **💾 Persistent Storage**: Powered by a robust SQLite backend with automatic migrations.
- **🌐 Dual Mode**: Switch between a standalone TUI and a headless API server for future frontend integrations.

---

## 🌐 Marketing Website

For more information about AgentBoard, including feature showcases, screenshots, and project updates, visit the official website:

**https://agentboard.ayande.xyz/**

---

## 🚀 Getting Started

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

```bash
# Clone the repository
git clone https://github.com/ayan-de/agent-board.git
cd agent-board

# Build the binary
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
| `j/k` or `↑/↓` | Navigate tickets |
| `Enter` | View/Edit ticket details |
| `a` | Create a new ticket |
| `d` | Delete selected ticket |
| `s` | Cycle ticket status |
| `i` | Toggle Agent Dashboard |
| `p` | Open Command Palette |
| `?` | Show Help |
| `q` | Exit |

---

## 🎨 Themes

AgentBoard ships with several builtin themes. You can also add your own JSON themes.

### Available Themes

| Theme | Description |
|-------|-------------|
| `agentboard` | The default AgentBoard theme |
| `catppuccin` | Soft, warm pastel theme |
| `dracula` | Dark theme with vibrant colors |
| `gruvbox` | Retro groove terminal theme |
| `matrix` | Cyberpunk green-on-black |
| `nord` | Clean, arctic color palette |
| `tokyonight` | Tokyo night city lights theme |

### Using Themes

Themes are configured in your project or global `config.toml`:

```toml
[tui]
theme = "dracula"
```

User themes can be placed in `~/.agentboard/themes/` or `<project>/.agentboard/themes/` as JSON files.

---

## 🛠️ Local Development

### Prerequisites

- **Go**: 1.21 or higher
- **tmux**: Required for tmux-mode agent spawning
- **Node.js/npm**: Required for MCP server integrations

### Development Workflow

1.  **Environment Setup**:
    Copy `.env.example` to `.env` and configure your LLM providers if using decomposition features.

2.  **Running in Debug Mode**:
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
