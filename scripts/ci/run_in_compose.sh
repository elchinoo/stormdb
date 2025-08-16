#!/usr/bin/env bash
set -euo pipefail
# Run the project's test suite inside the stormdb-dev container in the compose network.
# Usage: scripts/ci/run_in_compose.sh [--service stormdb-dev]
SERVICE=${1:-stormdb-dev}
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
cd "$ROOT"
# Ensure compose services are up (will start stormdb-dev and postgres-test if not present)
docker compose up -d ${SERVICE} postgres-test
# Build inside container and run all tests
docker compose exec -T ${SERVICE} /bin/bash -lc "cd /workspace && make run-tests"
