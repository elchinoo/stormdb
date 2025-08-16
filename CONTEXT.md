StormDB Project Context

Purpose

This file captures the current project status, what has been implemented, known issues, development workflows, and next steps so you (or the assistant) can resume work without loss of context.

When to open this file

- Before making large changes
- Before running containerized memcheck or CI
- When continuing feature work (API, tests, DB handling)

Status snapshot (as of 2025-08-17)

- Language: C11
- Build: Makefile with debug/release and sanitizer targets (ASAN/TSAN/UBSAN/MSAN)
- CI: GitHub Actions (Ubuntu/macOS, ASAN; UBSAN on Ubuntu)
- Docker: Multi-stage Dockerfiles; Docker Compose; Dockerfile.china and memcheck-china flow for China-friendly builds
- Tests: Smoke tests and unit tests (config defaults, PID lifecycle, plugin dir tolerance)
- Valgrind: memcheck flows work inside Linux containers; memcheck-china accounts for registry mirror issues

What is implemented

- Core subsystems:
  - YAML configuration parsing and defaults (`include/config.h`, `src/core/config.c`)
  - Logging: thread-safe, leveled, file output, size-based rotation and retention (`include/logging.h`, `src/logging/logging.c`)
  - PID file management (`include/pidfile.h`, `src/core/pidfile.c`)
  - Minimal API server (TCP listener) with restart support on port change (`include/api.h`, `src/api/coordinator.c`)
  - Metrics pipeline: ring buffer ingestion and consumer persisting metrics to DB (`include/metrics.h`, `src/metrics/metrics.c`)
  - Database abstraction using libpq with schema ensure and version check (`include/database.h`, `src/database/database.c`)
  - Plugin loader (dlopen) with directory load handling for platform-specific extensions (`include/plugin.h`, `src/plugin/plugin.c`)
  - Signal handling: SIGINT/SIGTERM for graceful shutdown; SIGHUP reload implemented to apply logging, API, DB, and plugin changes
  - Memory tracking hooks (`include/memory.h`, `src/memory/memory.c`)

- Dev tooling and docs:
  - Makefile: memcheck, memcheck-compose, memcheck-china, sanitizer targets, test runner
  - scripts/valgrind-test.sh and other helper scripts
  - Dockerfile.china for China-friendly base image usage
  - ARCHITECTURE.md and README.md updated with build/run/test instructions
  - CONTRIBUTING.md and LICENSE (GPLv3 placeholder) added
  - Unit and smoke tests under `test/` build and run locally

Known issues and important notes

- Cross-OS build artifacts: stale dependency files may reference macOS Homebrew include paths; always run `make clean` before building in containers
- Docker registry mirrors (Tencent/Aliyun/etc.) may intercept HEAD requests for image resolution causing 403 for local images; prefer memcheck-compose or memcheck-china flows to avoid remote lookups
- Dockerfile.china may emit a harmless warning about undeclared build ARGs for production multi-stage builds; dev target is functional

Reproducible commands and flows

Local (macOS/Linux host):
- Build debug: make debug
- Run binary: ./bin/stormdb -c config/stormdb.yaml
- Run smoke tests: make run-tests

Valgrind (Linux):
- Native (Linux host with valgrind): make memcheck
- Container (recommended on macOS): make memcheck-compose
- China-friendly container: make memcheck-china

Docker dev container:
- docker compose build stormdb-dev
- docker compose up -d stormdb-dev
- docker compose run --rm stormdb-dev bash -lc "make clean && make debug && ./bin/stormdb-debug -c config/stormdb.yaml"

CI (GitHub Actions):
- Workflow runs debug builds on Ubuntu/macOS and sanitizers; see .github/workflows/ci.yml

Files of interest (where to continue work)

- src/main.c - lifecycle, SIGHUP reload flow
- include/logging.h, src/logging/logging.c - logging and rotation
- include/database.h, src/database/database.c - libpq and schema/version management
- include/api.h, src/api/coordinator.c - API listener and restart logic
- include/metrics.h, src/metrics/metrics.c - metrics ingestion and DB persistence
- include/plugin.h, src/plugin/plugin.c - plugin ABI and loader
- include/config.h, src/core/config.c - YAML parsing and reload
- Makefile - builds, memcheck, docker flows, test runners
- scripts/valgrind-test.sh - helper script for valgrind flows
- Dockerfile.china - China-friendly build
- .github/workflows/ci.yml - CI matrix and jobs

Pending tasks / roadmap (short list)

1. Expand unit tests (YAML edge cases, plugin symbol validation, API endpoint behavior).
2. Integration tests with a postgres-test service (docker-compose) to validate database persistence end-to-end.
3. Improve DB reconnect backoff and error handling; instrument metrics for DB health.  <-- DONE: reconnect backoff, counters, last_error, exposed via API and metrics pipeline.
4. Implement JSON HTTP endpoints for health and metrics in the API. <-- DONE: /health endpoint implemented returning DB health JSON.
5. Add structured logging option (JSON) and optional rate-limiting or bounded memory pools.
6. Harden CI: add TSAN job (Linux), MSAN where possible, and run memcheck inside container as part of CI if feasible.

Memory manager

- Implemented a handle-based memory manager that enforces a configurable in-memory buffer limit (default 256MB). When the buffer is exhausted the manager swaps least-recently-used blocks to a temporary swap file on disk. The public API uses opaque handles; callers must use read/write APIs to access content. This guarantees the process will not allocate more than the configured in-memory budget (plus small manager overhead).

How to resume after these changes

- Config: `config/stormdb.yaml` now contains `memory.buffer_size_bytes` defaulting to 268435456 (256MB).
- Start the app as before; the memory manager is initialized at startup with the configured buffer size.
- The API `/health` endpoint is available on the configured port and returns DB health counters.
- Metrics pipeline periodically writes DB health metrics into the metrics table so they are persisted and queryable.

How to resume work quickly

- Open this file `CONTEXT.md` to refresh state.
- Build locally: make debug; run smoke tests: make run-tests
- For memcheck: prefer make memcheck-compose (containerized) and ensure `make clean` first
- To add tests: add files under `test/` and update the Makefile test runner; replicate sanitizer builds used in CI

Contact points / decision history

- Key decisions recorded here:
  - Use Makefile with multi-target sanitizers and memcheck flows
  - Provide China-friendly Dockerfile and memcheck-china target due to registry mirror behavior
  - Implement SIGHUP reload to apply logging/API/DB/plugin changes at runtime

Last update

- 2025-08-17: File created capturing current repository state, dev flows, and next steps.


