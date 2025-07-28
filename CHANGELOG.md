# Changelog

All notable changes to StormDB will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Enhanced Makefile with comprehensive build, test, and quality targets
- Docker support with multi-stage builds and docker-compose setup
- GitHub Actions CI/CD pipeline with automated testing and releases
- Comprehensive .gitignore file with proper exclusions
- golangci-lint configuration for code quality enforcement
- Air configuration for live reloading during development
- Security policy and vulnerability reporting guidelines
- Contributing guidelines with development workflows
- Performance optimization and troubleshooting documentation

### Changed
- Updated README.md with comprehensive installation and usage instructions
- Enhanced plugin Makefile with better build targets and testing
- Improved project structure with proper Docker and CI/CD integration

### Fixed
- Fixed plugin build process and dependency management
- Improved error handling and logging throughout the application

### Security
- Added security scanning with gosec and govulncheck
- Implemented container security best practices
- Enhanced dependency management with vulnerability checks

## [1.0.0] - 2024-XX-XX

### Added
- Initial release of StormDB PostgreSQL benchmarking tool
- Plugin architecture with dynamic workload loading
- Built-in workloads: TPC-C, Simple, Connection Overhead
- Plugin workloads: IMDB, Vector, E-commerce, RealWorld
- Comprehensive metrics collection and analysis
- PostgreSQL statistics integration (pg_stats, pg_stat_statements)
- Connection pooling and management
- YAML-based configuration system
- Signal handling for graceful shutdown
- Command-line interface with configuration overrides

### Plugin System
- Dynamic plugin discovery and loading
- Plugin metadata and version management
- Extensible workload interface
- Plugin lifecycle management (Initialize/Cleanup)
- Hot-loading of plugins without restart

### Workloads
- **TPC-C**: Industry-standard OLTP benchmark
- **Simple**: Basic read/write operations
- **Connection**: Connection overhead testing
- **IMDB**: Movie database with complex queries
- **Vector**: pgvector similarity search testing
- **E-commerce**: Retail platform simulation
- **RealWorld**: Enterprise business logic patterns

### Metrics & Monitoring
- Transaction performance metrics (TPS, latency)
- Latency percentiles (P50, P95, P99)
- Query type breakdown and analysis
- Worker-level performance tracking
- Real-time progress reporting
- PostgreSQL internal statistics
- Buffer cache and WAL monitoring
- Connection and lock tracking

### Configuration
- YAML-based configuration files
- Database connection settings
- Workload-specific parameters
- Plugin system configuration
- Monitoring and statistics options
- Command-line overrides

## Format Guidelines

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
