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
	"sync/atomic"
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

	// Parse durations - handle both v0.2 and legacy formats
	var bandDuration, warmupTime, cooldownTime time.Duration
	var err error

	// Determine which format to use
	if e.config.Progressive.TestDuration != "" {
		// v0.2 format
		bandDuration, err = time.ParseDuration(e.config.Progressive.TestDuration)
		if err != nil {
			return nil, fmt.Errorf("invalid test_duration: %w", err)
		}

		warmupTime, err = time.ParseDuration(e.config.Progressive.WarmupDuration)
		if err != nil {
			return nil, fmt.Errorf("invalid warmup_duration: %w", err)
		}

		cooldownTime, err = time.ParseDuration(e.config.Progressive.CooldownDuration)
		if err != nil {
			return nil, fmt.Errorf("invalid cooldown_duration: %w", err)
		}
	} else {
		// Legacy format
		bandDuration, err = time.ParseDuration(e.config.Progressive.BandDuration)
		if err != nil {
			return nil, fmt.Errorf("invalid band_duration: %w", err)
		}

		warmupTime, err = time.ParseDuration(e.config.Progressive.WarmupTime)
		if err != nil {
			return nil, fmt.Errorf("invalid warmup_time: %w", err)
		}

		cooldownTime, err = time.ParseDuration(e.config.Progressive.CooldownTime)
		if err != nil {
			return nil, fmt.Errorf("invalid cooldown_time: %w", err)
		}
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

		// Set duration based on format being used
		if e.config.Progressive.TestDuration != "" {
			bandConfig.Duration = e.config.Progressive.TestDuration
		} else {
			bandConfig.Duration = e.config.Progressive.BandDuration
		}

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
	case "balanced":
		sequence = e.generateBalancedSequence()
	case "synchronized":
		sequence = e.generateSynchronizedSequence()
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

	// Use bands configuration to limit the total number of bands generated
	maxBands := e.config.Progressive.Bands
	if maxBands <= 0 {
		maxBands = 64 // Default limit to prevent excessive band generation
	}

	// Use StepWorkers and StepConns if available, otherwise calculate based on bands
	stepWorkers := e.config.Progressive.StepWorkers
	stepConns := e.config.Progressive.StepConns

	if stepWorkers <= 0 || stepConns <= 0 {
		// Calculate steps to generate approximately maxBands combinations
		workerRange := e.config.Progressive.MaxWorkers - e.config.Progressive.MinWorkers
		connRange := e.config.Progressive.MaxConns - e.config.Progressive.MinConns

		// Try to balance worker and connection steps to get reasonable band count
		targetWorkerSteps := int(math.Sqrt(float64(maxBands)))
		targetConnSteps := maxBands / targetWorkerSteps

		if targetWorkerSteps > 0 && workerRange > 0 {
			stepWorkers = workerRange / targetWorkerSteps
			if stepWorkers <= 0 {
				stepWorkers = 1
			}
		} else {
			stepWorkers = 1
		}

		if targetConnSteps > 0 && connRange > 0 {
			stepConns = connRange / targetConnSteps
			if stepConns <= 0 {
				stepConns = 1
			}
		} else {
			stepConns = 1
		}
	}

	bandCount := 0
	// Use separate loops for workers and connections to create all combinations
	for workers := e.config.Progressive.MinWorkers; workers <= e.config.Progressive.MaxWorkers && bandCount < maxBands; workers += stepWorkers {
		for conns := e.config.Progressive.MinConns; conns <= e.config.Progressive.MaxConns && bandCount < maxBands; conns += stepConns {
			sequence = append(sequence, ScalingBand{
				Workers:     workers,
				Connections: conns,
			})
			bandCount++
		}
	}

	return sequence
}

