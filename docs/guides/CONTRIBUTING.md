# Contributing Guide

Thank you for your interest in contributing to StormDB! This guide covers everything you need to know to contribute effectively.

## Quick Start

1. **Fork the repository** on GitHub
2. **Clone your fork** locally
3. **Create a feature branch** for your changes
4. **Make your changes** and test them
5. **Submit a pull request** with a clear description

```bash
# Clone your fork
git clone https://github.com/yourusername/stormdb.git
cd stormdb

# Create feature branch
git checkout -b feature/your-feature-name

# Make changes and commit
git add .
git commit -m "Add your feature"

# Push and create PR
git push origin feature/your-feature-name
```

## Ways to Contribute

### Code Contributions
- Bug fixes
- New features
- Performance improvements
- Plugin development
- Test coverage improvements

### Documentation
- API documentation
- User guides
- Examples and tutorials
- Code comments
- README improvements

### Community Support
- Answer questions in discussions
- Help troubleshoot issues
- Review pull requests
- Share usage examples

### Testing
- Report bugs with detailed reproduction steps
- Test new features and provide feedback
- Performance testing and benchmarking
- Cross-platform testing

## Development Environment Setup

### Prerequisites

- **Go**: Version 1.24 or higher
- **Git**: For version control
- **Make**: Build automation
- **PostgreSQL**: Version 12+ for testing
- **Docker**: For containerized testing (optional)

### Environment Setup

```bash
# Clone the repository
git clone https://github.com/elchinoo/stormdb.git
cd stormdb

# Install dependencies
go mod download

# Install development tools
make dev-tools

# Build the project
make build

# Run tests
make test
```

### Development Tools Installation

```bash
# Install linters and formatters
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/go-critic/go-critic/cmd/gocritic@latest

# Install testing tools
go install github.com/onsi/ginkgo/v2/ginkgo@latest
go install gotest.tools/gotestsum@latest

# Install debugging tools
go install github.com/go-delve/delve/cmd/dlv@latest
```

## Code Style and Standards

### Go Code Style

We follow standard Go conventions plus additional guidelines:

```go
// Package documentation
// Package stormdb provides database load testing capabilities.
package main

import (
    // Standard library imports first
    "context"
    "fmt"
    "time"
    
    // Third-party imports
    "github.com/spf13/cobra"
    
    // Local imports
    "github.com/elchinoo/stormdb/internal/config"
    "github.com/elchinoo/stormdb/pkg/plugin"
)

// Exported types should have documentation
type WorkloadConfig struct {
    Type     string            `yaml:"type" json:"type"`
    Duration time.Duration     `yaml:"duration" json:"duration"`
    Options  map[string]interface{} `yaml:"options" json:"options"`
}

// Public functions should have documentation
// NewWorkloadConfig creates a new workload configuration with defaults.
func NewWorkloadConfig() *WorkloadConfig {
    return &WorkloadConfig{
        Type:     "basic",
        Duration: 5 * time.Minute,
        Options:  make(map[string]interface{}),
    }
}

// Private functions should have comments for complex logic
func (w *WorkloadConfig) validate() error {
    if w.Type == "" {
        return fmt.Errorf("workload type is required")
    }
    
    if w.Duration <= 0 {
        return fmt.Errorf("duration must be positive")
    }
    
    return nil
}
```

### Code Formatting

```bash
# Format code
go fmt ./...

# Organize imports
goimports -w .

# Run linter
golangci-lint run

# Fix common issues
go fix ./...
```

### Naming Conventions

- **Packages**: lowercase, single word when possible
- **Files**: lowercase with underscores (e.g., `config_loader.go`)
- **Types**: PascalCase (e.g., `WorkloadConfig`)
- **Functions**: PascalCase for exported, camelCase for unexported
- **Variables**: camelCase
- **Constants**: UPPER_CASE or PascalCase for exported

### Error Handling

```go
// Wrap errors with context
func processData(data []byte) error {
    result, err := parseData(data)
    if err != nil {
        return fmt.Errorf("failed to parse data: %w", err)
    }
    
    if err := validateResult(result); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    return nil
}

// Use specific error types when needed
type ConfigError struct {
    Field   string
    Message string
}

func (e ConfigError) Error() string {
    return fmt.Sprintf("config error in field %s: %s", e.Field, e.Message)
}
```

## Testing Guidelines

### Test Structure

```go
package main

import (
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestWorkloadConfig_Validate(t *testing.T) {
    tests := []struct {
        name    string
        config  WorkloadConfig
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid config",
            config: WorkloadConfig{
                Type:     "basic",
                Duration: 5 * time.Minute,
            },
            wantErr: false,
        },
        {
            name: "missing type",
            config: WorkloadConfig{
                Duration: 5 * time.Minute,
            },
            wantErr: true,
            errMsg:  "workload type is required",
        },
        {
            name: "invalid duration",
            config: WorkloadConfig{
                Type:     "basic",
                Duration: -1 * time.Second,
            },
            wantErr: true,
            errMsg:  "duration must be positive",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.validate()
            
            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Test Coverage

```bash
# Run tests with coverage
go test -race -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out

