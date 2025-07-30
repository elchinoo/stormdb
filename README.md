# StormDB - PostgreSQL Performance Testing Tool

[![CI/CD Pipeline](https://github.com/elchinoo/stormdb/actions/workflows/ci.yml/badge.svg)](https://github.com/elchinoo/stormdb/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/elchinoo/stormdb/branch/main/graph/badge.svg)](https://codecov.io/gh/elchinoo/stormdb)
[![Go Report Card](https://goreportcard.com/badge/github.com/elchinoo/stormdb)](https://goreportcard.com/report/github.com/elchinoo/stormdb)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/release/elchinoo/stormdb.svg)](https://github.com/elchinoo/stormdb/releases/latest)

StormDB is a comprehensive PostgreSQL benchmarking and load testing tool designed to help you understand your database performance characteristics. It features a modern plugin architecture that provides multiple workload types, detailed metrics analysis, and advanced monitoring capabilities.

## ✨ Key Features

### 🔌 Extensible Plugin Architecture
- **Dynamic Loading**: Load workload plugins at runtime without recompilation
- **Built-in Workloads**: Core workloads (TPCC, Simple, Connection Overhead) built into the binary
- **Plugin Discovery**: Automatic scanning and loading of plugin files (.so, .dll, .dylib)
- **Extensible System**: Easy development of custom workloads via plugin interface
- **Metadata Support**: Rich plugin metadata with version info and compatibility

### 🚀 Comprehensive Workload Types
- **TPC-C**: Industry-standard OLTP benchmark with realistic transaction processing
- **Simple/Mixed**: Basic read/write operations for quick testing and baseline performance
- **Connection Overhead**: Compare persistent vs transient connection performance

#### Plugin Workloads (Dynamically Loaded)
- **IMDB**: Movie database workload with complex queries and realistic data patterns
- **Vector Operations**: High-dimensional vector similarity search testing (requires pgvector)
- **E-commerce**: Modern retail platform with inventory, orders, and analytics
- **Real-world**: Enterprise application workloads with business logic patterns

### 📊 Advanced Metrics & Analysis
- **Transaction Performance**: TPS, latency percentiles, success rates
- **Query Analysis**: Breakdown by type (SELECT, INSERT, UPDATE, DELETE)
- **Latency Distribution**: P50, P95, P99 with histogram visualization
- **Worker-level Metrics**: Per-thread performance tracking
- **Time-series Data**: Performance over time with configurable intervals
- **Error Tracking**: Detailed error classification and reporting

### 🔍 PostgreSQL Deep Monitoring
- **Buffer Cache Statistics**: Hit ratios, blocks read/written
- **WAL Activity**: WAL records, bytes generated
- **Checkpoint Monitoring**: Requested vs timed checkpoints
- **Connection Tracking**: Active connections vs limits
- **pg_stat_statements**: Top queries by execution time (optional)
- **Lock Contention**: Deadlock and wait event tracking
- **Autovacuum Activity**: Monitoring background maintenance

### 🎯 Progressive Connection Scaling
- **Automated Scaling**: Test multiple worker/connection configurations in a single run
- **Mathematical Analysis**: Advanced statistical analysis with discrete derivatives and inflection points
- **Curve Fitting**: Linear, logarithmic, exponential, and logistic model fitting
- **Queueing Theory**: M/M/c queue modeling for bottleneck identification
- **Scaling Strategies**: Linear, exponential, and fibonacci scaling patterns
- **Optimal Configuration Discovery**: Automatically identifies best performance configurations
- **Export Options**: CSV and JSON export for further analysis and visualization

## 🏗️ Architecture

StormDB uses a modular plugin architecture that separates core functionality from workload implementations:

```
┌─────────────────────────────────────────────────────────────────┐
│                           StormDB Core                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │   Config Mgmt   │  │   Metrics & DB  │  │  Signal Handler │  │
│  │   • YAML Load   │  │   • PostgreSQL  │  │  • Graceful     │  │
│  │   • Validation  │  │   • Statistics  │  │    Shutdown     │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                      Workload Factory                           │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │  Built-in       │  │   Plugin        │  │   Dynamic       │  │
│  │  Workloads      │  │   Discovery     │  │   Loading       │  │
│  │  • TPCC         │  │   • Auto-scan   │  │   • Go plugins  │  │
│  │  • Simple       │  │   • Metadata    │  │   • .so/.dll    │  │
│  │  • Connection   │  │   • Validation  │  │   • Runtime     │  │
│  │    Overhead     │  │                 │  │     Loading     │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                        Plugin System                            │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │  IMDB Plugin    │  │  Vector Plugin  │  │ E-commerce      │  │
│  │  • Movie DB     │  │  • pgvector     │  │  Plugin         │  │
│  │  • Complex      │  │  • Similarity   │  │  • Orders       │  │
│  │    Queries      │  │    Search       │  │  • Inventory    │  │
│  │  • Analytics    │  │  • High-dim     │  │  • Analytics    │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │ E-commerce Basic│  │   Custom        │  │    Future       │  │
│  │  Plugin         │  │   Plugins       │  │   Plugins       │  │
│  │  • Enterprise   │  │  • User-defined │  │  • Community    │  │
│  │  • OLTP/OLAP    │  │  • Specific     │  │  • Extensions   │  │
│  │  • Business     │  │    Use cases    │  │                 │  │
│  │    Logic        │  │                 │  │                 │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Key Architecture Benefits

- **🔌 Plugin Architecture**: Extensible workload system with dynamic loading
- **🏗️ Modular Design**: Clear separation between core engine and workload logic  
- **🚀 Performance**: Efficient Go-based implementation with plugin hot-loading
- **🔧 Extensibility**: Easy to add custom workloads without modifying core code
- **📦 Distribution**: Plugins can be distributed and installed independently

For a detailed architectural overview, see [ARCHITECTURE.md](ARCHITECTURE.md).

## 🚀 Installation & Quick Start

### Prerequisites

- **Go 1.24+** for building from source
- **PostgreSQL 12+** with test database
- **Build tools**: `make`, `git`
- **Optional**: Docker for containerized usage

### Method 1: Download Pre-built Binaries

```bash
# Download latest release (Linux/macOS/Windows)
curl -L https://github.com/elchinoo/stormdb/releases/latest/download/stormdb-linux-amd64 -o stormdb
chmod +x stormdb

# Or use wget
wget https://github.com/elchinoo/stormdb/releases/latest/download/stormdb-linux-amd64 -O stormdb
```

### Method 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/elchinoo/stormdb.git
cd stormdb

# Install development tools (optional but recommended)
make dev-tools

# Build everything (binary + plugins)
make build-all

# Or build just the binary
make build

# Install to system PATH (optional)
make install
```

### Method 3: Docker

```bash
# Run with Docker
docker run --rm elchinoo/stormdb:latest --help

# Or use docker-compose for complete setup
docker-compose up -d postgres  # Start test database
docker-compose run stormdb --config /app/config/config_simple_connection.yaml
```

### Quick Test Run

```bash
# 1. Create a test configuration
cat > config_test.yaml << EOF
database:
  type: postgres
  host: localhost
  port: 5432
  dbname: postgres
  username: postgres
  password: yourpassword
  sslmode: disable

workload: simple
duration: 30s
workers: 4
connections: 8
summary_interval: 5s
EOF

# 2. Run a simple benchmark
./stormdb --config config_test.yaml

# 3. With PostgreSQL monitoring
./stormdb --config config_test.yaml --collect-pg-stats
```

## 📖 Usage Guide

### Building and Development

```bash
# Development workflow
make dev-tools          # Install development tools
make deps               # Update dependencies  
make dev-watch          # Watch for changes and rebuild
make pre-commit         # Run before committing

# Testing
make test               # Fast unit tests
make test-all           # Complete test suite
make test-coverage      # Generate coverage report
make validate-full      # Full validation (lint, security, tests)

# Plugin development
make plugins            # Build all plugins
make plugins-test       # Test all plugins
make list-plugins       # List available plugins
```

### Configuration Examples

#### Built-in Workloads (No plugins required)

```bash
# TPC-C benchmark
./stormdb --config config/config_tpcc.yaml --setup

# Simple read/write workload  
./stormdb --config config/config_simple_connection.yaml

# Connection overhead testing
./stormdb --config config/config_transient_connections.yaml
```

#### Plugin Workloads (Requires building plugins)

```bash
# IMDB complex queries
./stormdb --config config/config_imdb_mixed.yaml --rebuild

# Vector similarity search (requires pgvector)
./stormdb --config config/config_vector_cosine.yaml --setup

# E-commerce workload
./stormdb --config config/config_ecommerce_mixed.yaml --rebuild
```

## 🔧 Plugin System

StormDB's plugin system allows you to extend functionality without modifying core code. Plugins are dynamically loaded Go shared libraries.

### Plugin Configuration

Configure the plugin system in your YAML config:

```yaml
# Plugin system configuration
plugins:
  # Directories to search for plugin files (.so, .dll, .dylib)
  paths:
    - "./plugins"           # Development plugins
    - "./build/plugins"     # Built plugins
    - "/usr/local/lib/stormdb/plugins"  # System-wide plugins
  
  # Specific plugin files to load (optional)
  files:
    - "./build/plugins/imdb_plugin.so"
  
  # Automatically load all plugins found in search paths
  auto_load: true
```

### Available Plugins

| Plugin | Description | Workload Types |
|--------|-------------|----------------|
| **IMDB** | Movie database with complex queries | `imdb_read`, `imdb_write`, `imdb_mixed`, `imdb_sql` |
| **Vector** | High-dimensional vector operations | `pgvector_*`, `vector_cosine`, `vector_inner` |
| **E-commerce** | Retail platform simulation | `ecommerce_read`, `ecommerce_write`, `ecommerce_mixed` |
| **E-commerce Basic** | Basic e-commerce patterns | `ecommerce_basic` |

### Building Plugins

```bash
# Build all available plugins
make plugins

# Build specific plugin
cd plugins/imdb_plugin && go build -buildmode=plugin -o ../../build/plugins/imdb_plugin.so *.go

# Test plugins
make plugins-test

# Install system-wide
make plugins-install
```

### Plugin Development

Create custom plugins by implementing the `WorkloadPlugin` interface:

```go
package main

import "stormdb/pkg/plugin"

type MyPlugin struct{}

func (p *MyPlugin) GetMetadata() plugin.Metadata {
    return plugin.Metadata{
        Name:         "my_plugin",
        Version:      "1.0.0",
        Description:  "My custom workload plugin",
        WorkloadTypes: []string{"my_workload", "my_workload_read"},
    }
}

func (p *MyPlugin) CreateWorkload(workloadType string) (interface{}, error) {
    // Return your workload implementation
}

func (p *MyPlugin) Initialize() error   { return nil }
func (p *MyPlugin) Cleanup() error     { return nil }

// Plugin entry point
var Plugin MyPlugin
```

For detailed plugin development, see [`docs/PLUGIN_DEVELOPMENT.md`](docs/PLUGIN_DEVELOPMENT.md).

## 🐳 Docker Usage

### Pre-built Images

```bash
# Pull latest image
docker pull elchinoo/stormdb:latest

# Run with local config
docker run --rm -v $(pwd)/config:/app/config elchinoo/stormdb:latest \
  --config /app/config/config_simple_connection.yaml

# Interactive mode
docker run -it --rm elchinoo/stormdb:latest /bin/sh
```

### Development Setup

```bash
# Start PostgreSQL test database
docker-compose up -d postgres

# Run benchmarks
docker-compose run stormdb --config /app/config/config_simple_connection.yaml

# With monitoring stack
docker-compose --profile monitoring up -d
# Access Grafana at http://localhost:3000 (admin/admin)
```

### Custom Builds

```bash
# Build custom image
docker build -t my-stormdb .

# Multi-platform build
docker buildx build --platform linux/amd64,linux/arm64 -t my-stormdb .
```

## 📊 Performance Optimization

### Database Tuning Recommendations

```sql
-- Essential PostgreSQL settings for benchmarking
ALTER SYSTEM SET shared_buffers = '25% of RAM';
ALTER SYSTEM SET effective_cache_size = '75% of RAM';
ALTER SYSTEM SET maintenance_work_mem = '1GB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
SELECT pg_reload_conf();
```

### Connection Pool Optimization

```yaml
# Optimal connection pool settings
database:
  max_connections: 100      # Should be <= PostgreSQL max_connections
  max_idle_connections: 25  # ~25% of max_connections
  connection_max_lifetime: "1h"
  connection_max_idle_time: "15m"
```

### Monitoring Best Practices

```bash
# Enable comprehensive monitoring
./stormdb --config config.yaml \
  --collect-pg-stats \
  --pg-stat-statements \
  --summary-interval 10s

# Profile application performance
make profile-all
go tool pprof profiles/cpu.prof
```

## 🔍 Troubleshooting

### Common Issues

#### Connection Problems
```bash
# Test database connectivity
psql -h localhost -U postgres -d testdb -c "SELECT version();"

# Check connection limits
SELECT count(*) as current_connections, 
       setting::int as max_connections 
FROM pg_stat_activity, pg_settings 
WHERE name = 'max_connections';
```

#### Plugin Loading Errors
```bash
# Verify plugin exists and has correct permissions
ls -la build/plugins/
ldd build/plugins/imdb_plugin.so  # Check dependencies

# Debug plugin loading
GODEBUG=cgocheck=2 ./stormdb --config config.yaml
```

#### Performance Issues
```bash
# Check system resources
htop
iostat -x 1
sar -u 1

# Database performance
SELECT * FROM pg_stat_activity WHERE state = 'active';
SELECT * FROM pg_stat_database WHERE datname = 'your_db';
```

### Debug Mode

```bash
# Enable debug logging
./stormdb --config config.yaml --log-level debug

# Environment variables for debugging
export STORMDB_DEBUG=1
export STORMDB_TRACE_SQL=1
export GODEBUG=gctrace=1
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

### Development Workflow

```bash
# Setup development environment
git clone https://github.com/elchinoo/stormdb.git
cd stormdb
make dev-tools deps

# Create feature branch
git checkout -b feature/awesome-feature

# Make changes and test
make pre-commit
make test-all

# Submit pull request
git push origin feature/awesome-feature
```

### Code Quality Standards

- **Go Code**: Follow `gofmt`, `golint`, and `go vet` recommendations
- **Tests**: Maintain >80% test coverage
- **Documentation**: Update relevant docs for any changes
- **Security**: Run `make security` and `make vuln-check`

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- PostgreSQL community for the excellent database system
- Go team for the robust programming language and tooling
- Contributors and users who help improve StormDB
- Inspired by industry-standard benchmarking tools

## 📞 Support

- **Documentation**: [docs/](docs/)
- **GitHub Issues**: [Issues](https://github.com/elchinoo/stormdb/issues)
- **Discussions**: [GitHub Discussions](https://github.com/elchinoo/stormdb/discussions)
- **Security Issues**: See [SECURITY.md](SECURITY.md)

---

**Made with ❤️ by the StormDB team**

## Features

### � Plugin Architecture
- **Dynamic Loading**: Load workload plugins at runtime without recompilation
- **Built-in Workloads**: Core workloads (TPCC, Simple, Connection Overhead) built into the binary
- **Plugin Discovery**: Automatic scanning and loading of plugin files (.so, .dll, .dylib)
- **Extensible System**: Easy development of custom workloads via plugin interface
- **Metadata Support**: Rich plugin metadata with version info and compatibility

### �🚀 Workload Types
- **TPC-C**: Industry-standard OLTP benchmark with realistic transaction processing
- **Simple/Mixed**: Basic read/write operations for quick testing and baseline performance
- **Connection Overhead**: Compare persistent vs transient connection performance

#### Plugin Workloads (Dynamically Loaded)
- **IMDB**: Movie database workload with complex queries and realistic data patterns
- **Vector Operations**: High-dimensional vector similarity search testing (requires pgvector)
- **E-commerce**: Modern retail platform with inventory, orders, and analytics
- **Real-world**: Enterprise application workloads with business logic patterns

### 📊 Advanced Metrics & Analysis
- **Transaction Performance**: TPS, latency percentiles, success rates
- **Query Analysis**: Breakdown by type (SELECT, INSERT, UPDATE, DELETE)
- **Latency Distribution**: P50, P95, P99 with histogram visualization
- **Worker-level Metrics**: Per-thread performance tracking
- **Time-series Data**: Performance over time with configurable intervals
- **Error Tracking**: Detailed error classification and reporting

### 🔍 PostgreSQL Deep Monitoring
- **Buffer Cache Statistics**: Hit ratios, blocks read/written
- **WAL Activity**: WAL records, bytes generated
- **Checkpoint Monitoring**: Requested vs timed checkpoints
- **Connection Tracking**: Active connections vs limits
- **pg_stat_statements**: Top queries by execution time (optional)
- **Lock Contention**: Deadlock and wait event tracking
- **Autovacuum Activity**: Monitoring background maintenance

### 🔗 Connection Mode Testing
- **Persistent Connections**: Use connection pooling (default)
- **Transient Connections**: Create new connection per operation
- **Mixed Mode**: 50/50 split for overhead comparison
- **Overhead Analysis**: Detailed comparison of connection strategies

## Quick Start

### Building StormDB

```bash
# Build the main binary only
make build

# Build main binary + all plugins  
make build-all

# Build specific plugins
make plugins
```

### Basic Usage

```bash
# Run a built-in TPC-C benchmark
./stormdb --config config/config_tpcc.yaml

# Run plugin workloads (requires plugins to be built)
./stormdb --config config/config_imdb_mixed.yaml --collect-pg-stats
./stormdb --config config/config_vector_cosine.yaml
./stormdb --config config/config_ecommerce_mixed.yaml

# Test connection overhead (built-in)
./stormdb --config config/config_connection_overhead.yaml
```

### Setup Options

```bash
# Set up schema only (no data loading)
./stormdb --config config/config_tpcc.yaml --setup

# Full rebuild (drop, recreate, load data) - works with plugins
./stormdb --config config/config_imdb_mixed.yaml --rebuild
```

## Configuration

### Basic Configuration Structure

```yaml
database:
  type: postgres
  host: localhost
  port: 5432
  dbname: your_database
  username: your_user
  password: your_password
  sslmode: disable

workload: tpcc                    # Workload type
scale: 10                         # Scale factor (workload-dependent)
duration: "60s"                   # Test duration
workers: 8                        # Concurrent worker threads
connections: 16                   # Max connections in pool
summary_interval: "10s"           # Progress reporting interval

# PostgreSQL monitoring options
collect_pg_stats: true            # Enable PostgreSQL statistics collection
pg_stats_statements: true         # Enable pg_stat_statements (requires extension)

# Connection management options
connection_mode: "persistent"     # "persistent", "transient", or "mixed"
```

### Workload-Specific Options

#### IMDB Workload (Plugin)
```yaml
plugins:
  paths: ["./build/plugins"]
  files: ["./build/plugins/imdb_plugin.so"]
  auto_load: true

workload: "imdb_mixed"           # imdb_read, imdb_write, imdb_mixed
scale: 5000                      # Number of movies to generate

# Data loading options
data_loading:
  mode: "generate"               # generate, dump, sql
  filepath: "/path/to/data.sql"  # Required for dump/sql modes
```

#### Vector Workload (Plugin)
```yaml
plugins:
  paths: ["./build/plugins"]
  files: ["./build/plugins/vector_plugin.so"]
  auto_load: true

workload: "vector_1024_cosine"   # vector_1024, vector_1024_cosine, vector_1024_inner
scale: 10000                     # Number of vectors to generate
```

#### TPC-C Workload
```yaml
workload: "tpcc"
scale: 5                         # Number of warehouses
                                # Each warehouse = 10 districts, ~30K customers
```

## Available Workloads

### Built-in Workloads

| Workload | Description | Scale Meaning | Best For |
|----------|-------------|---------------|----------|
| `tpcc` | TPC-C OLTP benchmark | # of warehouses | Standard OLTP testing |
| `connection` | Connection mode comparison | N/A | Connection analysis |
| `simple` | Basic read/write ops | # of transactions | Quick testing |

### Plugin Workloads

StormDB now supports a plugin architecture for extended workloads. The following workloads are available as plugins:

| Plugin | Workloads | Description | Requirements |
|--------|-----------|-------------|---------------|
| **IMDB Plugin** | `imdb_read`, `imdb_write`, `imdb_mixed` | Movie database workloads with complex queries | PostgreSQL 12+ |
| **Vector Plugin** | `vector_1024`, `vector_1024_cosine`, `vector_1024_inner` | High-dimensional vector similarity search | pgvector extension |
| **E-commerce Basic Plugin** | `ecommerce_basic`, `ecommerce_basic_read`, `ecommerce_basic_write`, `ecommerce_basic_mixed`, `ecommerce_basic_oltp`, `ecommerce_basic_analytics` | Basic e-commerce patterns | PostgreSQL 12+ |
| **E-commerce Plugin** | `ecommerce`, `ecommerce_read`, `ecommerce_write`, `ecommerce_mixed`, `ecommerce_oltp`, `ecommerce_analytics` | Modern e-commerce platform simulation | pgvector extension |

### Building and Using Plugins

To build all plugins:
```bash
make plugins
# or specifically:
make build-all  # Builds both binary and plugins
```

To use plugin workloads, ensure they're specified in your configuration:
```yaml
plugins:
  paths:
    - "./build/plugins"
  auto_load: true

workload: "imdb_mixed"  # Plugin workload
```

## Monitoring & Analysis

### PostgreSQL Statistics Collection

Enable comprehensive PostgreSQL monitoring:

```bash
./stormdb --config your_config.yaml --collect-pg-stats --pg-stat-statements
```

This provides:
- **Buffer cache performance** (hit ratios, disk I/O)
- **WAL activity** (records, bytes generated)
- **Checkpoint behavior** (frequency, triggers)
- **Connection utilization** (active vs max)
- **Top queries** (execution time, frequency)
- **Lock contention** (deadlocks, waits)

### Connection Overhead Analysis

Test the impact of connection management strategies:

```yaml
workload: connection
connection_mode: mixed           # Tests both persistent and transient
```

Results show:
- **Transaction rate differences** between connection modes
- **Latency overhead** of connection establishment
- **Setup time per connection** for transient mode
- **Total connections created** during test

### Time-Series Metrics

All workloads provide time-series analysis:
- Performance over time buckets
- Latency evolution during test
- Query rate fluctuations
- Error rate tracking

## Command Line Options

```bash
./stormdb [flags]

Core Options:
  -c, --config string           Path to config file (default "config.yaml")
  -d, --duration string         Test duration, e.g., 30s, 1m (overrides config)
  -w, --workload string         Workload type (overrides config)
      --workers int             Number of worker threads (overrides config)
      --connections int         Max connections in pool (overrides config)

Database Options:
      --host string             Database host (overrides config)
      --port int                Database port (overrides config)
      --dbname string           Database name (overrides config)
  -u, --username string         Database username (overrides config)
  -p, --password string         Database password (overrides config)

Setup Options:
      --setup                   Ensure schema exists (create if needed)
  -r, --rebuild                 Rebuild: drop, recreate schema, and load data
      --scale int               Scale factor (overrides config)

Monitoring Options:
      --collect-pg-stats        Enable PostgreSQL statistics collection
      --pg-stat-statements      Enable pg_stat_statements collection
  -s, --summary-interval string Periodic summary interval (overrides config)
      --no-summary              Disable periodic summary reporting
```

## Example Workflows

### Performance Baseline Testing

```bash
# 1. Set up the database schema
./stormdb --config config/config_tpcc.yaml --setup

# 2. Run a comprehensive test with monitoring
./stormdb --config config/config_tpcc.yaml --collect-pg-stats --pg-stat-statements

# 3. Run connection analysis
./stormdb --config config/config_connection_overhead.yaml
```

### Workload Comparison

```bash
# Compare different workload types
./stormdb --workload tpcc --duration 60s --workers 8
./stormdb --workload imdb_mixed --duration 60s --workers 8
./stormdb --workload vector_1024_cosine --duration 60s --workers 8
```

### Scale Testing

```bash
# Test different scales
./stormdb --config config/config_tpcc.yaml --scale 1 --duration 30s
./stormdb --config config/config_tpcc.yaml --scale 5 --duration 30s
./stormdb --config config/config_tpcc.yaml --scale 10 --duration 30s
```

## Output Interpretation

### Key Metrics to Watch

1. **TPS (Transactions Per Second)**: Overall throughput
2. **Success Rate**: Percentage of successful transactions
3. **P95 Latency**: 95th percentile response time
4. **Buffer Cache Hit Ratio**: Should be >95% for good performance
5. **Connection Utilization**: Active vs max connections

### Performance Indicators

- **High TPS + Low Latency**: Excellent performance
- **Low Success Rate**: Database overload or configuration issues
- **High P99 Latency**: Potential outliers or resource contention
- **Low Buffer Hit Ratio**: May need more memory or I/O optimization
- **High Connection Usage**: May need connection pool tuning

## Troubleshooting

### Common Issues

1. **"role does not exist"**: Update database credentials in config
2. **"database does not exist"**: Create database or update config
3. **High error rates**: Check database capacity and configuration
4. **Low performance**: Consider increasing connections, workers, or database resources

### pg_stat_statements Setup

To enable query analysis:

```sql
-- Add to postgresql.conf
shared_preload_libraries = 'pg_stat_statements'
pg_stat_statements.track = all

-- Restart PostgreSQL, then:
CREATE EXTENSION pg_stat_statements;
```

## Configuration Examples

See the `config/` directory for example configurations:

- `config_tpcc.yaml` - TPC-C benchmark
- `config_imdb_mixed.yaml` - IMDB mixed workload
- `config_connection_overhead.yaml` - Connection testing
- `config_vector_cosine.yaml` - Vector similarity testing

## Contributing

StormDB is designed to be extensible. You can add new workloads by implementing the `Workload` interface in `internal/workload/`.

## License

This project is open source. Feel free to use, modify, and distribute according to your needs.
- **Advanced PostgreSQL features** including pgvector, JSONB, and complex queries
- **Extensible architecture** for custom workload development
- **Comprehensive metrics** and performance analysis
- **Production-ready scenarios** with automated systems and realistic data

### Why stormdb?

Modern applications demand sophisticated database testing that traditional tools can't provide:

- **E-commerce platforms** need complex inventory management, automated reordering, and customer analytics
- **Vector databases** require semantic search capabilities and similarity matching
- **OLTP/OLAP workloads** need mixed transaction patterns with realistic data relationships
- **Performance optimization** requires detailed metrics and comprehensive reporting

stormdb fills this gap by providing production-realistic workloads with advanced PostgreSQL integration.

## 🏗️ Architecture Overview

### Core Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         stormdb                             │
├─────────────────┬─────────────────┬─────────────────────────┤
│   CLI Interface │  Configuration  │    Metrics Engine       │
│                 │     Manager     │                         │
└─────────────────┼─────────────────┼─────────────────────────┘
                  │                 │
┌─────────────────▼─────────────────▼─────────────────────────┐
│                  Workload Factory                           │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────┐ │
│  │  E-Commerce │ │    IMDB     │ │   Vector    │ │  TPC-C  │ │
│  │   Workload  │ │  Workload   │ │  Workload   │ │Workload │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼─────────────────────────────────┐
│                 Database Layer                                │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────┐ │
│  │ Connection  │ │ Transaction │ │    Query    │ │ Metrics │ │
│  │   Pool      │ │  Manager    │ │  Executor   │ │Collector│ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │
                    ┌────────▼────────┐
                    │   PostgreSQL    │
                    │    Database     │
                    └─────────────────┘
```

### Design Principles

1. **Extensibility**: Plugin-based workload architecture
2. **Performance**: Optimized connection pooling and concurrent execution
3. **Realism**: Production-like scenarios with complex data relationships
4. **Observability**: Comprehensive metrics and performance monitoring
5. **Reliability**: Robust error handling and graceful degradation

### Component Architecture

#### 1. Workload Interface
All workloads implement a common interface providing:
- **Setup()**: Schema creation and data loading
- **Run()**: Load test execution
- **Cleanup()**: Resource cleanup and teardown

#### 2. Configuration System
- **YAML-based configuration** with validation
- **Environment variable support** for sensitive data
- **Profile-based configurations** for different scenarios
- **Hot-reload capabilities** for dynamic adjustments

#### 3. Metrics Engine
- **Real-time metrics collection** with configurable intervals
- **Histogram-based latency tracking** with custom buckets
- **Error categorization** and reporting
- **Performance analytics** and trend analysis

#### 4. Database Layer
- **Connection pooling** with pgx/v5 for optimal performance
- **Transaction management** with proper isolation levels
- **Query optimization** and execution planning
- **Resource monitoring** and leak detection

## 🚀 Methodologies and Approach

### Load Testing Methodology

stormdb employs a **multi-faceted approach** to database load testing:

#### 1. Realistic Workload Simulation
- **Production-like data patterns** with realistic relationships
- **Mixed operation types** (OLTP, OLAP, batch processing)
- **Temporal patterns** simulating business cycles
- **User behavior modeling** with think times and session patterns

#### 2. Concurrency and Scaling
- **Worker-based architecture** with configurable parallelism
- **Connection pool optimization** for maximum throughput
- **Resource contention simulation** with realistic locking patterns
- **Scalability testing** from single-user to high-concurrency scenarios

#### 3. Performance Measurement
- **Latency distribution analysis** with percentile calculations
- **Throughput measurement** (TPS/QPS) with trend analysis
- **Resource utilization tracking** (CPU, memory, I/O)
- **Error rate monitoring** with categorization and alerting

#### 4. Stress Testing Approach
- **Gradual load increase** to identify breaking points
- **Sustained load testing** for stability verification
- **Resource exhaustion scenarios** (connection limits, memory pressure)
- **Recovery testing** after system failures

### Data Generation Strategy

#### Realistic Data Patterns
- **Referential integrity** with proper foreign key relationships
- **Data distribution** matching real-world patterns
- **Temporal consistency** with realistic timestamps
- **Size scaling** based on configurable parameters

#### Advanced Data Types
- **JSONB documents** with nested structures and indexing
- **Vector embeddings** for semantic search scenarios
- **Geographic data** with spatial indexing
- **Time-series data** with partitioning strategies

### Testing Scenarios

#### 1. OLTP Workloads
- **High-frequency transactions** with minimal latency
- **Read-heavy patterns** typical of web applications
- **Write-intensive scenarios** for data ingestion
- **Mixed workloads** balancing reads and writes

#### 2. OLAP Workloads
- **Complex analytical queries** with multiple joins
- **Aggregation operations** over large datasets
- **Window functions** and advanced SQL features
- **Reporting scenarios** with varied query patterns

#### 3. Modern Application Patterns
- **Vector similarity search** for AI/ML applications
- **Full-text search** with complex ranking
- **Real-time analytics** with streaming data
- **Microservice patterns** with distributed transactions

## 📦 Available Workloads

### 1. E-Commerce Workload (`ecommerce`)
*Advanced e-commerce platform simulation with automated systems*

**Features:**
- Complete product catalog with vendor management
- Automated inventory control with purchase orders
- Customer analytics and segmentation  
- Vector-powered review similarity search (pgvector)
- Real-time pricing with margin protection

**Modes:**
- `ecommerce_mixed` - 75% reads, 25% writes (production-realistic)
- `ecommerce_read` - Catalog browsing and analytics
- `ecommerce_write` - Orders, inventory, and stock control
- `ecommerce_analytics` - Complex analytical queries
- `ecommerce_oltp` - High-frequency transactions

**Schema Highlights:**
- 10+ interconnected tables with referential integrity
- Automated triggers for stock control and pricing
- pgvector integration for semantic search
- Complex indexes for OLTP and analytical workloads

### 2. IMDB Movie Database (`imdb`)
*Hollywood movie database with complex relationships*

**Features:**
- Complete movie industry data model
- Celebrity relationships and filmography
- Box office analytics and trends
- Complex multi-table joins and aggregations

**Modes:**
- `imdb_read` - Search and analytics queries
- `imdb_write` - Data updates and modifications
- `imdb_mixed` - Balanced read/write operations

**Data Loading Options:**
- Generated synthetic data
- Real IMDB dataset import
- Custom SQL data loading

### 3. Vector Search Workload (`vector`)
*Advanced vector similarity search testing*

**Features:**
- High-dimensional vector operations (1024D)
- Multiple similarity metrics (L2, cosine, inner product)
- Index performance testing (IVFFlat, HNSW)
- Batch and real-time search scenarios

**Variants:**
- `vector_1024` - Standard L2 distance
- `vector_1024_cosine` - Cosine similarity  
- `vector_1024_inner` - Inner product similarity

### 4. TPC-C Benchmark (`tpcc`)
*Industry standard OLTP benchmark*

**Features:**
- Official TPC-C specification compliance
- Order processing and inventory management
- Warehouse distribution and logistics
- Standard performance metrics and reporting

### 5. Simple Workloads (`simple`)
*Basic load testing patterns*

**Features:**
- Configurable read/write ratios
- Simple table structures for baseline testing
- Adjustable complexity levels
- Quick setup for initial testing

**Modes:**
- `read` - Read-only operations
- `write` - Write-only operations  
- `mixed` - Configurable read/write mix

## 🛠️ Installation and Setup

### Prerequisites

#### System Requirements
- **Operating System**: Linux, macOS, or Windows
- **Go**: Version 1.24+ 
- **PostgreSQL**: Version 12+ (recommended: 15+)
- **Memory**: Minimum 4GB RAM (8GB+ recommended)
- **Storage**: SSD recommended for optimal performance

#### Optional Dependencies
- **pgvector**: For vector similarity search workloads
- **Docker**: For containerized deployments
- **Grafana/Prometheus**: For advanced monitoring

### Installation Methods

#### 1. From Source (Recommended)
```bash
# Clone the repository
git clone https://github.com/your-org/stormdb.git
cd stormdb

# Install dependencies
make deps

# Build the binary
make build

# Verify installation
./stormdb --help
```

#### 2. Using Go Install
```bash
go install github.com/your-org/stormdb/cmd/stormdb@latest
```

#### 3. Docker Container
```bash
docker pull your-org/stormdb:latest
docker run -it your-org/stormdb:latest --help
```

### Database Setup

#### 1. PostgreSQL Installation
```bash
# Ubuntu/Debian
sudo apt-get install postgresql postgresql-contrib

# macOS (Homebrew)
brew install postgresql

# Start PostgreSQL service
sudo systemctl start postgresql
# or
brew services start postgresql
```

#### 2. pgvector Extension (Optional)
```bash
# Install pgvector for vector workloads
git clone https://github.com/pgvector/pgvector.git
cd pgvector
make
sudo make install

# Enable in PostgreSQL
psql -c "CREATE EXTENSION vector;"
```

#### 3. Database Configuration
```sql
-- Create test database
CREATE DATABASE stormdb_test;

-- Create user (optional)
CREATE USER stormdb_user WITH PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE stormdb_test TO stormdb_user;

-- Optimize for testing (optional)
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
SELECT pg_reload_conf();
```

## 🚀 Quick Start Guide

### 1. Basic Load Test
```bash
# Run a quick e-commerce mixed workload
./stormdb -c config/config_ecommerce_mixed.yaml

# Run for specific duration with custom workers
./stormdb -c config/config_ecommerce_mixed.yaml --duration=2m --workers=20
```

### 2. Configuration Setup
Create `my_config.yaml`:
```yaml
database:
  type: postgres
  host: "localhost"
  port: 5432
  dbname: "stormdb_test"
  username: "stormdb_user"
  password: "secure_password"
  sslmode: "disable"

workload: "ecommerce_mixed"
workers: 10
duration: "60s"
scale: 1000
connections: 20

metrics:
  enabled: true
  interval: "5s"
  histogram_buckets: [1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]
```

### 3. Interactive Demo
```bash
# Run the e-commerce interactive demo
./demo_ecommerce_workload.sh

# Follow the menu to explore different workload modes
```

### 4. Schema Management
```bash
# Setup schema without running tests
./stormdb -c config/config_ecommerce_mixed.yaml --setup

# Rebuild schema and reload data
./stormdb -c config/config_ecommerce_mixed.yaml --rebuild

# Clean up after testing
./stormdb -c config/config_ecommerce_mixed.yaml --cleanup
```

## 📊 Configuration Guide

### Configuration Structure

#### Database Configuration
```yaml
database:
  type: "postgres"           # Database type (currently only postgres)
  host: "localhost"          # Database host
  port: 5432                # Database port
  dbname: "test_db"         # Database name
  username: "user"          # Database user
  password: "password"      # Database password
  sslmode: "disable"        # SSL mode (disable/require/verify-full)
```

#### Workload Configuration
```yaml
workload: "ecommerce_mixed"   # Workload type
workers: 10                   # Number of concurrent workers
duration: "60s"              # Test duration (Go duration format)
scale: 1000                  # Data scale factor
connections: 20              # Maximum database connections
```

#### Advanced Settings
```yaml
# Data loading configuration (for IMDB workload)
data_loading:
  mode: "generate"           # generate/dump/sql
  filepath: ""               # Path to data file

# Metrics configuration
metrics:
  enabled: true              # Enable metrics collection
  interval: "5s"             # Metrics collection interval
  histogram_buckets: [1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]
```

### Environment Variables

stormdb supports environment variable override for sensitive configuration:

```bash
export STORMDB_DB_HOST="production.db.example.com"
export STORMDB_DB_PASSWORD="$ecure_p@ssw0rd"
export STORMDB_DB_SSLMODE="require"

# Environment variables take precedence over config file values
```

### Configuration Profiles

#### Development Profile
```yaml
# config/dev.yaml
database:
  host: "localhost"
  dbname: "stormdb_dev"
workload: "simple"
workers: 2
duration: "30s"
scale: 100
```

#### Production Testing Profile  
```yaml
# config/prod.yaml
database:
  host: "prod-replica.internal"
  dbname: "production"
  sslmode: "require"
workload: "ecommerce_mixed"
workers: 50
duration: "10m"
scale: 10000
connections: 100
```

#### Performance Benchmarking Profile
```yaml
# config/benchmark.yaml
workload: "tpcc"
workers: 100
duration: "30m"
scale: 50000
connections: 200
metrics:
  enabled: true
  interval: "1s"
```

## 📈 Usage Examples

### Basic Usage Patterns

#### 1. Development Testing
```bash
# Quick validation test
./stormdb -c config/dev.yaml

# Schema validation only
./stormdb -c config/dev.yaml --setup

# Full rebuild for clean testing
./stormdb -c config/dev.yaml --rebuild
```

#### 2. Performance Testing
```bash
# Baseline performance test
./stormdb -c config/ecommerce_mixed.yaml -duration=5m

# High-concurrency stress test
./stormdb -c config/ecommerce_mixed.yaml -workers=50 -duration=15m

# Read-heavy load test
./stormdb -c config/ecommerce_read.yaml -workers=20 -duration=10m
```

#### 3. Comparative Analysis
```bash
# Test different workload modes
./stormdb -c config/ecommerce_read.yaml > results_read.log
./stormdb -c config/ecommerce_write.yaml > results_write.log
./stormdb -c config/ecommerce_analytics.yaml > results_analytics.log

# Compare vector vs traditional search
./stormdb -c config/vector_1024_cosine.yaml > results_vector.log
./stormdb -c config/simple_read.yaml > results_traditional.log
```

### Advanced Scenarios

#### 1. Continuous Load Testing
```bash
#!/bin/bash
# continuous_load.sh - Run continuous background load

while true; do
    echo "Starting load test cycle: $(date)"
    ./stormdb -c config/ecommerce_mixed.yaml -duration=10m
    
    # Brief pause between cycles
    sleep 60
done
```

#### 2. Progressive Load Testing
```bash
#!/bin/bash
# progressive_load.sh - Gradually increase load

WORKERS=(5 10 20 50 100)
for w in "${WORKERS[@]}"; do
    echo "Testing with $w workers"
    ./stormdb -c config/ecommerce_mixed.yaml -workers=$w -duration=5m > "results_${w}workers.log"
    
    # Analysis pause
    sleep 30
done
```

#### 3. Progressive Connection Scaling (Automated)
```bash
# Automated progressive scaling with mathematical analysis
./stormdb -c config/config_progressive_imdb.yaml --setup

# Enable progressive mode via CLI flag
./stormdb -c config/imdb_mixed.yaml --progressive --setup

# Quick scaling analysis
./stormdb -c config/progressive_tpcc.yaml --progressive
```

**Example Progressive Output:**
```
🎯 Starting progressive scaling test with 25 bands
📊 Strategy: linear, Band Duration: 30s, Warmup: 10s, Cooldown: 5s

🔄 Band 1/25: 10 workers, 20 connections
📊 Band 1 completed: 1,234 TPS, 45.2ms avg latency

🔄 Band 15/25: 70 workers, 100 connections  
📊 Band 15 completed: 4,123 TPS, 89.1ms avg latency

✅ Progressive scaling completed successfully
📊 Tested 25 bands, optimal config: 40 workers, 60 connections (2,341 TPS)
📈 Mathematical analysis: Linear scaling until band 12, diminishing returns detected
🎯 Recommendation: Use 40-50 workers for optimal efficiency
```

#### 4. Multi-Database Testing
```bash
#!/bin/bash
# multi_db_test.sh - Test across multiple database instances

DATABASES=("db1" "db2" "db3")
for db in "${DATABASES[@]}"; do
    echo "Testing database: $db"
    
    # Update config for current database
    sed "s/dbname: .*/dbname: $db/" config/template.yaml > "config/temp_$db.yaml"
    
    ./stormdb -c "config/temp_$db.yaml" -duration=5m > "results_$db.log"
    
    # Cleanup temp config
    rm "config/temp_$db.yaml"
done
```

### Monitoring and Analysis

#### 1. Real-time Monitoring
```bash
# Monitor performance in real-time
./stormdb -c config/ecommerce_mixed.yaml | grep -E "(TPS|Latency|Errors)"

# Detailed metrics output
./stormdb -c config/ecommerce_mixed.yaml | tee performance.log
```

#### 2. Log Analysis
```bash
# Extract key metrics
grep "Final Results" performance.log
grep "Error Summary" performance.log

# Parse latency percentiles
awk '/Latency Distribution:/{flag=1; next} /^[[:space:]]*$/{flag=0} flag' performance.log
```

#### 3. Performance Comparison
```bash
# Compare two test runs
diff -u baseline.log current.log | grep -E "^[+-].*TPS|^[+-].*Avg"

# Extract summary statistics
for log in *.log; do
    echo "=== $log ==="
    grep -E "TPS:|Average Latency:|Errors:" "$log"
done
```

## 🔧 Development and Extension

### Extending stormdb

#### 1. Creating Custom Workloads

stormdb's extensible architecture makes it easy to add custom workloads:

```go
// internal/workload/custom/custom.go
package custom

import (
    "context"
    "stormdb/pkg/types"
    "github.com/jackc/pgx/v5/pgxpool"
)

type CustomWorkload struct {
    Mode string
}

func (w *CustomWorkload) GetName() string {
    return "custom_" + w.Mode
}

func (w *CustomWorkload) Setup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
    // Create schema and load data
    return nil
}

func (w *CustomWorkload) Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
    // Execute workload operations
    return nil
}

func (w *CustomWorkload) Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
    // Clean up resources
    return nil
}
```

#### 2. Registering Custom Workloads

Add your workload to the factory:

```go
// internal/workload/factory.go
func Get(workloadType string) (Workload, error) {
    switch workloadType {
    // ... existing cases ...
    case "custom":
        return &custom.CustomWorkload{Mode: "default"}, nil
    case "custom_advanced":
        return &custom.CustomWorkload{Mode: "advanced"}, nil
    default:
        return nil, fmt.Errorf("unknown workload: %s", workloadType)
    }
}
```

#### 3. Advanced Workload Patterns

##### Workload with Multiple Operations
```go
type MultiOpWorkload struct {
    operations []OperationFunc
    weights    []int
}

func (w *MultiOpWorkload) selectOperation(rng *rand.Rand) OperationFunc {
    // Weighted random selection of operations
    total := 0
    for _, weight := range w.weights {
        total += weight
    }
    
    target := rng.Intn(total)
    current := 0
    
    for i, weight := range w.weights {
        current += weight
        if target < current {
            return w.operations[i]
        }
    }
    
    return w.operations[0] // fallback
}
```

##### State-Aware Workloads
```go
type StatefulWorkload struct {
    userSessions map[int]*UserSession
    sessionMutex sync.RWMutex
}

func (w *StatefulWorkload) getOrCreateSession(userID int) *UserSession {
    w.sessionMutex.RLock()
    session, exists := w.userSessions[userID]
    w.sessionMutex.RUnlock()
    
    if !exists {
        w.sessionMutex.Lock()
        session = &UserSession{
            UserID: userID,
            StartTime: time.Now(),
            Actions: make([]Action, 0),
        }
        w.userSessions[userID] = session
        w.sessionMutex.Unlock()
    }
    
    return session
}
```

### Development Workflow

#### 1. Environment Setup
```bash
# Clone and setup development environment
git clone https://github.com/your-org/stormdb.git
cd stormdb

# Install development dependencies
make deps

# Setup git hooks
cp scripts/pre-commit .git/hooks/
chmod +x .git/hooks/pre-commit
```

#### 2. Code Quality Standards
```bash
# Format code
make fmt

# Run linters
make lint

# Run tests
make test

# Check test coverage
make test-coverage
```

#### 3. Testing New Workloads
```bash
# Unit tests for workload logic
go test ./internal/workload/custom -v

# Integration tests with database
STORMDB_TEST_HOST=localhost make test-integration

# Load testing validation
./stormdb -c config/custom_workload.yaml --setup
./stormdb -c config/custom_workload.yaml -duration=30s
```

### Advanced Customization

#### 1. Custom Metrics Collection
```go
type CustomMetrics struct {
    *types.Metrics
    BusinessMetrics map[string]int64
    mutex          sync.RWMutex
}

func (cm *CustomMetrics) RecordBusinessMetric(name string, value int64) {
    cm.mutex.Lock()
    defer cm.mutex.Unlock()
    
    cm.BusinessMetrics[name] += value
}

func (cm *CustomMetrics) GetBusinessMetric(name string) int64 {
    cm.mutex.RLock()
    defer cm.mutex.RUnlock()
    
    return cm.BusinessMetrics[name]
}
```

#### 2. Custom Data Generators
```go
type DataGenerator interface {
    GenerateUser() *User
    GenerateProduct() *Product
    GenerateOrder(userID int, productIDs []int) *Order
}

type RealisticDataGenerator struct {
    faker  *gofakeit.Faker
    config *GeneratorConfig
}

func (g *RealisticDataGenerator) GenerateUser() *User {
    return &User{
        Email:    g.faker.Email(),
        Name:     g.faker.Name(),
        Country:  g.faker.Country(),
        JoinDate: g.faker.DateRange(time.Now().AddDate(-2, 0, 0), time.Now()),
    }
}
```

#### 3. Custom Configuration Extensions
```go
type CustomConfig struct {
    *types.Config
    CustomSettings struct {
        FeatureFlags map[string]bool `mapstructure:"feature_flags"`
        Thresholds   struct {
            ErrorRate     float64 `mapstructure:"error_rate"`
            LatencyP99    int     `mapstructure:"latency_p99"`
        } `mapstructure:"thresholds"`
    } `mapstructure:"custom"`
}
```

## 🚢 Deployment Guide

### Production Deployment

#### 1. Container Deployment
```dockerfile
# Dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o stormdb cmd/stormdb/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates postgresql-client
WORKDIR /app
COPY --from=builder /app/stormdb .
COPY --from=builder /app/config ./config
COPY --from=builder /app/docs ./docs

ENTRYPOINT ["./stormdb"]
```

```bash
# Build and deploy container
docker build -t stormdb:latest .
docker run -d --name stormdb-test \
  -v $(pwd)/config:/app/config \
  -e STORMDB_DB_HOST=database.internal \
  -e STORMDB_DB_PASSWORD=secure_password \
  stormdb:latest -c config/production.yaml
```

#### 2. Kubernetes Deployment
```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: stormdb
spec:
  replicas: 1
  selector:
    matchLabels:
      app: stormdb
  template:
    metadata:
      labels:
        app: stormdb
    spec:
      containers:
      - name: stormdb
        image: stormdb:latest
        args: ["-c", "/config/production.yaml"]
        volumeMounts:
        - name: config
          mountPath: /config
        env:
        - name: STORMDB_DB_HOST
          valueFrom:
            secretKeyRef:
              name: stormdb-secret
              key: db-host
        - name: STORMDB_DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: stormdb-secret
              key: db-password
      volumes:
      - name: config
        configMap:
          name: stormdb-config
```

#### 3. CI/CD Integration
```yaml
# .github/workflows/test.yml
name: stormdb CI/CD

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: 1.24
    
    - name: Install dependencies
      run: make deps
    
    - name: Run tests
      run: make test-all
      env:
        STORMDB_TEST_HOST: localhost
        STORMDB_TEST_PASSWORD: postgres
    
    - name: Build binary
      run: make build
    
    - name: Integration test
      run: ./stormdb -c config/test.yaml --setup
```

### High-Availability Deployment

#### 1. Load Balancer Configuration
```nginx
# nginx.conf
upstream stormdb_backends {
    server stormdb-1:8080;
    server stormdb-2:8080;
    server stormdb-3:8080;
}

server {
    listen 80;
    server_name stormdb.internal;
    
    location / {
        proxy_pass http://stormdb_backends;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

#### 2. Database Connection Pool Optimization
```yaml
# config/production.yaml
database:
  host: "postgres-cluster.internal"
  port: 5432
  dbname: "production"
  username: "stormdb_prod"
  password: "${STORMDB_DB_PASSWORD}"
  sslmode: "require"
  
# Connection pool settings
connections: 100
connection_timeout: "30s"
idle_timeout: "10m"
max_lifetime: "1h"

# Production workload settings
workload: "ecommerce_mixed"
workers: 50
duration: "1h"
scale: 100000

# Monitoring configuration
metrics:
  enabled: true
  interval: "10s"
  export_prometheus: true
  prometheus_port: 9090
```

#### 3. Monitoring and Alerting
```yaml
# monitoring/prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'stormdb'
    static_configs:
      - targets: ['stormdb:9090']

rule_files:
  - "stormdb_alerts.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']
```

```yaml
# monitoring/stormdb_alerts.yml
groups:
- name: stormdb
  rules:
  - alert: HighErrorRate
    expr: stormdb_error_rate > 0.05
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "stormdb error rate is high"
      
  - alert: HighLatency
    expr: stormdb_latency_p99 > 1000
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "stormdb latency is high"
```

### Performance Optimization

#### 1. Database Tuning
```sql
-- PostgreSQL optimization for stormdb
ALTER SYSTEM SET shared_buffers = '1GB';
ALTER SYSTEM SET effective_cache_size = '3GB';
ALTER SYSTEM SET maintenance_work_mem = '256MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 1000;
ALTER SYSTEM SET random_page_cost = 1.1;
ALTER SYSTEM SET effective_io_concurrency = 200;

-- Connection settings
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET max_prepared_transactions = 100;

-- Reload configuration
SELECT pg_reload_conf();
```

#### 2. System-Level Optimizations
```bash
# System optimization script
#!/bin/bash

# Increase file descriptor limits
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# Optimize network settings
echo "net.core.somaxconn = 65535" >> /etc/sysctl.conf
echo "net.ipv4.tcp_max_syn_backlog = 65535" >> /etc/sysctl.conf
echo "net.core.netdev_max_backlog = 5000" >> /etc/sysctl.conf

# Apply settings
sysctl -p
```

#### 3. Application-Level Tuning
```yaml
# config/high_performance.yaml
database:
  connections: 200
  connection_timeout: "5s"
  query_timeout: "30s"
  
workload: "ecommerce_mixed"
workers: 100
batch_size: 1000
think_time: "1ms"

# Optimize metrics collection
metrics:
  enabled: true
  interval: "30s"
  buffer_size: 10000
  async_processing: true
```

## 🔍 Monitoring and Observability

### Built-in Metrics

stormdb provides comprehensive metrics out of the box:

#### Performance Metrics
- **Throughput**: Transactions Per Second (TPS), Queries Per Second (QPS)
- **Latency**: Average, P50, P95, P99, P99.9 percentiles
- **Error Rates**: Total errors, errors by type, error percentages
- **Resource Usage**: Connection pool utilization, memory usage

#### Business Metrics
- **Operation Counts**: By operation type (read/write/analytics)
- **Data Volumes**: Rows processed, bytes transferred
- **Workload-Specific**: E-commerce orders, IMDB queries, vector searches

#### System Metrics
- **Database Connections**: Active, idle, failed connections
- **Transaction States**: Committed, rolled back, in-progress
- **Query Performance**: Execution plans, index usage

### Metrics Output Format

#### Console Output
```
============================================================
stormdb Load Test Results
============================================================
Test Duration: 5m0s
Total Workers: 20
Database: postgres://localhost:5432/test

Performance Summary:
  Total Queries: 89,534
  Successful: 89,122 (99.54%)
  Failed: 412 (0.46%)
  
  Throughput: 297.8 TPS
  Query Rate: 298.1 QPS

Latency Distribution:
  Average: 67.2ms
  P50: 45.1ms
  P95: 187.3ms
  P99: 312.8ms
  P99.9: 567.2ms

Error Summary:
  connection_timeout: 245 (59.5%)
  query_timeout: 123 (29.9%)
  constraint_violation: 44 (10.7%)

Workload Breakdown:
  Read Operations: 67,234 (75.2%)
  Write Operations: 22,288 (24.8%)
  
Database Performance:
  Connection Pool: 19/20 (95% utilization)
  Average Query Time: 67.2ms
  Index Hit Ratio: 98.7%
============================================================
```

#### JSON Output
```json
{
  "test_summary": {
    "duration": "5m0s",
    "workers": 20,
    "database": "postgres://localhost:5432/test"
  },
  "performance": {
    "total_queries": 89534,
    "successful": 89122,
    "failed": 412,
    "success_rate": 99.54,
    "tps": 297.8,
    "qps": 298.1
  },
  "latency": {
    "average_ms": 67.2,
    "percentiles": {
      "p50": 45.1,
      "p95": 187.3,
      "p99": 312.8,
      "p999": 567.2
    }
  },
  "errors": {
    "connection_timeout": 245,
    "query_timeout": 123,
    "constraint_violation": 44
  },
  "workload": {
    "read_operations": 67234,
    "write_operations": 22288
  }
}
```

### Integration with Monitoring Systems

#### Prometheus Integration
```go
// Export metrics to Prometheus
func (m *Metrics) ExportPrometheus() {
    prometheus.MustRegister(
        prometheus.NewGaugeFunc(
            prometheus.GaugeOpts{
                Name: "stormdb_tps",
                Help: "Transactions per second",
            },
            func() float64 { return float64(atomic.LoadInt64(&m.TPS)) },
        ),
    )
}
```

#### Grafana Dashboards
```json
{
  "dashboard": {
    "title": "stormdb Performance Dashboard",
    "panels": [
      {
        "title": "Throughput",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(stormdb_total_queries[5m])",
            "legendFormat": "QPS"
          }
        ]
      },
      {
        "title": "Latency Percentiles",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.50, stormdb_latency_histogram)",
            "legendFormat": "P50"
          },
          {
            "expr": "histogram_quantile(0.95, stormdb_latency_histogram)",
            "legendFormat": "P95"
          }
        ]
      }
    ]
  }
}
```

## 🧪 Testing Framework

### Test Architecture

stormdb includes a comprehensive testing framework:

```
test/
├── unit/                 # Unit tests (no external dependencies)
│   ├── config_test.go    # Configuration validation
│   ├── workload_test.go  # Workload factory tests
│   ├── metrics_test.go   # Metrics calculation tests
│   └── histogram_test.go # Histogram functionality tests
├── integration/          # Integration tests (require database)
│   └── workload_integration_test.go
├── load/                 # Load and performance tests
│   └── load_test.go      # Concurrency and stress tests
├── fixtures/             # Test configuration files
│   ├── valid_config.yaml
│   └── invalid_*.yaml
└── run_tests.sh         # Test runner script
```

### Running Tests

#### Unit Tests (Fast)
```bash
# Run all unit tests
make test-unit

