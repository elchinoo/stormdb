// Package bulkload implements a bulk load performance test plugin for StormDB v0.4-alpha
// This plugin tests different batch sizes with a fixed number of connections
package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elchinoo/stormdb/core"
	_ "github.com/lib/pq"
)

// BulkLoadPlugin implements the bulk load performance test
type BulkLoadPlugin struct {
	core        *core.CoreServices
	logger      core.Logger
	db          *sql.DB
	config      *BulkLoadConfig
	isRunning   int64
	stopChan    chan struct{}
	wg          sync.WaitGroup
	metrics     *BulkLoadMetrics
	testStarted time.Time
}

// BulkLoadConfig defines the configuration for bulk load tests
type BulkLoadConfig struct {
	// Database connection
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Database string `json:"database" yaml:"database"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	SSLMode  string `json:"ssl_mode" yaml:"ssl_mode"`

	// Test configuration
	Rebuild      bool          `json:"rebuild" yaml:"rebuild"`             // Force drop/recreate of database
	BatchSizes   []int         `json:"batch_sizes" yaml:"batch_sizes"`     // Batch sizes to test [1, 1000, 10000, 50000]
	Connections  int           `json:"connections" yaml:"connections"`     // Fixed number of connections (default: 20)
	Duration     time.Duration `json:"duration" yaml:"duration"`           // Duration per batch size
	WarmupTime   time.Duration `json:"warmup_time" yaml:"warmup_time"`     // Warmup before measurements
	ThinkTime    time.Duration `json:"think_time" yaml:"think_time"`       // Delay between batches
	TableName    string        `json:"table_name" yaml:"table_name"`       // Table name for bulk inserts
	DropTable    bool          `json:"drop_table" yaml:"drop_table"`       // Whether to drop/recreate table between tests
	GenerateData bool          `json:"generate_data" yaml:"generate_data"` // Whether to generate random data
	DataColumns  int           `json:"data_columns" yaml:"data_columns"`   // Number of data columns to create
	IndexColumns []string      `json:"index_columns" yaml:"index_columns"` // Columns to create indexes on
	Verbose      bool          `json:"verbose" yaml:"verbose"`             // Enable verbose logging
}

// BulkLoadMetrics tracks performance metrics for bulk load tests
type BulkLoadMetrics struct {
	mu                sync.RWMutex
	BatchResults      []BatchResult `json:"batch_results"`
	TotalTransactions int64         `json:"total_transactions"`
	TotalRowsInserted int64         `json:"total_rows_inserted"`
	TotalErrors       int64         `json:"total_errors"`
	StartTime         time.Time     `json:"start_time"`
	EndTime           time.Time     `json:"end_time"`
}

// BatchResult contains metrics for a specific batch size test
type BatchResult struct {
	BatchSize          int     `json:"batch_size"`
	Connections        int     `json:"connections"`
	TotalTransactions  int64   `json:"total_transactions"`
	TotalRowsInserted  int64   `json:"total_rows_inserted"`
	TotalErrors        int64   `json:"total_errors"`
	DurationSeconds    float64 `json:"duration_seconds"`
	TransactionsPerSec float64 `json:"transactions_per_sec"`
	RowsPerSec         float64 `json:"rows_per_sec"`
	AvgLatencyMs       float64 `json:"avg_latency_ms"`
	MinLatencyMs       float64 `json:"min_latency_ms"`
	MaxLatencyMs       float64 `json:"max_latency_ms"`
	ErrorRate          float64 `json:"error_rate"`
}

// WorkerStats tracks per-worker metrics
type WorkerStats struct {
	WorkerID     int           `json:"worker_id"`
	Transactions int64         `json:"transactions"`
	RowsInserted int64         `json:"rows_inserted"`
	Errors       int64         `json:"errors"`
	TotalLatency time.Duration `json:"total_latency"`
	MinLatency   time.Duration `json:"min_latency"`
	MaxLatency   time.Duration `json:"max_latency"`
}

// Plugin interface implementation

// Metadata returns plugin information
func (p *BulkLoadPlugin) Metadata() core.PluginMetadata {
	return core.PluginMetadata{
		Name:        "bulk-load",
		Version:     "1.0.0",
		Description: "Bulk load performance testing with different batch sizes",
		Author:      "StormDB Team",
		License:     "Apache-2.0",
		TestTypes:   []string{"bulk_insert", "batch_performance", "load_testing"},
		ConfigSchema: `{
			"type": "object",
			"required": ["host", "port", "database", "username", "password"],
			"properties": {
				"host": {"type": "string", "description": "Database host"},
				"port": {"type": "integer", "minimum": 1, "maximum": 65535},
				"database": {"type": "string", "description": "Database name"},
				"username": {"type": "string", "description": "Database username"},
				"password": {"type": "string", "description": "Database password"},
				"ssl_mode": {"type": "string", "enum": ["disable", "require", "verify-ca", "verify-full"], "default": "disable"},
				"rebuild": {"type": "boolean", "default": false, "description": "Force drop and recreate of the test database"},
				"batch_sizes": {"type": "array", "items": {"type": "integer", "minimum": 1}, "default": [1, 1000, 10000, 50000]},
				"connections": {"type": "integer", "minimum": 1, "maximum": 1000, "default": 20},
				"duration": {"type": "string", "pattern": "^[0-9]+[smh]$", "default": "5m"},
				"warmup_time": {"type": "string", "pattern": "^[0-9]+[smh]$", "default": "30s"},
				"think_time": {"type": "string", "pattern": "^[0-9]+[smh]$", "default": "10ms"},
				"table_name": {"type": "string", "default": "bulk_test_data"},
				"drop_table": {"type": "boolean", "default": true},
				"generate_data": {"type": "boolean", "default": true},
				"data_columns": {"type": "integer", "minimum": 1, "maximum": 50, "default": 10},
				"index_columns": {"type": "array", "items": {"type": "string"}, "default": []},
				"verbose": {"type": "boolean", "default": false}
			}
		}`,
		Dependencies: map[string]string{
			"go":     "1.21+",
			"driver": "github.com/lib/pq",
		},
	}
}

// Initialize sets up the plugin with core services
func (p *BulkLoadPlugin) Initialize(ctx context.Context, coreServices *core.CoreServices) error {
	p.core = coreServices
	p.logger = coreServices.Logger.WithPlugin("bulk-load")
	p.stopChan = make(chan struct{})
	p.metrics = &BulkLoadMetrics{
		BatchResults: make([]BatchResult, 0),
	}

	p.logger.Info("Bulk load plugin initialized successfully")
	return nil
}

// Validate checks the configuration
func (p *BulkLoadPlugin) Validate(config map[string]interface{}) error {
	var bulkConfig BulkLoadConfig

	// Manual parsing to handle duration strings properly
	if host, ok := config["host"]; ok {
		if hostStr, ok := host.(string); ok {
			bulkConfig.Host = hostStr
		}
	}
	if port, ok := config["port"]; ok {
		if portFloat, ok := port.(float64); ok {
			bulkConfig.Port = int(portFloat)
		} else if portInt, ok := port.(int); ok {
			bulkConfig.Port = portInt
		}
	}
	if database, ok := config["database"]; ok {
		if dbStr, ok := database.(string); ok {
			bulkConfig.Database = dbStr
		}
	}
	if username, ok := config["username"]; ok {
		if userStr, ok := username.(string); ok {
			bulkConfig.Username = userStr
		}
	}
	if password, ok := config["password"]; ok {
		if passStr, ok := password.(string); ok {
			bulkConfig.Password = passStr
		}
	}
	if sslMode, ok := config["ssl_mode"]; ok {
		if sslStr, ok := sslMode.(string); ok {
			bulkConfig.SSLMode = sslStr
		}
	}
	if connections, ok := config["connections"]; ok {
		if connFloat, ok := connections.(float64); ok {
			bulkConfig.Connections = int(connFloat)
		} else if connInt, ok := connections.(int); ok {
			bulkConfig.Connections = connInt
		}
	}
	if dataColumns, ok := config["data_columns"]; ok {
		if dcFloat, ok := dataColumns.(float64); ok {
			bulkConfig.DataColumns = int(dcFloat)
		} else if dcInt, ok := dataColumns.(int); ok {
			bulkConfig.DataColumns = dcInt
		}
	}
	if tableName, ok := config["table_name"]; ok {
		if tableStr, ok := tableName.(string); ok {
			bulkConfig.TableName = tableStr
		}
	}
	if dropTable, ok := config["drop_table"]; ok {
		if dropBool, ok := dropTable.(bool); ok {
			bulkConfig.DropTable = dropBool
		}
	}
	if generateData, ok := config["generate_data"]; ok {
		if genBool, ok := generateData.(bool); ok {
			bulkConfig.GenerateData = genBool
		}
	}
	if verbose, ok := config["verbose"]; ok {
		if verboseBool, ok := verbose.(bool); ok {
			bulkConfig.Verbose = verboseBool
		}
	}
	if rebuild, ok := config["rebuild"]; ok {
		if rebuildBool, ok := rebuild.(bool); ok {
			bulkConfig.Rebuild = rebuildBool
		}
	}

	// Parse batch sizes
	if batchSizes, ok := config["batch_sizes"]; ok {
		if batchArray, ok := batchSizes.([]interface{}); ok {
			for _, batch := range batchArray {
				if batchFloat, ok := batch.(float64); ok {
					bulkConfig.BatchSizes = append(bulkConfig.BatchSizes, int(batchFloat))
				} else if batchInt, ok := batch.(int); ok {
					bulkConfig.BatchSizes = append(bulkConfig.BatchSizes, batchInt)
				}
			}
		}
	}

	// Parse index columns
	if indexColumns, ok := config["index_columns"]; ok {
		if indexArray, ok := indexColumns.([]interface{}); ok {
			for _, idx := range indexArray {
				if idxStr, ok := idx.(string); ok {
					bulkConfig.IndexColumns = append(bulkConfig.IndexColumns, idxStr)
				}
			}
		}
	}

	// Parse duration strings
	if duration, ok := config["duration"]; ok {
		if durStr, ok := duration.(string); ok {
			if dur, err := time.ParseDuration(durStr); err == nil {
				bulkConfig.Duration = dur
			}
		}
	}
	if warmupTime, ok := config["warmup_time"]; ok {
		if warmupStr, ok := warmupTime.(string); ok {
			if warmup, err := time.ParseDuration(warmupStr); err == nil {
				bulkConfig.WarmupTime = warmup
			}
		}
	}
	if thinkTime, ok := config["think_time"]; ok {
		if thinkStr, ok := thinkTime.(string); ok {
			if think, err := time.ParseDuration(thinkStr); err == nil {
				bulkConfig.ThinkTime = think
			}
		}
	}

	// Set defaults FIRST, then validate
	if len(bulkConfig.BatchSizes) == 0 {
		bulkConfig.BatchSizes = []int{1, 1000, 10000, 50000}
	}
	if bulkConfig.Connections == 0 {
		bulkConfig.Connections = 20
	}
	if bulkConfig.Duration == 0 {
		bulkConfig.Duration = 5 * time.Minute
	}
	if bulkConfig.WarmupTime == 0 {
		bulkConfig.WarmupTime = 30 * time.Second
	}
	if bulkConfig.ThinkTime == 0 {
		bulkConfig.ThinkTime = 10 * time.Millisecond
	}
	if bulkConfig.TableName == "" {
		bulkConfig.TableName = "bulk_test_data"
	}
	if bulkConfig.DataColumns == 0 {
		bulkConfig.DataColumns = 10
	}
	if bulkConfig.SSLMode == "" {
		bulkConfig.SSLMode = "disable"
	}

	// Validate configuration AFTER setting defaults
	if bulkConfig.Host == "" {
		return fmt.Errorf("host is required")
	}
	if bulkConfig.Port <= 0 || bulkConfig.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if bulkConfig.Database == "" {
		return fmt.Errorf("database is required")
	}
	if bulkConfig.Username == "" {
		return fmt.Errorf("username is required")
	}
	if bulkConfig.Password == "" {
		return fmt.Errorf("password is required")
	}

	// Check for explicitly set invalid values (not defaults)
	if connections, ok := config["connections"]; ok {
		if connFloat, ok := connections.(float64); ok && (connFloat < 1 || connFloat > 1000) {
			return fmt.Errorf("connections must be between 1 and 1000")
		} else if connInt, ok := connections.(int); ok && (connInt < 1 || connInt > 1000) {
			return fmt.Errorf("connections must be between 1 and 1000")
		}
	}
	if dataColumns, ok := config["data_columns"]; ok {
		if dcFloat, ok := dataColumns.(float64); ok && (dcFloat < 1 || dcFloat > 50) {
			return fmt.Errorf("data_columns must be between 1 and 50")
		} else if dcInt, ok := dataColumns.(int); ok && (dcInt < 1 || dcInt > 50) {
			return fmt.Errorf("data_columns must be between 1 and 50")
		}
	}

	// Validate batch sizes
	for _, size := range bulkConfig.BatchSizes {
		if size < 1 {
			return fmt.Errorf("batch sizes must be positive integers")
		}
	}

	p.config = &bulkConfig
	p.logger.Info("Configuration validated successfully",
		core.Field{Key: "batch_sizes", Value: bulkConfig.BatchSizes},
		core.Field{Key: "connections", Value: bulkConfig.Connections},
		core.Field{Key: "duration", Value: bulkConfig.Duration},
	)

	return nil
}

// Execute runs the bulk load performance test
func (p *BulkLoadPlugin) Execute(ctx context.Context, config map[string]interface{}) error {
	if err := p.Validate(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Set running state
	if !atomic.CompareAndSwapInt64(&p.isRunning, 0, 1) {
		return fmt.Errorf("test is already running")
	}
	defer atomic.StoreInt64(&p.isRunning, 0)

	p.testStarted = time.Now()
	p.metrics.StartTime = p.testStarted

	p.logger.Info("Starting bulk load performance test",
		core.Field{Key: "batch_sizes", Value: p.config.BatchSizes},
		core.Field{Key: "connections", Value: p.config.Connections},
		core.Field{Key: "table_name", Value: p.config.TableName},
	)

	// Connect to database
	if err := p.connectDatabase(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer p.db.Close()

	// Rebuild database if requested
	if p.config.Rebuild {
		if err := p.rebuildDatabase(); err != nil {
			return fmt.Errorf("failed to rebuild database: %w", err)
		}
		// Reconnect after rebuild
		if err := p.connectDatabase(); err != nil {
			return fmt.Errorf("failed to reconnect to database after rebuild: %w", err)
		}
		defer p.db.Close()
	}

	// Create/prepare test table
	if err := p.setupTestTable(); err != nil {
		return fmt.Errorf("failed to setup test table: %w", err)
	}

	// Run tests for each batch size
	for i, batchSize := range p.config.BatchSizes {
		p.logger.Info("Starting test for batch size",
			core.Field{Key: "batch_size", Value: batchSize},
			core.Field{Key: "test", Value: fmt.Sprintf("%d/%d", i+1, len(p.config.BatchSizes))},
		)

		result, err := p.runBatchTest(ctx, batchSize)
		if err != nil {
			p.logger.Error("Batch test failed",
				core.Field{Key: "batch_size", Value: batchSize},
				core.Field{Key: "error", Value: err.Error()},
			)
			continue
		}

		p.metrics.mu.Lock()
		p.metrics.BatchResults = append(p.metrics.BatchResults, *result)
		p.metrics.TotalTransactions += result.TotalTransactions
		p.metrics.TotalRowsInserted += result.TotalRowsInserted
		p.metrics.TotalErrors += result.TotalErrors
		p.metrics.mu.Unlock()

		p.logger.Info("Batch test completed",
			core.Field{Key: "batch_size", Value: batchSize},
			core.Field{Key: "transactions", Value: result.TotalTransactions},
			core.Field{Key: "rows_inserted", Value: result.TotalRowsInserted},
			core.Field{Key: "tps", Value: result.TransactionsPerSec},
			core.Field{Key: "rows_per_sec", Value: result.RowsPerSec},
		)

		// Clear table data between tests if configured
		if p.config.DropTable && i < len(p.config.BatchSizes)-1 {
			if err := p.clearTableData(); err != nil {
				p.logger.Warn("Failed to clear table data", core.Field{Key: "error", Value: err.Error()})
			}
		}

		// Brief pause between tests
		time.Sleep(2 * time.Second)
	}

	p.metrics.EndTime = time.Now()
	p.logger.Info("Bulk load performance test completed",
		core.Field{Key: "total_duration", Value: p.metrics.EndTime.Sub(p.metrics.StartTime)},
		core.Field{Key: "total_transactions", Value: p.metrics.TotalTransactions},
		core.Field{Key: "total_rows", Value: p.metrics.TotalRowsInserted},
	)

	return nil
}

// Cleanup performs any necessary cleanup
func (p *BulkLoadPlugin) Cleanup(ctx context.Context) error {
	// Stop any running operations
	close(p.stopChan)
	p.wg.Wait()

	// Close database connection
	if p.db != nil {
		p.db.Close()
	}

	p.logger.Info("Bulk load plugin cleanup completed")
	return nil
}

// Helper methods

func (p *BulkLoadPlugin) rebuildDatabase() error {
	p.logger.Info("Rebuilding database", core.Field{Key: "database", Value: p.config.Database})

	// Connect to the default 'postgres' database to drop the target database
	defaultConnStr := fmt.Sprintf("host=%s port=%d dbname=postgres user=%s password=%s sslmode=%s",
		p.config.Host, p.config.Port, p.config.Username, p.config.Password, p.config.SSLMode)

	db, err := sql.Open("postgres", defaultConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to 'postgres' db for rebuild: %w", err)
	}
	defer db.Close()

	// Terminate existing connections
	terminateSQL := `
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = $1 AND pid <> pg_backend_pid()`
	if _, err := db.Exec(terminateSQL, p.config.Database); err != nil {
		p.logger.Warn("Could not terminate existing connections, proceeding anyway", core.Field{Key: "error", Value: err.Error()})
	}

	// Drop database
	dropSQL := fmt.Sprintf("DROP DATABASE IF EXISTS %s", p.config.Database)
	if _, err := db.Exec(dropSQL); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}
	p.logger.Info("Dropped database", core.Field{Key: "database", Value: p.config.Database})

	// Create database
	createSQL := fmt.Sprintf("CREATE DATABASE %s", p.config.Database)
	if _, err := db.Exec(createSQL); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	p.logger.Info("Created database", core.Field{Key: "database", Value: p.config.Database})

	return nil
}

// connectDatabase establishes database connection
func (p *BulkLoadPlugin) connectDatabase() error {
	connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		p.config.Host, p.config.Port, p.config.Database,
		p.config.Username, p.config.Password, p.config.SSLMode)

	var err error
	p.db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	p.db.SetMaxOpenConns(p.config.Connections + 5) // A few extra for management
	p.db.SetMaxIdleConns(p.config.Connections)
	p.db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := p.db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	p.logger.Info("Database connection established",
		core.Field{Key: "host", Value: p.config.Host},
		core.Field{Key: "database", Value: p.config.Database},
	)

	return nil
}

// setupTestTable creates or prepares the test table
func (p *BulkLoadPlugin) setupTestTable() error {
	tableName := p.config.TableName

	// Drop table if configured
	if p.config.DropTable {
		dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName)
		if _, err := p.db.Exec(dropSQL); err != nil {
			return fmt.Errorf("failed to drop table: %w", err)
		}
		p.logger.Info("Dropped existing table", core.Field{Key: "table", Value: tableName})
	}

	// Create table schema
	var columns []string
	columns = append(columns, "id SERIAL PRIMARY KEY")
	columns = append(columns, "created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP")

	// Add data columns
	for i := 1; i <= p.config.DataColumns; i++ {
		switch i % 4 {
		case 0:
			columns = append(columns, fmt.Sprintf("data_int_%d INTEGER", i))
		case 1:
			columns = append(columns, fmt.Sprintf("data_text_%d TEXT", i))
		case 2:
			columns = append(columns, fmt.Sprintf("data_float_%d FLOAT", i))
		case 3:
			columns = append(columns, fmt.Sprintf("data_bool_%d BOOLEAN", i))
		}
	}

	createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(columns, ", "))
	if _, err := p.db.Exec(createSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	p.logger.Info("Test table created",
		core.Field{Key: "table", Value: tableName},
		core.Field{Key: "columns", Value: len(columns)},
	)

	// Create indexes if configured
	for _, indexCol := range p.config.IndexColumns {
		indexName := fmt.Sprintf("idx_%s_%s", tableName, indexCol)
		indexSQL := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)", indexName, tableName, indexCol)
		if _, err := p.db.Exec(indexSQL); err != nil {
			p.logger.Warn("Failed to create index",
				core.Field{Key: "index", Value: indexName},
				core.Field{Key: "error", Value: err.Error()},
			)
		} else {
			p.logger.Info("Created index", core.Field{Key: "index", Value: indexName})
		}
	}

	return nil
}

// clearTableData truncates the test table
func (p *BulkLoadPlugin) clearTableData() error {
	truncateSQL := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY", p.config.TableName)
	if _, err := p.db.Exec(truncateSQL); err != nil {
		return fmt.Errorf("failed to truncate table: %w", err)
	}
	p.logger.Info("Table data cleared", core.Field{Key: "table", Value: p.config.TableName})
	return nil
}

// runBatchTest executes the test for a specific batch size
func (p *BulkLoadPlugin) runBatchTest(ctx context.Context, batchSize int) (*BatchResult, error) {
	result := &BatchResult{
		BatchSize:    batchSize,
		Connections:  p.config.Connections,
		MinLatencyMs: float64(time.Hour.Milliseconds()), // Initialize with high value
	}

	// Warmup phase
	p.logger.Info("Starting warmup phase", core.Field{Key: "duration", Value: p.config.WarmupTime})
	warmupCtx, warmupCancel := context.WithTimeout(ctx, p.config.WarmupTime)
	p.runWorkers(warmupCtx, batchSize, nil) // No stats collection during warmup
	warmupCancel()

	// Measurement phase
	p.logger.Info("Starting measurement phase", core.Field{Key: "duration", Value: p.config.Duration})

	workerStats := make([]*WorkerStats, p.config.Connections)
	for i := range workerStats {
		workerStats[i] = &WorkerStats{
			WorkerID:   i,
			MinLatency: time.Hour, // Initialize with high value
		}
	}

	measureCtx, measureCancel := context.WithTimeout(ctx, p.config.Duration)
	testStart := time.Now()

	p.runWorkers(measureCtx, batchSize, workerStats)
	measureCancel()

	testDuration := time.Since(testStart)
	result.DurationSeconds = testDuration.Seconds()

	// Aggregate worker statistics
	var totalLatency time.Duration
	var transactionCount int64

	for _, stats := range workerStats {
		result.TotalTransactions += stats.Transactions
		result.TotalRowsInserted += stats.RowsInserted
		result.TotalErrors += stats.Errors
		totalLatency += stats.TotalLatency
		transactionCount += stats.Transactions

		if stats.Transactions > 0 {
			minMs := float64(stats.MinLatency.Nanoseconds()) / 1000000.0
			maxMs := float64(stats.MaxLatency.Nanoseconds()) / 1000000.0

			if minMs < result.MinLatencyMs {
				result.MinLatencyMs = minMs
			}
			if maxMs > result.MaxLatencyMs {
				result.MaxLatencyMs = maxMs
			}
		}
	}

	// Calculate derived metrics
	if result.DurationSeconds > 0 {
		result.TransactionsPerSec = float64(result.TotalTransactions) / result.DurationSeconds
		result.RowsPerSec = float64(result.TotalRowsInserted) / result.DurationSeconds
	}

	if transactionCount > 0 {
		result.AvgLatencyMs = float64(totalLatency.Nanoseconds()) / float64(transactionCount) / 1000000.0
		result.ErrorRate = float64(result.TotalErrors) / float64(result.TotalTransactions+result.TotalErrors) * 100.0
	}

	return result, nil
}

// runWorkers executes concurrent workers for the test
func (p *BulkLoadPlugin) runWorkers(ctx context.Context, batchSize int, workerStats []*WorkerStats) {
	var wg sync.WaitGroup

	for i := 0; i < p.config.Connections; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			var stats *WorkerStats
			if workerStats != nil {
				stats = workerStats[workerID]
			}

			p.worker(ctx, workerID, batchSize, stats)
		}(i)
	}

	wg.Wait()
}

// worker performs bulk insert operations
func (p *BulkLoadPlugin) worker(ctx context.Context, workerID, batchSize int, stats *WorkerStats) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			start := time.Now()
			err := p.executeBulkInsert(batchSize)
			latency := time.Since(start)

			if stats != nil {
				if err != nil {
					atomic.AddInt64(&stats.Errors, 1)
				} else {
					atomic.AddInt64(&stats.Transactions, 1)
					atomic.AddInt64(&stats.RowsInserted, int64(batchSize))
					stats.TotalLatency += latency

					// Update min/max latency (simple approach, could be more sophisticated)
					if latency < stats.MinLatency {
						stats.MinLatency = latency
					}
					if latency > stats.MaxLatency {
						stats.MaxLatency = latency
					}
				}
			}

			if err != nil && p.config.Verbose {
				p.logger.Warn("Worker insert failed",
					core.Field{Key: "worker_id", Value: workerID},
					core.Field{Key: "batch_size", Value: batchSize},
					core.Field{Key: "error", Value: err.Error()},
				)
			}

			// Think time between batches
			if p.config.ThinkTime > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(p.config.ThinkTime):
				}
			}
		}
	}
}

// executeBulkInsert performs a single bulk insert operation
func (p *BulkLoadPlugin) executeBulkInsert(batchSize int) error {
	// Build bulk insert SQL
	var valuePlaceholders []string
	var values []interface{}

	placeholderIndex := 1
	for i := 0; i < batchSize; i++ {
		var rowPlaceholders []string

		// Add values for each data column
		for j := 1; j <= p.config.DataColumns; j++ {
			switch j % 4 {
			case 0: // Integer
				values = append(values, rand.Intn(1000000))
			case 1: // Text
				values = append(values, fmt.Sprintf("test_data_%d_%d", i, j))
			case 2: // Float
				values = append(values, rand.Float64()*1000.0)
			case 3: // Boolean
				values = append(values, rand.Intn(2) == 1)
			}
			rowPlaceholders = append(rowPlaceholders, fmt.Sprintf("$%d", placeholderIndex))
			placeholderIndex++
		}

		valuePlaceholders = append(valuePlaceholders, "("+strings.Join(rowPlaceholders, ", ")+")")
	}

	// Build column list (excluding id and created_at which have defaults)
	var columns []string
	for i := 1; i <= p.config.DataColumns; i++ {
		switch i % 4 {
		case 0:
			columns = append(columns, fmt.Sprintf("data_int_%d", i))
		case 1:
			columns = append(columns, fmt.Sprintf("data_text_%d", i))
		case 2:
			columns = append(columns, fmt.Sprintf("data_float_%d", i))
		case 3:
			columns = append(columns, fmt.Sprintf("data_bool_%d", i))
		}
	}

	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		p.config.TableName,
		strings.Join(columns, ", "),
		strings.Join(valuePlaceholders, ", "))

	_, err := p.db.Exec(insertSQL, values...)
	return err
}

// NewPlugin returns the plugin instance (required for plugin loading)
func NewPlugin() core.Plugin {
	return &BulkLoadPlugin{}
}

// Main function for standalone testing
func main() {
	// This is only used for testing the plugin as a standalone binary
	// The actual plugin loading uses NewPlugin()
	plugin := NewPlugin()
	fmt.Printf("Bulk Load Plugin: %s v%s\n",
		plugin.Metadata().Name,
		plugin.Metadata().Version)
}
