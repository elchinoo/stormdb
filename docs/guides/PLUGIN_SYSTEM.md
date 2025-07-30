# Plugin System Guide

This guide covers the StormDB plugin system, including how to use existing plugins and develop custom ones.

## Plugin System Overview

StormDB uses a plugin architecture to support custom workloads beyond the built-in types. Plugins are compiled as shared libraries (.so files on Linux/macOS, .dll on Windows) and dynamically loaded at runtime.

### Key Features

- **Dynamic Loading**: Plugins are loaded at runtime without recompiling StormDB
- **Language Agnostic**: Plugins can be written in any language that supports C exports
- **Isolated Execution**: Each plugin runs in its own context
- **Hot Reloading**: Plugins can be reloaded without restarting StormDB
- **Configuration Integration**: Plugin settings integrate with StormDB configuration

## Available Plugins

### IMDB Plugin

Simulates movie database operations based on IMDb data.

**Features:**
- Movie and actor searches
- Rating and review operations
- Complex analytical queries
- Data loading from CSV files

**Usage:**
```bash
stormdb --config config_imdb.yaml --workload imdb_plugin
```

**Configuration:**
```yaml
workload:
  type: "plugin"
  plugin_name: "imdb_plugin"
  plugin_config:
    data_file: "/path/to/imdb.csv"
    operation_weights:
      movie_search: 40
      actor_search: 30
      rating_insert: 20
      review_insert: 10
```

### E-commerce Plugin

Simulates online store operations.

**Features:**
- Product catalog operations
- Order processing
- Customer management
- Inventory updates
- Payment processing simulation

**Usage:**
```bash
stormdb --config config_ecommerce.yaml --workload ecommerce_plugin
```

**Configuration:**
```yaml
workload:
  type: "plugin"
  plugin_name: "ecommerce_plugin"
  plugin_config:
    catalog_size: 10000
    customer_count: 1000
    operation_weights:
      product_search: 50
      add_to_cart: 20
      checkout: 15
      browse_catalog: 15
```

### Vector Plugin

Specialized for pgvector operations and similarity search.

**Features:**
- Vector similarity searches
- Embedding operations
- Index performance testing
- Batch vector operations

**Usage:**
```bash
stormdb --config config_vector.yaml --workload vector_plugin
```

**Configuration:**
```yaml
workload:
  type: "plugin"
  plugin_name: "vector_plugin"
  plugin_config:
    vector_dimension: 1024
    index_type: "hnsw"
    similarity_function: "cosine"
    operation_weights:
      similarity_search: 60
      vector_insert: 25
      vector_update: 15
```

### Real World Plugin

Simulates real-world application patterns.

**Features:**
- Mixed read/write patterns
- Complex transaction scenarios
- Realistic data distributions
- User session simulation

**Usage:**
```bash
stormdb --config config_realworld.yaml --workload realworld_plugin
```

## Plugin Management

### Discovery and Loading

```bash
# List all available workload types (including plugins)
stormdb --list-workloads

# List only plugins
stormdb --list-plugins

# Show detailed plugin information
stormdb --plugin-info imdb_plugin

# Scan for plugins in all directories
stormdb --scan-plugins
```

### Plugin Directories

StormDB searches for plugins in these locations:
1. `./plugins/` (current directory)
2. `~/.stormdb/plugins/` (user directory)
3. `/usr/local/lib/stormdb/plugins/` (system directory)
4. Directory specified by `STORMDB_PLUGIN_PATH` environment variable

```bash
# Set custom plugin path
export STORMDB_PLUGIN_PATH="/path/to/my/plugins"

# Multiple paths (Unix-style)
export STORMDB_PLUGIN_PATH="/path/to/plugins1:/path/to/plugins2"
```

### Building Plugins

```bash
# Build all plugins
make build-plugins

# Build specific plugin
make build-plugin PLUGIN=imdb_plugin

# Install plugins to system directory
sudo make install-plugins

# Install to user directory
make install-plugins PREFIX=~/.stormdb
```

## Plugin Development

### Plugin Interface

Plugins must implement the following interface:

```go
type WorkloadPlugin interface {
    // Initialize the plugin with configuration
    Initialize(config map[string]interface{}) error
    
    // Get plugin metadata
    GetInfo() PluginInfo
    
    // Set up the database schema and data
    Setup(db *sql.DB) error
    
    // Clean up the database (optional)
    Cleanup(db *sql.DB) error
    
    // Execute a single operation
    ExecuteOperation(db *sql.DB, operationType string) error
    
    // Get available operation types and their weights
    GetOperationTypes() map[string]int
    
    // Get custom metrics (optional)
    GetMetrics() map[string]interface{}
}
```

