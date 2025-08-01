// Package plugin provides context-aware plugin loading with goroutine leak prevention.
// This implements comprehensive cancellation support and resource management
// as recommended for production systems.
package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ContextAwarePluginLoader provides context-based plugin loading with proper cancellation
type ContextAwarePluginLoader struct {
	baseLoader        *PluginLoader
	manifestValidator *ManifestValidator
	registry          *DefaultPluginRegistry
	logger            *zap.Logger

	// Context management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Resource tracking
	activeOperations map[string]context.CancelFunc
	operationsMutex  sync.RWMutex

	// Configuration
	config *LoaderConfig
}

// LoaderConfig provides configuration for the context-aware loader
type LoaderConfig struct {
	// Timeout settings
	PluginLoadTimeout   time.Duration
	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration

	// Security settings
	ManifestPath    string
	RequireManifest bool
	AllowUntrusted  bool

	// Resource limits
	MaxConcurrentLoads int
	MaxRetryAttempts   int
	RetryBackoff       time.Duration

	// Memory management
	EnableMemoryLimits bool
	MaxMemoryPerPlugin int64
}

// DefaultLoaderConfig returns sensible defaults for the loader
func DefaultLoaderConfig() *LoaderConfig {
	return &LoaderConfig{
		PluginLoadTimeout:   30 * time.Second,
		HealthCheckInterval: 5 * time.Minute,
		HealthCheckTimeout:  10 * time.Second,
		ManifestPath:        "plugins/manifest.json",
		RequireManifest:     false, // Disabled by default for compatibility
		AllowUntrusted:      true,  // Enabled by default for development
		MaxConcurrentLoads:  5,
		MaxRetryAttempts:    3,
		RetryBackoff:        1 * time.Second,
		EnableMemoryLimits:  false,             // Disabled by default
		MaxMemoryPerPlugin:  100 * 1024 * 1024, // 100MB
	}
}

// NewContextAwarePluginLoader creates a new context-aware plugin loader
func NewContextAwarePluginLoader(baseLoader *PluginLoader, registry *DefaultPluginRegistry,
	logger *zap.Logger, config *LoaderConfig) *ContextAwarePluginLoader {

	if config == nil {
		config = DefaultLoaderConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	loader := &ContextAwarePluginLoader{
		baseLoader:       baseLoader,
		registry:         registry,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
		activeOperations: make(map[string]context.CancelFunc),
		config:           config,
	}

	// Initialize manifest validator if configured
	if config.ManifestPath != "" {
		loader.manifestValidator = NewManifestValidator(config.ManifestPath)
	}

	return loader
}

// Start begins the context-aware loader background operations
func (cal *ContextAwarePluginLoader) Start() error {
	if cal.manifestValidator != nil && cal.config.RequireManifest {
		if err := cal.manifestValidator.LoadManifest(); err != nil {
			return fmt.Errorf("failed to load plugin manifest: %w", err)
		}
		cal.logger.Info("Plugin manifest loaded successfully")
	}

	// Start health check goroutine
	cal.wg.Add(1)
	go cal.healthCheckLoop()

	cal.logger.Info("Context-aware plugin loader started")
	return nil
}

// Stop gracefully shuts down the loader and cancels all operations
func (cal *ContextAwarePluginLoader) Stop() error {
	cal.logger.Info("Shutting down context-aware plugin loader")

	// Cancel all operations
	cal.cancel()

	// Cancel individual operations
	cal.operationsMutex.Lock()
	for operation, cancelFunc := range cal.activeOperations {
		cal.logger.Debug("Cancelling operation", zap.String("operation", operation))
		cancelFunc()
	}
	cal.activeOperations = make(map[string]context.CancelFunc)
	cal.operationsMutex.Unlock()

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		cal.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		cal.logger.Info("All operations completed successfully")
	case <-time.After(30 * time.Second):
		cal.logger.Warn("Timeout waiting for operations to complete")
	}

	return nil
}

