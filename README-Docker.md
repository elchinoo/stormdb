# StormDB Docker Integration Guide

This guide covers the complete Docker integration for StormDB, enabling cross-platform development, testing, and debugging with VS Code.

## 🐳 **Docker Environment Overview**

### **Available Containers**

| Container | Purpose | Target Platform | Features |
|-----------|---------|-----------------|----------|
| `stormdb-dev` | Development | Linux | Full toolchain, GDB, Valgrind |
| `stormdb-vscode` | VS Code Remote | Linux | SSH server, development tools |
| `stormdb-test` | CI/CD Testing | Linux | Automated testing, all sanitizers |
| `stormdb-cross` | Cross-compilation | Linux | Multi-arch compilers |
| `stormdb-prod` | Production | Linux | Minimal runtime environment |

### **Supported Architectures**
- **linux/amd64** (Intel/AMD 64-bit)
- **linux/arm64** (ARM 64-bit, Apple Silicon)
- **Cross-compilation** for ARM32, MIPS, etc.

## 🚀 **Quick Start**

### **Prerequisites**
```bash
# Install Docker and Docker Compose
# macOS
brew install docker docker-compose

# Ubuntu/Debian
sudo apt-get install docker.io docker-compose

# Verify installation
docker --version
docker compose version
```

### **Basic Usage**
```bash
# Start development environment
./scripts/docker-dev.sh run-dev

# Build all container variants
./scripts/docker-dev.sh build-all

# Run tests on Linux
./scripts/docker-dev.sh test-linux

# Cross-compile for multiple architectures
./scripts/cross-build.sh build release
```

## 🛠️ **Development Workflows**

### **1. Local Development with Docker**

#### **Interactive Development Session**
```bash
# Start development container
./scripts/docker-dev.sh run-dev

# Inside the container
make info          # Check platform
make debug         # Build debug version
make test-asan     # Test with AddressSanitizer
gdb bin/stormdb-debug  # Debug with GDB
```

#### **One-off Commands**
```bash
# Build without entering container
docker compose run --rm stormdb-dev make release

# Run specific tests
docker compose run --rm stormdb-dev make test-all-sanitizers

# Check memory leaks
docker compose run --rm stormdb-dev make memcheck
```

### **2. VS Code Remote Development**

#### **Setup VS Code Remote Container**
1. **Install Extensions**:
   - Remote - Containers
   - C/C++ Extension Pack
   - Docker

2. **Open in Container**:
   ```bash
   # Method 1: Use Command Palette
   # Ctrl+Shift+P → "Remote-Containers: Open Folder in Container"
   
   # Method 2: Use script
   ./scripts/docker-dev.sh run-vscode
   # Connect VS Code to ssh://root@localhost:2222 (password: stormdb)
   
   # Method 3: Use devcontainer.json (recommended)
   # Open folder in VS Code, it will auto-detect .devcontainer/
   ```

3. **Available VS Code Features**:
   - **IntelliSense** with full Linux headers
   - **Integrated debugging** with GDB
   - **Built-in tasks** for make targets
   - **Problem matcher** for compiler errors
   - **Sanitizer integration**

#### **VS Code Tasks Available**
- `Ctrl+Shift+P` → "Tasks: Run Task"
  - **Build: Debug** - Build debug version
  - **Build: Release** - Build optimized version
  - **Build: All Sanitizers** - Build all sanitizer variants
  - **Test: ASAN** - Run AddressSanitizer tests
  - **Valgrind: Memory Check** - Run Valgrind analysis
  - **Docker: Build Dev Image** - Rebuild development container

#### **Debugging in VS Code**
- `F5` or Debug → Start Debugging
  - **Debug StormDB** - Standard GDB debugging
  - **Debug StormDB with ASAN** - Debug with AddressSanitizer
  - **Debug with Valgrind** - Memory debugging
  - **Attach to Process** - Attach to running process

### **3. Cross-Platform Testing**

#### **Test on Multiple Linux Architectures**
```bash
# Build and test for all supported architectures
./scripts/cross-build.sh build release

# Test binaries using QEMU emulation
./scripts/cross-build.sh test

# Package for release
./scripts/cross-build.sh package v1.0.0
```

#### **CI/CD Integration**
```bash
# Run full test suite (suitable for CI)
docker compose up --build stormdb-test

# Build for production deployment
docker compose build stormdb-prod
```

## 🔧 **Advanced Configuration**

### **Custom Build Environments**

#### **Using Different Compilers**
```bash
# GCC development
docker compose run --rm \
  -e CC=gcc -e CXX=g++ \
  stormdb-dev make debug

# Clang development  
docker compose run --rm \
  -e CC=clang -e CXX=clang++ \
  stormdb-dev make debug

# Cross-compilation for ARM64
docker compose run --rm \
  -e CC=aarch64-linux-gnu-gcc \
  stormdb-cross make clean all
```

#### **Custom Build Flags**
```bash
# Debug with extra flags
docker compose run --rm \
  -e EXTRA_CFLAGS="-fsanitize=thread -fno-omit-frame-pointer" \
  stormdb-dev make debug

# Profile-guided optimization
docker compose run --rm \
  -e EXTRA_CFLAGS="-fprofile-generate" \
  stormdb-dev make release
```

### **Persistent Development Setup**

#### **Named Volumes for Fast Builds**
```yaml
# In docker-compose.yml (already configured)
volumes:
  stormdb-cache:     # Persistent build cache
  stormdb-bin:       # Persistent binaries
  stormdb-cross-cache: # Cross-compilation cache
```

