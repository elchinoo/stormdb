# StormDB Progressive Scaling Guide

## Overview

Progressive scaling is StormDB's advanced testing methodology that systematically increases database load across multiple bands to identify performance boundaries, scaling characteristics, and optimal configuration parameters. This comprehensive guide covers the enhanced progressive scaling system with statistical analysis, visualization, and resilience features.

## Key Features

### 🔄 Multi-Strategy Scaling
- **Linear**: Evenly distributed progression (recommended for initial analysis)
- **Exponential**: Aggressive scaling to quickly identify limits
- **Fibonacci**: Natural progression following mathematical sequence
- **Custom**: User-defined progression steps

### 📊 Advanced Mathematical Analysis
- **Statistical Analysis**: Complete descriptive statistics with confidence intervals
- **Derivative Analysis**: Performance acceleration/deceleration detection
- **Integral Analysis**: Cumulative efficiency and area-under-curve calculations
- **Regression Analysis**: Multi-model curve fitting (linear, polynomial, exponential, logarithmic)
- **Queueing Theory**: M/M/1 model analysis with Little's Law validation
- **Correlation Matrix**: Multi-variate relationship analysis
- **Trend Detection**: Direction, strength, and change point identification
- **Outlier Detection**: Statistical anomaly identification with Z-score analysis

### 🎯 Intelligent Test Management
- **Health Monitoring**: Real-time system health assessment
- **Early Termination**: Automatic stopping on excessive errors or performance degradation
- **Band Configuration**: Flexible test phases with warmup/cooldown periods
- **Quality Assessment**: Data completeness, consistency, and accuracy metrics

## Quick Start

### Basic Progressive Test

```bash
# Run a basic 5-band linear progression test
pgstorm progressive --config config.yaml --strategy linear --bands 5
```

### Custom Parameters

```bash
# Advanced progressive test with custom scaling
pgstorm progressive \
  --config config.yaml \
  --strategy exponential \
  --bands 8 \
  --min-workers 5 \
  --max-workers 200 \
  --min-connections 10 \
  --max-connections 500 \
  --test-duration 3m \
  --warmup-time 45s \
  --enable-analysis \
  --output results.json \
  --report report.html \
  --verbose
```

## Configuration

### YAML Configuration

```yaml
workload:
  progressive:
    enabled: true
    strategy: "linear"           # linear, exponential, fibonacci, custom
    bands: 5                     # Number of test phases
    
    # Scaling parameters
    min_workers: 10
    max_workers: 100
    min_connections: 10
    max_connections: 200
    
    # Timing
    test_duration: "2m"          # Duration per test band
    warmup_duration: "30s"       # Warmup time per band
    cooldown_duration: "15s"     # Cooldown between bands
    
    # Analysis
    enable_analysis: true        # Enable mathematical analysis
    max_latency_samples: 10000   # Memory management
    memory_limit_mb: 1024        # Total memory limit
```

### Command Line Options

| Flag | Description | Example |
|------|-------------|---------|
| `--config, -c` | Configuration file (required) | `--config test.yaml` |
| `--strategy` | Scaling strategy | `--strategy exponential` |
| `--bands` | Number of test bands | `--bands 8` |
| `--min-workers` | Minimum workers | `--min-workers 5` |
| `--max-workers` | Maximum workers | `--max-workers 200` |
| `--min-connections` | Minimum connections | `--min-connections 10` |
| `--max-connections` | Maximum connections | `--max-connections 500` |
| `--test-duration` | Duration per band | `--test-duration 3m` |
| `--warmup-time` | Warmup duration | `--warmup-time 45s` |
| `--cooldown-time` | Cooldown duration | `--cooldown-time 30s` |
| `--enable-analysis` | Enable comprehensive analysis | `--enable-analysis` |
| `--output, -o` | JSON results file | `--output results.json` |
| `--report` | HTML report file | `--report report.html` |
| `--verbose, -v` | Verbose output | `--verbose` |

## Scaling Strategies