# Check coverage percentage
go tool cover -func=coverage.out | grep total
```

### Integration Tests

```go
func TestIntegration_DatabaseConnection(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    
    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    // Test actual functionality
    config := &config.Database{
        Host:     "localhost",
        Port:     5432,
        Name:     "test_db",
        Username: "test_user",
        Password: "test_pass",
    }
    
    conn, err := database.Connect(config)
    require.NoError(t, err)
    defer conn.Close()
    
    // Verify connection works
    err = conn.Ping()
    assert.NoError(t, err)
}
```

### Test Utilities

```go
// test/helpers.go
package test

import (
    "database/sql"
    "testing"
    
    _ "github.com/lib/pq"
)

func SetupTestDB(t *testing.T) *sql.DB {
    t.Helper()
    
    db, err := sql.Open("postgres", "postgres://test:test@localhost/test?sslmode=disable")
    if err != nil {
        t.Skipf("PostgreSQL not available: %v", err)
    }
    
    if err := db.Ping(); err != nil {
        t.Skipf("PostgreSQL not accessible: %v", err)
    }
    
    return db
}

func CleanupTestDB(t *testing.T, db *sql.DB) {
    t.Helper()
    
    // Clean up test data
    _, err := db.Exec("TRUNCATE TABLE test_table")
    if err != nil {
        t.Logf("Failed to cleanup test data: %v", err)
    }
    
    db.Close()
}
```

## Plugin Development

### Plugin Interface Implementation

```go
package main

import (
    "C"
    "database/sql"
    
    "github.com/elchinoo/stormdb/pkg/plugin"
)

type MyPlugin struct {
    config map[string]interface{}
}

func (p *MyPlugin) Initialize(config map[string]interface{}) error {
    p.config = config
    return p.validateConfig()
}

func (p *MyPlugin) GetInfo() plugin.PluginInfo {
    return plugin.PluginInfo{
        Name:        "my_plugin",
        Version:     "1.0.0",
        Description: "Example plugin implementation",
        Author:      "Your Name <your.email@example.com>",
        License:     "MIT",
        Website:     "https://github.com/yourusername/my-plugin",
    }
}

// Export for plugin system
//export GetPlugin
func GetPlugin() plugin.WorkloadPlugin {
    return &MyPlugin{}
}

func main() {} // Required for plugin compilation
```

### Plugin Testing

```go
func TestPlugin_Initialize(t *testing.T) {
    plugin := &MyPlugin{}
    
    config := map[string]interface{}{
        "param1": "value1",
        "param2": 42,
    }
    
    err := plugin.Initialize(config)
    assert.NoError(t, err)
    
    info := plugin.GetInfo()
    assert.Equal(t, "my_plugin", info.Name)
    assert.Equal(t, "1.0.0", info.Version)
}
```

### Plugin Documentation

Create a README.md for your plugin:

```markdown
# My Plugin

Description of what your plugin does.

## Configuration

```yaml
workload:
  type: "plugin"
  plugin_name: "my_plugin"
  plugin_config:
    param1: "value1"
    param2: 42
```

## Operations

- `operation1`: Description of operation 1
- `operation2`: Description of operation 2

## Metrics

- `custom_metric1`: Description of metric 1
- `custom_metric2`: Description of metric 2
```

## Documentation Standards

### API Documentation

```go
// Package workload provides workload execution capabilities for database testing.
//
// This package implements various workload types including basic SQL operations,
// TPC-C benchmarks, and plugin-based custom workloads.
package workload

// WorkloadExecutor handles the execution of database workloads.
//
// The executor manages connection pools, coordinates concurrent operations,
// and collects performance metrics during test execution.
type WorkloadExecutor struct {
    config *Config
    db     *sql.DB
    stats  *Statistics
}

