package progressive

import (
	"fmt"
	"math"
	"sort"

	"github.com/elchinoo/stormdb/internal/logging"
	"go.uber.org/zap"
)

// AnalyticsEngine performs advanced mathematical analysis on progressive test results
type AnalyticsEngine struct {
	logger logging.StormDBLogger
}

// NewAnalyticsEngine creates a new analytics engine
func NewAnalyticsEngine(logger logging.StormDBLogger) *AnalyticsEngine {
	return &AnalyticsEngine{
		logger: logger.With(zap.String("component", "analytics_engine")),
	}
}

// AnalysisResult contains comprehensive analysis of progressive test results
type AnalysisResult struct {
	Summary             AnalysisSummary     `json:"summary"`
	StatisticalAnalysis StatisticalAnalysis `json:"statistical_analysis"`
	TrendAnalysis       TrendAnalysis       `json:"trend_analysis"`
	QueueingAnalysis    QueueingAnalysis    `json:"queueing_analysis"`
	ScalabilityAnalysis ScalabilityAnalysis `json:"scalability_analysis"`
	Recommendations     []string            `json:"recommendations"`
}

// AnalysisSummary provides high-level summary statistics
type AnalysisSummary struct {
	TotalBands       int     `json:"total_bands"`
	OptimalBand      int     `json:"optimal_band"`
	MaxThroughput    float64 `json:"max_throughput"`
	BestLatencyBand  int     `json:"best_latency_band"`
	MinLatencyP95    float64 `json:"min_latency_p95"`
	OverallStability float64 `json:"overall_stability"`
	ScalabilityScore float64 `json:"scalability_score"`
}

// StatisticalAnalysis contains detailed statistical metrics
type StatisticalAnalysis struct {
	ThroughputStats   DescriptiveStats  `json:"throughput_stats"`
	LatencyStats      DescriptiveStats  `json:"latency_stats"`
	ErrorRateStats    DescriptiveStats  `json:"error_rate_stats"`
	CorrelationMatrix CorrelationMatrix `json:"correlation_matrix"`
	Outliers          OutlierAnalysis   `json:"outliers"`
}

// DescriptiveStats contains standard statistical measures
type DescriptiveStats struct {
	Mean                   float64            `json:"mean"`
	Median                 float64            `json:"median"`
	Mode                   float64            `json:"mode"`
	StandardDeviation      float64            `json:"standard_deviation"`
	Variance               float64            `json:"variance"`
	CoefficientOfVariation float64            `json:"coefficient_of_variation"`
	Skewness               float64            `json:"skewness"`
	Kurtosis               float64            `json:"kurtosis"`
	ConfidenceInterval95   ConfidenceInterval `json:"confidence_interval_95"`
	ConfidenceInterval99   ConfidenceInterval `json:"confidence_interval_99"`
}

// TrendAnalysis contains trend and derivative analysis
type TrendAnalysis struct {
	ThroughputTrend    TrendData      `json:"throughput_trend"`
	LatencyTrend       TrendData      `json:"latency_trend"`
	ErrorRateTrend     TrendData      `json:"error_rate_trend"`
	DerivativeAnalysis DerivativeData `json:"derivative_analysis"`
	IntegralAnalysis   IntegralData   `json:"integral_analysis"`
	RegressionAnalysis RegressionData `json:"regression_analysis"`
}

// TrendData contains trend information
type TrendData struct {
	Direction     string  `json:"direction"` // "increasing", "decreasing", "stable", "volatile"
	Slope         float64 `json:"slope"`
	RSquared      float64 `json:"r_squared"`
	TrendStrength float64 `json:"trend_strength"` // 0.0 to 1.0
	ChangePoints  []int   `json:"change_points"`
}

// DerivativeData contains derivative analysis
type DerivativeData struct {
	FirstDerivative  []float64 `json:"first_derivative"`
	SecondDerivative []float64 `json:"second_derivative"`
	MaxAcceleration  float64   `json:"max_acceleration"`
	MaxDeceleration  float64   `json:"max_deceleration"`
	InflectionPoints []int     `json:"inflection_points"`
}

// IntegralData contains integral analysis
type IntegralData struct {
	ThroughputIntegral   float64 `json:"throughput_integral"`
	LatencyIntegral      float64 `json:"latency_integral"`
	AreaUnderCurve       float64 `json:"area_under_curve"`
	CumulativeEfficiency float64 `json:"cumulative_efficiency"`
}

