# Progressive Connection Scaling

StormDB's progressive scaling feature allows you to automatically test your PostgreSQL database performance across multiple connection and worker configurations in a single run. This advanced feature provides comprehensive mathematical analysis of performance characteristics, helping you find optimal configurations and understand system behavior under different load conditions.

## Overview

Progressive scaling runs your workload through a series of "bands" - each with different numbers of workers and database connections. It collects detailed metrics for each band and performs advanced statistical analysis to identify:

- Optimal worker/connection configurations
- Performance scaling patterns (linear, diminishing returns, saturation, degradation)
- Inflection points where adding resources becomes counterproductive
- Mathematical models describing your system's performance characteristics
- Queueing theory analysis for bottleneck identification

## Quick Start

### Enable Progressive Scaling

Add the progressive scaling configuration to your YAML config file:

```yaml
progressive:
  enabled: true
  min_workers: 10
  max_workers: 100
  step_workers: 10
  min_connections: 10
  max_connections: 100
  step_connections: 10
  band_duration: "30s"
  strategy: "linear"
  export_format: "both"
  export_path: "./results"
```

Or enable it via command line:

```bash
./stormdb --config config.yaml --progressive
```

### Run Progressive Scaling

```bash
# Use existing config with progressive section
./stormdb --config config_progressive_imdb.yaml --setup

# Enable via command line flag
./stormdb --config config_imdb.yaml --progressive --setup
```

## Configuration Options

### Core Progressive Settings

| Parameter | Description | Example | Required |
|-----------|-------------|---------|----------|
| `enabled` | Enable progressive scaling | `true` | Yes |
| `min_workers` | Starting number of workers | `10` | Yes |
| `max_workers` | Maximum number of workers | `100` | Yes |
| `step_workers` | Worker increment per band | `10` | Yes |
| `min_connections` | Starting connections | `10` | Yes |
| `max_connections` | Maximum connections | `100` | Yes |
| `step_connections` | Connection increment | `10` | Yes |
| `band_duration` | Duration per band | `"30s"` | Yes |

### Timing Configuration

| Parameter | Description | Default | Notes |
|-----------|-------------|---------|-------|
| `warmup_time` | Warmup before metrics collection | `"10s"` | Allows system to stabilize |
| `cooldown_time` | Rest time between bands | `"5s"` | Prevents interference |

### Scaling Strategies

#### Linear Strategy (Default)
Creates every combination of worker/connection values within the specified ranges.

```yaml
strategy: "linear"
min_workers: 10
max_workers: 30
step_workers: 10
min_connections: 20
max_connections: 40
step_connections: 10
```

**Bands created:** (10,20), (10,30), (10,40), (20,20), (20,30), (20,40), (30,20), (30,30), (30,40)

#### Exponential Strategy
Doubles values each step (respecting minimum increments).

```yaml
strategy: "exponential"
min_workers: 5
max_workers: 80
```

**Bands created:** 5→10→20→40→80 workers

#### Fibonacci Strategy
Uses fibonacci sequence for organic scaling patterns.

```yaml
strategy: "fibonacci"
min_workers: 1
max_workers: 55
```

**Bands created:** 1→1→2→3→5→8→13→21→34→55 workers

### Export Configuration

| Parameter | Description | Options |
|-----------|-------------|---------|
| `export_format` | Output format | `"csv"`, `"json"`, `"both"` |
| `export_path` | Output directory | `"./progressive_results"` |

## Mathematical Analysis Features

### Statistical Metrics (Per Band)

- **Basic Performance:** TPS, QPS, latency (avg, P50, P95, P99)
- **Variability:** Standard deviation, variance, coefficient of variation
- **Confidence:** 95% confidence intervals around mean latency
- **Efficiency:** TPS per worker, connection utilization rates

### Advanced Analysis

#### 1. Marginal Gains (Discrete Derivatives)
Calculates the performance gain per additional worker/connection:

```
ΔTPS/ΔWorkers = (TPS₂ - TPS₁) / (Workers₂ - Workers₁)
```

**Use case:** Identify when adding resources provides diminishing returns.

#### 2. Inflection Points (Second Derivatives)
Detects points where performance behavior changes:

```
Δ²TPS = (ΔTPS₂ - ΔTPS₁)
```

**Use case:** Find where scaling transitions from beneficial to harmful.

#### 3. Curve Fitting & Regression
Fits mathematical models to performance data:

- **Linear:** TPS = a × workers + b
- **Logarithmic:** TPS = a × ln(workers) + b  
- **Exponential:** TPS = a × e^(b × workers)

**Use case:** Predict performance at untested configurations.

#### 4. Queueing Theory Analysis
Models your system as an M/M/c queue:

- **Utilization:** ρ = λ/(c×μ) where λ=arrival rate, c=connections, μ=service rate
- **Wait Time:** Theoretical vs observed latency comparison
- **Bottleneck Detection:** CPU-bound, I/O-bound, or queue-bound identification

#### 5. Performance Region Classification
Automatically categorizes scaling behavior:

- **Linear Scaling:** Consistent performance gains
- **Diminishing Returns:** Decreasing marginal gains  
- **Saturation:** Minimal improvement with more resources
- **Degradation:** Performance decreases with more resources

## Sample Configurations

### High-Frequency Trading (Linear)
```yaml
progressive:
  enabled: true
  min_workers: 10
  max_workers: 50
  step_workers: 5
  min_connections: 20
  max_connections: 100
  step_connections: 10
  band_duration: "45s"
  warmup_time: "15s"
  strategy: "linear"
  export_format: "csv"
```

