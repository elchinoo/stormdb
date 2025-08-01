package progressive

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/elchinoo/stormdb/internal/config"
	"github.com/elchinoo/stormdb/internal/logging"
	"github.com/elchinoo/stormdb/pkg/types"
	"go.uber.org/zap"
)

// Strategy defines progressive scaling strategies
type Strategy string

const (
	StrategyLinear     Strategy = "linear"
	StrategyExponential Strategy = "exponential"
	StrategyFibonacci  Strategy = "fibonacci"
	StrategyCustom     Strategy = "custom"
)

// ProgressiveRunner manages progressive scaling test execution
type ProgressiveRunner struct {
	config    *config.ProgressiveConfig
	logger    logging.StormDBLogger
	bands     []BandConfig
	results   []BandResult
	analytics *AnalyticsEngine
	
	// State management
	currentBand int
	running     bool
	mutex       sync.RWMutex
}

// BandConfig defines configuration for a single test band
type BandConfig struct {
	BandID      int           `json:"band_id"`
	Workers     int           `json:"workers"`
	Connections int           `json:"connections"`
	Duration    time.Duration `json:"duration"`
	WarmupTime  time.Duration `json:"warmup_time"`
	CooldownTime time.Duration `json:"cooldown_time"`
}

// BandResult contains results from a single test band
type BandResult struct {
	BandConfig BandConfig                `json:"band_config"`
	StartTime  time.Time                 `json:"start_time"`
	EndTime    time.Time                 `json:"end_time"`
	Metrics    *BandMetrics              `json:"metrics"`
	Samples    []MetricSample            `json:"samples"`
	Health     BandHealth                `json:"health"`
	Errors     []string                  `json:"errors,omitempty"`
}

// BandMetrics contains aggregated metrics for a test band
type BandMetrics struct {
	// Throughput metrics
	TotalTransactions int64   `json:"total_transactions"`
	TotalQueries      int64   `json:"total_queries"`
	AvgTPS            float64 `json:"avg_tps"`
	AvgQPS            float64 `json:"avg_qps"`
	
	// Latency metrics (milliseconds)
	LatencyP50  float64 `json:"latency_p50_ms"`
	LatencyP90  float64 `json:"latency_p90_ms"`
	LatencyP95  float64 `json:"latency_p95_ms"`
	LatencyP99  float64 `json:"latency_p99_ms"`
	LatencyMean float64 `json:"latency_mean_ms"`
	LatencyStdDev float64 `json:"latency_stddev_ms"`
	
	// Variability metrics
	CoefficientOfVariation float64 `json:"coefficient_of_variation"`
	ConfidenceInterval95   ConfidenceInterval `json:"confidence_interval_95"`
	
	// Error metrics
	TotalErrors   int64   `json:"total_errors"`
	ErrorRate     float64 `json:"error_rate"`
	ErrorTypes    map[string]int64 `json:"error_types"`
}

// ConfidenceInterval represents a statistical confidence interval
type ConfidenceInterval struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
	Mean  float64 `json:"mean"`
}

// MetricSample represents a single metrics sample during the test
type MetricSample struct {
	Timestamp    time.Time `json:"timestamp"`
	ElapsedTime  time.Duration `json:"elapsed_time"`
	TPS          float64   `json:"tps"`
	QPS          float64   `json:"qps"`
	LatencyP95   float64   `json:"latency_p95_ms"`
	ErrorCount   int64     `json:"error_count"`
	ActiveConns  int       `json:"active_connections"`
}

// BandHealth tracks health status during band execution
type BandHealth struct {
	HealthyDuration   time.Duration `json:"healthy_duration"`
	UnhealthyDuration time.Duration `json:"unhealthy_duration"`
	HealthScore       float64       `json:"health_score"` // 0.0 to 1.0
	MaxErrorRate      float64       `json:"max_error_rate"`
	AvgErrorRate      float64       `json:"avg_error_rate"`
}

// NewProgressiveRunner creates a new progressive test runner
func NewProgressiveRunner(config *config.ProgressiveConfig, logger logging.StormDBLogger) (*ProgressiveRunner, error) {
	if config == nil {
		return nil, fmt.Errorf("progressive config cannot be nil")
	}
	if logger == nil {
		logger = logging.NewDefaultLogger()
	}

	runner := &ProgressiveRunner{
		config:    config,
		logger:    logger.With(zap.String("component", "progressive_runner")),
		analytics: NewAnalyticsEngine(logger),
	}

	// Generate band configurations
	if err := runner.generateBands(); err != nil {
		return nil, fmt.Errorf("failed to generate bands: %w", err)
	}

	return runner, nil
}

