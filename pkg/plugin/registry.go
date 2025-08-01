package plugin

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"plugin"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// PluginSecurity handles plugin security validation
type PluginSecurity struct {
	AllowedHashes    map[string]bool // SHA256 hashes of allowed plugin files
	TrustedAuthors   map[string]bool // Trusted plugin authors
	MaxPluginSize    int64           // Maximum allowed plugin file size
	RequireSignature bool            // Whether plugins must be signed
}

// DefaultPluginSecurity returns secure default settings
func DefaultPluginSecurity() *PluginSecurity {
	return &PluginSecurity{
		AllowedHashes:    make(map[string]bool),
		TrustedAuthors:   make(map[string]bool),
		MaxPluginSize:    50 * 1024 * 1024, // 50MB max
		RequireSignature: false,            // Disabled by default for development
	}
}

// DefaultPluginRegistry implements the PluginRegistry interface with
// comprehensive plugin management including health checks and version validation
type DefaultPluginRegistry struct {
	plugins        map[string]*PluginInfo
	dependencies   map[string][]string // plugin -> dependencies
	dependents     map[string][]string // plugin -> dependents
	security       *PluginSecurity
	mutex          sync.RWMutex
	logger         *zap.Logger
	apiVersion     string
	stormDBVersion string
	healthInterval time.Duration
	stopHealth     chan struct{}
	healthWg       sync.WaitGroup
}

// NewPluginRegistry creates a new plugin registry with specified configuration
func NewPluginRegistry(logger *zap.Logger, apiVersion, stormDBVersion string) *DefaultPluginRegistry {
	return NewPluginRegistryWithSecurity(logger, apiVersion, stormDBVersion, DefaultPluginSecurity())
}

// NewPluginRegistryWithSecurity creates a new plugin registry with security settings
func NewPluginRegistryWithSecurity(logger *zap.Logger, apiVersion, stormDBVersion string, security *PluginSecurity) *DefaultPluginRegistry {
	if logger == nil {
		logger = zap.NewNop()
	}
	if security == nil {
		security = DefaultPluginSecurity()
	}

	return &DefaultPluginRegistry{
		plugins:        make(map[string]*PluginInfo),
		dependencies:   make(map[string][]string),
		dependents:     make(map[string][]string),
		security:       security,
		logger:         logger,
		apiVersion:     apiVersion,
		stormDBVersion: stormDBVersion,
		healthInterval: 30 * time.Second,
		stopHealth:     make(chan struct{}),
	}
}

// Register adds a plugin to the registry after validation
func (r *DefaultPluginRegistry) Register(pluginInstance WorkloadPlugin) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	metadata := pluginInstance.GetMetadata()
	if err := r.validateMetadata(metadata); err != nil {
		return errors.Wrapf(err, "plugin validation failed for %s", metadata.Name)
	}

	// Check dependencies
	if err := r.checkDependencies(metadata); err != nil {
		return errors.Wrapf(err, "dependency check failed for %s", metadata.Name)
	}

	// Check for conflicts
	if existing, exists := r.plugins[metadata.Name]; exists {
		if existing.Loaded {
			return fmt.Errorf("plugin %s is already loaded", metadata.Name)
		}
	}

	// Initialize plugin
	if err := pluginInstance.Initialize(); err != nil {
		return errors.Wrapf(err, "plugin initialization failed for %s", metadata.Name)
	}

	pluginInfo := &PluginInfo{
		Metadata:        metadata,
		Loaded:          true,
		Plugin:          pluginInstance,
		LoadTime:        time.Now(),
		LastHealthCheck: time.Now(),
	}

	r.plugins[metadata.Name] = pluginInfo
	r.dependencies[metadata.Name] = metadata.Dependencies

	// Update dependents map
	for _, dep := range metadata.Dependencies {
		r.dependents[dep] = append(r.dependents[dep], metadata.Name)
	}

	r.logger.Info("Plugin registered successfully",
		zap.String("name", metadata.Name),
		zap.String("version", metadata.Version),
		zap.String("api_version", metadata.APIVersion),
		zap.Strings("dependencies", metadata.Dependencies),
	)

	return nil
}

