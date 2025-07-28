// Package plugin provides the plugin loading and management system for StormDB workloads.
// This package implements dynamic plugin loading, allowing external workload implementations to be loaded
// at runtime, allowing for dynamic extension of StormDB's capabilities without recompilation.
package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
)

// PluginLoader manages the loading, unloading, and discovery of workload plugins.
// It maintains a registry of available plugins and provides thread-safe access
// to plugin instances.
type PluginLoader struct {
	// plugins maps plugin names to their information and instances
	plugins map[string]*PluginInfo

	// workloadTypes maps workload type strings to the plugin that provides them
	workloadTypes map[string]string

	// mutex protects concurrent access to the plugin registry
	mutex sync.RWMutex

	// pluginPaths contains directories to search for plugin files
	pluginPaths []string
}

// NewPluginLoader creates a new plugin loader with the specified search paths.
// The loader will search for plugin files in the provided directories.
func NewPluginLoader(pluginPaths []string) *PluginLoader {
	return &PluginLoader{
		plugins:       make(map[string]*PluginInfo),
		workloadTypes: make(map[string]string),
		pluginPaths:   pluginPaths,
	}
}

// DiscoverPlugins scans the configured plugin paths for shared library files
// and attempts to load plugin metadata from each discovered file.
// Returns the number of plugins discovered and any error encountered.
func (pl *PluginLoader) DiscoverPlugins() (int, error) {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()

	discovered := 0
	var errors []string

	for _, path := range pl.pluginPaths {
		count, err := pl.discoverInPath(path)
		discovered += count
		if err != nil {
			errors = append(errors, fmt.Sprintf("path %s: %v", path, err))
		}
	}

	if len(errors) > 0 {
		return discovered, fmt.Errorf("plugin discovery errors: %s", strings.Join(errors, "; "))
	}

	return discovered, nil
}

// discoverInPath scans a single directory for plugin files
func (pl *PluginLoader) discoverInPath(path string) (int, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return 0, nil // Path doesn't exist, skip silently
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory: %w", err)
	}

	discovered := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !isPluginFile(filename) {
			continue
		}

		fullPath := filepath.Join(path, filename)
		if err := pl.loadPluginMetadata(fullPath); err != nil {
			// Log error but continue with other plugins
			fmt.Printf("Warning: Failed to load plugin %s: %v\n", fullPath, err)
			continue
		}

		discovered++
	}

	return discovered, nil
}

// isPluginFile checks if a filename appears to be a plugin shared library
func isPluginFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".so" || ext == ".dll" || ext == ".dylib"
}

// loadPluginMetadata loads plugin metadata without fully loading the plugin
func (pl *PluginLoader) loadPluginMetadata(pluginPath string) error {
	// For now, we'll load the plugin to get metadata
	// In a more sophisticated system, we might store metadata separately
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	symbol, err := p.Lookup("WorkloadPlugin")
	if err != nil {
		return fmt.Errorf("plugin does not export WorkloadPlugin symbol: %w", err)
	}

	workloadPlugin, ok := symbol.(WorkloadPlugin)
	if !ok {
		return fmt.Errorf("WorkloadPlugin symbol is not of correct type")
	}

	metadata := workloadPlugin.GetMetadata()
	if metadata == nil {
		return fmt.Errorf("plugin returned nil metadata")
	}

	// Validate metadata
	if metadata.Name == "" {
		return fmt.Errorf("plugin metadata missing required 'name' field")
	}
	if len(metadata.WorkloadTypes) == 0 {
		return fmt.Errorf("plugin metadata missing workload types")
	}

	// Check for conflicts with existing plugins
	if existing, exists := pl.plugins[metadata.Name]; exists {
		return fmt.Errorf("plugin name conflict: %s already loaded from %s", metadata.Name, existing.FilePath)
	}

	// Check for workload type conflicts
	for _, workloadType := range metadata.WorkloadTypes {
		if existingPlugin, exists := pl.workloadTypes[workloadType]; exists {
			return fmt.Errorf("workload type conflict: %s already provided by plugin %s", workloadType, existingPlugin)
		}
	}

	// Store plugin info
	pluginInfo := &PluginInfo{
		Metadata: metadata,
		FilePath: pluginPath,
		Loaded:   false,
		Plugin:   nil,
	}

	pl.plugins[metadata.Name] = pluginInfo

	// Register workload types
	for _, workloadType := range metadata.WorkloadTypes {
		pl.workloadTypes[workloadType] = metadata.Name
	}

	return nil
}

