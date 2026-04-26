# AgentBoard Install Script Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a cross-platform install script that verifies prerequisites (Go, tmux), detects the LLM provider environment, builds the binary, and prints setup instructions so users can run `agentboard` from the project root.

**Architecture:** A single `install.sh` (plus `install.ps1` for Windows) that runs pre-flight checks, installs missing dependencies via the system's package manager, detects available tools, builds the Go binary, and generates a helpful `~/.agentboard/config.toml` with auto-detected values. The script is idempotent and supports CI/CI environments via env vars.

**Tech Stack:** Bash 5+ (Linux/macOS/Termux), PowerShell 7+ (Windows), Go 1.21+, tmux, optional: Homebrew (macOS), apt/pacman/zypper (Linux).

---

## File Map

```
docs/superpowers/plans/2026-04-26-publish-distribute.md  — THIS PLAN

Scripts created:
  scripts/install.sh           — Cross-platform install (Linux, macOS, Termux)
  scripts/install.ps1          — Windows PowerShell install
  scripts/install_detect.go    — Go-based detection helper (reuses config/detection)

Config created on install:
  ~/.agentboard/config.toml    — Auto-generated with detected values

Binary placed:
  /usr/local/bin/agentboard    — System-wide (if running as root or with sudo)
  ./agentboard                  — Project root (default for dev)
```

---

## Task 1: Create `scripts/install.sh` — Core Install Script

**Files:**
- Create: `scripts/install.sh`

- [ ] **Step 1: Write the install.sh script header and helpers**

```bash
#!/usr/bin/env bash
# agentboard install script — supports Linux, macOS, Termux

set -euo pipefail

VERSION="${VERSION:-"latest"}"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
CONFIG_DIR="${CONFIG_DIR:-${HOME}/.agentboard}"
FORCE="${FORCE:-"false"}"

info() { echo "[INFO] $*" >&2; }
warn() { echo "[WARN] $*" >&2; }
err()  { echo "[ERROR] $*" >&2; exit 1; }

supports_color() {
    if [[ -z "${TERM:-}" ]] || [[ "${TERM}" == "dumb" ]]; then return 1; fi
    command -v tput >/dev/null 2>&1 && [[ $(tput colors 2>/dev/null || echo 0) -ge 8 ]]
}

header() {
    if supports_color; then
        echo -e "\033[1;36m==> $*\033[0m" >&2
    else
        echo "==> $*" >&2
    fi
}

detect_os() {
    case "$(uname -s)" in
        Linux*)
            if [[ -f /etc/os-release ]]; then
                . /etc/os-release
                echo "${ID:-linux}"
            else
                echo "linux"
            fi
            ;;
        Darwin*)  echo "darwin" ;;
        *-android|Android) echo "termux" ;;
        *)        echo "unknown" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "x86_64" ;;
        aarch64|arm64) echo "arm64" ;;
        armv7l|arm)    echo "arm" ;;
        *)             echo "$(uname -m)" ;;
    esac
}
```

- [ ] **Step 2: Add dependency checking functions**

```bash
check_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        return 1
    fi
    return 0
}

get_version() {
    "$1" version 2>/dev/null || "$1" --version 2>/dev/null | head -1 || echo "unknown"
}

check_go() {
    header "Checking Go"
    if check_cmd go; then
        local go_version
        go_version=$(go version 2>/dev/null | grep -oP 'go\d+\.\d+' || echo "unknown")
        info "Go found: $go_version"
        return 0
    else
        err "Go is not installed. Install from https://go.dev/dl/"
    fi
}

check_tmux() {
    header "Checking tmux"
    if check_cmd tmux; then
        info "tmux found: $(get_version tmux)"
        return 0
    else
        warn "tmux is not installed."
        return 1
    fi
}

install_tmux_linux() {
    local os="$1"
    header "Installing tmux"

    case "$os" in
        ubuntu|debian|linuxmint|pop)
            sudo apt-get update && sudo apt-get install -y tmux
            ;;
        fedora)
            sudo dnf install -y tmux
            ;;
        arch|manjaro|endeavouros)
            sudo pacman -S --noconfirm tmux
            ;;
        alpine)
            sudo apk add tmux
            ;;
        opensuse*|suse)
            sudo zypper install -y tmux
            ;;
        *)
            err "Unsupported distro: $os. Please install tmux manually: https://github.com/tmux/tmux"
            ;;
    esac
    info "tmux installed successfully."
}

install_tmux_darwin() {
    header "Installing tmux on macOS"
    if check_cmd brew; then
        brew install tmux
    else
        err "Homebrew not found. Install tmux via MacPorts: sudo port install tmux"
    fi
    info "tmux installed successfully."
}

install_tmux_termux() {
    header "Installing tmux on Termux"
    pkg install tmux
    info "tmux installed successfully."
}
```

