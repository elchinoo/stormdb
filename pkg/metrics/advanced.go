// Package metrics provides advanced statistical analysis and metrics calculation
package metrics

import (
	"fmt"
	"math"
	"sort"

	"github.com/elchinoo/stormdb/pkg/types"
)

// StatisticalResult represents the result of statistical significance testing
type StatisticalResult struct {
	TStatistic         float64            `json:"t_statistic"`
	PValue             float64            `json:"p_value"`
	IsSignificant      bool               `json:"is_significant"`
	DegreesOfFreedom   float64            `json:"degrees_of_freedom"`
	ConfidenceInterval ConfidenceInterval `json:"confidence_interval"`
}

// ConfidenceInterval represents a confidence interval
type ConfidenceInterval struct {
	Lower      float64 `json:"lower"`
	Upper      float64 `json:"upper"`
	Confidence float64 `json:"confidence"` // e.g., 0.95 for 95%
}

// ElasticityResult represents elasticity coefficient analysis
type ElasticityResult struct {
	Segment        string  `json:"segment"`
	DeltaTPS       float64 `json:"delta_tps"`
	BaselineTPS    float64 `json:"baseline_tps"`
	DeltaConns     int     `json:"delta_connections"`
	BaselineConns  int     `json:"baseline_connections"`
	Elasticity     float64 `json:"elasticity"`
	Interpretation string  `json:"interpretation"`
}

// QueueMetrics represents queueing theory analysis
type QueueMetrics struct {
	Connections   int     `json:"connections"`
	TPS           float64 `json:"tps"`
	LatencyP99    float64 `json:"latency_p99_ms"`
	Utilization   float64 `json:"utilization"`
	TotalRequests float64 `json:"total_requests_in_system"`
	QueueLength   float64 `json:"queue_length"`
	ServiceRate   float64 `json:"service_rate"`
	ArrivalRate   float64 `json:"arrival_rate"`
}

// CostBenefitResult represents cost-benefit analysis
type CostBenefitResult struct {
	Connections      int     `json:"connections"`
	Throughput       float64 `json:"throughput_tps"`
	ThroughputPct    float64 `json:"throughput_percentage"`
	Latency          float64 `json:"latency_p99_ms"`
	LatencyCost      float64 `json:"latency_cost_percentage"`
	BenefitCostRatio float64 `json:"benefit_cost_ratio"`
	Recommendation   string  `json:"recommendation"`
}

// BandResults represents results for a single band/connection level
type BandResults struct {
	Connections int            `json:"connections"`
	AvgTPS      float64        `json:"avg_tps"`
	StdDev      float64        `json:"std_dev"`
	LatencyP50  float64        `json:"latency_p50"`
	LatencyP95  float64        `json:"latency_p95"`
	LatencyP99  float64        `json:"latency_p99"`
	Samples     int            `json:"samples"`
	Metrics     *types.Metrics `json:"metrics,omitempty"`
}

// AdvancedAnalyzer provides comprehensive statistical analysis
type AdvancedAnalyzer struct {
	confidenceLevel float64
	alpha           float64
}

// NewAdvancedAnalyzer creates a new analyzer with the specified confidence level
func NewAdvancedAnalyzer(confidenceLevel float64) *AdvancedAnalyzer {
	return &AdvancedAnalyzer{
		confidenceLevel: confidenceLevel,
		alpha:           1.0 - confidenceLevel,
	}
}

// IsSignificantDifference performs Welch's t-test for unequal variances
func (a *AdvancedAnalyzer) IsSignificantDifference(band1, band2 BandResults) StatisticalResult {
	// Calculate standard errors
	se1 := band1.StdDev / math.Sqrt(float64(band1.Samples))
	se2 := band2.StdDev / math.Sqrt(float64(band2.Samples))
	seDiff := math.Sqrt(se1*se1 + se2*se2)

	// Calculate t-statistic
	tStat := (band2.AvgTPS - band1.AvgTPS) / seDiff

	// Calculate degrees of freedom using Welch-Satterthwaite equation
	df := a.calculateDegreesOfFreedom(se1, se2, band1.Samples, band2.Samples)

	// Calculate p-value (two-tailed test)
	pValue := 2 * (1 - a.studentTCDF(math.Abs(tStat), df))

	// Calculate confidence interval for the difference
	tCritical := a.studentTInverse(a.alpha/2, df)
	margin := tCritical * seDiff
	meanDiff := band2.AvgTPS - band1.AvgTPS

	ci := ConfidenceInterval{
		Lower:      meanDiff - margin,
		Upper:      meanDiff + margin,
		Confidence: a.confidenceLevel,
	}

	return StatisticalResult{
		TStatistic:         tStat,
		PValue:             pValue,
		IsSignificant:      pValue < a.alpha,
		DegreesOfFreedom:   df,
		ConfidenceInterval: ci,
	}
}

