# StormDB Bulk Copy Plugin - API Examples

## Overview

This document provides practical examples of using the StormDB bulk-copy plugin through various interfaces including configuration files, REST API calls, and command-line usage.

## Configuration Examples

### Basic Configuration
```yaml
# Basic bulk-copy configuration
host: "localhost"
port: 5432
database: "stormdb_test"
username: "stormdb_user"
password: "stormdb_pass"
ssl_mode: "disable"

# Test parameters
batch_sizes: [5000, 25000, 100000]
connections: 20
duration: "5m"
warmup_time: "30s"
think_time: "1ms"

# COPY settings
copy_format: "CSV"
copy_delimiter: ","
copy_header: false

# Table settings
table_name: "bulk_copy_test_data"
drop_table: true
data_columns: 10
index_columns: []
```

### High-Performance Configuration
```yaml
# Maximum throughput configuration
host: "prod-db.example.com"
port: 5432
database: "performance_test"
username: "perf_user"
password: "secure_password"
ssl_mode: "require"

# Aggressive performance settings
batch_sizes: [100000, 500000, 1000000]
connections: 50
duration: "10m"
warmup_time: "1m"
think_time: "0s"

# Optimal COPY settings for speed
copy_format: "BINARY"
copy_delimiter: ","
copy_header: false

# Minimal schema for maximum speed
table_name: "high_perf_test"
drop_table: true
data_columns: 5
index_columns: []  # No indexes for maximum insert speed
rebuild: false
verbose: false
```

### Memory-Constrained Configuration
```yaml
# Configuration for limited memory environments
host: "localhost"
port: 5432
database: "small_test"
username: "test_user"
password: "test_pass"

# Conservative settings
batch_sizes: [1000, 5000, 20000]
connections: 5
duration: "2m"
warmup_time: "15s"
think_time: "10ms"

# Standard COPY settings
copy_format: "CSV"
copy_delimiter: ","

# Smaller data footprint
table_name: "small_test_data"
drop_table: true
data_columns: 3
index_columns: []
```

## REST API Examples

### Starting a Test Run

#### Request
```bash
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "bulk-copy",
    "plugin_version": "1.0.0",
    "name": "Bulk Copy Performance Test",
    "description": "Testing COPY protocol performance with various batch sizes",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "stormdb_test",
      "username": "stormdb_user",
      "password": "stormdb_pass",
      "batch_sizes": [10000, 50000, 200000],
      "connections": 25,
      "duration": "5m",
      "copy_format": "CSV",
      "data_columns": 10
    }
  }'
```

#### Response
```json
{
  "id": 12345,
  "test_type_id": 3,
  "plugin_id": 7,
  "host": "localhost",
  "port": 5432,
  "db_name": "stormdb_test",
  "name": "Bulk Copy Performance Test",
  "description": "Testing COPY protocol performance with various batch sizes",
  "status": "pending",
  "config": {
    "host": "localhost",
    "port": 5432,
    "database": "stormdb_test",
    "username": "stormdb_user",
    "password": "stormdb_pass",
    "batch_sizes": [10000, 50000, 200000],
    "connections": 25,
    "duration": "300000000000",
    "copy_format": "CSV",
    "data_columns": 10
  },
  "created_at": "2024-01-15T10:30:00Z",
  "start_time": null,
  "end_time": null
}
```

### Checking Test Status

#### Request
```bash
curl -X GET http://localhost:8080/test-runs/12345
```

#### Response
```json
{
  "id": 12345,
  "test_type_id": 3,
  "plugin_id": 7,
  "host": "localhost",
  "port": 5432,
  "db_name": "stormdb_test",
  "name": "Bulk Copy Performance Test",
  "description": "Testing COPY protocol performance with various batch sizes",
  "status": "running",
  "config": {
    "host": "localhost",
    "port": 5432,
    "database": "stormdb_test",
    "batch_sizes": [10000, 50000, 200000],
    "connections": 25,
    "duration": "300000000000",
    "copy_format": "CSV"
  },
  "created_at": "2024-01-15T10:30:00Z",
  "start_time": "2024-01-15T10:30:15Z",
  "end_time": null
}
```

### Retrieving Test Results

#### Request
```bash
curl -X GET http://localhost:8080/test-runs/12345/results
```