- [ ] **Step 3: Add LLM provider detection**

```bash
detect_llm_provider() {
    header "Detecting LLM Provider"

    local provider=""
    local model=""

    if [[ -n "${AGENTBOARD_LLM_PROVIDER:-}" ]]; then
        provider="$AGENTBOARD_LLM_PROVIDER"
        info "Using provider from env: $provider"
    elif check_cmd ollama && ollama list >/dev/null 2>&1; then
        provider="ollama"
        model=$(ollama list 2>/dev/null | grep -E '^[a-zA-Z]' | head -1 | awk '{print $1}' || echo "llama3")
        info "Ollama detected — will use model: $model"
    elif [[ -n "${ANTHROPIC_API_KEY:-}" ]] || check_cmd claude 2>/dev/null; then
        provider="anthropic"
        model="claude-sonnet-4-20250514"
        info "Anthropic detected."
    elif [[ -n "${OPENAI_API_KEY:-}" ]]; then
        provider="openai"
        model="gpt-4o"
        info "OpenAI detected."
    else
        warn "No LLM provider detected. Set AGENTBOARD_LLM_PROVIDER env var or install ollama."
        provider="ollama"
        model="llama3"
    fi

    echo "$provider:$model"
}
```

- [ ] **Step 4: Add agent detection and Node.js/npm detection**

```bash
detect_agents() {
    header "Detecting available AI agents"

    local agents=()

    for agent in claude opencode cursor codex; do
        if check_cmd "$agent"; then
            agents+=("$agent")
            info "  Found: $agent"
        fi
    done

    if [[ ${#agents[@]} -eq 0 ]]; then
        info "  No AI agents found in PATH. Install claude-code, opencode, or cursor."
        echo "[]"
    else
        printf '%s\n' "${agents[@]}" | jq -R . | jq -s .
    fi
}

check_node_npm() {
    header "Checking Node.js/npm"
    if check_cmd node && check_cmd npm; then
        info "Node.js: $(node --version), npm: $(npm --version)"
    else
        warn "Node.js/npm not found. MCP features may not work."
        info "  Install: https://nodejs.org/"
    fi
}
```

- [ ] **Step 5: Add config generation and build steps**

```bash
generate_config() {
    header "Generating config at ${CONFIG_DIR}/config.toml"
    local provider_model="$1"
    local provider="${provider_model%%:*}"
    local model="${provider_model#*:}"
    local agents_json="${2:-[]}"
    local os="${3:-linux}"

    mkdir -p "$CONFIG_DIR"

    cat > "${CONFIG_DIR}/config.toml" <<EOF
# AgentBoard auto-generated config
# Generated on $(date -u +"%Y-%m-%dT%H:%M:%SZ")

[general]
project_name = "$(basename "$(pwd)")"
log = "info"

[db]
path = "\${config_dir}/board.db"

[llm]
provider = "$provider"
model = "$model"

[agents]
detected = $agents_json

[mcp]
# npm_path = "npm"
# node_path = "node"
EOF

    info "Config written to ${CONFIG_DIR}/config.toml"
}

build_binary() {
    header "Building agentboard"

    local install_path="${INSTALL_DIR}/agentboard"

    if [[ ! -d "$INSTALL_DIR" ]]; then
        mkdir -p "$INSTALL_DIR"
    fi

    if [[ "${VERSION:-}" == "latest" ]]; then
        go build -o "$install_path" ./cmd/agentboard
    else
        go build -o "$install_path" -ldflags="-X main.version=$VERSION" ./cmd/agentboard
    fi

    chmod +x "$install_path"
    info "Binary built and installed to $install_path"
}

add_to_path_check() {
    header "Checking PATH"
    if [[ ":$PATH:" == *":${INSTALL_DIR}:"* ]]; then
        info "${INSTALL_DIR} is in PATH"
    else
        warn "${INSTALL_DIR} is NOT in PATH. Add this to your shell config:"
        warn "  export PATH=\"\${HOME}/.local/bin:\$PATH\""
    fi
}
```

