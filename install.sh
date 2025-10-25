#!/bin/bash
set -e

# please installer script
# Usage: curl -sSL https://raw.githubusercontent.com/iishyfishyy/please/main/install.sh | bash

REPO="iishyfishyy/please"
BINARY_NAME="please"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $OS in
  linux)
    OS="Linux"
    ;;
  darwin)
    OS="Darwin"
    ;;
  mingw*|msys*|cygwin*)
    OS="Windows"
    ;;
  *)
    echo "Unsupported operating system: $OS"
    exit 1
    ;;
esac

case $ARCH in
  x86_64|amd64)
    ARCH="x86_64"
    ;;
  aarch64|arm64)
    ARCH="arm64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Get the latest release version
echo "Fetching latest release..."
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
  echo "Error: Could not fetch latest version"
  exit 1
fi

echo "Latest version: $LATEST_VERSION"

# Construct download URL
if [ "$OS" = "Windows" ]; then
  FILENAME="${BINARY_NAME}_${LATEST_VERSION#v}_${OS}_${ARCH}.zip"
else
  FILENAME="${BINARY_NAME}_${LATEST_VERSION#v}_${OS}_${ARCH}.tar.gz"
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/$FILENAME"

echo "Downloading $FILENAME..."
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

if ! curl -sL "$DOWNLOAD_URL" -o "$FILENAME"; then
  echo "Error: Failed to download $DOWNLOAD_URL"
  exit 1
fi

# Extract archive
echo "Extracting..."
if [ "$OS" = "Windows" ]; then
  unzip -q "$FILENAME"
else
  tar -xzf "$FILENAME"
fi

# Determine install location
if [ -w "/usr/local/bin" ]; then
  INSTALL_DIR="/usr/local/bin"
elif [ -w "$HOME/.local/bin" ]; then
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
else
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
  echo "Note: Installing to $INSTALL_DIR"
  echo "Make sure $INSTALL_DIR is in your PATH"
fi

# Install binary
echo "Installing to $INSTALL_DIR..."
if [ "$OS" = "Windows" ]; then
  mv "${BINARY_NAME}.exe" "$INSTALL_DIR/"
  BINARY_PATH="$INSTALL_DIR/${BINARY_NAME}.exe"
else
  mv "$BINARY_NAME" "$INSTALL_DIR/"
  chmod +x "$INSTALL_DIR/$BINARY_NAME"
  BINARY_PATH="$INSTALL_DIR/$BINARY_NAME"
fi

# Cleanup
cd - > /dev/null
rm -rf "$TMP_DIR"

echo ""
echo "✓ $BINARY_NAME installed successfully to $BINARY_PATH"
echo ""
echo "Next steps:"
echo "  1. Run: please configure"
echo "  2. Start using: please \"your command in natural language\""
echo ""
echo "For more information, visit: https://github.com/$REPO"
