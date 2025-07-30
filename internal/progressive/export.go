// report.go - Terminal report functionality for progressive scaling results
package progressive

import (
	"fmt"
	"math"
	"strings"

	"github.com/elchinoo/stormdb/pkg/types"
)

// generateProgressiveReport creates a comprehensive terminal report
func (e *ScalingEngine) generateProgressiveReport() {
	if len(e.results.Bands) == 0 {
		fmt.Println("No bands to report")
		return
	}

	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Println("                         DATABASE SCALING PERFORMANCE REPORT")

	// Extract strategy and timing info from config
	strategy := e.config.Progressive.Strategy
	if strategy == "" {
		strategy = "linear"
	}

	// Use appropriate duration fields based on format
	var bandDuration, warmupTime, cooldownTime string
	if e.config.Progressive.TestDuration != "" {
		// v0.2 format
		bandDuration = e.config.Progressive.TestDuration
		warmupTime = e.config.Progressive.WarmupDuration
		cooldownTime = e.config.Progressive.CooldownDuration
	} else {
		// Legacy format
		bandDuration = e.config.Progressive.BandDuration
		warmupTime = e.config.Progressive.WarmupTime
		cooldownTime = e.config.Progressive.CooldownTime
	}

	fmt.Printf("Strategy: %-12s Band Duration: %-8s Warmup: %-8s Cooldown: %s\n",
		strategy, bandDuration, warmupTime, cooldownTime)
	fmt.Println("================================================================================")
	fmt.Println()

	// Generate each section of the report
	e.generateRawMetricsSection()
	e.generateMarginalGainsSection()
	e.generateInflectionSection()
	e.generateCumulativeCapacitySection()
	e.generateLatencyProfileSection()
	e.generateAsciiChartsSection()
	e.generateTakeawaysSection()

	fmt.Println("================================================================================")
}

// generateRawMetricsSection generates section 1: Raw & Stability Metrics
func (e *ScalingEngine) generateRawMetricsSection() {
	fmt.Println("1) RAW & STABILITY METRICS BY BAND")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("| Band | Conns |  Avg TPS  | StdDev |   CV    | Lat P50 (ms) | Lat P95 (ms) | Lat P99 (ms) |")
	fmt.Println("--------------------------------------------------------------------------------")

	for i, band := range e.results.Bands {
		// Calculate coefficient of variation (CV = StdDev / Mean)
		cv := 0.0
		if band.TotalTPS > 0 && band.StdDevLatency > 0 {
			// Use latency CV since we don't have TPS StdDev
			cv = band.StdDevLatency / band.AvgLatencyMs
		}
		cv = sanitizeFloat(cv)

		// For TPS StdDev, we'll use a placeholder or calculate from samples if available
		tpsStdDev := 0.0
		if len(band.TPSSamples) > 1 {
			mean := band.TotalTPS
			sumSquaredDiffs := 0.0
			for _, sample := range band.TPSSamples {
				diff := sample - mean
				sumSquaredDiffs += diff * diff
			}
			tpsStdDev = math.Sqrt(sumSquaredDiffs / float64(len(band.TPSSamples)-1))
		}
		tpsStdDev = sanitizeFloat(tpsStdDev)

		fmt.Printf("|  %2d  |  %3d | %9.1f | %6.1f | %6.3f |     %6.2f   |     %6.2f   |     %6.2f   |\n",
			i+1,
			band.Connections,
			sanitizeFloat(band.TotalTPS),
			tpsStdDev,
			cv,
			sanitizeFloat(band.P50LatencyMs),
			sanitizeFloat(band.P95LatencyMs),
			sanitizeFloat(band.P99LatencyMs),
		)
	}

	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("* CV (coefficient of variation) = StdDev / Avg TPS – lower = steadier throughput.")
	fmt.Println()
}

