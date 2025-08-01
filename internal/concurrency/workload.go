// Package concurrency provides adaptive workload management
package concurrency

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// WorkloadManager manages adaptive workload distribution
type WorkloadManager struct {
	mu           sync.RWMutex
	logger       *zap.Logger
	backpressure *BackpressureController

	// Configuration
	maxConcurrency    int64
	adaptiveScaling   bool
	scalingAlgorithm  string
	targetUtilization float64

	// Workload state
	activeJobs    int64
	completedJobs int64
	failedJobs    int64
	totalLatency  int64

	// Adaptive parameters
	currentConcurrency int64
	lastAdjustment     time.Time
	adjustmentHistory  []AdjustmentEvent

	// Job queues
	highPriorityQueue   chan Job
	normalPriorityQueue chan Job
	lowPriorityQueue    chan Job

	// Worker management
	workers    []*Worker
	workerPool sync.Pool

	// Metrics
	metrics *WorkloadMetrics

	// Lifecycle
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
}

// Job represents a unit of work
type Job struct {
	ID          string
	Priority    Priority
	Type        string
	Payload     interface{}
	Context     context.Context
	CreatedAt   time.Time
	StartedAt   time.Time
	CompletedAt time.Time
	Duration    time.Duration
	Retries     int
	MaxRetries  int
	OnComplete  func(result JobResult)
	OnError     func(err error)
}

// Priority defines job priority levels
type Priority int

const (
	LowPriority Priority = iota
	NormalPriority
	HighPriority
)

// JobResult contains the result of job execution
type JobResult struct {
	JobID    string
	Success  bool
	Result   interface{}
	Error    error
	Duration time.Duration
	Retries  int
}

// Worker represents a worker goroutine
type Worker struct {
	ID        string
	manager   *WorkloadManager
	jobChan   chan Job
	active    bool
	startTime time.Time
	stats     WorkerStats
}

// WorkerStats tracks worker performance
type WorkerStats struct {
	JobsProcessed int64
	JobsFailed    int64
	TotalDuration time.Duration
	LastActivity  time.Time
}

// WorkloadMetrics tracks workload management metrics
type WorkloadMetrics struct {
	// Job metrics
	TotalJobs     int64 `json:"total_jobs"`
	ActiveJobs    int64 `json:"active_jobs"`
	CompletedJobs int64 `json:"completed_jobs"`
	FailedJobs    int64 `json:"failed_jobs"`
	RetryJobs     int64 `json:"retry_jobs"`

	// Queue metrics
	HighPriorityQueueSize   int64 `json:"high_priority_queue_size"`
	NormalPriorityQueueSize int64 `json:"normal_priority_queue_size"`
	LowPriorityQueueSize    int64 `json:"low_priority_queue_size"`

	// Performance metrics
	AverageJobDuration  time.Duration `json:"average_job_duration"`
	ThroughputPerSecond float64       `json:"throughput_per_second"`
	WorkerUtilization   float64       `json:"worker_utilization"`

	// Concurrency metrics
	CurrentConcurrency int64 `json:"current_concurrency"`
	PeakConcurrency    int64 `json:"peak_concurrency"`
	Adjustments        int64 `json:"adjustments"`

	// System metrics
	StartTime  time.Time `json:"start_time"`
	LastUpdate time.Time `json:"last_update"`
}

// AdjustmentEvent records concurrency adjustments
type AdjustmentEvent struct {
	Timestamp      time.Time `json:"timestamp"`
	OldConcurrency int64     `json:"old_concurrency"`
	NewConcurrency int64     `json:"new_concurrency"`
	Reason         string    `json:"reason"`
	Utilization    float64   `json:"utilization"`
	QueueDepth     int64     `json:"queue_depth"`
}

// WorkloadConfig contains workload manager configuration
type WorkloadConfig struct {
	MaxConcurrency       int64
	AdaptiveScaling      bool
	ScalingAlgorithm     string // "linear", "exponential", "pid"
	TargetUtilization    float64
	HighPriorityBuffer   int
	NormalPriorityBuffer int
	LowPriorityBuffer    int
	AdjustmentInterval   time.Duration
}

