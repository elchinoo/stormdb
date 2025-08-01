// E-commerce Workload Plugin for StormDB
// This plugin provides modern e-commerce platform workloads
package main

import (
	"fmt"

	"github.com/elchinoo/stormdb/pkg/plugin"
)

// ECommercePlugin implements the WorkloadPlugin interface for e-commerce workloads
type ECommercePlugin struct{}

// GetMetadata returns metadata about this plugin
func (p *ECommercePlugin) GetMetadata() *plugin.PluginMetadata {
	return &plugin.PluginMetadata{
		Name:        "ecommerce",
		Version:     "1.0.0",
		APIVersion:  "1.0",
		Description: "Modern e-commerce platform workloads with product searches, orders, and vector-powered recommendations",
		Author:      "StormDB Team",
		WorkloadTypes: []string{
			"ecommerce",
			"ecommerce_read",
			"ecommerce_write",
			"ecommerce_mixed",
			"ecommerce_oltp",
			"ecommerce_analytics",
		},
		RequiredExtensions:   []string{"vector"},
		MinPostgreSQLVersion: "12.0",
		Homepage:             "https://github.com/yourusername/stormdb",
	}
}

// CreateWorkload creates a workload instance for the specified type
func (p *ECommercePlugin) CreateWorkload(workloadType string) (plugin.Workload, error) {
	switch workloadType {
	case "ecommerce":
		return &ECommerceWorkload{Mode: "mixed"}, nil
	case "ecommerce_read":
		return &ECommerceWorkload{Mode: "read"}, nil
	case "ecommerce_write":
		return &ECommerceWorkload{Mode: "write"}, nil
	case "ecommerce_mixed":
		return &ECommerceWorkload{Mode: "mixed"}, nil
	case "ecommerce_oltp":
		return &ECommerceWorkload{Mode: "oltp"}, nil
	case "ecommerce_analytics":
		return &ECommerceWorkload{Mode: "analytics"}, nil
	default:
		return nil, fmt.Errorf("unsupported e-commerce workload type: %s", workloadType)
	}
}

// Initialize performs any one-time setup required by the plugin
func (p *ECommercePlugin) Initialize() error {
	return nil
}

// Cleanup performs any cleanup required when the plugin is unloaded
func (p *ECommercePlugin) Cleanup() error {
	return nil
}

// WorkloadPlugin is the exported symbol that StormDB will look for
var WorkloadPlugin ECommercePlugin
