#!/bin/bash
# ========================================================================
# StormDB Local Linux Development Environment
# Creates a local Linux environment for development and debugging
# Works without external container registries
# ========================================================================

set -e

# Configuration
LINUX_ENV_DIR="$HOME/.stormdb-linux-env"
CHROOT_DIR="$LINUX_ENV_DIR/chroot"
WORKSPACE_DIR="/workspace"

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

# Check if running on macOS
if [[ "$(uname)" != "Darwin" ]]; then
    log_error "This script is designed for macOS. For Linux hosts, use containers directly."
    exit 1
fi

# Function to set up Lima Linux VM
setup_lima_vm() {
    log_info "Setting up Lima Linux VM for development..."
    
    # Check if Lima is installed
    if ! command -v lima &> /dev/null; then
        log_info "Installing Lima via Homebrew..."
        brew install lima
    fi
    
    # Create Lima configuration for StormDB development
    cat > "$HOME/.lima/stormdb-dev.yaml" <<EOF
# StormDB Development VM Configuration
vmType: "qemu"
os: "Linux"
arch: "$(uname -m)"
images:
  - location: "https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-$(uname -m | sed 's/arm64/arm64/; s/x86_64/amd64/').img"
    arch: "$(uname -m)"
cpus: 4
memory: "4GiB"
disk: "20GiB"

# Mount workspace
mounts:
  - location: "$(pwd)"
    writable: true
    mountPoint: "/workspace"

# Port forwarding for debugging
portForwards:
  - guestPort: 22
    hostPort: 2225
  - guestPort: 5432
    hostPort: 5435

# Install development tools
provision:
  - mode: system
    script: |
      #!/bin/bash
      apt-get update
      apt-get install -y build-essential gcc g++ make gdb valgrind clang
      apt-get install -y git vim curl wget postgresql-client
      apt-get install -y libyaml-dev libpq-dev pkg-config
      
      # Create development user
      useradd -m -s /bin/bash -G sudo dev
      echo "dev ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers
      
      # Set up workspace
      mkdir -p /workspace
      chown dev:dev /workspace

# SSH configuration
ssh:
  localPort: 2225
  loadDotSSHPubKeys: true
EOF
    
    log_success "Lima configuration created"
    
    # Start the VM
    log_info "Starting StormDB development VM..."
    lima start stormdb-dev
    
    log_success "Linux development VM is ready!"
    log_info "Connect with: lima shell stormdb-dev"
    log_info "Or SSH with: ssh -p 2225 dev@localhost"
}

# Function to set up Multipass VM (alternative)
setup_multipass_vm() {
    log_info "Setting up Multipass Linux VM..."
    
    # Check if Multipass is installed
    if ! command -v multipass &> /dev/null; then
        log_info "Installing Multipass via Homebrew..."
        brew install multipass
    fi
    
    # Launch Ubuntu VM with development tools
    multipass launch --name stormdb-dev --cpus 4 --memory 4G --disk 20G
    
    # Install development tools
    multipass exec stormdb-dev -- sudo apt-get update
    multipass exec stormdb-dev -- sudo apt-get install -y \
        build-essential gcc g++ make gdb valgrind clang \
        git vim curl wget postgresql-client \
        libyaml-dev libpq-dev pkg-config
    
    # Mount workspace
    multipass mount "$(pwd)" stormdb-dev:/workspace
    
    log_success "Multipass VM is ready!"
    log_info "Connect with: multipass shell stormdb-dev"
    log_info "Workspace is mounted at: /workspace"
}

