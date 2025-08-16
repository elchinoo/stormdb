# StormDB Container Debugging Cheat Sheet

## Quick Start
```bash
# 1. Build development container
make alpine-build

# 2. Start development environment
make alpine-dev

# 3. Inside container: build and debug
make debug
gdb bin/stormdb-debug
```

## Container Build & Run Commands

### Development Container
```bash
make alpine-build     # Build development image (~200MB)
make alpine-dev       # Start interactive development container
make alpine-test      # Run tests in container
make alpine-clean     # Clean up containers
```

### Docker Compose (Full Environment)
```bash
make docker-compose-dev   # Start development + PostgreSQL
make docker-compose-test  # Run comprehensive tests
make docker-compose-down  # Stop all services
```

### Podman Compose (Rootless)
```bash
make podman-compose-dev   # Rootless development environment
make podman-compose-test  # Rootless testing
make podman-compose-down  # Stop all services
```

## Build Variants in Container

### Standard Builds
```bash
make all              # Default build
make debug            # Debug symbols + -O0
make release          # Optimized release build
```

### Sanitizer Builds (Advanced Debugging)
```bash
make asan             # AddressSanitizer (memory errors)
make tsan             # ThreadSanitizer (race conditions)
make ubsan            # UndefinedBehaviorSanitizer
make asan-ubsan       # Combined ASAN + UBSAN
make msan             # MemorySanitizer (Linux+Clang only)
```

### Testing Builds
```bash
make test-asan        # Test with AddressSanitizer
make test-tsan        # Test with ThreadSanitizer
make test-all-sanitizers  # Test all available sanitizers
make memcheck         # Valgrind memory check
```

## Debugging Tools Available

### GDB (GNU Debugger)
```bash
# Debug the debug build
gdb bin/stormdb-debug

# Common GDB commands
(gdb) run --help                # Run with arguments
(gdb) break main                # Set breakpoint
(gdb) continue                  # Continue execution
(gdb) backtrace                 # Show call stack
(gdb) print variable            # Print variable value
(gdb) info registers           # Show CPU registers
(gdb) disassemble function     # Show assembly
```

### AddressSanitizer (ASAN)
```bash
# Build with ASAN
make asan

# Run with leak detection
ASAN_OPTIONS=detect_leaks=1 bin/stormdb-asan

# Common ASAN options
export ASAN_OPTIONS=detect_leaks=1:abort_on_error=1:print_stats=1
```

### ThreadSanitizer (TSAN)
```bash
# Build with TSAN (detects race conditions)
make tsan

# Run with TSAN
bin/stormdb-tsan

# TSAN options
export TSAN_OPTIONS=halt_on_error=1:abort_on_error=1
```

### Valgrind (Memory Debugging)
```bash
# Use memcheck target (prefers ASAN on non-Linux)
make memcheck

# Manual Valgrind usage (Linux only)
valgrind --tool=memcheck --leak-check=full bin/stormdb-debug
```

## VS Code Remote Development

### Method 1: Remote Containers
1. Install "Remote - Containers" extension
2. Open project folder
3. Click "Reopen in Container"
4. VS Code connects automatically

### Method 2: Remote SSH
1. Start container: `make alpine-dev`
2. Container runs SSH on port 2222
3. Connect: `ssh root@localhost -p 2222`
4. Password: `stormdb`
5. Use "Remote - SSH" extension

### Debug Configuration
VS Code will automatically detect:
- GDB for native debugging
- Sanitizer output parsing
- Integrated terminal for container commands

## Container Environment Details

### File Locations
- **Source code**: `/workspace` (mounted from host)
- **Binaries**: `/workspace/bin/`
- **Build artifacts**: `/workspace/build/` and `/workspace/obj/`
- **Tools**: Standard Alpine package locations

### Environment Variables
```bash
CC=gcc                    # Default compiler
CXX=g++                   # C++ compiler
STORMDB_ENV=development   # Environment indicator
TERM=xterm-256color       # Terminal capabilities
```

### Ports (for remote development)
- **2222**: SSH for VS Code Remote
- **2223**: Alternative SSH port
- **5433**: PostgreSQL test database

## Common Debugging Scenarios

### Memory Issues
```bash
# Build with AddressSanitizer
make asan

# Run with comprehensive leak detection
ASAN_OPTIONS=detect_leaks=1:check_initialization_order=1 bin/stormdb-asan
```

### Performance Issues
```bash
# Build optimized release
make release

# Profile with built-in tools
time bin/stormdb-release --benchmark

# Use profiling tools (if available)
perf record bin/stormdb-release
```

### Threading Issues
```bash
# Build with ThreadSanitizer
make tsan

# Run multi-threaded workload
bin/stormdb-tsan --parallel-connections 10
```

### Cross-Platform Issues
```bash
# Test on multiple container platforms
make docker-compose-test    # x86_64 Linux
make podman-compose-test    # Current platform in rootless mode
```

## Troubleshooting

### Container Won't Start
```bash
# Check runtime availability
make container-detect

# Check logs
podman logs stormdb-dev-alpine
docker logs stormdb-dev-alpine
```

### Build Failures
```bash
# Clean and rebuild
make alpine-clean
make alpine-build

# Check container status
podman ps -a
docker ps -a
```

### Debug Symbols Missing
```bash
# Ensure debug build
make debug

# Check symbols
file bin/stormdb-debug
objdump -h bin/stormdb-debug | grep debug
```

### VS Code Connection Issues
```bash
# Verify SSH service
make alpine-dev
# Inside container:
service ssh status
ps aux | grep ssh
```

This cheat sheet covers all the essential debugging workflows with your optimized container setup!