- [ ] **Step 6: Add main function and CI mode**

```bash
main() {
    local os arch

    echo ""
    header "AgentBoard Installer"
    info "Version: ${VERSION:-"dev build"}"
    info "OS: $(uname -s), Arch: $(uname -m)"
    echo ""

    os=$(detect_os)
    arch=$(detect_arch)

    check_go

    local install_tmux="false"
    if ! check_tmux; then
        if [[ "${CI:-}" == "true" ]] || [[ "${AGENTBOARD_SKIP_TMUX:-}" == "true" ]]; then
            warn "Skipping tmux install (CI mode)"
        else
            read -p "Install tmux now? [y/N] " -n 1 -r reply || reply="n"
            echo ""
            if [[ "$reply" =~ ^[Yy]$ ]]; then
                install_tmux="true"
            fi
        fi
    fi

    if [[ "$install_tmux" == "true" ]]; then
        case "$os" in
            darwin)  install_tmux_darwin ;;
            ubuntu|debian|linuxmint|pop|fedora|arch|manjaro|endeavouros|alpine|opensuse*|suse) install_tmux_linux "$os" ;;
            termux)  install_tmux_termux ;;
            *)       err "Cannot auto-install tmux on $os. Install manually." ;;
        esac
    fi

    check_node_npm

    local provider_model
    provider_model=$(detect_llm_provider)
    local provider="${provider_model%%:*}"
    local model="${provider_model#*:}"

    local agents_json
    agents_json=$(detect_agents)

    check_node_npm

    generate_config "$provider_model" "$agents_json" "$os"

    build_binary

    add_to_path_check

    echo ""
    header "Installation complete!"
    info "Run agentboard from your project directory:"
    info "  cd /path/to/your/project"
    info "  agentboard"
    info ""
    info "Or from the agent-board repo root:"
    info "  ./agentboard"
    info ""
    info "First-time setup: Edit ~/.agentboard/config.toml to configure your LLM API keys."
}

main "$@"
```

---

## Task 2: Create `scripts/install.ps1` — Windows Install Script

**Files:**
- Create: `scripts/install.ps1`

- [ ] **Step 1: Write PowerShell install script**

```powershell
#!/usr/bin/env pwsh
# AgentBoard Windows Installer

param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:LOCALAPPDATA\agentboard",
    [switch]$SkipTmux,
    [switch]$Force
)

$ErrorActionPreference = "Stop"

function Write-Info { Write-Host "[INFO] $args" -ForegroundColor Cyan }
function Write-Warn { Write-Host "[WARN] $args" -ForegroundColor Yellow }
function Write-Err  { Write-Host "[ERROR] $args" -ForegroundColor Red; exit 1 }
function Write-Header { Write-Host "`n==> $args" -ForegroundColor Magenta }

Write-Header "AgentBoard Windows Installer"

# Check Go
Write-Header "Checking Go"
try {
    $goVersion = go version 2>$null
    if (-not $goVersion) { throw "go not in PATH" }
    Write-Info "Go found: $goVersion"
} catch {
    Write-Err "Go is not installed. Download from https://go.dev/dl/"
}

# Check tmux
Write-Header "Checking tmux"
$tmuxFound = $false
try {
    if (Get-Command tmux -ErrorAction SilentlyContinue) {
        Write-Info "tmux found"
        $tmuxFound = $true
    }
} catch { }