### Creating a Basic Plugin

#### 1. Project Structure

```
my_plugin/
├── go.mod
├── go.sum
├── main.go          # Plugin entry point
├── operations.go    # Operation implementations
├── setup.go         # Database setup
└── README.md
```

#### 2. Initialize Go Module

```bash
mkdir my_plugin
cd my_plugin
go mod init my_plugin

# Add StormDB dependency
go get github.com/elchinoo/stormdb/pkg/plugin
```

#### 3. Implement Plugin Interface

**main.go:**
```go
package main

import (
    "C"
    "database/sql"
    "fmt"
    
    "github.com/elchinoo/stormdb/pkg/plugin"
)

// Plugin implementation
type MyPlugin struct {
    config map[string]interface{}
}

// Initialize the plugin
func (p *MyPlugin) Initialize(config map[string]interface{}) error {
    p.config = config
    return nil
}

// Get plugin information
func (p *MyPlugin) GetInfo() plugin.PluginInfo {
    return plugin.PluginInfo{
        Name:        "my_plugin",
        Version:     "1.0.0",
        Description: "Custom workload plugin example",
        Author:      "Your Name",
    }
}

// Setup database schema
func (p *MyPlugin) Setup(db *sql.DB) error {
    schema := `
    CREATE TABLE IF NOT EXISTS my_table (
        id SERIAL PRIMARY KEY,
        name VARCHAR(100),
        value INTEGER,
        created_at TIMESTAMP DEFAULT NOW()
    );
    
    CREATE INDEX IF NOT EXISTS idx_my_table_name ON my_table(name);
    `
    
    _, err := db.Exec(schema)
    return err
}

// Clean up database
func (p *MyPlugin) Cleanup(db *sql.DB) error {
    _, err := db.Exec("DROP TABLE IF EXISTS my_table")
    return err
}

// Execute an operation
func (p *MyPlugin) ExecuteOperation(db *sql.DB, operationType string) error {
    switch operationType {
    case "insert":
        return p.insertOperation(db)
    case "select":
        return p.selectOperation(db)
    case "update":
        return p.updateOperation(db)
    default:
        return fmt.Errorf("unknown operation type: %s", operationType)
    }
}

// Get operation types and weights
func (p *MyPlugin) GetOperationTypes() map[string]int {
    return map[string]int{
        "insert": 30,
        "select": 60,
        "update": 10,
    }
}

// Get custom metrics
func (p *MyPlugin) GetMetrics() map[string]interface{} {
    return map[string]interface{}{
        "custom_metric_1": "value1",
        "custom_metric_2": 42,
    }
}

// Export functions for plugin interface
//export GetPlugin
func GetPlugin() *MyPlugin {
    return &MyPlugin{}
}

func main() {}
```

**operations.go:**
```go
package main

import (
    "database/sql"
    "fmt"
    "math/rand"
)

func (p *MyPlugin) insertOperation(db *sql.DB) error {
    name := fmt.Sprintf("name_%d", rand.Intn(1000))
    value := rand.Intn(100)
    
    _, err := db.Exec(
        "INSERT INTO my_table (name, value) VALUES ($1, $2)",
        name, value,
    )
    return err
}

func (p *MyPlugin) selectOperation(db *sql.DB) error {
    id := rand.Intn(1000) + 1
    
    var name string
    var value int
    err := db.QueryRow(
        "SELECT name, value FROM my_table WHERE id = $1",
        id,
    ).Scan(&name, &value)
    
    if err != nil && err != sql.ErrNoRows {
        return err
    }
    
    return nil
}

func (p *MyPlugin) updateOperation(db *sql.DB) error {
    id := rand.Intn(1000) + 1
    value := rand.Intn(100)
    
    _, err := db.Exec(
        "UPDATE my_table SET value = $1 WHERE id = $2",
        value, id,
    )
    return err
}
```

#### 4. Build the Plugin

**Makefile:**
```makefile
PLUGIN_NAME = my_plugin
BUILD_DIR = ../build/plugins

.PHONY: build clean

build:
	mkdir -p $(BUILD_DIR)
	go build -buildmode=plugin -o $(BUILD_DIR)/$(PLUGIN_NAME).so .

clean:
	rm -f $(BUILD_DIR)/$(PLUGIN_NAME).so

test:
	go test ./...

install: build
	cp $(BUILD_DIR)/$(PLUGIN_NAME).so ~/.stormdb/plugins/
```

Build the plugin:
```bash
make build
```

### Advanced Plugin Features

#### Configuration Handling

