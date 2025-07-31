# Database Backend Integration Guide

This guide explains how to use StormDB's new database backend feature for storing test results and enabling comprehensive performance analytics.

## Overview

The database backend feature provides:
- **Historical test result storage** - Track performance trends over time
- **Comprehensive metrics storage** - Store detailed transaction, latency, and PostgreSQL statistics
- **Performance comparison** - Compare test runs across different configurations
- **Data retention management** - Automatic cleanup of old test results
- **Analytics foundation** - Enable data-driven performance analysis

## Quick Start

### 1. Enable Database Backend

Add the following section to your StormDB configuration YAML:

```yaml
# Example: config_with_backend.yaml
database:
  type: "postgres"
  host: "localhost"
  port: 5432
  dbname: "testdb"
  username: "postgres"
  password: "password"
  sslmode: "disable"

# Results backend configuration
results_backend:
  enabled: true
  host: "localhost"
  port: 5432
  database: "stormdb_results"
  username: "postgres"
  password: "password"
  sslmode: "disable"
  retention_days: 30           # Keep results for 30 days
  store_raw_metrics: true      # Store individual transaction metrics
  store_pg_stats: true         # Store PostgreSQL statistics
  metrics_batch_size: 1000     # Batch size for metric insertion
  table_prefix: "stormdb_"     # Prefix for results tables

# Test metadata for better organization
test_metadata:
  test_name: "performance_baseline_v1"
  environment: "production"
  database_target: "PostgreSQL 16 on AWS RDS"
  tags: ["baseline", "production", "aws"]
  notes: "Baseline performance test after optimization"

# Your normal workload configuration
workload: "tpcc"
scale: 10
workers: 50
connections: 100
duration: "10m"
```

### 2. Run StormDB with Backend

```bash
# Run with database backend enabled
./stormdb -c config_with_backend.yaml --setup

# The backend will automatically:
# - Create results tables if they don't exist
# - Store test run metadata
# - Store detailed performance metrics
# - Store PostgreSQL statistics (if enabled)
```

## Database Backend Schema

The backend creates the following tables:

### Core Tables

1. **`stormdb_test_runs`** - Test run metadata
   - `id`, `test_name`, `workload`, `configuration`
   - `start_time`, `end_time`, `duration`
   - `workers`, `connections`, `scale`
   - `status`, `version`, `environment`

2. **`stormdb_test_results`** - Aggregated test results
   - `test_run_id`, `tps`, `tps_aborted`
   - `total_transactions`, `total_errors`
   - `latency_avg_ms`, `latency_p95_ms`, `latency_p99_ms`
   - `success_rate`

### Detailed Metrics Tables

3. **`stormdb_postgresql_stats`** - PostgreSQL statistics
   - `test_run_id`, `buffer_cache_hit_ratio`
   - `blocks_read`, `blocks_hit`, `blocks_written`
   - `wal_records`, `wal_bytes`, `deadlocks`
   - `active_connections`, `temp_files`, `temp_bytes`

4. **`stormdb_error_metrics`** - Error tracking
   - `test_run_id`, `error_type`, `error_count`
   - `first_occurrence`, `last_occurrence`

5. **`stormdb_workload_metrics`** - Workload-specific metrics
   - `test_run_id`, `metric_name`, `metric_value`
   - `metric_type`, `recorded_at`

6. **`stormdb_latency_metrics`** - Individual latency samples
   - `test_run_id`, `latency_ns`, `recorded_at`
   - `operation_type`

## Configuration Options

### Required Settings

```yaml
results_backend:
  enabled: true                # Must be true to activate backend
  host: "localhost"           # PostgreSQL host for results storage
  port: 5432                  # PostgreSQL port
  database: "results_db"      # Database name for results
  username: "user"            # Database username
  password: "pass"            # Database password
```

### Optional Settings

```yaml
results_backend:
  sslmode: "disable"          # SSL mode (disable/require/prefer)
  retention_days: 30          # Days to keep results (0 = forever)
  store_raw_metrics: true     # Store individual transaction metrics
  store_pg_stats: true        # Store PostgreSQL statistics
  metrics_batch_size: 1000    # Batch size for metric insertion
  table_prefix: "stormdb_"    # Prefix for all results tables
```

### Test Metadata

```yaml
test_metadata:
  test_name: "custom_test_name"           # Override auto-generated name
  environment: "staging"                  # Environment identifier
  database_target: "PostgreSQL 15"       # Target database description
  tags: ["tag1", "tag2"]                 # Tags for organization
  notes: "Test description"              # Free-form notes
```

## Usage Examples

### Basic Performance Tracking

```yaml
# config_basic_tracking.yaml
results_backend:
  enabled: true
  host: "localhost"
  port: 5432
  database: "perf_tracking"
  username: "postgres"
  password: "password"
  retention_days: 90

test_metadata:
  test_name: "daily_performance_check"
  environment: "production"
  tags: ["daily", "monitoring"]
```

### Detailed Analytics Setup

```yaml
# config_detailed_analytics.yaml
results_backend:
  enabled: true
  host: "analytics-db.example.com"
  port: 5432
  database: "performance_analytics"
  username: "analytics_user"
  password: "secure_password"
  sslmode: "require"
  
  # Store everything for detailed analysis
  store_raw_metrics: true
  store_pg_stats: true
  metrics_batch_size: 5000
  retention_days: 365        # Keep one year of data

test_metadata:
  test_name: "comprehensive_benchmark"
  environment: "production"
  database_target: "PostgreSQL 16 with 32GB RAM"
  tags: ["comprehensive", "annual", "baseline"]
  notes: "Annual comprehensive performance benchmark"
```

