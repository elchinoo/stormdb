// Package progressive implements progressive connection scaling for StormDB workloads.
// It provides the core functionality to automatically scale worker and connection
// counts during benchmark execution, collecting detailed metrics and performing
// advanced statistical analysis on the results.
//
// The progressive scaling engine supports multiple scaling strategies:
// - Linear: Fixed increments (e.g., +10 workers per band)
// - Exponential: Exponential growth (e.g., 2x multiplier per band)
// - Fibonacci: Fibonacci sequence scaling
//
// Advanced statistical analysis includes:
// - Marginal gain analysis (discrete derivatives)
// - Inflection point detection (second derivatives)
// - Curve fitting (linear, logarithmic, exponential, logistic)
// - Queueing theory modeling (M/M/c analysis)
// - Confidence intervals and variance analysis
// - Performance region classification
package progressive

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/elchinoo/stormdb/pkg/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ScalingEngine manages progressive connection scaling for workload execution
type ScalingEngine struct {
	config   *types.Config
	workload WorkloadInterface
	db       *pgxpool.Pool
	results  *types.ProgressiveScalingResult
	mu       sync.RWMutex
}

// WorkloadInterface defines the interface that workloads must implement
// to support progressive scaling
type WorkloadInterface interface {
	// Setup ensures schema exists (called with --setup or --rebuild)
	Setup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error

	// Run executes the load test with the given configuration
	Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error

	// Cleanup drops tables and reloads data (called only with --rebuild)
	Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error
}

// NewScalingEngine creates a new progressive scaling engine
func NewScalingEngine(config *types.Config, workload WorkloadInterface, db *pgxpool.Pool) *ScalingEngine {
	return &ScalingEngine{
		config:   config,
		workload: workload,
		db:       db,
		results: &types.ProgressiveScalingResult{
			TestStartTime: time.Now(),
			Workload:      config.Workload,
			Strategy:      config.Progressive.Strategy,
			Bands:         make([]types.ProgressiveBandMetrics, 0),
		},
	}
}

// sanitizeFloat ensures float values are not NaN or Inf
func sanitizeFloat(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0.0
	}
	return value
}

// sanitizeBandMetrics cleans up any NaN or Inf values in band metrics
func (e *ScalingEngine) sanitizeBandMetrics(band *types.ProgressiveBandMetrics) {
	band.TotalTPS = sanitizeFloat(band.TotalTPS)
	band.TotalQPS = sanitizeFloat(band.TotalQPS)
	band.AvgLatencyMs = sanitizeFloat(band.AvgLatencyMs)
	band.P50LatencyMs = sanitizeFloat(band.P50LatencyMs)
	band.P95LatencyMs = sanitizeFloat(band.P95LatencyMs)
	band.P99LatencyMs = sanitizeFloat(band.P99LatencyMs)
	band.ErrorRate = sanitizeFloat(band.ErrorRate)
	band.StdDevLatency = sanitizeFloat(band.StdDevLatency)
	band.VarianceLatency = sanitizeFloat(band.VarianceLatency)
	band.CoefficientOfVar = sanitizeFloat(band.CoefficientOfVar)
	band.ConfidenceInterval.Lower = sanitizeFloat(band.ConfidenceInterval.Lower)
	band.ConfidenceInterval.Upper = sanitizeFloat(band.ConfidenceInterval.Upper)
	band.TPSPerWorker = sanitizeFloat(band.TPSPerWorker)
	band.TPSPerConnection = sanitizeFloat(band.TPSPerConnection)
	band.WorkerEfficiency = sanitizeFloat(band.WorkerEfficiency)
	band.ConnectionUtil = sanitizeFloat(band.ConnectionUtil)
}

