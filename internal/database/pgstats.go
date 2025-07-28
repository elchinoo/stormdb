// Package database provides PostgreSQL database connectivity and statistics
// collection functionality for the StormDB benchmarking tool.
//
// This package implements version-aware PostgreSQL statistics collection,
// supporting PostgreSQL versions 15-18+ with automatic detection and
// adaptation to version-specific system views and column names.
//
// The statistics collector runs asynchronously during benchmark execution,
// gathering comprehensive performance metrics from various PostgreSQL
// system views including pg_stat_database, pg_stat_bgwriter, pg_stat_checkpointer,
// pg_stat_wal, and optionally pg_stat_statements.
package database

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/elchinoo/stormdb/pkg/types"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PgStatsCollector collects PostgreSQL statistics asynchronously during
// benchmark execution. It provides comprehensive database performance
// monitoring with version-aware compatibility for PostgreSQL 15-18+.
//
// The collector automatically detects the PostgreSQL version and adapts
// its queries to use the appropriate system views and column names for
// that version. Statistics are collected every 5 seconds by default and
// stored in thread-safe structures.
//
// Supported Statistics:
//   - Buffer cache hit ratios and I/O patterns
//   - WAL (Write-Ahead Log) activity and throughput
//   - Checkpoint frequency and performance
//   - Connection utilization and limits
//   - Lock contention and deadlock detection
//   - Query performance (with pg_stat_statements)
//   - Temporary file usage indicating memory pressure
//
// Version Compatibility:
//   - PostgreSQL 15: Full support with pg_stat_checkpointer
//   - PostgreSQL 16+: Enhanced pg_stat_io integration
//   - PostgreSQL 17+: Optimized buffer statistics collection
//   - PostgreSQL 18+: Ready for future enhancements
type PgStatsCollector struct {
	pool              *pgxpool.Pool          // Database connection pool for statistics queries
	metrics           *types.Metrics         // Target metrics structure for collected data
	collectInterval   time.Duration          // Interval between statistics collection cycles
	collectStatements bool                   // Whether to collect pg_stat_statements data
	ctx               context.Context        // Context for canceling the collection goroutine
	cancel            context.CancelFunc     // Function to cancel the collection goroutine
	pgVersion         int                    // PostgreSQL major version (15, 16, 17, 18, etc.)
	baselineStats     *types.PostgreSQLStats // Baseline statistics captured at workload start
	startTime         time.Time              // When statistics collection started
	workloadBaseline  *types.PostgreSQLStats // Precise baseline captured at workload start
}

// NewPgStatsCollector creates a new PostgreSQL statistics collector with
// automatic version detection and configuration.
//
// The collector is initialized with a database connection pool and will
// automatically detect the PostgreSQL version to ensure compatibility
// with version-specific system views and column names.
//
// Parameters:
//   - pool: Database connection pool for executing statistics queries
//   - metrics: Target metrics structure where collected data will be stored
//   - collectStatements: Enable pg_stat_statements query performance collection
//
// Returns:
//   - A configured PgStatsCollector ready to start collecting statistics
//
// The collector must be started with Start() and should be stopped with Stop()
// to properly clean up resources and stop the background goroutine.
//
// Example:
//
//	collector := NewPgStatsCollector(pool, metrics, true)
//	collector.Start()
//	defer collector.Stop()
func NewPgStatsCollector(pool *pgxpool.Pool, metrics *types.Metrics, collectStatements bool) *PgStatsCollector {
	ctx, cancel := context.WithCancel(context.Background())

	collector := &PgStatsCollector{
		pool:              pool,
		metrics:           metrics,
		collectInterval:   5 * time.Second, // Collect every 5 seconds
		collectStatements: collectStatements,
		ctx:               ctx,
		cancel:            cancel,
	}

	// Detect PostgreSQL version
	collector.detectVersion()

	return collector
}

