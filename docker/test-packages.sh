#!/bin/bash
# Package Testing Script for StormDB
# This script automates the building and testing of DEB and RPM packages
# across multiple Linux distributions using Docker

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DOCKER_DIR="$SCRIPT_DIR"
RESULTS_DIR="$DOCKER_DIR/test-results"

# Create results directory
mkdir -p "$RESULTS_DIR"

echo -e "${BLUE}=== StormDB Package Testing Suite ===${NC}"
echo -e "${BLUE}Project Root: $PROJECT_ROOT${NC}"
echo -e "${BLUE}Docker Dir: $DOCKER_DIR${NC}"
echo ""

# Function to print section headers
print_section() {
    echo -e "${YELLOW}=== $1 ===${NC}"
}

# Function to print success
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# Function to print error
print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to print info
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Clean up function
cleanup() {
    print_info "Cleaning up containers and images..."
    docker-compose -f "$DOCKER_DIR/docker-compose.yml" down --rmi all --volumes --remove-orphans 2>/dev/null || true
}

# Set up trap for cleanup
trap cleanup EXIT

# Parse command line arguments
DISTRIBUTIONS=()
SKIP_BUILD=false
SKIP_CLEANUP=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --distro|-d)
            DISTRIBUTIONS+=("$2")
            shift 2
            ;;
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --skip-cleanup)
            SKIP_CLEANUP=false
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --help|-h)
            cat << EOF
Usage: $0 [OPTIONS]

Options:
    -d, --distro DISTRO     Test specific distribution (ubuntu, debian, centos)
                           Can be specified multiple times. Default: all
    --skip-build           Skip the initial project build
    --skip-cleanup         Skip cleanup after testing
    -v, --verbose          Enable verbose output
    -h, --help             Show this help message

Examples:
    $0                     # Test all distributions
    $0 -d ubuntu           # Test only Ubuntu
    $0 -d ubuntu -d debian # Test Ubuntu and Debian
    $0 --skip-build        # Skip initial build, use existing artifacts
EOF
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Default to all distributions if none specified
if [ ${#DISTRIBUTIONS[@]} -eq 0 ]; then
    DISTRIBUTIONS=("ubuntu" "debian" "centos")
fi

print_info "Testing distributions: ${DISTRIBUTIONS[*]}"
echo ""

# Change to project root for building
cd "$PROJECT_ROOT"

# Build the project if not skipping
if [ "$SKIP_BUILD" = false ]; then
    print_section "Building StormDB Project"
    
    print_info "Cleaning previous builds..."
    make clean
    
    print_info "Building binary..."
    make build
    
    print_info "Building plugins..."
    make plugins
    
    print_success "Project build completed"
    echo ""
fi

# Change to docker directory
cd "$DOCKER_DIR"

# Function to test a specific distribution
test_distribution() {
    local distro=$1
    local service_name=""
    
    case $distro in
        ubuntu)
            service_name="ubuntu-deb"
            ;;
        debian)
            service_name="debian-deb"
            ;;
        centos)
            service_name="centos-rpm"
            ;;
        *)
            print_error "Unknown distribution: $distro"
            return 1
            ;;
    esac
    
    print_section "Testing $distro Distribution"
    
    print_info "Building $distro container..."
    if [ "$VERBOSE" = true ]; then
        docker-compose build "$service_name"
    else
        docker-compose build "$service_name" > "$RESULTS_DIR/$distro-build.log" 2>&1
    fi
    
    if [ $? -eq 0 ]; then
        print_success "$distro container built successfully"
    else
        print_error "$distro container build failed"
        if [ "$VERBOSE" = false ]; then
            print_info "Check build log: $RESULTS_DIR/$distro-build.log"
        fi
        return 1
    fi
    
    print_info "Running $distro package test..."
    if [ "$VERBOSE" = true ]; then
        docker-compose run --rm "$service_name" | tee "$RESULTS_DIR/$distro-test.log"
    else
        docker-compose run --rm "$service_name" > "$RESULTS_DIR/$distro-test.log" 2>&1
    fi
    
    if [ $? -eq 0 ]; then
        print_success "$distro package test passed"
        
        # Extract some key info from the test log
        print_info "Test Results Summary:"
        if [ -f "$RESULTS_DIR/$distro-test.log" ]; then
            grep -E "(Binary installed|Version|Plugins available|Config files)" "$RESULTS_DIR/$distro-test.log" | sed 's/^/    /'
        fi
    else
        print_error "$distro package test failed"
        print_info "Check test log: $RESULTS_DIR/$distro-test.log"
        return 1
    fi
    
    echo ""
}

# Test each distribution
FAILED_DISTROS=()
PASSED_DISTROS=()

for distro in "${DISTRIBUTIONS[@]}"; do
    if test_distribution "$distro"; then
        PASSED_DISTROS+=("$distro")
    else
        FAILED_DISTROS+=("$distro")
    fi
done

# Generate summary report
print_section "Test Summary Report"

echo "Timestamp: $(date)" > "$RESULTS_DIR/summary.txt"
echo "Tested Distributions: ${DISTRIBUTIONS[*]}" >> "$RESULTS_DIR/summary.txt"
echo "" >> "$RESULTS_DIR/summary.txt"

if [ ${#PASSED_DISTROS[@]} -gt 0 ]; then
    print_success "Passed distributions: ${PASSED_DISTROS[*]}"
    echo "Passed: ${PASSED_DISTROS[*]}" >> "$RESULTS_DIR/summary.txt"
fi

if [ ${#FAILED_DISTROS[@]} -gt 0 ]; then
    print_error "Failed distributions: ${FAILED_DISTROS[*]}"
    echo "Failed: ${FAILED_DISTROS[*]}" >> "$RESULTS_DIR/summary.txt"
    
    print_info "Check individual test logs in: $RESULTS_DIR/"
    echo ""
    echo "Failed Test Logs:"
    for distro in "${FAILED_DISTROS[@]}"; do
        echo "  - $RESULTS_DIR/$distro-test.log"
        echo "  - $RESULTS_DIR/$distro-build.log"
    done
fi

echo "" >> "$RESULTS_DIR/summary.txt"
echo "Detailed logs available in: $RESULTS_DIR/" >> "$RESULTS_DIR/summary.txt"

# Check if packages were generated
print_section "Generated Packages"
if [ -d "$PROJECT_ROOT/build/packages" ]; then
    print_info "Packages found in build/packages/:"
    ls -la "$PROJECT_ROOT/build/packages/"*.{deb,rpm} 2>/dev/null | sed 's/^/    /' || print_info "No packages found"
else
    print_info "No packages directory found"
fi

# Final exit status
if [ ${#FAILED_DISTROS[@]} -eq 0 ]; then
    print_success "All package tests passed! 🎉"
    echo ""
    print_info "Your packages are ready for distribution:"
    ls -1 "$PROJECT_ROOT/build/packages/"*.{deb,rpm} 2>/dev/null | sed 's/^/    ✅ /' || echo "    No packages found"
    exit 0
else
    print_error "Some package tests failed"
    exit 1
fi