// Validate checks if plugin metadata is compatible with current StormDB version
func (r *DefaultPluginRegistry) Validate(metadata *PluginMetadata) error {
	return r.validateMetadata(metadata)
}

// validateMetadata performs comprehensive validation of plugin metadata
func (r *DefaultPluginRegistry) validateMetadata(metadata *PluginMetadata) error {
	if metadata == nil {
		return errors.New("metadata cannot be nil")
	}

	if metadata.Name == "" {
		return errors.New("plugin name cannot be empty")
	}

	if metadata.Version == "" {
		return errors.New("plugin version cannot be empty")
	}

	// Validate semantic version format
	_, err := semver.NewVersion(metadata.Version)
	if err != nil {
		return errors.Wrapf(err, "invalid plugin version format: %s", metadata.Version)
	}

	// Validate API version compatibility
	if metadata.APIVersion != "" && metadata.APIVersion != r.apiVersion {
		return fmt.Errorf("incompatible API version: plugin requires %s, StormDB provides %s",
			metadata.APIVersion, r.apiVersion)
	}

	// Validate minimum StormDB version requirement
	if metadata.MinStormDB != "" {
		minRequired, err := semver.NewVersion(metadata.MinStormDB)
		if err != nil {
			return errors.Wrapf(err, "invalid min_stormdb version format: %s", metadata.MinStormDB)
		}

		current, err := semver.NewVersion(r.stormDBVersion)
		if err != nil {
			r.logger.Warn("Could not parse StormDB version for comparison",
				zap.String("version", r.stormDBVersion),
				zap.Error(err))
		} else if current.LessThan(minRequired) {
			return fmt.Errorf("StormDB version %s is below minimum required %s",
				r.stormDBVersion, metadata.MinStormDB)
		}
	}

	// Validate workload types
	if len(metadata.WorkloadTypes) == 0 {
		return errors.New("plugin must support at least one workload type")
	}

	return nil
}

// HealthCheck verifies if a plugin is still functional
func (r *DefaultPluginRegistry) HealthCheck(pluginName string) error {
	r.mutex.RLock()
	pluginInfo, exists := r.plugins[pluginName]
	r.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	if !pluginInfo.Loaded || pluginInfo.Plugin == nil {
		return fmt.Errorf("plugin %s is not loaded", pluginName)
	}

	// Try to get metadata as a simple health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("plugin %s panicked during health check: %v", pluginName, r)
			}
		}()

		// Simple health check - try to get metadata
		metadata := pluginInfo.Plugin.GetMetadata()
		if metadata == nil || metadata.Name != pluginName {
			done <- fmt.Errorf("plugin %s failed health check: invalid metadata", pluginName)
			return
		}
		done <- nil
	}()

	select {
	case err := <-done:
		if err == nil {
			r.mutex.Lock()
			pluginInfo.LastHealthCheck = time.Now()
			r.mutex.Unlock()
		}
		return err
	case <-ctx.Done():
		return fmt.Errorf("plugin %s health check timeout", pluginName)
	}
}