// Start begins collecting PostgreSQL statistics in a separate goroutine
func (c *PgStatsCollector) Start() {
	go c.collectLoop()
}

// CaptureWorkloadBaseline captures baseline statistics immediately before workload execution
// This provides more accurate delta calculations by excluding setup/preparation activity
func (c *PgStatsCollector) CaptureWorkloadBaseline() {
	c.startTime = time.Now()
	baseline := &types.PostgreSQLStats{}

	// Collect baseline database statistics
	if err := c.collectDatabaseStats(baseline); err != nil {
		log.Printf("Warning: Failed to collect workload baseline database stats: %v", err)
		return
	}

	// Collect baseline buffer cache statistics
	if err := c.collectBufferStats(baseline); err != nil {
		log.Printf("Warning: Failed to collect workload baseline buffer stats: %v", err)
	}

	// Collect baseline WAL statistics
	if err := c.collectWALStats(baseline); err != nil {
		log.Printf("Warning: Failed to collect workload baseline WAL stats: %v", err)
	}

	// Collect baseline checkpoint statistics
	if err := c.collectCheckpointStats(baseline); err != nil {
		log.Printf("Warning: Failed to collect workload baseline checkpoint stats: %v", err)
	}

	c.workloadBaseline = baseline
	log.Printf("📊 Captured PostgreSQL workload baseline statistics")
}

// CalculateFinalStats captures final statistics after workload completion and calculates deltas
// This provides the most accurate representation of database activity during the workload
func (c *PgStatsCollector) CalculateFinalStats() *types.PostgreSQLStats {
	final := &types.PostgreSQLStats{}

	// Collect final database statistics
	if err := c.collectDatabaseStats(final); err != nil {
		log.Printf("Warning: Failed to collect final database stats: %v", err)
		return nil
	}

	// Collect final buffer cache statistics
	if err := c.collectBufferStats(final); err != nil {
		log.Printf("Warning: Failed to collect final buffer stats: %v", err)
	}

	// Collect final WAL statistics
	if err := c.collectWALStats(final); err != nil {
		log.Printf("Warning: Failed to collect final WAL stats: %v", err)
	}

	// Collect final checkpoint statistics
	if err := c.collectCheckpointStats(final); err != nil {
		log.Printf("Warning: Failed to collect final checkpoint stats: %v", err)
	}

	// Collect current activity stats (point-in-time)
	if err := c.collectActivityStats(final); err != nil {
		log.Printf("Warning: Failed to collect final activity stats: %v", err)
	}

	// Calculate and return final deltas using workload baseline
	deltaStats := c.calculateWorkloadDeltas(final)

	// Collect pg_stat_statements for final summary
	if c.collectStatements {
		if err := c.collectStatementStats(deltaStats); err != nil {
			log.Printf("Warning: Failed to collect final statement stats: %v", err)
		}
	}

	log.Printf("📊 Calculated final PostgreSQL workload statistics")
	return deltaStats
}

// Stop stops the statistics collection
func (c *PgStatsCollector) Stop() {
	c.cancel()
}

// detectVersion detects the PostgreSQL major version
func (c *PgStatsCollector) detectVersion() {
	var version string
	err := c.pool.QueryRow(c.ctx, "SELECT version()").Scan(&version)
	if err != nil {
		log.Printf("Failed to detect PostgreSQL version: %v, assuming version 15", err)
		c.pgVersion = 15
		return
	}

	// Parse version string like "PostgreSQL 17.5 on ..."
	parts := strings.Fields(version)
	if len(parts) >= 2 {
		versionParts := strings.Split(parts[1], ".")
		if len(versionParts) >= 1 {
			if majorVersion, err := strconv.Atoi(versionParts[0]); err == nil {
				c.pgVersion = majorVersion
				log.Printf("Detected PostgreSQL version: %d", c.pgVersion)
				return
			}
		}
	}

	log.Printf("Failed to parse PostgreSQL version from: %s, assuming version 15", version)
	c.pgVersion = 15
}

