// Package types provides core data structures and interfaces for the StormDB
// PostgreSQL benchmarking tool. This package defines configuration, metrics,
// and statistical data types used throughout the application.
//
// The types package serves as the central contract between different components
// of StormDB, including workload generators, metrics collectors, and reporting
// systems. It ensures type safety and consistency across the entire benchmark
// framework.
package types

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Config represents the complete configuration for a StormDB benchmark run.
// It encompasses database connection parameters, workload specifications,
// performance settings, and monitoring options.
//
// The configuration is typically loaded from YAML files and validated before
// being used to initialize benchmark runs. Different workloads may utilize
// different subsets of the configuration options.
//
// Example usage:
//
//	cfg := &Config{}
//	// Load from YAML and run benchmark
type Config struct {
	// Database contains PostgreSQL connection and authentication parameters
	Database struct {
		Type     string `mapstructure:"type"`     // Database type (currently only "postgres")
		Host     string `mapstructure:"host"`     // PostgreSQL server hostname or IP
		Port     int    `mapstructure:"port"`     // PostgreSQL server port (default: 5432)
		Dbname   string `mapstructure:"dbname"`   // Target database name
		Username string `mapstructure:"username"` // Authentication username
		Password string `mapstructure:"password"` // Authentication password
		Sslmode  string `mapstructure:"sslmode"`  // SSL connection mode (disable/require/prefer)
	} `mapstructure:"database"`

	// DataLoading configures how sample data is loaded for specific workloads.
	// This is primarily used by the IMDB workload which supports multiple
	// data loading strategies including generated data, SQL dumps, and SQL scripts.
	DataLoading struct {
		Mode     string `mapstructure:"mode"`     // Loading mode: "generate", "dump", or "sql"
		FilePath string `mapstructure:"filepath"` // Path to dump/sql file when mode is "dump" or "sql"
	} `mapstructure:"data_loading"`

	// Core benchmark configuration
	Workload        string `mapstructure:"workload"`         // Workload type (imdb, ecommerce, tpcc, etc.)
	Mode            string `mapstructure:"mode"`             // Workload mode (read, write, mixed) for applicable workloads
	Scale           int    `mapstructure:"scale"`            // Scale factor for data generation
	Duration        string `mapstructure:"duration"`         // Benchmark duration (e.g., "5m", "30s")
	Workers         int    `mapstructure:"workers"`          // Number of concurrent worker threads
	Connections     int    `mapstructure:"connections"`      // Maximum database connections in pool
	SummaryInterval string `mapstructure:"summary_interval"` // Interval for progress reports (e.g., "10s", "30s")

	// PostgreSQL monitoring and statistics collection options
	CollectPgStats    bool `mapstructure:"collect_pg_stats"`    // Enable comprehensive PostgreSQL statistics collection
	PgStatsStatements bool `mapstructure:"pg_stats_statements"` // Enable pg_stat_statements query analysis

	// Connection management strategy for performance testing
	ConnectionMode string `mapstructure:"connection_mode"` // "persistent", "transient", or "mixed" for connection overhead analysis

	// Plugin system configuration
	Plugins struct {
		// Paths to search for plugin files (.so, .dll, .dylib)
		Paths []string `mapstructure:"paths"`
		// Specific plugin files to load (absolute or relative paths)
		Files []string `mapstructure:"files"`
		// Auto-load all plugins found in search paths
		AutoLoad bool `mapstructure:"auto_load"`
	} `mapstructure:"plugins"`
}

