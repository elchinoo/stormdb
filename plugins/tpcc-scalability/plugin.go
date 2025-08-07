// Package tpcc implements a comprehensive TPC-C scalability test plugin for StormDB v2
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/elchinoo/stormdb/core"
	"github.com/elchinoo/stormdb/plugins/tpcc-scalability/schema"
	"github.com/elchinoo/stormdb/plugins/tpcc-scalability/txn"
	_ "github.com/lib/pq"
)

// TPCCPlugin implements the TPC-C scalability test with enhanced metrics
type TPCCPlugin struct {
	core   *core.CoreServices
	logger core.Logger
	db     *sql.DB
	cfg    *TPCCConfig

	isRunning int64
	stopChan  chan struct{}
	stopOnce  sync.Once
	wg        sync.WaitGroup

	// Enhanced metrics system
	metrics *PerformanceMetrics

	// Legacy metrics for compatibility
	legacyMetrics *TPCCMetrics
	stats         *Stats

	// Transaction executor (can be overridden for tests)
	ExecTx func(db *sql.DB, txType string, warehouseID int, workerID int, metrics *PerformanceMetrics) error
}

// Duration is a wrapper around time.Duration that supports unmarshalling
// from JSON strings (e.g., "2m") or raw numbers (interpreted as seconds).
type Duration time.Duration

// UnmarshalJSON implements json.Unmarshaler.
func (d *Duration) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	// quoted string -> parse duration
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		dur, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		*d = Duration(dur)
		return nil
	}
	// numeric -> interpret as seconds
	var n float64
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*d = Duration(time.Duration(n) * time.Second)
	return nil
}

// MarshalJSON ensures Duration is always written as a string.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// String implements fmt.Stringer.
func (d Duration) String() string {
	return time.Duration(d).String()
}

// TPCCConfig holds configuration parameters for the test.
// TPCCConfig holds typed configuration parameters for the test.
// TPCCConfig holds configuration parameters for the test.
type TPCCConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSLMode  string `json:"ssl_mode"`

	Scale       int      `json:"scale"`
	Connections []int    `json:"connections"`
	Duration    Duration `json:"duration"`
	WarmupTime  Duration `json:"warmup_time"`
	ThinkTime   Duration `json:"think_time"`
	Mode        string   `json:"mode"`
	Rebuild     bool     `json:"rebuild"` // Added rebuild flag

	NewOrderPct       int `json:"new_order_pct"`
	PaymentPct        int `json:"payment_pct"`
	OrderStatusPct    int `json:"order_status_pct"`
	DeliveryPct       int `json:"delivery_pct"`
	StockLevelPct     int `json:"stock_level_pct"`
	CrossWarehousePct int `json:"cross_warehouse_pct"`

	EnableMetrics         bool     `json:"enable_metrics"`
	MetricsInterval       Duration `json:"metrics_interval"`
	StreamMetrics         bool     `json:"stream_metrics"`
	MaxErrorRate          float64  `json:"max_error_rate"`
	StopOnErrorLimit      bool     `json:"stop_on_error_limit"`
	EnableSupplierReorder bool     `json:"enable_supplier_reorder"`
	SupplierReorderPct    int      `json:"supplier_reorder_pct"`
	Verbose               bool     `json:"verbose"`
}

// NewTPCCConfig returns a TPCCConfig populated with default values.
func NewTPCCConfig() *TPCCConfig {
	return &TPCCConfig{
		WarmupTime:        Duration(10 * time.Second), // Default value for WarmupTime
		ThinkTime:         Duration(1 * time.Second),  // Default value for ThinkTime
		Port:              5432,
		SSLMode:           "disable",
		Mode:              "full",
		Scale:             1,
		Connections:       []int{1},
		NewOrderPct:       45,
		PaymentPct:        43,
		OrderStatusPct:    4,
		DeliveryPct:       4,
		StockLevelPct:     4,
		CrossWarehousePct: 0,
		EnableMetrics:     true,
		MetricsInterval:   Duration(time.Second),
		StreamMetrics:     true,
		MaxErrorRate:      0.05,
		StopOnErrorLimit:  true,
		Verbose:           true,
	}
}

// TPCCMetrics tracks runtime metrics and errors.
type TPCCMetrics struct {
	// mu                sync.RWMutex // removed unused
	TotalTransactions int64
	Errors            int64
	SecondlyMetrics   []SecondMetrics
}

// SecondMetrics captures metrics for each second.
type SecondMetrics struct {
	Timestamp time.Time
	Count     int64
	TPS       float64
	Errors    int64
}

// Stats holds aggregated test metrics collected from workers.
type Stats struct {
	DtStarted      time.Time
	DtEnded        time.Time
	NumConnections int
	NumInsert      int64
	NumUpdate      int64
	NumDelete      int64
	NumSelect      int64
	LatencySum     int64 // nanoseconds
	LatencyCount   int64
	NumRowInsert   int64
	NumRowUpdate   int64
	NumRowDelete   int64
	NumRowSelect   int64
}

// Start records the start time and number of connections.
func (s *Stats) Start(conns int) {
	s.DtStarted = time.Now()
	s.NumConnections = conns
}

// End records the end time.
func (s *Stats) End() {
	s.DtEnded = time.Now()
}

