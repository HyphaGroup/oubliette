#!/bin/bash
set -e

REPO="HyphaGroup/oubliette"
DEFAULT_INSTALL_DIR="$HOME/.oubliette/bin"

echo "🗝️  Oubliette Installer"
echo ""

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  darwin) ;;
  linux) ;;
  *)
    echo "Error: Unsupported platform: $OS"
    echo "Oubliette supports macOS (darwin) and Linux."
    exit 1
    ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Error: Unsupported architecture: $ARCH"
    echo "Oubliette supports amd64 and arm64."
    exit 1
    ;;
esac

BINARY_NAME="oubliette-${OS}-${ARCH}"
echo "Detected platform: ${OS}/${ARCH}"
echo ""

# Get latest release
echo "Fetching latest release..."
RELEASE_INFO=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest")
VERSION=$(echo "$RELEASE_INFO" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "Error: Could not determine latest version."
  echo "Check if releases exist at https://github.com/${REPO}/releases"
  exit 1
fi

echo "Latest version: $VERSION"
echo ""

# Prompt for install location
read -p "Install location [$DEFAULT_INSTALL_DIR]: " INSTALL_DIR
INSTALL_DIR="${INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"

# Expand ~ if present
INSTALL_DIR="${INSTALL_DIR/#\~/$HOME}"

# Create directory
mkdir -p "$INSTALL_DIR"

# Download binary and checksums
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

echo "Downloading $BINARY_NAME..."
curl -fsSL "$DOWNLOAD_URL" -o "$INSTALL_DIR/oubliette"

echo "Downloading checksums..."
CHECKSUMS=$(curl -fsSL "$CHECKSUMS_URL")

# Verify checksum
echo "Verifying checksum..."
EXPECTED_CHECKSUM=$(echo "$CHECKSUMS" | grep "$BINARY_NAME" | awk '{print $1}')
if [ -z "$EXPECTED_CHECKSUM" ]; then
  echo "Warning: Could not find checksum for $BINARY_NAME"
else
  ACTUAL_CHECKSUM=$(shasum -a 256 "$INSTALL_DIR/oubliette" | awk '{print $1}')
  if [ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]; then
    echo "Error: Checksum mismatch!"
    echo "  Expected: $EXPECTED_CHECKSUM"
    echo "  Actual:   $ACTUAL_CHECKSUM"
    rm -f "$INSTALL_DIR/oubliette"
    exit 1
  fi
  echo "Checksum verified ✓"
fi

# Make executable
chmod +x "$INSTALL_DIR/oubliette"

echo ""
echo "✅ Oubliette $VERSION installed to $INSTALL_DIR/oubliette"
echo ""

# Check if in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo "⚠️  $INSTALL_DIR is not in your PATH."
  echo ""
  echo "Add it to your shell config:"
  echo "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.zshrc"
  echo "  source ~/.zshrc"
  echo ""
fi

echo "Next steps:"
echo "  1. Run 'oubliette init' to set up configuration"
echo "  2. Run 'oubliette mcp --setup <tool>' to configure your AI tool"
echo "  3. Run 'oubliette' to start the server"