// PostgreSQLStats contains comprehensive PostgreSQL database statistics
// collected asynchronously during benchmark execution. These statistics provide
// insights into database performance, resource utilization, and system health.
//
// The statistics are gathered from various PostgreSQL system views including
// pg_stat_database, pg_stat_bgwriter, pg_stat_checkpointer, pg_stat_wal, and
// optionally pg_stat_statements. Collection is version-aware and automatically
// adapts to PostgreSQL versions 15-18+.
//
// All statistics are collected with thread-safe operations and can be safely
// accessed concurrently during benchmark execution.
type PostgreSQLStats struct {
	// Buffer cache statistics - measure memory efficiency and I/O patterns
	BufferCacheHitRatio float64 // Percentage of blocks served from buffer cache vs disk reads
	BlocksRead          int64   // Total blocks read from disk (cache misses)
	BlocksHit           int64   // Total blocks served from buffer cache (cache hits)
	BlocksWritten       int64   // Total blocks written by background writer and backends

	// Write-Ahead Log (WAL) statistics - measure transaction log activity
	WALRecords int64 // Total number of WAL records generated
	WALBytes   int64 // Total bytes of WAL data generated

	// Checkpoint statistics - measure checkpoint frequency and performance
	CheckpointsReq   int64 // Number of requested (manual) checkpoints
	CheckpointsTimed int64 // Number of scheduled (automatic) checkpoints

	// Temporary file statistics - indicate memory pressure and query complexity
	TempFiles int64 // Number of temporary files created for operations that exceeded work_mem
	TempBytes int64 // Total bytes of temporary file space used

	// Lock contention and concurrency statistics
	Deadlocks     int64 // Total number of deadlocks detected
	LockWaitCount int64 // Number of lock wait events (indicates contention)

	// Connection utilization statistics
	ActiveConnections int // Current number of active database connections
	MaxConnections    int // Maximum allowed connections (from PostgreSQL configuration)

	// Maintenance operation statistics
	AutovacuumCount int64 // Number of autovacuum operations performed

	// Query performance statistics (requires pg_stat_statements extension)
	TopQueries []QueryStats // Top queries by execution time or frequency

	// Metadata for statistics collection
	LastUpdated time.Time    // Timestamp of last statistics update
	mu          sync.RWMutex // Mutex protecting concurrent access to statistics
}

// QueryStats represents performance statistics for a single SQL query
// collected from PostgreSQL's pg_stat_statements extension. This data helps
// identify query performance patterns, optimization opportunities, and
// resource consumption patterns.
//
// QueryStats is typically used in conjunction with TopQueries to provide
// insights into the most resource-intensive or frequently executed queries
// during benchmark execution.
type QueryStats struct {
	Query       string    // The SQL query text (may be normalized/parameterized)
	Calls       int64     // Number of times this query was executed
	TotalTime   float64   // Total execution time across all calls (milliseconds)
	MeanTime    float64   // Average execution time per call (milliseconds)
	Rows        int64     // Total number of rows processed by this query
	HitPercent  float64   // Buffer cache hit percentage for this query
	LastUpdated time.Time // When these statistics were last collected
}

// ConnectionModeMetrics tracks performance metrics for a specific database
// connection management strategy. This is particularly useful for analyzing
// the performance impact of different connection patterns (persistent vs
// transient connections).
//
// The metrics are collected separately for each connection mode to enable
// direct comparison of connection overhead, latency impacts, and resource
// utilization patterns between different connection strategies.
//
// Thread Safety: All fields are protected by the embedded mutex and should
// be accessed through the provided methods for concurrent safety.
type ConnectionModeMetrics struct {
	TPS             int64        // Successfully completed transactions per second
	TPSAborted      int64        // Failed/aborted transactions per second
	QPS             int64        // Total queries executed per second
	Errors          int64        // Total number of errors encountered
	TransactionDur  []int64      // Individual transaction durations (nanoseconds)
	ConnectionSetup []int64      // Connection establishment times for transient connections (nanoseconds)
	ConnectionCount int64        // Total number of connections created (relevant for transient mode)
	Mu              sync.RWMutex // Mutex protecting concurrent access to all metrics
}