// NewWorkloadManager creates a new workload manager
func NewWorkloadManager(
	config WorkloadConfig,
	backpressure *BackpressureController,
	logger *zap.Logger,
) *WorkloadManager {

	ctx, cancel := context.WithCancel(context.Background())

	wm := &WorkloadManager{
		logger:              logger,
		backpressure:        backpressure,
		maxConcurrency:      config.MaxConcurrency,
		adaptiveScaling:     config.AdaptiveScaling,
		scalingAlgorithm:    config.ScalingAlgorithm,
		targetUtilization:   config.TargetUtilization,
		currentConcurrency:  config.MaxConcurrency / 2, // Start at 50%
		adjustmentHistory:   make([]AdjustmentEvent, 0),
		highPriorityQueue:   make(chan Job, config.HighPriorityBuffer),
		normalPriorityQueue: make(chan Job, config.NormalPriorityBuffer),
		lowPriorityQueue:    make(chan Job, config.LowPriorityBuffer),
		workers:             make([]*Worker, 0),
		metrics:             &WorkloadMetrics{StartTime: time.Now()},
		ctx:                 ctx,
		cancel:              cancel,
	}

	// Set defaults
	if wm.maxConcurrency <= 0 {
		wm.maxConcurrency = 100
	}
	if wm.targetUtilization <= 0 {
		wm.targetUtilization = 0.8
	}
	if wm.scalingAlgorithm == "" {
		wm.scalingAlgorithm = "linear"
	}

	// Initialize worker pool
	wm.workerPool.New = func() interface{} {
		return &Worker{
			manager: wm,
			jobChan: make(chan Job, 1),
		}
	}

	logger.Info("Workload manager created",
		zap.Int64("max_concurrency", wm.maxConcurrency),
		zap.Bool("adaptive_scaling", wm.adaptiveScaling),
		zap.String("scaling_algorithm", wm.scalingAlgorithm))

	return wm
}

// Start starts the workload manager
func (wm *WorkloadManager) Start() error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if wm.started {
		return fmt.Errorf("workload manager already started")
	}

	// Start initial workers
	for i := int64(0); i < wm.currentConcurrency; i++ {
		worker := wm.createWorker()
		wm.workers = append(wm.workers, worker)
		go worker.run(wm.ctx)
	}

	// Start job dispatcher
	go wm.dispatchJobs()

	// Start adaptive scaling routine
	if wm.adaptiveScaling {
		go wm.adaptiveConcurrencyRoutine()
	}

	wm.started = true
	wm.logger.Info("Workload manager started",
		zap.Int64("initial_concurrency", wm.currentConcurrency))

	return nil
}

// SubmitJob submits a job for execution
func (wm *WorkloadManager) SubmitJob(job Job) error {
	if !wm.started {
		return fmt.Errorf("workload manager not started")
	}

	job.CreatedAt = time.Now()
	if job.Context == nil {
		job.Context = context.Background()
	}
	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}

	// Select appropriate queue based on priority
	var queue chan Job
	switch job.Priority {
	case HighPriority:
		queue = wm.highPriorityQueue
		atomic.AddInt64(&wm.metrics.HighPriorityQueueSize, 1)
	case NormalPriority:
		queue = wm.normalPriorityQueue
		atomic.AddInt64(&wm.metrics.NormalPriorityQueueSize, 1)
	case LowPriority:
		queue = wm.lowPriorityQueue
		atomic.AddInt64(&wm.metrics.LowPriorityQueueSize, 1)
	default:
		queue = wm.normalPriorityQueue
		atomic.AddInt64(&wm.metrics.NormalPriorityQueueSize, 1)
	}

	// Try to enqueue with timeout
	select {
	case queue <- job:
		atomic.AddInt64(&wm.metrics.TotalJobs, 1)
		wm.logger.Debug("Job submitted", zap.String("job_id", job.ID), zap.Int("priority", int(job.Priority)))
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("job queue full, job %s rejected", job.ID)
	}
}

// GetMetrics returns current workload metrics
func (wm *WorkloadManager) GetMetrics() *WorkloadMetrics {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	wm.metrics.ActiveJobs = atomic.LoadInt64(&wm.activeJobs)
	wm.metrics.CompletedJobs = atomic.LoadInt64(&wm.completedJobs)
	wm.metrics.FailedJobs = atomic.LoadInt64(&wm.failedJobs)
	wm.metrics.CurrentConcurrency = wm.currentConcurrency
	wm.metrics.LastUpdate = time.Now()

	// Calculate utilization
	if len(wm.workers) > 0 {
		activeWorkers := int64(0)
		for _, worker := range wm.workers {
			if worker.active {
				activeWorkers++
			}
		}
		wm.metrics.WorkerUtilization = float64(activeWorkers) / float64(len(wm.workers))
	}

	// Calculate throughput
	duration := time.Since(wm.metrics.StartTime)
	if duration > 0 {
		wm.metrics.ThroughputPerSecond = float64(wm.metrics.CompletedJobs) / duration.Seconds()
	}

	return wm.metrics
}

