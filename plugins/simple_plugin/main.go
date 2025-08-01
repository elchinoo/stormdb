// Simple Workload Plugin
// This plugin provides basic CRUD operations for PostgreSQL performance testing.
// Supports multiple operation modes: simple, read, write, and mixed workloads.
package main

import (
	"context"
	"fmt"

	"github.com/elchinoo/stormdb/pkg/plugin"
	"github.com/elchinoo/stormdb/pkg/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SimpleWorkloadPlugin implements the WorkloadPlugin interface for simple workloads
type SimpleWorkloadPlugin struct{}

// WorkloadPlugin is the exported symbol that the plugin loader will look for
var WorkloadPlugin SimpleWorkloadPlugin

// GetMetadata returns metadata about this plugin
func (p *SimpleWorkloadPlugin) GetMetadata() *plugin.PluginMetadata {
	return &plugin.PluginMetadata{
		Name:        "simple_plugin",
		Version:     "1.0.0",
		Description: "Basic CRUD operation workloads with configurable read/write ratios",
		Author:      "StormDB Team",
		WorkloadTypes: []string{
			"simple",
			"read",
			"write",
			"mixed",
		},
		RequiredExtensions:   []string{}, // Simple workload doesn't require special extensions
		MinPostgreSQLVersion: "11.0",
		Homepage:             "https://github.com/elchinoo/stormdb",
	}
}

// CreateWorkload creates a Simple workload instance
func (p *SimpleWorkloadPlugin) CreateWorkload(workloadType string) (plugin.Workload, error) {
	switch workloadType {
	case "simple", "read", "write", "mixed":
		return &Generator{}, nil
	default:
		return nil, fmt.Errorf("unsupported workload type: %s", workloadType)
	}
}

// Initialize performs plugin initialization
func (p *SimpleWorkloadPlugin) Initialize() error {
	// Simple workload doesn't require special initialization
	return nil
}

// Cleanup performs plugin cleanup
func (p *SimpleWorkloadPlugin) Cleanup() error {
	// Simple workload doesn't require special cleanup
	return nil
}

// SimpleWorkloadWrapper wraps the existing Simple implementation to match the plugin interface
type SimpleWorkloadWrapper struct {
	generator *Generator
}

// Cleanup drops tables and reloads data (called only with --rebuild)
func (w *SimpleWorkloadWrapper) Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	return w.generator.Cleanup(ctx, db, cfg)
}

// Setup ensures schema exists (called with --setup or --rebuild)
func (w *SimpleWorkloadWrapper) Setup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	return w.generator.Setup(ctx, db, cfg)
}

// Run executes the load test
func (w *SimpleWorkloadWrapper) Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	return w.generator.Run(ctx, db, cfg, metrics)
}
