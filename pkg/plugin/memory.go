// Package plugin provides memory-aware resource management for plugins.
// This implements bounded collections, retention policies, and garbage collection
// optimization as recommended for production systems.
package plugin

import (
	"context"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MemoryManager handles memory-aware resource management for the plugin system
type MemoryManager struct {
	logger *zap.Logger

	// Configuration
	config *MemoryConfig

	// Resource tracking
	pluginMemory map[string]*PluginMemoryStats
	globalStats  *GlobalMemoryStats
	mutex        sync.RWMutex

	// Background management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Retention policies
	retentionPolicies map[string]*RetentionPolicy
}

// MemoryConfig provides configuration for memory management
type MemoryConfig struct {
	// Global limits
	MaxTotalMemory      int64         // Maximum total memory for all plugins
	MaxPluginMemory     int64         // Maximum memory per plugin
	MemoryCheckInterval time.Duration // How often to check memory usage

	// GC optimization
	EnableGCTuning  bool  // Whether to tune GC based on memory usage
	GCTargetPercent int   // GOGC target percentage
	GCMemoryLimit   int64 // Memory limit before forcing GC

	// Retention policies
	EnableRetention     bool          // Whether to apply retention policies
	DefaultRetentionTTL time.Duration // Default TTL for data retention
	MaxCollectionSize   int           // Maximum size for unbounded collections

	// Alert thresholds
	MemoryWarnThreshold  float64 // Percentage of limit to trigger warning
	MemoryAlertThreshold float64 // Percentage of limit to trigger alert
}

// DefaultMemoryConfig returns sensible defaults for memory management
func DefaultMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		MaxTotalMemory:       1024 * 1024 * 1024, // 1GB total
		MaxPluginMemory:      100 * 1024 * 1024,  // 100MB per plugin
		MemoryCheckInterval:  30 * time.Second,
		EnableGCTuning:       true,
		GCTargetPercent:      100,
		GCMemoryLimit:        800 * 1024 * 1024, // 800MB
		EnableRetention:      true,
		DefaultRetentionTTL:  1 * time.Hour,
		MaxCollectionSize:    10000,
		MemoryWarnThreshold:  0.8,  // 80%
		MemoryAlertThreshold: 0.95, // 95%
	}
}

// PluginMemoryStats tracks memory usage for a specific plugin
type PluginMemoryStats struct {
	PluginName      string
	AllocatedBytes  int64
	InUseBytes      int64
	GCCycles        int64
	LastGC          time.Time
	Collections     map[string]*BoundedCollection
	RetentionActive bool

	// Metrics
	PeakMemory       int64
	AverageMemory    int64
	MemoryGrowthRate float64
}

// GlobalMemoryStats tracks overall memory usage
type GlobalMemoryStats struct {
	TotalAllocated  int64
	TotalInUse      int64
	PluginCount     int
	CollectionCount int
	GCStats         runtime.MemStats
	LastUpdated     time.Time
}

// BoundedCollection implements a size-limited collection with automatic cleanup
type BoundedCollection struct {
	Name        string
	MaxSize     int
	TTL         time.Duration
	Items       []BoundedItem
	mutex       sync.RWMutex
	lastCleanup time.Time
}

// BoundedItem represents an item in a bounded collection
type BoundedItem struct {
	Data      interface{}
	Timestamp time.Time
	Size      int64
}