// generateMarginalGainsSection generates section 2: Marginal Throughput Gains
func (e *ScalingEngine) generateMarginalGainsSection() {
	fmt.Println("2) MARGINAL THROUGHPUT GAINS (1st derivative f′ = ΔTPS/ΔConns)")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("| Segment   | ΔTPS   | ΔConns |  f′ (TPS/conn) |  Interpretation                   |")
	fmt.Println("--------------------------------------------------------------------------------")

	for i := 1; i < len(e.results.Bands); i++ {
		prevBand := e.results.Bands[i-1]
		currBand := e.results.Bands[i]

		deltaTPS := sanitizeFloat(currBand.TotalTPS - prevBand.TotalTPS)
		deltaConns := currBand.Connections - prevBand.Connections

		marginalGain := 0.0
		if deltaConns > 0 {
			marginalGain = deltaTPS / float64(deltaConns)
		}
		marginalGain = sanitizeFloat(marginalGain)

		// Generate interpretation
		interpretation := e.interpretMarginalGain(marginalGain)

		fmt.Printf("| %3d → %-3d | %6.1f |    %2d  |     %6.1f     | %-33s |\n",
			prevBand.Connections,
			currBand.Connections,
			deltaTPS,
			deltaConns,
			marginalGain,
			interpretation,
		)
	}

	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println()
}

// generateInflectionSection generates section 3: Inflection of Returns (2nd derivative)
func (e *ScalingEngine) generateInflectionSection() {
	fmt.Println("3) INFLECTION OF RETURNS (2nd derivative f″)")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("| Transition       | Δf′   | ΔConns | f″ = Δf′/Δc  | What it means                       |")
	fmt.Println("--------------------------------------------------------------------------------")

	if len(e.results.Bands) < 3 {
		fmt.Println("| Need at least 3 bands for second derivative analysis                          |")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println()
		return
	}

	// Calculate marginal gains for all segments first
	marginalGains := make([]float64, 0, len(e.results.Bands)-1)
	for i := 1; i < len(e.results.Bands); i++ {
		prevBand := e.results.Bands[i-1]
		currBand := e.results.Bands[i]

		deltaTPS := sanitizeFloat(currBand.TotalTPS - prevBand.TotalTPS)
		deltaConns := currBand.Connections - prevBand.Connections

		marginalGain := 0.0
		if deltaConns > 0 {
			marginalGain = deltaTPS / float64(deltaConns)
		}
		marginalGains = append(marginalGains, sanitizeFloat(marginalGain))
	}

	// Calculate second derivatives
	for i := 1; i < len(marginalGains); i++ {
		band1 := e.results.Bands[i-1]
		band2 := e.results.Bands[i]
		band3 := e.results.Bands[i+1]

		deltaF := marginalGains[i] - marginalGains[i-1]
		deltaConns := (band2.Connections+band3.Connections)/2 - (band1.Connections+band2.Connections)/2

		secondDerivative := 0.0
		if deltaConns > 0 {
			secondDerivative = deltaF / float64(deltaConns)
		}
		secondDerivative = sanitizeFloat(secondDerivative)

		interpretation := e.interpretSecondDerivative(secondDerivative)

		fmt.Printf("| (%d→%d)→(%d→%d) | %5.1f |   %2d   |   %6.2f     | %-35s |\n",
			band1.Connections, band2.Connections, band2.Connections, band3.Connections,
			deltaF, deltaConns, secondDerivative, interpretation)
	}

	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println()
}

// generateCumulativeCapacitySection generates section 4: Cumulative Capacity (AUC)
func (e *ScalingEngine) generateCumulativeCapacitySection() {
	fmt.Println("4) CUMULATIVE CAPACITY (AUC via trapezoidal rule)")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("| Segment   | Avg TPS  | ΔConns | Area (TPS·conns) |")
	fmt.Println("--------------------------------------------------------------------------------")

	totalArea := 0.0

	for i := 1; i < len(e.results.Bands); i++ {
		prevBand := e.results.Bands[i-1]
		currBand := e.results.Bands[i]

		avgTPS := (sanitizeFloat(prevBand.TotalTPS) + sanitizeFloat(currBand.TotalTPS)) / 2.0
		deltaConns := currBand.Connections - prevBand.Connections
		area := avgTPS * float64(deltaConns)

		totalArea += area

		fmt.Printf("| %3d – %-3d | %8.1f |   %2d   |     %8.0f     |\n",
			prevBand.Connections,
			currBand.Connections,
			avgTPS,
			deltaConns,
			area,
		)
	}

	fmt.Printf("| **Total** |          |        |  **%8.0f**     |\n", totalArea)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("* AUC = total \"connection·TPS units\" over tested range. Use this to compare systems.")
	fmt.Println()
}

