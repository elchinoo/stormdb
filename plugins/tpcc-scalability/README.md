# TPC-C Scalability Plugin for StormDB v2

A comprehensive TPC-C inspired performance testing plugin that performs incremental connection scaling tests to evaluate PostgreSQL database performance under varying load conditions.

## Overview

The TPC-C (Transaction Processing Performance Council Benchmark C) Scalability Plugin implements a simplified version of the TPC-C benchmark designed to test database performance with incrementally increasing connection counts. This plugin is perfect for understanding how your PostgreSQL database performs as the workload scales.

## Features

- **Incremental Connection Testing**: Tests performance at multiple connection levels (default: 48, 96, 192, 256)
- **TPC-C Transaction Mix**: Implements all five TPC-C transaction types with configurable percentages
- **Comprehensive Metrics**: Collects detailed performance metrics including latency, throughput, and error rates
- **Configurable Workload**: Adjustable scale factor, test duration, think time, and transaction mix
- **Automatic Schema Management**: Creates and populates TPC-C database schema automatically
- **Batch Result Storage**: Efficient storage of test results for analysis
- **Real-time Monitoring**: Live status updates during test execution

## Transaction Types

The plugin implements all five standard TPC-C transactions:

1. **New Order (45%)**: Creates new customer orders
2. **Payment (43%)**: Processes customer payments
3. **Order Status (4%)**: Queries order status
4. **Delivery (4%)**: Processes order deliveries
5. **Stock Level (4%)**: Checks inventory levels

## Installation

### Build the Plugin

```bash
cd plugins/tpcc-scalability
make plugin
```

### Verify Installation

```bash
ls -la ../../build/plugins/tpcc-scalability.so
```

## Configuration

### Basic Configuration

```yaml
host: "localhost"
port: 5432
database: "tpcc_test"
username: "postgres"
password: "postgres"
ssl_mode: "disable"
scale: 10
connections: [48, 96, 192, 256]
duration: "5m"
warmup_time: "30s"
think_time: "100ms"
new_order_pct: 45
payment_pct: 43
order_status_pct: 4
delivery_pct: 4
stock_level_pct: 4
batch_size: 100
enable_metrics: true
log_transactions: false
```

### Configuration Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `host` | string | `localhost` | Database host |
| `port` | int | `5432` | Database port |
| `database` | string | `tpcc_test` | Database name |
| `username` | string | `postgres` | Database user |
| `password` | string | `postgres` | Database password |
| `ssl_mode` | string | `disable` | SSL mode |
| `scale` | int | `10` | Number of warehouses (scale factor) |
| `connections` | []int | `[48,96,192,256]` | Connection levels to test |
| `duration` | string | `5m` | Duration per connection level |
| `warmup_time` | string | `30s` | Warmup time before measurements |
| `think_time` | string | `100ms` | Delay between transactions |
| `new_order_pct` | int | `45` | New Order transaction percentage |
| `payment_pct` | int | `43` | Payment transaction percentage |
| `order_status_pct` | int | `4` | Order Status transaction percentage |
| `delivery_pct` | int | `4` | Delivery transaction percentage |
| `stock_level_pct` | int | `4` | Stock Level transaction percentage |
| `batch_size` | int | `100` | Batch size for result storage |
| `enable_metrics` | bool | `true` | Enable detailed metrics collection |
| `log_transactions` | bool | `false` | Log individual transactions |

### Predefined Configurations

#### Quick Test (2 minutes)
```yaml
scale: 5
connections: [24, 48]
duration: "2m"
warmup_time: "15s"
```

#### Stress Test (10 minutes)
```yaml
scale: 50
connections: [128, 256, 512, 1000]
duration: "10m"
warmup_time: "1m"
think_time: "50ms"
```

#### Latency Test (15 minutes)
```yaml
scale: 20
connections: [48, 96, 144, 192]
duration: "15m"
warmup_time: "2m"
think_time: "200ms"
enable_metrics: true
log_transactions: true
```

## Usage

### Using StormDB v2 API

```bash
# Start a standard TPC-C test
curl -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "TPC-C Scalability Test",
    "description": "4-level connection scaling test",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "tpcc_test",
      "username": "postgres",
      "password": "postgres",
      "scale": 10,
      "connections": [48, 96, 192, 256],
      "duration": "5m"
    }
  }'
```

### Monitor Test Progress

```bash
# Get test status
curl "http://localhost:8080/test-runs/{test_run_id}"

# Get real-time results
curl "http://localhost:8080/test-runs/{test_run_id}/results"
```

## Test Execution Flow