type Metrics struct {
	// Transaction metrics
	TPS        int64 // Total committed transactions
	TPSAborted int64 // Total aborted/failed transactions

	// Query metrics by type
	QPS           int64 // Total queries executed
	SelectQueries int64 // SELECT queries
	InsertQueries int64 // INSERT queries
	UpdateQueries int64 // UPDATE queries
	DeleteQueries int64 // DELETE queries

	RowsRead       int64
	RowsModified   int64
	Errors         int64
	ErrorTypes     map[string]int64
	TransactionDur []int64 // in nanoseconds

	// Optional: per-transaction counters
	NewOrderCount    int64
	PaymentCount     int64
	OrderStatusCount int64
	ThinkCount       int64

	// Latency histogram buckets (in milliseconds)
	LatencyHistogram map[string]int64 // bucket_name -> count

	// Per-worker metrics tracking
	WorkerMetrics map[int]*WorkerStats // worker_id -> stats

	// Time-series metrics tracking
	TimeSeries     *TimeSeriesMetrics
	BucketInterval time.Duration // Interval for time buckets (e.g., 1s, 5s)

	// PostgreSQL statistics (collected asynchronously)
	PgStats *PostgreSQLStats

	// Connection mode metrics (for connection overhead testing)
	PersistentConnMetrics *ConnectionModeMetrics // Metrics for persistent connections
	TransientConnMetrics  *ConnectionModeMetrics // Metrics for transient connections

	// Mutex to protect slices and maps
	Mu sync.Mutex // Protects slices and maps
}

// TimeSeriesMetrics tracks metrics over time buckets
type TimeSeriesMetrics struct {
	Buckets       []TimeBucket // Time-ordered buckets
	CurrentBucket *TimeBucket  // Currently active bucket
	StartTime     time.Time    // When collection started
	Mu            sync.RWMutex // Protects time series data
}

// TimeBucket represents metrics for a specific time interval
type TimeBucket struct {
	StartTime    time.Time
	EndTime      time.Time
	QPS          int64
	TPS          int64
	Errors       int64
	RowsRead     int64
	RowsModified int64
	Latencies    []int64 // All latencies in this bucket (ns)

	// Query-level metrics
	RowsPerQuery []int64 // Rows returned/modified per query
	StmtsPerTxn  []int   // Statements per transaction
	RowsPerTxn   []int64 // Rows touched per transaction

	// Query type breakdown
	SelectQueries int64
	InsertQueries int64
	UpdateQueries int64
	DeleteQueries int64

	// Query-specific row counts
	SelectRows []int64 // Rows returned per SELECT query
	UpdateRows []int64 // Rows modified per UPDATE query
	InsertRows []int64 // Rows inserted per INSERT query
	DeleteRows []int64 // Rows deleted per DELETE query
}

// WorkerStats tracks metrics for an individual worker
type WorkerStats struct {
	WorkerID       int
	TPS            int64      // Committed transactions for this worker
	TPSAborted     int64      // Aborted transactions for this worker
	QPS            int64      // Total queries for this worker
	Errors         int64      // Errors for this worker
	TransactionDur []int64    // Latencies in nanoseconds for this worker
	Mu             sync.Mutex // Protects this worker's data
}

// LatencyBucket defines histogram bucket boundaries (in milliseconds)
var LatencyBuckets = []float64{
	0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 20.0, 50.0, 100.0, 200.0, 500.0, 1000.0,
}

// GetLatencyBucket returns the bucket name for a given latency in nanoseconds
func GetLatencyBucket(latencyNs int64) string {
	latencyMs := float64(latencyNs) / 1e6 // Convert ns to ms

	for _, bucket := range LatencyBuckets {
		if latencyMs <= bucket {
			return fmt.Sprintf("%.1fms", bucket)
		}
	}
	return "+inf"
}

// InitializeLatencyHistogram initializes the histogram buckets
func (m *Metrics) InitializeLatencyHistogram() {
	m.Mu.Lock()
	defer m.Mu.Unlock()

	if m.LatencyHistogram == nil {
		m.LatencyHistogram = make(map[string]int64)
	}

	// Initialize all buckets to 0
	for _, bucket := range LatencyBuckets {
		bucketName := fmt.Sprintf("%.1fms", bucket)
		m.LatencyHistogram[bucketName] = 0
	}
	m.LatencyHistogram["+inf"] = 0
}

