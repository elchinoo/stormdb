# Plugin System Troubleshooting Guide

## Common Issues and Solutions

### Plugin Runtime Compatibility Issues

#### Problem: "plugin was built with a different version of package internal/runtime/sys"

This error occurs when plugins were built with a different Go runtime version than the main application.

**Root Cause**: Go plugins are extremely sensitive to build environment differences, including:
- Different Go versions (even patch versions)
- Different compilation flags
- Different operating system environments
- Different library versions

**Solutions**:

1. **Rebuild All Plugins** (Recommended)
   ```bash
   # Clean existing plugins
   make clean
   
   # Rebuild everything with current Go environment
   make build-all
   ```

2. **Development Environment Setup**
   ```bash
   # Ensure consistent Go version across team
   go version
   
   # Use exact same build environment
   make clean && make build-all
   ```

3. **CI/CD Pipeline**
   ```bash
   # In CI, always build plugins and main app together
   - name: Build StormDB
     run: |
       make clean
       make build-all
       make test
   ```

### Plugin Name Conflicts

#### Problem: "plugin name conflict: xxx already loaded from yyy"

This occurs when multiple plugins declare the same name or when the same plugin is loaded multiple times.

**Solutions**:

1. **Check Plugin Names**
   ```bash
   # List all available plugins
   pgstorm plugins list
   ```

2. **Unique Plugin Names**
   Ensure each plugin has a unique name in its metadata:
   ```go
   func (p *MyPlugin) GetMetadata() *plugin.PluginMetadata {
       return &plugin.PluginMetadata{
           Name:         "unique_plugin_name", // Must be unique
           Version:      "1.0.0",
           WorkloadTypes: []string{"my_workload"},
       }
   }
   ```

3. **Plugin Path Configuration**
   ```yaml
   plugins:
     paths:
       - "./plugins"           # Source plugins
       - "./build/plugins"     # Built plugins
     enable_discovery: true
   ```

### Plugin Loading Failures

#### Problem: Plugins fail to load silently

**Diagnostic Steps**:

1. **Enable Verbose Logging**
   ```bash
   pgstorm progressive --config test.yaml --verbose
   ```

2. **Check Plugin Paths**
   ```bash
   ls -la build/plugins/
   ```

3. **Verify Plugin Symbols**
   ```bash
   nm -D build/plugins/plugin_name.so | grep WorkloadPlugin
   ```

### Development vs Production Considerations

#### Development Environment
- Plugins may fail due to frequent rebuilds
- Use `make clean && make build-all` for consistency
- Consider disabling plugin loading for unit tests

#### Production Environment
- Build all components in same environment
- Use container builds for consistency
- Implement graceful fallbacks

### Plugin Architecture Best Practices

1. **Graceful Degradation**
   ```go
   // In your application
   factory, err := workload.NewFactory(cfg)
   if err != nil {
       log.Warn("Plugin system unavailable, using basic functionality")
       // Fall back to core functionality
   }
   ```

2. **Plugin Health Checks**
   ```go
   // Regular plugin health monitoring
   for _, plugin := range pluginLoader.ListPlugins() {
       if err := plugin.HealthCheck(); err != nil {
           log.Warn("Plugin %s health check failed: %v", plugin.Name, err)
       }
   }
   ```

3. **Error Recovery**
   ```go
   // Implement retry logic for transient failures
   func (pl *PluginLoader) LoadPluginWithRetry(name string, maxRetries int) error {
       for i := 0; i < maxRetries; i++ {
           if err := pl.LoadPlugin(name); err == nil {
               return nil
           }
           time.Sleep(time.Second * time.Duration(i+1))
       }
       return fmt.Errorf("failed to load plugin after %d retries", maxRetries)
   }
   ```

## Integration with Testing

### Unit Tests
```go
// Skip plugin tests in unit testing
func TestWorkloadFactory(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping plugin tests in short mode")
    }
    // Plugin-dependent test code
}
```

### Integration Tests
```go
// Graceful plugin test handling
func TestPluginIntegration(t *testing.T) {
    factory, err := workload.NewFactory(cfg)
    if err != nil {
        t.Skip("Plugin system unavailable, skipping integration test")
    }
    // Test with available plugins
}
```

### CI/CD Pipeline
```yaml
# Example GitHub Actions workflow
- name: Build and Test
  run: |
    make clean
    make build-all
    make test
    
# Alternative: Skip plugin tests in CI
- name: Unit Tests Only
  run: make test -short
```

## Monitoring and Observability

### Plugin Metrics
```go
// Add plugin loading metrics
prometheus.CounterVec{
    Name: "plugin_load_attempts_total",
    Help: "Number of plugin load attempts",
    LabelNames: []string{"plugin_name", "status"},
}
```

### Health Endpoints
```go
// HTTP endpoint for plugin health
func PluginHealthHandler(w http.ResponseWriter, r *http.Request) {
    status := make(map[string]string)
    for _, plugin := range pluginLoader.ListPlugins() {
        if plugin.Loaded {
            status[plugin.Name] = "healthy"
        } else {
            status[plugin.Name] = "failed"
        }
    }
    json.NewEncoder(w).Encode(status)
}
```

## Future Improvements

1. **Plugin Metadata Caching**
   - Store plugin metadata separately from binaries
   - Avoid loading plugins just for discovery

2. **Hot Plugin Reloading**
   - Implement safe plugin reloading mechanisms
   - Support plugin updates without restart

3. **Plugin Sandboxing**
   - Isolate plugin execution
   - Implement resource limits

4. **Alternative Plugin Systems**
   - Consider WASM-based plugins for better portability
   - Evaluate RPC-based plugin architectures

For more information, see the [Plugin Development Guide](PLUGIN_DEVELOPMENT.md).
