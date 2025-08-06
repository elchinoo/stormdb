# Bulk Load Performance Test Plugin

A high-performance bulk load testing plugin for StormDB v2 that evaluates database performance across different batch sizes with a fixed number of connections.

## Overview

The Bulk Load plugin is designed to help database administrators and performance engineers understand how different batch sizes affect bulk insert performance. Unlike connection scaling tests, this plugin focuses on finding the optimal batch size for bulk operations while maintaining a consistent connection count.

## Key Features

- **Batch Size Testing**: Tests multiple batch sizes (default: 1, 1000, 10000, 50000 rows)
- **Fixed Connection Model**: Uses a consistent number of connections across all tests
- **Comprehensive Metrics**: Tracks transactions per second, rows per second, latency statistics, and error rates
- **Flexible Schema**: Configurable table structure with multiple data types
- **Index Testing**: Optional index creation for realistic performance scenarios
- **Warmup Phase**: Configurable warmup period for accurate measurements
- **Real-time Monitoring**: Live progress tracking via REST API

## Installation

### Prerequisites

- Go 1.21 or later
- PostgreSQL database
- StormDB v2 core system

### Build the Plugin

```bash
cd plugins/bulk-load
make plugin
```

This creates `bulk-load.so` which can be loaded by StormDB v2.

### Install to StormDB

```bash
make install PLUGIN_DIR=/path/to/stormdb/plugins
```

## Configuration

### Basic Configuration

```yaml
host: "localhost"
port: 5432
database: "stormdb_test"
username: "stormdb"
password: "stormdb_password"
```

### Complete Configuration Options

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `host` | string | required | Database host |
| `port` | integer | required | Database port |
| `database` | string | required | Database name |
| `username` | string | required | Database username |
| `password` | string | required | Database password |
| `ssl_mode` | string | `"disable"` | SSL mode (disable, require, verify-ca, verify-full) |
| `batch_sizes` | array | `[1, 1000, 10000, 50000]` | Array of batch sizes to test |
| `connections` | integer | `20` | Fixed number of connections |
| `duration` | string | `"5m"` | Duration per batch size test |
| `warmup_time` | string | `"30s"` | Warmup period before measurements |
| `think_time` | string | `"10ms"` | Delay between batch operations |
| `table_name` | string | `"bulk_test_data"` | Test table name |
| `drop_table` | boolean | `true` | Drop/recreate table between tests |
| `generate_data` | boolean | `true` | Generate random test data |
| `data_columns` | integer | `10` | Number of data columns to create |
| `index_columns` | array | `[]` | Columns to create indexes on |
| `verbose` | boolean | `false` | Enable verbose logging |

## Usage Examples

### Quick Test
```yaml
batch_sizes: [1, 100, 1000]
connections: 10
duration: "2m"
data_columns: 5
```

### Performance Comparison
```yaml
batch_sizes: [1, 10, 100, 500, 1000, 5000, 10000, 25000, 50000]
connections: 20
duration: "3m"
data_columns: 8
```

### Large Scale Test
```yaml
batch_sizes: [1000, 10000, 50000, 100000]
connections: 50
duration: "10m"
data_columns: 20
index_columns: ["data_int_4", "data_text_1", "data_float_6"]
```

## API Usage

### Start a Test

```bash
curl -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "bulk-load",
    "name": "Bulk Load Performance Test",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "stormdb_test",
      "username": "stormdb",
      "password": "stormdb_password",
      "batch_sizes": [1, 1000, 10000, 50000],
      "connections": 20
    }
  }'
```

### Monitor Progress

```bash
curl "http://localhost:8080/test-runs/{run_id}"
```

### Get Results

```bash
curl "http://localhost:8080/test-runs/{run_id}/results"
```

## Performance Metrics

The plugin collects comprehensive performance metrics for each batch size:

### Transaction Metrics
- **Total Transactions**: Number of successful bulk insert operations
- **Transactions Per Second (TPS)**: Throughput of bulk operations
- **Total Rows Inserted**: Aggregate row count across all transactions
- **Rows Per Second**: Row insertion rate

### Latency Metrics
- **Average Latency**: Mean response time for bulk operations
- **Minimum Latency**: Best-case response time
- **Maximum Latency**: Worst-case response time

### Error Metrics
- **Total Errors**: Count of failed operations
- **Error Rate**: Percentage of failed operations