// RegressionData contains curve fitting results
type RegressionData struct {
	LinearRegression     LinearFit      `json:"linear_regression"`
	PolynomialRegression PolynomialFit  `json:"polynomial_regression"`
	ExponentialFit       ExponentialFit `json:"exponential_fit"`
	LogarithmicFit       LogarithmicFit `json:"logarithmic_fit"`
	BestFitModel         string         `json:"best_fit_model"`
	BestFitR2            float64        `json:"best_fit_r2"`
}

// LinearFit contains linear regression results
type LinearFit struct {
	Slope     float64 `json:"slope"`
	Intercept float64 `json:"intercept"`
	RSquared  float64 `json:"r_squared"`
	PValue    float64 `json:"p_value"`
	Equation  string  `json:"equation"`
}

// PolynomialFit contains polynomial regression results
type PolynomialFit struct {
	Coefficients []float64 `json:"coefficients"`
	Degree       int       `json:"degree"`
	RSquared     float64   `json:"r_squared"`
	Equation     string    `json:"equation"`
}

// ExponentialFit contains exponential curve fitting results
type ExponentialFit struct {
	A        float64 `json:"a"` // y = A * e^(B*x)
	B        float64 `json:"b"`
	RSquared float64 `json:"r_squared"`
	Equation string  `json:"equation"`
}

// LogarithmicFit contains logarithmic curve fitting results
type LogarithmicFit struct {
	A        float64 `json:"a"` // y = A * ln(x) + B
	B        float64 `json:"b"`
	RSquared float64 `json:"r_squared"`
	Equation string  `json:"equation"`
}

// QueueingAnalysis contains queueing theory analysis
type QueueingAnalysis struct {
	QueueingModel        string  `json:"queueing_model"`
	ServiceRate          float64 `json:"service_rate"`
	ArrivalRate          float64 `json:"arrival_rate"`
	UtilizationFactor    float64 `json:"utilization_factor"`
	AverageQueueLength   float64 `json:"average_queue_length"`
	AverageWaitTime      float64 `json:"average_wait_time"`
	AverageSystemTime    float64 `json:"average_system_time"`
	ThroughputCapacity   float64 `json:"throughput_capacity"`
	SaturationPoint      float64 `json:"saturation_point"`
	LittlesLawValidation bool    `json:"littles_law_validation"`
}

// ScalabilityAnalysis contains scalability metrics
type ScalabilityAnalysis struct {
	LinearScalabilityScore float64                 `json:"linear_scalability_score"`
	ScalabilityBreakpoints []ScalabilityBreakpoint `json:"scalability_breakpoints"`
	OptimalConnectionRange ConnectionRange         `json:"optimal_connection_range"`
	EfficiencyMetrics      EfficiencyMetrics       `json:"efficiency_metrics"`
	BottleneckAnalysis     BottleneckAnalysis      `json:"bottleneck_analysis"`
}

// ScalabilityBreakpoint identifies points where scalability changes
type ScalabilityBreakpoint struct {
	BandID      int     `json:"band_id"`
	Connections int     `json:"connections"`
	Type        string  `json:"type"` // "knee", "cliff", "plateau"
	Impact      float64 `json:"impact"`
	Description string  `json:"description"`
}

// ConnectionRange defines optimal connection ranges
type ConnectionRange struct {
	Min        int     `json:"min"`
	Max        int     `json:"max"`
	Optimal    int     `json:"optimal"`
	Confidence float64 `json:"confidence"`
}

// EfficiencyMetrics contains efficiency calculations
type EfficiencyMetrics struct {
	ThroughputPerConnection float64 `json:"throughput_per_connection"`
	LatencyEfficiency       float64 `json:"latency_efficiency"`
	ResourceUtilization     float64 `json:"resource_utilization"`
	CostEfficiency          float64 `json:"cost_efficiency"`
}

// BottleneckAnalysis identifies performance bottlenecks
type BottleneckAnalysis struct {
	PrimaryBottleneck    string   `json:"primary_bottleneck"`
	BottleneckFactors    []string `json:"bottleneck_factors"`
	ImpactAssessment     string   `json:"impact_assessment"`
	MitigationStrategies []string `json:"mitigation_strategies"`
}

// CorrelationMatrix contains correlation coefficients between metrics
type CorrelationMatrix struct {
	ConnectionsVsThroughput float64 `json:"connections_vs_throughput"`
	ConnectionsVsLatency    float64 `json:"connections_vs_latency"`
	ThroughputVsLatency     float64 `json:"throughput_vs_latency"`
	ErrorRateVsLatency      float64 `json:"error_rate_vs_latency"`
	ErrorRateVsThroughput   float64 `json:"error_rate_vs_throughput"`
}

