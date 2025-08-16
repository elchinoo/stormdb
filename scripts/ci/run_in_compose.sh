#!/usr/bin/env bash
set -euo pipefail
# Run the project's test suite inside the stormdb-dev container in the compose network.
# Usage: scripts/ci/run_in_compose.sh [--service stormdb-dev]
SERVICE=${1:-stormdb-dev}
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
cd "$ROOT"
# Ensure compose services are up (will start stormdb-dev and postgres-test if not present)
docker compose up -d ${SERVICE} postgres-test
# Environment variables for tests to connect to postgres-test service
POSTGRES_TEST_HOST=${POSTGRES_TEST_HOST:-postgres-test}
POSTGRES_TEST_PORT=${POSTGRES_TEST_PORT:-5432}
POSTGRES_TEST_DB=${POSTGRES_TEST_DB:-stormdb_test}
POSTGRES_TEST_USER=${POSTGRES_TEST_USER:-stormdb}
POSTGRES_TEST_PASSWORD=${POSTGRES_TEST_PASSWORD:-stormdb_test}

# Build inside container and run all tests
docker compose exec -T ${SERVICE} /bin/bash -lc "cd /workspace && \
  export POSTGRES_TEST_HOST=${POSTGRES_TEST_HOST} POSTGRES_TEST_PORT=${POSTGRES_TEST_PORT} POSTGRES_TEST_DB=${POSTGRES_TEST_DB} POSTGRES_TEST_USER=${POSTGRES_TEST_USER} POSTGRES_TEST_PASSWORD=${POSTGRES_TEST_PASSWORD} && make run-tests"