#### Response
```json
{
  "test_run_id": 12345,
  "results": [
    {
      "id": 67890,
      "test_run_id": 12345,
      "metric_id": 2,
      "start_time": "2024-01-15T10:30:15Z",
      "end_time": "2024-01-15T10:30:16Z",
      "value": 45000.0,
      "tags": {
        "metric_type": "interval_transaction_rate",
        "batch_size": 10000,
        "connections": 25,
        "copy_format": "CSV",
        "interval_transactions": 45000,
        "interval_seconds": 1.0
      },
      "active_connections": 25,
      "active_workers": 25
    },
    {
      "id": 67891,
      "test_run_id": 12345,
      "metric_id": 1,
      "start_time": "2024-01-15T10:30:15Z",
      "end_time": "2024-01-15T10:30:16Z",
      "value": 450000.0,
      "tags": {
        "metric_type": "interval_row_rate",
        "batch_size": 10000,
        "connections": 25,
        "copy_format": "CSV",
        "interval_rows": 450000,
        "interval_seconds": 1.0
      },
      "active_connections": 25,
      "active_workers": 25
    }
  ],
  "total_count": 2,
  "page": 1,
  "page_size": 100
}
```

### Cancelling a Test Run

#### Request
```bash
curl -X POST http://localhost:8080/test-runs/12345/cancel
```

#### Response
```json
{
  "message": "test run cancelled",
  "id": 12345
}
```

## Command Line Examples

### Basic Test Execution
```bash
# Run with default configuration
stormdb run --plugin bulk-copy --config /path/to/config.yaml

# Run with inline configuration
stormdb run --plugin bulk-copy \
  --set host=localhost \
  --set port=5432 \
  --set database=test_db \
  --set username=user \
  --set password=pass \
  --set "batch_sizes=[10000,50000,100000]" \
  --set connections=20 \
  --set duration=5m
```

### High-Performance Testing
```bash
# Maximum throughput test
stormdb run --plugin bulk-copy \
  --config high-perf.yaml \
  --set "batch_sizes=[500000,1000000]" \
  --set connections=50 \
  --set copy_format=BINARY \
  --set think_time=0s \
  --set duration=10m
```

### Quick Validation Test
```bash
# Quick test for validation
stormdb run --plugin bulk-copy \
  --set host=localhost \
  --set port=5432 \
  --set database=test \
  --set username=test \
  --set password=test \
  --set "batch_sizes=[1000]" \
  --set connections=5 \
  --set duration=30s \
  --set verbose=true
```

### Comparison Testing
```bash
# Run bulk-copy test
stormdb run --plugin bulk-copy \
  --config comparison-test.yaml \
  --output copy-results.json

# Run bulk-insert test for comparison
stormdb run --plugin bulk-insert \
  --config comparison-test.yaml \
  --output insert-results.json

# Compare results
stormdb compare copy-results.json insert-results.json
```

## Configuration File Examples

### production.yaml
```yaml
# Production environment configuration
host: "prod-db-01.company.com"
port: 5432
database: "performance_metrics"
username: "perf_tester"
password: "${POSTGRES_PASSWORD}"  # Environment variable
ssl_mode: "require"

# Production test parameters
batch_sizes: [50000, 250000, 1000000]
connections: 40
duration: "15m"
warmup_time: "2m"
think_time: "1ms"

# COPY configuration
copy_format: "CSV"
copy_delimiter: ","
copy_header: false

# Table configuration
table_name: "prod_bulk_test"
drop_table: true
data_columns: 15
index_columns: ["data_int_1", "data_int_5"]
rebuild: false
verbose: false
```

### development.yaml
```yaml
# Development environment configuration
host: "localhost"
port: 5432
database: "dev_stormdb"
username: "dev_user"
password: "dev_password"
ssl_mode: "disable"

# Development test parameters
batch_sizes: [1000, 5000, 10000]
connections: 10
duration: "2m"
warmup_time: "15s"
think_time: "10ms"

# COPY configuration
copy_format: "CSV"
copy_delimiter: ","
copy_header: false

# Table configuration
table_name: "dev_bulk_test"
drop_table: true
data_columns: 8
index_columns: []
rebuild: true  # Rebuild database each time
verbose: true  # Detailed logging
```

### ci.yaml
```yaml
# Continuous Integration test configuration
host: "ci-postgres"
port: 5432
database: "ci_test"
username: "ci_user"
password: "ci_password"
ssl_mode: "disable"

# Fast CI test parameters
batch_sizes: [1000, 5000]
connections: 5
duration: "30s"
warmup_time: "5s"
think_time: "5ms"

# COPY configuration
copy_format: "CSV"
copy_delimiter: ","

# Minimal table for CI
table_name: "ci_bulk_test"
drop_table: true
data_columns: 5
index_columns: []
rebuild: true
verbose: false
```