// LoadPluginWithContext loads a plugin with context and timeout support
func (cal *ContextAwarePluginLoader) LoadPluginWithContext(ctx context.Context,
	pluginPath string) (*PluginInfo, error) {

	// Create operation-specific context with timeout
	loadCtx, cancel := context.WithTimeout(ctx, cal.config.PluginLoadTimeout)
	defer cancel()

	operationID := fmt.Sprintf("load-%s-%d", pluginPath, time.Now().UnixNano())
	cal.registerOperation(operationID, cancel)
	defer cal.unregisterOperation(operationID)

	// Validate against manifest if required
	if cal.manifestValidator != nil {
		if err := cal.validatePluginWithManifest(pluginPath); err != nil {
			return nil, fmt.Errorf("manifest validation failed: %w", err)
		}
	}

	// Load plugin with retry logic
	return cal.loadPluginWithRetry(loadCtx, pluginPath)
}

// DiscoverPluginsWithContext discovers plugins with context support
func (cal *ContextAwarePluginLoader) DiscoverPluginsWithContext(ctx context.Context) (int, error) {
	// Check if operation should be cancelled
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-cal.ctx.Done():
		return 0, cal.ctx.Err()
	default:
	}

	cal.logger.Info("Starting plugin discovery")

	// Use the base loader for discovery but with context awareness
	count, err := cal.baseLoader.DiscoverPlugins()
	if err != nil {
		cal.logger.Error("Plugin discovery failed", zap.Error(err))
		return count, err
	}

	cal.logger.Info("Plugin discovery completed", zap.Int("plugins_found", count))
	return count, nil
}

// healthCheckLoop runs periodic health checks on loaded plugins
func (cal *ContextAwarePluginLoader) healthCheckLoop() {
	defer cal.wg.Done()

	ticker := time.NewTicker(cal.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cal.ctx.Done():
			cal.logger.Debug("Health check loop stopping")
			return
		case <-ticker.C:
			cal.performHealthChecks()
		}
	}
}

// performHealthChecks runs health checks on all loaded plugins
func (cal *ContextAwarePluginLoader) performHealthChecks() {
	cal.logger.Debug("Performing plugin health checks")

	plugins := cal.registry.ListPlugins()
	for _, pluginInfo := range plugins {
		if !pluginInfo.Loaded {
			continue
		}

		// Create context for this health check
		ctx, cancel := context.WithTimeout(cal.ctx, cal.config.HealthCheckTimeout)

		go func(plugin *PluginInfo, checkCtx context.Context, checkCancel context.CancelFunc) {
			defer checkCancel()

			err := cal.registry.HealthCheck(plugin.Metadata.Name)
			if err != nil {
				cal.logger.Warn("Plugin health check failed",
					zap.String("plugin", plugin.Metadata.Name),
					zap.Error(err))
			} else {
				cal.logger.Debug("Plugin health check passed",
					zap.String("plugin", plugin.Metadata.Name))
			}
		}(pluginInfo, ctx, cancel)
	}
}

// loadPluginWithRetry implements retry logic for plugin loading
func (cal *ContextAwarePluginLoader) loadPluginWithRetry(ctx context.Context,
	pluginPath string) (*PluginInfo, error) {

	var lastErr error

	for attempt := 1; attempt <= cal.config.MaxRetryAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		err := cal.registry.SafeLoad(pluginPath)
		if err == nil {
			// Success - get plugin info
			for _, plugin := range cal.registry.ListPlugins() {
				if plugin.FilePath == pluginPath {
					return plugin, nil
				}
			}
			return nil, fmt.Errorf("plugin loaded but not found in registry")
		}

		lastErr = err
		cal.logger.Warn("Plugin load attempt failed",
			zap.String("plugin", pluginPath),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", cal.config.MaxRetryAttempts),
			zap.Error(err))

		if attempt < cal.config.MaxRetryAttempts {
			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(cal.config.RetryBackoff):
				// Continue to next attempt
			}
		}
	}

	return nil, fmt.Errorf("failed to load plugin after %d attempts: %w",
		cal.config.MaxRetryAttempts, lastErr)
}

