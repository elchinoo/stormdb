# Progressive Connection Scaling v0.2

StormDB's progressive scaling feature is an advanced automated performance testing system that systematically evaluates your PostgreSQL database across multiple connection and worker configurations. This scientific approach eliminates guesswork in performance tuning by providing comprehensive mathematical analysis, bottleneck identification, and optimal configuration recommendations.

## 🎯 Why Progressive Scaling?

### The Traditional Problem
Database performance tuning typically involves:
- **Manual Configuration Testing**: Time-consuming trial-and-error
- **Guesswork**: Estimating optimal settings based on system resources
- **Limited Understanding**: No insight into scaling behavior or bottlenecks
- **Incomplete Analysis**: Testing only a few configurations
- **Subjective Decisions**: Choosing configurations without scientific basis

### The Progressive Scaling Solution
Progressive scaling automates and scientifically enhances this process:
- **Automated Testing**: Test 6-25+ configurations in a single run
- **Mathematical Analysis**: Apply statistical methods to understand performance patterns
- **Bottleneck Identification**: Use queueing theory to classify limitation types
- **Optimal Discovery**: AI-driven recommendation of best configurations
- **Scientific Export**: Comprehensive data for research and production planning

## 🏗️ Architecture & Implementation

Progressive scaling is built on a modular architecture designed for extensibility and scientific rigor:

```
┌─────────────────────────────────────────────────────────────────┐
│                 Progressive Scaling Engine v0.2                 │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │  Scaling Logic  │  │   Mathematical  │  │   Export &      │  │
│  │  • Linear       │  │   Analysis      │  │   Reporting     │  │
│  │  • Exponential  │  │  • Derivatives  │  │  • CSV/JSON     │  │
│  │  • Fibonacci    │  │  • Curve Fit    │  │  • Optimal      │  │
│  │  • Validation   │  │  • Queue Theory │  │    Detection    │  │
│  │  • Sanitization │  │  • NaN/Inf Safe │  │  • Real-time    │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                    Workload Interface Adapter                   │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │    Built-in     │  │     Plugin      │  │   Dynamic       │  │
│  │   Workloads     │  │   Workloads     │  │   Loading       │  │
│  │  • TPCC         │  │  • E-commerce   │  │  • Any Plugin   │  │
│  │  • Simple       │  │  • IMDB         │  │  • Automatic    │  │
│  │  • Connection   │  │  • Vector       │  │    Adaptation   │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                      Band Execution Engine                      │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │    Band 1       │  │    Band 2       │  │    Band N       │  │
│  │  10W, 20C       │  │  20W, 40C       │  │  60W, 120C      │  │
│  │  Warmup: 60s    │  │  Warmup: 60s    │  │  Warmup: 60s    │  │
│  │  Test: 1800s    │  │  Test: 1800s    │  │  Test: 1800s    │  │
│  │  Cooldown: 30s  │  │  Cooldown: 30s  │  │  Cooldown: 30s  │  │
│  │  Metrics ✓      │  │  Metrics ✓      │  │  Metrics ✓      │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Core Components

**1. Scaling Engine (`internal/progressive/engine.go`)**
- Orchestrates band execution and metric collection
- Implements scaling strategies (linear, exponential, fibonacci)
- Manages workload interface adaptation
- Provides real-time progress updates

**2. Mathematical Analysis (`internal/progressive/analysis.go`)**
- Calculates discrete derivatives (marginal gains)
- Detects inflection points (second derivatives)
- Performs curve fitting (linear, log, exponential, logistic)
- Applies queueing theory (M/M/c modeling)
- Classifies performance regions

**3. Export System (`internal/progressive/export.go`)**
- Generates CSV and JSON exports
- Identifies optimal configurations
- Provides scientific data for external analysis
- Includes comprehensive metadata

**4. Configuration Validation**
- Parameter validation and sanitization
- Strategy-specific configuration checks
- Error handling and user feedback

## Overview

Progressive scaling runs your workload through a series of "bands" - each with different numbers of workers and database connections. It collects detailed metrics for each band and performs advanced statistical analysis to identify:

- **Optimal Configurations**: Best worker/connection combinations for your workload
- **Scaling Patterns**: Linear scaling, diminishing returns, saturation, degradation regions
- **Inflection Points**: Critical points where adding resources becomes counterproductive  
- **Mathematical Models**: Curve fitting to predict performance at untested configurations
- **Bottleneck Classification**: Scientific identification of limitation types (CPU, I/O, queue, memory)
- **Efficiency Metrics**: TPS per worker, connection utilization, cost-effectiveness analysis

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

## Configuration Options v0.2

Progressive scaling uses a comprehensive configuration system that defines both test parameters and scaling behavior. All configurations must match the precise YAML format to ensure compatibility with v0.2's enhanced validation system.

### Basic Configuration Format

```yaml
progressive:
  enabled: true
  strategy: "exponential"          # linear, exponential, fibonacci
  min_workers: 10                  # Starting point (must be > 0)
  max_workers: 60                  # Ending point (must be > min_workers)
  min_connections: 20              # Starting connections (must be > 0)
  max_connections: 120             # Ending connections (must be > min_connections)
  test_duration: "30m"             # Duration per band (recommended: 30m for production)
  warmup_duration: "60s"           # Warmup time per band
  cooldown_duration: "30s"         # Cooldown time per band
  bands: 6                         # Number of test configurations (3-25 recommended)
  export_csv: true                 # Enable CSV export
  export_json: true                # Enable JSON export
  enable_analysis: true            # Enable mathematical analysis
