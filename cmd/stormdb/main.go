// cmd/stormdb/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/elchinoo/stormdb/internal/config"
	"github.com/elchinoo/stormdb/internal/database"
	"github.com/elchinoo/stormdb/internal/metrics"
	"github.com/elchinoo/stormdb/internal/progressive"
	"github.com/elchinoo/stormdb/internal/results"
	"github.com/elchinoo/stormdb/internal/workload"
	"github.com/elchinoo/stormdb/pkg/types"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

// Version information (set by build system via ldflags)
var (
	Version   = "v0.1.0-beta" // Version string
	GitCommit = "unknown"     // Git commit hash
	BuildTime = "unknown"     // Build timestamp
	GoVersion = "unknown"     // Go version used for build
)

func main() {
	var (
		configFile        string
		setup             bool
		rebuild           bool
		host              string
		port              int
		dbname            string
		username          string
		password          string
		workload          string
		workers           int
		duration          string
		scale             int
		connections       int
		summaryInterval   string
		noSummary         bool
		collectPgStats    bool
		pgStatsStatements bool
		showVersion       bool
		progressiveMode   bool
		enableProfiling   bool
		profilingPort     string
	)

	rootCmd := &cobra.Command{
		Use:   "stormdb",
		Short: "A extensible database load testing tool",
		RunE: func(_ *cobra.Command, _ []string) error {
			if showVersion {
				fmt.Printf("StormDB v%s\n", Version)
				fmt.Printf("  Git Commit: %s\n", GitCommit)
				fmt.Printf("  Build Time: %s\n", BuildTime)
				fmt.Printf("  Go Version: %s\n", GoVersion)
				return nil
			}
			return runLoadTest(configFile, setup, rebuild, &CLIOptions{
				Host:              host,
				Port:              port,
				Dbname:            dbname,
				Username:          username,
				Password:          password,
				Workload:          workload,
				Workers:           workers,
				Duration:          duration,
				Scale:             scale,
				Connections:       connections,
				SummaryInterval:   summaryInterval,
				NoSummary:         noSummary,
				CollectPgStats:    collectPgStats,
				PgStatsStatements: pgStatsStatements,
				ProgressiveMode:   progressiveMode,
				EnableProfiling:   enableProfiling,
				ProfilingPort:     profilingPort,
			})
		},
	}

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("StormDB v%s\n", Version)
			fmt.Printf("  Git Commit: %s\n", GitCommit)
			fmt.Printf("  Build Time: %s\n", BuildTime)
			fmt.Printf("  Go Version: %s\n", GoVersion)
		},
	}
	rootCmd.AddCommand(versionCmd)
	
	// Plugins command
	rootCmd.AddCommand(createPluginsCommand())

	// File and setup options
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Path to config file")
	rootCmd.Flags().BoolVar(&setup, "setup", false, "Ensure schema exists (create if needed, but do not load data)")
	rootCmd.Flags().BoolVarP(&rebuild, "rebuild", "r", false, "Rebuild: drop, recreate schema, and load data")

	// Database connection options (override config file)
	rootCmd.Flags().StringVar(&host, "host", "", "Database host (overrides config)")
	rootCmd.Flags().IntVar(&port, "port", 0, "Database port (overrides config)")
	rootCmd.Flags().StringVar(&dbname, "dbname", "", "Database name (overrides config)")
	rootCmd.Flags().StringVarP(&username, "username", "u", "", "Database username (overrides config)")
	rootCmd.Flags().StringVarP(&password, "password", "p", "", "Database password (overrides config)")

	// Workload options (override config file)
	rootCmd.Flags().StringVarP(&workload, "workload", "w", "", "Workload type (overrides config)")
	rootCmd.Flags().IntVar(&workers, "workers", 0, "Number of worker threads (overrides config)")
	rootCmd.Flags().StringVarP(&duration, "duration", "d", "", "Test duration, e.g., 30s, 1m (overrides config)")
	rootCmd.Flags().IntVar(&scale, "scale", 0, "Scale factor (overrides config)")
	rootCmd.Flags().IntVar(&connections, "connections", 0, "Max connections in pool (overrides config)")
	rootCmd.Flags().StringVarP(&summaryInterval, "summary-interval", "s", "", "Periodic summary interval, e.g., 10s, 30s (overrides config)")
	rootCmd.Flags().BoolVar(&noSummary, "no-summary", false, "Disable periodic summary reporting")
	rootCmd.Flags().BoolVar(&collectPgStats, "collect-pg-stats", false, "Enable PostgreSQL statistics collection")
	rootCmd.Flags().BoolVar(&pgStatsStatements, "pg-stat-statements", false, "Enable pg_stat_statements collection (requires extension)")
	rootCmd.Flags().BoolVar(&progressiveMode, "progressive", false, "Enable progressive connection scaling (overrides config)")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "V", false, "Show version information and exit")
	rootCmd.Flags().BoolVar(&enableProfiling, "profile", false, "Enable memory profiling server")
	rootCmd.Flags().StringVar(&profilingPort, "profile-port", "6060", "Port for profiling server (default: 6060)")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// CLIOptions holds command-line overrides
