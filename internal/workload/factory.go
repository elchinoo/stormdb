// Package workload provides core workload management functionality
package workload

import (
	"fmt"

	"github.com/elchinoo/stormdb/pkg/plugin"
	"github.com/elchinoo/stormdb/pkg/types"
)

// Factory manages workload creation with plugin system integration
type Factory struct {
	pluginLoader *plugin.PluginLoader
}

// NewFactory creates a new workload factory with plugin system integration
func NewFactory(cfg *types.Config) (*Factory, error) {
	// Initialize plugin loader with search paths from config
	var pluginPaths []string
	if len(cfg.Plugins.Paths) > 0 {
		pluginPaths = cfg.Plugins.Paths
	} else {
		// Default plugin paths
		pluginPaths = []string{"./plugins", "./build/plugins"}
	}

	pluginLoader := plugin.NewPluginLoader(pluginPaths)

	return &Factory{
		pluginLoader: pluginLoader,
	}, nil
}

// Initialize performs any one-time setup required by the factory
func (f *Factory) Initialize() error {
	// Plugin loader will handle initialization
	return nil
}

// Cleanup performs any cleanup required when the factory is disposed
func (f *Factory) Cleanup() error {
	// Plugin loader will handle cleanup
	return nil
}

// DiscoverPlugins scans for available plugins
func (f *Factory) DiscoverPlugins() (int, error) {
	return f.pluginLoader.DiscoverPlugins()
}

// Get creates a workload instance for the specified type.
// This method uses the plugin system to support dynamically loaded plugin workloads.
func (f *Factory) Get(workloadType string) (plugin.Workload, error) {
	// Try plugin system
	workload, err := f.pluginLoader.GetWorkload(workloadType)
	if err != nil {
		return nil, fmt.Errorf("failed to create workload '%s': %w", workloadType, err)
	}

	return workload, nil
}

// GetAvailableWorkloads returns a list of all available workload types
func (f *Factory) GetAvailableWorkloads() []string {
	// Only use plugin workloads
	return f.pluginLoader.GetSupportedWorkloadTypes()
}
