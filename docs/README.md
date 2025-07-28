# StormDB Documentation

Welcome to the StormDB documentation. This directory contains comprehensive documentation for the PostgreSQL benchmarking tool with plugin architecture.

## Documentation Structure

### API Documentation
- **`api/`** - Auto-generated Go documentation for all packages and types
  - `stormdb.txt` - Main package documentation
  - `types.txt` - Core types and data structures
  - `workload.txt` - Workload implementation interfaces
  - `metrics.txt` - Metrics collection and reporting

### User Guides
- **`PLUGIN_DEVELOPMENT.md`** - How to develop custom workload plugins
- **`IMDB_WORKLOAD.md`** - Internet Movie Database plugin workload documentation  
- **`ECOMMERCE_WORKLOAD.md`** - E-commerce plugin workload documentation
- **`TROUBLESHOOTING.md`** - Common issues and solutions
- **`SIGNAL_HANDLING.md`** - Signal handling and graceful shutdown

### Technical Documentation
- **`IMDB_DATA_LOADING.md`** - IMDB data loading strategies and options for plugin workloads

## Generating Documentation

### View Live Documentation
```bash
make docs
```
This starts a local documentation server at http://localhost:6060

### Generate Static Documentation
```bash
make docs-generate
```
This creates static documentation files in the `api/` directory.

### Generate Coverage Reports
```bash
make test-coverage
```
This creates HTML coverage reports with detailed analysis.

## Documentation Standards

### Go Documentation
All Go code follows standard Go documentation conventions:

- Package-level documentation explains the purpose and scope
- Type documentation includes usage examples and thread-safety notes
- Function documentation describes parameters, return values, and behavior
- Examples are provided for complex APIs

### Markdown Documentation
User-facing documentation follows these standards:

- Clear headings and structure
- Code examples with syntax highlighting
- Step-by-step instructions for complex procedures
- Cross-references to related documentation

## Contributing to Documentation

When adding new features or modifying existing code:

1. Update relevant Go documentation comments
2. Add examples to demonstrate usage
3. Update user guides if the change affects user-facing behavior
4. Regenerate documentation with `make docs-generate`
5. Verify documentation renders correctly with `make docs`

## Quick Reference

### Most Important Files
- `../README.md` - Main project overview and getting started
- `TROUBLESHOOTING.md` - First stop for issues
- `api/types.txt` - Core data structures reference

### Building and Testing
- `make build` - Build the stormdb binary
- `make test` - Run unit tests
- `make validate` - Run all validation checks
- `make help` - Show all available make targets