### Capacity Planning (Exponential)
```yaml
progressive:
  enabled: true
  min_workers: 1
  max_workers: 128
  step_workers: 1
  min_connections: 5
  max_connections: 200
  step_connections: 5
  band_duration: "1m"
  strategy: "exponential"
  export_format: "json"
```

### Research & Analysis (Fibonacci)
```yaml
progressive:
  enabled: true
  min_workers: 1
  max_workers: 89
  step_workers: 1
  min_connections: 2
  max_connections: 144
  step_connections: 2
  band_duration: "2m"
  warmup_time: "30s"
  cooldown_time: "15s"
  strategy: "fibonacci"
  export_format: "both"
```

## Output & Results

### Real-time Output
```
🎯 Starting progressive scaling test with 25 bands
📊 Strategy: linear, Band Duration: 30s, Warmup: 10s, Cooldown: 5s

🔄 Band 1/25: 10 workers, 20 connections
🔥 Warming up for 10s...
📊 Band 1 completed: 1,234 TPS, 45.2ms avg latency

🔄 Band 2/25: 10 workers, 30 connections
📊 Band 2 completed: 1,456 TPS, 42.1ms avg latency
...

✅ Progressive scaling completed successfully
📊 Tested 25 bands, optimal config: 40 workers, 60 connections (2,341 TPS)
```

### CSV Export
```csv
band_id,workers,connections,total_tps,avg_latency_ms,p95_latency_ms,error_rate,...
1,10,20,1234.50,45.20,89.30,0.02,...
2,10,30,1456.20,42.10,82.40,0.01,...
```

### JSON Export
```json
{
  "test_start_time": "2025-01-15T10:30:00Z",
  "workload": "imdb",
  "strategy": "linear",
  "bands": [...],
  "optimal_config": {
    "workers": 40,
    "connections": 60,
    "tps": 2341.50,
    "reasoning": "Selected for optimal efficiency while maintaining high throughput"
  },
  "analysis": {
    "marginal_gains": [...],
    "inflection_points": [...],
    "curve_fitting": {...},
    "queueing_theory": {...},
    "recommendations": [...]
  }
}
```

## Performance Recommendations

Progressive scaling generates actionable recommendations:

### Configuration Recommendations
- Optimal worker/connection counts based on efficiency analysis
- Warnings about over-saturation or resource contention
- Suggestions for production deployment ranges

### System Recommendations  
- Database tuning suggestions based on bottleneck analysis
- Hardware upgrade recommendations for scaling limitations
- Connection pool sizing guidance

### Example Recommendations
```json
{
  "type": "configuration",
  "priority": "high", 
  "category": "workers",
  "suggestion": "Consider optimal worker count around band 15 where performance growth slows",
  "expected_gain": 15.0,
  "confidence": 0.8
}
```

## Use Cases

### 1. Capacity Planning
Determine maximum sustainable load and optimal resource allocation for production systems.

### 2. Performance Tuning
Identify optimal PostgreSQL configuration parameters and connection pool sizes.

### 3. Hardware Sizing
Understand how additional CPU cores or memory affects performance scaling.

### 4. Cost Optimization
Find the minimum resources needed to meet performance requirements.

### 5. Research & Analysis
Generate data for academic research, performance modeling, or system optimization studies.

## Best Practices

### 1. Test Duration Selection
- **Short bands (15-30s):** Quick configuration exploration
- **Medium bands (1-2m):** Balanced accuracy and runtime
- **Long bands (5m+):** High precision for production planning

### 2. Scaling Strategy Selection
- **Linear:** Comprehensive analysis, longer runtime
- **Exponential:** Quick saturation point identification
- **Fibonacci:** Natural progression, good for research

### 3. Resource Ranges
- Start with small ranges to understand behavior
- Extend ranges based on initial findings
- Consider hardware limitations when setting maximums

### 4. Monitoring Integration
- Always enable `collect_pg_stats` for comprehensive analysis
- Use `pg_stats_statements` for query-level insights
- Monitor system resources during tests

### 5. Result Analysis
- Focus on inflection points for optimal configurations
- Use confidence intervals to assess result reliability
- Compare queueing theory predictions with observations

## Troubleshooting

### Common Issues

**Long execution times:**
- Reduce band duration or scaling ranges
- Use exponential strategy for faster exploration
- Consider fewer connection/worker combinations

**Inconsistent results:**
- Increase warmup_time for system stabilization
- Add cooldown_time between bands
- Check for external system interference

**Memory usage:**
- Limit latency sample collection (automatically handled)
- Use CSV export for memory efficiency
- Monitor system resources during execution

**Database errors:**
- Ensure max_connections doesn't exceed PostgreSQL limits
- Verify sufficient database resources
- Check for connection leaks in workload implementation

### Debug Mode
Enable detailed logging:
```bash
export STORMDB_LOG_LEVEL=debug
./stormdb --config config_progressive.yaml --progressive
```

## Integration with CI/CD

Progressive scaling can be integrated into continuous integration pipelines for performance regression testing:

```yaml
# GitHub Actions example
- name: Performance Regression Test
  run: |
    ./stormdb --config config_progressive_ci.yaml --progressive
    # Analyze results and fail on performance regression
    python analyze_progressive_results.py
```

## Extending Progressive Scaling

The progressive scaling engine is designed to be extensible. You can:

1. **Add new scaling strategies** by implementing the strategy interface
2. **Extend analysis methods** with additional mathematical models
3. **Customize export formats** for integration with other tools
4. **Add new performance metrics** specific to your use cases

For advanced customization, see the developer documentation in `docs/PLUGIN_DEVELOPMENT.md`.
