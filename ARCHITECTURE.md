# StormDB (C) – Architecture and Development Guide

## Overview
StormDB is a cross-platform C11 application designed to run reproducible PostgreSQL performance experiments. It ingests metrics, persists them to a PostgreSQL database, exposes a minimal API for probes/health, and supports dynamic plugins and runtime reload.

Key goals:
- Deterministic builds and behavior across macOS and Linux.
- Robust logging with rotation and retention (size-based).
- Safe runtime reload (SIGHUP): apply logging changes, restart API on port changes, and re-establish DB connections with schema/version checks.
- Metrics pipeline: non-blocking ingestion with a consumer persisting to DB.
- DevX: strong Makefile with sanitizers, valgrind, and Docker flows (including China-friendly mirrors).

## High-level architecture
- main (src/main.c): Process lifecycle and coordination. Initializes subsystems (logging, config, pidfile, memory, database, metrics, API, plugins), handles signals, supervises graceful shutdown, and implements reload.
- Core
  - Config (src/core/config.c, include/config.h): YAML-based configuration; default provisioning; reload path.
  - Cmdline (src/core/cmdline.c, include/cmdline.h): CLI parsing and help/version output.
  - PID file (src/core/pidfile.c, include/pidfile.h): Single-instance enforcement and lifecycle management.
- Logging (src/logging/logging.c, include/logging.h): Thread-safe, leveled logs with size-based rotation and retention.
- Database (src/database/database.c, include/database.h): libpq-based connection mgmt; schema ensure; version check; query helpers; metrics persistence.
- API (src/api/coordinator.c, include/api.h): Minimal TCP listener serving health/ready responses; restartable.
- Metrics (src/metrics/metrics.c, include/metrics.h): Lock-free/ring-buffer style queue; consumer inserts into DB.
- Plugins (src/plugin/plugin.c, include/plugin.h): dlopen-based plugin loader and registry; directory load with platform-appropriate shared library extensions.
- Platform (src/platform/platform.c, include/platform.h): Cross-OS shims for threading, sockets, signals, and files.
- Memory (src/memory/memory.c, include/memory.h): Basic allocation tracking hooks for future leak checks/limits.

## Data flow
1) Startup: main loads config, sets logging (level/file/rotation), creates PID file, initializes memory, DB, metrics, API, and plugins.
2) Runtime: producers submit metric_t to ring buffer; consumer thread dequeues and uses database_insert_metric().
3) Reload (SIGHUP):
   - config_reload(), then:
   - logging_set_level(), logging_set_file(), logging_set_rotation().
   - api_restart() if port changed.
   - database_reconnect() if needed; database_ensure_schema() and database_check_version("1.0").
   - plugin_load_from_directory() if enabled.
4) Shutdown (SIGTERM/INT): stop API, stop metrics, cleanup DB/memory/logging, remove PID file.

## Build, Test and Usage (Detailed)

This section provides concrete, reproducible steps to build, test, run, and validate StormDB across macOS and Linux (including containerized flows used in regions with restricted registry access).

### Prerequisites

- A C11-capable toolchain (GCC/Clang)
- make and pkg-config
- libyaml development headers
- libpq (PostgreSQL client library) development headers
- Docker & Docker Compose (for containerized flows)
- Valgrind (Linux only) if you plan to run native memcheck

macOS (Homebrew example):

- Install dependencies:
  - brew install pkg-config libyaml libpq
  - For libpq you may need to expose pkg-config and binaries:
    - echo 'export PATH="/opt/homebrew/opt/libpq/bin:$PATH"' >> ~/.zshrc
    - echo 'export PKG_CONFIG_PATH="/opt/homebrew/opt/libpq/lib/pkgconfig:$PKG_CONFIG_PATH"' >> ~/.zshrc

Linux (Debian/Ubuntu example):

- sudo apt update && sudo apt install -y build-essential pkg-config libyaml-dev libpq-dev make

Note: For CI and sanitizer builds, ensure compiler supports the requested sanitizers (clang for MSAN/MSAN-friendly builds on Linux).

### Local build

- Build default (release-ish):
  - make
- Build debug: make debug
- Build release: make release
- Sanitizer builds:
  - make asan
  - make ubsan
  - make tsan
  - make msan (Linux/Clang only)
- Build and run tests for sanitizer variants (the targets ensure correct flags and run small smoke/unit tests):
  - make test-asan
  - make test-ubsan
  - make test-tsan
  - make test-all-sanitizers

### Running the binary (examples)

- Run with explicit config path (recommended):
  - bin/stormdb -c config/stormdb.yaml
- Show version and build information:
  - bin/stormdb --version
- Help:
  - bin/stormdb --help

Runtime notes:
- Logs default to stderr unless `logging.file` is set in the YAML config.
- Logging rotation (size-based) is configured with `logging.max_size` and `logging.max_files`.
- SIGHUP triggers a reload sequence that will:
  - reload the YAML config
  - apply new logging settings (level/file/rotation)
  - restart the API listener if its port changed
  - reconnect to the database and ensure schema + version
  - reload plugins from the configured plugin directory