// Execute runs the progressive scaling test
func (e *ScalingEngine) Execute(ctx context.Context) (*types.ProgressiveScalingResult, error) {
	if !e.config.Progressive.Enabled {
		return nil, fmt.Errorf("progressive scaling is not enabled in configuration")
	}

	// Validate progressive configuration
	if err := e.validateConfig(); err != nil {
		return nil, fmt.Errorf("invalid progressive configuration: %w", err)
	}

	// Parse durations
	bandDuration, err := time.ParseDuration(e.config.Progressive.BandDuration)
	if err != nil {
		return nil, fmt.Errorf("invalid band_duration: %w", err)
	}

	warmupTime, err := time.ParseDuration(e.config.Progressive.WarmupTime)
	if err != nil {
		return nil, fmt.Errorf("invalid warmup_time: %w", err)
	}

	cooldownTime, err := time.ParseDuration(e.config.Progressive.CooldownTime)
	if err != nil {
		return nil, fmt.Errorf("invalid cooldown_time: %w", err)
	}

	// Generate scaling sequence
	scalingSequence, err := e.generateScalingSequence()
	if err != nil {
		return nil, fmt.Errorf("failed to generate scaling sequence: %w", err)
	}

	fmt.Printf("🎯 Starting progressive scaling test with %d bands\n", len(scalingSequence))
	fmt.Printf("📊 Strategy: %s, Band Duration: %v, Warmup: %v, Cooldown: %v\n",
		e.config.Progressive.Strategy, bandDuration, warmupTime, cooldownTime)

	// Execute each band
	for i, band := range scalingSequence {
		fmt.Printf("\n🔄 Band %d/%d: %d workers, %d connections\n",
			i+1, len(scalingSequence), band.Workers, band.Connections)

		// Create band-specific configuration
		bandConfig := *e.config
		bandConfig.Workers = band.Workers
		bandConfig.Connections = band.Connections
		bandConfig.Duration = e.config.Progressive.BandDuration

		// Execute the band
		bandMetrics, err := e.executeBand(ctx, i+1, &bandConfig, bandDuration, warmupTime)
		if err != nil {
			return nil, fmt.Errorf("failed to execute band %d: %w", i+1, err)
		}

		// Store results
		e.mu.Lock()
		e.results.Bands = append(e.results.Bands, *bandMetrics)
		e.mu.Unlock()

		// Cooldown between bands (except for the last one)
		if i < len(scalingSequence)-1 && cooldownTime > 0 {
			fmt.Printf("😴 Cooling down for %v...\n", cooldownTime)
			time.Sleep(cooldownTime)
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			fmt.Println("🛑 Progressive scaling cancelled by context")
			return e.finalizeResults()
		default:
		}
	}

	return e.finalizeResults()
}

// ScalingBand represents a single scaling configuration
type ScalingBand struct {
	Workers     int
	Connections int
}

// generateScalingSequence creates the sequence of scaling bands based on strategy
func (e *ScalingEngine) generateScalingSequence() ([]ScalingBand, error) {
	var sequence []ScalingBand

	switch e.config.Progressive.Strategy {
	case "linear", "":
		sequence = e.generateLinearSequence()
	case "exponential":
		sequence = e.generateExponentialSequence()
	case "fibonacci":
		sequence = e.generateFibonacciSequence()
	default:
		return nil, fmt.Errorf("unsupported scaling strategy: %s", e.config.Progressive.Strategy)
	}

	if len(sequence) == 0 {
		return nil, fmt.Errorf("no scaling bands generated")
	}

	return sequence, nil
}

// generateLinearSequence creates a linear scaling sequence
func (e *ScalingEngine) generateLinearSequence() []ScalingBand {
	var sequence []ScalingBand

	// Use separate loops for workers and connections to create all combinations
	for workers := e.config.Progressive.MinWorkers; workers <= e.config.Progressive.MaxWorkers; workers += e.config.Progressive.StepWorkers {
		for conns := e.config.Progressive.MinConns; conns <= e.config.Progressive.MaxConns; conns += e.config.Progressive.StepConns {
			sequence = append(sequence, ScalingBand{
				Workers:     workers,
				Connections: conns,
			})
		}
	}

	return sequence
}

