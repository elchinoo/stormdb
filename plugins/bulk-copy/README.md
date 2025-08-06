# StormDB Bulk Copy Plugin

## Overview

The Bulk Copy plugin is a high-performance PostgreSQL bulk data loading test plugin for StormDB v2. It uses PostgreSQL's native COPY protocol instead of INSERT statements to achieve maximum throughput when loading large amounts of data.

## Key Features

- **COPY Protocol**: Uses PostgreSQL's COPY FROM STDIN for optimal bulk loading performance
- **Multiple Formats**: Supports CSV, TEXT, and BINARY copy formats
- **Configurable Batch Sizes**: Test different batch sizes to find optimal performance
- **Real-time Metrics**: Delta-based metrics showing actual work done per second
- **Concurrent Workers**: Configurable number of concurrent connections
- **Flexible Schema**: Configurable number and types of data columns
- **Performance Tracking**: Comprehensive metrics including bytes/second transfer rates

## Performance Comparison

The bulk-copy plugin typically achieves significantly higher throughput compared to the bulk-insert plugin:

- **bulk-insert**: Uses INSERT statements, typical performance 10K-50K rows/second
- **bulk-copy**: Uses COPY protocol, typical performance 100K-1M+ rows/second

## Configuration

### Required Parameters

```yaml
host: "localhost"           # Database host
port: 5432                 # Database port
database: "test_db"        # Database name
username: "user"           # Database username
password: "password"       # Database password
```

### Test Configuration

```yaml
batch_sizes: [5000, 25000, 100000, 500000]  # Batch sizes to test
connections: 20                              # Number of concurrent connections
duration: "5m"                              # Duration per batch size
warmup_time: "30s"                          # Warmup period
think_time: "1ms"                           # Delay between batches
```

### COPY-Specific Configuration

```yaml
copy_format: "CSV"          # CSV, TEXT, or BINARY
copy_header: false          # Include header for CSV format
copy_delimiter: ","         # Delimiter for CSV format
```

### Table Configuration

```yaml
table_name: "bulk_copy_test_data"   # Table name
drop_table: true                    # Recreate table between tests
data_columns: 10                    # Number of data columns
index_columns: []                   # Columns to index (affects performance)
```

## COPY Protocol Formats

### CSV Format
- Most commonly used and portable
- Good balance of performance and compatibility
- Supports custom delimiters
- Human-readable format

### TEXT Format
- Tab-delimited format
- Slightly better performance than CSV
- Less flexible than CSV format

### BINARY Format
- Highest performance format
- Not human-readable
- Most efficient for large datasets
- Platform-dependent

## Performance Optimization Tips

### For Maximum Throughput
```yaml
batch_sizes: [100000, 500000, 1000000]
connections: 50
copy_format: "BINARY"
index_columns: []           # No indexes during load
think_time: "0s"           # No delay between batches
```

### For Balanced Performance
```yaml
batch_sizes: [10000, 50000, 200000]
connections: 25
copy_format: "CSV"
index_columns: ["primary_lookup_column"]
think_time: "5ms"
```

### For Memory-Constrained Environments
```yaml
batch_sizes: [5000, 20000, 50000]
connections: 10
copy_format: "CSV"
data_columns: 5            # Fewer columns
```

## Metrics Collected

### Real-time Metrics (every second)
- **interval_transaction_rate**: COPY operations per second
- **interval_row_rate**: Rows inserted per second
- **interval_avg_latency**: Average latency per operation

### Final Results (per batch size)
- **transactions_per_sec**: Average COPY operations per second
- **rows_per_sec**: Average rows inserted per second
- **bytes_per_sec**: Average data transfer rate
- **avg_latency_ms**: Average operation latency
- **error_rate**: Percentage of failed operations

## Database Schema

The plugin creates a test table with the following structure:

```sql
CREATE TABLE bulk_copy_test_data (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data_int_1 INTEGER,
    data_text_2 TEXT,
    data_float_3 FLOAT,
    data_bool_4 BOOLEAN,
    -- ... additional columns based on data_columns setting
);
```

## Usage Examples

### Basic Usage
```bash
# Build the plugin
cd plugins/bulk-copy
make build

# Run with default configuration
stormdb run --plugin bulk-copy --config config-example.yaml
```

### High-Performance Testing
```yaml
# config-high-perf.yaml
host: "localhost"
port: 5432
database: "perf_test"
username: "test_user"
password: "test_pass"
batch_sizes: [500000, 1000000]
connections: 50
duration: "10m"
copy_format: "BINARY"
index_columns: []
data_columns: 5
think_time: "0s"
```

### Comparison Testing
To compare COPY vs INSERT performance:

1. Run bulk-insert plugin with INSERT statements
2. Run bulk-copy plugin with same batch sizes
3. Compare throughput metrics

## Troubleshooting

### Low Performance Issues
1. **Check batch sizes**: Larger batches generally perform better with COPY
2. **Reduce indexes**: Indexes slow down bulk loading
3. **Use BINARY format**: For maximum performance
4. **Tune connections**: Too many connections can cause contention

### Memory Issues
1. **Reduce batch sizes**: Smaller batches use less memory
2. **Reduce connections**: Fewer concurrent operations
3. **Reduce data_columns**: Less data per row

### Connection Issues
1. **Check database limits**: `max_connections` setting
2. **Verify credentials**: Ensure user has COPY privileges
3. **Network connectivity**: Ensure stable connection

## Development

### Building
```bash
make build          # Build the plugin
make test           # Run tests
make check          # Run all checks (format, lint, test)
```

### Testing
```bash
make test           # Unit tests
make debug          # Debug build with race detection
```

## Dependencies

- **Go**: 1.21 or later
- **PostgreSQL**: 12 or later
- **Driver**: github.com/lib/pq

## Comparison with Other Plugins

| Feature | bulk-insert | bulk-copy | tpcc-scalability |
|---------|-----------|-----------|------------------|
| Protocol | INSERT | COPY | Mixed SQL |
| Throughput | Medium | High | Variable |
| Use Case | General | Bulk Loading | OLTP Workload |
| Complexity | Simple | Medium | Complex |

## License

Apache 2.0 - See main project license.