// Record updates metrics for a single operation.
func (s *Stats) Record(op string, latency time.Duration, rowsAffected int) {
	// accumulate latency and count atomically
	atomic.AddInt64(&s.LatencySum, latency.Nanoseconds())
	atomic.AddInt64(&s.LatencyCount, 1)
	switch op {
	case "insert":
		atomic.AddInt64(&s.NumInsert, 1)
		atomic.AddInt64(&s.NumRowInsert, int64(rowsAffected))
	case "update":
		atomic.AddInt64(&s.NumUpdate, 1)
		atomic.AddInt64(&s.NumRowUpdate, int64(rowsAffected))
	case "delete":
		atomic.AddInt64(&s.NumDelete, 1)
		atomic.AddInt64(&s.NumRowDelete, int64(rowsAffected))
	case "select":
		atomic.AddInt64(&s.NumSelect, 1)
		atomic.AddInt64(&s.NumRowSelect, int64(rowsAffected))
	}
}

// SnapshotAndReset returns a snapshot of current metrics and resets counters.
func (s *Stats) SnapshotAndReset() *Stats {
	snap := Stats{
		DtStarted:      s.DtStarted,
		DtEnded:        time.Now(),
		NumConnections: s.NumConnections,
		NumInsert:      atomic.SwapInt64(&s.NumInsert, 0),
		NumUpdate:      atomic.SwapInt64(&s.NumUpdate, 0),
		NumDelete:      atomic.SwapInt64(&s.NumDelete, 0),
		NumSelect:      atomic.SwapInt64(&s.NumSelect, 0),
		LatencySum:     atomic.SwapInt64(&s.LatencySum, 0),
		LatencyCount:   atomic.SwapInt64(&s.LatencyCount, 0),
		NumRowInsert:   atomic.SwapInt64(&s.NumRowInsert, 0),
		NumRowUpdate:   atomic.SwapInt64(&s.NumRowUpdate, 0),
		NumRowDelete:   atomic.SwapInt64(&s.NumRowDelete, 0),
		NumRowSelect:   atomic.SwapInt64(&s.NumRowSelect, 0),
	}
	// reset end time for next interval
	s.DtEnded = time.Time{}
	return &snap
}

// NewPlugin constructs and returns a new TPCCPlugin.
func NewPlugin() core.Plugin {
	return &TPCCPlugin{}
}

// Plugin is the exported plugin instance.
var Plugin TPCCPlugin

// Metadata returns plugin metadata for registration.
func (p *TPCCPlugin) Metadata() core.PluginMetadata {
	// Load the config schema
	schemaPath := filepath.Join(filepath.Dir(os.Args[0]), "plugins", "tpcc-scalability", "config-schema.json")
	schemaBytes, err := os.ReadFile(schemaPath)
	var configSchema string
	if err != nil {
		// Fallback to empty schema if file not found
		configSchema = ""
	} else {
		configSchema = string(schemaBytes)
	}

	return core.PluginMetadata{
		Name:         "tpcc-scalability",
		Version:      "2.0.0",
		Description:  "TPC-C scalability test plugin with enhanced metrics and error handling",
		Author:       "StormDB Team",
		License:      "MIT",
		TestTypes:    []string{"tpcc", "scalability", "performance"},
		ConfigSchema: configSchema,
		Dependencies: map[string]string{"postgresql": ">=12.0"},
	}
}

// Initialize sets up core services and metrics.
func (p *TPCCPlugin) Initialize(ctx context.Context, cs *core.CoreServices) error {
	p.core = cs
	p.logger = cs.Logger.WithPlugin(p.Metadata().Name)
	p.stopChan = make(chan struct{})

	// Initialize enhanced metrics system
	p.metrics = NewPerformanceMetrics(p.db)

	// Initialize legacy metrics for compatibility
	p.legacyMetrics = &TPCCMetrics{SecondlyMetrics: make([]SecondMetrics, 0)}
	p.stats = &Stats{}

	// Set default transaction executor
	p.ExecTx = p.executeTransactionWithMetrics
	return nil
}

// Validate parses and validates configuration data.
func (p *TPCCPlugin) Validate(cfgData map[string]interface{}) error {
	// Decode JSON config into typed struct, using custom Duration type
	cfg := NewTPCCConfig()
	raw, err := json.Marshal(cfgData)
	if err != nil {
		return fmt.Errorf("failed to marshal config data: %w", err)
	}
	if err := json.Unmarshal(raw, cfg); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}
	// Validate required fields
	if cfg.Host == "" {
		return fmt.Errorf("host is required")
	}
	if cfg.Database == "" {
		return fmt.Errorf("database is required")
	}
	if cfg.Scale <= 0 {
		return fmt.Errorf("scale must be positive")
	}
	if len(cfg.Connections) == 0 {
		return fmt.Errorf("at least one connection level is required")
	}
	if cfg.Duration <= 0 {
		return fmt.Errorf("duration is required")
	}
	// Validate mode
	validModes := map[string]bool{"setup": true, "run": true, "rebuild": true, "full": true}
	if !validModes[cfg.Mode] {
		return fmt.Errorf("invalid mode: %s", cfg.Mode)
	}
	// Validate transaction percentage sum <= 100
	sum := cfg.NewOrderPct + cfg.PaymentPct + cfg.OrderStatusPct + cfg.DeliveryPct + cfg.StockLevelPct + cfg.CrossWarehousePct
	if cfg.EnableSupplierReorder {
		sum += cfg.SupplierReorderPct
	}
	if sum > 100 {
		return fmt.Errorf("percentage weights sum to %d, must be <= 100", sum)
	}
	if sum == 0 {
		return fmt.Errorf("at least one transaction type percentage must be > 0")
	}
	// Validate error rate
	if cfg.MaxErrorRate < 0 || cfg.MaxErrorRate > 1 {
		return fmt.Errorf("max_error_rate must be between 0 and 1, got %f", cfg.MaxErrorRate)
	}
	// Validate connection counts
	for i, conn := range cfg.Connections {
		if conn <= 0 {
			return fmt.Errorf("connections[%d] must be positive, got %d", i, conn)
		}
		if conn > 1000 {
			return fmt.Errorf("connections[%d] too high (%d), maximum 1000 for safety", i, conn)
		}
	}
	p.cfg = cfg
	return nil
}

