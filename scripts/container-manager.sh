#!/bin/bash
# ========================================================================
# Container Runtime Detection and Management Script
# Supports both Docker and Podman with alternative registries
# ========================================================================

set -e

# Configuration
CONTAINER_RUNTIME=""
IMAGE_NAME="stormdb"
IMAGE_TAG="latest"
REGISTRY=""
DEFAULT_DOCKERFILE="Dockerfile.alpine"

# China-friendly registries (uncomment the one you prefer)
# CHINA_REGISTRIES=(
#     "registry.cn-hangzhou.aliyuncs.com"  # Alibaba Cloud
#     "ccr.ccs.tencentyun.com"             # Tencent Cloud
#     "registry.cn-beijing.volcengine.com" # ByteDance/Volcano Engine
#     "docker.m.daocloud.io"               # DaoCloud
#     "registry.docker-cn.com"             # Docker China Official
# )

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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

# Detect available container runtime
detect_runtime() {
    if command -v podman &> /dev/null; then
        CONTAINER_RUNTIME="podman"
        log_success "Found Podman runtime"
    elif command -v docker &> /dev/null; then
        CONTAINER_RUNTIME="docker"
        log_success "Found Docker runtime"
    else
        log_error "Neither Docker nor Podman found. Please install one of them."
        exit 1
    fi
}

# Check if we're in China (heuristic based on common indicators)
is_in_china() {
    # Check timezone
    if [[ "$TZ" == *"Asia/Shanghai"* ]] || [[ "$TZ" == *"Asia/Beijing"* ]]; then
        return 0
    fi
    
    # Check locale
    if [[ "$LANG" == *"zh_CN"* ]]; then
        return 0
    fi
    
    # Check if Great Firewall blocks Docker Hub (timeout-based)
    if ! timeout 3 curl -s --max-time 3 https://hub.docker.com &> /dev/null; then
        log_warning "Docker Hub appears to be unreachable (possible GFW blocking)"
        return 0
    fi
    
    return 1
}

# Setup registry mirrors for China
setup_china_registry() {
    if [[ "$CONTAINER_RUNTIME" == "docker" ]]; then
        log_info "Setting up Docker registry mirrors for China..."
        
        # Create or update Docker daemon configuration
        local daemon_json="/etc/docker/daemon.json"
        local user_daemon_json="$HOME/.docker/daemon.json"
        
        cat > "$user_daemon_json" <<EOF
{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://registry.docker-cn.com",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com"
  ],
  "insecure-registries": [],
  "experimental": false
}
EOF
        log_success "Created Docker registry mirror configuration at $user_daemon_json"
        log_warning "You may need to restart Docker daemon for changes to take effect"
        
    elif [[ "$CONTAINER_RUNTIME" == "podman" ]]; then
        log_info "Setting up Podman registry configuration for China..."
        
        # Create registries.conf for Podman
        local registries_conf="$HOME/.config/containers/registries.conf"
        mkdir -p "$(dirname "$registries_conf")"
        
        cat > "$registries_conf" <<EOF
# Registries configuration for China
[registries.search]
registries = ['registry.cn-hangzhou.aliyuncs.com', 'ccr.ccs.tencentyun.com', 'docker.io']

[registries.insecure]
registries = []

[registries.block]
registries = []

[[registry]]
prefix = "docker.io"
location = "docker.m.daocloud.io"

[[registry]]
prefix = "docker.io"
location = "registry.docker-cn.com"
EOF
        log_success "Created Podman registry configuration at $registries_conf"
    fi
}

# Build container image
build_image() {
    local dockerfile="${1:-$DEFAULT_DOCKERFILE}"
    local stage="${2:-development}"
    local full_image_name="${REGISTRY:+$REGISTRY/}${IMAGE_NAME}:${IMAGE_TAG}-${stage}"
    
    log_info "Building image: $full_image_name using $dockerfile (stage: $stage)"
    
    if [[ "$CONTAINER_RUNTIME" == "podman" ]]; then
        podman build \
            --file "$dockerfile" \
            --target "$stage" \
            --tag "$full_image_name" \
            --layers \
            .
    else
        docker build \
            --file "$dockerfile" \
            --target "$stage" \
            --tag "$full_image_name" \
            .
    fi
    
    log_success "Built image: $full_image_name"
}

