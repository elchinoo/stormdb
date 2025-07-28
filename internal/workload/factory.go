// Package workload provides core workload management functionality
package workload

import (
	"fmt"

	"github.com/elchinoo/stormdb/pkg/plugin"
	"github.com/elchinoo/stormdb/pkg/types"
)

// Factory manages workload creation with plugin system integration
type Factory struct {
	pluginLoader  *plugin.PluginLoader
	builtinPlugin *plugin.BuiltinWorkloadPlugin
}

// NewFactory creates a new workload factory with plugin system integration
func NewFactory(cfg *types.Config) (*Factory, error) {
	// Initialize plugin loader with search paths from config
	var pluginPaths []string
	if len(cfg.Plugins.Paths) > 0 {
		pluginPaths = cfg.Plugins.Paths
	} else {
		// Default plugin paths
		pluginPaths = []string{"./plugins", "./plugins/workloads"}
	}

	pluginLoader := plugin.NewPluginLoader(pluginPaths)
	builtinPlugin := plugin.NewBuiltinWorkloadPlugin("core")

	return &Factory{
		pluginLoader:  pluginLoader,
		builtinPlugin: builtinPlugin,
	}, nil
}

// Initialize performs any one-time setup required by the factory
func (f *Factory) Initialize() error {
	return f.builtinPlugin.Initialize()
}

// Cleanup performs any cleanup required when the factory is disposed
func (f *Factory) Cleanup() error {
	return f.builtinPlugin.Cleanup()
}

// DiscoverPlugins scans for available plugins
func (f *Factory) DiscoverPlugins() (int, error) {
	return f.pluginLoader.DiscoverPlugins()
}

// Get creates a workload instance for the specified type.
// This method integrates with the plugin system to support both
// built-in workloads and dynamically loaded plugin workloads.
func (f *Factory) Get(workloadType string) (plugin.Workload, error) {
	// First try built-in workloads
	if workload, err := f.builtinPlugin.CreateWorkload(workloadType); err == nil {
		return workload, nil
	}

	// Then try plugin system
	workload, err := f.pluginLoader.GetWorkload(workloadType)
	if err != nil {
		return nil, fmt.Errorf("failed to create workload '%s': %w", workloadType, err)
	}

	return workload, nil
}

// GetAvailableWorkloads returns a list of all available workload types
func (f *Factory) GetAvailableWorkloads() []string {
	// Combine built-in and plugin workloads
	builtinTypes := f.builtinPlugin.GetSupportedWorkloads()
	pluginTypes := f.pluginLoader.GetSupportedWorkloadTypes()

	// Use a map to deduplicate
	allTypes := make(map[string]bool)
	for _, t := range builtinTypes {
		allTypes[t] = true
	}
	for _, t := range pluginTypes {
		allTypes[t] = true
	}

	// Convert back to slice
	result := make([]string, 0, len(allTypes))
	for t := range allTypes {
		result = append(result, t)
	}

	return result
}
