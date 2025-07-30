# Troubleshooting Guide

This guide helps diagnose and resolve common issues with StormDB.

## Common Issues and Solutions

### Installation Issues

#### Permission Denied Errors

**Problem:**
```bash
bash: ./stormdb: Permission denied
```

**Solution:**
```bash
# Fix executable permissions
chmod +x stormdb

# Or download with execute permissions
curl -L https://github.com/elchinoo/stormdb/releases/latest/download/stormdb-linux-amd64 -o stormdb
chmod +x stormdb
```

#### Plugin Loading Errors

**Problem:**
```
Error: plugin "imdb_plugin" not found
```

**Solution:**
```bash
# Check plugin directory
ls -la plugins/

# Verify plugin path
echo $STORMDB_PLUGIN_PATH

# Set plugin path
export STORMDB_PLUGIN_PATH="./plugins"

# Or copy to default location
mkdir -p ~/.stormdb/plugins
cp plugins/*.so ~/.stormdb/plugins/
```

#### Shared Library Errors (Linux/macOS)

**Problem:**
```
Error loading plugin: cannot open shared object file
```

**Solution:**
```bash
# Check dependencies
ldd plugins/imdb_plugin.so  # Linux
otool -L plugins/imdb_plugin.so  # macOS

# Install missing dependencies (Ubuntu/Debian)
sudo apt-get install libc6-dev

# Verify architecture match
file stormdb
file plugins/imdb_plugin.so
```

### Configuration Issues

#### Invalid Configuration Format

**Problem:**
```
Error: yaml: line 5: mapping values are not allowed in this context
```

**Solution:**
```bash
# Validate YAML syntax
python -c "import yaml; yaml.safe_load(open('config.yaml'))"

# Or use online YAML validator
# Check indentation (use spaces, not tabs)
# Ensure proper key-value format
```

#### Missing Required Fields

**Problem:**
```
Error: database host is required
```

**Solution:**
```yaml
# Ensure all required fields are present
database:
  host: "localhost"          # Required
  port: 5432                # Required
  name: "testdb"            # Required
  username: "testuser"      # Required
  password: "password"      # Required
```

#### Invalid Duration Format

**Problem:**
```
Error: invalid duration format "5 minutes"
```

**Solution:**
```yaml
# Use Go duration format
workload:
  duration: "5m"           # Correct
  # duration: "5 minutes"  # Wrong
  
# Valid formats:
# "30s" - 30 seconds
# "5m"  - 5 minutes
# "2h"  - 2 hours
# "1h30m" - 1 hour 30 minutes
```

### Database Connection Issues

#### Connection Refused

**Problem:**
```
Error: connection refused
```

**Solution:**
```bash
# Check PostgreSQL is running
sudo systemctl status postgresql  # Linux
brew services list | grep postgres  # macOS

# Start PostgreSQL if stopped
sudo systemctl start postgresql  # Linux
brew services start postgresql  # macOS

# Check connection manually
psql -h localhost -p 5432 -U testuser -d testdb
```

#### Authentication Failed

**Problem:**
```
Error: password authentication failed for user "testuser"
```

**Solution:**
```sql
-- Check user exists
SELECT * FROM pg_user WHERE usename = 'testuser';

-- Create user if missing
CREATE USER testuser WITH PASSWORD 'password';

-- Grant necessary permissions
GRANT ALL PRIVILEGES ON DATABASE testdb TO testuser;
GRANT CREATE ON SCHEMA public TO testuser;
```

#### SSL/TLS Issues

**Problem:**
```
Error: SSL connection required
```

**Solution:**
```yaml
# Configure SSL in config
database:
  ssl_mode: "require"        # or "disable" for testing
  ssl_cert: "/path/to/client.crt"
  ssl_key: "/path/to/client.key"
  ssl_root_cert: "/path/to/ca.crt"
```

#### Connection Pool Exhaustion

**Problem:**
```
Error: could not acquire connection from pool
```

**Solution:**
```yaml
# Increase pool size
connection_pool:
  max_connections: 100       # Increase from default
  acquire_timeout: "60s"     # Increase timeout
  
# Or reduce concurrent users
workload:
  concurrent_users: 10       # Reduce load
```

### Runtime Issues

#### High Memory Usage

**Problem:**
```
StormDB consuming excessive memory
```

**Solution:**
```yaml
# Limit memory usage
metrics:
  buffer_size: 1000          # Reduce from default
  collection_interval: "5s"  # Reduce frequency

connection_pool:
  max_connections: 20        # Reduce pool size
  
# Disable unnecessary features
metrics:
  system_metrics: false      # If not needed
  detailed_logging: false
```

#### Slow Performance

