// RealWorld Workload Plugin for StormDB
// This plugin provides realistic enterprise application workloads
package main

import (
	"fmt"

	"github.com/elchinoo/stormdb/pkg/plugin"
)

// RealWorldPlugin implements the WorkloadPlugin interface for real-world workloads
type RealWorldPlugin struct{}

// GetMetadata returns metadata about this plugin
func (p *RealWorldPlugin) GetMetadata() *plugin.PluginMetadata {
	return &plugin.PluginMetadata{
		Name:        "realworld",
		Version:     "1.0.0",
		Description: "Realistic enterprise application workloads with complex business logic",
		Author:      "StormDB Team",
		WorkloadTypes: []string{
			"realworld",
			"realworld_read",
			"realworld_write",
			"realworld_mixed",
			"realworld_oltp",
			"realworld_analytics",
		},
		RequiredExtensions:   []string{},
		MinPostgreSQLVersion: "12.0",
		Homepage:             "https://github.com/yourusername/stormdb",
	}
}

// CreateWorkload creates a workload instance for the specified type
func (p *RealWorldPlugin) CreateWorkload(workloadType string) (plugin.Workload, error) {
	switch workloadType {
	case "realworld":
		return &RealWorldWorkload{Mode: "mixed"}, nil
	case "realworld_read":
		return &RealWorldWorkload{Mode: "read"}, nil
	case "realworld_write":
		return &RealWorldWorkload{Mode: "write"}, nil
	case "realworld_mixed":
		return &RealWorldWorkload{Mode: "mixed"}, nil
	case "realworld_oltp":
		return &RealWorldWorkload{Mode: "oltp"}, nil
	case "realworld_analytics":
		return &RealWorldWorkload{Mode: "analytics"}, nil
	default:
		return nil, fmt.Errorf("unsupported realworld workload type: %s", workloadType)
	}
}

// Initialize performs any one-time setup required by the plugin
func (p *RealWorldPlugin) Initialize() error {
	return nil
}

// Cleanup performs any cleanup required when the plugin is unloaded
func (p *RealWorldPlugin) Cleanup() error {
	return nil
}

// WorkloadPlugin is the exported symbol that StormDB will look for
var WorkloadPlugin RealWorldPlugin
