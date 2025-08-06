// Package scheduler provides task scheduling and execution for StormDB v2
// Handles test execution orchestration and worker pool management
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/elchinoo/stormdb/core"
)

// Manager implements the SchedulerManager interface
type Manager struct {
	config        *core.SchedulerConfig
	storage       core.StorageManager
	pluginManager core.PluginManager
	logger        core.Logger

	// Worker pool
	workers        []chan core.Task
	taskQueue      chan core.Task
	scheduledTasks map[string]*scheduledTask

	// State management
	isRunning bool
	stopChan  chan struct{}
	wg        sync.WaitGroup
	mu        sync.RWMutex
}

// scheduledTask represents a task that runs on an interval
type scheduledTask struct {
	id       string
	task     core.Task
	interval time.Duration
	ticker   *time.Ticker
	stopChan chan struct{}
}

// TestExecutionTask implements the Task interface for test execution
type TestExecutionTask struct {
	id         string
	pluginName string
	config     map[string]interface{}
	storage    core.StorageManager
	plugin     core.PluginManager
	logger     core.Logger
}

// NewManager creates a new scheduler manager
func NewManager(config *core.SchedulerConfig, storage core.StorageManager, pluginManager core.PluginManager, logger core.Logger) *Manager {
	return &Manager{
		config:         config,
		storage:        storage,
		pluginManager:  pluginManager,
		logger:         logger.WithFields(core.Field{Key: "component", Value: "scheduler"}),
		scheduledTasks: make(map[string]*scheduledTask),
		stopChan:       make(chan struct{}),
	}
}

// Start initializes and starts the scheduler
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	if !m.config.Enabled {
		m.logger.Info("scheduler is disabled")
		return nil
	}

	m.logger.Info("starting scheduler",
		core.Field{Key: "worker_pool_size", Value: m.config.WorkerPoolSize},
	)

	// Initialize task queue
	m.taskQueue = make(chan core.Task, m.config.WorkerPoolSize*2)

	// Initialize worker pool
	m.workers = make([]chan core.Task, m.config.WorkerPoolSize)
	for i := 0; i < m.config.WorkerPoolSize; i++ {
		workerChan := make(chan core.Task)
		m.workers[i] = workerChan

		m.wg.Add(1)
		go m.worker(i, workerChan)
	}

	// Start task dispatcher
	m.wg.Add(1)
	go m.dispatcher()

	m.isRunning = true

	m.logger.Info("scheduler started successfully")
	return nil
}

// Stop shuts down the scheduler
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return nil
	}

	m.logger.Info("stopping scheduler")

	// Stop all scheduled tasks
	for _, task := range m.scheduledTasks {
		task.ticker.Stop()
		close(task.stopChan)
	}

	// Signal shutdown
	close(m.stopChan)

	// Close task queue
	close(m.taskQueue)

	// Close worker channels
	for _, workerChan := range m.workers {
		close(workerChan)
	}

	m.isRunning = false

	// Wait for all goroutines to finish
	go func() {
		m.wg.Wait()
		m.logger.Info("scheduler stopped")
	}()

	return nil
}

// IsRunning returns whether the scheduler is currently running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// GetStatus returns current scheduler status
func (m *Manager) GetStatus() core.SchedulerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	queueLength := 0
	if m.taskQueue != nil {
		queueLength = len(m.taskQueue)
	}

	return core.SchedulerStatus{
		Running:            m.isRunning,
		WorkerCount:        len(m.workers),
		QueueLength:        queueLength,
		ScheduledTaskCount: len(m.scheduledTasks),
	}
}

// SubmitTask submits a task for immediate execution
func (m *Manager) SubmitTask(task core.Task) error {
	if !m.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	select {
	case m.taskQueue <- task:
		m.logger.Debug("task submitted",
			core.Field{Key: "task_id", Value: task.ID()},
			core.Field{Key: "task_type", Value: task.Type()},
		)
		return nil
	default:
		return fmt.Errorf("task queue is full")
	}
}

