#!/bin/bash
# ubuntu_build.sh - Build script for Ubuntu systems
# Handles common Ubuntu build issues automatically

set -e

echo "🐧 StormDB Ubuntu Build Script"
echo "=============================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check Go version
print_status "Checking Go version..."
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.23+ first."
    echo "See UBUNTU_BUILD.md for installation instructions."
    exit 1
fi

GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
REQUIRED_VERSION="1.23"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    print_warning "Go version $GO_VERSION detected. Recommended: $REQUIRED_VERSION+"
    print_warning "Some development tools may not work properly."
else
    print_status "Go version $GO_VERSION is suitable."
fi

# Set environment variables for Ubuntu
export CGO_ENABLED=1
export GOPROXY="https://proxy.golang.org,direct"
export GOSUMDB="sum.golang.org"

print_status "Environment configured for Ubuntu build."

# Clean previous builds
print_status "Cleaning previous builds..."
make clean 2>/dev/null || true

# Install dependencies
print_status "Installing Go dependencies..."
if ! make deps; then
    print_error "Failed to install dependencies"
    exit 1
fi

# Build main binary
print_status "Building StormDB binary..."
if ! make build; then
    print_error "Failed to build main binary"
    exit 1
fi

# Build plugins
print_status "Building plugins..."
if ! make plugins; then
    print_error "Failed to build plugins"
    exit 1
fi

# Verify build
print_status "Verifying build..."
if [ -f "build/stormdb" ]; then
    print_status "✅ Build successful!"
    echo ""
    echo "Binary location: $(pwd)/build/stormdb"
    echo "Plugins location: $(pwd)/build/plugins/"
    echo ""
    echo "Quick test:"
    ./build/stormdb --help | head -10
    echo ""
    echo "To run a simple test:"
    echo "  ./build/stormdb -c config/config_simple_connection.yaml --help"
else
    print_error "Build verification failed - binary not found"
    exit 1
fi

# Optional: Install development tools
read -p "Install development tools? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_status "Installing minimal development tools..."
    if make dev-tools-minimal; then
        print_status "✅ Development tools installed"
    else
        print_warning "Some development tools failed to install (this is usually OK)"
    fi
fi

print_status "🎉 Ubuntu build process complete!"
echo ""
echo "Next steps:"
echo "1. Update configuration files in config/ with your database settings"
echo "2. Run a test: ./build/stormdb -c config/config_simple_connection.yaml -duration=10s"
echo "3. See README.md for usage examples"
