# Usage Guide

This guide covers all command-line options and usage patterns for StormDB.

## Basic Usage

### Quick Start

```bash
# Run with default configuration
stormdb --config config.yaml

# Run with duration override
stormdb --config config.yaml --duration 5m

# Run specific workload type
stormdb --config config.yaml --workload basic

# Run with custom output
stormdb --config config.yaml --output results.json
```

### Command Line Options

#### Core Options

```bash
stormdb [OPTIONS]

Options:
  -c, --config FILE          Configuration file path (required)
  -d, --duration DURATION    Override test duration (e.g., 5m, 1h, 30s)
  -w, --workload TYPE        Override workload type
  -o, --output FILE          Output file path
  -f, --format FORMAT        Output format (console, json, csv, html)
  -v, --verbose              Enable verbose output
  -q, --quiet                Quiet mode (minimal output)
  -h, --help                 Show help message
      --version              Show version information
```

#### Database Options

```bash
Database Connection:
      --host HOST            Database host (default: localhost)
      --port PORT            Database port (default: 5432)
      --database NAME        Database name
      --username USER        Database username
      --password PASS        Database password
      --ssl-mode MODE        SSL mode (disable, require, verify-ca, verify-full)
      --connection-string    PostgreSQL connection string
```

#### Workload Options

```bash
Workload Control:
      --users COUNT          Number of concurrent users
      --duration TIME        Test duration
      --ramp-up TIME         Ramp-up time
      --ramp-down TIME       Ramp-down time
      --think-time TIME      Think time between transactions
      --max-transactions N   Maximum transactions per user
```

#### Monitoring Options

```bash
Monitoring:
      --metrics              Enable metrics collection
      --metrics-interval     Metrics collection interval
      --pg-stats             Enable PostgreSQL statistics
      --system-metrics       Enable system metrics collection
      --dashboard            Start web dashboard
      --dashboard-port       Dashboard port (default: 8080)
```

#### Plugin Options

```bash
Plugin Management:
      --plugin-path PATH     Plugin directory path
      --list-plugins         List available plugins
      --plugin-info PLUGIN   Show plugin information
      --scan-plugins         Scan for plugins in all directories
```

#### Utility Options

```bash
Utilities:
      --validate             Validate configuration without running
      --test-connection      Test database connection
      --list-workloads       List available workload types
      --dry-run              Show what would be executed
      --show-config          Display resolved configuration
```

## Configuration Override

### Command Line Overrides

Many configuration options can be overridden from the command line:

```bash
# Override database connection
stormdb --config config.yaml \
        --host production-db.example.com \
        --port 5432 \
        --database prod_db \
        --username prod_user

# Override workload settings
stormdb --config config.yaml \
        --users 50 \
        --duration 10m \
        --ramp-up 2m

# Override output settings
stormdb --config config.yaml \
        --format json \
        --output results-$(date +%Y%m%d-%H%M%S).json
```

### Environment Variable Overrides

```bash
# Set environment variables
export STORMDB_DB_HOST="localhost"
export STORMDB_DB_PASSWORD="secure_password"
export STORMDB_USERS="20"

# Run with environment overrides
stormdb --config config.yaml
```

### Precedence Order

Configuration values are resolved in this order (highest to lowest precedence):
1. Command line arguments
2. Environment variables
3. Configuration file
4. Default values

## Running Different Workload Types

### Basic SQL Workload

```bash
# Run basic workload
stormdb --config config_basic.yaml

# Override concurrent users
stormdb --config config_basic.yaml --users 25

# Run for specific duration
stormdb --config config_basic.yaml --duration 1h
```

### TPC-C Benchmark

```bash
# Run TPC-C with default settings
stormdb --config config_tpcc.yaml

# Run TPC-C with specific warehouse count
stormdb --config config_tpcc.yaml --warehouses 10

# Run TPC-C with custom duration
stormdb --config config_tpcc.yaml --duration 30m
```

### Plugin-based Workloads

```bash
# List available plugins
stormdb --list-plugins

# Run IMDB workload
stormdb --config config_imdb.yaml --workload imdb_plugin

# Run e-commerce workload
stormdb --config config_ecommerce.yaml --workload ecommerce_plugin

# Run vector workload
stormdb --config config_vector.yaml --workload vector_plugin
```