if (-not $tmuxFound -and -not $SkipTmux) {
    Write-Warn "tmux is not installed. Install via: choco install tmux  OR  winget install tmux"
}

# Check Node.js
Write-Header "Checking Node.js/npm"
try {
    $nodeVersion = node --version 2>$null
    $npmVersion = npm --version 2>$null
    if ($nodeVersion) {
        Write-Info "Node.js: $nodeVersion, npm: $npmVersion"
    } else {
        Write-Warn "Node.js not found. MCP features may not work."
    }
} catch {
    Write-Warn "Node.js not found. MCP features may not work."
}

# Detect LLM Provider
Write-Header "Detecting LLM Provider"
$provider = "ollama"
$model = "llama3"

if ($env:AGENTBOARD_LLM_PROVIDER) {
    $provider = $env:AGENTBOARD_LLM_PROVIDER
    Write-Info "Using provider from env: $provider"
} elseif (Get-Command ollama -ErrorAction SilentlyContinue) {
    $provider = "ollama"
    Write-Info "Ollama detected"
} elseif ($env:ANTHROPIC_API_KEY) {
    $provider = "anthropic"
    $model = "claude-sonnet-4-20250514"
    Write-Info "Anthropic API key detected"
} elseif ($env:OPENAI_API_KEY) {
    $provider = "openai"
    $model = "gpt-4o"
    Write-Info "OpenAI API key detected"
} else {
    Write-Warn "No LLM provider detected. Set AGENTBOARD_LLM_PROVIDER env var."
}

# Create config
Write-Header "Generating config"
$configDir = "$env:USERPROFILE\.agentboard"
$null = New-Item -ItemType Directory -Force -Path $configDir

$projectName = Split-Path -Leaf (Get-Location)

@"
# AgentBoard auto-generated config
# Generated on $(Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")

[general]
project_name = "$projectName"
log = "info"

[db]
path = "`${config_dir}\board.db"

[llm]
provider = "$provider"
model = "$model"
"@ | Set-Content -Path "$configDir\config.toml" -Encoding UTF8

Write-Info "Config written to $configDir\config.toml"

# Build binary
Write-Header "Building agentboard"
$binaryPath = Join-Path $InstallDir "agentboard.exe"
$null = New-Item -ItemType Directory -Force -Path $InstallDir

if ($Version -eq "latest") {
    go build -o $binaryPath .\cmd\agentboard
} else {
    go build -o $binaryPath -ldflags="-X main.version=$Version" .\cmd\agentboard
}

Write-Info "Binary built at $binaryPath"

Write-Header "Installation complete!"
Write-Info "Run: $binaryPath"
Write-Info "Or add $InstallDir to your PATH."
```

---

## Task 3: Create `scripts/install_detect.go` — Go Detection Helper

**Files:**
- Create: `scripts/install_detect.go`

- [ ] **Step 1: Write Go-based system detection utility**

```go
// install_detect.go — reuses AgentBoard's internal detection logic
// to print system info for the install script.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func main() {
	fmt.Printf("OS: %s\n", runtime.GOOS)
	fmt.Printf("Arch: %s\n", runtime.GOARCH)
	fmt.Printf("GOARCH: %s\n", runtime.GOARCH)
	fmt.Printf("GOOS: %s\n", runtime.GOOS)

	detect := func(name string, args ...string) bool {
		cmd := exec.Command(name, args...)
		cmd.Stdout = nil
		cmd.Stderr = nil
		return cmd.Run() == nil
	}

	fmt.Printf("HasGo: %v\n", detect("go", "version"))
	fmt.Printf("HasTmux: %v\n", detect("tmux", "-V"))
	fmt.Printf("HasNode: %v\n", detect("node", "--version"))
	fmt.Printf("HasNpm: %v\n", detect("npm", "--version"))
	fmt.Printf("HasOllama: %v\n", detect("ollama", "list"))
	fmt.Printf("HasClaude: %v\n", detect("claude", "--version"))

	if v, err := exec.Command("go", "version").Output(); err == nil {
		fmt.Printf("GoVersion: %s\n", string(v))
	}

	os.Exit(0)
}
```

---

## Task 4: Create `Makefile` Targets for Distribution

**Files:**
- Modify: `Makefile` (create if not exists)

- [ ] **Step 1: Add install and distribution targets to Makefile**

```makefile
.PHONY: install install-local build build-all clean lint test

