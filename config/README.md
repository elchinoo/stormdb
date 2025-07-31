# StormDB Configuration Templates

This directory contains consolidated configuration templates for different workload types. Each template includes multiple examples with different scenarios, all in one file for easier management.

## Configuration Files Overview

### 🏪 `workload_ecommerce.yaml`
E-commerce workload simulations with real-world database usage patterns
- **ecommerce_mixed**: Balanced read/write operations (Default)
- **ecommerce_read**: Read-heavy scenarios (product browsing, searches)
- **ecommerce_write**: Write-intensive scenarios (orders, inventory)
- **ecommerce_analytics**: Analytical/reporting workloads
- **ecommerce_basic**: Simple e-commerce operations
- **Progressive scaling**: Long-term scaling tests

### 📊 `workload_tpcc.yaml`
TPC-C standard OLTP benchmark configurations
- **Basic TPC-C**: Small scale development testing (Default)
- **Performance TPC-C**: Medium scale performance testing
- **Progressive TPC-C**: Large scale with progressive scaling
- Includes transaction mix configuration and performance tuning

### 🎬 `workload_imdb.yaml`
Internet Movie Database workload with complex queries and large datasets
- **imdb_mixed**: Balanced read/write operations (Default)
- **imdb_read**: Read-intensive scenarios (searches, analytics)
- **imdb_write**: Write-heavy scenarios (data ingestion, updates)
- **imdb_dump**: Bulk data loading operations
- **Progressive scaling**: Long-term scaling analysis

### 🔍 `workload_pgvector.yaml`
PostgreSQL vector database operations for AI/ML workloads
- **vector_cosine**: Cosine similarity search (Default)
- **vector_inner**: Inner product similarity
- **vector_ingestion**: Single inserts, batch inserts, COPY operations
- **vector_read**: Indexed and sequential scan searches
- **vector_update**: Vector update operations
- **Progressive scaling**: Vector workload scaling

### 🔧 `workload_simple.yaml`
Basic connectivity and simple query workloads for testing
- **simple**: Basic connectivity testing (Default)
- **connection**: Connection overhead testing
- **transient**: Frequent connection creation/destruction
- **synchronized**: Synchronized load testing
- **minimal**: Minimal resource testing

### 🎭 `workload_demo.yaml`
Demonstration and showcase configurations
- **monitoring_showcase**: Demonstrates monitoring features (Default)
- **feature_showcase**: Shows all StormDB features
- **progressive_demo**: Progressive scaling demonstration
- **backend_demo**: Database backend features
- **plugin_demo**: Plugin system demonstration
- **strategy_demo**: Different scaling strategies

## How to Use

### 1. Choose Your Workload Template
Copy the appropriate template for your use case:
```bash
cp workload_tpcc.yaml my_tpcc_test.yaml
```

### 2. Uncomment Your Desired Configuration
Each template has one active configuration (uncommented) and several example configurations (commented). To switch configurations:

1. Comment out the current active configuration
2. Uncomment the configuration you want to use
3. Adjust parameters as needed

### 3. Configure Database Connection
Update the database connection settings in the `database` section:
```yaml
database:
  type: postgres
  host: "your-host"
  port: 5432
  dbname: "your-database"
  username: "your-username"
  password: "your-password"
  sslmode: "disable"
```

### 4. Enable Optional Features
Uncomment sections as needed:
- **Results Backend**: For comprehensive analytics and result storage
- **Test Metadata**: For tracking and organizing test results
- **PostgreSQL Monitoring**: For enhanced database statistics
- **Progressive Scaling**: For advanced scaling analysis

## Configuration Sections

### 🔌 Database Connection
Basic PostgreSQL connection parameters. Required for all workloads.

### 📈 Results Backend (Optional)
Stores test results in a separate PostgreSQL database for analytics:
- Comprehensive test result tracking
- Long-term performance analysis
- PostgreSQL statistics storage
- Configurable retention policies

### 🏷️ Test Metadata (Optional)
Organizes and tracks test executions:
- Test names and descriptions
- Environment identification
- Tagging system for categorization
- Notes and documentation

### 🔧 Plugin Configuration
Configures the plugin system for specialized workloads:
- Plugin search paths
- Specific plugin files
- Auto-loading behavior

### ⚡ Progressive Scaling (Optional)
Advanced scaling configuration for long-term tests:
- Linear, exponential, or step scaling strategies
- Memory management for long-running tests
- Comprehensive analysis and reporting
- Configurable test bands and durations

### 📊 Monitoring and Metrics
PostgreSQL statistics collection and performance metrics:
- pg_stat_statements integration
- Custom latency percentiles
- Real-time metrics display
- Export capabilities

## Examples

### Quick TPC-C Test
```bash
# Use the default basic TPC-C configuration
./pgstorm -config config/workload_tpcc.yaml
```

### E-commerce Read-Heavy Test
1. Edit `workload_ecommerce.yaml`
2. Comment out the default `ecommerce_mixed` section
3. Uncomment the `ecommerce_read` section
4. Run: `./pgstorm -config config/workload_ecommerce.yaml`

### Progressive Scaling with Analytics
1. Choose your workload template
2. Uncomment the `results_backend` section
3. Uncomment the `progressive` section
4. Configure database connections
5. Run your test

## Migration from Old Configuration Files

The new templates replace the numerous individual configuration files. Here's the mapping:

### Old → New Mapping
- `config_tpcc*.yaml` → `workload_tpcc.yaml`
- `config_ecommerce*.yaml` → `workload_ecommerce.yaml`
- `config_imdb*.yaml` → `workload_imdb.yaml`
- `config_pgvector*.yaml` → `workload_pgvector.yaml`
- `config_simple*.yaml` → `workload_simple.yaml`
- `config_*demo*.yaml` → `workload_demo.yaml`

### Benefits of Consolidation
✅ **Reduced file count**: From 40+ files to 6 templates  
✅ **Better organization**: Related configurations in one place  
✅ **Easier maintenance**: Single file per workload type  
✅ **Complete examples**: Multiple scenarios with full context  
✅ **Documentation**: Comprehensive comments and explanations  
✅ **Consistency**: Standardized structure across all workloads  

## Advanced Features

### Memory Management
Progressive scaling tests include memory management:
```yaml
progressive:
  max_latency_samples: 50000  # Limit samples per band
  memory_limit_mb: 512        # Total memory limit
```

### Database Backend Analytics
Store results for long-term analysis:
```yaml
results_backend:
  enabled: true
  retention_days: 90
  store_raw_metrics: true
  store_pg_stats: true
```

### Plugin System
Load specialized workload plugins:
```yaml
plugins:
  files:
    - "./build/plugins/ecommerce_plugin.so"
  auto_load: true
```

## Support

For questions about configuration templates:
1. Check the comments in each template file
2. Review the workload-specific configuration sections
3. See the original configuration files (in the old format) for reference
4. Consult the main StormDB documentation
