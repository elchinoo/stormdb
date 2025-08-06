# StormDB v2 Testing Guide

This guide provides comprehensive instructions for testing the StormDB v2 application with both the TPC-C Scalability and Bulk Load plugins.

## 🚀 Quick Start Testing

### Prerequisites

1. **PostgreSQL Database**
```bash
# Start PostgreSQL (if using Docker)
docker run --name postgres-test \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=stormdb \
  -p 5432:5432 \
  -d postgres:15

# Or use existing PostgreSQL installation
```

2. **Database Setup**
```sql
-- Connect to PostgreSQL and create test database
CREATE DATABASE stormdb_test;
CREATE USER stormdb WITH PASSWORD 'stormdb_password';
GRANT ALL PRIVILEGES ON DATABASE stormdb_test TO stormdb;
```

### Step 1: Build and Start StormDB v2

```bash
# From the v2 directory
cd /path/to/stormdb/v2

# Build everything (core + plugins)
make build-all

# Verify plugins are built
ls -la build/plugins/
# Should show: bulk-load.so and tpcc-scalability.so

# Start StormDB v2 server
STORMDB_PLUGIN_DIR=./build/plugins ./build/stormdb
```

### Step 2: Verify Service Health

```bash
# Check if StormDB is running
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","timestamp":"...","services":{"database":"healthy","scheduler":"healthy"}}

# List available plugins
curl http://localhost:8080/plugins | jq '.'

# Expected: Both "bulk-load" and "tpcc-scalability" plugins listed
```

## 🧪 Testing the Bulk Load Plugin

### Basic Bulk Load Test

```bash
# Quick 2-minute test with small batches
curl -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "bulk-load",
    "name": "Quick Bulk Load Test",
    "description": "Testing small batch sizes for 2 minutes",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "stormdb_test",
      "username": "stormdb",
      "password": "stormdb_password",
      "ssl_mode": "disable",
      "batch_sizes": [1, 100, 1000],
      "connections": 10,
      "duration": "2m",
      "warmup_time": "10s",
      "table_name": "quick_bulk_test",
      "data_columns": 5,
      "verbose": true
    }
  }'
```

### Standard Bulk Load Test

```bash
# Full test with default batch sizes
curl -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "bulk-load", 
    "name": "Standard Bulk Load Performance Test",
    "description": "Testing batch sizes: 1, 1000, 10000, 50000 rows",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "stormdb_test",
      "username": "stormdb",
      "password": "stormdb_password",
      "batch_sizes": [1, 1000, 10000, 50000],
      "connections": 20,
      "duration": "5m",
      "table_name": "bulk_test_data"
    }
  }'
```

## 🏁 Testing the TPC-C Scalability Plugin

### Quick TPC-C Test

```bash
# 2-minute TPC-C test with small connection counts
curl -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "Quick TPC-C Test",
    "description": "Testing small connection scaling for 2 minutes", 
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "stormdb_test",
      "username": "stormdb",
      "password": "stormdb_password",
      "ssl_mode": "disable",
      "scale": 5,
      "connections": [10, 20],
      "duration": "2m",
      "warmup_time": "10s"
    }
  }'
```

### Standard TPC-C Test

```bash
# Full TPC-C scalability test
curl -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "TPC-C Scalability Test", 
    "description": "Testing connection scaling: 48, 96, 192, 256",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "stormdb_test",
      "username": "stormdb",
      "password": "stormdb_password",
      "scale": 10,
      "connections": [48, 96, 192, 256],
      "duration": "5m"
    }
  }'
```

## 📊 Monitoring Test Progress

### Get Test Status

```bash
# Replace {run_id} with the ID returned from test creation
RUN_ID=1

# Check test status
curl "http://localhost:8080/test-runs/${RUN_ID}" | jq '{
  id: .id,
  name: .name,
  status: .status,
  started: .start_time,
  plugin: .plugin_name
}'
```

### Real-time Monitoring Script

Create a monitoring script:

```bash
#!/bin/bash
# monitor_test.sh
RUN_ID=$1

if [ -z "$RUN_ID" ]; then
  echo "Usage: $0 <run_id>"
  exit 1
fi

echo "Monitoring test run $RUN_ID..."
echo "Press Ctrl+C to stop monitoring"

while true; do
  STATUS=$(curl -s "http://localhost:8080/test-runs/${RUN_ID}" | jq -r '.status')
  NAME=$(curl -s "http://localhost:8080/test-runs/${RUN_ID}" | jq -r '.name')
  
  echo "$(date): $NAME - Status: $STATUS"
  
  if [ "$STATUS" = "succeeded" ] || [ "$STATUS" = "failed" ]; then
    echo "Test completed with status: $STATUS"
    break
  fi
  
  sleep 5
done
```