### Progressive Scaling

```bash
# Run progressive scaling test
stormdb --config config_progressive.yaml

# Override scaling parameters
stormdb --config config_progressive.yaml \
        --initial-users 1 \
        --max-users 100 \
        --step-size 10 \
        --step-duration 3m
```

## Output and Reporting

### Console Output

```bash
# Normal console output
stormdb --config config.yaml

# Verbose output with detailed metrics
stormdb --config config.yaml --verbose

# Quiet output (errors only)
stormdb --config config.yaml --quiet

# Show live progress
stormdb --config config.yaml --progress
```

### File Output

```bash
# JSON output
stormdb --config config.yaml --format json --output results.json

# CSV output for analysis
stormdb --config config.yaml --format csv --output results.csv

# HTML report
stormdb --config config.yaml --format html --output report.html

# Multiple formats
stormdb --config config.yaml \
        --format json --output results.json \
        --format html --output report.html
```

### Timestamped Output

```bash
# Add timestamp to filename
stormdb --config config.yaml \
        --output "results-$(date +%Y%m%d-%H%M%S).json"

# Or use built-in timestamp option
stormdb --config config.yaml \
        --output results.json \
        --timestamp-suffix
```

## Monitoring and Dashboards

### Real-time Monitoring

```bash
# Enable web dashboard
stormdb --config config.yaml --dashboard

# Custom dashboard port
stormdb --config config.yaml --dashboard --dashboard-port 9090

# Dashboard with system metrics
stormdb --config config.yaml --dashboard --system-metrics
```

### Metrics Collection

```bash
# Enable all metrics
stormdb --config config.yaml \
        --metrics \
        --pg-stats \
        --system-metrics

# Custom metrics interval
stormdb --config config.yaml \
        --metrics \
        --metrics-interval 5s
```

### External Monitoring Integration

```bash
# Export metrics to external system
stormdb --config config.yaml \
        --metrics-export prometheus \
        --metrics-port 9091

# Send metrics to webhook
stormdb --config config.yaml \
        --webhook-url https://monitoring.example.com/metrics
```

## Advanced Usage Patterns

### Batch Testing

```bash
#!/bin/bash
# Run multiple test configurations

configs=("config_low.yaml" "config_medium.yaml" "config_high.yaml")

for config in "${configs[@]}"; do
  echo "Running test with $config..."
  stormdb --config "$config" \
          --output "results_$(basename "$config" .yaml).json"
  echo "Completed $config"
done
```

### Automated Testing

```bash
#!/bin/bash
# Automated performance regression testing

# Run baseline test
stormdb --config baseline.yaml --output baseline_results.json

# Run current test
stormdb --config current.yaml --output current_results.json

# Compare results (requires custom comparison script)
./compare_results.py baseline_results.json current_results.json
```

### Continuous Integration

```bash
# CI/CD pipeline example
#!/bin/bash

# Validate configuration
stormdb --config ci_config.yaml --validate
if [ $? -ne 0 ]; then
  echo "Configuration validation failed"
  exit 1
fi

# Test database connection
stormdb --config ci_config.yaml --test-connection
if [ $? -ne 0 ]; then
  echo "Database connection failed"
  exit 1
fi

# Run performance test
stormdb --config ci_config.yaml \
        --duration 5m \
        --output ci_results.json

# Check for regressions
./check_performance_regression.py ci_results.json
```

### Load Testing Scenarios

```bash
# Gradual load increase
stormdb --config config.yaml \
        --users 10 \
        --duration 5m \
        --ramp-up 1m \
        --ramp-down 30s

# Spike testing
stormdb --config config.yaml \
        --users 100 \
        --duration 2m \
        --ramp-up 10s \
        --ramp-down 10s

# Sustained load
stormdb --config config.yaml \
        --users 50 \
        --duration 1h \
        --ramp-up 5m \
        --ramp-down 5m
```

## Debugging and Troubleshooting

### Configuration Debugging

```bash
# Validate configuration
stormdb --config config.yaml --validate

# Show resolved configuration
stormdb --config config.yaml --show-config

# Dry run (show what would be executed)
stormdb --config config.yaml --dry-run

# Test database connection only
stormdb --config config.yaml --test-connection
```

