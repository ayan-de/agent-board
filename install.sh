#!/bin/bash
set -e

REPO="ayan-de/agent-board"
BASE_URL="https://github.com/${REPO}/releases/latest/download"
INSTALL_DIR="${HOME}/.local/bin"

# OS detection
detect_os() {
  case "$(uname -s)" in
    Linux*)
      if [ -f /proc/version ] && grep -q Android /proc/version 2>/dev/null; then
        echo "android"
      else
        echo "linux"
      fi
      ;;
    Darwin*) echo "darwin" ;;
    *)       echo "unknown" ;;
  esac
}

# Architecture detection
detect_arch() {
  case "$(uname -m)" in
    x86_64)           echo "x86_64" ;;
    aarch64|arm64)    echo "arm64" ;;
    armv7l|arm)       echo "arm" ;;
    *)                echo "unknown" ;;
  esac
}

# Check if a command exists
has() { command -v "$1" &>/dev/null; }

# Install package using detected package manager
install() {
  local pkg="$1"
  if has apt-get; then
    echo "[apt] Installing $pkg..."
    sudo apt-get install -y "$pkg" 2>/dev/null || apt-get install -y "$pkg"
  elif has pacman; then
    echo "[pacman] Installing $pkg..."
    sudo pacman -S --noconfirm "$pkg"
  elif has brew; then
    echo "[brew] Installing $pkg..."
    brew install "$pkg"
  elif has pkg; then
    echo "[pkg] Installing $pkg..."
    pkg install -y "$pkg"
  else
    echo "ERROR: No supported package manager found. Please install $pkg manually."
    return 1
  fi
}

# Ensure dependency is installed if not on PATH
ensure() {
  local cmd="$1"
  local pkg="${2:-$1}"
  if ! has "$cmd"; then
    install "$pkg"
  else
    echo "[OK] $cmd already installed"
  fi
}

echo "=== AgentBoard Installer ==="
echo ""

OS="$(detect_os)"
ARCH="$(detect_arch)"
echo "Detected: $OS / $ARCH"
echo ""

# Binary name mapping
case "$OS-$ARCH" in
  linux-x86_64)    BINARY="agentboard-linux-x86_64" ;;
  linux-arm64)     BINARY="agentboard-linux-arm64" ;;
  darwin-x86_64)   BINARY="agentboard-darwin-x86_64" ;;
  darwin-arm64)    BINARY="agentboard-darwin-arm64" ;;
  android-arm64)   BINARY="agentboard-android-arm64" ;;
  *)               echo "ERROR: Unsupported platform: $OS-$ARCH"; exit 1 ;;
esac

echo "Checking dependencies..."
ensure git git
ensure tmux tmux
ensure go golang
ensure npm npm
ensure node nodejs
echo ""

echo "Downloading agentboard..."
mkdir -p "$INSTALL_DIR"
curl -L "${BASE_URL}/${BINARY}" -o "${INSTALL_DIR}/agentboard"
chmod +x "${INSTALL_DIR}/agentboard"

echo ""
echo "=== Installation complete ==="
echo ""
echo "Add AgentBoard to your PATH:"
echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
echo ""
echo "Then run:"
echo "  agentboard init"
echo ""
echo "Or start using agentboard directly:"
echo "  ${INSTALL_DIR}/agentboard"