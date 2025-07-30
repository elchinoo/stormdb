// analysis.go - Advanced mathematical analysis for progressive scaling
package progressive

import (
	"fmt"
	"math"

	"github.com/elchinoo/stormdb/pkg/types"
)

// performAnalysis conducts comprehensive mathematical analysis of progressive scaling results
func (e *ScalingEngine) performAnalysis() error {
	if len(e.results.Bands) < 2 {
		return fmt.Errorf("need at least 2 bands for analysis, got: %d", len(e.results.Bands))
	}

	analysis := &types.ProgressiveAnalysis{}

	// Calculate marginal gains (discrete derivatives)
	if err := e.calculateMarginalGains(analysis); err != nil {
		return fmt.Errorf("failed to calculate marginal gains: %w", err)
	}

	// Detect inflection points (second derivatives)
	if err := e.detectInflectionPoints(analysis); err != nil {
		return fmt.Errorf("failed to detect inflection points: %w", err)
	}

	// Perform curve fitting
	if err := e.performCurveFitting(analysis); err != nil {
		return fmt.Errorf("failed to perform curve fitting: %w", err)
	}

	// Calculate cumulative capacity
	if err := e.calculateCumulativeCapacity(analysis); err != nil {
		return fmt.Errorf("failed to calculate cumulative capacity: %w", err)
	}

	// Perform queueing theory analysis
	if err := e.performQueueingAnalysis(analysis); err != nil {
		return fmt.Errorf("failed to perform queueing analysis: %w", err)
	}

	// Classify performance regions
	if err := e.classifyPerformanceRegions(analysis); err != nil {
		return fmt.Errorf("failed to classify performance regions: %w", err)
	}

	// Generate recommendations
	e.generateRecommendations(analysis)

	e.results.Analysis = *analysis
	return nil
}

// calculateMarginalGains computes discrete derivatives for marginal gain analysis
func (e *ScalingEngine) calculateMarginalGains(analysis *types.ProgressiveAnalysis) error {
	bands := e.results.Bands
	analysis.MarginalGains = make([]struct {
		BandID           int     `json:"band_id"`
		WorkerDelta      int     `json:"worker_delta"`
		ConnectionDelta  int     `json:"connection_delta"`
		TPSDelta         float64 `json:"tps_delta"`
		TPSPerWorker     float64 `json:"tps_per_worker"`
		TPSPerConnection float64 `json:"tps_per_connection"`
		EfficiencyDelta  float64 `json:"efficiency_delta"`
		LatencyDelta     float64 `json:"latency_delta"`
	}, len(bands)-1)

	for i := 1; i < len(bands); i++ {
		curr := bands[i]
		prev := bands[i-1]

		workerDelta := curr.Workers - prev.Workers
		connDelta := curr.Connections - prev.Connections
		tpsDelta := curr.TotalTPS - prev.TotalTPS
		latencyDelta := curr.AvgLatencyMs - prev.AvgLatencyMs
		efficiencyDelta := curr.WorkerEfficiency - prev.WorkerEfficiency

		var tpsPerWorker, tpsPerConn float64
		if workerDelta != 0 {
			tpsPerWorker = tpsDelta / float64(workerDelta)
			// Check for NaN or Inf values
			if math.IsNaN(tpsPerWorker) || math.IsInf(tpsPerWorker, 0) {
				tpsPerWorker = 0.0
			}
		}
		if connDelta != 0 {
			tpsPerConn = tpsDelta / float64(connDelta)
			// Check for NaN or Inf values
			if math.IsNaN(tpsPerConn) || math.IsInf(tpsPerConn, 0) {
				tpsPerConn = 0.0
			}
		}

		// Sanitize all float values
		if math.IsNaN(tpsDelta) || math.IsInf(tpsDelta, 0) {
			tpsDelta = 0.0
		}
		if math.IsNaN(latencyDelta) || math.IsInf(latencyDelta, 0) {
			latencyDelta = 0.0
		}
		if math.IsNaN(efficiencyDelta) || math.IsInf(efficiencyDelta, 0) {
			efficiencyDelta = 0.0
		}

		analysis.MarginalGains[i-1] = struct {
			BandID           int     `json:"band_id"`
			WorkerDelta      int     `json:"worker_delta"`
			ConnectionDelta  int     `json:"connection_delta"`
			TPSDelta         float64 `json:"tps_delta"`
			TPSPerWorker     float64 `json:"tps_per_worker"`
			TPSPerConnection float64 `json:"tps_per_connection"`
			EfficiencyDelta  float64 `json:"efficiency_delta"`
			LatencyDelta     float64 `json:"latency_delta"`
		}{
			BandID:           curr.BandID,
			WorkerDelta:      workerDelta,
			ConnectionDelta:  connDelta,
			TPSDelta:         tpsDelta,
			TPSPerWorker:     tpsPerWorker,
			TPSPerConnection: tpsPerConn,
			EfficiencyDelta:  efficiencyDelta,
			LatencyDelta:     latencyDelta,
		}
	}

	return nil
}

