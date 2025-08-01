// Package resilience provides resilience features like checkpointing, circuit breakers, and recovery
package resilience

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/elchinoo/stormdb/pkg/types"
	"go.uber.org/zap"
)

// CheckpointManager handles saving and restoring test state
type CheckpointManager struct {
	mu       sync.RWMutex
	logger   *zap.Logger
	basePath string
	interval time.Duration
	maxFiles int
	enabled  bool
	ctx      context.Context
	cancel   context.CancelFunc

	// Current checkpoint state
	currentCheckpoint *Checkpoint
}

// Checkpoint represents a point-in-time snapshot of test execution
type Checkpoint struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	TestMetadata  TestMetadata           `json:"test_metadata"`
	BandProgress  BandProgress           `json:"band_progress"`
	Metrics       CheckpointMetrics      `json:"metrics"`
	Configuration map[string]interface{} `json:"configuration"`
	State         TestState              `json:"state"`
	Recovery      RecoveryInfo           `json:"recovery_info"`
}

// TestMetadata contains information about the test
type TestMetadata struct {
	TestID       string    `json:"test_id"`
	StartTime    time.Time `json:"start_time"`
	WorkloadType string    `json:"workload_type"`
	Strategy     string    `json:"strategy"`
	TotalBands   int       `json:"total_bands"`
	CurrentBand  int       `json:"current_band"`
}

// BandProgress tracks progress through progressive scaling bands
type BandProgress struct {
	CompletedBands []BandResult `json:"completed_bands"`
	CurrentBand    BandState    `json:"current_band"`
	RemainingBands []BandPlan   `json:"remaining_bands"`
}

// BandResult contains results from a completed band
type BandResult struct {
	BandNumber  int            `json:"band_number"`
	Connections int            `json:"connections"`
	Workers     int            `json:"workers"`
	Duration    time.Duration  `json:"duration"`
	StartTime   time.Time      `json:"start_time"`
	EndTime     time.Time      `json:"end_time"`
	Metrics     *types.Metrics `json:"metrics"`
	Successful  bool           `json:"successful"`
	ErrorCount  int            `json:"error_count"`
	Errors      []string       `json:"errors,omitempty"`
}

// BandState tracks the current band execution state
type BandState struct {
	BandNumber       int            `json:"band_number"`
	Connections      int            `json:"connections"`
	Workers          int            `json:"workers"`
	StartTime        time.Time      `json:"start_time"`
	ElapsedTime      time.Duration  `json:"elapsed_time"`
	ExpectedDuration time.Duration  `json:"expected_duration"`
	CompletionPct    float64        `json:"completion_percentage"`
	CurrentMetrics   *types.Metrics `json:"current_metrics"`
}

// BandPlan defines parameters for future bands
type BandPlan struct {
	BandNumber  int           `json:"band_number"`
	Connections int           `json:"connections"`
	Workers     int           `json:"workers"`
	Duration    time.Duration `json:"duration"`
}

// CheckpointMetrics contains aggregated metrics for checkpointing
type CheckpointMetrics struct {
	TotalTransactions int64                 `json:"total_transactions"`
	TotalErrors       int64                 `json:"total_errors"`
	AverageLatency    float64               `json:"average_latency"`
	ThroughputTrend   []ThroughputDataPoint `json:"throughput_trend"`
	LatencyTrend      []LatencyDataPoint    `json:"latency_trend"`
	ErrorRate         float64               `json:"error_rate"`
	ResourceUsage     ResourceUsageSnapshot `json:"resource_usage"`
}

// ThroughputDataPoint represents a point in throughput trend
type ThroughputDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	TPS       float64   `json:"tps"`
	BandID    int       `json:"band_id"`
}

// LatencyDataPoint represents a point in latency trend
type LatencyDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	P50       float64   `json:"p50"`
	P95       float64   `json:"p95"`
	P99       float64   `json:"p99"`
	BandID    int       `json:"band_id"`
}

// ResourceUsageSnapshot captures resource usage at checkpoint time
type ResourceUsageSnapshot struct {
	CPUPercent   float64   `json:"cpu_percent"`
	MemoryMB     float64   `json:"memory_mb"`
	Goroutines   int       `json:"goroutines"`
	OpenFiles    int       `json:"open_files"`
	NetworkConns int       `json:"network_connections"`
	Timestamp    time.Time `json:"timestamp"`
}