// RecordLatency records a latency measurement in the histogram
func (m *Metrics) RecordLatency(latencyNs int64) {
	bucket := GetLatencyBucket(latencyNs)

	m.Mu.Lock()
	m.LatencyHistogram[bucket]++
	m.Mu.Unlock()
}

// RecordQuery increments the appropriate query counter based on query type
func (m *Metrics) RecordQuery(queryType string) {
	atomic.AddInt64(&m.QPS, 1)

	switch queryType {
	case "SELECT":
		atomic.AddInt64(&m.SelectQueries, 1)
	case "INSERT":
		atomic.AddInt64(&m.InsertQueries, 1)
	case "UPDATE":
		atomic.AddInt64(&m.UpdateQueries, 1)
	case "DELETE":
		atomic.AddInt64(&m.DeleteQueries, 1)
	}
}

// RecordTransaction increments transaction counters
func (m *Metrics) RecordTransaction(success bool) {
	if success {
		atomic.AddInt64(&m.TPS, 1)
	} else {
		atomic.AddInt64(&m.TPSAborted, 1)
	}
}

// GetQueryType extracts the query type from a SQL statement
func GetQueryType(query string) string {
	query = strings.TrimSpace(strings.ToUpper(query))

	if strings.HasPrefix(query, "SELECT") || strings.HasPrefix(query, "WITH") {
		return "SELECT"
	} else if strings.HasPrefix(query, "INSERT") {
		return "INSERT"
	} else if strings.HasPrefix(query, "UPDATE") {
		return "UPDATE"
	} else if strings.HasPrefix(query, "DELETE") {
		return "DELETE"
	}

	return "OTHER"
}

// InitializeWorkerMetrics initializes per-worker tracking for the given number of workers
func (m *Metrics) InitializeWorkerMetrics(numWorkers int) {
	m.Mu.Lock()
	defer m.Mu.Unlock()

	if m.WorkerMetrics == nil {
		m.WorkerMetrics = make(map[int]*WorkerStats)
	}

	for i := 0; i < numWorkers; i++ {
		m.WorkerMetrics[i] = &WorkerStats{
			WorkerID:       i,
			TransactionDur: make([]int64, 0),
		}
	}
}

// RecordWorkerTransaction records transaction metrics for a specific worker
func (m *Metrics) RecordWorkerTransaction(workerID int, success bool, latencyNs int64) {
	// Also record in global metrics
	m.RecordTransaction(success)

	// Record latency globally
	m.Mu.Lock()
	m.TransactionDur = append(m.TransactionDur, latencyNs)
	m.Mu.Unlock()
	m.RecordLatency(latencyNs)

	// Record per-worker metrics
	if worker, exists := m.WorkerMetrics[workerID]; exists {
		worker.Mu.Lock()
		defer worker.Mu.Unlock()

		if success {
			atomic.AddInt64(&worker.TPS, 1)
		} else {
			atomic.AddInt64(&worker.TPSAborted, 1)
		}
		worker.TransactionDur = append(worker.TransactionDur, latencyNs)
	}
}

// RecordWorkerQuery records query metrics for a specific worker
func (m *Metrics) RecordWorkerQuery(workerID int, queryType string) {
	// Also record in global metrics
	m.RecordQuery(queryType)

	// Record per-worker metrics
	if worker, exists := m.WorkerMetrics[workerID]; exists {
		atomic.AddInt64(&worker.QPS, 1)
	}
}