// collectLoop runs the main collection loop
func (c *PgStatsCollector) collectLoop() {
	ticker := time.NewTicker(c.collectInterval)
	defer ticker.Stop()

	// Collect initial statistics
	c.collectStats()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.collectStats()
		}
	}
}

// collectStats collects all PostgreSQL statistics and calculates deltas from baseline
func (c *PgStatsCollector) collectStats() {
	current := &types.PostgreSQLStats{}

	// Collect current database statistics
	if err := c.collectDatabaseStats(current); err != nil {
		log.Printf("Error collecting database stats: %v", err)
		return
	}

	// Collect current buffer cache statistics
	if err := c.collectBufferStats(current); err != nil {
		log.Printf("Error collecting buffer stats: %v", err)
	}

	// Collect current WAL statistics
	if err := c.collectWALStats(current); err != nil {
		log.Printf("Error collecting WAL stats: %v", err)
	}

	// Collect current checkpoint statistics
	if err := c.collectCheckpointStats(current); err != nil {
		log.Printf("Error collecting checkpoint stats: %v", err)
	}

	// Collect current lock and activity statistics (these are usually point-in-time)
	if err := c.collectActivityStats(current); err != nil {
		log.Printf("Error collecting activity stats: %v", err)
	}

	// Calculate deltas from baseline for cumulative statistics
	deltaStats := c.calculateDeltas(current)

	// Collect pg_stat_statements if enabled (these need special handling)
	if c.collectStatements {
		if err := c.collectStatementStats(deltaStats); err != nil {
			log.Printf("Error collecting statement stats: %v", err)
		}
	}

	// Update metrics
	c.metrics.UpdatePgStats(deltaStats)
}

// calculateDeltas calculates the difference between current and baseline statistics
func (c *PgStatsCollector) calculateDeltas(current *types.PostgreSQLStats) *types.PostgreSQLStats {
	if c.baselineStats == nil {
		// If no baseline, return current values (shouldn't happen in normal operation)
		return current
	}

	delta := &types.PostgreSQLStats{}

	// Calculate deltas for cumulative counters
	delta.BlocksRead = current.BlocksRead - c.baselineStats.BlocksRead
	delta.BlocksHit = current.BlocksHit - c.baselineStats.BlocksHit
	delta.BlocksWritten = current.BlocksWritten - c.baselineStats.BlocksWritten
	delta.WALRecords = current.WALRecords - c.baselineStats.WALRecords
	delta.WALBytes = current.WALBytes - c.baselineStats.WALBytes
	delta.CheckpointsReq = current.CheckpointsReq - c.baselineStats.CheckpointsReq
	delta.CheckpointsTimed = current.CheckpointsTimed - c.baselineStats.CheckpointsTimed
	delta.TempFiles = current.TempFiles - c.baselineStats.TempFiles
	delta.TempBytes = current.TempBytes - c.baselineStats.TempBytes
	delta.Deadlocks = current.Deadlocks - c.baselineStats.Deadlocks

	// Calculate buffer cache hit ratio using deltas
	totalReads := delta.BlocksRead + delta.BlocksHit
	if totalReads > 0 {
		delta.BufferCacheHitRatio = (float64(delta.BlocksHit) / float64(totalReads)) * 100
	} else {
		delta.BufferCacheHitRatio = 100.0 // No reads means perfect cache hit ratio
	}

	// Point-in-time statistics (not cumulative) - use current values
	delta.LockWaitCount = current.LockWaitCount
	delta.ActiveConnections = current.ActiveConnections
	delta.MaxConnections = current.MaxConnections
	delta.AutovacuumCount = current.AutovacuumCount
	delta.TopQueries = current.TopQueries

	delta.LastUpdated = time.Now()

	return delta
}

