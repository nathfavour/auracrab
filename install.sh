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

# --- Intelligent Build-from-Source Detection ---
BUILD_FROM_SOURCE=false
if command -v go >/dev/null 2>&1 && command -v git >/dev/null 2>&1; then
    GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    # Check if Go version is >= 1.21
    V_MAJOR=$(echo $GO_VERSION | cut -d. -f1)
    V_MINOR=$(echo $GO_VERSION | cut -d. -f2)
    if [ "$V_MAJOR" -gt 1 ] || { [ "$V_MAJOR" -eq 1 ] && [ "$V_MINOR" -ge 21 ]; }; then
        BUILD_FROM_SOURCE=true
        echo "Detected Go $GO_VERSION and Git. Defaulting to build from source."
    else
        echo "Found Go $GO_VERSION, but 1.21+ is required for building. Falling back to binary."
    fi
fi

# Override via env var if needed
if [ "$AURACRAB_FORCE_BINARY" = "true" ]; then
    BUILD_FROM_SOURCE=false
fi

# --- Ensure vibeauracle is installed ---
if ! command -v vibeaura >/dev/null 2>&1; then
    echo "Installing vibeauracle (dependency)..."
    curl -fsSL "https://raw.githubusercontent.com/$VIBEAURACLE_REPO/release/install.sh" | bash
else
    echo "vibeauracle already installed."
fi

# --- Install auracrab ---
if [ "$BUILD_FROM_SOURCE" = "true" ]; then
    echo "Building auracrab from source (master)..."
    TMP_DIR=$(mktemp -d)
    git clone --depth 1 https://github.com/$AURACRAB_REPO.git "$TMP_DIR"
    cd "$TMP_DIR"
    
    # Inject version metadata if possible
    VERSION=$(git describe --tags --always || echo "master")
    LDFLAGS="-s -w -X github.com/nathfavour/auracrab/internal/cli.Version=$VERSION"
    
    go build -ldflags "$LDFLAGS" -o auracrab ./cmd/auracrab
    
    if [ -w "$INSTALL_DIR" ]; then
        mv auracrab "$INSTALL_DIR/auracrab"
    else
        sudo mv auracrab "$INSTALL_DIR/auracrab"
    fi
    cd - > /dev/null
    rm -rf "$TMP_DIR"
else
    echo "Fetching auracrab release metadata..."
    LATEST_TAG=$(curl -fsSL "https://api.github.com/repos/$AURACRAB_REPO/releases/latest" 2>/dev/null | grep -oE '"tag_name": *"[^"]+"' | head -n 1 | cut -d'"' -f4 || echo "")

    if [ -z "$LATEST_TAG" ]; then
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
        sudo mv auracrab_tmp "$INSTALL_DIR/auracrab"
    fi
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
