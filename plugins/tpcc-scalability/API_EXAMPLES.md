# TPC-C Scalability Plugin API Examples

This document provides comprehensive examples of using the TPC-C Scalability plugin through the StormDB REST API.

## Table of Contents
- [Basic Usage](#basic-usage)
- [Configuration Examples](#configuration-examples)
- [Advanced Testing](#advanced-testing)
- [Monitoring and Results](#monitoring-and-results)
- [Performance Testing Scenarios](#performance-testing-scenarios)

## Basic Usage

### Simple TPC-C Test
```bash
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "Basic TPC-C Test",
    "description": "Simple TPC-C test with default settings",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "testdb",
      "username": "postgres",
      "password": "postgres",
      "ssl_mode": "disable",
      "scale": 1,
      "connections": [10, 20],
      "duration": "2m",
      "warmup_time": "30s",
      "rebuild": true
    }
  }'
```

### Quick Development Test
```bash
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "Quick Dev Test",
    "description": "Fast test for development and debugging",
    "config": {
      "host": "localhost",
      "database": "testdb", 
      "username": "postgres",
      "password": "postgres",
      "scale": 1,
      "connections": [5],
      "duration": "30s",
      "warmup_time": "10s",
      "think_time": "1ms",
      "drop_tables": true,
      "rebuild": true,
      "verbose": true
    }
  }'
```

## Configuration Examples

### Standard TPC-C Compliance Test
```bash
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "TPC-C Standard Test",
    "description": "TPC-C compliant test with standard transaction mix",
    "config": {
      "host": "localhost",
      "database": "tpcc",
      "username": "postgres",
      "password": "password",
      "scale": 10,
      "connections": [50, 100, 200],
      "duration": "10m",
      "warmup_time": "2m",
      "think_time": "1ms",
      "new_order_pct": 45,
      "payment_pct": 43,
      "order_status_pct": 4,
      "delivery_pct": 4,
      "stock_level_pct": 4,
      "cross_warehouse": 15,
      "drop_tables": true,
      "rebuild": true,
      "enable_metrics": true,
      "verbose": true
    }
  }'
```

### Custom Transaction Mix
```bash
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "Custom Mix Test",
    "description": "Custom transaction mix for specific testing",
    "config": {
      "host": "localhost",
      "database": "tpcc",
      "username": "postgres",
      "password": "password",
      "scale": 5,
      "connections": [25, 50],
      "duration": "5m",
      "warmup_time": "1m",
      "new_order_pct": 60,
      "payment_pct": 30,
      "order_status_pct": 3,
      "delivery_pct": 4,
      "stock_level_pct": 3,
      "cross_warehouse": 10,
      "rebuild": true,
      "enable_metrics": true
    }
  }'
```

## Advanced Testing

### Multi-Scale Performance Test
```bash
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "Multi-Scale Performance",
    "description": "Large scale performance testing with multiple warehouse scales",
    "config": {
      "host": "performance-db.internal",
      "port": 5432,
      "database": "tpcc_perf",
      "username": "tpcc_user",
      "password": "secure_password",
      "ssl_mode": "require",
      "scale": 50,
      "connections": [100, 200, 400, 800, 1600],
      "duration": "30m",
      "warmup_time": "5m",
      "think_time": "0ms",
      "cross_warehouse": 15,
      "drop_tables": false,
      "rebuild": false,
      "enable_metrics": true,
      "log_transactions": true,
      "verbose": true
    }
  }'
```

### High-Frequency Testing
```bash
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "High Frequency Test",
    "description": "Maximum throughput testing with no think time",
    "config": {
      "host": "localhost",
      "database": "tpcc",
      "username": "postgres",
      "password": "password",
      "scale": 20,
      "connections": [500, 1000],
      "duration": "15m",
      "warmup_time": "3m",
      "think_time": "0ms",
      "new_order_pct": 45,
      "payment_pct": 43,
      "order_status_pct": 4,
      "delivery_pct": 4,
      "stock_level_pct": 4,
      "cross_warehouse": 15,
      "enable_metrics": true,
      "log_transactions": false,
      "verbose": false
    }
  }'
```

### Long-Duration Stability Test
```bash
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "24-Hour Stability Test",
    "description": "Long-duration test for stability and endurance",
    "config": {
      "host": "localhost",
      "database": "tpcc_stability",
      "username": "postgres",
      "password": "password",
      "scale": 10,
      "connections": [100],
      "duration": "24h",
      "warmup_time": "30m",
      "think_time": "10ms",
      "cross_warehouse": 15,
      "rebuild": true,
      "enable_metrics": true,
      "log_transactions": false,
      "verbose": false
    }
  }'
```

## Monitoring and Results

### Check Test Status
```bash
# Get test run status
curl -X GET "http://localhost:8080/test-runs/{test_run_id}" | jq

# Get test results
curl -X GET "http://localhost:8080/test-runs/{test_run_id}/results" | jq

# Get test logs
curl -X GET "http://localhost:8080/test-runs/{test_run_id}/logs" | jq
```

### Real-time Monitoring
```bash
# Monitor running test
while true; do
  curl -s "http://localhost:8080/test-runs/{test_run_id}" | jq '.status'
  sleep 5
done

# Monitor metrics during test
curl -s "http://localhost:8080/test-runs/{test_run_id}/results" | \
  jq '.results[] | select(.tags.metric_type == "interval_tps") | {time: .end_time, tps: .value}'
```

### Get Detailed Results
```bash
# Get final TpmC results
curl -s "http://localhost:8080/test-runs/{test_run_id}/results" | \
  jq '.results[] | select(.tags.metric_type == "total_transactions" and .tags.test_phase == "final_results")'

# Get latency statistics
curl -s "http://localhost:8080/test-runs/{test_run_id}/results" | \
  jq '.results[] | select(.tags.metric_type == "avg_latency_ms")'

# Get error information (if any)
curl -s "http://localhost:8080/test-runs/{test_run_id}" | \
  jq 'select(.error_message != null) | {error_message, error_details, logs_url}'
```

## Performance Testing Scenarios

### Scalability Analysis
```bash
# Test increasing connection levels
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "Scalability Analysis",
    "description": "Test performance across different connection levels",
    "config": {
      "host": "localhost",
      "database": "tpcc_scale",
      "username": "postgres", 
      "password": "password",
      "scale": 25,
      "connections": [10, 25, 50, 100, 200, 400, 800],
      "duration": "10m",
      "warmup_time": "2m",
      "think_time": "5ms",
      "enable_metrics": true,
      "rebuild": true
    }
  }'
```

### Warehouse Scale Comparison
```bash
# Small scale (1 warehouse)
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "Small Scale Test (1 warehouse)",
    "config": {
      "scale": 1,
      "connections": [50, 100],
      "duration": "10m",
      "host": "localhost",
      "database": "tpcc_small",
      "username": "postgres",
      "password": "password",
      "rebuild": true
    }
  }'

# Medium scale (10 warehouses)  
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability", 
    "name": "Medium Scale Test (10 warehouses)",
    "config": {
      "scale": 10,
      "connections": [50, 100],
      "duration": "10m",
      "host": "localhost",
      "database": "tpcc_medium",
      "username": "postgres",
      "password": "password",
      "rebuild": true
    }
  }'

# Large scale (100 warehouses)
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "Large Scale Test (100 warehouses)", 
    "config": {
      "scale": 100,
      "connections": [50, 100],
      "duration": "10m",
      "host": "localhost",
      "database": "tpcc_large",
      "username": "postgres",
      "password": "password", 
      "rebuild": true
    }
  }'
```

### Cross-Warehouse Impact Analysis
```bash
# Low cross-warehouse (5%)
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "Low Cross-Warehouse (5%)",
    "config": {
      "host": "localhost",
      "database": "tpcc_cross_low",
      "username": "postgres",
      "password": "password",
      "scale": 10,
      "connections": [100],
      "duration": "10m",
      "cross_warehouse": 5,
      "rebuild": true
    }
  }'

# Standard cross-warehouse (15%)
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "Standard Cross-Warehouse (15%)",
    "config": {
      "host": "localhost",
      "database": "tpcc_cross_std", 
      "username": "postgres",
      "password": "password",
      "scale": 10,
      "connections": [100],
      "duration": "10m",
      "cross_warehouse": 15,
      "rebuild": true
    }
  }'

# High cross-warehouse (50%)
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "High Cross-Warehouse (50%)",
    "config": {
      "host": "localhost",
      "database": "tpcc_cross_high",
      "username": "postgres", 
      "password": "password",
      "scale": 10,
      "connections": [100],
      "duration": "10m",
      "cross_warehouse": 50,
      "rebuild": true
    }
  }'
```

## Response Examples

### Successful Test Creation
```json
{
  "test_run_id": 15,
  "status": "pending",
  "message": "Test run created successfully",
  "plugin_name": "tpcc-scalability",
  "name": "TPC-C Scalability Test",
  "created_at": "2025-08-06T10:00:00Z"
}
```

### Test Status Response
```json
{
  "test_run_id": 15,
  "plugin_name": "tpcc-scalability",
  "name": "TPC-C Scalability Test",
  "status": "running",
  "start_time": "2025-08-06T10:00:30Z",
  "current_phase": "measurement",
  "progress": 0.6,
  "total_transactions": 45230,
  "current_tpmc": 2845.5,
  "active_connections": 100,
  "error_message": null
}
```

### Final Results
```json
{
  "test_run_id": 15,
  "plugin_name": "tpcc-scalability", 
  "status": "completed",
  "start_time": "2025-08-06T10:00:30Z",
  "end_time": "2025-08-06T10:45:30Z",
  "total_duration": "45m",
  "total_transactions": 125470,
  "final_tpmc": 2788.2,
  "avg_latency_ms": 3.5,
  "p95_latency_ms": 8.2,
  "p99_latency_ms": 15.6,
  "error_rate": 0.001,
  "scale_factor": 10,
  "connection_levels_tested": [50, 100, 200]
}
```

## Error Handling

### Configuration Validation Error
```json
{
  "error": "Configuration validation failed",
  "details": "transaction percentages must sum to 100, got 95",
  "field": "transaction_mix",
  "provided_values": {
    "new_order_pct": 45,
    "payment_pct": 40, 
    "order_status_pct": 4,
    "delivery_pct": 4,
    "stock_level_pct": 2
  }
}
```

### Database Connection Error
```json
{
  "test_run_id": 16,
  "status": "failed", 
  "error_message": "Database connection failed",
  "error_details": {
    "error": "pq: password authentication failed for user \"tpcc_user\"",
    "host": "db.example.com",
    "database": "tpcc_prod"
  },
  "logs_url": "http://localhost:8080/test-runs/16/logs"
}
```