// calculateWorkloadDeltas calculates the difference between final and workload baseline statistics
// This provides the most accurate representation of database activity during workload execution
func (c *PgStatsCollector) calculateWorkloadDeltas(final *types.PostgreSQLStats) *types.PostgreSQLStats {
	if c.workloadBaseline == nil {
		// Fallback to regular baseline if workload baseline not available
		log.Printf("Warning: No workload baseline available, using regular baseline")
		return c.calculateDeltas(final)
	}

	delta := &types.PostgreSQLStats{}

	// Calculate deltas for cumulative counters using workload baseline
	delta.BlocksRead = final.BlocksRead - c.workloadBaseline.BlocksRead
	delta.BlocksHit = final.BlocksHit - c.workloadBaseline.BlocksHit
	delta.BlocksWritten = final.BlocksWritten - c.workloadBaseline.BlocksWritten
	delta.WALRecords = final.WALRecords - c.workloadBaseline.WALRecords
	delta.WALBytes = final.WALBytes - c.workloadBaseline.WALBytes
	delta.CheckpointsReq = final.CheckpointsReq - c.workloadBaseline.CheckpointsReq
	delta.CheckpointsTimed = final.CheckpointsTimed - c.workloadBaseline.CheckpointsTimed
	delta.TempFiles = final.TempFiles - c.workloadBaseline.TempFiles
	delta.TempBytes = final.TempBytes - c.workloadBaseline.TempBytes
	delta.Deadlocks = final.Deadlocks - c.workloadBaseline.Deadlocks

	// Calculate buffer cache hit ratio using workload deltas
	totalReads := delta.BlocksRead + delta.BlocksHit
	if totalReads > 0 {
		delta.BufferCacheHitRatio = (float64(delta.BlocksHit) / float64(totalReads)) * 100
	} else {
		delta.BufferCacheHitRatio = 100.0 // No reads means perfect cache hit ratio
	}

	// Point-in-time statistics (not cumulative) - use final values
	delta.LockWaitCount = final.LockWaitCount
	delta.ActiveConnections = final.ActiveConnections
	delta.MaxConnections = final.MaxConnections
	delta.AutovacuumCount = final.AutovacuumCount
	delta.TopQueries = final.TopQueries

	delta.LastUpdated = time.Now()

	return delta
}

// collectDatabaseStats collects basic database statistics
func (c *PgStatsCollector) collectDatabaseStats(stats *types.PostgreSQLStats) error {
	query := "SELECT blks_read, blks_hit, deadlocks, temp_files, temp_bytes FROM pg_stat_database WHERE datname = current_database()"

	var tempFiles, tempBytes *int64

	err := c.pool.QueryRow(c.ctx, query).Scan(
		&stats.BlocksRead,
		&stats.BlocksHit,
		&stats.Deadlocks,
		&tempFiles,
		&tempBytes,
	)

	if err != nil {
		return fmt.Errorf("failed to collect database stats: %w", err)
	}

	// Handle null values
	if tempFiles != nil {
		stats.TempFiles = *tempFiles
	}
	if tempBytes != nil {
		stats.TempBytes = *tempBytes
	}

	// Note: Buffer cache hit ratio is calculated in calculateDeltas() using delta values
	// to show the ratio for the test period, not the entire database lifetime

	return nil
}

// collectBufferStats collects buffer and background writer statistics based on PostgreSQL version
func (c *PgStatsCollector) collectBufferStats(stats *types.PostgreSQLStats) error {
	switch {
	case c.pgVersion >= 16:
		// PostgreSQL 16+: Use pg_stat_io for more detailed I/O statistics
		return c.collectBufferStatsV16Plus(stats)
	case c.pgVersion >= 15:
		// PostgreSQL 15: Still has buffers_backend in pg_stat_bgwriter
		return c.collectBufferStatsV15(stats)
	default:
		// PostgreSQL 14 and earlier
		return c.collectBufferStatsLegacy(stats)
	}
}