// Execute runs the plugin logic based on the configured mode.
func (p *TPCCPlugin) Execute(ctx context.Context, cfgData map[string]interface{}) error {
	// mark running and defer clear
	atomic.StoreInt64(&p.isRunning, 1)
	defer atomic.StoreInt64(&p.isRunning, 0)
	// Parse and validate config
	if err := p.Validate(cfgData); err != nil {
		p.logger.Error("Configuration validation failed",
			core.Field{Key: "error_message", Value: err.Error()},
			core.Field{Key: "location", Value: "Execute.Validate"},
			core.Field{Key: "config_data", Value: fmt.Sprintf("%+v", cfgData)})
		return fmt.Errorf("validation error: %w", err)
	}

	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		p.cfg.Host, p.cfg.Port, p.cfg.Database, p.cfg.Username, p.cfg.Password, p.cfg.SSLMode)
	var err error
	p.db, err = sql.Open("postgres", dsn)
	if err != nil {
		p.logger.Error("Failed to open database connection",
			core.Field{Key: "error_message", Value: err.Error()},
			core.Field{Key: "location", Value: "Execute.sql.Open"},
			core.Field{Key: "host", Value: p.cfg.Host},
			core.Field{Key: "port", Value: p.cfg.Port},
			core.Field{Key: "database", Value: p.cfg.Database},
			core.Field{Key: "username", Value: p.cfg.Username},
			core.Field{Key: "ssl_mode", Value: p.cfg.SSLMode})
		return fmt.Errorf("failed to open db: %w", err)
	}

	// Configure connection pool based on maximum expected connections
	maxConnections := 0
	for _, conn := range p.cfg.Connections {
		if conn > maxConnections {
			maxConnections = conn
		}
	}
	// Set pool size to accommodate all workers plus some overhead
	poolSize := maxConnections + 10
	p.db.SetMaxOpenConns(poolSize)
	p.db.SetMaxIdleConns(poolSize / 2)
	p.db.SetConnMaxLifetime(30 * time.Minute)
	p.db.SetConnMaxIdleTime(5 * time.Minute)

	p.logger.Info("Database connection pool configured",
		core.Field{Key: "max_open_connections", Value: poolSize},
		core.Field{Key: "max_idle_connections", Value: poolSize / 2},
		core.Field{Key: "max_worker_connections", Value: maxConnections})

	if err := p.db.PingContext(ctx); err != nil {
		p.logger.Error("Database ping failed",
			core.Field{Key: "error_message", Value: err.Error()},
			core.Field{Key: "location", Value: "Execute.db.PingContext"},
			core.Field{Key: "host", Value: p.cfg.Host},
			core.Field{Key: "port", Value: p.cfg.Port},
			core.Field{Key: "database", Value: p.cfg.Database})
		return fmt.Errorf("failed to ping db: %w", err)
	}
	// Wrap context with cancel to allow signal handler to stop execution
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Ensure cleanup of resources when execution finishes
	defer func() {
		_ = p.Cleanup(ctx)
		p.logger.Info("Cleanup complete")
	}()

	// Handle rebuild flag - if rebuild is true, force rebuild mode regardless of mode setting
	mode := p.cfg.Mode
	if p.cfg.Rebuild {
		mode = "rebuild"
		p.logger.Info("Rebuild flag set, forcing rebuild mode")
	}

	// Dispatch based on effective mode
	switch mode {
	case "setup":
		return p.executeSetup(ctx)
	case "run":
		return p.executeRunOnly(ctx)
	case "rebuild":
		return p.executeRebuild(ctx)
	case "full":
		// trap SIGINT/SIGTERM for graceful shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			sig := <-sigCh
			p.logger.Warn("signal received, shutting down", core.Field{Key: "signal", Value: sig})
			// signal to stop workload and cancel context
			_ = p.Stop()
			cancel()
		}()
		// Execute full workload
		err := p.executeFull(ctx)
		// stop listening to signals
		signal.Stop(sigCh)
		return err
	default:
		return fmt.Errorf("unsupported mode: %s", p.cfg.Mode)
	}
}

// setupSchema creates or updates database schema
func (p *TPCCPlugin) setupSchema(ctx context.Context) error {
	p.logger.Info("Migrating SQL schema for TPC-C")
	// run migrations: prefer plugin-specific files if present, else fallback to root migrations
	pluginDir := "plugins/tpcc-scalability/migrations"
	pluginPattern := filepath.Join(pluginDir, "*.up.sql")
	pluginFiles, err := filepath.Glob(pluginPattern)
	if err != nil {
		p.logger.Error("Failed to glob plugin migration files",
			core.Field{Key: "error_message", Value: err.Error()},
			core.Field{Key: "location", Value: "setupSchema.Glob"},
			core.Field{Key: "pattern", Value: pluginPattern})
	}

	if len(pluginFiles) > 0 {
		p.logger.Info("Applying migrations from plugin", core.Field{Key: "dir", Value: pluginDir})
		if err := schema.Migrate(ctx, p.db, pluginDir); err != nil {
			p.logger.Error("Plugin schema migration failed",
				core.Field{Key: "error_message", Value: err.Error()},
				core.Field{Key: "location", Value: "setupSchema.schema.Migrate"},
				core.Field{Key: "migration_dir", Value: pluginDir},
				core.Field{Key: "files_found", Value: len(pluginFiles)})
			return fmt.Errorf("schema migration failed (plugin): %w", err)
		}
		return nil
	}

	// fallback to root migrations directory
	localDir := "migrations"
	localPattern := filepath.Join(localDir, "*.up.sql")
	localFiles, err := filepath.Glob(localPattern)
	if err != nil {
		p.logger.Error("Failed to glob root migration files",
			core.Field{Key: "error_message", Value: err.Error()},
			core.Field{Key: "location", Value: "setupSchema.Glob"},
			core.Field{Key: "pattern", Value: localPattern})
	}

	if len(localFiles) > 0 {
		p.logger.Info("Applying migrations from root", core.Field{Key: "dir", Value: localDir})
		if err := schema.Migrate(ctx, p.db, localDir); err != nil {
			p.logger.Error("Root schema migration failed",
				core.Field{Key: "error_message", Value: err.Error()},
				core.Field{Key: "location", Value: "setupSchema.schema.Migrate"},
				core.Field{Key: "migration_dir", Value: localDir},
				core.Field{Key: "files_found", Value: len(localFiles)})
			return fmt.Errorf("schema migration failed (root): %w", err)
		}
		return nil
	}

	p.logger.Error("No migration files found",
		core.Field{Key: "error_message", Value: "no SQL migration files available"},
		core.Field{Key: "location", Value: "setupSchema.noFiles"},
		core.Field{Key: "plugin_dir", Value: pluginDir},
		core.Field{Key: "root_dir", Value: localDir})
	return fmt.Errorf("no migration files found in %s or %s", pluginDir, localDir)
}

