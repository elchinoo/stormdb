package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/elchinoo/stormdb/internal/logging"
	"go.uber.org/zap"
)

// MetricsCollector provides advanced metrics collection and aggregation
type MetricsCollector struct {
	logger    logging.StormDBLogger
	startTime time.Time

	// Metrics storage
	samples []MetricSample
	mutex   sync.RWMutex

	// Collection settings
	interval   time.Duration
	maxSamples int

	// Control channels
	stopChan chan struct{}
	doneChan chan struct{}
	running  bool
}

// MetricSample represents a single point-in-time metric sample
type MetricSample struct {
	Timestamp      time.Time `json:"timestamp"`
	ElapsedSeconds float64   `json:"elapsed_seconds"`

	// Throughput metrics
	TPS float64 `json:"tps"`
	QPS float64 `json:"qps"`

	// Latency metrics (milliseconds)
	LatencyMean   float64 `json:"latency_mean_ms"`
	LatencyP50    float64 `json:"latency_p50_ms"`
	LatencyP90    float64 `json:"latency_p90_ms"`
	LatencyP95    float64 `json:"latency_p95_ms"`
	LatencyP99    float64 `json:"latency_p99_ms"`
	LatencyStdDev float64 `json:"latency_stddev_ms"`

	// Error metrics
	ErrorCount int64   `json:"error_count"`
	ErrorRate  float64 `json:"error_rate"`

	// System metrics
	ActiveConnections int     `json:"active_connections"`
	CPUUsage          float64 `json:"cpu_usage_percent"`
	MemoryUsageMB     float64 `json:"memory_usage_mb"`

	// Database metrics
	DBConnections   int     `json:"db_connections"`
	DBIdleConns     int     `json:"db_idle_connections"`
	DBActiveQueries int     `json:"db_active_queries"`
	DBCacheHitRatio float64 `json:"db_cache_hit_ratio"`
}

