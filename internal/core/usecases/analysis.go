// internal/core/usecases/analysis.go
package usecases

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"

	"github.com/elchinoo/stormdb/internal/core/domain"
	"github.com/elchinoo/stormdb/internal/core/ports"
)

// AnalysisUseCase provides advanced performance analysis capabilities
type AnalysisUseCase struct {
	testRepo        ports.TestExecutionRepository
	metricsRepo     ports.MetricsRepository
	analysisService ports.AnalysisService
}

func NewAnalysisUseCase(
	testRepo ports.TestExecutionRepository,
	metricsRepo ports.MetricsRepository,
	analysisService ports.AnalysisService,
) *AnalysisUseCase {
	return &AnalysisUseCase{
		testRepo:        testRepo,
		metricsRepo:     metricsRepo,
		analysisService: analysisService,
	}
}

// AnalyzeTestResults performs comprehensive analysis on test execution results
func (uc *AnalysisUseCase) AnalyzeTestResults(ctx context.Context, executionID string) (*AnalysisResult, error) {
	// 1. Load test execution and results
	testExecution, err := uc.testRepo.GetByID(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load test execution: %w", err)
	}

	if testExecution.Results == nil {
		return nil, fmt.Errorf("test execution has no results")
	}

	// 2. Extract band results for analysis
	var bands []domain.BandResults
	if testExecution.Results.ProgressiveResults != nil {
		bands = testExecution.Results.ProgressiveResults.Bands
	} else if testExecution.Results.SingleBandResults != nil {
		bands = []domain.BandResults{*testExecution.Results.SingleBandResults}
	} else {
		return nil, fmt.Errorf("no band results found")
	}

	// 3. Perform mathematical analysis
	analysis, err := uc.analysisService.CalculateStatistics(bands)
	if err != nil {
		return nil, fmt.Errorf("statistical analysis failed: %w", err)
	}

	// 4. Generate recommendations
	recommendations := uc.generateRecommendations(bands, analysis)

	// 5. Create comprehensive analysis result
	result := &AnalysisResult{
		ExecutionID:     executionID,
		TestExecution:   testExecution,
		Analysis:        analysis,
		Recommendations: recommendations,
		Summary:         uc.generateSummary(bands, analysis),
	}

	return result, nil
}

// CompareExecutions compares multiple test executions
func (uc *AnalysisUseCase) CompareExecutions(ctx context.Context, executionIDs []string) (*ComparisonResult, error) {
	if len(executionIDs) < 2 {
		return nil, fmt.Errorf("at least 2 executions required for comparison")
	}

	var executions []*domain.TestExecution
	var analyses []*domain.PerformanceAnalysis

	// Load all executions and their analyses
	for _, id := range executionIDs {
		execution, err := uc.testRepo.GetByID(ctx, id)
		if err != nil {
			log.Printf("Warning: failed to load execution %s: %v", id, err)
			continue
		}

		if execution.Results == nil || execution.Results.Analysis == nil {
			log.Printf("Warning: execution %s has no analysis results", id)
			continue
		}

		executions = append(executions, execution)
		analyses = append(analyses, execution.Results.Analysis)
	}

	if len(executions) < 2 {
		return nil, fmt.Errorf("insufficient valid executions for comparison")
	}

	// Perform comparison analysis
	comparison := uc.performComparison(executions, analyses)

	return comparison, nil
}

// FindOptimalConfiguration determines the best configuration from historical data
func (uc *AnalysisUseCase) FindOptimalConfiguration(ctx context.Context, workloadType string, criteria OptimizationCriteria) (*OptimalConfigurationResult, error) {
	// Load historical executions for this workload type
	statusCompleted := domain.StatusCompleted
	filters := ports.TestExecutionFilters{
		WorkloadType: &workloadType,
		Status:       &statusCompleted,
		Limit:        100, // Limit to recent executions
	}

	executions, err := uc.testRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to load historical executions: %w", err)
	}

	if len(executions) == 0 {
		return nil, fmt.Errorf("no historical data found for workload type %s", workloadType)
	}

	// Analyze all executions to find optimal configuration
	optimal := uc.findOptimalFromHistory(executions, criteria)

	return optimal, nil
}