# Function to use existing container with offline setup
setup_offline_container() {
    log_info "Setting up offline container development..."
    
    # Use your existing UBI image but configure it for offline use
    # Create a development script that works within the constraints
    
    cat > "$HOME/.stormdb-container-dev.sh" <<'EOF'
#!/bin/bash
# Run development session in existing container
set -e

CONTAINER_ID=$(podman run -d --rm \
    --user root \
    -v "$(pwd):/workspace" \
    -w /workspace \
    -e TERM=xterm-256color \
    b39f0b68f0c6 \
    sleep infinity)

echo "Container started: $CONTAINER_ID"

# Check what's already available
echo "Checking available tools..."
podman exec -it "$CONTAINER_ID" bash -c "
    echo 'Available compilers:'
    which gcc || echo 'No GCC'
    which clang || echo 'No Clang'
    which make || echo 'No Make'
    which gdb || echo 'No GDB'
    
    echo -e '\nSystem info:'
    cat /etc/os-release
    
    echo -e '\nStarting development shell...'
    cd /workspace
    exec bash
"

# Clean up on exit
trap "podman stop $CONTAINER_ID" EXIT
EOF
    
    chmod +x "$HOME/.stormdb-container-dev.sh"
    
    log_success "Offline container development script created"
    log_info "Run with: $HOME/.stormdb-container-dev.sh"
}

# Function to create cross-compilation setup
setup_cross_compilation() {
    log_info "Setting up cross-compilation for Linux on macOS..."
    
    # Install cross-compilation toolchain
    if ! brew list | grep -q "x86_64-linux-gnu-gcc\|aarch64-linux-gnu-gcc"; then
        log_info "Installing cross-compilation toolchain..."
        brew tap messense/macos-cross-toolchains
        
        if [[ "$(uname -m)" == "arm64" ]]; then
            brew install x86_64-unknown-linux-gnu
            brew install aarch64-unknown-linux-gnu
        else
            brew install x86_64-unknown-linux-gnu
        fi
    fi
    
    # Create cross-compilation Makefile extension
    cat > "Makefile.cross" <<'EOF'
# Cross-compilation targets for Linux
CROSS_X86_64_CC = x86_64-unknown-linux-gnu-gcc
CROSS_AARCH64_CC = aarch64-unknown-linux-gnu-gcc

# Cross-compile for x86_64 Linux
cross-linux-x86_64:
	CC=$(CROSS_X86_64_CC) $(MAKE) all

# Cross-compile for ARM64 Linux  
cross-linux-aarch64:
	CC=$(CROSS_AARCH64_CC) $(MAKE) all

# Cross-compile for both architectures
cross-linux-all: cross-linux-x86_64 cross-linux-aarch64

.PHONY: cross-linux-x86_64 cross-linux-aarch64 cross-linux-all
EOF
    
    log_success "Cross-compilation setup completed"
    log_info "Use: make -f Makefile.cross cross-linux-x86_64"
}

# Main menu
show_menu() {
    echo "StormDB Linux Development Setup for China"
    echo "=========================================="
    echo "Choose your preferred solution:"
    echo ""
    echo "1) Lima VM (Recommended) - Full Linux environment"
    echo "2) Multipass VM - Alternative Linux environment"
    echo "3) Offline Container - Use existing images without network"
    echo "4) Cross-compilation - Compile for Linux on macOS"
    echo "5) All solutions - Set up everything"
    echo "q) Quit"
    echo ""
}

# Main execution
case "${1:-menu}" in
    "lima")
        setup_lima_vm
        ;;
    "multipass")
        setup_multipass_vm
        ;;
    "offline")
        setup_offline_container
        ;;
    "cross")
        setup_cross_compilation
        ;;
    "all")
        setup_lima_vm
        setup_multipass_vm
        setup_offline_container
        setup_cross_compilation
        ;;
    "menu"|*)
        show_menu
        read -p "Enter your choice (1-5, q): " choice
        case $choice in
            1) setup_lima_vm ;;
            2) setup_multipass_vm ;;
            3) setup_offline_container ;;
            4) setup_cross_compilation ;;
            5) 
                setup_lima_vm
                setup_multipass_vm
                setup_offline_container
                setup_cross_compilation
                ;;
            q|Q) log_info "Goodbye!" ;;
            *) log_error "Invalid choice" ;;
        esac
        ;;
esac
