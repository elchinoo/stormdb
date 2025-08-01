// internal/core/domain/entities.go
package domain

import (
	"context"
	"time"
)

// TestExecution represents a complete test run with its configuration and results
type TestExecution struct {
	ID           string
	Name         string
	WorkloadType string
	Config       TestConfiguration
	Results      *TestResults
	Status       ExecutionStatus
	StartTime    time.Time
	EndTime      *time.Time
	Error        error
}

// TestConfiguration holds all test parameters
type TestConfiguration struct {
	// Core parameters
	WorkloadType string
	Duration     time.Duration

	// Progressive scaling parameters
	ProgressiveConfig *ProgressiveScalingConfig

	// Database connection
	DatabaseConfig DatabaseConfiguration

	// Workload-specific parameters
	WorkloadParams map[string]interface{}
}

// ProgressiveScalingConfig defines progressive scaling behavior
type ProgressiveScalingConfig struct {
	// Connection scaling
	MinConnections  int
	MaxConnections  int
	ConnectionSteps []int // Exact steps, or nil for calculated steps

	// Worker scaling
	MinWorkers  int
	MaxWorkers  int
	WorkerSteps []int // Exact steps, or nil for calculated steps

	// Timing
	BandDuration time.Duration
	WarmupTime   time.Duration
	CooldownTime time.Duration

	// Scaling strategy
	Strategy ScalingStrategy

	// Memory management
	MaxMemoryMB       int
	MaxLatencySamples int
	SampleInterval    time.Duration
}

// ScalingStrategy defines how connections/workers are increased
type ScalingStrategy string

const (
	StrategyLinear      ScalingStrategy = "linear"
	StrategyExponential ScalingStrategy = "exponential"
	StrategyFibonacci   ScalingStrategy = "fibonacci"
	StrategyCustom      ScalingStrategy = "custom"
)

// DatabaseConfiguration holds database connection details
type DatabaseConfiguration struct {
	Host        string
	Port        int
	Database    string
	Username    string
	Password    string
	SSLMode     string
	MaxPoolSize int
}

// ExecutionStatus represents the current state of a test execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
)

// TestResults contains all results from a test execution
type TestResults struct {
	// Progressive scaling results
	ProgressiveResults *ProgressiveResults

	// Single band results (for non-progressive tests)
	SingleBandResults *BandResults

	// Analysis results
	Analysis *PerformanceAnalysis

	// Raw metrics (limited for memory efficiency)
	RawMetrics *RawMetrics
}

// ProgressiveResults holds results from progressive scaling tests
type ProgressiveResults struct {
	Bands             []BandResults
	OptimalBand       *BandResults
	TotalCapacity     float64 // Area under the curve
	ScalingEfficiency float64
	BottleneckType    BottleneckType
}

// BandResults represents metrics for a single band (connection/worker configuration)
type BandResults struct {
	BandID      int
	Workers     int
	Connections int
	Duration    time.Duration

	// Core performance metrics
	Performance PerformanceMetrics

	// Efficiency metrics
	Efficiency EfficiencyMetrics

	// Stability metrics
	Stability StabilityMetrics

	// Resource utilization
	Resources ResourceMetrics
}

// PerformanceMetrics holds core performance measurements
type PerformanceMetrics struct {
	// Throughput
	TotalTPS float64
	TotalQPS float64

	// Latency (in milliseconds)
	AvgLatency float64
	P50Latency float64
	P95Latency float64
	P99Latency float64

	// Errors
	ErrorCount int64
	ErrorRate  float64 // Percentage

	// Operation counts
	SelectQueries int64
	InsertQueries int64
	UpdateQueries int64
	DeleteQueries int64

	// Row counts
	RowsRead     int64
	RowsModified int64
}

// EfficiencyMetrics measures efficiency per resource unit
type EfficiencyMetrics struct {
	TPSPerWorker     float64
	TPSPerConnection float64
	MarginalGain     float64 // TPS gain compared to previous band
	MarginalCost     float64 // Resource cost for marginal gain
	ROI              float64 // Return on investment
}

// StabilityMetrics measures consistency and reliability
type StabilityMetrics struct {
	// Variability
	TPSStdDev              float64
	LatencyStdDev          float64
	CoefficientOfVariation float64

	// Confidence intervals
	TPSConfidenceInterval     ConfidenceInterval
	LatencyConfidenceInterval ConfidenceInterval

	// Consistency over time
	PerformanceDrift float64 // Change from start to end of band
}

