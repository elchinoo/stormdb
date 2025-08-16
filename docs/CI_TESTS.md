Running tests inside Docker Compose

This repository provides a convenient way to run the full test suite inside a Docker Compose development container so tests run in a hermetic environment with the postgres-test service available.

Quick start:

- Make sure Docker and Docker Compose are installed.
- From the repository root run:

  ./scripts/ci/run_in_compose.sh stormdb-dev

This will bring up `stormdb-dev` and `postgres-test` services and run `make run-tests` inside the container.

CI usage:

- In CI, call the script from the workspace with the same working directory (repository root). Example (GitHub Actions):

  - name: Run tests in Compose
    run: ./scripts/ci/run_in_compose.sh stormdb-dev

Notes:

- The script will start `postgres-test` and `stormdb-dev` services and run the `run-tests` Makefile target inside the container.
- The test run will produce artifacts in `./test-results` when running in the `stormdb-test` service or when configured in the Makefile.