```

### Core Progressive Settings

| Parameter | Description | Example | Required |
|-----------|-------------|---------|----------|
| `enabled` | Enable progressive scaling | `true` | Yes |
| `strategy` | Scaling algorithm | `"exponential"` | Yes |
| `min_workers` | Starting number of workers | `10` | Yes |
| `max_workers` | Maximum number of workers | `100` | Yes |
| `min_connections` | Starting connections | `20` | Yes |
| `max_connections` | Maximum connections | `200` | Yes |
| `test_duration` | Duration per band | `"30m"` | No (default: "30m") |
| `warmup_duration` | Warmup before metrics | `"60s"` | No (default: "60s") |
| `cooldown_duration` | Rest time between bands | `"30s"` | No (default: "30s") |
| `bands` | Number of test configurations | `6` | No (default: 5) |
| `export_csv` | Export results to CSV | `true` | No (default: true) |
| `export_json` | Export results to JSON | `true` | No (default: true) |
| `enable_analysis` | Enable mathematical analysis | `true` | No (default: true) |

### Scaling Strategies Explained

**1. Linear Scaling (`strategy: "linear"`)**
- **Best For**: Initial exploration, understanding baseline behavior
- **Pattern**: Equal increments between min and max values
- **Example**: 10→20→30→40→50→60 workers
- **Use Case**: Simple workloads, first-time testing

```yaml
progressive:
  strategy: "linear"
  min_workers: 10
  max_workers: 60
  bands: 6
  # Generates: [10, 20, 30, 40, 50, 60] workers
```

**2. Exponential Scaling (`strategy: "exponential"`)**
- **Best For**: Wide-range exploration, finding scaling limits
- **Pattern**: Exponential growth from min to max
- **Example**: 10→14→20→28→40→60 workers (approximately)
- **Use Case**: Complex systems, unknown performance characteristics

```yaml
progressive:
  strategy: "exponential"
  min_workers: 5
  max_workers: 80
  bands: 8
  # Generates exponential distribution between 5 and 80
```

**3. Fibonacci Scaling (`strategy: "fibonacci"`)**
- **Best For**: Natural scaling patterns, production optimization
- **Pattern**: Fibonacci-like progression
- **Example**: 10→16→26→42→68 workers (scaled to fit range)
- **Use Case**: Fine-tuning, natural load distribution

```yaml
progressive:
  strategy: "fibonacci"
  min_workers: 8
  max_workers: 100
  bands: 7
  # Generates fibonacci-like progression scaled to range
