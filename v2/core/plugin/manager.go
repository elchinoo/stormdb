// Package plugin provides dynamic plugin loading for StormDB v2
// Handles plugin discovery, loading, validation, and lifecycle management
package plugin

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"

	"github.com/elchinoo/stormdb/v2/core"
)

// Manager implements the PluginManager interface
type Manager struct {
	plugins     map[string]core.Plugin
	pluginPaths map[string]string
	config      *core.CoreConfig
	storage     core.StorageManager
	logger      core.Logger
	mu          sync.RWMutex
}

// NewManager creates a new plugin manager
func NewManager(config *core.CoreConfig, storage core.StorageManager, logger core.Logger) *Manager {
	return &Manager{
		plugins:     make(map[string]core.Plugin),
		pluginPaths: make(map[string]string),
		config:      config,
		storage:     storage,
		logger:      logger.WithFields(core.Field{Key: "component", Value: "plugin"}),
	}
}

// LoadPlugin loads a single plugin from a file path
func (m *Manager) LoadPlugin(path string) (core.Plugin, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("loading plugin", core.Field{Key: "path", Value: path})

	// Validate plugin file
	if err := m.ValidatePluginFile(path); err != nil {
		return nil, fmt.Errorf("plugin validation failed: %w", err)
	}

	// Load the plugin shared library
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look for the NewPlugin symbol
	newPluginSymbol, err := p.Lookup("NewPlugin")
	if err != nil {
		return nil, fmt.Errorf("plugin does not export NewPlugin function: %w", err)
	}

	// Cast to the expected function type
	newPlugin, ok := newPluginSymbol.(func() core.Plugin)
	if !ok {
		return nil, fmt.Errorf("NewPlugin function has incorrect signature")
	}

	// Create plugin instance
	pluginInstance := newPlugin()
	if pluginInstance == nil {
		return nil, fmt.Errorf("NewPlugin returned nil")
	}

	// Validate the plugin
	if err := m.ValidatePlugin(pluginInstance); err != nil {
		return nil, fmt.Errorf("plugin validation failed: %w", err)
	}

	// Get plugin metadata
	metadata := pluginInstance.Metadata()

	// Calculate and verify SHA256
	sha256Hash, err := m.calculateSHA256(path)
	if err != nil {
		m.logger.Warn("failed to calculate SHA256",
			core.Field{Key: "path", Value: path},
			core.Field{Key: "error", Value: err.Error()},
		)
	} else {
		metadata.SHA256 = sha256Hash
	}

	// Register in storage
	if m.storage != nil {
		_, err := m.storage.RegisterPlugin(context.Background(), metadata)
		if err != nil {
			m.logger.Warn("failed to register plugin in storage",
				core.Field{Key: "plugin", Value: metadata.Name},
				core.Field{Key: "error", Value: err.Error()},
			)
		}
	}

	// Store in memory
	pluginKey := fmt.Sprintf("%s@%s", metadata.Name, metadata.Version)
	m.plugins[pluginKey] = pluginInstance
	m.pluginPaths[pluginKey] = path

	m.logger.Info("plugin loaded successfully",
		core.Field{Key: "name", Value: metadata.Name},
		core.Field{Key: "version", Value: metadata.Version},
		core.Field{Key: "path", Value: path},
	)

	return pluginInstance, nil
}

// LoadPlugins discovers and loads all plugins from configured directories
func (m *Manager) LoadPlugins() error {
	m.logger.Info("starting plugin discovery",
		core.Field{Key: "directories", Value: m.config.PluginDirs},
	)

	var loadedCount int
	var errors []string

	for _, dir := range m.config.PluginDirs {
		count, errs := m.loadPluginsFromDirectory(dir)
		loadedCount += count
		errors = append(errors, errs...)
	}

	if len(errors) > 0 {
		m.logger.Warn("some plugins failed to load",
			core.Field{Key: "error_count", Value: len(errors)},
			core.Field{Key: "errors", Value: strings.Join(errors, "; ")},
		)
	}

	m.logger.Info("plugin discovery completed",
		core.Field{Key: "loaded_count", Value: loadedCount},
		core.Field{Key: "error_count", Value: len(errors)},
	)

	return nil
}

// loadPluginsFromDirectory loads all plugins from a specific directory
func (m *Manager) loadPluginsFromDirectory(dir string) (int, []string) {
	var loadedCount int
	var errors []string

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		m.logger.Debug("plugin directory does not exist", core.Field{Key: "directory", Value: dir})
		return 0, nil
	}

	// Find all .so files in the directory
	pattern := filepath.Join(dir, "*.so")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		errors = append(errors, fmt.Sprintf("failed to glob %s: %v", pattern, err))
		return 0, errors
	}

	// Load each plugin file
	for _, match := range matches {
		if _, err := m.LoadPlugin(match); err != nil {
			errors = append(errors, fmt.Sprintf("failed to load %s: %v", match, err))
		} else {
			loadedCount++
		}
	}

	return loadedCount, errors
}

// UnloadPlugins unloads all loaded plugins
func (m *Manager) UnloadPlugins() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("unloading all plugins", core.Field{Key: "count", Value: len(m.plugins)})

	var errors []string
	for name, plugin := range m.plugins {
		if err := plugin.Cleanup(context.Background()); err != nil {
			errors = append(errors, fmt.Sprintf("failed to cleanup %s: %v", name, err))
		}
	}

	// Clear the maps
	m.plugins = make(map[string]core.Plugin)
	m.pluginPaths = make(map[string]string)

	if len(errors) > 0 {
		return fmt.Errorf("errors during plugin unload: %s", strings.Join(errors, "; "))
	}

	m.logger.Info("all plugins unloaded")
	return nil
}

