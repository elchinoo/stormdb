// Package tpcc implements a TPC-C inspired scalability test plugin for StormDB v0.4-alpha.1
// This plugin performs incremental connection testing with configurable parameters
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elchinoo/stormdb/core"
	_ "github.com/lib/pq"
)

// TPCCPlugin implements the TPC-C scalability test
type TPCCPlugin struct {
	core        *core.CoreServices
	logger      core.Logger
	db          *sql.DB
	config      *TPCCConfig
	isRunning   int64
	stopChan    chan struct{}
	wg          sync.WaitGroup
	metrics     *TPCCMetrics
	testStarted time.Time
}

// TPCCConfig defines the configuration for TPC-C scalability tests
type TPCCConfig struct {
	// Database connection
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Database string `json:"database" yaml:"database"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	SSLMode  string `json:"ssl_mode" yaml:"ssl_mode"`

	// Test configuration
	Scale       int           `json:"scale" yaml:"scale"`             // TPC-C scale factor (warehouses)
	Connections []int         `json:"connections" yaml:"connections"` // Connection counts to test [48, 96, 192, 256]
	Duration    time.Duration `json:"duration" yaml:"duration"`       // Duration per connection level
	WarmupTime  time.Duration `json:"warmup_time" yaml:"warmup_time"` // Warmup before measurements
	ThinkTime   time.Duration `json:"think_time" yaml:"think_time"`   // Delay between transactions

	// Transaction mix (percentages)
	NewOrderPct    int `json:"new_order_pct" yaml:"new_order_pct"`       // 45%
	PaymentPct     int `json:"payment_pct" yaml:"payment_pct"`           // 43%
	OrderStatusPct int `json:"order_status_pct" yaml:"order_status_pct"` // 4%
	DeliveryPct    int `json:"delivery_pct" yaml:"delivery_pct"`         // 4%
	StockLevelPct  int `json:"stock_level_pct" yaml:"stock_level_pct"`   // 4%

	// Performance settings
	BatchSize       int  `json:"batch_size" yaml:"batch_size"`             // Batch operations where possible
	EnableMetrics   bool `json:"enable_metrics" yaml:"enable_metrics"`     // Detailed metrics collection
	LogTransactions bool `json:"log_transactions" yaml:"log_transactions"` // Log individual transactions
}

// TPCCMetrics tracks performance metrics during test execution
type TPCCMetrics struct {
	mu sync.RWMutex

	// Transaction counters
	NewOrderCount    int64 `json:"new_order_count"`
	PaymentCount     int64 `json:"payment_count"`
	OrderStatusCount int64 `json:"order_status_count"`
	DeliveryCount    int64 `json:"delivery_count"`
	StockLevelCount  int64 `json:"stock_level_count"`

	// Error counters
	ErrorCount   int64 `json:"error_count"`
	TimeoutCount int64 `json:"timeout_count"`

	// Timing statistics
	TotalLatency time.Duration `json:"total_latency_ns"`
	MinLatency   time.Duration `json:"min_latency_ns"`
	MaxLatency   time.Duration `json:"max_latency_ns"`

	// Connection level metrics
	CurrentConnections int    `json:"current_connections"`
	TestPhase          string `json:"test_phase"`
}

// Default configuration values
var defaultConfig = TPCCConfig{
	Scale:           10,
	Connections:     []int{48, 96, 192, 256},
	Duration:        5 * time.Minute,
	WarmupTime:      30 * time.Second,
	ThinkTime:       100 * time.Millisecond,
	NewOrderPct:     45,
	PaymentPct:      43,
	OrderStatusPct:  4,
	DeliveryPct:     4,
	StockLevelPct:   4,
	BatchSize:       100,
	EnableMetrics:   true,
	LogTransactions: false,
	SSLMode:         "disable",
}

// Plugin interface implementation

// Metadata returns plugin metadata
func (p *TPCCPlugin) Metadata() core.PluginMetadata {
	return core.PluginMetadata{
		Name:         "tpcc-scalability",
		Version:      "1.0.0",
		Description:  "TPC-C inspired scalability test with incremental connection testing",
		Author:       "StormDB Team",
		License:      "MIT",
		TestTypes:    []string{"scalability", "tpcc", "oltp"},
		ConfigSchema: getConfigSchema(),
		Dependencies: map[string]string{
			"postgresql": ">=12.0",
		},
	}
}

// Initialize sets up the plugin with core services
func (p *TPCCPlugin) Initialize(ctx context.Context, coreServices *core.CoreServices) error {
	p.core = coreServices
	p.logger = coreServices.Logger.WithPlugin("tpcc-scalability")
	p.stopChan = make(chan struct{})
	p.metrics = &TPCCMetrics{
		MinLatency: time.Hour, // Initialize to high value
	}

	p.logger.Info("TPC-C scalability plugin initialized")
	return nil
}

// Validate checks the plugin configuration
func (p *TPCCPlugin) Validate(config map[string]interface{}) error {
	cfg, err := parseConfig(config)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate connection counts
	if len(cfg.Connections) == 0 {
		return fmt.Errorf("connections array cannot be empty")
	}

	for _, conn := range cfg.Connections {
		if conn <= 0 {
			return fmt.Errorf("connection count must be positive: %d", conn)
		}
		if conn > 1000 {
			return fmt.Errorf("connection count too high (max 1000): %d", conn)
		}
	}

	// Validate scale factor
	if cfg.Scale <= 0 {
		return fmt.Errorf("scale factor must be positive: %d", cfg.Scale)
	}

	// Validate transaction mix
	totalPct := cfg.NewOrderPct + cfg.PaymentPct + cfg.OrderStatusPct + cfg.DeliveryPct + cfg.StockLevelPct
	if totalPct != 100 {
		return fmt.Errorf("transaction percentages must sum to 100, got %d", totalPct)
	}

	// Validate duration
	if cfg.Duration <= 0 {
		return fmt.Errorf("test duration must be positive")
	}

	p.config = cfg
	return nil
}

// Execute runs the TPC-C scalability test
func (p *TPCCPlugin) Execute(ctx context.Context, config map[string]interface{}) error {
	if err := p.Validate(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	if !atomic.CompareAndSwapInt64(&p.isRunning, 0, 1) {
		return fmt.Errorf("plugin is already running")
	}
	defer atomic.StoreInt64(&p.isRunning, 0)

	p.testStarted = time.Now()
	p.logger.Info("starting TPC-C scalability test",
		core.Field{Key: "scale", Value: p.config.Scale},
		core.Field{Key: "connection_levels", Value: len(p.config.Connections)},
	)

	// Connect to database
	if err := p.connectDB(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer p.db.Close()

	// Prepare database schema
	if err := p.prepareSchema(ctx); err != nil {
		return fmt.Errorf("failed to prepare schema: %w", err)
	}

	// Populate test data
	if err := p.populateData(ctx); err != nil {
		return fmt.Errorf("failed to populate test data: %w", err)
	}

	// Create test run
	testRun := &core.TestRun{
		TestTypeID:  1, // Will be resolved by storage layer
		PluginID:    1, // Will be resolved by storage layer
		PluginVer:   "1.0.0",
		Host:        p.config.Host,
		Port:        p.config.Port,
		DBName:      p.config.Database,
		Name:        fmt.Sprintf("TPC-C Scalability Test - Scale %d", p.config.Scale),
		Description: fmt.Sprintf("Incremental connection test: %v", p.config.Connections),
		Status:      core.StatusRunning,
		Config:      config,
		CreatedAt:   time.Now(),
	}

	runID, err := p.core.Storage.CreateTestRun(ctx, testRun)
	if err != nil {
		return fmt.Errorf("failed to create test run: %w", err)
	}

	p.logger.Info("test run created",
		core.Field{Key: "run_id", Value: runID},
	)

	// Execute test for each connection level
	for i, connCount := range p.config.Connections {
		p.logger.Info("starting connection level test",
			core.Field{Key: "level", Value: i + 1},
			core.Field{Key: "connections", Value: connCount},
		)

		p.metrics.CurrentConnections = connCount
		p.metrics.TestPhase = fmt.Sprintf("Level %d/%d", i+1, len(p.config.Connections))

		if err := p.runConnectionLevel(ctx, runID, connCount); err != nil {
			p.logger.Error("connection level test failed",
				core.Field{Key: "level", Value: i + 1},
				core.Field{Key: "connections", Value: connCount},
				core.Field{Key: "error", Value: err.Error()},
			)

			// Continue with next level unless context is cancelled
			if ctx.Err() != nil {
				break
			}
		}
	}

	// Update test run status
	if err := p.core.Storage.UpdateTestRunStatus(ctx, runID, core.StatusSucceeded); err != nil {
		p.logger.Warn("failed to update test run status", core.Field{Key: "error", Value: err.Error()})
	}

	p.logger.Info("TPC-C scalability test completed",
		core.Field{Key: "run_id", Value: runID},
		core.Field{Key: "duration", Value: time.Since(p.testStarted)},
	)

	return nil
}

// Cleanup performs cleanup operations
func (p *TPCCPlugin) Cleanup(ctx context.Context) error {
	p.logger.Info("cleaning up TPC-C plugin")

	// Signal stop
	if p.stopChan != nil {
		close(p.stopChan)
	}

	// Wait for goroutines
	p.wg.Wait()

	// Close database connection
	if p.db != nil {
		if err := p.db.Close(); err != nil {
			p.logger.Warn("error closing database connection", core.Field{Key: "error", Value: err.Error()})
		}
	}

	return nil
}

// Helper methods

func (p *TPCCPlugin) connectDB() error {
	connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		p.config.Host, p.config.Port, p.config.Database,
		p.config.Username, p.config.Password, p.config.SSLMode)

	var err error
	p.db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := p.db.PingContext(ctx); err != nil {
		return err
	}

	p.logger.Info("database connection established")
	return nil
}

func parseConfig(config map[string]interface{}) (*TPCCConfig, error) {
	// Start with defaults
	cfg := defaultConfig

	// Marshal to JSON and back to populate struct
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (p *TPCCPlugin) prepareSchema(ctx context.Context) error {
	p.logger.Info("preparing TPC-C database schema")

	// Create TPC-C tables if they don't exist
	schemas := []string{
		`CREATE TABLE IF NOT EXISTS tpcc_warehouse (
			w_id INTEGER PRIMARY KEY,
			w_name VARCHAR(10),
			w_street_1 VARCHAR(20),
			w_street_2 VARCHAR(20),
			w_city VARCHAR(20),
			w_state CHAR(2),
			w_zip CHAR(9),
			w_tax DECIMAL(4,2),
			w_ytd DECIMAL(12,2)
		)`,
		`CREATE TABLE IF NOT EXISTS tpcc_district (
			d_id INTEGER,
			d_w_id INTEGER,
			d_name VARCHAR(10),
			d_street_1 VARCHAR(20),
			d_street_2 VARCHAR(20),
			d_city VARCHAR(20),
			d_state CHAR(2),
			d_zip CHAR(9),
			d_tax DECIMAL(4,2),
			d_ytd DECIMAL(12,2),
			d_next_o_id INTEGER,
			PRIMARY KEY (d_w_id, d_id)
		)`,
		`CREATE TABLE IF NOT EXISTS tpcc_customer (
			c_id INTEGER,
			c_d_id INTEGER,
			c_w_id INTEGER,
			c_first VARCHAR(16),
			c_middle CHAR(2),
			c_last VARCHAR(16),
			c_street_1 VARCHAR(20),
			c_street_2 VARCHAR(20),
			c_city VARCHAR(20),
			c_state CHAR(2),
			c_zip CHAR(9),
			c_phone CHAR(16),
			c_since TIMESTAMP,
			c_credit CHAR(2),
			c_credit_lim DECIMAL(12,2),
			c_discount DECIMAL(4,2),
			c_balance DECIMAL(12,2),
			c_ytd_payment DECIMAL(12,2),
			c_payment_cnt INTEGER,
			c_delivery_cnt INTEGER,
			c_data TEXT,
			PRIMARY KEY (c_w_id, c_d_id, c_id)
		)`,
		`CREATE TABLE IF NOT EXISTS tpcc_item (
			i_id INTEGER PRIMARY KEY,
			i_im_id INTEGER,
			i_name VARCHAR(24),
			i_price DECIMAL(5,2),
			i_data VARCHAR(50)
		)`,
		`CREATE TABLE IF NOT EXISTS tpcc_stock (
			s_i_id INTEGER,
			s_w_id INTEGER,
			s_quantity INTEGER,
			s_dist_01 CHAR(24),
			s_dist_02 CHAR(24),
			s_dist_03 CHAR(24),
			s_dist_04 CHAR(24),
			s_dist_05 CHAR(24),
			s_dist_06 CHAR(24),
			s_dist_07 CHAR(24),
			s_dist_08 CHAR(24),
			s_dist_09 CHAR(24),
			s_dist_10 CHAR(24),
			s_ytd INTEGER,
			s_order_cnt INTEGER,
			s_remote_cnt INTEGER,
			s_data VARCHAR(50),
			PRIMARY KEY (s_w_id, s_i_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_customer_last ON tpcc_customer (c_w_id, c_d_id, c_last, c_first)`,
		`CREATE INDEX IF NOT EXISTS idx_stock_quantity ON tpcc_stock (s_w_id, s_i_id, s_quantity)`,
	}

	for _, schema := range schemas {
		if _, err := p.db.ExecContext(ctx, schema); err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}
	}

	p.logger.Info("TPC-C schema prepared successfully")
	return nil
}

func (p *TPCCPlugin) populateData(ctx context.Context) error {
	p.logger.Info("populating TPC-C test data", core.Field{Key: "scale", Value: p.config.Scale})

	// Check if data already exists
	var count int
	if err := p.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tpcc_warehouse").Scan(&count); err != nil {
		return fmt.Errorf("failed to check existing data: %w", err)
	}

	if count >= p.config.Scale {
		p.logger.Info("test data already exists", core.Field{Key: "warehouses", Value: count})
		return nil
	}

	// Populate warehouses
	for w := 1; w <= p.config.Scale; w++ {
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO tpcc_warehouse (w_id, w_name, w_street_1, w_street_2, w_city, w_state, w_zip, w_tax, w_ytd)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (w_id) DO NOTHING`,
			w, fmt.Sprintf("Warehouse_%d", w), "123 Main St", "", "Anytown", "NY", "12345-1234", 0.10, 300000.00)
		if err != nil {
			return fmt.Errorf("failed to insert warehouse %d: %w", w, err)
		}

		// Populate districts for this warehouse
		for d := 1; d <= 10; d++ {
			_, err := p.db.ExecContext(ctx, `
				INSERT INTO tpcc_district (d_id, d_w_id, d_name, d_street_1, d_street_2, d_city, d_state, d_zip, d_tax, d_ytd, d_next_o_id)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
				ON CONFLICT (d_w_id, d_id) DO NOTHING`,
				d, w, fmt.Sprintf("District_%d", d), "456 Oak Ave", "", "Anytown", "NY", "12345-1234", 0.10, 30000.00, 3001)
			if err != nil {
				return fmt.Errorf("failed to insert district %d/%d: %w", w, d, err)
			}
		}
	}

	// Populate items (shared across all warehouses)
	if err := p.populateItems(ctx); err != nil {
		return fmt.Errorf("failed to populate items: %w", err)
	}

	// Populate stock for each warehouse
	for w := 1; w <= p.config.Scale; w++ {
		if err := p.populateStock(ctx, w); err != nil {
			return fmt.Errorf("failed to populate stock for warehouse %d: %w", w, err)
		}
	}

	p.logger.Info("TPC-C test data populated successfully")
	return nil
}