// TestState represents the overall test execution state
type TestState struct {
	Status   string                 `json:"status"`   // "running", "paused", "completed", "failed"
	Phase    string                 `json:"phase"`    // "warmup", "execution", "cooldown", "analysis"
	Progress float64                `json:"progress"` // 0.0 to 1.0
	Errors   []ExecutionError       `json:"errors"`
	Warnings []string               `json:"warnings"`
	Context  map[string]interface{} `json:"context"`
}

// ExecutionError represents an error during test execution
type ExecutionError struct {
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Component   string    `json:"component"`
	Recoverable bool      `json:"recoverable"`
	Stack       string    `json:"stack,omitempty"`
}

// RecoveryInfo contains information needed for recovery
type RecoveryInfo struct {
	CanRecover       bool             `json:"can_recover"`
	RecoveryStrategy string           `json:"recovery_strategy"`
	LastGoodState    time.Time        `json:"last_good_state"`
	FailureReason    string           `json:"failure_reason,omitempty"`
	RecoveryActions  []RecoveryAction `json:"recovery_actions"`
}

// RecoveryAction defines an action to take during recovery
type RecoveryAction struct {
	Type        string                 `json:"type"` // "skip_band", "retry_band", "reduce_load", "abort"
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager(logger *zap.Logger, basePath string) *CheckpointManager {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &CheckpointManager{
		logger:   logger,
		basePath: basePath,
		interval: 30 * time.Second, // Default checkpoint interval
		maxFiles: 10,               // Keep last 10 checkpoints
		enabled:  true,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Configure sets checkpoint manager options
func (cm *CheckpointManager) Configure(interval time.Duration, maxFiles int, enabled bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.interval = interval
	cm.maxFiles = maxFiles
	cm.enabled = enabled

	cm.logger.Info("Checkpoint manager configured",
		zap.Duration("interval", interval),
		zap.Int("max_files", maxFiles),
		zap.Bool("enabled", enabled))
}

// StartPeriodicCheckpoints starts automatic checkpoint creation
func (cm *CheckpointManager) StartPeriodicCheckpoints() {
	if !cm.enabled {
		return
	}

	go cm.checkpointRoutine()
	cm.logger.Info("Periodic checkpoints started", zap.Duration("interval", cm.interval))
}

// CreateCheckpoint creates a checkpoint of the current test state
func (cm *CheckpointManager) CreateCheckpoint(
	testMetadata TestMetadata,
	bandProgress BandProgress,
	metrics CheckpointMetrics,
	config map[string]interface{},
	state TestState,
) error {

	if !cm.enabled {
		return nil
	}

	checkpoint := &Checkpoint{
		ID:            cm.generateCheckpointID(),
		Timestamp:     time.Now(),
		TestMetadata:  testMetadata,
		BandProgress:  bandProgress,
		Metrics:       metrics,
		Configuration: config,
		State:         state,
		Recovery:      cm.generateRecoveryInfo(state, bandProgress),
	}

	cm.mu.Lock()
	cm.currentCheckpoint = checkpoint
	cm.mu.Unlock()

	// Save to disk
	if err := cm.saveCheckpoint(checkpoint); err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	// Clean up old checkpoints
	if err := cm.cleanupOldCheckpoints(); err != nil {
		cm.logger.Warn("Failed to cleanup old checkpoints", zap.Error(err))
	}

	cm.logger.Info("Checkpoint created",
		zap.String("id", checkpoint.ID),
		zap.String("status", state.Status),
		zap.Float64("progress", state.Progress))

	return nil
}

// RestoreFromCheckpoint restores test state from the latest checkpoint
func (cm *CheckpointManager) RestoreFromCheckpoint() (*Checkpoint, error) {
	checkpointPath, err := cm.findLatestCheckpoint()
	if err != nil {
		return nil, fmt.Errorf("failed to find latest checkpoint: %w", err)
	}

	if checkpointPath == "" {
		return nil, fmt.Errorf("no checkpoints found")
	}

	checkpoint, err := cm.loadCheckpoint(checkpointPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	cm.mu.Lock()
	cm.currentCheckpoint = checkpoint
	cm.mu.Unlock()

	cm.logger.Info("Restored from checkpoint",
		zap.String("id", checkpoint.ID),
		zap.Time("timestamp", checkpoint.Timestamp),
		zap.String("status", checkpoint.State.Status))

	return checkpoint, nil
}

// RestoreFromSpecificCheckpoint restores from a specific checkpoint ID
func (cm *CheckpointManager) RestoreFromSpecificCheckpoint(checkpointID string) (*Checkpoint, error) {
	checkpointPath := cm.getCheckpointPath(checkpointID)

	checkpoint, err := cm.loadCheckpoint(checkpointPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint %s: %w", checkpointID, err)
	}

	cm.mu.Lock()
	cm.currentCheckpoint = checkpoint
	cm.mu.Unlock()

	cm.logger.Info("Restored from specific checkpoint",
		zap.String("id", checkpointID),
		zap.Time("timestamp", checkpoint.Timestamp))

	return checkpoint, nil
}

// ListCheckpoints returns a list of available checkpoints
func (cm *CheckpointManager) ListCheckpoints() ([]CheckpointInfo, error) {
	files, err := filepath.Glob(filepath.Join(cm.basePath, "checkpoint_*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoint files: %w", err)
	}

	var checkpoints []CheckpointInfo
	for _, file := range files {
		info, err := cm.getCheckpointInfo(file)
		if err != nil {
			cm.logger.Warn("Failed to get checkpoint info", zap.String("file", file), zap.Error(err))
			continue
		}
		checkpoints = append(checkpoints, info)
	}

	return checkpoints, nil
}

// CheckpointInfo contains basic information about a checkpoint
type CheckpointInfo struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	TestID    string    `json:"test_id"`
	Status    string    `json:"status"`
	Progress  float64   `json:"progress"`
	BandCount int       `json:"band_count"`
	FilePath  string    `json:"file_path"`
	Size      int64     `json:"size_bytes"`
}

// DeleteCheckpoint removes a specific checkpoint
func (cm *CheckpointManager) DeleteCheckpoint(checkpointID string) error {
	checkpointPath := cm.getCheckpointPath(checkpointID)

	if err := os.Remove(checkpointPath); err != nil {
		return fmt.Errorf("failed to delete checkpoint %s: %w", checkpointID, err)
	}

	cm.logger.Info("Checkpoint deleted", zap.String("id", checkpointID))
	return nil
}

// GetCurrentCheckpoint returns the current checkpoint
func (cm *CheckpointManager) GetCurrentCheckpoint() *Checkpoint {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.currentCheckpoint
}

// Stop stops the checkpoint manager
func (cm *CheckpointManager) Stop() {
	cm.cancel()
	cm.logger.Info("Checkpoint manager stopped")
}

// Private methods

func (cm *CheckpointManager) checkpointRoutine() {
	ticker := time.NewTicker(cm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			// Only create automatic checkpoints if we have a current checkpoint to update
			cm.mu.RLock()
			current := cm.currentCheckpoint
			cm.mu.RUnlock()

			if current != nil && current.State.Status == "running" {
				// Update progress and create new checkpoint
				updatedCheckpoint := *current
				updatedCheckpoint.ID = cm.generateCheckpointID()
				updatedCheckpoint.Timestamp = time.Now()

				if err := cm.saveCheckpoint(&updatedCheckpoint); err != nil {
					cm.logger.Error("Failed to create periodic checkpoint", zap.Error(err))
				}
			}
		}
	}
}

func (cm *CheckpointManager) generateCheckpointID() string {
	return fmt.Sprintf("chkpt_%d", time.Now().UnixNano())
}

func (cm *CheckpointManager) generateRecoveryInfo(state TestState, progress BandProgress) RecoveryInfo {
	canRecover := state.Status != "completed" && len(progress.CompletedBands) > 0
	strategy := "continue_from_checkpoint"

	if state.Status == "failed" {
		strategy = "retry_current_band"
		if len(state.Errors) > 3 {
			strategy = "reduce_load_and_retry"
		}
	}

	var actions []RecoveryAction
	if canRecover {
		actions = append(actions, RecoveryAction{
			Type:        "continue",
			Description: "Continue test execution from current band",
			Parameters: map[string]interface{}{
				"start_band": progress.CurrentBand.BandNumber,
			},
		})

		if state.Status == "failed" {
			actions = append(actions, RecoveryAction{
				Type:        "retry_band",
				Description: "Retry the current band with same parameters",
				Parameters: map[string]interface{}{
					"band_number": progress.CurrentBand.BandNumber,
					"max_retries": 3,
				},
			})

			actions = append(actions, RecoveryAction{
				Type:        "reduce_load",
				Description: "Reduce load and retry current band",
				Parameters: map[string]interface{}{
					"band_number":    progress.CurrentBand.BandNumber,
					"reduce_factor":  0.5,
					"retry_duration": "30s",
				},
			})
		}
	}

	return RecoveryInfo{
		CanRecover:       canRecover,
		RecoveryStrategy: strategy,
		LastGoodState:    time.Now(),
		RecoveryActions:  actions,
	}
}

func (cm *CheckpointManager) saveCheckpoint(checkpoint *Checkpoint) error {
	// Ensure directory exists
	if err := os.MkdirAll(cm.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	filePath := cm.getCheckpointPath(checkpoint.ID)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create checkpoint file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(checkpoint); err != nil {
		return fmt.Errorf("failed to encode checkpoint: %w", err)
	}

	return nil
}

func (cm *CheckpointManager) loadCheckpoint(filePath string) (*Checkpoint, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open checkpoint file: %w", err)
	}
	defer file.Close()

	var checkpoint Checkpoint
	decoder := json.NewDecoder(file)

	if err := decoder.Decode(&checkpoint); err != nil {
		return nil, fmt.Errorf("failed to decode checkpoint: %w", err)
	}

	return &checkpoint, nil
}

func (cm *CheckpointManager) getCheckpointPath(checkpointID string) string {
	return filepath.Join(cm.basePath, fmt.Sprintf("checkpoint_%s.json", checkpointID))
}

func (cm *CheckpointManager) findLatestCheckpoint() (string, error) {
	files, err := filepath.Glob(filepath.Join(cm.basePath, "checkpoint_*.json"))
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", nil
	}

	// Find the most recent file
	var latestFile string
	var latestTime time.Time

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestFile = file
		}
	}

	return latestFile, nil
}

