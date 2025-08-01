// internal/infrastructure/adapters/streaming_metrics_collector.go
package adapters

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elchinoo/stormdb/internal/core/domain"
	"github.com/elchinoo/stormdb/internal/core/ports"
)

// StreamingMetricsCollectorImpl implements memory-efficient metrics collection using Welford's method
type StreamingMetricsCollectorImpl struct {
	// Configuration
	maxLatencySamples int
	maxTPSSamples     int
	
	// Current collection state
	bandID           int
	startTime        time.Time
	endTime          time.Time
	isCollecting     bool
	
	// Transaction metrics (streaming statistics)
	totalTransactions atomic.Int64
	successCount      atomic.Int64
	errorCount        atomic.Int64
	
	// Latency statistics (Welford's algorithm for memory efficiency)
	latencyMutex      sync.RWMutex
	latencyCount      int64
	latencyMean       float64
	latencyM2         float64 // Sum of squares of differences from mean
	minLatency        float64
	maxLatency        float64
	latencyQuantiles  *QuantileTracker
	
	// TPS/QPS tracking (sliding window)
	tpsMutex          sync.RWMutex
	tpsWindow         []TPSPoint
	tpsWindowSize     int
	currentWindowIdx  int
	
	// Query metrics
	queryMutex        sync.RWMutex
	queryCount        atomic.Int64
	queryTypes        map[string]int64
	totalRowsAffected atomic.Int64
	
	// Error tracking
	errorMutex        sync.RWMutex
	errorTypes        map[string]int64
	
	// Custom metrics
	customMutex       sync.RWMutex
	customMetrics     map[string]float64
	
	// Memory management
	memoryLimitMB     int
	
	// Listeners for real-time monitoring
	listenerMutex     sync.RWMutex
	listeners         []ports.MetricsListener
}

type TPSPoint struct {
	Timestamp    time.Time
	Transactions int64
}

type QuantileTracker struct {
	samples []float64
	sorted  bool
	maxSize int
	mutex   sync.RWMutex
}

type LatencyStatistics struct {
	Mean   float64
	StdDev float64
	Min    float64
	Max    float64
	P50    float64
	P95    float64
	P99    float64
}

func NewStreamingMetricsCollector(maxLatencySamples, maxTPSSamples int) *StreamingMetricsCollectorImpl {
	return &StreamingMetricsCollectorImpl{
		maxLatencySamples: maxLatencySamples,
		maxTPSSamples:     maxTPSSamples,
		tpsWindowSize:     min(maxTPSSamples, 1000), // Reasonable window size
		tpsWindow:         make([]TPSPoint, min(maxTPSSamples, 1000)),
		queryTypes:        make(map[string]int64),
		errorTypes:        make(map[string]int64),
		customMetrics:     make(map[string]float64),
		latencyQuantiles:  NewQuantileTracker(maxLatencySamples),
		memoryLimitMB:     100, // Default 100MB limit
		minLatency:        math.MaxFloat64,
		maxLatency:        0,
	}
}

func NewQuantileTracker(maxSize int) *QuantileTracker {
	return &QuantileTracker{
		samples: make([]float64, 0, maxSize),
		maxSize: maxSize,
	}
}

// QuantileTracker methods

func (qt *QuantileTracker) Add(value float64) {
	qt.mutex.Lock()
	defer qt.mutex.Unlock()
	
	if len(qt.samples) >= qt.maxSize {
		// Remove oldest sample (FIFO)
		qt.samples = qt.samples[1:]
	}
	
	qt.samples = append(qt.samples, value)
	qt.sorted = false
}

func (qt *QuantileTracker) Reset() {
	qt.mutex.Lock()
	defer qt.mutex.Unlock()
	
	qt.samples = qt.samples[:0]
	qt.sorted = false
}

func (qt *QuantileTracker) Size() int {
	qt.mutex.RLock()
	defer qt.mutex.RUnlock()
	
	return len(qt.samples)
}

func (qt *QuantileTracker) Resize(newMaxSize int) {
	qt.mutex.Lock()
	defer qt.mutex.Unlock()
	
	qt.maxSize = newMaxSize
	if len(qt.samples) > newMaxSize {
		qt.samples = qt.samples[len(qt.samples)-newMaxSize:]
	}
}

func (qt *QuantileTracker) GetSamples() []float64 {
	qt.mutex.RLock()
	defer qt.mutex.RUnlock()
	
	result := make([]float64, len(qt.samples))
	copy(result, qt.samples)
	return result
}

