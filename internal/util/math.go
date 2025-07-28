// internal/util/math.go
package util

import (
	"math"
	"sort"

	"github.com/elchinoo/stormdb/pkg/types"
)

func CalculatePercentiles(data []int64, percentiles []int) []int64 {
	if len(data) == 0 {
		return []int64{0}
	}
	sort.Slice(data, func(i, j int) bool { return data[i] < data[j] })

	var result []int64
	n := len(data)
	for _, p := range percentiles {
		idx := (p * n) / 100
		if idx >= n {
			idx = n - 1
		}
		result = append(result, data[idx])
	}
	return result
}

func Stats(data []int64) (avg, minVal, maxVal, stddev int64) {
	if len(data) == 0 {
		return 0, 0, 0, 0
	}

	minVal, maxVal = data[0], data[0]
	var sum int64
	for _, v := range data {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
		sum += v
	}
	avg = sum / int64(len(data))

	var sumSq float64
	for _, v := range data {
		diff := float64(v - avg)
		sumSq += diff * diff
	}
	stddev = int64(math.Sqrt(sumSq / float64(len(data))))
	return avg, minVal, maxVal, stddev
}

// DistributionStats calculates comprehensive distribution shape metrics
type DistributionStats struct {
	P25      int64   // 25th percentile
	P75      int64   // 75th percentile
	IQR      int64   // Inter-quartile range (P75 - P25)
	MAD      float64 // Mean absolute deviation
	Skewness float64 // Measure of asymmetry
	Kurtosis float64 // Measure of tail heaviness
	CoV      float64 // Coefficient of variation (StdDev/Mean)
}

// CalculateDistributionStats computes advanced distribution shape metrics
func CalculateDistributionStats(data []int64) DistributionStats {
	if len(data) == 0 {
		return DistributionStats{}
	}

	// Sort data for percentile calculations
	sorted := make([]int64, len(data))
	copy(sorted, data)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	// Calculate P25 and P75
	n := len(sorted)
	p25Idx := (25 * n) / 100
	if p25Idx >= n {
		p25Idx = n - 1
	}
	p75Idx := (75 * n) / 100
	if p75Idx >= n {
		p75Idx = n - 1
	}

	p25 := sorted[p25Idx]
	p75 := sorted[p75Idx]
	iqr := p75 - p25

	// Calculate basic stats for other metrics
	avg, _, _, stddev := Stats(data)
	avgFloat := float64(avg)
	stddevFloat := float64(stddev)

	// Mean Absolute Deviation (MAD)
	var madSum float64
	for _, v := range data {
		madSum += math.Abs(float64(v) - avgFloat)
	}
	mad := madSum / float64(n)

	// Coefficient of Variation
	var cov float64
	if avgFloat != 0 {
		cov = stddevFloat / avgFloat
	}

	// Skewness and Kurtosis
	var skewSum, kurtSum float64
	for _, v := range data {
		deviation := float64(v) - avgFloat
		if stddevFloat != 0 {
			normalized := deviation / stddevFloat
			skewSum += normalized * normalized * normalized
			kurtSum += normalized * normalized * normalized * normalized
		}
	}

	var skewness, kurtosis float64
	if n > 0 {
		skewness = skewSum / float64(n)
		kurtosis = (kurtSum / float64(n)) - 3.0 // Subtract 3 for excess kurtosis
	}

	return DistributionStats{
		P25:      p25,
		P75:      p75,
		IQR:      iqr,
		MAD:      mad,
		Skewness: skewness,
		Kurtosis: kurtosis,
		CoV:      cov,
	}
}

// WorkerPerformanceStats holds performance statistics for a worker
type WorkerPerformanceStats struct {
	WorkerID    int
	TPS         float64
	QPS         float64
	SuccessRate float64
	P50Latency  float64 // in milliseconds
	P95Latency  float64 // in milliseconds
	AvgLatency  float64 // in milliseconds
	StdDev      float64 // in milliseconds
	CoV         float64 // Coefficient of variation
	ErrorCount  int64
}

// CalculateWorkerStats computes performance statistics for a worker
func CalculateWorkerStats(workerID int, tps, tpsAborted, qps, errors int64, latencies []int64, durationSec float64) WorkerPerformanceStats {
	totalTxns := tps + tpsAborted
	successRate := 100.0
	if totalTxns > 0 {
		successRate = float64(tps) / float64(totalTxns) * 100.0
	}

	var p50, p95, avg, stddev float64
	var cov float64

	if len(latencies) > 0 {
		// Calculate percentiles
		pvals := CalculatePercentiles(latencies, []int{50, 95})
		p50 = float64(pvals[0]) / 1e6 // Convert to ms
		p95 = float64(pvals[1]) / 1e6 // Convert to ms

		// Calculate basic stats
		avgNs, _, _, stddevNs := Stats(latencies)
		avg = float64(avgNs) / 1e6       // Convert to ms
		stddev = float64(stddevNs) / 1e6 // Convert to ms

		// Calculate coefficient of variation
		if avg != 0 {
			cov = stddev / avg
		}
	}

	return WorkerPerformanceStats{
		WorkerID:    workerID,
		TPS:         float64(tps) / durationSec,
		QPS:         float64(qps) / durationSec,
		SuccessRate: successRate,
		P50Latency:  p50,
		P95Latency:  p95,
		AvgLatency:  avg,
		StdDev:      stddev,
		CoV:         cov,
		ErrorCount:  errors,
	}
}

