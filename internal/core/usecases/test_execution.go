// internal/core/usecases/test_execution.go
package usecases

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/elchinoo/stormdb/internal/core/domain"
	"github.com/elchinoo/stormdb/internal/core/ports"
	"github.com/google/uuid"
)

// TestExecutionUseCase orchestrates the complete test execution lifecycle
type TestExecutionUseCase struct {
	testRepo         ports.TestExecutionRepository
	metricsRepo      ports.MetricsRepository
	configRepo       ports.ConfigurationRepository
	analysisService  ports.AnalysisService
	metricsCollector ports.StreamingMetricsCollector
	workloadRegistry ports.WorkloadRegistry
	executionEngine  ports.TestExecutionEngine

	// Active executions tracking (thread-safe)
	activeExecutions sync.Map // map[string]*ExecutionContext
}

type ExecutionContext struct {
	ID          string
	CancelFunc  context.CancelFunc
	Status      domain.ExecutionStatus
	StartTime   time.Time
	CurrentBand int
	TotalBands  int
	Results     *domain.TestResults
	Config      *domain.TestConfiguration
	mutex       sync.RWMutex
}

func NewTestExecutionUseCase(
	testRepo ports.TestExecutionRepository,
	metricsRepo ports.MetricsRepository,
	configRepo ports.ConfigurationRepository,
	analysisService ports.AnalysisService,
	metricsCollector ports.StreamingMetricsCollector,
	workloadRegistry ports.WorkloadRegistry,
	executionEngine ports.TestExecutionEngine,
) *TestExecutionUseCase {
	return &TestExecutionUseCase{
		testRepo:         testRepo,
		metricsRepo:      metricsRepo,
		configRepo:       configRepo,
		analysisService:  analysisService,
		metricsCollector: metricsCollector,
		workloadRegistry: workloadRegistry,
		executionEngine:  executionEngine,
	}
}

// ExecuteTest runs a complete test with progressive scaling and comprehensive analysis
func (uc *TestExecutionUseCase) ExecuteTest(ctx context.Context, configName string, options ExecutionOptions) (*TestExecutionResult, error) {
	// 1. Load and validate configuration
	config, err := uc.configRepo.GetConfiguration(ctx, configName)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration '%s': %w", configName, err)
	}

	if err := uc.validateConfiguration(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// 2. Create execution context
	executionID := uuid.New().String()
	executionCtx, cancelFunc := context.WithCancel(ctx)

	execContext := &ExecutionContext{
		ID:         executionID,
		CancelFunc: cancelFunc,
		Status:     domain.StatusRunning,
		StartTime:  time.Now(),
		TotalBands: len(config.ProgressiveConfig.WorkerSteps),
		Config:     config,
	}

	uc.activeExecutions.Store(executionID, execContext)
	defer uc.activeExecutions.Delete(executionID)

	// 3. Create test execution record
	testExecution := &domain.TestExecution{
		ID:           executionID,
		Name:         options.Name,
		WorkloadType: config.WorkloadType,
		Config:       *config,
		Status:       domain.StatusRunning,
		StartTime:    time.Now(),
	}

	if err := uc.testRepo.Store(ctx, testExecution); err != nil {
		return nil, fmt.Errorf("failed to store test execution: %w", err)
	}

	// 4. Execute test with progressive scaling
	results, err := uc.executeWithProgressiveScaling(executionCtx, execContext, config, options.ProgressCallback)
	if err != nil {
		uc.updateExecutionStatus(executionID, domain.StatusFailed, err.Error())
		return nil, fmt.Errorf("test execution failed: %w", err)
	}

	// 5. Perform comprehensive analysis
	if results.ProgressiveResults != nil {
		analysis, err := uc.analysisService.CalculateStatistics(results.ProgressiveResults.Bands)
		if err != nil {
			log.Printf("Warning: failed to calculate analysis: %v", err)
		} else {
			results.Analysis = analysis
		}
	}

	// 6. Store final results
	testExecution.Status = domain.StatusCompleted
	testExecution.EndTime = &time.Time{}
	*testExecution.EndTime = time.Now()
	testExecution.Results = results

	if err := uc.testRepo.Store(ctx, testExecution); err != nil {
		log.Printf("Warning: failed to update test execution: %v", err)
	}

	if err := uc.testRepo.StoreResults(ctx, executionID, results); err != nil {
		log.Printf("Warning: failed to store test results: %v", err)
	}

	return &TestExecutionResult{
		ExecutionID: executionID,
		Results:     results,
		Duration:    time.Since(testExecution.StartTime),
	}, nil
}

