package workerpool

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elchinoo/stormdb/internal/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Job represents a unit of work to be processed by the worker pool
type Job interface {
	Execute(ctx context.Context) Result
	ID() string
	Priority() int
}

// Result represents the result of job execution
type Result interface {
	JobID() string
	Error() error
	Duration() time.Duration
	Metrics() map[string]interface{}
}

// WorkerPool manages a pool of workers for processing jobs
type WorkerPool struct {
	workers     int
	jobs        chan Job
	results     chan Result
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	logger      logging.StormDBLogger
	
	// Metrics
	jobsProcessed   int64
	jobsSuccessful  int64
	jobsFailed      int64
	totalDuration   int64 // nanoseconds
	
	// Configuration
	bufferSize      int
	shutdownTimeout time.Duration
	
	// Status
	running bool
	mutex   sync.RWMutex
}

// WorkerPoolConfig configures the worker pool
type WorkerPoolConfig struct {
	Workers         int
	BufferSize      int
	ShutdownTimeout time.Duration
	Logger          logging.StormDBLogger
}

// NewWorkerPool creates a new worker pool with the specified configuration
func NewWorkerPool(config WorkerPoolConfig) *WorkerPool {
	if config.Workers <= 0 {
		config.Workers = 4
	}
	if config.BufferSize <= 0 {
		config.BufferSize = config.Workers * 2
	}
	if config.ShutdownTimeout <= 0 {
		config.ShutdownTimeout = 30 * time.Second
	}
	if config.Logger == nil {
		config.Logger = logging.NewDefaultLogger()
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	return &WorkerPool{
		workers:         config.Workers,
		jobs:            make(chan Job, config.BufferSize),
		results:         make(chan Result, config.BufferSize),
		ctx:             ctx,
		cancel:          cancel,
		logger:          config.Logger,
		bufferSize:      config.BufferSize,
		shutdownTimeout: config.ShutdownTimeout,
	}
}

// Start begins processing jobs with the configured number of workers
func (wp *WorkerPool) Start() error {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	
	if wp.running {
		return errors.New("worker pool is already running")
	}
	
	wp.logger.Info("Starting worker pool",
		zap.Int("workers", wp.workers),
		zap.Int("buffer_size", wp.bufferSize),
	)
	
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
	
	wp.running = true
	
	wp.logger.Info("Worker pool started successfully",
		zap.Int("workers", wp.workers),
	)
	
	return nil
}

// Submit adds a job to the work queue
func (wp *WorkerPool) Submit(job Job) error {
	wp.mutex.RLock()
	running := wp.running
	wp.mutex.RUnlock()
	
	if !running {
		return errors.New("worker pool is not running")
	}
	
	select {
	case wp.jobs <- job:
		return nil
	case <-wp.ctx.Done():
		return errors.New("worker pool is shutting down")
	default:
		return errors.New("job queue is full")
	}
}

// Results returns the results channel for reading job results
func (wp *WorkerPool) Results() <-chan Result {
	return wp.results
}

// Shutdown gracefully shuts down the worker pool
func (wp *WorkerPool) Shutdown() error {
	wp.mutex.Lock()
	if !wp.running {
		wp.mutex.Unlock()
		return nil
	}
	wp.running = false
	wp.mutex.Unlock()
	
	wp.logger.Info("Shutting down worker pool",
		zap.Duration("timeout", wp.shutdownTimeout),
	)
	
	// Close jobs channel to signal workers to stop
	close(wp.jobs)
	
	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		wp.logger.Info("Worker pool shutdown completed successfully")
	case <-time.After(wp.shutdownTimeout):
		wp.logger.Warn("Worker pool shutdown timeout exceeded, forcing shutdown")
		wp.cancel()
		// Wait a bit more for forced shutdown
		select {
		case <-done:
			wp.logger.Info("Forced shutdown completed")
		case <-time.After(5 * time.Second):
			wp.logger.Error("Failed to shutdown worker pool within timeout", nil)
			return errors.New("shutdown timeout exceeded")
		}
	}
	
	// Close results channel
	close(wp.results)
	
	return nil
}

// Stats returns current worker pool statistics
func (wp *WorkerPool) Stats() WorkerPoolStats {
	return WorkerPoolStats{
		Workers:        wp.workers,
		JobsProcessed:  atomic.LoadInt64(&wp.jobsProcessed),
		JobsSuccessful: atomic.LoadInt64(&wp.jobsSuccessful),
		JobsFailed:     atomic.LoadInt64(&wp.jobsFailed),
		AverageDuration: time.Duration(atomic.LoadInt64(&wp.totalDuration) / 
			max(atomic.LoadInt64(&wp.jobsProcessed), 1)),
		Running:        wp.isRunning(),
	}
}

// WorkerPoolStats contains worker pool statistics
type WorkerPoolStats struct {
	Workers         int
	JobsProcessed   int64
	JobsSuccessful  int64
	JobsFailed      int64
	AverageDuration time.Duration
	Running         bool
}

// worker processes jobs from the job queue
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	
	workerLogger := wp.logger.With(zap.Int("worker_id", id))
	workerLogger.Debug("Worker started")
	
	defer func() {
		if r := recover(); r != nil {
			workerLogger.Error("Worker panicked", 
				fmt.Errorf("panic: %v", r),
				zap.Any("panic_value", r),
			)
		}
		workerLogger.Debug("Worker stopped")
	}()
	
	for {
		select {
		case job, ok := <-wp.jobs:
			if !ok {
				// Jobs channel closed, worker should exit
				return
			}
			
			wp.processJob(workerLogger, job)
			
		case <-wp.ctx.Done():
			// Context cancelled, worker should exit
			return
		}
	}
}

