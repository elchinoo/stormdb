# Performance Optimization Guide

This guide provides comprehensive guidance on optimizing StormDB performance for various testing scenarios.

## Overview

Performance optimization in StormDB involves tuning multiple layers:
- Database connection and pooling
- Workload configuration
- System resources
- PostgreSQL settings
- Monitoring and metrics collection

## Connection Pool Optimization

### Connection Pool Sizing

```yaml
connection_pool:
  # Start with reasonable defaults
  max_connections: 50              # Don't exceed PostgreSQL max_connections
  initial_connections: 10          # Pre-warm the pool
  min_connections: 5               # Maintain minimum connections
  
  # Tune based on workload
  acquire_timeout: "30s"           # Time to wait for connection
  idle_timeout: "10m"              # Close idle connections
  connection_lifetime: "1h"        # Maximum connection age
```

### Pool Sizing Guidelines

**High Concurrency Workloads:**
```yaml
connection_pool:
  max_connections: 100
  initial_connections: 25
  min_connections: 10
```

**Low Latency Workloads:**
```yaml
connection_pool:
  max_connections: 20
  initial_connections: 15
  min_connections: 10
  acquire_timeout: "5s"
```

**Long-Running Tests:**
```yaml
connection_pool:
  max_connections: 50
  connection_lifetime: "4h"
  idle_timeout: "30m"
  health_check_interval: "60s"
```

### Connection Pool Monitoring

```yaml
metrics:
  connection_metrics: true
  
# Monitor these key metrics:
# - pool_active_connections
# - pool_idle_connections
# - pool_wait_duration
# - connection_acquisition_failures
```

## Workload Optimization

### Operation Distribution

Optimize operation weights based on your testing goals:

```yaml
workload:
  operations:
    # Read-heavy workload (typical web application)
    - type: "select"
      weight: 80
    - type: "insert"
      weight: 15
    - type: "update"
      weight: 5
      
    # Write-heavy workload (data ingestion)
    - type: "insert"
      weight: 70
    - type: "update"
      weight: 20
    - type: "select"
      weight: 10
      
    # Balanced workload (OLTP)
    - type: "select"
      weight: 50
    - type: "insert"
      weight: 25
    - type: "update"
      weight: 20
    - type: "delete"
      weight: 5
```

### Batch Operations

For high-throughput scenarios:

```yaml
workload:
  plugin_config:
    batch_size: 1000              # Insert/update in batches
    batch_timeout: "100ms"        # Maximum wait for batch
    use_copy: true                # Use PostgreSQL COPY for inserts
    prepared_statements: true     # Cache prepared statements
```

### Transaction Optimization

```yaml
workload:
  operations:
    - type: "transaction"
      weight: 100
      transaction_size: 5         # Operations per transaction
      isolation_level: "read_committed"
      read_only: false
```

## System Resource Optimization

### Memory Configuration

**StormDB Settings:**
```yaml
metrics:
  buffer_size: 10000             # Metric buffer size
  max_memory: "2GB"              # Memory limit for StormDB
  
connection_pool:
  statement_cache_size: 1000     # Prepared statement cache
```

**System Settings:**
```bash
# Increase memory limits
ulimit -m unlimited
ulimit -v unlimited

# Set memory overcommit (Linux)
echo 1 > /proc/sys/vm/overcommit_memory
```

### CPU Optimization

```yaml
# Distribute load across CPU cores
workload:
  concurrent_users: 8           # Match CPU core count
  worker_threads: 8             # One thread per core
  
# Enable CPU affinity (Linux)
system:
  cpu_affinity: [0,1,2,3]      # Pin to specific cores
```

### I/O Optimization

```bash
# Use fast storage for temporary files
export TMPDIR=/fast/storage/tmp

# Optimize disk scheduler (Linux)
echo deadline > /sys/block/sda/queue/scheduler

# Increase I/O limits
ulimit -n 65536               # File descriptors
```

## PostgreSQL Optimization

### Connection Settings

