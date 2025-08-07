# TPC-C Scalability Plugin

A comprehensive TPC-C (Transaction Processing Performance Council Benchmark C) scalability testing plugin for StormDB v2. This plugin implements the full TPC-C specification with supplier reordering extensions to measure database performance characteristics under realistic OLTP workloads.

## Overview

TPC-C simulates a complete computing environment where a population of users executes transactions against a database. The benchmark is centered around the principal activities (transactions) of an order-entry environment:

- **New-Order** (45%): Enter a new order from a customer
- **Payment** (43%): Update customer balance to record a payment
- **Order-Status** (4%): Retrieve status of customer's most recent order
- **Delivery** (4%): Process a batch of orders for delivery
- **Stock-Level** (4%): Monitor warehouse inventory levels

## Features

### Core TPC-C Implementation
- Complete TPC-C schema with all 8 core tables
- Standard TPC-C transaction mix with configurable percentages
- Proper warehouse/district/customer hierarchical data model
- Support for multiple scale factors (1x = 1 warehouse)

### Supplier Reordering Extensions
- Extended schema with supplier management
- Purchase order and goods receipt processing
- Inventory reordering automation
- Supply chain transaction simulation

### Seed Data Management
- Multi-country support with localized data
- Realistic names and company data by region
- Configurable data populations per country
- Geographic distribution modeling

### Performance Testing
- Incremental connection testing
- Configurable warmup and measurement phases
- Real-time metrics collection
- Delta-based throughput calculation
- Comprehensive error reporting

## Schema

### Core TPC-C Tables
1. **warehouse** - Warehouse master data
2. **district** - 10 districts per warehouse
3. **customer** - 3,000 customers per district
4. **item** - Static catalog of 100,000 items
5. **stock** - Inventory (100,000 × warehouses)
6. **order** - Order headers (90,000 per district)
7. **order_line** - Order line items (~9 per order)
8. **new_order** - Pending orders queue
9. **history** - Payment transaction history

### Supplier Extensions
10. **supplier** - Supplier master data
11. **purchase_order** - Purchase orders to suppliers
12. **purchase_order_line** - PO line items
13. **goods_receipt** - Received shipments
14. **goods_receipt_line** - Receipt line items

### Seed Tables
15. **countries** - Country master data
16. **cities** - Cities by country
17. **first_names** - Localized first names
18. **family_names** - Localized family names
19. **company_names** - Company names by industry

## Configuration

```json
{
  "plugin_name": "tpcc-scalability",
  "config": {
    "host": "localhost",
    "port": 5432,
    "database": "tpcc",
    "username": "postgres", 
    "password": "password",
    "ssl_mode": "disable",
    
    "mode": "full",
    "scale": 1,
    "connections": [10, 20, 50],
    "duration": "15m",
    "warmup_time": "30s",
    "think_time": "0ms",
    
    "new_order_pct": 45,
    "payment_pct": 43,
    "order_status_pct": 4,
    "delivery_pct": 4,
    "stock_level_pct": 4,
    "cross_warehouse": 15,
    
    "enable_supplier_reorder": false,
    "supplier_reorder_pct": 0,
    "min_stock_level": 10,
    "reorder_quantity": 100,
    
    "max_error_rate": 0.05,
    "error_window": "1m",
    "stop_on_error_limit": true,
    
    "drop_tables": true,
    "rebuild": true,
    "enable_metrics": true,
    "stream_metrics": true,
    "metrics_interval": "1s",
    "log_transactions": false,
    "verbose": true
  }
}
```

### Configuration Parameters

#### Database Connection
- `host`: Database server hostname
- `port`: Database port (default: 5432)
- `database`: Target database name
- `username`: Database username
- `password`: Database password
- `ssl_mode`: SSL connection mode

#### Test Parameters
- `mode`: Execution mode - "setup", "run", "rebuild", "full" (default: "full")
- `scale`: Number of warehouses (1x = 1 warehouse)
- `connections`: Array of connection levels to test
- `duration`: Total test duration (divided by number of connection levels)
- `warmup_time`: Warmup period duration
- `think_time`: Delay between transactions

#### Execution Modes
- `setup`: Create schema and populate data only (no test execution)
- `run`: Run tests only (assumes schema and data exist)  
- `rebuild`: Drop tables, recreate schema, populate data, then run tests
- `full`: Smart setup (create/populate if needed) + run tests (default)