// GetAdjustmentHistory returns the concurrency adjustment history
func (wm *WorkloadManager) GetAdjustmentHistory() []AdjustmentEvent {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	history := make([]AdjustmentEvent, len(wm.adjustmentHistory))
	copy(history, wm.adjustmentHistory)
	return history
}

// Stop stops the workload manager
func (wm *WorkloadManager) Stop() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if !wm.started {
		return
	}

	wm.cancel()

	// Close job queues
	close(wm.highPriorityQueue)
	close(wm.normalPriorityQueue)
	close(wm.lowPriorityQueue)

	// Wait for workers to finish
	for _, worker := range wm.workers {
		close(worker.jobChan)
	}

	wm.started = false
	wm.logger.Info("Workload manager stopped")
}

// Private methods

func (wm *WorkloadManager) createWorker() *Worker {
	worker := wm.workerPool.Get().(*Worker)
	worker.ID = fmt.Sprintf("worker_%d", time.Now().UnixNano())
	worker.startTime = time.Now()
	worker.active = false
	worker.stats = WorkerStats{LastActivity: time.Now()}

	return worker
}

func (wm *WorkloadManager) dispatchJobs() {
	for {
		select {
		case <-wm.ctx.Done():
			return
		case job := <-wm.highPriorityQueue:
			wm.dispatchJob(job)
			atomic.AddInt64(&wm.metrics.HighPriorityQueueSize, -1)
		case job := <-wm.normalPriorityQueue:
			wm.dispatchJob(job)
			atomic.AddInt64(&wm.metrics.NormalPriorityQueueSize, -1)
		case job := <-wm.lowPriorityQueue:
			wm.dispatchJob(job)
			atomic.AddInt64(&wm.metrics.LowPriorityQueueSize, -1)
		}
	}
}

func (wm *WorkloadManager) dispatchJob(job Job) {
	// Find an available worker
	wm.mu.RLock()
	var availableWorker *Worker
	for _, worker := range wm.workers {
		if !worker.active {
			availableWorker = worker
			break
		}
	}
	wm.mu.RUnlock()

	if availableWorker != nil {
		select {
		case availableWorker.jobChan <- job:
			// Job dispatched successfully
		default:
			// Worker is busy, retry or queue
			wm.retryJob(job)
		}
	} else {
		// No available workers, retry or queue
		wm.retryJob(job)
	}
}

func (wm *WorkloadManager) retryJob(job Job) {
	job.Retries++
	if job.Retries <= job.MaxRetries {
		// Put job back in queue
		go func() {
			time.Sleep(time.Duration(job.Retries) * time.Second) // Exponential backoff
			_ = wm.SubmitJob(job)                                // Ignore error on retry submission
		}()
		atomic.AddInt64(&wm.metrics.RetryJobs, 1)
	} else {
		// Max retries exceeded
		atomic.AddInt64(&wm.failedJobs, 1)
		if job.OnError != nil {
			job.OnError(fmt.Errorf("max retries exceeded"))
		}
	}
}

func (wm *WorkloadManager) adaptiveConcurrencyRoutine() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wm.ctx.Done():
			return
		case <-ticker.C:
			wm.adjustConcurrency()
		}
	}
}

