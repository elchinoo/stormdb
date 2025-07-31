// Package plugin provides built-in workload adapters that wrap existing workloads
// to work with the plugin system. This allows existing workloads to be treated
// as plugins without requiring code changes, while providing a migration path
// to true plugin-based workloads.
//
// NOTE: Most workloads have been moved to dedicated plugins. This now only
// contains core workloads that remain built-in: TPCC, Simple, and Connection Overhead.
package plugin

import (
	"fmt"

	"github.com/elchinoo/stormdb/internal/workload/bulk_insert"
	"github.com/elchinoo/stormdb/internal/workload/simple"
	connection_overhead "github.com/elchinoo/stormdb/internal/workload/simple_connection"
	"github.com/elchinoo/stormdb/internal/workload/tpcc"
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
	// Return all built-in workload types
	return []string{"tpcc", "simple", "mixed", "read", "write", "simple_connection", "bulk_insert"}
}

// CreateWorkload creates a workload instance for the specified type
// Only handles core built-in workloads: TPCC, Simple, Connection Overhead, and Bulk Insert
func (p *BuiltinWorkloadPlugin) CreateWorkload(workloadType string) (Workload, error) {
	switch workloadType {
	case "tpcc":
		return &tpcc.TPCC{}, nil
	case "simple", "mixed", "read", "write":
		return &simple.Generator{}, nil
	case "simple_connection":
		return &connection_overhead.ConnectionWorkload{}, nil
	case "bulk_insert":
		return &bulk_insert.Generator{}, nil
	default:
		return nil, fmt.Errorf("unsupported built-in workload type: %s", workloadType)
	}
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