// generateBalancedSequence creates a balanced scaling sequence where workers and connections scale together
func (e *ScalingEngine) generateBalancedSequence() []ScalingBand {
	var sequence []ScalingBand

	// Use bands configuration to determine number of steps
	bands := e.config.Progressive.Bands
	if bands <= 0 {
		bands = 8 // Default to 8 bands if not specified
	}

	// For balanced scaling, we scale workers and connections proportionally
	workerRange := e.config.Progressive.MaxWorkers - e.config.Progressive.MinWorkers
	connRange := e.config.Progressive.MaxConns - e.config.Progressive.MinConns

	// Calculate step sizes based on number of bands
	workerStep := float64(workerRange) / float64(bands-1)
	connStep := float64(connRange) / float64(bands-1)

	for i := 0; i < bands; i++ {
		workers := e.config.Progressive.MinWorkers + int(float64(i)*workerStep)
		conns := e.config.Progressive.MinConns + int(float64(i)*connStep)

		// Ensure we don't exceed maximums
		if workers > e.config.Progressive.MaxWorkers {
			workers = e.config.Progressive.MaxWorkers
		}
		if conns > e.config.Progressive.MaxConns {
			conns = e.config.Progressive.MaxConns
		}

		sequence = append(sequence, ScalingBand{
			Workers:     workers,
			Connections: conns,
		})
	}

	return sequence
}

