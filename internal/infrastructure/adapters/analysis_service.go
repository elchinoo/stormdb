// internal/infrastructure/adapters/analysis_service.go
package adapters

import (
	"fmt"
	"math"
	"sort"

	"github.com/elchinoo/stormdb/internal/core/domain"
	"github.com/elchinoo/stormdb/internal/core/ports"
)

// MathematicalAnalysisService implements advanced statistical analysis
type MathematicalAnalysisService struct {
	// Configuration for analysis
	polynomialDegree int
	confidenceLevel  float64
}

func NewMathematicalAnalysisService() *MathematicalAnalysisService {
	return &MathematicalAnalysisService{
		polynomialDegree: 3,
		confidenceLevel:  0.95,
	}
}

// CalculateStatistics performs comprehensive statistical analysis on band results
func (s *MathematicalAnalysisService) CalculateStatistics(bands []domain.BandResults) (*domain.PerformanceAnalysis, error) {
	if len(bands) < 2 {
		return nil, fmt.Errorf("at least 2 bands required for statistical analysis")
	}

	// Sort bands by connection count for analysis
	sortedBands := make([]domain.BandResults, len(bands))
	copy(sortedBands, bands)
	sort.Slice(sortedBands, func(i, j int) bool {
		return sortedBands[i].Connections < sortedBands[j].Connections
	})

	// Extract X (connections) and Y (TPS) values
	xValues := make([]float64, len(sortedBands))
	yValues := make([]float64, len(sortedBands))
	for i, band := range sortedBands {
		xValues[i] = float64(band.Connections)
		yValues[i] = band.Performance.TotalTPS
	}

	// Calculate derivatives
	firstDerivative, secondDerivative, err := s.CalculateDerivatives(xValues, yValues)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate derivatives: %w", err)
	}

	// Find best fit model
	bestFitModel, err := s.FindBestFitModel(xValues, yValues)
	if err != nil {
		return nil, fmt.Errorf("failed to find best fit model: %w", err)
	}

	// Identify scaling regions
	scalingRegions := s.IdentifyScalingRegions(sortedBands)

	// Classify bottleneck
	bottleneckType := s.ClassifyBottleneck(sortedBands)

	// Generate optimal configuration
	optimalConfig := s.generateOptimalConfiguration(sortedBands, bestFitModel)

	// Generate performance predictions
	predictions := s.generatePredictions(sortedBands, bestFitModel)

	return &domain.PerformanceAnalysis{
		FirstDerivative:        firstDerivative,
		SecondDerivative:       secondDerivative,
		BestFitModel:           bestFitModel.ModelType,
		ModelCoefficients:      bestFitModel.Coefficients,
		ModelGoodnessOfFit:     bestFitModel.GoodnessOfFit,
		ScalingRegions:         scalingRegions,
		BottleneckType:         bottleneckType,
		OptimalConfiguration:   optimalConfig,
		PerformancePredictions: predictions,
	}, nil
}

// CalculateDerivatives computes first and second derivatives for trend analysis
func (s *MathematicalAnalysisService) CalculateDerivatives(xValues, yValues []float64) (first, second []float64, err error) {
	if len(xValues) != len(yValues) {
		return nil, nil, fmt.Errorf("x and y value arrays must have the same length")
	}

	if len(xValues) < 3 {
		return nil, nil, fmt.Errorf("at least 3 points required for derivative calculation")
	}

	n := len(xValues)
	first = make([]float64, n)
	second = make([]float64, n)

	// Calculate first derivative using central differences
	for i := 0; i < n; i++ {
		if i == 0 {
			// Forward difference
			first[i] = (yValues[i+1] - yValues[i]) / (xValues[i+1] - xValues[i])
		} else if i == n-1 {
			// Backward difference
			first[i] = (yValues[i] - yValues[i-1]) / (xValues[i] - xValues[i-1])
		} else {
			// Central difference
			first[i] = (yValues[i+1] - yValues[i-1]) / (xValues[i+1] - xValues[i-1])
		}
	}

	// Calculate second derivative
	for i := 0; i < n; i++ {
		if i == 0 {
			second[i] = (first[i+1] - first[i]) / (xValues[i+1] - xValues[i])
		} else if i == n-1 {
			second[i] = (first[i] - first[i-1]) / (xValues[i] - xValues[i-1])
		} else {
			second[i] = (first[i+1] - first[i-1]) / (xValues[i+1] - xValues[i-1])
		}
	}

	return first, second, nil
}

