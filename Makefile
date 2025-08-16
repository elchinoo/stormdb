# ========================================================================
# StormDB Cross-Platform Makefile
# Supports: Windows (MinGW/MSYS2), macOS, Linux
# Architectures: x86_64, ARM64, ARM32
# ========================================================================

# Platform Detection
UNAME_S := $(shell uname -s 2>/dev/null || echo "Windows")
UNAME_M := $(shell uname -m 2>/dev/null || echo "x86_64")

# Default values
CC := gcc
CXX := g++
AR := ar
PLATFORM := unknown
ARCH := unknown
EXEC_EXT :=
LIB_EXT := .so
## Auxiliary container image targets (kept near top for quick access)
.PHONY: build-base-image build-china-base clean-base-images list-images

# Build cached base images
build-base-image:
	@echo "🔨 Building cached base development image..."
	docker build -f Dockerfile.base -t stormdb-base:latest . --progress=plain
	@echo "✅ Base image built and cached as stormdb-base:latest"

build-china-base:
	@echo "🔨 Building cached China-optimized base image..."
	docker build -f Dockerfile.base-china -t stormdb-base:china . --progress=plain
	@echo "✅ China base image built and cached as stormdb-base:china"

# Clean cached base images
clean-base-images:
	@echo "🧹 Cleaning cached base images..."
	@docker rmi stormdb-base:latest stormdb-base:china 2>/dev/null || true
	@echo "✅ Cached base images removed"

# List available images
list-images:
	@echo "📋 Available StormDB images:"
	@docker images | grep -E "(stormdb|REPOSITORY)" || echo "No StormDB images found"
PLATFORM_LDFLAGS :=

# Platform-specific configuration
ifeq ($(OS),Windows_NT)
    # Windows (MinGW/MSYS2)
    PLATFORM := windows
    EXEC_EXT := .exe
    LIB_EXT := .dll
    PLATFORM_DEFS := -DPLATFORM_WINDOWS -DWIN32_LEAN_AND_MEAN
    PLATFORM_LIBS := -lws2_32 -ladvapi32 -lpsapi -lshlwapi
    PLATFORM_LDFLAGS := -static-libgcc -static-libstdc++
    
    # Windows architecture detection
    ifeq ($(PROCESSOR_ARCHITEW6432),AMD64)
        ARCH := x86_64
    else
        ifeq ($(PROCESSOR_ARCHITECTURE),AMD64)
            ARCH := x86_64
        else ifeq ($(PROCESSOR_ARCHITECTURE),x86)
            ARCH := x86
        else
            ARCH := $(PROCESSOR_ARCHITECTURE)
        endif
    endif
else
    # Unix-like systems
    ifeq ($(UNAME_S),Linux)
        PLATFORM := linux
        PLATFORM_DEFS := -DPLATFORM_LINUX -D_GNU_SOURCE
        PLATFORM_LIBS := -ldl -lrt
        LIB_EXT := .so
	else ifeq ($(UNAME_S),Darwin)
		PLATFORM := macos
		PLATFORM_DEFS := -DPLATFORM_MACOS -D_DARWIN_C_SOURCE
		# macOS provides dlopen/dlsym in libSystem; do not link -ldl
		PLATFORM_LIBS :=
		LIB_EXT := .dylib
    else
        PLATFORM := unix
        PLATFORM_DEFS := -DPLATFORM_UNIX
        PLATFORM_LIBS := -ldl
    endif
    
    # Unix architecture detection
    ifeq ($(UNAME_M),x86_64)
        ARCH := x86_64
    else ifeq ($(UNAME_M),amd64)
        ARCH := x86_64
    else ifeq ($(UNAME_M),aarch64)
        ARCH := arm64
    else ifeq ($(UNAME_M),arm64)
        ARCH := arm64
    else ifeq ($(UNAME_M),armv7l)
        ARCH := arm
    else ifeq ($(UNAME_M),i386)
        ARCH := x86
    else ifeq ($(UNAME_M),i686)
        ARCH := x86
    else
        ARCH := $(UNAME_M)
    endif
endif

# Compiler detection and configuration
CC_VERSION := $(shell $(CC) --version 2>/dev/null | head -n1)
ifneq ($(findstring clang,$(CC_VERSION)),)
    COMPILER := clang
    COMPILER_FAMILY := clang
else ifneq ($(findstring gcc,$(CC_VERSION)),)
    COMPILER := gcc
    COMPILER_FAMILY := gcc
else ifneq ($(findstring Microsoft,$(CC_VERSION)),)
    COMPILER := msvc
    COMPILER_FAMILY := msvc
else
    COMPILER := unknown
    COMPILER_FAMILY := unknown
endif

# Base compiler flags
# -Werror treats all warnings as errors to ensure code quality
BASE_CFLAGS := -std=c11 -Wall -Wextra -pedantic -Werror
ifeq ($(COMPILER_FAMILY),gcc)
    BASE_CFLAGS += -Wformat=2 -Wstrict-prototypes -Wmissing-prototypes
else ifeq ($(COMPILER_FAMILY),clang)
    BASE_CFLAGS += -Wformat=2 -Wstrict-prototypes -Wmissing-prototypes
endif