// executeWithProgressiveScaling implements the core progressive scaling logic
func (uc *TestExecutionUseCase) executeWithProgressiveScaling(
	ctx context.Context,
	execContext *ExecutionContext,
	config *domain.TestConfiguration,
	progressCallback func(bandID int, totalBands int, results *domain.BandResults),
) (*domain.TestResults, error) {

	if config.ProgressiveConfig == nil {
		return nil, fmt.Errorf("progressive scaling configuration is required")
	}

	workers := config.ProgressiveConfig.WorkerSteps
	connections := config.ProgressiveConfig.ConnectionSteps
	bandDuration := config.ProgressiveConfig.BandDuration

	if len(workers) != len(connections) {
		return nil, fmt.Errorf("workers and connections arrays must have the same length")
	}

	var bandResults []domain.BandResults

	for bandID, workerCount := range workers {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("execution cancelled")
		default:
		}

		// Update execution context
		execContext.mutex.Lock()
		execContext.CurrentBand = bandID
		execContext.mutex.Unlock()

		log.Printf("Starting band %d: %d workers, %d connections", bandID, workerCount, connections[bandID])

		// Execute band
		bandResult, err := uc.executeBand(ctx, bandID, config, workerCount, connections[bandID], bandDuration)
		if err != nil {
			return nil, fmt.Errorf("band %d failed: %w", bandID, err)
		}

		bandResults = append(bandResults, *bandResult)

		// Store intermediate results
		if err := uc.metricsRepo.StoreAggregatedMetrics(ctx, execContext.ID, bandID, bandResult); err != nil {
			log.Printf("Warning: failed to store band %d metrics: %v", bandID, err)
		}

		// Notify progress
		if progressCallback != nil {
			progressCallback(bandID, len(workers), bandResult)
		}

		log.Printf("Band %d completed: TPS=%.2f, Latency P95=%.2fms",
			bandID, bandResult.Performance.TotalTPS, bandResult.Performance.P95Latency)
	}

	results := &domain.TestResults{
		ProgressiveResults: &domain.ProgressiveResults{
			Bands: bandResults,
		},
	}

	// Find optimal band
	if len(bandResults) > 0 {
		optimalIdx := 0
		maxTPS := bandResults[0].Performance.TotalTPS
		for i, band := range bandResults {
			if band.Performance.TotalTPS > maxTPS {
				maxTPS = band.Performance.TotalTPS
				optimalIdx = i
			}
		}
		results.ProgressiveResults.OptimalBand = &bandResults[optimalIdx]
	}

	return results, nil
}

// executeBand runs a single band with streaming metrics collection
func (uc *TestExecutionUseCase) executeBand(
	ctx context.Context,
	bandID int,
	config *domain.TestConfiguration,
	workers int,
	connections int,
	duration time.Duration,
) (*domain.BandResults, error) {

	// Get workload implementation
	workload, err := uc.workloadRegistry.Get(config.WorkloadType)
	if err != nil {
		return nil, fmt.Errorf("failed to get workload '%s': %w", config.WorkloadType, err)
	}

	// Start streaming metrics collection
	uc.metricsCollector.StartCollection(bandID, duration)
	defer func() {
		// Ensure collection is stopped
		uc.metricsCollector.StopCollection()
	}()

	// Create band context with timeout
	bandCtx, cancel := context.WithTimeout(ctx, duration+10*time.Second) // Buffer for cleanup
	defer cancel()

	// Create configuration for this band
	bandConfig := *config // Copy the config
	// Note: Workers and connections are handled by the workload implementation
	// They should be passed through WorkloadParams or similar mechanism
	if bandConfig.WorkloadParams == nil {
		bandConfig.WorkloadParams = make(map[string]interface{})
	}
	bandConfig.WorkloadParams["workers"] = workers
	bandConfig.WorkloadParams["connections"] = connections

	// Execute workload
	startTime := time.Now()
	err = workload.Run(bandCtx, bandConfig, uc.metricsCollector)
	endTime := time.Now()
	actualDuration := endTime.Sub(startTime)

	if err != nil {
		return nil, fmt.Errorf("workload execution failed: %w", err)
	}

	// Collect final metrics
	bandResults := uc.metricsCollector.StopCollection()
	bandResults.BandID = bandID
	bandResults.Workers = workers
	bandResults.Connections = connections
	bandResults.Duration = actualDuration

	return bandResults, nil
}

