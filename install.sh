#!/bin/bash
# StormDB Installation Script for Linux/macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/elchinoo/stormdb/main/install.sh | bash
# Or: curl -fsSL https://raw.githubusercontent.com/elchinoo/stormdb/main/install.sh | bash -s v0.1.0-alpha.2

set -e

# Configuration
REPO="elchinoo/stormdb"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="stormdb"
VERSION="${1:-latest}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
    exit 1
}

# Detect OS and architecture
detect_platform() {
    local os
    local arch
    
    case "$(uname -s)" in
        Linux*) os="linux" ;;
        Darwin*) os="darwin" ;;
        *) log_error "Unsupported operating system: $(uname -s)" ;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        i386|i686) arch="386" ;;
        *) log_error "Unsupported architecture: $(uname -m)" ;;
    esac
    
    echo "${os}-${arch}"
}

# Get latest version from GitHub API
get_latest_version() {
    local latest_url="https://api.github.com/repos/${REPO}/releases/latest"
    
    if command -v curl >/dev/null 2>&1; then
        curl -s "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' 2>/dev/null
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' 2>/dev/null
    else
        log_error "Neither curl nor wget is available. Please install one of them."
    fi
}

# Check if version exists
check_version_exists() {
    local version="$1"
    local check_url="https://api.github.com/repos/${REPO}/releases/tags/${version}"
    
    if command -v curl >/dev/null 2>&1; then
        if ! curl -s --fail "$check_url" >/dev/null 2>&1; then
            return 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q --spider "$check_url" 2>/dev/null; then
            return 1
        fi
    fi
    return 0
}

# Download and install binary
install_binary() {
    local platform="$1"
    local version="$2"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${BINARY_NAME}-${platform}.tar.gz"
    local temp_dir
    
    log_info "Downloading StormDB ${version} for ${platform}..."
    
    temp_dir=$(mktemp -d)
    cd "$temp_dir"
    
    # Download binary
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$download_url" -o "${BINARY_NAME}.tar.gz"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$download_url" -O "${BINARY_NAME}.tar.gz"
    else
        log_error "Neither curl nor wget is available. Please install one of them."
    fi
    
    # Extract binary
    log_info "Extracting binary..."
    tar -xzf "${BINARY_NAME}.tar.gz"
    
    # Make executable
    chmod +x "$BINARY_NAME"
    
    # Install binary
    log_info "Installing to ${INSTALL_DIR}..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "$BINARY_NAME" "$INSTALL_DIR/"
    else
        sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
    fi
    
    # Clean up
    cd - >/dev/null
    rm -rf "$temp_dir"
    
    log_success "StormDB installed successfully!"
}

# Verify installation
verify_installation() {
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        local installed_version
        installed_version=$("$BINARY_NAME" --version 2>/dev/null | head -n1 || echo "unknown")
        log_success "Installation verified: $installed_version"
        log_info "Try running: $BINARY_NAME --help"
    else
        log_warning "Binary installed but not found in PATH. You may need to:"
        log_warning "  export PATH=\"${INSTALL_DIR}:\$PATH\""
        log_warning "Or restart your terminal."
    fi
}

# Check if running with sufficient privileges
check_privileges() {
    if [ ! -w "$INSTALL_DIR" ] && [ "$(id -u)" -ne 0 ]; then
        log_warning "Installation requires sudo privileges for writing to $INSTALL_DIR"
        log_info "You may be prompted for your password..."
    fi
}

# Main installation process
main() {
    echo "🌟 StormDB Installation Script"
    echo "=============================="
    echo ""
    
    # Detect platform
    local platform
    platform=$(detect_platform)
    log_info "Detected platform: $platform"
    
    # Determine version
    local install_version
    if [ "$VERSION" = "latest" ]; then
        log_info "Fetching latest version..."
        install_version=$(get_latest_version)
        if [ -z "$install_version" ]; then
            log_error "Failed to fetch latest version. Please specify a version explicitly."
        fi
        log_info "Latest version: $install_version"
    else
        install_version="$VERSION"
        log_info "Installing version: $install_version"
        
        # Check if version exists
        if ! check_version_exists "$install_version"; then
            log_error "Version $install_version not found. Please check https://github.com/${REPO}/releases"
        fi
    fi
    
    # Check privileges
    check_privileges
    
    # Check if already installed
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        local current_version
        current_version=$("$BINARY_NAME" --version 2>/dev/null | head -n1 || echo "unknown")
        log_warning "StormDB is already installed: $current_version"
        printf "Do you want to continue with installation? [y/N] "
        read -r response
        case "$response" in
            [yY][eE][sS]|[yY]) 
                log_info "Proceeding with installation..."
                ;;
            *)
                log_info "Installation cancelled."
                exit 0
                ;;
        esac
    fi
    
    # Install binary
    install_binary "$platform" "$install_version"
    
    # Verify installation
    verify_installation
    
    echo ""
    echo "🎉 Installation completed!"
    echo ""
    echo "📖 Next steps:"
    echo "  1. Run 'stormdb --help' to see available options"
    echo "  2. Download sample configs: https://github.com/${REPO}/tree/main/config"
    echo "  3. Read the docs: https://github.com/${REPO}/blob/main/README.md"
    echo ""
    echo "🐛 Need help? https://github.com/${REPO}/issues"
}

# Run main function
main "$@"