// ScheduleTask schedules a recurring task
func (m *Manager) ScheduleTask(taskID string, task core.Task, interval time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	// Check if task is already scheduled
	if _, exists := m.scheduledTasks[taskID]; exists {
		return fmt.Errorf("task %s is already scheduled", taskID)
	}

	// Create scheduled task
	scheduledTask := &scheduledTask{
		id:       taskID,
		task:     task,
		interval: interval,
		ticker:   time.NewTicker(interval),
		stopChan: make(chan struct{}),
	}

	m.scheduledTasks[taskID] = scheduledTask

	// Start the scheduler goroutine
	m.wg.Add(1)
	go m.runScheduledTask(scheduledTask)

	m.logger.Info("task scheduled",
		core.Field{Key: "task_id", Value: taskID},
		core.Field{Key: "interval", Value: interval.String()},
	)

	return nil
}

// CancelTask cancels a scheduled task
func (m *Manager) CancelTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	scheduledTask, exists := m.scheduledTasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Stop the task
	scheduledTask.ticker.Stop()
	close(scheduledTask.stopChan)

	// Remove from map
	delete(m.scheduledTasks, taskID)

	m.logger.Info("task cancelled", core.Field{Key: "task_id", Value: taskID})
	return nil
}

// ScheduleTest schedules a test execution
func (m *Manager) ScheduleTest(ctx context.Context, pluginName string, config map[string]interface{}) (int64, error) {
	// Get plugin
	plugin, err := m.pluginManager.GetPlugin(pluginName)
	if err != nil {
		return 0, fmt.Errorf("plugin %s not found: %w", pluginName, err)
	}

	// Create test run record
	testRun := &core.TestRun{
		Name:        fmt.Sprintf("Test run %s", time.Now().Format("2006-01-02 15:04:05")),
		Description: fmt.Sprintf("Scheduled test execution for plugin %s", pluginName),
		Status:      core.StatusPending,
		Config:      config,
		PluginVer:   plugin.Metadata().Version,
		Host:        "localhost", // Default, should come from config
		Port:        5432,        // Default, should come from config
		DBName:      "test",      // Default, should come from config
	}

	runID, err := m.storage.CreateTestRun(ctx, testRun)
	if err != nil {
		return 0, fmt.Errorf("failed to create test run: %w", err)
	}

	// Create execution task
	task := &TestExecutionTask{
		id:         fmt.Sprintf("test-%d", runID),
		pluginName: pluginName,
		config:     config,
		storage:    m.storage,
		plugin:     m.pluginManager,
		logger:     m.logger,
	}

	// Submit for execution
	if err := m.SubmitTask(task); err != nil {
		return 0, fmt.Errorf("failed to submit test task: %w", err)
	}

	m.logger.Info("test scheduled",
		core.Field{Key: "test_run_id", Value: runID},
		core.Field{Key: "plugin", Value: pluginName},
	)

	return runID, nil
}

// CancelTest cancels a running test
func (m *Manager) CancelTest(ctx context.Context, runID int64) error {
	// Update test run status to aborted
	err := m.storage.UpdateTestRunStatus(ctx, runID, core.StatusAborted)
	if err != nil {
		return fmt.Errorf("failed to update test run status: %w", err)
	}

	m.logger.Info("test cancelled", core.Field{Key: "test_run_id", Value: runID})
	return nil
}

// GetRunStatus gets the current status of a test run
func (m *Manager) GetRunStatus(ctx context.Context, runID int64) (core.ServiceStatus, error) {
	testRun, err := m.storage.GetTestRun(ctx, runID)
	if err != nil {
		return "", fmt.Errorf("failed to get test run: %w", err)
	}

	return testRun.Status, nil
}

