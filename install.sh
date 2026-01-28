#!/bin/bash

# Auracrab Universal Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/nathfavour/auracrab/master/install.sh | bash

set -e

AURACRAB_REPO="nathfavour/auracrab"
VIBEAURACLE_REPO="nathfavour/vibeauracle"
GITHUB_URL="https://github.com/$AURACRAB_REPO"

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

# --- Intelligent Update Detection ---
echo "Checking for updates..."
REMOTE_SHA=""
LATEST_TAG=""

if command -v git >/dev/null 2>&1; then
    # Try to get the latest commit SHA of master
    REMOTE_SHA=$(git ls-remote "$GITHUB_URL.git" HEAD | awk '{print $1}')
fi

UP_TO_DATE=false
if command -v auracrab >/dev/null 2>&1; then
    LOCAL_COMMIT=$(auracrab version --short-commit 2>/dev/null || true)
    if [ -n "$REMOTE_SHA" ] && [ "$LOCAL_COMMIT" = "${REMOTE_SHA:0:7}" ]; then
        UP_TO_DATE=true
    fi
fi

if [ "$UP_TO_DATE" = "true" ]; then
    echo "Auracrab is already up to date (${REMOTE_SHA:0:7})."
else
    # --- Intelligent Build-from-Source Detection ---
    BUILD_FROM_SOURCE=false
    if command -v go >/dev/null 2>&1 && command -v git >/dev/null 2>&1; then
        GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
        V_MAJOR=$(echo $GO_VERSION | cut -d. -f1)
        V_MINOR=$(echo $GO_VERSION | cut -d. -f2)
        if [ "$V_MAJOR" -gt 1 ] || { [ "$V_MAJOR" -eq 1 ] && [ "$V_MINOR" -ge 21 ]; }; then
            BUILD_FROM_SOURCE=true
            echo "Detected Go $GO_VERSION and Git. Building from source."
        fi
    fi

    if [ "$AURACRAB_FORCE_BINARY" = "true" ]; then
        BUILD_FROM_SOURCE=false
    fi

    if [ "$BUILD_FROM_SOURCE" = "true" ]; then
        # Check if we are already in the auracrab source directory
        if [ -f "go.mod" ] && grep -q "github.com/nathfavour/auracrab" go.mod;
 then
            echo "Current directory is auracrab source."
            # Only pull if changes actually exist remotely
            if [ -n "$REMOTE_SHA" ]; then
                LOCAL_REPO_SHA=$(git rev-parse HEAD)
                if [ "$LOCAL_REPO_SHA" != "$REMOTE_SHA" ]; then
                    echo "Updating local repository..."
                    git pull origin master || true
                fi
            fi
            SRC_DIR="$PWD"
            CLEANUP=false
        else
            echo "Cloning auracrab source..."
            TMP_DIR=$(mktemp -d)
            git clone --depth 1 "$GITHUB_URL.git" "$TMP_DIR"
            SRC_DIR="$TMP_DIR"
            CLEANUP=true
        fi

        cd "$SRC_DIR"
        
        VERSION=$(git describe --tags --always || echo "dev")
        COMMIT=$(git rev-parse --short HEAD || echo "none")
        DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
        
        LDFLAGS="-s -w -X github.com/nathfavour/auracrab/internal/cli.Version=$VERSION \
                       -X github.com/nathfavour/auracrab/internal/cli.Commit=$COMMIT \
                       -X github.com/nathfavour/auracrab/internal/cli.BuildDate=$DATE"
        
        echo "Building auracrab ($VERSION)..."
        go build -ldflags "$LDFLAGS" -o auracrab ./cmd/auracrab
        
        if [ -w "$INSTALL_DIR" ]; then
            mv auracrab "$INSTALL_DIR/auracrab"
        else
            sudo mv auracrab "$INSTALL_DIR/auracrab"
        fi
        
        if [ "$CLEANUP" = "true" ]; then
            cd - > /dev/null
            rm -rf "$SRC_DIR"
        fi
    else
        echo "Installing from binary..."
        LATEST_TAG_RELEASE=$(curl -fsSL "https://api.github.com/repos/$AURACRAB_REPO/releases/latest" 2>/dev/null | grep -oE '"tag_name": *"[^"]+"' | head -n 1 | cut -d'"' -f4 || echo "latest")
        
        BINARY_NAME="auracrab-${OS}-${ARCH}"
        DOWNLOAD_URL="$GITHUB_URL/releases/download/$LATEST_TAG_RELEASE/$BINARY_NAME"
        if [ "$LATEST_TAG_RELEASE" = "latest" ]; then
             DOWNLOAD_URL="$GITHUB_URL/releases/latest/download/$BINARY_NAME"
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
fi

# --- Ensure vibeauracle is installed ---
if ! command -v vibeaura >/dev/null 2>&1; then
    echo "Installing vibeauracle (dependency)..."
    curl -fsSL "https://raw.githubusercontent.com/$VIBEAURACLE_REPO/release/install.sh" | bash
else
    # Vibeauracle installer already handles its own up-to-date checks
    curl -fsSL "https://raw.githubusercontent.com/$VIBEAURACLE_REPO/release/install.sh" | bash > /dev/null 2>&1
fi

# Add to PATH if necessary
SHELL_RC=""
[ -f "$HOME/.zshrc" ] && SHELL_RC="$HOME/.zshrc"
[ -f "$HOME/.bashrc" ] && [ -z "$SHELL_RC" ] && SHELL_RC="$HOME/.bashrc"

if [ -n "$SHELL_RC" ]; then
    if ! grep -q "$INSTALL_DIR" "$SHELL_RC" 2>/dev/null;
 then
        echo "" >> "$SHELL_RC"
        echo "# auracrab path" >> "$SHELL_RC"
        echo "export PATH=\"
$PATH:$INSTALL_DIR\"" >> "$SHELL_RC"
        echo "Added $INSTALL_DIR to $SHELL_RC"
    fi
fi

echo "Done. Run 'auracrab start' to begin."