### Development/Testing Setup

```yaml
# config_dev.yaml
results_backend:
  enabled: true
  host: "localhost"
  port: 5432
  database: "dev_results"
  username: "dev"
  password: "dev"
  retention_days: 7          # Short retention for development
  store_raw_metrics: false   # Reduce storage overhead
  store_pg_stats: true

test_metadata:
  environment: "development"
  tags: ["development", "testing"]
```

## Querying Results

### Basic Queries

```sql
-- Get recent test runs
SELECT test_name, workload, start_time, duration, 
       workers, connections, environment
FROM stormdb_test_runs 
ORDER BY start_time DESC 
LIMIT 10;

-- Get performance summary for a specific test
SELECT tr.test_name, tr.start_time,
       res.tps, res.latency_p95_ms, res.success_rate
FROM stormdb_test_runs tr
JOIN stormdb_test_results res ON tr.id = res.test_run_id
WHERE tr.test_name = 'performance_baseline_v1'
ORDER BY tr.start_time DESC;

-- Compare performance across environments
SELECT tr.environment, 
       AVG(res.tps) as avg_tps,
       AVG(res.latency_p95_ms) as avg_p95_latency
FROM stormdb_test_runs tr
JOIN stormdb_test_results res ON tr.id = res.test_run_id
WHERE tr.workload = 'tpcc' AND tr.scale = 10
GROUP BY tr.environment;
```

### Advanced Analytics

```sql
-- Performance trend over time
SELECT DATE(start_time) as test_date,
       AVG(res.tps) as daily_avg_tps,
       MAX(res.tps) as daily_max_tps,
       AVG(res.latency_p95_ms) as daily_avg_latency
FROM stormdb_test_runs tr
JOIN stormdb_test_results res ON tr.id = res.test_run_id
WHERE tr.test_name = 'daily_performance_check'
  AND tr.start_time >= NOW() - INTERVAL '30 days'
GROUP BY DATE(start_time)
ORDER BY test_date;

-- PostgreSQL statistics correlation
SELECT tr.workers, tr.connections,
       res.tps, 
       pg.buffer_cache_hit_ratio,
       pg.active_connections,
       pg.deadlocks
FROM stormdb_test_runs tr
JOIN stormdb_test_results res ON tr.id = res.test_run_id
JOIN stormdb_postgresql_stats pg ON tr.id = pg.test_run_id
WHERE tr.workload = 'tpcc'
ORDER BY res.tps DESC;
```

## Integration with Monitoring

### Grafana Dashboard Setup

1. **Add PostgreSQL data source** pointing to your results database
2. **Create panels** for key metrics:
   - TPS trends over time
   - Latency percentiles
   - Error rates
   - PostgreSQL statistics

### Example Grafana Query

```sql
-- TPS over time for Grafana
SELECT 
  start_time as time,
  tps
FROM stormdb_test_runs tr
JOIN stormdb_test_results res ON tr.id = res.test_run_id
WHERE $__timeFilter(start_time)
  AND tr.test_name = '$test_name'
ORDER BY start_time
```

## Maintenance

### Automatic Cleanup

The backend automatically performs maintenance:
- **Old result cleanup** based on `retention_days`
- **Table optimization** for better query performance

### Manual Maintenance

```sql
-- Check storage usage
SELECT 
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE tablename LIKE 'stormdb_%';

-- Manual cleanup (if needed)
DELETE FROM stormdb_test_runs 
WHERE start_time < NOW() - INTERVAL '90 days';
```

## Troubleshooting

### Common Issues

1. **Connection failures**
   ```
   Error: failed to create results backend: connection refused
   ```
   - Check database connectivity
   - Verify credentials and permissions
   - Ensure PostgreSQL is running

2. **Table creation errors**
   ```
   Error: permission denied for schema public
   ```
   - Grant CREATE privileges to the user
   - Consider using a dedicated schema

3. **Storage issues**
   ```
   Warning: Failed to store test results: disk full
   ```
   - Monitor disk space
   - Adjust retention settings
   - Consider disabling raw metrics storage

### Best Practices

1. **Use dedicated database** for results storage
2. **Monitor storage growth** especially with raw metrics
3. **Set appropriate retention** based on your needs
4. **Use tags and metadata** for better organization
5. **Regular backup** of results database
6. **Index optimization** for better query performance

## Performance Considerations

- **Raw metrics storage** can generate significant data volume
- **Batch size** affects insertion performance vs memory usage
- **Retention settings** directly impact storage requirements
- **Indexing** may be needed for large datasets

For optimal performance with large datasets, consider:
- Partitioning tables by date
- Creating appropriate indexes
- Using connection pooling
- Regular VACUUM and ANALYZE operations

## API Integration

The results backend provides programmatic access through the Go API:

```go
// Create backend from config
backend, err := results.CreateBackendFromConfig(cfg)

// Store test results
err = results.StoreTestResults(ctx, backend, cfg, metrics, startTime, endTime)

// Get recent test runs
testRuns, err := results.GetRecentTestRuns(ctx, backend, "tpcc", 10)

// Compare performance
comparison, err := results.CompareTestPerformance(ctx, backend, testRunID1, testRunID2)
```

This enables integration with custom monitoring and analytics tools.
