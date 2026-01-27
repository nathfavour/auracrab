#!/bin/bash

# Auracrab Universal Installer
# Also installs vibeauracle as a dependency.

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

# --- Install vibeauracle ---
echo "Installing vibeauracle (dependency)..."
curl -fsSL "https://raw.githubusercontent.com/$VIBEAURACLE_REPO/release/install.sh" | bash

# --- Install auracrab ---
echo "Fetching auracrab release metadata..."
LATEST_TAG=$(curl -fsSL "https://api.github.com/repos/$AURACRAB_REPO/releases/latest" | grep -oE '"tag_name": *"[^"]+"' | head -n 1 | cut -d'"' -f4)

if [ -z "$LATEST_TAG" ]; then
    # Fallback to 'latest' if tag discovery fails
    LATEST_TAG="latest"
fi

BINARY_NAME="auracrab-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/$AURACRAB_REPO/releases/download/$LATEST_TAG/$BINARY_NAME"

echo "Downloading auracrab $LATEST_TAG..."
curl -L "$DOWNLOAD_URL" -o auracrab_tmp
chmod +x auracrab_tmp

if [ -w "$INSTALL_DIR" ]; then
    mv auracrab_tmp "$INSTALL_DIR/auracrab"
else
    sudo mv auracrab_tmp "$INSTALL_DIR/auracrab"
fi

echo "Successfully installed auracrab to $INSTALL_DIR/auracrab"

# Add to PATH if necessary
SHELL_RC=""
[ -f "$HOME/.zshrc" ] && SHELL_RC="$HOME/.zshrc"
[ -f "$HOME/.bashrc" ] && [ -z "$SHELL_RC" ] && SHELL_RC="$HOME/.bashrc"

if [ -n "$SHELL_RC" ]; then
    if ! grep -q "$INSTALL_DIR" "$SHELL_RC" 2>/dev/null; then
        echo "export PATH=\"$PATH:$INSTALL_DIR\"" >> "$SHELL_RC"
        echo "Added $INSTALL_DIR to $SHELL_RC"
    fi
fi

echo "Installation complete. Run 'auracrab start' to begin."