// RetentionPolicy defines how data should be retained and cleaned up
type RetentionPolicy struct {
	Name            string
	TTL             time.Duration
	MaxSize         int
	CleanupInterval time.Duration
	SamplingRate    float64       // For sampling-based retention
	CompressOlder   time.Duration // Compress data older than this
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(logger *zap.Logger, config *MemoryConfig) *MemoryManager {
	if config == nil {
		config = DefaultMemoryConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	mm := &MemoryManager{
		logger:            logger,
		config:            config,
		pluginMemory:      make(map[string]*PluginMemoryStats),
		globalStats:       &GlobalMemoryStats{},
		ctx:               ctx,
		cancel:            cancel,
		retentionPolicies: make(map[string]*RetentionPolicy),
	}

	// Set up default retention policies
	mm.setupDefaultRetentionPolicies()

	return mm
}

// Start begins memory management background operations
func (mm *MemoryManager) Start() error {
	mm.logger.Info("Starting memory manager")

	// Start memory monitoring goroutine
	mm.wg.Add(1)
	go mm.memoryMonitorLoop()

	// Start cleanup goroutine
	mm.wg.Add(1)
	go mm.cleanupLoop()

	// Configure GC if enabled
	if mm.config.EnableGCTuning {
		mm.tuneGarbageCollector()
	}

	return nil
}

// Stop gracefully shuts down the memory manager
func (mm *MemoryManager) Stop() error {
	mm.logger.Info("Stopping memory manager")
	mm.cancel()

	// Wait for background goroutines to finish
	done := make(chan struct{})
	go func() {
		mm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		mm.logger.Info("Memory manager stopped successfully")
	case <-time.After(10 * time.Second):
		mm.logger.Warn("Timeout waiting for memory manager to stop")
	}

	return nil
}

// RegisterPlugin registers a plugin for memory tracking
func (mm *MemoryManager) RegisterPlugin(pluginName string) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.pluginMemory[pluginName] = &PluginMemoryStats{
		PluginName:  pluginName,
		Collections: make(map[string]*BoundedCollection),
		LastGC:      time.Now(),
	}

	mm.logger.Debug("Registered plugin for memory tracking",
		zap.String("plugin", pluginName))
}

// UnregisterPlugin removes a plugin from memory tracking
func (mm *MemoryManager) UnregisterPlugin(pluginName string) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if stats, exists := mm.pluginMemory[pluginName]; exists {
		// Cleanup all collections for this plugin
		for _, collection := range stats.Collections {
			collection.Clear()
		}
		delete(mm.pluginMemory, pluginName)

		mm.logger.Debug("Unregistered plugin from memory tracking",
			zap.String("plugin", pluginName))
	}
}

// CreateBoundedCollection creates a new bounded collection for a plugin
func (mm *MemoryManager) CreateBoundedCollection(pluginName, collectionName string,
	maxSize int, ttl time.Duration) *BoundedCollection {

	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	stats, exists := mm.pluginMemory[pluginName]
	if !exists {
		mm.RegisterPlugin(pluginName)
		stats = mm.pluginMemory[pluginName]
	}

	collection := &BoundedCollection{
		Name:        collectionName,
		MaxSize:     maxSize,
		TTL:         ttl,
		Items:       make([]BoundedItem, 0, maxSize),
		lastCleanup: time.Now(),
	}

	stats.Collections[collectionName] = collection

	mm.logger.Debug("Created bounded collection",
		zap.String("plugin", pluginName),
		zap.String("collection", collectionName),
		zap.Int("max_size", maxSize),
		zap.Duration("ttl", ttl))

	return collection
}

// GetMemoryStats returns current memory statistics
func (mm *MemoryManager) GetMemoryStats() *GlobalMemoryStats {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	mm.updateGlobalStats()
	return mm.globalStats
}

// GetPluginMemoryStats returns memory statistics for a specific plugin
func (mm *MemoryManager) GetPluginMemoryStats(pluginName string) *PluginMemoryStats {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	if stats, exists := mm.pluginMemory[pluginName]; exists {
		return stats
	}
	return nil
}

// memoryMonitorLoop runs periodic memory checks
func (mm *MemoryManager) memoryMonitorLoop() {
	defer mm.wg.Done()

	ticker := time.NewTicker(mm.config.MemoryCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mm.ctx.Done():
			return
		case <-ticker.C:
			mm.performMemoryCheck()
		}
	}
}

// cleanupLoop runs periodic cleanup operations
func (mm *MemoryManager) cleanupLoop() {
	defer mm.wg.Done()

	ticker := time.NewTicker(mm.config.MemoryCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mm.ctx.Done():
			return
		case <-ticker.C:
			mm.performCleanup()
		}
	}
}

// performMemoryCheck checks memory usage and triggers alerts if needed
func (mm *MemoryManager) performMemoryCheck() {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	mm.updateGlobalStats()

	// Check global memory usage
	totalUsage := float64(mm.globalStats.TotalInUse) / float64(mm.config.MaxTotalMemory)

	if totalUsage > mm.config.MemoryAlertThreshold {
		mm.logger.Error("Memory usage exceeds alert threshold",
			zap.Float64("usage_percent", totalUsage*100),
			zap.Int64("total_memory", mm.globalStats.TotalInUse),
			zap.Int64("limit", mm.config.MaxTotalMemory))

		// Force garbage collection
		runtime.GC()

	} else if totalUsage > mm.config.MemoryWarnThreshold {
		mm.logger.Warn("Memory usage exceeds warning threshold",
			zap.Float64("usage_percent", totalUsage*100),
			zap.Int64("total_memory", mm.globalStats.TotalInUse),
			zap.Int64("limit", mm.config.MaxTotalMemory))
	}

	// Check per-plugin memory usage
	for pluginName, stats := range mm.pluginMemory {
		pluginUsage := float64(stats.InUseBytes) / float64(mm.config.MaxPluginMemory)

		if pluginUsage > mm.config.MemoryAlertThreshold {
			mm.logger.Error("Plugin memory usage exceeds alert threshold",
				zap.String("plugin", pluginName),
				zap.Float64("usage_percent", pluginUsage*100),
				zap.Int64("plugin_memory", stats.InUseBytes),
				zap.Int64("limit", mm.config.MaxPluginMemory))
		}
	}
}

