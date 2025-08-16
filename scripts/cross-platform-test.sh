#!/bin/bash
# ========================================================================
# StormDB Cross-Platform Testing Script (Optimized)
# Tests on both ARM64 (native) and x86_64 (emulated) platforms with cached images
# ========================================================================

set -e

PROJECT_DIR="$(pwd)"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to test on native ARM64 (macOS)
test_native_arm64() {
    log_info "Testing on native ARM64 (macOS)..."
    
    echo "🍎 macOS ARM64 Native Build"
    echo "============================"
    
    # Clean and build
    echo -e "${BLUE}[CLEAN]${NC} Cleaning previous builds..."
    make clean > /dev/null 2>&1
    
    echo -e "${BLUE}[PLATFORM]${NC} Platform information:"
    make info
    
    echo -e "${CYAN}[BUILD]${NC} Building debug version..."
    make debug
    
    # Test the build
    if [[ -f "bin/stormdb-debug" ]]; then
        echo -e "${GREEN}✅ ARM64 build successful${NC}"
        ./bin/stormdb-debug --version
        
        echo -e "${CYAN}[BUILD]${NC} Building AddressSanitizer version..."
        # Test with sanitizers if available
        if make asan 2>/dev/null; then
            echo -e "${GREEN}✅ ARM64 AddressSanitizer build successful${NC}"
            ./bin/stormdb-asan --version
        fi
    else
        echo -e "${RED}❌ ARM64 build failed${NC}"
        return 1
    fi
}

# Function to test on x86_64 Linux (emulated)
test_x86_64_linux() {
    log_info "Testing on x86_64 Linux (emulated)..."
    
    echo "🐧 x86_64 Linux Emulated Build"
    echo "==============================="
    
    # Check for cached base images
    local base_image=""
    local install_deps="true"
    
    if docker image inspect stormdb-builder-ubuntu:latest >/dev/null 2>&1; then
        echo -e "${GREEN}[CACHE]${NC} Using existing stormdb-builder-ubuntu:latest"
        base_image="stormdb-builder-ubuntu:latest"
    else
        log_error "No working base image found. Please run 'make linux-dev-quick' first."
        return 1
    fi
    
    echo -e "${BLUE}[START]${NC} Starting x86_64 Linux container..."
    
    # Use existing working image
    docker run --rm --platform linux/amd64 \
        -v "$PROJECT_DIR:/workspace" \
        -w /workspace \
        "$base_image" \
        bash -c '
            echo "[INFO] Building on x86_64 Linux..."
            
            echo "[DEPS] Installing dependencies..."
            apt-get update -qq && apt-get install -y -qq libyaml-dev libpq-dev pkg-config
            
            echo "[CLEAN] Cleaning previous builds..."
            make clean > /dev/null 2>&1
            
            echo "[PLATFORM] Platform information:"
            make info
            
            echo "[BUILD] Building debug version..."
            make debug
            
            echo "[TEST] Testing x86_64 build..."
            if [[ -f "bin/stormdb-debug" ]]; then
                echo "✅ x86_64 Linux build successful"
                ./bin/stormdb-debug --version
                
                echo "[BUILD] Building AddressSanitizer version..."
                if make asan 2>/dev/null; then
                    echo "✅ x86_64 AddressSanitizer build successful"
                    ./bin/stormdb-asan --version
                else
                    echo "⚠️ AddressSanitizer build failed (might be expected)"
                fi
                
                echo "[INFO] Binary architecture:"
                file bin/stormdb-debug
                
                echo "✅ x86_64 Linux testing completed"
            else
                echo "❌ x86_64 Linux build failed"
                exit 1
            fi
        '
}

# Function to compare results
compare_results() {
    log_info "Cross-Platform Comparison"
    echo "=========================="
    
    echo "📊 Build Results Summary:"
    echo ""
    
    # Check if binaries exist from different platforms
    if [[ -f "bin/stormdb-debug" ]]; then
        echo "Current binary info:"
        file bin/stormdb-debug 2>/dev/null || echo "file command not available"
        ls -la bin/stormdb-debug
        
        echo ""
        echo "Binary size comparison:"
        if [[ -f "bin/stormdb" ]]; then
            echo "Release: $(ls -lh bin/stormdb | awk '{print $5}')"
        fi
        if [[ -f "bin/stormdb-debug" ]]; then
            echo "Debug:   $(ls -lh bin/stormdb-debug | awk '{print $5}')"
        fi
        if [[ -f "bin/stormdb-asan" ]]; then
            echo "ASAN:    $(ls -lh bin/stormdb-asan | awk '{print $5}')"
        fi
    fi
}

# Main execution
main() {
    echo "🔄 StormDB Cross-Platform Testing"
    echo "=================================="
    echo ""
    
    case "${1:-all}" in
        "native"|"arm64"|"macos")
            test_native_arm64
            ;;
        "x86_64"|"x86"|"linux")
            test_x86_64_linux
            ;;
        "all")
            log_info "Running comprehensive cross-platform tests..."
            echo ""
            
            # Test native ARM64 first
            test_native_arm64
            echo ""
            
            # Test x86_64 Linux
            test_x86_64_linux
            echo ""
            
            # Compare results
            compare_results
            
            log_success "Cross-platform testing completed!"
            ;;
        "help"|"--help"|"-h")
            echo "Usage: $0 [PLATFORM]"
            echo ""
            echo "Platforms:"
            echo "  native      - Test on native ARM64 macOS"
            echo "  x86_64      - Test on x86_64 Linux (emulated)"
            echo "  all         - Test on all available platforms (default)"
            echo "  help        - Show this help"
            echo ""
            echo "Examples:"
            echo "  $0                    # Test all platforms"
            echo "  $0 x86_64            # Test only x86_64 Linux"
            echo "  $0 native            # Test only native macOS"
            ;;
        *)
            log_error "Unknown platform: $1"
            echo "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