#### **Custom Development Container**
```dockerfile
# Create custom development image
FROM stormdb-dev:latest

# Install additional tools
RUN apt-get update && apt-get install -y \
    your-favorite-editor \
    custom-debugging-tool

# Custom development setup
COPY your-config-files /home/dev/
```

## 🐛 **Debugging Guide**

### **GDB in Container**
```bash
# Start development container
./scripts/docker-dev.sh shell

# Debug with GDB
gdb bin/stormdb-debug
(gdb) run --version
(gdb) backtrace
(gdb) info registers
(gdb) quit
```

### **Valgrind Analysis**
```bash
# Memory leak detection
docker compose run --rm stormdb-dev \
  valgrind --leak-check=full --track-origins=yes \
  bin/stormdb-debug --version

# Comprehensive memory check
make memcheck  # Uses platform-appropriate tool
```

### **Sanitizer Debugging**
```bash
# AddressSanitizer with detailed output
ASAN_OPTIONS="abort_on_error=1:detect_stack_use_after_return=1" \
docker compose run --rm stormdb-dev bin/stormdb-asan --version

# ThreadSanitizer for race conditions
TSAN_OPTIONS="halt_on_error=1:report_bugs=1" \
docker compose run --rm stormdb-dev bin/stormdb-tsan

# UndefinedBehaviorSanitizer
UBSAN_OPTIONS="halt_on_error=1:print_stacktrace=1" \
docker compose run --rm stormdb-dev bin/stormdb-ubsan
```

### **Core Dump Analysis**
```bash
# Enable core dumps in container
docker compose run --rm \
  --cap-add=SYS_PTRACE \
  --security-opt seccomp:unconfined \
  stormdb-dev bash -c "
    ulimit -c unlimited
    echo core.%p > /proc/sys/kernel/core_pattern
    ./bin/stormdb-debug  # Run until crash
    gdb bin/stormdb-debug core.*
  "
```

## 🧪 **Testing Strategies**

### **Automated Testing Pipeline**
```bash
# Complete test pipeline
docker compose up --build stormdb-test

# Individual test stages
docker compose run --rm stormdb-dev make test-asan
docker compose run --rm stormdb-dev make test-tsan  
docker compose run --rm stormdb-dev make test-ubsan
docker compose run --rm stormdb-dev make memcheck
```

### **Performance Testing**
```bash
# Build optimized versions
docker compose run --rm stormdb-dev make release

# Performance comparison
docker compose run --rm stormdb-dev bash -c "
  time bin/stormdb-debug --version
  time bin/stormdb-release --version
  time bin/stormdb-asan --version
"
```

### **Stress Testing**
```bash
# Long-running stress tests
docker compose run --rm stormdb-dev bash -c "
  for i in {1..1000}; do
    bin/stormdb-asan --version >/dev/null || break
    echo 'Iteration $i completed'
  done
"
```

## 🚢 **Production Deployment**

### **Production Container**
```bash
# Build production image
docker compose build stormdb-prod

# Run production container
docker run -d \
  --name stormdb-prod \
  -v ./config:/config:ro \
  -e STORMDB_CONFIG=/config/stormdb.yaml \
  stormdb-prod:latest
```

### **Multi-Architecture Production Builds**
```bash
# Build for multiple architectures
./scripts/cross-build.sh build release

# Package for distribution
./scripts/cross-build.sh package v1.0.0

# Results in: stormdb-v1.0.0-multi-arch.tar.gz
```

## 🔍 **Troubleshooting**

### **Common Issues**

#### **Permission Problems**
```bash
# Fix file ownership after container use
sudo chown -R $USER:$USER obj/ bin/

# Or use user mapping in container
docker compose run --rm \
  --user $(id -u):$(id -g) \
  stormdb-dev make debug
```

#### **Container Build Failures**
```bash
# Clean Docker cache
./scripts/docker-dev.sh clean

# Rebuild without cache
docker compose build --no-cache stormdb-dev

# Check Docker resource limits
docker system df
docker system prune -f
```

#### **VS Code Connection Issues**
```bash
# Restart VS Code container
docker compose restart stormdb-vscode

# Check SSH connection
ssh -p 2222 root@localhost

# Reset VS Code remote connection
# VS Code → Remote Explorer → Reload Window
```

### **Debug Container Issues**
```bash
# Check container logs
./scripts/docker-dev.sh logs stormdb-dev

# Inspect container
docker compose exec stormdb-dev bash -c "
  echo 'Container info:'
  uname -a
  gcc --version
  make --version
  ls -la /workspace
"

# Check container status
./scripts/docker-dev.sh status
```

## 📚 **Best Practices**

### **Development Workflow**
1. **Use VS Code devcontainer** for consistent environment
2. **Leverage build cache** with named volumes
3. **Run sanitizers regularly** during development
4. **Test cross-platform** before committing
5. **Use production container** for final testing

### **Performance Optimization**
1. **Multi-stage builds** minimize image size
2. **Layer caching** speeds up rebuilds
3. **Named volumes** persist build artifacts
4. **Parallel builds** use all CPU cores

### **Security Considerations**
1. **Non-root user** in production containers
2. **Minimal runtime** dependencies
3. **Security scanning** of container images
4. **Regular base image updates**

## 🔗 **Integration Examples**

### **GitHub Actions CI/CD**
```yaml
# .github/workflows/docker-ci.yml
name: Docker CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Run Docker tests
        run: docker compose up --build stormdb-test
```

### **GitLab CI/CD**
```yaml
# .gitlab-ci.yml
test:linux:
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker compose up --build stormdb-test
```

This Docker integration provides a complete development, testing, and deployment environment for StormDB across multiple platforms and architectures.
