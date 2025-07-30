#!/bin/bash
# Native Docker-based build system for StormDB packages
# Builds packages inside Linux containers using native compilation
set -e

echo "=== StormDB Native Docker Package Builder ==="

# Configuration
PROJECT_ROOT="$(pwd)"
BUILD_OUTPUT="${PROJECT_ROOT}/build/packages"
DOCKER_BUILD_DIR="/workspace/build"

# Build options
BUILD_DEB=${BUILD_DEB:-true}
BUILD_RPM=${BUILD_RPM:-true}
VERBOSE=${VERBOSE:-false}

# Create output directory
mkdir -p "${BUILD_OUTPUT}"

print_info() {
    echo "ℹ️  $1"
}

print_success() {
    echo "✅ $1"
}

print_error() {
    echo "❌ $1"
}

build_in_container() {
    local distro=$1
    local package_type=$2
    local dockerfile=$3
    
    print_info "Building $(echo $package_type | tr 'a-z' 'A-Z') package in $distro container..."
    
    # Build the container and create package natively inside it
    docker buildx build \
        --platform linux/amd64 \
        --file "${dockerfile}" \
        --tag "stormdb-builder-${distro}" \
        --build-arg BUILD_TYPE="${package_type}" \
        . || {
        print_error "Failed to build ${distro} container"
        return 1
    }
    
    # Run container to create package and copy it out
    docker run \
        --platform linux/amd64 \
        --rm \
        --volume "${BUILD_OUTPUT}:/output" \
        "stormdb-builder-${distro}" \
        /bin/bash -c "
            echo '=== Native ${distro} Build Starting ==='
            
            # Build project natively in Linux
            make clean
            make build-native-x86_64
            make plugins-native-x86_64
            
            # Create $(echo $package_type | tr 'a-z' 'A-Z') package natively
            make package-${package_type}
            
            # Copy packages to output
            cp -v build/packages/*.${package_type} /output/ 2>/dev/null || true
            
            echo '=== Native ${distro} Build Complete ==='
        " || {
        print_error "Failed to build ${package_type} package in ${distro}"
        return 1
    }
    
    print_success "$(echo $package_type | tr 'a-z' 'A-Z') package built successfully in ${distro}"
}

# Build DEB package in Ubuntu
if [ "$BUILD_DEB" = true ]; then
    print_info "=== Building DEB Package (Ubuntu) ==="
    if build_in_container "ubuntu" "deb" "docker/Dockerfile.ubuntu-builder"; then
        print_success "DEB package build completed"
    else
        print_error "DEB package build failed"
    fi
    echo ""
fi

# Build RPM package in CentOS
if [ "$BUILD_RPM" = true ]; then
    print_info "=== Building RPM Package (CentOS) ==="
    if build_in_container "centos" "rpm" "docker/Dockerfile.centos-builder"; then
        print_success "RPM package build completed"
    else
        print_error "RPM package build failed"
    fi
    echo ""
fi

# Summary
print_info "=== Build Summary ==="
if [ -d "${BUILD_OUTPUT}" ]; then
    echo "📦 Packages created:"
    ls -la "${BUILD_OUTPUT}"/*.{deb,rpm} 2>/dev/null | sed 's/^/   /' || echo "   No packages found"
else
    print_error "No packages were created"
fi

echo ""
print_info "🎯 All packages are built natively inside x86_64 Linux containers"
print_info "💻 No cross-compilation, no host dependencies, clean architecture"