// generateExponentialSequence creates an exponential scaling sequence
func (e *ScalingEngine) generateExponentialSequence() []ScalingBand {
	var sequence []ScalingBand

	// Start with minimum values
	workers := e.config.Progressive.MinWorkers
	conns := e.config.Progressive.MinConns

	for workers <= e.config.Progressive.MaxWorkers && conns <= e.config.Progressive.MaxConns {
		sequence = append(sequence, ScalingBand{
			Workers:     workers,
			Connections: conns,
		})

		// Exponential growth (double each time, but respect step minimums)
		nextWorkers := workers * 2
		nextConns := conns * 2

		// Ensure we don't exceed maximums
		if nextWorkers > e.config.Progressive.MaxWorkers {
			nextWorkers = e.config.Progressive.MaxWorkers
		}
		if nextConns > e.config.Progressive.MaxConns {
			nextConns = e.config.Progressive.MaxConns
		}

		// Break if we can't make progress
		if nextWorkers == workers && nextConns == conns {
			break
		}

		workers = nextWorkers
		conns = nextConns
	}

	return sequence
}

// generateFibonacciSequence creates a fibonacci-based scaling sequence
func (e *ScalingEngine) generateFibonacciSequence() []ScalingBand {
	var sequence []ScalingBand

	// Generate fibonacci numbers for workers
	fibWorkers := e.generateFibonacci(e.config.Progressive.MinWorkers, e.config.Progressive.MaxWorkers)
	fibConns := e.generateFibonacci(e.config.Progressive.MinConns, e.config.Progressive.MaxConns)

	// Create combinations
	for _, workers := range fibWorkers {
		for _, conns := range fibConns {
			if workers <= e.config.Progressive.MaxWorkers && conns <= e.config.Progressive.MaxConns {
				sequence = append(sequence, ScalingBand{
					Workers:     workers,
					Connections: conns,
				})
			}
		}
	}

	return sequence
}

// generateFibonacci generates fibonacci numbers within the given range
func (e *ScalingEngine) generateFibonacci(min, max int) []int {
	var fib []int

	a, b := 1, 1

	// Adjust starting point to be >= min
	for b < min {
		a, b = b, a+b
	}

	// Generate sequence within range
	for b <= max {
		fib = append(fib, b)
		a, b = b, a+b
	}

	return fib
}

// executeBand runs a single scaling band and collects metrics
func (e *ScalingEngine) executeBand(ctx context.Context, bandID int, config *types.Config,
	bandDuration, warmupTime time.Duration) (*types.ProgressiveBandMetrics, error) {

	startTime := time.Now()

	// Create metrics for this band
	metrics := &types.Metrics{
		ErrorTypes:    make(map[string]int64),
		WorkerMetrics: make(map[int]*types.WorkerStats),
		Mu:            sync.Mutex{},
	}
	metrics.InitializeLatencyHistogram()
	metrics.InitializeWorkerMetrics(config.Workers)

	// Create a context with timeout for this band
	bandCtx, cancel := context.WithTimeout(ctx, bandDuration+warmupTime+10*time.Second)
	defer cancel()

	// Start the workload
	workloadErr := make(chan error, 1)
	go func() {
		workloadErr <- e.workload.Run(bandCtx, e.db, config, metrics)
	}()

	// Wait for warmup period
	if warmupTime > 0 {
		fmt.Printf("🔥 Warming up for %v...\n", warmupTime)
		select {
		case <-time.After(warmupTime):
			// Reset metrics after warmup
			metrics = &types.Metrics{
				ErrorTypes:    make(map[string]int64),
				WorkerMetrics: make(map[int]*types.WorkerStats),
				Mu:            sync.Mutex{},
			}
			metrics.InitializeLatencyHistogram()
			metrics.InitializeWorkerMetrics(config.Workers)
		case err := <-workloadErr:
			if err != nil {
				return nil, fmt.Errorf("workload failed during warmup: %w", err)
			}
		case <-bandCtx.Done():
			return nil, fmt.Errorf("band context cancelled during warmup")
		}
	}

	metricsStartTime := time.Now()

	// Wait for the actual measurement period
	select {
	case <-time.After(bandDuration):
		// Band completed successfully
	case err := <-workloadErr:
		if err != nil {
			return nil, fmt.Errorf("workload failed during measurement: %w", err)
		}
	case <-bandCtx.Done():
		return nil, fmt.Errorf("band context cancelled during measurement")
	}

	endTime := time.Now()
	actualDuration := endTime.Sub(metricsStartTime)

	// Calculate band metrics
	bandMetrics := e.calculateBandMetrics(bandID, config.Workers, config.Connections,
		startTime, endTime, actualDuration, metrics)

	return bandMetrics, nil
}