// detectInflectionPoints identifies points where performance characteristics change
func (e *ScalingEngine) detectInflectionPoints(analysis *types.ProgressiveAnalysis) error {
	if len(analysis.MarginalGains) < 2 {
		return nil // Need at least 2 marginal gains for second derivatives
	}

	analysis.InflectionPoints = make([]struct {
		BandID           int     `json:"band_id"`
		Type             string  `json:"type"`
		Metric           string  `json:"metric"`
		SecondDerivative float64 `json:"second_derivative"`
		Significance     string  `json:"significance"`
		Description      string  `json:"description"`
	}, 0)

	// Analyze TPS second derivatives
	for i := 1; i < len(analysis.MarginalGains); i++ {
		curr := analysis.MarginalGains[i]
		prev := analysis.MarginalGains[i-1]

		// Second derivative of TPS
		tpsSecondDeriv := curr.TPSDelta - prev.TPSDelta

		// Detect significant inflection points
		if math.Abs(tpsSecondDeriv) > 1.0 { // Configurable threshold
			inflectionType := "acceleration"
			if tpsSecondDeriv < 0 {
				inflectionType = "deceleration"
			}

			significance := "low"
			if math.Abs(tpsSecondDeriv) > 5.0 {
				significance = "medium"
			}
			if math.Abs(tpsSecondDeriv) > 10.0 {
				significance = "high"
			}

			description := fmt.Sprintf("TPS growth %s detected (Δ²TPS: %.2f)", inflectionType, tpsSecondDeriv)

			analysis.InflectionPoints = append(analysis.InflectionPoints, struct {
				BandID           int     `json:"band_id"`
				Type             string  `json:"type"`
				Metric           string  `json:"metric"`
				SecondDerivative float64 `json:"second_derivative"`
				Significance     string  `json:"significance"`
				Description      string  `json:"description"`
			}{
				BandID:           curr.BandID,
				Type:             inflectionType,
				Metric:           "tps",
				SecondDerivative: tpsSecondDeriv,
				Significance:     significance,
				Description:      description,
			})
		}

		// Analyze latency second derivatives
		latencySecondDeriv := curr.LatencyDelta - prev.LatencyDelta
		if math.Abs(latencySecondDeriv) > 5.0 { // Configurable threshold for latency (ms)
			inflectionType := "latency_spike"
			if latencySecondDeriv < 0 {
				inflectionType = "latency_improvement"
			}

			significance := "medium"
			if math.Abs(latencySecondDeriv) > 20.0 {
				significance = "high"
			}

			description := fmt.Sprintf("Latency %s detected (Δ²Latency: %.2fms)", inflectionType, latencySecondDeriv)

			analysis.InflectionPoints = append(analysis.InflectionPoints, struct {
				BandID           int     `json:"band_id"`
				Type             string  `json:"type"`
				Metric           string  `json:"metric"`
				SecondDerivative float64 `json:"second_derivative"`
				Significance     string  `json:"significance"`
				Description      string  `json:"description"`
			}{
				BandID:           curr.BandID,
				Type:             inflectionType,
				Metric:           "latency",
				SecondDerivative: latencySecondDeriv,
				Significance:     significance,
				Description:      description,
			})
		}
	}

	return nil
}

