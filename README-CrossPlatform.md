# StormDB Cross-Platform Build System

StormDB is now fully cross-platform, supporting Windows, macOS, and Linux across different CPU architectures with comprehensive debugging and testing capabilities.

## Supported Platforms

### Operating Systems
- **Linux** - Full support with all sanitizers
- **macOS** - Full support except MemorySanitizer  
- **Windows** - Limited sanitizer support (ASAN/UBSAN via MinGW/MSYS2)

### Architectures
- **x86_64** (Intel/AMD 64-bit)
- **ARM64** (Apple Silicon, ARM 64-bit)
- **ARM32** (ARM 32-bit)
- **x86** (Intel/AMD 32-bit)

### Compilers
- **GCC** - Full support on Linux/Windows
- **Clang** - Full support on all platforms
- **MSVC** - Planned support for Windows

## Quick Start

### Platform Detection
```bash
make info  # Show current platform, architecture, and compiler
```

### Basic Build
```bash
make        # Build default optimized binary
make all    # Same as above
```

### Debug Builds
```bash
make debug  # Build with debug symbols and no optimization
```

### Release Build
```bash
make release  # Build optimized for production
```

## Sanitizer Support

StormDB includes comprehensive memory and thread safety checking through multiple sanitizers:

### AddressSanitizer (ASAN)
Detects memory errors like buffer overflows, use-after-free, and memory leaks.
```bash
make asan        # Build with AddressSanitizer
make test-asan   # Build and test ASAN binary
```

### ThreadSanitizer (TSAN)
Detects data races and thread safety issues.
```bash
make tsan        # Build with ThreadSanitizer (Unix only)
make test-tsan   # Build and test TSAN binary
```

### UndefinedBehaviorSanitizer (UBSAN)
Detects undefined behavior like integer overflow, null pointer dereference.
```bash
make ubsan       # Build with UndefinedBehaviorSanitizer
make test-ubsan  # Build and test UBSAN binary
```

### MemorySanitizer (MSAN)
Detects uninitialized memory reads (Linux + Clang only).
```bash
make msan        # Build with MemorySanitizer (Linux + Clang only)
make test-msan   # Build and test MSAN binary
```

### Combined Sanitizers
```bash
make asan-ubsan        # Build with both ASAN and UBSAN
make test-asan-ubsan   # Build and test combined sanitizers
```

### Test All Sanitizers
```bash
make test-all-sanitizers  # Build and test all available sanitizers
```

## Platform-Specific Features

### Windows (MinGW/MSYS2)
- **Supported**: ASAN, UBSAN
- **Not Supported**: TSAN, MSAN
- **Dependencies**: MinGW-w64, MSYS2, pkg-config
- **Libraries**: ws2_32, advapi32, psapi, shlwapi

### macOS
- **Supported**: ASAN, TSAN, UBSAN
- **Not Supported**: MSAN
- **Dependencies**: Homebrew, pkg-config
- **Libraries**: Dynamic library loading via dylib

### Linux
- **Supported**: All sanitizers (ASAN, TSAN, UBSAN, MSAN)
- **Dependencies**: pkg-config, build-essential
- **Libraries**: Full POSIX support

## Cross-Platform Abstraction Layer

StormDB includes a comprehensive platform abstraction layer (`src/platform/`) that provides:

### File Operations
- Cross-platform file I/O
- File locking mechanisms
- Directory operations
- Path handling

### Process Management
- Process creation and termination
- PID file management
- Process monitoring

### Signal Handling
- Unix signal handling
- Windows console control events
- Signal blocking/unblocking

### Thread Operations
- Thread creation and management
- Mutex operations
- Thread-safe operations

### Socket Operations
- Cross-platform networking
- Socket creation and management
- Address resolution

### Memory Management
- Platform-specific memory allocation
- Memory mapping
- Memory protection

## Build System Architecture

The Makefile automatically detects:
- **Operating System**: Windows, macOS, Linux, Unix
- **Architecture**: x86_64, ARM64, ARM32, x86
- **Compiler**: GCC, Clang, MSVC
- **Available Libraries**: pkg-config integration

### Code Quality Standards

StormDB enforces strict code quality through compiler warnings:
- **Warnings as Errors**: `-Werror` treats all warnings as compilation errors
- **Comprehensive Warnings**: `-Wall -Wextra -pedantic` for maximum warning coverage
- **Format Security**: `-Wformat=2` for printf-style function safety
- **Function Prototypes**: `-Wstrict-prototypes -Wmissing-prototypes` for function declarations

This ensures zero-warning code across all platforms and build types.

### Compiler Flags by Platform

#### Windows (MinGW)
- `-DPLATFORM_WINDOWS -DWIN32_LEAN_AND_MEAN`
- Static linking: `-static-libgcc -static-libstdc++`
- Libraries: `-lws2_32 -ladvapi32 -lpsapi -lshlwapi`

#### macOS
- `-DPLATFORM_MACOS -D_DARWIN_C_SOURCE`
- Libraries: `-ldl`
- Clang optimizations enabled

#### Linux
- `-DPLATFORM_LINUX -D_GNU_SOURCE`
- Libraries: `-ldl -lrt`
- Full sanitizer support

## Development Workflow

### Setting Up Development Environment

#### Windows (MSYS2)
```bash
# Install MSYS2 and dependencies
pacman -S mingw-w64-x86_64-gcc
pacman -S mingw-w64-x86_64-pkg-config
pacman -S mingw-w64-x86_64-yaml
pacman -S mingw-w64-x86_64-postgresql
```

