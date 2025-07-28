#!/bin/bash
# Cross-platform build script for StormDB
# This script builds binaries for multiple platforms and architectures

set -e

# Configuration
VERSION=${VERSION:-"dev"}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
BUILD_DIR="build/release"
LDFLAGS="-X main.version=${VERSION} -X main.commit=${COMMIT} -s -w"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[BUILD]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Platform configurations
declare -A PLATFORMS=(
    ["linux/amd64"]="linux amd64 1 "
    ["linux/arm64"]="linux arm64 1 "
    ["linux/386"]="linux 386 1 "
    ["darwin/amd64"]="darwin amd64 0 "
    ["darwin/arm64"]="darwin arm64 0 "
    ["windows/amd64"]="windows amd64 1 .exe"
    ["windows/386"]="windows 386 1 .exe"
)

# Check dependencies
check_dependencies() {
    log "Checking dependencies..."
    
    if ! command -v go &> /dev/null; then
        error "Go is not installed or not in PATH"
        exit 1
    fi
    
    GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    log "Found Go version: $GO_VERSION"
    
    # Check for cross-compilation tools
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if ! dpkg -l | grep -q mingw-w64; then
            warn "mingw-w64 not found. Windows builds may fail."
            warn "Install with: sudo apt-get install gcc-mingw-w64"
        fi
    fi
}

# Setup build environment
setup_build_env() {
    log "Setting up build environment..."
    
    # Create build directories
    rm -rf "$BUILD_DIR"
    mkdir -p "$BUILD_DIR"
    
    # Create subdirectories for different asset types
    mkdir -p "$BUILD_DIR/binaries"
    mkdir -p "$BUILD_DIR/packages"
    mkdir -p "$BUILD_DIR/checksums"
}

# Build binary for specific platform
build_binary() {
    local platform=$1
    local goos=$2
    local goarch=$3
    local cgo_enabled=$4
    local ext=$5
    
    log "Building binary for $platform..."
    
    local binary_name="stormdb-$platform"
    local full_binary_name="$binary_name$ext"
    
    # Set environment variables
    export GOOS=$goos
    export GOARCH=$goarch
    export CGO_ENABLED=$cgo_enabled
    
    # Set cross-compilation toolchain for CGO
    if [[ $cgo_enabled == "1" ]]; then
        case "$goos-$goarch" in
            "windows-amd64")
                export CC=x86_64-w64-mingw32-gcc
                export CXX=x86_64-w64-mingw32-g++
                ;;
            "windows-386")
                export CC=i686-w64-mingw32-gcc
                export CXX=i686-w64-mingw32-g++
                ;;
            "linux-386")
                export CC="gcc -m32"
                ;;
        esac
    fi
    
    # Build the binary
    if go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/binaries/$full_binary_name" ./cmd/stormdb; then
        success "Built $full_binary_name"
        
        # Calculate checksum
        cd "$BUILD_DIR/binaries"
        sha256sum "$full_binary_name" > "../checksums/$full_binary_name.sha256"
        cd - > /dev/null
        
        return 0
    else
        error "Failed to build $full_binary_name"
        return 1
    fi
}

# Build plugins (Linux only, requires CGO)
build_plugins() {
    local platform=$1
    local goos=$2
    local goarch=$3
    
    # Only build plugins for Linux platforms with CGO
    if [[ "$goos" != "linux" ]]; then
        return 0
    fi
    
    log "Building plugins for $platform..."
    
    local plugin_dir="$BUILD_DIR/plugins-$platform"
    mkdir -p "$plugin_dir"
    
    export GOOS=$goos
    export GOARCH=$goarch
    export CGO_ENABLED=1
    
    # Set cross-compilation toolchain
    case "$goarch" in
        "386")
            export CC="gcc -m32"
            ;;
    esac
    
    # Build each plugin
    cd plugins
    local plugin_count=0
    for plugin_path in */; do
        local plugin_name=${plugin_path%/}
        log "Building plugin: $plugin_name"
        
        cd "$plugin_path"
        if go build -buildmode=plugin -ldflags="-s -w" -o "../../$plugin_dir/${plugin_name}.so" .; then
            success "Built plugin $plugin_name"
            ((plugin_count++))
        else
            error "Failed to build plugin $plugin_name"
        fi
        cd ..
    done
    cd ..
    
    log "Built $plugin_count plugins for $platform"
}