// OutlierAnalysis identifies statistical outliers
type OutlierAnalysis struct {
	ThroughputOutliers []OutlierData `json:"throughput_outliers"`
	LatencyOutliers    []OutlierData `json:"latency_outliers"`
	ErrorOutliers      []OutlierData `json:"error_outliers"`
}

// OutlierData contains information about detected outliers
type OutlierData struct {
	BandID   int     `json:"band_id"`
	Value    float64 `json:"value"`
	ZScore   float64 `json:"z_score"`
	Severity string  `json:"severity"` // "mild", "moderate", "extreme"
}

// Analyze performs comprehensive analysis on progressive test results
func (ae *AnalyticsEngine) Analyze(results []BandResult) error {
	if len(results) < 2 {
		return fmt.Errorf("need at least 2 band results for analysis")
	}

	ae.logger.Info("Starting comprehensive analysis",
		zap.Int("band_count", len(results)),
	)

	// Extract time series data
	throughputSeries := ae.extractThroughputSeries(results)
	latencySeries := ae.extractLatencySeries(results)
	errorRateSeries := ae.extractErrorRateSeries(results)
	connectionSeries := ae.extractConnectionSeries(results)

	// Perform statistical analysis
	statsAnalysis := ae.performStatisticalAnalysis(throughputSeries, latencySeries, errorRateSeries)

	// Perform trend analysis
	trendAnalysis := ae.performTrendAnalysis(throughputSeries, latencySeries, errorRateSeries)

	// Perform queueing analysis
	queueingAnalysis := ae.performQueueingAnalysis(results)

	// Perform scalability analysis
	scalabilityAnalysis := ae.performScalabilityAnalysis(connectionSeries, throughputSeries, latencySeries)

	// Generate summary
	summary := ae.generateSummary(results, throughputSeries, latencySeries)

	// Generate recommendations
	recommendations := ae.generateRecommendations(statsAnalysis, trendAnalysis, queueingAnalysis, scalabilityAnalysis)

	analysisResult := AnalysisResult{
		Summary:             summary,
		StatisticalAnalysis: statsAnalysis,
		TrendAnalysis:       trendAnalysis,
		QueueingAnalysis:    queueingAnalysis,
		ScalabilityAnalysis: scalabilityAnalysis,
		Recommendations:     recommendations,
	}

	ae.logger.Info("Analysis completed",
		zap.Float64("scalability_score", analysisResult.Summary.ScalabilityScore),
		zap.Int("recommendations_count", len(analysisResult.Recommendations)),
	)

	return nil
}

// extractThroughputSeries extracts throughput data points
func (ae *AnalyticsEngine) extractThroughputSeries(results []BandResult) []float64 {
	series := make([]float64, len(results))
	for i, result := range results {
		if result.Metrics != nil {
			series[i] = result.Metrics.AvgTPS
		}
	}
	return series
}

// extractLatencySeries extracts latency data points
func (ae *AnalyticsEngine) extractLatencySeries(results []BandResult) []float64 {
	series := make([]float64, len(results))
	for i, result := range results {
		if result.Metrics != nil {
			series[i] = result.Metrics.LatencyP95
		}
	}
	return series
}

// extractErrorRateSeries extracts error rate data points
func (ae *AnalyticsEngine) extractErrorRateSeries(results []BandResult) []float64 {
	series := make([]float64, len(results))
	for i, result := range results {
		if result.Metrics != nil {
			series[i] = result.Metrics.ErrorRate
		}
	}
	return series
}

// extractConnectionSeries extracts connection count data points
func (ae *AnalyticsEngine) extractConnectionSeries(results []BandResult) []float64 {
	series := make([]float64, len(results))
	for i, result := range results {
		series[i] = float64(result.BandConfig.Connections)
	}
	return series
}

// performStatisticalAnalysis calculates comprehensive statistics
func (ae *AnalyticsEngine) performStatisticalAnalysis(throughput, latency, errorRate []float64) StatisticalAnalysis {
	return StatisticalAnalysis{
		ThroughputStats:   ae.calculateDescriptiveStats(throughput),
		LatencyStats:      ae.calculateDescriptiveStats(latency),
		ErrorRateStats:    ae.calculateDescriptiveStats(errorRate),
		CorrelationMatrix: ae.calculateCorrelationMatrix(throughput, latency, errorRate),
		Outliers:          ae.detectOutliers(throughput, latency, errorRate),
	}
}