// performCurveFitting attempts to fit mathematical models to the performance data
func (e *ScalingEngine) performCurveFitting(analysis *types.ProgressiveAnalysis) error {
	bands := e.results.Bands
	if len(bands) < 3 {
		return nil // Need at least 3 points for meaningful curve fitting
	}

	// Extract data points
	workers := make([]float64, len(bands))
	tps := make([]float64, len(bands))
	for i, band := range bands {
		workers[i] = float64(band.Workers)
		tps[i] = band.TotalTPS
	}

	// Try different curve fitting models
	models := []string{"linear", "logarithmic", "exponential"}
	bestModel := "linear"
	bestRSquared := 0.0
	bestCoeffs := []float64{}
	bestFormula := ""

	for _, model := range models {
		coeffs, rSquared, formula := e.fitModel(model, workers, tps)
		if rSquared > bestRSquared {
			bestModel = model
			bestRSquared = rSquared
			bestCoeffs = coeffs
			bestFormula = formula
		}
	}

	// Calculate RMSE for the best model
	rmse := e.calculateRMSE(bestModel, bestCoeffs, workers, tps)

	// Generate predictions and residuals
	predictions := make([]struct {
		Workers      int     `json:"workers"`
		Connections  int     `json:"connections"`
		PredictedTPS float64 `json:"predicted_tps"`
		ActualTPS    float64 `json:"actual_tps"`
		Residual     float64 `json:"residual"`
	}, len(bands))

	for i, band := range bands {
		predicted := e.predictTPS(bestModel, bestCoeffs, float64(band.Workers))
		residual := band.TotalTPS - predicted

		predictions[i] = struct {
			Workers      int     `json:"workers"`
			Connections  int     `json:"connections"`
			PredictedTPS float64 `json:"predicted_tps"`
			ActualTPS    float64 `json:"actual_tps"`
			Residual     float64 `json:"residual"`
		}{
			Workers:      band.Workers,
			Connections:  band.Connections,
			PredictedTPS: predicted,
			ActualTPS:    band.TotalTPS,
			Residual:     residual,
		}
	}

	analysis.CurveFitting = struct {
		Model        string    `json:"model"`
		Coefficients []float64 `json:"coefficients"`
		RSquared     float64   `json:"r_squared"`
		RMSE         float64   `json:"rmse"`
		Predictions  []struct {
			Workers      int     `json:"workers"`
			Connections  int     `json:"connections"`
			PredictedTPS float64 `json:"predicted_tps"`
			ActualTPS    float64 `json:"actual_tps"`
			Residual     float64 `json:"residual"`
		} `json:"predictions"`
		Formula string `json:"formula"`
	}{
		Model:        bestModel,
		Coefficients: bestCoeffs,
		RSquared:     bestRSquared,
		RMSE:         rmse,
		Predictions:  predictions,
		Formula:      bestFormula,
	}

	return nil
}

// calculateCumulativeCapacity computes integral-based capacity metrics
func (e *ScalingEngine) calculateCumulativeCapacity(analysis *types.ProgressiveAnalysis) error {
	bands := e.results.Bands
	if len(bands) < 2 {
		return nil
	}

	// Calculate area under the TPS curve using trapezoidal rule
	var totalArea float64
	var peakCapacity float64
	var totalDuration float64

	for i := 1; i < len(bands); i++ {
		prev := bands[i-1]
		curr := bands[i]

		// Width (assuming uniform time intervals for simplicity)
		width := 1.0 // Each band represents one unit of progression

		// Heights
		h1 := prev.TotalTPS
		h2 := curr.TotalTPS

		// Trapezoidal area
		area := width * (h1 + h2) / 2
		totalArea += area

		// Track peak capacity
		if h2 > peakCapacity {
			peakCapacity = h2
		}

		totalDuration += width
	}

	// Calculate average capacity
	avgCapacity := totalArea / totalDuration

	// Calculate efficiency relative to theoretical peak
	var capacityEfficiency float64
	if peakCapacity > 0 {
		capacityEfficiency = avgCapacity / peakCapacity * 100
	}

	analysis.CumulativeCapacity = struct {
		TotalAreaUnderCurve float64 `json:"total_area_under_curve"`
		AverageCapacity     float64 `json:"average_capacity"`
		PeakCapacity        float64 `json:"peak_capacity"`
		CapacityEfficiency  float64 `json:"capacity_efficiency"`
	}{
		TotalAreaUnderCurve: totalArea,
		AverageCapacity:     avgCapacity,
		PeakCapacity:        peakCapacity,
		CapacityEfficiency:  capacityEfficiency,
	}

	return nil
}

