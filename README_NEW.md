# StormDB - PostgreSQL Performance Testing Tool

[![CI/CD Pipeline](https://github.com/elchinoo/stormdb/actions/workflows/ci.yml/badge.svg)](https://github.com/elchinoo/stormdb/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/elchinoo/stormdb/branch/main/graph/badge.svg)](https://codecov.io/gh/elchinoo/stormdb)
[![Go Report Card](https://goreportcard.com/badge/github.com/elchinoo/stormdb)](https://goreportcard.com/report/github.com/elchinoo/stormdb)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/release/elchinoo/stormdb.svg)](https://github.com/elchinoo/stormdb/releases/latest)

StormDB is a comprehensive PostgreSQL benchmarking and load testing tool designed to help you understand your database performance characteristics. It features a modern plugin architecture that provides multiple workload types, detailed metrics analysis, and advanced monitoring capabilities.

## 📋 Table of Contents

- [Key Features](#-key-features)
- [Why StormDB?](#-why-stormdb)
- [Architecture](#️-architecture)
- [Installation](#-installation)
- [Quick Start](#-quick-start)
- [Documentation](#-documentation)
- [Contributing](#-contributing)
- [License](#-license)
- [Support](#-support)

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
- **Automated Discovery**: Systematically test multiple worker/connection configurations
- **Mathematical Analysis**: Advanced statistical analysis including discrete derivatives, inflection points, and curve fitting
- **Queueing Theory**: M/M/c queue modeling for scientific bottleneck identification  
- **Scaling Strategies**: Linear (thorough), exponential (fast), and fibonacci (research) scaling patterns
- **Bottleneck Classification**: Automatic identification of CPU, I/O, queue, and memory bottlenecks
- **Optimal Configuration**: AI-driven recommendation of best performance configurations
- **Scientific Export**: Comprehensive CSV/JSON export with statistical analysis for research and production planning
- **Real-time Analysis**: Live mathematical insights during test execution

## 🤔 Why StormDB?

### Born from Real-World Need
StormDB was created to address the gap between simple database benchmarking tools and the complex performance analysis needs of modern PostgreSQL deployments. Traditional tools often provide basic metrics but lack the depth needed for production optimization.

### Scientific Approach
Unlike basic load testing tools, StormDB applies mathematical rigor to performance analysis:
- **Statistical Foundation**: Proper sampling, confidence intervals, and variance analysis
- **Performance Modeling**: Queue theory and mathematical models for bottleneck identification
- **Predictive Analysis**: Curve fitting and extrapolation for capacity planning
- **Research-Grade Output**: Publication-ready analysis with proper methodology documentation

### Production-Ready Features
- **Enterprise Monitoring**: Deep PostgreSQL internals monitoring beyond basic metrics
- **Plugin Extensibility**: Adapt to any workload without modifying core code
- **Graceful Handling**: Proper signal handling, connection management, and error recovery
- **Professional Reporting**: Comprehensive reports suitable for stakeholder presentations

### Developer-Friendly
- **Modern Go Architecture**: Clean, maintainable codebase with excellent performance
- **Rich Configuration**: YAML-based configuration with validation and examples
- **Extensive Documentation**: Complete guides, examples, and API documentation
- **Active Development**: Regular updates, bug fixes, and feature additions

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
│  │   Realworld     │  │   Custom        │  │    Future       │  │
│  │   Plugin        │  │   Plugins       │  │   Plugins       │  │
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

## 🚀 Installation

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
# Pull pre-built image
docker pull elchinoo/stormdb:latest

# Or build locally
docker build -t stormdb .

# Run with custom config
docker run --rm -v $(pwd)/config:/config stormdb --config /config/my-config.yaml
```

## 🚀 Quick Start

### 1. Database Setup

```sql
-- Create test database and user
CREATE DATABASE stormdb_test;
CREATE USER stormdb_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE stormdb_test TO stormdb_user;
```

### 2. Basic Configuration

Create a `config.yaml` file:

```yaml
database:
  host: "localhost"
  port: 5432
  user: "stormdb_user"
  password: "your_password"
  dbname: "stormdb_test"
  sslmode: "disable"

workload:
  type: "simple"
  workers: 4
  duration: "30s"
  
monitoring:
  enabled: true
  interval: "1s"
```

### 3. Run Your First Test

```bash
# Setup database schema (first time only)
./stormdb --setup --config config.yaml

# Run a simple performance test
./stormdb --config config.yaml

# Run with progressive scaling
./stormdb --progressive --config config.yaml
```

### 4. View Results

StormDB provides comprehensive output including:
- Transaction performance metrics (TPS, latency percentiles)
- PostgreSQL internal statistics
- Progressive scaling analysis with recommendations
- Detailed error reporting and diagnostics

## 📚 Documentation

### User Guides
- **[Installation Guide](docs/guides/INSTALLATION.md)** - Detailed installation instructions for all platforms
- **[Configuration Guide](docs/guides/CONFIGURATION.md)** - Complete configuration reference
- **[Usage Guide](docs/guides/USAGE.md)** - Command-line usage and examples
- **[Plugin System Guide](docs/guides/PLUGIN_SYSTEM.md)** - Working with plugins
- **[Performance Optimization](docs/guides/PERFORMANCE_OPTIMIZATION.md)** - Database tuning recommendations
- **[Troubleshooting](docs/guides/TROUBLESHOOTING.md)** - Common issues and solutions

### Technical Documentation
- **[Architecture](ARCHITECTURE.md)** - System design and component overview
- **[Plugin Development](docs/PLUGIN_DEVELOPMENT.md)** - Creating custom workload plugins
- **[Progressive Scaling](docs/COMPREHENSIVE_PGVECTOR_TESTING.md)** - Advanced mathematical analysis
- **[Contributing](docs/guides/CONTRIBUTING.md)** - Development workflow and guidelines

### API Reference
- **[Go API Documentation](https://pkg.go.dev/github.com/elchinoo/stormdb)** - Complete API reference
- **[Plugin Interface](pkg/plugin/interface.go)** - Plugin development interface

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](docs/guides/CONTRIBUTING.md) for details.

### Quick Start for Contributors

```bash
# Fork the repository and clone your fork
git clone https://github.com/your-username/stormdb.git
cd stormdb

# Install development tools
make dev-tools

# Run tests
make test

# Build and test your changes
make build-all
make test-integration
```

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- PostgreSQL community for the amazing database system
- TPC-C benchmark specification for standardized testing methodology  
- Go plugin system for enabling our extensible architecture
- Contributors and users who help improve StormDB

## 📞 Support

- **Documentation**: [GitHub Pages](https://elchinoo.github.io/stormdb/)
- **Issues**: [GitHub Issues](https://github.com/elchinoo/stormdb/issues)
- **Discussions**: [GitHub Discussions](https://github.com/elchinoo/stormdb/discussions)
- **Security**: For security issues, please email [security@stormdb.dev](mailto:security@stormdb.dev)

---

**Built with ❤️ for the PostgreSQL community**