// ListActiveRuns lists all currently active test runs
func (m *Manager) ListActiveRuns(ctx context.Context) ([]core.TestRun, error) {
	// This is a simplified implementation
	// In a full implementation, you'd query for running/pending tests
	runs, err := m.storage.ListTestRuns(ctx, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list test runs: %w", err)
	}

	var activeRuns []core.TestRun
	for _, run := range runs {
		if run.Status == core.StatusRunning || run.Status == core.StatusPending {
			activeRuns = append(activeRuns, run)
		}
	}

	return activeRuns, nil
}

// worker is a worker goroutine that processes tasks
func (m *Manager) worker(id int, tasks chan core.Task) {
	defer m.wg.Done()

	workerLogger := m.logger.WithFields(core.Field{Key: "worker_id", Value: id})
	workerLogger.Debug("worker started")

	for {
		select {
		case task, ok := <-tasks:
			if !ok {
				workerLogger.Debug("worker stopping")
				return
			}

			workerLogger.Debug("executing task",
				core.Field{Key: "task_id", Value: task.ID()},
				core.Field{Key: "task_type", Value: task.Type()},
			)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			if err := task.Execute(ctx); err != nil {
				workerLogger.Error("task execution failed",
					core.Field{Key: "task_id", Value: task.ID()},
					core.Field{Key: "error", Value: err.Error()},
				)
			} else {
				workerLogger.Debug("task completed successfully",
					core.Field{Key: "task_id", Value: task.ID()},
				)
			}
			cancel()

		case <-m.stopChan:
			workerLogger.Debug("worker stopping due to shutdown signal")
			return
		}
	}
}

// dispatcher distributes tasks to workers
func (m *Manager) dispatcher() {
	defer m.wg.Done()

	m.logger.Debug("dispatcher started")

	workerIndex := 0
	for {
		select {
		case task, ok := <-m.taskQueue:
			if !ok {
				m.logger.Debug("dispatcher stopping")
				return
			}

			// Round-robin assignment to workers
			select {
			case m.workers[workerIndex] <- task:
				workerIndex = (workerIndex + 1) % len(m.workers)
			case <-m.stopChan:
				m.logger.Debug("dispatcher stopping due to shutdown signal")
				return
			}

		case <-m.stopChan:
			m.logger.Debug("dispatcher stopping due to shutdown signal")
			return
		}
	}
}

// runScheduledTask runs a scheduled task on its interval
func (m *Manager) runScheduledTask(task *scheduledTask) {
	defer m.wg.Done()

	for {
		select {
		case <-task.ticker.C:
			// Submit the task for execution
			select {
			case m.taskQueue <- task.task:
				m.logger.Debug("scheduled task submitted",
					core.Field{Key: "task_id", Value: task.id},
				)
			default:
				m.logger.Warn("task queue full, skipping scheduled task",
					core.Field{Key: "task_id", Value: task.id},
				)
			}

		case <-task.stopChan:
			m.logger.Debug("scheduled task stopped",
				core.Field{Key: "task_id", Value: task.id},
			)
			return

		case <-m.stopChan:
			m.logger.Debug("scheduled task stopping due to shutdown",
				core.Field{Key: "task_id", Value: task.id},
			)
			return
		}
	}
}

// TestExecutionTask methods

// ID returns the task ID
func (t *TestExecutionTask) ID() string {
	return t.id
}

// Type returns the task type
func (t *TestExecutionTask) Type() string {
	return "test_execution"
}

// Execute executes the test
func (t *TestExecutionTask) Execute(ctx context.Context) error {
	t.logger.Info("executing test",
		core.Field{Key: "task_id", Value: t.id},
		core.Field{Key: "plugin", Value: t.pluginName},
	)

	// Get plugin
	plugin, err := t.plugin.GetPlugin(t.pluginName)
	if err != nil {
		return fmt.Errorf("failed to get plugin: %w", err)
	}

	// Execute plugin
	if err := plugin.Execute(ctx, t.config); err != nil {
		return fmt.Errorf("plugin execution failed: %w", err)
	}

	t.logger.Info("test execution completed",
		core.Field{Key: "task_id", Value: t.id},
	)

	return nil
}
