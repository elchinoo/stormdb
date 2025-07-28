# Build and Release Guide

This document describes how to build StormDB binaries, create packages, and prepare releases.

## Table of Contents

- [Quick Start](#quick-start)
- [Local Development](#local-development)
- [Release Process](#release-process)
- [Platform Support](#platform-support)
- [Package Formats](#package-formats)
- [Docker Images](#docker-images)
- [GitHub Actions](#github-actions)
- [Troubleshooting](#troubleshooting)

## Quick Start

### Prerequisites

1. **Go 1.21+**: Required for building the application
2. **Docker**: For building Docker images
3. **Ruby + FPM**: For creating Linux packages
   ```bash
   # Install FPM (Effing Package Management)
   gem install fpm
   ```
4. **Cross-compilation tools**: Automatically handled by Go

### Basic Build Commands

```bash
# Build for current platform
make build

# Build all plugins
make plugins

# Build release version with optimizations
make release-build

# Build cross-platform binaries
make release-cross

# Create Linux packages (DEB/RPM)
make release-packages

# Full release build (everything)
make release-full
```

## Local Development

### Development Workflow

```bash
# First time setup
make dev-tools deps build-all

# Development cycle
make dev-watch          # Watch and rebuild on changes
make pre-commit         # Run checks before committing

# Testing
make test               # Quick tests
make test-all           # Full test suite
make validate-full      # Complete validation
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `VERSION` | Version tag for builds | `git describe --tags` |
| `REGISTRY` | Docker registry | `localhost:5000` |
| `GODOC_PORT` | Documentation server port | `6060` |
| `STORMDB_TEST_HOST` | PostgreSQL host for tests | `localhost` |
| `STORMDB_TEST_DB` | PostgreSQL database for tests | `storm` |
| `STORMDB_TEST_USER` | PostgreSQL user for tests | - |
| `STORMDB_TEST_PASS` | PostgreSQL password for tests | - |

## Release Process

### 1. Pre-release Checks

```bash
# Run comprehensive validation
make release-check
```

This command:
- Validates the codebase
- Runs full test suite
- Performs security checks
- Verifies version information

### 2. Build Release Artifacts

#### Option A: Local Cross-platform Build

```bash
# Build binaries for all supported platforms
make release-cross
```

#### Option B: GitHub Actions (Recommended)

Push a git tag to trigger automated builds:

```bash
git tag v0.1.0-alpha.2
git push origin v0.1.0-alpha.2
```

### 3. Package Creation

```bash
# Create DEB package (Ubuntu/Debian)
make release-package-deb

# Create RPM package (CentOS/Fedora/Amazon Linux)
make release-package-rpm

# Create both packages
make release-packages
```

### 4. Docker Images

```bash
# Build Docker images
make release-docker

# Build and push to registry
make release-docker-push
```

### 5. Generate Checksums

```bash
# Generate SHA256 checksums for all artifacts
make release-checksums
```

### 6. Complete Release

```bash
# Full release process (all of the above)
make release-full
```

## Platform Support

### Supported Platforms

| OS | Architecture | Binary | Package | Docker |
|----|-------------|--------|---------|---------|
| Linux | amd64 | ✅ | DEB/RPM | ✅ |
| Linux | arm64 | ✅ | DEB/RPM | ✅ |
| Linux | 386 | ✅ | DEB/RPM | ❌ |
| macOS | amd64 | ✅ | ❌ | ❌ |
| macOS | arm64 | ✅ | ❌ | ❌ |
| Windows | amd64 | ✅ | ❌ | ❌ |
| Windows | 386 | ✅ | ❌ | ❌ |

### Build Matrix

The cross-platform build script generates binaries for:

- **Linux**: amd64, arm64, 386
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64, 386

## Package Formats

### DEB Packages (Ubuntu/Debian)

- **Target**: Ubuntu 20.04+, Debian 11+
- **Architecture**: amd64, arm64
- **Installation**: `sudo dpkg -i stormdb_*.deb`
- **Service**: Systemd service included
- **Config**: `/etc/stormdb/`
- **Binary**: `/usr/local/bin/stormdb`

### RPM Packages (Red Hat/CentOS/Fedora)

- **Target**: CentOS 8+, Fedora 35+, Amazon Linux 2
- **Architecture**: amd64, arm64
- **Installation**: `sudo rpm -i stormdb-*.rpm`
- **Service**: Systemd service included
- **Config**: `/etc/stormdb/`
- **Binary**: `/usr/local/bin/stormdb`

### Package Features

Both package formats include:

- Systemd service configuration
- Automatic user creation (`stormdb`)
- Configuration files in `/etc/stormdb/`
- Post-install and pre-remove scripts
- Documentation in `/usr/share/doc/stormdb/`

## Docker Images

### Multi-architecture Support

Docker images are built for:
- `linux/amd64`
- `linux/arm64`

### Image Variants

```bash
# Latest stable release
docker pull ghcr.io/elchinoo/stormdb:latest

# Specific version
docker pull ghcr.io/elchinoo/stormdb:v0.1.0-alpha.1

# Development version
docker pull ghcr.io/elchinoo/stormdb:main
```

### Usage

```bash
# Run with configuration
docker run -v $(pwd)/config:/config \
  ghcr.io/elchinoo/stormdb:latest \
  -config /config/my-workload.yaml

# Interactive mode
docker run -it --rm \
  ghcr.io/elchinoo/stormdb:latest \
  --help
```

## GitHub Actions

### Workflow Triggers

The release workflow is triggered by:

1. **Tag Push**: Pushing a tag matching `v*` (e.g., `v0.1.0-alpha.1`)
2. **Manual Dispatch**: Using the GitHub Actions UI

### Build Process

1. **Multi-platform Binaries**: Cross-compilation for all supported platforms
2. **Linux Packages**: DEB and RPM package creation using FPM
3. **Docker Images**: Multi-architecture Docker builds
4. **GitHub Release**: Automatic release creation with artifacts
5. **Registry Push**: Images pushed to GitHub Container Registry and Docker Hub

### Artifacts

Each release includes:

- Compressed binaries for all platforms
- DEB packages for Ubuntu/Debian
- RPM packages for CentOS/Fedora/Amazon Linux
- SHA256 checksums file
- Docker images in multiple registries

## Troubleshooting

### Common Issues

#### FPM Not Found

```bash
# Install FPM
gem install fpm

# Or on macOS with Homebrew
brew install fpm
```

#### Cross-compilation Failures

```bash
# Clean and rebuild
make clean release-cross

# Check Go version (requires 1.21+)
go version
```

#### Docker Build Issues

```bash
# Clean Docker cache
docker system prune -af

# Rebuild base images
docker pull golang:1.21-alpine
docker pull alpine:3.18
```

#### Package Installation Issues

```bash
# DEB package dependencies
sudo apt-get update
sudo apt-get install -f

# RPM package dependencies  
sudo yum install -y postgresql-client
# or
sudo dnf install -y postgresql
```

### Build Debugging

Enable verbose output:

```bash
# Verbose make output
make V=1 release-cross

# Go build with verbose flags
export GO_FLAGS="-v -x"
make release-build
```

### Log Locations

- **Local builds**: `build/` directory
- **GitHub Actions**: Workflow run logs
- **Package logs**: Check systemd journal after installation
  ```bash
  sudo journalctl -u stormdb -f
  ```

## Advanced Configuration

### Custom Registry

```bash
# Use custom Docker registry
export REGISTRY=my-registry.com/stormdb
make release-docker-push
```

### Custom Version

```bash
# Override version
export VERSION=v0.1.0-custom
make release-build
```

### Selective Builds

```bash
# Only build specific platform
GOOS=linux GOARCH=amd64 make release-build

# Only create DEB package
make release-package-deb
```

## Contributing

When adding new build targets or modifying the build process:

1. Update this documentation
2. Test on multiple platforms
3. Verify GitHub Actions workflow
4. Update version compatibility matrices
5. Add appropriate error handling

For questions or issues with the build system, please open an issue on GitHub.
