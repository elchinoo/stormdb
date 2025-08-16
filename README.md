# StormDB - Database Performance Testing Tool

StormDB is a modern database performance testing tool written in C11. It focuses on benchmarking and stress-testing PostgreSQL by executing workloads (via plugins), tracking metrics, and providing a robust, cross-platform runtime.

## Project Structure

```text
stormdb-c/
├── include/                 # Public headers
│   ├── cmdline.h            # Command line parsing
│   ├── config.h             # Configuration structures and API
│   ├── database.h           # Database abstraction (libpq)
│   ├── logging.h            # Logging API
│   ├── pidfile.h            # PID file management
│   ├── platform.h           # Cross-platform abstraction layer
│   ├── plugin.h             # Plugin API
│   └── stormdb.h            # App-wide constants (includes version)
├── src/
│   ├── main.c               # Application entry point
│   ├── core/                # Core utilities
│   │   ├── cmdline.c
│   │   ├── config.c
│   │   └── pidfile.c
│   ├── logging/             # Logging implementation
│   │   └── logging.c
│   ├── database/            # PostgreSQL integration
│   │   └── database.c
│   ├── plugin/              # Plugin loader/manager
│   │   └── plugin.c
│   └── platform/            # Cross-platform implementation
│       └── platform.c
├── config/
│   ├── stormdb.yaml         # Sample/default configuration
│   └── test.yaml            # Alternate/test configuration
├── docs/                    # Additional documentation
├── bin/                     # Built binaries (populated after build)
├── obj/                     # Object files per build type
├── VERSION.md               # Versioning guidelines
└── Makefile                 # Build configuration and tasks
```

## Features

Current, implemented components:
- Command line interface: help/version, DB/API overrides, verbose/quiet flags
- Logging: thread-safe, leveled logging to stderr or file, size-based rotation with retention
- YAML configuration: defaults + parsing of database and API sections; config reload on SIGHUP applies logging/API/DB/plugin changes
- PID file: locked PID file creation and stale PID cleanup
- PostgreSQL connectivity: libpq-based connect/reconnect and query execution
- Plugin system: dynamic loading of shared libraries from a directory (Linux/macOS extensions handled)
- Cross-platform layer: OS/arch/compiler info, files, threads, signals, sockets, time, dynamic libs
- Signals: SIGINT/SIGTERM for graceful shutdown; SIGHUP reload implemented and applies runtime changes
- Metrics pipeline: ring buffer ingestion and consumer thread persisting to DB (basic implementation)

Planned/WIP:
- Expanded unit and integration tests (more YAML edge cases and API endpoint testing)
- JSON metrics/health endpoints
- Structured logging (JSON) option

## Signal Behavior

- SIGINT/SIGTERM: graceful shutdown
- SIGHUP: reloads YAML config and applies logging, API, DB, and plugin changes at runtime

## Configuration

See `config/stormdb.yaml` for a working example. The config parser provides defaults; you may override via CLI.

Example recommended invocation:

```sh
bin/stormdb -c config/stormdb.yaml
```

## Building

### Prerequisites

- GCC or Clang with C11 support
- pkg-config
- libyaml development library
- libpq (PostgreSQL client library)
- make
- Docker & Docker Compose (optional, for containerized testing)

macOS (Homebrew):

```sh
brew install libyaml libpq pkg-config
# For libpq, you may need:
echo 'export PATH="/opt/homebrew/opt/libpq/bin:$PATH"' >> ~/.zshrc
echo 'export PKG_CONFIG_PATH="/opt/homebrew/opt/libpq/lib/pkgconfig:$PKG_CONFIG_PATH"' >> ~/.zshrc
```

### Build Commands

```sh
# Build the default binary
make

# Debug or Release builds
make debug
make release

# Platform/build info
make info

# Clean build artifacts
make clean

# Sanitizer builds and tests
make asan | tsan | ubsan | asan-ubsan
make test-asan | test-tsan | test-ubsan | test-all-sanitizers

# Memory checks
make memcheck
# Containerized memcheck (recommended on macOS)
make memcheck-compose
# China-friendly containerized memcheck
make memcheck-china
```

## Usage

```sh
# From repo root (recommended to specify config path)
bin/stormdb -c config/stormdb.yaml

# Debug build
bin/stormdb-debug --version

# Help/Version
bin/stormdb --help
bin/stormdb --version
```

Runtime notes:
- Logs default to stderr unless `logging.file` is set in the YAML config.
- Logging rotation is configured with `logging.max_size` and `logging.max_files`.
- SIGHUP triggers a reload sequence as described in Signal Behavior.

Example: send SIGHUP to the running PID (from `daemon.pid_file`):

```sh
kill -HUP $(cat /tmp/stormdb.pid)
```

## Testing

- Run unit and smoke tests via Makefile:

```sh
make run-tests
make test-asan
```

- Unit tests live in `test/` and currently cover config defaults, PID lifecycle, and plugin directory loading tolerance. Add new tests under `test/` and update `Makefile` test runner when needed.

## Memory checking (Valgrind)

- Native Linux:

```sh
make memcheck
# Produces valgrind.log
```

- Containerized (recommended for macOS / to avoid cross-OS header leakage):

```sh
make memcheck-compose
# Runs make clean && make memcheck inside stormdb-dev container
```

- China-friendly containerized memcheck (when Docker Hub / mirrors are restricted):

```sh
make memcheck-china
# Builds stormdb:china-dev using Dockerfile.china and runs memcheck inside that container
```

Notes:
- Always run `make clean` before cross-OS builds in containers to avoid stale dependency files referencing host-specific include paths.
- If you encounter registry mirror HEAD/403 errors while running Docker image IDs or locally-tagged images, prefer `memcheck-compose` or `memcheck-china`.

## Docker Compose / Development container

Build and run the development container:

```sh
docker compose build stormdb-dev
docker compose up -d stormdb-dev
docker compose run --rm stormdb-dev bash -lc "make clean && make debug && ./bin/stormdb-debug -c config/stormdb.yaml"
```

Quick one-liner (dev):

```sh
docker compose run --rm stormdb-dev bash -lc "make clean && make debug && ./bin/stormdb-debug -c config/stormdb.yaml"
```

## CI

- GitHub Actions workflow builds on Ubuntu/macOS, runs smoke + ASAN tests, and runs UBSAN on Ubuntu.

## Plugin development

- Plugins are shared libraries exposing a small ABI. Place built plugin files in the configured `plugin.plugin_dir`.
- Supported extensions: `.so` (Linux), `.dylib` (macOS).

## Version Management

Version strings live in `include/stormdb.h`:

```c
#define STORMDB_VERSION "1.0.0"
#define STORMDB_API_VERSION "1.0"
```

The `--version` output includes the compile date/time. See `VERSION.md` for guidelines on versioning and plugin compatibility.

## Contributing

- Add feature branches and open PRs. Maintain tests and CI green on the PR.
- Document API/ABI changes in `ARCHITECTURE.md` and public headers.

## License

Specify your license in this file and in a LICENSE file at the repo root.