// Execute runs the configured workload for the specified duration.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - duration: How long to run the workload
//
// Returns:
//   - WorkloadResult: Aggregated performance metrics
//   - error: Any error that occurred during execution
//
// Example:
//   executor := NewWorkloadExecutor(config)
//   result, err := executor.Execute(ctx, 5*time.Minute)
//   if err != nil {
//       log.Fatal(err)
//   }
//   fmt.Printf("TPS: %.2f\n", result.TransactionsPerSecond)
func (w *WorkloadExecutor) Execute(ctx context.Context, duration time.Duration) (*WorkloadResult, error) {
    // Implementation
}
```

### User Documentation

- Use clear, concise language
- Provide complete examples
- Include common use cases
- Add troubleshooting sections
- Keep documentation up to date

### Code Comments

```go
// validateDuration ensures the duration is within acceptable limits.
// It returns an error if the duration is negative, zero, or exceeds
// the maximum allowed test duration of 24 hours.
func validateDuration(d time.Duration) error {
    const maxDuration = 24 * time.Hour
    
    if d <= 0 {
        return fmt.Errorf("duration must be positive, got %v", d)
    }
    
    if d > maxDuration {
        return fmt.Errorf("duration exceeds maximum of %v, got %v", maxDuration, d)
    }
    
    return nil
}
```

## Commit Guidelines

### Commit Message Format

```
type(scope): brief description

Longer description explaining the change in detail.
Include motivation for the change and contrast with
previous behavior.

Fixes #123
Closes #456
```

### Commit Types

- **feat**: New feature
- **fix**: Bug fix
- **docs**: Documentation changes
- **style**: Code style changes (formatting, etc.)
- **refactor**: Code refactoring
- **test**: Adding or modifying tests
- **perf**: Performance improvements
- **chore**: Maintenance tasks

### Examples

```bash
# Feature addition
git commit -m "feat(plugin): add vector similarity search plugin

Implements pgvector-based similarity search operations with
configurable distance metrics (cosine, euclidean, inner product).

Includes batch operations for improved performance and
comprehensive metrics collection.

Closes #234"

# Bug fix
git commit -m "fix(connection): handle connection pool exhaustion gracefully

Previously, the application would panic when the connection pool
was exhausted. Now it waits for available connections with a
configurable timeout and returns a meaningful error.

Fixes #567"

# Documentation
git commit -m "docs(config): add examples for progressive scaling

Added comprehensive examples showing how to configure
progressive scaling tests with different strategies
and statistical analysis options."
```

## Pull Request Process

### Before Submitting

1. **Run tests**: Ensure all tests pass
2. **Run linters**: Fix any style issues
3. **Update documentation**: Keep docs current
4. **Test locally**: Verify changes work as expected
5. **Review your changes**: Self-review before submitting

```bash
# Pre-submission checklist
make test           # Run all tests
make lint          # Check code style
make build         # Verify build works
make docs          # Update documentation
```

### Pull Request Template

```markdown
## Description

Brief description of the changes and their purpose.

## Type of Change

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Testing

- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
- [ ] I have added integration tests for any new features

## Documentation

- [ ] I have updated the documentation accordingly
- [ ] I have added docstrings/comments to new functions
- [ ] Any configuration changes are documented

## Checklist

- [ ] My code follows the style guidelines of this project
- [ ] I have performed a self-review of my own code
- [ ] My changes generate no new warnings
- [ ] I have checked my code and corrected any misspellings
```

### Review Process

1. **Automated checks**: CI/CD pipeline runs tests
2. **Code review**: Maintainers review the code
3. **Discussion**: Address feedback and questions
4. **Approval**: Once approved, PR can be merged
5. **Merge**: Maintainer merges the PR

## Release Process

### Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR**: Incompatible API changes
- **MINOR**: New functionality (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Checklist

1. **Update version** in relevant files
2. **Update CHANGELOG.md** with new features and fixes
3. **Create release tag** with proper version
4. **Build and test** release artifacts
5. **Create GitHub release** with release notes
6. **Update documentation** for new version

### Creating a Release

```bash
# Update version
echo "v1.2.3" > VERSION

# Update changelog
# Edit CHANGELOG.md

# Commit changes
git add VERSION CHANGELOG.md
git commit -m "chore: prepare release v1.2.3"

# Create and push tag
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3

# Build release artifacts
make release
```

## Community Guidelines

### Code of Conduct

- Be respectful and inclusive
- Welcome newcomers and help them learn
- Focus on constructive feedback
- Respect different opinions and approaches
- Keep discussions professional

### Communication

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and general discussion
- **Pull Requests**: Code review and technical discussion
- **Email**: Security issues only

### Getting Help

- **Documentation**: Check the guides first
- **Search**: Look for existing issues/discussions
- **Ask**: Create new issue or discussion if needed
- **Provide context**: Include relevant details

## Recognition

Contributors are recognized in:
- **CONTRIBUTORS.md**: All contributors listed
- **Release notes**: Major contributors highlighted
- **README.md**: Core maintainers and significant contributors

## Next Steps

- Read the [Configuration Guide](CONFIGURATION.md) to understand the system
- Check [Usage Guide](USAGE.md) for command-line interface
- Look at [Plugin System](PLUGIN_SYSTEM.md) for plugin development
- Review existing code in the repository
- Start with a small contribution (documentation, tests, bug fixes)

Thank you for contributing to StormDB!
