#!/usr/bin/env bash
# E2E integration script for tpcc-scalability plugin
# Automates setup→run→cleanup across multiple connection levels and verifies tables and metrics
set -euo pipefail

# Configuration
HOST=localhost
PORT=5432
DB=tpcc
USER=postgres
PASSWORD=postgres
API_URL=http://localhost:8080

# Helper: wait for test run completion and verify results
run_full() {
  local conns=$1
  local duration=$2

  echo "===> Running full mode with connections=${conns}, duration=${duration}"
  # Trigger full mode via API
  runID=$(curl -s -X POST "${API_URL}/test-runs" \
    -H "Content-Type: application/json" \
    -d '{"plugin_name":"tpcc-scalability","config":'"{"host":"'${HOST}'","port":'${PORT}',"database":"'${DB}'","username":"'${USER}'","password":"'${PASSWORD}'","rebuild":true,"scale":1,"connections":['"${conns}"'],"duration":"'${duration}'","warmup_time":"10s"}'"'}' | jq -r .id)
  echo "Started test run: ${runID}"

  # Poll status until completed
  while true; do
    status=$(curl -s "${API_URL}/test-runs/${runID}" | jq -r .status)
    echo "  Status: ${status}"
    if [[ "${status}" == "completed" ]]; then
      break
    fi
    sleep 5
  done

  echo "Test run ${runID} completed"
  # Verify tables exist
  for tbl in warehouse district customer item stock order order_line; do
    if ! psql "postgresql://${USER}:${PASSWORD}@${HOST}:${PORT}/${DB}" -c "\d ${tbl}" &>/dev/null; then
      echo "ERROR: Table ${tbl} not found" >&2
      exit 1
    fi
  done
  echo "All expected tables verified"

  # Verify that some metrics were stored
  result_count=$(curl -s "${API_URL}/test-runs/${runID}/results" | jq '. | length')
  echo "Stored result count: ${result_count}"
  if [[ ${result_count} -lt 1 ]]; then
    echo "ERROR: No metrics stored for run ${runID}" >&2
    exit 1
  fi
  echo "Metrics verify passed"
}

# Array of connection levels to test
declare -a LEVELS=(10 20 30)
# Duration per level
DURATION="1m"

for c in "${LEVELS[@]}"; do
  run_full ${c} ${DURATION}
done

echo "E2E tpcc-scalability integration tests PASSED"
