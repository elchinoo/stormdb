# StormDB v0.1.0-alpha.1 Release Notes

**Release Date**: January 28, 2025  
**Repository**: [github.com/elchinoo/stormdb](https://github.com/elchinoo/stormdb)  
**Release Tag**: [v0.1.0-alpha.1](https://github.com/elchinoo/stormdb/releases/tag/v0.1.0-alpha.1)

---

## 🎉 Welcome to StormDB Alpha!

We're excited to announce the **first alpha release** of StormDB, a comprehensive PostgreSQL benchmarking and load testing tool designed to help you understand your database performance characteristics. This initial release introduces a modern plugin architecture that provides multiple workload types, detailed metrics analysis, and advanced monitoring capabilities.

## ✨ What's New in v0.1.0-alpha.1

### 🚀 Core Features

#### **Plugin Architecture Revolution**
- **Dynamic Loading**: Load workload plugins at runtime without recompilation
- **Extensible System**: Easy development of custom workloads via plugin interface  
- **Plugin Discovery**: Automatic scanning and loading of plugin files (.so, .dll, .dylib)
- **Metadata Support**: Rich plugin metadata with version info and compatibility
- **Lifecycle Management**: Proper plugin initialization and cleanup

#### **Comprehensive Workload Types**

**Built-in Workloads** (No plugins required):
- **TPC-C**: Industry-standard OLTP benchmark with realistic transaction processing
- **Simple/Mixed**: Basic read/write operations for quick testing and baseline performance
- **Connection Overhead**: Compare persistent vs transient connection performance

**Plugin Workloads** (Dynamically loaded):
- **IMDB Plugin**: Movie database workload with complex queries and realistic data patterns
- **Vector Plugin**: High-dimensional vector similarity search testing (requires pgvector)
- **E-commerce Plugin**: Modern retail platform with inventory, orders, and analytics
- **E-commerce Basic Plugin**: Basic e-commerce workloads with standard OLTP patterns

#### **Advanced Metrics & Analysis**
- **Transaction Performance**: TPS, latency percentiles, success rates
- **Query Analysis**: Breakdown by type (SELECT, INSERT, UPDATE, DELETE)
- **Latency Distribution**: P50, P95, P99 with histogram visualization
- **Worker-level Metrics**: Per-thread performance tracking
- **Time-series Data**: Performance over time with configurable intervals
- **Error Tracking**: Detailed error classification and reporting

#### **PostgreSQL Deep Monitoring**
- **Buffer Cache Statistics**: Hit ratios, blocks read/written
- **WAL Activity**: WAL records, bytes generated
- **Checkpoint Monitoring**: Requested vs timed checkpoints
- **Connection Tracking**: Active connections vs limits
- **pg_stat_statements**: Top queries by execution time (optional)
- **Lock Contention**: Deadlock and wait event tracking
- **Autovacuum Activity**: Monitoring background maintenance

### 🛠️ Development & Operations

#### **Production-Ready Infrastructure**
- **Docker Support**: Multi-stage containerization with complete C toolchain for CGO plugins
- **GitHub Actions**: Automated CI/CD pipeline with comprehensive testing
- **Make-based Build System**: Comprehensive targets for building, testing, and quality checks
- **Cross-Platform**: Support for Linux, macOS, and Windows

#### **Developer Experience**
- **Comprehensive Documentation**: README, architecture docs, plugin development guides
- **Configuration Examples**: 20+ example YAML configurations for different scenarios
- **Testing Suite**: 26 passing unit tests, integration tests, and load tests
- **Code Quality**: golangci-lint configuration, pre-commit hooks, security scanning
- **Live Reloading**: Air configuration for hot-reload during development

#### **Security & Quality**
- **Security Scanning**: gosec and govulncheck integration
- **Vulnerability Management**: Automated dependency checking
- **Container Security**: Best practices for secure containerization
- **Input Validation**: Comprehensive validation of configuration and inputs

## 📦 Installation & Quick Start

### **Requirements**
- **Go 1.24+** for building from source
- **PostgreSQL 12+** (recommended: 15+)
- **Build tools**: `make`, `git`
- **Optional**: Docker for containerized usage

### **Installation Methods**

#### **Method 1: Download Pre-built Binaries**
```bash
# Download latest release (Linux/macOS/Windows)
curl -L https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.1/stormdb-linux-amd64 -o stormdb
chmod +x stormdb
```

#### **Method 2: Build from Source**
```bash
# Clone the repository
git clone https://github.com/elchinoo/stormdb.git
cd stormdb

# Build everything (binary + plugins)
make build-all

# Or build just the binary
make build
```

#### **Method 3: Docker**
```bash
# Run with Docker
docker run --rm elchinoo/stormdb:v0.1.0-alpha.1 --help

# Or use docker-compose for complete setup
docker-compose up -d postgres  # Start test database
docker-compose run stormdb --config /app/config/config_simple_connection.yaml
```

### **Quick Test Run**
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

## 🔧 Plugin System

### **Available Plugins**

| Plugin | Description | Workload Types | Requirements |
|--------|-------------|----------------|--------------|
| **IMDB** | Movie database with complex queries | `imdb_read`, `imdb_write`, `imdb_mixed`, `imdb_sql` | PostgreSQL 12+ |
| **Vector** | High-dimensional vector operations | `pgvector_*`, `vector_cosine`, `vector_inner` | pgvector extension |
| **E-commerce** | Retail platform simulation | `ecommerce_read`, `ecommerce_write`, `ecommerce_mixed` | PostgreSQL 12+ |
| **E-commerce Basic** | Basic e-commerce patterns | `ecommerce_basic` | PostgreSQL 12+ |

### **Building Plugins**
```bash
# Build all available plugins
make plugins

# Build everything at once
make build-all

# Test plugins
make test-all
```

### **Plugin Configuration**
```yaml
# Plugin system configuration
plugins:
  # Directories to search for plugin files
  paths:
    - "./plugins"
    - "./build/plugins"
  
  # Automatically load all plugins found
  auto_load: true

workload: "imdb_mixed"  # Use plugin workload
```

## 📊 Configuration Examples

### **Built-in Workloads**
```bash
# TPC-C benchmark
./stormdb --config config/config_tpcc.yaml --setup

# Simple read/write workload  
./stormdb --config config/config_simple_connection.yaml

# Connection overhead testing
./stormdb --config config/config_transient_connections.yaml
```

### **Plugin Workloads**
```bash
# IMDB complex queries
./stormdb --config config/config_imdb_mixed.yaml --rebuild

# Vector similarity search (requires pgvector)
./stormdb --config config/config_vector_cosine.yaml --setup

# E-commerce workload
./stormdb --config config/config_ecommerce_mixed.yaml --rebuild
```

## 📈 Key Metrics & Analysis

### **Performance Metrics**
- **Throughput**: Transactions Per Second (TPS)
- **Latency**: P50, P95, P99, P99.9 percentiles
- **Success Rate**: Percentage of successful operations
- **Error Tracking**: Categorized error analysis

### **Database Insights**
- **Connection Pool**: Utilization and efficiency metrics
- **Buffer Cache**: Hit ratios and I/O patterns
- **Query Performance**: Execution times and plan analysis
- **Resource Usage**: Memory, CPU, and storage metrics

### **Workload Analysis**
- **Operation Breakdown**: Read vs write operation distribution
- **Worker Performance**: Per-thread execution statistics  
- **Time-Series**: Performance evolution over test duration
- **Comparative Analysis**: Multiple workload comparison capabilities

## 🚧 Known Limitations (Alpha Release)

### **Current Limitations**
- **Plugin Hot-Loading**: Plugins require application restart to reload (planned for future release)
- **Windows Plugin Support**: Limited testing on Windows platform - feedback welcome
- **Built-in Dashboard**: No integrated monitoring dashboard (Grafana/Prometheus recommended)
- **Advanced Scheduling**: Complex workload scheduling patterns not yet supported

### **Alpha Considerations**
- **API Stability**: Plugin interface is experimental and may change before 1.0.0
- **Configuration Format**: Current format is stable but may be extended
- **Performance Optimization**: Some performance optimizations planned for beta releases
- **Documentation**: Some advanced features may have limited documentation

## 🛣️ Roadmap to 1.0.0

### **Beta Release Plans (v0.2.0-beta.1)**
- **Plugin Hot-Loading**: Runtime plugin reload without restart
- **Enhanced Windows Support**: Improved plugin compilation and testing
- **Built-in Web Dashboard**: Integrated monitoring and visualization
- **Advanced Workload Patterns**: Complex scheduling and sequencing
- **Performance Optimizations**: Connection pooling and query execution improvements

### **Release Candidate Plans (v1.0.0-rc.1)**
- **API Stabilization**: Final plugin interface and configuration format
- **Comprehensive Testing**: Extended testing across platforms and PostgreSQL versions
- **Production Hardening**: Enhanced error handling and recovery mechanisms
- **Documentation Completion**: Complete user guides and API documentation

### **1.0.0 Release Goals**
- **Production Ready**: Stable APIs and battle-tested functionality
- **Enterprise Features**: Advanced monitoring, reporting, and integration capabilities
- **Community Ecosystem**: Third-party plugin support and contribution framework
- **Long-term Support**: Commitment to backward compatibility and maintenance

## 🤝 Community & Contribution

### **Getting Started**
- **📚 Documentation**: [docs/](https://github.com/elchinoo/stormdb/tree/main/docs)
- **💬 Discussions**: [GitHub Discussions](https://github.com/elchinoo/stormdb/discussions)
- **🐛 Issues**: [GitHub Issues](https://github.com/elchinoo/stormdb/issues)
- **🔒 Security**: [SECURITY.md](https://github.com/elchinoo/stormdb/blob/main/SECURITY.md)

### **How to Contribute**
1. **🍴 Fork** the repository
2. **🌟 Create** a feature branch
3. **🧪 Test** your changes thoroughly
4. **📝 Document** new features
5. **📤 Submit** a pull request

### **Development Setup**
```bash
# Setup development environment
git clone https://github.com/elchinoo/stormdb.git
cd stormdb
make dev-tools deps

# Run tests
make test-all

# Check code quality
make lint validate-full
```

## 🙏 Acknowledgments

### **Special Thanks**
- **PostgreSQL Community**: For the excellent database system that makes this all possible
- **Go Team**: For the robust programming language and excellent tooling
- **pgvector Team**: For the innovative vector similarity search capabilities
- **Early Adopters**: Thank you for your patience and feedback during development

### **Technology Stack**
- **Language**: Go 1.24+ with CGO support
- **Database**: PostgreSQL 12+ with optional extensions
- **Containerization**: Docker with multi-stage builds
- **CI/CD**: GitHub Actions with comprehensive testing
- **Documentation**: Markdown with comprehensive examples

## 📞 Support & Feedback

### **Getting Help**
- **📖 Documentation**: Start with the [README.md](https://github.com/elchinoo/stormdb/blob/main/README.md)
- **❓ Questions**: Use [GitHub Discussions](https://github.com/elchinoo/stormdb/discussions) for general questions
- **🐛 Bug Reports**: Submit detailed bug reports to [GitHub Issues](https://github.com/elchinoo/stormdb/issues)
- **💡 Feature Requests**: Share your ideas in [GitHub Discussions](https://github.com/elchinoo/stormdb/discussions)

### **Alpha Feedback Priority**
We're especially interested in feedback on:
1. **Plugin System**: Ease of use, API design, development experience
2. **Performance**: Throughput, latency, resource usage in your environment
3. **Compatibility**: PostgreSQL versions, operating systems, edge cases
4. **Documentation**: Clarity, completeness, missing information
5. **Installation**: Build process, dependencies, deployment experience

## 📄 License

StormDB is released under the **MIT License**. See [LICENSE](https://github.com/elchinoo/stormdb/blob/main/LICENSE) for details.

---

## 🚀 Next Steps

1. **📥 Download**: Get StormDB v0.1.0-alpha.1 from the [releases page](https://github.com/elchinoo/stormdb/releases/tag/v0.1.0-alpha.1)
2. **🧪 Try It**: Run the quick start examples above
3. **📊 Explore**: Test different workloads and analyze your PostgreSQL performance
4. **💬 Share**: Join our community and share your experience
5. **🤝 Contribute**: Help us make StormDB even better

**Welcome to the StormDB community! We're excited to see what you'll build and discover with this powerful PostgreSQL benchmarking tool.**

---

*Made with ❤️ by the StormDB team*  
*Release v0.1.0-alpha.1 - January 28, 2025*