# Run specific test
go test ./internal/config -v

# Run with coverage
go test -cover ./internal/...
```

#### Integration Tests (Require Database)
```bash
# Setup test database
export STORMDB_TEST_HOST=localhost
export STORMDB_TEST_DB=stormdb_test
export STORMDB_TEST_USER=postgres
export STORMDB_TEST_PASSWORD=password

# Run integration tests
make test-integration

# Run specific integration test
go test ./test/integration -v -run TestECommerceWorkload
```

#### Load Tests (Performance Validation)
```bash
# Run load tests
make test-load

# Run stress tests (long-running)
make test-stress

# Full test suite
make test-all
```

### Custom Test Development

#### Unit Test Example
```go
// plugins/ecommerce_plugin/ecommerce_test.go  
func TestECommerceWorkloadSetup(t *testing.T) {
    // Mock database
    db := &mockDB{}
    
    // Create workload
    workload := &ECommerceWorkload{Mode: "mixed"}
    
    // Test setup
    err := workload.Setup(context.Background(), db, &types.Config{
        Scale: 100,
    })
    
    assert.NoError(t, err)
    assert.True(t, db.SchemaCreated())
}
```

#### Integration Test Example
```go
// test/integration/workload_integration_test.go
func TestWorkloadIntegration(t *testing.T) {
    // Connect to test database
    db := connectTestDB(t)
    defer db.Close()
    
    workloads := []string{
        "ecommerce_mixed",
        "imdb_read",
        "vector_1024",
    }
    
    for _, workloadType := range workloads {
        t.Run(workloadType, func(t *testing.T) {
            testWorkloadExecution(t, db, workloadType)
        })
    }
}
```

### Performance Benchmarking

#### Benchmark Tests
```go
// benchmark_test.go
func BenchmarkECommerceRead(b *testing.B) {
    db := setupBenchmarkDB(b)
    workload := &ecommerce.ECommerceWorkload{Mode: "read"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        workload.executeReadOperation(context.Background(), db, rand.New(rand.NewSource(1)))
    }
}