## Advanced Usage Examples

### Custom Data Schema Testing
```yaml
# Test with custom column types and indexes
host: "localhost"
port: 5432
database: "schema_test"
username: "test_user"
password: "test_pass"

batch_sizes: [10000, 50000]
connections: 15
duration: "3m"
copy_format: "CSV"

# Custom schema configuration
table_name: "custom_schema_test"
data_columns: 20  # More columns for complex schema
index_columns: ["data_int_1", "data_text_2", "data_int_5"]
drop_table: true
```

### Network Performance Testing
```yaml
# Test over network connection
host: "remote-db.example.com"
port: 5432
database: "network_test"
username: "network_user"
password: "network_pass"
ssl_mode: "require"

# Network-optimized settings
batch_sizes: [25000, 100000, 400000]
connections: 30
duration: "8m"
think_time: "2ms"  # Account for network latency

# Efficient format for network transfer
copy_format: "BINARY"  # Most efficient over network
```

### Memory Usage Testing
```yaml
# Test memory consumption patterns
host: "localhost"
port: 5432
database: "memory_test"
username: "memory_user"
password: "memory_pass"

# Varying batch sizes to test memory usage
batch_sizes: [1000, 10000, 100000, 1000000]
connections: 20
duration: "5m"

# Large data columns to stress memory
data_columns: 25
copy_format: "CSV"
table_name: "memory_stress_test"
```

## Error Handling Examples

### Connection Error Response
```json
{
  "error": "failed to connect to database: dial tcp 127.0.0.1:5432: connect: connection refused",
  "code": "CONNECTION_ERROR",
  "timestamp": "2024-01-15T10:30:00Z",
  "test_run_id": 12345
}
```

### Configuration Error Response
```json
{
  "error": "configuration validation failed: copy_format must be one of: CSV, TEXT, BINARY",
  "code": "VALIDATION_ERROR",
  "timestamp": "2024-01-15T10:30:00Z",
  "field": "copy_format",
  "value": "INVALID"
}
```

### Database Error Response
```json
{
  "error": "failed to execute bulk copy: pq: permission denied for table bulk_copy_test_data",
  "code": "DATABASE_ERROR",
  "timestamp": "2024-01-15T10:35:00Z",
  "test_run_id": 12345,
  "batch_size": 10000
}
```

## Monitoring and Metrics Examples

### Real-time Metrics Query
```bash
# Get real-time metrics for a running test
curl -X GET "http://localhost:8080/test-runs/12345/results?metric=interval_transaction_rate&limit=10"
```

### Performance Comparison Query
```bash
# Get results by metric type for comparison
curl -X GET "http://localhost:8080/metrics/interval_transaction_rate/results?limit=100"
```

### Historical Results Query
```bash
# Get historical results for row insertion rate
curl -X GET "http://localhost:8080/metrics/interval_row_rate/results?limit=200"
```

## Integration Examples

### Docker Compose Usage
```yaml
# docker-compose.yml
version: '3.8'
services:
  stormdb:
    image: stormdb:latest
    environment:
      - POSTGRES_HOST=postgres
      - POSTGRES_PORT=5432
      - POSTGRES_DB=stormdb
      - POSTGRES_USER=stormdb
      - POSTGRES_PASSWORD=stormdb
    depends_on:
      - postgres
    volumes:
      - ./config:/config
    command: >
      stormdb run 
      --plugin bulk-copy 
      --config /config/bulk-copy.yaml

  postgres:
    image: postgres:15
    environment:
      - POSTGRES_DB=stormdb
      - POSTGRES_USER=stormdb
      - POSTGRES_PASSWORD=stormdb
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

### Kubernetes Job Example
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: stormdb-bulk-copy-test
spec:
  template:
    spec:
      containers:
      - name: stormdb
        image: stormdb:latest
        env:
        - name: POSTGRES_HOST
          value: "postgres-service"
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        command: ["stormdb"]
        args: [
          "run",
          "--plugin", "bulk-copy",
          "--set", "host=$(POSTGRES_HOST)",
          "--set", "password=$(POSTGRES_PASSWORD)",
          "--set", "batch_sizes=[50000,200000]",
          "--set", "duration=10m"
        ]
      restartPolicy: Never
  backoffLimit: 3
```

This comprehensive set of examples demonstrates the flexibility and power of the StormDB bulk-copy plugin across various use cases and deployment scenarios.
