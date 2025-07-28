// IMDB Workload Plugin for StormDB
// This plugin provides various IMDB movie database workloads
package main

import (
	"fmt"

	"github.com/elchinoo/stormdb/pkg/plugin"
)

// IMDBPlugin implements the WorkloadPlugin interface for IMDB workloads
type IMDBPlugin struct{}

// GetMetadata returns metadata about this plugin
func (p *IMDBPlugin) GetMetadata() *plugin.PluginMetadata {
	return &plugin.PluginMetadata{
		Name:        "imdb",
		Version:     "1.0.0",
		Description: "IMDB movie database workloads with complex queries and realistic data patterns",
		Author:      "StormDB Team",
		WorkloadTypes: []string{
			"imdb",
			"imdb_read",
			"imdb_write",
			"imdb_mixed",
		},
		RequiredExtensions:   []string{},
		MinPostgreSQLVersion: "12.0",
		Homepage:             "https://github.com/yourusername/stormdb",
	}
}

// CreateWorkload creates a workload instance for the specified type
func (p *IMDBPlugin) CreateWorkload(workloadType string) (plugin.Workload, error) {
	switch workloadType {
	case "imdb":
		return &IMDBWorkload{Mode: "mixed"}, nil
	case "imdb_read":
		return &IMDBWorkload{Mode: "read"}, nil
	case "imdb_write":
		return &IMDBWorkload{Mode: "write"}, nil
	case "imdb_mixed":
		return &IMDBWorkload{Mode: "mixed"}, nil
	default:
		return nil, fmt.Errorf("unsupported IMDB workload type: %s", workloadType)
	}
}

// Initialize performs any one-time setup required by the plugin
func (p *IMDBPlugin) Initialize() error {
	return nil
}

// Cleanup performs any cleanup required when the plugin is unloaded
func (p *IMDBPlugin) Cleanup() error {
	return nil
}

// WorkloadPlugin is the exported symbol that StormDB will look for
var WorkloadPlugin IMDBPlugin