### Verbose Logging

```bash
# Enable debug logging
stormdb --config config.yaml --log-level debug

# Log SQL queries
stormdb --config config.yaml --log-queries

# Log with timestamps
stormdb --config config.yaml --log-timestamps

# Save logs to file
stormdb --config config.yaml --log-file debug.log
```

### Error Handling

```bash
# Continue on errors
stormdb --config config.yaml --continue-on-error

# Maximum error threshold
stormdb --config config.yaml --max-errors 10

# Retry failed operations
stormdb --config config.yaml --retry-attempts 3
```

## Plugin Management

### Plugin Discovery

```bash
# List all available workload types
stormdb --list-workloads

# List installed plugins
stormdb --list-plugins

# Show plugin information
stormdb --plugin-info imdb_plugin

# Scan for plugins in directories
stormdb --scan-plugins
```

### Plugin Development

```bash
# Validate plugin
stormdb --validate-plugin /path/to/plugin.so

# Test plugin loading
stormdb --test-plugin /path/to/plugin.so

# Plugin debugging
stormdb --config config.yaml --plugin-debug
```

## Performance Optimization

### Connection Pool Tuning

```bash
# Optimize for high concurrency
stormdb --config config.yaml \
        --max-connections 100 \
        --initial-connections 20

# Optimize for low latency
stormdb --config config.yaml \
        --max-connections 10 \
        --prepared-statements
```

### Memory Management

```bash
# Limit memory usage
stormdb --config config.yaml \
        --max-memory 1GB \
        --metric-buffer-size 1000

# Enable memory profiling
stormdb --config config.yaml --memory-profile
```

### I/O Optimization

```bash
# Batch operations
stormdb --config config.yaml \
        --batch-size 1000 \
        --batch-timeout 100ms

# Async operations
stormdb --config config.yaml --async-operations
```

## Integration Examples

### Docker Integration

```bash
# Run in Docker
docker run --rm \
  -v $(pwd)/config:/config \
  -v $(pwd)/results:/results \
  elchinoo/stormdb:latest \
  --config /config/docker_config.yaml \
  --output /results/docker_results.json

# Docker Compose
docker-compose up stormdb
```

### Kubernetes Integration

```bash
# Run as Kubernetes job
kubectl apply -f stormdb-job.yaml

# Monitor job progress
kubectl logs -f job/stormdb-test

# Get results
kubectl cp stormdb-test-pod:/results/results.json ./results.json
```

### Cloud Integration

```bash
# AWS RDS testing
stormdb --config config.yaml \
        --host mydb.cluster-xyz.us-west-2.rds.amazonaws.com \
        --ssl-mode require

# Google Cloud SQL testing
stormdb --config config.yaml \
        --host 10.0.0.5 \
        --ssl-mode require \
        --ssl-cert client-cert.pem
```

## Best Practices

### Test Planning

1. **Start Small**: Begin with low concurrency and short duration
2. **Gradual Scaling**: Increase load gradually to find limits
3. **Baseline Testing**: Establish baseline performance metrics
4. **Environment Consistency**: Use consistent test environments

### Configuration Management

1. **Version Control**: Store configurations in version control
2. **Environment-specific**: Use separate configs for dev/test/prod
3. **Parameterization**: Use environment variables for dynamic values
4. **Documentation**: Document configuration choices and reasoning

### Monitoring and Analysis

1. **Multiple Metrics**: Monitor both application and database metrics
2. **Trend Analysis**: Look for performance trends over time
3. **Resource Correlation**: Correlate performance with resource usage
4. **Error Analysis**: Analyze error patterns and causes

### Result Interpretation

1. **Statistical Significance**: Ensure sufficient sample sizes
2. **Outlier Analysis**: Investigate performance outliers
3. **Context Awareness**: Consider external factors affecting performance
4. **Actionable Insights**: Focus on metrics that drive optimization decisions

## Next Steps

- [Configuration Guide](CONFIGURATION.md) - Detailed configuration options
- [Plugin System](PLUGIN_SYSTEM.md) - Custom workload development
- [Performance Optimization](PERFORMANCE_OPTIMIZATION.md) - Tuning tips
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues and solutions