# Create distribution packages
create_packages() {
    log "Creating distribution packages..."
    
    for platform_config in "${!PLATFORMS[@]}"; do
        local config=(${PLATFORMS[$platform_config]})
        local goos=${config[0]}
        local goarch=${config[1]}
        local ext=${config[3]}
        local platform=$(echo $platform_config | tr '/' '-')
        
        local binary_name="stormdb-$platform$ext"
        local package_name="stormdb-$platform"
        
        # Check if binary exists
        if [[ ! -f "$BUILD_DIR/binaries/$binary_name" ]]; then
            warn "Binary $binary_name not found, skipping package creation"
            continue
        fi
        
        log "Creating package for $platform..."
        
        # Create package directory structure
        local pkg_dir="$BUILD_DIR/packages/$package_name"
        mkdir -p "$pkg_dir"
        
        # Copy binary
        cp "$BUILD_DIR/binaries/$binary_name" "$pkg_dir/"
        
        # Copy plugins if they exist
        local plugin_dir="$BUILD_DIR/plugins-$platform"
        if [[ -d "$plugin_dir" ]]; then
            mkdir -p "$pkg_dir/plugins"
            cp "$plugin_dir"/* "$pkg_dir/plugins/"
        fi
        
        # Copy documentation and configuration
        cp README.md CHANGELOG.md "$pkg_dir/"
        [[ -f LICENSE ]] && cp LICENSE "$pkg_dir/"
        cp -r config "$pkg_dir/"
        
        # Create installation script for Unix systems
        if [[ "$goos" != "windows" ]]; then
            cat > "$pkg_dir/install.sh" << 'EOF'
#!/bin/bash
# StormDB Installation Script

set -e

INSTALL_DIR=${INSTALL_DIR:-/usr/local/bin}
CONFIG_DIR=${CONFIG_DIR:-$HOME/.config/stormdb}
PLUGIN_DIR=${PLUGIN_DIR:-$HOME/.local/lib/stormdb/plugins}

echo "Installing StormDB..."

# Create directories
mkdir -p "$CONFIG_DIR" "$PLUGIN_DIR"

# Install binary
sudo cp stormdb* "$INSTALL_DIR/" 2>/dev/null || cp stormdb* "$INSTALL_DIR/"
sudo chmod +x "$INSTALL_DIR"/stormdb* 2>/dev/null || chmod +x "$INSTALL_DIR"/stormdb*

# Install plugins
if [[ -d plugins ]]; then
    cp plugins/* "$PLUGIN_DIR/"
fi

# Install configuration examples
cp -r config/* "$CONFIG_DIR/"

echo "StormDB installed successfully!"
echo "Binary: $INSTALL_DIR/stormdb"
echo "Config: $CONFIG_DIR"
echo "Plugins: $PLUGIN_DIR"
EOF
            chmod +x "$pkg_dir/install.sh"
        fi
        
        # Create archive
        cd "$BUILD_DIR/packages"
        if [[ "$goos" == "windows" ]]; then
            # Create ZIP for Windows
            if command -v zip &> /dev/null; then
                zip -r "$package_name.zip" "$package_name"
                sha256sum "$package_name.zip" > "../checksums/$package_name.zip.sha256"
                success "Created $package_name.zip"
            else
                warn "zip command not found, skipping Windows package creation"
            fi
        else
            # Create tar.gz for Unix systems
            tar -czf "$package_name.tar.gz" "$package_name"
            sha256sum "$package_name.tar.gz" > "../checksums/$package_name.tar.gz.sha256"
            success "Created $package_name.tar.gz"
        fi
        cd - > /dev/null
        
        # Clean up temporary directory
        rm -rf "$pkg_dir"
    done
}

# Generate build summary
generate_summary() {
    log "Generating build summary..."
    
    local summary_file="$BUILD_DIR/BUILD_SUMMARY.md"
    cat > "$summary_file" << EOF
# StormDB Build Summary

**Version**: $VERSION  
**Commit**: $COMMIT  
**Build Date**: $(date -u +"%Y-%m-%d %H:%M:%S UTC")  
**Build Host**: $(hostname)

## Binaries Built

| Platform | Binary | Size | Checksum |
|----------|--------|------|----------|
EOF
    
    # Add binary information
    for file in "$BUILD_DIR/binaries"/*; do
        if [[ -f "$file" && ! "$file" =~ \.sha256$ ]]; then
            local filename=$(basename "$file")
            local size=$(du -h "$file" | cut -f1)
            local checksum=$(cat "$BUILD_DIR/checksums/$filename.sha256" | cut -d' ' -f1)
            echo "| $filename | $filename | $size | \`${checksum:0:16}...\` |" >> "$summary_file"
        fi
    done
    
    cat >> "$summary_file" << EOF

## Packages Created

EOF
    
    # Add package information
    for file in "$BUILD_DIR/packages"/*.{tar.gz,zip} 2>/dev/null; do
        if [[ -f "$file" ]]; then
            local filename=$(basename "$file")
            local size=$(du -h "$file" | cut -f1)
            echo "- **$filename** ($size)" >> "$summary_file"
        fi
    done
    
    cat >> "$summary_file" << EOF

## Quick Start

### Download and Run
\`\`\`bash
# Linux AMD64
curl -L "https://github.com/elchinoo/stormdb/releases/download/$VERSION/stormdb-linux-amd64.tar.gz" | tar xz
./stormdb-linux-amd64/stormdb --help

# macOS AMD64
curl -L "https://github.com/elchinoo/stormdb/releases/download/$VERSION/stormdb-darwin-amd64.tar.gz" | tar xz
./stormdb-darwin-amd64/stormdb --help

# Windows AMD64
# Download stormdb-windows-amd64.zip and extract
\`\`\`

### Verify Checksums
\`\`\`bash
# Download checksum file
curl -L "https://github.com/elchinoo/stormdb/releases/download/$VERSION/stormdb-linux-amd64.tar.gz.sha256"

# Verify
sha256sum -c stormdb-linux-amd64.tar.gz.sha256
\`\`\`
EOF
    
    success "Build summary created: $summary_file"
}

# Main build process
main() {
    log "Starting StormDB cross-platform build process..."
    log "Version: $VERSION"
    log "Commit: $COMMIT"
    
    check_dependencies
    setup_build_env
    
    local built_count=0
    local failed_count=0
    
    # Build binaries for each platform
    for platform_config in "${!PLATFORMS[@]}"; do
        local config=(${PLATFORMS[$platform_config]})
        local goos=${config[0]}
        local goarch=${config[1]}
        local cgo_enabled=${config[2]}
        local ext=${config[3]}
        local platform=$(echo $platform_config | tr '/' '-')
        
        if build_binary "$platform" "$goos" "$goarch" "$cgo_enabled" "$ext"; then
            ((built_count++))
            
            # Build plugins for supported platforms
            build_plugins "$platform" "$goos" "$goarch"
        else
            ((failed_count++))
        fi
    done
    
    # Create distribution packages
    create_packages
    
    # Generate build summary
    generate_summary
    
    # Final summary
    log "Build process completed!"
    success "Built $built_count binaries successfully"
    if [[ $failed_count -gt 0 ]]; then
        warn "$failed_count builds failed"
    fi
    
    log "Build artifacts located in: $BUILD_DIR"
    log "Binaries: $BUILD_DIR/binaries/"
    log "Packages: $BUILD_DIR/packages/"
    log "Checksums: $BUILD_DIR/checksums/"
    log "Summary: $BUILD_DIR/BUILD_SUMMARY.md"
}

# Script entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
