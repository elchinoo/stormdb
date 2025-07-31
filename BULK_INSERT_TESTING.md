# Bulk Insert Workload - Testing Guide

## Overview

The bulk insert workload has been successfully implemented and integrated into StormDB v0.2. This document provides testing instructions and usage examples.

## Features Implemented

### Core Architecture
- ✅ **Producer-Consumer Pattern**: Lock-free ring buffer implementation
- ✅ **Progressive Batch Sizing**: Tests 1, 100, 1000, 10000, 50000 records per transaction
- ✅ **Method Comparison**: INSERT vs COPY performance analysis
- ✅ **Configurable Threading**: Adjustable producer and consumer thread counts
- ✅ **Memory Management**: Configurable buffer sizes and memory limits

### Database Schema
- ✅ **Comprehensive Data Types**: 18 different column types including:
  - Text data (various lengths)
  - Numeric data (integers, decimals, floats)
  - Temporal data (timestamps, dates, times)
  - Semi-structured data (JSONB)
  - Binary data (BYTEA)
  - Arrays (TEXT[])
  - Network types (INET)
  - Geometric types (POINT)
  - Enums and UUIDs

### Indexing Strategy
- ✅ **Multiple Index Types**: B-tree, GIN, Hash, Partial, Composite indexes
- ✅ **Performance Testing**: Index impact on bulk insert operations

### Integration
- ✅ **Progressive Scaling**: Integration with StormDB's progressive scaling system
- ✅ **Metrics Collection**: Comprehensive performance metrics
- ✅ **Configuration System**: YAML-based configuration with validation
- ✅ **Plugin System**: Registered as built-in workload

## Quick Testing

### Prerequisites
```bash
# Ensure PostgreSQL is running and accessible
# Default configuration expects:
# - Host: localhost:5432
# - Database: postgres
# - User: postgres
# - Password: password
```

### Simple Test (1 minute)
```bash
# Test with small batch sizes - quick validation
./build/stormdb --config config/bulk_insert_simple.yaml \
  --host localhost --dbname postgres --username postgres --password password \
  --duration 30s --setup
```

### Comprehensive Test (10 minutes)
```bash
# Full progressive test with multiple batch sizes and methods
./build/stormdb --config config/workload_bulk_insert.yaml \
  --host localhost --dbname stormdb_test --username postgres --password password \
  --rebuild
```

### Method Comparison Test
```bash
# Focus on INSERT vs COPY comparison
./build/stormdb --config config/workload_bulk_insert.yaml \
  --workers 2 --connections 4 --duration 2m --setup
```

## Configuration Examples

### Minimal Configuration
```yaml
database:
  type: postgres
  host: "localhost"
  port: 5432
  dbname: "test_db"
  username: "postgres"
  password: "password"
  sslmode: "disable"

workload: "bulk_insert"
workers: 2
connections: 4
duration: "1m"

workload_config:
  ring_buffer_size: 1000
  producer_threads: 1
  batch_sizes: [1, 100, 1000]
  test_insert_method: true
```

### Progressive Scaling Configuration
```yaml
workload: "bulk_insert"

progressive:
  enabled: true
  strategy: "linear"
  min_workers: 1
  max_workers: 4
  min_connections: 2
  max_connections: 8
  test_duration: "2m"
  bands: 6

workload_config:
  ring_buffer_size: 10000
  producer_threads: 2
  batch_sizes: [1, 100, 1000, 10000]
  test_insert_method: true
```

## Expected Performance Patterns

### INSERT Method
- **Small Batches (1-100)**: Consistent latency, moderate throughput
- **Medium Batches (1000-10000)**: Increasing throughput, higher latency variance
- **Large Batches (50000+)**: Maximum throughput, potential memory pressure

### COPY Method
- **Small Batches**: Higher per-operation overhead
- **Large Batches**: Superior throughput, efficient bulk processing
- **Memory Usage**: More efficient for large datasets