BINARY_NAME := agentboard
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_LDFLAGS := -X main.version=$(VERSION)

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet

install: build
	install -Dm755 $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	install -Dm644 internal/config/testdata/config.toml ~/.agentboard/config.toml 2>/dev/null || true
	@echo "Installed to /usr/local/bin/$(BINARY_NAME)"

install-local: build
	install -Dm755 $(BINARY_NAME) ~/.local/bin/$(BINARY_NAME)
	@echo "Installed to ~/.local/bin/$(BINARY_NAME)"
	@echo "Add ~/.local/bin to your PATH if needed."

build:
	CGO_ENABLED=0 $(GOBUILD) -ldflags="$(BUILD_LDFLAGS)" -o $(BINARY_NAME) ./cmd/agentboard

build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64

build-linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(BUILD_LDFLAGS)" -o $(BINARY_NAME)-linux-amd64 ./cmd/agentboard

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(BUILD_LDFLAGS)" -o $(BINARY_NAME)-linux-arm64 ./cmd/agentboard

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(BUILD_LDFLAGS)" -o $(BINARY_NAME)-darwin-amd64 ./cmd/agentboard

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(BUILD_LDFLAGS)" -o $(BINARY_NAME)-darwin-arm64 ./cmd/agentboard

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -f /usr/local/bin/$(BINARY_NAME)
	rm -f ~/.local/bin/$(BINARY_NAME)

lint:
	$(GOVET) ./...

test:
	$(GOTEST) ./...
```

---

## Task 5: Update README.md with Install Instructions

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add quick-install section to README**

Replace the existing Installation section with:

```markdown
## 🚀 Quick Install

### One-Line Install (Linux/macOS/Termux)

```bash
curl -fsSL https://raw.githubusercontent.com/ayan-de/agent-board/main/scripts/install.sh | bash
```

### Manual Install

```bash
git clone https://github.com/ayan-de/agent-board.git
cd agent-board
make install-local   # installs to ~/.local/bin/agentboard
```

The install script will:
- Verify Go 1.21+ is installed
- Check for tmux and offer to install if missing
- Detect your LLM provider (Ollama, Anthropic, OpenAI)
- Auto-generate `~/.agentboard/config.toml`
- Build the binary

### Windows

```powershell
irm https://raw.githubusercontent.com/ayan-de/agent-board/main/scripts/install.ps1 | iex
```

---

## Task 6: Add `scripts/` to `.gitignore`

**Files:**
- Modify: `.gitignore`

- [ ] **Step 1: Append install script entries to .gitignore**

```gitignore
# Install scripts (committed separately)
scripts/install.sh
scripts/install.ps1
scripts/install_detect.go
```

---

## Task 7: Run Verification

- [ ] **Step 1: Run shellcheck on install.sh**

```bash
shellcheck scripts/install.sh
```

Expected: no errors (SCxxxx warnings acceptable)

- [ ] **Step 2: Test build via make**

```bash
make build
./agentboard --version 2>&1 || true
```

Expected: binary compiles and runs

- [ ] **Step 3: Test install script logic manually**

```bash
bash -n scripts/install.sh && echo "Syntax OK"
```

Expected: Syntax OK

---

## Self-Review Checklist

- [x] **Spec coverage**: All user requirements covered (OS check, tmux install, provider detection, config generation, `agentboard` at project root)
- [ ] **Placeholder scan**: No "TBD", "TODO", or vague implementation steps
- [ ] **Type consistency**: N/A (shell + PowerShell, no Go type consistency concerns)

---

## Execution Options

**Plan complete and saved to `docs/superpowers/plans/2026-04-26-publish-distribute.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
