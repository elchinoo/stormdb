# Configuration Guide

This guide covers all aspects of configuring StormDB for your testing needs.

## Configuration Overview

StormDB uses YAML configuration files to define:
- Database connection parameters
- Workload types and settings
- Execution parameters
- Monitoring and metrics
- Output formatting

## Basic Configuration Structure

```yaml
# Database connection
database:
  host: "localhost"
  port: 5432
  name: "testdb"
  username: "testuser"
  password: "password"
  ssl_mode: "disable"

# Connection pool settings
connection_pool:
  max_connections: 10
  initial_connections: 2
  connection_lifetime: "30m"
  retry_attempts: 3

# Workload configuration
workload:
  type: "basic"
  duration: "5m"
  ramp_up_time: "30s"
  ramp_down_time: "30s"

# Metrics and monitoring
metrics:
  collection_interval: "1s"
  pg_stats_enabled: true
  system_metrics: true

# Output configuration
output:
  format: "console"
  file: "results.json"
  detailed_logs: true
```

## Database Configuration

### Connection Parameters

```yaml
database:
  # Basic connection settings
  host: "localhost"                    # Database host
  port: 5432                          # Database port
  name: "stormdb_test"                # Database name
  username: "stormdb_user"            # Username
  password: "secure_password"         # Password
  
  # SSL/TLS settings
  ssl_mode: "require"                 # disable, require, verify-ca, verify-full
  ssl_cert: "/path/to/client.crt"     # Client certificate (optional)
  ssl_key: "/path/to/client.key"      # Client key (optional)
  ssl_root_cert: "/path/to/ca.crt"    # Root CA certificate (optional)
  
  # Advanced connection options
  connect_timeout: "10s"              # Connection timeout
  statement_timeout: "0"              # Statement timeout (0 = no timeout)
  idle_in_transaction_timeout: "0"    # Idle transaction timeout
  
  # Application settings
  application_name: "stormdb"         # Application name in pg_stat_activity
  search_path: "public,extensions"    # Schema search path
```

### Connection Pool Settings

```yaml
connection_pool:
  # Pool size settings
  max_connections: 50                 # Maximum connections in pool
  initial_connections: 5              # Initial connections to create
  min_connections: 2                  # Minimum connections to maintain
  
  # Lifecycle settings
  connection_lifetime: "1h"           # Maximum connection lifetime
  idle_timeout: "10m"                 # Idle connection timeout
  acquire_timeout: "30s"              # Timeout for acquiring connection
  
  # Retry and health settings
  retry_attempts: 3                   # Number of connection retry attempts
  retry_delay: "1s"                   # Delay between retry attempts
  health_check_interval: "30s"        # Connection health check interval
  
  # Performance tuning
  prepared_statement_cache: true      # Enable prepared statement caching
  statement_cache_size: 100           # Number of statements to cache
```

### Multiple Database Support

```yaml
databases:
  primary:
    host: "db1.example.com"
    port: 5432
    name: "app_db"
    username: "app_user"
    password: "app_pass"
    
  secondary:
    host: "db2.example.com"
    port: 5432
    name: "analytics_db"
    username: "analytics_user"
    password: "analytics_pass"

# Use specific database in workload
workload:
  type: "basic"
  database: "secondary"  # References databases.secondary
```

## Workload Configuration

### Basic Workload

```yaml
workload:
  type: "basic"
  
  # Execution control
  duration: "10m"                     # Test duration
  ramp_up_time: "1m"                  # Gradual ramp-up period
  ramp_down_time: "30s"               # Gradual ramp-down period
  
  # Concurrency settings
  concurrent_users: 10                # Number of concurrent connections
  max_transactions_per_user: 1000     # Limit per user (0 = unlimited)
  
  # Transaction mix
  operations:
    - type: "select"
      weight: 70                      # 70% of operations
      query: "SELECT * FROM users WHERE id = $1"
      parameters: ["random_int(1,1000)"]
      
    - type: "insert"
      weight: 20                      # 20% of operations
      query: "INSERT INTO logs (user_id, message) VALUES ($1, $2)"
      parameters: ["random_int(1,1000)", "random_string(50)"]
      
    - type: "update"
      weight: 10                      # 10% of operations
      query: "UPDATE users SET last_login = NOW() WHERE id = $1"
      parameters: ["random_int(1,1000)"]
```

### TPC-C Workload

```yaml
workload:
  type: "tpcc"
  
  # TPC-C specific settings
  warehouses: 10                      # Number of warehouses
  scale_factor: 1.0                   # Scale factor
  
  # Transaction mix (TPC-C standard percentages)
  transaction_mix:
    new_order: 45                     # New order transactions
    payment: 43                       # Payment transactions
    order_status: 4                   # Order status queries
    delivery: 4                       # Delivery transactions
    stock_level: 4                    # Stock level queries
    
  # Performance settings
  think_time: "0ms"                   # Think time between transactions
  keying_time: "0ms"                  # Keying time simulation
  
  # Data generation
  use_random_data: true               # Generate random test data
  preserve_data: false                # Keep data after test
```