// performQueueingAnalysis applies queueing theory to analyze performance
func (e *ScalingEngine) performQueueingAnalysis(analysis *types.ProgressiveAnalysis) error {
	bands := e.results.Bands

	utilization := make([]struct {
		BandID      int     `json:"band_id"`
		Rho         float64 `json:"rho"`
		ArrivalRate float64 `json:"arrival_rate"`
		ServiceRate float64 `json:"service_rate"`
		Servers     int     `json:"servers"`
	}, len(bands))

	waitTimes := make([]struct {
		BandID            int     `json:"band_id"`
		PredictedWaitMs   float64 `json:"predicted_wait_ms"`
		ObservedLatencyMs float64 `json:"observed_latency_ms"`
		Deviation         float64 `json:"deviation"`
		BottleneckType    string  `json:"bottleneck_type"`
	}, len(bands))

	for i, band := range bands {
		// M/M/c model parameters
		servers := band.Connections
		arrivalRate := band.TotalTPS                       // λ (transactions/sec)
		totalServiceRate := band.TotalTPS                  // Total service rate
		serviceRate := totalServiceRate / float64(servers) // μ per server

		// Utilization factor ρ = λ/(c*μ)
		var rho float64
		if servers > 0 && serviceRate > 0 {
			rho = arrivalRate / (float64(servers) * serviceRate)
		}

		utilization[i] = struct {
			BandID      int     `json:"band_id"`
			Rho         float64 `json:"rho"`
			ArrivalRate float64 `json:"arrival_rate"`
			ServiceRate float64 `json:"service_rate"`
			Servers     int     `json:"servers"`
		}{
			BandID:      band.BandID,
			Rho:         rho,
			ArrivalRate: arrivalRate,
			ServiceRate: serviceRate,
			Servers:     servers,
		}

		// Simplified M/M/c wait time prediction (using approximation)
		var predictedWaitMs float64
		if rho < 1.0 && servers > 0 && serviceRate > 0 {
			// Simplified Erlang-C approximation
			waitTimeCalc := (1.0 / serviceRate) * 1000 * rho / (1.0 - rho) // Convert to ms
			if math.IsNaN(waitTimeCalc) || math.IsInf(waitTimeCalc, 0) {
				predictedWaitMs = 10000.0 // System is saturated, set to 10 seconds
			} else {
				predictedWaitMs = waitTimeCalc
			}
		} else {
			predictedWaitMs = 10000.0 // System is saturated, set to 10 seconds
		}

		// Sanitize rho value
		if math.IsNaN(rho) || math.IsInf(rho, 0) {
			rho = 1.0 // Assume saturated system
		}

		// Determine bottleneck type based on utilization and latency patterns
		bottleneckType := "cpu"
		if rho > 0.8 {
			bottleneckType = "queue"
		} else if band.AvgLatencyMs > predictedWaitMs*2 {
			bottleneckType = "io"
		} else if band.ErrorRate > 1.0 {
			bottleneckType = "contention"
		}

		deviation := band.AvgLatencyMs - predictedWaitMs

		waitTimes[i] = struct {
			BandID            int     `json:"band_id"`
			PredictedWaitMs   float64 `json:"predicted_wait_ms"`
			ObservedLatencyMs float64 `json:"observed_latency_ms"`
			Deviation         float64 `json:"deviation"`
			BottleneckType    string  `json:"bottleneck_type"`
		}{
			BandID:            band.BandID,
			PredictedWaitMs:   predictedWaitMs,
			ObservedLatencyMs: band.AvgLatencyMs,
			Deviation:         deviation,
			BottleneckType:    bottleneckType,
		}
	}

	analysis.QueueingTheory = struct {
		ModelType   string `json:"model_type"`
		Utilization []struct {
			BandID      int     `json:"band_id"`
			Rho         float64 `json:"rho"`
			ArrivalRate float64 `json:"arrival_rate"`
			ServiceRate float64 `json:"service_rate"`
			Servers     int     `json:"servers"`
		} `json:"utilization"`
		PredictedWaitTimes []struct {
			BandID            int     `json:"band_id"`
			PredictedWaitMs   float64 `json:"predicted_wait_ms"`
			ObservedLatencyMs float64 `json:"observed_latency_ms"`
			Deviation         float64 `json:"deviation"`
			BottleneckType    string  `json:"bottleneck_type"`
		} `json:"predicted_wait_times"`
	}{
		ModelType:          "M/M/c",
		Utilization:        utilization,
		PredictedWaitTimes: waitTimes,
	}

	return nil
}