// SafeLoad loads a plugin with error recovery and isolation
func (r *DefaultPluginRegistry) SafeLoad(pluginPath string) error {
	r.logger.Info("Attempting to load plugin", zap.String("path", pluginPath))

	// Validate path
	if pluginPath == "" {
		return errors.New("plugin path cannot be empty")
	}

	absPath, err := filepath.Abs(pluginPath)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve plugin path: %s", pluginPath)
	}

	// Load plugin with panic recovery
	var loadedPlugin *plugin.Plugin
	var loadErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				loadErr = fmt.Errorf("plugin loading panicked: %v", r)
			}
		}()

		loadedPlugin, loadErr = plugin.Open(absPath)
	}()

	if loadErr != nil {
		return errors.Wrapf(loadErr, "failed to load plugin from %s", absPath)
	}

	// Look up the WorkloadPlugin symbol
	symbol, err := loadedPlugin.Lookup("WorkloadPlugin")
	if err != nil {
		return errors.Wrapf(err, "plugin %s does not export WorkloadPlugin symbol", absPath)
	}

	// Type assertion with error recovery
	var workloadPlugin WorkloadPlugin
	var assertErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				assertErr = fmt.Errorf("type assertion panicked: %v", r)
			}
		}()

		var ok bool
		workloadPlugin, ok = symbol.(WorkloadPlugin)
		if !ok {
			assertErr = fmt.Errorf("WorkloadPlugin symbol has wrong type")
		}
	}()

	if assertErr != nil {
		return errors.Wrapf(assertErr, "invalid WorkloadPlugin in %s", absPath)
	}

	// Initialize plugin with error recovery
	var initErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				initErr = fmt.Errorf("plugin initialization panicked: %v", r)
			}
		}()

		initErr = workloadPlugin.Initialize()
	}()

	if initErr != nil {
		return errors.Wrapf(initErr, "failed to initialize plugin %s", absPath)
	}

	// Register the plugin
	metadata := workloadPlugin.GetMetadata()
	if err := r.Register(workloadPlugin); err != nil {
		// Try to cleanup the plugin if registration fails
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					r.logger.Error("Plugin cleanup panicked during failed registration",
						zap.String("plugin", metadata.Name),
						zap.Any("panic", rec))
				}
			}()
			_ = workloadPlugin.Cleanup()
		}()
		return errors.Wrapf(err, "failed to register plugin %s", metadata.Name)
	}

	// Update plugin info with file path
	r.mutex.Lock()
	if pluginInfo, exists := r.plugins[metadata.Name]; exists {
		pluginInfo.FilePath = absPath
	}
	r.mutex.Unlock()

	r.logger.Info("Plugin loaded successfully",
		zap.String("name", metadata.Name),
		zap.String("version", metadata.Version),
		zap.String("path", absPath),
	)

	return nil
}

// GetPlugin returns a loaded plugin by name
func (r *DefaultPluginRegistry) GetPlugin(name string) (WorkloadPlugin, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	pluginInfo, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	if !pluginInfo.Loaded || pluginInfo.Plugin == nil {
		return nil, fmt.Errorf("plugin %s is not loaded", name)
	}

	return pluginInfo.Plugin, nil
}

// ListPlugins returns all registered plugins
func (r *DefaultPluginRegistry) ListPlugins() []*PluginInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	plugins := make([]*PluginInfo, 0, len(r.plugins))
	for _, pluginInfo := range r.plugins {
		// Create a copy to avoid race conditions
		copy := *pluginInfo
		plugins = append(plugins, &copy)
	}

	return plugins
}

// UnloadPlugin safely unloads a plugin
func (r *DefaultPluginRegistry) UnloadPlugin(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	pluginInfo, exists := r.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	if pluginInfo.Loaded && pluginInfo.Plugin != nil {
		// Try to cleanup the plugin
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					r.logger.Error("Plugin cleanup panicked",
						zap.String("plugin", name),
						zap.Any("panic", rec))
				}
			}()
			_ = pluginInfo.Plugin.Cleanup()
		}()
	}

	// Mark as unloaded
	pluginInfo.Loaded = false
	pluginInfo.Plugin = nil

	r.logger.Info("Plugin unloaded",
		zap.String("name", name),
	)

	return nil
}

// StartHealthMonitoring begins periodic health checks for all loaded plugins
func (r *DefaultPluginRegistry) StartHealthMonitoring() {
	r.healthWg.Add(1)
	go func() {
		defer r.healthWg.Done()
		ticker := time.NewTicker(r.healthInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				r.performHealthChecks()
			case <-r.stopHealth:
				return
			}
		}
	}()
}

// StopHealthMonitoring stops the health monitoring goroutine
func (r *DefaultPluginRegistry) StopHealthMonitoring() {
	close(r.stopHealth)
	r.healthWg.Wait()
}

// performHealthChecks runs health checks on all loaded plugins
func (r *DefaultPluginRegistry) performHealthChecks() {
	plugins := r.ListPlugins()

	for _, pluginInfo := range plugins {
		if !pluginInfo.Loaded {
			continue
		}

		if err := r.HealthCheck(pluginInfo.Metadata.Name); err != nil {
			r.logger.Error("Plugin health check failed",
				zap.String("plugin", pluginInfo.Metadata.Name),
				zap.Error(err),
			)

			// Mark plugin as unhealthy but don't unload automatically
			// Let the application decide what to do
		}
	}
}

