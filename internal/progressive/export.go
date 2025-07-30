// export.go - Data export functionality for progressive scaling results
package progressive

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/elchinoo/stormdb/pkg/types"
)

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

// exportResults exports the progressive scaling results to files
func (e *ScalingEngine) exportResults() error {
	if e.config.Progressive.ExportPath == "" {
		return fmt.Errorf("export path not configured")
	}

	// Ensure export directory exists
	if err := os.MkdirAll(e.config.Progressive.ExportPath, 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	baseFilename := fmt.Sprintf("progressive_scaling_%s_%s", e.config.Workload, timestamp)

	switch e.config.Progressive.ExportFormat {
	case "csv":
		return e.exportCSV(filepath.Join(e.config.Progressive.ExportPath, baseFilename+".csv"))
	case "json":
		return e.exportJSON(filepath.Join(e.config.Progressive.ExportPath, baseFilename+".json"))
	case "both":
		if err := e.exportCSV(filepath.Join(e.config.Progressive.ExportPath, baseFilename+".csv")); err != nil {
			return err
		}
		return e.exportJSON(filepath.Join(e.config.Progressive.ExportPath, baseFilename+".json"))
	default:
		return e.exportJSON(filepath.Join(e.config.Progressive.ExportPath, baseFilename+".json"))
	}
}

// exportCSV exports results in CSV format
func (e *ScalingEngine) exportCSV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"band_id", "workers", "connections", "duration_sec", "total_tps", "total_qps",
		"avg_latency_ms", "p50_latency_ms", "p95_latency_ms", "p99_latency_ms",
		"error_rate", "total_errors", "stddev_latency", "variance_latency",
		"coefficient_of_var", "confidence_lower", "confidence_upper",
		"tps_per_worker", "tps_per_connection", "worker_efficiency", "connection_util",
	}

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, band := range e.results.Bands {
		row := []string{
			strconv.Itoa(band.BandID),
			strconv.Itoa(band.Workers),
			strconv.Itoa(band.Connections),
			fmt.Sprintf("%.2f", band.Duration.Seconds()),
			fmt.Sprintf("%.2f", band.TotalTPS),
			fmt.Sprintf("%.2f", band.TotalQPS),
			fmt.Sprintf("%.2f", band.AvgLatencyMs),
			fmt.Sprintf("%.2f", band.P50LatencyMs),
			fmt.Sprintf("%.2f", band.P95LatencyMs),
			fmt.Sprintf("%.2f", band.P99LatencyMs),
			fmt.Sprintf("%.2f", band.ErrorRate),
			strconv.FormatInt(band.TotalErrors, 10),
			fmt.Sprintf("%.2f", band.StdDevLatency),
			fmt.Sprintf("%.2f", band.VarianceLatency),
			fmt.Sprintf("%.4f", band.CoefficientOfVar),
			fmt.Sprintf("%.2f", band.ConfidenceInterval.Lower),
			fmt.Sprintf("%.2f", band.ConfidenceInterval.Upper),
			fmt.Sprintf("%.2f", band.TPSPerWorker),
			fmt.Sprintf("%.2f", band.TPSPerConnection),
			fmt.Sprintf("%.2f", band.WorkerEfficiency),
			fmt.Sprintf("%.2f", band.ConnectionUtil),
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	fmt.Printf("📊 Exported CSV results to: %s\n", filename)
	return nil
}

// exportJSON exports results in JSON format
func (e *ScalingEngine) exportJSON(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create JSON file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print

	if err := encoder.Encode(e.results); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	fmt.Printf("📊 Exported JSON results to: %s\n", filename)
	return nil
}
