# Connection Overhead Workload Plugin

This plugin provides connection overhead testing for PostgreSQL performance analysis and connection pool optimization.

## Overview

The connection overhead workload is specifically designed to measure and analyze:

- Connection establishment overhead
- Connection pool efficiency
- Connection reuse patterns  
- Connection-related bottlenecks
- Pool size optimization

## Features

- Minimal query overhead to isolate connection costs
- Configurable connection patterns
- Connection lifecycle measurement
- Pool utilization analysis
- Connection establishment timing

## Configuration

The connection workload supports the following configuration parameters:

```yaml
workload:
  type: "simple_connection"
  query_type: "ping"           # Type of minimal query to execute
  connection_pattern: "reuse"   # "reuse", "fresh", "mixed"
  min_connections: 1           # Minimum connections in pool
  max_connections: 100         # Maximum connections in pool
  connection_lifetime: "1h"    # Connection lifetime
  idle_timeout: "30m"          # Idle connection timeout
```

## Operation Types

### Connection Patterns
- **reuse**: Maximize connection reuse from pool
- **fresh**: Force new connections (simulates burst traffic)
- **mixed**: Realistic mix of reuse and new connections

### Query Types
- **ping**: Simple `SELECT 1` queries
- **time**: `SELECT NOW()` for minimal server load
- **version**: `SELECT version()` for constant response

## Performance Characteristics

This workload helps identify:
- **Connection Pool Bottlenecks**: When pool size becomes limiting factor
- **Connection Overhead**: Time spent establishing vs. using connections
- **Pool Configuration**: Optimal min/max pool sizes
- **Connection Leaks**: Connections not properly returned to pool
- **Scaling Limits**: Maximum sustainable concurrent connections

## Metrics

Key metrics measured:
- Connection establishment time
- Connection acquisition time from pool
- Connection utilization ratio
- Pool exhaustion events
- Connection lifetime distribution

## Usage

```bash
# Test connection pool efficiency
pgstorm --workload simple_connection --duration 60s --workers 50 --connections 10

# Test connection scalability
pgstorm --workload simple_connection --duration 300s --workers 100 --connections 50

# Test connection overhead with fresh connections
pgstorm --workload simple_connection --duration 60s --workers 20 --connections 1 --connection-pattern fresh

# Progressive scaling to find connection limits
pgstorm --workload simple_connection --progressive-scaling --min-connections 1 --max-connections 100
```

## Analysis

Use this workload to:

1. **Optimize Pool Size**: Find the sweet spot between resource usage and performance
2. **Identify Bottlenecks**: Determine if connections are limiting factor
3. **Plan Capacity**: Understand connection requirements for target load
4. **Debug Pool Issues**: Identify connection leaks or pool exhaustion
5. **Compare Configurations**: Test different pool settings