// populateData inserts seed and TPC-C workload data
func (p *TPCCPlugin) populateData(ctx context.Context) error {
	p.logger.Info("Populating data for TPC-C")
	if err := p.populateSeedData(ctx); err != nil {
		p.logger.Error("Seed data population failed",
			core.Field{Key: "error_message", Value: err.Error()},
			core.Field{Key: "location", Value: "populateData.populateSeedData"},
			core.Field{Key: "scale", Value: p.cfg.Scale})
		return fmt.Errorf("populateSeedData failed: %w", err)
	}
	if err := p.populateTPCCData(ctx); err != nil {
		p.logger.Error("TPC-C data population failed",
			core.Field{Key: "error_message", Value: err.Error()},
			core.Field{Key: "location", Value: "populateData.populateTPCCData"},
			core.Field{Key: "scale", Value: p.cfg.Scale})
		return fmt.Errorf("populateTPCCData failed: %w", err)
	}
	return nil
}

// executeSetup performs only schema setup
func (p *TPCCPlugin) executeSetup(ctx context.Context) error {
	return p.setupSchema(ctx)
}

// executeRunOnly runs tests without schema changes
func (p *TPCCPlugin) executeRunOnly(ctx context.Context) error {
	return p.runTestWorkload(ctx)
}

// executeRebuild rebuilds schema and data, then runs tests
func (p *TPCCPlugin) executeRebuild(ctx context.Context) error {
	if err := p.dropTables(ctx); err != nil {
		p.logger.Error("Table drop failed during rebuild",
			core.Field{Key: "error_message", Value: err.Error()},
			core.Field{Key: "location", Value: "executeRebuild.dropTables"})
		// Continue despite drop failure - tables might not exist
	}
	if err := p.setupSchema(ctx); err != nil {
		return err
	}
	if err := p.populateData(ctx); err != nil {
		return err
	}
	return p.runTestWorkload(ctx)
}

// executeFull does setup if needed and then runs tests
func (p *TPCCPlugin) executeFull(ctx context.Context) error {
	if err := p.setupSchema(ctx); err != nil {
		return err
	}
	// check data presence not implemented
	if err := p.populateData(ctx); err != nil {
		return err
	}
	return p.runTestWorkload(ctx)
}

// populateSeedData creates initial TPC-C seed tables: warehouses and districts
func (p *TPCCPlugin) populateSeedData(ctx context.Context) error {
	p.logger.Info("Populating seed data: warehouses and districts")
	for wID := 1; wID <= p.cfg.Scale; wID++ {
		// insert warehouse with tax default
		_, err := p.db.ExecContext(ctx,
			`INSERT INTO warehouse(w_id, w_name, w_street_1, w_city, w_state, w_zip, w_ytd, w_tax)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (w_id) DO NOTHING`,
			wID,
			fmt.Sprintf("Warehouse-%d", wID),
			"Street1", "City", "ST", "00000", 3000000,
			0.0,
		)
		if err != nil {
			p.logger.Error("Failed to insert warehouse data",
				core.Field{Key: "error_message", Value: err.Error()},
				core.Field{Key: "location", Value: "populateSeedData.warehouse"},
				core.Field{Key: "warehouse_id", Value: wID},
				core.Field{Key: "scale", Value: p.cfg.Scale})
			return fmt.Errorf("populateSeedData warehouse %d: %w", wID, err)
		}
		// insert 10 districts per warehouse
		for dID := 1; dID <= 10; dID++ {
			// insert district, explicitly providing next_o_id and tax
			_, err := p.db.ExecContext(ctx,
				`INSERT INTO district(
					d_id, d_w_id, d_name, d_street_1, d_city, d_state, d_zip,
					d_next_o_id, d_ytd, d_tax
				) VALUES(
					$1, $2, $3, $4, $5, $6, $7,
					$8, $9, $10
				) ON CONFLICT (d_w_id, d_id) DO NOTHING`,
				dID, wID,
				fmt.Sprintf("District-%d", dID),
				"Street1", "City", "ST", "00000",
				3001,
				30000,
				0.0,
			)
			if err != nil {
				p.logger.Error("Failed to insert district data",
					core.Field{Key: "error_message", Value: err.Error()},
					core.Field{Key: "location", Value: "populateSeedData.district"},
					core.Field{Key: "warehouse_id", Value: wID},
					core.Field{Key: "district_id", Value: dID})
				return fmt.Errorf("populateSeedData district %d.%d: %w", wID, dID, err)
			}
		}
	}
	return nil
}