```

### Advanced Configuration Examples

**Production E-commerce Testing (3-hour comprehensive)**
```yaml
progressive:
  enabled: true
  strategy: "exponential"
  min_workers: 20
  max_workers: 200
  min_connections: 40
  max_connections: 400
  test_duration: "30m"              # 30 minutes per band
  warmup_duration: "120s"           # 2-minute warmup
  cooldown_duration: "60s"          # 1-minute cooldown
  bands: 8                          # 8 configurations total
  export_csv: true
  export_json: true
  enable_analysis: true
```

**Quick Development Testing (30-minute rapid)**
```yaml
progressive:
  enabled: true
  strategy: "linear"
  min_workers: 5
  max_workers: 25
  min_connections: 10
  max_connections: 50
  test_duration: "3m"               # 3 minutes per band
  warmup_duration: "30s"            # 30-second warmup
  cooldown_duration: "15s"          # 15-second cooldown
  bands: 5                          # 5 configurations
  export_csv: true
  export_json: false                # Skip JSON for speed
  enable_analysis: true
```

**High-Precision Research (6-hour scientific)**
```yaml
progressive:
  enabled: true
  strategy: "fibonacci"
  min_workers: 10
  max_workers: 150
  min_connections: 25
  max_connections: 300
  test_duration: "45m"              # 45 minutes per band
  warmup_duration: "180s"           # 3-minute warmup
  cooldown_duration: "120s"         # 2-minute cooldown
  bands: 12                         # 12 configurations
  export_csv: true
  export_json: true
  enable_analysis: true