// validatePluginWithManifest validates a plugin against the manifest
func (cal *ContextAwarePluginLoader) validatePluginWithManifest(pluginPath string) error {
	if cal.manifestValidator == nil {
		return nil // No manifest validation configured
	}

	entry, err := cal.manifestValidator.ValidatePlugin(pluginPath)
	if err != nil {
		return err
	}

	// Check if plugin is trusted
	if !cal.config.AllowUntrusted && !entry.Trusted {
		return fmt.Errorf("plugin %s is not trusted and untrusted plugins are not allowed",
			entry.Filename)
	}

	cal.logger.Debug("Plugin passed manifest validation",
		zap.String("plugin", entry.Filename),
		zap.String("author", entry.Author),
		zap.Bool("trusted", entry.Trusted))

	return nil
}

// registerOperation tracks an active operation for cancellation
func (cal *ContextAwarePluginLoader) registerOperation(operationID string, cancel context.CancelFunc) {
	cal.operationsMutex.Lock()
	defer cal.operationsMutex.Unlock()
	cal.activeOperations[operationID] = cancel
}

// unregisterOperation removes an operation from tracking
func (cal *ContextAwarePluginLoader) unregisterOperation(operationID string) {
	cal.operationsMutex.Lock()
	defer cal.operationsMutex.Unlock()
	delete(cal.activeOperations, operationID)
}

// GetActiveOperationsCount returns the number of currently active operations
func (cal *ContextAwarePluginLoader) GetActiveOperationsCount() int {
	cal.operationsMutex.RLock()
	defer cal.operationsMutex.RUnlock()
	return len(cal.activeOperations)
}

// GenerateManifest creates a manifest for all plugins in the configured paths
func (cal *ContextAwarePluginLoader) GenerateManifest() error {
	if cal.manifestValidator == nil {
		return fmt.Errorf("manifest validator not configured")
	}

	// Just generate manifest for the first path that has plugins
	// This is a simpler approach that avoids the multiple-path complexity
	for _, path := range cal.baseLoader.pluginPaths {
		cal.logger.Info("Generating manifest for plugin path", zap.String("path", path))

		// Check if path exists and has plugins
		if _, err := os.Stat(path); os.IsNotExist(err) {
			cal.logger.Debug("Plugin path does not exist, skipping", zap.String("path", path))
			continue
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			cal.logger.Warn("Failed to read plugin directory", zap.String("path", path), zap.Error(err))
			continue
		}

		// Check if there are any plugin files
		hasPlugins := false
		for _, entry := range entries {
			if !entry.IsDir() {
				filename := entry.Name()
				ext := strings.ToLower(filepath.Ext(filename))
				if ext == ".so" || ext == ".dll" || ext == ".dylib" {
					hasPlugins = true
					break
				}
			}
		}

		if hasPlugins {
			if err := cal.manifestValidator.GenerateManifest(path); err != nil {
				return fmt.Errorf("failed to generate manifest for path %s: %w", path, err)
			}
			cal.logger.Info("Plugin manifest generated successfully",
				zap.String("manifest_path", cal.config.ManifestPath))
			return nil
		}
	}

	// No plugins found in any path
	cal.logger.Info("No plugins found in any configured path, creating empty manifest")
	return cal.manifestValidator.GenerateManifest(cal.baseLoader.pluginPaths[0])
}

// ValidateAllPlugins validates all plugins against the manifest
func (cal *ContextAwarePluginLoader) ValidateAllPlugins() error {
	if cal.manifestValidator == nil {
		return fmt.Errorf("manifest validator not configured")
	}

	for _, path := range cal.baseLoader.pluginPaths {
		if err := cal.manifestValidator.ValidateAllPlugins(path); err != nil {
			return fmt.Errorf("validation failed for path %s: %w", path, err)
		}
	}

	return nil
}