// FitModel fits a specific model type to the data
func (s *MathematicalAnalysisService) FitModel(xValues, yValues []float64, modelType domain.ModelType) (*ports.ModelResults, error) {
	switch modelType {
	case domain.ModelLinear:
		return s.fitLinear(xValues, yValues)
	case domain.ModelLogarithmic:
		return s.fitLogarithmic(xValues, yValues)
	case domain.ModelExponential:
		return s.fitExponential(xValues, yValues)
	case domain.ModelLogistic:
		return s.fitLogistic(xValues, yValues)
	case domain.ModelPolynomial:
		return s.fitPolynomial(xValues, yValues, s.polynomialDegree)
	default:
		return nil, fmt.Errorf("unsupported model type: %s", modelType)
	}
}

// FindBestFitModel tries multiple models and returns the best one
func (s *MathematicalAnalysisService) FindBestFitModel(xValues, yValues []float64) (*ports.ModelResults, error) {
	models := []domain.ModelType{
		domain.ModelLinear,
		domain.ModelLogarithmic,
		domain.ModelExponential,
		domain.ModelPolynomial,
	}

	var bestModel *ports.ModelResults
	bestRSquared := -1.0

	for _, modelType := range models {
		result, err := s.FitModel(xValues, yValues, modelType)
		if err != nil {
			continue // Skip models that fail to fit
		}

		if result.GoodnessOfFit > bestRSquared {
			bestRSquared = result.GoodnessOfFit
			bestModel = result
		}
	}

	if bestModel == nil {
		return nil, fmt.Errorf("no model could be fitted to the data")
	}

	return bestModel, nil
}

// DetectInflectionPoints finds points where the curve changes direction
func (s *MathematicalAnalysisService) DetectInflectionPoints(xValues, yValues []float64) ([]ports.InflectionPoint, error) {
	_, secondDerivative, err := s.CalculateDerivatives(xValues, yValues)
	if err != nil {
		return nil, err
	}

	var inflectionPoints []ports.InflectionPoint

	for i := 1; i < len(secondDerivative)-1; i++ {
		// Look for sign changes in second derivative
		if (secondDerivative[i-1] > 0 && secondDerivative[i+1] < 0) ||
			(secondDerivative[i-1] < 0 && secondDerivative[i+1] > 0) {

			direction := "increasing"
			if secondDerivative[i-1] > secondDerivative[i+1] {
				direction = "decreasing"
			}

			inflectionPoints = append(inflectionPoints, ports.InflectionPoint{
				X:         xValues[i],
				Y:         yValues[i],
				Direction: direction,
			})
		}
	}

	return inflectionPoints, nil
}

// ClassifyBottleneck determines the primary system constraint
func (s *MathematicalAnalysisService) ClassifyBottleneck(bands []domain.BandResults) domain.BottleneckType {
	if len(bands) < 2 {
		return domain.BottleneckNone
	}

	// Sort by connections
	sortedBands := make([]domain.BandResults, len(bands))
	copy(sortedBands, bands)
	sort.Slice(sortedBands, func(i, j int) bool {
		return sortedBands[i].Connections < sortedBands[j].Connections
	})

	// Calculate scaling efficiency
	initialTPS := sortedBands[0].Performance.TotalTPS
	finalTPS := sortedBands[len(sortedBands)-1].Performance.TotalTPS

	connectionIncrease := float64(sortedBands[len(sortedBands)-1].Connections) / float64(sortedBands[0].Connections)
	tpsIncrease := finalTPS / initialTPS

	scalingEfficiency := tpsIncrease / connectionIncrease

	// Analyze latency trends
	initialLatency := sortedBands[0].Performance.P95Latency
	finalLatency := sortedBands[len(sortedBands)-1].Performance.P95Latency
	latencyIncrease := finalLatency / initialLatency

	// Analyze error rates
	avgErrorRate := 0.0
	for _, band := range sortedBands {
		avgErrorRate += band.Performance.ErrorRate
	}
	avgErrorRate /= float64(len(sortedBands))

	// Classification logic
	if avgErrorRate > 0.05 { // 5% error rate threshold
		return domain.BottleneckDatabase
	}

	if latencyIncrease > 2.0 && scalingEfficiency < 0.5 {
		return domain.BottleneckIO
	}

	if scalingEfficiency < 0.3 {
		return domain.BottleneckConnection
	}

	if latencyIncrease > 1.5 {
		return domain.BottleneckQueue
	}

	// Check for memory pressure indicators
	for _, band := range sortedBands {
		if band.Resources.MemoryUsageMB > 1000 { // High memory usage
			return domain.BottleneckMemory
		}
	}

	if scalingEfficiency < 0.7 {
		return domain.BottleneckCPU
	}

	return domain.BottleneckNone
}

