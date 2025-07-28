# RealWorld Workload Plugin

This plugin provides realistic enterprise application workloads for StormDB with complex business logic patterns.

## Supported Workload Types

- `realworld` - Mixed enterprise workload 
- `realworld_read` - Read-heavy enterprise queries
- `realworld_write` - Write-heavy enterprise operations
- `realworld_mixed` - Balanced read/write workload
- `realworld_oltp` - OLTP-focused transactional workload
- `realworld_analytics` - Analytics-focused reporting workload

## Building the Plugin

```bash
go build -buildmode=plugin -o realworld_plugin.so main.go
```

## Configuration

Use any of the supported workload types in your StormDB configuration:

```yaml
workload: "realworld_mixed"
scale: 1000  # Dataset size parameter
```

## Workload Characteristics

- **OLTP Mode**: High-frequency transactional operations
- **Analytics Mode**: Complex queries with aggregations and joins
- **Mixed Mode**: Realistic blend of transactional and analytical workloads
- **Enterprise Patterns**: User management, permissions, audit trails