// CalculateElasticity computes elasticity coefficients for performance scaling
func (a *AdvancedAnalyzer) CalculateElasticity(bands []BandResults) []ElasticityResult {
	results := make([]ElasticityResult, len(bands)-1)

	for i := 1; i < len(bands); i++ {
		deltaTPS := bands[i].AvgTPS - bands[i-1].AvgTPS
		deltaConns := bands[i].Connections - bands[i-1].Connections

		// Elasticity = (% change in throughput) / (% change in connections)
		throughputChange := deltaTPS / bands[i-1].AvgTPS
		connectionChange := float64(deltaConns) / float64(bands[i-1].Connections)

		elasticity := throughputChange / connectionChange

		results[i-1] = ElasticityResult{
			Segment:        fmt.Sprintf("%d→%d", bands[i-1].Connections, bands[i].Connections),
			DeltaTPS:       deltaTPS,
			BaselineTPS:    bands[i-1].AvgTPS,
			DeltaConns:     deltaConns,
			BaselineConns:  bands[i-1].Connections,
			Elasticity:     elasticity,
			Interpretation: a.interpretElasticity(elasticity),
		}
	}

	return results
}

// CalculateQueueMetrics performs queueing theory analysis
func (a *AdvancedAnalyzer) CalculateQueueMetrics(bands []BandResults, maxTPS float64) []QueueMetrics {
	metrics := make([]QueueMetrics, len(bands))

	for i, band := range bands {
		// Convert P99 latency from ms to seconds
		latencySec := band.LatencyP99 / 1000.0

		// Service rate (μ) - maximum possible throughput
		serviceRate := maxTPS

		// Arrival rate (λ) - actual throughput
		arrivalRate := band.AvgTPS

		// Utilization (ρ = λ/μ)
		utilization := arrivalRate / serviceRate

		// Total requests in system using Little's Law (L = λ × W)
		totalRequests := arrivalRate * latencySec

		// Queue length (Lq = L - ρ)
		queueLength := totalRequests - utilization

		metrics[i] = QueueMetrics{
			Connections:   band.Connections,
			TPS:           band.AvgTPS,
			LatencyP99:    band.LatencyP99,
			Utilization:   utilization,
			TotalRequests: totalRequests,
			QueueLength:   math.Max(0, queueLength), // Can't be negative
			ServiceRate:   serviceRate,
			ArrivalRate:   arrivalRate,
		}
	}

	return metrics
}

// CalculateCostBenefit performs cost-benefit analysis
func (a *AdvancedAnalyzer) CalculateCostBenefit(bands []BandResults) []CostBenefitResult {
	if len(bands) == 0 {
		return []CostBenefitResult{}
	}

	results := make([]CostBenefitResult, len(bands))

	// Find peak TPS for normalization
	peakTPS := 0.0
	for _, band := range bands {
		if band.AvgTPS > peakTPS {
			peakTPS = band.AvgTPS
		}
	}

	// Use the band with the best latency as baseline (usually the first one)
	baselineLatency := bands[0].LatencyP99

	for i, band := range bands {
		// Calculate throughput percentage of peak
		throughputPct := (band.AvgTPS / peakTPS) * 100.0

		// Calculate latency cost compared to baseline
		latencyCost := ((band.LatencyP99 - baselineLatency) / baselineLatency) * 100.0

		// Calculate benefit/cost ratio
		var benefitCostRatio float64
		if latencyCost > 0 {
			benefitCostRatio = throughputPct / (100.0 + latencyCost)
		} else {
			benefitCostRatio = throughputPct / 100.0
		}

		results[i] = CostBenefitResult{
			Connections:      band.Connections,
			Throughput:       band.AvgTPS,
			ThroughputPct:    throughputPct,
			Latency:          band.LatencyP99,
			LatencyCost:      latencyCost,
			BenefitCostRatio: benefitCostRatio,
			Recommendation:   a.determineRecommendation(benefitCostRatio, throughputPct, latencyCost),
		}
	}

	return results
}

// CalculateConfidenceInterval calculates confidence interval for a metric
func (a *AdvancedAnalyzer) CalculateConfidenceInterval(mean, stdDev float64, samples int) ConfidenceInterval {
	se := stdDev / math.Sqrt(float64(samples))
	df := float64(samples - 1)

	// Use t-distribution for small samples, normal for large samples
	var critical float64
	if samples >= 30 {
		critical = a.normalInverse(a.alpha / 2)
	} else {
		critical = a.studentTInverse(a.alpha/2, df)
	}

	margin := critical * se

	return ConfidenceInterval{
		Lower:      mean - margin,
		Upper:      mean + margin,
		Confidence: a.confidenceLevel,
	}
}

// Private helper methods

func (a *AdvancedAnalyzer) calculateDegreesOfFreedom(se1, se2 float64, n1, n2 int) float64 {
	// Welch-Satterthwaite equation
	numerator := math.Pow(se1*se1+se2*se2, 2)
	denominator := (math.Pow(se1, 4) / float64(n1-1)) + (math.Pow(se2, 4) / float64(n2-1))
	return numerator / denominator
}