### Plugin-based Workloads

```yaml
workload:
  type: "plugin"
  plugin_name: "imdb_plugin"          # Name of the plugin
  
  # Plugin-specific configuration
  plugin_config:
    data_file: "/path/to/imdb.csv"
    batch_size: 1000
    concurrent_loaders: 4
    
    # Custom operation weights
    operation_weights:
      movie_search: 40
      actor_search: 30
      rating_insert: 20
      review_insert: 10
```

## Advanced Workload Features

### Progressive Scaling

```yaml
workload:
  type: "basic"
  
  # Progressive scaling configuration
  progressive_scaling:
    enabled: true
    initial_users: 1
    max_users: 100
    step_size: 5                      # Users to add each step
    step_duration: "2m"               # Duration of each step
    
    # Scaling strategy
    strategy: "linear"                # linear, exponential, custom
    
    # Custom scaling steps
    custom_steps:
      - users: 1
        duration: "1m"
      - users: 5
        duration: "2m"
      - users: 10
        duration: "3m"
      - users: 20
        duration: "2m"
    
    # Analysis settings
    stability_threshold: 0.05         # 5% coefficient of variation
    minimum_samples: 30               # Minimum samples for analysis
    
    # Statistical analysis
    statistics:
      confidence_level: 0.95          # 95% confidence intervals
      outlier_removal: true           # Remove statistical outliers
      smoothing_enabled: true         # Apply data smoothing
```

### Dynamic Workloads

```yaml
workload:
  type: "dynamic"
  
  # Time-based phases
  phases:
    - name: "morning_load"
      start_time: "08:00"
      duration: "2h"
      concurrent_users: 20
      operations:
        - type: "select"
          weight: 80
        - type: "insert"
          weight: 20
          
    - name: "peak_load"
      start_time: "10:00"
      duration: "4h"
      concurrent_users: 100
      operations:
        - type: "select"
          weight: 60
        - type: "insert"
          weight: 25
        - type: "update"
          weight: 15
          
    - name: "evening_load"
      start_time: "18:00"
      duration: "3h"
      concurrent_users: 50
      operations:
        - type: "select"
          weight: 70
        - type: "update"
          weight: 30
```

## Monitoring and Metrics

### Basic Metrics Collection

```yaml
metrics:
  # Collection settings
  enabled: true
  collection_interval: "1s"           # How often to collect metrics
  buffer_size: 10000                  # Metric buffer size
  
  # Database metrics
  pg_stats_enabled: true              # Enable PostgreSQL statistics
  pg_stats_interval: "5s"             # PostgreSQL stats collection interval
  track_slow_queries: true            # Track slow queries
  slow_query_threshold: "1s"          # Slow query threshold
  
  # System metrics
  system_metrics: true                # Enable system metrics
  system_interval: "5s"               # System metrics interval
  track_cpu: true                     # Track CPU usage
  track_memory: true                  # Track memory usage
  track_disk_io: true                 # Track disk I/O
  track_network: true                 # Track network I/O
  
  # Application metrics
  transaction_metrics: true           # Track transaction metrics
  connection_metrics: true            # Track connection pool metrics
  error_tracking: true                # Track errors and failures
```

### Advanced Monitoring

```yaml
metrics:
  # Real-time monitoring
  real_time_dashboard: true           # Enable real-time dashboard
  dashboard_port: 8080                # Dashboard web interface port
  update_interval: "1s"               # Dashboard update interval
  
  # Alerting
  alerts:
    enabled: true
    thresholds:
      response_time_p95: "1s"         # Alert if P95 > 1s
      error_rate: 0.05                # Alert if error rate > 5%
      connection_utilization: 0.8     # Alert if pool utilization > 80%
      
  # Detailed logging
  detailed_logging:
    enabled: true
    log_level: "info"                 # debug, info, warn, error
    log_queries: false                # Log all SQL queries (verbose)
    log_parameters: false             # Log query parameters (sensitive)
    log_results: false                # Log query results (very verbose)
    
  # Custom metrics
  custom_metrics:
    - name: "business_transactions"
      query: "SELECT COUNT(*) FROM orders WHERE created_at > NOW() - INTERVAL '1 minute'"
      interval: "30s"
      
    - name: "active_sessions"
      query: "SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'active'"
      interval: "10s"
```

## Output Configuration

### Console Output