// generateRecommendations creates actionable recommendations based on analysis
func (uc *AnalysisUseCase) generateRecommendations(bands []domain.BandResults, analysis *domain.PerformanceAnalysis) []Recommendation {
	var recommendations []Recommendation

	// Analyze bottleneck type
	switch analysis.BottleneckType {
	case domain.BottleneckConnection:
		recommendations = append(recommendations, Recommendation{
			Type:        "connection_optimization",
			Priority:    "high",
			Description: "Connection pooling is the limiting factor. Consider increasing connection pool size or optimizing connection reuse.",
			Action:      "Increase max connections gradually and monitor connection utilization",
		})

	case domain.BottleneckCPU:
		recommendations = append(recommendations, Recommendation{
			Type:        "cpu_optimization",
			Priority:    "high",
			Description: "CPU utilization is limiting performance. Consider reducing computational complexity or scaling horizontally.",
			Action:      "Optimize queries, add indexes, or increase CPU resources",
		})

	case domain.BottleneckIO:
		recommendations = append(recommendations, Recommendation{
			Type:        "io_optimization",
			Priority:    "high",
			Description: "I/O operations are the bottleneck. Consider optimizing disk access patterns or using faster storage.",
			Action:      "Optimize queries, consider SSD storage, or implement read replicas",
		})

	case domain.BottleneckMemory:
		recommendations = append(recommendations, Recommendation{
			Type:        "memory_optimization",
			Priority:    "high",
			Description: "Memory limitations are affecting performance. Consider increasing available memory or optimizing memory usage.",
			Action:      "Increase RAM, optimize query plans, or implement memory-efficient algorithms",
		})
	}

	// Analyze scaling regions
	for _, region := range analysis.ScalingRegions {
		switch region.Classification {
		case domain.RegionLinearScaling:
			recommendations = append(recommendations, Recommendation{
				Type:        "scaling_opportunity",
				Priority:    "medium",
				Description: fmt.Sprintf("Linear scaling detected in range %d-%d connections. This is optimal scaling behavior.", region.StartConnections, region.EndConnections),
				Action:      "Consider operating in this range for predictable performance",
			})

		case domain.RegionDiminishingReturns:
			recommendations = append(recommendations, Recommendation{
				Type:        "scaling_limit",
				Priority:    "medium",
				Description: fmt.Sprintf("Diminishing returns detected at %d+ connections. Additional connections provide minimal benefit.", region.StartConnections),
				Action:      "Avoid exceeding this connection count unless necessary",
			})

		case domain.RegionDegradation:
			recommendations = append(recommendations, Recommendation{
				Type:        "scaling_warning",
				Priority:    "high",
				Description: fmt.Sprintf("Performance degradation detected at %d+ connections. High connection counts are harmful.", region.StartConnections),
				Action:      "Stay below this connection threshold to avoid performance degradation",
			})
		}
	}

	// Analyze optimal configuration
	if analysis.OptimalConfiguration.Confidence > 0.8 {
		recommendations = append(recommendations, Recommendation{
			Type:        "optimal_config",
			Priority:    "high",
			Description: fmt.Sprintf("High-confidence optimal configuration identified: %d workers, %d connections", analysis.OptimalConfiguration.OptimalWorkers, analysis.OptimalConfiguration.OptimalConnections),
			Action:      fmt.Sprintf("Use %d workers and %d connections for best performance", analysis.OptimalConfiguration.OptimalWorkers, analysis.OptimalConfiguration.OptimalConnections),
		})
	}

	return recommendations
}

// generateSummary creates a human-readable summary of the analysis
func (uc *AnalysisUseCase) generateSummary(bands []domain.BandResults, analysis *domain.PerformanceAnalysis) Summary {
	if len(bands) == 0 {
		return Summary{
			OverallRating: "unknown",
			KeyFindings:   []string{"No data available for analysis"},
		}
	}

	// Find best performing band
	bestBand := bands[0]
	maxTPS := bestBand.Performance.TotalTPS
	for _, band := range bands {
		if band.Performance.TotalTPS > maxTPS {
			maxTPS = band.Performance.TotalTPS
			bestBand = band
		}
	}

	// Calculate overall efficiency
	efficiency := uc.calculateOverallEfficiency(bands)
	var rating string
	switch {
	case efficiency > 0.8:
		rating = "excellent"
	case efficiency > 0.6:
		rating = "good"
	case efficiency > 0.4:
		rating = "fair"
	default:
		rating = "poor"
	}

	// Generate key findings
	var findings []string
	findings = append(findings, fmt.Sprintf("Peak performance: %.2f TPS at %d workers/%d connections", maxTPS, bestBand.Workers, bestBand.Connections))
	findings = append(findings, fmt.Sprintf("Primary bottleneck: %s", analysis.BottleneckType))
	findings = append(findings, fmt.Sprintf("Scaling efficiency: %.1f%%", efficiency*100))

	if analysis.OptimalConfiguration.Confidence > 0.7 {
		findings = append(findings, fmt.Sprintf("Recommended configuration: %d workers, %d connections (%.1f%% confidence)",
			analysis.OptimalConfiguration.OptimalWorkers,
			analysis.OptimalConfiguration.OptimalConnections,
			analysis.OptimalConfiguration.Confidence*100))
	}

	return Summary{
		OverallRating: rating,
		KeyFindings:   findings,
		BestConfiguration: ConfigurationSummary{
			Workers:     bestBand.Workers,
			Connections: bestBand.Connections,
			TPS:         bestBand.Performance.TotalTPS,
			Latency:     bestBand.Performance.P95Latency,
		},
	}
}

