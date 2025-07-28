# StormDB Plugin Development Guide

This guide explains how to create custom workload plugins for StormDB, allowing you to extend the benchmarking tool with your own test scenarios without modifying the core application.

## Plugin System Overview

StormDB uses Go's plugin system to dynamically load workload plugins at runtime. Plugins are compiled as shared libraries (`.so` files on Linux/macOS, `.dll` on Windows) and can be loaded from paths specified in configuration files.

### Key Benefits:
- **Extensibility**: Create custom workloads for specific use cases
- **No Recompilation**: Add new workloads without rebuilding StormDB
- **Distribution**: Share workloads as standalone plugin files
- **Isolation**: Plugin failures don't crash the main application
- **Versioning**: Each plugin maintains its own version and dependencies

## Plugin Interface

All workload plugins must implement the `WorkloadPlugin` interface:

```go
type WorkloadPlugin interface {
    // GetMetadata returns information about this plugin
    GetMetadata() *PluginMetadata
    
    // CreateWorkload creates a new workload instance for the specified type
    CreateWorkload(workloadType string) (Workload, error)
    
    // Initialize is called once when the plugin is loaded
    Initialize() error
    
    // Cleanup is called when the plugin is being unloaded
    Cleanup() error
}
```

Each plugin must also provide workload implementations that satisfy the `Workload` interface:

```go
type Workload interface {
    // Cleanup drops tables and reloads data (called only with --rebuild)
    Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error
    
    // Setup ensures schema exists (called with --setup or --rebuild)
    Setup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error
    
    // Run executes the load test
    Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error
}
```

## Creating a Plugin

### 1. Project Structure

Create a new Go module for your plugin:

```bash
mkdir my-stormdb-plugin
cd my-stormdb-plugin
go mod init my-stormdb-plugin
```

### 2. Plugin Implementation

Create `main.go` with your plugin implementation:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "math/rand"
    "time"
    
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

// Import the required types from StormDB
// Note: You'll need to import these from the actual StormDB module
// For this example, we'll define them locally

type PluginMetadata struct {
    Name        string   `json:"name"`
    Version     string   `json:"version"`
    Description string   `json:"description"`
    Author      string   `json:"author"`
    WorkloadTypes []string `json:"workload_types"`
    RequiredExtensions []string `json:"required_extensions,omitempty"`
    MinPostgreSQLVersion string `json:"min_postgresql_version,omitempty"`
    Homepage    string   `json:"homepage,omitempty"`
}

type WorkloadPlugin interface {
    GetMetadata() *PluginMetadata
    CreateWorkload(workloadType string) (Workload, error)
    Initialize() error
    Cleanup() error
}

type Workload interface {
    Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *Config) error
    Setup(ctx context.Context, db *pgxpool.Pool, cfg *Config) error
    Run(ctx context.Context, db *pgxpool.Pool, cfg *Config, metrics *Metrics) error
}

// Your plugin implementation
type MyWorkloadPlugin struct{}

func (p *MyWorkloadPlugin) GetMetadata() *PluginMetadata {
    return &PluginMetadata{
        Name:        "my_custom_workload",
        Version:     "1.0.0",
        Description: "Custom workload for specific testing scenarios",
        Author:      "Your Name <your.email@example.com>",
        WorkloadTypes: []string{
            "my_workload",
            "my_workload_read",
            "my_workload_write",
        },
        RequiredExtensions: []string{}, // e.g., ["pgvector", "pg_stat_statements"]
        MinPostgreSQLVersion: "13.0",
        Homepage: "https://github.com/yourname/my-stormdb-plugin",
    }
}

func (p *MyWorkloadPlugin) CreateWorkload(workloadType string) (Workload, error) {
    switch workloadType {
    case "my_workload":
        return &MyWorkload{Mode: "mixed"}, nil
    case "my_workload_read":
        return &MyWorkload{Mode: "read"}, nil
    case "my_workload_write":
        return &MyWorkload{Mode: "write"}, nil
    default:
        return nil, fmt.Errorf("unsupported workload type: %s", workloadType)
    }
}