// classifyPerformanceRegions identifies different performance scaling regions
func (e *ScalingEngine) classifyPerformanceRegions(analysis *types.ProgressiveAnalysis) error {
	bands := e.results.Bands
	if len(bands) < 3 {
		return nil
	}

	regions := make([]struct {
		StartBand   int     `json:"start_band"`
		EndBand     int     `json:"end_band"`
		Region      string  `json:"region"`
		Confidence  float64 `json:"confidence"`
		Description string  `json:"description"`
	}, 0)

	// Analyze TPS growth patterns to classify regions
	currentRegion := ""
	regionStart := 1

	for i := 1; i < len(bands)-1; i++ {
		prev := bands[i-1]
		curr := bands[i]
		next := bands[i+1]

		// Calculate growth rates
		growth1 := (curr.TotalTPS - prev.TotalTPS) / prev.TotalTPS
		growth2 := (next.TotalTPS - curr.TotalTPS) / curr.TotalTPS

		// Classify current behavior
		var region string
		var confidence float64

		if growth1 > 0.1 && growth2 > 0.1 && math.Abs(growth1-growth2) < 0.05 {
			region = "linear_scaling"
			confidence = 0.8
		} else if growth1 > 0.05 && growth2 < growth1*0.7 {
			region = "diminishing_returns"
			confidence = 0.7
		} else if growth1 < 0.02 && growth2 < 0.02 {
			region = "saturation"
			confidence = 0.9
		} else if growth1 < 0 || growth2 < 0 {
			region = "degradation"
			confidence = 0.8
		} else {
			region = "transitional"
			confidence = 0.5
		}

		// Check if region changed
		if region != currentRegion {
			// End previous region
			if currentRegion != "" {
				description := e.generateRegionDescription(currentRegion)
				regions = append(regions, struct {
					StartBand   int     `json:"start_band"`
					EndBand     int     `json:"end_band"`
					Region      string  `json:"region"`
					Confidence  float64 `json:"confidence"`
					Description string  `json:"description"`
				}{
					StartBand:   regionStart,
					EndBand:     i,
					Region:      currentRegion,
					Confidence:  confidence,
					Description: description,
				})
			}

			// Start new region
			currentRegion = region
			regionStart = i + 1
		}
	}

	// Close final region
	if currentRegion != "" {
		description := e.generateRegionDescription(currentRegion)
		regions = append(regions, struct {
			StartBand   int     `json:"start_band"`
			EndBand     int     `json:"end_band"`
			Region      string  `json:"region"`
			Confidence  float64 `json:"confidence"`
			Description string  `json:"description"`
		}{
			StartBand:   regionStart,
			EndBand:     len(bands),
			Region:      currentRegion,
			Confidence:  0.7,
			Description: description,
		})
	}

	analysis.PerformanceRegions = regions
	return nil
}