// calculateDescriptiveStats calculates comprehensive descriptive statistics
func (ae *AnalyticsEngine) calculateDescriptiveStats(data []float64) DescriptiveStats {
	if len(data) == 0 {
		return DescriptiveStats{}
	}

	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)

	// Basic statistics
	mean := ae.mean(data)
	variance := ae.variance(data, mean)
	stdDev := math.Sqrt(variance)
	median := ae.median(sorted)

	stats := DescriptiveStats{
		Mean:              mean,
		Median:            median,
		StandardDeviation: stdDev,
		Variance:          variance,
	}

	// Coefficient of variation
	if mean != 0 {
		stats.CoefficientOfVariation = stdDev / math.Abs(mean)
	}

	// Skewness and kurtosis
	stats.Skewness = ae.skewness(data, mean, stdDev)
	stats.Kurtosis = ae.kurtosis(data, mean, stdDev)

	// Confidence intervals
	if len(data) > 1 {
		sem := stdDev / math.Sqrt(float64(len(data)))

		// 95% CI
		margin95 := 1.96 * sem
		stats.ConfidenceInterval95 = ConfidenceInterval{
			Lower: mean - margin95,
			Upper: mean + margin95,
			Mean:  mean,
		}

		// 99% CI
		margin99 := 2.576 * sem
		stats.ConfidenceInterval99 = ConfidenceInterval{
			Lower: mean - margin99,
			Upper: mean + margin99,
			Mean:  mean,
		}
	}

	return stats
}

// performTrendAnalysis analyzes trends and derivatives
func (ae *AnalyticsEngine) performTrendAnalysis(throughput, latency, errorRate []float64) TrendAnalysis {
	return TrendAnalysis{
		ThroughputTrend:    ae.analyzeTrend(throughput),
		LatencyTrend:       ae.analyzeTrend(latency),
		ErrorRateTrend:     ae.analyzeTrend(errorRate),
		DerivativeAnalysis: ae.calculateDerivatives(throughput),
		IntegralAnalysis:   ae.calculateIntegrals(throughput, latency),
		RegressionAnalysis: ae.performRegressionAnalysis(throughput),
	}
}

// performQueueingAnalysis applies queueing theory
func (ae *AnalyticsEngine) performQueueingAnalysis(results []BandResult) QueueingAnalysis {
	if len(results) == 0 {
		return QueueingAnalysis{}
	}

	// Calculate average service rate and arrival rate
	totalThroughput := 0.0
	totalLatency := 0.0
	validResults := 0

	for _, result := range results {
		if result.Metrics != nil && result.Metrics.AvgTPS > 0 {
			totalThroughput += result.Metrics.AvgTPS
			totalLatency += result.Metrics.LatencyMean
			validResults++
		}
	}

	if validResults == 0 {
		return QueueingAnalysis{}
	}

	avgThroughput := totalThroughput / float64(validResults)
	avgLatency := totalLatency / float64(validResults) / 1000.0 // Convert to seconds

	// Simple M/M/1 queueing model assumptions
	serviceRate := avgThroughput / avgLatency // λ/W approximation
	arrivalRate := avgThroughput
	utilizationFactor := arrivalRate / serviceRate

	// M/M/1 formulas
	avgQueueLength := (utilizationFactor * utilizationFactor) / (1 - utilizationFactor)
	avgWaitTime := avgQueueLength / arrivalRate
	avgSystemTime := avgWaitTime + (1 / serviceRate)

	return QueueingAnalysis{
		QueueingModel:        "M/M/1",
		ServiceRate:          serviceRate,
		ArrivalRate:          arrivalRate,
		UtilizationFactor:    utilizationFactor,
		AverageQueueLength:   avgQueueLength,
		AverageWaitTime:      avgWaitTime,
		AverageSystemTime:    avgSystemTime,
		ThroughputCapacity:   serviceRate * 0.8, // 80% capacity recommendation
		SaturationPoint:      serviceRate,
		LittlesLawValidation: math.Abs(avgQueueLength-(arrivalRate*avgSystemTime)) < 0.1,
	}
}