```go
func (p *MyPlugin) Initialize(config map[string]interface{}) error {
    p.config = config
    
    // Parse configuration with defaults
    if tableSize, ok := config["table_size"].(int); ok {
        p.tableSize = tableSize
    } else {
        p.tableSize = 1000 // default
    }
    
    if batchSize, ok := config["batch_size"].(int); ok {
        p.batchSize = batchSize
    } else {
        p.batchSize = 100 // default
    }
    
    return p.validateConfig()
}

func (p *MyPlugin) validateConfig() error {
    if p.tableSize <= 0 {
        return fmt.Errorf("table_size must be positive")
    }
    
    if p.batchSize <= 0 {
        return fmt.Errorf("batch_size must be positive")
    }
    
    return nil
}
```

#### Custom Metrics

```go
type MyPlugin struct {
    config         map[string]interface{}
    operationCount map[string]int64
    mutex          sync.RWMutex
}

func (p *MyPlugin) ExecuteOperation(db *sql.DB, operationType string) error {
    // Increment operation counter
    p.mutex.Lock()
    p.operationCount[operationType]++
    p.mutex.Unlock()
    
    // Execute operation
    return p.doOperation(db, operationType)
}

func (p *MyPlugin) GetMetrics() map[string]interface{} {
    p.mutex.RLock()
    defer p.mutex.RUnlock()
    
    metrics := make(map[string]interface{})
    for op, count := range p.operationCount {
        metrics[fmt.Sprintf("operations_%s", op)] = count
    }
    
    // Add custom business metrics
    metrics["table_size"] = p.getCurrentTableSize()
    metrics["avg_value"] = p.getAverageValue()
    
    return metrics
}
```

#### Data Loading

```go
func (p *MyPlugin) Setup(db *sql.DB) error {
    // Create schema
    if err := p.createSchema(db); err != nil {
        return err
    }
    
    // Load initial data
    if dataFile, ok := p.config["data_file"].(string); ok {
        return p.loadDataFromFile(db, dataFile)
    }
    
    // Generate synthetic data
    return p.generateSyntheticData(db)
}

func (p *MyPlugin) loadDataFromFile(db *sql.DB, filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()
    
    // Use COPY for efficient bulk loading
    stmt, err := db.Prepare("COPY my_table (name, value) FROM STDIN WITH CSV")
    if err != nil {
        return err
    }
    defer stmt.Close()
    
    // Process CSV file
    reader := csv.NewReader(file)
    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
        
        // Process record
        _, err = stmt.Exec(record[0], record[1])
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

#### Transaction Support

```go
func (p *MyPlugin) ExecuteOperation(db *sql.DB, operationType string) error {
    if operationType == "complex_transaction" {
        return p.executeTransaction(db)
    }
    
    return p.executeSimpleOperation(db, operationType)
}

func (p *MyPlugin) executeTransaction(db *sql.DB) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Multiple operations in transaction
    if err := p.insertInTransaction(tx); err != nil {
        return err
    }
    
    if err := p.updateInTransaction(tx); err != nil {
        return err
    }
    
    if err := p.selectInTransaction(tx); err != nil {
        return err
    }
    
    return tx.Commit()
}
```

## Plugin Configuration

### YAML Configuration

```yaml
workload:
  type: "plugin"
  plugin_name: "my_plugin"
  plugin_config:
    # Plugin-specific settings
    table_size: 10000
    batch_size: 500
    data_file: "/path/to/data.csv"
    
    # Operation weights
    operation_weights:
      insert: 30
      select: 60
      update: 10
    
    # Custom parameters
    custom_param1: "value1"
    custom_param2: 42
    nested_config:
      sub_param: "nested_value"
```

### Environment Variables

```bash
# Plugin-specific environment variables
export MY_PLUGIN_DATA_FILE="/path/to/data.csv"
export MY_PLUGIN_BATCH_SIZE="1000"

# Use in configuration
# plugin_config:
#   data_file: "${MY_PLUGIN_DATA_FILE}"
#   batch_size: "${MY_PLUGIN_BATCH_SIZE}"
```

## Testing Plugins

### Unit Testing

```go
// plugin_test.go
package main

import (
    "testing"
    "database/sql"
    _ "github.com/lib/pq"
)

func TestPluginInitialization(t *testing.T) {
    plugin := &MyPlugin{}
    
    config := map[string]interface{}{
        "table_size": 1000,
        "batch_size": 100,
    }
    
    err := plugin.Initialize(config)
    if err != nil {
        t.Fatalf("Plugin initialization failed: %v", err)
    }
    
    if plugin.tableSize != 1000 {
        t.Errorf("Expected table_size 1000, got %d", plugin.tableSize)
    }
}

