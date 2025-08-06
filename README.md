# StormDB v2 - Modular PostgreSQL Performance Testing Framework

StormDB v2 is a complete redesign of the PostgreSQL performance testing framework with a modular, plugin-based architecture.

## Architecture Overview

The v2 architecture is built around a core service that provides essential functionality to loadable plugins:

### Core Services

- **Database Manager**: Connection pooling, health monitoring, transaction support
- **Storage Manager**: Test run tracking, result storage, metadata management  
- **Plugin Manager**: Dynamic plugin loading, validation, lifecycle management
- **Scheduler Manager**: Task scheduling, worker pools, background execution
- **Configuration Manager**: Multi-format config loading, validation, plugin-specific configs
- **Logging Manager**: Structured logging with JSON/text formats, plugin context

### Plugin System

Plugins implement the `Plugin` interface and receive access to core services:

```go
type Plugin interface {
    Metadata() PluginMetadata
    Initialize(ctx context.Context, core *CoreServices) error
    Validate(config map[string]interface{}) error
    Execute(ctx context.Context, config map[string]interface{}) error
    Cleanup(ctx context.Context) error
}
```

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 12+
- Make (optional, for convenience commands)

### Building

```bash
# Install dependencies
make deps

# Build the application
make build

# Run in development mode
make dev
```

### Configuration

Create `config/core.yaml`:

```yaml
database:
  host: "localhost"
  port: 5432
  database: "stormdb"
  username: "postgres"
  password: "your_password"
  ssl_mode: "disable"

scheduler:
  worker_pool_size: 10

plugin_dirs:
  - "./plugins"
```

### Running

```bash
# With default config
./build/stormdb

# With custom config
STORMDB_CONFIG=/path/to/config.yaml ./build/stormdb
```

## Directory Structure

```
v2/
├── cmd/stormdb/           # Application entry point
├── core/                  # Core service implementations
│   ├── config/           # Configuration management
│   ├── database/         # Database connection management
│   ├── logging/          # Structured logging
│   ├── plugin/           # Plugin loading and management
│   ├── scheduler/        # Task scheduling and execution
│   ├── storage/          # Test result storage
│   └── types.go          # Core interfaces and types
├── migrations/           # Database schema migrations
├── config/              # Configuration files
├── plugins/             # Plugin implementations
└── examples/            # Example plugins and configs
```

## Database Schema

The core schema includes:

- `test_type`: Test type definitions
- `plugin`: Plugin registry and metadata
- `test_metric`: Metric definitions (latency, throughput, etc.)
- `test_run`: Test execution tracking
- `test_run_result`: Individual measurement results

## Plugin Development

### Creating a Plugin

1. Implement the `Plugin` interface
2. Build as a shared library (`.so` file)
3. Place in a configured plugin directory
4. Create plugin-specific configuration

### Example Plugin Structure

```go
package main

import (
    "context"
    "github.com/elchinoo/stormdb/v2/core"
)

type MyPlugin struct {
    core *core.CoreServices
}

func (p *MyPlugin) Metadata() core.PluginMetadata {
    return core.PluginMetadata{
        Name:        "my-plugin",
        Version:     "1.0.0",
        Description: "Example plugin",
    }
}

func (p *MyPlugin) Initialize(ctx context.Context, coreServices *core.CoreServices) error {
    p.core = coreServices
    return nil
}

// Implement other interface methods...

// Plugin entry point
func NewPlugin() core.Plugin {
    return &MyPlugin{}
}
```

## Security Improvements

- No hardcoded credentials in source code
- Plugin integrity verification with SHA256 checksums
- Secure configuration management
- Structured logging without sensitive data exposure

## Migration from v1

The v2 architecture is a complete rewrite and is not backward compatible with v1. The modular design allows for:

- Better separation of concerns
- Easier testing and maintenance
- Plugin-based extensibility
- Improved security posture
- Better performance monitoring

## Development

### Running Tests

```bash
make test
```

### Code Quality

```bash
# Format code
make fmt

# Lint (requires golangci-lint)
make lint

# Security check (requires gosec)
make security
```

### Contributing

1. Follow the modular architecture principles
2. Implement proper error handling and logging
3. Add tests for new functionality
4. Update documentation

## License

[Include your license information here]
