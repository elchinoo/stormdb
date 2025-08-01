// Package plugin provides built-in workload adapters that wrap existing workloads
// to work with the plugin system. This allows existing workloads to be treated
// as plugins without requiring code changes, while providing a migration path
// to true plugin-based workloads.
//
// NOTE: Most workloads have been moved to dedicated plugins. This file is kept
// for compatibility but no longer defines built-in workloads.
package plugin

import (
	"fmt"
)

// BuiltinWorkloadPlugin wraps existing built-in workloads to work with the plugin system.
// This allows the existing workloads to be used alongside dynamically loaded plugins
// without requiring code changes.
type BuiltinWorkloadPlugin struct {
	// Name of the workload this plugin handles
	Name string
}

// NewBuiltinWorkloadPlugin creates a new built-in workload plugin adapter
func NewBuiltinWorkloadPlugin(name string) *BuiltinWorkloadPlugin {
	return &BuiltinWorkloadPlugin{
		Name: name,
	}
}

// GetName returns the name of this plugin
func (p *BuiltinWorkloadPlugin) GetName() string {
	return p.Name
}

// GetVersion returns the version of this plugin
func (p *BuiltinWorkloadPlugin) GetVersion() string {
	return "1.0.0"
}

// GetDescription returns a description of this plugin
func (p *BuiltinWorkloadPlugin) GetDescription() string {
	return fmt.Sprintf("Built-in adapter for %s workload", p.Name)
}

// GetSupportedWorkloads returns the list of workload types this plugin supports
func (p *BuiltinWorkloadPlugin) GetSupportedWorkloads() []string {
	// All workloads are now provided by dedicated plugins
	return []string{}
}

// CreateWorkload creates a workload instance for the specified type
// All workloads have been moved to dedicated plugins
func (p *BuiltinWorkloadPlugin) CreateWorkload(workloadType string) (Workload, error) {
	return nil, fmt.Errorf("unsupported built-in workload type: %s (all workloads are now plugin-based)", workloadType)
}

// Initialize performs any one-time setup required by the plugin
func (p *BuiltinWorkloadPlugin) Initialize() error {
	// Built-in workloads don't require special initialization
	return nil
}

// Cleanup performs any cleanup required when the plugin is unloaded
func (p *BuiltinWorkloadPlugin) Cleanup() error {
	// Built-in workloads don't require special cleanup
	return nil
}
