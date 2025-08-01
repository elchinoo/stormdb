package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/elchinoo/stormdb/internal/config"
	"github.com/elchinoo/stormdb/internal/logging"
	"github.com/elchinoo/stormdb/internal/progressive"
	"github.com/elchinoo/stormdb/pkg/types"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// progressiveCmd represents the progressive command
var progressiveCmd = &cobra.Command{
	Use:   "progressive",
	Short: "Run progressive scaling tests",
	Long: `Run progressive scaling tests that automatically increment connection counts
and workers to identify optimal scaling characteristics and bottlenecks.

Examples:
  # Run a basic progressive test with linear scaling
  stormdb progressive --config config.yaml --strategy linear --bands 5

  # Run exponential scaling with custom parameters
  stormdb progressive --config config.yaml --strategy exponential \
    --min-workers 10 --max-workers 100 --min-connections 10 --max-connections 200

  # Run with comprehensive analysis
  stormdb progressive --config config.yaml --enable-analysis \
    --output results.json --report report.html`,
	RunE: runProgressiveTest,
}

// Progressive test flags
var (
	progressiveConfigFile     string
	progressiveStrategy       string
	progressiveBands          int
	progressiveMinWorkers     int
	progressiveMaxWorkers     int
	progressiveMinConns       int
	progressiveMaxConns       int
	progressiveTestDuration   time.Duration
	progressiveWarmupTime     time.Duration
	progressiveCooldownTime   time.Duration
	progressiveEnableAnalysis bool
	progressiveOutputFile     string
	progressiveReportFile     string
	progressiveVerbose        bool
)

func init() {
	rootCmd.AddCommand(progressiveCmd)

	// Configuration flags
	progressiveCmd.Flags().StringVarP(&progressiveConfigFile, "config", "c", "", "Configuration file path (required)")
	progressiveCmd.Flags().StringVar(&progressiveStrategy, "strategy", "linear", "Scaling strategy (linear, exponential, fibonacci)")
	progressiveCmd.Flags().IntVar(&progressiveBands, "bands", 5, "Number of test bands")

	// Scaling parameters
	progressiveCmd.Flags().IntVar(&progressiveMinWorkers, "min-workers", 0, "Minimum number of workers (0 = use config)")
	progressiveCmd.Flags().IntVar(&progressiveMaxWorkers, "max-workers", 0, "Maximum number of workers (0 = use config)")
	progressiveCmd.Flags().IntVar(&progressiveMinConns, "min-connections", 0, "Minimum connections (0 = use config)")
	progressiveCmd.Flags().IntVar(&progressiveMaxConns, "max-connections", 0, "Maximum connections (0 = use config)")

	// Timing parameters
	progressiveCmd.Flags().DurationVar(&progressiveTestDuration, "test-duration", 0, "Duration per test band (0 = use config)")
	progressiveCmd.Flags().DurationVar(&progressiveWarmupTime, "warmup-time", 0, "Warmup time per band (0 = use config)")
	progressiveCmd.Flags().DurationVar(&progressiveCooldownTime, "cooldown-time", 0, "Cooldown time per band (0 = use config)")

	// Analysis and output
	progressiveCmd.Flags().BoolVar(&progressiveEnableAnalysis, "enable-analysis", true, "Enable comprehensive analysis")
	progressiveCmd.Flags().StringVarP(&progressiveOutputFile, "output", "o", "", "Output file for results (JSON)")
	progressiveCmd.Flags().StringVar(&progressiveReportFile, "report", "", "Generate HTML report file")
	progressiveCmd.Flags().BoolVarP(&progressiveVerbose, "verbose", "v", false, "Verbose output")

	// Required flags
	if err := progressiveCmd.MarkFlagRequired("config"); err != nil {
		// This should never happen, but handle gracefully
		return
	}
}