#### macOS (Homebrew)
```bash
# Install dependencies
brew install libyaml
brew install postgresql
brew install pkg-config
```

#### Linux (Ubuntu/Debian)
```bash
# Install dependencies
sudo apt-get update
sudo apt-get install build-essential
sudo apt-get install libyaml-dev
sudo apt-get install libpq-dev
sudo apt-get install pkg-config
```

### Testing Workflow

1. **Platform Check**
   ```bash
   make info  # Verify platform detection
   ```

2. **Debug Build**
   ```bash
   make debug  # Build with full debugging
   ```

3. **Sanitizer Testing**
   ```bash
   make test-all-sanitizers  # Test memory/thread safety
   ```

4. **Release Build**
   ```bash
   make release  # Final optimized build
   ```

### Memory Debugging

#### Using Valgrind (Linux)
```bash
make memcheck  # Automatic Valgrind or ASAN testing
```

#### Manual Sanitizer Testing
```bash
# Address Sanitizer with detailed options
ASAN_OPTIONS="abort_on_error=1:detect_stack_use_after_return=1:check_initialization_order=1" ./bin/stormdb-asan

# Thread Sanitizer
TSAN_OPTIONS="halt_on_error=1:abort_on_error=1:report_bugs=1" ./bin/stormdb-tsan

# Undefined Behavior Sanitizer
UBSAN_OPTIONS="halt_on_error=1:abort_on_error=1:print_stacktrace=1" ./bin/stormdb-ubsan
```

## Available Make Targets

### Build Targets
- `all` - Build main binary (default)
- `debug` - Debug build with symbols
- `release` - Optimized release build
- `asan` - AddressSanitizer build
- `tsan` - ThreadSanitizer build
- `msan` - MemorySanitizer build
- `ubsan` - UndefinedBehaviorSanitizer build
- `asan-ubsan` - Combined ASAN+UBSAN build

### Testing Targets
- `test-asan` - Test AddressSanitizer build
- `test-tsan` - Test ThreadSanitizer build
- `test-msan` - Test MemorySanitizer build
- `test-ubsan` - Test UndefinedBehaviorSanitizer build
- `test-asan-ubsan` - Test combined sanitizers
- `test-all-sanitizers` - Test all available sanitizers
- `memcheck` - Memory checking (Valgrind or ASAN)

### Utility Targets
- `info` - Show platform and build information
- `clean` - Remove build artifacts
- `clean-all` - Remove all generated files
- `distclean` - Clean everything including logs
- `install` - Install StormDB (Unix only)
- `uninstall` - Uninstall StormDB (Unix only)
- `help` - Show all available targets

## Troubleshooting

### Common Issues

#### Missing Dependencies
```bash
# Check if pkg-config can find dependencies
pkg-config --cflags --libs yaml-0.1
pkg-config --cflags --libs libpq
```

#### Compiler Detection
```bash
# Verify compiler
gcc --version
clang --version
```

#### Platform Detection Issues
```bash
# Check platform detection variables
make info
uname -s  # Operating system
uname -m  # Architecture
```

### Windows-Specific Issues

#### MinGW Path Issues
```bash
# Ensure MinGW is in PATH
which gcc
export PATH="/mingw64/bin:$PATH"
```

#### Missing Windows Libraries
```bash
# Install missing libraries in MSYS2
pacman -S mingw-w64-x86_64-toolchain
```

### macOS-Specific Issues

#### Homebrew Dependencies
```bash
# Update Homebrew
brew update
brew upgrade

# Reinstall dependencies
brew reinstall libyaml postgresql
```

#### XCode Command Line Tools
```bash
xcode-select --install
```

### Linux-Specific Issues

#### Missing Development Headers
```bash
# Install development packages
sudo apt-get install libyaml-dev libpq-dev
sudo yum install libyaml-devel postgresql-devel  # Red Hat/CentOS
```

## Contributing

When contributing cross-platform code:

1. **Test on Multiple Platforms**: Test builds on Windows, macOS, and Linux
2. **Use Platform Abstraction**: Use functions from `platform.h` instead of direct system calls
3. **Check Sanitizers**: Ensure all sanitizer builds pass
4. **Update Documentation**: Update this README for new platform features

### Adding New Platform Support

1. Add platform detection in `Makefile`
2. Add platform-specific defines in `include/platform.h`
3. Implement platform-specific functions in `src/platform/platform.c`
4. Test with all available sanitizers
5. Update documentation

## Performance Notes

### Build Performance
- **Debug builds**: Slower execution, full debugging info
- **Release builds**: Optimized for speed, minimal debugging info
- **Sanitizer builds**: Significantly slower, extensive checking

### Runtime Performance by Platform
- **Linux**: Best performance, full optimization support
- **macOS**: Good performance, ARM64 optimizations
- **Windows**: Good performance, some limitations with sanitizers

## License

Cross-platform build system maintains the same license as StormDB.

## Support

For platform-specific issues:
- **Linux**: Use standard package managers and build tools
- **macOS**: Use Homebrew for dependencies
- **Windows**: Use MSYS2/MinGW-w64 environment

For sanitizer issues, refer to:
- [AddressSanitizer Documentation](https://clang.llvm.org/docs/AddressSanitizer.html)
- [ThreadSanitizer Documentation](https://clang.llvm.org/docs/ThreadSanitizer.html)
- [UndefinedBehaviorSanitizer Documentation](https://clang.llvm.org/docs/UndefinedBehaviorSanitizer.html)