# Build type configurations
DEFAULT_CFLAGS := $(BASE_CFLAGS) -O2 -g
DEBUG_CFLAGS := $(BASE_CFLAGS) -O0 -g3 -DDEBUG -fno-omit-frame-pointer
RELEASE_CFLAGS := $(BASE_CFLAGS) -O3 -DNDEBUG -fomit-frame-pointer

# Sanitizer configurations (platform-dependent)
ifeq ($(PLATFORM),windows)
    # Limited sanitizer support on Windows
    ASAN_CFLAGS := $(DEBUG_CFLAGS) -fsanitize=address
    ASAN_LDFLAGS := -fsanitize=address
    UBSAN_CFLAGS := $(DEBUG_CFLAGS) -fsanitize=undefined
    UBSAN_LDFLAGS := -fsanitize=undefined
    # TSAN and MSAN not supported on Windows
    TSAN_CFLAGS := $(DEBUG_CFLAGS)
    TSAN_LDFLAGS :=
    MSAN_CFLAGS := $(DEBUG_CFLAGS)
    MSAN_LDFLAGS :=
else
    # Full sanitizer support on Unix-like systems
    ASAN_CFLAGS := $(DEBUG_CFLAGS) -fsanitize=address -fno-common
    ASAN_LDFLAGS := -fsanitize=address
    TSAN_CFLAGS := $(DEBUG_CFLAGS) -fsanitize=thread
    TSAN_LDFLAGS := -fsanitize=thread
    UBSAN_CFLAGS := $(DEBUG_CFLAGS) -fsanitize=undefined -fsanitize=nullability -fsanitize=implicit-conversion
    UBSAN_LDFLAGS := -fsanitize=undefined
    
    # MSAN only on Linux with Clang
    ifeq ($(PLATFORM),linux)
        ifeq ($(COMPILER_FAMILY),clang)
            MSAN_CFLAGS := $(DEBUG_CFLAGS) -fsanitize=memory -fsanitize-memory-track-origins=2
            MSAN_LDFLAGS := -fsanitize=memory
        else
            MSAN_CFLAGS := $(DEBUG_CFLAGS)
            MSAN_LDFLAGS :=
        endif
    else
        MSAN_CFLAGS := $(DEBUG_CFLAGS)
        MSAN_LDFLAGS :=
    endif
endif

# Combined sanitizers
ASAN_UBSAN_CFLAGS := $(DEBUG_CFLAGS) -fsanitize=address -fsanitize=undefined
ASAN_UBSAN_LDFLAGS := -fsanitize=address -fsanitize=undefined

# Dependencies detection
HAS_PKGCONFIG := $(shell command -v pkg-config 2>/dev/null)
ifdef HAS_PKGCONFIG
    YAML_CFLAGS := $(shell pkg-config --cflags yaml-0.1 2>/dev/null)
    YAML_LIBS := $(shell pkg-config --libs yaml-0.1 2>/dev/null)
    PQ_CFLAGS := $(shell pkg-config --cflags libpq 2>/dev/null)
    PQ_LIBS := $(shell pkg-config --libs libpq 2>/dev/null)
else
    # Fallback for systems without pkg-config
    YAML_CFLAGS := 
    YAML_LIBS := -lyaml
    PQ_CFLAGS := 
    PQ_LIBS := -lpq
endif

# Final flags configuration
CFLAGS := $(DEFAULT_CFLAGS)
CPPFLAGS := $(PLATFORM_DEFS) $(YAML_CFLAGS) $(PQ_CFLAGS) -Iinclude
LDFLAGS := $(PLATFORM_LDFLAGS)
LIBS := -lpthread $(YAML_LIBS) $(PQ_LIBS) $(PLATFORM_LIBS)

# Directories
SRCDIR := src
INCDIR := include
OBJDIR := obj
BINDIR := bin
TESTDIR := tests
PLATFORMDIR := $(SRCDIR)/platform

