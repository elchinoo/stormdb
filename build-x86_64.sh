#!/bin/bash
# Quick x86_64 package builder for StormDB
set -e

echo "=== StormDB x86_64 Package Builder ==="

# Build x86_64 static binary
echo "🚀 Building x86_64 static binary..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty)" \
    -o build/stormdb-linux-x86_64 \
    cmd/stormdb/main.go

echo "✅ x86_64 binary created: $(file build/stormdb-linux-x86_64)"

# Build plugins (these need to be built in Docker for proper x86_64 .so files)
echo "🔌 Note: For production, plugins should be built in x86_64 Linux environment"
echo "    Current plugins are $(uname -m) architecture"

# Create DEB package with x86_64 binary
echo "📦 Creating x86_64 DEB package..."
mkdir -p build/packages
mkdir -p build/release

# Use the x86_64 binary for packaging
cp build/stormdb-linux-x86_64 build/release/stormdb
make release-package-deb

echo "✅ x86_64 packages created:"
ls -la build/packages/*.deb 2>/dev/null || echo "No DEB packages found"
ls -la build/packages/*.rpm 2>/dev/null || echo "No RPM packages found"

echo ""
echo "📋 Architecture verification:"
echo "Binary: $(file build/stormdb-linux-x86_64)"
echo "DEB Package: $(dpkg-deb --info build/packages/*.deb 2>/dev/null | grep Architecture || echo 'Check manually')"