// calculateOverallEfficiency computes scaling efficiency across all bands
func (uc *AnalysisUseCase) calculateOverallEfficiency(bands []domain.BandResults) float64 {
	if len(bands) < 2 {
		return 0.0
	}

	// Sort bands by worker count
	sortedBands := make([]domain.BandResults, len(bands))
	copy(sortedBands, bands)
	sort.Slice(sortedBands, func(i, j int) bool {
		return sortedBands[i].Workers < sortedBands[j].Workers
	})

	// Calculate efficiency as ratio of actual scaling to ideal linear scaling
	baseline := sortedBands[0].Performance.TotalTPS
	maxBand := sortedBands[len(sortedBands)-1]

	actualGain := maxBand.Performance.TotalTPS - baseline
	idealGain := baseline * float64(maxBand.Workers-sortedBands[0].Workers) / float64(sortedBands[0].Workers)

	if idealGain <= 0 {
		return 0.0
	}

	efficiency := actualGain / idealGain
	if efficiency > 1.0 {
		efficiency = 1.0 // Cap at 100%
	}

	return efficiency
}

// performComparison analyzes differences between multiple executions
func (uc *AnalysisUseCase) performComparison(executions []*domain.TestExecution, analyses []*domain.PerformanceAnalysis) *ComparisonResult {
	// Find best performer by peak TPS
	bestIdx := 0
	maxTPS := 0.0

	for i, execution := range executions {
		var peakTPS float64
		if execution.Results.ProgressiveResults != nil && execution.Results.ProgressiveResults.OptimalBand != nil {
			peakTPS = execution.Results.ProgressiveResults.OptimalBand.Performance.TotalTPS
		} else if execution.Results.SingleBandResults != nil {
			peakTPS = execution.Results.SingleBandResults.Performance.TotalTPS
		}

		if peakTPS > maxTPS {
			maxTPS = peakTPS
			bestIdx = i
		}
	}

	// Generate comparison insights
	insights := []ComparisonInsight{
		{
			Category:     "performance",
			Description:  fmt.Sprintf("Best performer: %s with %.2f TPS", executions[bestIdx].Name, maxTPS),
			Significance: "high",
		},
	}

	// Compare bottleneck types
	bottleneckCounts := make(map[domain.BottleneckType]int)
	for _, analysis := range analyses {
		bottleneckCounts[analysis.BottleneckType]++
	}

	for bottleneck, count := range bottleneckCounts {
		if count > 1 {
			insights = append(insights, ComparisonInsight{
				Category:     "bottleneck",
				Description:  fmt.Sprintf("%s bottleneck detected in %d out of %d tests", bottleneck, count, len(analyses)),
				Significance: "medium",
			})
		}
	}

	return &ComparisonResult{
		ExecutionCount: len(executions),
		BestPerformer:  executions[bestIdx],
		Insights:       insights,
		Summary:        fmt.Sprintf("Compared %d test executions. Best performance: %.2f TPS", len(executions), maxTPS),
	}
}

