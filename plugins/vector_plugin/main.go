// Vector Workload Plugin for StormDB
// This plugin provides high-dimensional vector similarity search testing
package main

import (
	"fmt"
	"strings"

	"github.com/elchinoo/stormdb/pkg/plugin"
)

// VectorPlugin implements the WorkloadPlugin interface for vector workloads
type VectorPlugin struct{}

// GetMetadata returns metadata about this plugin
func (p *VectorPlugin) GetMetadata() *plugin.PluginMetadata {
	return &plugin.PluginMetadata{
		Name:        "vector",
		Version:     "2.0.0",
		APIVersion:  "1.0",
		Description: "Comprehensive pgvector testing with ingestion, updates, reads, indexes, and accuracy analysis",
		Author:      "StormDB Team",
		WorkloadTypes: []string{
			// Comprehensive pgvector workloads
			"pgvector_ingestion_single",
			"pgvector_ingestion_batch",
			"pgvector_ingestion_copy",
			"pgvector_update_single",
			"pgvector_update_batch",
			"pgvector_read_scan",
			"pgvector_read_indexed",
		},
		RequiredExtensions:   []string{"vector"},
		MinPostgreSQLVersion: "12.0",
		Homepage:             "https://github.com/yourusername/stormdb",
	}
}

// CreateWorkload creates a workload instance for the specified type
func (p *VectorPlugin) CreateWorkload(workloadType string) (plugin.Workload, error) {
	// Comprehensive pgvector workloads
	if strings.HasPrefix(workloadType, "pgvector_") {
		return &ComprehensivePgVectorWorkload{}, nil
	}

	return nil, fmt.Errorf("unsupported vector workload type: %s", workloadType)
}

// Initialize performs any one-time setup required by the plugin
func (p *VectorPlugin) Initialize() error {
	return nil
}

// Cleanup performs any cleanup required when the plugin is unloaded
func (p *VectorPlugin) Cleanup() error {
	return nil
}

// WorkloadPlugin is the exported symbol that StormDB will look for
var WorkloadPlugin VectorPlugin