### Linear Strategy
```
Workers:     [10, 32, 55, 77, 100]
Connections: [10, 32, 55, 77, 100]
```
- **Use Case**: Initial performance characterization
- **Benefits**: Even progression, easy to interpret
- **Best For**: Understanding baseline scaling behavior

### Exponential Strategy
```
Workers:     [10, 18, 32, 56, 100]
Connections: [10, 18, 32, 56, 100]
```
- **Use Case**: Rapid bottleneck identification
- **Benefits**: Quickly reaches system limits
- **Best For**: Finding maximum sustainable load

### Fibonacci Strategy
```
Workers:     [10, 16, 26, 42, 68, 100]
Connections: [10, 16, 26, 42, 68, 100]
```
- **Use Case**: Natural progression analysis
- **Benefits**: Mathematical elegance, smooth progression
- **Best For**: Comprehensive scaling analysis

## Mathematical Analysis

### Statistical Metrics

#### Descriptive Statistics
- **Central Tendency**: Mean, median, mode
- **Variability**: Standard deviation, variance, coefficient of variation
- **Distribution Shape**: Skewness, kurtosis
- **Quartiles**: Q1, Q3, IQR (Interquartile Range)
- **Confidence Intervals**: 95% and 99% confidence bounds

#### Example Output
```json
{
  "tps_stats": {
    "mean": 1247.8,
    "median": 1245.2,
    "standard_deviation": 127.4,
    "coefficient_of_variation": 0.102,
    "confidence_interval_95": {
      "lower": 1193.2,
      "upper": 1302.4,
      "mean": 1247.8
    }
  }
}
```

### Trend Analysis

#### Derivative Analysis
- **First Derivative**: Rate of change (velocity)
- **Second Derivative**: Acceleration/deceleration
- **Inflection Points**: Performance turning points
- **Change Point Detection**: Automatic regime identification

#### Example Analysis
```json
{
  "derivative_analysis": {
    "max_acceleration": 45.7,
    "max_deceleration": -23.1,
    "inflection_points": [3, 6],
    "performance_regime": "linear_then_plateau"
  }
}
```

### Queueing Theory Analysis

#### M/M/1 Model Metrics
- **Service Rate (μ)**: Requests processed per second
- **Arrival Rate (λ)**: Incoming request rate
- **Utilization Factor (ρ)**: λ/μ ratio
- **Average Queue Length**: E[N] = ρ²/(1-ρ)
- **Average Wait Time**: E[W] = ρ/(μ-λ)
- **Little's Law Validation**: N = λW verification

#### Example Output
```json
{
  "queueing_analysis": {
    "queueing_model": "M/M/1",
    "service_rate": 1342.5,
    "arrival_rate": 1205.7,
    "utilization_factor": 0.898,
    "average_queue_length": 8.12,
    "average_wait_time": 0.0067,
    "saturation_point": 1342.5,
    "littles_law_validation": true
  }
}
```

### Scalability Analysis

#### Linear Scalability Score
Measures how well performance scales linearly with resources:
```
Score = Correlation(Resources, Performance)
Range: 0.0 (poor scaling) to 1.0 (perfect linear scaling)
```

#### Breakpoint Detection
Automatic identification of scaling breakpoints:
- **Knee Point**: Diminishing returns begin
- **Cliff Point**: Performance drops significantly  
- **Plateau Point**: Performance levels off

#### Example Analysis
```json
{
  "scalability_analysis": {
    "linear_scalability_score": 0.847,
    "scalability_breakpoints": [
      {
        "band_id": 4,
        "connections": 80,
        "type": "knee",
        "impact": 0.23,
        "description": "Diminishing returns begin at 80 connections"
      }
    ],
    "optimal_connection_range": {
      "min": 20,
      "max": 80,
      "optimal": 60,
      "confidence": 0.92
    }
  }
}
```

## Results and Reporting

### Console Output
Real-time progress with summary table:
```
================================================================================
PROGRESSIVE TEST RESULTS SUMMARY
================================================================================
Total Bands Executed: 5

Band   Workers    Connections  TPS        QPS        Latency P95  Errors
--------------------------------------------------------------------------------
1      10         10           142.3      213.4      45.2         0
2      32         32           456.7      684.9      52.1         0
3      55         55           789.2      1183.8     67.3         2
4      77         77           967.4      1451.1     94.7         8
5      100        100          1023.1     1534.7     145.2        24

Optimal Configuration: Band 5 (1023.1 TPS)
================================================================================
```

