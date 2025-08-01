# TPC-C Workload Plugin

This plugin provides the TPC-C (Transaction Processing Performance Council - C) benchmark workload for PostgreSQL performance testing.

## Overview

TPC-C is an industry-standard OLTP (Online Transaction Processing) benchmark that simulates a wholesale supplier managing orders for a configurable number of warehouses. It includes:

- **New Order**: Process new customer orders
- **Payment**: Process customer payments
- **Order Status**: Query order status
- **Delivery**: Process order deliveries
- **Stock Level**: Check inventory levels

## Features

- Complete TPC-C schema implementation
- Configurable warehouse count
- All five TPC-C transaction types
- Automatic data generation
- Schema cleanup and rebuild support

## Configuration

The TPC-C workload supports the following configuration parameters:

```yaml
workload:
  type: "tpcc"
  warehouses: 10          # Number of warehouses to simulate
  scale_factor: 1.0       # Scale factor for data generation
  think_time: false       # Whether to include think time between transactions
  transaction_mix:        # Transaction mix percentages
    new_order: 45
    payment: 43
    order_status: 4
    delivery: 4
    stock_level: 4
```

## Schema

The TPC-C workload creates the following tables:
- warehouse
- district
- customer
- history
- new_orders
- orders
- order_line
- item
- stock

## Performance Characteristics

TPC-C is CPU and I/O intensive, testing:
- Complex multi-table joins
- Transaction isolation
- Deadlock handling
- Mixed read/write workloads
- Referential integrity constraints

## Usage

```bash
# Setup TPC-C schema
pgstorm --setup --workload tpcc --warehouses 10

# Run TPC-C benchmark
pgstorm --workload tpcc --duration 300s --workers 20 --connections 20

# Rebuild schema (drops and recreates all tables)
pgstorm --rebuild --workload tpcc --warehouses 10
```
