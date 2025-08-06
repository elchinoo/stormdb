# TPC-C Scalability Plugin - API Examples

This document provides curl examples for interacting with the TPC-C scalability plugin through the StormDB v2 API.

## Prerequisites

1. StormDB v2 is running: `./build/stormdb -config ./config/core.yaml`
2. PostgreSQL database is available and accessible
3. TPC-C plugin is built and loaded: `make plugin` in the plugin directory

## API Base URL

```bash
export API_BASE="http://localhost:8080"
```

## 1. Check API Health

```bash
curl -X GET "${API_BASE}/health" \
  -H "Content-Type: application/json" | jq
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2025-08-05T18:30:00Z",
  "services": {
    "database": "healthy",
    "scheduler": "running"
  }
}
```

## 2. List Available Plugins

```bash
curl -X GET "${API_BASE}/plugins" \
  -H "Content-Type: application/json" | jq
```

Expected response:
```json
[
  {
    "name": "tpcc-scalability",
    "version": "1.0.0",
    "description": "TPC-C inspired scalability test with incremental connection testing",
    "author": "StormDB Team",
    "test_types": ["scalability", "tpcc", "oltp"],
    "status": "loaded"
  }
]
```

## 3. Get Plugin Details

```bash
curl -X GET "${API_BASE}/plugins/tpcc-scalability" \
  -H "Content-Type: application/json" | jq
```

## 4. Create a TPC-C Test Run

### Quick Test (2 minutes, 2 connection levels)

```bash
curl -X POST "${API_BASE}/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "TPC-C Quick Test",
    "description": "Quick TPC-C scalability test with 2 connection levels",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "tpcc_test",
      "username": "postgres",
      "password": "postgres",
      "ssl_mode": "disable",
      "scale": 5,
      "connections": [24, 48],
      "duration": "2m",
      "warmup_time": "15s",
      "think_time": "100ms",
      "new_order_pct": 45,
      "payment_pct": 43,
      "order_status_pct": 4,
      "delivery_pct": 4,
      "stock_level_pct": 4,
      "batch_size": 100,
      "enable_metrics": true,
      "log_transactions": false
    }
  }' | jq
```

### Standard Test (5 minutes, 4 connection levels)

```bash
curl -X POST "${API_BASE}/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "TPC-C Standard Scalability Test",
    "description": "Standard TPC-C test with incremental connection scaling",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "tpcc_test",
      "username": "postgres",
      "password": "postgres",
      "ssl_mode": "disable",
      "scale": 10,
      "connections": [48, 96, 192, 256],
      "duration": "5m",
      "warmup_time": "30s",
      "think_time": "100ms",
      "new_order_pct": 45,
      "payment_pct": 43,
      "order_status_pct": 4,
      "delivery_pct": 4,
      "stock_level_pct": 4,
      "batch_size": 100,
      "enable_metrics": true,
      "log_transactions": false
    }
  }' | jq
```

### Stress Test (10 minutes, high connection counts)

```bash
curl -X POST "${API_BASE}/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "TPC-C Stress Test",
    "description": "High-load TPC-C test with aggressive connection scaling",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "tpcc_test",
      "username": "postgres",
      "password": "postgres",
      "ssl_mode": "disable",
      "scale": 20,
      "connections": [128, 256, 512, 1000],
      "duration": "10m",
      "warmup_time": "1m",
      "think_time": "50ms",
      "new_order_pct": 50,
      "payment_pct": 40,
      "order_status_pct": 3,
      "delivery_pct": 3,
      "stock_level_pct": 4,
      "batch_size": 200,
      "enable_metrics": true,
      "log_transactions": false
    }
  }' | jq
```

Expected response (for any test):
```json
{
  "test_run_id": 123,
  "status": "pending",
  "message": "Test run created successfully"
}
```

## 5. Monitor Test Run Status

```bash
# Get specific test run status
TEST_RUN_ID=123
curl -X GET "${API_BASE}/test-runs/${TEST_RUN_ID}" \
  -H "Content-Type: application/json" | jq

# List all test runs
curl -X GET "${API_BASE}/test-runs" \
  -H "Content-Type: application/json" | jq

# List test runs with pagination
curl -X GET "${API_BASE}/test-runs?limit=10&offset=0" \
  -H "Content-Type: application/json" | jq
```

## 6. Get Test Results

```bash
# Get all results for a test run
TEST_RUN_ID=123
curl -X GET "${API_BASE}/test-runs/${TEST_RUN_ID}/results" \
  -H "Content-Type: application/json" | jq

# Get results for specific metric
curl -X GET "${API_BASE}/results/metrics/ROW_INSERT?limit=100" \
  -H "Content-Type: application/json" | jq
```