// generateRecommendations creates actionable recommendations based on analysis
func (e *ScalingEngine) generateRecommendations(analysis *types.ProgressiveAnalysis) {
	recommendations := make([]struct {
		Type         string  `json:"type"`
		Priority     string  `json:"priority"`
		Category     string  `json:"category"`
		Suggestion   string  `json:"suggestion"`
		ExpectedGain float64 `json:"expected_gain"`
		Confidence   float64 `json:"confidence"`
	}, 0)

	// Analyze inflection points for recommendations
	for _, inflection := range analysis.InflectionPoints {
		if inflection.Type == "deceleration" && inflection.Significance == "high" {
			rec := struct {
				Type         string  `json:"type"`
				Priority     string  `json:"priority"`
				Category     string  `json:"category"`
				Suggestion   string  `json:"suggestion"`
				ExpectedGain float64 `json:"expected_gain"`
				Confidence   float64 `json:"confidence"`
			}{
				Type:         "configuration",
				Priority:     "high",
				Category:     "workers",
				Suggestion:   fmt.Sprintf("Consider optimal worker count around band %d where performance growth slows", inflection.BandID),
				ExpectedGain: 15.0,
				Confidence:   0.8,
			}
			recommendations = append(recommendations, rec)
		}
	}

	// Analyze utilization for connection recommendations
	for _, util := range analysis.QueueingTheory.Utilization {
		if util.Rho > 0.8 {
			rec := struct {
				Type         string  `json:"type"`
				Priority     string  `json:"priority"`
				Category     string  `json:"category"`
				Suggestion   string  `json:"suggestion"`
				ExpectedGain float64 `json:"expected_gain"`
				Confidence   float64 `json:"confidence"`
			}{
				Type:         "configuration",
				Priority:     "medium",
				Category:     "connections",
				Suggestion:   fmt.Sprintf("Band %d shows high utilization (%.2f). Consider increasing connection pool size", util.BandID, util.Rho),
				ExpectedGain: 10.0,
				Confidence:   0.7,
			}
			recommendations = append(recommendations, rec)
		}
	}

	// Add general recommendations based on performance regions
	for _, region := range analysis.PerformanceRegions {
		switch region.Region {
		case "linear_scaling":
			rec := struct {
				Type         string  `json:"type"`
				Priority     string  `json:"priority"`
				Category     string  `json:"category"`
				Suggestion   string  `json:"suggestion"`
				ExpectedGain float64 `json:"expected_gain"`
				Confidence   float64 `json:"confidence"`
			}{
				Type:         "configuration",
				Priority:     "low",
				Category:     "workers",
				Suggestion:   fmt.Sprintf("Bands %d-%d show good linear scaling. This configuration range is well-suited for production", region.StartBand, region.EndBand),
				ExpectedGain: 0.0,
				Confidence:   region.Confidence,
			}
			recommendations = append(recommendations, rec)
		case "degradation":
			rec := struct {
				Type         string  `json:"type"`
				Priority     string  `json:"priority"`
				Category     string  `json:"category"`
				Suggestion   string  `json:"suggestion"`
				ExpectedGain float64 `json:"expected_gain"`
				Confidence   float64 `json:"confidence"`
			}{
				Type:         "configuration",
				Priority:     "high",
				Category:     "system",
				Suggestion:   fmt.Sprintf("Bands %d-%d show performance degradation. Investigate resource contention or database tuning", region.StartBand, region.EndBand),
				ExpectedGain: 25.0,
				Confidence:   region.Confidence,
			}
			recommendations = append(recommendations, rec)
		}
	}

	analysis.Recommendations = recommendations
}

// Helper methods for curve fitting

// fitModel fits a mathematical model to the data points
func (e *ScalingEngine) fitModel(model string, x, y []float64) ([]float64, float64, string) {
	switch model {
	case "linear":
		return e.fitLinear(x, y)
	case "logarithmic":
		return e.fitLogarithmic(x, y)
	case "exponential":
		return e.fitExponential(x, y)
	default:
		return e.fitLinear(x, y)
	}
}

// fitLinear performs linear regression: y = ax + b
func (e *ScalingEngine) fitLinear(x, y []float64) ([]float64, float64, string) {
	n := float64(len(x))
	var sumX, sumY, sumXY, sumX2 float64

	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}

	// Calculate coefficients
	a := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	b := (sumY - a*sumX) / n

	// Calculate R-squared
	rSquared := e.calculateRSquared(x, y, func(xi float64) float64 { return a*xi + b })

	formula := fmt.Sprintf("TPS = %.2f * workers + %.2f", a, b)
	return []float64{a, b}, rSquared, formula
}

