# Container Optimization and China Accessibility Guide

This document explains the optimized container setup for StormDB, including smaller Alpine-based images and China-friendly configurations for teams working behind the Great Firewall.

## Overview

We've enhanced the StormDB build system with:
1. **Alpine Linux base images** - Much smaller than Ubuntu (5-50MB vs 100-500MB)
2. **Podman support** - Rootless, more secure alternative to Docker
3. **China registry mirrors** - Automatic detection and configuration for China
4. **Multi-runtime compatibility** - Works with both Docker and Podman

## Image Size Comparison

| Image Type | Ubuntu Base | Alpine Base | Size Reduction |
|------------|-------------|-------------|----------------|
| Development | ~800MB | ~200MB | 75% smaller |
| Testing | ~900MB | ~250MB | 72% smaller |
| Production | ~150MB | ~15MB | 90% smaller |

## Quick Start

### 1. Detect Available Runtime
```bash
make container-detect
```

### 2. Setup for China (if needed)
```bash
make container-setup
```
This automatically detects if you're in China and configures registry mirrors.

### 3. Build Alpine-based Images
```bash
make alpine-build
```

### 4. Start Development Environment
```bash
make alpine-dev
```

## Container Options

### Alpine-based Containers (Recommended)
- **Smaller images**: 70-90% size reduction
- **Faster pulls**: Less data to download
- **More secure**: Minimal attack surface
- **Works in China**: Alternative registries configured

```bash
# Single container commands
make alpine-build    # Build development image
make alpine-dev      # Start development container
make alpine-test     # Run tests in container
make alpine-clean    # Clean up containers

# Docker Compose (Alpine)
make docker-compose-up     # Start all services
make docker-compose-dev    # Development environment
make docker-compose-test   # Run tests
make docker-compose-down   # Stop all services

# Podman Compose (rootless)
make podman-compose-up     # Start all services
make podman-compose-dev    # Development environment  
make podman-compose-test   # Run tests
make podman-compose-down   # Stop all services
```

### Legacy Ubuntu-based Containers
Still available for compatibility:
```bash
make docker-build
make docker-dev
make docker-test
make docker-clean
```

## China Accessibility Features

### Automatic Detection
The system automatically detects if you're in China by checking:
- Timezone (Asia/Shanghai, Asia/Beijing)
- Locale (zh_CN)
- Docker Hub connectivity

### Registry Mirrors
When China is detected, the system configures:

**For Docker:**
- DaoCloud: `docker.m.daocloud.io`
- Docker China: `registry.docker-cn.com`
- NetEase: `hub-mirror.c.163.com`
- Baidu: `mirror.baidubce.com`

**For Podman:**
- Alibaba Cloud: `registry.cn-hangzhou.aliyuncs.com`
- Tencent Cloud: `ccr.ccs.tencentyun.com`
- DaoCloud fallback

### Manual Setup for China
If automatic detection doesn't work:

```bash
# Export environment variable
export STORMDB_CHINA=true

# Then run setup
make container-setup
```

## Podman vs Docker

### Why Podman?
- **Rootless**: More secure, no root daemon
- **Compatible**: Drop-in replacement for Docker
- **Better for CI/CD**: No daemon dependency
- **China-friendly**: Better registry support

### Installing Podman

**macOS:**
```bash
brew install podman
podman machine init
podman machine start
```

**Linux (RHEL/CentOS/Fedora):**
```bash
sudo dnf install podman podman-compose
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt update
sudo apt install podman podman-compose
```

### Podman Configuration
Rootless Podman works out of the box with our configurations. The system automatically:
- Uses SELinux labels for volumes (`:Z`)
- Runs containers as non-root users
- Configures registry mirrors

## Container Files Explained

### Dockerfile.alpine
Multi-stage Alpine-based Dockerfile with stages:
- `base`: Minimal build tools
- `development`: Full dev environment with debuggers
- `testing`: CI/CD optimized
- `production`: Minimal runtime (15MB)
- `vscode`: VS Code remote development

### docker-compose.alpine.yml
Optimized Docker Compose with:
- Alpine-based services
- Persistent volumes
- Health checks
- Network isolation

### podman-compose.yml
Podman-specific configuration with:
- Rootless compatibility
- SELinux labels
- Security optimizations

### scripts/container-manager.sh
Universal script that:
- Detects Docker/Podman
- Configures China mirrors
- Manages container lifecycle
- Supports both runtimes

## VS Code Integration

### Remote Development
The containers include SSH servers for VS Code remote development:
- Port 2222: Development container
- Port 2223: VS Code container

Connect via VS Code Remote-Containers or Remote-SSH.

### DevContainer
Use the `.devcontainer/` configuration for seamless VS Code integration:
```bash
# Open in VS Code
code .
# Select "Reopen in Container" when prompted
```

## Troubleshooting

### Container Runtime Issues
```bash
# Check what's available
make container-detect

# Docker not working?
docker version
docker-compose version

# Try Podman instead
brew install podman
make container-setup
```

### China Connectivity Issues
```bash
# Test Docker Hub access
curl -I https://hub.docker.com

# Force China setup
export STORMDB_CHINA=true
make container-setup

# Use Podman with Chinese registries
make podman-compose-up
```

### Image Pull Issues
```bash
# Check configured registries
docker info | grep -A 10 "Registry Mirrors"

# Or for Podman
podman info | grep -A 10 registries

# Try alternative registry manually
docker pull registry.cn-hangzhou.aliyuncs.com/library/alpine:3.19
```

### Build Failures
```bash
# Clean everything and rebuild
make alpine-clean
make alpine-build

# Check build logs
docker logs stormdb-dev-alpine

# Use development shell
make alpine-dev
```

## Performance Tips

### Faster Builds
1. Use Alpine images (much smaller)
2. Layer caching works better with smaller images
3. Multi-stage builds reduce final image size

### Faster Development
1. Use volume mounts for source code
2. Keep development container running
3. Use VS Code remote development

### Network Optimization
1. Use local registry mirrors
2. Pre-pull base images
3. Use container registry cache

## Security Considerations

### Alpine Benefits
- Minimal attack surface
- Regular security updates
- No unnecessary packages

### Podman Benefits
- Rootless operation
- No root daemon
- Better process isolation

### Production Deployment
- Use minimal production images
- Run as non-root user
- Use read-only root filesystem
- Enable security scanning

## Integration Examples

### CI/CD Pipeline
```bash
# In your CI/CD script
make container-detect
make container-setup
make alpine-test
```

### Development Workflow
```bash
# Start development environment
make alpine-dev

# In container: build and test
make all
make test-all-sanitizers

# Outside container: clean up
make alpine-clean
```

### Cross-platform Testing
```bash
# Test on multiple platforms
make docker-compose-test  # x86_64
make podman-compose-test  # ARM64 (if on ARM Mac)
```

This optimized setup provides a modern, efficient, and globally accessible development environment for StormDB.