// GetPlugin retrieves a loaded plugin by name (latest version)
func (m *Manager) GetPlugin(name string) (core.Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Look for exact match first
	if plugin, exists := m.plugins[name]; exists {
		return plugin, nil
	}

	// Look for name@version pattern
	for key, plugin := range m.plugins {
		if strings.HasPrefix(key, name+"@") {
			return plugin, nil
		}
	}

	return nil, fmt.Errorf("plugin %s not found", name)
}

// GetLoadedPlugins returns all loaded plugins
func (m *Manager) GetLoadedPlugins() []core.Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]core.Plugin, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// ListPlugins returns metadata for all loaded plugins
func (m *Manager) ListPlugins() []core.PluginMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metadata := make([]core.PluginMetadata, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		metadata = append(metadata, plugin.Metadata())
	}

	return metadata
}

// ValidatePlugin validates a plugin instance
func (m *Manager) ValidatePlugin(plugin core.Plugin) error {
	if plugin == nil {
		return fmt.Errorf("plugin is nil")
	}

	metadata := plugin.Metadata()

	// Validate required metadata fields
	if metadata.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if metadata.Version == "" {
		return fmt.Errorf("plugin version is required")
	}

	// Validate version format (basic semantic versioning check)
	if !isValidVersion(metadata.Version) {
		return fmt.Errorf("invalid version format: %s", metadata.Version)
	}

	// Validate that plugin implements required methods
	testConfig := make(map[string]interface{})

	// Test validation method
	if err := plugin.Validate(testConfig); err != nil {
		// This is expected for empty config, but method should exist
		m.logger.Debug("plugin validation test completed",
			core.Field{Key: "plugin", Value: metadata.Name},
		)
	}

	return nil
}

// ValidatePluginFile validates a plugin file before loading
func (m *Manager) ValidatePluginFile(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("plugin file does not exist: %s", path)
	}

	// Check file extension
	if !strings.HasSuffix(strings.ToLower(path), ".so") {
		return fmt.Errorf("plugin file must have .so extension: %s", path)
	}

	// Check file permissions
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	if fileInfo.Mode()&0111 == 0 {
		return fmt.Errorf("plugin file is not executable: %s", path)
	}

	return nil
}

// RegisterPlugin registers a plugin instance (for manually created plugins)
func (m *Manager) RegisterPlugin(plugin core.Plugin) error {
	if err := m.ValidatePlugin(plugin); err != nil {
		return fmt.Errorf("plugin validation failed: %w", err)
	}

	metadata := plugin.Metadata()
	pluginKey := fmt.Sprintf("%s@%s", metadata.Name, metadata.Version)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already registered
	if _, exists := m.plugins[pluginKey]; exists {
		return fmt.Errorf("plugin %s already registered", pluginKey)
	}

	m.plugins[pluginKey] = plugin

	m.logger.Info("plugin registered",
		core.Field{Key: "name", Value: metadata.Name},
		core.Field{Key: "version", Value: metadata.Version},
	)

	return nil
}

// UnregisterPlugin removes a plugin from the registry
func (m *Manager) UnregisterPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find and remove the plugin
	var found bool
	for key := range m.plugins {
		if strings.HasPrefix(key, name+"@") {
			delete(m.plugins, key)
			delete(m.pluginPaths, key)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("plugin %s not found", name)
	}

	m.logger.Info("plugin unregistered", core.Field{Key: "name", Value: name})
	return nil
}

// InitializePlugin initializes a plugin with core services
func (m *Manager) InitializePlugin(plugin core.Plugin, coreServices *core.CoreServices) error {
	metadata := plugin.Metadata()

	m.logger.Info("initializing plugin",
		core.Field{Key: "name", Value: metadata.Name},
		core.Field{Key: "version", Value: metadata.Version},
	)

	ctx := context.Background()
	if err := plugin.Initialize(ctx, coreServices); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", metadata.Name, err)
	}

	m.logger.Info("plugin initialized successfully",
		core.Field{Key: "name", Value: metadata.Name},
	)

	return nil
}

// GetPluginByTestType returns the first plugin that supports a test type
func (m *Manager) GetPluginByTestType(testType string) (core.Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, plugin := range m.plugins {
		metadata := plugin.Metadata()
		for _, supportedType := range metadata.TestTypes {
			if supportedType == testType {
				return plugin, nil
			}
		}
	}

	return nil, fmt.Errorf("no plugin found for test type: %s", testType)
}

// calculateSHA256 calculates the SHA256 hash of a file
func (m *Manager) calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// isValidVersion performs basic semantic version validation
func isValidVersion(version string) bool {
	// Basic check for X.Y.Z format
	parts := strings.Split(version, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return false
	}

	for _, part := range parts {
		if part == "" {
			return false
		}
		// Could add more rigorous validation here
	}

	return true
}

// GetPluginStatus returns runtime status information for plugins
func (m *Manager) GetPluginStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := map[string]interface{}{
		"loaded_count": len(m.plugins),
		"plugins":      make([]map[string]interface{}, 0, len(m.plugins)),
	}

	for key, plugin := range m.plugins {
		metadata := plugin.Metadata()
		pluginStatus := map[string]interface{}{
			"key":         key,
			"name":        metadata.Name,
			"version":     metadata.Version,
			"description": metadata.Description,
			"test_types":  metadata.TestTypes,
			"path":        m.pluginPaths[key],
		}
		status["plugins"] = append(status["plugins"].([]map[string]interface{}), pluginStatus)
	}

	return status
}
