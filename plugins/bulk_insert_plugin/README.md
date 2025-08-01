# Bulk Insert Workload

The bulk insert workload is designed to comprehensively test PostgreSQL's bulk data insertion performance across different methods, batch sizes, and concurrency patterns. It implements a sophisticated producer-consumer architecture with lock-free ring buffers to achieve maximum throughput while providing detailed performance insights.

## Overview

This workload addresses a critical performance testing gap by providing:

- **Method Comparison**: INSERT vs COPY performance analysis
- **Progressive Batch Sizing**: Testing batch sizes from 1 to 50,000 records per transaction
- **Producer-Consumer Pattern**: High-throughput data generation with configurable buffering
- **Comprehensive Schema**: Diverse data types to stress different PostgreSQL subsystems
- **Progressive Scaling**: Worker and connection scaling to find optimal configurations

## Architecture

### Producer-Consumer Pattern

The workload implements a lock-free ring buffer architecture:

```
┌─────────────┐    Ring Buffer    ┌─────────────┐
│  Producer   │ ─────────────────→ │  Consumer   │
│  Threads    │                   │  Threads    │
│             │ ←─────────────────  │             │
└─────────────┘   Back-pressure    └─────────────┘
```

- **Producers**: Generate realistic test data continuously
- **Ring Buffer**: Lock-free circular buffer with configurable capacity
- **Consumers**: Perform bulk INSERT or COPY operations
- **Back-pressure**: Automatic flow control when buffer is full

### Database Schema

The test table includes diverse data types to stress different aspects of PostgreSQL:

```sql
CREATE TABLE bulk_insert_test (
    id BIGSERIAL PRIMARY KEY,                    -- Sequential ID
    short_text VARCHAR(50),                      -- Short strings
    medium_text VARCHAR(500),                    -- Medium text
    long_text TEXT,                              -- Long text blobs
    int_value INTEGER,                           -- 32-bit integers
    bigint_value BIGINT,                         -- 64-bit integers
    decimal_value DECIMAL(15,4),                 -- Fixed-point numbers
    float_value DOUBLE PRECISION,                -- Floating-point numbers
    created_timestamp TIMESTAMPTZ DEFAULT NOW(), -- Temporal data
    event_date DATE,                             -- Date values
    event_time TIME,                             -- Time values
    is_active BOOLEAN DEFAULT TRUE,              -- Boolean flags
    metadata JSONB,                              -- Semi-structured data
    data_blob BYTEA,                             -- Binary data
    external_id UUID DEFAULT gen_random_uuid(),  -- UUID values
    status_enum bulk_status DEFAULT 'pending',   -- Enum values
    tags TEXT[],                                 -- Array data
    client_ip INET,                              -- Network addresses
    location POINT                               -- Geometric data
);
```

### Indexing Strategy

Multiple index types test different access patterns:
- B-tree indexes for range queries
- Composite indexes for multi-column operations
- Partial indexes for filtered queries
- GIN indexes for JSONB operations
- Hash indexes for equality lookups

## Configuration

### Basic Configuration

```yaml
workload: "bulk_insert"
scale: 1000

workload_config:
  ring_buffer_size: 50000
  producer_threads: 2
  batch_sizes: [1, 100, 1000, 10000, 50000]
  test_insert_method: true
  data_seed: 12345
  max_memory_mb: 256
  collect_metrics: true
```

### Progressive Scaling

The workload supports progressive scaling to test performance across different concurrency levels:

```yaml
progressive:
  enabled: true
  strategy: "linear"
  min_workers: 2
  max_workers: 10
  min_connections: 4
  max_connections: 20
  test_duration: "5m"
  warmup_duration: "30s"
  cooldown_duration: "15s"
  bands: 10
```

## Configuration Parameters

### Core Settings

| Parameter | Default | Description |
|-----------|---------|-------------|
| `ring_buffer_size` | 50000 | Size of the circular buffer (must be power of 2) |
| `producer_threads` | 2 | Number of data producer threads |
| `batch_sizes` | [1, 100, 1000, 10000, 50000] | Array of batch sizes to test |
| `test_insert_method` | true | Test both INSERT and COPY methods |
| `data_seed` | 0 | Seed for data generation (0 = random) |
| `max_memory_mb` | 256 | Maximum memory usage for data generation |
| `collect_metrics` | true | Enable detailed metrics collection |

### Data Generation

The workload generates realistic data patterns:

- **Text Data**: Variable-length strings with realistic content
- **Numeric Data**: Distributed integers and floating-point numbers
- **Temporal Data**: Dates and times within realistic ranges
- **JSON Data**: Nested structures with common metadata patterns
- **Binary Data**: Random binary blobs of varying sizes
- **Arrays**: String arrays with realistic tag patterns
- **Network Data**: Valid IPv4 and IPv6 addresses
- **Geometric Data**: Coordinate pairs for location data

## Usage Examples

### Quick Test

```bash
# Simple test with small batch sizes
./stormdb run -c config/bulk_insert_simple.yaml --setup
```

### Comprehensive Performance Test

```bash
# Full progressive scaling test
./stormdb run -c config/workload_bulk_insert.yaml --rebuild
```

### INSERT vs COPY Comparison

```bash
# Focus on method comparison
./stormdb run -c config/workload_bulk_insert.yaml \
  --override "workload_config.batch_sizes=[1000,10000]"
```

## Performance Insights

### Expected Results

1. **COPY Performance**: Should show superior throughput for large batch sizes
2. **INSERT Consistency**: May provide more consistent latency for small batches
3. **Memory Usage**: Should scale with batch size and buffer configuration
4. **Concurrency**: Optimal worker/connection ratios depend on hardware and PostgreSQL configuration

### Metrics Collected

- **Throughput**: Records/second and transactions/second
- **Latency**: Distribution across different percentiles
- **Memory Usage**: Buffer utilization and data generation overhead
- **PostgreSQL Stats**: Buffer cache, WAL generation, lock contention
- **System Resources**: CPU, memory, and I/O utilization

### Analysis Recommendations

1. **Batch Size Optimization**: Find the sweet spot for your workload
2. **Method Selection**: Choose INSERT vs COPY based on requirements
3. **Concurrency Tuning**: Optimize worker and connection counts
4. **Memory Management**: Balance buffer size with system memory
5. **Index Impact**: Measure index maintenance overhead

## Troubleshooting

### Common Issues

1. **Buffer Overflow**: Increase `ring_buffer_size` or reduce `producer_threads`
2. **Memory Pressure**: Reduce `max_memory_mb` or batch sizes
3. **Connection Limits**: Ensure PostgreSQL `max_connections` is adequate
4. **Slow Performance**: Check PostgreSQL configuration (shared_buffers, work_mem)

### Performance Tuning

1. **Producer Threads**: Usually 1-2 threads per CPU core is optimal
2. **Ring Buffer**: Size should be 10-100x the largest batch size
3. **Connections**: Typically 2x the number of workers is sufficient
4. **Batch Sizes**: Start with [1, 100, 1000] for initial testing

## Integration

The bulk insert workload integrates seamlessly with StormDB's features:

- **Progressive Scaling**: Automatic scaling across multiple configurations
- **Results Backend**: Persistent storage of all metrics and results
- **Plugin System**: Can be extended or customized via plugins
- **Configuration Management**: YAML-based configuration with validation
- **Monitoring**: Real-time metrics and PostgreSQL statistics collection

This workload provides essential insights for any application performing bulk data operations, from ETL pipelines to real-time analytics systems.