// InitializeTimeSeries initializes time-series metrics collection
func (m *Metrics) InitializeTimeSeries(bucketInterval time.Duration) {
	m.BucketInterval = bucketInterval
	m.TimeSeries = &TimeSeriesMetrics{
		Buckets:   make([]TimeBucket, 0),
		StartTime: time.Now(),
	}

	// Create first bucket
	m.TimeSeries.CurrentBucket = &TimeBucket{
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(bucketInterval),
		Latencies:    make([]int64, 0),
		RowsPerQuery: make([]int64, 0),
		StmtsPerTxn:  make([]int, 0),
		RowsPerTxn:   make([]int64, 0),
		SelectRows:   make([]int64, 0),
		UpdateRows:   make([]int64, 0),
		InsertRows:   make([]int64, 0),
		DeleteRows:   make([]int64, 0),
	}
}

// RotateBucketIfNeeded checks if current bucket should be rotated
func (m *Metrics) RotateBucketIfNeeded() {
	if m.TimeSeries == nil {
		return
	}

	m.TimeSeries.Mu.Lock()
	defer m.TimeSeries.Mu.Unlock()

	now := time.Now()
	if now.After(m.TimeSeries.CurrentBucket.EndTime) {
		// Archive current bucket
		m.TimeSeries.Buckets = append(m.TimeSeries.Buckets, *m.TimeSeries.CurrentBucket)

		// Create new bucket
		m.TimeSeries.CurrentBucket = &TimeBucket{
			StartTime:    now,
			EndTime:      now.Add(m.BucketInterval),
			Latencies:    make([]int64, 0),
			RowsPerQuery: make([]int64, 0),
			StmtsPerTxn:  make([]int, 0),
			RowsPerTxn:   make([]int64, 0),
			SelectRows:   make([]int64, 0),
			UpdateRows:   make([]int64, 0),
			InsertRows:   make([]int64, 0),
			DeleteRows:   make([]int64, 0),
		}
	}
}

// FinalizeTimeSeries finalizes the current bucket for analysis
func (m *Metrics) FinalizeTimeSeries() {
	if m.TimeSeries == nil || m.TimeSeries.CurrentBucket == nil {
		return
	}

	m.TimeSeries.Mu.Lock()
	defer m.TimeSeries.Mu.Unlock()

	// Only finalize if the current bucket has data
	if m.TimeSeries.CurrentBucket.TPS > 0 || len(m.TimeSeries.CurrentBucket.Latencies) > 0 {
		// Set the end time to now for the final bucket
		m.TimeSeries.CurrentBucket.EndTime = time.Now()
		m.TimeSeries.Buckets = append(m.TimeSeries.Buckets, *m.TimeSeries.CurrentBucket)
	}
}

// RecordTimeSeriesTransaction records transaction metrics in current bucket
func (m *Metrics) RecordTimeSeriesTransaction(success bool, latencyNs int64, stmtsCount int, rowsCount int64) {
	if m.TimeSeries == nil {
		return
	}

	m.RotateBucketIfNeeded()

	m.TimeSeries.Mu.Lock()
	defer m.TimeSeries.Mu.Unlock()

	bucket := m.TimeSeries.CurrentBucket
	if success {
		atomic.AddInt64(&bucket.TPS, 1)
	}

	// Record QPS (every transaction involves at least one query)
	atomic.AddInt64(&bucket.QPS, 1)

	bucket.Latencies = append(bucket.Latencies, latencyNs)
	bucket.StmtsPerTxn = append(bucket.StmtsPerTxn, stmtsCount)
	bucket.RowsPerTxn = append(bucket.RowsPerTxn, rowsCount)
}