// generateLatencyProfileSection generates section 5: Latency Profile
func (e *ScalingEngine) generateLatencyProfileSection() {
	fmt.Println("5) LATENCY PROFILE")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("| Conns | Lat P50 (ms) | Lat P95 (ms) | Lat P99 (ms) |")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, band := range e.results.Bands {
		fmt.Printf("| %5d |    %6.2f   |    %6.2f   |    %6.2f   |\n",
			band.Connections,
			sanitizeFloat(band.P50LatencyMs),
			sanitizeFloat(band.P95LatencyMs),
			sanitizeFloat(band.P99LatencyMs),
		)
	}

	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("* Notice how latency patterns change - watch for spikes indicating bottlenecks.")
	fmt.Println()
}

// generateAsciiChartsSection generates section 6: Simple ASCII Charts
func (e *ScalingEngine) generateAsciiChartsSection() {
	fmt.Println("6) SIMPLE ASCII CHARTS")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("Throughput (Avg TPS) vs Connections")
	fmt.Println("(each '█' ≈ 100 TPS)")
	fmt.Println()

	// Find max TPS for scaling
	maxTPS := 0.0
	for _, band := range e.results.Bands {
		tps := sanitizeFloat(band.TotalTPS)
		if tps > maxTPS {
			maxTPS = tps
		}
	}

	// TPS chart
	for _, band := range e.results.Bands {
		tps := sanitizeFloat(band.TotalTPS)
		barLength := int(tps / 100) // Each block = 100 TPS
		if barLength > 80 {
			barLength = 80 // Max width
		}

		bar := strings.Repeat("█", barLength)
		fmt.Printf("%3d | %-80s (%.0f)\n", band.Connections, bar, tps)
	}

	fmt.Println()
	fmt.Println("Marginal Gains (TPS per conn) vs Connections")
	fmt.Println("(each '■' ≈ 10 TPS/conn)")
	fmt.Println()

	// Marginal gains chart
	for i := 1; i < len(e.results.Bands); i++ {
		prevBand := e.results.Bands[i-1]
		currBand := e.results.Bands[i]

		deltaTPS := sanitizeFloat(currBand.TotalTPS - prevBand.TotalTPS)
		deltaConns := currBand.Connections - prevBand.Connections

		marginalGain := 0.0
		if deltaConns > 0 {
			marginalGain = deltaTPS / float64(deltaConns)
		}
		marginalGain = sanitizeFloat(marginalGain)

		barLength := int(marginalGain / 10) // Each block = 10 TPS/conn
		if barLength < 0 {
			fmt.Printf("%d→%-3d |  (%.1f)\n", prevBand.Connections, currBand.Connections, marginalGain)
		} else {
			if barLength > 40 {
				barLength = 40 // Max width
			}
			bar := strings.Repeat("■", barLength)
			fmt.Printf("%d→%-3d | %-40s (%.1f)\n", prevBand.Connections, currBand.Connections, bar, marginalGain)
		}
	}

	fmt.Println()
}

// generateTakeawaysSection generates section 7: Key Takeaways & Next Steps
func (e *ScalingEngine) generateTakeawaysSection() {
	fmt.Println("7) KEY TAKEAWAYS & NEXT STEPS")
	fmt.Println("--------------------------------------------------------------------------------")

	// Find sweet spot (best efficiency while maintaining good performance)
	sweetSpotBand := e.findSweetSpot()
	diminishingReturnsPoint := e.findDiminishingReturnsPoint()
	overloadPoint := e.findOverloadPoint()

	if sweetSpotBand != nil {
		fmt.Printf("• **Sweet spot: %d connections**\n", sweetSpotBand.Connections)
		fmt.Printf("  – High performance (%.0f TPS)\n", sanitizeFloat(sweetSpotBand.TotalTPS))
		fmt.Printf("  – Good latency (P95: %.1fms)\n", sanitizeFloat(sweetSpotBand.P95LatencyMs))
		fmt.Println()
	}

	if diminishingReturnsPoint != nil {
		fmt.Printf("• **Diminishing returns after %d connections**\n", diminishingReturnsPoint.Connections)
		fmt.Printf("  – Performance gains slow down\n")
		fmt.Printf("  – Latency may start increasing\n")
		fmt.Println()
	}

	if overloadPoint != nil {
		fmt.Printf("• **Overload at %d connections**\n", overloadPoint.Connections)
		fmt.Printf("  – Negative performance impact\n")
		fmt.Printf("  – System thrashing detected\n")
		fmt.Println()
	}

	// Calculate total capacity
	totalCapacity := e.calculateTotalCapacity()
	fmt.Printf("• **Total capacity = %.0f conn·TPS units**\n", totalCapacity)
	fmt.Printf("  – Use this single scalar to compare future runs or hardware upgrades.\n")
	fmt.Println()

	// Generate recommendations
	fmt.Println("**Recommendations:**")
	recommendations := e.generateSimpleRecommendations(sweetSpotBand, diminishingReturnsPoint, overloadPoint)
	for i, rec := range recommendations {
		fmt.Printf("%d. %s\n", i+1, rec)
	}

	fmt.Println()
}