// populateTPCCData inserts TPC-C dynamic data (customers, items, stock) into tables
func (p *TPCCPlugin) populateTPCCData(ctx context.Context) error {
	p.logger.Info("Populating TPC-C dynamic data: items, stock, customers")
	const (
		NumItems              = 100000
		DistrictsPerWarehouse = 10
		CustomersPerDistrict  = 3000
	)
	// seed items
	for i := 1; i <= NumItems; i++ {
		if _, err := p.db.ExecContext(ctx,
			`INSERT INTO item(i_id, i_name, i_price, i_data) VALUES($1, $2, $3, $4)
			  ON CONFLICT (i_id) DO NOTHING`,
			i,
			fmt.Sprintf("Item-%d", i),
			float64(rand.Intn(10000))/100.0,
			"data",
		); err != nil {
			p.logger.Error("Failed to insert item data",
				core.Field{Key: "error_message", Value: err.Error()},
				core.Field{Key: "location", Value: "populateTPCCData.item"},
				core.Field{Key: "item_id", Value: i},
				core.Field{Key: "total_items", Value: NumItems})
			return fmt.Errorf("populateTPCCData item %d: %w", i, err)
		}
	}
	// seed stock for each warehouse and item
	for w := 1; w <= p.cfg.Scale; w++ {
		for i := 1; i <= NumItems; i++ {
			// insert stock with all required TPC-C columns
			if _, err := p.db.ExecContext(ctx,
				`INSERT INTO stock(
					s_w_id, s_i_id, s_quantity,
					s_dist_01, s_dist_02, s_dist_03,
					s_dist_04, s_dist_05, s_dist_06,
					s_dist_07, s_dist_08, s_dist_09,
					s_dist_10, s_order_point, s_ytd,
					s_order_cnt, s_remote_cnt, s_reorder_qty,
					s_data
				) VALUES(
					$1, $2, $3,
					$4, $5, $6,
					$7, $8, $9,
					$10, $11, $12,
					$13, $14, $15,
					$16, $17, $18,
					$19
				) ON CONFLICT (s_w_id, s_i_id) DO NOTHING`,
				w, i, 100, // s_w_id, s_i_id, s_quantity
				"dist", "dist", "dist", // s_dist_01, s_dist_02, s_dist_03
				"dist", "dist", "dist", // s_dist_04, s_dist_05, s_dist_06
				"dist", "dist", "dist", // s_dist_07, s_dist_08, s_dist_09
				"dist", 0, 0, // s_dist_10, s_order_point, s_ytd
				0, 0, 0, // s_order_cnt, s_remote_cnt, s_reorder_qty
				"data", // s_data
			); err != nil {
				p.logger.Error("Failed to insert stock data",
					core.Field{Key: "error_message", Value: err.Error()},
					core.Field{Key: "location", Value: "populateTPCCData.stock"},
					core.Field{Key: "warehouse_id", Value: w},
					core.Field{Key: "item_id", Value: i},
					core.Field{Key: "scale", Value: p.cfg.Scale})
				return fmt.Errorf("populateTPCCData stock %d.%d: %w", w, i, err)
			}
		}
	}
	// seed customers
	for w := 1; w <= p.cfg.Scale; w++ {
		for d := 1; d <= DistrictsPerWarehouse; d++ {
			for c := 1; c <= CustomersPerDistrict; c++ {
				if _, err := p.db.ExecContext(ctx,
					`INSERT INTO customer(
						c_w_id, c_d_id, c_id, c_balance,
						c_first, c_middle, c_last,
						c_street_1, c_street_2, c_city, c_state, c_zip,
						c_phone, c_since, c_credit, c_credit_lim,
						c_discount, c_data, c_ytd_payment,
						c_payment_cnt, c_delivery_cnt
					) VALUES(
						$1, $2, $3, $4,
						$5, $6, $7,
						$8, $9, $10, $11, $12,
						$13, $14, $15, $16,
						$17, $18, $19,
						$20, $21
					) ON CONFLICT (c_w_id, c_d_id, c_id) DO NOTHING`,
					w, d, c, 0.0, // c_w_id, c_d_id, c_id, c_balance
					fmt.Sprintf("First%d", c), "OE", fmt.Sprintf("Last%d", c), // c_first, c_middle, c_last
					"Street1", "Street2", "City", "ST", "00000", // c_street_1, c_street_2, c_city, c_state, c_zip
					"1234567890123456", time.Now(), "GC", 50000.00, // c_phone, c_since, c_credit, c_credit_lim
					0.0, "customer data", 0.0, // c_discount, c_data, c_ytd_payment
					0, 0, // c_payment_cnt, c_delivery_cnt
				); err != nil {
					p.logger.Error("Failed to populate customer data",
						core.Field{Key: "warehouse_id", Value: w},
						core.Field{Key: "district_id", Value: d},
						core.Field{Key: "customer_id", Value: c},
						core.Field{Key: "scale", Value: p.cfg.Scale},
						core.Field{Key: "error_message", Value: err.Error()},
						core.Field{Key: "location", Value: "populateTPCCData.customers"})
					return fmt.Errorf("populateTPCCData customer %d.%d.%d: %w", w, d, c, err)
				}
			}
		}
	}
	return nil
}