// generateBands creates band configurations based on the strategy
func (pr *ProgressiveRunner) generateBands() error {
	pr.bands = make([]BandConfig, pr.config.Bands)
	
	workerSteps := pr.generateSteps(pr.config.MinWorkers, pr.config.MaxWorkers, pr.config.Bands, Strategy(pr.config.Strategy))
	connSteps := pr.generateSteps(pr.config.MinConnections, pr.config.MaxConnections, pr.config.Bands, Strategy(pr.config.Strategy))
	
	if len(workerSteps) != pr.config.Bands || len(connSteps) != pr.config.Bands {
		return fmt.Errorf("failed to generate correct number of steps: workers=%d, connections=%d, expected=%d",
			len(workerSteps), len(connSteps), pr.config.Bands)
	}

	for i := 0; i < pr.config.Bands; i++ {
		pr.bands[i] = BandConfig{
			BandID:       i + 1,
			Workers:      workerSteps[i],
			Connections:  connSteps[i],
			Duration:     pr.config.TestDuration,
			WarmupTime:   pr.config.WarmupDuration,
			CooldownTime: pr.config.CooldownDuration,
		}
	}

	pr.logger.Info("Generated progressive test bands",
		zap.Int("total_bands", len(pr.bands)),
		zap.String("strategy", string(pr.config.Strategy)),
		zap.Any("worker_progression", workerSteps),
		zap.Any("connection_progression", connSteps),
	)

	return nil
}

// generateSteps creates progression steps based on the specified strategy
func (pr *ProgressiveRunner) generateSteps(min, max, bands int, strategy Strategy) []int {
	if bands <= 1 {
		return []int{max}
	}

	switch strategy {
	case StrategyLinear:
		return pr.generateLinearSteps(min, max, bands)
	case StrategyExponential:
		return pr.generateExponentialSteps(min, max, bands)
	case StrategyFibonacci:
		return pr.generateFibonacciSteps(min, max, bands)
	default:
		pr.logger.Warn("Unknown strategy, using linear", zap.String("strategy", string(strategy)))
		return pr.generateLinearSteps(min, max, bands)
	}
}

// generateLinearSteps creates linearly spaced steps
func (pr *ProgressiveRunner) generateLinearSteps(min, max, bands int) []int {
	steps := make([]int, bands)
	if bands == 1 {
		steps[0] = max
		return steps
	}
	
	step := float64(max-min) / float64(bands-1)
	for i := 0; i < bands; i++ {
		steps[i] = min + int(float64(i)*step)
	}
	steps[bands-1] = max // Ensure last step is exactly max
	return steps
}

// generateExponentialSteps creates exponentially spaced steps
func (pr *ProgressiveRunner) generateExponentialSteps(min, max, bands int) []int {
	steps := make([]int, bands)
	if bands == 1 {
		steps[0] = max
		return steps
	}
	
	// Use exponential growth: value = min * (max/min)^(i/(bands-1))
	ratio := float64(max) / float64(min)
	for i := 0; i < bands; i++ {
		exponent := float64(i) / float64(bands-1)
		value := float64(min) * math.Pow(ratio, exponent)
		steps[i] = int(math.Round(value))
	}
	steps[bands-1] = max // Ensure last step is exactly max
	return steps
}

// generateFibonacciSteps creates Fibonacci-based progression
func (pr *ProgressiveRunner) generateFibonacciSteps(min, max, bands int) []int {
	steps := make([]int, bands)
	if bands == 1 {
		steps[0] = max
		return steps
	}
	
	// Generate Fibonacci sequence
	fib := make([]int, bands)
	if bands >= 1 {
		fib[0] = 1
	}
	if bands >= 2 {
		fib[1] = 1
	}
	for i := 2; i < bands; i++ {
		fib[i] = fib[i-1] + fib[i-2]
	}
	
	// Scale Fibonacci to fit min-max range
	fibMax := fib[bands-1]
	for i := 0; i < bands; i++ {
		scaled := float64(fib[i]) / float64(fibMax)
		value := float64(min) + scaled*float64(max-min)
		steps[i] = int(math.Round(value))
	}
	
	return steps
}