// RecordTimeSeriesQuery records query metrics in current bucket
func (m *Metrics) RecordTimeSeriesQuery(queryType string, rowsAffected int64) {
	if m.TimeSeries == nil {
		return
	}

	m.RotateBucketIfNeeded()

	m.TimeSeries.Mu.Lock()
	defer m.TimeSeries.Mu.Unlock()

	bucket := m.TimeSeries.CurrentBucket
	atomic.AddInt64(&bucket.QPS, 1)
	bucket.RowsPerQuery = append(bucket.RowsPerQuery, rowsAffected)

	switch queryType {
	case "SELECT":
		atomic.AddInt64(&bucket.SelectQueries, 1)
		bucket.SelectRows = append(bucket.SelectRows, rowsAffected)
		atomic.AddInt64(&bucket.RowsRead, rowsAffected)
	case "INSERT":
		atomic.AddInt64(&bucket.InsertQueries, 1)
		bucket.InsertRows = append(bucket.InsertRows, rowsAffected)
		atomic.AddInt64(&bucket.RowsModified, rowsAffected)
	case "UPDATE":
		atomic.AddInt64(&bucket.UpdateQueries, 1)
		bucket.UpdateRows = append(bucket.UpdateRows, rowsAffected)
		atomic.AddInt64(&bucket.RowsModified, rowsAffected)
	case "DELETE":
		atomic.AddInt64(&bucket.DeleteQueries, 1)
		bucket.DeleteRows = append(bucket.DeleteRows, rowsAffected)
		atomic.AddInt64(&bucket.RowsModified, rowsAffected)
	}
}

// RecordTimeSeriesError records error in current bucket
func (m *Metrics) RecordTimeSeriesError() {
	if m.TimeSeries == nil {
		return
	}

	m.RotateBucketIfNeeded()

	m.TimeSeries.Mu.Lock()
	defer m.TimeSeries.Mu.Unlock()

	atomic.AddInt64(&m.TimeSeries.CurrentBucket.Errors, 1)
}

// RecordWorkerError records error metrics for a specific worker
func (m *Metrics) RecordWorkerError(workerID int) {
	// Also record in global metrics
	atomic.AddInt64(&m.Errors, 1)

	// Record per-worker metrics
	if worker, exists := m.WorkerMetrics[workerID]; exists {
		atomic.AddInt64(&worker.Errors, 1)
	}
}

// UpdatePgStats updates PostgreSQL statistics (thread-safe)
func (m *Metrics) UpdatePgStats(stats *PostgreSQLStats) {
	if m.PgStats == nil {
		m.PgStats = &PostgreSQLStats{}
	}

	m.PgStats.mu.Lock()
	defer m.PgStats.mu.Unlock()

	// Copy fields without the mutex
	m.PgStats.BufferCacheHitRatio = stats.BufferCacheHitRatio
	m.PgStats.BlocksRead = stats.BlocksRead
	m.PgStats.BlocksHit = stats.BlocksHit
	m.PgStats.BlocksWritten = stats.BlocksWritten
	m.PgStats.WALRecords = stats.WALRecords
	m.PgStats.WALBytes = stats.WALBytes
	m.PgStats.CheckpointsReq = stats.CheckpointsReq
	m.PgStats.CheckpointsTimed = stats.CheckpointsTimed
	m.PgStats.TempFiles = stats.TempFiles
	m.PgStats.TempBytes = stats.TempBytes
	m.PgStats.Deadlocks = stats.Deadlocks
	m.PgStats.LockWaitCount = stats.LockWaitCount
	m.PgStats.ActiveConnections = stats.ActiveConnections
	m.PgStats.MaxConnections = stats.MaxConnections
	m.PgStats.AutovacuumCount = stats.AutovacuumCount
	m.PgStats.TopQueries = append([]QueryStats(nil), stats.TopQueries...) // Deep copy slice
	m.PgStats.LastUpdated = time.Now()
}