// findOptimalFromHistory analyzes historical data to find optimal configuration
func (uc *AnalysisUseCase) findOptimalFromHistory(executions []*domain.TestExecution, criteria OptimizationCriteria) *OptimalConfigurationResult {
	var candidates []ConfigurationCandidate

	for _, execution := range executions {
		if execution.Results == nil {
			continue
		}

		var bands []domain.BandResults
		if execution.Results.ProgressiveResults != nil {
			bands = execution.Results.ProgressiveResults.Bands
		} else if execution.Results.SingleBandResults != nil {
			bands = []domain.BandResults{*execution.Results.SingleBandResults}
		}

		for _, band := range bands {
			score := uc.calculateScore(band, criteria)
			candidates = append(candidates, ConfigurationCandidate{
				Workers:     band.Workers,
				Connections: band.Connections,
				TPS:         band.Performance.TotalTPS,
				Latency:     band.Performance.P95Latency,
				Score:       score,
				ExecutionID: execution.ID,
			})
		}
	}

	if len(candidates) == 0 {
		return &OptimalConfigurationResult{
			Found:  false,
			Reason: "No valid configuration data found in historical executions",
		}
	}

	// Sort by score and pick the best
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	best := candidates[0]

	return &OptimalConfigurationResult{
		Found: true,
		OptimalConfiguration: domain.RecommendedConfiguration{
			OptimalWorkers:     best.Workers,
			OptimalConnections: best.Connections,
			ExpectedTPS:        best.TPS,
			ExpectedLatency:    best.Latency,
			Confidence:         uc.calculateConfidence(candidates, best),
			Reasoning:          fmt.Sprintf("Based on analysis of %d historical configurations", len(candidates)),
		},
		AlternativeConfigurations: candidates[:min(5, len(candidates))], // Top 5 alternatives
	}
}

// calculateScore computes a composite score based on optimization criteria
func (uc *AnalysisUseCase) calculateScore(band domain.BandResults, criteria OptimizationCriteria) float64 {
	score := 0.0

	// Normalize TPS (higher is better)
	if criteria.MaxExpectedTPS > 0 {
		tpsScore := band.Performance.TotalTPS / criteria.MaxExpectedTPS
		score += tpsScore * criteria.TPSWeight
	}

	// Normalize latency (lower is better)
	if criteria.MaxAcceptableLatency > 0 {
		latencyScore := 1.0 - (band.Performance.P95Latency / criteria.MaxAcceptableLatency)
		if latencyScore < 0 {
			latencyScore = 0
		}
		score += latencyScore * criteria.LatencyWeight
	}

	// Resource efficiency (lower resource usage is better)
	if criteria.ResourceWeight > 0 {
		resourceScore := 1.0 / float64(band.Workers*band.Connections)
		score += resourceScore * criteria.ResourceWeight
	}

	return score
}

// calculateConfidence determines confidence level based on data consistency
func (uc *AnalysisUseCase) calculateConfidence(candidates []ConfigurationCandidate, best ConfigurationCandidate) float64 {
	if len(candidates) < 2 {
		return 0.5 // Low confidence with insufficient data
	}

	// Calculate score variance
	scores := make([]float64, len(candidates))
	for i, candidate := range candidates {
		scores[i] = candidate.Score
	}

	mean := 0.0
	for _, score := range scores {
		mean += score
	}
	mean /= float64(len(scores))

	variance := 0.0
	for _, score := range scores {
		variance += math.Pow(score-mean, 2)
	}
	variance /= float64(len(scores))

	// Higher variance = lower confidence
	confidence := 1.0 / (1.0 + variance)

	// Boost confidence if the best score is significantly higher than others
	if len(candidates) > 1 {
		gap := best.Score - candidates[1].Score
		if gap > 0.1 {
			confidence += gap * 0.5
		}
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Supporting types for analysis use case

type AnalysisResult struct {
	ExecutionID     string
	TestExecution   *domain.TestExecution
	Analysis        *domain.PerformanceAnalysis
	Recommendations []Recommendation
	Summary         Summary
}

type Recommendation struct {
	Type        string
	Priority    string
	Description string
	Action      string
}

type Summary struct {
	OverallRating     string
	KeyFindings       []string
	BestConfiguration ConfigurationSummary
}

type ConfigurationSummary struct {
	Workers     int
	Connections int
	TPS         float64
	Latency     float64
}

type ComparisonResult struct {
	ExecutionCount int
	BestPerformer  *domain.TestExecution
	Insights       []ComparisonInsight
	Summary        string
}

type ComparisonInsight struct {
	Category     string
	Description  string
	Significance string
}

type OptimalConfigurationResult struct {
	Found                     bool
	OptimalConfiguration      domain.RecommendedConfiguration
	AlternativeConfigurations []ConfigurationCandidate
	Reason                    string
}

type OptimizationCriteria struct {
	TPSWeight            float64
	LatencyWeight        float64
	ResourceWeight       float64
	MaxExpectedTPS       float64
	MaxAcceptableLatency float64
}

type ConfigurationCandidate struct {
	Workers     int
	Connections int
	TPS         float64
	Latency     float64
	Score       float64
	ExecutionID string
}