type CLIOptions struct {
	Host              string
	Port              int
	Dbname            string
	Username          string
	Password          string
	Workload          string
	Workers           int
	Duration          string
	Scale             int
	Connections       int
	SummaryInterval   string
	NoSummary         bool
	CollectPgStats    bool
	PgStatsStatements bool
	ProgressiveMode   bool
	EnableProfiling   bool
	ProfilingPort     string
}

func runLoadTest(configFile string, setup bool, rebuild bool, cliOpts *CLIOptions) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply CLI overrides to config
	applyCliOverrides(cfg, cliOpts)

	// Start profiling server if enabled
	if cliOpts.EnableProfiling {
		go func() {
			log.Printf("🔍 Starting memory profiling server on port %s", cliOpts.ProfilingPort)
			log.Printf("📊 Access profiling at: http://localhost:%s/debug/pprof/", cliOpts.ProfilingPort)
			log.Printf("💾 Memory profile: http://localhost:%s/debug/pprof/heap", cliOpts.ProfilingPort)
			log.Printf("⚡ CPU profile: http://localhost:%s/debug/pprof/profile", cliOpts.ProfilingPort)

			// Force garbage collection and print memory stats periodically
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			go func() {
				for range ticker.C {
					runtime.GC()
					var m runtime.MemStats
					runtime.ReadMemStats(&m)
					log.Printf("📈 Memory: Alloc=%dMB, TotalAlloc=%dMB, Sys=%dMB, NumGC=%d",
						bToMb(m.Alloc), bToMb(m.TotalAlloc), bToMb(m.Sys), m.NumGC)
				}
			}()

			if err := http.ListenAndServe(":"+cliOpts.ProfilingPort, nil); err != nil {
				log.Printf("Warning: Failed to start profiling server: %v", err)
			}
		}()
	}

	duration, err := time.ParseDuration(cfg.Duration)
	if err != nil {
		return fmt.Errorf("invalid duration '%s': %w", cfg.Duration, err)
	}

	// Handle summary interval with default and no-summary flag
	var summaryInterval time.Duration
	if cliOpts.NoSummary {
		// If --no-summary is set, disable summary reporting
		summaryInterval = 0
	} else if cfg.SummaryInterval != "" {
		// If summary interval is configured, use it
		summaryInterval, err = time.ParseDuration(cfg.SummaryInterval)
		if err != nil {
			return fmt.Errorf("invalid summary_interval '%s': %w", cfg.SummaryInterval, err)
		}
	} else {
		// Default to 10 seconds if not configured and not disabled
		summaryInterval = 10 * time.Second
	}

	db, err := database.NewPostgres(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Initialize workload factory with plugin support
	factory, err := workload.NewFactory(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize workload factory: %w", err)
	}
	defer func() { _ = factory.Cleanup() }()

	// Discover and initialize plugins
	if err := factory.Initialize(); err != nil {
		log.Printf("Warning: Failed to initialize factory: %v", err)
	}

	// Discover plugins (this will search configured plugin paths)
	pluginCount, err := factory.DiscoverPlugins()
	if err != nil {
		log.Printf("Warning: Plugin discovery issues: %v", err)
	} else if pluginCount > 0 {
		log.Printf("🔌 Discovered %d plugin(s)", pluginCount)
	}

	// Create workload instance
	wl, err := factory.Get(cfg.Workload)
	if err != nil {
		return fmt.Errorf("failed to create workload '%s': %w", cfg.Workload, err)
	}

	// -------------------------------
	// Phase 1: Schema & Data Control
	// -------------------------------

	switch {
	case rebuild:
		log.Printf("💥 Rebuilding: dropping and recreating schema + data")
		if err := wl.Cleanup(context.Background(), db.Pool, cfg); err != nil {
			return fmt.Errorf("failed to cleanup: %w", err)
		}
		// ✅ Add Setup after Cleanup
		if err := wl.Setup(context.Background(), db.Pool, cfg); err != nil {
			return fmt.Errorf("failed to setup after rebuild: %w", err)
		}

	case setup:
		log.Printf("🔧 Ensuring schema exists (no data load)")
		if err := wl.Setup(context.Background(), db.Pool, cfg); err != nil {
			return fmt.Errorf("failed to setup schema: %w", err)
		}

	default:
		log.Printf("⏭️  Skipping setup (--setup or --rebuild not used). Assuming schema and data exist.")
	}

	// -------------------------------
	// Phase 2: Progressive Scaling or Regular Workload
	// -------------------------------

	// Check if progressive scaling is enabled
	if cfg.Progressive.Enabled {
		log.Printf("🎯 Starting progressive scaling mode")

		// Create a workload adapter for the progressive engine
		workloadAdapter := &WorkloadAdapter{workload: wl}

		// Create progressive scaling engine
		engine := progressive.NewScalingEngine(cfg, workloadAdapter, db.Pool)

		// Execute progressive scaling
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Set up signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Start progressive scaling in a goroutine
		resultChan := make(chan *types.ProgressiveScalingResult, 1)
		errChan := make(chan error, 1)

		go func() {
			result, err := engine.Execute(ctx)
			if err != nil {
				errChan <- err
				return
			}
			resultChan <- result
		}()

		// Wait for completion or signal
		select {
		case result := <-resultChan:
			log.Printf("✅ Progressive scaling completed successfully")
			log.Printf("📊 Tested %d bands, optimal config: %d workers, %d connections (%.2f TPS)",
				len(result.Bands), result.OptimalConfig.Workers, result.OptimalConfig.Connections, result.OptimalConfig.TPS)
			return nil
		case err := <-errChan:
			return fmt.Errorf("progressive scaling failed: %w", err)
		case sig := <-sigChan:
			log.Printf("🛑 Received signal %v, shutting down progressive scaling...", sig)
			cancel()
			// Wait a bit for graceful shutdown
			select {
			case <-resultChan:
				log.Printf("✅ Progressive scaling completed after signal")
			case <-time.After(10 * time.Second):
				log.Printf("⚠️  Progressive scaling shutdown timeout")
			}
			return fmt.Errorf("interrupted by signal: %v", sig)
		}
	}

	// -------------------------------
	// Phase 2: Run the regular workload
	// -------------------------------

	log.Printf("🚀 Starting %s workload for %v with %d workers", cfg.Workload, duration, cfg.Workers)

	metricsData := &types.Metrics{
		ErrorTypes: make(map[string]int64),
		Mu:         sync.Mutex{},
	}

	// Initialize latency histogram
	metricsData.InitializeLatencyHistogram()

	// Start PostgreSQL statistics collector if enabled
	var pgStatsCollector *database.PgStatsCollector
	if cfg.CollectPgStats {
		pgStatsCollector = database.NewPgStatsCollector(db.Pool, metricsData, cfg.PgStatsStatements)
		pgStatsCollector.Start()
		log.Printf("📊 PostgreSQL statistics collection enabled (pg_stat_statements: %v)", cfg.PgStatsStatements)
		defer pgStatsCollector.Stop()
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start workload in a goroutine
	errChan := make(chan error, 1)
	go func() {
		// Capture baseline statistics immediately before workload starts
		if cfg.CollectPgStats && pgStatsCollector != nil {
			pgStatsCollector.CaptureWorkloadBaseline()
		}

		errChan <- wl.Run(ctx, db.Pool, cfg, metricsData)
	}()

	// Start periodic summary reporting if interval is configured
	var summaryTicker *time.Ticker
	var summaryDone chan bool
	startTime := time.Now() // Always record start time for results storage
	if summaryInterval > 0 {
		summaryTicker = time.NewTicker(summaryInterval)
		summaryDone = make(chan bool)
		go func() {
			for {
				select {
				case <-summaryTicker.C:
					elapsed := time.Since(startTime)
					metrics.ReportSummary(cfg, metricsData, elapsed)
				case <-summaryDone:
					return
				}
			}
		}()
	}

	// Wait for either completion, error, or signal
	var workloadErr error
	var interrupted bool

	select {
	case workloadErr = <-errChan:
		// Workload completed normally
	case sig := <-sigChan:
		log.Printf("\n🛑 Received signal %v, shutting down gracefully...", sig)
		interrupted = true
		cancel() // Cancel the context to stop the workload

		// Wait a bit for workload to finish gracefully
		select {
		case workloadErr = <-errChan:
			// Workload finished
		case <-time.After(5 * time.Second):
			log.Printf("⚠️  Workload didn't finish in 5 seconds, forcing shutdown")
		}
	}

	// Clean up periodic summary ticker
	if summaryTicker != nil {
		summaryTicker.Stop()
		summaryDone <- true
	}

	// Calculate final PostgreSQL statistics after workload completion
	if cfg.CollectPgStats && pgStatsCollector != nil {
		if finalStats := pgStatsCollector.CalculateFinalStats(); finalStats != nil {
			metricsData.UpdatePgStats(finalStats)
		}
	}

	// -------------------------------
	// Phase 3: Store results in database backend (if configured)
	// -------------------------------

	// Record end time for test results storage
	testEndTime := time.Now()

	// Initialize and store test results in database backend if configured
	if resultsBackend, err := results.CreateBackendFromConfig(cfg); err != nil {
		log.Printf("⚠️  Failed to create results backend: %v", err)
	} else if resultsBackend != nil {
		defer resultsBackend.Close()

		// Store test results
		if err := results.StoreTestResults(context.Background(), resultsBackend, cfg, metricsData, startTime, testEndTime); err != nil {
			log.Printf("⚠️  Failed to store test results: %v", err)
		} else {
			log.Printf("💾 Test results stored in database backend")
		}

		// Perform maintenance (cleanup old results)
		if err := results.PerformMaintenance(context.Background(), resultsBackend); err != nil {
			log.Printf("⚠️  Failed to perform backend maintenance: %v", err)
		}
	}

	// -------------------------------
	// Phase 4: Report results
	// -------------------------------

	if interrupted {
		log.Printf("\n📊 Final Summary (interrupted):")
		metrics.ReportWithContext(cfg, metricsData, true, time.Now())
	} else {
		log.Printf("\n📊 Final Summary:")
		metrics.Report(cfg, metricsData)
	}

	if workloadErr != nil && !interrupted {
		return fmt.Errorf("workload failed: %w", workloadErr)
	}

	return nil
}

// applyCliOverrides applies command-line options to the config, giving CLI higher priority
func applyCliOverrides(cfg *types.Config, cliOpts *CLIOptions) {
	// Database overrides
	if cliOpts.Host != "" {
		cfg.Database.Host = cliOpts.Host
	}
	if cliOpts.Port > 0 {
		cfg.Database.Port = cliOpts.Port
	}
	if cliOpts.Dbname != "" {
		cfg.Database.Dbname = cliOpts.Dbname
	}
	if cliOpts.Username != "" {
		cfg.Database.Username = cliOpts.Username
	}
	if cliOpts.Password != "" {
		cfg.Database.Password = cliOpts.Password
	}

	// Workload overrides
	if cliOpts.Workload != "" {
		cfg.Workload = cliOpts.Workload
	}
	if cliOpts.Workers > 0 {
		cfg.Workers = cliOpts.Workers
	}
	if cliOpts.Duration != "" {
		cfg.Duration = cliOpts.Duration
	}
	if cliOpts.Scale > 0 {
		cfg.Scale = cliOpts.Scale
	}
	if cliOpts.Connections > 0 {
		cfg.Connections = cliOpts.Connections
	}
	if cliOpts.SummaryInterval != "" {
		cfg.SummaryInterval = cliOpts.SummaryInterval
	}

	// PostgreSQL statistics overrides
	if cliOpts.CollectPgStats {
		cfg.CollectPgStats = true
	}
	if cliOpts.PgStatsStatements {
		cfg.PgStatsStatements = true
	}

	// Progressive scaling override
	if cliOpts.ProgressiveMode {
		cfg.Progressive.Enabled = true
	}
}

// WorkloadAdapter adapts the plugin workload interface to the progressive engine interface
type WorkloadAdapter struct {
	workload workload.Workload
}

// Setup ensures schema exists (called with --setup or --rebuild)
func (w *WorkloadAdapter) Setup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	return w.workload.Setup(ctx, db, cfg)
}

// Run executes the load test with the given configuration
func (w *WorkloadAdapter) Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	return w.workload.Run(ctx, db, cfg, metrics)
}

// Cleanup drops tables and reloads data (called only with --rebuild)
func (w *WorkloadAdapter) Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	return w.workload.Cleanup(ctx, db, cfg)
}

// bToMb converts bytes to megabytes
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
