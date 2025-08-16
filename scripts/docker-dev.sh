#!/bin/bash
# ========================================================================
# StormDB Docker Development Helper Script
# Quick commands for Docker-based development workflow
# ========================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
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

# Show usage
show_usage() {
    echo "StormDB Docker Development Helper"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  build-dev        Build development container"
    echo "  build-all        Build all container variants"
    echo "  run-dev          Start development container (interactive)"
    echo "  run-vscode       Start VS Code development container"
    echo "  test-linux       Run tests in Linux container"
    echo "  test-all         Run tests on all platforms"
    echo "  cross-compile    Build for multiple architectures"
    echo "  clean            Clean up Docker containers and images"
    echo "  shell            Open shell in development container"
    echo "  logs             Show container logs"
    echo "  status           Show container status"
    echo "  help             Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 build-dev     # Build development environment"
    echo "  $0 run-dev       # Start interactive development session"
    echo "  $0 test-linux    # Run full test suite on Linux"
    echo ""
}

# Build development container
build_dev() {
    log_info "Building development container..."
    docker compose build stormdb-dev
    log_success "Development container built successfully"
}

# Build all containers
build_all() {
    log_info "Building all container variants..."
    docker compose build
    log_success "All containers built successfully"
}

# Run development container interactively
run_dev() {
    log_info "Starting development container..."
    docker compose up -d stormdb-dev
    log_success "Development container started"
    log_info "Opening interactive shell..."
    docker compose exec stormdb-dev bash
}

# Run VS Code development container
run_vscode() {
    log_info "Starting VS Code development container..."
    docker compose up -d stormdb-vscode
    log_success "VS Code container started on port 2222"
    log_info "Connect VS Code to: ssh://root@localhost:2222"
    log_info "Password: stormdb"
}

# Test on Linux
test_linux() {
    log_info "Running tests on Linux..."
    docker compose up --build stormdb-test
    log_success "Linux tests completed"
}

# Test on all platforms
test_all() {
    log_info "Running tests on all platforms..."
    
    # Build for multiple architectures
    log_info "Building for linux/amd64..."
    docker buildx build --platform linux/amd64 --target testing .
    
    log_info "Building for linux/arm64..."
    docker buildx build --platform linux/arm64 --target testing .
    
    log_success "Multi-platform tests completed"
}

# Cross-compile for multiple architectures
cross_compile() {
    log_info "Cross-compiling for multiple architectures..."
    docker compose up --build stormdb-cross
    
    log_info "Entering cross-compilation container..."
    docker compose exec stormdb-cross bash -c "
        echo 'Cross-compilation environment ready'
        echo 'Available cross-compilers:'
        echo '  gcc-aarch64-linux-gnu (ARM64)'
        echo '  gcc-arm-linux-gnueabihf (ARM32)'
        echo ''
        echo 'Example usage:'
        echo '  CC=aarch64-linux-gnu-gcc make clean all'
        echo '  CC=arm-linux-gnueabihf-gcc make clean all'
        bash
    "
}

# Clean up containers and images
clean() {
    log_warning "Cleaning up Docker containers and images..."
    
    # Stop all containers
    docker compose down -v
    
    # Remove unused images
    docker image prune -f
    
    # Remove build cache
    docker builder prune -f
    
    log_success "Cleanup completed"
}

# Open shell in development container
shell() {
    log_info "Opening shell in development container..."
    
    # Check if container is running
    if ! docker compose ps stormdb-dev | grep -q "Up"; then
        log_info "Starting development container first..."
        docker compose up -d stormdb-dev
    fi
    
    docker compose exec stormdb-dev bash
}

# Show container logs
logs() {
    local service=${1:-stormdb-dev}
    log_info "Showing logs for $service..."
    docker compose logs -f "$service"
}

# Show container status
status() {
    log_info "Container status:"
    docker compose ps
    echo ""
    log_info "Image sizes:"
    docker images | grep stormdb
}

# Main command dispatcher
main() {
    case "${1:-help}" in
        "build-dev")
            build_dev
            ;;
        "build-all")
            build_all
            ;;
        "run-dev")
            run_dev
            ;;
        "run-vscode")
            run_vscode
            ;;
        "test-linux")
            test_linux
            ;;
        "test-all")
            test_all
            ;;
        "cross-compile")
            cross_compile
            ;;
        "clean")
            clean
            ;;
        "shell")
            shell
            ;;
        "logs")
            logs "$2"
            ;;
        "status")
            status
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

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    log_error "Docker is not installed or not in PATH"
    exit 1
fi

# Check if Docker Compose is available
if ! docker compose version &> /dev/null; then
    log_error "Docker Compose is not available"
    exit 1
fi

# Run main function
main "$@"