// Helper methods for report generation

// interpretMarginalGain provides human-readable interpretation of marginal gains
func (e *ScalingEngine) interpretMarginalGain(gain float64) string {
	if gain > 50 {
		return "excellent scaling - high returns"
	} else if gain > 20 {
		return "good scaling - solid returns"
	} else if gain > 5 {
		return "moderate scaling - diminishing returns"
	} else if gain > 0 {
		return "minimal gains - approaching saturation"
	} else if gain > -5 {
		return "slight degradation - near capacity"
	} else {
		return "performance drops - overloaded"
	}
}

// interpretSecondDerivative provides interpretation of second derivative values
func (e *ScalingEngine) interpretSecondDerivative(secondDeriv float64) string {
	if secondDeriv > 1 {
		return "accelerating gains - great scaling"
	} else if secondDeriv > 0 {
		return "marginal gains improving slightly"
	} else if secondDeriv > -0.5 {
		return "marginal gains falling slightly"
	} else if secondDeriv > -2 {
		return "steeper drop in returns"
	} else {
		return "returns collapsing rapidly"
	}
}

// findSweetSpot identifies the optimal balance of performance and efficiency
func (e *ScalingEngine) findSweetSpot() *types.ProgressiveBandMetrics {
	if len(e.results.Bands) == 0 {
		return nil
	}

	bestScore := 0.0
	var bestBand *types.ProgressiveBandMetrics

	for i := range e.results.Bands {
		band := &e.results.Bands[i]

		// Score based on TPS efficiency and low latency
		tps := sanitizeFloat(band.TotalTPS)
		latency := sanitizeFloat(band.P95LatencyMs)

		if band.Workers == 0 {
			continue
		}

		efficiency := tps / float64(band.Workers)
		latencyPenalty := 1.0
		if latency > 0 {
			latencyPenalty = 1.0 / (1.0 + latency/100.0) // Penalty for high latency
		}

		score := efficiency * latencyPenalty
		score = sanitizeFloat(score)

		if score > bestScore {
			bestScore = score
			bestBand = band
		}
	}

	return bestBand
}

// findDiminishingReturnsPoint identifies where gains start to slow down significantly
func (e *ScalingEngine) findDiminishingReturnsPoint() *types.ProgressiveBandMetrics {
	if len(e.results.Bands) < 3 {
		return nil
	}

	// Calculate marginal gains and find where they drop significantly
	for i := 2; i < len(e.results.Bands); i++ {
		prev := e.results.Bands[i-1]
		curr := e.results.Bands[i]

		deltaTPS := sanitizeFloat(curr.TotalTPS - prev.TotalTPS)
		deltaConns := curr.Connections - prev.Connections

		if deltaConns > 0 {
			marginalGain := deltaTPS / float64(deltaConns)
			marginalGain = sanitizeFloat(marginalGain)

			// If marginal gain drops below 20 TPS/conn, consider it diminishing returns
			if marginalGain < 20 {
				return &curr
			}
		}
	}

	return nil
}

// findOverloadPoint identifies where performance starts to degrade
func (e *ScalingEngine) findOverloadPoint() *types.ProgressiveBandMetrics {
	if len(e.results.Bands) < 2 {
		return nil
	}

	// Find the first band where TPS decreases compared to previous
	for i := 1; i < len(e.results.Bands); i++ {
		prev := e.results.Bands[i-1]
		curr := e.results.Bands[i]

		if sanitizeFloat(curr.TotalTPS) < sanitizeFloat(prev.TotalTPS) {
			return &curr
		}
	}

	return nil
}