// TimeSeriesStats holds time-series analysis results
type TimeSeriesStats struct {
	PearsonCorrelation   float64      // Correlation between QPS and latency
	SpearmanCorrelation  float64      // Rank-based correlation
	LatencySlope         float64      // ms extra per 100 QPS increase
	PeakQPS              float64      // Maximum QPS observed
	MedianQPS            float64      // Median QPS
	PeakLatency          float64      // Maximum average latency
	MedianLatency        float64      // Median average latency
	LoadStabilityRegions []LoadRegion // Identified stability regions
}

// LoadRegion represents a region of stable or degrading performance
type LoadRegion struct {
	StartBucket     int
	EndBucket       int
	QPSRange        [2]float64 // [min, max] QPS
	LatencyRange    [2]float64 // [min, max] avg latency
	IsStable        bool       // True if latency is stable despite QPS changes
	DegradationRate float64    // Rate of latency increase (ms per 100 QPS)
}

// AnalyzeTimeSeries performs comprehensive time-series analysis
func AnalyzeTimeSeries(buckets []types.TimeBucket) TimeSeriesStats {
	if len(buckets) < 2 {
		return TimeSeriesStats{}
	}

	// Extract QPS and average latency series
	qpsSeries := make([]float64, len(buckets))
	latencySeries := make([]float64, len(buckets))

	for i, bucket := range buckets {
		duration := bucket.EndTime.Sub(bucket.StartTime).Seconds()
		qpsSeries[i] = float64(bucket.QPS) / duration

		// Calculate average latency for this bucket
		if len(bucket.Latencies) > 0 {
			var sum int64
			for _, lat := range bucket.Latencies {
				sum += lat
			}
			latencySeries[i] = float64(sum) / float64(len(bucket.Latencies)) / 1e6 // Convert to ms
		}
	}

	// Calculate correlations
	pearson := calculatePearsonCorr(qpsSeries, latencySeries)
	spearman := calculateSpearmanCorr(qpsSeries, latencySeries)

	// Calculate linear regression slope (ms per 100 QPS)
	slope := calculateLinearSlope(qpsSeries, latencySeries) * 100

	// Find peak and median values
	peakQPS := findMax(qpsSeries)
	medianQPS := calculateMedian(qpsSeries)
	peakLatency := findMax(latencySeries)
	medianLatency := calculateMedian(latencySeries)

	// Identify load stability regions
	regions := identifyLoadRegions(qpsSeries, latencySeries)

	return TimeSeriesStats{
		PearsonCorrelation:   pearson,
		SpearmanCorrelation:  spearman,
		LatencySlope:         slope,
		PeakQPS:              peakQPS,
		MedianQPS:            medianQPS,
		PeakLatency:          peakLatency,
		MedianLatency:        medianLatency,
		LoadStabilityRegions: regions,
	}
}

// calculatePearsonCorr calculates Pearson correlation coefficient
func calculatePearsonCorr(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0.0
	}

	// Calculate means
	meanX := calculateMean(x)
	meanY := calculateMean(y)

	// Calculate correlation
	var num, denX, denY float64
	for i := range x {
		dx := x[i] - meanX
		dy := y[i] - meanY
		num += dx * dy
		denX += dx * dx
		denY += dy * dy
	}

	if denX == 0 || denY == 0 {
		return 0.0
	}

	return num / math.Sqrt(denX*denY)
}

// calculateSpearmanCorr calculates Spearman rank correlation
func calculateSpearmanCorr(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0.0
	}

	// Convert to ranks
	xRanks := convertToRanks(x)
	yRanks := convertToRanks(y)

	// Calculate Pearson correlation of ranks
	return calculatePearsonCorr(xRanks, yRanks)
}

// calculateLinearSlope calculates the slope of linear regression
func calculateLinearSlope(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0.0
	}

	meanX := calculateMean(x)
	meanY := calculateMean(y)

	var num, den float64
	for i := range x {
		dx := x[i] - meanX
		num += dx * (y[i] - meanY)
		den += dx * dx
	}

	if den == 0 {
		return 0.0
	}

	return num / den
}

// Helper functions
func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2.0
	}
	return sorted[n/2]
}

func findMax(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	maxVal := values[0]
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func convertToRanks(values []float64) []float64 {
	n := len(values)
	if n == 0 {
		return nil
	}

	// Create value-index pairs
	type ValueIndex struct {
		Value float64
		Index int
	}

	pairs := make([]ValueIndex, n)
	for i, v := range values {
		pairs[i] = ValueIndex{Value: v, Index: i}
	}

	// Sort by value
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value < pairs[j].Value
	})

	// Assign ranks
	ranks := make([]float64, n)
	for i, pair := range pairs {
		ranks[pair.Index] = float64(i + 1)
	}

	return ranks
}

func identifyLoadRegions(qps, latency []float64) []LoadRegion {
	// Simple implementation: identify regions where latency is stable vs growing
	regions := make([]LoadRegion, 0)

	if len(qps) < 3 {
		return regions
	}

	// Look for regions of 3+ consecutive buckets with similar behavior
	windowSize := 3
	for i := 0; i <= len(qps)-windowSize; i++ {
		qpsWindow := qps[i : i+windowSize]
		latWindow := latency[i : i+windowSize]

		// Calculate trend in this window
		slope := calculateLinearSlope(qpsWindow, latWindow)

		// Determine if region is stable (slope < 0.1 ms per QPS)
		isStable := math.Abs(slope) < 0.1

		region := LoadRegion{
			StartBucket:     i,
			EndBucket:       i + windowSize - 1,
			QPSRange:        [2]float64{findMin(qpsWindow), findMax(qpsWindow)},
			LatencyRange:    [2]float64{findMin(latWindow), findMax(latWindow)},
			IsStable:        isStable,
			DegradationRate: slope * 100, // Per 100 QPS
		}

		regions = append(regions, region)
	}

	return regions
}

func findMin(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	minVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}