func (p *MyWorkloadPlugin) Initialize() error {
    log.Printf("Initializing My Custom Workload Plugin v1.0.0")
    return nil
}

func (p *MyWorkloadPlugin) Cleanup() error {
    log.Printf("Cleaning up My Custom Workload Plugin")
    return nil
}

// Your workload implementation
type MyWorkload struct {
    Mode string
}

func (w *MyWorkload) Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *Config) error {
    log.Printf("Cleaning up MyWorkload schema")
    
    _, err := db.Exec(ctx, `
        DROP TABLE IF EXISTS my_test_table CASCADE;
    `)
    
    return err
}

func (w *MyWorkload) Setup(ctx context.Context, db *pgxpool.Pool, cfg *Config) error {
    log.Printf("Setting up MyWorkload schema")
    
    _, err := db.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS my_test_table (
            id SERIAL PRIMARY KEY,
            data TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT NOW()
        );
        
        CREATE INDEX IF NOT EXISTS idx_my_test_table_created_at 
        ON my_test_table(created_at);
    `)
    
    if err != nil {
        return fmt.Errorf("failed to create schema: %w", err)
    }
    
    // Load initial data if needed
    return w.loadInitialData(ctx, db, cfg)
}

func (w *MyWorkload) loadInitialData(ctx context.Context, db *pgxpool.Pool, cfg *Config) error {
    // Check if data already exists
    var count int
    err := db.QueryRow(ctx, "SELECT COUNT(*) FROM my_test_table").Scan(&count)
    if err != nil {
        return err
    }
    
    if count > 0 {
        log.Printf("MyWorkload: Using existing data (%d rows)", count)
        return nil
    }
    
    log.Printf("MyWorkload: Loading initial data (scale: %d)", cfg.Scale)
    
    // Load test data based on scale factor
    batch := &pgx.Batch{}
    recordCount := cfg.Scale * 1000 // 1000 records per scale unit
    
    for i := 0; i < recordCount; i++ {
        data := fmt.Sprintf("Test data record %d", i)
        batch.Queue("INSERT INTO my_test_table (data) VALUES ($1)", data)
    }
    
    br := db.SendBatch(ctx, batch)
    defer br.Close()
    
    for i := 0; i < recordCount; i++ {
        _, err := br.Exec()
        if err != nil {
            return fmt.Errorf("failed to insert data at record %d: %w", i, err)
        }
    }
    
    log.Printf("MyWorkload: Loaded %d records", recordCount)
    return nil
}

func (w *MyWorkload) Run(ctx context.Context, db *pgxpool.Pool, cfg *Config, metrics *Metrics) error {
    // Run workload based on mode
    switch w.Mode {
    case "read":
        return w.runReadWorkload(ctx, db, cfg, metrics)
    case "write":
        return w.runWriteWorkload(ctx, db, cfg, metrics)
    case "mixed":
        // Randomly choose between read and write operations
        if rand.Float64() < 0.7 { // 70% reads, 30% writes
            return w.runReadWorkload(ctx, db, cfg, metrics)
        } else {
            return w.runWriteWorkload(ctx, db, cfg, metrics)
        }
    default:
        return fmt.Errorf("unsupported workload mode: %s", w.Mode)
    }
}

func (w *MyWorkload) runReadWorkload(ctx context.Context, db *pgxpool.Pool, cfg *Config, metrics *Metrics) error {
    start := time.Now()
    
    // Example read operation
    var id int
    var data string
    var createdAt time.Time
    
    err := db.QueryRow(ctx, `
        SELECT id, data, created_at 
        FROM my_test_table 
        ORDER BY RANDOM() 
        LIMIT 1
    `).Scan(&id, &data, &createdAt)
    
    duration := time.Since(start)
    
    if err != nil {
        metrics.RecordError()
        return fmt.Errorf("read operation failed: %w", err)
    }
    
    metrics.RecordOperation(duration)
    return nil
}

func (w *MyWorkload) runWriteWorkload(ctx context.Context, db *pgxpool.Pool, cfg *Config, metrics *Metrics) error {
    start := time.Now()
    
    // Example write operation
    data := fmt.Sprintf("New data at %s", time.Now().Format(time.RFC3339))
    
    _, err := db.Exec(ctx, `
        INSERT INTO my_test_table (data) VALUES ($1)
    `, data)
    
    duration := time.Since(start)
    
    if err != nil {
        metrics.RecordError()
        return fmt.Errorf("write operation failed: %w", err)
    }
    
    metrics.RecordOperation(duration)
    return nil
}

// Export the plugin symbol - this is required for the plugin system
var WorkloadPlugin MyWorkloadPlugin
```

### 3. Building the Plugin

Build your plugin as a shared library:

```bash
# Linux/macOS
go build -buildmode=plugin -o my_workload.so main.go

# Windows
go build -buildmode=plugin -o my_workload.dll main.go
```

### 4. Configuration

Create a configuration file that includes your plugin:

```yaml
database:
  type: postgres
  host: localhost
  port: 5432
  dbname: test
  username: postgres
  password: postgres
  sslmode: disable

plugins:
  paths:
    - "./plugins"
  files:
    - "./my_workload.so"
  auto_load: true

workload: "my_workload"
mode: "mixed"
scale: 10
duration: "60s"
workers: 4
connections: 10
```

### 5. Running with Your Plugin

```bash
./stormdb -c config_with_plugin.yaml
```

## Best Practices

### Error Handling
- Always return descriptive errors
- Use context for cancellation support
- Handle database connection failures gracefully

### Performance
- Use connection pooling efficiently
- Batch operations when possible
- Avoid blocking operations in the Run method

### Logging
- Use structured logging
- Log important events (setup, errors, significant operations)
- Avoid excessive logging in hot paths

### Configuration
- Support all standard StormDB configuration options
- Use the Scale parameter to control data volume
- Respect the Mode parameter for different operation types

### Testing
- Test your plugin with different scale factors
- Verify cleanup operations work correctly
- Test error conditions and recovery

## Advanced Features

### Custom Metrics
You can extend the metrics system to track custom statistics:

```go
func (w *MyWorkload) Run(ctx context.Context, db *pgxpool.Pool, cfg *Config, metrics *Metrics) error {
    start := time.Now()
    
    // Your operation here
    result := w.performCustomOperation(ctx, db)
    
    duration := time.Since(start)
    
    // Record standard metrics
    metrics.RecordOperation(duration)
    
    // Record custom metrics (if supported)
    if customMetrics, ok := metrics.(*CustomMetrics); ok {
        customMetrics.RecordCustomStat("my_custom_metric", result.Value)
    }
    
    return nil
}
```

### Multi-Phase Operations
For complex workloads, you can implement multiple phases:

```go
func (w *MyWorkload) Run(ctx context.Context, db *pgxpool.Pool, cfg *Config, metrics *Metrics) error {
    // Phase 1: Setup transaction
    tx, err := db.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)
    
    // Phase 2: Multiple related operations
    err = w.performPhase1(ctx, tx)
    if err != nil {
        return err
    }
    
    err = w.performPhase2(ctx, tx)
    if err != nil {
        return err
    }
    
    // Phase 3: Commit transaction
    return tx.Commit(ctx)
}
```

## Troubleshooting

### Plugin Loading Issues
- Ensure the plugin exports the `WorkloadPlugin` symbol
- Check that all required dependencies are available
- Verify the plugin was built for the correct architecture

### Runtime Errors
- Check PostgreSQL logs for database-specific errors
- Use StormDB's verbose logging to see plugin operations
- Validate your SQL statements and data types

### Performance Issues
- Monitor connection pool usage
- Check for long-running transactions
- Use EXPLAIN ANALYZE to optimize queries

## Examples Repository

For more examples and templates, visit: https://github.com/elchinoo/stormdb-plugin-examples

This repository contains:
- Complete plugin templates
- Real-world example plugins
- Testing utilities
- CI/CD configurations for plugin builds
