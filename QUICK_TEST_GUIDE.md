# Quick Testing Guide for StormDB v2

## Testing Your Application

You have several options to test StormDB v2 with the bulk load and TPC-C plugins:

## Option 1: Automated Demo (Recommended)

The easiest way to test everything:

```bash
# Full automated demo (includes PostgreSQL setup)
./demo_test.sh

# If you have PostgreSQL running already
./demo_test.sh --skip-postgres

# Just build and start (no demo tests)
./demo_test.sh --skip-tests
```

## Option 2: Manual Step-by-Step Testing

### 1. Prerequisites
```bash
# Install required tools
brew install jq postgresql  # macOS
# or
sudo apt-get install jq postgresql-client  # Ubuntu
```

### 2. Start PostgreSQL
```bash
# Using Docker (easiest)
docker run --name postgres-stormdb \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=stormdb \
  -p 5432:5432 \
  -d postgres:15

# Wait for it to start
docker exec postgres-stormdb pg_isready -U postgres
```

### 3. Build StormDB
```bash
make build-all
```

### 4. Start StormDB
```bash
STORMDB_PLUGIN_DIR=./build/plugins ./build/stormdb
```

### 5. Test Basic Functionality
```bash
# Health check
curl http://localhost:8080/health

# List plugins
curl http://localhost:8080/plugins | jq
```

## Option 3: Manual API Testing

### Test Bulk Load Plugin

1. **Create a bulk load test:**
```bash
curl -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "bulk-load",
    "name": "My Bulk Load Test",
    "description": "Testing different batch sizes",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "stormdb",
      "username": "postgres",
      "password": "postgres",
      "ssl_mode": "disable",
      "batch_sizes": [1, 1000, 10000],
      "connections": 20,
      "duration": "60s",
      "warmup_time": "10s"
    }
  }'
```

2. **Monitor test progress:**
```bash
# Replace TEST_ID with the ID from the response above
curl http://localhost:8080/test-runs/TEST_ID
```

3. **Get results:**
```bash
curl http://localhost:8080/test-runs/TEST_ID/results | jq
```

### Test TPC-C Plugin

1. **Create a TPC-C test:**
```bash
curl -X POST "http://localhost:8080/test-runs" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "tpcc-scalability",
    "name": "My TPC-C Test",
    "description": "Testing connection scaling",
    "config": {
      "host": "localhost",
      "port": 5432,
      "database": "stormdb",
      "username": "postgres",
      "password": "postgres",
      "ssl_mode": "disable",
      "scale": 5,
      "connections": [5, 10, 20, 50],
      "duration": "120s",
      "warmup_time": "15s"
    }
  }'
```

2. **Monitor and get results** (same as bulk load)

## Understanding the Results

### Bulk Load Results
- **batch_size**: Number of rows per transaction
- **transactions_per_sec**: Transactions completed per second
- **rows_per_sec**: Rows inserted per second
- **avg_latency_ms**: Average latency per transaction

### TPC-C Results
- **connections**: Number of concurrent connections
- **tps**: Transactions per second
- **avg_latency_ms**: Average transaction latency
- **transaction_types**: Breakdown by transaction type

## Configuration Options

### Bulk Load Plugin
- `batch_sizes`: Array of batch sizes to test (e.g., [1, 1000, 10000, 50000])
- `connections`: Fixed number of connections (default: 20)
- `duration`: How long to run each test
- `warmup_time`: Time to warm up before measuring
- `table_name`: Custom table name (optional)
- `data_columns`: Number of data columns to create

### TPC-C Plugin
- `scale`: TPC-C scale factor (number of warehouses)
- `connections`: Array of connection counts to test
- `duration`: How long to run each test
- `warmup_time`: Time to warm up before measuring

## Troubleshooting

### Common Issues

1. **"Plugin not found"**
   - Make sure you built with `make build-all`
   - Check that `build/plugins/` contains `.so` files
   - Ensure `STORMDB_PLUGIN_DIR` environment variable is set

2. **"Connection refused"**
   - Make sure PostgreSQL is running
   - Check the database configuration in your test

3. **"Permission denied"**
   - Make sure the PostgreSQL user has proper permissions
   - Check that the database exists

### Logs
- StormDB logs: Check console output or redirect to file
- Plugin logs: Plugins log through StormDB's logging system
- PostgreSQL logs: Check Docker logs with `docker logs postgres-stormdb`

## Performance Tips

1. **For accurate results:**
   - Use dedicated hardware
   - Ensure PostgreSQL is properly tuned
   - Run tests multiple times and average results

2. **For larger tests:**
   - Increase duration to 2-5 minutes
   - Use higher scale factors for TPC-C
   - Test with more connections (up to your system limits)

3. **For production-like testing:**
   - Use realistic data sizes
   - Test with network latency
   - Consider using connection pooling

## Next Steps

After testing:
1. Analyze which batch sizes perform best for your workload
2. Compare TPC-C results across different connection counts
3. Use the results to optimize your database configuration
4. Consider creating custom plugins for your specific use cases

## Getting Help

- Check the full `TESTING_GUIDE.md` for more detailed instructions
- Review plugin source code in `plugins/` directory
- Check StormDB logs for detailed error messages