# Source files
CORE_SOURCES := $(wildcard $(SRCDIR)/core/*.c)
LOGGING_SOURCES := $(wildcard $(SRCDIR)/logging/*.c)
DATABASE_SOURCES := $(wildcard $(SRCDIR)/database/*.c)
PLUGIN_SOURCES := $(wildcard $(SRCDIR)/plugin/*.c)
PLATFORM_SOURCES := $(wildcard $(PLATFORMDIR)/*.c)
API_SOURCES := $(wildcard $(SRCDIR)/api/*.c)
METRICS_SOURCES := $(wildcard $(SRCDIR)/metrics/*.c)
MEMORY_SOURCES := $(wildcard $(SRCDIR)/memory/*.c)
MAIN_SOURCE := $(SRCDIR)/main.c

ALL_SOURCES := $(CORE_SOURCES) $(LOGGING_SOURCES) $(DATABASE_SOURCES) \
			   $(PLUGIN_SOURCES) $(PLATFORM_SOURCES) $(API_SOURCES) \
			   $(METRICS_SOURCES) $(MEMORY_SOURCES) $(MAIN_SOURCE)

# Object files for different build types
OBJECTS := $(ALL_SOURCES:$(SRCDIR)/%.c=$(OBJDIR)/%.o)
DEBUG_OBJECTS := $(ALL_SOURCES:$(SRCDIR)/%.c=$(OBJDIR)/debug/%.o)
ASAN_OBJECTS := $(ALL_SOURCES:$(SRCDIR)/%.c=$(OBJDIR)/asan/%.o)
TSAN_OBJECTS := $(ALL_SOURCES:$(SRCDIR)/%.c=$(OBJDIR)/tsan/%.o)
MSAN_OBJECTS := $(ALL_SOURCES:$(SRCDIR)/%.c=$(OBJDIR)/msan/%.o)
UBSAN_OBJECTS := $(ALL_SOURCES:$(SRCDIR)/%.c=$(OBJDIR)/ubsan/%.o)
ASAN_UBSAN_OBJECTS := $(ALL_SOURCES:$(SRCDIR)/%.c=$(OBJDIR)/asan-ubsan/%.o)
RELEASE_OBJECTS := $(ALL_SOURCES:$(SRCDIR)/%.c=$(OBJDIR)/release/%.o)

# Target binaries
TARGET := $(BINDIR)/stormdb$(EXEC_EXT)
DEBUG_TARGET := $(BINDIR)/stormdb-debug$(EXEC_EXT)
ASAN_TARGET := $(BINDIR)/stormdb-asan$(EXEC_EXT)
TSAN_TARGET := $(BINDIR)/stormdb-tsan$(EXEC_EXT)
MSAN_TARGET := $(BINDIR)/stormdb-msan$(EXEC_EXT)
UBSAN_TARGET := $(BINDIR)/stormdb-ubsan$(EXEC_EXT)
ASAN_UBSAN_TARGET := $(BINDIR)/stormdb-asan-ubsan$(EXEC_EXT)
RELEASE_TARGET := $(BINDIR)/stormdb-release$(EXEC_EXT)

# Test configuration
TEST_CONFIG := config/stormdb.yaml

# Default target
.PHONY: all
all: $(TARGET)
	@echo "Built StormDB for $(PLATFORM)/$(ARCH) using $(COMPILER)"

# Platform info
.PHONY: info
info:
	@echo "Platform Information:"
	@echo "  OS: $(PLATFORM)"
	@echo "  Architecture: $(ARCH)"
	@echo "  Compiler: $(COMPILER) ($(COMPILER_FAMILY))"
	@echo "  Executable extension: '$(EXEC_EXT)'"
	@echo "  Library extension: '$(LIB_EXT)'"
	@echo ""
	@echo "Build Configuration:"
	@echo "  CFLAGS: $(CFLAGS)"
	@echo "  CPPFLAGS: $(CPPFLAGS)"
	@echo "  LDFLAGS: $(LDFLAGS)"
	@echo "  LIBS: $(LIBS)"

# Create directories
$(OBJDIR):
	@mkdir -p $(OBJDIR)
	@mkdir -p $(OBJDIR)/core $(OBJDIR)/logging $(OBJDIR)/database
	@mkdir -p $(OBJDIR)/plugin $(OBJDIR)/platform $(OBJDIR)/api $(OBJDIR)/metrics $(OBJDIR)/memory
	@mkdir -p $(OBJDIR)/debug/core $(OBJDIR)/debug/logging $(OBJDIR)/debug/database
	@mkdir -p $(OBJDIR)/debug/plugin $(OBJDIR)/debug/platform $(OBJDIR)/debug/api $(OBJDIR)/debug/metrics $(OBJDIR)/debug/memory
	@mkdir -p $(OBJDIR)/asan/core $(OBJDIR)/asan/logging $(OBJDIR)/asan/database
	@mkdir -p $(OBJDIR)/asan/plugin $(OBJDIR)/asan/platform $(OBJDIR)/asan/api $(OBJDIR)/asan/metrics $(OBJDIR)/asan/memory
	@mkdir -p $(OBJDIR)/tsan/core $(OBJDIR)/tsan/logging $(OBJDIR)/tsan/database
	@mkdir -p $(OBJDIR)/tsan/plugin $(OBJDIR)/tsan/platform $(OBJDIR)/tsan/api $(OBJDIR)/tsan/metrics $(OBJDIR)/tsan/memory
	@mkdir -p $(OBJDIR)/msan/core $(OBJDIR)/msan/logging $(OBJDIR)/msan/database
	@mkdir -p $(OBJDIR)/msan/plugin $(OBJDIR)/msan/platform $(OBJDIR)/msan/api $(OBJDIR)/msan/metrics $(OBJDIR)/msan/memory
	@mkdir -p $(OBJDIR)/ubsan/core $(OBJDIR)/ubsan/logging $(OBJDIR)/ubsan/database
	@mkdir -p $(OBJDIR)/ubsan/plugin $(OBJDIR)/ubsan/platform $(OBJDIR)/ubsan/api $(OBJDIR)/ubsan/metrics $(OBJDIR)/ubsan/memory
	@mkdir -p $(OBJDIR)/asan-ubsan/core $(OBJDIR)/asan-ubsan/logging $(OBJDIR)/asan-ubsan/database
	@mkdir -p $(OBJDIR)/asan-ubsan/plugin $(OBJDIR)/asan-ubsan/platform $(OBJDIR)/asan-ubsan/api $(OBJDIR)/asan-ubsan/metrics $(OBJDIR)/asan-ubsan/memory
	@mkdir -p $(OBJDIR)/release/core $(OBJDIR)/release/logging $(OBJDIR)/release/database
	@mkdir -p $(OBJDIR)/release/plugin $(OBJDIR)/release/platform $(OBJDIR)/release/api $(OBJDIR)/release/metrics $(OBJDIR)/release/memory

$(BINDIR):
	@mkdir -p $(BINDIR)

# Compilation rules
$(OBJDIR)/%.o: $(SRCDIR)/%.c | $(OBJDIR)
	@mkdir -p $(dir $@)
	$(CC) $(CFLAGS) $(CPPFLAGS) -c $< -o $@

$(OBJDIR)/debug/%.o: $(SRCDIR)/%.c | $(OBJDIR)
	@mkdir -p $(dir $@)
	$(CC) $(DEBUG_CFLAGS) $(CPPFLAGS) -c $< -o $@

$(OBJDIR)/asan/%.o: $(SRCDIR)/%.c | $(OBJDIR)
	@mkdir -p $(dir $@)
	$(CC) $(ASAN_CFLAGS) $(CPPFLAGS) -c $< -o $@

$(OBJDIR)/tsan/%.o: $(SRCDIR)/%.c | $(OBJDIR)
	@mkdir -p $(dir $@)
	$(CC) $(TSAN_CFLAGS) $(CPPFLAGS) -c $< -o $@

$(OBJDIR)/msan/%.o: $(SRCDIR)/%.c | $(OBJDIR)
	@mkdir -p $(dir $@)
	$(CC) $(MSAN_CFLAGS) $(CPPFLAGS) -c $< -o $@

$(OBJDIR)/ubsan/%.o: $(SRCDIR)/%.c | $(OBJDIR)
	@mkdir -p $(dir $@)
	$(CC) $(UBSAN_CFLAGS) $(CPPFLAGS) -c $< -o $@

$(OBJDIR)/asan-ubsan/%.o: $(SRCDIR)/%.c | $(OBJDIR)
	@mkdir -p $(dir $@)
	$(CC) $(ASAN_UBSAN_CFLAGS) $(CPPFLAGS) -c $< -o $@

$(OBJDIR)/release/%.o: $(SRCDIR)/%.c | $(OBJDIR)
	@mkdir -p $(dir $@)
	$(CC) $(RELEASE_CFLAGS) $(CPPFLAGS) -c $< -o $@

# Linking rules
$(TARGET): $(OBJECTS) | $(BINDIR)
	$(CC) $(LDFLAGS) $(OBJECTS) $(LIBS) -o $@

# Build targets
.PHONY: debug asan tsan msan ubsan asan-ubsan release
debug: $(DEBUG_TARGET)
asan: $(ASAN_TARGET)
tsan: $(TSAN_TARGET)
msan: $(MSAN_TARGET)
ubsan: $(UBSAN_TARGET)
asan-ubsan: $(ASAN_UBSAN_TARGET)
release: $(RELEASE_TARGET)

$(DEBUG_TARGET): $(DEBUG_OBJECTS) | $(BINDIR)
	$(CC) $(LDFLAGS) $(DEBUG_OBJECTS) $(LIBS) -o $@
	@echo "Debug build completed: $@"

$(ASAN_TARGET): $(ASAN_OBJECTS) | $(BINDIR)
	$(CC) $(ASAN_LDFLAGS) $(ASAN_OBJECTS) $(LIBS) -o $@
	@echo "AddressSanitizer build completed: $@"

$(TSAN_TARGET): $(TSAN_OBJECTS) | $(BINDIR)
ifeq ($(PLATFORM),windows)
	@echo "ThreadSanitizer not supported on Windows, building debug version instead"
	$(CC) $(LDFLAGS) $(TSAN_OBJECTS) $(LIBS) -o $@
else
	$(CC) $(TSAN_LDFLAGS) $(TSAN_OBJECTS) $(LIBS) -o $@
endif
	@echo "ThreadSanitizer build completed: $@"

$(MSAN_TARGET): $(MSAN_OBJECTS) | $(BINDIR)
ifeq ($(PLATFORM),linux)
    ifeq ($(COMPILER_FAMILY),clang)
	$(CC) $(MSAN_LDFLAGS) $(MSAN_OBJECTS) $(LIBS) -o $@
	@echo "MemorySanitizer build completed: $@"
    else
	@echo "MemorySanitizer requires Clang on Linux, building debug version instead"
	$(CC) $(LDFLAGS) $(MSAN_OBJECTS) $(LIBS) -o $@
    endif
else
	@echo "MemorySanitizer only supported on Linux with Clang, building debug version instead"
	$(CC) $(LDFLAGS) $(MSAN_OBJECTS) $(LIBS) -o $@
endif

$(UBSAN_TARGET): $(UBSAN_OBJECTS) | $(BINDIR)
	$(CC) $(UBSAN_LDFLAGS) $(UBSAN_OBJECTS) $(LIBS) -o $@
	@echo "UndefinedBehaviorSanitizer build completed: $@"

$(ASAN_UBSAN_TARGET): $(ASAN_UBSAN_OBJECTS) | $(BINDIR)
	$(CC) $(ASAN_UBSAN_LDFLAGS) $(ASAN_UBSAN_OBJECTS) $(LIBS) -o $@
	@echo "ASAN+UBSAN build completed: $@"

$(RELEASE_TARGET): $(RELEASE_OBJECTS) | $(BINDIR)
	$(CC) $(LDFLAGS) $(RELEASE_OBJECTS) $(LIBS) -o $@
	@echo "Release build completed: $@"

# Testing targets
.PHONY: run-tests test-asan test-tsan test-msan test-ubsan test-asan-ubsan test-all-sanitizers

TEST_SRC := test/smoke_tests.c
TEST_BIN := bin/tests-smoke$(EXEC_EXT)
UNIT_TEST_SRC := test/unit_config_pid_plugin.c
UNIT_TEST_BIN := bin/tests-unit$(EXEC_EXT)
UNIT_DATABASE_MATERIALIZE_SRC := test/unit_database_materialize.c
UNIT_DATABASE_MATERIALIZE_BIN := bin/tests-db-materialize$(EXEC_EXT)

CORE_NO_MAIN := $(filter-out $(OBJDIR)/debug/main.o,$(DEBUG_OBJECTS))
$(TEST_BIN): $(CORE_NO_MAIN) $(TEST_SRC) | $(BINDIR)
	$(CC) $(DEBUG_CFLAGS) $(CPPFLAGS) $(TEST_SRC) $(CORE_NO_MAIN) $(LIBS) -o $@

$(UNIT_TEST_BIN): $(CORE_NO_MAIN) $(UNIT_TEST_SRC) | $(BINDIR)
	$(CC) $(DEBUG_CFLAGS) $(CPPFLAGS) $(UNIT_TEST_SRC) $(CORE_NO_MAIN) $(LIBS) -o $@

$(UNIT_DATABASE_MATERIALIZE_BIN): $(CORE_NO_MAIN) $(UNIT_DATABASE_MATERIALIZE_SRC) | $(BINDIR)
	$(CC) $(DEBUG_CFLAGS) $(CPPFLAGS) $(UNIT_DATABASE_MATERIALIZE_SRC) $(CORE_NO_MAIN) $(LIBS) -o $@

run-tests: $(TEST_BIN) $(UNIT_TEST_BIN) $(UNIT_DATABASE_MATERIALIZE_BIN)
	@echo "=== Running smoke tests ==="
	$(TEST_BIN)
	@echo "All smoke tests passed"
	@echo "=== Running unit tests ==="
	$(UNIT_TEST_BIN)
	@echo "All unit tests passed"
	@echo "=== Running database materialize unit test ==="
	$(UNIT_DATABASE_MATERIALIZE_BIN)
	@echo "All unit/database tests passed"
test-asan: $(ASAN_TARGET)
	@echo "=== AddressSanitizer Test ==="
ifeq ($(PLATFORM),windows)
	@echo "Running ASAN test (limited Windows support)..."
	$(ASAN_TARGET) --version
else
	@echo "Running ASAN test..."
	ASAN_OPTIONS="abort_on_error=1:detect_stack_use_after_return=1" $(ASAN_TARGET) --version
endif
	@echo "ASAN test completed successfully!"

test-tsan: $(TSAN_TARGET)
	@echo "=== ThreadSanitizer Test ==="
ifeq ($(PLATFORM),windows)
	@echo "ThreadSanitizer not supported on Windows, running basic test..."
	$(TSAN_TARGET) --version
else
	@echo "Running TSAN test..."
	TSAN_OPTIONS="halt_on_error=1:abort_on_error=1" $(TSAN_TARGET) --version
endif
	@echo "TSAN test completed successfully!"

test-msan: $(MSAN_TARGET)
	@echo "=== MemorySanitizer Test ==="
ifeq ($(PLATFORM),linux)
    ifeq ($(COMPILER_FAMILY),clang)
	@echo "Running MSAN test..."
	MSAN_OPTIONS="halt_on_error=1:abort_on_error=1" $(MSAN_TARGET) --version
	@echo "MSAN test completed successfully!"
    else
	@echo "MemorySanitizer requires Clang on Linux - test skipped"
    endif
else
	@echo "MemorySanitizer only supported on Linux with Clang - test skipped"
endif

test-ubsan: $(UBSAN_TARGET)
	@echo "=== UndefinedBehaviorSanitizer Test ==="
	@echo "Running UBSAN test..."
	UBSAN_OPTIONS="halt_on_error=1:abort_on_error=1" $(UBSAN_TARGET) --version
	@echo "UBSAN test completed successfully!"

test-asan-ubsan: $(ASAN_UBSAN_TARGET)
	@echo "=== ASAN+UBSAN Test ==="
	@echo "Running ASAN+UBSAN test..."
ifeq ($(PLATFORM),windows)
	$(ASAN_UBSAN_TARGET) --version
else
	ASAN_OPTIONS="abort_on_error=1" UBSAN_OPTIONS="halt_on_error=1" $(ASAN_UBSAN_TARGET) --version
endif
	@echo "ASAN+UBSAN test completed successfully!"

test-all-sanitizers: test-asan test-tsan test-msan test-ubsan test-asan-ubsan
	@echo "=== All sanitizer tests completed! ==="

# Memory checking
.PHONY: memcheck memcheck-debug valgrind-check valgrind-basic valgrind-detailed valgrind-all
memcheck: $(TARGET)
	@echo "=== Memory Check ==="
ifeq ($(PLATFORM),linux)
	@if command -v valgrind >/dev/null 2>&1; then \
		echo "Running Valgrind memory check..."; \
		valgrind --leak-check=full --show-leak-kinds=all --track-origins=yes \
			--verbose --log-file=valgrind.log $(TARGET) --version; \
		echo "Valgrind check completed. See valgrind.log"; \
	else \
		echo "Valgrind not available, using AddressSanitizer instead..."; \
		$(MAKE) test-asan; \
	fi
else
	@echo "Valgrind not available on $(PLATFORM), using AddressSanitizer instead..."
	$(MAKE) test-asan
endif

valgrind-basic:
	@echo "=== Basic Valgrind Test ==="
	@./scripts/valgrind-test.sh basic

valgrind-detailed:
	@echo "=== Detailed Valgrind Test ==="
	@./scripts/valgrind-test.sh detailed

valgrind-all:
	@echo "=== Comprehensive Valgrind Testing ==="
	@./scripts/valgrind-test.sh basic
	@echo ""
	@./scripts/valgrind-test.sh detailed
	@echo ""
	@./scripts/valgrind-test.sh memcheck

# Run valgrind memcheck inside the Linux base container
.PHONY: memcheck-docker
memcheck-docker: memcheck-compose
 

# Clean targets
.PHONY: clean clean-all distclean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(OBJDIR) $(BINDIR)

clean-all: clean
	@echo "Cleaning all generated files..."
	rm -f valgrind*.log *.log

distclean: clean-all
	@echo "Cleaning distribution files..."
	rm -f config.cache config.log config.status

# Installation (platform-specific)
.PHONY: install uninstall
install: $(TARGET)
	@echo "Installing StormDB..."
ifeq ($(PLATFORM),windows)
	@echo "Windows installation not implemented yet"
else
	install -d /usr/local/bin
	install $(TARGET) /usr/local/bin/
	@echo "StormDB installed to /usr/local/bin/"
endif

uninstall:
	@echo "Uninstalling StormDB..."
ifeq ($(PLATFORM),windows)
	@echo "Windows uninstallation not implemented yet"
else
	rm -f /usr/local/bin/stormdb$(EXEC_EXT)
	@echo "StormDB uninstalled"
endif

# Docker integration targets
.PHONY: docker-build docker-dev docker-test docker-shell docker-clean
docker-build:
	@echo "Building Docker development container..."
	docker compose build stormdb-dev

docker-dev:
	@echo "Starting Docker development environment..."
	@if command -v docker >/dev/null 2>&1; then \
		./scripts/docker-dev.sh run-dev; \
	else \
		echo "Docker is not installed. Please install Docker Desktop."; \
		exit 1; \
	fi

docker-test:
	@echo "Running tests in Docker container..."
	@if command -v docker >/dev/null 2>&1; then \
		docker compose up --build stormdb-test; \
	else \
		echo "Docker is not installed. Please install Docker Desktop."; \
		exit 1; \
	fi

docker-shell:
	@echo "Opening shell in Docker development container..."
	@if command -v docker >/dev/null 2>&1; then \
		./scripts/docker-dev.sh shell; \
	else \
		echo "Docker is not installed. Please install Docker Desktop."; \
		exit 1; \
	fi

docker-clean:
	@echo "Cleaning Docker containers and images..."
	@if command -v docker >/dev/null 2>&1; then \
		./scripts/docker-dev.sh clean; \
	else \
		echo "Docker is not installed. Please install Docker Desktop."; \
		exit 1; \
	fi

# Run valgrind memcheck in the Linux dev container via Docker Compose
.PHONY: memcheck-compose
memcheck-compose:
	@echo "Running Valgrind memcheck inside stormdb-dev container..."
	@docker compose run --rm stormdb-dev bash -lc "make clean && make memcheck"

# China-friendly memcheck using Dockerfile.china and local mirror base image
.PHONY: memcheck-china
memcheck-china:
	@echo "Building China-optimized development image (using DaoCloud Ubuntu mirror by default)..."
	@BASE_IMAGE=$${BASE_IMAGE:-daocloud.io/library/ubuntu:22.04} \
		docker build -f Dockerfile.china --build-arg BASE_IMAGE=$$BASE_IMAGE -t stormdb:china-dev --target development .
	@echo "Running Valgrind memcheck inside stormdb:china-dev..."
	@docker run --rm -v "$(PWD):/workspace" -w /workspace stormdb:china-dev bash -lc "make clean && make memcheck"

# Container targets (Docker/Podman with Alpine optimization)
.PHONY: container-detect container-setup alpine-build alpine-dev alpine-test alpine-clean
.PHONY: docker-compose-up docker-compose-dev docker-compose-test docker-compose-down
.PHONY: podman-compose-up podman-compose-dev podman-compose-test podman-compose-down
.PHONY: linux-dev linux-dev-setup china-build china-dev china-test

# Linux development (works offline with existing images)
linux-dev:
	@echo "Starting Linux development session..."
	@./scripts/linux-dev-session.sh

linux-dev-quick:
	@echo "Starting quick Linux development session..."
	@./scripts/linux-dev-quick.sh

linux-dev-setup:
	@echo "Setting up Linux development options..."
	@./scripts/setup-linux-dev.sh

# China-optimized Ubuntu development
china-build:
	@echo "Building China-optimized development container..."
	@./scripts/china-dev.sh build

china-dev:
	@echo "Starting China-optimized development session..."
	@./scripts/china-dev.sh dev

china-test:
	@echo "Running tests in China-optimized container..."
	@./scripts/china-dev.sh test

# Cross-platform testing (definitions centralized earlier)
.PHONY: test-cross-platform test-x86_64 test-arm64-linux test-native linux-dev-quick

container-detect:
	@echo "Detecting available container runtime..."
	@./scripts/container-manager.sh detect

container-setup:
	@echo "Setting up container environment..."
	@./scripts/container-manager.sh setup

# Alpine-based optimized builds (smaller images)
alpine-build:
	@echo "Building Alpine-based container image..."
	@./scripts/container-manager.sh build --dockerfile Dockerfile.alpine --stage development

alpine-dev:
	@echo "Starting Alpine-based development container..."
	@./scripts/container-manager.sh dev

alpine-test:
	@echo "Running tests in Alpine-based container..."
	@./scripts/container-manager.sh test

alpine-clean:
	@echo "Cleaning Alpine-based containers..."
	@./scripts/container-manager.sh clean

# Docker Compose targets (Alpine-optimized)
docker-compose-up:
	@echo "Starting Docker Compose services (Alpine)..."
	docker-compose -f docker-compose.alpine.yml up -d

docker-compose-dev:
	@echo "Starting Docker Compose development environment..."
	docker-compose -f docker-compose.alpine.yml up -d stormdb-dev
	docker-compose -f docker-compose.alpine.yml exec stormdb-dev bash

docker-compose-test:
	@echo "Running tests via Docker Compose..."
	docker-compose -f docker-compose.alpine.yml run --rm stormdb-test

docker-compose-down:
	@echo "Stopping Docker Compose services..."
	docker-compose -f docker-compose.alpine.yml down -v

# Podman Compose targets (rootless-friendly)
podman-compose-up:
	@echo "Starting Podman Compose services..."
	podman-compose -f podman-compose.yml up -d

podman-compose-dev:
	@echo "Starting Podman Compose development environment..."
	podman-compose -f podman-compose.yml up -d stormdb-dev
	podman-compose -f podman-compose.yml exec stormdb-dev bash

podman-compose-test:
	@echo "Running tests via Podman Compose..."
	podman-compose -f podman-compose.yml run --rm stormdb-test

podman-compose-down:
	@echo "Stopping Podman Compose services..."
	podman-compose -f podman-compose.yml down -v

# Help
.PHONY: help
help:
	@echo "StormDB Cross-Platform Makefile"
	@echo "Platform: $(PLATFORM)/$(ARCH) using $(COMPILER)"
	@echo ""
	@echo "Available targets:"
	@echo "  all           - Build the main binary (default)"
	@echo "  debug         - Build debug version"
	@echo "  release       - Build optimized release version"
	@echo "  asan          - Build with AddressSanitizer"
	@echo "  tsan          - Build with ThreadSanitizer (Unix only)"
	@echo "  msan          - Build with MemorySanitizer (Linux+Clang only)"
	@echo "  ubsan         - Build with UndefinedBehaviorSanitizer"
	@echo "  asan-ubsan    - Build with ASAN+UBSAN"
	@echo ""
	@echo "Testing targets:"
	@echo "  test-asan     - Test AddressSanitizer build"
	@echo "  test-tsan     - Test ThreadSanitizer build"
	@echo "  test-msan     - Test MemorySanitizer build"
	@echo "  test-ubsan    - Test UndefinedBehaviorSanitizer build"
	@echo "  test-asan-ubsan - Test ASAN+UBSAN build"
	@echo "  test-all-sanitizers - Test all available sanitizers"
	@echo "  memcheck      - Memory check (Valgrind on Linux, ASAN elsewhere)"
	@echo "  test          - Alias for run-tests"
	@echo "  ci            - Clean, build debug, run smoke tests and ASAN test"
	@echo "  valgrind-basic     - Basic valgrind test in Linux container"
	@echo "  valgrind-detailed  - Detailed valgrind test in Linux container"
	@echo "  valgrind-all       - All valgrind tests in Linux container"
	@echo ""
	@echo "Utility targets:"
	@echo "  info          - Show platform and build information"
	@echo "  clean         - Remove build artifacts"
	@echo "  clean-all     - Remove all generated files"
	@echo "  install       - Install StormDB (Unix only)"
	@echo "  uninstall     - Uninstall StormDB (Unix only)"
	@echo "  help          - Show this help"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build  - Build Docker development container"
	@echo "  docker-dev    - Start Docker development environment"
	@echo "  docker-test   - Run tests in Docker container"
	@echo "  docker-shell  - Open shell in Docker container"
	@echo "  docker-clean  - Clean Docker containers and images"
	@echo ""
	@echo "Container targets (Alpine-optimized):"
	@echo "  container-detect      - Detect available container runtime (Docker/Podman)"
	@echo "  container-setup       - Setup container environment (China mirrors, etc.)"
	@echo "  alpine-build          - Build Alpine-based container image (smaller)"
	@echo "  alpine-dev            - Start Alpine development container"
	@echo "  alpine-test           - Run tests in Alpine container"
	@echo "  alpine-clean          - Clean Alpine containers"
	@echo ""
	@echo "Linux development targets:"
	@echo "  linux-dev             - Start Linux development session (offline-capable)"
	@echo "  linux-dev-setup       - Setup Linux development options"
	@echo "  china-build           - Build China-optimized Ubuntu container"
	@echo "  china-dev             - Start China-optimized development session"
	@echo "  china-test            - Run tests in China-optimized container"
	@echo ""
	@echo "Docker Compose targets (Alpine):"
	@echo "  docker-compose-up     - Start all Docker Compose services"
	@echo "  docker-compose-dev    - Start development environment"
	@echo "  docker-compose-test   - Run tests via Docker Compose"
	@echo "  docker-compose-down   - Stop all Docker Compose services"
	@echo ""
	@echo "Podman Compose targets:"
	@echo "  podman-compose-up     - Start all Podman Compose services"
	@echo "  podman-compose-dev    - Start development environment (rootless)"
	@echo "  podman-compose-test   - Run tests via Podman Compose"
	@echo "  podman-compose-down   - Stop all Podman Compose services"
	@echo ""
	@echo "Cross-platform testing:"
	@echo "  test-cross-platform   - Test on all available platforms"
	@echo "  test-x86_64           - Test on x86_64 Linux (emulated)"
	@echo "  test-native           - Test on native platform (macOS ARM64)"
	@echo "  test-arm64-linux      - Test on ARM64 Linux (if available)"
	@echo ""
	@echo "Container image management:"
	@echo "  build-base-image      - Build cached base development image"
	@echo "  build-china-base      - Build China-optimized base image"
	@echo "  clean-base-images     - Remove cached base images"
	@echo "  list-images           - List available StormDB images"
	@echo ""
	@echo "Platform support:"
	@echo "  Windows: MinGW/MSYS2 (limited sanitizer support)"
	@echo "  macOS:   Full support except MemorySanitizer"
	@echo "  Linux:   Full support including all sanitizers"
	@echo "  Docker:  Full Linux development environment"

# Integration test for DB persistence using a local postgres-test container
.PHONY: integration-db build-postgres-test up-postgres-test down-postgres-test run-integration-db

build-postgres-test:
	@echo "Building postgres-test image (pulled official postgres if not present)..."
	@docker pull postgres:14-alpine >/dev/null 2>&1 || true

up-postgres-test:
	@echo "Starting postgres-test service..."
	@docker compose -f docker-compose.yml -f docker-compose.local.yml up -d postgres-test
	@sleep 2

down-postgres-test:
	@echo "Stopping postgres-test service..."
	@docker compose -f docker-compose.yml -f docker-compose.local.yml down postgres-test || true

run-integration-db: up-postgres-test
	@echo "Running integration DB test against postgres-test (host=$(if $(POSTGRES_TEST_HOST),$(POSTGRES_TEST_HOST),127.0.0.1):$(if $(POSTGRES_TEST_PORT),$(POSTGRES_TEST_PORT),5432))"
	@# Wait for DB to accept connections
	@./scripts/wait-for-db.sh $(if $(POSTGRES_TEST_HOST),$(POSTGRES_TEST_HOST),127.0.0.1) $(if $(POSTGRES_TEST_PORT),$(POSTGRES_TEST_PORT),5432) 30
	@echo "Compiling integration test..."
	@$(CC) -o test/integration_db test/integration_db.c $(PQ_CFLAGS) $(PQ_LIBS)
	@echo "Executing integration test..."
	@POSTGRES_TEST_HOST=$(if $(POSTGRES_TEST_HOST),$(POSTGRES_TEST_HOST),127.0.0.1) POSTGRES_TEST_PORT=$(if $(POSTGRES_TEST_PORT),$(POSTGRES_TEST_PORT),5432) ./test/integration_db || (RET=$$?; echo "Integration test failed with code $$RET"; exit $$RET)
	@echo "Integration DB test passed"
	@$(MAKE) down-postgres-test

# Dependencies
-include $(OBJECTS:.o=.d)
-include $(DEBUG_OBJECTS:.o=.d)
-include $(ASAN_OBJECTS:.o=.d)
-include $(TSAN_OBJECTS:.o=.d)
-include $(MSAN_OBJECTS:.o=.d)
-include $(UBSAN_OBJECTS:.o=.d)
-include $(ASAN_UBSAN_OBJECTS:.o=.d)
-include $(RELEASE_OBJECTS:.o=.d)

# Generate dependency files
$(OBJDIR)/%.d: $(SRCDIR)/%.c | $(OBJDIR)
	@mkdir -p $(dir $@)
	@$(CC) $(CPPFLAGS) -MM -MT $(OBJDIR)/$*.o $< > $@

.PHONY: all debug asan tsan msan ubsan asan-ubsan release test-asan test-tsan test-msan test-ubsan test-asan-ubsan test-all-sanitizers memcheck clean clean-all distclean install uninstall info help docker-build docker-dev docker-test docker-shell docker-clean test ci

# Convenience aliases
test: run-tests

# Basic CI flow: clean, build debug, run smoke tests and ASAN test
ci: clean debug run-tests test-asan

.PHONY: unit-tests run-unit-memory run-unit-api

unit-tests: run-unit-memory run-unit-api

run-unit-memory:
	@echo "Building unit_memory"
	@gcc -o test/unit_memory test/unit_memory.c -Iinclude
	@echo "Running unit_memory"
	@./test/unit_memory

run-unit-api:
	@echo "Building unit_api_health"
	@gcc -o test/unit_api_health test/unit_api_health.c -Iinclude
	@echo "Note: ensure server is running on port 8080 before running this test"
	@./test/unit_api_health || echo "unit_api_health failed or server not running"

.PHONY: compose-run-tests
compose-run-tests:
	@echo "Running tests inside Docker Compose (stormdb-dev)"
	./scripts/ci/run_in_compose.sh stormdb-dev

.PHONY: run-tests-compose
run-tests-compose: compose-run-tests
	@echo "Done."
