# IMDB Dataset Workload Plugin

The IMDB workload plugin simulates realistic movie database operations using a schema modeled after the Internet Movie Database. It provides multiple distinct testing modes for comprehensive database performance analysis.

## Plugin Architecture

The IMDB workload is implemented as a **plugin** in StormDB's modular architecture:

- **Plugin Location**: `plugins/imdb_plugin/`
- **Binary Output**: `build/plugins/imdb_plugin.so`
- **Requirements**: PostgreSQL 12+
- **Auto-loading**: Automatically discovered when plugins are enabled

## 🎬 Schema Overview

### Tables Structure
```sql
movies (1000+ records)
├── id, title, release_year, genre, rating
├── director, budget, revenue, runtime
└── created_at, updated_at

actors (500+ records)  
├── id, name, birth_year, nationality
└── created_at

movie_actors (junction table)
├── movie_id, actor_id, role, billing_order
└── created_at

reviews (3000+ records)
├── id, movie_id, user_name, rating
├── review_text, helpful_votes
└── created_at
```

### Optimized Indexes
- Genre, rating, and year indexes on movies
- Name index on actors  
- Foreign key indexes on all relationships
- Composite indexes for common query patterns

## 🚀 Workload Types

### 1. READ-ONLY (`imdb_read`)
**Purpose**: Intensive read testing with complex analytical queries

**Operations**:
- `searchMoviesByGenre`: Find movies by genre with rating sorting
- `getMovieDetails`: Detailed movie info with cast information
- `getTopRatedMovies`: Highest rated movies across all genres
- `getActorMovies`: Complete filmography for specific actors
- `getMovieReviews`: User reviews with helpfulness ranking
- `searchMoviesByYear`: Movies by release year with metadata
- `getRecentReviews`: Latest reviews across all movies

**Configuration**:
```yaml
workload: "imdb_read"
workers: 4              # Multiple workers for read scaling
connections: 8          # Higher connection pool for reads
scale: 1000            # Large dataset for realistic testing
```

**Performance Profile**:
- High throughput (150+ TPS)
- Sub-millisecond average latency
- Complex JOIN operations
- Aggregation and sorting workloads

### 2. WRITE-ONLY (`imdb_write`)
**Purpose**: Intensive write testing with inserts and updates

**Operations**:
- `insertNewReview`: Add user reviews with ratings
- `updateMovieRating`: Update movie average ratings
- `insertNewMovie`: Add new movies with metadata
- `updateReviewHelpfulness`: Increment helpful vote counts
- `insertNewActor`: Add new actors to database
- `addMovieActor`: Link actors to movies with roles

**Configuration**:
```yaml
workload: "imdb_write"
workers: 2              # Fewer workers to avoid write contention
connections: 4          # Balanced for write operations
scale: 500             # Smaller initial dataset
```

**Performance Profile**:
- Moderate throughput (50-100 TPS)
- Higher latency due to write operations
- Transaction safety and consistency
- Foreign key constraint validation

### 3. MIXED (`imdb_mixed`)
**Purpose**: Realistic application simulation with balanced operations

**Operation Mix**:
- 70% READ operations (all read types)
- 30% WRITE operations (all write types)

**Configuration**:
```yaml
workload: "imdb_mixed"
workers: 3              # Balanced worker count
connections: 6          # Adequate for mixed operations  
scale: 750             # Medium dataset
duration: "90s"        # Longer test for mixed patterns
```

**Performance Profile**:
- Balanced throughput (75-125 TPS)
- Variable latency based on operation mix
- Realistic application usage patterns
- Read/write contention scenarios

## 📊 Usage Examples

### Quick Testing
```bash
# Test read performance
./stormdb -c config/config_imdb_read.yaml

# Write workload
./stormdb -c config/config_imdb_write.yaml

# Mixed workload
./stormdb -c config/config_imdb_mixed.yaml
```

### Setup and Rebuild
```bash
# Setup schema and data (first time)
./stormdb -c config/config_imdb_read.yaml --setup

# Rebuild from scratch
./stormdb -c config/config_imdb_read.yaml --rebuild
```

### Interactive Demo
```bash
./demo_imdb_workloads.sh
```

## 🔍 Performance Analysis

### Key Metrics to Monitor

**READ Workload**:
- **TPS**: 150-200+ (high read throughput)
- **Latency P95**: <3ms (fast query response)
- **Operations**: Search patterns, JOIN performance
- **Bottlenecks**: Index effectiveness, query optimization

**WRITE Workload**:
- **TPS**: 50-100 (write-limited throughput)  
- **Latency P95**: 5-15ms (transaction overhead)
- **Operations**: Insert/update performance
- **Bottlenecks**: Lock contention, constraint validation

**MIXED Workload**:
- **TPS**: 75-150 (balanced throughput)
- **Latency Distribution**: Bimodal (fast reads, slower writes)
- **Operations**: Real-world usage simulation
- **Bottlenecks**: Read/write contention, connection pooling

### Sample Output
```
--- RESULTS ---
Duration: 60s
Workers: 4
Total Transactions: 9240
TPS: 154.00
QPS: 154.00
Rows Read/sec: 154.00
Latency Percentiles (ms): P50=0.41 P90=1.55 P95=2.62 P99=5.93
Query Errors: 0

Latency Histogram:
  <= 0.5ms: 4066    (44% - very fast queries)
  <= 1.0ms: 1341    (14% - fast queries)
  <= 2.0ms: 295     (3% - moderate queries)
  <= 5.0ms: 427     (5% - acceptable queries)
  <= 10.0ms: 74     (1% - slower queries)
```

## 🎯 Use Cases

### Database Performance Testing
- **Indexing Strategy**: Test different index configurations
- **Query Optimization**: Identify slow query patterns  
- **Connection Pooling**: Optimize pool sizes for workload types
- **Resource Scaling**: Test performance under different loads

### Application Development
- **ORM Performance**: Test object-relational mapping efficiency
- **Caching Strategy**: Identify cacheable query patterns
- **Read Replica Testing**: Validate read/write splitting
- **Connection Management**: Test application connection handling

### Infrastructure Testing
- **Hardware Sizing**: Determine optimal server configurations
- **Cloud Performance**: Test managed database performance
- **Backup Impact**: Measure performance during backup operations
- **Failover Testing**: Validate high availability configurations

## 🔧 Customization

### Scaling Data Size
```yaml
scale: 2000  # 2000 movies, 1000 actors, 6000 reviews
```

### Adjusting Operation Mix (Mixed Mode)
Edit `imdb.go` to modify the read/write ratio:
```go
if rng.Intn(100) < 80 { // 80% reads, 20% writes
```

### Adding Custom Operations
Implement new operations in `operations.go`:
```go
func (w *IMDBWorkload) customOperation(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
    // Your custom database operation
}
```

The IMDB workload provides comprehensive, realistic database testing scenarios that mirror real-world application patterns! 🎬📊
