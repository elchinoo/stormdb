#!/bin/bash
# ========================================================================
# StormDB China-Optimized Linux Development
# Uses DaoCloud mirror for reliable container access
# ========================================================================

set -e

# Configuration
IMAGE_NAME="stormdb:china-dev"
CONTAINER_NAME="stormdb-linux-dev"
PROJECT_DIR="$(pwd)"

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

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to build the development container
build_container() {
    log_info "Building China-optimized Linux development container..."
    
    local BASE_IMAGE_ARG
    BASE_IMAGE_ARG=${BASE_IMAGE:-daocloud.io/library/ubuntu:22.04}
    docker build \
        -f Dockerfile.china \
        --build-arg BASE_IMAGE="$BASE_IMAGE_ARG" \
        -t "$IMAGE_NAME" \
        --target development \
        .
    
    log_success "Container built successfully: $IMAGE_NAME"
}

# Function to start development session
start_dev_session() {
    log_info "Starting Linux development session..."
    
    # Stop existing container if running
    if docker ps -q --filter "name=$CONTAINER_NAME" | grep -q .; then
        log_warning "Stopping existing container: $CONTAINER_NAME"
        docker stop "$CONTAINER_NAME" || true
        docker rm "$CONTAINER_NAME" || true
    fi
    
    # Start development container
    docker run -it --rm \
        --name "$CONTAINER_NAME" \
        --volume "$PROJECT_DIR:/workspace" \
        --workdir "/workspace" \
        --env TERM=xterm-256color \
        --env STORMDB_ENV=development \
        "$IMAGE_NAME" \
        bash
}

# Function to run tests in container
run_tests() {
    log_info "Running tests in Linux container..."
    
    docker run --rm \
        --volume "$PROJECT_DIR:/workspace" \
        --workdir "/workspace" \
        --env STORMDB_ENV=testing \
        "$IMAGE_NAME" \
        bash -c "
            echo '🔧 Building StormDB in Linux container...'
            make clean
            make all
            make debug
            
            echo '🧪 Running basic tests...'
            ./bin/stormdb --version
            ./bin/stormdb-debug --version
            
            echo '🛡️ Testing with AddressSanitizer...'
            if make asan; then
                ./bin/stormdb-asan --version
                echo '✅ AddressSanitizer build successful'
            else
                echo '❌ AddressSanitizer build failed'
            fi
            
            echo '🏁 Linux testing completed!'
        "
}

# Function to build production image
build_production() {
    log_info "Building production container..."
    
    docker build \
        -f Dockerfile.china \
        -t "stormdb:china-prod" \
        --target production \
        .
    
    log_success "Production container built: stormdb:china-prod"
    
    # Test production container
    log_info "Testing production container..."
    docker run --rm stormdb:china-prod
}

# Function to clean up containers and images
cleanup() {
    log_info "Cleaning up containers and images..."
    
    # Stop and remove containers
    docker ps -aq --filter "name=stormdb*" | xargs -r docker stop || true
    docker ps -aq --filter "name=stormdb*" | xargs -r docker rm || true
    
    # Remove images
    docker images -q "stormdb:china*" | xargs -r docker rmi || true
    
    log_success "Cleanup completed"
}

# Function to show container info
show_info() {
    log_info "StormDB China-Optimized Container Information"
    echo "=============================================="
    echo ""
    echo "🐳 Available Images:"
    docker images | grep -E "(stormdb|alpine)" || echo "No StormDB images found"
    echo ""
    echo "🏃 Running Containers:"
    docker ps --filter "name=stormdb*" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" || echo "No StormDB containers running"
    echo ""
    echo "💾 Image Sizes:"
    docker images --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}" | grep -E "(stormdb|alpine)" || echo "No size info available"
}

# Show usage
usage() {
    cat <<EOF
StormDB China-Optimized Linux Development

Usage: $0 [COMMAND]

Commands:
    build       - Build development container
    dev         - Start development session
    test        - Run tests in container
    prod        - Build production container
    info        - Show container information
    clean       - Clean up containers and images
    help        - Show this help

Examples:
    $0 build     # Build the development container
    $0 dev       # Start interactive development session
    $0 test      # Run comprehensive tests
    $0 prod      # Build minimal production image

The development container includes:
    - Alpine Linux 3.19 (11.8MB base)
    - GCC, Make, GDB, Valgrind
    - AddressSanitizer support
    - Full debugging capabilities
EOF
}

# Main execution
case "${1:-help}" in
    build)
        build_container
        ;;
    dev)
        start_dev_session
        ;;
    test)
        run_tests
        ;;
    prod)
        build_production
        ;;
    info)
        show_info
        ;;
    clean)
        cleanup
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        log_error "Unknown command: $1"
        usage
        exit 1
        ;;
esac