**Problem:**
```
Tests running slower than expected
```

**Solution:**
```yaml
# Optimize connection pool
connection_pool:
  initial_connections: 10    # Pre-warm pool
  prepared_statement_cache: true

# Check PostgreSQL settings
# shared_buffers, work_mem, effective_cache_size

# Optimize workload
workload:
  operations:
    - type: "select"
      query: "SELECT * FROM table WHERE id = $1"  # Use indexes
      parameters: ["random_int(1,1000)"]
```

#### Plugin Crashes

**Problem:**
```
Plugin caused segmentation fault
```

**Solution:**
```bash
# Enable debug mode
stormdb --config config.yaml --log-level debug --plugin-debug

# Test plugin independently
stormdb --test-plugin plugins/plugin.so

# Check plugin logs
tail -f /var/log/stormdb/plugin.log

# Use different plugin version
# or disable problematic plugin
```

### Output Issues

#### No Output Generated

**Problem:**
```
Test completed but no output file generated
```

**Solution:**
```bash
# Check output permissions
ls -la $(dirname "output_file.json")

# Specify absolute path
stormdb --config config.yaml --output /tmp/results.json

# Check disk space
df -h /tmp
```

#### Malformed JSON Output

**Problem:**
```
Error: invalid JSON in output file
```

**Solution:**
```yaml
# Enable JSON validation
output:
  json_pretty: true
  validate_json: true

# Check for interruption during write
# Ensure sufficient disk space
# Use atomic writes
output:
  atomic_writes: true
```

#### Missing Metrics

**Problem:**
```
Expected metrics not present in output
```

**Solution:**
```yaml
# Enable all metrics
metrics:
  enabled: true
  pg_stats_enabled: true
  system_metrics: true
  transaction_metrics: true
  connection_metrics: true
  
# Check metric collection interval
metrics:
  collection_interval: "1s"  # More frequent collection
```

### PostgreSQL-Specific Issues

#### Lock Contention

**Problem:**
```
High lock wait times reported
```

**Solution:**
```sql
-- Check current locks
SELECT * FROM pg_locks WHERE NOT granted;

-- Check blocking queries
SELECT 
  blocked_locks.pid AS blocked_pid,
  blocked_activity.usename AS blocked_user,
  blocking_locks.pid AS blocking_pid,
  blocking_activity.usename AS blocking_user,
  blocked_activity.query AS blocked_statement,
  blocking_activity.query AS current_statement_in_blocking_process
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity 
  ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks 
  ON blocking_locks.locktype = blocked_locks.locktype
  AND blocking_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
  AND blocking_locks.page IS NOT DISTINCT FROM blocked_locks.page
  AND blocking_locks.tuple IS NOT DISTINCT FROM blocked_locks.tuple
  AND blocking_locks.virtualxid IS NOT DISTINCT FROM blocked_locks.virtualxid
  AND blocking_locks.transactionid IS NOT DISTINCT FROM blocked_locks.transactionid
  AND blocking_locks.classid IS NOT DISTINCT FROM blocked_locks.classid
  AND blocking_locks.objid IS NOT DISTINCT FROM blocked_locks.objid
  AND blocking_locks.objsubid IS NOT DISTINCT FROM blocked_locks.objsubid
  AND blocking_locks.pid != blocked_locks.pid
JOIN pg_catalog.pg_stat_activity blocking_activity 
  ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted;
```

#### Deadlocks

**Problem:**
```
Deadlock detected errors
```

**Solution:**
```yaml
# Reduce transaction complexity
workload:
  operations:
    - type: "simple_select"  # Avoid complex transactions
      
# Add retry logic
connection_pool:
  retry_attempts: 3
  retry_delay: "100ms"

# Order operations consistently
# to avoid deadlock cycles
```

#### Connection Limit Exceeded

**Problem:**
```
Error: remaining connection slots are reserved
```

**Solution:**
```sql
-- Check current connections
SELECT count(*) FROM pg_stat_activity;

-- Check connection limit
SELECT setting FROM pg_settings WHERE name = 'max_connections';

-- Increase max_connections (postgresql.conf)
max_connections = 200

-- Or reduce StormDB connections
```

```yaml
connection_pool:
  max_connections: 50  # Less than PostgreSQL max_connections
```

### Advanced Troubleshooting

#### Enable Debug Logging

```bash
# Maximum verbosity
stormdb --config config.yaml \
        --log-level debug \
        --verbose \
        --log-file debug.log

# Plugin-specific debugging
stormdb --config config.yaml \
        --plugin-debug \
        --log-queries \
        --log-parameters
```

#### Performance Profiling