#### Transaction Mix
- `new_order_pct`: New order transaction percentage (default: 45%)
- `payment_pct`: Payment transaction percentage (default: 43%)
- `order_status_pct`: Order status percentage (default: 4%)
- `delivery_pct`: Delivery transaction percentage (default: 4%)
- `stock_level_pct`: Stock level percentage (default: 4%)
- `cross_warehouse`: Cross-warehouse transaction percentage (default: 15%)

#### Supplier Reordering Extensions
- `enable_supplier_reorder`: Enable supplier reordering transactions
- `supplier_reorder_pct`: Percentage of supplier reorder transactions (default: 5%)
- `min_stock_level`: Minimum stock level to trigger reorder (default: 10)
- `reorder_quantity`: Default reorder quantity (default: 100)
- `reorder_lead_time`: Lead time for reorders (default: "7d")
- `supplier_response_time`: Supplier response time variation (default: "1h")

#### Error Rate Limiting
- `max_error_rate`: Maximum error rate (0.0-1.0) before stopping test (default: 0.05)
- `error_window`: Time window for error rate calculation (default: "1m")
- `stop_on_error_limit`: Stop test when error limit is reached (default: true)

#### Real-time Metrics
- `stream_metrics`: Enable real-time metrics streaming (default: true)
- `metrics_interval`: Metrics reporting interval (default: "1s")
- `stream_batch_size`: Batch size for streaming (default: 50)
- `stream_flush_time`: Max time before forced flush (default: "1s")

#### Schema Options
- `drop_tables`: Drop existing tables before test
- `rebuild`: Repopulate data before test
- `enable_metrics`: Enable background metrics collection
- `log_transactions`: Log individual transaction details
- `verbose`: Enable verbose logging

## Metrics

The plugin collects comprehensive metrics including:

### Transaction Metrics
- Total transaction count by type
- Transaction latencies (min/max/avg/p95/p99)
- Throughput (tpmC - transactions per minute)
- Error rates and types

### System Metrics
- Active connections
- Active workers
- Database response times
- Resource utilization

### Business Metrics
- Order processing rates
- Payment volumes
- Inventory turnover
- Supply chain efficiency

## Building

```bash
# Build the plugin
make plugin

# Run tests
make test

# Clean build artifacts
make clean
```

## Usage Examples

### Basic TPC-C Test
```bash
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "Basic TPC-C Test",
    "config": {
      "host": "localhost",
      "database": "tpcc",
      "username": "postgres",
      "password": "password",
      "scale": 1,
      "connections": [10, 20],
      "duration": "2m",
      "warmup_time": "30s",
      "rebuild": true
    }
  }'
```

### Multi-Scale Performance Test
```bash
curl -X POST http://localhost:8080/test-runs \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability", 
    "name": "Multi-Scale TPC-C Test",
    "config": {
      "host": "localhost",
      "database": "tpcc",
      "username": "postgres",
      "password": "password",
      "scale": 5,
      "connections": [50, 100, 200, 400],
      "duration": "10m",
      "warmup_time": "2m",
      "think_time": "10ms",
      "enable_metrics": true,
      "verbose": true,
      "rebuild": true
    }
  }'
```

## TPC-C Standard Compliance

This implementation follows the TPC-C specification v5.11 with the following characteristics:

- **Data Scale**: Configurable warehouse count (1x = 1 warehouse)
- **Transaction Mix**: Standard 45/43/4/4/4 percentage distribution
- **Data Locality**: 85% local warehouse, 15% remote warehouse transactions
- **Response Time**: Sub-5 second 90th percentile requirement
- **Throughput Metric**: tpmC (new-order transactions per minute)

## Performance Expectations

Typical performance characteristics:

- **Small Scale (1-5 warehouses)**: 1,000-10,000 tpmC
- **Medium Scale (10-50 warehouses)**: 10,000-100,000 tpmC  
- **Large Scale (100+ warehouses)**: 100,000+ tpmC

Results depend on hardware, database configuration, and system tuning.

## References

- [TPC-C Specification v5.11](http://www.tpc.org/tpcc/)
- [TPC-C Standard Benchmark Results](http://www.tpc.org/tpcc/results/)
- [PostgreSQL Performance Tuning](https://wiki.postgresql.org/wiki/Performance_Optimization)
