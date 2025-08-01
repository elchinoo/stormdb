# StormDB Plugin Development Guide

## Overview

This guide provides comprehensive information for developing plugins for the StormDB performance testing framework. StormDB plugins are Go shared libraries (.so files) that implement specific workload patterns and testing scenarios.

## Table of Contents

1. [Plugin Architecture](#plugin-architecture)
2. [Getting Started](#getting-started)
3. [Plugin Interface](#plugin-interface)
4. [Security Guidelines](#security-guidelines)
5. [Dependency Management](#dependency-management)
6. [Testing and Validation](#testing-and-validation)
7. [Best Practices](#best-practices)
8. [Advanced Features](#advanced-features)
9. [Deployment Guide](#deployment-guide)
10. [Troubleshooting](#troubleshooting)

## Plugin Architecture

### Core Concepts

StormDB plugins are based on a modular architecture that allows for:
- **Dynamic Loading**: Plugins are loaded at runtime
- **Type Safety**: Strong typing through Go interfaces
- **Security**: Comprehensive security validation
- **Dependency Management**: Automatic dependency resolution
- **Health Monitoring**: Continuous health checking

### Plugin Lifecycle

1. **Discovery**: Plugin files are discovered in configured directories
2. **Validation**: Security validation including checksum verification
3. **Loading**: Plugin is loaded into memory and symbols resolved
4. **Initialization**: Plugin initialization hooks are called
5. **Registration**: Plugin is registered with the framework
6. **Execution**: Plugin methods are called during test execution
7. **Cleanup**: Plugin resources are cleaned up on unload

## Getting Started

### Prerequisites

- Go 1.21 or later
- CGO enabled
- Access to PostgreSQL database for testing

### Creating Your First Plugin

1. **Create Plugin Directory**
```bash
mkdir my-plugin
cd my-plugin
go mod init my-plugin
```

2. **Create Main Plugin File**
```go
// main.go
package main

import (
    "context"
    "database/sql"
    "log"
    
    "github.com/elchinoo/stormdb/pkg/plugin"
    "github.com/elchinoo/stormdb/pkg/types"
)

// MyPlugin implements the Plugin interface
type MyPlugin struct {
    name        string
    description string
    version     string
    config      map[string]interface{}
}

// Plugin interface implementation
func (p *MyPlugin) Name() string {
    return p.name
}

func (p *MyPlugin) Description() string {
    return p.description
}

func (p *MyPlugin) Version() string {
    return p.version
}

func (p *MyPlugin) Initialize(config map[string]interface{}) error {
    p.config = config
    log.Printf("Plugin %s initialized with config: %v", p.name, config)
    return nil
}

func (p *MyPlugin) Execute(ctx context.Context, db *sql.DB, workerID int, 
                          config map[string]interface{}) (*types.OperationResult, error) {
    
    // Your plugin logic here
    start := time.Now()
    
    // Example: Simple SELECT query
    _, err := db.ExecContext(ctx, "SELECT 1")
    if err != nil {
        return &types.OperationResult{
            Success:   false,
            Duration:  time.Since(start),
            Error:     err.Error(),
            Operation: "select",
        }, err
    }
    
    return &types.OperationResult{
        Success:   true,
        Duration:  time.Since(start),
        Operation: "select",
    }, nil
}

func (p *MyPlugin) Cleanup() error {
    log.Printf("Plugin %s cleanup completed", p.name)
    return nil
}

func (p *MyPlugin) HealthCheck() error {
    // Implement health check logic
    return nil
}

// Plugin factory function - required export
func NewPlugin() plugin.Plugin {
    return &MyPlugin{
        name:        "my-plugin",
        description: "My custom StormDB plugin",
        version:     "1.0.0",
    }
}

func main() {
    // Required for shared library
}
```

3. **Build Plugin**
```bash
go build -buildmode=plugin -o my-plugin.so main.go
```

### Plugin Metadata

Create a metadata file for your plugin:

```yaml
# my-plugin.yaml
name: my-plugin
version: 1.0.0
description: My custom StormDB plugin
author: Your Name
license: MIT
dependencies:
  - postgresql-driver: ">=1.0.0"
tags:
  - database
  - postgresql
  - performance
security:
  checksum: sha256:abcd1234... # Generate with sha256sum
  signature: ... # Optional PGP signature
```

## Plugin Interface

### Core Interface

All plugins must implement the `Plugin` interface:

```go
type Plugin interface {
    // Basic plugin information
    Name() string
    Description() string
    Version() string
    
    // Lifecycle methods
    Initialize(config map[string]interface{}) error
    Execute(ctx context.Context, db *sql.DB, workerID int, 
           config map[string]interface{}) (*types.OperationResult, error)
    Cleanup() error
    HealthCheck() error
}
```

### Extended Interfaces

For advanced functionality, implement additional interfaces:

#### ConfigurablePlugin
```go
type ConfigurablePlugin interface {
    Plugin
    GetConfigSchema() map[string]interface{}
    ValidateConfig(config map[string]interface{}) error
}
```

#### MetricsPlugin
```go
type MetricsPlugin interface {
    Plugin
    GetMetrics() map[string]interface{}
    ResetMetrics()
}
```

#### BatchPlugin
```go
type BatchPlugin interface {
    Plugin
    ExecuteBatch(ctx context.Context, db *sql.DB, workerID int,
                config map[string]interface{}, batchSize int) ([]*types.OperationResult, error)
}
```

### Operation Result Structure

```go
type OperationResult struct {
    Success     bool                   `json:"success"`
    Duration    time.Duration          `json:"duration"`
    Operation   string                 `json:"operation"`
    Error       string                 `json:"error,omitempty"`
    RowsAffected int64                 `json:"rows_affected,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
```

## Security Guidelines

### Checksum Validation

Always provide checksums for your plugins:

```bash
# Generate SHA256 checksum
sha256sum my-plugin.so > my-plugin.sha256

# Include in metadata
echo "checksum: sha256:$(cat my-plugin.sha256 | cut -d' ' -f1)" >> my-plugin.yaml
```

### Input Validation

Validate all inputs in your plugin:

```go
func (p *MyPlugin) Execute(ctx context.Context, db *sql.DB, workerID int, 
                          config map[string]interface{}) (*types.OperationResult, error) {
    
    // Validate context
    if ctx == nil {
        return nil, fmt.Errorf("context cannot be nil")
    }
    
    // Validate database connection
    if db == nil {
        return nil, fmt.Errorf("database connection cannot be nil")
    }
    
    // Validate worker ID
    if workerID < 0 {
        return nil, fmt.Errorf("worker ID must be non-negative")
    }
    
    // Validate configuration
    if err := p.validateExecuteConfig(config); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }
    
    // Your plugin logic here...
}

func (p *MyPlugin) validateExecuteConfig(config map[string]interface{}) error {
    // Implement configuration validation
    required := []string{"table_name", "operation_type"}
    for _, key := range required {
        if _, ok := config[key]; !ok {
            return fmt.Errorf("missing required configuration: %s", key)
        }
    }
    return nil
}
```

### SQL Injection Prevention

Use parameterized queries:

```go
func (p *MyPlugin) executeQuery(ctx context.Context, db *sql.DB, tableName string, id int) error {
    // GOOD: Parameterized query
    query := "SELECT * FROM " + tableName + " WHERE id = $1"
    _, err := db.ExecContext(ctx, query, id)
    
    // BAD: String concatenation (vulnerable to SQL injection)
    // query := fmt.Sprintf("SELECT * FROM %s WHERE id = %d", tableName, id)
    
    return err
}
```

## Dependency Management

### Declaring Dependencies

In your plugin metadata:

```yaml
dependencies:
  # Framework dependencies
  - stormdb-core: ">=2.0.0"
  - postgresql-driver: ">=1.0.0"
  
  # External libraries
  - github.com/lib/pq: ">=1.10.0"
  - github.com/golang-migrate/migrate/v4: ">=4.0.0"
```

### Go Module Dependencies

In your `go.mod`:

```go
module my-plugin

go 1.21

require (
    github.com/elchinoo/stormdb v2.0.0
    github.com/lib/pq v1.10.9
    github.com/golang-migrate/migrate/v4 v4.16.2
)
```

### Runtime Dependency Checking

Implement dependency validation:

```go
func (p *MyPlugin) Initialize(config map[string]interface{}) error {
    // Check runtime dependencies
    if err := p.checkDependencies(); err != nil {
        return fmt.Errorf("dependency check failed: %w", err)
    }
    
    // Initialize plugin
    return p.doInitialize(config)
}

func (p *MyPlugin) checkDependencies() error {
    // Check database driver
    if !p.isDatabaseDriverAvailable() {
        return fmt.Errorf("postgresql driver not available")
    }
    
    // Check required database features
    if !p.areDatabaseFeaturesAvailable() {
        return fmt.Errorf("required database features not available")
    }
    
    return nil
}
```

## Testing and Validation

### Unit Testing

Create comprehensive unit tests:

```go
// main_test.go
package main

import (
    "context"
    "database/sql"
    "testing"
    "time"
    
    "github.com/DATA-DOG/go-sqlmock"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMyPlugin_Execute(t *testing.T) {
    // Create mock database
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()
    
    // Set up expectations
    mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 1))
    
    // Create plugin instance
    plugin := &MyPlugin{
        name:        "test-plugin",
        description: "Test plugin",
        version:     "1.0.0",
    }
    
    // Initialize plugin
    config := map[string]interface{}{
        "table_name": "test_table",
        "operation_type": "select",
    }
    err = plugin.Initialize(config)
    require.NoError(t, err)
    
    // Execute plugin
    ctx := context.Background()
    result, err := plugin.Execute(ctx, db, 1, config)
    
    // Assert results
    assert.NoError(t, err)
    assert.True(t, result.Success)
    assert.Equal(t, "select", result.Operation)
    assert.True(t, result.Duration > 0)
    
    // Verify all expectations were met
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMyPlugin_HealthCheck(t *testing.T) {
    plugin := &MyPlugin{
        name:        "test-plugin",
        description: "Test plugin",
        version:     "1.0.0",
    }
    
    err := plugin.HealthCheck()
    assert.NoError(t, err)
}
```

### Integration Testing

Create integration tests:

```go
// integration_test.go
//go:build integration
// +build integration

package main

import (
    "context"
    "database/sql"
    "os"
    "testing"
    
    _ "github.com/lib/pq"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMyPlugin_Integration(t *testing.T) {
    // Get database URL from environment
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        t.Skip("DATABASE_URL not set, skipping integration test")
    }
    
    // Connect to database
    db, err := sql.Open("postgres", dbURL)
    require.NoError(t, err)
    defer db.Close()
    
    // Create plugin instance
    plugin := NewPlugin()
    
    // Initialize plugin
    config := map[string]interface{}{
        "table_name": "test_table",
        "operation_type": "select",
    }
    err = plugin.Initialize(config)
    require.NoError(t, err)
    
    // Execute plugin multiple times
    ctx := context.Background()
    for i := 0; i < 10; i++ {
        result, err := plugin.Execute(ctx, db, i, config)
        assert.NoError(t, err)
        assert.True(t, result.Success)
    }
    
    // Cleanup
    err = plugin.Cleanup()
    assert.NoError(t, err)
}
```

### Performance Testing

Test plugin performance:

```go
func BenchmarkMyPlugin_Execute(b *testing.B) {
    // Setup
    db, mock, _ := sqlmock.New()
    defer db.Close()
    
    mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 1)).AnyTimes()
    
    plugin := NewPlugin()
    config := map[string]interface{}{
        "table_name": "test_table",
        "operation_type": "select",
    }
    plugin.Initialize(config)
    
    ctx := context.Background()
    
    // Benchmark
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := plugin.Execute(ctx, db, 1, config)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Best Practices

### Error Handling

Implement comprehensive error handling:

```go
func (p *MyPlugin) Execute(ctx context.Context, db *sql.DB, workerID int, 
                          config map[string]interface{}) (*types.OperationResult, error) {
    start := time.Now()
    
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Plugin %s panic in worker %d: %v", p.name, workerID, r)
        }
    }()
    
    // Validate inputs
    if err := p.validateInputs(ctx, db, workerID, config); err != nil {
        return &types.OperationResult{
            Success:   false,
            Duration:  time.Since(start),
            Error:     err.Error(),
            Operation: "validation",
        }, err
    }
    
    // Execute operation with timeout
    execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    result, err := p.executeOperation(execCtx, db, workerID, config)
    if err != nil {
        return &types.OperationResult{
            Success:   false,
            Duration:  time.Since(start),
            Error:     err.Error(),
            Operation: result.Operation,
        }, err
    }
    
    result.Duration = time.Since(start)
    return result, nil
}
```

### Logging

Use structured logging:

```go
import "go.uber.org/zap"

type MyPlugin struct {
    logger *zap.Logger
    // other fields...
}

func (p *MyPlugin) Initialize(config map[string]interface{}) error {
    // Initialize logger
    p.logger = zap.NewProduction()
    
    p.logger.Info("Plugin initializing",
        zap.String("name", p.name),
        zap.String("version", p.version),
        zap.Any("config", config))
    
    return nil
}

func (p *MyPlugin) Execute(ctx context.Context, db *sql.DB, workerID int, 
                          config map[string]interface{}) (*types.OperationResult, error) {
    
    p.logger.Debug("Executing operation",
        zap.Int("worker_id", workerID),
        zap.String("operation", "select"))
    
    // Execute operation...
    
    p.logger.Info("Operation completed",
        zap.Int("worker_id", workerID),
        zap.Duration("duration", result.Duration),
        zap.Bool("success", result.Success))
    
    return result, nil
}
```

### Configuration Management

Implement robust configuration handling:

```go
type PluginConfig struct {
    TableName     string            `yaml:"table_name" validate:"required"`
    OperationType string            `yaml:"operation_type" validate:"required,oneof=select insert update delete"`
    BatchSize     int               `yaml:"batch_size" validate:"min=1,max=1000"`
    Timeout       time.Duration     `yaml:"timeout" validate:"min=1s,max=300s"`
    Parameters    map[string]string `yaml:"parameters"`
}

func (p *MyPlugin) Initialize(config map[string]interface{}) error {
    // Parse configuration
    var pluginConfig PluginConfig
    if err := mapstructure.Decode(config, &pluginConfig); err != nil {
        return fmt.Errorf("failed to decode configuration: %w", err)
    }
    
    // Validate configuration
    if err := validator.Struct(&pluginConfig); err != nil {
        return fmt.Errorf("configuration validation failed: %w", err)
    }
    
    p.config = pluginConfig
    return nil
}
```

### Resource Management

Properly manage resources:

```go
type MyPlugin struct {
    connections *sql.DB
    statements  map[string]*sql.Stmt
    mu          sync.RWMutex
}

func (p *MyPlugin) Initialize(config map[string]interface{}) error {
    // Prepare statements
    p.statements = make(map[string]*sql.Stmt)
    
    selectStmt, err := p.connections.Prepare("SELECT * FROM table WHERE id = $1")
    if err != nil {
        return fmt.Errorf("failed to prepare select statement: %w", err)
    }
    p.statements["select"] = selectStmt
    
    return nil
}

func (p *MyPlugin) Cleanup() error {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    // Close prepared statements
    for name, stmt := range p.statements {
        if err := stmt.Close(); err != nil {
            p.logger.Error("Failed to close statement",
                zap.String("statement", name),
                zap.Error(err))
        }
    }
    
    // Close database connections
    if p.connections != nil {
        if err := p.connections.Close(); err != nil {
            return fmt.Errorf("failed to close database connections: %w", err)
        }
    }
    
    return nil
}
```

## Advanced Features

### Custom Metrics

Implement custom metrics collection:

```go
type MyPlugin struct {
    metrics *PluginMetrics
    // other fields...
}

type PluginMetrics struct {
    mu                sync.RWMutex
    OperationsCount   map[string]int64  `json:"operations_count"`
    TotalDuration     time.Duration     `json:"total_duration"`
    AverageDuration   time.Duration     `json:"average_duration"`
    ErrorCount        int64             `json:"error_count"`
    LastOperation     time.Time         `json:"last_operation"`
}

func (p *MyPlugin) GetMetrics() map[string]interface{} {
    p.metrics.mu.RLock()
    defer p.metrics.mu.RUnlock()
    
    return map[string]interface{}{
        "operations_count":  p.metrics.OperationsCount,
        "total_duration":    p.metrics.TotalDuration,
        "average_duration":  p.metrics.AverageDuration,
        "error_count":       p.metrics.ErrorCount,
        "last_operation":    p.metrics.LastOperation,
    }
}

func (p *MyPlugin) recordMetrics(operation string, duration time.Duration, success bool) {
    p.metrics.mu.Lock()
    defer p.metrics.mu.Unlock()
    
    if p.metrics.OperationsCount == nil {
        p.metrics.OperationsCount = make(map[string]int64)
    }
    
    p.metrics.OperationsCount[operation]++
    p.metrics.TotalDuration += duration
    p.metrics.LastOperation = time.Now()
    
    if !success {
        p.metrics.ErrorCount++
    }
    
    // Calculate average duration
    totalOps := int64(0)
    for _, count := range p.metrics.OperationsCount {
        totalOps += count
    }
    if totalOps > 0 {
        p.metrics.AverageDuration = p.metrics.TotalDuration / time.Duration(totalOps)
    }
}
```

### Batch Operations

Implement batch operation support:

```go
func (p *MyPlugin) ExecuteBatch(ctx context.Context, db *sql.DB, workerID int,
                               config map[string]interface{}, batchSize int) ([]*types.OperationResult, error) {
    
    results := make([]*types.OperationResult, 0, batchSize)
    
    // Begin transaction
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()
    
    // Execute batch operations
    for i := 0; i < batchSize; i++ {
        result, err := p.executeSingleOperation(ctx, tx, workerID, config, i)
        if err != nil {
            return results, fmt.Errorf("batch operation %d failed: %w", i, err)
        }
        results = append(results, result)
    }
    
    // Commit transaction
    if err := tx.Commit(); err != nil {
        return results, fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    return results, nil
}
```

### Connection Pooling

Implement custom connection pooling:

```go
type ConnectionPool struct {
    mu          sync.RWMutex
    connections chan *sql.DB
    factory     func() (*sql.DB, error)
    maxSize     int
    currentSize int
}

func NewConnectionPool(factory func() (*sql.DB, error), maxSize int) *ConnectionPool {
    return &ConnectionPool{
        connections: make(chan *sql.DB, maxSize),
        factory:     factory,
        maxSize:     maxSize,
    }
}

func (cp *ConnectionPool) Get() (*sql.DB, error) {
    select {
    case conn := <-cp.connections:
        return conn, nil
    default:
        // Create new connection if pool is empty and not at max size
        cp.mu.Lock()
        if cp.currentSize < cp.maxSize {
            cp.currentSize++
            cp.mu.Unlock()
            return cp.factory()
        }
        cp.mu.Unlock()
        
        // Wait for available connection
        return <-cp.connections, nil
    }
}

func (cp *ConnectionPool) Put(conn *sql.DB) {
    select {
    case cp.connections <- conn:
        // Connection returned to pool
    default:
        // Pool is full, close connection
        conn.Close()
        cp.mu.Lock()
        cp.currentSize--
        cp.mu.Unlock()
    }
}
```

## Deployment Guide

### Building for Production

```bash
# Build with optimizations
go build -ldflags="-s -w" -buildmode=plugin -o my-plugin.so main.go

# Strip debug symbols (optional)
strip my-plugin.so

# Generate checksum
sha256sum my-plugin.so > my-plugin.sha256
```

### Deployment Structure

```
plugins/
├── my-plugin/
│   ├── my-plugin.so
│   ├── my-plugin.yaml
│   ├── my-plugin.sha256
│   ├── README.md
│   └── examples/
│       ├── basic-config.yaml
│       └── advanced-config.yaml
```

### Configuration Template

```yaml
# my-plugin-config.yaml
plugin: my-plugin
version: 1.0.0

# Plugin-specific configuration
parameters:
  table_name: performance_test
  operation_type: select
  batch_size: 100
  timeout: 30s
  
# Connection configuration
database:
  host: localhost
  port: 5432
  database: testdb
  user: testuser
  password: testpass
  
# Performance settings
workers: 10
duration: 300s
rate_limit: 1000
```

### Monitoring Setup

```yaml
# monitoring-config.yaml
metrics:
  enabled: true
  interval: 10s
  exporters:
    - prometheus
    - json
    
logging:
  level: info
  format: json
  output: /var/log/stormdb/my-plugin.log
  
health_checks:
  enabled: true
  interval: 30s
  timeout: 10s
```

## Troubleshooting

### Common Issues

#### Plugin Loading Errors

**Issue**: Plugin fails to load with "symbol not found" error
**Solution**: Ensure all required symbols are exported and plugin is built with correct Go version

**Issue**: Security validation fails
**Solution**: Verify checksum matches and plugin is signed correctly

#### Runtime Errors

**Issue**: Database connection failures
**Solution**: Check database connectivity and credentials

**Issue**: Context cancellation errors
**Solution**: Properly handle context cancellation in plugin code

#### Performance Issues

**Issue**: Plugin execution is slow
**Solution**: Profile plugin code and optimize database queries

**Issue**: Memory leaks
**Solution**: Ensure proper resource cleanup and avoid goroutine leaks

### Debugging Tips

1. **Enable Debug Logging**
```go
logger, _ := zap.NewDevelopment()
```

2. **Use Context Tracing**
```go
import "go.opentelemetry.io/otel/trace"

func (p *MyPlugin) Execute(ctx context.Context, ...) {
    ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("my-plugin").Start(ctx, "execute")
    defer span.End()
    
    // Your plugin logic
}
```

3. **Add Metrics**
```go
import "github.com/prometheus/client_golang/prometheus"

var (
    operationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "plugin_operation_duration_seconds",
            Help: "Duration of plugin operations",
        },
        []string{"operation", "worker_id"},
    )
)
```

### Performance Optimization

1. **Use Prepared Statements**
2. **Implement Connection Pooling**
3. **Batch Operations When Possible**
4. **Profile Memory Usage**
5. **Monitor Goroutine Count**

## Examples

See the `examples/plugins/` directory for complete plugin examples:

- **Simple Example**: Basic plugin with SELECT operations
- **E-commerce Plugin**: Complex plugin with multiple operation types
- **Vector Plugin**: Plugin for pgvector operations
- **IMDB Plugin**: Plugin for complex analytical queries

Each example includes complete source code, configuration files, and documentation.

## Support

For support and questions:
- Create an issue in the GitHub repository
- Join the community discussions
- Refer to the API documentation
- Check the troubleshooting guide

Happy plugin development!