// LoadPlugin loads a specific plugin by name, making it available for use.
// The plugin must have been discovered first via DiscoverPlugins().
func (pl *PluginLoader) LoadPlugin(pluginName string) error {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()

	pluginInfo, exists := pl.plugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin not found: %s", pluginName)
	}

	if pluginInfo.Loaded {
		return nil // Already loaded
	}

	// Load the plugin
	p, err := plugin.Open(pluginInfo.FilePath)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	symbol, err := p.Lookup("WorkloadPlugin")
	if err != nil {
		return fmt.Errorf("plugin does not export WorkloadPlugin symbol: %w", err)
	}

	workloadPlugin, ok := symbol.(WorkloadPlugin)
	if !ok {
		return fmt.Errorf("WorkloadPlugin symbol is not of correct type")
	}

	// Initialize the plugin
	if err := workloadPlugin.Initialize(); err != nil {
		return fmt.Errorf("plugin initialization failed: %w", err)
	}

	pluginInfo.Plugin = workloadPlugin
	pluginInfo.Loaded = true

	return nil
}

// UnloadPlugin unloads a specific plugin by name, cleaning up its resources.
func (pl *PluginLoader) UnloadPlugin(pluginName string) error {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()

	pluginInfo, exists := pl.plugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin not found: %s", pluginName)
	}

	if !pluginInfo.Loaded {
		return nil // Already unloaded
	}

	// Clean up the plugin
	if err := pluginInfo.Plugin.Cleanup(); err != nil {
		// Log error but continue with unloading
		fmt.Printf("Warning: Plugin cleanup failed for %s: %v\n", pluginName, err)
	}

	pluginInfo.Plugin = nil
	pluginInfo.Loaded = false

	return nil
}

// GetWorkload creates a workload instance for the specified workload type.
// The workload type must be supported by one of the loaded plugins.
func (pl *PluginLoader) GetWorkload(workloadType string) (Workload, error) {
	pl.mutex.RLock()
	defer pl.mutex.RUnlock()

	pluginName, exists := pl.workloadTypes[workloadType]
	if !exists {
		return nil, fmt.Errorf("unsupported workload type: %s", workloadType)
	}

	pluginInfo, exists := pl.plugins[pluginName]
	if !exists {
		return nil, fmt.Errorf("plugin not found for workload type %s: %s", workloadType, pluginName)
	}

	if !pluginInfo.Loaded {
		// Auto-load the plugin if it's not loaded
		pl.mutex.RUnlock()
		if err := pl.LoadPlugin(pluginName); err != nil {
			pl.mutex.RLock()
			return nil, fmt.Errorf("failed to load plugin %s for workload type %s: %w", pluginName, workloadType, err)
		}
		pl.mutex.RLock()
	}

	return pluginInfo.Plugin.CreateWorkload(workloadType)
}

// ListPlugins returns information about all discovered plugins
func (pl *PluginLoader) ListPlugins() []*PluginInfo {
	pl.mutex.RLock()
	defer pl.mutex.RUnlock()

	plugins := make([]*PluginInfo, 0, len(pl.plugins))
	for _, pluginInfo := range pl.plugins {
		// Create a copy to avoid race conditions
		info := &PluginInfo{
			Metadata: pluginInfo.Metadata,
			FilePath: pluginInfo.FilePath,
			Loaded:   pluginInfo.Loaded,
			Plugin:   nil, // Don't expose the actual plugin instance
		}
		plugins = append(plugins, info)
	}

	return plugins
}

// GetSupportedWorkloadTypes returns all workload types supported by discovered plugins
func (pl *PluginLoader) GetSupportedWorkloadTypes() []string {
	pl.mutex.RLock()
	defer pl.mutex.RUnlock()

	types := make([]string, 0, len(pl.workloadTypes))
	for workloadType := range pl.workloadTypes {
		types = append(types, workloadType)
	}

	return types
}

// LoadAllPlugins loads all discovered plugins
func (pl *PluginLoader) LoadAllPlugins() error {
	pl.mutex.RLock()
	pluginNames := make([]string, 0, len(pl.plugins))
	for name := range pl.plugins {
		pluginNames = append(pluginNames, name)
	}
	pl.mutex.RUnlock()

	var errors []string
	for _, name := range pluginNames {
		if err := pl.LoadPlugin(name); err != nil {
			errors = append(errors, fmt.Sprintf("plugin %s: %v", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to load some plugins: %s", strings.Join(errors, "; "))
	}

	return nil
}

// UnloadAllPlugins unloads all loaded plugins
func (pl *PluginLoader) UnloadAllPlugins() error {
	pl.mutex.RLock()
	pluginNames := make([]string, 0, len(pl.plugins))
	for name, info := range pl.plugins {
		if info.Loaded {
			pluginNames = append(pluginNames, name)
		}
	}
	pl.mutex.RUnlock()

	var errors []string
	for _, name := range pluginNames {
		if err := pl.UnloadPlugin(name); err != nil {
			errors = append(errors, fmt.Sprintf("plugin %s: %v", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to unload some plugins: %s", strings.Join(errors, "; "))
	}

	return nil
}
