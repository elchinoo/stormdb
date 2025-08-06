// Package bulkinsert implements a bulk insert performance test plugin for StormDB v0.4-alpha
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

// BulkInsertPlugin implements the bulk insert performance test
type BulkInsertPlugin struct {
	core           *core.CoreServices
	logger         core.Logger
	db             *sql.DB
	config         *BulkInsertConfig
	isRunning      int64
	stopChan       chan struct{}
	wg             sync.WaitGroup
	metrics        *BulkInsertMetrics
	testStarted    time.Time
	currentWorkers []*WorkerStats // Live worker stats for real-time metrics
	workersMu      sync.RWMutex   // Protect access to currentWorkers
	currentBatch   int            // Current batch size being tested
	currentBatchMu sync.RWMutex   // Protect access to currentBatch

	// Previous metrics for delta calculation
	prevTransactions int64        // Previous total transactions
	prevRows         int64        // Previous total rows
	prevSaveTime     time.Time    // Previous save time for rate calculation
	prevMetricsMu    sync.RWMutex // Protect access to previous metrics
}

// BulkInsertConfig defines the configuration for bulk insert tests
type BulkInsertConfig struct {
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

// BulkInsertMetrics tracks performance metrics for bulk insert tests
type BulkInsertMetrics struct {
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
func (p *BulkInsertPlugin) Metadata() core.PluginMetadata {
	return core.PluginMetadata{
		Name:        "bulk-insert",
		Version:     "1.0.0",
		Description: "Bulk insert performance testing with different batch sizes",
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
				"table_name": {"type": "string", "default": "bulk_insert_test_data"},
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
func (p *BulkInsertPlugin) Initialize(ctx context.Context, coreServices *core.CoreServices) error {
	p.core = coreServices
	p.logger = coreServices.Logger.WithPlugin("bulk-insert")
	p.stopChan = make(chan struct{})
	p.metrics = &BulkInsertMetrics{
		BatchResults: make([]BatchResult, 0),
	}

	p.logger.Info("Bulk insert plugin initialized successfully")
	return nil
}

// Validate checks the configuration
func (p *BulkInsertPlugin) Validate(config map[string]interface{}) error {
	var bulkConfig BulkInsertConfig

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
		bulkConfig.TableName = "bulk_insert_test_data"
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
func (p *BulkInsertPlugin) Execute(ctx context.Context, config map[string]interface{}) error {
	if err := p.Validate(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Extract test run ID from context and add to logger
	if testRunID, ok := ctx.Value("test_run_id").(int64); ok {
		p.logger = p.logger.WithFields(core.Field{Key: "test_run_id", Value: testRunID})
		p.logger.Info("Plugin executing with test run ID", core.Field{Key: "test_run_id", Value: testRunID})
	} else {
		p.logger.Warn("No test run ID found in context - this may cause issues with result storage")
	}

	// Set running state
	if !atomic.CompareAndSwapInt64(&p.isRunning, 0, 1) {
		return fmt.Errorf("test is already running")
	}
	defer atomic.StoreInt64(&p.isRunning, 0)

	p.testStarted = time.Now()
	p.metrics.StartTime = p.testStarted

	// Start background metrics saver goroutine
	metricsCtx, cancelMetrics := context.WithCancel(ctx)
	metricsDone := make(chan struct{})
	go p.backgroundMetricsSaver(metricsCtx, metricsDone)
	defer func() {
		cancelMetrics()
		<-metricsDone // Wait for goroutine to finish
	}()

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

		p.logger.Info("Metrics collection update",
			core.Field{Key: "batch_size", Value: batchSize},
			core.Field{Key: "total_transactions", Value: result.TotalTransactions},
			core.Field{Key: "total_rows", Value: result.TotalRowsInserted},
			core.Field{Key: "errors", Value: result.TotalErrors},
			core.Field{Key: "accumulated_total_transactions", Value: p.metrics.TotalTransactions},
			core.Field{Key: "accumulated_total_rows", Value: p.metrics.TotalRowsInserted})

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
	p.logger.Info("Bulk insert performance test completed",
		core.Field{Key: "total_duration", Value: p.metrics.EndTime.Sub(p.metrics.StartTime)},
		core.Field{Key: "total_transactions", Value: p.metrics.TotalTransactions},
		core.Field{Key: "total_rows", Value: p.metrics.TotalRowsInserted},
	)

	// Store results in database
	if err := p.storeResults(ctx); err != nil {
		p.logger.Error("Failed to store test results", core.Field{Key: "error", Value: err.Error()})
		// Don't fail the entire test, just log the error
	} else {
		p.logger.Info("Test results stored successfully")
	}

	// Reset current batch tracking
	p.currentBatchMu.Lock()
	p.currentBatch = 0
	p.currentBatchMu.Unlock()

	// Reset previous metrics tracking
	p.prevMetricsMu.Lock()
	p.prevTransactions = 0
	p.prevRows = 0
	p.prevSaveTime = time.Time{}
	p.prevMetricsMu.Unlock()

	return nil
}

// Cleanup performs any necessary cleanup
func (p *BulkInsertPlugin) Cleanup(ctx context.Context) error {
	// Stop any running operations
	close(p.stopChan)
	p.wg.Wait()

	// Close database connection
	if p.db != nil {
		p.db.Close()
	}

	p.logger.Info("Bulk insert plugin cleanup completed")
	return nil
}

// Helper methods

func (p *BulkInsertPlugin) rebuildDatabase() error {
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
func (p *BulkInsertPlugin) connectDatabase() error {
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
func (p *BulkInsertPlugin) setupTestTable() error {
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
func (p *BulkInsertPlugin) clearTableData() error {
	truncateSQL := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY", p.config.TableName)
	if _, err := p.db.Exec(truncateSQL); err != nil {
		return fmt.Errorf("failed to truncate table: %w", err)
	}
	p.logger.Info("Table data cleared", core.Field{Key: "table", Value: p.config.TableName})
	return nil
}

// runBatchTest executes the test for a specific batch size
func (p *BulkInsertPlugin) runBatchTest(ctx context.Context, batchSize int) (*BatchResult, error) {
	result := &BatchResult{
		BatchSize:    batchSize,
		Connections:  p.config.Connections,
		MinLatencyMs: float64(time.Hour.Milliseconds()), // Initialize with high value
	}

	// Set current batch size for background metrics
	p.currentBatchMu.Lock()
	p.currentBatch = batchSize
	p.currentBatchMu.Unlock()

	// Initialize previous metrics for delta calculation
	p.prevMetricsMu.Lock()
	p.prevTransactions = 0
	p.prevRows = 0
	p.prevSaveTime = time.Time{} // Reset to zero value
	p.prevMetricsMu.Unlock()

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

	// Store worker stats in plugin for background metrics access
	p.workersMu.Lock()
	p.currentWorkers = workerStats
	p.workersMu.Unlock()

	measureCtx, measureCancel := context.WithTimeout(ctx, p.config.Duration)
	testStart := time.Now()

	p.runWorkers(measureCtx, batchSize, workerStats)
	measureCancel()

	// Clear worker stats after measurement
	p.workersMu.Lock()
	p.currentWorkers = nil
	p.workersMu.Unlock()

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
func (p *BulkInsertPlugin) runWorkers(ctx context.Context, batchSize int, workerStats []*WorkerStats) {
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
func (p *BulkInsertPlugin) worker(ctx context.Context, workerID, batchSize int, stats *WorkerStats) {
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
func (p *BulkInsertPlugin) executeBulkInsert(batchSize int) error {
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

// backgroundMetricsSaver continuously saves metrics every second while test is running
// Enhanced version with real-time worker statistics and comprehensive logging
func (p *BulkInsertPlugin) backgroundMetricsSaver(ctx context.Context, done chan<- struct{}) {
	defer close(done)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	iteration := 0

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Background metrics saver stopping")
			return
		case <-ticker.C:
			iteration++

			now := time.Now()

			// Get current accumulated metrics
			p.metrics.mu.RLock()
			accumulatedTransactions := p.metrics.TotalTransactions
			accumulatedRows := p.metrics.TotalRowsInserted
			p.metrics.mu.RUnlock()

			// Get current live worker statistics
			p.workersMu.RLock()
			currentWorkers := make([]*WorkerStats, len(p.currentWorkers))
			copy(currentWorkers, p.currentWorkers)
			p.workersMu.RUnlock()

			// Calculate live transactions/rows from current workers
			var liveTransactions, liveRows int64
			for _, worker := range currentWorkers {
				if worker != nil {
					liveTransactions += atomic.LoadInt64(&worker.Transactions)
					liveRows += atomic.LoadInt64(&worker.RowsInserted)
				}
			}

			// Debug logging
			p.logger.Debug("Background metrics check",
				core.Field{Key: "save_iteration", Value: iteration},
				core.Field{Key: "accumulated_transactions", Value: accumulatedTransactions},
				core.Field{Key: "live_transactions", Value: liveTransactions},
				core.Field{Key: "worker_count", Value: len(currentWorkers)},
			)

			// Total transactions = accumulated from completed batches + live from current batch
			totalTransactions := accumulatedTransactions + liveTransactions
			totalRows := accumulatedRows + liveRows

			// Save cumulative metrics as snapshot
			if totalTransactions > 0 {
				// Calculate elapsed time for rates
				elapsed := now.Sub(p.metrics.StartTime).Seconds()
				if elapsed > 0 {
					transactionRate := float64(totalTransactions) / elapsed
					rowRate := float64(totalRows) / elapsed

					err := p.saveCurrentMetrics(ctx, iteration)
					if err != nil {
						p.logger.Error("Failed to save background metrics",
							core.Field{Key: "error", Value: err.Error()},
							core.Field{Key: "save_iteration", Value: iteration})
					} else {
						p.logger.Debug("Background metrics saved",
							core.Field{Key: "save_iteration", Value: iteration},
							core.Field{Key: "total_transactions", Value: totalTransactions},
							core.Field{Key: "total_rows", Value: totalRows},
							core.Field{Key: "transaction_rate", Value: transactionRate},
							core.Field{Key: "row_rate", Value: rowRate},
							core.Field{Key: "active_workers", Value: len(currentWorkers)})
					}
				}
			} else {
				p.logger.Debug("No metrics to save - no transactions yet",
					core.Field{Key: "save_iteration", Value: iteration},
					core.Field{Key: "worker_count", Value: len(currentWorkers)})
			}
		}
	}
}

// saveCurrentMetrics saves current accumulated metrics to database
func (p *BulkInsertPlugin) saveCurrentMetrics(ctx context.Context, iteration int) error {
	if p.core == nil || p.core.Storage == nil {
		return fmt.Errorf("core services not available")
	}

	// Extract test run ID from context
	testRunID, ok := ctx.Value("test_run_id").(int64)
	if !ok {
		return fmt.Errorf("test_run_id not found in context")
	}

	// Get metric IDs
	rowInsertMetric, err := p.core.Storage.GetMetric(ctx, "ROW_INSERT")
	if err != nil {
		return fmt.Errorf("failed to get ROW_INSERT metric: %w", err)
	}

	latencyAvgMetric, err := p.core.Storage.GetMetric(ctx, "LATENCY_AVG")
	if err != nil {
		return fmt.Errorf("failed to get LATENCY_AVG metric: %w", err)
	}

	throughputMetric, err := p.core.Storage.GetMetric(ctx, "THROUGHPUT")
	if err != nil {
		return fmt.Errorf("failed to get THROUGHPUT metric: %w", err)
	}

	now := time.Now()
	var results []core.TestResult

	// Get current batch size and worker stats
	p.currentBatchMu.RLock()
	currentBatchSize := p.currentBatch
	p.currentBatchMu.RUnlock()

	p.workersMu.RLock()
	currentWorkers := p.currentWorkers
	activeWorkers := len(currentWorkers)
	activeConnections := p.config.Connections
	p.workersMu.RUnlock()

	// If no current batch is running, don't save metrics
	if currentBatchSize == 0 || currentWorkers == nil {
		return nil
	}

	// Calculate current live totals from active workers
	var liveTransactions, liveRows int64
	var totalLatency time.Duration
	var operationCount int64

	for _, worker := range currentWorkers {
		if worker != nil {
			transactions := atomic.LoadInt64(&worker.Transactions)
			rows := atomic.LoadInt64(&worker.RowsInserted)

			liveTransactions += transactions
			liveRows += rows
			totalLatency += worker.TotalLatency
			operationCount += transactions
		}
	}

	// Get previous values for delta calculation
	p.prevMetricsMu.Lock()
	prevTransactions := p.prevTransactions
	prevRows := p.prevRows
	prevTime := p.prevSaveTime

	// Calculate deltas (incremental values since last save)
	deltaTransactions := liveTransactions - prevTransactions
	deltaRows := liveRows - prevRows
	timeDelta := now.Sub(prevTime).Seconds()

	// Update previous values for next iteration
	p.prevTransactions = liveTransactions
	p.prevRows = liveRows
	p.prevSaveTime = now
	p.prevMetricsMu.Unlock()

	// Skip first iteration since we don't have previous values yet
	if prevTime.IsZero() || timeDelta <= 0 {
		return nil
	}

	// Only save if we have incremental data
	if deltaTransactions > 0 && timeDelta > 0 {
		// Calculate rates per second for this interval
		transactionRate := float64(deltaTransactions) / timeDelta
		rowRate := float64(deltaRows) / timeDelta

		// Store incremental transaction rate
		results = append(results, core.TestResult{
			TestRunID:         testRunID,
			MetricID:          throughputMetric.ID,
			StartTime:         prevTime,
			EndTime:           now,
			Value:             transactionRate,
			ActiveConnections: &activeConnections,
			ActiveWorkers:     &activeWorkers,
			Tags: map[string]interface{}{
				"metric_type":           "interval_transaction_rate",
				"iteration":             iteration,
				"connections":           p.config.Connections,
				"batch_size":            currentBatchSize,
				"interval_transactions": deltaTransactions,
				"interval_rows":         deltaRows,
				"interval_seconds":      timeDelta,
				"test_phase":            "measurement",
			},
		})

		// Store incremental row insertion rate
		results = append(results, core.TestResult{
			TestRunID:         testRunID,
			MetricID:          rowInsertMetric.ID,
			StartTime:         prevTime,
			EndTime:           now,
			Value:             rowRate,
			ActiveConnections: &activeConnections,
			ActiveWorkers:     &activeWorkers,
			Tags: map[string]interface{}{
				"metric_type":           "interval_row_rate",
				"iteration":             iteration,
				"connections":           p.config.Connections,
				"batch_size":            currentBatchSize,
				"interval_transactions": deltaTransactions,
				"interval_rows":         deltaRows,
				"interval_seconds":      timeDelta,
				"test_phase":            "measurement",
			},
		})

		// Calculate average latency for this interval (using current running average)
		if operationCount > 0 {
			avgLatencyMs := float64(totalLatency.Nanoseconds()) / float64(operationCount) / 1000000.0

			results = append(results, core.TestResult{
				TestRunID:         testRunID,
				MetricID:          latencyAvgMetric.ID,
				StartTime:         prevTime,
				EndTime:           now,
				Value:             avgLatencyMs,
				ActiveConnections: &activeConnections,
				ActiveWorkers:     &activeWorkers,
				Tags: map[string]interface{}{
					"metric_type":           "interval_avg_latency",
					"iteration":             iteration,
					"connections":           p.config.Connections,
					"batch_size":            currentBatchSize,
					"interval_transactions": deltaTransactions,
					"interval_rows":         deltaRows,
					"interval_seconds":      timeDelta,
					"test_phase":            "measurement",
				},
			})
		}
	}

	if len(results) > 0 {
		p.logger.Debug("Saving interval metrics",
			core.Field{Key: "test_run_id", Value: testRunID},
			core.Field{Key: "result_count", Value: len(results)},
			core.Field{Key: "iteration", Value: iteration},
			core.Field{Key: "batch_size", Value: currentBatchSize},
			core.Field{Key: "delta_transactions", Value: deltaTransactions},
			core.Field{Key: "interval_seconds", Value: timeDelta})

		return p.core.Storage.StoreResults(ctx, results)
	}

	return nil
} // storeResults converts plugin metrics to core.TestResult and stores them in the database
func (p *BulkInsertPlugin) storeResults(ctx context.Context) error {
	if p.core == nil || p.core.Storage == nil {
		return fmt.Errorf("core services not available")
	}

	// Extract test run ID from context
	testRunID, ok := ctx.Value("test_run_id").(int64)
	if !ok {
		p.logger.Warn("test run ID not found in context, results may not be associated correctly")
		testRunID = 0
	}

	// Look up metric IDs from database instead of using hardcoded values
	rowInsertMetric, err := p.core.Storage.GetMetric(ctx, "ROW_INSERT")
	if err != nil {
		return fmt.Errorf("failed to get ROW_INSERT metric: %w", err)
	}

	latencyAvgMetric, err := p.core.Storage.GetMetric(ctx, "LATENCY_AVG")
	if err != nil {
		return fmt.Errorf("failed to get LATENCY_AVG metric: %w", err)
	}

	var results []core.TestResult
	now := time.Now()

	if len(p.metrics.BatchResults) == 0 {
		p.logger.Warn("No batch results to store - test may not have run properly")
		return nil
	}

	p.logger.Info("Processing batch results for storage",
		core.Field{Key: "batch_count", Value: len(p.metrics.BatchResults)},
		core.Field{Key: "total_transactions", Value: p.metrics.TotalTransactions},
		core.Field{Key: "total_rows", Value: p.metrics.TotalRowsInserted},
	)

	// Active connections and workers for final results
	activeConnections := p.config.Connections
	activeWorkers := 0 // No workers active at completion

	// Convert each batch result to database format
	for _, batch := range p.metrics.BatchResults {
		// Store transaction rate
		results = append(results, core.TestResult{
			TestRunID:         testRunID,
			MetricID:          rowInsertMetric.ID,
			StartTime:         p.metrics.StartTime,
			EndTime:           now,
			Value:             batch.TransactionsPerSec,
			ActiveConnections: &activeConnections,
			ActiveWorkers:     &activeWorkers,
			Tags: map[string]interface{}{
				"metric_type":        "transactions_per_sec",
				"iteration":          nil, // Not applicable for final batch results
				"connections":        batch.Connections,
				"batch_size":         batch.BatchSize,
				"total_transactions": batch.TotalTransactions,
				"total_rows":         batch.TotalRowsInserted,
				"test_phase":         "final_results",
			},
		})

		// Store rows per second
		results = append(results, core.TestResult{
			TestRunID:         testRunID,
			MetricID:          rowInsertMetric.ID,
			StartTime:         p.metrics.StartTime,
			EndTime:           now,
			Value:             batch.RowsPerSec,
			ActiveConnections: &activeConnections,
			ActiveWorkers:     &activeWorkers,
			Tags: map[string]interface{}{
				"metric_type":        "rows_per_sec",
				"iteration":          nil, // Not applicable for final batch results
				"connections":        batch.Connections,
				"batch_size":         batch.BatchSize,
				"total_transactions": batch.TotalTransactions,
				"total_rows":         batch.TotalRowsInserted,
				"test_phase":         "final_results",
			},
		})

		// Store average latency
		results = append(results, core.TestResult{
			TestRunID:         testRunID,
			MetricID:          latencyAvgMetric.ID,
			StartTime:         p.metrics.StartTime,
			EndTime:           now,
			Value:             batch.AvgLatencyMs,
			ActiveConnections: &activeConnections,
			ActiveWorkers:     &activeWorkers,
			Tags: map[string]interface{}{
				"metric_type":        "avg_latency_ms",
				"iteration":          nil, // Not applicable for final batch results
				"connections":        batch.Connections,
				"batch_size":         batch.BatchSize,
				"total_transactions": batch.TotalTransactions,
				"total_rows":         batch.TotalRowsInserted,
				"test_phase":         "final_results",
			},
		})
	}

	p.logger.Info("Storing test results",
		core.Field{Key: "test_run_id", Value: testRunID},
		core.Field{Key: "result_count", Value: len(results)},
		core.Field{Key: "batch_count", Value: len(p.metrics.BatchResults)},
		core.Field{Key: "metrics_start_time", Value: p.metrics.StartTime},
		core.Field{Key: "metrics_end_time", Value: p.metrics.EndTime},
		core.Field{Key: "total_transactions", Value: p.metrics.TotalTransactions},
		core.Field{Key: "total_rows", Value: p.metrics.TotalRowsInserted},
	)

	// Store all results
	storeErr := p.core.Storage.StoreResults(ctx, results)
	if storeErr != nil {
		p.logger.Error("Failed to store results to database", core.Field{Key: "error", Value: storeErr.Error()})
		return storeErr
	}

	p.logger.Info("Successfully stored test results to database",
		core.Field{Key: "stored_results", Value: len(results)},
	)
	return nil
}

// NewPlugin returns the plugin instance (required for plugin loading)
func NewPlugin() core.Plugin {
	return &BulkInsertPlugin{}
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