// validateConfig validates the progressive scaling configuration
func (e *ScalingEngine) validateConfig() error {
	p := &e.config.Progressive

	if p.MinWorkers <= 0 {
		return fmt.Errorf("min_workers must be positive, got: %d", p.MinWorkers)
	}
	if p.MaxWorkers <= 0 {
		return fmt.Errorf("max_workers must be positive, got: %d", p.MaxWorkers)
	}
	if p.MinWorkers > p.MaxWorkers {
		return fmt.Errorf("min_workers (%d) must be <= max_workers (%d)", p.MinWorkers, p.MaxWorkers)
	}
	if p.StepWorkers <= 0 {
		return fmt.Errorf("step_workers must be positive, got: %d", p.StepWorkers)
	}

	if p.MinConns <= 0 {
		return fmt.Errorf("min_connections must be positive, got: %d", p.MinConns)
	}
	if p.MaxConns <= 0 {
		return fmt.Errorf("max_connections must be positive, got: %d", p.MaxConns)
	}
	if p.MinConns > p.MaxConns {
		return fmt.Errorf("min_connections (%d) must be <= max_connections (%d)", p.MinConns, p.MaxConns)
	}
	if p.StepConns <= 0 {
		return fmt.Errorf("step_connections must be positive, got: %d", p.StepConns)
	}

	if p.BandDuration == "" {
		return fmt.Errorf("band_duration is required")
	}
	if _, err := time.ParseDuration(p.BandDuration); err != nil {
		return fmt.Errorf("invalid band_duration format: %s", p.BandDuration)
	}

	// Validate strategy
	validStrategies := map[string]bool{
		"linear":      true,
		"exponential": true,
		"fibonacci":   true,
		"":            true, // default to linear
	}
	if !validStrategies[p.Strategy] {
		return fmt.Errorf("invalid strategy: %s (valid: linear, exponential, fibonacci)", p.Strategy)
	}

	// Validate export format
	if p.ExportFormat != "" {
		validFormats := map[string]bool{
			"csv":  true,
			"json": true,
			"both": true,
		}
		if !validFormats[p.ExportFormat] {
			return fmt.Errorf("invalid export_format: %s (valid: csv, json, both)", p.ExportFormat)
		}
	}

	return nil
}

