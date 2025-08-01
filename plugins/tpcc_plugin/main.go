// TPCC Workload Plugin
// This plugin provides TPC-C (Transaction Processing Performance Council - C)
// benchmark workload implementation for PostgreSQL performance testing.
//
// TPC-C is an industry-standard OLTP benchmark that simulates a wholesale
// supplier managing orders for a configurable number of warehouses.
package main

import (
	"context"
	"fmt"

	"github.com/elchinoo/stormdb/pkg/plugin"
	"github.com/elchinoo/stormdb/pkg/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TPCCWorkloadPlugin implements the WorkloadPlugin interface for TPC-C workload
type TPCCWorkloadPlugin struct{}

// WorkloadPlugin is the exported symbol that the plugin loader will look for
var WorkloadPlugin TPCCWorkloadPlugin

// GetMetadata returns metadata about this plugin
func (p *TPCCWorkloadPlugin) GetMetadata() *plugin.PluginMetadata {
	return &plugin.PluginMetadata{
		Name:        "tpcc_plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0",
		Description: "TPC-C (Transaction Processing Performance Council - C) benchmark workload for OLTP testing",
		Author:      "StormDB Team",
		WorkloadTypes: []string{
			"tpcc",
		},
		RequiredExtensions:   []string{}, // TPC-C doesn't require special extensions
		MinPostgreSQLVersion: "12.0",
		Homepage:             "https://github.com/elchinoo/stormdb",
	}
}

// CreateWorkload creates a TPCC workload instance
func (p *TPCCWorkloadPlugin) CreateWorkload(workloadType string) (plugin.Workload, error) {
	if workloadType != "tpcc" {
		return nil, fmt.Errorf("unsupported workload type: %s", workloadType)
	}
	return &TPCC{}, nil
} // Initialize performs plugin initialization
func (p *TPCCWorkloadPlugin) Initialize() error {
	// TPC-C workload doesn't require special initialization
	return nil
}

// Cleanup performs plugin cleanup
func (p *TPCCWorkloadPlugin) Cleanup() error {
	// TPC-C workload doesn't require special cleanup
	return nil
}

// TPCCWorkloadWrapper wraps the existing TPCC implementation to match the plugin interface
type TPCCWorkloadWrapper struct {
	tpcc *TPCC
}

// Cleanup drops tables and reloads data (called only with --rebuild)
func (w *TPCCWorkloadWrapper) Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	return w.tpcc.Cleanup(ctx, db, cfg)
}

// Setup ensures schema exists (called with --setup or --rebuild)
func (w *TPCCWorkloadWrapper) Setup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	return w.tpcc.Setup(ctx, db, cfg)
}

// Run executes the load test
func (w *TPCCWorkloadWrapper) Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	return w.tpcc.Run(ctx, db, cfg, metrics)
}