func TestOperations(t *testing.T) {
    // Setup test database
    db, err := sql.Open("postgres", "postgres://test:test@localhost/test?sslmode=disable")
    if err != nil {
        t.Skip("Database not available")
    }
    defer db.Close()
    
    plugin := &MyPlugin{}
    config := map[string]interface{}{
        "table_size": 100,
    }
    
    if err := plugin.Initialize(config); err != nil {
        t.Fatalf("Plugin initialization failed: %v", err)
    }
    
    if err := plugin.Setup(db); err != nil {
        t.Fatalf("Plugin setup failed: %v", err)
    }
    
    // Test operations
    operations := []string{"insert", "select", "update"}
    for _, op := range operations {
        if err := plugin.ExecuteOperation(db, op); err != nil {
            t.Errorf("Operation %s failed: %v", op, err)
        }
    }
    
    // Cleanup
    plugin.Cleanup(db)
}
```

### Integration Testing

```bash
# Test plugin loading
stormdb --test-plugin ./build/plugins/my_plugin.so

# Test with minimal configuration
stormdb --config test_config.yaml --duration 30s

# Validate plugin metrics
stormdb --config test_config.yaml --duration 1m --format json | jq '.metrics'
```

## Plugin Distribution

### Packaging

```bash
# Create distribution package
mkdir -p dist/my_plugin
cp build/plugins/my_plugin.so dist/my_plugin/
cp README.md dist/my_plugin/
cp example_config.yaml dist/my_plugin/

# Create tarball
tar -czf my_plugin-1.0.0.tar.gz -C dist my_plugin
```

### Installation Script

```bash
#!/bin/bash
# install_plugin.sh

PLUGIN_NAME="my_plugin"
PLUGIN_VERSION="1.0.0"
INSTALL_DIR="$HOME/.stormdb/plugins"

# Create directory
mkdir -p "$INSTALL_DIR"

# Download and install
curl -L "https://releases.example.com/${PLUGIN_NAME}-${PLUGIN_VERSION}.tar.gz" | \
    tar -xz -C /tmp

cp "/tmp/${PLUGIN_NAME}/${PLUGIN_NAME}.so" "$INSTALL_DIR/"

echo "Plugin ${PLUGIN_NAME} ${PLUGIN_VERSION} installed successfully"
```

## Best Practices

### Development

1. **Error Handling**: Always handle errors gracefully
2. **Resource Management**: Properly close database connections and files
3. **Thread Safety**: Use appropriate synchronization for shared state
4. **Configuration Validation**: Validate configuration parameters
5. **Documentation**: Document plugin configuration and usage

### Performance

1. **Prepared Statements**: Use prepared statements for repeated queries
2. **Connection Reuse**: Reuse database connections efficiently
3. **Batch Operations**: Batch operations when possible
4. **Memory Management**: Avoid memory leaks in long-running tests

### Security

1. **Input Validation**: Validate all input parameters
2. **SQL Injection**: Use parameterized queries
3. **Resource Limits**: Implement appropriate resource limits
4. **Error Messages**: Don't expose sensitive information in error messages

### Testing

1. **Unit Tests**: Write comprehensive unit tests
2. **Integration Tests**: Test with real database connections
3. **Performance Tests**: Validate plugin performance
4. **Edge Cases**: Test error conditions and edge cases

## Troubleshooting

### Common Issues

**Plugin Not Found:**
```bash
# Check plugin path
echo $STORMDB_PLUGIN_PATH

# List available plugins
stormdb --scan-plugins

# Verify file exists and is executable
ls -la plugins/my_plugin.so
```

**Plugin Loading Errors:**
```bash
# Check plugin dependencies
ldd plugins/my_plugin.so  # Linux
otool -L plugins/my_plugin.so  # macOS

# Test plugin loading
stormdb --test-plugin plugins/my_plugin.so
```

**Configuration Errors:**
```bash
# Validate configuration
stormdb --config config.yaml --validate

# Show resolved configuration
stormdb --config config.yaml --show-config
```

### Debugging

```bash
# Enable debug logging
stormdb --config config.yaml --log-level debug

# Plugin-specific debugging
stormdb --config config.yaml --plugin-debug

# Save detailed logs
stormdb --config config.yaml --log-file plugin_debug.log
```

## Next Steps

- [Configuration Guide](CONFIGURATION.md) - Configure plugin settings
- [Usage Guide](USAGE.md) - Command-line options for plugins
- [Performance Optimization](PERFORMANCE_OPTIMIZATION.md) - Optimize plugin performance
- [Examples](../examples/plugins/) - Sample plugin implementations
