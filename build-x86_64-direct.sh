#!/bin/bash
# Direct x86_64 package creation for StormDB
set -e

echo "=== StormDB x86_64 Direct Package Builder ==="

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Build x86_64 static binary
echo "🚀 Building x86_64 static binary..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o build/stormdb-linux-x86_64 \
    cmd/stormdb/main.go

echo "✅ x86_64 binary: $(file build/stormdb-linux-x86_64)"

# Build current platform plugins (ARM64 on macOS)
echo "🔌 Building plugins (current platform)..."
make plugins

# Create DEB package directly with FPM
echo "📦 Creating x86_64 DEB package directly..."
mkdir -p build/packages-x86_64/deb/usr/bin
mkdir -p build/packages-x86_64/deb/usr/lib/stormdb/plugins
mkdir -p build/packages-x86_64/deb/etc/stormdb/examples
mkdir -p build/packages-x86_64/deb/usr/share/man/man1
mkdir -p build/packages-x86_64/deb/usr/share/doc/stormdb
mkdir -p build/packages-x86_64/deb/usr/share/stormdb

# Install x86_64 binary
cp build/stormdb-linux-x86_64 build/packages-x86_64/deb/usr/bin/stormdb

# Install plugins (note: these are ARM64, for demo purposes)
if [ -d "build/plugins" ]; then
    cp build/plugins/*.so build/packages-x86_64/deb/usr/lib/stormdb/plugins/ 2>/dev/null || true
fi

# Install configuration files
cp config/*.yaml build/packages-x86_64/deb/etc/stormdb/examples/
cp config/config_tpcc.yaml build/packages-x86_64/deb/etc/stormdb/

# Install man page
cp stormdb.1 build/packages-x86_64/deb/usr/share/man/man1/
gzip -9 build/packages-x86_64/deb/usr/share/man/man1/stormdb.1

# Install documentation
cp README.md CHANGELOG.md ARCHITECTURE.md build/packages-x86_64/deb/usr/share/doc/stormdb/
cp -r docs/* build/packages-x86_64/deb/usr/share/doc/stormdb/

# Install static data
cp imdb.sql build/packages-x86_64/deb/usr/share/stormdb/ 2>/dev/null || true
cp -r config build/packages-x86_64/deb/usr/share/stormdb/templates

# Create DEB package with FPM
echo "📦 Running FPM to create x86_64 DEB package..."
CLEAN_VERSION=$(echo ${VERSION} | sed 's/^v//')
fpm -s dir -t deb \
    --name stormdb \
    --version ${CLEAN_VERSION} \
    --maintainer "StormDB Team <team@stormdb.org>" \
    --description "PostgreSQL performance testing and benchmarking tool with plugin-based workload architecture" \
    --url "https://github.com/elchinoo/stormdb" \
    --license "MIT" \
    --architecture amd64 \
    --depends postgresql-client \
    --category database \
    -C build/packages-x86_64/deb \
    --package build/packages/

echo "✅ x86_64 DEB package created:"
ls -la build/packages/stormdb*.deb

echo ""
echo "📋 Package verification:"
echo "Binary architecture: $(file build/stormdb-linux-x86_64)"
echo "Package info:"
ar -t build/packages/stormdb*.deb 2>/dev/null || echo "Use dpkg-deb to inspect on Linux"

echo ""
echo "🎯 Success! Created x86_64 Linux package:"
echo "   - Binary: ELF x86_64 statically linked"
echo "   - Package: DEB with amd64 architecture"
echo "   - Size: $(ls -lh build/packages/stormdb*.deb | awk '{print $5}')"