func BenchmarkVectorSearch(b *testing.B) {
    db := setupBenchmarkDB(b)
    workload := &vector.Workload1024{SimilarityMetric: "cosine"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        workload.executeVectorQuery(context.Background(), db, generateRandomVector())
    }
}
```

#### Performance Regression Tests
```bash
#!/bin/bash
# scripts/regression_test.sh

# Run baseline benchmark
go test -bench=. -benchmem > baseline.txt

# Run current benchmark  
go test -bench=. -benchmem > current.txt

# Compare results
benchcmp baseline.txt current.txt
```

## 🔒 Security Considerations

### Database Security

#### Connection Security
```yaml
# config/secure.yaml
database:
  type: postgres
  host: "secure-db.internal"
  port: 5432
  dbname: "secure_test"
  username: "stormdb_user"
  password: "${STORMDB_DB_PASSWORD}"  # From environment
  sslmode: "require"
  sslcert: "/certs/client.crt"
  sslkey: "/certs/client.key"
  sslrootcert: "/certs/ca.crt"
```

#### Access Control
```sql
-- Create restricted user for testing
CREATE USER stormdb_test WITH PASSWORD 'secure_random_password';

-- Grant minimal required permissions
GRANT CONNECT ON DATABASE test_db TO stormdb_test;
GRANT USAGE ON SCHEMA public TO stormdb_test;
GRANT CREATE ON SCHEMA public TO stormdb_test;

