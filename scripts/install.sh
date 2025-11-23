#!/usr/bin/env bash

set -e

# Collab Installation Script
# Installs the collab binary and sets up configuration directories

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Installation directories
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/collab-cli"
DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/collab-cli"

# Binary name
BINARY_NAME="collab"

echo "Collab Installation"
echo "==================="
echo ""

# Check if binary exists
if [ ! -f "$BINARY_NAME" ]; then
    echo -e "${RED}Error: $BINARY_NAME binary not found${NC}"
    echo "Please run 'make build' first to compile the binary"
    exit 1
fi

# Check binary is executable
if [ ! -x "$BINARY_NAME" ]; then
    echo -e "${YELLOW}Warning: $BINARY_NAME is not executable, making it executable${NC}"
    chmod +x "$BINARY_NAME"
fi

# Create installation directory
echo "Creating installation directory: $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"

# Install binary
echo "Installing $BINARY_NAME to $INSTALL_DIR..."
cp "$BINARY_NAME" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

# Create config directory
echo "Creating config directory: $CONFIG_DIR"
mkdir -p "$CONFIG_DIR"

# Create data directory (for sessions)
echo "Creating data directory: $DATA_DIR"
mkdir -p "$DATA_DIR/sessions"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo -e "${YELLOW}Warning: $INSTALL_DIR is not in your PATH${NC}"
    echo ""
    echo "Add it to your PATH by adding this line to your shell profile:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    echo ""

    # Detect shell and provide specific instructions
    if [ -n "$BASH_VERSION" ]; then
        echo "For bash, add to ~/.bashrc or ~/.bash_profile:"
        echo "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.bashrc"
    elif [ -n "$ZSH_VERSION" ]; then
        echo "For zsh, add to ~/.zshrc:"
        echo "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.zshrc"
    else
        echo "Add to your shell profile (~/.profile, ~/.bashrc, ~/.zshrc, etc.)"
    fi
    echo ""
fi

# Display completion info
echo ""
echo -e "${GREEN}✓ Installation complete!${NC}"
echo ""
echo "Installed files:"
echo "  Binary: $INSTALL_DIR/$BINARY_NAME"
echo "  Config: $CONFIG_DIR/"
echo "  Data:   $DATA_DIR/"
echo ""
echo "Verify installation:"
echo "  $BINARY_NAME version"
echo ""
echo "Generate shell completions (optional):"
echo "  # Bash (Linux):"
echo "  sudo $BINARY_NAME completion bash > /etc/bash_completion.d/$BINARY_NAME"
echo "  # Bash (macOS):"
echo "  $BINARY_NAME completion bash > \$(brew --prefix)/etc/bash_completion.d/$BINARY_NAME"
echo "  # Zsh:"
echo "  $BINARY_NAME completion zsh > \"\${fpath[1]}/_$BINARY_NAME\""
echo "  # Fish:"
echo "  $BINARY_NAME completion fish > ~/.config/fish/completions/$BINARY_NAME.fish"
echo ""
echo "Get started:"
echo "  $BINARY_NAME run examples/simple-agreement.md --watch"
echo ""

# Optionally create example config file
if [ ! -f "$CONFIG_DIR/config.json" ]; then
    echo "Creating example config file: $CONFIG_DIR/config.json"
    cat > "$CONFIG_DIR/config.json" << 'EOF'
{
  "workspace_dir": "",
  "log_level": "info",
  "default_timeout": 60,
  "anthropic_api_key": ""
}
EOF
    echo ""
    echo -e "${YELLOW}Note: Set your ANTHROPIC_API_KEY in $CONFIG_DIR/config.json or as environment variable${NC}"
    echo "  export ANTHROPIC_API_KEY=\"your-api-key-here\""
    echo ""
fi
