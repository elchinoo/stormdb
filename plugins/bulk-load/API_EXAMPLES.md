# Bulk Load Plugin API Examples

This document provides examples of how to use the Bulk Load plugin via the StormDB v2 REST API.

## Base URL

All API calls assume StormDB v2 is running on `http://localhost:8080`

## 1. Create and Start a Bulk Load Test

### Quick Test (Small Batches)
```bash
curl -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "bulk-load",
    "name": "Quick Bulk Load Test",
    "description": "Testing small batch sizes with minimal duration",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "stormdb",
      "username": "postgres",
      "password": "postgres",
      "ssl_mode": "disable",
      "batch_sizes": [1, 100],
      "connections": 10,
      "duration": "2m",
      "warmup_time": "10s",
      "think_time": "5ms",
      "table_name": "quick_bulk_test",
      "drop_table": true,
      "data_columns": 5,
      "verbose": true
    }
  }'
```

### Standard Bulk Load Test
```bash
curl -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "bulk-load",
    "name": "Standard Bulk Load Performance Test",
    "description": "Testing standard batch sizes: 1, 1000, 10000, 50000 rows",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "stormdb_test",
      "username": "stormdb",
      "password": "stormdb_password",
      "batch_sizes": [1, 1000, 10000, 50000],
      "connections": 20,
      "duration": "5m",
      "warmup_time": "30s",
      "table_name": "bulk_test_data",
      "data_columns": 10,
      "index_columns": ["data_int_4", "data_text_1"]
    }
  }'
```

### Large Scale Performance Test
```bash
curl -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "bulk-load",
    "name": "Large Scale Bulk Load Test",
    "description": "Testing large batch sizes with high connection count",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "stormdb_test",
      "username": "stormdb",
      "password": "stormdb_password",
      "batch_sizes": [1000, 10000, 50000, 100000],
      "connections": 50,
      "duration": "10m",
      "warmup_time": "1m",
      "data_columns": 20,
      "index_columns": ["data_int_4", "data_text_1", "data_float_6"]
    }
  }'
```

## 2. Monitor Test Progress

### Get Test Run Status
```bash
# Replace {run_id} with the ID returned from the create test call
curl "http://localhost:8080/test-runs/{run_id}"
```

### Get All Active Test Runs
```bash
curl "http://localhost:8080/test-runs?status=running"
```

### Real-time Monitoring (with jq for formatting)
```bash
#!/bin/bash
RUN_ID=$1

while true; do
  echo "=== $(date) ==="
  curl -s "http://localhost:8080/test-runs/${RUN_ID}" | jq '{
    id: .id,
    status: .status,
    name: .name,
    started: .start_time,
    duration: (if .end_time then (.end_time | fromdateiso8601) - (.start_time | fromdateiso8601) else null end)
  }'
  sleep 5
done
```

## 3. Retrieve Test Results

### Get Complete Test Results
```bash
curl "http://localhost:8080/test-runs/{run_id}/results" | jq '.'
```

### Get Formatted Summary Results
```bash
curl -s "http://localhost:8080/test-runs/{run_id}/results" | jq '{
  test_summary: {
    total_transactions: .total_transactions,
    total_rows_inserted: .total_rows_inserted,
    total_errors: .total_errors,
    test_duration: (.end_time | fromdateiso8601) - (.start_time | fromdateiso8601)
  },
  batch_results: [
    .batch_results[] | {
      batch_size: .batch_size,
      connections: .connections,
      transactions_per_sec: .transactions_per_sec,
      rows_per_sec: .rows_per_sec,
      avg_latency_ms: .avg_latency_ms,
      error_rate: .error_rate
    }
  ]
}'
```

### Extract Performance Metrics Only
```bash
curl -s "http://localhost:8080/test-runs/{run_id}/results" | jq '.batch_results[] | {
  batch_size,
  transactions_per_sec,
  rows_per_sec,
  avg_latency_ms
}' | sort_by(.batch_size)
```

## 4. Cancel Running Test

```bash
curl -X DELETE "http://localhost:8080/test-runs/{run_id}"
```

## 5. List All Test Runs (with filtering)

### Get All Test Runs
```bash
curl "http://localhost:8080/test-runs"
```

### Get Recent Bulk Load Tests
```bash
curl "http://localhost:8080/test-runs?plugin_name=bulk-load&limit=10"
```

### Get Failed Tests
```bash
curl "http://localhost:8080/test-runs?status=failed"
```

## 6. Advanced Monitoring Scripts

### Performance Comparison Script
```bash
#!/bin/bash
# compare_bulk_performance.sh

RUN_ID1=$1
RUN_ID2=$2

echo "Comparing bulk load performance between runs $RUN_ID1 and $RUN_ID2"
echo

echo "=== Run $RUN_ID1 ==="
curl -s "http://localhost:8080/test-runs/${RUN_ID1}/results" | jq -r '
  .batch_results[] | 
  "Batch Size: \(.batch_size) | TPS: \(.transactions_per_sec | floor) | Rows/sec: \(.rows_per_sec | floor) | Latency: \(.avg_latency_ms)ms"
'

echo
echo "=== Run $RUN_ID2 ==="
curl -s "http://localhost:8080/test-runs/${RUN_ID2}/results" | jq -r '
  .batch_results[] | 
  "Batch Size: \(.batch_size) | TPS: \(.transactions_per_sec | floor) | Rows/sec: \(.rows_per_sec | floor) | Latency: \(.avg_latency_ms)ms"
'
```

