#!/bin/bash
# Cached Docker Build System for StormDB
# This uses pre-built base images to avoid downloading dependencies every time
set -e

echo "=== StormDB Cached Docker Package Builder ==="

# Configuration
BUILD_OUTPUT="$(pwd)/build/packages"
BUILD_DEB=${BUILD_DEB:-true}
BUILD_RPM=${BUILD_RPM:-true}

mkdir -p "${BUILD_OUTPUT}"

# Colors and formatting
print_success() {
    echo "✅ $1"
}

print_info() {
    echo "ℹ️  $1"
}

print_error() {
    echo "❌ $1"
}

print_warning() {
    echo "⚠️  $1"
}

# Function to build or use cached base images
ensure_base_image() {
    local distro=$1
    local dockerfile=$2
    local image_name="stormdb-${distro}-base"
    
    # Check if base image exists
    if docker images --format "table {{.Repository}}:{{.Tag}}" | grep -q "${image_name}:latest"; then
        print_info "Using cached base image: ${image_name}:latest"
    else
        print_info "Building base image: ${image_name}:latest (this may take a few minutes first time)"
        docker build \
            --platform linux/amd64 \
            --file "docker/${dockerfile}" \
            --tag "${image_name}:latest" \
            . || {
                print_error "Failed to build base image ${image_name}"
                return 1
            }
        print_success "Base image ${image_name}:latest created and cached"
    fi
}

# Function to build package using cached base
build_with_cached_base() {
    local distro=$1
    local package_type=$2
    local dockerfile=$3
    local base_dockerfile=$4
    
    print_info "Building $(echo $package_type | tr 'a-z' 'A-Z') package using cached ${distro} base..."
    
    # Ensure base image exists
    ensure_base_image "${distro}" "${base_dockerfile}" || return 1
    
    # Build the actual package (this should be fast since base is cached)
    docker build \
        --platform linux/amd64 \
        --file "${dockerfile}" \
        --tag "stormdb-builder-${distro}" \
        . || {
        print_error "Failed to build ${distro} package"
        return 1
    }
    
    # Extract package from container
    docker run \
        --platform linux/amd64 \
        --rm \
        --volume "${BUILD_OUTPUT}:/output" \
        "stormdb-builder-${distro}" \
        /bin/bash -c "
            echo '=== Extracting ${package_type} package ==='
            
            # Run the build
            /usr/local/bin/build-native.sh
            
            # Copy packages to output
            find /workspace/build/packages -name '*.${package_type}' -exec cp -v {} /output/ \\; 2>/dev/null || echo 'No ${package_type} packages found'
            
            echo '=== ${distro} package extraction complete ==='
        " || {
        print_error "Failed to extract ${package_type} package from ${distro}"
        return 1
    }
    
    print_success "$(echo $package_type | tr 'a-z' 'A-Z') package built using cached ${distro} base"
}

# Build DEB package using cached Ubuntu base
if [ "$BUILD_DEB" = true ]; then
    print_info "=== Building DEB Package (Ubuntu with cache) ==="
    if build_with_cached_base "ubuntu" "deb" "docker/Dockerfile.ubuntu-builder" "Dockerfile.ubuntu-base"; then
        print_success "DEB package build completed using cache"
    else
        print_error "DEB package build failed"
    fi
    echo ""
fi

# Build RPM package using cached CentOS base  
if [ "$BUILD_RPM" = true ]; then
    print_info "=== Building RPM Package (CentOS with cache) ==="
    if build_with_cached_base "centos" "rpm" "docker/Dockerfile.centos-builder" "Dockerfile.centos-base"; then
        print_success "RPM package build completed using cache"
    else
        print_error "RPM package build failed"
    fi
    echo ""
fi

# Summary
print_info "=== Build Summary ==="
if [ -d "${BUILD_OUTPUT}" ] && [ "$(ls -A ${BUILD_OUTPUT} 2>/dev/null)" ]; then
    echo "📦 Packages created:"
    ls -la "${BUILD_OUTPUT}"
else
    print_warning "No packages were created"
fi

echo ""
print_info "🚀 Cached Docker build system benefits:"
print_info "   ✅ Base images cached (Go, Ruby, system packages)"
print_info "   ✅ Only source code changes trigger rebuilds"
print_info "   ✅ Subsequent builds are much faster"
print_info "   ✅ Native x86_64 Linux compilation"

echo ""
print_info "💡 To rebuild base images (after system updates):"
echo "   docker rmi stormdb-ubuntu-base stormdb-centos-base"
echo "   ./build-docker-cached.sh"
