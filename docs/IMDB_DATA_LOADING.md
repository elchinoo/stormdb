# IMDB Workload Data Loading Modes

The IMDB workload now supports three different data loading modes to provide flexibility in how you populate your test database.

## Data Loading Modes

### 1. Generate Mode (Default)
**Mode**: `generate`
**Description**: Generates synthetic IMDB-like data using stormdb's built-in data generator.

```yaml
data_loading:
  mode: "generate"
  # No filepath needed
```

**Features**:
- Creates realistic movie, actor, and user data
- Scales based on the `scale` parameter in your config
- Fast setup for testing and development
- No external file dependencies

### 2. Dump Mode
**Mode**: `dump`
**Description**: Loads data from a PostgreSQL dump file using `pg_restore`.

```yaml
data_loading:
  mode: "dump"
  filepath: "/path/to/your/imdb.dump"
```

**Requirements**:
- PostgreSQL dump file (created with `pg_dump`)
- `pg_restore` command available on the system
- Database credentials with restore permissions

**Features**:
- Fast loading of large datasets
- Preserves exact data structure and relationships
- Best for production-like testing with real data volumes

### 3. SQL Mode
**Mode**: `sql`
**Description**: Executes SQL statements from a script file.

```yaml
data_loading:
  mode: "sql"
  filepath: "/path/to/your/imdb.sql"
```

**Requirements**:
- SQL script file with INSERT/UPDATE statements
- Compatible with PostgreSQL SQL syntax

**Features**:
- Maximum flexibility for custom data loading
- Supports complex data transformations
- Good for loading from different data sources

## Configuration Examples

### Example 1: Using Your Dump File
```yaml
database:
  type: postgres
  host: "localhost"
  port: 5432
  dbname: "test"
  username: "storm_user"
  password: "storm_pwd!123"
  sslmode: "disable"

data_loading:
  mode: "dump"
  filepath: "/Users/storm_user/data/imdb_sample.dump"

workload: "imdb_mixed"
scale: 1000      # Ignored when using dump/sql modes
duration: "90s"
workers: 3
connections: 6
```

### Example 2: Using SQL Script
```yaml
data_loading:
  mode: "sql"
  filepath: "/Users/storm_user/data/imdb_data.sql"
```

### Example 3: Generate Synthetic Data (Default)
```yaml
# No data_loading section needed, or explicitly specify:
data_loading:
  mode: "generate"
```

## Usage Tips

1. **For Development**: Use `generate` mode for quick testing and development
2. **For Production Testing**: Use `dump` mode with real production data dumps
3. **For Custom Data**: Use `sql` mode when you need to load from specific sources

## Error Handling

- File not found errors will be reported clearly
- `pg_restore` errors will show detailed output
- SQL parsing errors will indicate which statement failed

## Performance Notes

- **Dump mode**: Fastest for large datasets (uses `pg_restore`)
- **SQL mode**: Slower for large files due to statement-by-statement execution
- **Generate mode**: Fast for moderate scales, memory-efficient

## Security Considerations

- Dump and SQL files should be stored securely
- Database credentials are passed via environment variables to external tools
- File paths are validated before execution
