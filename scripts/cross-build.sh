#!/bin/bash
# ========================================================================
# StormDB Cross-Platform Build Script
# Build for multiple architectures using Docker
# ========================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Supported architectures
ARCHITECTURES=("x86_64" "aarch64" "armv7l")
CROSS_COMPILERS=("gcc" "aarch64-linux-gnu-gcc" "arm-linux-gnueabihf-gcc")

# Build directory
BUILD_DIR="build-cross"

# Create build directory structure
setup_build_dir() {
    log_info "Setting up cross-compilation build directory..."
    
    rm -rf "$BUILD_DIR"
    mkdir -p "$BUILD_DIR"
    
    for arch in "${ARCHITECTURES[@]}"; do
        mkdir -p "$BUILD_DIR/$arch"/{bin,obj}
    done
    
    log_success "Build directory structure created"
}

# Build for specific architecture
build_for_arch() {
    local arch="$1"
    local compiler="$2"
    local build_type="${3:-release}"
    
    log_info "Building for $arch using $compiler ($build_type)..."
    
    # Copy source to build directory
    cp -r src include Makefile "$BUILD_DIR/$arch/"
    
    # Build in container
    docker run --rm \
        -v "$(pwd)/$BUILD_DIR/$arch:/workspace" \
        -w /workspace \
        -e CC="$compiler" \
        -e ARCH="$arch" \
        stormdb-cross:latest \
        bash -c "
            make clean
            make info
            make $build_type
            echo 'Build completed for $arch'
        "
    
    # Copy results back
    if [ -f "$BUILD_DIR/$arch/bin/stormdb-$build_type" ]; then
        cp "$BUILD_DIR/$arch/bin/stormdb-$build_type" "$BUILD_DIR/stormdb-$build_type-$arch"
        log_success "Binary for $arch available at $BUILD_DIR/stormdb-$build_type-$arch"
    else
        log_error "Build failed for $arch"
        return 1
    fi
}

# Build for all architectures
build_all() {
    local build_type="${1:-release}"
    
    log_info "Building for all architectures ($build_type)..."
    
    setup_build_dir
    
    # Build cross-compilation container first
    log_info "Building cross-compilation container..."
    docker compose build stormdb-cross
    
    # Build for each architecture
    for i in "${!ARCHITECTURES[@]}"; do
        arch="${ARCHITECTURES[$i]}"
        compiler="${CROSS_COMPILERS[$i]}"
        
        if ! build_for_arch "$arch" "$compiler" "$build_type"; then
            log_error "Failed to build for $arch"
            continue
        fi
    done
    
    log_success "Cross-compilation completed"
    
    # Show results
    log_info "Built binaries:"
    ls -la "$BUILD_DIR"/stormdb-* 2>/dev/null || log_info "No binaries found"
}

# Test binaries using QEMU
test_binaries() {
    log_info "Testing cross-compiled binaries with QEMU..."
    
    # Install QEMU if not available
    if ! command -v qemu-aarch64-static &> /dev/null; then
        log_info "Installing QEMU for testing..."
        docker run --rm \
            -v "$(pwd)/$BUILD_DIR:/workspace" \
            ubuntu:22.04 \
            bash -c "
                apt-get update
                apt-get install -y qemu-user-static
                cp /usr/bin/qemu-*-static /workspace/
            "
    fi
    
    # Test each binary
    for binary in "$BUILD_DIR"/stormdb-*; do
        if [ -f "$binary" ]; then
            log_info "Testing $(basename "$binary")..."
            
            # Determine architecture and test
            if [[ "$binary" == *"aarch64"* ]]; then
                docker run --rm \
                    -v "$(pwd)/$BUILD_DIR:/workspace" \
                    -w /workspace \
                    ubuntu:22.04 \
                    bash -c "
                        apt-get update && apt-get install -y qemu-user-static
                        qemu-aarch64-static $(basename "$binary") --version
                    "
            elif [[ "$binary" == *"armv7l"* ]]; then
                docker run --rm \
                    -v "$(pwd)/$BUILD_DIR:/workspace" \
                    -w /workspace \
                    ubuntu:22.04 \
                    bash -c "
                        apt-get update && apt-get install -y qemu-user-static
                        qemu-arm-static $(basename "$binary") --version
                    "
            else
                # x86_64 - can run natively
                "$binary" --version
            fi
            
            log_success "$(basename "$binary") works correctly"
        fi
    done
}

# Package binaries
package_binaries() {
    local version="${1:-$(date +%Y%m%d)}"
    
    log_info "Packaging binaries for release..."
    
    # Create release directory
    local release_dir="release-$version"
    mkdir -p "$release_dir"
    
    # Copy binaries
    for binary in "$BUILD_DIR"/stormdb-*; do
        if [ -f "$binary" ]; then
            cp "$binary" "$release_dir/"
        fi
    done
    
    # Create checksums
    cd "$release_dir"
    sha256sum stormdb-* > checksums.sha256
    cd ..
    
    # Create tarball
    tar -czf "stormdb-$version-multi-arch.tar.gz" "$release_dir"
    
    log_success "Release package created: stormdb-$version-multi-arch.tar.gz"
}

# Show usage
show_usage() {
    echo "StormDB Cross-Platform Build Script"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  build [type]     Build for all architectures (default: release)"
    echo "  test             Test cross-compiled binaries"
    echo "  package [ver]    Package binaries for release"
    echo "  clean            Clean build artifacts"
    echo "  help             Show this help"
    echo ""
    echo "Build types: debug, release, asan"
    echo ""
    echo "Examples:"
    echo "  $0 build release    # Build release binaries for all architectures"
    echo "  $0 build debug      # Build debug binaries for all architectures"
    echo "  $0 test             # Test all built binaries"
    echo "  $0 package v1.0.0   # Package binaries for release"
    echo ""
}

# Clean build artifacts
clean() {
    log_info "Cleaning cross-compilation artifacts..."
    rm -rf "$BUILD_DIR" release-* *.tar.gz
    log_success "Cleanup completed"
}

# Main function
main() {
    case "${1:-help}" in
        "build")
            build_all "$2"
            ;;
        "test")
            test_binaries
            ;;
        "package")
            package_binaries "$2"
            ;;
        "clean")
            clean
            ;;
        "help"|"--help"|"-h")
            show_usage
            ;;
        *)
            log_error "Unknown command: $1"
            echo ""
            show_usage
            exit 1
            ;;
    esac
}

# Check requirements
if ! command -v docker &> /dev/null; then
    log_error "Docker is required for cross-compilation"
    exit 1
fi

# Run main function
main "$@"