-- Restrict access to specific tables only
GRANT SELECT, INSERT, UPDATE, DELETE ON specific_tables TO stormdb_test;
```

#### Data Privacy
```yaml
# Use synthetic data for sensitive environments
workload: "ecommerce_mixed"
data_generation:
  mode: "synthetic"
  anonymize_pii: true
  mask_sensitive_fields: true
  
# Avoid production data in test environments
database:
  production_data_forbidden: true
  require_test_suffix: true  # Database name must end with _test
```

### Network Security

#### TLS Configuration
```yaml
database:
  sslmode: "require"
  sslcert: "/secure/certs/client.crt"
  sslkey: "/secure/certs/client.key"
  sslrootcert: "/secure/certs/ca.crt"

# Verify server certificates
security:
  verify_ssl_certs: true
  allowed_hosts: ["secure-db.internal", "backup-db.internal"]
```

#### Firewall Rules
```bash
# Allow connections only from specific IPs
iptables -A INPUT -p tcp --dport 5432 -s 10.0.1.0/24 -j ACCEPT
iptables -A INPUT -p tcp --dport 5432 -j DROP

# Monitor connections
iptables -A INPUT -p tcp --dport 5432 -j LOG --log-prefix "STORMDB_DB_ACCESS: "
```

### Operational Security

#### Secrets Management
```bash
# Use external secret management
export STORMDB_DB_PASSWORD=$(vault kv get -field=password secret/stormdb/db)
export STORMDB_API_KEY=$(aws ssm get-parameter --name stormdb-api-key --with-decryption --query Parameter.Value --output text)