// generateSynchronizedSequence creates a sequence where workers equals connections
func (e *ScalingEngine) generateSynchronizedSequence() []ScalingBand {
	var sequence []ScalingBand

	// Use workers as the primary scaling factor, set connections = workers
	stepWorkers := e.config.Progressive.StepWorkers
	if stepWorkers <= 0 {
		stepWorkers = 1 // Default step
	}

	for workers := e.config.Progressive.MinWorkers; workers <= e.config.Progressive.MaxWorkers; workers += stepWorkers {
		// Set connections equal to workers, but respect min/max constraints
		conns := workers
		if conns < e.config.Progressive.MinConns {
			conns = e.config.Progressive.MinConns
		}
		if conns > e.config.Progressive.MaxConns {
			conns = e.config.Progressive.MaxConns
		}

		sequence = append(sequence, ScalingBand{
			Workers:     workers,
			Connections: conns,
		})
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

// RunPhaseSample represents a metrics sample collected during the run phase
type RunPhaseSample struct {
	Timestamp  time.Time
	TPS        float64
	QPS        float64
	ErrorRate  float64
	P50Latency float64
	P95Latency float64
	P99Latency float64
}

// collectRunPhaseMetrics collects periodic samples during the run phase only
func (e *ScalingEngine) collectRunPhaseMetrics(ctx context.Context, metrics *types.Metrics,
	runDuration time.Duration, sampleInterval time.Duration) []RunPhaseSample {

	var samples []RunPhaseSample

	runStartTime := time.Now()
	var lastTPS, lastQPS, lastErrors int64

	// Initialize baseline counts at run-phase start
	lastTPS = atomic.LoadInt64(&metrics.TPS)
	lastQPS = atomic.LoadInt64(&metrics.QPS)
	lastErrors = atomic.LoadInt64(&metrics.Errors)

	// Calculate the number of samples we should collect
	// For a 20s run with 5s intervals, we want samples at: 5s, 10s, 15s, 20s (4 total)
	numSamples := int(runDuration / sampleInterval)

	for i := 1; i <= numSamples; i++ {
		// Wait for the next sample time
		time.Sleep(sampleInterval)

		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return samples
		default:
		}

		currentTime := time.Now()

		// Capture current totals
		currentTPS := atomic.LoadInt64(&metrics.TPS)
		currentQPS := atomic.LoadInt64(&metrics.QPS)
		currentErrors := atomic.LoadInt64(&metrics.Errors)

		// Calculate rates over the sample interval
		intervalSec := sampleInterval.Seconds()
		tpsRate := float64(currentTPS-lastTPS) / intervalSec
		qpsRate := float64(currentQPS-lastQPS) / intervalSec
		errorRate := float64(currentErrors-lastErrors) / intervalSec

		// Calculate latency percentiles from current samples
		var p50, p95, p99 float64
		metrics.Mu.Lock()
		if len(metrics.TransactionDur) > 0 {
			latencies := make([]float64, len(metrics.TransactionDur))
			for i, ns := range metrics.TransactionDur {
				latencies[i] = float64(ns) / 1e6 // ns to ms
			}
			sort.Float64s(latencies)
			p50 = percentile(latencies, 0.5)
			p95 = percentile(latencies, 0.95)
			p99 = percentile(latencies, 0.99)
		}
		metrics.Mu.Unlock()

		// Store sample with elapsed time from run start for debugging
		elapsed := currentTime.Sub(runStartTime)
		sample := RunPhaseSample{
			Timestamp:  currentTime,
			TPS:        tpsRate,
			QPS:        qpsRate,
			ErrorRate:  errorRate,
			P50Latency: p50,
			P95Latency: p95,
			P99Latency: p99,
		}
		samples = append(samples, sample)

		fmt.Printf("📊 Run-phase sample %d/%d at %v: TPS=%.1f, P50=%.2fms, P95=%.2fms, P99=%.2fms\n",
			i, numSamples, elapsed.Round(time.Second), tpsRate, p50, p95, p99)

		// Update last values for next iteration
		lastTPS = currentTPS
		lastQPS = currentQPS
		lastErrors = currentErrors
	}

	return samples
} // executeBand runs a single scaling band and collects metrics
func (e *ScalingEngine) executeBand(ctx context.Context, bandID int, config *types.Config,
	bandDuration, warmupTime time.Duration) (*types.ProgressiveBandMetrics, error) {

	startTime := time.Now()

	// Create a band-specific connection pool with the exact number of connections needed
	dsn := fmt.Sprintf(
		"user=%s password=%s host=%s port=%d dbname=%s sslmode=%s pool_max_conns=%d pool_min_conns=%d pool_max_conn_lifetime=1h pool_max_conn_idle_time=30m pool_health_check_period=1m connect_timeout=10",
		config.Database.Username, config.Database.Password,
		config.Database.Host, config.Database.Port,
		config.Database.Dbname, config.Database.Sslmode,
		config.Connections, config.Connections/2, // min connections = half of max
	)

	bandPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create band-specific connection pool: %w", err)
	}
	defer bandPool.Close()

	// Test the band connection pool
	if err := bandPool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database with band pool: %w", err)
	}

	// Create metrics for this band with memory limits
	metrics := &types.Metrics{
		ErrorTypes:    make(map[string]int64),
		WorkerMetrics: make(map[int]*types.WorkerStats),
		Mu:            sync.Mutex{},
	}

	// Apply memory limits from progressive configuration
	if e.config.Progressive.MaxLatencySamples > 0 {
		metrics.MaxLatencySamples = e.config.Progressive.MaxLatencySamples
		fmt.Printf("💾 Memory limit: %d latency samples per band\n", metrics.MaxLatencySamples)
	} else {
		// Default memory-safe limit for 8GB systems: 50,000 samples = ~400KB per worker
		defaultLimit := 50000
		metrics.MaxLatencySamples = defaultLimit
		fmt.Printf("💾 Using default memory limit: %d latency samples per band\n", defaultLimit)
	}

	metrics.InitializeLatencyHistogram()
	metrics.InitializeWorkerMetrics(config.Workers)

	// Create a context with timeout for this band (warmup + run phase + buffer)
	totalBandTime := warmupTime + bandDuration + 10*time.Second
	bandCtx, cancel := context.WithTimeout(ctx, totalBandTime)
	defer cancel()

	// Start the workload with the band-specific pool
	workloadErr := make(chan error, 1)
	go func() {
		workloadErr <- e.workload.Run(bandCtx, bandPool, config, metrics)
	}()

	// Wait for warmup period
	var runPhaseMetrics []RunPhaseSample
	sampleInterval := 5 * time.Second // Collect samples every 5 seconds

	if warmupTime > 0 {
		fmt.Printf("🔥 Warming up for %v...\n", warmupTime)
		select {
		case <-time.After(warmupTime):
			// Warmup complete, now collect run-phase samples
			fmt.Printf("📊 Collecting run-phase metrics for %v...\n", bandDuration)

			// Start run-phase metrics collection - this will run for the full band duration
			// Add a small buffer to ensure the final sample can be collected
			runCtx, runCancel := context.WithTimeout(bandCtx, bandDuration+1*time.Second)
			defer runCancel()

			runPhaseMetrics = e.collectRunPhaseMetrics(runCtx, metrics, bandDuration, sampleInterval)

		case err := <-workloadErr:
			if err != nil {
				return nil, fmt.Errorf("workload failed during warmup: %w", err)
			}
		case <-bandCtx.Done():
			return nil, fmt.Errorf("band context cancelled during warmup")
		}
	} else {
		// No warmup, start collecting run-phase metrics immediately
		fmt.Printf("📊 Collecting run-phase metrics for %v...\n", bandDuration)

		runCtx, runCancel := context.WithTimeout(bandCtx, bandDuration+1*time.Second)
		defer runCancel()

		runPhaseMetrics = e.collectRunPhaseMetrics(runCtx, metrics, bandDuration, sampleInterval)
	}

	// Run-phase metrics collection is complete, workload may still be running
	// Wait for workload to complete gracefully
	select {
	case err := <-workloadErr:
		if err != nil {
			return nil, fmt.Errorf("workload failed: %w", err)
		}
	case <-time.After(5 * time.Second): // Give workload 5 seconds to complete gracefully
		// Continue with analysis
	case <-bandCtx.Done():
		// Context timeout, but we have our metrics
	}

	endTime := time.Now()
	actualDuration := endTime.Sub(startTime)

	// Calculate band metrics from run-phase samples
	bandMetrics := e.calculateBandMetricsFromSamples(bandID, config.Workers, config.Connections,
		startTime, endTime, actualDuration, runPhaseMetrics)

	return bandMetrics, nil
}

