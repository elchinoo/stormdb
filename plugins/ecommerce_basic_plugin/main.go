// E-commerce Basic Workload Plugin for StormDB
// This plugin provides basic e-commerce platform workloads with standard OLTP patterns
package main

import (
	"fmt"

	"github.com/elchinoo/stormdb/pkg/plugin"
)

// ECommerceBasicPlugin implements the WorkloadPlugin interface for basic e-commerce workloads
type ECommerceBasicPlugin struct{}

// GetMetadata returns metadata about this plugin
func (p *ECommerceBasicPlugin) GetMetadata() *plugin.PluginMetadata {
	return &plugin.PluginMetadata{
		Name:        "ecommerce_basic",
		Version:     "1.0.0",
		APIVersion:  "1.0",
		Description: "Basic e-commerce platform workloads with standard OLTP patterns",
		Author:      "StormDB Team",
		WorkloadTypes: []string{
			"ecommerce_basic",
			"ecommerce_basic_read",
			"ecommerce_basic_write",
			"ecommerce_basic_mixed",
			"ecommerce_basic_oltp",
			"ecommerce_basic_analytics",
		},
		RequiredExtensions:   []string{},
		MinPostgreSQLVersion: "12.0",
		Homepage:             "https://github.com/yourusername/stormdb",
	}
}

// CreateWorkload creates a workload instance for the specified type
func (p *ECommerceBasicPlugin) CreateWorkload(workloadType string) (plugin.Workload, error) {
	switch workloadType {
	case "ecommerce_basic":
		return &ECommerceBasicWorkload{Mode: "mixed"}, nil
	case "ecommerce_basic_read":
		return &ECommerceBasicWorkload{Mode: "read"}, nil
	case "ecommerce_basic_write":
		return &ECommerceBasicWorkload{Mode: "write"}, nil
	case "ecommerce_basic_mixed":
		return &ECommerceBasicWorkload{Mode: "mixed"}, nil
	case "ecommerce_basic_oltp":
		return &ECommerceBasicWorkload{Mode: "oltp"}, nil
	case "ecommerce_basic_analytics":
		return &ECommerceBasicWorkload{Mode: "analytics"}, nil
	default:
		return nil, fmt.Errorf("unsupported ecommerce_basic workload type: %s", workloadType)
	}
}

// Initialize performs any one-time setup required by the plugin
func (p *ECommerceBasicPlugin) Initialize() error {
	return nil
}

// Cleanup performs any cleanup required when the plugin is unloaded
func (p *ECommerceBasicPlugin) Cleanup() error {
	return nil
}

// WorkloadPlugin is the exported symbol that StormDB will look for
var WorkloadPlugin ECommerceBasicPlugin