# Avoid credentials in config files
./stormdb -c config/production.yaml  # Uses environment variables
```

#### Audit Logging
```yaml
# config/audit.yaml
logging:
  level: "info"
  format: "json"
  audit_enabled: true
  sensitive_fields: ["password", "token", "key"]
  
audit:
  log_connections: true
  log_queries: false  # Avoid logging sensitive data
  log_errors: true
  log_performance: true
```

## 🛠️ Troubleshooting Guide

### Common Issues and Solutions

#### 1. Connection Issues

**Problem**: "Failed to connect to database"
```
Error: failed to connect to database: connection refused
```

**Solutions**:
```bash
# Check PostgreSQL service
sudo systemctl status postgresql
sudo systemctl start postgresql

# Verify connection settings
psql -h localhost -p 5432 -U postgres -d test_db

# Check network connectivity
telnet localhost 5432

# Verify authentication
grep -E "host|local" /etc/postgresql/*/main/pg_hba.conf
```

#### 2. Schema Creation Failures

**Problem**: "Permission denied for schema creation"
```
Error: failed to create schema: permission denied for schema public
```

**Solutions**:
```sql
-- Grant schema permissions
GRANT CREATE ON SCHEMA public TO stormdb_user;
GRANT USAGE ON SCHEMA public TO stormdb_user;

-- Or create dedicated schema
CREATE SCHEMA stormdb_test AUTHORIZATION stormdb_user;
ALTER USER stormdb_user SET search_path = stormdb_test;
```

#### 3. pgvector Extension Issues

**Problem**: "pgvector extension not available"
```
Warning: pgvector extension not available, review vectors will be disabled
```

**Solutions**:
```bash
# Install pgvector
git clone https://github.com/pgvector/pgvector.git
cd pgvector
make
sudo make install

# Enable extension
psql -c "CREATE EXTENSION IF NOT EXISTS vector;"

# Verify installation
psql -c "SELECT * FROM pg_extension WHERE extname = 'vector';"
```

#### 4. Performance Issues

**Problem**: Low throughput and high latency
```
Performance Summary:
  Throughput: 12.3 TPS (expected: >100 TPS)
  Average Latency: 2,345ms (expected: <100ms)
```

**Solutions**:
```bash
# Check database configuration
psql -c "SHOW shared_buffers;"
psql -c "SHOW effective_cache_size;"

# Optimize connection pool
# Increase connections in config
connections: 50  # Up from 10

# Check for lock contention
psql -c "SELECT * FROM pg_locks WHERE NOT granted;"

# Analyze slow queries
psql -c "SELECT query, mean_time, calls FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"
```

#### 5. Memory Issues

**Problem**: Out of memory errors
```
Error: failed to allocate memory for query result
```

**Solutions**:
```yaml
# Reduce batch sizes
workload_config:
  batch_size: 100  # Reduce from 1000
  
# Limit concurrent workers
workers: 10  # Reduce from 50

# Optimize query complexity
query_optimization:
  use_streaming: true
  limit_result_size: 1000
```

### Performance Optimization

#### Database Tuning Checklist
```sql
-- Memory settings
ALTER SYSTEM SET shared_buffers = '25% of RAM';
ALTER SYSTEM SET effective_cache_size = '75% of RAM';
ALTER SYSTEM SET work_mem = '4MB';

-- Connection settings
ALTER SYSTEM SET max_connections = '200';
ALTER SYSTEM SET max_prepared_transactions = '100';

-- Checkpoint settings
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';

-- Query planner settings
ALTER SYSTEM SET random_page_cost = 1.1;  # For SSDs
ALTER SYSTEM SET effective_io_concurrency = 200;

SELECT pg_reload_conf();
```

#### Index Optimization
```sql
-- Check missing indexes
SELECT schemaname, tablename, attname, n_distinct, correlation
FROM pg_stats
WHERE schemaname = 'public'
  AND n_distinct > 100
  AND correlation < 0.1;

-- Analyze query performance
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM products WHERE category = 'Electronics';

-- Create optimized indexes
CREATE INDEX CONCURRENTLY idx_products_category_active 
ON products(category) WHERE is_active = true;
```

### Monitoring and Debugging

#### Debug Mode
```bash
# Enable debug logging
./stormdb -c config/debug.yaml --log-level=debug

# Trace query execution
STORMDB_TRACE_QUERIES=true ./stormdb -c config/test.yaml
```

#### Connection Monitoring
```sql
-- Monitor connection usage
SELECT 
    datname,
    usename,
    application_name,
    client_addr,
    state,
    query_start,
    state_change
FROM pg_stat_activity 
WHERE application_name LIKE 'stormdb%';

-- Check connection limits
SELECT 
    max_conn,
    used,
    res_for_super,
    max_conn-used-res_for_super AS available
FROM 
    (SELECT count(*) used FROM pg_stat_activity) t1,
    (SELECT setting::int res_for_super FROM pg_settings WHERE name='superuser_reserved_connections') t2,
    (SELECT setting::int max_conn FROM pg_settings WHERE name='max_connections') t3;
```

### Getting Help

#### Log Analysis
```bash
# Extract error patterns
grep -E "ERROR|FATAL|PANIC" stormdb.log | sort | uniq -c

# Analyze performance patterns
grep "TPS\|Latency" stormdb.log | tail -20

# Connection issues
grep -i "connection" stormdb.log | grep -i "fail"
```

#### Community Support
- **GitHub Issues**: Report bugs and feature requests
- **Documentation**: Check docs/ directory for detailed guides
- **Examples**: See config/ directory for configuration examples
- **Stack Overflow**: Tag questions with `stormdb` and `postgresql`

#### Professional Support
- **Consulting**: Available for enterprise deployments
- **Custom Development**: Specialized workloads and integrations
- **Training**: Team training and best practices workshops

## 📚 Additional Resources

### Documentation Links
- [Progressive Scaling Guide](docs/PROGRESSIVE_SCALING.md) - Connection scaling and mathematical analysis
- [E-Commerce Workload Guide](docs/ECOMMERCE_WORKLOAD.md) - Comprehensive e-commerce testing
- [IMDB Workload Guide](docs/IMDB_WORKLOAD.md) - Movie database testing scenarios
- [Vector Search Guide](docs/VECTOR_WORKLOAD.md) - pgvector integration and testing
- [Signal Handling Guide](docs/SIGNAL_HANDLING.md) - Graceful shutdown and monitoring
- [Troubleshooting Guide](docs/TROUBLESHOOTING.md) - Common issues and solutions

### Configuration Examples
- [Progressive Scaling Configurations](config/config_progressive_*.yaml) - Progressive connection scaling
- [E-Commerce Configurations](config/config_ecommerce_*.yaml) - Various e-commerce scenarios
- [IMDB Configurations](config/config_imdb_*.yaml) - Movie database testing
- [Vector Configurations](config/config_vector_*.yaml) - Vector similarity search
- [Performance Configurations](config/config_benchmark_*.yaml) - High-performance testing

### Demo Scripts
- [E-Commerce Demo](demo_ecommerce_workload.sh) - Interactive e-commerce workload demo
- [IMDB Demo](demo_imdb_workloads.sh) - Movie database workload examples
- [Signal Handling Demo](demo_signal_handling.sh) - Graceful shutdown demonstration

### External Resources
- [PostgreSQL Performance Tuning](https://www.postgresql.org/docs/current/performance-tips.html)
- [pgvector Documentation](https://github.com/pgvector/pgvector)
- [Go Database Best Practices](https://go.dev/doc/database/index)
- [Load Testing Methodology](https://en.wikipedia.org/wiki/Load_testing)

## 🔌 Plugin Development

StormDB supports a powerful plugin system that allows you to create custom workloads without modifying the core codebase.

### Creating a Plugin

1. **Create a new plugin directory**:
```bash
mkdir plugins/my_workload_plugin
cd plugins/my_workload_plugin
```

2. **Initialize the module**:
```bash
go mod init stormdb/plugins/my_workload_plugin
```

3. **Implement the WorkloadPlugin interface**:
```go
package main

import (
    "fmt"
    "stormdb/pkg/plugin"
)

type MyPlugin struct{}

func (p *MyPlugin) GetMetadata() *plugin.PluginMetadata {
    return &plugin.PluginMetadata{
        Name:        "my_workload",
        Version:     "1.0.0",
        Description: "Custom workload for specific testing",
        Author:      "Your Name",
        WorkloadTypes: []string{"my_workload", "my_workload_read"},
    }
}

func (p *MyPlugin) CreateWorkload(workloadType string) (plugin.Workload, error) {
    switch workloadType {
    case "my_workload":
        return &MyWorkload{}, nil
    default:
        return nil, fmt.Errorf("unsupported workload type: %s", workloadType)
    }
}

func (p *MyPlugin) Initialize() error { return nil }
func (p *MyPlugin) Cleanup() error { return nil }

var WorkloadPlugin MyPlugin
```

4. **Build the plugin**:
```bash
go build -buildmode=plugin -o my_workload_plugin.so main.go
```

### Plugin Configuration

Add your plugin to the configuration:
```yaml
plugins:
  paths:
    - "./build/plugins"
  files:
    - "./build/plugins/my_workload_plugin.so"
  auto_load: true

workload: "my_workload"
```

### Plugin Best Practices

- **Error Handling**: Provide clear error messages
- **Documentation**: Include README and examples
- **Testing**: Write unit tests for your workload
- **Dependencies**: Minimize external dependencies
- **Performance**: Optimize for concurrent execution

See `examples/plugins/simple_example/` for a complete example and `docs/PLUGIN_DEVELOPMENT.md` for detailed documentation.

## 🤝 Contributing

### How to Contribute

We welcome contributions to stormdb! Here's how you can help:

#### 1. Bug Reports
- Use GitHub Issues to report bugs
- Include reproduction steps and environment details
- Provide logs and configuration files when possible

#### 2. Feature Requests
- Describe the use case and expected behavior
- Consider backward compatibility implications
- Provide implementation suggestions if possible

#### 3. Code Contributions
```bash
# Fork the repository
git clone https://github.com/your-username/stormdb.git
cd stormdb

# Create feature branch
git checkout -b feature/new-workload

# Make changes and test
make test-all

# Submit pull request
git push origin feature/new-workload
```

#### 4. Documentation Improvements
- Fix typos and clarify instructions
- Add examples and use cases
- Improve configuration documentation

### Development Guidelines

#### Code Standards
- **Go Style**: Follow Go formatting and naming conventions
- **Testing**: Include unit and integration tests for new features
- **Documentation**: Update relevant documentation files
- **Compatibility**: Maintain backward compatibility when possible

#### Workload Development
- **Interface Compliance**: Implement the Workload interface completely
- **Error Handling**: Provide comprehensive error handling and recovery
- **Performance**: Optimize for high-concurrency scenarios
- **Documentation**: Include workload-specific documentation

#### Commit Standards
```bash
# Use conventional commit format
git commit -m "feat(ecommerce): add automated stock control system"
git commit -m "fix(vector): handle pgvector extension not available"
git commit -m "docs(readme): add deployment section"
```

### Community Guidelines

- **Be Respectful**: Maintain a welcoming and inclusive environment
- **Collaborate**: Work together to improve the project
- **Learn**: Share knowledge and help others learn
- **Quality**: Prioritize code quality and testing

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

```
MIT License

Copyright (c) 2025 stormdb contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

---

<div align="center">

**stormdb** - Powering PostgreSQL Performance Testing

*Built with ❤️ for the PostgreSQL community*

[![GitHub Stars](https://img.shields.io/github/stars/your-org/stormdb?style=social)](https://github.com/your-org/stormdb)
[![Twitter Follow](https://img.shields.io/twitter/follow/stormdb?style=social)](https://twitter.com/stormdb)

</div>