// performScalabilityAnalysis analyzes scalability characteristics
func (ae *AnalyticsEngine) performScalabilityAnalysis(connections, throughput, latency []float64) ScalabilityAnalysis {
	if len(connections) != len(throughput) || len(connections) < 2 {
		return ScalabilityAnalysis{}
	}

	// Calculate linear scalability score
	linearScore := ae.calculateLinearScalabilityScore(connections, throughput)

	// Detect scalability breakpoints
	breakpoints := ae.detectScalabilityBreakpoints(connections, throughput, latency)

	// Find optimal connection range
	optimalRange := ae.findOptimalConnectionRange(connections, throughput, latency)

	// Calculate efficiency metrics
	efficiency := ae.calculateEfficiencyMetrics(connections, throughput, latency)

	// Analyze bottlenecks
	bottlenecks := ae.analyzeBottlenecks(connections, throughput, latency)

	return ScalabilityAnalysis{
		LinearScalabilityScore: linearScore,
		ScalabilityBreakpoints: breakpoints,
		OptimalConnectionRange: optimalRange,
		EfficiencyMetrics:      efficiency,
		BottleneckAnalysis:     bottlenecks,
	}
}

// generateSummary creates high-level analysis summary
func (ae *AnalyticsEngine) generateSummary(results []BandResult, throughput, latency []float64) AnalysisSummary {
	if len(results) == 0 {
		return AnalysisSummary{}
	}

	// Find optimal band (highest throughput with acceptable latency)
	optimalBand := 0
	maxThroughput := 0.0
	bestLatencyBand := 0
	minLatencyP95 := math.Inf(1)

	for i, result := range results {
		if result.Metrics != nil {
			if result.Metrics.AvgTPS > maxThroughput {
				maxThroughput = result.Metrics.AvgTPS
				optimalBand = i + 1
			}

			if result.Metrics.LatencyP95 < minLatencyP95 {
				minLatencyP95 = result.Metrics.LatencyP95
				bestLatencyBand = i + 1
			}
		}
	}

	// Calculate overall stability (inverse of coefficient of variation)
	stability := 0.0
	if len(throughput) > 0 {
		cv := ae.calculateDescriptiveStats(throughput).CoefficientOfVariation
		stability = 1.0 / (1.0 + cv)
	}

	// Calculate scalability score based on throughput trend
	scalabilityScore := ae.calculateScalabilityScore(throughput)

	return AnalysisSummary{
		TotalBands:       len(results),
		OptimalBand:      optimalBand,
		MaxThroughput:    maxThroughput,
		BestLatencyBand:  bestLatencyBand,
		MinLatencyP95:    minLatencyP95,
		OverallStability: stability,
		ScalabilityScore: scalabilityScore,
	}
}

// generateRecommendations creates actionable recommendations
func (ae *AnalyticsEngine) generateRecommendations(stats StatisticalAnalysis, trend TrendAnalysis, queueing QueueingAnalysis, scalability ScalabilityAnalysis) []string {
	recommendations := []string{}

	// Throughput recommendations
	if trend.ThroughputTrend.Direction == "decreasing" {
		recommendations = append(recommendations, "Throughput shows decreasing trend - investigate bottlenecks at higher connection counts")
	}

	// Latency recommendations
	if stats.LatencyStats.Mean > 100 { // 100ms threshold
		recommendations = append(recommendations, "Average latency exceeds 100ms - consider optimizing queries or connection pooling")
	}

	// Scalability recommendations
	if scalability.LinearScalabilityScore < 0.7 {
		recommendations = append(recommendations, "Poor linear scalability detected - system may have concurrency bottlenecks")
	}

	// Queueing recommendations
	if queueing.UtilizationFactor > 0.8 {
		recommendations = append(recommendations, "High utilization factor detected - approaching saturation point")
	}

	// Error rate recommendations
	if stats.ErrorRateStats.Mean > 0.05 {
		recommendations = append(recommendations, "Error rate exceeds 5% - investigate connection handling and timeout configurations")
	}

	// Stability recommendations
	if stats.ThroughputStats.CoefficientOfVariation > 0.3 {
		recommendations = append(recommendations, "High throughput variability - consider workload balancing or connection pool optimization")
	}

	// Default recommendation if none found
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System shows good performance characteristics across tested connection ranges")
	}

	return recommendations
}

// Helper functions for statistical calculations

func (ae *AnalyticsEngine) mean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func (ae *AnalyticsEngine) variance(data []float64, mean float64) float64 {
	if len(data) <= 1 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(data)-1)
}