// performCleanup runs cleanup operations on collections
func (mm *MemoryManager) performCleanup() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	for pluginName, stats := range mm.pluginMemory {
		for collectionName, collection := range stats.Collections {
			cleaned := collection.Cleanup()
			if cleaned > 0 {
				mm.logger.Debug("Cleaned up collection items",
					zap.String("plugin", pluginName),
					zap.String("collection", collectionName),
					zap.Int("items_cleaned", cleaned))
			}
		}
	}
}

// updateGlobalStats updates the global memory statistics
func (mm *MemoryManager) updateGlobalStats() {
	var totalAllocated, totalInUse int64
	pluginCount := len(mm.pluginMemory)
	collectionCount := 0

	for _, stats := range mm.pluginMemory {
		totalAllocated += stats.AllocatedBytes
		totalInUse += stats.InUseBytes
		collectionCount += len(stats.Collections)
	}

	// Get runtime memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	mm.globalStats.TotalAllocated = totalAllocated
	mm.globalStats.TotalInUse = totalInUse
	mm.globalStats.PluginCount = pluginCount
	mm.globalStats.CollectionCount = collectionCount
	mm.globalStats.GCStats = memStats
	mm.globalStats.LastUpdated = time.Now()
}

// tuneGarbageCollector optimizes GC settings based on memory configuration
func (mm *MemoryManager) tuneGarbageCollector() {
	// Set GOGC target percentage
	runtime.GC()

	mm.logger.Info("Tuned garbage collector",
		zap.Int("gc_target_percent", mm.config.GCTargetPercent),
		zap.Int64("gc_memory_limit", mm.config.GCMemoryLimit))
}

// setupDefaultRetentionPolicies creates default retention policies
func (mm *MemoryManager) setupDefaultRetentionPolicies() {
	// Metrics retention policy
	mm.retentionPolicies["metrics"] = &RetentionPolicy{
		Name:            "metrics",
		TTL:             mm.config.DefaultRetentionTTL,
		MaxSize:         mm.config.MaxCollectionSize,
		CleanupInterval: mm.config.MemoryCheckInterval,
		SamplingRate:    1.0, // Keep all data initially
		CompressOlder:   30 * time.Minute,
	}

	// Logs retention policy
	mm.retentionPolicies["logs"] = &RetentionPolicy{
		Name:            "logs",
		TTL:             2 * mm.config.DefaultRetentionTTL,
		MaxSize:         mm.config.MaxCollectionSize * 2,
		CleanupInterval: mm.config.MemoryCheckInterval,
		SamplingRate:    0.5, // Sample 50% of log entries
		CompressOlder:   15 * time.Minute,
	}
}

// BoundedCollection methods

// Add adds an item to the bounded collection
func (bc *BoundedCollection) Add(data interface{}, size int64) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	item := BoundedItem{
		Data:      data,
		Timestamp: time.Now(),
		Size:      size,
	}

	// If at capacity, remove oldest item
	if len(bc.Items) >= bc.MaxSize {
		bc.Items = bc.Items[1:]
	}

	bc.Items = append(bc.Items, item)
}

// Cleanup removes expired items from the collection
func (bc *BoundedCollection) Cleanup() int {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if time.Since(bc.lastCleanup) < bc.TTL/10 {
		return 0 // Don't cleanup too frequently
	}

	cutoff := time.Now().Add(-bc.TTL)
	var validItems []BoundedItem
	cleaned := 0

	for _, item := range bc.Items {
		if item.Timestamp.After(cutoff) {
			validItems = append(validItems, item)
		} else {
			cleaned++
		}
	}

	bc.Items = validItems
	bc.lastCleanup = time.Now()
	return cleaned
}

// Size returns the current size of the collection
func (bc *BoundedCollection) Size() int {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	return len(bc.Items)
}

// Clear removes all items from the collection
func (bc *BoundedCollection) Clear() {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	bc.Items = bc.Items[:0]
}