```bash
# Make it executable and use it
chmod +x monitor_test.sh
./monitor_test.sh 1
```

## 📈 Retrieving and Analyzing Results

### Get Complete Results

```bash
# Get full test results
curl "http://localhost:8080/test-runs/${RUN_ID}/results" | jq '.'
```

### Bulk Load Results Analysis

```bash
# Extract bulk load performance summary
curl -s "http://localhost:8080/test-runs/${RUN_ID}/results" | jq '
  if .batch_results then
    {
      plugin: "bulk-load",
      summary: {
        total_transactions: .total_transactions,
        total_rows: .total_rows_inserted,
        total_errors: .total_errors
      },
      performance: [
        .batch_results[] | {
          batch_size: .batch_size,
          connections: .connections, 
          tps: .transactions_per_sec,
          rows_per_sec: .rows_per_sec,
          avg_latency_ms: .avg_latency_ms,
          error_rate: .error_rate
        }
      ]
    }
  else
    "No results available yet"
  end
'
```

### TPC-C Results Analysis

```bash
# Extract TPC-C scalability summary
curl -s "http://localhost:8080/test-runs/${RUN_ID}/results" | jq '
  if .connection_results then
    {
      plugin: "tpcc-scalability", 
      summary: {
        total_transactions: .total_transactions,
        scale_factor: .scale,
        test_duration: .total_duration_seconds
      },
      scalability: [
        .connection_results[] | {
          connections: .connections,
          total_transactions: .total_transactions,
          tps: .tps,
          avg_latency_ms: .avg_latency_ms,
          transaction_breakdown: .transaction_breakdown
        }
      ]
    }
  else
    "No results available yet"
  end
'
```

## 🔧 Advanced Testing Scenarios

### Load Testing with Multiple Concurrent Tests

```bash
#!/bin/bash
# concurrent_tests.sh - Run multiple tests simultaneously

echo "Starting concurrent tests..."

# Start bulk load test
BULK_ID=$(curl -s -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "bulk-load",
    "config": {
      "host": "localhost", "port": 5432, "database": "stormdb_test",
      "username": "stormdb", "password": "stormdb_password",
      "batch_sizes": [1000, 10000], "connections": 15, "duration": "3m",
      "table_name": "concurrent_bulk_test"
    }
  }' | jq -r '.id')

echo "Started bulk load test: $BULK_ID"

# Start TPC-C test (different database/schema to avoid conflicts)
TPCC_ID=$(curl -s -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "config": {
      "host": "localhost", "port": 5432, "database": "stormdb_test",
      "username": "stormdb", "password": "stormdb_password", 
      "scale": 5, "connections": [25, 50], "duration": "3m"
    }
  }' | jq -r '.id')

echo "Started TPC-C test: $TPCC_ID"

# Monitor both tests
echo "Monitoring both tests..."
while true; do
  BULK_STATUS=$(curl -s "http://localhost:8080/test-runs/${BULK_ID}" | jq -r '.status')
  TPCC_STATUS=$(curl -s "http://localhost:8080/test-runs/${TPCC_ID}" | jq -r '.status')
  
  echo "$(date): Bulk Load: $BULK_STATUS, TPC-C: $TPCC_STATUS"
  
  if [ "$BULK_STATUS" != "running" ] && [ "$TPCC_STATUS" != "running" ]; then
    echo "Both tests completed"
    break
  fi
  
  sleep 10
done
```

### Performance Comparison Testing