// collectWALStats collects WAL statistics (PostgreSQL 14+)
func (c *PgStatsCollector) collectWALStats(stats *types.PostgreSQLStats) error {
	// Try PostgreSQL 14+ first
	query := "SELECT wal_records, wal_bytes FROM pg_stat_wal"

	err := c.pool.QueryRow(c.ctx, query).Scan(&stats.WALRecords, &stats.WALBytes)
	if err != nil {
		// Fallback to older method or set to 0
		stats.WALRecords = 0
		stats.WALBytes = 0
		return nil // Don't treat this as an error for older PostgreSQL versions
	}

	return nil
}

// collectCheckpointStats collects checkpoint statistics based on PostgreSQL version
func (c *PgStatsCollector) collectCheckpointStats(stats *types.PostgreSQLStats) error {
	if c.pgVersion >= 15 {
		// PostgreSQL 15+: Use pg_stat_checkpointer
		return c.collectCheckpointStatsV15Plus(stats)
	}
	// PostgreSQL 14 and earlier: Use pg_stat_bgwriter
	return c.collectCheckpointStatsLegacy(stats)
} // collectActivityStats collects connection and activity statistics
func (c *PgStatsCollector) collectActivityStats(stats *types.PostgreSQLStats) error {
	// Get current connections
	var activeConnections int
	err := c.pool.QueryRow(c.ctx, "SELECT count(*) FROM pg_stat_activity WHERE state = 'active'").Scan(&activeConnections)
	if err != nil {
		return fmt.Errorf("failed to collect active connections: %w", err)
	}
	stats.ActiveConnections = activeConnections

	// Get max connections setting
	var maxConnections int
	err = c.pool.QueryRow(c.ctx, "SELECT setting::int FROM pg_settings WHERE name = 'max_connections'").Scan(&maxConnections)
	if err != nil {
		return fmt.Errorf("failed to collect max connections: %w", err)
	}
	stats.MaxConnections = maxConnections

	return nil
}

// collectStatementStats collects pg_stat_statements statistics if available
func (c *PgStatsCollector) collectStatementStats(stats *types.PostgreSQLStats) error {
	// Check if pg_stat_statements is available
	var exists bool
	err := c.pool.QueryRow(c.ctx, "SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements')").Scan(&exists)

	if err != nil || !exists {
		return nil // Extension not available, skip
	}

	// Get top 5 queries by total time
	query := `
SELECT 
query,
calls,
total_exec_time,
mean_exec_time,
rows
FROM pg_stat_statements 
WHERE query NOT LIKE '%pg_stat_statements%'
ORDER BY total_exec_time DESC 
LIMIT 5
`

	rows, err := c.pool.Query(c.ctx, query)
	if err != nil {
		return fmt.Errorf("failed to collect statement stats: %w", err)
	}
	defer rows.Close()

	var topQueries []types.QueryStats
	for rows.Next() {
		var qs types.QueryStats

		err := rows.Scan(
			&qs.Query,
			&qs.Calls,
			&qs.TotalTime,
			&qs.MeanTime,
			&qs.Rows,
		)
		if err != nil {
			continue
		}

		qs.LastUpdated = time.Now()
		topQueries = append(topQueries, qs)
	}

	stats.TopQueries = topQueries
	return nil
}

// Version-specific buffer statistics collection methods

// collectBufferStatsV16Plus collects buffer statistics for PostgreSQL 16+
func (c *PgStatsCollector) collectBufferStatsV16Plus(stats *types.PostgreSQLStats) error {
	// PostgreSQL 16+ has enhanced pg_stat_io with better backend write tracking
	// Note: pg_stat_io doesn't have op_type column, writes column directly tracks write operations
	query := `SELECT 
		COALESCE((SELECT buffers_clean FROM pg_stat_bgwriter), 0) as clean_writes,
		COALESCE(SUM(CASE WHEN context = 'normal' AND object = 'relation' THEN writes ELSE 0 END), 0) as backend_writes
		FROM pg_stat_io`

	var cleanWrites, backendWrites int64
	err := c.pool.QueryRow(c.ctx, query).Scan(&cleanWrites, &backendWrites)
	if err != nil {
		// Fallback to simpler query if pg_stat_io is not available
		fallbackQuery := "SELECT COALESCE(buffers_clean, 0) FROM pg_stat_bgwriter"
		err = c.pool.QueryRow(c.ctx, fallbackQuery).Scan(&cleanWrites)
		if err != nil {
			return fmt.Errorf("failed to collect buffer stats: %w", err)
		}
		backendWrites = 0
	}

	stats.BlocksWritten = cleanWrites + backendWrites
	return nil
}

