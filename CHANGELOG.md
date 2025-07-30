# Changelog

All notable changes to StormDB will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Placeholder for future changes

### Changed
- Placeholder for future changes

### Fixed
- Placeholder for future changes

## [0.1.0-alpha.1] - 2025-01-28

### Added

### Added
- **🚀 Initial Alpha Release**: First public release of StormDB PostgreSQL Performance Testing Tool
- **🔌 Plugin Architecture**: Extensible workload system with dynamic plugin loading
- **🏗️ Core Framework**: Complete benchmarking framework with Go 1.24+ support
- **📊 Comprehensive Metrics**: Transaction performance, latency percentiles, error tracking
- **🛠️ Build System**: Make-based build system with plugin compilation support
- **🐳 Docker Support**: Multi-stage containerization with CGO plugin support
- **🧪 Testing Suite**: Unit, integration, and load tests (26 passing unit tests)
- **📚 Documentation**: Comprehensive README, architecture docs, and usage guides
- **⚙️ CI/CD Pipeline**: GitHub Actions workflow for automated testing and releases

#### Built-in Workloads
- **TPC-C**: Industry-standard OLTP benchmark with realistic transaction processing
- **Simple/Mixed**: Basic read/write operations for quick testing and baseline performance  
- **Connection Overhead**: Compare persistent vs transient connection performance

#### Plugin Workloads (Dynamically Loaded)
- **IMDB Plugin**: Movie database workload with complex queries and realistic data patterns
- **Vector Plugin**: High-dimensional vector similarity search testing (requires pgvector)
- **E-commerce Plugin**: Modern retail platform with inventory, orders, and analytics
- **E-commerce Basic Plugin**: Basic e-commerce workloads with standard OLTP patterns

#### Core Features
- **Dynamic Plugin Loading**: Load workload plugins at runtime without recompilation
- **Plugin Discovery**: Automatic scanning and loading of plugin files (.so, .dll, .dylib)
- **Configuration System**: YAML-based configuration with validation and environment variable support
- **Connection Pooling**: Optimized PostgreSQL connection management with pgx/v5
- **Signal Handling**: Graceful shutdown and interrupt handling
- **PostgreSQL Monitoring**: Deep database statistics collection and analysis

#### Development & Operations
- **Enhanced Makefile**: Comprehensive build, test, and quality targets
- **Docker Containerization**: Multi-stage builds with docker-compose setup
- **Security Scanning**: gosec and govulncheck integration
- **Code Quality**: golangci-lint configuration and pre-commit hooks
- **Live Reloading**: Air configuration for development workflow
- **Cross-Platform**: Support for Linux, macOS, and Windows

### Infrastructure & Tooling
### Infrastructure & Tooling
- **GitHub Actions**: Automated CI/CD pipeline with testing and release workflows
- **Comprehensive .gitignore**: Proper exclusions for build artifacts, IDE files, and credentials
- **Security Policy**: Vulnerability reporting guidelines and security best practices
- **Contributing Guidelines**: Development workflows and code quality standards
- **Performance Documentation**: Optimization guides and troubleshooting resources

### Technical Specifications
- **Go Version**: Requires Go 1.24+ for building from source
- **PostgreSQL**: Compatible with PostgreSQL 12+ (recommended: 15+)
- **CGO Support**: Full CGO compilation for plugin system functionality
- **Dependencies**: Minimal external dependencies with security-focused package selection
- **Architecture**: Modular design with clear separation between core engine and workload logic

### Configuration & Usage
- **119 Files**: Complete project with ~24K lines of code
- **Configuration Examples**: 20+ example YAML configurations for different scenarios
- **Command-Line Interface**: Comprehensive CLI with configuration overrides
- **Schema Management**: Automated setup, rebuild, and cleanup operations
- **Multiple Installation Methods**: Source build, Docker, and binary releases

### Known Limitations (Alpha Release)
- **Plugin Hot-Loading**: Plugins require restart to reload (planned for future release)
- **Windows Plugin Support**: Limited testing on Windows platform
- **Monitoring Dashboard**: Built-in dashboard not yet implemented (Grafana recommended)
- **Advanced Scheduling**: Complex workload scheduling patterns not yet supported

### Migration Notes
- This is the initial alpha release - no migration required
- Configuration format is stable but may be extended in future releases
- Plugin interface is experimental and may change before 1.0.0

## [1.0.0] - Future Release

### Version Format
- Use [Semantic Versioning](https://semver.org/)
- Format: `[MAJOR.MINOR.PATCH] - YYYY-MM-DD`
- Link to release tag: `[1.0.0]: https://github.com/elchinoo/stormdb/releases/tag/v1.0.0`

### Change Categories
- **Added**: New features
- **Changed**: Changes in existing functionality
- **Deprecated**: Soon-to-be removed features
- **Removed**: Removed features
- **Fixed**: Bug fixes
- **Security**: Security improvements

### Entry Format
- Use present tense: "Add feature" not "Added feature"
- Be descriptive but concise
- Include relevant issue/PR numbers: `Fix connection pool leak (#123)`
- Group related changes together
- Use bullet points for multiple items

### Examples

#### Adding a New Feature
```
### Added
- New Redis workload plugin for caching performance testing (#45)
- Support for PostgreSQL 16 with enhanced monitoring (#67)
- Configuration validation with detailed error messages (#89)
```

#### Bug Fixes
```
### Fixed
- Connection pool not properly closed on shutdown (#123)
- Metrics calculation error for high-latency operations (#145)
- Plugin loading failure on Windows systems (#167)
```

#### Security Updates
```
### Security
- Updated dependencies to fix CVE-2024-1234 in yaml parser
- Enhanced input validation to prevent SQL injection in custom queries
- Improved plugin sandbox isolation for untrusted plugins
```

---

**Note**: This changelog is automatically updated during the release process. 
For the latest development changes, see the commit history on the main branch.

## Release Links

- [v0.1.0-alpha.1]: https://github.com/elchinoo/stormdb/releases/tag/v0.1.0-alpha.1