func (cm *CheckpointManager) getCheckpointInfo(filePath string) (CheckpointInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return CheckpointInfo{}, err
	}
	defer file.Close()

	var checkpoint Checkpoint
	decoder := json.NewDecoder(file)

	if err := decoder.Decode(&checkpoint); err != nil {
		return CheckpointInfo{}, err
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return CheckpointInfo{}, err
	}

	return CheckpointInfo{
		ID:        checkpoint.ID,
		Timestamp: checkpoint.Timestamp,
		TestID:    checkpoint.TestMetadata.TestID,
		Status:    checkpoint.State.Status,
		Progress:  checkpoint.State.Progress,
		BandCount: len(checkpoint.BandProgress.CompletedBands),
		FilePath:  filePath,
		Size:      info.Size(),
	}, nil
}

func (cm *CheckpointManager) cleanupOldCheckpoints() error {
	checkpoints, err := cm.ListCheckpoints()
	if err != nil {
		return err
	}

	// Sort by timestamp (newest first)
	for i := 0; i < len(checkpoints)-1; i++ {
		for j := i + 1; j < len(checkpoints); j++ {
			if checkpoints[i].Timestamp.Before(checkpoints[j].Timestamp) {
				checkpoints[i], checkpoints[j] = checkpoints[j], checkpoints[i]
			}
		}
	}

	// Delete old checkpoints if we exceed maxFiles
	if len(checkpoints) > cm.maxFiles {
		for i := cm.maxFiles; i < len(checkpoints); i++ {
			if err := os.Remove(checkpoints[i].FilePath); err != nil {
				cm.logger.Warn("Failed to remove old checkpoint",
					zap.String("file", checkpoints[i].FilePath),
					zap.Error(err))
			} else {
				cm.logger.Debug("Removed old checkpoint",
					zap.String("id", checkpoints[i].ID))
			}
		}
	}

	return nil
}

