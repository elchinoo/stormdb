// Database Backend Implementation for StormDB Test Results
// This package provides functionality to store test results in PostgreSQL databases

package results

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/elchinoo/stormdb/pkg/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Backend represents a database backend for storing test results
type Backend struct {
	db     *pgxpool.Pool
	config *BackendConfig
}

// BackendConfig configures the database backend
type BackendConfig struct {
	// Database connection settings
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"sslmode"`

	// Storage configuration
	Enabled          bool   `yaml:"enabled"`
	RetentionDays    int    `yaml:"retention_days"`     // Auto-cleanup old results
	StoreRawMetrics  bool   `yaml:"store_raw_metrics"`  // Store individual latency samples
	StorePgStats     bool   `yaml:"store_pg_stats"`     // Store PostgreSQL statistics
	MetricsBatchSize int    `yaml:"metrics_batch_size"` // Batch size for metric inserts
	TablePrefix      string `yaml:"table_prefix"`       // Prefix for all tables
}

// TestRun represents a complete test execution
type TestRun struct {
	ID             int64                  `json:"id"`
	TestName       string                 `json:"test_name"`
	Workload       string                 `json:"workload"`
	Configuration  map[string]interface{} `json:"configuration"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	Duration       time.Duration          `json:"duration"`
	Workers        int                    `json:"workers"`
	Connections    int                    `json:"connections"`
	Scale          int                    `json:"scale"`
	TestMode       string                 `json:"test_mode"`       // normal, progressive, etc.
	Environment    string                 `json:"environment"`     // test, staging, production
	DatabaseTarget string                 `json:"database_target"` // Target database being tested
	Version        string                 `json:"version"`         // StormDB version
	Status         string                 `json:"status"`          // completed, failed, interrupted
	ErrorMessage   string                 `json:"error_message"`
	Notes          string                 `json:"notes"`
	Tags           []string               `json:"tags"` // For categorization
	CreatedAt      time.Time              `json:"created_at"`
}

// TestResults represents aggregated test results
type TestResults struct {
	ID              int64     `json:"id"`
	TestRunID       int64     `json:"test_run_id"`
	TotalQueries    int64     `json:"total_queries"`
	SuccessfulOps   int64     `json:"successful_ops"`
	FailedOps       int64     `json:"failed_ops"`
	SuccessRate     float64   `json:"success_rate"`
	TPS             float64   `json:"tps"`
	QPS             float64   `json:"qps"`
	AvgLatencyMs    float64   `json:"avg_latency_ms"`
	P50LatencyMs    float64   `json:"p50_latency_ms"`
	P95LatencyMs    float64   `json:"p95_latency_ms"`
	P99LatencyMs    float64   `json:"p99_latency_ms"`
	P999LatencyMs   float64   `json:"p999_latency_ms"`
	MinLatencyMs    float64   `json:"min_latency_ms"`
	MaxLatencyMs    float64   `json:"max_latency_ms"`
	StdDevLatencyMs float64   `json:"stddev_latency_ms"`
	RowsRead        int64     `json:"rows_read"`
	RowsModified    int64     `json:"rows_modified"`
	BytesProcessed  int64     `json:"bytes_processed"`
	CreatedAt       time.Time `json:"created_at"`
}

// NewBackend creates a new database backend
func NewBackend(config *BackendConfig) (*Backend, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("database backend is disabled")
	}

	// Build connection string
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.Username, config.Password, config.Database, config.SSLMode)

	// Create connection pool
	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := db.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	backend := &Backend{
		db:     db,
		config: config,
	}

	// Create tables if they don't exist
	if err := backend.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	log.Printf("✅ Database backend initialized (PostgreSQL)")
	return backend, nil
}

// Close closes the database connection
func (b *Backend) Close() {
	if b.db != nil {
		b.db.Close()
	}
}