func (ae *AnalyticsEngine) median(sortedData []float64) float64 {
	n := len(sortedData)
	if n == 0 {
		return 0
	}
	if n%2 == 0 {
		return (sortedData[n/2-1] + sortedData[n/2]) / 2
	}
	return sortedData[n/2]
}

func (ae *AnalyticsEngine) skewness(data []float64, mean, stdDev float64) float64 {
	if len(data) < 3 || stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		normalized := (v - mean) / stdDev
		sum += normalized * normalized * normalized
	}
	return sum / float64(len(data))
}

func (ae *AnalyticsEngine) kurtosis(data []float64, mean, stdDev float64) float64 {
	if len(data) < 4 || stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		normalized := (v - mean) / stdDev
		sum += normalized * normalized * normalized * normalized
	}
	return (sum / float64(len(data))) - 3.0 // Excess kurtosis
}

func (ae *AnalyticsEngine) analyzeTrend(data []float64) TrendData {
	if len(data) < 2 {
		return TrendData{Direction: "insufficient_data"}
	}

	// Simple linear regression for trend
	n := float64(len(data))
	sumX := n * (n - 1) / 2 // Sum of 0, 1, 2, ..., n-1
	sumY := ae.mean(data) * n
	sumXY := 0.0
	sumX2 := 0.0

	for i, y := range data {
		x := float64(i)
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	direction := "stable"
	if slope > 0.1 {
		direction = "increasing"
	} else if slope < -0.1 {
		direction = "decreasing"
	}

	// Calculate R-squared
	yMean := ae.mean(data)
	ssRes := 0.0
	ssTot := 0.0
	for i, y := range data {
		x := float64(i)
		predicted := slope*x + (sumY-slope*sumX)/n
		ssRes += (y - predicted) * (y - predicted)
		ssTot += (y - yMean) * (y - yMean)
	}

	rSquared := 0.0
	if ssTot > 0 {
		rSquared = 1.0 - (ssRes / ssTot)
	}

	return TrendData{
		Direction:     direction,
		Slope:         slope,
		RSquared:      rSquared,
		TrendStrength: math.Abs(rSquared),
	}
}

func (ae *AnalyticsEngine) calculateDerivatives(data []float64) DerivativeData {
	if len(data) < 2 {
		return DerivativeData{}
	}

	// First derivative (rate of change)
	firstDeriv := make([]float64, len(data)-1)
	for i := 0; i < len(data)-1; i++ {
		firstDeriv[i] = data[i+1] - data[i]
	}

	// Second derivative (acceleration)
	secondDeriv := make([]float64, 0)
	if len(firstDeriv) > 1 {
		secondDeriv = make([]float64, len(firstDeriv)-1)
		for i := 0; i < len(firstDeriv)-1; i++ {
			secondDeriv[i] = firstDeriv[i+1] - firstDeriv[i]
		}
	}

	// Find max acceleration and deceleration
	maxAccel := 0.0
	maxDecel := 0.0
	for _, accel := range secondDeriv {
		if accel > maxAccel {
			maxAccel = accel
		}
		if accel < maxDecel {
			maxDecel = accel
		}
	}

	return DerivativeData{
		FirstDerivative:  firstDeriv,
		SecondDerivative: secondDeriv,
		MaxAcceleration:  maxAccel,
		MaxDeceleration:  maxDecel,
	}
}

func (ae *AnalyticsEngine) calculateIntegrals(throughput, latency []float64) IntegralData {
	// Simple trapezoidal rule integration
	throughputIntegral := ae.trapezoidalIntegration(throughput)
	latencyIntegral := ae.trapezoidalIntegration(latency)

	// Area under the curve for combined efficiency
	auc := 0.0
	if len(throughput) == len(latency) {
		for i := 0; i < len(throughput); i++ {
			if latency[i] > 0 {
				efficiency := throughput[i] / latency[i]
				auc += efficiency
			}
		}
	}

	return IntegralData{
		ThroughputIntegral:   throughputIntegral,
		LatencyIntegral:      latencyIntegral,
		AreaUnderCurve:       auc,
		CumulativeEfficiency: auc / float64(len(throughput)),
	}
}

func (ae *AnalyticsEngine) trapezoidalIntegration(data []float64) float64 {
	if len(data) < 2 {
		return 0
	}

	integral := 0.0
	for i := 0; i < len(data)-1; i++ {
		integral += (data[i] + data[i+1]) / 2.0
	}
	return integral
}

func (ae *AnalyticsEngine) performRegressionAnalysis(data []float64) RegressionData {
	// Linear regression
	linearFit := ae.calculateLinearRegression(data)

	// For now, return just linear regression
	// In a full implementation, you'd add polynomial, exponential, and logarithmic fits
	return RegressionData{
		LinearRegression: linearFit,
		BestFitModel:     "linear",
		BestFitR2:        linearFit.RSquared,
	}
}

func (ae *AnalyticsEngine) calculateLinearRegression(data []float64) LinearFit {
	if len(data) < 2 {
		return LinearFit{}
	}

	n := float64(len(data))
	sumX := n * (n - 1) / 2
	sumY := ae.mean(data) * n
	sumXY := 0.0
	sumX2 := 0.0

	for i, y := range data {
		x := float64(i)
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// Calculate R-squared
	yMean := ae.mean(data)
	ssRes := 0.0
	ssTot := 0.0
	for i, y := range data {
		x := float64(i)
		predicted := slope*x + intercept
		ssRes += (y - predicted) * (y - predicted)
		ssTot += (y - yMean) * (y - yMean)
	}

	rSquared := 0.0
	if ssTot > 0 {
		rSquared = 1.0 - (ssRes / ssTot)
	}

	return LinearFit{
		Slope:     slope,
		Intercept: intercept,
		RSquared:  rSquared,
		Equation:  fmt.Sprintf("y = %.3fx + %.3f", slope, intercept),
	}
}

func (ae *AnalyticsEngine) calculateCorrelationMatrix(throughput, latency, errorRate []float64) CorrelationMatrix {
	// Simple correlation coefficient calculation
	return CorrelationMatrix{
		ThroughputVsLatency:   ae.correlation(throughput, latency),
		ErrorRateVsLatency:    ae.correlation(errorRate, latency),
		ErrorRateVsThroughput: ae.correlation(errorRate, throughput),
	}
}

func (ae *AnalyticsEngine) correlation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}

	xMean := ae.mean(x)
	yMean := ae.mean(y)

	numerator := 0.0
	xSumSq := 0.0
	ySumSq := 0.0

	for i := 0; i < len(x); i++ {
		xDiff := x[i] - xMean
		yDiff := y[i] - yMean
		numerator += xDiff * yDiff
		xSumSq += xDiff * xDiff
		ySumSq += yDiff * yDiff
	}

	denominator := math.Sqrt(xSumSq * ySumSq)
	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

func (ae *AnalyticsEngine) detectOutliers(throughput, latency, errorRate []float64) OutlierAnalysis {
	return OutlierAnalysis{
		ThroughputOutliers: ae.findOutliers(throughput),
		LatencyOutliers:    ae.findOutliers(latency),
		ErrorOutliers:      ae.findOutliers(errorRate),
	}
}

func (ae *AnalyticsEngine) findOutliers(data []float64) []OutlierData {
	if len(data) < 3 {
		return []OutlierData{}
	}

	mean := ae.mean(data)
	stdDev := math.Sqrt(ae.variance(data, mean))
	outliers := []OutlierData{}

	for i, value := range data {
		zScore := math.Abs(value-mean) / stdDev
		if zScore > 2.0 { // 2 standard deviations
			severity := "mild"
			if zScore > 3.0 {
				severity = "extreme"
			} else if zScore > 2.5 {
				severity = "moderate"
			}

			outliers = append(outliers, OutlierData{
				BandID:   i + 1,
				Value:    value,
				ZScore:   zScore,
				Severity: severity,
			})
		}
	}

	return outliers
}

// Placeholder implementations for scalability analysis functions
func (ae *AnalyticsEngine) calculateLinearScalabilityScore(connections, throughput []float64) float64 {
	if len(connections) != len(throughput) || len(connections) < 2 {
		return 0
	}
	return ae.correlation(connections, throughput)
}

func (ae *AnalyticsEngine) detectScalabilityBreakpoints(connections, throughput, latency []float64) []ScalabilityBreakpoint {
	// Simplified implementation - detect significant drops in throughput efficiency
	breakpoints := []ScalabilityBreakpoint{}

	if len(connections) < 3 {
		return breakpoints
	}

	for i := 1; i < len(throughput)-1; i++ {
		efficiency := throughput[i] / connections[i]
		prevEfficiency := throughput[i-1] / connections[i-1]

		if efficiency < prevEfficiency*0.8 { // 20% drop
			breakpoints = append(breakpoints, ScalabilityBreakpoint{
				BandID:      i + 1,
				Connections: int(connections[i]),
				Type:        "cliff",
				Impact:      (prevEfficiency - efficiency) / prevEfficiency,
				Description: fmt.Sprintf("Efficiency drop of %.1f%% at %d connections",
					((prevEfficiency-efficiency)/prevEfficiency)*100, int(connections[i])),
			})
		}
	}

	return breakpoints
}

func (ae *AnalyticsEngine) findOptimalConnectionRange(connections, throughput, latency []float64) ConnectionRange {
	if len(connections) == 0 {
		return ConnectionRange{}
	}

	// Find connection count with best throughput/latency ratio
	bestRatio := 0.0
	optimalIdx := 0

	for i := 0; i < len(throughput); i++ {
		if latency[i] > 0 {
			ratio := throughput[i] / latency[i]
			if ratio > bestRatio {
				bestRatio = ratio
				optimalIdx = i
			}
		}
	}

	optimal := int(connections[optimalIdx])

	return ConnectionRange{
		Min:        int(connections[0]),
		Max:        int(connections[len(connections)-1]),
		Optimal:    optimal,
		Confidence: 0.85, // Placeholder confidence
	}
}

func (ae *AnalyticsEngine) calculateEfficiencyMetrics(connections, throughput, latency []float64) EfficiencyMetrics {
	if len(connections) == 0 {
		return EfficiencyMetrics{}
	}

	// Calculate average throughput per connection
	totalThroughputPerConn := 0.0
	for i := 0; i < len(throughput); i++ {
		if connections[i] > 0 {
			totalThroughputPerConn += throughput[i] / connections[i]
		}
	}
	avgThroughputPerConn := totalThroughputPerConn / float64(len(connections))

	// Calculate latency efficiency (inverse of latency)
	avgLatencyEff := 0.0
	for _, lat := range latency {
		if lat > 0 {
			avgLatencyEff += 1.0 / lat
		}
	}
	avgLatencyEff /= float64(len(latency))

	return EfficiencyMetrics{
		ThroughputPerConnection: avgThroughputPerConn,
		LatencyEfficiency:       avgLatencyEff,
		ResourceUtilization:     0.75, // Placeholder
		CostEfficiency:          0.80, // Placeholder
	}
}

func (ae *AnalyticsEngine) analyzeBottlenecks(connections, throughput, latency []float64) BottleneckAnalysis {
	// Simplified bottleneck analysis
	bottleneckFactors := []string{}
	primaryBottleneck := "unknown"

	// Check for throughput plateau
	if len(throughput) > 2 {
		lastThird := len(throughput) * 2 / 3
		recentThroughput := throughput[lastThird:]
		if ae.calculateDescriptiveStats(recentThroughput).CoefficientOfVariation < 0.1 {
			bottleneckFactors = append(bottleneckFactors, "throughput_plateau")
			primaryBottleneck = "concurrency_limit"
		}
	}

	// Check for latency spike
	avgLatency := ae.mean(latency)
	maxLatency := latency[0]
	for _, lat := range latency {
		if lat > maxLatency {
			maxLatency = lat
		}
	}

	if maxLatency > avgLatency*2 {
		bottleneckFactors = append(bottleneckFactors, "latency_spike")
		if primaryBottleneck == "unknown" {
			primaryBottleneck = "resource_contention"
		}
	}

	mitigationStrategies := []string{
		"Analyze connection pool configuration",
		"Review database query optimization",
		"Monitor system resource utilization",
		"Consider horizontal scaling options",
	}

	return BottleneckAnalysis{
		PrimaryBottleneck:    primaryBottleneck,
		BottleneckFactors:    bottleneckFactors,
		ImpactAssessment:     "moderate", // Placeholder
		MitigationStrategies: mitigationStrategies,
	}
}

func (ae *AnalyticsEngine) calculateScalabilityScore(throughput []float64) float64 {
	if len(throughput) < 2 {
		return 0
	}

	// Simple scalability score based on throughput growth consistency
	growthRates := make([]float64, len(throughput)-1)
	for i := 1; i < len(throughput); i++ {
		if throughput[i-1] > 0 {
			growthRates[i-1] = throughput[i] / throughput[i-1]
		}
	}

	// Score based on consistency of growth (lower variance = better scalability)
	if len(growthRates) > 0 {
		mean := ae.mean(growthRates)
		variance := ae.variance(growthRates, mean)
		cv := math.Sqrt(variance) / mean
		return math.Max(0, 1.0-cv) // Convert to 0-1 score
	}

	return 0
}