func (wm *WorkloadManager) adjustConcurrency() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Calculate current utilization
	activeWorkers := int64(0)
	for _, worker := range wm.workers {
		if worker.active {
			activeWorkers++
		}
	}

	utilization := float64(activeWorkers) / float64(len(wm.workers))
	queueDepth := atomic.LoadInt64(&wm.metrics.HighPriorityQueueSize) +
		atomic.LoadInt64(&wm.metrics.NormalPriorityQueueSize) +
		atomic.LoadInt64(&wm.metrics.LowPriorityQueueSize)

	oldConcurrency := wm.currentConcurrency
	var newConcurrency int64
	var reason string

	// Determine if adjustment is needed
	if utilization > wm.targetUtilization && queueDepth > 0 {
		// Scale up
		switch wm.scalingAlgorithm {
		case "linear":
			newConcurrency = wm.currentConcurrency + 1
		case "exponential":
			newConcurrency = int64(float64(wm.currentConcurrency) * 1.2)
		default:
			newConcurrency = wm.currentConcurrency + 1
		}
		reason = "high_utilization"
	} else if utilization < wm.targetUtilization*0.5 && queueDepth == 0 {
		// Scale down
		switch wm.scalingAlgorithm {
		case "linear":
			newConcurrency = wm.currentConcurrency - 1
		case "exponential":
			newConcurrency = int64(float64(wm.currentConcurrency) * 0.8)
		default:
			newConcurrency = wm.currentConcurrency - 1
		}
		reason = "low_utilization"
	} else {
		// No adjustment needed
		return
	}

	// Apply limits
	if newConcurrency > wm.maxConcurrency {
		newConcurrency = wm.maxConcurrency
	}
	if newConcurrency < 1 {
		newConcurrency = 1
	}

	// If no change, return
	if newConcurrency == oldConcurrency {
		return
	}

	// Apply the change
	if newConcurrency > oldConcurrency {
		// Add workers
		for i := oldConcurrency; i < newConcurrency; i++ {
			worker := wm.createWorker()
			wm.workers = append(wm.workers, worker)
			go worker.run(wm.ctx)
		}
	} else {
		// Remove workers (gracefully)
		workersToRemove := oldConcurrency - newConcurrency
		for i := int64(0); i < workersToRemove && len(wm.workers) > 0; i++ {
			worker := wm.workers[len(wm.workers)-1]
			wm.workers = wm.workers[:len(wm.workers)-1]
			close(worker.jobChan)
		}
	}

	wm.currentConcurrency = newConcurrency
	wm.lastAdjustment = time.Now()

	// Record adjustment
	event := AdjustmentEvent{
		Timestamp:      wm.lastAdjustment,
		OldConcurrency: oldConcurrency,
		NewConcurrency: newConcurrency,
		Reason:         reason,
		Utilization:    utilization,
		QueueDepth:     queueDepth,
	}
	wm.adjustmentHistory = append(wm.adjustmentHistory, event)
	atomic.AddInt64(&wm.metrics.Adjustments, 1)

	wm.logger.Info("Concurrency adjusted",
		zap.Int64("from", oldConcurrency),
		zap.Int64("to", newConcurrency),
		zap.String("reason", reason),
		zap.Float64("utilization", utilization))
}

// Worker methods

func (w *Worker) run(ctx context.Context) {
	defer func() {
		w.manager.workerPool.Put(w)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-w.jobChan:
			if !ok {
				return
			}
			w.processJob(job)
		}
	}
}

func (w *Worker) processJob(job Job) {
	w.active = true
	atomic.AddInt64(&w.manager.activeJobs, 1)

	defer func() {
		w.active = false
		atomic.AddInt64(&w.manager.activeJobs, -1)
		w.stats.LastActivity = time.Now()
	}()

	start := time.Now()
	job.StartedAt = start

	// Simulate job execution
	// In real implementation, this would call the actual job handler
	result := w.executeJob(job)

	job.CompletedAt = time.Now()
	job.Duration = job.CompletedAt.Sub(start)

	// Update statistics
	atomic.AddInt64(&w.stats.JobsProcessed, 1)
	w.stats.TotalDuration += job.Duration
	atomic.AddInt64(&w.manager.totalLatency, int64(job.Duration))

	if result.Success {
		atomic.AddInt64(&w.manager.completedJobs, 1)
		if job.OnComplete != nil {
			job.OnComplete(result)
		}
	} else {
		atomic.AddInt64(&w.stats.JobsFailed, 1)
		atomic.AddInt64(&w.manager.failedJobs, 1)
		if job.OnError != nil {
			job.OnError(result.Error)
		}
	}

	w.manager.logger.Debug("Job completed",
		zap.String("job_id", job.ID),
		zap.String("worker_id", w.ID),
		zap.Duration("duration", job.Duration),
		zap.Bool("success", result.Success))
}

func (w *Worker) executeJob(job Job) JobResult {
	// Placeholder job execution
	// In real implementation, this would dispatch to job-specific handlers

	// Simulate work
	time.Sleep(time.Millisecond * 10)

	return JobResult{
		JobID:    job.ID,
		Success:  true,
		Duration: time.Since(job.StartedAt),
		Retries:  job.Retries,
	}
}