// calculateBandMetricsFromSamples computes comprehensive metrics from run-phase samples
func (e *ScalingEngine) calculateBandMetricsFromSamples(bandID, workers, connections int,
	startTime, endTime time.Time, duration time.Duration, samples []RunPhaseSample) *types.ProgressiveBandMetrics {

	band := &types.ProgressiveBandMetrics{
		BandID:      bandID,
		Workers:     workers,
		Connections: connections,
		StartTime:   startTime,
		EndTime:     endTime,
		Duration:    duration,
	}

	if len(samples) == 0 {
		// No samples collected, return zero metrics
		e.sanitizeBandMetrics(band)
		return band
	}

	// Extract TPS values from run-phase samples
	tpsValues := make([]float64, len(samples))
	qpsValues := make([]float64, len(samples))
	p50Values := make([]float64, len(samples))
	p95Values := make([]float64, len(samples))
	p99Values := make([]float64, len(samples))

	for i, sample := range samples {
		tpsValues[i] = sample.TPS
		qpsValues[i] = sample.QPS
		p50Values[i] = sample.P50Latency
		p95Values[i] = sample.P95Latency
		p99Values[i] = sample.P99Latency
	}

	// Calculate TPS statistics
	band.TotalTPS = average(tpsValues)

	// Store TPS samples for further analysis
	band.TPSSamples = make([]float64, len(tpsValues))
	copy(band.TPSSamples, tpsValues)

	// Calculate QPS statistics
	band.TotalQPS = average(qpsValues)

	// Calculate latency statistics from samples
	band.AvgLatencyMs = average(p50Values) // Use P50 as representative of average
	band.P50LatencyMs = average(p50Values)
	band.P95LatencyMs = average(p95Values)
	band.P99LatencyMs = average(p99Values)

	// Calculate latency standard deviation
	band.StdDevLatency = standardDeviation(p50Values, band.AvgLatencyMs)
	band.VarianceLatency = band.StdDevLatency * band.StdDevLatency
	if band.AvgLatencyMs > 0 {
		band.CoefficientOfVar = band.StdDevLatency / band.AvgLatencyMs
	}

	// Calculate 95% confidence interval for latency
	n := float64(len(samples))
	standardError := band.StdDevLatency / math.Sqrt(n)
	margin := 1.96 * standardError // 95% confidence interval
	band.ConfidenceInterval.Lower = band.AvgLatencyMs - margin
	band.ConfidenceInterval.Upper = band.AvgLatencyMs + margin

	// Calculate efficiency metrics
	if workers > 0 {
		band.TPSPerWorker = band.TotalTPS / float64(workers)
		band.WorkerEfficiency = (band.TPSPerWorker / band.TotalTPS) * 100
	}
	if connections > 0 {
		band.TPSPerConnection = band.TotalTPS / float64(connections)
		band.ConnectionUtil = (band.TPSPerConnection / band.TotalTPS) * 100
	}

	// Error rate (average from samples)
	if len(samples) > 0 {
		errorSum := 0.0
		for _, sample := range samples {
			errorSum += sample.ErrorRate
		}
		band.ErrorRate = errorSum / float64(len(samples))
	}

	// Store TPS standard deviation and coefficient of variation
	// Create a custom field to store TPS statistics for the report
	// We'll add these to the band metrics structure

	// Note: We need to modify the report generation to use these proper statistics
	// For now, let's store them in a way that the report can access

	// Sanitize all float values to prevent NaN/Inf propagation
	e.sanitizeBandMetrics(band)

	return band
}

