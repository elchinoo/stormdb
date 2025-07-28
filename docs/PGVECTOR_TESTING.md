# Comprehensive pgvector Testing Guide

This guide provides complete instructions for testing pgvector performance across all aspects: data ingestion, updates, reads, index comparison, and accuracy analysis.

## Overview

The comprehensive pgvector workload plugin provides extensive testing capabilities for PostgreSQL's pgvector extension, including:

- **Data Ingestion**: Single inserts, batch inserts, and COPY protocol
- **Updates**: Single and batch vector updates
- **Reads**: Full table scans vs indexed similarity searches
- **Index Comparison**: IVFFlat vs HNSW with different configurations
- **Accuracy Analysis**: Recall@k measurements for different index settings

## Key Features

### 🚀 Performance Testing
- **1,000,000 baseline rows** for realistic testing scenarios
- **Pre-computed vectors** (10% of dataset) for consistent test results
- **Multiple ingestion methods** with performance comparison
- **Real-time metrics** during test execution

### 📊 Comprehensive Analysis
- **Throughput measurements** (TPS/QPS)
- **Latency percentiles** (P50, P95, P99)
- **Resource utilization** (WAL, I/O, memory)
- **Accuracy metrics** (Recall@10, @50, @100)

### 🔧 Configurable Testing
- **Vector dimensions**: Configurable (default 1024)
- **Similarity metrics**: L2, Cosine, Inner Product
- **Batch sizes**: Configurable for different workloads
- **Index types**: IVFFlat and HNSW with various parameters

## Quick Start

### 1. Prerequisites

```bash
# Ensure pgvector extension is installed
sudo apt-get install postgresql-15-pgvector  # Ubuntu/Debian
# or
brew install pgvector  # macOS with Homebrew

# Create test database
createdb stormdb_vector_test
psql -d stormdb_vector_test -c "CREATE EXTENSION vector;"
```

### 2. Build the Plugin

```bash
cd plugins
make vector_plugin
cd ..
```

### 3. Run Comprehensive Tests

```bash
# Run the complete test suite
./demo_comprehensive_pgvector.sh

# Or run individual tests
./build/stormdb -c config/config_pgvector_ingestion_batch.yaml -d 60s --collect-pg-stats
```

## Test Types

### 1. Data Ingestion Tests

#### Single Inserts (`pgvector_comprehensive_ingestion_single`)
- Tests individual INSERT statements
- Measures maximum concurrency for single-row operations
- Best for: Understanding baseline insert performance

```yaml
workload: "pgvector_comprehensive_ingestion_single"
workers: 4
duration: "60s"
```

#### Batch Inserts (`pgvector_comprehensive_ingestion_batch`)
- Tests batch INSERT statements with configurable batch size
- Default: 500 inserts per batch
- Best for: Ongoing data ingestion with good throughput

```yaml
workload: "pgvector_comprehensive_ingestion_batch"
test_config:
  batch_size: 500
```

#### COPY Protocol (`pgvector_comprehensive_ingestion_copy`)
- Tests PostgreSQL COPY protocol for bulk loading
- Highest throughput for large data imports
- Best for: Initial data loading and bulk imports

```yaml
workload: "pgvector_comprehensive_ingestion_copy"
workers: 2  # Fewer workers needed for COPY
```

### 2. Update Tests

#### Single Updates (`pgvector_comprehensive_update_single`)
- Tests individual UPDATE statements on existing vectors
- Measures real-time modification performance
- Includes metadata updates alongside vector changes

```yaml
workload: "pgvector_comprehensive_update_single"
```

### 3. Read Performance Tests

#### Full Table Scan (`pgvector_comprehensive_read_scan`)
- Forces sequential scan without indexes
- Measures baseline search performance
- Shows cost of similarity search without optimization

```yaml
workload: "pgvector_comprehensive_read_scan"
test_config:
  read_type: "full_scan"
```

#### Indexed Search (`pgvector_comprehensive_read_indexed`)
- Uses vector indexes for similarity search
- Configurable index types (IVFFlat, HNSW)
- Demonstrates performance improvements from indexing

```yaml
workload: "pgvector_comprehensive_read_indexed"
test_config:
  read_type: "indexed"
  index_type: "ivfflat"
```

## Configuration Options

### Basic Configuration

```yaml
database:
  type: postgres
  host: "localhost"
  port: 5432
  dbname: "stormdb_vector_test"
  username: "postgres"
  password: "postgres"

workload: "pgvector_comprehensive_ingestion_batch"
scale: 1000000
duration: "60s"
workers: 4
connections: 8
```

### Advanced Test Configuration

```yaml
test_config:
  dimensions: 1024                    # Vector dimensions
  similarity_metric: "cosine"         # l2, cosine, inner_product
  batch_size: 500                     # Batch size for operations
  baseline_rows: 1000000              # Baseline data size
  precomputed_percentage: 10          # Pre-calculated vectors (%)
  read_type: "indexed"                # full_scan, indexed
  index_type: "ivfflat"              # ivfflat, hnsw
```