## 7. Cancel Running Test

```bash
TEST_RUN_ID=123
curl -X POST "${API_BASE}/test-runs/${TEST_RUN_ID}/cancel" \
  -H "Content-Type: application/json" | jq
```

## 8. System Status and Monitoring

```bash
# Get overall system status
curl -X GET "${API_BASE}/status" \
  -H "Content-Type: application/json" | jq

# Reload plugins (useful during development)
curl -X POST "${API_BASE}/plugins/reload" \
  -H "Content-Type: application/json" | jq
```

## 9. Real-time Monitoring Script

```bash
#!/bin/bash
# monitor-tpcc.sh - Monitor a running TPC-C test

if [ -z "$1" ]; then
    echo "Usage: $0 <test_run_id>"
    exit 1
fi

TEST_RUN_ID=$1
API_BASE="http://localhost:8080"

echo "Monitoring TPC-C test run: $TEST_RUN_ID"
echo "Press Ctrl+C to stop monitoring"

while true; do
    clear
    echo "=== TPC-C Test Run Monitor ==="
    echo "Test Run ID: $TEST_RUN_ID"
    echo "Time: $(date)"
    echo
    
    # Get test run status
    curl -s "${API_BASE}/test-runs/${TEST_RUN_ID}" | jq -r '
        "Status: \(.status)",
        "Plugin: \(.plugin_name) v\(.plugin_version)",
        "Name: \(.name)",
        "Started: \(.start_time // "Not started")",
        "Duration: \(if .start_time and .end_time then 
            (.end_time | strptime("%Y-%m-%dT%H:%M:%S") | mktime) - 
            (.start_time | strptime("%Y-%m-%dT%H:%M:%S") | mktime) 
        else "Running..." end)"
    '
    
    echo
    echo "=== System Status ==="
    curl -s "${API_BASE}/status" | jq -r '
        "Scheduler: \(.scheduler.running)",
        "Workers: \(.scheduler.worker_count)",
        "Queue: \(.scheduler.queue_length) tasks"
    '
    
    sleep 5
done
```

## 10. Results Analysis Script

```bash
#!/bin/bash
# analyze-tpcc.sh - Analyze TPC-C test results

if [ -z "$1" ]; then
    echo "Usage: $0 <test_run_id>"
    exit 1
fi

TEST_RUN_ID=$1
API_BASE="http://localhost:8080"

echo "Analyzing TPC-C test results for run: $TEST_RUN_ID"

# Get test summary
curl -s "${API_BASE}/test-runs/${TEST_RUN_ID}" | jq '
    {
        "test_run": .id,
        "name": .name,
        "status": .status,
        "duration": .end_time,
        "config": {
            "scale": .config.scale,
            "connections": .config.connections,
            "duration": .config.duration
        }
    }'

echo
echo "=== Performance Results ==="

# Get and analyze results
curl -s "${API_BASE}/test-runs/${TEST_RUN_ID}/results" | jq -r '
    group_by(.tags | fromjson | .connections) | 
    map({
        "connections": (.[0].tags | fromjson | .connections),
        "count": length,
        "avg_latency_ms": (map(.value / 1000000) | add / length),
        "min_latency_ms": (map(.value / 1000000) | min),
        "max_latency_ms": (map(.value / 1000000) | max)
    }) |
    sort_by(.connections) |
    .[] |
    "Connections: \(.connections), Count: \(.count), Avg: \(.avg_latency_ms | round)ms, Min: \(.min_latency_ms | round)ms, Max: \(.max_latency_ms | round)ms"
'
```

## Environment Variables

You can also use environment variables for configuration:

```bash
export STORMDB_API_HOST=localhost
export STORMDB_API_PORT=8080
export STORMDB_DATABASE_HOST=localhost
export STORMDB_DATABASE_PORT=5432
export STORMDB_DATABASE_NAME=tpcc_test
export STORMDB_DATABASE_USER=postgres
export STORMDB_DATABASE_PASSWORD=postgres
```

## Error Handling

Common HTTP status codes:
- `200 OK` - Request successful
- `201 Created` - Test run created
- `400 Bad Request` - Invalid configuration
- `404 Not Found` - Test run or plugin not found
- `409 Conflict` - Plugin already running
- `500 Internal Server Error` - Server error

Example error response:
```json
{
  "error": "Configuration validation failed",
  "message": "connection count must be positive: -1",
  "timestamp": "2025-08-05T18:30:00Z"
}
```