// average calculates the arithmetic mean of a slice of float64 values
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// validateConfig validates the progressive scaling configuration
func (e *ScalingEngine) validateConfig() error {
	p := &e.config.Progressive

	// Basic validation
	if p.MinWorkers <= 0 {
		return fmt.Errorf("min_workers must be positive, got: %d", p.MinWorkers)
	}
	if p.MaxWorkers <= 0 {
		return fmt.Errorf("max_workers must be positive, got: %d", p.MaxWorkers)
	}
	if p.MinWorkers > p.MaxWorkers {
		return fmt.Errorf("min_workers (%d) must be <= max_workers (%d)", p.MinWorkers, p.MaxWorkers)
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

	// Check if using v0.2 format (preferred) or legacy format
	usingV2Format := p.Bands > 0 || p.TestDuration != ""

	if usingV2Format {
		// v0.2 format validation
		if p.Bands > 0 {
			if p.Bands < 3 {
				return fmt.Errorf("bands must be at least 3 for meaningful analysis, got: %d", p.Bands)
			}
			if p.Bands > 25 {
				return fmt.Errorf("bands cannot exceed 25 for practical reasons, got: %d", p.Bands)
			}
		}

		// Use defaults if not specified
		if p.TestDuration == "" {
			p.TestDuration = "30m"
		}
		if p.WarmupDuration == "" {
			p.WarmupDuration = "60s"
		}
		if p.CooldownDuration == "" {
			p.CooldownDuration = "30s"
		}
		if p.Bands == 0 {
			p.Bands = 5
		}
	} else {
		// Legacy format validation
		if p.StepWorkers <= 0 {
			return fmt.Errorf("step_workers must be positive, got: %d", p.StepWorkers)
		}
		if p.StepConns <= 0 {
			return fmt.Errorf("step_connections must be positive, got: %d", p.StepConns)
		}
		if p.BandDuration == "" {
			return fmt.Errorf("band_duration is required")
		}
	}

	// Validate strategy
	validStrategies := map[string]bool{
		"linear":       true,
		"balanced":     true,
		"synchronized": true,
		"exponential":  true,
		"fibonacci":    true,
		"":             true, // default to linear
	}
	if !validStrategies[p.Strategy] {
		return fmt.Errorf("invalid strategy: %s (valid: linear, balanced, synchronized, exponential, fibonacci)", p.Strategy)
	}

	return nil
}

// calculateBandMetrics computes comprehensive metrics for a completed band
// TODO: This function will be used in future analysis features
//nolint:unused
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

	// Generate comprehensive terminal report
	e.generateProgressiveReport()

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