### Live Progress Monitor
```bash
#!/bin/bash
# monitor_bulk_load.sh

RUN_ID=$1
REFRESH_INTERVAL=${2:-5}

echo "Monitoring bulk load test run $RUN_ID (refresh every ${REFRESH_INTERVAL}s)"
echo "Press Ctrl+C to stop monitoring"
echo

while true; do
  clear
  echo "=== Bulk Load Test Monitor === $(date)"
  echo
  
  # Get test status
  STATUS=$(curl -s "http://localhost:8080/test-runs/${RUN_ID}" | jq -r '.status')
  NAME=$(curl -s "http://localhost:8080/test-runs/${RUN_ID}" | jq -r '.name')
  
  echo "Test: $NAME"
  echo "Status: $STATUS"
  echo
  
  if [ "$STATUS" = "running" ]; then
    echo "⏳ Test is currently running..."
    echo
    
    # Show any available intermediate results
    curl -s "http://localhost:8080/test-runs/${RUN_ID}/results" 2>/dev/null | jq -r '
      if .batch_results then
        "Completed batch tests:",
        (.batch_results[] | "  • Batch \(.batch_size): \(.transactions_per_sec | floor) TPS, \(.avg_latency_ms)ms avg latency")
      else
        "No results available yet..."
      end
    ' 2>/dev/null || echo "No results available yet..."
    
  elif [ "$STATUS" = "succeeded" ]; then
    echo "✅ Test completed successfully!"
    echo
    
    curl -s "http://localhost:8080/test-runs/${RUN_ID}/results" | jq -r '
      "Summary:",
      "  Total Transactions: \(.total_transactions)",
      "  Total Rows Inserted: \(.total_rows_inserted)",
      "  Total Errors: \(.total_errors)",
      "",
      "Results by Batch Size:",
      (.batch_results[] | "  • \(.batch_size) rows: \(.transactions_per_sec | floor) TPS, \(.rows_per_sec | floor) rows/sec, \(.avg_latency_ms)ms latency")
    '
    break
    
  elif [ "$STATUS" = "failed" ]; then
    echo "❌ Test failed"
    break
    
  else
    echo "Status: $STATUS"
  fi
  
  sleep $REFRESH_INTERVAL
done
```

### CSV Export Script
```bash
#!/bin/bash
# export_bulk_results.sh

RUN_ID=$1
OUTPUT_FILE=${2:-bulk_load_results_${RUN_ID}.csv}

echo "Exporting bulk load results to $OUTPUT_FILE"

# CSV Header
echo "batch_size,connections,total_transactions,total_rows_inserted,duration_seconds,transactions_per_sec,rows_per_sec,avg_latency_ms,min_latency_ms,max_latency_ms,error_rate" > "$OUTPUT_FILE"

# Export data
curl -s "http://localhost:8080/test-runs/${RUN_ID}/results" | jq -r '
  .batch_results[] | 
  [.batch_size, .connections, .total_transactions, .total_rows_inserted, .duration_seconds, .transactions_per_sec, .rows_per_sec, .avg_latency_ms, .min_latency_ms, .max_latency_ms, .error_rate] | 
  @csv
' >> "$OUTPUT_FILE"

echo "Results exported to $OUTPUT_FILE"
```

## 7. Environment Variables

You can use environment variables for sensitive configuration:

```bash
export BULK_DB_PASSWORD="your_secure_password"

curl -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "bulk-load",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "stormdb_test",
      "username": "stormdb",
      "password": "'$BULK_DB_PASSWORD'",
      "batch_sizes": [1, 1000, 10000, 50000]
    }
  }'
```

## 8. Troubleshooting API Calls

### Check Plugin Availability
```bash
curl "http://localhost:8080/plugins" | jq '.[] | select(.name == "bulk-load")'
```

### Validate Configuration Before Creating Test
```bash
curl -X POST "http://localhost:8080/plugins/bulk-load/validate" \
  -H "Content-Type: application/json" \
  -d '{
    "host": "localhost",
    "port": 5432,
    "database": "stormdb_test",
    "username": "stormdb",
    "password": "stormdb_password",
    "batch_sizes": [1, 1000, 10000, 50000]
  }'
```

### Get StormDB Health Status
```bash
curl "http://localhost:8080/health"
```

### Get API Version
```bash
curl "http://localhost:8080/version"
```

These examples provide a comprehensive guide for interacting with the Bulk Load plugin through the StormDB v2 REST API. Adjust the configuration parameters based on your specific testing requirements and database setup.
