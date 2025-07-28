# Vector Workload Plugin

This plugin provides high-dimensional vector similarity search testing for StormDB using pgvector extension.

## Supported Workload Types

- `vector_1024` - L2 distance similarity search with 1024-dimensional vectors
- `vector_1024_cosine` - Cosine similarity search with 1024-dimensional vectors  
- `vector_1024_inner` - Inner product similarity search with 1024-dimensional vectors

## Requirements

- PostgreSQL with pgvector extension installed
- Minimum PostgreSQL version: 12.0

## Building the Plugin

```bash
go build -buildmode=plugin -o vector_plugin.so main.go
```

## Configuration

Use any of the supported workload types in your StormDB configuration:

```yaml
workload: "vector_1024_cosine" 
scale: 10000  # Number of vectors to generate
```

## Performance Notes

Vector operations can be computationally intensive. Consider:
- Using appropriate indexes (HNSW, IVFFlat)
- Tuning vector dimensions based on your use case
- Monitoring memory usage for large vector datasets