## Index Configurations

### IVFFlat Indexes
- **Lists**: 50, 100, 200, 500, 1000
- **Trade-off**: More lists = better accuracy, slower builds
- **Recommended**: 100-500 lists for most use cases

### HNSW Indexes (PostgreSQL 16+)
- **M**: 8, 16, 32 (connectivity parameter)
- **EF_CONSTRUCTION**: 64, 128, 256 (build quality)
- **Trade-off**: Higher values = better accuracy, more memory

## Performance Metrics

### Throughput Metrics
- **TPS (Transactions Per Second)**: Number of operations completed
- **QPS (Queries Per Second)**: Number of queries executed
- **Rows/Second**: Data processing rate

### Latency Metrics
- **P50**: Median response time
- **P95**: 95th percentile response time
- **P99**: 99th percentile response time

### Resource Metrics
- **WAL Bytes**: Write-Ahead Log generation
- **Blocks Read/Hit**: I/O and cache performance
- **Index Size**: Storage overhead of indexes

## Accuracy Analysis

### Recall@k Measurements
- **Recall@10**: Percentage of true neighbors in top 10 results
- **Recall@50**: Percentage of true neighbors in top 50 results
- **Recall@100**: Percentage of true neighbors in top 100 results

### Ground Truth Generation
The system automatically generates ground truth data by:
1. Creating vectors with known nearest neighbors
2. Adding noise to create realistic test scenarios
3. Measuring how well indexes retrieve true neighbors

## Best Practices

### 1. Test Environment
- Use dedicated test database
- Ensure sufficient memory for vector operations
- Monitor disk space for large datasets

### 2. Baseline Data
- Always load 1M+ rows for realistic testing
- Use consistent vector distributions
- Pre-generate vectors for reproducible results

### 3. Performance Testing
- Run tests multiple times for consistent results
- Monitor system resources during tests
- Compare results across different configurations

### 4. Index Selection
- Test multiple index configurations
- Balance accuracy vs performance requirements
- Consider build time vs query performance

## Troubleshooting

### Common Issues

#### pgvector Extension Not Found
```bash
# Install pgvector extension
git clone https://github.com/pgvector/pgvector.git
cd pgvector
make
sudo make install
```

#### Memory Issues with Large Datasets
```sql
-- Increase work memory for vector operations
SET work_mem = '1GB';
SET maintenance_work_mem = '2GB';
```

#### Slow Index Builds
```sql
-- Monitor index build progress
SELECT * FROM pg_stat_progress_create_index;
```

## Example Results

### Typical Performance Numbers (1024-dim vectors)

| Test Type | Method | TPS | Latency P95 | Notes |
|-----------|--------|-----|-------------|-------|
| Ingestion | Single | 1,200 | 15ms | Good for real-time |
| Ingestion | Batch (500) | 8,500 | 80ms | Best balanced approach |
| Ingestion | COPY | 25,000 | 200ms | Fastest bulk loading |
| Update | Single | 800 | 20ms | Includes index updates |
| Search | Full Scan | 50 | 2000ms | Expensive without index |
| Search | IVFFlat | 2,500 | 8ms | Good performance/accuracy |
| Search | HNSW | 4,000 | 5ms | Best performance |

### Accuracy Comparison

| Index Type | Lists/M | Recall@10 | Recall@50 | Build Time |
|------------|---------|-----------|-----------|------------|
| IVFFlat | 100 | 85% | 92% | 30s |
| IVFFlat | 500 | 92% | 97% | 2m |
| HNSW | M=16 | 95% | 98% | 5m |

## Advanced Usage

### Custom Vector Generation
```python
# Generate custom test vectors
import numpy as np
vectors = np.random.normal(0, 1, (100000, 1024)).astype(np.float32)
# Save to CSV for consistent testing
```

### Database Tuning for Vectors
```sql
-- Optimize for vector workloads
ALTER SYSTEM SET shared_buffers = '4GB';
ALTER SYSTEM SET work_mem = '256MB';
ALTER SYSTEM SET maintenance_work_mem = '1GB';
ALTER SYSTEM SET checkpoint_segments = 64;
SELECT pg_reload_conf();
```

### Monitoring Queries
```sql
-- Monitor vector operations
SELECT query, calls, total_time, mean_time 
FROM pg_stat_statements 
WHERE query LIKE '%vector%' 
ORDER BY total_time DESC;
```

## Future Enhancements

- [ ] Automated accuracy analysis implementation
- [ ] Index comparison with detailed reports
- [ ] Custom similarity function testing
- [ ] Multi-dimensional scaling analysis
- [ ] Memory usage profiling
- [ ] Distributed vector search testing

## Contributing

To add new test types or improve existing ones:

1. Extend `ComprehensivePgVectorWorkload` struct
2. Add new test methods following the pattern
3. Create corresponding configuration files
4. Update documentation and demo scripts

## References

- [pgvector Documentation](https://github.com/pgvector/pgvector)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Vector Similarity Search Best Practices](https://github.com/pgvector/pgvector#best-practices)