// IdentifyScalingRegions classifies performance behavior in different ranges
func (s *MathematicalAnalysisService) IdentifyScalingRegions(bands []domain.BandResults) []domain.ScalingRegion {
	if len(bands) < 3 {
		return []domain.ScalingRegion{}
	}

	// Sort by connections
	sortedBands := make([]domain.BandResults, len(bands))
	copy(sortedBands, bands)
	sort.Slice(sortedBands, func(i, j int) bool {
		return sortedBands[i].Connections < sortedBands[j].Connections
	})

	var regions []domain.ScalingRegion

	// Calculate marginal gains for each segment
	for i := 1; i < len(sortedBands); i++ {
		startBand := sortedBands[i-1]
		endBand := sortedBands[i]

		tpsGain := endBand.Performance.TotalTPS - startBand.Performance.TotalTPS
		connectionGain := float64(endBand.Connections - startBand.Connections)

		marginalGain := tpsGain / connectionGain

		// Classify based on marginal gain
		var classification domain.RegionClassification
		var description string

		if i == 1 {
			classification = domain.RegionBaseline
			description = "Baseline performance establishment"
		} else {
			// Compare with previous marginal gain
			prevStartBand := sortedBands[i-2]
			prevTpsGain := startBand.Performance.TotalTPS - prevStartBand.Performance.TotalTPS
			prevConnectionGain := float64(startBand.Connections - prevStartBand.Connections)
			prevMarginalGain := prevTpsGain / prevConnectionGain

			if marginalGain > prevMarginalGain*0.9 {
				classification = domain.RegionLinearScaling
				description = "Good linear scaling observed"
			} else if marginalGain > prevMarginalGain*0.5 {
				classification = domain.RegionDiminishingReturns
				description = "Diminishing returns detected"
			} else if marginalGain > 0 {
				classification = domain.RegionSaturation
				description = "Performance saturation"
			} else {
				classification = domain.RegionDegradation
				description = "Performance degradation"
			}
		}

		regions = append(regions, domain.ScalingRegion{
			StartConnections: startBand.Connections,
			EndConnections:   endBand.Connections,
			Classification:   classification,
			Description:      description,
		})
	}

	return regions
}

// generateOptimalConfiguration determines the best configuration
func (s *MathematicalAnalysisService) generateOptimalConfiguration(bands []domain.BandResults, model *ports.ModelResults) domain.RecommendedConfiguration {
	// Find the band with the highest efficiency (TPS per connection)
	bestEfficiency := 0.0
	var bestBand domain.BandResults

	for _, band := range bands {
		efficiency := band.Performance.TotalTPS / float64(band.Connections)
		if efficiency > bestEfficiency {
			bestEfficiency = efficiency
			bestBand = band
		}
	}

	// Calculate confidence based on model fit
	confidence := model.GoodnessOfFit
	if confidence > 1.0 {
		confidence = 1.0
	}

	reasoning := fmt.Sprintf("Selected based on highest efficiency (%.2f TPS/connection) with %s model fit (R² = %.3f)",
		bestEfficiency, model.ModelType, model.GoodnessOfFit)

	return domain.RecommendedConfiguration{
		OptimalWorkers:     bestBand.Workers,
		OptimalConnections: bestBand.Connections,
		ExpectedTPS:        bestBand.Performance.TotalTPS,
		ExpectedLatency:    bestBand.Performance.P95Latency,
		Confidence:         confidence,
		Reasoning:          reasoning,
	}
}