```bash
# CPU profiling
stormdb --config config.yaml --cpu-profile cpu.prof

# Memory profiling
stormdb --config config.yaml --memory-profile mem.prof

# Analyze profiles (requires Go tools)
go tool pprof cpu.prof
go tool pprof mem.prof
```

#### Network Debugging

```bash
# Monitor network connections
netstat -an | grep :5432

# Monitor TCP traffic
tcpdump -i any port 5432

# Check DNS resolution
nslookup your-db-host

# Test network connectivity
telnet your-db-host 5432
```

#### Database Debugging

```sql
-- Enable query logging (postgresql.conf)
log_statement = 'all'
log_min_duration_statement = 0

-- Monitor active queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query 
FROM pg_stat_activity 
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes';

-- Check database statistics
SELECT * FROM pg_stat_database WHERE datname = 'your_db';

-- Monitor table access
SELECT * FROM pg_stat_user_tables;
```

## Diagnostic Commands

### System Information

```bash
# Check StormDB version
stormdb --version

# Check system resources
free -h          # Memory
df -h           # Disk space
cat /proc/cpuinfo | grep processor | wc -l  # CPU cores

# Check PostgreSQL version
psql --version
```

### Configuration Validation

```bash
# Validate configuration
stormdb --config config.yaml --validate

# Show resolved configuration
stormdb --config config.yaml --show-config

# Test database connection
stormdb --config config.yaml --test-connection

# Check plugin loading
stormdb --config config.yaml --scan-plugins
```

### Runtime Diagnostics

```bash
# Monitor StormDB process
top -p $(pgrep stormdb)

# Check open files
lsof -p $(pgrep stormdb)

# Monitor network connections
ss -tuln | grep stormdb

# Check memory usage
cat /proc/$(pgrep stormdb)/status | grep Vm
```

## Error Code Reference

### Exit Codes

- `0` - Success
- `1` - General error
- `2` - Configuration error
- `3` - Database connection error
- `4` - Plugin error
- `5` - Resource error (memory, disk, etc.)
- `6` - Signal interrupted (SIGINT, SIGTERM)

### Common Error Messages

**Configuration Errors:**
- `invalid configuration file format`
- `missing required field: database.host`
- `invalid duration format`
- `workload type not supported`

**Connection Errors:**
- `connection refused`
- `authentication failed`
- `SSL connection required`
- `connection timeout`

**Plugin Errors:**
- `plugin not found`
- `plugin initialization failed`
- `plugin operation failed`
- `incompatible plugin version`

**Runtime Errors:**
- `out of memory`
- `disk space insufficient`
- `maximum connections exceeded`
- `operation timeout`

## Getting Help

### Log Analysis

```bash
# Search for errors in logs
grep -i error stormdb.log

# Check for warnings
grep -i warning stormdb.log

# Look for performance issues
grep -i "slow\|timeout\|failed" stormdb.log

# Analyze connection issues
grep -i "connection\|pool" stormdb.log
```

### Community Resources

- [GitHub Issues](https://github.com/elchinoo/stormdb/issues) - Report bugs and get help
- [GitHub Discussions](https://github.com/elchinoo/stormdb/discussions) - Community support
- [Documentation](../README.md) - Comprehensive documentation

### Creating Bug Reports

When reporting issues, include:

1. **Version Information:**
   ```bash
   stormdb --version
   ```

2. **Configuration File:**
   ```yaml
   # Sanitized configuration (remove passwords)
   ```

3. **Error Messages:**
   ```
   Complete error output
   ```

4. **System Information:**
   ```bash
   uname -a
   cat /etc/os-release
   psql --version
   ```

5. **Steps to Reproduce:**
   ```
   1. Step one
   2. Step two
   3. Error occurs
   ```

6. **Log Files:**
   ```bash
   # Include relevant log entries
   stormdb --config config.yaml --log-level debug --log-file debug.log
   ```

## Prevention Best Practices

### Configuration Management
- Use version control for configurations
- Validate configurations before use
- Document configuration changes
- Test in development environment first

### Monitoring
- Set up comprehensive monitoring
- Configure alerting for critical issues
- Monitor trends over time
- Regular health checks

### Maintenance
- Keep StormDB updated
- Update plugins regularly
- Monitor PostgreSQL performance
- Regular backup testing

### Testing
- Start with simple configurations
- Gradually increase complexity
- Test in isolated environments
- Validate results regularly

## Next Steps

- [Configuration Guide](CONFIGURATION.md) - Optimize your configuration
- [Performance Optimization](PERFORMANCE_OPTIMIZATION.md) - Improve performance
- [Usage Guide](USAGE.md) - Learn all command options
- [Plugin System](PLUGIN_SYSTEM.md) - Work with plugins effectively
