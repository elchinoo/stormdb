# IMDB Workload Plugin

This plugin provides IMDB movie database workloads for StormDB with complex queries and realistic data patterns.

## Supported Workload Types

- `imdb` - Mixed read/write workload (70% read, 30% write)
- `imdb_read` - Read-heavy workload with complex movie queries
- `imdb_write` - Write-heavy workload with movie data updates
- `imdb_mixed` - Balanced read/write workload

## Building the Plugin

```bash
go build -buildmode=plugin -o imdb_plugin.so main.go
```

## Configuration

Use any of the supported workload types in your StormDB configuration:

```yaml
workload: "imdb_mixed"
scale: 5000  # Number of movies to generate
```

## Data Loading Options

The plugin supports three data loading modes:

- `generate` - Generate synthetic movie data
- `dump` - Load from PostgreSQL dump file  
- `sql` - Load from SQL script file

Configure via the `data_loading` section in your configuration file.
