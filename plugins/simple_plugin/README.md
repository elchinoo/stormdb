# Simple Workload Plugin

This plugin provides basic CRUD operation workloads for PostgreSQL performance testing with configurable read/write ratios.

## Overview

The simple workload plugin offers multiple operation modes for testing different aspects of database performance:

- **simple**: Mixed read/write operations with balanced load
- **read**: Read-only operations for testing query performance
- **write**: Write-only operations for testing insert/update performance
- **mixed**: Configurable mix of read/write operations

## Features

- Configurable table schema
- Variable record sizes
- Adjustable read/write ratios
- Random data generation
- Automatic table management
- Simple primary key-based operations

## Configuration

The simple workload supports the following configuration parameters:

```yaml
workload:
  type: "simple"           # or "read", "write", "mixed"
  table_name: "test_table" # Name of the test table
  record_count: 100000     # Number of records to maintain
  record_size: 1024        # Size of each record in bytes
  read_ratio: 0.7          # Ratio of read operations (0.0-1.0)
  write_ratio: 0.3         # Ratio of write operations (0.0-1.0)
  batch_size: 1            # Number of operations per transaction
```

## Schema

The simple workload creates a test table with:
- `id` (PRIMARY KEY): Auto-incrementing integer
- `data` (TEXT): Variable-length data field
- `created_at` (TIMESTAMP): Record creation timestamp
- `updated_at` (TIMESTAMP): Last update timestamp

## Operation Types

### Read Operations
- **SELECT**: Random record retrieval by primary key
- **SCAN**: Range scans over record ranges
- **COUNT**: Aggregate operations

### Write Operations
- **INSERT**: New record creation
- **UPDATE**: Modify existing records
- **DELETE**: Remove records (with automatic regeneration)

## Performance Characteristics

The simple workload is ideal for:
- Baseline performance testing
- Connection overhead measurement
- Basic CRUD operation benchmarking
- Scalability testing with varying read/write ratios

## Usage

```bash
# Setup simple schema
pgstorm --setup --workload simple --record-count 100000

# Run read-only test
pgstorm --workload read --duration 60s --workers 10 --connections 10

# Run write-only test
pgstorm --workload write --duration 60s --workers 5 --connections 5

# Run mixed workload
pgstorm --workload mixed --duration 300s --workers 20 --connections 20 --read-ratio 0.8

# Rebuild schema (drops and recreates table)
pgstorm --rebuild --workload simple --record-count 100000
```
