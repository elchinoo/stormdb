# Simple Example Plugin

This is a demonstration plugin for StormDB that implements a simple counter workload.

## Building the Plugin

```bash
go build -buildmode=plugin -o simple_example.so main.go
```

## Using the Plugin

1. Build the plugin as shown above
2. Copy `simple_example.so` to your StormDB plugins directory
3. Configure StormDB to use the plugin:

```yaml
plugins:
  paths:
    - "./plugins"
  auto_load: true

workload: "simple_counter"  # or "simple_counter_read", "simple_counter_write"
```

## Workload Description

The simple counter workload creates a table with named counters and performs:

- **Read operations**: Select random counter values
- **Write operations**: Increment random counters
- **Mixed mode**: 70% reads, 30% writes

This workload is ideal for testing basic database performance and understanding the plugin system.