```sql
-- postgresql.conf optimizations for testing

-- Connection settings
max_connections = 200           -- Allow sufficient connections
superuser_reserved_connections = 3

-- Memory settings
shared_buffers = '2GB'         -- 25% of system RAM
effective_cache_size = '6GB'   -- 75% of system RAM
work_mem = '16MB'              -- Per-connection work memory
maintenance_work_mem = '512MB' -- Maintenance operations

-- WAL settings
wal_buffers = '64MB'
checkpoint_completion_target = 0.9
max_wal_size = '4GB'
min_wal_size = '1GB'

-- Query planner
random_page_cost = 1.1         -- For SSD storage
effective_io_concurrency = 200 -- For SSD storage

-- Logging and monitoring
log_statement = 'none'         -- Reduce logging overhead
log_min_duration_statement = 1000  -- Log slow queries only
track_activities = on
track_counts = on
track_io_timing = on
```

### Index Optimization

Create appropriate indexes for your workload:

```sql
-- Query-specific indexes
CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
CREATE INDEX CONCURRENTLY idx_orders_user_date ON orders(user_id, created_at);

-- Partial indexes for common filters
CREATE INDEX CONCURRENTLY idx_orders_active 
ON orders(user_id) WHERE status = 'active';

-- Composite indexes for complex queries
CREATE INDEX CONCURRENTLY idx_products_category_price 
ON products(category_id, price) WHERE active = true;
```

### Table Optimization

```sql
-- Optimize table storage
ALTER TABLE large_table SET (fillfactor = 90);

-- Partition large tables
CREATE TABLE orders_2024 PARTITION OF orders
FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');

-- Use appropriate data types
ALTER TABLE measurements 
ALTER COLUMN value TYPE numeric(10,2);  -- Instead of text
```

## Monitoring Optimization

### Metrics Collection Tuning

```yaml
metrics:
  # Balance detail vs. performance
  collection_interval: "5s"      # Reduce frequency for less overhead
  pg_stats_interval: "10s"       # PostgreSQL stats less frequently
  system_interval: "5s"          # System metrics
  
  # Selective monitoring
  pg_stats_enabled: true
  system_metrics: false          # Disable if not needed
  detailed_logging: false        # Reduce I/O overhead
  
  # Optimize metric storage
  buffer_size: 50000             # Large buffer to reduce writes
  compression: true              # Compress metric data
```

### Dashboard Optimization

```yaml
metrics:
  real_time_dashboard: true
  dashboard_port: 8080
  update_interval: "2s"          # Balance responsiveness vs. load
  
  # Limit displayed metrics
  dashboard_metrics:
    - "transactions_per_second"
    - "average_response_time"
    - "error_rate"
    - "active_connections"
```

## Progressive Scaling Optimization

### Scaling Strategy

```yaml
workload:
  progressive_scaling:
    enabled: true
    
    # Optimize scaling parameters
    initial_users: 1
    max_users: 100
    step_size: 10                # Larger steps for faster scaling
    step_duration: "3m"          # Sufficient time for stabilization
    
    # Statistical accuracy
    stability_threshold: 0.1     # 10% coefficient of variation
    minimum_samples: 50          # More samples for accuracy
    confidence_level: 0.95       # 95% confidence intervals
    
    # Performance optimization
    outlier_removal: true        # Remove statistical outliers
    smoothing_enabled: true      # Reduce noise in measurements
```

### Analysis Optimization

```yaml
workload:
  progressive_scaling:
    # Faster analysis
    analysis_method: "online"    # Real-time vs. batch analysis
    sample_size: 1000           # Limit sample size for speed
    
    # Memory optimization
    data_retention: "1h"        # Limit historical data
    compression: true           # Compress stored measurements
```

## Network Optimization

### Connection Settings

```yaml
database:
  # TCP optimization
  connect_timeout: "10s"
  statement_timeout: "30s"
  
  # Keep-alive settings
  tcp_keepalive: true
  tcp_keepalive_idle: 600      # 10 minutes
  tcp_keepalive_interval: 60   # 1 minute
  tcp_keepalive_count: 3
```

### SSL/TLS Optimization

```yaml
database:
  # SSL settings for performance
  ssl_mode: "require"          # Use SSL but don't verify
  ssl_compression: false       # Disable SSL compression
  
  # For testing only (not production)
  ssl_mode: "disable"          # Best performance, no security
```

## Plugin-Specific Optimizations

### IMDB Plugin

```yaml
workload:
  plugin_config:
    # Data loading optimization
    batch_size: 5000
    concurrent_loaders: 4
    use_copy: true             # Use COPY for bulk loading
    
    # Query optimization
    prepared_statements: true
    connection_pooling: true
    
    # Memory optimization
    cache_size: 10000          # Cache frequently accessed data
```

