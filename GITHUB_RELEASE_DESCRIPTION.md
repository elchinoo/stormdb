# StormDB v0.1.0-alpha.1 - First Alpha Release! 🎉

**StormDB** is a comprehensive PostgreSQL benchmarking and load testing tool with a modern plugin architecture for advanced performance testing.

## 🚀 Highlights

- **🔌 Plugin Architecture**: Dynamic workload loading with 4 built-in plugins (IMDB, Vector, E-commerce, RealWorld)
- **📊 Built-in Workloads**: TPC-C, Simple operations, Connection overhead testing
- **🧪 Comprehensive Testing**: 26 passing unit tests + integration/load tests
- **🐳 Production Ready**: Docker support with CGO plugin compilation
- **📈 Advanced Metrics**: Transaction performance, latency percentiles, PostgreSQL monitoring
- **⚙️ Easy Setup**: Make-based build system with extensive documentation

## 📦 Quick Start

### Download & Run
```bash
# Download binary
curl -L https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.1/stormdb-linux-amd64 -o stormdb
chmod +x stormdb

# Run simple benchmark
./stormdb --config config/config_simple_connection.yaml
```

### Build from Source
```bash
git clone https://github.com/elchinoo/stormdb.git
cd stormdb
make build-all  # Builds binary + all plugins
```

### Docker
```bash
docker run --rm elchinoo/stormdb:v0.1.0-alpha.1 --help
```

## 🔧 Available Workloads

**Built-in** (no plugins needed):
- `tpcc` - Industry-standard OLTP benchmark
- `simple` - Basic read/write operations  
- `connection` - Connection overhead analysis

**Plugin-based** (requires `make plugins`):
- `imdb_mixed` - Movie database with complex queries
- `vector_cosine` - pgvector similarity search (requires pgvector)
- `ecommerce_mixed` - Modern retail platform simulation
- `realworld` - Enterprise business logic patterns

## 📊 What You Get

```
============================================================
StormDB Load Test Results
============================================================
Test Duration: 2m0s
Total Workers: 10
Database: postgres://localhost:5432/test

Performance Summary:
  Total Queries: 23,456
  Query Rate: 195.5 QPS
  Success Rate: 99.8%

Latency Distribution:
  Average: 51.2ms
  P50: 45.1ms
  P95: 89.3ms
  P99: 156.7ms

PostgreSQL Stats:
  Buffer Cache Hit Ratio: 99.2%
  Active Connections: 9/10
  Checkpoints: 3 (all scheduled)
============================================================
```

## 🚧 Alpha Notes

This is an **alpha release** - APIs may change before 1.0.0. Known limitations:
- Plugin hot-loading requires restart
- Limited Windows plugin testing
- No built-in dashboard (use Grafana/Prometheus)

## 📚 Documentation

- **[README.md](https://github.com/elchinoo/stormdb/blob/main/README.md)** - Complete usage guide
- **[ARCHITECTURE.md](https://github.com/elchinoo/stormdb/blob/main/ARCHITECTURE.md)** - System design
- **[docs/](https://github.com/elchinoo/stormdb/tree/main/docs)** - Detailed documentation
- **[RELEASE_NOTES_v0.1.0-alpha.1.md](https://github.com/elchinoo/stormdb/blob/main/RELEASE_NOTES_v0.1.0-alpha.1.md)** - Full release notes

## 🤝 Community

- **💬 Discussions**: Ask questions and share experiences
- **🐛 Issues**: Report bugs and request features  
- **🔒 Security**: See SECURITY.md for vulnerability reporting

---

**Requirements**: Go 1.24+, PostgreSQL 12+  
**Tested on**: Linux, macOS, Docker  
**License**: MIT

We'd love your feedback on this alpha release! 🙏