```yaml
output:
  format: "console"                   # console, json, csv, html
  
  # Console display options
  colors: true                        # Enable colored output
  progress_bar: true                  # Show progress bar
  live_updates: true                  # Show live metric updates
  update_interval: "1s"               # Live update interval
  
  # Detail level
  verbosity: "normal"                 # quiet, normal, verbose, debug
  show_queries: false                 # Show SQL queries in output
  show_errors: true                   # Show error details
  
  # Summary options
  summary_statistics: true            # Show summary at end
  percentiles: [50, 90, 95, 99]      # Percentiles to display
```

### File Output

```yaml
output:
  # File output settings
  file: "results.json"                # Output file path
  file_format: "json"                 # json, csv, html, xml
  append_timestamp: true              # Append timestamp to filename
  
  # JSON output options
  json_pretty: true                   # Pretty-print JSON
  json_include_raw: true              # Include raw measurements
  
  # CSV output options
  csv_separator: ","                  # CSV field separator
  csv_headers: true                   # Include headers in CSV
  
  # Compression
  compress: true                      # Compress output files
  compression_format: "gzip"          # gzip, bzip2, xz
```

### Multiple Output Formats

```yaml
output:
  formats:
    - type: "console"
      verbosity: "normal"
      colors: true
      
    - type: "file"
      file: "detailed_results.json"
      format: "json"
      include_raw_data: true
      
    - type: "file"
      file: "summary_results.csv"
      format: "csv"
      summary_only: true
      
    - type: "webhook"
      url: "https://monitoring.example.com/webhook"
      format: "json"
      auth_token: "Bearer your-token-here"
```

## Environment Variables

StormDB supports configuration through environment variables:

```bash
# Database connection
export STORMDB_DB_HOST="localhost"
export STORMDB_DB_PORT="5432"
export STORMDB_DB_NAME="testdb"
export STORMDB_DB_USER="testuser"
export STORMDB_DB_PASSWORD="password"

# Plugin settings
export STORMDB_PLUGIN_PATH="/path/to/plugins"

# Output settings
export STORMDB_OUTPUT_FORMAT="json"
export STORMDB_OUTPUT_FILE="results.json"

# General settings
export STORMDB_CONFIG="/path/to/config.yaml"
export STORMDB_LOG_LEVEL="info"
```

## Configuration Templates

StormDB includes several configuration templates:

### Performance Testing Template

```bash
# Copy template
cp config/templates/performance_test.yaml my_config.yaml

# Customize for your environment
stormdb --config my_config.yaml --validate
```

### Load Testing Template

```bash
# Copy template
cp config/templates/load_test.yaml my_config.yaml

# Edit for your specific load patterns
vim my_config.yaml
```

### Monitoring Template

```bash
# Copy template
cp config/templates/monitoring.yaml my_config.yaml

# Configure monitoring endpoints
stormdb --config my_config.yaml --test-connection
```

## Configuration Validation

### Validate Configuration

```bash
# Validate configuration file
stormdb --config config.yaml --validate

# Validate with detailed output
stormdb --config config.yaml --validate --verbose

# Check specific sections
stormdb --config config.yaml --validate-section database
stormdb --config config.yaml --validate-section workload
```

### Common Validation Errors

**Invalid Duration Format:**
```yaml
# Wrong
duration: "5 minutes"

# Correct
duration: "5m"
```

**Missing Required Fields:**
```yaml
# Missing required database fields
database:
  host: "localhost"
  # Missing: port, name, username, password
```

**Invalid Workload Type:**
```yaml
# Wrong
workload:
  type: "invalid_type"

# Correct - check available types
stormdb --list-workloads
```

## Configuration Best Practices

### Security

```yaml
# Use environment variables for sensitive data
database:
  password: "${STORMDB_DB_PASSWORD}"
  ssl_cert: "${STORMDB_SSL_CERT}"

# Enable SSL in production
database:
  ssl_mode: "require"
  ssl_root_cert: "/path/to/ca.crt"
```

### Performance

```yaml
# Optimize connection pool for your workload
connection_pool:
  max_connections: 50  # Don't exceed PostgreSQL max_connections
  initial_connections: 10  # Start with reasonable number
  
# Use appropriate collection intervals
metrics:
  collection_interval: "1s"  # 1s for detailed analysis
  pg_stats_interval: "5s"    # 5s sufficient for PostgreSQL stats
```

### Monitoring

```yaml
# Enable comprehensive monitoring
metrics:
  pg_stats_enabled: true
  system_metrics: true
  error_tracking: true
  
# Configure alerts for production
alerts:
  enabled: true
  thresholds:
    response_time_p95: "500ms"
    error_rate: 0.01  # 1%
```

## Next Steps

- [Usage Guide](USAGE.md) - Learn command-line options
- [Plugin System](PLUGIN_SYSTEM.md) - Work with custom workloads
- [Examples](../examples/) - Sample configurations
- [Troubleshooting](TROUBLESHOOTING.md) - Common configuration issues