```

## Mathematical Analysis Features v0.2

Progressive scaling v0.2 includes enhanced mathematical analysis with NaN/Inf protection and scientific rigor:

### Statistical Metrics (Per Band)

- **Core Performance**: TPS, QPS, latency (avg, P50, P95, P99)
- **Variability Analysis**: Standard deviation, variance, coefficient of variation
- **Confidence Intervals**: 95% confidence around mean latency
- **Efficiency Metrics**: TPS per worker, connection utilization rates
- **Cost Analysis**: Performance per resource unit

### Advanced Analysis (New in v0.2)

#### 1. Marginal Gains Analysis (Enhanced Discrete Derivatives)
Calculates the performance gain per additional worker/connection with NaN protection:

```
ΔTPS/ΔWorkers = (TPS₂ - TPS₁) / (Workers₂ - Workers₁)
```

**v0.2 Improvements**:
- NaN/Inf sanitization for all calculations
- Protected division by zero
- Enhanced marginal efficiency analysis
- Real-time marginal gains tracking

#### 2. Inflection Point Detection (Second Derivatives)
Identifies critical scaling points where adding resources becomes counterproductive:

```
d²TPS/dWorkers² = (Δ₂TPS - Δ₁TPS) / ΔWorkers
```

**Interpretation**:
- **Positive**: Accelerating returns (good scaling)
- **Negative**: Diminishing returns (approaching saturation)
- **Zero**: Inflection point (optimal scaling region)

#### 3. Enhanced Curve Fitting
Fits mathematical models to predict performance at untested configurations:

- **Linear Model**: `TPS = a × Workers + b`
- **Logarithmic Model**: `TPS = a × log(Workers) + b`
- **Exponential Model**: `TPS = a × e^(b × Workers)`
- **Logistic Model**: `TPS = L / (1 + e^(-k(Workers-x₀)))`

**v0.2 Improvements**:
- R² correlation coefficients for model quality
- NaN-safe mathematical operations
- Enhanced model selection algorithms
- Predictive confidence intervals

#### 4. Queueing Theory Analysis (M/M/c Modeling)
Applies Little's Law and queueing theory to classify system bottlenecks:

```
Utilization = ArrivalRate / (ServiceRate × Servers)
QueueLength = λ × W (Little's Law)
```

**Classifications**:
- **CPU-bound**: `ρ > 0.8`, low queue variance
- **I/O-bound**: High latency variance, moderate utilization
- **Memory-bound**: Degrading performance despite low CPU
- **Queue-bound**: High queue lengths, variable service times

#### 5. Performance Region Classification
Automatically categorizes scaling behavior:

- **Linear Scaling**: Consistent marginal gains
- **Diminishing Returns**: Decreasing marginal gains
- **Saturation**: Near-zero marginal gains
- **Degradation**: Negative marginal gains

### Bottleneck Identification System

Progressive scaling v0.2 includes enhanced bottleneck detection:

```go
type BottleneckAnalysis struct {
    PrimaryBottleneck   string   // CPU, Memory, I/O, Network, Queue
    ConfidenceLevel     float64  // 0.0 - 1.0 confidence in classification
    UtilizationMetrics  map[string]float64
    Recommendations     []string
    OptimalConfiguration Band
}
```

**Detection Algorithms**:
1. **CPU Analysis**: CPU utilization, context switches, load average
2. **Memory Analysis**: Memory usage patterns, GC frequency
3. **I/O Analysis**: Disk I/O patterns, wait times
4. **Network Analysis**: Network latency, bandwidth utilization
5. **Queue Analysis**: Connection pool utilization, lock contention

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

## Output & Results v0.2

### Real-time Console Output
```
🎯 Starting Progressive Scaling v0.2 Analysis
📊 Strategy: exponential | Test Duration: 30m | Bands: 6
📈 Range: 10→60 workers, 20→120 connections

═══════════════════════════════════════════════════════════════════
🔄 Band 1/6: 10 workers, 20 connections
🔥 Warming up for 60s...
📊 Running test for 30m00s...
✅ Band 1 Complete: 1,234 TPS | 45.2ms avg | 62.1ms P95
   📈 Marginal Gain: N/A (baseline)
   🎯 Efficiency: 123.4 TPS/worker

🔄 Band 2/6: 14 workers, 28 connections  
🔥 Warming up for 60s...
📊 Running test for 30m00s...
✅ Band 2 Complete: 1,678 TPS | 41.8ms avg | 58.3ms P95
   📈 Marginal Gain: +111.0 TPS/worker (excellent scaling)
   🎯 Efficiency: 119.9 TPS/worker

🔄 Band 3/6: 20 workers, 40 connections
📊 Band 3 Complete: 2,156 TPS | 39.4ms avg | 55.7ms P95
   📈 Marginal Gain: +79.7 TPS/worker (good scaling)
   🎯 Efficiency: 107.8 TPS/worker

🔄 Band 4/6: 28 workers, 56 connections
📊 Band 4 Complete: 2,534 TPS | 42.1ms avg | 61.2ms P95
   📈 Marginal Gain: +47.3 TPS/worker (diminishing returns)
   🎯 Efficiency: 90.5 TPS/worker
   ⚠️  Potential bottleneck detected: I/O bound

🔄 Band 5/6: 40 workers, 80 connections
📊 Band 5 Complete: 2,623 TPS | 48.9ms avg | 69.4ms P95
   📈 Marginal Gain: +7.4 TPS/worker (approaching saturation)
   🎯 Efficiency: 65.6 TPS/worker

🔄 Band 6/6: 60 workers, 120 connections
📊 Band 6 Complete: 2,591 TPS | 54.2ms avg | 78.1ms P95
   📈 Marginal Gain: -1.6 TPS/worker (performance degradation)
   🎯 Efficiency: 43.2 TPS/worker
   🚨 Degradation detected: over-provisioned

═══════════════════════════════════════════════════════════════════
✅ Progressive Scaling Analysis Complete

🏆 OPTIMAL CONFIGURATION DETECTED:
   Workers: 28 | Connections: 56 | Performance: 2,534 TPS
   Efficiency: 90.5 TPS/worker | Avg Latency: 42.1ms

📊 SCALING ANALYSIS:
   • Linear Scaling Region: Bands 1-2 (excellent performance gains)
   • Diminishing Returns: Bands 3-4 (moderate gains, I/O bottleneck)
   • Saturation Point: Band 5 (minimal gains)
   • Degradation Region: Band 6 (performance loss)

🔬 MATHEMATICAL MODELS:
   • Best Fit: Logistic Model (R²=0.94)
   • Predicted Peak: 2,640 TPS at 32 workers
   • Saturation Threshold: 35 workers

🎯 BOTTLENECK CLASSIFICATION:
   Primary: I/O-bound (confidence: 87%)
   Secondary: Connection pool contention
   
📂 Results exported to:
   • progressive_scaling_results_20241215_143022.csv
   • progressive_scaling_results_20241215_143022.json
```

### CSV Export Format v0.2
```csv
band,workers,connections,duration_seconds,tps,avg_latency_ms,p50_latency_ms,p95_latency_ms,p99_latency_ms,std_dev_latency,errors,efficiency_tps_per_worker,marginal_gain_tps_per_worker,scaling_region,bottleneck_type
1,10,20,1800,1234.5,45.2,43.1,62.1,89.4,12.3,0,123.45,0.00,baseline,none
2,14,28,1800,1678.3,41.8,39.7,58.3,84.2,11.1,0,119.88,111.00,linear,none
3,20,40,1800,2156.7,39.4,37.2,55.7,79.8,10.8,0,107.84,79.70,linear,none
4,28,56,1800,2534.2,42.1,40.3,61.2,87.5,13.2,0,90.51,47.25,diminishing,io_bound
5,40,80,1800,2623.1,48.9,46.8,69.4,95.3,15.7,2,65.58,7.43,saturation,io_bound
6,60,120,1800,2591.4,54.2,52.1,78.1,108.7,18.9,8,43.19,-1.59,degradation,over_provisioned
```

### JSON Export Format v0.2
```json
{
  "metadata": {
    "test_name": "progressive_scaling_analysis",
    "version": "v0.2-alpha",
    "timestamp": "2024-12-15T14:30:22Z",
    "duration_total_seconds": 11700,
    "strategy": "exponential",
    "bands_tested": 6,
    "workload_type": "imdb_plugin"
  },
  "configuration": {
    "min_workers": 10,
    "max_workers": 60,
    "min_connections": 20,
    "max_connections": 120,
    "test_duration": "30m",
    "warmup_duration": "60s",
    "cooldown_duration": "30s"
  },
  "bands": [
    {
      "band_number": 1,
      "workers": 10,
      "connections": 20,
      "duration_seconds": 1800,
      "performance": {
        "tps": 1234.5,
        "avg_latency_ms": 45.2,
        "p50_latency_ms": 43.1,
        "p95_latency_ms": 62.1,
        "p99_latency_ms": 89.4,
        "std_dev_latency": 12.3,
        "errors": 0
      },
      "analysis": {
        "efficiency_tps_per_worker": 123.45,
        "marginal_gain_tps_per_worker": 0.0,
        "scaling_region": "baseline",
        "bottleneck_type": "none"
      }
    }
  ],
  "analysis": {
    "optimal_configuration": {
      "workers": 28,
      "connections": 56,
      "tps": 2534.2,
      "efficiency": 90.51,
      "confidence": 0.94
    },
    "mathematical_models": {
      "best_fit": "logistic",
      "r_squared": 0.94,
      "predicted_peak_tps": 2640.0,
      "predicted_peak_workers": 32
    },
    "bottleneck_analysis": {
      "primary_bottleneck": "io_bound",
      "confidence": 0.87,
      "secondary_factors": ["connection_pool_contention"],
      "recommendations": [
        "Optimize I/O subsystem",
        "Consider connection pooling tuning",
        "Monitor disk utilization"
      ]
    },
    "scaling_regions": {
      "linear_scaling": {"bands": [1, 2], "description": "Excellent scaling"},
      "diminishing_returns": {"bands": [3, 4], "description": "Moderate gains"},
      "saturation": {"bands": [5], "description": "Minimal gains"},
      "degradation": {"bands": [6], "description": "Performance loss"}
    }
  }
}
```
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