// processJob executes a single job and records the result
func (wp *WorkerPool) processJob(logger logging.StormDBLogger, job Job) {
	start := time.Now()
	
	// Execute job with panic recovery
	var result Result
	func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Job execution panicked",
					fmt.Errorf("panic: %v", r),
					zap.String("job_id", job.ID()),
					zap.Any("panic_value", r),
				)
				result = &panicResult{
					jobID:    job.ID(),
					error:    fmt.Errorf("job panicked: %v", r),
					duration: time.Since(start),
				}
			}
		}()
		
		result = job.Execute(wp.ctx)
	}()
	
	duration := time.Since(start)
	
	// Update metrics
	atomic.AddInt64(&wp.jobsProcessed, 1)
	atomic.AddInt64(&wp.totalDuration, int64(duration))
	
	if result.Error() != nil {
		atomic.AddInt64(&wp.jobsFailed, 1)
		logger.Debug("Job failed",
			zap.String("job_id", job.ID()),
			zap.Error(result.Error()),
			zap.Duration("duration", duration),
		)
	} else {
		atomic.AddInt64(&wp.jobsSuccessful, 1)
		logger.Debug("Job completed successfully",
			zap.String("job_id", job.ID()),
			zap.Duration("duration", duration),
		)
	}
	
	// Send result
	select {
	case wp.results <- result:
		// Result sent successfully
	case <-wp.ctx.Done():
		// Context cancelled, don't block
		logger.Debug("Failed to send result due to context cancellation",
			zap.String("job_id", job.ID()),
		)
	default:
		// Results channel full, log warning
		logger.Warn("Results channel full, dropping result",
			zap.String("job_id", job.ID()),
		)
	}
}

// isRunning returns whether the worker pool is currently running
func (wp *WorkerPool) isRunning() bool {
	wp.mutex.RLock()
	defer wp.mutex.RUnlock()
	return wp.running
}

// panicResult represents a result from a job that panicked
type panicResult struct {
	jobID    string
	error    error
	duration time.Duration
}

func (pr *panicResult) JobID() string {
	return pr.jobID
}

func (pr *panicResult) Error() error {
	return pr.error
}

func (pr *panicResult) Duration() time.Duration {
	return pr.duration
}

func (pr *panicResult) Metrics() map[string]interface{} {
	return map[string]interface{}{
		"panicked": true,
	}
}

// max returns the maximum of two int64 values
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// Priority queue implementation for ordered job processing
type PriorityWorkerPool struct {
	*WorkerPool
	priorityJobs chan Job
}

// NewPriorityWorkerPool creates a worker pool that processes jobs by priority
func NewPriorityWorkerPool(config WorkerPoolConfig) *PriorityWorkerPool {
	base := NewWorkerPool(config)
	return &PriorityWorkerPool{
		WorkerPool:   base,
		priorityJobs: make(chan Job, config.BufferSize),
	}
}

// SubmitPriority submits a job with priority handling
func (pwp *PriorityWorkerPool) SubmitPriority(job Job) error {
	if job.Priority() > 0 {
		select {
		case pwp.priorityJobs <- job:
			return nil
		case <-pwp.ctx.Done():
			return errors.New("worker pool is shutting down")
		default:
			return errors.New("priority job queue is full")
		}
	}
	return pwp.Submit(job)
}

// ResourceMonitor tracks worker pool resource usage
type ResourceMonitor struct {
	pool    *WorkerPool
	metrics chan ResourceMetrics
	stop    chan struct{}
	logger  logging.StormDBLogger
}

// ResourceMetrics contains resource usage information
type ResourceMetrics struct {
	Timestamp       time.Time
	ActiveWorkers   int
	QueueLength     int
	ProcessingRate  float64 // jobs per second
	ErrorRate       float64 // errors per second
	MemoryUsageMB   float64
}

// NewResourceMonitor creates a resource monitor for a worker pool
func NewResourceMonitor(pool *WorkerPool, logger logging.StormDBLogger) *ResourceMonitor {
	return &ResourceMonitor{
		pool:    pool,
		metrics: make(chan ResourceMetrics, 100),
		stop:    make(chan struct{}),
		logger:  logger,
	}
}

// Start begins resource monitoring
func (rm *ResourceMonitor) Start(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		var lastProcessed int64
		var lastFailed int64
		lastTime := time.Now()
		
		for {
			select {
			case <-ticker.C:
				stats := rm.pool.Stats()
				now := time.Now()
				elapsed := now.Sub(lastTime).Seconds()
				
				processingRate := float64(stats.JobsProcessed-lastProcessed) / elapsed
				errorRate := float64(stats.JobsFailed-lastFailed) / elapsed
				
				metrics := ResourceMetrics{
					Timestamp:      now,
					ActiveWorkers:  stats.Workers,
					QueueLength:    len(rm.pool.jobs),
					ProcessingRate: processingRate,
					ErrorRate:      errorRate,
					// MemoryUsageMB would need runtime.ReadMemStats() implementation
				}
				
				select {
				case rm.metrics <- metrics:
				default:
					rm.logger.Warn("Resource metrics channel full, dropping sample")
				}
				
				lastProcessed = stats.JobsProcessed
				lastFailed = stats.JobsFailed
				lastTime = now
				
			case <-rm.stop:
				return
			}
		}
	}()
}

// Stop stops resource monitoring
func (rm *ResourceMonitor) Stop() {
	close(rm.stop)
}

// Metrics returns the metrics channel
func (rm *ResourceMonitor) Metrics() <-chan ResourceMetrics {
	return rm.metrics
}