// fitLogarithmic performs logarithmic regression: y = a * ln(x) + b
func (e *ScalingEngine) fitLogarithmic(x, y []float64) ([]float64, float64, string) {
	// Transform x values to ln(x)
	lnX := make([]float64, len(x))
	for i, xi := range x {
		if xi <= 0 {
			lnX[i] = 0.001 // Avoid log(0)
		} else {
			lnX[i] = math.Log(xi)
		}
	}

	// Fit linear regression on transformed data
	coeffs, rSquared, _ := e.fitLinear(lnX, y)
	formula := fmt.Sprintf("TPS = %.2f * ln(workers) + %.2f", coeffs[0], coeffs[1])

	return coeffs, rSquared, formula
}

// fitExponential performs exponential regression: y = a * e^(bx)
func (e *ScalingEngine) fitExponential(x, y []float64) ([]float64, float64, string) {
	// Transform y values to ln(y)
	lnY := make([]float64, len(y))
	validPoints := 0

	for i, yi := range y {
		if yi > 0 {
			lnY[validPoints] = math.Log(yi)
			x[validPoints] = x[i]
			validPoints++
		}
	}

	if validPoints < 2 {
		// Fallback to linear if exponential fit isn't possible
		return e.fitLinear(x, y)
	}

	// Truncate slices to valid points
	x = x[:validPoints]
	lnY = lnY[:validPoints]

	// Fit linear regression on semi-log data
	coeffs, _, _ := e.fitLinear(x, lnY)

	// Transform back: if ln(y) = bx + ln(a), then y = a * e^(bx)
	a := math.Exp(coeffs[1])
	b := coeffs[0]

	// Calculate R-squared for original exponential model
	rSquared := e.calculateRSquared(x, y, func(xi float64) float64 { return a * math.Exp(b*xi) })

	formula := fmt.Sprintf("TPS = %.2f * e^(%.4f * workers)", a, b)
	return []float64{a, b}, rSquared, formula
}

// calculateRSquared computes the coefficient of determination
func (e *ScalingEngine) calculateRSquared(x, y []float64, predictFunc func(float64) float64) float64 {
	// Calculate mean of y
	var sumY float64
	for _, yi := range y {
		sumY += yi
	}
	meanY := sumY / float64(len(y))

	// Calculate total sum of squares and residual sum of squares
	var tss, rss float64
	for i, yi := range y {
		predicted := predictFunc(x[i])
		tss += (yi - meanY) * (yi - meanY)
		rss += (yi - predicted) * (yi - predicted)
	}

	if tss == 0 {
		return 0
	}

	return 1 - (rss / tss)
}

// calculateRMSE computes root mean square error for a model
func (e *ScalingEngine) calculateRMSE(model string, coeffs []float64, x, y []float64) float64 {
	var sumSquaredErrors float64

	for i, xi := range x {
		predicted := e.predictTPS(model, coeffs, xi)
		error := y[i] - predicted
		sumSquaredErrors += error * error
	}

	return math.Sqrt(sumSquaredErrors / float64(len(x)))
}

// predictTPS predicts TPS value using the fitted model
func (e *ScalingEngine) predictTPS(model string, coeffs []float64, x float64) float64 {
	switch model {
	case "linear":
		if len(coeffs) >= 2 {
			return coeffs[0]*x + coeffs[1]
		}
	case "logarithmic":
		if len(coeffs) >= 2 {
			return coeffs[0]*math.Log(x) + coeffs[1]
		}
	case "exponential":
		if len(coeffs) >= 2 {
			return coeffs[0] * math.Exp(coeffs[1]*x)
		}
	}
	return 0
}

// generateRegionDescription creates human-readable descriptions for performance regions
func (e *ScalingEngine) generateRegionDescription(region string) string {
	switch region {
	case "linear_scaling":
		return "Performance scales linearly with resource increases. Optimal efficiency region."
	case "diminishing_returns":
		return "Performance gains decrease with additional resources. Consider cost-benefit analysis."
	case "saturation":
		return "System has reached maximum capacity. Additional resources provide minimal benefit."
	case "degradation":
		return "Performance decreases with additional resources. System may be over-saturated or experiencing contention."
	case "transitional":
		return "Performance behavior is transitioning between different scaling patterns."
	default:
		return "Unclassified performance behavior."
	}
}