func runProgressiveTest(cmd *cobra.Command, args []string) error {
	// Load base configuration
	baseConfig, err := config.Load(progressiveConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create enhanced configuration with progressive settings
	enhancedConfig, err := createProgressiveConfig(baseConfig)
	if err != nil {
		return fmt.Errorf("failed to create progressive config: %w", err)
	}

	// Initialize logger
	loggerConfig := logging.LoggerConfig{
		Level:       "info",
		Format:      "json",
		Development: false,
	}
	logger, err := logging.NewLogger(loggerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() {
		if syncErr := logger.Sync(); syncErr != nil {
			// Ignore errors from stderr/stdout sync on exit
		}
	}()

	if progressiveVerbose {
		logger = logger.With(zap.Bool("verbose", true))
	}

	logger.Info("Starting progressive scaling test",
		zap.String("config_file", progressiveConfigFile),
		zap.String("strategy", progressiveStrategy),
		zap.Int("bands", progressiveBands),
		zap.Bool("analysis_enabled", progressiveEnableAnalysis),
	)

	// Create progressive runner
	runner, err := progressive.NewProgressiveRunner(enhancedConfig.Workload.Progressive, logger)
	if err != nil {
		return fmt.Errorf("failed to create progressive runner: %w", err)
	}

	// Create context with timeout
	totalDuration := estimateTotalDuration(enhancedConfig.Workload.Progressive)
	ctx, cancel := context.WithTimeout(context.Background(), totalDuration+5*time.Minute)
	defer cancel()

	// Print test plan
	if err := printTestPlan(runner, logger); err != nil {
		logger.Warn("Failed to print test plan", zap.Error(err))
	}

	// Create workload runner function
	workloadRunner := createWorkloadRunner(enhancedConfig, logger)

	// Start the progressive test
	if err := runner.Run(ctx, workloadRunner); err != nil {
		return fmt.Errorf("progressive test failed: %w", err)
	}

	// Get results
	results := runner.GetResults()

	logger.Info("Progressive test completed",
		zap.Int("completed_bands", len(results)),
		zap.Duration("total_duration", time.Since(time.Now())),
	)

	// Output results
	if err := outputResults(results, logger); err != nil {
		logger.Error("Failed to output results", err)
		return err
	}

	// Generate report if requested
	if progressiveReportFile != "" {
		if err := generateHTMLReport(results, progressiveReportFile, logger); err != nil {
			logger.Error("Failed to generate HTML report", err)
			return err
		}
		logger.Info("HTML report generated", zap.String("file", progressiveReportFile))
	}

	return nil
}

// createProgressiveConfig creates progressive configuration from base config and CLI flags
func createProgressiveConfig(baseConfig *types.Config) (*config.StormDBConfig, error) {
	// Parse duration
	duration, err := time.ParseDuration(baseConfig.Duration)
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %w", err)
	}

	// Parse summary interval
	summaryInterval := 10 * time.Second
	if baseConfig.SummaryInterval != "" {
		summaryInterval, err = time.ParseDuration(baseConfig.SummaryInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid summary interval: %w", err)
		}
	}

	// Convert base config to enhanced config
	enhanced := &config.StormDBConfig{
		Version: "1.0.0",
		Database: config.DatabaseConfig{
			Type:              "postgres",
			Host:              baseConfig.Database.Host,
			Port:              baseConfig.Database.Port,
			Database:          baseConfig.Database.Dbname,
			Username:          baseConfig.Database.Username,
			Password:          baseConfig.Database.Password,
			SSLMode:           baseConfig.Database.Sslmode,
			MaxConnections:    max(baseConfig.Connections, 100),
			MinConnections:    1,
			ConnectTimeout:    30 * time.Second,
			MaxConnLifetime:   time.Hour,
			MaxConnIdleTime:   15 * time.Minute,
			HealthCheckPeriod: 5 * time.Minute,
		},
		Workload: config.WorkloadConfig{
			Type:            baseConfig.Workload,
			Duration:        duration,
			Workers:         baseConfig.Workers,
			Connections:     baseConfig.Connections,
			Scale:           baseConfig.Scale,
			SummaryInterval: summaryInterval,
			Config:          make(map[string]interface{}),
		},
		Logger: config.LoggerConfig{
			Level:       "info",
			Format:      "json",
			Development: false,
		},
	}

	// Create progressive configuration
	progressiveConfig := &config.ProgressiveConfig{
		Enabled:           true,
		Strategy:          progressiveStrategy,
		Bands:             progressiveBands,
		EnableAnalysis:    progressiveEnableAnalysis,
		TestDuration:      progressiveTestDuration,
		WarmupDuration:    progressiveWarmupTime,
		CooldownDuration:  progressiveCooldownTime,
		MaxLatencySamples: 10000,
		MemoryLimitMB:     1024,
	}

	// Apply CLI overrides or use defaults
	if progressiveMinWorkers > 0 {
		progressiveConfig.MinWorkers = progressiveMinWorkers
	} else {
		progressiveConfig.MinWorkers = max(1, baseConfig.Workers/4)
	}

	if progressiveMaxWorkers > 0 {
		progressiveConfig.MaxWorkers = progressiveMaxWorkers
	} else {
		progressiveConfig.MaxWorkers = baseConfig.Workers * 2
	}

	if progressiveMinConns > 0 {
		progressiveConfig.MinConnections = progressiveMinConns
	} else {
		progressiveConfig.MinConnections = max(1, baseConfig.Connections/4)
	}

	if progressiveMaxConns > 0 {
		progressiveConfig.MaxConnections = progressiveMaxConns
	} else {
		progressiveConfig.MaxConnections = baseConfig.Connections * 2
	}

	if progressiveTestDuration > 0 {
		progressiveConfig.TestDuration = progressiveTestDuration
	} else {
		progressiveConfig.TestDuration = duration
	}

	if progressiveWarmupTime > 0 {
		progressiveConfig.WarmupDuration = progressiveWarmupTime
	} else {
		progressiveConfig.WarmupDuration = 30 * time.Second
	}

	if progressiveCooldownTime > 0 {
		progressiveConfig.CooldownDuration = progressiveCooldownTime
	} else {
		progressiveConfig.CooldownDuration = 10 * time.Second
	}

	enhanced.Workload.Progressive = progressiveConfig

	return enhanced, nil
}

// estimateTotalDuration estimates total test duration
func estimateTotalDuration(config *config.ProgressiveConfig) time.Duration {
	if config == nil {
		return time.Hour // Default timeout
	}

	bandDuration := config.TestDuration + config.WarmupDuration + config.CooldownDuration
	totalDuration := time.Duration(config.Bands) * bandDuration

	// Add 20% buffer
	return totalDuration + totalDuration/5
}

// printTestPlan prints the test execution plan
func printTestPlan(runner *progressive.ProgressiveRunner, logger logging.StormDBLogger) error {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("PROGRESSIVE SCALING TEST PLAN")
	fmt.Println(strings.Repeat("=", 80))

	// This would print the test plan details
	// Implementation depends on exposing plan information from runner

	fmt.Println(strings.Repeat("=", 80) + "\n")
	return nil
}

// createWorkloadRunner creates a workload runner function
func createWorkloadRunner(config *config.StormDBConfig, logger logging.StormDBLogger) func(context.Context, progressive.BandConfig) (*types.Metrics, error) {
	return func(ctx context.Context, bandConfig progressive.BandConfig) (*types.Metrics, error) {
		logger.Info("Starting workload for band",
			zap.Int("band_id", bandConfig.BandID),
			zap.Int("workers", bandConfig.Workers),
			zap.Int("connections", bandConfig.Connections),
			zap.Duration("duration", bandConfig.Duration),
		)

		// Create a mock metrics response for now
		// In production, this would integrate with the actual workload system
		metrics := &types.Metrics{
			TPS:            int64(bandConfig.Workers * 100), // Mock TPS
			QPS:            int64(bandConfig.Workers * 150), // Mock QPS
			Errors:         0,
			ErrorTypes:     make(map[string]int64),
			TransactionDur: make([]int64, 0),
		}

		// Simulate workload execution
		select {
		case <-time.After(bandConfig.Duration):
			// Normal completion
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		logger.Info("Completed workload for band",
			zap.Int("band_id", bandConfig.BandID),
			zap.Int64("tps", metrics.TPS),
			zap.Int64("qps", metrics.QPS),
		)

		return metrics, nil
	}
}

// outputResults outputs test results to various formats
func outputResults(results []progressive.BandResult, logger logging.StormDBLogger) error {
	// Print summary to console
	printResultsSummary(results)

	// Save to JSON file if specified
	if progressiveOutputFile != "" {
		if err := saveResultsToJSON(results, progressiveOutputFile); err != nil {
			return fmt.Errorf("failed to save results to JSON: %w", err)
		}
		logger.Info("Results saved to JSON", zap.String("file", progressiveOutputFile))
	}

	return nil
}

// printResultsSummary prints a summary of results to console
func printResultsSummary(results []progressive.BandResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("PROGRESSIVE TEST RESULTS SUMMARY")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf("Total Bands Executed: %d\n", len(results))

	if len(results) == 0 {
		fmt.Println("No results to display.")
		return
	}

	// Print band-by-band summary
	fmt.Printf("\n%-6s %-10s %-12s %-10s %-10s %-12s %-10s\n",
		"Band", "Workers", "Connections", "TPS", "QPS", "Latency P95", "Errors")
	fmt.Println(strings.Repeat("-", 80))

	maxTPS := 0.0
	optimalBand := 0

	for i, result := range results {
		if result.Metrics != nil {
			if result.Metrics.AvgTPS > maxTPS {
				maxTPS = result.Metrics.AvgTPS
				optimalBand = i + 1
			}

			fmt.Printf("%-6d %-10d %-12d %-10.1f %-10.1f %-12.1f %-10d\n",
				result.BandConfig.BandID,
				result.BandConfig.Workers,
				result.BandConfig.Connections,
				result.Metrics.AvgTPS,
				result.Metrics.AvgQPS,
				result.Metrics.LatencyP95,
				result.Metrics.TotalErrors,
			)
		}
	}

	fmt.Printf("\nOptimal Configuration: Band %d (%.1f TPS)\n", optimalBand, maxTPS)
	fmt.Println(strings.Repeat("=", 80) + "\n")
}

// saveResultsToJSON saves results to a JSON file
func saveResultsToJSON(results []progressive.BandResult, filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// For now, just create an empty file
	// In production, this would marshal results to JSON
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write placeholder JSON
	_, err = file.WriteString(`{"progressive_test_results": [], "status": "placeholder"}`)
	return err
}

// generateHTMLReport generates an HTML report
func generateHTMLReport(results []progressive.BandResult, filename string, logger logging.StormDBLogger) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create basic HTML report
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write basic HTML structure
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Progressive Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 10px; }
        .summary { margin: 20px 0; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Progressive Scaling Test Report</h1>
        <p>Generated: ` + time.Now().Format(time.RFC3339) + `</p>
    </div>
    
    <div class="summary">
        <h2>Test Summary</h2>
        <p>Total bands executed: ` + fmt.Sprintf("%d", len(results)) + `</p>
    </div>
    
    <h2>Detailed Results</h2>
    <p>Detailed results would be displayed here in a production implementation.</p>
</body>
</html>`

	_, err = file.WriteString(html)
	return err
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
