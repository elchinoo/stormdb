// Connection Overhead Workload Plugin
// This plugin provides connection overhead testing for PostgreSQL performance analysis.
//
// The connection workload is designed to measure the overhead of connection management,
// connection pooling efficiency, and identify connection-related bottlenecks.
package main

import (
	"context"
	"fmt"

	"github.com/elchinoo/stormdb/pkg/plugin"
	"github.com/elchinoo/stormdb/pkg/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectionWorkloadPlugin implements the WorkloadPlugin interface for connection overhead testing
type ConnectionWorkloadPlugin struct{}

// WorkloadPlugin is the exported symbol that the plugin loader will look for
var WorkloadPlugin ConnectionWorkloadPlugin

// GetMetadata returns metadata about this plugin
func (p *ConnectionWorkloadPlugin) GetMetadata() *plugin.PluginMetadata {
	return &plugin.PluginMetadata{
		Name:        "connection_plugin",
		Version:     "1.0.0",
		Description: "Connection overhead testing workload for measuring connection pool efficiency and connection-related bottlenecks",
		Author:      "StormDB Team",
		WorkloadTypes: []string{
			"simple_connection",
			"connection_overhead",
		},
		RequiredExtensions:   []string{}, // Connection testing doesn't require special extensions
		MinPostgreSQLVersion: "11.0",
		Homepage:             "https://github.com/elchinoo/stormdb",
	}
}

// CreateWorkload creates a Connection overhead workload instance
func (p *ConnectionWorkloadPlugin) CreateWorkload(workloadType string) (plugin.Workload, error) {
	switch workloadType {
	case "connection", "simple_connection":
		return &ConnectionWorkloadWrapper{
			workload: &ConnectionWorkload{},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported workload type: %s", workloadType)
	}
} // Initialize performs plugin initialization
func (p *ConnectionWorkloadPlugin) Initialize() error {
	// Connection workload doesn't require special initialization
	return nil
}

// Cleanup performs plugin cleanup
func (p *ConnectionWorkloadPlugin) Cleanup() error {
	// Connection workload doesn't require special cleanup
	return nil
}

// ConnectionWorkloadWrapper wraps the existing connection workload implementation
type ConnectionWorkloadWrapper struct {
	workload *ConnectionWorkload
}

// Cleanup drops tables and reloads data (called only with --rebuild)
func (w *ConnectionWorkloadWrapper) Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	return w.workload.Cleanup(ctx, db, cfg)
}

// Setup ensures schema exists (called with --setup or --rebuild)
func (w *ConnectionWorkloadWrapper) Setup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	return w.workload.Setup(ctx, db, cfg)
}

// Run executes the load test
func (w *ConnectionWorkloadWrapper) Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	return w.workload.Run(ctx, db, cfg, metrics)
}