// GetPgStats returns a copy of PostgreSQL statistics (thread-safe)
func (m *Metrics) GetPgStats() *PostgreSQLStats {
	if m.PgStats == nil {
		return nil
	}

	m.PgStats.mu.RLock()
	defer m.PgStats.mu.RUnlock()

	// Return a copy without mutex
	return &PostgreSQLStats{
		BufferCacheHitRatio: m.PgStats.BufferCacheHitRatio,
		BlocksRead:          m.PgStats.BlocksRead,
		BlocksHit:           m.PgStats.BlocksHit,
		BlocksWritten:       m.PgStats.BlocksWritten,
		WALRecords:          m.PgStats.WALRecords,
		WALBytes:            m.PgStats.WALBytes,
		CheckpointsReq:      m.PgStats.CheckpointsReq,
		CheckpointsTimed:    m.PgStats.CheckpointsTimed,
		TempFiles:           m.PgStats.TempFiles,
		TempBytes:           m.PgStats.TempBytes,
		Deadlocks:           m.PgStats.Deadlocks,
		LockWaitCount:       m.PgStats.LockWaitCount,
		ActiveConnections:   m.PgStats.ActiveConnections,
		MaxConnections:      m.PgStats.MaxConnections,
		AutovacuumCount:     m.PgStats.AutovacuumCount,
		TopQueries:          append([]QueryStats(nil), m.PgStats.TopQueries...), // Deep copy slice
		LastUpdated:         m.PgStats.LastUpdated,
	}
}

// RecordConnectionModeTransaction records a transaction for a specific connection mode
func (m *Metrics) RecordConnectionModeTransaction(mode string, success bool, duration int64) {
	var connMetrics *ConnectionModeMetrics

	// Initialize if needed and get the appropriate metrics
	if mode == "persistent" {
		if m.PersistentConnMetrics == nil {
			m.PersistentConnMetrics = &ConnectionModeMetrics{}
		}
		connMetrics = m.PersistentConnMetrics
	} else if mode == "transient" {
		if m.TransientConnMetrics == nil {
			m.TransientConnMetrics = &ConnectionModeMetrics{}
		}
		connMetrics = m.TransientConnMetrics
	} else {
		return // Unknown mode
	}

	connMetrics.Mu.Lock()
	defer connMetrics.Mu.Unlock()

	if success {
		connMetrics.TPS++
	} else {
		connMetrics.TPSAborted++
	}

	connMetrics.TransactionDur = append(connMetrics.TransactionDur, duration)
}

// RecordConnectionModeQuery records a query for a specific connection mode
func (m *Metrics) RecordConnectionModeQuery(mode string) {
	var connMetrics *ConnectionModeMetrics

	if mode == "persistent" {
		if m.PersistentConnMetrics == nil {
			m.PersistentConnMetrics = &ConnectionModeMetrics{}
		}
		connMetrics = m.PersistentConnMetrics
	} else if mode == "transient" {
		if m.TransientConnMetrics == nil {
			m.TransientConnMetrics = &ConnectionModeMetrics{}
		}
		connMetrics = m.TransientConnMetrics
	} else {
		return // Unknown mode
	}

	connMetrics.Mu.Lock()
	defer connMetrics.Mu.Unlock()

	connMetrics.QPS++
}

// RecordConnectionModeError records an error for a specific connection mode
func (m *Metrics) RecordConnectionModeError(mode string) {
	var connMetrics *ConnectionModeMetrics

	if mode == "persistent" {
		if m.PersistentConnMetrics == nil {
			m.PersistentConnMetrics = &ConnectionModeMetrics{}
		}
		connMetrics = m.PersistentConnMetrics
	} else if mode == "transient" {
		if m.TransientConnMetrics == nil {
			m.TransientConnMetrics = &ConnectionModeMetrics{}
		}
		connMetrics = m.TransientConnMetrics
	} else {
		return // Unknown mode
	}

	connMetrics.Mu.Lock()
	defer connMetrics.Mu.Unlock()

	connMetrics.Errors++
}

// RecordConnectionSetup records connection setup time for transient connections
func (m *Metrics) RecordConnectionSetup(setupTime int64) {
	if m.TransientConnMetrics == nil {
		m.TransientConnMetrics = &ConnectionModeMetrics{}
	}

	m.TransientConnMetrics.Mu.Lock()
	defer m.TransientConnMetrics.Mu.Unlock()

	m.TransientConnMetrics.ConnectionSetup = append(m.TransientConnMetrics.ConnectionSetup, setupTime)
	m.TransientConnMetrics.ConnectionCount++
}
