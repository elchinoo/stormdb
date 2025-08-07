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
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/elchinoo/stormdb/core"
	"github.com/elchinoo/stormdb/plugins/tpcc-scalability/schema"
	"github.com/elchinoo/stormdb/plugins/tpcc-scalability/txn"
	_ "github.com/lib/pq"
)

// TPCCPlugin implements the TPC-C scalability test skeleton
type TPCCPlugin struct {
	core   *core.CoreServices
	logger core.Logger
	db     *sql.DB
	cfg    *TPCCConfig

	isRunning int64
	stopChan  chan struct{}
	stopOnce  sync.Once // ensure stopChan closed only once on error limit
	wg        sync.WaitGroup

	metrics *TPCCMetrics
	stats   *Stats
	// ExecTx executes a transaction; can be overridden for tests
	ExecTx func(db *sql.DB, txType string, warehouseID int) error
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
	return core.PluginMetadata{
		Name:         "tpcc-scalability",
		Version:      "2.0.0",
		Description:  "TPC-C scalability test plugin",
		Author:       "StormDB Team",
		License:      "MIT",
		TestTypes:    []string{"tpcc", "scalability", "performance"},
		Dependencies: map[string]string{"postgresql": ">=12.0"},
	}
}

// Initialize sets up core services and metrics.
func (p *TPCCPlugin) Initialize(ctx context.Context, cs *core.CoreServices) error {
	p.core = cs
	p.logger = cs.Logger.WithPlugin(p.Metadata().Name)
	p.stopChan = make(chan struct{})
	p.metrics = &TPCCMetrics{SecondlyMetrics: make([]SecondMetrics, 0)}
	p.stats = &Stats{}
	// default transaction executor
	p.ExecTx = p.executeTransaction
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
		return fmt.Errorf("validation error: %w", err)
	}
	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		p.cfg.Host, p.cfg.Port, p.cfg.Database, p.cfg.Username, p.cfg.Password, p.cfg.SSLMode)
	var err error
	p.db, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}
	if err := p.db.PingContext(ctx); err != nil {
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
	// Dispatch based on mode
	switch p.cfg.Mode {
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
	pluginFiles, _ := filepath.Glob(pluginPattern)
	if len(pluginFiles) > 0 {
		p.logger.Info("Applying migrations from plugin", core.Field{Key: "dir", Value: pluginDir})
		if err := schema.Migrate(ctx, p.db, pluginDir); err != nil {
			return fmt.Errorf("schema migration failed (plugin): %w", err)
		}
		return nil
	}
	// fallback to root migrations directory
	localDir := "migrations"
	localPattern := filepath.Join(localDir, "*.up.sql")
	localFiles, _ := filepath.Glob(localPattern)
	if len(localFiles) > 0 {
		p.logger.Info("Applying migrations from root", core.Field{Key: "dir", Value: localDir})
		if err := schema.Migrate(ctx, p.db, localDir); err != nil {
			return fmt.Errorf("schema migration failed (root): %w", err)
		}
		return nil
	}
	return fmt.Errorf("no migration files found in %s or %s", pluginDir, localDir)
}

// populateData inserts seed and TPC-C workload data
func (p *TPCCPlugin) populateData(ctx context.Context) error {
	p.logger.Info("Populating data for TPC-C")
	if err := p.populateSeedData(ctx); err != nil {
		return fmt.Errorf("populateSeedData failed: %w", err)
	}
	if err := p.populateTPCCData(ctx); err != nil {
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
		p.logger.Warn("dropTables failed", core.Field{Key: "error", Value: err.Error()})
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
					return fmt.Errorf("populateTPCCData customer %d.%d.%d: %w", w, d, c, err)
				}
			}
		}
	}
	return nil
}