// ProgressTracker helps track and update test progress
type ProgressTracker struct {
	mu            sync.RWMutex
	checkpointMgr *CheckpointManager
	logger        *zap.Logger

	// Current state
	testMetadata   TestMetadata
	bandProgress   BandProgress
	currentMetrics CheckpointMetrics
	config         map[string]interface{}
	state          TestState

	// Progress tracking
	lastUpdate     time.Time
	updateInterval time.Duration
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(checkpointMgr *CheckpointManager, logger *zap.Logger) *ProgressTracker {
	return &ProgressTracker{
		checkpointMgr:  checkpointMgr,
		logger:         logger,
		updateInterval: 10 * time.Second,
	}
}

// InitializeTest initializes tracking for a new test
func (pt *ProgressTracker) InitializeTest(testID string, workloadType string, strategy string, totalBands int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.testMetadata = TestMetadata{
		TestID:       testID,
		StartTime:    time.Now(),
		WorkloadType: workloadType,
		Strategy:     strategy,
		TotalBands:   totalBands,
		CurrentBand:  0,
	}

	pt.state = TestState{
		Status:   "running",
		Phase:    "warmup",
		Progress: 0.0,
		Context:  make(map[string]interface{}),
	}

	pt.bandProgress = BandProgress{
		CompletedBands: make([]BandResult, 0),
		RemainingBands: make([]BandPlan, 0),
	}

	pt.logger.Info("Test tracking initialized",
		zap.String("test_id", testID),
		zap.String("workload", workloadType),
		zap.Int("total_bands", totalBands))
}

// UpdateProgress updates the current test progress
func (pt *ProgressTracker) UpdateProgress(bandNumber int, progress float64, metrics *types.Metrics) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.testMetadata.CurrentBand = bandNumber
	pt.state.Progress = progress

	// Update current band state
	pt.bandProgress.CurrentBand = BandState{
		BandNumber:     bandNumber,
		ElapsedTime:    time.Since(pt.testMetadata.StartTime),
		CompletionPct:  progress * 100,
		CurrentMetrics: metrics,
	}

	// Check if we should create a checkpoint
	if time.Since(pt.lastUpdate) >= pt.updateInterval {
		pt.createProgressCheckpoint()
		pt.lastUpdate = time.Now()
	}
}