1. **Initialization**: Plugin connects to database and validates configuration
2. **Schema Preparation**: Creates TPC-C tables if they don't exist
3. **Data Population**: Populates warehouses, districts, customers, items, and stock
4. **Connection Level Testing**: For each connection count:
   - **Warmup Phase**: Runs transactions without recording metrics
   - **Measurement Phase**: Records all transaction metrics
   - **Result Collection**: Stores results in batches
5. **Completion**: Aggregates results and updates test run status

## Database Schema

The plugin automatically creates the following TPC-C tables:

- `tpcc_warehouse`: Warehouse master data
- `tpcc_district`: District information per warehouse
- `tpcc_customer`: Customer records
- `tpcc_item`: Item catalog (shared across warehouses)
- `tpcc_stock`: Stock levels per warehouse/item

## Metrics Collected

### Transaction Metrics
- **Transaction Counts**: Per transaction type
- **Latency Statistics**: Min, max, average latency
- **Throughput**: Transactions per second (TPS)
- **Error Counts**: Failed transactions and timeouts

### Performance Metrics
- **Response Time**: Individual transaction response times
- **Concurrency**: Active connection utilization
- **Database Stats**: Cache hit ratios, lock waits

### Scalability Metrics
- **Throughput vs. Connections**: TPS at each connection level
- **Latency vs. Load**: Response time degradation
- **Resource Utilization**: CPU, memory, I/O patterns

## Test Results Analysis

### Connection Level Summary
```json
{
  "connections": 48,
  "duration": "5m0s",
  "total_transactions": 12450,
  "tps": 41.5,
  "avg_latency_ms": 12,
  "min_latency_ms": 2,
  "max_latency_ms": 85,
  "errors": 0,
  "transaction_breakdown": {
    "new_order": 5603,
    "payment": 5354,
    "order_status": 497,
    "delivery": 498,
    "stock_level": 498
  }
}
```

### Performance Trends
- **Linear Scaling**: TPS increases proportionally with connections
- **Saturation Point**: Connection count where TPS plateaus
- **Latency Degradation**: Point where response times increase significantly
- **Error Threshold**: Connection count where errors begin to occur

## Troubleshooting

### Common Issues

#### Database Connection Errors
```
Error: failed to connect to database: dial tcp: connection refused
```
**Solution**: Ensure PostgreSQL is running and accessible

#### Schema Creation Failures
```
Error: failed to create schema: permission denied
```
**Solution**: Ensure database user has CREATE privileges

#### High Memory Usage
```
Warning: memory usage exceeding limits
```
**Solution**: Reduce scale factor or batch size

#### Transaction Timeouts
```
Error: context deadline exceeded
```
**Solution**: Increase database connection limits or reduce concurrent connections

### Performance Tuning

#### Database Configuration
```sql
-- Increase connection limits
ALTER SYSTEM SET max_connections = 1000;

-- Optimize memory settings
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET work_mem = '4MB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';

-- Improve checkpoint behavior
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
```

#### Plugin Configuration
```yaml
# For high-throughput testing
scale: 20
connections: [100, 200, 400, 800]
think_time: "50ms"
batch_size: 200

# For latency-sensitive testing  
scale: 10
connections: [24, 48, 72, 96]
think_time: "200ms"
enable_metrics: true
log_transactions: true
```

## Testing Best Practices

### Test Environment
- Use dedicated test database
- Ensure sufficient hardware resources
- Monitor system metrics during tests
- Isolate from production workloads

### Test Design
- Start with small scale factors
- Gradually increase connection counts
- Allow adequate warmup time
- Run multiple iterations for consistency

### Result Interpretation
- Compare against baseline measurements
- Look for inflection points in performance curves
- Consider both throughput and latency metrics
- Document configuration and environment details

## Development

### Running Tests
```bash
# Run plugin unit tests
make test

# Run with coverage
make test-coverage

# Run benchmarks
make bench
```

### Code Structure
```
plugin.go          # Main plugin implementation
plugin_test.go     # Unit tests
config-example.yaml # Configuration examples
API_EXAMPLES.md    # API usage examples
Makefile          # Build automation
go.mod            # Go module definition
```

### Contributing
1. Follow Go coding standards
2. Add tests for new functionality
3. Update documentation
4. Ensure backward compatibility

## License

This plugin is part of StormDB v2 and is licensed under the MIT License.

## Support

For issues, questions, or contributions:
- GitHub Issues: [StormDB Repository](https://github.com/elchinoo/stormdb)
- Documentation: [StormDB v2 Docs](https://github.com/elchinoo/stormdb/tree/v2-redesign-core)
- API Examples: [API_EXAMPLES.md](./API_EXAMPLES.md)