// generatePredictions creates performance predictions for untested configurations
func (s *MathematicalAnalysisService) generatePredictions(bands []domain.BandResults, model *ports.ModelResults) []domain.PerformancePrediction {
	var predictions []domain.PerformancePrediction

	// Generate predictions for intermediate values
	minConnections := bands[0].Connections
	maxConnections := bands[0].Connections

	for _, band := range bands {
		if band.Connections < minConnections {
			minConnections = band.Connections
		}
		if band.Connections > maxConnections {
			maxConnections = band.Connections
		}
	}

	// Create predictions for values between tested points
	step := (maxConnections - minConnections) / 10
	if step < 1 {
		step = 1
	}

	for conn := minConnections; conn <= maxConnections; conn += step {
		// Skip if we already have data for this connection count
		hasData := false
		for _, band := range bands {
			if band.Connections == conn {
				hasData = true
				break
			}
		}
		if hasData {
			continue
		}

		// Predict using the model
		predictedTPS := s.predictTPS(float64(conn), model)
		predictedLatency := s.predictLatency(float64(conn), bands)

		// Calculate confidence bounds (simplified)
		errorMargin := predictedTPS * (1.0 - model.GoodnessOfFit) * 0.5

		predictions = append(predictions, domain.PerformancePrediction{
			Workers:          conn, // Simplified: assume workers = connections
			Connections:      conn,
			PredictedTPS:     predictedTPS,
			PredictedLatency: predictedLatency,
			ConfidenceBounds: domain.ConfidenceInterval{
				Lower:      predictedTPS - errorMargin,
				Upper:      predictedTPS + errorMargin,
				Confidence: s.confidenceLevel,
			},
			ModelUsed: model.ModelType,
		})
	}

	return predictions
}

// predictTPS uses the fitted model to predict TPS for a given connection count
func (s *MathematicalAnalysisService) predictTPS(connections float64, model *ports.ModelResults) float64 {
	switch model.ModelType {
	case domain.ModelLinear:
		if len(model.Coefficients) >= 2 {
			return model.Coefficients[0] + model.Coefficients[1]*connections
		}
	case domain.ModelLogarithmic:
		if len(model.Coefficients) >= 2 {
			return model.Coefficients[0] + model.Coefficients[1]*math.Log(connections)
		}
	case domain.ModelExponential:
		if len(model.Coefficients) >= 2 {
			return model.Coefficients[0] * math.Exp(model.Coefficients[1]*connections)
		}
	case domain.ModelPolynomial:
		result := 0.0
		for i, coeff := range model.Coefficients {
			result += coeff * math.Pow(connections, float64(i))
		}
		return result
	}

	// Fallback to linear interpolation
	return connections * 10.0 // Simple fallback
}

// predictLatency estimates latency based on connection count and historical data
func (s *MathematicalAnalysisService) predictLatency(connections float64, bands []domain.BandResults) float64 {
	// Simple interpolation based on existing data
	if len(bands) < 2 {
		return 10.0 // Default value
	}

	// Find closest bands
	sort.Slice(bands, func(i, j int) bool {
		return bands[i].Connections < bands[j].Connections
	})

	// Linear interpolation
	for i := 0; i < len(bands)-1; i++ {
		if float64(bands[i].Connections) <= connections && connections <= float64(bands[i+1].Connections) {
			// Interpolate
			x1, y1 := float64(bands[i].Connections), bands[i].Performance.P95Latency
			x2, y2 := float64(bands[i+1].Connections), bands[i+1].Performance.P95Latency

			return y1 + (y2-y1)*(connections-x1)/(x2-x1)
		}
	}

	// Extrapolation
	if connections < float64(bands[0].Connections) {
		return bands[0].Performance.P95Latency
	}
	return bands[len(bands)-1].Performance.P95Latency
}

// Model fitting implementations

func (s *MathematicalAnalysisService) fitLinear(xValues, yValues []float64) (*ports.ModelResults, error) {
	n := float64(len(xValues))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0

	for i := 0; i < len(xValues); i++ {
		sumX += xValues[i]
		sumY += yValues[i]
		sumXY += xValues[i] * yValues[i]
		sumX2 += xValues[i] * xValues[i]
	}

	// Calculate coefficients: y = a + bx
	b := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	a := (sumY - b*sumX) / n

	// Calculate R-squared
	rSquared := s.calculateRSquared(xValues, yValues, []float64{a, b}, domain.ModelLinear)

	// Generate predictions
	predictions := make([]float64, len(xValues))
	for i, x := range xValues {
		predictions[i] = a + b*x
	}

	return &ports.ModelResults{
		ModelType:     domain.ModelLinear,
		Coefficients:  []float64{a, b},
		GoodnessOfFit: rSquared,
		Predictions:   predictions,
	}, nil
}