func (a *AdvancedAnalyzer) interpretElasticity(e float64) string {
	switch {
	case e > 1.0:
		return "Super-linear scaling - excellent returns"
	case e >= 0.8:
		return "Near-perfect scaling - high returns"
	case e >= 0.5:
		return "Good scaling - moderate returns"
	case e >= 0.2:
		return "Diminishing returns - limited scaling"
	case e > 0:
		return "Significant diminishing returns"
	default:
		return "Negative scaling - performance degradation"
	}
}

func (a *AdvancedAnalyzer) determineRecommendation(ratio, throughputPct, latencyCost float64) string {
	switch {
	case ratio > 0.8 && latencyCost < 50:
		return "Excellent - High throughput with acceptable latency"
	case ratio > 0.6 && latencyCost < 100:
		return "Good - Balanced throughput and latency trade-off"
	case ratio > 0.4:
		return "Fair - Consider if latency requirements allow"
	case latencyCost > 200:
		return "Poor - Latency cost too high for minimal throughput gain"
	default:
		return "Suboptimal - Diminishing returns"
	}
}

// Statistical distribution functions (simplified implementations)

func (a *AdvancedAnalyzer) studentTCDF(t, df float64) float64 {
	// Simplified Student's t CDF approximation
	// For production use, consider using a proper statistical library
	if df >= 30 {
		return a.normalCDF(t)
	}

	// Approximation for smaller degrees of freedom
	x := t / math.Sqrt(df)
	return 0.5 + (x*(1-x*x/6))/(2*math.Sqrt(math.Pi))
}

func (a *AdvancedAnalyzer) normalCDF(x float64) float64 {
	// Standard normal CDF approximation
	return 0.5 * (1 + a.erf(x/math.Sqrt2))
}

func (a *AdvancedAnalyzer) erf(x float64) float64 {
	// Error function approximation
	a1, a2, a3, a4, a5 := 0.254829592, -0.284496736, 1.421413741, -1.453152027, 1.061405429
	p := 0.3275911

	sign := 1.0
	if x < 0 {
		sign = -1
		x = -x
	}

	t := 1.0 / (1.0 + p*x)
	y := 1.0 - (((((a5*t+a4)*t)+a3)*t+a2)*t+a1)*t*math.Exp(-x*x)

	return sign * y
}

func (a *AdvancedAnalyzer) studentTInverse(p, df float64) float64 {
	// Simplified t-distribution inverse CDF
	// For production use, consider using a proper statistical library
	if df >= 30 {
		return a.normalInverse(p)
	}

	// Approximation for t-distribution
	z := a.normalInverse(p)
	correction := (z*z*z + z) / (4 * df)
	return z + correction
}

func (a *AdvancedAnalyzer) normalInverse(p float64) float64 {
	// Standard normal inverse CDF approximation (Box-Muller transformation simplified)
	if p <= 0 || p >= 1 {
		return 0
	}

	// Use rational approximation for the inverse
	c0, c1, c2 := 2.515517, 0.802853, 0.010328
	d1, d2, d3 := 1.432788, 0.189269, 0.001308

	if p > 0.5 {
		p = 1 - p
	}

	t := math.Sqrt(-2 * math.Log(p))
	result := t - (c0+c1*t+c2*t*t)/(1+d1*t+d2*t*t+d3*t*t*t)

	if p <= 0.5 {
		return result
	}
	return -result
}

// FindOptimalConnectionLevel finds the connection level with the best benefit/cost ratio
func (a *AdvancedAnalyzer) FindOptimalConnectionLevel(costBenefit []CostBenefitResult) *CostBenefitResult {
	if len(costBenefit) == 0 {
		return nil
	}

	optimal := &costBenefit[0]
	for i := 1; i < len(costBenefit); i++ {
		if costBenefit[i].BenefitCostRatio > optimal.BenefitCostRatio {
			optimal = &costBenefit[i]
		}
	}

	return optimal
}

// CalculatePercentiles calculates percentiles from a slice of values
func CalculatePercentiles(values []float64, percentiles []float64) map[float64]float64 {
	if len(values) == 0 {
		return make(map[float64]float64)
	}

	// Sort values
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	result := make(map[float64]float64)

	for _, p := range percentiles {
		if p < 0 || p > 100 {
			continue
		}

		index := (p / 100.0) * float64(len(sorted)-1)
		lower := int(math.Floor(index))
		upper := int(math.Ceil(index))

		if lower == upper {
			result[p] = sorted[lower]
		} else {
			// Linear interpolation
			weight := index - float64(lower)
			result[p] = sorted[lower]*(1-weight) + sorted[upper]*weight
		}
	}

	return result
}

// CalculateStandardDeviation calculates standard deviation of values
func CalculateStandardDeviation(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// Calculate variance
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values) - 1) // Sample standard deviation

	return math.Sqrt(variance)
}