```bash
#!/bin/bash
# performance_comparison.sh - Compare different configurations

echo "Running performance comparison tests..."

# Test 1: Low connections
LOW_CONN_ID=$(curl -s -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "bulk-load",
    "name": "Low Connections Test",
    "config": {
      "host": "localhost", "port": 5432, "database": "stormdb_test",
      "username": "stormdb", "password": "stormdb_password",
      "batch_sizes": [1000, 10000], "connections": 5, "duration": "2m",
      "table_name": "low_conn_test"
    }
  }' | jq -r '.id')

# Wait for first test to complete
while true; do
  STATUS=$(curl -s "http://localhost:8080/test-runs/${LOW_CONN_ID}" | jq -r '.status')
  if [ "$STATUS" != "running" ]; then break; fi
  sleep 5
done

# Test 2: High connections
HIGH_CONN_ID=$(curl -s -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "bulk-load",
    "name": "High Connections Test", 
    "config": {
      "host": "localhost", "port": 5432, "database": "stormdb_test",
      "username": "stormdb", "password": "stormdb_password",
      "batch_sizes": [1000, 10000], "connections": 50, "duration": "2m",
      "table_name": "high_conn_test"
    }
  }' | jq -r '.id')

echo "Comparison tests started: Low=$LOW_CONN_ID, High=$HIGH_CONN_ID"
```

## 🧪 Unit Testing

### Run Plugin Unit Tests

```bash
# Test bulk load plugin
cd plugins/bulk-load
make test

# Test TPC-C plugin  
cd ../tpcc-scalability
make test

# Run all tests from main directory
cd ../..
make test-all
```

### Coverage Testing

```bash
# Generate coverage reports
cd plugins/bulk-load
make coverage
open coverage.html  # View coverage report

cd ../tpcc-scalability
make coverage
open coverage.html
```

## 🐛 Troubleshooting

### Common Issues

#### 1. Plugin Loading Errors
```bash
# Check plugin directory
ls -la build/plugins/

# Verify plugin format
file build/plugins/*.so

# Check StormDB logs
tail -f /var/log/stormdb.log  # If logging to file
```

#### 2. Database Connection Issues
```bash
# Test database connectivity
psql -h localhost -p 5432 -U stormdb -d stormdb_test -c "SELECT 1;"

# Check database permissions
psql -h localhost -p 5432 -U stormdb -d stormdb_test -c "CREATE TABLE test_permissions (id INT);"
```

#### 3. API Connection Issues
```bash
# Check if StormDB is running
ps aux | grep stormdb

# Check port availability
netstat -an | grep 8080

# Test basic connectivity
curl -v http://localhost:8080/health
```

#### 4. Performance Issues
```bash
# Monitor system resources during tests
top -p $(pgrep stormdb)

# Check database activity
psql -h localhost -p 5432 -U postgres -c "SELECT * FROM pg_stat_activity WHERE application_name LIKE 'stormdb%';"

# Monitor disk I/O
iostat -x 1
```

### Debug Mode

```bash
# Start StormDB with debug logging
STORMDB_LOG_LEVEL=debug STORMDB_PLUGIN_DIR=./build/plugins ./build/stormdb

# Enable verbose plugin logging
# Add "verbose": true to plugin configurations
```

## 📝 Test Reporting

### Generate Test Report

```bash
#!/bin/bash
# generate_report.sh
RUN_ID=$1
PLUGIN_NAME=$2

echo "# Test Report for Run $RUN_ID" > report_${RUN_ID}.md
echo "Plugin: $PLUGIN_NAME" >> report_${RUN_ID}.md
echo "Generated: $(date)" >> report_${RUN_ID}.md
echo "" >> report_${RUN_ID}.md

# Get test details
curl -s "http://localhost:8080/test-runs/${RUN_ID}" | jq -r '
  "## Test Configuration",
  "- Name: " + .name,
  "- Status: " + .status, 
  "- Started: " + .start_time,
  "- Duration: " + (if .end_time then (.end_time | fromdateiso8601) - (.start_time | fromdateiso8601) | tostring + " seconds" else "In progress" end),
  ""
' >> report_${RUN_ID}.md

# Get results
curl -s "http://localhost:8080/test-runs/${RUN_ID}/results" | jq -r '
  "## Results Summary",
  (if .total_transactions then "- Total Transactions: " + (.total_transactions | tostring) else "" end),
  (if .total_rows_inserted then "- Total Rows Inserted: " + (.total_rows_inserted | tostring) else "" end),
  (if .total_errors then "- Total Errors: " + (.total_errors | tostring) else "" end)
' >> report_${RUN_ID}.md

echo "Report generated: report_${RUN_ID}.md"
```

This comprehensive testing guide covers all aspects of testing StormDB v2 with both plugins. Start with the quick tests to verify everything is working, then proceed to more comprehensive testing scenarios based on your needs.