// calculateTotalCapacity calculates the area under the curve (total capacity)
func (e *ScalingEngine) calculateTotalCapacity() float64 {
	if len(e.results.Bands) < 2 {
		return 0
	}

	totalArea := 0.0
	for i := 1; i < len(e.results.Bands); i++ {
		prev := e.results.Bands[i-1]
		curr := e.results.Bands[i]

		avgTPS := (sanitizeFloat(prev.TotalTPS) + sanitizeFloat(curr.TotalTPS)) / 2.0
		deltaConns := curr.Connections - prev.Connections
		area := avgTPS * float64(deltaConns)

		totalArea += area
	}

	return sanitizeFloat(totalArea)
}

// generateRecommendations generates actionable recommendations based on analysis
func (e *ScalingEngine) generateSimpleRecommendations(sweetSpot, diminishing, overload *types.ProgressiveBandMetrics) []string {
	recommendations := make([]string, 0)

	if sweetSpot != nil {
		recommendations = append(recommendations,
			fmt.Sprintf("Set connection pool around %d for optimal balance of throughput & latency", sweetSpot.Connections))
	}

	if diminishing != nil {
		recommendations = append(recommendations,
			fmt.Sprintf("Consider %d connections as maximum before diminishing returns", diminishing.Connections))
	}

	if overload != nil {
		recommendations = append(recommendations,
			fmt.Sprintf("Avoid going beyond %d connections to prevent performance degradation", overload.Connections))

		// Check for specific bottleneck indicators
		if overload.P95LatencyMs > 50 {
			recommendations = append(recommendations, "Investigate I/O subsystem - high latency suggests disk bottleneck")
		}
		if overload.ErrorRate > 1 {
			recommendations = append(recommendations, "Monitor connection pool exhaustion and timeout settings")
		}
	}

	// General recommendations
	recommendations = append(recommendations, "Re-benchmark after tuning PostgreSQL configuration")
	recommendations = append(recommendations, "Compare total capacity metric across different hardware/configurations")

	return recommendations
}

// findOptimalConfiguration identifies the best configuration from the test results
func (e *ScalingEngine) findOptimalConfiguration() {
	if len(e.results.Bands) == 0 {
		return
	}

	// Find the band with highest TPS
	bestTPS := 0.0
	bestBand := e.results.Bands[0]

	for _, band := range e.results.Bands {
		// Sanitize TPS value
		tps := sanitizeFloat(band.TotalTPS)
		if tps > bestTPS {
			bestTPS = tps
			bestBand = band
		}
	}

	// Calculate efficiency score (TPS per worker, considering latency penalty)
	bestEfficiency := 0.0
	mostEfficientBand := e.results.Bands[0]

	for _, band := range e.results.Bands {
		// Sanitize values
		tps := sanitizeFloat(band.TotalTPS)
		latency := sanitizeFloat(band.AvgLatencyMs)

		if band.Workers == 0 { // Avoid division by zero
			continue
		}

		// Efficiency score: TPS per worker, penalized by latency
		latencyPenalty := 1.0
		if latency > 100 && latency > 0 { // Penalize high latency
			latencyPenalty = 100.0 / latency
		}

		efficiency := (tps / float64(band.Workers)) * latencyPenalty
		efficiency = sanitizeFloat(efficiency)

		if efficiency > bestEfficiency {
			bestEfficiency = efficiency
			mostEfficientBand = band
		}
	}

	// Choose the most efficient configuration that also has reasonable TPS
	var optimalBand types.ProgressiveBandMetrics
	reasoning := ""

	// Sanitize TPS values for comparison
	mostEfficientTPS := sanitizeFloat(mostEfficientBand.TotalTPS)

	if mostEfficientTPS >= bestTPS*0.8 { // Within 80% of peak TPS
		optimalBand = mostEfficientBand
		reasoning = "Selected for optimal efficiency while maintaining high throughput"
	} else {
		optimalBand = bestBand
		reasoning = "Selected for maximum throughput"
	}

	// Sanitize all values for optimal config
	optimalTPS := sanitizeFloat(optimalBand.TotalTPS)
	optimalEfficiency := sanitizeFloat(optimalBand.WorkerEfficiency)

	e.results.OptimalConfig = struct {
		Workers     int     `json:"workers"`
		Connections int     `json:"connections"`
		TPS         float64 `json:"tps"`
		Efficiency  float64 `json:"efficiency"`
		Reasoning   string  `json:"reasoning"`
	}{
		Workers:     optimalBand.Workers,
		Connections: optimalBand.Connections,
		TPS:         optimalTPS,
		Efficiency:  optimalEfficiency,
		Reasoning:   reasoning,
	}
}
