#!/bin/bash
# Simple local package testing script
# Tests package creation and basic validation without Docker

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

echo -e "${BLUE}=== StormDB Local Package Testing ===${NC}"
echo -e "${BLUE}Project Root: $PROJECT_ROOT${NC}"
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

# Change to project root
cd "$PROJECT_ROOT"

print_section "Building StormDB Project"

print_info "Cleaning previous builds..."
make clean

print_info "Building binary..."
make build

print_info "Building plugins..."
make plugins

print_success "Project build completed"
echo ""

print_section "Creating Packages"

# Try to build DEB package
print_info "Attempting to build DEB package..."
if make release-package-deb 2>/dev/null; then
    print_success "DEB package built successfully"
    DEB_SUCCESS=true
else
    print_error "DEB package build failed (expected on macOS without fpm)"
    DEB_SUCCESS=false
fi

# Try to build RPM package
print_info "Attempting to build RPM package..."
if make release-package-rpm 2>/dev/null; then
    print_success "RPM package built successfully"
    RPM_SUCCESS=true
else
    print_error "RPM package build failed (expected on macOS without rpmbuild)"
    RPM_SUCCESS=false
fi

echo ""
print_section "Package Validation"

# Check if packages were created
if [ -d "build/packages" ]; then
    print_info "Packages found:"
    ls -la build/packages/ | grep -E '\.(deb|rpm)$' | sed 's/^/    /' || print_info "No package files found"
    
    # Check DEB package structure if it exists
    for deb_file in build/packages/*.deb; do
        if [ -f "$deb_file" ]; then
            print_info "DEB package structure validation:"
            # Use ar to extract and check contents (macOS compatible)
            if command -v ar >/dev/null 2>&1; then
                ar tv "$deb_file" | sed 's/^/    /'
            else
                print_info "Cannot validate DEB structure (ar not available)"
            fi
        fi
        break  # Only check first DEB file
    done
    
    # Check build directory structure
    print_info "Build directory structure:"
    find build -type f -name "*.so" -o -name "stormdb" | head -10 | sed 's/^/    /'
    
else
    print_info "No packages directory found"
fi

print_section "Binary Validation"

# Test the binary
if [ -f "build/stormdb" ]; then
    print_info "Testing binary functionality..."
    
    # Test basic execution
    if ./build/stormdb --help >/dev/null 2>&1; then
        print_success "Binary executes correctly"
    else
        print_error "Binary execution failed"
    fi
    
    # Check binary info
    print_info "Binary information:"
    file build/stormdb | sed 's/^/    /'
    ls -la build/stormdb | sed 's/^/    /'
    
else
    print_error "Binary not found at build/stormdb"
fi

print_section "Plugin Validation"

# Test plugins
if [ -d "build/plugins" ]; then
    plugin_count=$(ls build/plugins/*.so 2>/dev/null | wc -l || echo 0)
    print_info "Found $plugin_count plugins:"
    if ls build/plugins/*.so >/dev/null 2>&1; then
        ls -la build/plugins/*.so | sed 's/^/    /'
    else
        print_info "No plugin files found"
    fi
    
    # Check if we have the renamed plugin
    if [ -f "build/plugins/ecommerce_basic_plugin.so" ]; then
        print_success "Renamed ecommerce_basic_plugin found"
    else
        print_error "Renamed ecommerce_basic_plugin NOT found"
    fi
    
else
    print_error "Plugins directory not found"
fi

print_section "Configuration Validation"

# Check configuration files
if ls config/*.yaml >/dev/null 2>&1; then
    config_count=$(ls config/*.yaml | wc -l)
    print_info "Found $config_count configuration files"
else
    config_count=0
    print_info "Found $config_count configuration files"
fi

# Check for key configurations
key_configs=("config_tpcc.yaml" "config_ecommerce_basic.yaml" "config_simple_connection.yaml")
for config in "${key_configs[@]}"; do
    if [ -f "config/$config" ]; then
        print_success "Key config found: $config"
    else
        print_error "Key config missing: $config"
    fi
done

print_section "Documentation Validation"

# Check documentation
if [ -f "stormdb.1" ]; then
    print_success "Man page found: stormdb.1"
    print_info "Man page info:"
    head -3 stormdb.1 | sed 's/^/    /'
else
    print_error "Man page not found"
fi

if [ -d "docs" ]; then
    doc_count=$(find docs -name "*.md" | wc -l)
    print_info "Found $doc_count documentation files"
else
    print_error "Documentation directory not found"
fi

print_section "Summary"

echo ""
print_info "Local build and validation completed"
print_info "For full distribution testing, use: make test-packages"
print_info "For Docker-based testing, ensure Docker is running and try:"
print_info "  docker --version"
print_info "  docker-compose --version"
echo ""

if [ "$DEB_SUCCESS" = true ] || [ "$RPM_SUCCESS" = true ]; then
    print_success "At least one package type built successfully"
    echo ""
    print_info "Next steps:"
    print_info "1. Run 'make test-packages' for full multi-distro testing"
    print_info "2. Review packages in build/packages/"
    print_info "3. Test installation on target Linux systems"
    exit 0
else
    print_info "Package building requires Linux environment or Docker"
    print_info "Current build artifacts are ready for Docker-based testing"
    exit 0
fi