### Vector Plugin

```yaml
workload:
  plugin_config:
    # Vector operations optimization
    vector_dimension: 768      # Optimize for your use case
    batch_size: 100           # Batch vector operations
    
    # Index optimization
    index_type: "hnsw"        # Faster than ivfflat for queries
    index_params:
      m: 16                   # HNSW parameter
      ef_construction: 64     # Build-time parameter
      ef_search: 32           # Query-time parameter
```

### E-commerce Plugin

```yaml
workload:
  plugin_config:
    # Data size optimization
    catalog_size: 10000       # Optimize for memory usage
    customer_count: 1000
    
    # Operation optimization
    cart_session_timeout: "30m"
    inventory_check_batch: 100
```

## Performance Testing Best Practices

### Test Environment

1. **Dedicated Hardware**: Use dedicated test hardware
2. **Consistent Environment**: Ensure consistent test conditions
3. **Baseline Testing**: Establish baseline performance
4. **Isolation**: Isolate test traffic from production

### Test Design

```yaml
# Gradual ramp-up for accurate measurements
workload:
  ramp_up_time: "5m"          # Allow system to warm up
  duration: "30m"             # Sufficient test duration
  ramp_down_time: "2m"        # Graceful shutdown
  
  # Realistic concurrency patterns
  concurrent_users: 50        # Match expected load
  think_time: "100ms"         # Realistic user behavior
```

### Measurement Accuracy

```yaml
metrics:
  # High-frequency sampling
  collection_interval: "1s"
  
  # Statistical accuracy
  confidence_level: 0.95
  minimum_samples: 100
  outlier_removal: true
  
  # Comprehensive metrics
  pg_stats_enabled: true
  system_metrics: true
  transaction_metrics: true
```

## Performance Tuning Checklist

### Database Configuration
- [ ] Connection pool sized appropriately
- [ ] PostgreSQL configuration optimized
- [ ] Appropriate indexes created
- [ ] Table storage optimized
- [ ] Query plans analyzed

### System Resources
- [ ] Sufficient memory allocated
- [ ] CPU resources available
- [ ] Fast storage configured
- [ ] Network latency minimized
- [ ] System limits increased

### StormDB Configuration
- [ ] Workload operations optimized
- [ ] Metrics collection tuned
- [ ] Output format efficient
- [ ] Plugin configuration optimized
- [ ] Connection settings tuned

### Monitoring
- [ ] Key metrics identified
- [ ] Dashboard configured
- [ ] Alerting set up
- [ ] Performance baselines established
- [ ] Trend analysis enabled

## Performance Troubleshooting

### Common Performance Issues

**High Response Times:**
```bash
# Check PostgreSQL slow queries
SELECT query, mean_time, calls 
FROM pg_stat_statements 
ORDER BY mean_time DESC LIMIT 10;

# Check connection pool utilization
stormdb --config config.yaml --format json | jq '.metrics.pool_*'

# Check system resources
top -p $(pgrep stormdb)
```

**Low Throughput:**
```bash
# Check connection limits
SELECT setting FROM pg_settings WHERE name = 'max_connections';

# Check current connections
SELECT count(*) FROM pg_stat_activity;

# Check lock contention
SELECT * FROM pg_locks WHERE NOT granted;
```

**Memory Issues:**
```bash
# Check memory usage
ps aux | grep stormdb
cat /proc/$(pgrep stormdb)/status | grep Vm

# Check PostgreSQL memory
SELECT * FROM pg_stat_database WHERE datname = 'your_db';
```

### Performance Analysis

```bash
# Generate performance report
stormdb --config config.yaml \
        --format json \
        --output perf_report.json \
        --metrics \
        --pg-stats

# Analyze results
jq '.summary.performance' perf_report.json
jq '.metrics.response_times' perf_report.json
jq '.postgresql_stats' perf_report.json
```

## Next Steps

- [Configuration Guide](CONFIGURATION.md) - Optimize configuration settings
- [Usage Guide](USAGE.md) - Performance-related command options
- [Troubleshooting](TROUBLESHOOTING.md) - Diagnose performance issues
- [Plugin System](PLUGIN_SYSTEM.md) - Optimize custom workloads
