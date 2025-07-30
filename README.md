# StormDB

StormDB is a powerful, extensible database load testing and performance benchmarking tool specifically designed for PostgreSQL. It provides comprehensive workload simulation capabilities with advanced statistical analysis, plugin architecture, and real-time monitoring.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Configuration Examples](#configuration-examples)
- [Available Plugins](#available-plugins)
- [Command Line Usage](#command-line-usage)
- [Output Formats](#output-formats)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [Support](#support)

## Features

### Core Capabilities
- **Multiple Workload Types**: Built-in support for basic SQL operations, TPC-C benchmarks, and custom plugin-based workloads
- **Advanced Connection Management**: Sophisticated connection pooling with health monitoring and automatic retry mechanisms
- **Real-time Metrics**: Comprehensive performance monitoring with PostgreSQL statistics integration
- **Progressive Scaling**: Intelligent load scaling with statistical analysis to find optimal performance points
- **Plugin Architecture**: Extensible system supporting custom workloads for specific use cases

### Statistical Analysis
- **Confidence Intervals**: Statistical confidence in performance measurements
- **Outlier Detection**: Automatic identification and handling of statistical outliers
- **Trend Analysis**: Performance trend detection across scaling phases
- **Marginal Gains Analysis**: Detailed analysis of performance improvements between scaling levels

### Built-in Workloads
- **Basic SQL Operations**: Configurable mix of SELECT, INSERT, UPDATE, DELETE operations
- **TPC-C Benchmark**: Industry-standard OLTP benchmark implementation
- **Vector Operations**: pgvector similarity search and embedding operations (via plugin)
- **E-commerce Simulation**: Real-world e-commerce workload patterns (via plugin)
- **IMDB Dataset**: Movie database operations with complex analytical queries (via plugin)

### Monitoring and Reporting
- **Real-time Dashboard**: Web-based monitoring interface
- **Multiple Output Formats**: JSON, CSV, HTML, and console output
- **PostgreSQL Integration**: Deep integration with pg_stat_* views for database insights
- **System Metrics**: CPU, memory, and I/O monitoring during tests
- **Custom Metrics**: Plugin-specific metrics and KPIs

## Quick Start

### Installation

**Download Pre-built Binary:**
```bash
# Linux
curl -L https://github.com/elchinoo/stormdb/releases/latest/download/stormdb-linux-amd64 -o stormdb
chmod +x stormdb

# macOS
curl -L https://github.com/elchinoo/stormdb/releases/latest/download/stormdb-darwin-amd64 -o stormdb
chmod +x stormdb
```

**Build from Source:**
```bash
git clone https://github.com/elchinoo/stormdb.git
cd stormdb
make build
```

### Basic Usage

1. **Configure your database connection** in a YAML file:
```yaml
database:
  host: "localhost"
  port: 5432
  name: "testdb"
  username: "testuser"
  password: "password"

workload:
  type: "basic"
  duration: "5m"
  concurrent_users: 10
```

2. **Run a basic test**:
```bash
./stormdb --config config.yaml
```

3. **View results** in real-time or export to file:
```bash
./stormdb --config config.yaml --output results.json
```

## Architecture

### Core Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CLI & Config  │    │  Workload       │    │  Database       │
│   Management    │◄──►│  Orchestration  │◄──►│  Connection     │
└─────────────────┘    └─────────────────┘    │  Management     │
                                              └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Metrics &     │    │   Progressive   │    │   Plugin        │
│   Monitoring    │    │   Scaling       │    │   System        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Plugin Architecture

StormDB's plugin system allows for extensible workload types:

- **Built-in Workloads**: Basic SQL operations, TPC-C
- **Plugin Interface**: Standardized API for custom workloads  
- **Dynamic Loading**: Runtime plugin discovery and loading
- **Isolated Execution**: Each plugin runs in its own context

## Configuration Examples

### Basic Workload

```yaml
database:
  host: "localhost"
  port: 5432
  name: "benchmarkdb"
  username: "benchuser"
  password: "securepassword"

workload:
  type: "basic"
  duration: "10m"
  concurrent_users: 25
  
  operations:
    - type: "select"
      weight: 70
      query: "SELECT * FROM users WHERE id = $1"
      parameters: ["random_int(1,10000)"]
      
    - type: "insert"
      weight: 20
      query: "INSERT INTO logs (user_id, action) VALUES ($1, $2)"
      parameters: ["random_int(1,10000)", "random_string(20)"]
```

### Progressive Scaling

```yaml
workload:
  type: "basic"
  
  progressive_scaling:
    enabled: true
    initial_users: 1
    max_users: 100
    step_size: 5
    step_duration: "3m"
    
    # Statistical analysis
    stability_threshold: 0.05
    confidence_level: 0.95
    minimum_samples: 50
```

### Plugin Configuration

```yaml
workload:
  type: "plugin"
  plugin_name: "imdb_plugin"
  plugin_config:
    data_file: "imdb_data.csv"
    operation_weights:
      movie_search: 40
      actor_search: 30
      rating_insert: 20
      review_insert: 10
```

## Available Plugins

### IMDB Plugin
Simulates movie database operations using IMDB dataset with complex analytical queries and data loading capabilities.

### E-commerce Plugin  
Simulates online store operations including product searches, cart management, and order processing.

### Vector Plugin
Specialized for pgvector similarity search operations with configurable distance metrics and index types.

### Real World Plugin
Simulates realistic application patterns with mixed read/write operations and user session management.

## Command Line Usage

### Basic Options
```bash
stormdb [OPTIONS]

Options:
  -c, --config FILE          Configuration file (required)
  -d, --duration DURATION    Test duration (e.g., 5m, 1h)
  -u, --users COUNT          Concurrent users
  -o, --output FILE          Output file path
  -f, --format FORMAT        Output format (console, json, csv, html)
  -v, --verbose              Verbose output
```

### Advanced Options
```bash
Database:
      --host HOST            Database host
      --port PORT            Database port  
      --database NAME        Database name

Monitoring:
      --dashboard            Enable web dashboard
      --metrics              Enable metrics collection
      --pg-stats             Enable PostgreSQL statistics

Utilities:
      --validate             Validate configuration
      --test-connection      Test database connection
      --list-plugins         List available plugins
      --version              Show version information
```

## Output Formats

### Console Output
Real-time metrics display with progress indicators:
```
StormDB v1.0.0 - Database Performance Testing

Progress: [████████████████████████████████████████] 100%
Duration: 5m0s | Users: 25 | TPS: 1,247.3 | Avg Response: 18.2ms

Final Results:
├─ Transactions: 374,190
├─ TPS (avg): 1,247.3
├─ Response Time (avg): 18.2ms
├─ Response Time (p95): 45.1ms
├─ Error Rate: 0.02%
└─ Connections Used: 25/50
```

### JSON Output
Comprehensive structured data suitable for analysis and integration with other tools.

### CSV Output
Tabular format for spreadsheet analysis and data processing.

### HTML Output
Rich formatted reports with charts and visualizations.

## Documentation

### 📖 Comprehensive Guides

- **[Installation Guide](docs/guides/INSTALLATION.md)** - Detailed installation instructions for all platforms
- **[Configuration Guide](docs/guides/CONFIGURATION.md)** - Complete configuration reference with examples  
- **[Usage Guide](docs/guides/USAGE.md)** - Command-line options and usage patterns
- **[Plugin System Guide](docs/guides/PLUGIN_SYSTEM.md)** - Creating and using custom workload plugins
- **[Performance Optimization](docs/guides/PERFORMANCE_OPTIMIZATION.md)** - Tuning tips and best practices
- **[Troubleshooting Guide](docs/guides/TROUBLESHOOTING.md)** - Common issues and solutions

### 🔧 Advanced Topics

- **[Progressive Scaling](docs/COMPREHENSIVE_PGVECTOR_TESTING.md)** - Statistical analysis and performance discovery
- **[Vector Testing](docs/COMPREHENSIVE_PGVECTOR_TESTING.md)** - pgvector similarity search optimization
- **[Signal Handling](docs/SIGNAL_HANDLING.md)** - Process management and graceful shutdowns

### 💡 Examples and Use Cases

- **[E-commerce Workload](docs/ECOMMERCE_WORKLOAD.md)** - Online store simulation patterns
- **[IMDB Data Loading](docs/IMDB_DATA_LOADING.md)** - Large dataset import and testing
- **[IMDB Workload](docs/IMDB_WORKLOAD.md)** - Complex analytical query patterns

### 🛠️ Development

- **[Plugin Development](docs/PLUGIN_DEVELOPMENT.md)** - Create custom workload plugins
- **[Contributing Guide](docs/guides/CONTRIBUTING.md)** - Contribute to StormDB development

## Contributing

We welcome contributions! Please see our [Contributing Guide](docs/guides/CONTRIBUTING.md) for details on:

- Code style and standards
- Testing requirements  
- Pull request process
- Development environment setup

### Ways to Contribute

- **Code**: Bug fixes, new features, performance improvements
- **Documentation**: User guides, examples, API documentation
- **Testing**: Bug reports, feature testing, performance validation
- **Community**: Answer questions, share examples, provide feedback

## Support

### Community Resources

- **[GitHub Issues](https://github.com/elchinoo/stormdb/issues)** - Report bugs and request features
- **[GitHub Discussions](https://github.com/elchinoo/stormdb/discussions)** - Ask questions and share ideas
- **[Documentation](docs/)** - Comprehensive guides and examples

### Getting Help

1. **Check the documentation** - Most questions are answered in our guides
2. **Search existing issues** - Your question may already be answered
3. **Create a new issue** - For bugs, include reproduction steps
4. **Start a discussion** - For questions and general help

## License

StormDB is released under the [MIT License](LICENSE).

---

**Need help getting started?** Check out our [Installation Guide](docs/guides/INSTALLATION.md) and [Configuration Guide](docs/guides/CONFIGURATION.md) for step-by-step instructions.
