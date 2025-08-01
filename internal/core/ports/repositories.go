// internal/core/ports/repositories.go
package ports

import (
	"context"
	"time"

	"github.com/elchinoo/stormdb/internal/core/domain"
)

// TestExecutionRepository manages persistence of test executions and results
type TestExecutionRepository interface {
	// Test execution management
	Store(ctx context.Context, execution *domain.TestExecution) error
	GetByID(ctx context.Context, id string) (*domain.TestExecution, error)
	List(ctx context.Context, filters TestExecutionFilters) ([]*domain.TestExecution, error)
	Delete(ctx context.Context, id string) error

	// Results management
	StoreResults(ctx context.Context, executionID string, results *domain.TestResults) error
	GetResults(ctx context.Context, executionID string) (*domain.TestResults, error)

	// Analytics queries
	GetPerformanceTrends(ctx context.Context, workloadType string, timeRange TimeRange) ([]PerformanceTrend, error)
	CompareExecutions(ctx context.Context, executionIDs []string) (*ExecutionComparison, error)
}

// MetricsRepository handles storage and retrieval of detailed metrics
type MetricsRepository interface {
	// Raw metrics storage (streaming-friendly)
	StoreMetricBatch(ctx context.Context, executionID string, bandID int, metrics []MetricPoint) error

	// Aggregated metrics
	StoreAggregatedMetrics(ctx context.Context, executionID string, bandID int, aggregated *domain.BandResults) error
	GetAggregatedMetrics(ctx context.Context, executionID string) ([]domain.BandResults, error)

	// Time-series data
	GetTimeSeriesData(ctx context.Context, executionID string, bandID int, metricType string) ([]TimeSeriesPoint, error)

	// Cleanup old data
	CleanupOldMetrics(ctx context.Context, olderThan time.Time) error
}

// ConfigurationRepository manages test configurations and templates
type ConfigurationRepository interface {
	// Configuration management
	StoreConfiguration(ctx context.Context, config *domain.TestConfiguration) error
	GetConfiguration(ctx context.Context, name string) (*domain.TestConfiguration, error)
	ListConfigurations(ctx context.Context) ([]*ConfigurationSummary, error)
	DeleteConfiguration(ctx context.Context, name string) error

	// Templates
	GetTemplate(ctx context.Context, workloadType string) (*domain.TestConfiguration, error)
	ListTemplates(ctx context.Context) ([]*ConfigurationSummary, error)
}

// AnalysisService defines advanced mathematical analysis capabilities
type AnalysisService interface {
	// Statistical analysis
	CalculateStatistics(bands []domain.BandResults) (*domain.PerformanceAnalysis, error)

	// Curve fitting and modeling
	FitModel(xValues, yValues []float64, modelType domain.ModelType) (*ModelResults, error)
	FindBestFitModel(xValues, yValues []float64) (*ModelResults, error)

	// Derivatives and trends
	CalculateDerivatives(xValues, yValues []float64) (first, second []float64, err error)
	DetectInflectionPoints(xValues, yValues []float64) ([]InflectionPoint, error)

	// Bottleneck identification
	ClassifyBottleneck(bands []domain.BandResults) domain.BottleneckType
	IdentifyScalingRegions(bands []domain.BandResults) []domain.ScalingRegion

	// Recommendations
	GenerateRecommendations(analysis *domain.PerformanceAnalysis) *domain.RecommendedConfiguration
	PredictPerformance(analysis *domain.PerformanceAnalysis, workers, connections int) (*domain.PerformancePrediction, error)
}

// StreamingMetricsCollector provides memory-efficient metrics collection
type StreamingMetricsCollector interface {
	domain.MetricsCollector

	// Streaming statistics (Welford's method for memory efficiency)
	StartCollection(bandID int, expectedDuration time.Duration)
	StopCollection() *domain.BandResults

	// Real-time monitoring
	GetCurrentSnapshot() *MetricsSnapshot
	RegisterListener(listener MetricsListener)

	// Memory management
	SetMemoryLimits(maxLatencySamples, maxTPSSamples int)
	GetMemoryUsage() MemoryUsage
}

// WorkloadRegistry manages available workload implementations
type WorkloadRegistry interface {
	Register(name string, workload domain.WorkloadInterface)
	Get(name string) (domain.WorkloadInterface, error)
	List() []WorkloadInfo

	// Plugin support
	LoadPlugin(pluginPath string) error
	UnloadPlugin(name string) error
}

// TestExecutionEngine orchestrates test execution
type TestExecutionEngine interface {
	// Single test execution
	Execute(ctx context.Context, config *domain.TestConfiguration) (*domain.TestResults, error)

	// Progressive scaling execution
	ExecuteProgressive(ctx context.Context, config *domain.TestConfiguration,
		progressCallback ProgressCallback) (*domain.TestResults, error)

	// Execution control
	Cancel(ctx context.Context, executionID string) error
	GetStatus(ctx context.Context, executionID string) (*ExecutionStatus, error)
}

// Supporting types for interfaces

type TestExecutionFilters struct {
	WorkloadType *string
	Status       *domain.ExecutionStatus
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
	Offset       int
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}

type PerformanceTrend struct {
	Timestamp time.Time
	TPS       float64
	Latency   float64
	Workers   int
}

type ExecutionComparison struct {
	Executions []ExecutionSummary
	Analysis   ComparisonAnalysis
}

type ExecutionSummary struct {
	ID           string
	Name         string
	WorkloadType string
	TPS          float64
	Latency      float64
	Workers      int
	Connections  int
	Timestamp    time.Time
}

type ComparisonAnalysis struct {
	BestPerformer   string
	MostEfficient   string
	MostStable      string
	Recommendations []string
}

type MetricPoint struct {
	Timestamp  time.Time
	MetricType string
	Value      float64
	BandID     int
	WorkerID   *int
}

type TimeSeriesPoint struct {
	Timestamp time.Time
	Value     float64
}

type ConfigurationSummary struct {
	Name         string
	WorkloadType string
	Description  string
	LastUsed     *time.Time
	CreatedAt    time.Time
}

type ModelResults struct {
	ModelType        domain.ModelType
	Coefficients     []float64
	GoodnessOfFit    float64 // R-squared
	Predictions      []float64
	ConfidenceBounds []domain.ConfidenceInterval
}

type InflectionPoint struct {
	X         float64
	Y         float64
	Direction string // "increasing" or "decreasing"
}

type MetricsSnapshot struct {
	Timestamp     time.Time
	TPS           float64
	QPS           float64
	LatencyP50    float64
	LatencyP95    float64
	LatencyP99    float64
	ErrorRate     float64
	ActiveWorkers int
}

type MetricsListener interface {
	OnSnapshot(snapshot *MetricsSnapshot)
	OnBandComplete(bandID int, results *domain.BandResults)
	OnError(err error)
}

type MemoryUsage struct {
	LatencySamplesCount int
	TPSSamplesCount     int
	EstimatedMemoryMB   float64
}

type WorkloadInfo struct {
	Name           string
	Description    string
	SupportedModes []string
	IsPlugin       bool
	Version        string
}

type ProgressCallback func(bandID int, totalBands int, currentResults *domain.BandResults)

type ExecutionStatus struct {
	Status       domain.ExecutionStatus
	CurrentBand  int
	TotalBands   int
	Progress     float64 // 0.0 to 1.0
	LastUpdate   time.Time
	ErrorMessage string
}
