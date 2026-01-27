#!/bin/bash

# Auracrab Universal Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/nathfavour/auracrab/master/install.sh | bash

set -e

AURACRAB_REPO="nathfavour/auracrab"
VIBEAURACLE_REPO="nathfavour/vibeauracle"

# Detect OS and Arch
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64) ARCH="amd64" ;; 
    aarch64|arm64) ARCH="arm64" ;; 
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;; 
esac

if [ "$OS" = "linux" ]; then
    if [ -n "$TERMUX_VERSION" ] || [ -d "/data/data/com.termux" ]; then
        OS="android"
    fi
elif [ "$OS" != "darwin" ]; then
    echo "Unsupported OS: $OS"
    exit 1
fi

echo "Detected Platform: $OS/$ARCH"

# Install Directory
if [ "$OS" = "android" ]; then
    INSTALL_DIR="$HOME/bin"
else
    # Prefer standard Go bin or ~/.local/bin
    if [ -n "$GOPATH" ]; then
        INSTALL_DIR="$GOPATH/bin"
    elif [ -d "$HOME/go/bin" ]; then
        INSTALL_DIR="$HOME/go/bin"
    elif [ -d "$HOME/.local/bin" ]; then
        INSTALL_DIR="$HOME/.local/bin"
    else
        INSTALL_DIR="/usr/local/bin"
    fi
fi

mkdir -p "$INSTALL_DIR" 2>/dev/null || true

# --- Ensure vibeauracle is installed ---
if ! command -v vibeaura >/dev/null 2>&1; then
    echo "Installing vibeauracle (dependency)..."
    curl -fsSL "https://raw.githubusercontent.com/$VIBEAURACLE_REPO/release/install.sh" | bash
else
    echo "vibeauracle already installed."
fi

# --- Install auracrab ---
echo "Fetching auracrab release metadata..."
# Try to get the latest tag from GitHub API
LATEST_TAG=$(curl -fsSL "https://api.github.com/repos/$AURACRAB_REPO/releases/latest" 2>/dev/null | grep -oE '"tag_name": *"[^"]+"' | head -n 1 | cut -d'"' -f4 || echo "")

if [ -z "$LATEST_TAG" ]; then
    # Fallback: track latest rolling tag
    LATEST_TAG="latest"
fi

echo "Resolved version: $LATEST_TAG"

BINARY_NAME="auracrab-${OS}-${ARCH}"
if [ "$LATEST_TAG" = "latest" ]; then
    DOWNLOAD_URL="https://github.com/$AURACRAB_REPO/releases/latest/download/$BINARY_NAME"
else
    DOWNLOAD_URL="https://github.com/$AURACRAB_REPO/releases/download/$LATEST_TAG/$BINARY_NAME"
fi

echo "Downloading auracrab from $DOWNLOAD_URL..."
curl -L "$DOWNLOAD_URL" -o auracrab_tmp
chmod +x auracrab_tmp

if [ -w "$INSTALL_DIR" ]; then
    mv auracrab_tmp "$INSTALL_DIR/auracrab"
else
    echo "Requesting sudo to install to $INSTALL_DIR..."
    sudo mv auracrab_tmp "$INSTALL_DIR/auracrab"
fi

echo "Successfully installed auracrab to $INSTALL_DIR/auracrab"

# Add to PATH if necessary
SHELL_RC=""
[ -f "$HOME/.zshrc" ] && SHELL_RC="$HOME/.zshrc"
[ -f "$HOME/.bashrc" ] && [ -z "$SHELL_RC" ] && SHELL_RC="$HOME/.bashrc"

if [ -n "$SHELL_RC" ]; then
    if ! grep -q "$INSTALL_DIR" "$SHELL_RC" 2>/dev/null; then
        echo "" >> "$SHELL_RC"
        echo "# auracrab path" >> "$SHELL_RC"
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_RC"
        echo "Added $INSTALL_DIR to $SHELL_RC"
    fi
fi

echo "Installation complete. Run 'auracrab start' to begin."