### JSON Output
Complete results with mathematical analysis:
```bash
pgstorm progressive --config test.yaml --output results.json
```

### HTML Report
Visual report with charts and analysis:
```bash
pgstorm progressive --config test.yaml --report report.html
```

## Best Practices

### Test Planning

1. **Start Small**: Begin with 3-5 bands for initial assessment
2. **Conservative Ranges**: Start with 2-4x scaling factors
3. **Adequate Duration**: Allow 2-5 minutes per band for stable metrics
4. **Warmup Periods**: Use 30-60 second warmup to stabilize performance

### Analysis Interpretation

1. **Linear Scalability Score**:
   - \> 0.9: Excellent scalability
   - 0.7-0.9: Good scalability  
   - 0.5-0.7: Moderate scalability
   - < 0.5: Poor scalability, investigate bottlenecks

2. **Utilization Factor**:
   - < 0.8: System not saturated, can handle more load
   - 0.8-0.9: Approaching saturation, monitor closely
   - \> 0.9: Saturated, performance degradation likely

3. **Coefficient of Variation**:
   - < 0.15: Stable performance
   - 0.15-0.3: Moderate variability
   - \> 0.3: High variability, investigate causes

### Resource Planning

1. **Memory**: Allocate 100-200MB per 10,000 latency samples
2. **Duration**: Plan for 10-30 minutes total test time
3. **Database**: Ensure sufficient connections for max_connections parameter
4. **System Resources**: Monitor CPU and memory during tests

## Troubleshooting

### Common Issues

#### High Error Rates
```
Problem: Error rate > 5% in later bands
Solution: Reduce max_connections or increase connection pool size
Analysis: Check error types in ErrorTypes field
```

#### Performance Degradation
```
Problem: TPS decreases in later bands
Solution: Investigate database bottlenecks, check connection limits
Analysis: Review queueing analysis for saturation indicators
```

#### Memory Issues
```
Problem: Out of memory during collection
Solution: Reduce max_latency_samples or memory_limit_mb
Analysis: Monitor memory usage in real-time
```

### Early Termination

The system automatically terminates early if:
- Error rate exceeds 10%
- Health score drops below 50%
- Performance degrades by more than 50%

### Performance Tips

1. **Database Tuning**: Ensure adequate shared_buffers, work_mem
2. **Connection Pooling**: Use pgbouncer for connection management
3. **System Resources**: Monitor CPU, memory, and I/O during tests
4. **Network**: Ensure low latency between client and database

## Advanced Usage

### Custom Strategies

For custom progression patterns, use the `custom` strategy with explicit steps:

```yaml
progressive:
  strategy: "custom"
  custom_steps: [5, 12, 25, 50, 85, 120, 150]
```

### Integration with CI/CD

```bash
#!/bin/bash
# Automated performance regression testing
pgstorm progressive \
  --config ci-config.yaml \
  --output "results-$(date +%Y%m%d).json" \
  --bands 3 \
  --test-duration 1m

# Parse results for performance regression
if [ $(jq '.summary.scalability_score < 0.7' results-*.json) = "true" ]; then
  echo "Performance regression detected"
  exit 1
fi
```

### API Integration

Results can be integrated with monitoring systems:

```bash
# Extract key metrics for monitoring
jq '{
  max_tps: .summary.max_throughput,
  scalability_score: .summary.scalability_score,
  optimal_connections: .scalability_analysis.optimal_connection_range.optimal
}' results.json
```

## References

- [Queueing Theory Fundamentals](https://en.wikipedia.org/wiki/Queueing_theory)
- [Little's Law](https://en.wikipedia.org/wiki/Little%27s_law)
- [Performance Testing Best Practices](https://performance.gov/)
- [PostgreSQL Performance Tuning](https://wiki.postgresql.org/wiki/Performance_Optimization)
