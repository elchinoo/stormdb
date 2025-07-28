# Troubleshooting Connection and Timeout Errors

## Common Error Types

### 1. Context Deadline Exceeded (Timeout Errors)
```
[3] timeout: context deadline exceeded
```

**Causes:**
- Operations taking longer than expected
- Database overload or slow queries
- Network latency issues
- Insufficient connection pool size

**Solutions:**
- ✅ **Reduce worker count**: Lower `workers` to reduce load
- ✅ **Increase connection timeout**: Enhanced with 30s operation timeout + retry logic
- ✅ **Optimize scale**: Reduce `scale` parameter for faster setup
- ✅ **Connection pool tuning**: Added min/max connection settings

### 2. Connection Closed Errors
```
[1] failed to deallocate cached statement(s): conn closed
```

**Causes:**
- Connection pool exhaustion
- Connections being closed unexpectedly
- Database restart or network interruption
- Prepared statement cleanup issues

**Solutions:**
- ✅ **Enhanced connection pool**: Added connection lifetime and health checks
- ✅ **Retry logic**: 3 retries with exponential backoff
- ✅ **Better resource cleanup**: Improved connection handling
- ✅ **Connection ratio**: Better workers-to-connections ratio

## Enhanced Configuration

### Recommended Settings for Stability

```yaml
# Optimized for reliability
database:
  type: postgres
  host: "localhost"
  port: 5432
  dbname: "test"
  username: "your_user"
  password: "your_password"
  sslmode: "disable"

workload: "vector_1024_cosine"
scale: 500          # Reduced from 1000
duration: "30s"
workers: 2          # Reduced from 4  
connections: 4      # 2:1 connection-to-worker ratio
```

### Progressive Load Testing

Start small and scale up:

```bash
# 1. Small scale test
./stormdb -c config_small.yaml    # scale: 100, workers: 1

# 2. Medium scale test  
./stormdb -c config_medium.yaml   # scale: 500, workers: 2

# 3. Full scale test
./stormdb -c config_large.yaml    # scale: 1000, workers: 4
```

## Database Optimization

### PostgreSQL Settings

Add these to `postgresql.conf`:

```conf
# Connection settings
max_connections = 100
shared_buffers = 256MB
effective_cache_size = 1GB

# Statement timeout (prevent runaway queries)
statement_timeout = '30s'
lock_timeout = '10s'

# Connection pooling
tcp_keepalives_idle = 300
tcp_keepalives_interval = 30
tcp_keepalives_count = 3
```

### pgvector Optimization

```sql
-- Ensure proper indexing
CREATE INDEX CONCURRENTLY ON items_1024 
USING ivfflat (embedding vector_cosine_ops) 
WITH (lists = 100);

-- Update statistics
ANALYZE items_1024;
```

## Monitoring & Debugging

### Real-time Monitoring

stormdb now includes:
- ✅ 5-second interval reporting
- ✅ Error type breakdown  
- ✅ Retry attempt tracking
- ✅ Latency histograms

### Check Database Status

```sql
-- Active connections
SELECT count(*) FROM pg_stat_activity;

-- Long-running queries  
SELECT pid, now() - pg_stat_activity.query_start AS duration, query 
FROM pg_stat_activity 
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes';

-- Lock conflicts
SELECT * FROM pg_locks WHERE NOT granted;
```

## Error Recovery

### Automatic Features Added

1. **Retry Logic**: Up to 3 retries with exponential backoff
2. **Operation Timeouts**: 30-second timeout per operation
3. **Connection Health**: Periodic health checks and connection cycling
4. **Graceful Degradation**: Continues testing even with some failures

### Manual Recovery

If errors persist:

```bash
# 1. Restart PostgreSQL
sudo systemctl restart postgresql

# 2. Clear connections
sudo -u postgres psql -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='test' AND pid <> pg_backend_pid()"

# 3. Restart with minimal load
./stormdb -c config_minimal.yaml
```

Your enhanced stormdb now handles these errors much more gracefully! 🚀