// runTestWorkload runs the TPC-C workload per connection levels with enhanced metrics
func (p *TPCCPlugin) runTestWorkload(ctx context.Context) error {
	levels := p.cfg.Connections
	if len(levels) == 0 {
		return fmt.Errorf("no connection levels defined")
	}

	perLevel := time.Duration(p.cfg.Duration) / time.Duration(len(levels))
	p.logger.Info("Test plan",
		core.Field{Key: "total_duration", Value: p.cfg.Duration},
		core.Field{Key: "levels", Value: len(levels)},
		core.Field{Key: "duration_per_level", Value: perLevel})

	for idx, conns := range levels {
		// Reset metrics for this level
		p.metrics.Reset()

		p.logger.Info("Starting level",
			core.Field{Key: "level", Value: idx + 1},
			core.Field{Key: "connections", Value: conns})

		// Start metrics collection for this level
		p.metrics.Start(conns)

		// Warmup phase
		if p.cfg.WarmupTime > 0 {
			p.logger.Info("Warmup phase",
				core.Field{Key: "duration", Value: p.cfg.WarmupTime})
			warmCtx, cancel := context.WithTimeout(ctx, time.Duration(p.cfg.WarmupTime))
			p.runWorkload(warmCtx, conns, time.Duration(p.cfg.ThinkTime))
			cancel()

			// Reset metrics after warmup
			p.metrics.Reset()
			p.metrics.Start(conns)
		}

		// Measurement phase
		p.logger.Info("Measurement phase",
			core.Field{Key: "connections", Value: conns},
			core.Field{Key: "duration", Value: perLevel})

		measureCtx, cancel := context.WithTimeout(ctx, perLevel)

		// Start real-time reporting ticker
		ticker := time.NewTicker(time.Duration(p.cfg.MetricsInterval))
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-measureCtx.Done():
					return
				case <-p.stopChan:
					return
				case <-ticker.C:
					// Get interval snapshot (resets counters)
					snap := p.metrics.SnapshotAndReset()

					// Calculate metrics for this interval
					ops := snap.TotalOperations()
					tps := snap.OperationsPerSecond()
					avgLatency := snap.AverageLatencyMS()
					errorRate := snap.ErrorRate()

					// Store in legacy format for compatibility
					p.legacyMetrics.SecondlyMetrics = append(p.legacyMetrics.SecondlyMetrics, SecondMetrics{
						Timestamp: time.Now(),
						Count:     ops,
						TPS:       tps,
						Errors:    snap.NumErrors,
					})

					// Console logging (once per second)
					if p.cfg.Verbose {
						p.logger.Info("Interval stats",
							core.Field{Key: "connections", Value: snap.NumConnections},
							core.Field{Key: "ops_total", Value: ops},
							core.Field{Key: "tps", Value: tps},
							core.Field{Key: "avg_latency_ms", Value: avgLatency},
							core.Field{Key: "errors", Value: snap.NumErrors},
							core.Field{Key: "error_rate", Value: errorRate},
							core.Field{Key: "inserts", Value: snap.NumInsert},
							core.Field{Key: "updates", Value: snap.NumUpdate},
							core.Field{Key: "deletes", Value: snap.NumDelete},
							core.Field{Key: "selects", Value: snap.NumSelect},
						)
					}

					// Persist metrics to storage (once per second)
					if p.cfg.StreamMetrics && ops > 0 {
						results := []core.TestResult{{
							StartTime:         snap.DtStarted,
							EndTime:           snap.DtEnd,
							Value:             float64(ops),
							ActiveConnections: &snap.NumConnections,
							Tags: map[string]interface{}{
								"dt_started":      snap.DtStarted.UnixNano(),
								"dt_end":          snap.DtEnd.UnixNano(),
								"num_connections": snap.NumConnections,
								"num_insert":      snap.NumInsert,
								"num_update":      snap.NumUpdate,
								"num_delete":      snap.NumDelete,
								"num_select":      snap.NumSelect,
								"latency_sum":     snap.LatencySum,
								"latency_count":   snap.LatencyCount,
								"num_row_insert":  snap.NumRowInsert,
								"num_row_update":  snap.NumRowUpdate,
								"num_row_delete":  snap.NumRowDelete,
								"num_row_select":  snap.NumRowSelect,
								"avg_latency_ms":  avgLatency,
								"error_rate":      errorRate,
								"tps":             tps,
							},
						}}
						_ = p.core.Storage.StoreResults(ctx, results)
					}
				}
			}
		}()

		// Run the actual workload
		p.runWorkload(measureCtx, conns, time.Duration(p.cfg.ThinkTime))
		cancel()

		// End metrics collection for this level
		p.metrics.End()

		// Check error rate limit
		if p.cfg.StopOnErrorLimit {
			finalSnap := p.metrics.Snapshot()
			if finalSnap.TotalOperations() > 0 {
				errorRate := finalSnap.ErrorRate()
				if errorRate > p.cfg.MaxErrorRate*100 { // MaxErrorRate is 0-1, ErrorRate() returns 0-100
					p.logger.Warn("Error rate limit exceeded, stopping workload",
						core.Field{Key: "error_rate", Value: errorRate},
						core.Field{Key: "limit", Value: p.cfg.MaxErrorRate * 100})
					return nil
				}
			}
		}
	}

	// Final summary after all levels
	finalSnap := p.metrics.Snapshot()
	if finalSnap.TotalOperations() > 0 {
		avgLatency := finalSnap.AverageLatencyMS()
		finalResults := []core.TestResult{{
			StartTime:         finalSnap.DtStarted,
			EndTime:           finalSnap.DtEnd,
			Value:             float64(finalSnap.TotalOperations()),
			ActiveConnections: &finalSnap.NumConnections,
			Tags: map[string]interface{}{
				"dt_started":      finalSnap.DtStarted.UnixNano(),
				"dt_end":          finalSnap.DtEnd.UnixNano(),
				"num_connections": finalSnap.NumConnections,
				"num_insert":      finalSnap.NumInsert,
				"num_update":      finalSnap.NumUpdate,
				"num_delete":      finalSnap.NumDelete,
				"num_select":      finalSnap.NumSelect,
				"latency_sum":     finalSnap.LatencySum,
				"latency_count":   finalSnap.LatencyCount,
				"num_row_insert":  finalSnap.NumRowInsert,
				"num_row_update":  finalSnap.NumRowUpdate,
				"num_row_delete":  finalSnap.NumRowDelete,
				"num_row_select":  finalSnap.NumRowSelect,
				"avg_latency_ms":  avgLatency,
				"error_rate":      finalSnap.ErrorRate(),
				"summary":         "final",
			},
		}}

		if p.cfg.StreamMetrics {
			_ = p.core.Storage.StoreResults(ctx, finalResults)
		}
	}

	return nil
}