func (p *TPCCPlugin) populateItems(ctx context.Context) error {
	// Check if items already exist
	var count int
	if err := p.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tpcc_item").Scan(&count); err != nil {
		return err
	}

	if count >= 100000 {
		return nil // Items already populated
	}

	p.logger.Info("populating items table")

	// Create items in batches
	batchSize := 1000
	for start := 1; start <= 100000; start += batchSize {
		end := start + batchSize - 1
		if end > 100000 {
			end = 100000
		}

		if err := p.insertItemBatch(ctx, start, end); err != nil {
			return err
		}
	}

	return nil
}

func (p *TPCCPlugin) insertItemBatch(ctx context.Context, start, end int) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO tpcc_item (i_id, i_im_id, i_name, i_price, i_data)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (i_id) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := start; i <= end; i++ {
		_, err := stmt.ExecContext(ctx, i, rand.Intn(10000),
			fmt.Sprintf("Item_%d", i),
			float64(rand.Intn(10000))/100.0,
			fmt.Sprintf("Item %d data", i))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (p *TPCCPlugin) populateStock(ctx context.Context, warehouseID int) error {
	p.logger.Info("populating stock for warehouse", core.Field{Key: "warehouse", Value: warehouseID})

	// Create stock in batches
	batchSize := 1000
	for start := 1; start <= 100000; start += batchSize {
		end := start + batchSize - 1
		if end > 100000 {
			end = 100000
		}

		if err := p.insertStockBatch(ctx, warehouseID, start, end); err != nil {
			return err
		}
	}

	return nil
}

func (p *TPCCPlugin) insertStockBatch(ctx context.Context, warehouseID, start, end int) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO tpcc_stock (s_i_id, s_w_id, s_quantity, s_dist_01, s_dist_02, s_dist_03, s_dist_04, s_dist_05,
			s_dist_06, s_dist_07, s_dist_08, s_dist_09, s_dist_10, s_ytd, s_order_cnt, s_remote_cnt, s_data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (s_w_id, s_i_id) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := start; i <= end; i++ {
		_, err := stmt.ExecContext(ctx, i, warehouseID, rand.Intn(100)+10,
			"DIST_01", "DIST_02", "DIST_03", "DIST_04", "DIST_05",
			"DIST_06", "DIST_07", "DIST_08", "DIST_09", "DIST_10",
			0, 0, 0, fmt.Sprintf("Stock %d data", i))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (p *TPCCPlugin) runConnectionLevel(ctx context.Context, runID int64, connCount int) error {
	p.logger.Info("starting connection level execution",
		core.Field{Key: "run_id", Value: runID},
		core.Field{Key: "connections", Value: connCount},
	)

	// Reset metrics for this level
	p.resetMetrics()
	p.metrics.CurrentConnections = connCount

	// Create worker pool
	var wg sync.WaitGroup
	resultsChan := make(chan *core.TestResult, connCount*100)

	// Start result collector
	go p.collectResults(ctx, runID, resultsChan)

	// Warmup phase
	if p.config.WarmupTime > 0 {
		p.metrics.TestPhase = "warmup"
		p.logger.Info("starting warmup phase", core.Field{Key: "duration", Value: p.config.WarmupTime})
		p.runWorkers(ctx, connCount, p.config.WarmupTime, false, resultsChan, &wg)
		wg.Wait()
	}

	// Measurement phase
	p.metrics.TestPhase = "measurement"
	p.logger.Info("starting measurement phase", core.Field{Key: "duration", Value: p.config.Duration})
	measurementStart := time.Now()
	p.runWorkers(ctx, connCount, p.config.Duration, true, resultsChan, &wg)
	wg.Wait()
	measurementDuration := time.Since(measurementStart)

	// Close results channel
	close(resultsChan)

	// Log summary
	p.logConnectionLevelSummary(connCount, measurementDuration)

	return nil
}

func (p *TPCCPlugin) runWorkers(ctx context.Context, connCount int, duration time.Duration, collectMetrics bool, resultsChan chan<- *core.TestResult, wg *sync.WaitGroup) {
	stopTime := time.Now().Add(duration)

	for i := 0; i < connCount; i++ {
		wg.Add(1)
		go p.worker(ctx, i, stopTime, collectMetrics, resultsChan, wg)
	}
}

func (p *TPCCPlugin) worker(ctx context.Context, workerID int, stopTime time.Time, collectMetrics bool, resultsChan chan<- *core.TestResult, wg *sync.WaitGroup) {
	defer wg.Done()

	// Create connection for this worker
	connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		p.config.Host, p.config.Port, p.config.Database,
		p.config.Username, p.config.Password, p.config.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		p.logger.Error("worker failed to connect to database",
			core.Field{Key: "worker_id", Value: workerID},
			core.Field{Key: "error", Value: err.Error()},
		)
		return
	}
	defer db.Close()

	for time.Now().Before(stopTime) {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		default:
		}

		// Execute transaction
		txType := p.selectTransactionType()
		start := time.Now()
		err := p.executeTransaction(ctx, db, txType)
		latency := time.Since(start)

		if collectMetrics {
			p.updateMetrics(txType, latency, err)

			if p.config.EnableMetrics && resultsChan != nil {
				result := &core.TestResult{
					TestRunID: 0, // Will be set by collector
					MetricID:  p.getMetricID(txType),
					StartTime: start,
					EndTime:   start.Add(latency),
					Value:     float64(latency.Nanoseconds()),
					Tags:      fmt.Sprintf(`{"worker_id":%d,"tx_type":"%s","connections":%d}`, workerID, txType, p.metrics.CurrentConnections),
				}

				select {
				case resultsChan <- result:
				default:
					// Channel full, skip this result
				}
			}
		}

		// Think time
		if p.config.ThinkTime > 0 {
			time.Sleep(p.config.ThinkTime)
		}
	}
}

func (p *TPCCPlugin) selectTransactionType() string {
	r := rand.Intn(100)

	if r < p.config.NewOrderPct {
		return "new_order"
	}
	r -= p.config.NewOrderPct

	if r < p.config.PaymentPct {
		return "payment"
	}
	r -= p.config.PaymentPct

	if r < p.config.OrderStatusPct {
		return "order_status"
	}
	r -= p.config.OrderStatusPct

	if r < p.config.DeliveryPct {
		return "delivery"
	}

	return "stock_level"
}

func (p *TPCCPlugin) executeTransaction(ctx context.Context, db *sql.DB, txType string) error {
	switch txType {
	case "new_order":
		return p.executeNewOrder(ctx, db)
	case "payment":
		return p.executePayment(ctx, db)
	case "order_status":
		return p.executeOrderStatus(ctx, db)
	case "delivery":
		return p.executeDelivery(ctx, db)
	case "stock_level":
		return p.executeStockLevel(ctx, db)
	default:
		return fmt.Errorf("unknown transaction type: %s", txType)
	}
}

func (p *TPCCPlugin) executeNewOrder(ctx context.Context, db *sql.DB) error {
	// Simplified new order transaction
	warehouseID := rand.Intn(p.config.Scale) + 1

	var quantity int
	err := db.QueryRowContext(ctx,
		"SELECT s_quantity FROM tpcc_stock WHERE s_w_id = $1 AND s_i_id = $2",
		warehouseID, rand.Intn(100000)+1).Scan(&quantity)

	return err
}

func (p *TPCCPlugin) executePayment(ctx context.Context, db *sql.DB) error {
	// Simplified payment transaction
	warehouseID := rand.Intn(p.config.Scale) + 1
	districtID := rand.Intn(10) + 1
	customerID := rand.Intn(3000) + 1

	_, err := db.ExecContext(ctx,
		"UPDATE tpcc_customer SET c_balance = c_balance - $1 WHERE c_w_id = $2 AND c_d_id = $3 AND c_id = $4",
		100.0, warehouseID, districtID, customerID)

	return err
}

func (p *TPCCPlugin) executeOrderStatus(ctx context.Context, db *sql.DB) error {
	// Simplified order status transaction
	warehouseID := rand.Intn(p.config.Scale) + 1
	districtID := rand.Intn(10) + 1
	customerID := rand.Intn(3000) + 1

	var balance float64
	err := db.QueryRowContext(ctx,
		"SELECT c_balance FROM tpcc_customer WHERE c_w_id = $1 AND c_d_id = $2 AND c_id = $3",
		warehouseID, districtID, customerID).Scan(&balance)

	return err
}

func (p *TPCCPlugin) executeDelivery(ctx context.Context, db *sql.DB) error {
	// Simplified delivery transaction
	warehouseID := rand.Intn(p.config.Scale) + 1

	for districtID := 1; districtID <= 10; districtID++ {
		_, err := db.ExecContext(ctx,
			"UPDATE tpcc_district SET d_next_o_id = d_next_o_id + 1 WHERE d_w_id = $1 AND d_id = $2",
			warehouseID, districtID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *TPCCPlugin) executeStockLevel(ctx context.Context, db *sql.DB) error {
	// Simplified stock level transaction
	warehouseID := rand.Intn(p.config.Scale) + 1

	var count int
	err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tpcc_stock WHERE s_w_id = $1 AND s_quantity < $2",
		warehouseID, 20).Scan(&count)

	return err
}

func (p *TPCCPlugin) updateMetrics(txType string, latency time.Duration, err error) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	if err != nil {
		p.metrics.ErrorCount++
		return
	}

	// Update transaction counters
	switch txType {
	case "new_order":
		p.metrics.NewOrderCount++
	case "payment":
		p.metrics.PaymentCount++
	case "order_status":
		p.metrics.OrderStatusCount++
	case "delivery":
		p.metrics.DeliveryCount++
	case "stock_level":
		p.metrics.StockLevelCount++
	}

	// Update latency statistics
	p.metrics.TotalLatency += latency
	if latency < p.metrics.MinLatency {
		p.metrics.MinLatency = latency
	}
	if latency > p.metrics.MaxLatency {
		p.metrics.MaxLatency = latency
	}
}

func (p *TPCCPlugin) resetMetrics() {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	p.metrics.NewOrderCount = 0
	p.metrics.PaymentCount = 0
	p.metrics.OrderStatusCount = 0
	p.metrics.DeliveryCount = 0
	p.metrics.StockLevelCount = 0
	p.metrics.ErrorCount = 0
	p.metrics.TimeoutCount = 0
	p.metrics.TotalLatency = 0
	p.metrics.MinLatency = time.Hour
	p.metrics.MaxLatency = 0
}

func (p *TPCCPlugin) getMetricID(txType string) int {
	// Map transaction types to metric IDs
	switch txType {
	case "new_order":
		return 1
	case "payment":
		return 2
	case "order_status":
		return 3
	case "delivery":
		return 4
	case "stock_level":
		return 5
	default:
		return 1
	}
}

func (p *TPCCPlugin) collectResults(ctx context.Context, runID int64, resultsChan <-chan *core.TestResult) {
	batch := make([]core.TestResult, 0, p.config.BatchSize)

	for result := range resultsChan {
		result.TestRunID = runID
		batch = append(batch, *result)

		if len(batch) >= p.config.BatchSize {
			if err := p.core.Storage.StoreResults(ctx, batch); err != nil {
				p.logger.Error("failed to store results batch", core.Field{Key: "error", Value: err.Error()})
			}
			batch = batch[:0]
		}
	}

	// Store remaining results
	if len(batch) > 0 {
		if err := p.core.Storage.StoreResults(ctx, batch); err != nil {
			p.logger.Error("failed to store final results batch", core.Field{Key: "error", Value: err.Error()})
		}
	}
}

func (p *TPCCPlugin) logConnectionLevelSummary(connCount int, duration time.Duration) {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	totalTxns := p.metrics.NewOrderCount + p.metrics.PaymentCount +
		p.metrics.OrderStatusCount + p.metrics.DeliveryCount + p.metrics.StockLevelCount

	tps := float64(totalTxns) / duration.Seconds()
	avgLatency := time.Duration(0)
	if totalTxns > 0 {
		avgLatency = p.metrics.TotalLatency / time.Duration(totalTxns)
	}

	p.logger.Info("connection level completed",
		core.Field{Key: "connections", Value: connCount},
		core.Field{Key: "duration", Value: duration},
		core.Field{Key: "total_transactions", Value: totalTxns},
		core.Field{Key: "tps", Value: tps},
		core.Field{Key: "avg_latency_ms", Value: avgLatency.Milliseconds()},
		core.Field{Key: "min_latency_ms", Value: p.metrics.MinLatency.Milliseconds()},
		core.Field{Key: "max_latency_ms", Value: p.metrics.MaxLatency.Milliseconds()},
		core.Field{Key: "errors", Value: p.metrics.ErrorCount},
		core.Field{Key: "new_order", Value: p.metrics.NewOrderCount},
		core.Field{Key: "payment", Value: p.metrics.PaymentCount},
		core.Field{Key: "order_status", Value: p.metrics.OrderStatusCount},
		core.Field{Key: "delivery", Value: p.metrics.DeliveryCount},
		core.Field{Key: "stock_level", Value: p.metrics.StockLevelCount},
	)
}

func getConfigSchema() string {
	return `{
		"type": "object",
		"properties": {
			"host": {"type": "string", "default": "localhost"},
			"port": {"type": "integer", "default": 5432},
			"database": {"type": "string", "default": "tpcc"},
			"username": {"type": "string", "default": "postgres"},
			"password": {"type": "string", "default": "postgres"},
			"ssl_mode": {"type": "string", "default": "disable"},
			"scale": {"type": "integer", "minimum": 1, "default": 10},
			"connections": {
				"type": "array",
				"items": {"type": "integer", "minimum": 1, "maximum": 1000},
				"default": [48, 96, 192, 256]
			},
			"duration": {"type": "string", "default": "5m"},
			"warmup_time": {"type": "string", "default": "30s"},
			"think_time": {"type": "string", "default": "100ms"},
			"new_order_pct": {"type": "integer", "minimum": 0, "maximum": 100, "default": 45},
			"payment_pct": {"type": "integer", "minimum": 0, "maximum": 100, "default": 43},
			"order_status_pct": {"type": "integer", "minimum": 0, "maximum": 100, "default": 4},
			"delivery_pct": {"type": "integer", "minimum": 0, "maximum": 100, "default": 4},
			"stock_level_pct": {"type": "integer", "minimum": 0, "maximum": 100, "default": 4},
			"batch_size": {"type": "integer", "minimum": 1, "default": 100},
			"enable_metrics": {"type": "boolean", "default": true},
			"log_transactions": {"type": "boolean", "default": false}
		},
		"required": ["host", "port", "database", "username", "password"]
	}`
}

// NewPlugin returns a new instance of the TPC-C plugin (required for plugin loading)
func NewPlugin() core.Plugin {
	return &TPCCPlugin{}
}

// Export the plugin (required for Go plugin system)
var Plugin TPCCPlugin