// Run executes the progressive test
func (pr *ProgressiveRunner) Run(ctx context.Context, workloadRunner func(context.Context, BandConfig) (*types.Metrics, error)) error {
	pr.mutex.Lock()
	if pr.running {
		pr.mutex.Unlock()
		return fmt.Errorf("progressive test is already running")
	}
	pr.running = true
	pr.currentBand = 0
	pr.results = make([]BandResult, 0, len(pr.bands))
	pr.mutex.Unlock()

	defer func() {
		pr.mutex.Lock()
		pr.running = false
		pr.mutex.Unlock()
	}()

	pr.logger.Info("Starting progressive test execution",
		zap.Int("total_bands", len(pr.bands)),
		zap.Duration("estimated_duration", pr.estimateTotalDuration()),
	)

	for i, band := range pr.bands {
		pr.mutex.Lock()
		pr.currentBand = i + 1
		pr.mutex.Unlock()

		pr.logger.Info("Starting test band",
			zap.Int("band_id", band.BandID),
			zap.Int("workers", band.Workers),
			zap.Int("connections", band.Connections),
			zap.Duration("duration", band.Duration),
		)

		result := pr.runBand(ctx, band, workloadRunner)
		if len(result.Errors) > 0 {
			pr.logger.Error("Band execution had errors", nil,
				zap.Int("band_id", band.BandID),
				zap.Strings("errors", result.Errors),
			)
		}

		pr.results = append(pr.results, result)

		// Check for early termination conditions
		if pr.shouldTerminateEarly(result) {
			pr.logger.Warn("Early termination triggered",
				zap.Int("completed_bands", len(pr.results)),
				zap.Int("total_bands", len(pr.bands)),
				zap.String("reason", "excessive_errors_or_degradation"),
			)
			break
		}

		// Optional cooldown between bands
		if i < len(pr.bands)-1 && band.CooldownTime > 0 {
			pr.logger.Debug("Cooldown between bands",
				zap.Duration("cooldown_time", band.CooldownTime),
			)
			time.Sleep(band.CooldownTime)
		}
	}

	// Perform analytics
	if pr.config.EnableAnalysis && len(pr.results) > 1 {
		if err := pr.performAnalysis(); err != nil {
			pr.logger.Error("Failed to perform analysis", err)
		}
	}

	pr.logger.Info("Progressive test completed",
		zap.Int("completed_bands", len(pr.results)),
		zap.Int("total_bands", len(pr.bands)),
	)

	return nil
}

// runBand executes a single test band
func (pr *ProgressiveRunner) runBand(ctx context.Context, band BandConfig, workloadRunner func(context.Context, BandConfig) (*types.Metrics, error)) BandResult {
	result := BandResult{
		BandConfig: band,
		StartTime:  time.Now(),
		Samples:    make([]MetricSample, 0),
	}

	// Create band context with timeout
	bandCtx, cancel := context.WithTimeout(ctx, band.Duration+band.WarmupTime+band.CooldownTime+30*time.Second)
	defer cancel()

	// Warmup phase
	if band.WarmupTime > 0 {
		pr.logger.Debug("Starting warmup phase",
			zap.Int("band_id", band.BandID),
			zap.Duration("warmup_duration", band.WarmupTime),
		)
		time.Sleep(band.WarmupTime)
	}

	// Main test execution with metrics collection
	metricsCollector := pr.startMetricsCollection(bandCtx, band)
	
	metrics, err := workloadRunner(bandCtx, band)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("workload execution failed: %v", err))
	}

	// Stop metrics collection and get samples
	result.Samples = pr.stopMetricsCollection(metricsCollector)

	result.EndTime = time.Now()

	// Process metrics if available
	if metrics != nil {
		result.Metrics = pr.processMetrics(metrics, result.Samples)
		result.Health = pr.calculateBandHealth(result.Samples)
	}

	return result
}

// MetricsCollector handles periodic metrics collection during band execution
type MetricsCollector struct {
	samples  []MetricSample
	stop     chan struct{}
	mutex    sync.Mutex
	interval time.Duration
}