// CompleteBand marks a band as completed
func (pt *ProgressTracker) CompleteBand(bandNumber int, result BandResult) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.bandProgress.CompletedBands = append(pt.bandProgress.CompletedBands, result)

	// Update overall progress
	progress := float64(len(pt.bandProgress.CompletedBands)) / float64(pt.testMetadata.TotalBands)
	pt.state.Progress = progress

	pt.logger.Info("Band completed",
		zap.Int("band", bandNumber),
		zap.Float64("progress", progress*100),
		zap.Bool("successful", result.Successful))

	// Create checkpoint for completed band
	pt.createProgressCheckpoint()
}

// HandleError records an error and determines recovery action
func (pt *ProgressTracker) HandleError(err error, component string, recoverable bool) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	execError := ExecutionError{
		Timestamp:   time.Now(),
		Type:        "execution_error",
		Message:     err.Error(),
		Component:   component,
		Recoverable: recoverable,
	}

	pt.state.Errors = append(pt.state.Errors, execError)

	if !recoverable {
		pt.state.Status = "failed"
	}

	pt.logger.Error("Execution error recorded",
		zap.String("component", component),
		zap.Bool("recoverable", recoverable),
		zap.Error(err))

	// Create checkpoint after error
	pt.createProgressCheckpoint()
}

// CompleteTest marks the test as completed
func (pt *ProgressTracker) CompleteTest() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.state.Status = "completed"
	pt.state.Progress = 1.0
	pt.state.Phase = "analysis"

	pt.logger.Info("Test completed",
		zap.String("test_id", pt.testMetadata.TestID),
		zap.Int("completed_bands", len(pt.bandProgress.CompletedBands)),
		zap.Duration("total_duration", time.Since(pt.testMetadata.StartTime)))

	// Create final checkpoint
	pt.createProgressCheckpoint()
}

// GetCurrentProgress returns the current progress state
func (pt *ProgressTracker) GetCurrentProgress() (float64, TestState) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.state.Progress, pt.state
}

// private method to create checkpoints
func (pt *ProgressTracker) createProgressCheckpoint() {
	if pt.checkpointMgr != nil {
		err := pt.checkpointMgr.CreateCheckpoint(
			pt.testMetadata,
			pt.bandProgress,
			pt.currentMetrics,
			pt.config,
			pt.state,
		)
		if err != nil {
			pt.logger.Error("Failed to create progress checkpoint", zap.Error(err))
		}
	}
}