func (qt *QuantileTracker) GetPercentiles(percentiles []float64) []float64 {
	qt.mutex.Lock()
	defer qt.mutex.Unlock()
	
	if len(qt.samples) == 0 {
		return make([]float64, len(percentiles))
	}
	
	// Sort if needed
	if !qt.sorted {
		sort.Float64s(qt.samples)
		qt.sorted = true
	}
	
	result := make([]float64, len(percentiles))
	for i, p := range percentiles {
		result[i] = qt.getPercentile(p)
	}
	
	return result
}

func (qt *QuantileTracker) getPercentile(p float64) float64 {
	if len(qt.samples) == 0 {
		return 0
	}
	
	if p <= 0 {
		return qt.samples[0]
	}
	if p >= 1 {
		return qt.samples[len(qt.samples)-1]
	}
	
	index := p * float64(len(qt.samples)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	
	if lower == upper {
		return qt.samples[lower]
	}
	
	// Linear interpolation
	weight := index - float64(lower)
	return qt.samples[lower]*(1-weight) + qt.samples[upper]*weight
}

// StartCollection begins metrics collection for a new band
func (c *StreamingMetricsCollectorImpl) StartCollection(bandID int, expectedDuration time.Duration) {
	c.bandID = bandID
	c.startTime = time.Now()
	c.isCollecting = true
	
	// Reset all counters
	c.totalTransactions.Store(0)
	c.successCount.Store(0)
	c.errorCount.Store(0)
	c.queryCount.Store(0)
	c.totalRowsAffected.Store(0)
	
	// Reset streaming statistics
	c.latencyMutex.Lock()
	c.latencyCount = 0
	c.latencyMean = 0
	c.latencyM2 = 0
	c.minLatency = math.MaxFloat64
	c.maxLatency = 0
	c.latencyMutex.Unlock()
	
	// Reset TPS window
	c.tpsMutex.Lock()
	c.tpsWindow = make([]TPSPoint, c.tpsWindowSize)
	c.currentWindowIdx = 0
	c.tpsMutex.Unlock()
	
	// Reset quantile tracker
	c.latencyQuantiles.Reset()
	
	// Clear maps
	c.queryMutex.Lock()
	c.queryTypes = make(map[string]int64)
	c.queryMutex.Unlock()
	
	c.errorMutex.Lock()
	c.errorTypes = make(map[string]int64)
	c.errorMutex.Unlock()
	
	c.customMutex.Lock()
	c.customMetrics = make(map[string]float64)
	c.customMutex.Unlock()
}

// StopCollection ends metrics collection and returns aggregated results
func (c *StreamingMetricsCollectorImpl) StopCollection() *domain.BandResults {
	c.isCollecting = false
	c.endTime = time.Now()
	
	duration := c.endTime.Sub(c.startTime)
	
	// Calculate final statistics
	totalTxns := c.totalTransactions.Load()
	errors := c.errorCount.Load()
	
	// TPS calculation
	tps := float64(totalTxns) / duration.Seconds()
	qps := float64(c.queryCount.Load()) / duration.Seconds()
	
	// Error rate
	errorRate := 0.0
	if totalTxns > 0 {
		errorRate = float64(errors) / float64(totalTxns)
	}
	
	// Latency statistics
	c.latencyMutex.RLock()
	latencyStats := c.calculateLatencyStatistics()
	c.latencyMutex.RUnlock()
	
	// Efficiency metrics
	efficiency := c.calculateEfficiencyMetrics(duration, totalTxns)
	
	// Stability metrics
	stability := c.calculateStabilityMetrics()
	
	// Resource metrics (basic implementation)
	resources := c.calculateResourceMetrics()
	
	// Notify listeners of completion
	c.notifyBandComplete(&domain.BandResults{
		BandID:      c.bandID,
		Duration:    duration,
		Performance: domain.PerformanceMetrics{
			TotalTPS:   tps,
			TotalQPS:   qps,
			AvgLatency: latencyStats.Mean,
			P50Latency: latencyStats.P50,
			P95Latency: latencyStats.P95,
			P99Latency: latencyStats.P99,
			ErrorRate:  errorRate,
		},
		Efficiency: efficiency,
		Stability:  stability,
		Resources:  resources,
	})
	
	return &domain.BandResults{
		BandID:      c.bandID,
		Duration:    duration,
		Performance: domain.PerformanceMetrics{
			TotalTPS:   tps,
			TotalQPS:   qps,
			AvgLatency: latencyStats.Mean,
			P50Latency: latencyStats.P50,
			P95Latency: latencyStats.P95,
			P99Latency: latencyStats.P99,
			ErrorRate:  errorRate,
		},
		Efficiency: efficiency,
		Stability:  stability,
		Resources:  resources,
	}
}

// RecordTransaction records a completed transaction with latency
func (c *StreamingMetricsCollectorImpl) RecordTransaction(success bool, latencyNs int64) {
	if !c.isCollecting {
		return
	}
	
	c.totalTransactions.Add(1)
	
	if success {
		c.successCount.Add(1)
	} else {
		c.errorCount.Add(1)
	}
	
	// Update latency statistics using Welford's algorithm
	latencyMs := float64(latencyNs) / 1e6 // Convert to milliseconds
	c.updateLatencyStatistics(latencyMs)
	
	// Update TPS window
	c.updateTPSWindow()
	
	// Notify listeners
	c.notifySnapshot()
}

// updateLatencyStatistics implements Welford's algorithm for online variance calculation
func (c *StreamingMetricsCollectorImpl) updateLatencyStatistics(latencyMs float64) {
	c.latencyMutex.Lock()
	defer c.latencyMutex.Unlock()
	
	c.latencyCount++
	
	// Welford's algorithm
	delta := latencyMs - c.latencyMean
	c.latencyMean += delta / float64(c.latencyCount)
	delta2 := latencyMs - c.latencyMean
	c.latencyM2 += delta * delta2
	
	// Update min/max
	if latencyMs < c.minLatency {
		c.minLatency = latencyMs
	}
	if latencyMs > c.maxLatency {
		c.maxLatency = latencyMs
	}
	
	// Add to quantile tracker (memory limited)
	c.latencyQuantiles.Add(latencyMs)
}

// calculateLatencyStatistics computes percentiles and other latency metrics
func (c *StreamingMetricsCollectorImpl) calculateLatencyStatistics() LatencyStatistics {
	stats := LatencyStatistics{
		Mean: c.latencyMean,
		Min:  c.minLatency,
		Max:  c.maxLatency,
	}
	
	// Calculate standard deviation
	if c.latencyCount > 1 {
		variance := c.latencyM2 / float64(c.latencyCount-1)
		stats.StdDev = math.Sqrt(variance)
	}
	
	// Calculate percentiles from quantile tracker
	percentiles := c.latencyQuantiles.GetPercentiles([]float64{0.5, 0.95, 0.99})
	if len(percentiles) >= 3 {
		stats.P50 = percentiles[0]
		stats.P95 = percentiles[1]
		stats.P99 = percentiles[2]
	}
	
	return stats
}

// updateTPSWindow maintains a sliding window of transaction timestamps
func (c *StreamingMetricsCollectorImpl) updateTPSWindow() {
	c.tpsMutex.Lock()
	defer c.tpsMutex.Unlock()
	
	now := time.Now()
	c.tpsWindow[c.currentWindowIdx] = TPSPoint{
		Timestamp:    now,
		Transactions: c.totalTransactions.Load(),
	}
	c.currentWindowIdx = (c.currentWindowIdx + 1) % c.tpsWindowSize
}

// RecordQuery records query execution details
func (c *StreamingMetricsCollectorImpl) RecordQuery(queryType string, rowsAffected int64) {
	if !c.isCollecting {
		return
	}
	
	c.queryCount.Add(1)
	c.totalRowsAffected.Add(rowsAffected)
	
	c.queryMutex.Lock()
	c.queryTypes[queryType]++
	c.queryMutex.Unlock()
}

// RecordError records error details
func (c *StreamingMetricsCollectorImpl) RecordError(err error) {
	if !c.isCollecting {
		return
	}
	
	errorType := "unknown"
	if err != nil {
		errorType = fmt.Sprintf("%T", err)
	}
	
	c.errorMutex.Lock()
	c.errorTypes[errorType]++
	c.errorMutex.Unlock()
}

// RecordCustomMetric records custom metrics
func (c *StreamingMetricsCollectorImpl) RecordCustomMetric(name string, value float64) {
	if !c.isCollecting {
		return
	}
	
	c.customMutex.Lock()
	c.customMetrics[name] = value
	c.customMutex.Unlock()
}

// GetCurrentTPS returns the current transactions per second
func (c *StreamingMetricsCollectorImpl) GetCurrentTPS() float64 {
	if !c.isCollecting {
		return 0
	}
	
	duration := time.Since(c.startTime).Seconds()
	if duration <= 0 {
		return 0
	}
	
	return float64(c.totalTransactions.Load()) / duration
}

// GetCurrentLatencyP95 returns the current 95th percentile latency
func (c *StreamingMetricsCollectorImpl) GetCurrentLatencyP95() float64 {
	percentiles := c.latencyQuantiles.GetPercentiles([]float64{0.95})
	if len(percentiles) > 0 {
		return percentiles[0]
	}
	return 0
}

// GetCurrentErrorRate returns the current error rate
func (c *StreamingMetricsCollectorImpl) GetCurrentErrorRate() float64 {
	total := c.totalTransactions.Load()
	if total == 0 {
		return 0
	}
	
	errors := c.errorCount.Load()
	return float64(errors) / float64(total)
}

// TakeSnapshot creates a snapshot of current metrics for analysis
func (c *StreamingMetricsCollectorImpl) TakeSnapshot() *domain.RawMetrics {
	// Create histogram from quantile tracker
	histogram := make(map[int]int64)
	samples := c.latencyQuantiles.GetSamples()
	
	for _, sample := range samples {
		bucket := int(sample / 10) * 10 // 10ms buckets
		histogram[bucket]++
	}
	
	// Get TPS samples
	c.tpsMutex.RLock()
	tpsSamples := make([]float64, 0, len(c.tpsWindow))
	timestamps := make([]time.Time, 0, len(c.tpsWindow))
	
	for i := 0; i < len(c.tpsWindow); i++ {
		point := c.tpsWindow[i]
		if !point.Timestamp.IsZero() {
			tpsSamples = append(tpsSamples, float64(point.Transactions))
			timestamps = append(timestamps, point.Timestamp)
		}
	}
	c.tpsMutex.RUnlock()
	
	// Get error types
	c.errorMutex.RLock()
	errorTypes := make(map[string]int64)
	for k, v := range c.errorTypes {
		errorTypes[k] = v
	}
	c.errorMutex.RUnlock()
	
	return &domain.RawMetrics{
		LatencyHistogram: histogram,
		TPSSamples:       tpsSamples,
		QPSSamples:       []float64{}, // Could be enhanced
		ErrorTypes:       errorTypes,
		SampleTimestamps: timestamps,
	}
}

// GetCurrentSnapshot returns current performance snapshot
func (c *StreamingMetricsCollectorImpl) GetCurrentSnapshot() *ports.MetricsSnapshot {
	return &ports.MetricsSnapshot{
		Timestamp:     time.Now(),
		TPS:          c.GetCurrentTPS(),
		QPS:          float64(c.queryCount.Load()) / time.Since(c.startTime).Seconds(),
		LatencyP50:   c.getQuickPercentile(0.5),
		LatencyP95:   c.GetCurrentLatencyP95(),
		LatencyP99:   c.getQuickPercentile(0.99),
		ErrorRate:    c.GetCurrentErrorRate(),
		ActiveWorkers: c.bandID, // Simplified - in real implementation, track active workers
	}
}

// getQuickPercentile calculates percentile without full sorting (approximation)
func (c *StreamingMetricsCollectorImpl) getQuickPercentile(p float64) float64 {
	percentiles := c.latencyQuantiles.GetPercentiles([]float64{p})
	if len(percentiles) > 0 {
		return percentiles[0]
	}
	return 0
}

// RegisterListener adds a metrics listener
func (c *StreamingMetricsCollectorImpl) RegisterListener(listener ports.MetricsListener) {
	c.listenerMutex.Lock()
	defer c.listenerMutex.Unlock()
	c.listeners = append(c.listeners, listener)
}

// SetMemoryLimits configures memory usage limits
func (c *StreamingMetricsCollectorImpl) SetMemoryLimits(maxLatencySamples, maxTPSSamples int) {
	c.maxLatencySamples = maxLatencySamples
	c.maxTPSSamples = maxTPSSamples
	
	// Resize quantile tracker
	c.latencyQuantiles.Resize(maxLatencySamples)
	
	// Resize TPS window
	c.tpsMutex.Lock()
	newSize := min(maxTPSSamples, 1000)
	c.tpsWindow = make([]TPSPoint, newSize)
	c.tpsWindowSize = newSize
	c.currentWindowIdx = 0
	c.tpsMutex.Unlock()
}

// GetMemoryUsage returns current memory usage statistics
func (c *StreamingMetricsCollectorImpl) GetMemoryUsage() ports.MemoryUsage {
	latencySamples := c.latencyQuantiles.Size()
	tpsSamples := len(c.tpsWindow)
	
	// Rough estimate of memory usage
	estimatedMB := float64(latencySamples*8 + tpsSamples*16) / (1024 * 1024)
	
	return ports.MemoryUsage{
		LatencySamplesCount: latencySamples,
		TPSSamplesCount:     tpsSamples,
		EstimatedMemoryMB:   estimatedMB,
	}
}

// Helper methods for metrics calculation

func (c *StreamingMetricsCollectorImpl) calculateEfficiencyMetrics(duration time.Duration, totalTxns int64) domain.EfficiencyMetrics {
	return domain.EfficiencyMetrics{
		TPSPerWorker:     float64(totalTxns) / duration.Seconds(), // Will be adjusted per worker
		TPSPerConnection: float64(totalTxns) / duration.Seconds(), // Will be adjusted per connection
		MarginalGain:     0.0, // Would be calculated comparing to previous band
		MarginalCost:     0.0, // Cost of additional resources
		ROI:              0.0, // Return on investment
	}
}

func (c *StreamingMetricsCollectorImpl) calculateStabilityMetrics() domain.StabilityMetrics {
	c.latencyMutex.RLock()
	variance := 0.0
	if c.latencyCount > 1 {
		variance = c.latencyM2 / float64(c.latencyCount-1)
	}
	cv := 0.0
	if c.latencyMean > 0 {
		cv = math.Sqrt(variance) / c.latencyMean
	}
	c.latencyMutex.RUnlock()
	
	tpsStdDev := c.calculateTPSStdDev()
	
	return domain.StabilityMetrics{
		TPSStdDev:              tpsStdDev,
		LatencyStdDev:          math.Sqrt(variance),
		CoefficientOfVariation: cv,
		TPSConfidenceInterval: domain.ConfidenceInterval{
			Lower:      0.0, // Would calculate based on distribution
			Upper:      0.0,
			Confidence: 0.95,
		},
		LatencyConfidenceInterval: domain.ConfidenceInterval{
			Lower:      0.0, // Would calculate based on distribution
			Upper:      0.0,
			Confidence: 0.95,
		},
		PerformanceDrift: 0.0, // Would track change over time
	}
}

func (c *StreamingMetricsCollectorImpl) calculateTPSStdDev() float64 {
	c.tpsMutex.RLock()
	defer c.tpsMutex.RUnlock()
	
	if len(c.tpsWindow) < 2 {
		return 0
	}
	
	// Calculate mean TPS
	var sum float64
	count := 0
	for _, point := range c.tpsWindow {
		if !point.Timestamp.IsZero() {
			sum += float64(point.Transactions)
			count++
		}
	}
	
	if count == 0 {
		return 0
	}
	
	mean := sum / float64(count)
	
	// Calculate variance
	var variance float64
	for _, point := range c.tpsWindow {
		if !point.Timestamp.IsZero() {
			diff := float64(point.Transactions) - mean
			variance += diff * diff
		}
	}
	
	variance /= float64(count)
	return math.Sqrt(variance)
}

func (c *StreamingMetricsCollectorImpl) calculateResourceMetrics() domain.ResourceMetrics {
	// Simplified resource metrics - in production, would integrate with system monitoring
	return domain.ResourceMetrics{
		ConnectionUtilization: 0.8, // Placeholder
		WorkerUtilization:     0.9, // Placeholder
		MemoryUsageMB:         c.GetMemoryUsage().EstimatedMemoryMB,
		CPUUtilization:        0.0, // Not implemented
	}
}

func (c *StreamingMetricsCollectorImpl) notifySnapshot() {
	if len(c.listeners) == 0 {
		return
	}
	
	snapshot := c.GetCurrentSnapshot()
	
	c.listenerMutex.RLock()
	defer c.listenerMutex.RUnlock()
	
	for _, listener := range c.listeners {
		go func(l ports.MetricsListener) {
			defer func() {
				if r := recover(); r != nil {
					// Log error in production
				}
			}()
			l.OnSnapshot(snapshot)
		}(listener)
	}
}

func (c *StreamingMetricsCollectorImpl) notifyBandComplete(results *domain.BandResults) {
	if len(c.listeners) == 0 {
		return
	}
	
	c.listenerMutex.RLock()
	defer c.listenerMutex.RUnlock()
	
	for _, listener := range c.listeners {
		go func(l ports.MetricsListener) {
			defer func() {
				if r := recover(); r != nil {
					// Log error in production
				}
			}()
			l.OnBandComplete(results.BandID, results)
		}(listener)
	}
}
