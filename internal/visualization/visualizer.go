// Package visualization provides comprehensive visualization for progressive scaling results
package visualization

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/elchinoo/stormdb/pkg/metrics"
	"go.uber.org/zap"
)

// Visualizer creates various visualizations for progressive scaling results
type Visualizer struct {
	logger *zap.Logger
}

// NewVisualizer creates a new visualizer instance
func NewVisualizer(logger *zap.Logger) *Visualizer {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Visualizer{
		logger: logger,
	}
}

// VisualizationData contains all data needed for visualization
type VisualizationData struct {
	Metadata     TestMetadata                `json:"metadata"`
	BandResults  []metrics.BandResults       `json:"band_results"`
	Statistics   []metrics.StatisticalResult `json:"statistics"`
	Elasticity   []metrics.ElasticityResult  `json:"elasticity"`
	QueueMetrics []metrics.QueueMetrics      `json:"queue_metrics"`
	CostBenefit  []metrics.CostBenefitResult `json:"cost_benefit"`
	Summary      PerformanceSummary          `json:"summary"`
	Charts       map[string]ChartData        `json:"charts"`
	Tables       map[string]TableData        `json:"tables"`
	Insights     []PerformanceInsight        `json:"insights"`
}

// TestMetadata contains information about the test
type TestMetadata struct {
	TestName     string    `json:"test_name"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Duration     string    `json:"duration"`
	DatabaseInfo string    `json:"database_info"`
	WorkloadType string    `json:"workload_type"`
	TotalBands   int       `json:"total_bands"`
	Strategy     string    `json:"strategy"`
	GeneratedAt  time.Time `json:"generated_at"`
}

// ChartData represents data for various chart types
type ChartData struct {
	Type   string        `json:"type"`
	Title  string        `json:"title"`
	XLabel string        `json:"x_label"`
	YLabel string        `json:"y_label"`
	Series []ChartSeries `json:"series"`
}

// ChartSeries represents a data series in a chart
type ChartSeries struct {
	Name  string       `json:"name"`
	Data  []ChartPoint `json:"data"`
	Color string       `json:"color"`
	Style string       `json:"style"`
}

// ChartPoint represents a single point in a chart
type ChartPoint struct {
	X     interface{} `json:"x"`
	Y     float64     `json:"y"`
	Label string      `json:"label,omitempty"`
}

// TableData represents tabular data
type TableData struct {
	Title   string     `json:"title"`
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
	Summary string     `json:"summary,omitempty"`
}

// PerformanceSummary provides high-level performance insights
type PerformanceSummary struct {
	OptimalConnections  int      `json:"optimal_connections"`
	MaxThroughput       float64  `json:"max_throughput"`
	MinLatency          float64  `json:"min_latency"`
	ScalingEfficiency   float64  `json:"scaling_efficiency"`
	RecommendedWorkload string   `json:"recommended_workload"`
	PerformanceGrade    string   `json:"performance_grade"`
	KeyFindings         []string `json:"key_findings"`
}

// PerformanceInsight represents actionable insights
type PerformanceInsight struct {
	Type        string `json:"type"`     // "optimization", "warning", "recommendation"
	Priority    string `json:"priority"` // "high", "medium", "low"
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Evidence    string `json:"evidence"`
}

// GenerateReport creates a comprehensive visualization report
func (v *Visualizer) GenerateReport(
	bands []metrics.BandResults,
	statistics []metrics.StatisticalResult,
	elasticity []metrics.ElasticityResult,
	queueMetrics []metrics.QueueMetrics,
	costBenefit []metrics.CostBenefitResult,
	metadata TestMetadata,
) (*VisualizationData, error) {

	data := &VisualizationData{
		Metadata:     metadata,
		BandResults:  bands,
		Statistics:   statistics,
		Elasticity:   elasticity,
		QueueMetrics: queueMetrics,
		CostBenefit:  costBenefit,
		Charts:       make(map[string]ChartData),
		Tables:       make(map[string]TableData),
	}

	// Generate summary
	data.Summary = v.generateSummary(bands, costBenefit)

	// Generate charts
	data.Charts["throughput"] = v.createThroughputChart(bands)
	data.Charts["latency"] = v.createLatencyChart(bands)
	data.Charts["elasticity"] = v.createElasticityChart(elasticity)
	data.Charts["queue_analysis"] = v.createQueueAnalysisChart(queueMetrics)
	data.Charts["cost_benefit"] = v.createCostBenefitChart(costBenefit)
	data.Charts["scaling_efficiency"] = v.createScalingEfficiencyChart(bands)

	// Generate tables
	data.Tables["performance_summary"] = v.createPerformanceTable(bands)
	data.Tables["statistical_significance"] = v.createStatisticalTable(statistics)
	data.Tables["elasticity_analysis"] = v.createElasticityTable(elasticity)
	data.Tables["queue_theory"] = v.createQueueTable(queueMetrics)
	data.Tables["cost_benefit_analysis"] = v.createCostBenefitTable(costBenefit)

	// Generate insights
	data.Insights = v.generateInsights(bands, elasticity, queueMetrics, costBenefit)

	v.logger.Info("Visualization report generated successfully",
		zap.Int("bands", len(bands)),
		zap.Int("charts", len(data.Charts)),
		zap.Int("tables", len(data.Tables)),
		zap.Int("insights", len(data.Insights)))

	return data, nil
}

// ExportHTML exports the visualization as an interactive HTML report
func (v *Visualizer) ExportHTML(data *VisualizationData, outputPath string) error {
	htmlTemplate := v.getHTMLTemplate()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Execute template
	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	v.logger.Info("HTML report exported", zap.String("path", outputPath))
	return nil
}

// ExportJSON exports the visualization data as JSON
func (v *Visualizer) ExportJSON(data *VisualizationData, outputPath string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	v.logger.Info("JSON report exported", zap.String("path", outputPath))
	return nil
}

// ExportCSV exports tables as CSV files
func (v *Visualizer) ExportCSV(data *VisualizationData, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for name, table := range data.Tables {
		csvPath := filepath.Join(outputDir, name+".csv")
		if err := v.writeCSV(table, csvPath); err != nil {
			return fmt.Errorf("failed to write CSV %s: %w", name, err)
		}
	}

	v.logger.Info("CSV files exported",
		zap.String("directory", outputDir),
		zap.Int("files", len(data.Tables)))

	return nil
}

// Private methods for chart generation

func (v *Visualizer) createThroughputChart(bands []metrics.BandResults) ChartData {
	data := make([]ChartPoint, len(bands))
	for i, band := range bands {
		data[i] = ChartPoint{
			X:     band.Connections,
			Y:     band.AvgTPS,
			Label: fmt.Sprintf("%.1f TPS", band.AvgTPS),
		}
	}

	return ChartData{
		Type:   "line",
		Title:  "Throughput vs Connections",
		XLabel: "Number of Connections",
		YLabel: "Transactions Per Second (TPS)",
		Series: []ChartSeries{
			{
				Name:  "Average TPS",
				Data:  data,
				Color: "#2E86AB",
				Style: "solid",
			},
		},
	}
}

func (v *Visualizer) createLatencyChart(bands []metrics.BandResults) ChartData {
	p50Data := make([]ChartPoint, len(bands))
	p95Data := make([]ChartPoint, len(bands))
	p99Data := make([]ChartPoint, len(bands))

	for i, band := range bands {
		p50Data[i] = ChartPoint{X: band.Connections, Y: band.LatencyP50}
		p95Data[i] = ChartPoint{X: band.Connections, Y: band.LatencyP95}
		p99Data[i] = ChartPoint{X: band.Connections, Y: band.LatencyP99}
	}

	return ChartData{
		Type:   "line",
		Title:  "Latency Percentiles vs Connections",
		XLabel: "Number of Connections",
		YLabel: "Latency (ms)",
		Series: []ChartSeries{
			{Name: "P50", Data: p50Data, Color: "#A23B72", Style: "solid"},
			{Name: "P95", Data: p95Data, Color: "#F18F01", Style: "dashed"},
			{Name: "P99", Data: p99Data, Color: "#C73E1D", Style: "dotted"},
		},
	}
}

func (v *Visualizer) createElasticityChart(elasticity []metrics.ElasticityResult) ChartData {
	data := make([]ChartPoint, len(elasticity))
	for i, e := range elasticity {
		data[i] = ChartPoint{
			X:     e.Segment,
			Y:     e.Elasticity,
			Label: fmt.Sprintf("%.2f", e.Elasticity),
		}
	}

	return ChartData{
		Type:   "bar",
		Title:  "Elasticity Coefficient Analysis",
		XLabel: "Connection Segments",
		YLabel: "Elasticity Coefficient",
		Series: []ChartSeries{
			{
				Name:  "Elasticity",
				Data:  data,
				Color: "#3F7CAC",
				Style: "solid",
			},
		},
	}
}

func (v *Visualizer) createQueueAnalysisChart(queueMetrics []metrics.QueueMetrics) ChartData {
	utilizationData := make([]ChartPoint, len(queueMetrics))
	queueLengthData := make([]ChartPoint, len(queueMetrics))

	for i, q := range queueMetrics {
		utilizationData[i] = ChartPoint{
			X: q.Connections,
			Y: q.Utilization * 100, // Convert to percentage
		}
		queueLengthData[i] = ChartPoint{
			X: q.Connections,
			Y: q.QueueLength,
		}
	}

	return ChartData{
		Type:   "line",
		Title:  "Queueing Theory Analysis",
		XLabel: "Number of Connections",
		YLabel: "Utilization (%) / Queue Length",
		Series: []ChartSeries{
			{Name: "Utilization %", Data: utilizationData, Color: "#95B8D1", Style: "solid"},
			{Name: "Queue Length", Data: queueLengthData, Color: "#E09F3E", Style: "dashed"},
		},
	}
}

func (v *Visualizer) createCostBenefitChart(costBenefit []metrics.CostBenefitResult) ChartData {
	data := make([]ChartPoint, len(costBenefit))
	for i, cb := range costBenefit {
		data[i] = ChartPoint{
			X:     cb.Connections,
			Y:     cb.BenefitCostRatio,
			Label: fmt.Sprintf("%.2f", cb.BenefitCostRatio),
		}
	}

	return ChartData{
		Type:   "line",
		Title:  "Cost-Benefit Analysis",
		XLabel: "Number of Connections",
		YLabel: "Benefit/Cost Ratio",
		Series: []ChartSeries{
			{
				Name:  "Benefit/Cost Ratio",
				Data:  data,
				Color: "#540D6E",
				Style: "solid",
			},
		},
	}
}

func (v *Visualizer) createScalingEfficiencyChart(bands []metrics.BandResults) ChartData {
	if len(bands) < 2 {
		return ChartData{}
	}

	data := make([]ChartPoint, len(bands)-1)
	baseline := bands[0].AvgTPS

	for i := 1; i < len(bands); i++ {
		efficiency := (bands[i].AvgTPS / baseline) / float64(bands[i].Connections/bands[0].Connections)
		data[i-1] = ChartPoint{
			X:     bands[i].Connections,
			Y:     efficiency * 100, // Convert to percentage
			Label: fmt.Sprintf("%.1f%%", efficiency*100),
		}
	}

	return ChartData{
		Type:   "bar",
		Title:  "Scaling Efficiency",
		XLabel: "Number of Connections",
		YLabel: "Efficiency (%)",
		Series: []ChartSeries{
			{
				Name:  "Scaling Efficiency",
				Data:  data,
				Color: "#277DA1",
				Style: "solid",
			},
		},
	}
}

// Private methods for table generation

func (v *Visualizer) createPerformanceTable(bands []metrics.BandResults) TableData {
	headers := []string{"Connections", "Avg TPS", "P50 (ms)", "P95 (ms)", "P99 (ms)", "Std Dev", "Samples"}
	rows := make([][]string, len(bands))

	for i, band := range bands {
		rows[i] = []string{
			fmt.Sprintf("%d", band.Connections),
			fmt.Sprintf("%.1f", band.AvgTPS),
			fmt.Sprintf("%.2f", band.LatencyP50),
			fmt.Sprintf("%.2f", band.LatencyP95),
			fmt.Sprintf("%.2f", band.LatencyP99),
			fmt.Sprintf("%.2f", band.StdDev),
			fmt.Sprintf("%d", band.Samples),
		}
	}

	return TableData{
		Title:   "Performance Summary",
		Headers: headers,
		Rows:    rows,
		Summary: fmt.Sprintf("Performance data across %d connection levels", len(bands)),
	}
}

func (v *Visualizer) createStatisticalTable(statistics []metrics.StatisticalResult) TableData {
	headers := []string{"Segment", "T-Statistic", "P-Value", "Significant", "Confidence Interval"}
	rows := make([][]string, len(statistics))

	for i, stat := range statistics {
		significance := "No"
		if stat.IsSignificant {
			significance = "Yes"
		}

		ci := fmt.Sprintf("[%.2f, %.2f]",
			stat.ConfidenceInterval.Lower,
			stat.ConfidenceInterval.Upper)

		rows[i] = []string{
			fmt.Sprintf("Segment %d", i+1),
			fmt.Sprintf("%.3f", stat.TStatistic),
			fmt.Sprintf("%.6f", stat.PValue),
			significance,
			ci,
		}
	}

	return TableData{
		Title:   "Statistical Significance Testing",
		Headers: headers,
		Rows:    rows,
		Summary: "Welch's t-test results for performance differences between connection levels",
	}
}

func (v *Visualizer) createElasticityTable(elasticity []metrics.ElasticityResult) TableData {
	headers := []string{"Segment", "ΔTPS", "Baseline TPS", "ΔConns", "Baseline Conns", "Elasticity", "Interpretation"}
	rows := make([][]string, len(elasticity))

	for i, e := range elasticity {
		rows[i] = []string{
			e.Segment,
			fmt.Sprintf("%.1f", e.DeltaTPS),
			fmt.Sprintf("%.1f", e.BaselineTPS),
			fmt.Sprintf("%d", e.DeltaConns),
			fmt.Sprintf("%d", e.BaselineConns),
			fmt.Sprintf("%.2f", e.Elasticity),
			e.Interpretation,
		}
	}

	return TableData{
		Title:   "Elasticity Coefficient Analysis",
		Headers: headers,
		Rows:    rows,
		Summary: "Performance elasticity analysis showing scaling efficiency",
	}
}

func (v *Visualizer) createQueueTable(queueMetrics []metrics.QueueMetrics) TableData {
	headers := []string{"Connections", "TPS", "P99 Lat (ms)", "Utilization", "Total Requests", "Queue Length"}
	rows := make([][]string, len(queueMetrics))

	for i, q := range queueMetrics {
		rows[i] = []string{
			fmt.Sprintf("%d", q.Connections),
			fmt.Sprintf("%.1f", q.TPS),
			fmt.Sprintf("%.2f", q.LatencyP99),
			fmt.Sprintf("%.1f%%", q.Utilization*100),
			fmt.Sprintf("%.2f", q.TotalRequests),
			fmt.Sprintf("%.2f", q.QueueLength),
		}
	}

	return TableData{
		Title:   "Queueing Theory Analysis",
		Headers: headers,
		Rows:    rows,
		Summary: "Queueing theory metrics showing system behavior under load",
	}
}

func (v *Visualizer) createCostBenefitTable(costBenefit []metrics.CostBenefitResult) TableData {
	headers := []string{"Connections", "TPS", "P99 Latency", "Throughput Gain", "Latency Cost", "Benefit/Cost", "Recommendation"}
	rows := make([][]string, len(costBenefit))

	for i, cb := range costBenefit {
		rows[i] = []string{
			fmt.Sprintf("%d", cb.Connections),
			fmt.Sprintf("%.1f", cb.Throughput),
			fmt.Sprintf("%.2f ms", cb.Latency),
			fmt.Sprintf("%.1f%%", cb.ThroughputPct),
			fmt.Sprintf("%.1f%%", cb.LatencyCost),
			fmt.Sprintf("%.2f", cb.BenefitCostRatio),
			cb.Recommendation,
		}
	}

	return TableData{
		Title:   "Cost-Benefit Analysis",
		Headers: headers,
		Rows:    rows,
		Summary: "Cost-benefit analysis of different connection levels",
	}
}

// Private methods for summary and insights

func (v *Visualizer) generateSummary(bands []metrics.BandResults, costBenefit []metrics.CostBenefitResult) PerformanceSummary {
	if len(bands) == 0 {
		return PerformanceSummary{}
	}

	// Find optimal connections
	optimalConnections := bands[0].Connections
	maxTPS := bands[0].AvgTPS

	for _, band := range bands {
		if band.AvgTPS > maxTPS {
			maxTPS = band.AvgTPS
			optimalConnections = band.Connections
		}
	}

	// Find minimum latency
	minLatency := bands[0].LatencyP99
	for _, band := range bands {
		if band.LatencyP99 < minLatency {
			minLatency = band.LatencyP99
		}
	}

	// Calculate scaling efficiency
	scalingEfficiency := 100.0
	if len(bands) > 1 {
		baseline := bands[0].AvgTPS
		final := bands[len(bands)-1].AvgTPS
		connectionRatio := float64(bands[len(bands)-1].Connections) / float64(bands[0].Connections)
		tpsRatio := final / baseline
		scalingEfficiency = (tpsRatio / connectionRatio) * 100
	}

	// Determine performance grade
	grade := v.calculatePerformanceGrade(scalingEfficiency, costBenefit)

	// Generate key findings
	keyFindings := v.generateKeyFindings(bands, scalingEfficiency)

	return PerformanceSummary{
		OptimalConnections:  optimalConnections,
		MaxThroughput:       maxTPS,
		MinLatency:          minLatency,
		ScalingEfficiency:   scalingEfficiency,
		RecommendedWorkload: fmt.Sprintf("%d connections", optimalConnections),
		PerformanceGrade:    grade,
		KeyFindings:         keyFindings,
	}
}

func (v *Visualizer) calculatePerformanceGrade(scalingEfficiency float64, costBenefit []metrics.CostBenefitResult) string {
	switch {
	case scalingEfficiency >= 90:
		return "A+ (Excellent)"
	case scalingEfficiency >= 80:
		return "A (Very Good)"
	case scalingEfficiency >= 70:
		return "B (Good)"
	case scalingEfficiency >= 60:
		return "C (Fair)"
	case scalingEfficiency >= 50:
		return "D (Poor)"
	default:
		return "F (Failing)"
	}
}

func (v *Visualizer) generateKeyFindings(bands []metrics.BandResults, scalingEfficiency float64) []string {
	findings := []string{}

	if len(bands) < 2 {
		return findings
	}

	// Throughput analysis
	maxTPS := 0.0
	maxTPSConnections := 0
	for _, band := range bands {
		if band.AvgTPS > maxTPS {
			maxTPS = band.AvgTPS
			maxTPSConnections = band.Connections
		}
	}

	findings = append(findings, fmt.Sprintf("Peak throughput of %.1f TPS achieved at %d connections",
		maxTPS, maxTPSConnections))

	// Scaling efficiency
	if scalingEfficiency > 80 {
		findings = append(findings, "Excellent scaling efficiency - system scales well with additional connections")
	} else if scalingEfficiency > 60 {
		findings = append(findings, "Good scaling efficiency with some diminishing returns")
	} else {
		findings = append(findings, "Poor scaling efficiency - consider optimizing before adding more connections")
	}

	// Latency analysis
	latencyIncrease := bands[len(bands)-1].LatencyP99 / bands[0].LatencyP99
	if latencyIncrease > 3 {
		findings = append(findings, "Significant latency degradation at higher connection counts")
	} else if latencyIncrease > 1.5 {
		findings = append(findings, "Moderate latency increase with higher connections")
	}

	return findings
}

func (v *Visualizer) generateInsights(
	bands []metrics.BandResults,
	elasticity []metrics.ElasticityResult,
	queueMetrics []metrics.QueueMetrics,
	costBenefit []metrics.CostBenefitResult,
) []PerformanceInsight {

	insights := []PerformanceInsight{}

	// Elasticity insights
	for _, e := range elasticity {
		if e.Elasticity < 0.2 && e.Elasticity > 0 {
			insights = append(insights, PerformanceInsight{
				Type:        "warning",
				Priority:    "medium",
				Title:       "Diminishing Returns Detected",
				Description: fmt.Sprintf("Segment %s shows significant diminishing returns", e.Segment),
				Action:      "Consider optimizing system before scaling further",
				Evidence:    fmt.Sprintf("Elasticity coefficient: %.2f", e.Elasticity),
			})
		}

		if e.Elasticity < 0 {
			insights = append(insights, PerformanceInsight{
				Type:        "warning",
				Priority:    "high",
				Title:       "Performance Degradation",
				Description: fmt.Sprintf("Segment %s shows negative scaling", e.Segment),
				Action:      "Investigate bottlenecks - reduce connections or optimize queries",
				Evidence:    fmt.Sprintf("Negative elasticity: %.2f", e.Elasticity),
			})
		}
	}

	// Queue analysis insights
	for _, q := range queueMetrics {
		if q.Utilization > 0.9 {
			insights = append(insights, PerformanceInsight{
				Type:     "warning",
				Priority: "high",
				Title:    "High System Utilization",
				Description: fmt.Sprintf("System utilization at %.1f%% with %d connections",
					q.Utilization*100, q.Connections),
				Action:   "System is near saturation - monitor closely",
				Evidence: fmt.Sprintf("Queue length: %.1f requests", q.QueueLength),
			})
		}

		if q.QueueLength > 10 {
			insights = append(insights, PerformanceInsight{
				Type:     "optimization",
				Priority: "medium",
				Title:    "Growing Queue Length",
				Description: fmt.Sprintf("Queue length of %.1f requests at %d connections",
					q.QueueLength, q.Connections),
				Action:   "Consider connection pooling optimization",
				Evidence: fmt.Sprintf("P99 latency: %.2f ms", q.LatencyP99),
			})
		}
	}

	// Cost-benefit insights
	bestRatio := 0.0
	bestConnections := 0
	for _, cb := range costBenefit {
		if cb.BenefitCostRatio > bestRatio {
			bestRatio = cb.BenefitCostRatio
			bestConnections = cb.Connections
		}
	}

	if bestConnections > 0 {
		insights = append(insights, PerformanceInsight{
			Type:        "recommendation",
			Priority:    "high",
			Title:       "Optimal Connection Level",
			Description: fmt.Sprintf("Best benefit/cost ratio at %d connections", bestConnections),
			Action:      fmt.Sprintf("Consider using %d connections for production workload", bestConnections),
			Evidence:    fmt.Sprintf("Benefit/cost ratio: %.2f", bestRatio),
		})
	}

	return insights
}

// Helper methods

func (v *Visualizer) writeCSV(table TableData, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write headers
	if _, err := fmt.Fprintf(file, "%s\n", strings.Join(table.Headers, ",")); err != nil {
		return err
	}

	// Write rows
	for _, row := range table.Rows {
		if _, err := fmt.Fprintf(file, "%s\n", strings.Join(row, ",")); err != nil {
			return err
		}
	}

	return nil
}

func (v *Visualizer) getHTMLTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Metadata.TestName}} - Performance Analysis Report</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { text-align: center; margin-bottom: 30px; }
        .metadata { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; margin-bottom: 30px; }
        .metadata-item { background: #f8f9fa; padding: 15px; border-radius: 5px; }
        .summary { background: #e8f4fd; padding: 20px; border-radius: 5px; margin-bottom: 30px; }
        .chart-container { margin: 20px 0; height: 400px; }
        .table-container { margin: 20px 0; overflow-x: auto; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f8f9fa; font-weight: bold; }
        .insights { margin: 20px 0; }
        .insight { padding: 15px; margin: 10px 0; border-radius: 5px; }
        .insight.warning { background: #fff3cd; border-left: 4px solid #ffc107; }
        .insight.optimization { background: #d1ecf1; border-left: 4px solid #17a2b8; }
        .insight.recommendation { background: #d4edda; border-left: 4px solid #28a745; }
        .grade { font-size: 24px; font-weight: bold; color: #007bff; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Metadata.TestName}}</h1>
            <p>Performance Analysis Report</p>
            <p>Generated on {{.Metadata.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>
        </div>

        <div class="metadata">
            <div class="metadata-item">
                <strong>Workload Type:</strong><br>{{.Metadata.WorkloadType}}
            </div>
            <div class="metadata-item">
                <strong>Duration:</strong><br>{{.Metadata.Duration}}
            </div>
            <div class="metadata-item">
                <strong>Total Bands:</strong><br>{{.Metadata.TotalBands}}
            </div>
            <div class="metadata-item">
                <strong>Strategy:</strong><br>{{.Metadata.Strategy}}
            </div>
        </div>

        <div class="summary">
            <h2>Performance Summary</h2>
            <div class="grade">Grade: {{.Summary.PerformanceGrade}}</div>
            <p><strong>Optimal Connections:</strong> {{.Summary.OptimalConnections}}</p>
            <p><strong>Max Throughput:</strong> {{printf "%.1f" .Summary.MaxThroughput}} TPS</p>
            <p><strong>Min Latency:</strong> {{printf "%.2f" .Summary.MinLatency}} ms</p>
            <p><strong>Scaling Efficiency:</strong> {{printf "%.1f" .Summary.ScalingEfficiency}}%</p>
            
            <h3>Key Findings:</h3>
            <ul>
                {{range .Summary.KeyFindings}}
                <li>{{.}}</li>
                {{end}}
            </ul>
        </div>

        <div class="insights">
            <h2>Performance Insights</h2>
            {{range .Insights}}
            <div class="insight {{.Type}}">
                <h4>{{.Title}} ({{.Priority}} priority)</h4>
                <p>{{.Description}}</p>
                <p><strong>Action:</strong> {{.Action}}</p>
                <p><small>Evidence: {{.Evidence}}</small></p>
            </div>
            {{end}}
        </div>

        {{range $name, $table := .Tables}}
        <div class="table-container">
            <h3>{{$table.Title}}</h3>
            <table>
                <thead>
                    <tr>
                        {{range $table.Headers}}
                        <th>{{.}}</th>
                        {{end}}
                    </tr>
                </thead>
                <tbody>
                    {{range $table.Rows}}
                    <tr>
                        {{range .}}
                        <td>{{.}}</td>
                        {{end}}
                    </tr>
                    {{end}}
                </tbody>
            </table>
            {{if $table.Summary}}
            <p><small>{{$table.Summary}}</small></p>
            {{end}}
        </div>
        {{end}}
    </div>
</body>
</html>`
}