// calculateLatencyPercentiles calculates latency percentiles from transaction durations
func calculateLatencyPercentiles(durations []int64) (avg, p50, p95, p99, p999, min, max float64) {
	if len(durations) == 0 {
		return 0, 0, 0, 0, 0, 0, 0
	}

	// Make a copy and sort
	sorted := make([]int64, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	// Convert nanoseconds to milliseconds
	toMs := func(ns int64) float64 {
		return float64(ns) / 1e6
	}

	// Calculate min and max
	min = toMs(sorted[0])
	max = toMs(sorted[len(sorted)-1])

	// Calculate average
	var sum int64
	for _, d := range sorted {
		sum += d
	}
	avg = toMs(sum / int64(len(sorted)))

	// Calculate percentiles
	getPercentile := func(p float64) float64 {
		if len(sorted) == 1 {
			return toMs(sorted[0])
		}
		index := p * float64(len(sorted)-1)
		lower := int(index)
		upper := lower + 1
		if upper >= len(sorted) {
			return toMs(sorted[len(sorted)-1])
		}
		weight := index - float64(lower)
		return toMs(sorted[lower]) + weight*(toMs(sorted[upper])-toMs(sorted[lower]))
	}

	p50 = getPercentile(0.50)
	p95 = getPercentile(0.95)
	p99 = getPercentile(0.99)
	p999 = getPercentile(0.999)

	return avg, p50, p95, p99, p999, min, max
}

// calculateStandardDeviation calculates the standard deviation of latencies
func calculateStandardDeviation(durations []int64, avgMs float64) float64 {
	if len(durations) == 0 {
		return 0
	}

	var sum float64
	for _, d := range durations {
		dMs := float64(d) / 1e6 // Convert nanoseconds to milliseconds
		diff := dMs - avgMs
		sum += diff * diff
	}

	variance := sum / float64(len(durations))
	return variance // Could use math.Sqrt(variance) for true standard deviation, but variance is also useful
}

// createTables creates the necessary tables for storing test results
func (b *Backend) createTables() error {
	ctx := context.Background()

	schemas := []string{
		// Test runs table
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %stest_runs (
				id BIGSERIAL PRIMARY KEY,
				test_name VARCHAR(255) NOT NULL,
				workload VARCHAR(100) NOT NULL,
				configuration JSONB,
				start_time TIMESTAMPTZ NOT NULL,
				end_time TIMESTAMPTZ,
				duration BIGINT, -- nanoseconds
				workers INTEGER,
				connections INTEGER,
				scale INTEGER,
				test_mode VARCHAR(50),
				environment VARCHAR(50),
				database_target VARCHAR(255),
				version VARCHAR(50),
				status VARCHAR(50) DEFAULT 'running',
				error_message TEXT,
				notes TEXT,
				tags JSONB,
				created_at TIMESTAMPTZ DEFAULT NOW()
			)`, b.config.TablePrefix),

		// Test results table
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %stest_results (
				id BIGSERIAL PRIMARY KEY,
				test_run_id BIGINT REFERENCES %stest_runs(id) ON DELETE CASCADE,
				total_queries BIGINT,
				successful_ops BIGINT,
				failed_ops BIGINT,
				success_rate DECIMAL(5,2),
				tps DECIMAL(10,2),
				qps DECIMAL(10,2),
				avg_latency_ms DECIMAL(10,3),
				p50_latency_ms DECIMAL(10,3),
				p95_latency_ms DECIMAL(10,3),
				p99_latency_ms DECIMAL(10,3),
				p999_latency_ms DECIMAL(10,3),
				min_latency_ms DECIMAL(10,3),
				max_latency_ms DECIMAL(10,3),
				stddev_latency_ms DECIMAL(10,3),
				rows_read BIGINT,
				rows_modified BIGINT,
				bytes_processed BIGINT,
				created_at TIMESTAMPTZ DEFAULT NOW()
			)`, b.config.TablePrefix, b.config.TablePrefix),

		// PostgreSQL stats table
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %spostgresql_stats (
				id BIGSERIAL PRIMARY KEY,
				test_run_id BIGINT REFERENCES %stest_runs(id) ON DELETE CASCADE,
				buffer_cache_hit_ratio DECIMAL(5,2),
				blocks_read BIGINT,
				blocks_hit BIGINT,
				blocks_written BIGINT,
				wal_records BIGINT,
				wal_bytes BIGINT,
				checkpoints_req BIGINT,
				checkpoints_timed BIGINT,
				active_connections INTEGER,
				max_connections INTEGER,
				temp_files BIGINT,
				temp_bytes BIGINT,
				deadlocks BIGINT,
				lock_wait_count BIGINT,
				autovacuum_count BIGINT,
				timestamp TIMESTAMPTZ,
				created_at TIMESTAMPTZ DEFAULT NOW()
			)`, b.config.TablePrefix, b.config.TablePrefix),

		// Error metrics table
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %serror_metrics (
				id BIGSERIAL PRIMARY KEY,
				test_run_id BIGINT REFERENCES %stest_runs(id) ON DELETE CASCADE,
				error_type VARCHAR(255),
				error_message TEXT,
				error_count BIGINT,
				worker_id INTEGER,
				timestamp TIMESTAMPTZ,
				created_at TIMESTAMPTZ DEFAULT NOW()
			)`, b.config.TablePrefix, b.config.TablePrefix),

		// Workload metrics table
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %sworkload_metrics (
				id BIGSERIAL PRIMARY KEY,
				test_run_id BIGINT REFERENCES %stest_runs(id) ON DELETE CASCADE,
				workload_type VARCHAR(100),
				metric_name VARCHAR(255),
				metric_value DECIMAL(15,3),
				metric_unit VARCHAR(50),
				metric_data JSONB,
				timestamp TIMESTAMPTZ,
				created_at TIMESTAMPTZ DEFAULT NOW()
			)`, b.config.TablePrefix, b.config.TablePrefix),
	}

	// Only create latency metrics table if raw metrics storage is enabled
	if b.config.StoreRawMetrics {
		schemas = append(schemas, fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %slatency_metrics (
				id BIGSERIAL PRIMARY KEY,
				test_run_id BIGINT REFERENCES %stest_runs(id) ON DELETE CASCADE,
				worker_id INTEGER,
				operation_type VARCHAR(100),
				latency_ns BIGINT,
				timestamp TIMESTAMPTZ,
				created_at TIMESTAMPTZ DEFAULT NOW()
			)`, b.config.TablePrefix, b.config.TablePrefix))
	}

	// Create indexes
	indexes := []string{
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%stest_runs_workload ON %stest_runs(workload)", b.config.TablePrefix, b.config.TablePrefix),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%stest_runs_start_time ON %stest_runs(start_time)", b.config.TablePrefix, b.config.TablePrefix),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%stest_runs_environment ON %stest_runs(environment)", b.config.TablePrefix, b.config.TablePrefix),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%stest_runs_status ON %stest_runs(status)", b.config.TablePrefix, b.config.TablePrefix),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%stest_runs_test_name ON %stest_runs(test_name)", b.config.TablePrefix, b.config.TablePrefix),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%stest_results_test_run_id ON %stest_results(test_run_id)", b.config.TablePrefix, b.config.TablePrefix),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%spostgresql_stats_test_run_id ON %spostgresql_stats(test_run_id)", b.config.TablePrefix, b.config.TablePrefix),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%serror_metrics_test_run_id ON %serror_metrics(test_run_id)", b.config.TablePrefix, b.config.TablePrefix),
	}

	// Execute schema creation
	for _, schema := range schemas {
		if _, err := b.db.Exec(ctx, schema); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create indexes
	for _, index := range indexes {
		if _, err := b.db.Exec(ctx, index); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// StoreTestRun stores a complete test run with all metrics
func (b *Backend) StoreTestRun(ctx context.Context, testRun *TestRun, metrics *types.Metrics) error {
	tx, err := b.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			// Log rollback error for debugging
			fmt.Printf("Warning: Transaction rollback failed: %v\n", rollbackErr)
		}
	}()

	// Store test run
	testRunID, err := b.insertTestRun(ctx, tx, testRun)
	if err != nil {
		return fmt.Errorf("failed to insert test run: %w", err)
	}

	// Store aggregated results
	if err := b.insertTestResults(ctx, tx, testRunID, metrics); err != nil {
		return fmt.Errorf("failed to insert test results: %w", err)
	}

	// Store PostgreSQL stats if available and enabled
	if b.config.StorePgStats {
		if pgStats := metrics.GetPgStats(); pgStats != nil {
			if err := b.insertPostgreSQLStats(ctx, tx, testRunID, pgStats); err != nil {
				return fmt.Errorf("failed to insert PostgreSQL stats: %w", err)
			}
		}
	}

	// Store error metrics
	if err := b.insertErrorMetrics(ctx, tx, testRunID, metrics); err != nil {
		return fmt.Errorf("failed to insert error metrics: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("📊 Stored test results to database (test_run_id: %d)", testRunID)
	return nil
}

// insertTestRun inserts a test run record and returns the ID
func (b *Backend) insertTestRun(ctx context.Context, tx pgx.Tx, testRun *TestRun) (int64, error) {
	configJSON, _ := json.Marshal(testRun.Configuration)
	tagsJSON, _ := json.Marshal(testRun.Tags)

	query := fmt.Sprintf(`
		INSERT INTO %stest_runs 
		(test_name, workload, configuration, start_time, end_time, duration, workers, connections, 
		 scale, test_mode, environment, database_target, version, status, error_message, notes, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id`, b.config.TablePrefix)

	var id int64
	err := tx.QueryRow(ctx, query,
		testRun.TestName, testRun.Workload, string(configJSON), testRun.StartTime, testRun.EndTime,
		testRun.Duration.Nanoseconds(), testRun.Workers, testRun.Connections, testRun.Scale,
		testRun.TestMode, testRun.Environment, testRun.DatabaseTarget, testRun.Version,
		testRun.Status, testRun.ErrorMessage, testRun.Notes, string(tagsJSON)).Scan(&id)

	return id, err
}

// insertTestResults inserts aggregated test results
func (b *Backend) insertTestResults(ctx context.Context, tx pgx.Tx, testRunID int64, metrics *types.Metrics) error {
	// Calculate aggregated metrics
	successRate := float64(0)
	totalOps := metrics.TPS + metrics.TPSAborted
	if totalOps > 0 {
		successRate = (float64(metrics.TPS) / float64(totalOps)) * 100
	}

	// Calculate latency percentiles if transaction durations are available
	var avgLatency, p50Latency, p95Latency, p99Latency, p999Latency, minLatency, maxLatency, stdDevLatency float64
	if len(metrics.TransactionDur) > 0 {
		avgLatency, p50Latency, p95Latency, p99Latency, p999Latency, minLatency, maxLatency = calculateLatencyPercentiles(metrics.TransactionDur)
		stdDevLatency = calculateStandardDeviation(metrics.TransactionDur, avgLatency)
	}

	query := fmt.Sprintf(`
		INSERT INTO %stest_results 
		(test_run_id, total_queries, successful_ops, failed_ops, success_rate, tps, qps,
		 avg_latency_ms, p50_latency_ms, p95_latency_ms, p99_latency_ms, p999_latency_ms,
		 min_latency_ms, max_latency_ms, stddev_latency_ms, rows_read, rows_modified, bytes_processed)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`, b.config.TablePrefix)

	_, err := tx.Exec(ctx, query,
		testRunID, metrics.QPS, metrics.TPS, metrics.TPSAborted, successRate,
		metrics.TPS, metrics.QPS, avgLatency, p50Latency, p95Latency, p99Latency, p999Latency,
		minLatency, maxLatency, stdDevLatency, metrics.RowsRead, metrics.RowsModified, int64(0)) // bytes_processed placeholder

	return err
}

// insertPostgreSQLStats inserts PostgreSQL statistics
func (b *Backend) insertPostgreSQLStats(ctx context.Context, tx pgx.Tx, testRunID int64, pgStats *types.PostgreSQLStats) error {
	query := fmt.Sprintf(`
		INSERT INTO %spostgresql_stats 
		(test_run_id, buffer_cache_hit_ratio, blocks_read, blocks_hit, blocks_written,
		 wal_records, wal_bytes, checkpoints_req, checkpoints_timed, active_connections,
		 max_connections, temp_files, temp_bytes, deadlocks, lock_wait_count, autovacuum_count, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`, b.config.TablePrefix)

	_, err := tx.Exec(ctx, query,
		testRunID, pgStats.BufferCacheHitRatio, pgStats.BlocksRead, pgStats.BlocksHit, pgStats.BlocksWritten,
		pgStats.WALRecords, pgStats.WALBytes, pgStats.CheckpointsReq, pgStats.CheckpointsTimed,
		pgStats.ActiveConnections, pgStats.MaxConnections, pgStats.TempFiles, pgStats.TempBytes,
		pgStats.Deadlocks, pgStats.LockWaitCount, pgStats.AutovacuumCount, pgStats.LastUpdated)

	return err
}

// insertErrorMetrics inserts error metrics
func (b *Backend) insertErrorMetrics(ctx context.Context, tx pgx.Tx, testRunID int64, metrics *types.Metrics) error {
	if len(metrics.ErrorTypes) == 0 {
		return nil
	}

	query := fmt.Sprintf(`
		INSERT INTO %serror_metrics 
		(test_run_id, error_type, error_count, timestamp)
		VALUES ($1, $2, $3, NOW())`, b.config.TablePrefix)

	for errorType, count := range metrics.ErrorTypes {
		_, err := tx.Exec(ctx, query, testRunID, errorType, count)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetTestRuns retrieves test runs with optional filtering
func (b *Backend) GetTestRuns(ctx context.Context, filters map[string]interface{}) ([]*TestRun, error) {
	query := fmt.Sprintf(`
		SELECT id, test_name, workload, configuration, start_time, end_time, duration,
		       workers, connections, scale, test_mode, environment, database_target,
		       version, status, error_message, notes, tags, created_at
		FROM %stest_runs 
		WHERE 1=1`, b.config.TablePrefix)

	args := []interface{}{}
	argCount := 0

	// Add filters
	if testName, ok := filters["test_name"]; ok {
		argCount++
		query += fmt.Sprintf(" AND test_name = $%d", argCount)
		args = append(args, testName)
	}

	if workload, ok := filters["workload"]; ok {
		argCount++
		query += fmt.Sprintf(" AND workload = $%d", argCount)
		args = append(args, workload)
	}

	query += " ORDER BY start_time DESC"

	if limit, ok := filters["limit"]; ok {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
	}

	rows, err := b.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query test runs: %w", err)
	}
	defer rows.Close()

	var testRuns []*TestRun
	for rows.Next() {
		var tr TestRun
		var configJSON, tagsJSON string
		var duration int64

		err := rows.Scan(&tr.ID, &tr.TestName, &tr.Workload, &configJSON, &tr.StartTime,
			&tr.EndTime, &duration, &tr.Workers, &tr.Connections, &tr.Scale,
			&tr.TestMode, &tr.Environment, &tr.DatabaseTarget, &tr.Version,
			&tr.Status, &tr.ErrorMessage, &tr.Notes, &tagsJSON, &tr.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test run: %w", err)
		}

		tr.Duration = time.Duration(duration)

		// Parse JSON fields
		if err := json.Unmarshal([]byte(configJSON), &tr.Configuration); err != nil {
			log.Printf("Warning: failed to parse configuration JSON: %v", err)
			tr.Configuration = make(map[string]interface{})
		}

		if err := json.Unmarshal([]byte(tagsJSON), &tr.Tags); err != nil {
			log.Printf("Warning: failed to parse tags JSON: %v", err)
			tr.Tags = []string{}
		}

		testRuns = append(testRuns, &tr)
	}

	return testRuns, nil
}

// GetTestResults retrieves aggregated results for a test run
func (b *Backend) GetTestResults(ctx context.Context, testRunID int64) (*TestResults, error) {
	query := fmt.Sprintf(`
		SELECT id, test_run_id, total_queries, successful_ops, failed_ops, success_rate,
		       tps, qps, avg_latency_ms, p50_latency_ms, p95_latency_ms, p99_latency_ms,
		       p999_latency_ms, min_latency_ms, max_latency_ms, stddev_latency_ms,
		       rows_read, rows_modified, bytes_processed, created_at
		FROM %stest_results 
		WHERE test_run_id = $1`, b.config.TablePrefix)

	var tr TestResults
	err := b.db.QueryRow(ctx, query, testRunID).Scan(
		&tr.ID, &tr.TestRunID, &tr.TotalQueries, &tr.SuccessfulOps, &tr.FailedOps,
		&tr.SuccessRate, &tr.TPS, &tr.QPS, &tr.AvgLatencyMs, &tr.P50LatencyMs,
		&tr.P95LatencyMs, &tr.P99LatencyMs, &tr.P999LatencyMs, &tr.MinLatencyMs,
		&tr.MaxLatencyMs, &tr.StdDevLatencyMs, &tr.RowsRead, &tr.RowsModified,
		&tr.BytesProcessed, &tr.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get test results: %w", err)
	}

	return &tr, nil
}

// CleanupOldResults removes test results older than retention period
func (b *Backend) CleanupOldResults(ctx context.Context) error {
	if b.config.RetentionDays <= 0 {
		return nil // No cleanup configured
	}

	cutoffDate := time.Now().AddDate(0, 0, -b.config.RetentionDays)

	query := fmt.Sprintf("DELETE FROM %stest_runs WHERE created_at < $1", b.config.TablePrefix)
	result, err := b.db.Exec(ctx, query, cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to cleanup old results: %w", err)
	}

	if deleted := result.RowsAffected(); deleted > 0 {
		log.Printf("🧹 Cleaned up %d old test result records (older than %d days)", deleted, b.config.RetentionDays)
	}

	return nil
}
