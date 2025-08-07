# TPC-C Scalability Plugin - Enhanced Configuration Examples

## Basic Quick Test
```json
{
  "plugin_name": "tpcc-scalability",
  "name": "TPC-C Quick Test",
  "description": "Quick test with enhanced metrics tracking",
  "config": {
    "host": "localhost",
    "port": 5432,
    "database": "testdb",
    "username": "postgres",
    "password": "postgres",
    "ssl_mode": "disable",
    "mode": "full",
    "scale": 1,
    "connections": [5, 10],
    "duration": "2m",
    "warmup_time": "10s",
    "think_time": "100ms",
    "new_order_pct": 45,
    "payment_pct": 43,
    "order_status_pct": 4,
    "delivery_pct": 4,
    "stock_level_pct": 4,
    "enable_metrics": true,
    "stream_metrics": true,
    "metrics_interval": "1s",
    "max_error_rate": 0.05,
    "stop_on_error_limit": true,
    "verbose": true
  }
}
```

## Multi-Level Scalability Test
```json
{
  "plugin_name": "tpcc-scalability",
  "name": "TPC-C Multi-Level Scalability",
  "description": "Test scalability across multiple connection levels with enhanced metrics",
  "config": {
    "host": "localhost", 
    "port": 5432,
    "database": "tpcc_test",
    "username": "postgres",
    "password": "postgres",
    "ssl_mode": "disable",
    "mode": "full",
    "scale": 10,
    "connections": [10, 25, 50, 100, 200],
    "duration": "15m",
    "warmup_time": "30s",
    "think_time": "10ms",
    "new_order_pct": 45,
    "payment_pct": 43,
    "order_status_pct": 4,
    "delivery_pct": 4,
    "stock_level_pct": 4,
    "enable_metrics": true,
    "stream_metrics": true,
    "metrics_interval": "1s",
    "max_error_rate": 0.10,
    "stop_on_error_limit": true,
    "verbose": true
  }
}
```

## Supplier Reorder Extension Test
```json
{
  "plugin_name": "tpcc-scalability",
  "name": "TPC-C with Supplier Reorder",
  "description": "TPC-C test including supplier reorder transactions",
  "config": {
    "host": "localhost",
    "port": 5432, 
    "database": "tpcc_supplier",
    "username": "postgres",
    "password": "postgres",
    "ssl_mode": "disable",
    "mode": "full",
    "scale": 5,
    "connections": [20, 40],
    "duration": "10m",
    "warmup_time": "20s",
    "think_time": "50ms",
    "new_order_pct": 40,
    "payment_pct": 38,
    "order_status_pct": 4,
    "delivery_pct": 4,
    "stock_level_pct": 4,
    "enable_supplier_reorder": true,
    "supplier_reorder_pct": 10,
    "enable_metrics": true,
    "stream_metrics": true,
    "metrics_interval": "1s",
    "max_error_rate": 0.05,
    "stop_on_error_limit": true,
    "verbose": true
  }
}
```

## Setup Only Mode
```json
{
  "plugin_name": "tpcc-scalability",
  "name": "TPC-C Schema Setup",
  "description": "Setup TPC-C schema and data only",
  "config": {
    "host": "localhost",
    "port": 5432,
    "database": "tpcc_setup",
    "username": "postgres", 
    "password": "postgres",
    "ssl_mode": "disable",
    "mode": "setup",
    "scale": 20,
    "verbose": true
  }
}
```

## Run Only Mode (After Setup)
```json
{
  "plugin_name": "tpcc-scalability",
  "name": "TPC-C Run Only",
  "description": "Run tests on pre-setup database",
  "config": {
    "host": "localhost",
    "port": 5432,
    "database": "tpcc_setup",
    "username": "postgres",
    "password": "postgres", 
    "ssl_mode": "disable",
    "mode": "run",
    "scale": 20,
    "connections": [50, 100, 150],
    "duration": "20m",
    "warmup_time": "60s",
    "think_time": "5ms",
    "new_order_pct": 45,
    "payment_pct": 43,
    "order_status_pct": 4,
    "delivery_pct": 4,
    "stock_level_pct": 4,
    "enable_metrics": true,
    "stream_metrics": true,
    "metrics_interval": "1s",
    "max_error_rate": 0.03,
    "stop_on_error_limit": true,
    "verbose": false
  }
}
```

## High-Performance Test
```json
{
  "plugin_name": "tpcc-scalability", 
  "name": "High-Performance TPC-C",
  "description": "High throughput test with minimal think time",
  "config": {
    "host": "localhost",
    "port": 5432,
    "database": "tpcc_perf",
    "username": "postgres",
    "password": "postgres",
    "ssl_mode": "disable", 
    "mode": "rebuild",
    "scale": 50,
    "connections": [100, 200, 300, 400],
    "duration": "30m",
    "warmup_time": "2m",
    "think_time": "1ms",
    "new_order_pct": 45,
    "payment_pct": 43,
    "order_status_pct": 4,
    "delivery_pct": 4,
    "stock_level_pct": 4,
    "enable_metrics": true,
    "stream_metrics": true,
    "metrics_interval": "1s",
    "max_error_rate": 0.02,
    "stop_on_error_limit": true,
    "verbose": false
  }
}
```

# Key Features of Enhanced TPC-C Plugin

## Enhanced Metrics System
- **Atomic Operations**: All metrics use atomic operations for thread safety
- **Batched Updates**: Workers accumulate metrics for 500ms before batch update
- **Comprehensive Tracking**: Tracks operations, rows, latency, and errors separately
- **Per-Second Reporting**: Metrics reported exactly once per second, not per transaction

## Connection Level Testing  
- **Multiple Levels**: Test with different connection counts in sequence
- **Duration Distribution**: Total duration divided equally among connection levels
- **Separate Warmup**: Each level gets its own warmup period
- **Independent Metrics**: Metrics reset between levels for clean comparison

## Error Rate Limiting
- **Configurable Threshold**: Set maximum acceptable error rate (0.0-1.0)
- **Real-time Monitoring**: Checks error rate per operation and per second
- **Graceful Shutdown**: Stops workload when error rate exceeded
- **Worker-level Detection**: Individual workers can trigger shutdown

## Transaction Types
- **Standard TPC-C**: New-Order, Payment, Order-Status, Delivery, Stock-Level
- **Supplier Reorder**: Optional extension for supply chain simulation
- **Configurable Mix**: Percentage-based transaction type selection
- **Enhanced Tracking**: Each transaction type tracked separately

## Operational Modes
- **setup**: Create schema and populate data only
- **run**: Execute tests on existing data
- **rebuild**: Drop tables, recreate schema, populate data, then run tests
- **full**: Ensure schema exists, populate if needed, then run tests

## Real-time Metrics Structure
```json
{
  "dt_started": "2025-08-07T15:30:00Z",
  "dt_end": "2025-08-07T15:30:01Z", 
  "num_connections": 50,
  "num_insert": 1250,
  "num_update": 890,
  "num_delete": 45,
  "num_select": 2300,
  "latency_sum": 15000000000,
  "latency_count": 4485,
  "num_row_insert": 1250,
  "num_row_update": 890,
  "num_row_delete": 45,
  "num_row_select": 3200,
  "avg_latency_ms": 3.34,
  "tps": 4485.0,
  "error_rate": 0.02
}
```

## Performance Optimizations
- **Lock-free Metrics**: Atomic operations minimize contention
- **Batch Processing**: Worker metrics batched every 500ms
- **Connection Pooling**: Efficient database connection management
- **Think Time Control**: Configurable delays between transactions
- **Warmup Separation**: Warmup metrics don't affect measurement phase
