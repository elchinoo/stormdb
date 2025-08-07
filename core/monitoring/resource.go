// Package monitoring provides resource usage tracking for StormDB v2
package monitoring

import (
	"runtime"
	"sync"
	"time"

	"github.com/elchinoo/stormdb/core"
)

// ResourceStats represents current resource usage statistics
type ResourceStats struct {
	CPUPercent     float64                `json:"cpu_percent"`
	MemoryUsageMB  float64                `json:"memory_usage_mb"`
	MemoryAllocMB  float64                `json:"memory_alloc_mb"`
	GoroutineCount int                    `json:"goroutine_count"`
	GCRuns         uint32                 `json:"gc_runs"`
	DatabaseConns  int                    `json:"database_connections"`
	ActiveTests    int                    `json:"active_tests"`
	Timestamp      time.Time              `json:"timestamp"`
	PluginStats    map[string]interface{} `json:"plugin_stats,omitempty"`
}

// ResourceMonitor tracks system resource usage
type ResourceMonitor struct {
	logger         core.Logger
	storage        core.StorageManager
	interval       time.Duration
	stopChan       chan struct{}
	mu             sync.RWMutex
	lastStats      ResourceStats
	pluginMonitors map[string]PluginResourceMonitor
	isRunning      bool
}

// PluginResourceMonitor interface for plugin-specific resource monitoring
type PluginResourceMonitor interface {
	GetResourceUsage() map[string]interface{}
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(logger core.Logger, storage core.StorageManager, interval time.Duration) *ResourceMonitor {
	return &ResourceMonitor{
		logger:         logger.WithFields(core.Field{Key: "component", Value: "resource_monitor"}),
		storage:        storage,
		interval:       interval,
		stopChan:       make(chan struct{}),
		pluginMonitors: make(map[string]PluginResourceMonitor),
	}
}

// Start begins resource monitoring
func (rm *ResourceMonitor) Start() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.isRunning {
		return nil
	}

	rm.isRunning = true
	go rm.monitoringLoop()
	rm.logger.Info("resource monitoring started", core.Field{Key: "interval", Value: rm.interval})
	return nil
}

// Stop stops resource monitoring
func (rm *ResourceMonitor) Stop() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.isRunning {
		return nil
	}

	close(rm.stopChan)
	rm.isRunning = false
	rm.logger.Info("resource monitoring stopped")
	return nil
}

// RegisterPlugin registers a plugin for resource monitoring
func (rm *ResourceMonitor) RegisterPlugin(pluginName string, monitor PluginResourceMonitor) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.pluginMonitors[pluginName] = monitor
	rm.logger.Info("plugin registered for resource monitoring", core.Field{Key: "plugin", Value: pluginName})
}

// UnregisterPlugin unregisters a plugin from resource monitoring
func (rm *ResourceMonitor) UnregisterPlugin(pluginName string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.pluginMonitors, pluginName)
	rm.logger.Info("plugin unregistered from resource monitoring", core.Field{Key: "plugin", Value: pluginName})
}

// GetCurrentStats returns the most recent resource statistics
func (rm *ResourceMonitor) GetCurrentStats() ResourceStats {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.lastStats
}

// monitoringLoop runs the resource monitoring loop
func (rm *ResourceMonitor) monitoringLoop() {
	ticker := time.NewTicker(rm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := rm.collectStats()
			rm.updateStats(stats)
			rm.persistStats(stats)
		case <-rm.stopChan:
			return
		}
	}
}

// collectStats collects current system resource statistics
func (rm *ResourceMonitor) collectStats() ResourceStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := ResourceStats{
		MemoryUsageMB:  float64(m.Sys) / 1024 / 1024,
		MemoryAllocMB:  float64(m.Alloc) / 1024 / 1024,
		GoroutineCount: runtime.NumGoroutine(),
		GCRuns:         m.NumGC,
		Timestamp:      time.Now(),
		PluginStats:    make(map[string]interface{}),
	}

	// Collect plugin-specific stats
	rm.mu.RLock()
	for pluginName, monitor := range rm.pluginMonitors {
		stats.PluginStats[pluginName] = monitor.GetResourceUsage()
	}
	rm.mu.RUnlock()

	return stats
}

// updateStats updates the internal stats cache
func (rm *ResourceMonitor) updateStats(stats ResourceStats) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.lastStats = stats
}

// persistStats persists resource statistics to storage
func (rm *ResourceMonitor) persistStats(stats ResourceStats) {
	// Log the stats as a structured log entry
	rm.logger.Debug("resource stats collected",
		core.Field{Key: "memory_usage_mb", Value: stats.MemoryUsageMB},
		core.Field{Key: "memory_alloc_mb", Value: stats.MemoryAllocMB},
		core.Field{Key: "goroutines", Value: stats.GoroutineCount},
		core.Field{Key: "gc_runs", Value: stats.GCRuns},
	)

	// TODO: Persist to database when resource_usage table is available
	// For now, we just log the information
}

// IsRunning returns whether the monitor is currently running
func (rm *ResourceMonitor) IsRunning() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.isRunning
}

// GetPluginStats returns resource stats for a specific plugin
func (rm *ResourceMonitor) GetPluginStats(pluginName string) (map[string]interface{}, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	stats, exists := rm.lastStats.PluginStats[pluginName]
	if !exists {
		return nil, false
	}
	return stats.(map[string]interface{}), true
}