// Shutdown cleanly shuts down the registry and all plugins
func (r *DefaultPluginRegistry) Shutdown() error {
	r.StopHealthMonitoring()

	r.mutex.Lock()
	defer r.mutex.Unlock()

	var errors []error
	for name := range r.plugins {
		if err := r.unloadPluginUnsafe(name); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to unload some plugins: %v", errors)
	}

	return nil
}

// unloadPluginUnsafe unloads a plugin without acquiring mutex (internal use only)
func (r *DefaultPluginRegistry) unloadPluginUnsafe(name string) error {
	pluginInfo, exists := r.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	if pluginInfo.Loaded && pluginInfo.Plugin != nil {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					r.logger.Error("Plugin cleanup panicked during shutdown",
						zap.String("plugin", name),
						zap.Any("panic", rec))
				}
			}()
			_ = pluginInfo.Plugin.Cleanup()
		}()
	}

	pluginInfo.Loaded = false
	pluginInfo.Plugin = nil

	return nil
}

// checkDependencies validates that all plugin dependencies are available
func (r *DefaultPluginRegistry) checkDependencies(metadata *PluginMetadata) error {
	for _, dep := range metadata.Dependencies {
		if _, exists := r.plugins[dep]; !exists {
			return fmt.Errorf("dependency %s not found", dep)
		}
	}
	return nil
}

// GetDependencyGraph returns the dependency relationships
func (r *DefaultPluginRegistry) GetDependencyGraph() map[string][]string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	graph := make(map[string][]string)
	for plugin, deps := range r.dependencies {
		graph[plugin] = make([]string, len(deps))
		copy(graph[plugin], deps)
	}

	return graph
}

// ValidatePluginSecurity checks plugin file security
func (r *DefaultPluginRegistry) ValidatePluginSecurity(pluginPath string) error {
	// Check file size
	fileInfo, err := os.Stat(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to stat plugin file: %w", err)
	}

	if fileInfo.Size() > r.security.MaxPluginSize {
		return fmt.Errorf("plugin file too large: %d bytes (max %d)",
			fileInfo.Size(), r.security.MaxPluginSize)
	}

	// Check hash if allowed hashes are configured
	if len(r.security.AllowedHashes) > 0 {
		hash, err := r.calculateFileHash(pluginPath)
		if err != nil {
			return fmt.Errorf("failed to calculate file hash: %w", err)
		}

		if !r.security.AllowedHashes[hash] {
			return fmt.Errorf("plugin hash %s not in allowed list", hash)
		}
	}

	return nil
}

// calculateFileHash computes SHA256 hash of a file
func (r *DefaultPluginRegistry) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// AddTrustedAuthor adds an author to the trusted list
func (r *DefaultPluginRegistry) AddTrustedAuthor(author string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.security.TrustedAuthors[author] = true
}

// RemoveTrustedAuthor removes an author from the trusted list
func (r *DefaultPluginRegistry) RemoveTrustedAuthor(author string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.security.TrustedAuthors, author)
}

// AddAllowedHash adds a file hash to the allowed list
func (r *DefaultPluginRegistry) AddAllowedHash(hash string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.security.AllowedHashes[hash] = true
}

// RemoveAllowedHash removes a file hash from the allowed list
func (r *DefaultPluginRegistry) RemoveAllowedHash(hash string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.security.AllowedHashes, hash)
}

// GetSecurityConfig returns a copy of the current security configuration
func (r *DefaultPluginRegistry) GetSecurityConfig() PluginSecurity {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	config := PluginSecurity{
		AllowedHashes:    make(map[string]bool),
		TrustedAuthors:   make(map[string]bool),
		MaxPluginSize:    r.security.MaxPluginSize,
		RequireSignature: r.security.RequireSignature,
	}

	for hash := range r.security.AllowedHashes {
		config.AllowedHashes[hash] = true
	}

	for author := range r.security.TrustedAuthors {
		config.TrustedAuthors[author] = true
	}

	return config
}