// ConfidenceInterval represents a statistical confidence interval
type ConfidenceInterval struct {
	Lower      float64
	Upper      float64
	Confidence float64 // e.g., 0.95 for 95%
}

// ResourceMetrics tracks resource utilization
type ResourceMetrics struct {
	ConnectionUtilization float64 // Percentage
	WorkerUtilization     float64 // Percentage
	MemoryUsageMB         float64
	CPUUtilization        float64 // If available
}

// PerformanceAnalysis contains advanced mathematical analysis
type PerformanceAnalysis struct {
	// Derivatives and trends
	FirstDerivative  []float64 // Marginal gains
	SecondDerivative []float64 // Inflection points

	// Model fitting
	BestFitModel       ModelType
	ModelCoefficients  []float64
	ModelGoodnessOfFit float64 // R-squared

	// Classifications
	ScalingRegions []ScalingRegion
	BottleneckType BottleneckType

	// Recommendations
	OptimalConfiguration   RecommendedConfiguration
	PerformancePredictions []PerformancePrediction
}

// ModelType represents different mathematical models for curve fitting
type ModelType string

const (
	ModelLinear      ModelType = "linear"
	ModelLogarithmic ModelType = "logarithmic"
	ModelExponential ModelType = "exponential"
	ModelLogistic    ModelType = "logistic"
	ModelPolynomial  ModelType = "polynomial"
)

// ScalingRegion classifies performance behavior in different scaling ranges
type ScalingRegion struct {
	StartConnections int
	EndConnections   int
	Classification   RegionClassification
	Description      string
}

// RegionClassification categorizes scaling behavior
type RegionClassification string

const (
	RegionBaseline           RegionClassification = "baseline"
	RegionLinearScaling      RegionClassification = "linear_scaling"
	RegionDiminishingReturns RegionClassification = "diminishing_returns"
	RegionSaturation         RegionClassification = "saturation"
	RegionDegradation        RegionClassification = "degradation"
)

// BottleneckType identifies the primary system constraint
type BottleneckType string

const (
	BottleneckNone       BottleneckType = "none"
	BottleneckCPU        BottleneckType = "cpu_bound"
	BottleneckIO         BottleneckType = "io_bound"
	BottleneckMemory     BottleneckType = "memory_bound"
	BottleneckQueue      BottleneckType = "queue_bound"
	BottleneckConnection BottleneckType = "connection_bound"
	BottleneckDatabase   BottleneckType = "database_bound"
)

// RecommendedConfiguration provides optimal settings
type RecommendedConfiguration struct {
	OptimalWorkers     int
	OptimalConnections int
	ExpectedTPS        float64
	ExpectedLatency    float64
	Confidence         float64
	Reasoning          string
}

// PerformancePrediction forecasts performance at untested configurations
type PerformancePrediction struct {
	Workers          int
	Connections      int
	PredictedTPS     float64
	PredictedLatency float64
	ConfidenceBounds ConfidenceInterval
	ModelUsed        ModelType
}

// RawMetrics contains limited raw measurement data for analysis
type RawMetrics struct {
	// Streaming statistics (memory efficient)
	LatencyHistogram map[int]int64 // Bucket -> count
	TPSSamples       []float64     // Limited samples for analysis
	QPSSamples       []float64     // Limited samples for analysis

	// Error details
	ErrorTypes map[string]int64

	// Timestamps for time-series analysis
	SampleTimestamps []time.Time
}

// WorkloadInterface defines the contract for all workload implementations
type WorkloadInterface interface {
	// Lifecycle
	Setup(ctx context.Context, config TestConfiguration) error
	Cleanup(ctx context.Context, config TestConfiguration) error

	// Execution
	Run(ctx context.Context, config TestConfiguration, collector MetricsCollector) error

	// Metadata
	Name() string
	Description() string
	SupportedModes() []string
}

// MetricsCollector defines the interface for collecting performance metrics
type MetricsCollector interface {
	// Basic metrics
	RecordTransaction(success bool, latencyNs int64)
	RecordQuery(queryType string, rowsAffected int64)
	RecordError(err error)

	// Advanced metrics
	RecordCustomMetric(name string, value float64)

	// Streaming statistics (memory efficient)
	GetCurrentTPS() float64
	GetCurrentLatencyP95() float64
	GetCurrentErrorRate() float64

	// Snapshot for analysis (limited data)
	TakeSnapshot() *RawMetrics
}