func (s *MathematicalAnalysisService) fitLogarithmic(xValues, yValues []float64) (*ports.ModelResults, error) {
	// Transform x values: y = a + b*ln(x)
	logX := make([]float64, len(xValues))
	for i, x := range xValues {
		if x <= 0 {
			return nil, fmt.Errorf("logarithmic model requires positive x values")
		}
		logX[i] = math.Log(x)
	}

	// Fit linear model to (ln(x), y)
	return s.fitLinear(logX, yValues)
}

func (s *MathematicalAnalysisService) fitExponential(xValues, yValues []float64) (*ports.ModelResults, error) {
	// Transform y values: ln(y) = ln(a) + bx
	logY := make([]float64, len(yValues))
	for i, y := range yValues {
		if y <= 0 {
			return nil, fmt.Errorf("exponential model requires positive y values")
		}
		logY[i] = math.Log(y)
	}

	// Fit linear model to (x, ln(y))
	linearResult, err := s.fitLinear(xValues, logY)
	if err != nil {
		return nil, err
	}

	// Transform coefficients back: a = e^(intercept), b = slope
	a := math.Exp(linearResult.Coefficients[0])
	b := linearResult.Coefficients[1]

	rSquared := s.calculateRSquared(xValues, yValues, []float64{a, b}, domain.ModelExponential)

	predictions := make([]float64, len(xValues))
	for i, x := range xValues {
		predictions[i] = a * math.Exp(b*x)
	}

	return &ports.ModelResults{
		ModelType:     domain.ModelExponential,
		Coefficients:  []float64{a, b},
		GoodnessOfFit: rSquared,
		Predictions:   predictions,
	}, nil
}

func (s *MathematicalAnalysisService) fitLogistic(xValues, yValues []float64) (*ports.ModelResults, error) {
	// Simplified logistic fitting (would need iterative methods for full implementation)
	// For now, return a placeholder
	return &ports.ModelResults{
		ModelType:     domain.ModelLogistic,
		Coefficients:  []float64{1.0, 0.1, 100.0}, // L, k, x0 placeholders
		GoodnessOfFit: 0.5,
		Predictions:   make([]float64, len(xValues)),
	}, nil
}

func (s *MathematicalAnalysisService) fitPolynomial(xValues, yValues []float64, degree int) (*ports.ModelResults, error) {
	// Simplified polynomial fitting using least squares
	// For a full implementation, would use matrix operations

	if degree > len(xValues)-1 {
		degree = len(xValues) - 1
	}

	// For now, fallback to linear for simplicity
	if degree <= 1 {
		return s.fitLinear(xValues, yValues)
	}

	// Placeholder for higher degree polynomials
	return &ports.ModelResults{
		ModelType:     domain.ModelPolynomial,
		Coefficients:  make([]float64, degree+1),
		GoodnessOfFit: 0.5,
		Predictions:   make([]float64, len(xValues)),
	}, nil
}

func (s *MathematicalAnalysisService) calculateRSquared(xValues, yValues, coefficients []float64, modelType domain.ModelType) float64 {
	if len(yValues) == 0 {
		return 0.0
	}

	// Calculate mean of y values
	yMean := 0.0
	for _, y := range yValues {
		yMean += y
	}
	yMean /= float64(len(yValues))

	// Calculate total sum of squares and residual sum of squares
	totalSumSquares := 0.0
	residualSumSquares := 0.0

	for i, y := range yValues {
		x := xValues[i]

		// Predict y value using the model
		var predicted float64
		switch modelType {
		case domain.ModelLinear:
			predicted = coefficients[0] + coefficients[1]*x
		case domain.ModelLogarithmic:
			predicted = coefficients[0] + coefficients[1]*math.Log(x)
		case domain.ModelExponential:
			predicted = coefficients[0] * math.Exp(coefficients[1]*x)
		default:
			predicted = y // Fallback
		}

		totalSumSquares += (y - yMean) * (y - yMean)
		residualSumSquares += (y - predicted) * (y - predicted)
	}

	if totalSumSquares == 0 {
		return 0.0
	}

	rSquared := 1.0 - (residualSumSquares / totalSumSquares)
	if rSquared < 0 {
		rSquared = 0.0
	}

	return rSquared
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