// runTestWorkload runs the TPC-C workload per connection levels
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
		// reset error and transaction counters for this level
		atomic.StoreInt64(&p.metrics.TotalTransactions, 0)
		atomic.StoreInt64(&p.metrics.Errors, 0)
		p.logger.Info("Starting level",
			core.Field{Key: "level", Value: idx + 1},
			core.Field{Key: "connections", Value: conns})
		// record stats start for this level
		p.stats.Start(conns)
		// Warmup
		if p.cfg.WarmupTime > 0 {
			p.logger.Info("Warmup phase",
				core.Field{Key: "duration", Value: p.cfg.WarmupTime})
			warmCtx, cancel := context.WithTimeout(ctx, time.Duration(p.cfg.WarmupTime))
			p.runWorkload(warmCtx, conns, time.Duration(p.cfg.ThinkTime))
			cancel()
		}
		// Measurement
		p.logger.Info("Measurement phase",
			core.Field{Key: "connections", Value: conns}, core.Field{Key: "duration", Value: perLevel})
		measureCtx, cancel := context.WithTimeout(ctx, perLevel)
		// real-time reporting ticker
		ticker := time.NewTicker(time.Duration(p.cfg.MetricsInterval))
		// track last error count for per-second delta
		lastErrCount := atomic.LoadInt64(&p.metrics.Errors)
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-measureCtx.Done():
					return
				case <-p.stopChan:
					return
				case <-ticker.C:
					snap := p.stats.SnapshotAndReset()
					// compute per-second metrics
					ops := snap.NumInsert + snap.NumUpdate + snap.NumDelete + snap.NumSelect
					durationSec := time.Duration(p.cfg.MetricsInterval).Seconds()
					tps := float64(ops) / durationSec
					// compute errors delta
					currErr := atomic.LoadInt64(&p.metrics.Errors)
					errs := currErr - lastErrCount
					lastErrCount = currErr
					// append to in-memory slice
					p.metrics.SecondlyMetrics = append(p.metrics.SecondlyMetrics, SecondMetrics{
						Timestamp: time.Now(),
						Count:     ops,
						TPS:       tps,
						Errors:    errs,
					})
					// console logging
					if p.cfg.Verbose {
						p.logger.Info("Interval stats",
							core.Field{Key: "connections", Value: snap.NumConnections},
							core.Field{Key: "ops", Value: ops},
							core.Field{Key: "tps", Value: tps},
							core.Field{Key: "errors", Value: errs},
						)
					}
					// persist metrics to storage via storage-enabled logger
					if p.cfg.StreamMetrics {
						storeLog := p.logger.WithStorage(p.core.Storage)
						storeLog.Info("Persist interval stats",
							core.Field{Key: "connections", Value: snap.NumConnections},
							core.Field{Key: "ops", Value: ops},
							core.Field{Key: "tps", Value: tps},
							core.Field{Key: "errors", Value: errs},
						)
						// also store TestResult
						results := []core.TestResult{{
							StartTime:         snap.DtStarted,
							EndTime:           snap.DtEnded,
							Value:             float64(ops),
							ActiveConnections: &snap.NumConnections,
							Tags: map[string]interface{}{
								"num_insert":     snap.NumInsert,
								"num_update":     snap.NumUpdate,
								"num_delete":     snap.NumDelete,
								"num_select":     snap.NumSelect,
								"latency_sum":    snap.LatencySum,
								"latency_count":  snap.LatencyCount,
								"num_row_insert": snap.NumRowInsert,
								"num_row_update": snap.NumRowUpdate,
								"num_row_delete": snap.NumRowDelete,
								"num_row_select": snap.NumRowSelect,
								"avg_latency_ms": float64(snap.LatencySum) / float64(snap.LatencyCount) / 1e6,
							},
						}}
						_ = p.core.Storage.StoreResults(ctx, results)
					}
				}
			}
		}()
		p.runWorkload(measureCtx, conns, time.Duration(p.cfg.ThinkTime))
		cancel()
		// record stats end for this level
		p.stats.End()
		// check error rate limit
		if p.cfg.StopOnErrorLimit {
			total := atomic.LoadInt64(&p.metrics.TotalTransactions)
			errs := atomic.LoadInt64(&p.metrics.Errors)
			if total > 0 {
				rate := float64(errs) / float64(total)
				if rate > p.cfg.MaxErrorRate {
					p.logger.Warn("Error rate limit exceeded, stopping workload", core.Field{Key: "error_rate", Value: rate})
					return nil
				}
			}
		}
	}
	// final summary
	finalSnap := p.stats.SnapshotAndReset()
	// calculate average latency for final summary
	avgMs := float64(0)
	if finalSnap.LatencyCount > 0 {
		avgMs = float64(finalSnap.LatencySum) / float64(finalSnap.LatencyCount) / 1e6
	}
	finalResults := []core.TestResult{{
		StartTime:         finalSnap.DtStarted,
		EndTime:           finalSnap.DtEnded,
		Value:             float64(finalSnap.NumInsert + finalSnap.NumUpdate + finalSnap.NumDelete + finalSnap.NumSelect),
		ActiveConnections: &finalSnap.NumConnections,
		Tags: map[string]interface{}{
			"num_insert":     finalSnap.NumInsert,
			"num_update":     finalSnap.NumUpdate,
			"num_delete":     finalSnap.NumDelete,
			"num_select":     finalSnap.NumSelect,
			"latency_sum":    finalSnap.LatencySum,
			"latency_count":  finalSnap.LatencyCount,
			"num_row_insert": finalSnap.NumRowInsert,
			"num_row_update": finalSnap.NumRowUpdate,
			"num_row_delete": finalSnap.NumRowDelete,
			"num_row_select": finalSnap.NumRowSelect,
			"avg_latency_ms": avgMs,
		},
	}}
	if p.cfg.StreamMetrics {
		_ = p.core.Storage.StoreResults(ctx, finalResults)
	}
	return nil
}