### Ring Buffer Utilization
- **Under-utilized (<50%)**: Increase producer threads or decrease batch sizes
- **Over-utilized (>90%)**: Increase buffer size or add consumer workers
- **Optimal Range**: 60-80% utilization for best performance

## Troubleshooting

### Common Issues

1. **Connection Refused**
   ```bash
   # Verify PostgreSQL is running
   sudo systemctl status postgresql
   # Or check if Docker container is running
   docker ps | grep postgres
   ```

2. **Permission Denied**
   ```bash
   # Ensure user has CREATE TABLE permissions
   GRANT CREATE ON DATABASE test_db TO postgres;
   ```

3. **Memory Pressure**
   ```yaml
   # Reduce memory usage in config
   workload_config:
     ring_buffer_size: 1000  # Reduce from default 50000
     max_memory_mb: 64       # Reduce from default 256
   ```

4. **High Latency**
   ```yaml
   # Optimize for lower latency
   workload_config:
     batch_sizes: [1, 10, 100]  # Focus on smaller batches
     producer_threads: 1        # Reduce contention
   ```

### Performance Tuning

1. **PostgreSQL Configuration**
   ```sql
   -- Optimize for bulk operations
   SET shared_buffers = '256MB';
   SET work_mem = '64MB';
   SET maintenance_work_mem = '256MB';
   SET wal_buffers = '16MB';
   SET checkpoint_completion_target = 0.9;
   ```

2. **System Resources**
   ```bash
   # Monitor during test
   htop                    # CPU and memory usage
   iostat -x 1             # I/O utilization
   pg_stat_activity        # PostgreSQL activity
   ```

## Development and Extension

### Adding New Data Types
1. Update `DataRecord` struct in `data_generator.go`
2. Add generation logic in `generateRecord()`
3. Update schema in `schema.go`
4. Modify insert methods in `generator.go`

### Custom Batch Size Patterns
```yaml
workload_config:
  batch_sizes: [1, 5, 25, 125, 625]  # Geometric progression
  # or
  batch_sizes: [1, 1, 2, 3, 5, 8, 13, 21]  # Fibonacci sequence
```

### Advanced Configuration
```yaml
workload_config:
  ring_buffer_size: 100000     # Large buffer for high throughput
  producer_threads: 4          # Multiple producers
  batch_sizes: [1, 10, 100, 1000, 10000, 50000, 100000]
  test_insert_method: true
  data_seed: 12345            # Reproducible data
  max_memory_mb: 512          # Higher memory limit
  collect_metrics: true       # Detailed metrics
```

## Integration with CI/CD

### Basic Performance Test
```bash
#!/bin/bash
# Add to CI pipeline for performance regression testing
./build/stormdb --config config/bulk_insert_simple.yaml \
  --duration 30s --setup \
  --host $DB_HOST --dbname $DB_NAME \
  --username $DB_USER --password $DB_PASS

# Parse results for performance thresholds
if [ $? -eq 0 ]; then
  echo "Bulk insert performance test passed"
else
  echo "Bulk insert performance test failed"
  exit 1
fi
```

## Version Information

- **StormDB Version**: v0.2-alpha.3+
- **Feature Branch**: `feature/bulk-insert-workload`
- **Workload Type**: `bulk_insert`
- **Status**: ✅ Ready for testing and usage

## Next Steps

1. **Performance Baseline**: Establish baseline performance metrics
2. **Advanced Features**: Consider adding compression testing, parallel COPY
3. **Cloud Integration**: Test with cloud PostgreSQL services
4. **Documentation**: Add to official StormDB documentation
5. **Examples**: Create more use-case specific examples

## Support

For issues, questions, or feature requests related to the bulk insert workload:

1. Check this testing guide first
2. Review the main README.md for general StormDB usage
3. Examine the configuration examples in `config/`
4. Check the detailed documentation in `internal/workload/bulk_insert/README.md`
