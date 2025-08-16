#!/bin/bash
# ========================================================================
# StormDB Valgrind Testing Script
# Runs comprehensive memory checks using valgrind in Linux container
# ========================================================================

set -e

PROJECT_DIR="$(pwd)"
IMAGE_NAME="local/stormdb-base:latest"
IMAGE_PLATFORM="linux/amd64"

# Color codes
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

check_base_image() {
    if ! docker image inspect "$IMAGE_NAME" >/dev/null 2>&1; then
        log_error "Base image '$IMAGE_NAME' not found. Building it now..."
        # Build and load the image into the local Docker engine (avoid registry lookups)
        if command -v docker >/dev/null 2>&1 && docker buildx version >/dev/null 2>&1; then
            docker buildx build -f Dockerfile.base -t "$IMAGE_NAME" --platform "$IMAGE_PLATFORM" --load . --progress=plain
        else
            docker build -f Dockerfile.base -t "$IMAGE_NAME" . --progress=plain
        fi
        log_success "Base image built and cached as $IMAGE_NAME"
    fi
}

run_valgrind_test() {
    local test_type="${1:-basic}"
    local binary="${2:-bin/stormdb-debug}"
    local extra_args="${3:-}"
    
    log_info "Running valgrind $test_type test..."
    
    # Resolve image reference to local image ID to avoid any registry lookups
    local IMAGE_REF
    IMAGE_REF="$(docker images -q "$IMAGE_NAME" 2>/dev/null)"
    if [ -z "$IMAGE_REF" ]; then
        check_base_image
        IMAGE_REF="$(docker images -q "$IMAGE_NAME" 2>/dev/null)"
    fi

    docker run --rm --platform "$IMAGE_PLATFORM" \
        -v "$PROJECT_DIR:/workspace" \
        -w /workspace \
        "$IMAGE_REF" \
        bash -c "
            echo '[DEPS] Installing dependencies...'
            apt-get update -qq && apt-get install -y -qq libyaml-dev libpq-dev pkg-config valgrind
            
            echo '[BUILD] Building debug version...'
            make clean && make debug
            
            echo '[VALGRIND] Running $test_type test...'
            case '$test_type' in
                'basic')
                    valgrind --leak-check=full \\
                             --show-leak-kinds=all \\
                             --track-origins=yes \\
                             --log-file=valgrind-$test_type.log \\
                             $extra_args \\
                             $binary --version
                    ;;
                'detailed')
                    valgrind --leak-check=full \\
                             --show-leak-kinds=all \\
                             --track-origins=yes \\
                             --track-fds=yes \\
                             --verbose \\
                             --log-file=valgrind-$test_type.log \\
                             $extra_args \\
                             $binary --version
                    ;;
                'memcheck')
                    valgrind --tool=memcheck \\
                             --leak-check=full \\
                             --show-leak-kinds=all \\
                             --track-origins=yes \\
                             --log-file=valgrind-$test_type.log \\
                             $extra_args \\
                             $binary --version
                    ;;
                'cachegrind')
                    valgrind --tool=cachegrind \\
                             --log-file=valgrind-$test_type.log \\
                             $extra_args \\
                             $binary --version
                    ;;
                'callgrind')
                    valgrind --tool=callgrind \\
                             --log-file=valgrind-$test_type.log \\
                             $extra_args \\
                             $binary --version
                    ;;
                'helgrind')
                    valgrind --tool=helgrind \\
                             --log-file=valgrind-$test_type.log \\
                             $extra_args \\
                             $binary --version
                    ;;
                *)
                    echo 'Unknown test type: $test_type'
                    exit 1
                    ;;
            esac
            
            echo '[RESULTS] Valgrind output:'
            cat valgrind-$test_type.log
        "
}

print_usage() {
    echo "Usage: $0 [TEST_TYPE] [BINARY] [EXTRA_ARGS]"
    echo ""
    echo "Test Types:"
    echo "  basic       - Basic memory leak check (default)"
    echo "  detailed    - Detailed memory check with file descriptors"
    echo "  memcheck    - Full memory error detection"
    echo "  cachegrind  - Cache and branch-prediction profiling"
    echo "  callgrind   - Call-graph generation"
    echo "  helgrind    - Thread error detection"
    echo ""
    echo "Binaries:"
    echo "  bin/stormdb-debug    - Debug build (default)"
    echo "  bin/stormdb          - Regular build"
    echo "  bin/stormdb-release  - Release build"
    echo ""
    echo "Examples:"
    echo "  $0                               # Basic test with debug binary"
    echo "  $0 detailed                      # Detailed test with debug binary"
    echo "  $0 memcheck bin/stormdb-release  # Memcheck test with release binary"
    echo "  $0 basic bin/stormdb-debug '--suppressions=valgrind.supp'"
}

main() {
    case "${1:-basic}" in
        "help"|"--help"|"-h")
            print_usage
            exit 0
            ;;
        *)
            echo "🧪 StormDB Valgrind Testing"
            echo "============================"
            echo ""
            
            check_base_image
            run_valgrind_test "${1:-basic}" "${2:-bin/stormdb-debug}" "${3:-}"
            
            log_success "Valgrind test completed!"
            echo ""
            echo "📄 Log files generated:"
            ls -la valgrind-*.log 2>/dev/null || echo "No log files found"
            ;;
    esac
}

main "$@"