Example: send SIGHUP to the running PID (from `daemon.pid_file` or printed PID):

  kill -HUP $(cat /tmp/stormdb.pid)

### Tests

- Unit & smoke tests are run via the Makefile test targets:
  - make run-tests  # runs smoke and unit tests
  - make test-asan  # runs tests built with ASAN

- The tests included cover config defaults, PID lifecycle, and plugin-directory tolerance. Add new unit tests to `test/` and wire them into the test runner.

### Memory checking (Valgrind)

- Native Linux:
  - On a Linux machine with Valgrind installed: make memcheck
  - The Makefile runs the built binary under Valgrind and stores `valgrind.log` in the workspace

- Containerized (recommended for macOS / to avoid cross-OS header leakage):
  - make memcheck-compose
    - This uses `docker compose run --rm stormdb-dev` to run `make clean && make memcheck` inside the development container that matches the Linux environment used for CI.

- China-friendly containerized memcheck (when Docker Hub / mirrors are restricted):
  - make memcheck-china
    - Builds `stormdb:china-dev` via `Dockerfile.china` using configurable local-base-image arguments and runs `make clean && make memcheck` inside that container.
    - If your environment uses internal mirrors (Tencent/DaoCloud/Aliyun), set the build-time ARGs or pre-pull the base images from your accessible registry and set them as `--build-arg BASE_IMAGE=...` during the Docker build step.

Notes:
- Always run `make clean` before cross-OS builds in containers to avoid stale dependency files referencing host-specific include paths (common cause of macOS header leakage into Linux builds).
- If you encounter registry mirror HEAD/403 errors while running Docker image IDs or locally-tagged images, prefer the compose approach or the China-optimized `Dockerfile.china` build which avoids remote lookups.

### Docker Compose / Development container

- Build and run the development container (compose):
  - docker compose build stormdb-dev
  - docker compose up -d stormdb-dev
  - docker compose run --rm stormdb-dev bash -lc "make clean && make debug && ./bin/stormdb-debug -c config/stormdb.yaml"

- Quick one-liner (dev):
  - docker compose run --rm stormdb-dev bash -lc "make clean && make debug && ./bin/stormdb-debug -c config/stormdb.yaml"

### CI

- A GitHub Actions workflow is present (Ubuntu/macOS): builds debug, runs smoke tests, runs ASAN. An UBSAN job runs on Ubuntu.
- To mirror CI locally, use act or reproduce the same matrix of build commands inside your environment (Ubuntu image, Clang for some sanitizers).

### Plugin development

- Plugins are shared libraries exposing a small ABI. Place built plugin files in the configured `plugin.plugin_dir`.
- Supported extensions: `.so` (Linux), `.dylib` (macOS). The loader handles platform extension differences.

### Troubleshooting

- Common failure: host headers (macOS) leaking into container builds — always run `make clean` before container memcheck or container builds.
- Docker registry mirror failures (403 on HEAD): use the `memcheck-compose` or `memcheck-china` flows to avoid remote HEADs for local images.
- Database connection failures: ensure `database.host/port/user/password` in the YAML are correct and reachable from the runtime (container vs host network differences).

## Where to look in the code

- src/main.c: lifecycle, signal handling, reload, and coordination.
- src/core/config.c: YAML parse, defaults, reload plumbing.
- src/core/cmdline.c: argument parsing; help/version.
- src/core/pidfile.c: PID file create/check/remove.
- src/logging/logging.c: leveled logs, rotation.
- src/database/database.c: PostgreSQL connectivity and helpers.
- src/api/coordinator.c: minimal API server with restart support.
- src/metrics/metrics.c: ring buffer and consumer thread inserting into DB.
- src/platform/platform.c: cross-platform syscalls and types.
- src/plugin/plugin.c: plugin loading and registry.
- src/memory/memory.c: memory tracker hooks.
- test/smoke_tests.c: smoke harness invoking --help/--version.
- test/unit_config_pid_plugin.c: unit checks for defaults, PID lifecycle, plugin directory tolerance.

## Objectives – implemented vs next
Implemented:
- Cross-platform build and sanitizer targets.
- Logging with rotation and retention; configurable at runtime.
- DB schema ensure; version check; metrics persistence path.
- API minimal server; restart on reload when port changes.
- SIGHUP reload covering logging, API, DB, and plugins.
- PID file lifecycle and config defaults.
- Docker/compose flows with China-friendly mirrors; Valgrind integration.
- CI for smoke + sanitizer tests.

Next steps:
- Expand unit tests: YAML edge cases, plugin symbol validation, API endpoint behavior.
- Add integration tests against postgres-test service.
- Tighten error handling and backoff in database_reconnect.
- Extend API to expose metrics and health endpoints (JSON).
- Optional: structured logging (JSON), bounded memory pools.
- TSAN job in CI (Linux only) and optional MSAN under Clang/Linux.

## Conventions
- Warnings-as-errors across builds; strict flags.
- Public APIs documented in headers; internal helpers documented inline.
- Avoid platform-specific code in modules other than platform/.

---
This document should evolve with the code. Keep it accurate and pragmatic.