// validateConfiguration ensures the configuration is valid for execution
func (uc *TestExecutionUseCase) validateConfiguration(config *domain.TestConfiguration) error {
	if config.WorkloadType == "" {
		return fmt.Errorf("workload type is required")
	}

	if _, err := uc.workloadRegistry.Get(config.WorkloadType); err != nil {
		return fmt.Errorf("unsupported workload type '%s': %w", config.WorkloadType, err)
	}

	if config.ProgressiveConfig == nil {
		return fmt.Errorf("progressive scaling configuration is required")
	}

	if config.ProgressiveConfig.BandDuration <= 0 {
		return fmt.Errorf("band duration must be positive")
	}

	if len(config.ProgressiveConfig.WorkerSteps) == 0 {
		return fmt.Errorf("at least one worker configuration is required")
	}

	if len(config.ProgressiveConfig.WorkerSteps) != len(config.ProgressiveConfig.ConnectionSteps) {
		return fmt.Errorf("workers and connections arrays must have the same length")
	}

	for i, workers := range config.ProgressiveConfig.WorkerSteps {
		if workers <= 0 {
			return fmt.Errorf("workers count must be positive (band %d)", i)
		}
		if config.ProgressiveConfig.ConnectionSteps[i] <= 0 {
			return fmt.Errorf("connections count must be positive (band %d)", i)
		}
	}

	return nil
}

// updateExecutionStatus updates the status of an active execution
func (uc *TestExecutionUseCase) updateExecutionStatus(executionID string, status domain.ExecutionStatus, errorMsg string) {
	if execContextValue, exists := uc.activeExecutions.Load(executionID); exists {
		execContext := execContextValue.(*ExecutionContext)
		execContext.mutex.Lock()
		execContext.Status = status
		execContext.mutex.Unlock()

		// Also update in repository (best effort)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if testExecution, err := uc.testRepo.GetByID(ctx, executionID); err == nil {
			testExecution.Status = status
			if errorMsg != "" {
				// Store error in a simple way - the domain.TestExecution.Error field
				testExecution.Error = fmt.Errorf("%s", errorMsg)
			}
			if status == domain.StatusCompleted || status == domain.StatusFailed {
				now := time.Now()
				testExecution.EndTime = &now
			}
			if err := uc.testRepo.Store(ctx, testExecution); err != nil {
				// Log error but don't stop monitoring
			}
		}
	}
}

// GetExecutionStatus returns the current status of an execution
func (uc *TestExecutionUseCase) GetExecutionStatus(executionID string) (*ExecutionStatusInfo, bool) {
	if execContextValue, exists := uc.activeExecutions.Load(executionID); exists {
		execContext := execContextValue.(*ExecutionContext)
		execContext.mutex.RLock()
		defer execContext.mutex.RUnlock()

		progress := float64(execContext.CurrentBand) / float64(execContext.TotalBands)

		return &ExecutionStatusInfo{
			ID:          execContext.ID,
			Status:      execContext.Status,
			CurrentBand: execContext.CurrentBand,
			TotalBands:  execContext.TotalBands,
			Progress:    progress,
			Duration:    time.Since(execContext.StartTime),
		}, true
	}
	return nil, false
}

// CancelExecution cancels a running execution
func (uc *TestExecutionUseCase) CancelExecution(executionID string) error {
	if execContextValue, exists := uc.activeExecutions.Load(executionID); exists {
		execContext := execContextValue.(*ExecutionContext)
		execContext.CancelFunc()
		uc.updateExecutionStatus(executionID, domain.StatusCancelled, "Cancelled by user")
		return nil
	}
	return fmt.Errorf("execution %s not found or not active", executionID)
}

// Supporting types for the use case

type ExecutionOptions struct {
	Name             string
	ProgressCallback func(bandID int, totalBands int, results *domain.BandResults)
}

type TestExecutionResult struct {
	ExecutionID string
	Results     *domain.TestResults
	Duration    time.Duration
}

type ExecutionStatusInfo struct {
	ID          string
	Status      domain.ExecutionStatus
	CurrentBand int
	TotalBands  int
	Progress    float64
	Duration    time.Duration
}