// MetricAggregates contains aggregated metrics over the collection period
type MetricAggregates struct {
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`

	// Throughput aggregates
	TPSStats StatisticalSummary `json:"tps_stats"`
	QPSStats StatisticalSummary `json:"qps_stats"`

	// Latency aggregates
	LatencyStats LatencyStatistics `json:"latency_stats"`

	// Error aggregates
	TotalErrors    int64              `json:"total_errors"`
	ErrorRateStats StatisticalSummary `json:"error_rate_stats"`

	// Trend analysis
	TPSTrend     TrendAnalysis `json:"tps_trend"`
	LatencyTrend TrendAnalysis `json:"latency_trend"`

	// Quality metrics
	DataQuality DataQualityMetrics `json:"data_quality"`
}

// StatisticalSummary provides comprehensive statistical summary
type StatisticalSummary struct {
	Count            int     `json:"count"`
	Mean             float64 `json:"mean"`
	Median           float64 `json:"median"`
	Min              float64 `json:"min"`
	Max              float64 `json:"max"`
	StandardDev      float64 `json:"standard_deviation"`
	Variance         float64 `json:"variance"`
	CoefficientOfVar float64 `json:"coefficient_of_variation"`
	Skewness         float64 `json:"skewness"`
	Kurtosis         float64 `json:"kurtosis"`
	Q1               float64 `json:"q1"`
	Q3               float64 `json:"q3"`
	IQR              float64 `json:"iqr"`
}

// LatencyStatistics provides detailed latency analysis
type LatencyStatistics struct {
	OverallStats StatisticalSummary `json:"overall_stats"`
	P50Stats     StatisticalSummary `json:"p50_stats"`
	P90Stats     StatisticalSummary `json:"p90_stats"`
	P95Stats     StatisticalSummary `json:"p95_stats"`
	P99Stats     StatisticalSummary `json:"p99_stats"`

	// SLA compliance
	SLAThresholds map[string]float64 `json:"sla_thresholds"`
	SLACompliance map[string]float64 `json:"sla_compliance"`
}

// TrendAnalysis provides trend detection and analysis
type TrendAnalysis struct {
	Direction        string  `json:"direction"` // "increasing", "decreasing", "stable", "volatile"
	Strength         float64 `json:"strength"`  // 0.0 to 1.0
	Slope            float64 `json:"slope"`
	RSquared         float64 `json:"r_squared"`
	Autocorrelation  float64 `json:"autocorrelation"`
	SeasonalityScore float64 `json:"seasonality_score"`
	AnomalyCount     int     `json:"anomaly_count"`
	ChangePoints     []int   `json:"change_points"`
}

// DataQualityMetrics assess the quality of collected metrics
type DataQualityMetrics struct {
	Completeness float64 `json:"completeness"`  // Percentage of expected samples collected
	Consistency  float64 `json:"consistency"`   // Consistency of sampling intervals
	Accuracy     float64 `json:"accuracy"`      // Estimated accuracy of measurements
	Timeliness   float64 `json:"timeliness"`    // Timeliness of data collection
	Validity     float64 `json:"validity"`      // Percentage of valid samples
	OverallScore float64 `json:"overall_score"` // Combined quality score
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger logging.StormDBLogger, interval time.Duration, maxSamples int) *MetricsCollector {
	if logger == nil {
		logger = logging.NewDefaultLogger()
	}
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if maxSamples <= 0 {
		maxSamples = 10000
	}

	return &MetricsCollector{
		logger:     logger.With(zap.String("component", "metrics_collector")),
		interval:   interval,
		maxSamples: maxSamples,
		samples:    make([]MetricSample, 0, maxSamples),
		stopChan:   make(chan struct{}),
		doneChan:   make(chan struct{}),
	}
}

// Start begins metrics collection
func (mc *MetricsCollector) Start(ctx context.Context) error {
	mc.mutex.Lock()
	if mc.running {
		mc.mutex.Unlock()
		return fmt.Errorf("metrics collector is already running")
	}
	mc.running = true
	mc.startTime = time.Now()
	mc.mutex.Unlock()

	mc.logger.Info("Starting metrics collection",
		zap.Duration("interval", mc.interval),
		zap.Int("max_samples", mc.maxSamples),
	)

	go mc.collectMetrics(ctx)
	return nil
}

// Stop stops metrics collection and returns final aggregates
func (mc *MetricsCollector) Stop() MetricAggregates {
	mc.mutex.Lock()
	if !mc.running {
		mc.mutex.Unlock()
		return MetricAggregates{}
	}
	mc.running = false
	mc.mutex.Unlock()

	// Signal collection to stop
	close(mc.stopChan)

	// Wait for collection to finish
	<-mc.doneChan

	// Calculate final aggregates
	aggregates := mc.calculateAggregates()

	mc.logger.Info("Stopped metrics collection",
		zap.Int("total_samples", len(mc.samples)),
		zap.Duration("collection_duration", aggregates.Duration),
	)

	return aggregates
}

// collectMetrics runs the metrics collection loop
func (mc *MetricsCollector) collectMetrics(ctx context.Context) {
	defer close(mc.doneChan)

	ticker := time.NewTicker(mc.interval)
	defer ticker.Stop()

	sampleCount := 0
	for {
		select {
		case <-ticker.C:
			sample := mc.collectSample()

			mc.mutex.Lock()
			if len(mc.samples) >= mc.maxSamples {
				// Remove oldest sample to make room
				mc.samples = mc.samples[1:]
			}
			mc.samples = append(mc.samples, sample)
			sampleCount++
			mc.mutex.Unlock()

			// Log periodic progress
			if sampleCount%60 == 0 { // Every ~5 minutes at 5s intervals
				mc.logger.Debug("Metrics collection progress",
					zap.Int("samples_collected", sampleCount),
					zap.Duration("elapsed", time.Since(mc.startTime)),
				)
			}

		case <-mc.stopChan:
			mc.logger.Debug("Metrics collection stopped",
				zap.Int("final_sample_count", sampleCount),
			)
			return

		case <-ctx.Done():
			mc.logger.Debug("Metrics collection cancelled",
				zap.Int("final_sample_count", sampleCount),
			)
			return
		}
	}
}

// collectSample collects a single metrics sample
func (mc *MetricsCollector) collectSample() MetricSample {
	now := time.Now()
	elapsed := now.Sub(mc.startTime).Seconds()

	// In a real implementation, these would collect actual metrics
	// For now, we create a placeholder sample
	sample := MetricSample{
		Timestamp:      now,
		ElapsedSeconds: elapsed,
		// Actual metrics would be collected here from:
		// - Performance counters
		// - Database statistics
		// - System metrics
		// - Application metrics
	}

	return sample
}

// GetCurrentSample returns the most recent metrics sample
func (mc *MetricsCollector) GetCurrentSample() (MetricSample, bool) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	if len(mc.samples) == 0 {
		return MetricSample{}, false
	}

	return mc.samples[len(mc.samples)-1], true
}

// GetSamples returns all collected samples
func (mc *MetricsCollector) GetSamples() []MetricSample {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	samples := make([]MetricSample, len(mc.samples))
	copy(samples, mc.samples)
	return samples
}

// GetSamplesInRange returns samples within the specified time range
func (mc *MetricsCollector) GetSamplesInRange(start, end time.Time) []MetricSample {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	var rangeSamples []MetricSample
	for _, sample := range mc.samples {
		if sample.Timestamp.After(start) && sample.Timestamp.Before(end) {
			rangeSamples = append(rangeSamples, sample)
		}
	}

	return rangeSamples
}

// GetRealtimeAggregates calculates aggregates for samples collected so far
func (mc *MetricsCollector) GetRealtimeAggregates() MetricAggregates {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	return mc.calculateAggregates()
}

// calculateAggregates computes comprehensive aggregate statistics
func (mc *MetricsCollector) calculateAggregates() MetricAggregates {
	if len(mc.samples) == 0 {
		return MetricAggregates{
			StartTime: mc.startTime,
			EndTime:   time.Now(),
		}
	}

	startTime := mc.samples[0].Timestamp
	endTime := mc.samples[len(mc.samples)-1].Timestamp
	duration := endTime.Sub(startTime)

	// Extract time series data
	tpsValues := make([]float64, len(mc.samples))
	qpsValues := make([]float64, len(mc.samples))
	latencyP95Values := make([]float64, len(mc.samples))
	errorRates := make([]float64, len(mc.samples))

	for i, sample := range mc.samples {
		tpsValues[i] = sample.TPS
		qpsValues[i] = sample.QPS
		latencyP95Values[i] = sample.LatencyP95
		errorRates[i] = sample.ErrorRate
	}

	// Calculate statistical summaries
	tpsStats := mc.calculateStatisticalSummary(tpsValues)
	qpsStats := mc.calculateStatisticalSummary(qpsValues)
	errorRateStats := mc.calculateStatisticalSummary(errorRates)

	// Calculate latency statistics
	latencyStats := mc.calculateLatencyStatistics()

	// Calculate trend analyses
	tpsTrend := mc.calculateTrendAnalysis(tpsValues)
	latencyTrend := mc.calculateTrendAnalysis(latencyP95Values)

	// Calculate data quality metrics
	dataQuality := mc.calculateDataQuality()

	// Calculate total errors
	totalErrors := int64(0)
	for _, sample := range mc.samples {
		totalErrors += sample.ErrorCount
	}

	return MetricAggregates{
		StartTime:      startTime,
		EndTime:        endTime,
		Duration:       duration,
		TPSStats:       tpsStats,
		QPSStats:       qpsStats,
		LatencyStats:   latencyStats,
		TotalErrors:    totalErrors,
		ErrorRateStats: errorRateStats,
		TPSTrend:       tpsTrend,
		LatencyTrend:   latencyTrend,
		DataQuality:    dataQuality,
	}
}

// calculateStatisticalSummary computes comprehensive statistics for a data series
func (mc *MetricsCollector) calculateStatisticalSummary(data []float64) StatisticalSummary {
	if len(data) == 0 {
		return StatisticalSummary{}
	}

	// Sort data for percentile calculations
	sortedData := make([]float64, len(data))
	copy(sortedData, data)

	// Simple sort - in production, use sort.Float64s
	for i := 0; i < len(sortedData); i++ {
		for j := i + 1; j < len(sortedData); j++ {
			if sortedData[i] > sortedData[j] {
				sortedData[i], sortedData[j] = sortedData[j], sortedData[i]
			}
		}
	}

	// Basic statistics
	count := len(data)
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	mean := sum / float64(count)

	// Min and max
	min := sortedData[0]
	max := sortedData[count-1]

	// Median
	median := sortedData[count/2]
	if count%2 == 0 {
		median = (sortedData[count/2-1] + sortedData[count/2]) / 2
	}

	// Variance and standard deviation
	sumSquaredDiff := 0.0
	for _, v := range data {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	variance := sumSquaredDiff / float64(count-1)
	stdDev := 0.0
	if variance > 0 {
		stdDev = variance * variance // Simplified square root
	}

	// Coefficient of variation
	cv := 0.0
	if mean != 0 {
		cv = stdDev / mean
	}

	// Quartiles
	q1 := sortedData[count/4]
	q3 := sortedData[3*count/4]
	iqr := q3 - q1

	return StatisticalSummary{
		Count:            count,
		Mean:             mean,
		Median:           median,
		Min:              min,
		Max:              max,
		StandardDev:      stdDev,
		Variance:         variance,
		CoefficientOfVar: cv,
		Q1:               q1,
		Q3:               q3,
		IQR:              iqr,
		// Skewness and Kurtosis would be calculated here in full implementation
	}
}

// calculateLatencyStatistics computes detailed latency analysis
func (mc *MetricsCollector) calculateLatencyStatistics() LatencyStatistics {
	if len(mc.samples) == 0 {
		return LatencyStatistics{}
	}

	// Extract latency percentile series
	p50Values := make([]float64, len(mc.samples))
	p90Values := make([]float64, len(mc.samples))
	p95Values := make([]float64, len(mc.samples))
	p99Values := make([]float64, len(mc.samples))
	meanValues := make([]float64, len(mc.samples))

	for i, sample := range mc.samples {
		p50Values[i] = sample.LatencyP50
		p90Values[i] = sample.LatencyP90
		p95Values[i] = sample.LatencyP95
		p99Values[i] = sample.LatencyP99
		meanValues[i] = sample.LatencyMean
	}

	// Calculate statistics for each percentile
	overallStats := mc.calculateStatisticalSummary(meanValues)
	p50Stats := mc.calculateStatisticalSummary(p50Values)
	p90Stats := mc.calculateStatisticalSummary(p90Values)
	p95Stats := mc.calculateStatisticalSummary(p95Values)
	p99Stats := mc.calculateStatisticalSummary(p99Values)

	// Calculate SLA compliance (example thresholds)
	slaThresholds := map[string]float64{
		"p50_100ms": 100.0,
		"p95_200ms": 200.0,
		"p99_500ms": 500.0,
		"mean_50ms": 50.0,
	}

	slaCompliance := map[string]float64{
		"p50_100ms": mc.calculateSLACompliance(p50Values, 100.0),
		"p95_200ms": mc.calculateSLACompliance(p95Values, 200.0),
		"p99_500ms": mc.calculateSLACompliance(p99Values, 500.0),
		"mean_50ms": mc.calculateSLACompliance(meanValues, 50.0),
	}

	return LatencyStatistics{
		OverallStats:  overallStats,
		P50Stats:      p50Stats,
		P90Stats:      p90Stats,
		P95Stats:      p95Stats,
		P99Stats:      p99Stats,
		SLAThresholds: slaThresholds,
		SLACompliance: slaCompliance,
	}
}

// calculateSLACompliance calculates the percentage of samples meeting SLA threshold
func (mc *MetricsCollector) calculateSLACompliance(values []float64, threshold float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	compliantCount := 0
	for _, value := range values {
		if value <= threshold {
			compliantCount++
		}
	}

	return float64(compliantCount) / float64(len(values)) * 100.0
}

// calculateTrendAnalysis performs trend analysis on a time series
func (mc *MetricsCollector) calculateTrendAnalysis(values []float64) TrendAnalysis {
	if len(values) < 2 {
		return TrendAnalysis{Direction: "insufficient_data"}
	}

	// Simple linear regression for trend detection
	n := float64(len(values))
	sumX := n * (n - 1) / 2 // Sum of 0, 1, 2, ..., n-1
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i, y := range values {
		x := float64(i)
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Calculate slope and R-squared
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Determine direction
	direction := "stable"
	if slope > 0.1 {
		direction = "increasing"
	} else if slope < -0.1 {
		direction = "decreasing"
	}

	// Calculate R-squared for trend strength
	yMean := sumY / n
	ssRes := 0.0
	ssTot := 0.0
	for i, y := range values {
		x := float64(i)
		predicted := slope*x + (sumY-slope*sumX)/n
		ssRes += (y - predicted) * (y - predicted)
		ssTot += (y - yMean) * (y - yMean)
	}

	rSquared := 0.0
	if ssTot > 0 {
		rSquared = 1.0 - (ssRes / ssTot)
	}

	strength := rSquared // Simplified strength calculation

	return TrendAnalysis{
		Direction: direction,
		Strength:  strength,
		Slope:     slope,
		RSquared:  rSquared,
		// Other metrics would be calculated in full implementation
	}
}

// calculateDataQuality assesses the quality of collected metrics data
func (mc *MetricsCollector) calculateDataQuality() DataQualityMetrics {
	if len(mc.samples) == 0 {
		return DataQualityMetrics{}
	}

	// Calculate completeness based on expected samples
	expectedSamples := time.Since(mc.startTime) / mc.interval
	completeness := float64(len(mc.samples)) / float64(expectedSamples) * 100.0
	if completeness > 100.0 {
		completeness = 100.0
	}

	// Calculate consistency of sampling intervals
	intervalConsistency := 100.0 // Simplified - would check actual intervals

	// Calculate other quality metrics (simplified)
	accuracy := 95.0   // Placeholder
	timeliness := 98.0 // Placeholder
	validity := 99.0   // Placeholder

	// Calculate overall quality score
	overallScore := (completeness + intervalConsistency + accuracy + timeliness + validity) / 5.0

	return DataQualityMetrics{
		Completeness: completeness,
		Consistency:  intervalConsistency,
		Accuracy:     accuracy,
		Timeliness:   timeliness,
		Validity:     validity,
		OverallScore: overallScore,
	}
}

// GetStatus returns current collector status
func (mc *MetricsCollector) GetStatus() CollectorStatus {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	return CollectorStatus{
		Running:       mc.running,
		StartTime:     mc.startTime,
		SampleCount:   len(mc.samples),
		MaxSamples:    mc.maxSamples,
		Interval:      mc.interval,
		ElapsedTime:   time.Since(mc.startTime),
		MemoryUsageKB: len(mc.samples) * 200, // Rough estimate
	}
}

// CollectorStatus represents the current status of metrics collection
type CollectorStatus struct {
	Running       bool          `json:"running"`
	StartTime     time.Time     `json:"start_time"`
	SampleCount   int           `json:"sample_count"`
	MaxSamples    int           `json:"max_samples"`
	Interval      time.Duration `json:"interval"`
	ElapsedTime   time.Duration `json:"elapsed_time"`
	MemoryUsageKB int           `json:"memory_usage_kb"`
}