### Example Results

```json
{
  "batch_results": [
    {
      "batch_size": 1,
      "connections": 20,
      "total_transactions": 18750,
      "total_rows_inserted": 18750,
      "transactions_per_sec": 62.5,
      "rows_per_sec": 62.5,
      "avg_latency_ms": 15.2,
      "error_rate": 0.0
    },
    {
      "batch_size": 1000,
      "connections": 20,
      "total_transactions": 750,
      "total_rows_inserted": 750000,
      "transactions_per_sec": 2.5,
      "rows_per_sec": 2500.0,
      "avg_latency_ms": 12.8,
      "error_rate": 0.0
    }
  ]
}
```

## Database Schema

The plugin creates a configurable test table with the following structure:

```sql
CREATE TABLE bulk_test_data (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data_int_1 INTEGER,
    data_text_2 TEXT,
    data_float_3 FLOAT,
    data_bool_4 BOOLEAN,
    -- ... additional columns based on data_columns setting
);
```

### Data Types

The plugin generates realistic test data:
- **Integers**: Random values up to 1,000,000
- **Text**: Formatted strings with test identifiers
- **Floats**: Random decimal values up to 1,000.0
- **Booleans**: Random true/false values

## Performance Tuning

### Optimal Batch Sizes

Results typically show:
- **Single row inserts (1)**: High latency, low throughput
- **Small batches (100-1000)**: Balanced performance
- **Medium batches (5000-10000)**: Often optimal for many workloads
- **Large batches (50000+)**: May hit memory or lock limits

### Database Configuration

For optimal results, consider:

```sql
-- Increase work memory for large batches
SET work_mem = '256MB';

-- Adjust checkpoint settings
SET checkpoint_segments = 64;
SET checkpoint_completion_target = 0.9;

-- Optimize logging
SET synchronous_commit = off;  -- Only for testing!
```

### Connection Tuning

Choose connection count based on:
- **CPU cores**: Typically 2-4x core count
- **Memory availability**: Each connection uses memory
- **Database limits**: Check `max_connections`

## Testing Best Practices

### Environment Setup

1. **Dedicated test database**: Avoid production data
2. **Sufficient disk space**: Large batches generate significant data
3. **Stable network**: Minimize network-related variance
4. **Consistent hardware**: Use same system for comparative tests

### Test Configuration

1. **Warmup period**: Allow system to reach steady state
2. **Sufficient duration**: At least 2-5 minutes per batch size
3. **Multiple runs**: Average results across multiple test runs
4. **Baseline testing**: Test with minimal load first

### Result Analysis

1. **Look for patterns**: How does performance scale with batch size?
2. **Identify bottlenecks**: Where does performance plateau?
3. **Consider memory usage**: Monitor system resources
4. **Evaluate trade-offs**: Balance throughput vs. latency requirements

## Troubleshooting

### Common Issues

#### Connection Errors
```
Error: failed to connect to database
```
- Verify database credentials
- Check network connectivity
- Ensure PostgreSQL is running
- Validate SSL mode settings

#### Permission Errors
```
Error: permission denied for table creation
```
- Grant CREATE permissions to test user
- Ensure user can create/drop tables
- Check schema permissions

#### Memory Issues
```
Error: out of memory
```
- Reduce batch sizes
- Decrease connection count
- Increase database work_mem
- Monitor system memory usage

#### Performance Issues
```
Error: test timeout
```
- Increase test duration
- Reduce batch sizes
- Check database performance
- Monitor resource utilization

### Debugging

Enable verbose logging:
```yaml
verbose: true
```

Check StormDB logs:
```bash
tail -f /var/log/stormdb/stormdb.log
```

Monitor database activity:
```sql
SELECT * FROM pg_stat_activity WHERE application_name LIKE 'stormdb%';
```

## Development

### Running Tests

```bash
make test
```

### Code Coverage

```bash
make coverage
```

### Benchmarks

```bash
make benchmark
```

### Code Quality

```bash
make check  # Runs format, vet, lint, and test
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the full test suite
6. Submit a pull request

## License

Apache License 2.0 - see LICENSE file for details.

## Support

For issues and questions:
- GitHub Issues: https://github.com/elchinoo/stormdb/issues
- Documentation: https://github.com/elchinoo/stormdb/wiki
- Community: https://github.com/elchinoo/stormdb/discussions