// runWorkload starts workers that execute transactions with enhanced metrics tracking
func (p *TPCCPlugin) runWorkload(ctx context.Context, connections int, thinkTime time.Duration) {
	p.wg.Add(connections)

	for i := 0; i < connections; i++ {
		go func(workerID int) {
			defer p.wg.Done()

			// Register this worker with the metrics system
			metricsWorkerID := p.metrics.RegisterWorker()
			defer p.metrics.UnregisterWorker(metricsWorkerID)

			// Local counters for tracking transaction totals
			var totalTransactions, totalErrors int64

			for {
				select {
				case <-ctx.Done():
					return
				case <-p.stopChan:
					return
				default:
				}

				// Choose and execute transaction
				op := chooseTransactionType(p.cfg)
				// Distribute warehouses across workers to reduce contention on the same warehouse
				warehouse := (workerID % p.cfg.Scale) + 1

				// Occasionally use random warehouse for cross-warehouse transactions
				if rand.Float64() < 0.1 {
					warehouse = rand.Intn(p.cfg.Scale) + 1
				}

				// Execute transaction with enhanced metrics
				var err error

				if p.ExecTx != nil {
					err = p.ExecTx(p.db, op, warehouse, metricsWorkerID, p.metrics)
				} else {
					err = p.executeTransactionWithMetrics(p.db, op, warehouse, metricsWorkerID, p.metrics)
				}

				isError := err != nil

				// Enhanced error logging with context
				if isError {
					p.logger.Error("Transaction execution failed",
						core.Field{Key: "worker_id", Value: workerID},
						core.Field{Key: "metrics_worker_id", Value: metricsWorkerID},
						core.Field{Key: "transaction_type", Value: op},
						core.Field{Key: "warehouse_id", Value: warehouse},
						core.Field{Key: "error_message", Value: err.Error()},
						core.Field{Key: "location", Value: "runWorkload.executeTransaction"},
						core.Field{Key: "total_transactions", Value: totalTransactions + 1},
						core.Field{Key: "total_errors", Value: totalErrors + 1})
				}

				// Update local counters
				totalTransactions++
				if isError {
					totalErrors++
				}

				// Note: Individual database operations are recorded within transaction implementations
				// No need to record transaction-level metrics here as we want raw operation data

				// Check error rate limit per operation (use local counters for immediate response)
				// Only check after a minimum number of transactions to avoid false positives
				minTransactionsForErrorCheck := int64(10)
				if p.cfg.StopOnErrorLimit && totalTransactions >= minTransactionsForErrorCheck {
					localErrorRate := float64(totalErrors) / float64(totalTransactions)
					if localErrorRate > p.cfg.MaxErrorRate {
						// Signal stop to all workers
						p.stopOnce.Do(func() {
							close(p.stopChan)
							p.logger.Warn("Error rate limit exceeded in worker",
								core.Field{Key: "worker_id", Value: workerID},
								core.Field{Key: "error_rate", Value: localErrorRate},
								core.Field{Key: "limit", Value: p.cfg.MaxErrorRate},
								core.Field{Key: "total_transactions", Value: totalTransactions},
								core.Field{Key: "total_errors", Value: totalErrors})
						})
						return
					}
				}

				// Think time between transactions
				if thinkTime > 0 {
					time.Sleep(thinkTime)
				}
			}
		}(i)
	}

	p.wg.Wait()
}

// dropTables drops existing TPC-C tables for a clean rebuild
func (p *TPCCPlugin) dropTables(ctx context.Context) error {
	p.logger.Info("Dropping existing TPC-C tables")
	stmts := []string{
		"DROP TABLE IF EXISTS history CASCADE",
		"DROP TABLE IF EXISTS order_line CASCADE",
		"DROP TABLE IF EXISTS new_order CASCADE",
		"DROP TABLE IF EXISTS orders CASCADE",
		"DROP TABLE IF EXISTS stock CASCADE",
		"DROP TABLE IF EXISTS customer CASCADE",
		"DROP TABLE IF EXISTS district CASCADE",
		"DROP TABLE IF EXISTS warehouse CASCADE",
	}
	for _, sql := range stmts {
		if _, err := p.db.ExecContext(ctx, sql); err != nil {
			return fmt.Errorf("dropTables failed on %s: %w", sql, err)
		}
	}
	return nil
}

// Stop signals the plugin to terminate running tests.
func (p *TPCCPlugin) Stop() error {
	// close stopChan once to signal all workers
	p.stopOnce.Do(func() { close(p.stopChan) })
	p.wg.Wait()

	// Stop the metrics system gracefully
	if p.metrics != nil {
		p.metrics.Stop()
	}

	return nil
}

// Cleanup releases resources and closes database connection.
func (p *TPCCPlugin) Cleanup(ctx context.Context) error {
	// ensure workers are stopped
	p.stopOnce.Do(func() { close(p.stopChan) })
	p.wg.Wait()

	// Stop the metrics system gracefully
	if p.metrics != nil {
		p.metrics.Stop()
	}

	// close database connection
	if p.db != nil {
		_ = p.db.Close()
	}
	return nil
}