# Run development container
run_dev() {
    local image_name="${REGISTRY:+$REGISTRY/}${IMAGE_NAME}:${IMAGE_TAG}-development"
    local container_name="stormdb-dev"
    
    log_info "Starting development container: $container_name"
    
    # Stop existing container if running
    if $CONTAINER_RUNTIME ps -q --filter "name=$container_name" | grep -q .; then
        log_warning "Stopping existing container: $container_name"
        $CONTAINER_RUNTIME stop "$container_name" || true
        $CONTAINER_RUNTIME rm "$container_name" || true
    fi
    
    $CONTAINER_RUNTIME run -it --rm \
        --name "$container_name" \
        --volume "$(pwd):/workspace" \
        --workdir "/workspace" \
        --publish 2222:22 \
        "$image_name" \
        bash
}

# Run tests in container
run_tests() {
    local image_name="${REGISTRY:+$REGISTRY/}${IMAGE_NAME}:${IMAGE_TAG}-testing"
    
    log_info "Running tests in container"
    
    $CONTAINER_RUNTIME run --rm \
        --volume "$(pwd):/workspace" \
        --workdir "/workspace" \
        "$image_name"
}

# Clean up containers and images
cleanup() {
    log_info "Cleaning up containers and images..."
    
    # Stop and remove containers
    $CONTAINER_RUNTIME ps -aq --filter "name=stormdb*" | xargs -r $CONTAINER_RUNTIME stop || true
    $CONTAINER_RUNTIME ps -aq --filter "name=stormdb*" | xargs -r $CONTAINER_RUNTIME rm || true
    
    # Remove images
    $CONTAINER_RUNTIME images -q "$IMAGE_NAME:*" | xargs -r $CONTAINER_RUNTIME rmi || true
    
    log_success "Cleanup completed"
}

# Show usage information
usage() {
    cat <<EOF
Container Runtime Management Script (Docker/Podman)

Usage: $0 [COMMAND] [OPTIONS]

Commands:
    detect      - Detect available container runtime
    setup       - Setup registry mirrors (auto-detects China)
    build       - Build container image
    dev         - Start development container
    test        - Run tests in container
    clean       - Clean up containers and images
    help        - Show this help message

Options:
    --dockerfile FILE   - Specify Dockerfile (default: $DEFAULT_DOCKERFILE)
    --stage STAGE      - Build stage (default: development)
    --registry URL     - Container registry URL
    --tag TAG          - Image tag (default: $IMAGE_TAG)

Examples:
    $0 detect                                    # Detect runtime
    $0 setup                                     # Setup China mirrors
    $0 build --stage development                 # Build dev image
    $0 build --dockerfile Dockerfile.alpine     # Build with Alpine
    $0 dev                                       # Start dev container
    $0 test                                      # Run tests
    $0 clean                                     # Clean up

Environment Variables:
    STORMDB_REGISTRY    - Default registry URL
    STORMDB_TAG         - Default image tag
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dockerfile)
            DEFAULT_DOCKERFILE="$2"
            shift 2
            ;;
        --stage)
            STAGE="$2"
            shift 2
            ;;
        --registry)
            REGISTRY="$2"
            shift 2
            ;;
        --tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        detect|setup|build|dev|test|clean|help)
            COMMAND="$1"
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Set defaults from environment if available
REGISTRY="${REGISTRY:-$STORMDB_REGISTRY}"
IMAGE_TAG="${IMAGE_TAG:-$STORMDB_TAG:-latest}"

# Main execution
case "${COMMAND:-detect}" in
    detect)
        detect_runtime
        ;;
    setup)
        detect_runtime
        if is_in_china; then
            log_info "China detected - setting up registry mirrors"
            setup_china_registry
        else
            log_info "Not in China - using default registries"
        fi
        ;;
    build)
        detect_runtime
        build_image "$DEFAULT_DOCKERFILE" "${STAGE:-development}"
        ;;
    dev)
        detect_runtime
        run_dev
        ;;
    test)
        detect_runtime
        run_tests
        ;;
    clean)
        detect_runtime
        cleanup
        ;;
    help)
        usage
        ;;
    *)
        log_error "Unknown command: ${COMMAND}"
        usage
        exit 1
        ;;
esac