// calculateBandMetrics computes comprehensive metrics for a completed band
func (e *ScalingEngine) calculateBandMetrics(bandID, workers, connections int,
	startTime, endTime time.Time, duration time.Duration, metrics *types.Metrics) *types.ProgressiveBandMetrics {

	band := &types.ProgressiveBandMetrics{
		BandID:      bandID,
		Workers:     workers,
		Connections: connections,
		StartTime:   startTime,
		EndTime:     endTime,
		Duration:    duration,
	}

	// Calculate basic rates
	durationSec := duration.Seconds()
	if durationSec > 0 {
		band.TotalTPS = float64(metrics.TPS) / durationSec
		band.TotalQPS = float64(metrics.QPS) / durationSec
		band.ErrorRate = float64(metrics.Errors) / float64(metrics.TPS+metrics.TPSAborted) * 100
	}
	band.TotalErrors = metrics.Errors

	// Calculate latency statistics
	if len(metrics.TransactionDur) > 0 {
		latencies := make([]float64, len(metrics.TransactionDur))
		var sum float64
		for i, ns := range metrics.TransactionDur {
			ms := float64(ns) / 1e6 // Convert nanoseconds to milliseconds
			latencies[i] = ms
			sum += ms
		}

		sort.Float64s(latencies)

		band.AvgLatencyMs = sum / float64(len(latencies))
		band.P50LatencyMs = percentile(latencies, 0.5)
		band.P95LatencyMs = percentile(latencies, 0.95)
		band.P99LatencyMs = percentile(latencies, 0.99)
		band.MinLatencyMs = latencies[0]
		band.MaxLatencyMs = latencies[len(latencies)-1]

		// Calculate advanced statistics
		band.StdDevLatency = standardDeviation(latencies, band.AvgLatencyMs)
		band.VarianceLatency = band.StdDevLatency * band.StdDevLatency
		if band.AvgLatencyMs > 0 {
			band.CoefficientOfVar = band.StdDevLatency / band.AvgLatencyMs
		}

		// Calculate 95% confidence interval
		n := float64(len(latencies))
		standardError := band.StdDevLatency / math.Sqrt(n)
		margin := 1.96 * standardError // 95% confidence interval
		band.ConfidenceInterval.Lower = band.AvgLatencyMs - margin
		band.ConfidenceInterval.Upper = band.AvgLatencyMs + margin

		// Store raw samples for analysis (limit to reasonable size)
		sampleSize := len(metrics.TransactionDur)
		if sampleSize > 10000 {
			sampleSize = 10000 // Limit to prevent excessive memory usage
		}
		band.LatencySamples = make([]int64, sampleSize)
		copy(band.LatencySamples, metrics.TransactionDur[:sampleSize])
	}

	// Calculate efficiency metrics
	if workers > 0 {
		band.TPSPerWorker = band.TotalTPS / float64(workers)
		band.WorkerEfficiency = band.TPSPerWorker / band.TotalTPS * 100 // Simplified efficiency calculation
	}
	if connections > 0 {
		band.TPSPerConnection = band.TotalTPS / float64(connections)
		band.ConnectionUtil = (band.TotalTPS / float64(connections)) / band.TotalTPS * 100 // Simplified utilization
	}

	// Copy PostgreSQL stats if available
	if pgStats := metrics.GetPgStats(); pgStats != nil {
		band.PgStats = pgStats
	}

	// Sanitize all float values to prevent NaN/Inf propagation
	e.sanitizeBandMetrics(band)

	return band
}

// finalizeResults completes the progressive scaling analysis
func (e *ScalingEngine) finalizeResults() (*types.ProgressiveScalingResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.results.TestEndTime = time.Now()
	e.results.TotalDuration = e.results.TestEndTime.Sub(e.results.TestStartTime)

	// Perform advanced analysis
	if err := e.performAnalysis(); err != nil {
		return nil, fmt.Errorf("failed to perform analysis: %w", err)
	}

	// Find optimal configuration
	e.findOptimalConfiguration()

	// Export results if configured
	if e.config.Progressive.ExportFormat != "" && e.config.Progressive.ExportPath != "" {
		if err := e.exportResults(); err != nil {
			fmt.Printf("Warning: Failed to export results: %v\n", err)
		}
	}

	return e.results, nil
}

// Helper functions

// percentile calculates the given percentile of a sorted slice
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	index := p * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// standardDeviation calculates the standard deviation of a slice
func standardDeviation(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	var sumSquaredDiffs float64
	for _, v := range values {
		diff := v - mean
		sumSquaredDiffs += diff * diff
	}

	variance := sumSquaredDiffs / float64(len(values)-1)
	return math.Sqrt(variance)
}