// Health performs a comprehensive health check of the plugin
func (p *TPCCPlugin) Health(ctx context.Context) core.PluginHealth {
	health := core.PluginHealth{
		Status:       core.PluginStatusHealthy,
		LastCheck:    time.Now(),
		Metrics:      make(map[string]interface{}),
		Dependencies: []core.DependencyHealth{},
	}

	// Check database connection
	if p.db == nil {
		health.Status = core.PluginStatusFailed
		health.Message = "Database connection not initialized"
		health.Dependencies = append(health.Dependencies, core.DependencyHealth{
			Name:    "database",
			Type:    "database",
			Status:  core.PluginStatusFailed,
			Message: "Connection not established",
		})
		return health
	}

	// Test database connectivity
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := p.db.PingContext(dbCtx); err != nil {
		health.Status = core.PluginStatusDegraded
		health.Message = "Database connection issues detected"
		health.Dependencies = append(health.Dependencies, core.DependencyHealth{
			Name:    "database",
			Type:    "database",
			Status:  core.PluginStatusFailed,
			Message: fmt.Sprintf("Ping failed: %v", err),
		})
	} else {
		health.Dependencies = append(health.Dependencies, core.DependencyHealth{
			Name:   "database",
			Type:   "database",
			Status: core.PluginStatusHealthy,
		})
	}

	// Check metrics system health
	if p.metrics != nil {
		snapshot := p.metrics.Snapshot()
		health.Metrics["metrics_system"] = "active"
		health.Metrics["total_operations"] = snapshot.TotalOperations
		health.Metrics["total_errors"] = snapshot.NumErrors
		health.Metrics["connections"] = snapshot.NumConnections
		health.Metrics["latency_avg_ms"] = float64(snapshot.LatencySum) / float64(snapshot.LatencyCount) / 1e6
	} else {
		health.Metrics["metrics_system"] = "not_initialized"
		if health.Status == core.PluginStatusHealthy {
			health.Status = core.PluginStatusDegraded
			health.Message = "Metrics system not initialized"
		}
	}

	// Check configuration
	if p.cfg == nil {
		if health.Status == core.PluginStatusHealthy {
			health.Status = core.PluginStatusDegraded
			health.Message = "Plugin not configured"
		}
		health.Metrics["configured"] = false
	} else {
		health.Metrics["configured"] = true
		health.Metrics["scale"] = p.cfg.Scale
		health.Metrics["max_connections"] = p.cfg.Connections
	}

	// Add runtime metrics
	health.Metrics["goroutines"] = runtime.NumGoroutine()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	health.Metrics["memory_usage_mb"] = float64(m.Alloc) / 1024 / 1024
	health.Metrics["gc_runs"] = m.NumGC

	return health
}

// GetResourceUsage implements PluginResourceMonitor interface
func (p *TPCCPlugin) GetResourceUsage() map[string]interface{} {
	usage := make(map[string]interface{})

	// Basic plugin state
	usage["configured"] = p.cfg != nil
	usage["database_connected"] = p.db != nil

	if p.cfg != nil {
		usage["scale"] = p.cfg.Scale
		usage["max_connections"] = p.cfg.Connections
	}

	// Metrics information
	if p.metrics != nil {
		snapshot := p.metrics.Snapshot()
		usage["total_operations"] = snapshot.TotalOperations
		usage["total_errors"] = snapshot.NumErrors
		usage["connections_active"] = snapshot.NumConnections

		if snapshot.LatencyCount > 0 {
			usage["avg_latency_ms"] = float64(snapshot.LatencySum) / float64(snapshot.LatencyCount) / 1e6
		}

		usage["operations_breakdown"] = map[string]interface{}{
			"inserts": snapshot.NumInsert,
			"updates": snapshot.NumUpdate,
			"selects": snapshot.NumSelect,
			"deletes": snapshot.NumDelete,
		}
	}

	return usage
}

// GetStatus returns current plugin status and basic metrics.
func (p *TPCCPlugin) GetStatus() map[string]interface{} {
	running := atomic.LoadInt64(&p.isRunning) == 1
	return map[string]interface{}{"running": running}
}

// chooseTransactionType selects a transaction type based on configured percentages
func chooseTransactionType(cfg *TPCCConfig) string {
	r := rand.Intn(100)
	cumulative := cfg.NewOrderPct
	if r < cumulative {
		return "new_order"
	}
	cumulative += cfg.PaymentPct
	if r < cumulative {
		return "payment"
	}
	cumulative += cfg.OrderStatusPct
	if r < cumulative {
		return "order_status"
	}
	cumulative += cfg.DeliveryPct
	if r < cumulative {
		return "delivery"
	}
	cumulative += cfg.StockLevelPct
	if r < cumulative {
		return "stock_level"
	}
	if cfg.EnableSupplierReorder {
		cumulative += cfg.SupplierReorderPct
		if r < cumulative {
			return "supplier_reorder"
		}
	}
	// default to new_order
	return "new_order"
}

// executeTransactionWithMetrics dispatches TPC-C transactions with enhanced metrics tracking
func (p *TPCCPlugin) executeTransactionWithMetrics(db *sql.DB, txType string, warehouseID int, workerID int, metrics *PerformanceMetrics) error {
	// Delegate to transaction package with enhanced context
	return txn.ExecuteTransactionWithMetrics(
		context.Background(),
		db,
		txn.TransactionType(txType),
		warehouseID,
		p.cfg.Scale,
		workerID,
		metrics,
	)
}