// collectBufferStatsV15 collects buffer statistics for PostgreSQL 15
func (c *PgStatsCollector) collectBufferStatsV15(stats *types.PostgreSQLStats) error {
	// PostgreSQL 15 still has buffers_backend in pg_stat_bgwriter
	query := "SELECT COALESCE(buffers_clean, 0), COALESCE(buffers_backend, 0) FROM pg_stat_bgwriter"

	var cleanWrites, backendWrites int64
	err := c.pool.QueryRow(c.ctx, query).Scan(&cleanWrites, &backendWrites)
	if err != nil {
		return fmt.Errorf("failed to collect buffer stats: %w", err)
	}

	stats.BlocksWritten = cleanWrites + backendWrites
	return nil
}

// collectBufferStatsLegacy collects buffer statistics for PostgreSQL 14 and earlier
func (c *PgStatsCollector) collectBufferStatsLegacy(stats *types.PostgreSQLStats) error {
	// Older versions have buffers_backend in pg_stat_bgwriter
	query := "SELECT COALESCE(buffers_clean, 0), COALESCE(buffers_backend, 0) FROM pg_stat_bgwriter"

	var cleanWrites, backendWrites int64
	err := c.pool.QueryRow(c.ctx, query).Scan(&cleanWrites, &backendWrites)
	if err != nil {
		// If buffers_backend doesn't exist, just get buffers_clean
		fallbackQuery := "SELECT COALESCE(buffers_clean, 0) FROM pg_stat_bgwriter"
		err = c.pool.QueryRow(c.ctx, fallbackQuery).Scan(&cleanWrites)
		if err != nil {
			return fmt.Errorf("failed to collect buffer stats: %w", err)
		}
		backendWrites = 0
	}

	stats.BlocksWritten = cleanWrites + backendWrites
	return nil
}

// Version-specific checkpoint statistics collection methods

// collectCheckpointStatsV15Plus collects checkpoint statistics for PostgreSQL 15+
func (c *PgStatsCollector) collectCheckpointStatsV15Plus(stats *types.PostgreSQLStats) error {
	// PostgreSQL 15+ uses pg_stat_checkpointer with different column names
	query := "SELECT COALESCE(num_requested, 0), COALESCE(num_timed, 0) FROM pg_stat_checkpointer"

	err := c.pool.QueryRow(c.ctx, query).Scan(&stats.CheckpointsReq, &stats.CheckpointsTimed)
	if err != nil {
		return fmt.Errorf("failed to collect checkpoint stats: %w", err)
	}

	return nil
}

// collectCheckpointStatsLegacy collects checkpoint statistics for PostgreSQL 14 and earlier
func (c *PgStatsCollector) collectCheckpointStatsLegacy(stats *types.PostgreSQLStats) error {
	// PostgreSQL 14 and earlier use pg_stat_bgwriter
	query := "SELECT COALESCE(checkpoints_req, 0), COALESCE(checkpoints_timed, 0) FROM pg_stat_bgwriter"

	err := c.pool.QueryRow(c.ctx, query).Scan(&stats.CheckpointsReq, &stats.CheckpointsTimed)
	if err != nil {
		// Set to 0 if columns don't exist (very old versions)
		stats.CheckpointsReq = 0
		stats.CheckpointsTimed = 0
		return nil // Don't treat as fatal error
	}

	return nil
}