// runWorkload starts workers that execute transactions until context is done
func (p *TPCCPlugin) runWorkload(ctx context.Context, connections int, thinkTime time.Duration) {
	p.wg.Add(connections)
	for i := 0; i < connections; i++ {
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-p.stopChan:
					return
				default:
				}
				// choose and execute transaction
				op := chooseTransactionType(p.cfg)
				// pick random warehouse
				warehouse := rand.Intn(p.cfg.Scale) + 1
				start := time.Now()
				// execute transaction via ExecTx or fallback to method
				executor := p.ExecTx
				if executor == nil {
					executor = p.executeTransaction
				}
				err := executor(p.db, op, warehouse)
				// count metrics
				total := atomic.AddInt64(&p.metrics.TotalTransactions, 1)
				if err != nil {
					atomic.AddInt64(&p.metrics.Errors, 1)
				}
				// check error rate limit per operation
				if p.cfg.StopOnErrorLimit {
					errs := atomic.LoadInt64(&p.metrics.Errors)
					if total > 0 && float64(errs)/float64(total) > p.cfg.MaxErrorRate {
						// stop all workers once
						p.stopOnce.Do(func() { close(p.stopChan) })
						return
					}
				}
				latency := time.Since(start)
				// record one row per operation
				p.stats.Record(op, latency, 1)
				time.Sleep(thinkTime)
			}
		}()
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
	return nil
}

// Cleanup releases resources and closes database connection.
func (p *TPCCPlugin) Cleanup(ctx context.Context) error {
	// ensure workers are stopped
	p.stopOnce.Do(func() { close(p.stopChan) })
	p.wg.Wait()
	// close database connection
	if p.db != nil {
		_ = p.db.Close()
	}
	return nil
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

// executeTransaction dispatches TPC-C transactions based on type.
func (p *TPCCPlugin) executeTransaction(db *sql.DB, txType string, warehouseID int) error {
	// delegate transaction execution to txn package
	return txn.ExecuteTransaction(
		context.Background(),
		db,
		txn.TransactionType(txType),
		warehouseID,
		p.cfg.Scale,
	)
}
