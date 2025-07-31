# Memory-Efficient Configuration Examples for TPCC Data Loading
# 
# The data_loading section controls memory usage during bulk operations
# Each customer record uses approximately 200 bytes of memory

# LOW MEMORY CONFIGURATION (Small VPS, containers with <1GB RAM)
data_loading:
  batch_size: 10000        # 10K records = ~2MB memory usage
  max_memory_mb: 50        # Strict 50MB limit

# MEDIUM MEMORY CONFIGURATION (Standard servers with 2-8GB RAM)  
data_loading:
  batch_size: 50000        # 50K records = ~10MB memory usage
  max_memory_mb: 100       # 100MB limit

# HIGH MEMORY CONFIGURATION (Powerful servers with 16GB+ RAM)
data_loading:
  batch_size: 250000       # 250K records = ~50MB memory usage
  max_memory_mb: 500       # 500MB limit

# MAXIMUM PERFORMANCE (High-memory systems, no memory constraints)
data_loading:
  batch_size: 1000000      # 1M records = ~200MB memory usage
  max_memory_mb: 1000      # 1GB limit

# Memory Usage Calculation:
# - Each customer record: ~200 bytes
# - Batch memory = batch_size * 200 bytes
# - The system will use the smaller of batch_size or max_memory_mb constraint

# Scale Impact Examples:
# Scale 10:  3M customers  = 300 batches @ 10K/batch OR 60 batches @ 50K/batch
# Scale 50:  15M customers = 1500 batches @ 10K/batch OR 300 batches @ 50K/batch
# Scale 100: 30M customers = 3000 batches @ 10K/batch OR 600 batches @ 50K/batch