// startMetricsCollection begins collecting metrics at regular intervals
func (pr *ProgressiveRunner) startMetricsCollection(ctx context.Context, band BandConfig) *MetricsCollector {
	collector := &MetricsCollector{
		samples:  make([]MetricSample, 0),
		stop:     make(chan struct{}),
		interval: 5 * time.Second, // Collect every 5 seconds
	}

	go func() {
		ticker := time.NewTicker(collector.interval)
		defer ticker.Stop()
		
		startTime := time.Now()

		for {
			select {
			case <-ticker.C:
				// In a real implementation, this would collect actual metrics
				// For now, we'll create placeholder samples
				sample := MetricSample{
					Timestamp:   time.Now(),
					ElapsedTime: time.Since(startTime),
					ActiveConns: band.Connections,
					// Other metrics would be collected from the actual workload
				}
				
				collector.mutex.Lock()
				collector.samples = append(collector.samples, sample)
				collector.mutex.Unlock()

			case <-collector.stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return collector
}

// stopMetricsCollection stops metrics collection and returns collected samples
func (pr *ProgressiveRunner) stopMetricsCollection(collector *MetricsCollector) []MetricSample {
	close(collector.stop)
	
	collector.mutex.Lock()
	defer collector.mutex.Unlock()
	
	samples := make([]MetricSample, len(collector.samples))
	copy(samples, collector.samples)
	return samples
}

// processMetrics converts raw metrics to band metrics
func (pr *ProgressiveRunner) processMetrics(metrics *types.Metrics, samples []MetricSample) *BandMetrics {
	// Calculate latency percentiles
	latencies := metrics.TransactionDur
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	bandMetrics := &BandMetrics{
		TotalTransactions: metrics.TPS,
		TotalQueries:      metrics.QPS,
		TotalErrors:       metrics.Errors,
		ErrorTypes:        metrics.ErrorTypes,
	}

	if len(latencies) > 0 {
		// Convert nanoseconds to milliseconds
		nsToMs := func(ns int64) float64 {
			return float64(ns) / 1e6
		}

		bandMetrics.LatencyP50 = nsToMs(pr.percentile(latencies, 0.50))
		bandMetrics.LatencyP90 = nsToMs(pr.percentile(latencies, 0.90))
		bandMetrics.LatencyP95 = nsToMs(pr.percentile(latencies, 0.95))
		bandMetrics.LatencyP99 = nsToMs(pr.percentile(latencies, 0.99))

		// Calculate mean and standard deviation
		sum := int64(0)
		for _, latency := range latencies {
			sum += latency
		}
		mean := float64(sum) / float64(len(latencies))
		bandMetrics.LatencyMean = nsToMs(int64(mean))

		// Standard deviation
		sumSquaredDiff := float64(0)
		for _, latency := range latencies {
			diff := float64(latency) - mean
			sumSquaredDiff += diff * diff
		}
		stdDev := math.Sqrt(sumSquaredDiff / float64(len(latencies)))
		bandMetrics.LatencyStdDev = nsToMs(int64(stdDev))

		// Coefficient of variation
		if bandMetrics.LatencyMean > 0 {
			bandMetrics.CoefficientOfVariation = bandMetrics.LatencyStdDev / bandMetrics.LatencyMean
		}

		// 95% Confidence interval (assuming normal distribution)
		if len(latencies) > 1 {
			sem := bandMetrics.LatencyStdDev / math.Sqrt(float64(len(latencies)))
			margin := 1.96 * sem // 95% CI
			bandMetrics.ConfidenceInterval95 = ConfidenceInterval{
				Lower: bandMetrics.LatencyMean - margin,
				Upper: bandMetrics.LatencyMean + margin,
				Mean:  bandMetrics.LatencyMean,
			}
		}
	}

	// Calculate rates from samples
	if len(samples) > 1 {
		totalDuration := samples[len(samples)-1].ElapsedTime.Seconds()
		if totalDuration > 0 {
			bandMetrics.AvgTPS = float64(bandMetrics.TotalTransactions) / totalDuration
			bandMetrics.AvgQPS = float64(bandMetrics.TotalQueries) / totalDuration
			bandMetrics.ErrorRate = float64(bandMetrics.TotalErrors) / float64(bandMetrics.TotalTransactions)
		}
	}

	return bandMetrics
}

// percentile calculates the specified percentile from sorted latency data
func (pr *ProgressiveRunner) percentile(sortedData []int64, p float64) int64 {
	if len(sortedData) == 0 {
		return 0
	}
	if p <= 0 {
		return sortedData[0]
	}
	if p >= 1 {
		return sortedData[len(sortedData)-1]
	}

	index := p * float64(len(sortedData)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sortedData[lower]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return int64(float64(sortedData[lower])*(1-weight) + float64(sortedData[upper])*weight)
}

// calculateBandHealth determines health metrics for a band
func (pr *ProgressiveRunner) calculateBandHealth(samples []MetricSample) BandHealth {
	if len(samples) == 0 {
		return BandHealth{HealthScore: 0.0}
	}

	healthyCount := 0
	totalErrorRate := 0.0
	maxErrorRate := 0.0

	for _, sample := range samples {
		errorRate := float64(sample.ErrorCount) / math.Max(1, sample.TPS) // Avoid division by zero
		totalErrorRate += errorRate
		if errorRate > maxErrorRate {
			maxErrorRate = errorRate
		}
		
		// Consider healthy if error rate < 5% and latency < 100ms
		if errorRate < 0.05 && sample.LatencyP95 < 100 {
			healthyCount++
		}
	}

	avgErrorRate := totalErrorRate / float64(len(samples))
	healthScore := float64(healthyCount) / float64(len(samples))

	totalDuration := time.Duration(0)
	if len(samples) > 0 {
		totalDuration = samples[len(samples)-1].ElapsedTime
	}

	return BandHealth{
		HealthyDuration:   time.Duration(float64(totalDuration) * healthScore),
		UnhealthyDuration: time.Duration(float64(totalDuration) * (1 - healthScore)),
		HealthScore:       healthScore,
		MaxErrorRate:      maxErrorRate,
		AvgErrorRate:      avgErrorRate,
	}
}

// shouldTerminateEarly determines if the test should be terminated early
func (pr *ProgressiveRunner) shouldTerminateEarly(result BandResult) bool {
	// Terminate if error rate is too high
	if result.Health.AvgErrorRate > 0.1 { // 10% error rate
		return true
	}

	// Terminate if health score is too low
	if result.Health.HealthScore < 0.5 { // Less than 50% healthy
		return true
	}

	// Terminate if performance has degraded significantly
	if len(pr.results) > 1 {
		prevResult := pr.results[len(pr.results)-2]
		if prevResult.Metrics != nil && result.Metrics != nil {
			// Check for significant performance degradation (>50% drop in TPS)
			degradation := (prevResult.Metrics.AvgTPS - result.Metrics.AvgTPS) / prevResult.Metrics.AvgTPS
			if degradation > 0.5 {
				return true
			}
		}
	}

	return false
}

// estimateTotalDuration estimates the total test duration
func (pr *ProgressiveRunner) estimateTotalDuration() time.Duration {
	total := time.Duration(0)
	for _, band := range pr.bands {
		total += band.WarmupTime + band.Duration + band.CooldownTime
	}
	return total
}

// GetProgress returns current test progress
func (pr *ProgressiveRunner) GetProgress() ProgressInfo {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	return ProgressInfo{
		Running:        pr.running,
		CurrentBand:    pr.currentBand,
		TotalBands:     len(pr.bands),
		CompletedBands: len(pr.results),
		EstimatedRemaining: pr.estimateRemainingDuration(),
	}
}

// ProgressInfo contains current progress information
type ProgressInfo struct {
	Running            bool          `json:"running"`
	CurrentBand        int           `json:"current_band"`
	TotalBands         int           `json:"total_bands"`
	CompletedBands     int           `json:"completed_bands"`
	EstimatedRemaining time.Duration `json:"estimated_remaining"`
}

// estimateRemainingDuration estimates remaining test duration
func (pr *ProgressiveRunner) estimateRemainingDuration() time.Duration {
	if !pr.running || pr.currentBand >= len(pr.bands) {
		return 0
	}

	remaining := time.Duration(0)
	for i := pr.currentBand - 1; i < len(pr.bands); i++ {
		band := pr.bands[i]
		remaining += band.WarmupTime + band.Duration + band.CooldownTime
	}
	return remaining
}

// GetResults returns all completed band results
func (pr *ProgressiveRunner) GetResults() []BandResult {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	results := make([]BandResult, len(pr.results))
	copy(results, pr.results)
	return results
}

// performAnalysis runs advanced analytics on the completed results
func (pr *ProgressiveRunner) performAnalysis() error {
	if pr.analytics == nil {
		return fmt.Errorf("analytics engine not available")
	}

	return pr.analytics.Analyze(pr.results)
}
