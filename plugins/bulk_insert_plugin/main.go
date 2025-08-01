// Bulk Insert Workload Plugin
// This plugin provides bulk insert performance testing for PostgreSQL with progressive scaling.
//
// The bulk insert workload is designed to test high-throughput insert performance,
// identify bottlenecks in bulk data loading, and optimize batch size configurations.
package main

import (
	"context"
	"fmt"

	"github.com/elchinoo/stormdb/pkg/plugin"
	"github.com/elchinoo/stormdb/pkg/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BulkInsertWorkloadPlugin implements the WorkloadPlugin interface for bulk insert testing
type BulkInsertWorkloadPlugin struct{}

// WorkloadPlugin is the exported symbol that the plugin loader will look for
var WorkloadPlugin BulkInsertWorkloadPlugin

// GetMetadata returns metadata about this plugin
func (p *BulkInsertWorkloadPlugin) GetMetadata() *plugin.PluginMetadata {
	return &plugin.PluginMetadata{
		Name:        "bulk_insert_plugin",
		Version:     "1.0.0",
		APIVersion:  "1.0",
		Description: "High-throughput bulk insert performance testing with progressive scaling and bottleneck identification",
		Author:      "StormDB Team",
		WorkloadTypes: []string{
			"bulk_insert",
		},
		RequiredExtensions:   []string{}, // Bulk insert doesn't require special extensions
		MinPostgreSQLVersion: "11.0",
		Homepage:             "https://github.com/elchinoo/stormdb",
	}
}

// CreateWorkload creates a Bulk Insert workload instance
func (p *BulkInsertWorkloadPlugin) CreateWorkload(workloadType string) (plugin.Workload, error) {
	if workloadType != "bulk_insert" {
		return nil, fmt.Errorf("unsupported workload type: %s", workloadType)
	}

	// Create a new generator instance to avoid nil pointer issues
	generator := &Generator{}
	if generator == nil {
		return nil, fmt.Errorf("failed to create generator instance")
	}

	return &BulkInsertWorkloadWrapper{
		generator: generator,
	}, nil
} // Initialize performs plugin initialization
func (p *BulkInsertWorkloadPlugin) Initialize() error {
	// Bulk insert workload doesn't require special initialization
	return nil
}

// Cleanup performs plugin cleanup
func (p *BulkInsertWorkloadPlugin) Cleanup() error {
	// Bulk insert workload doesn't require special cleanup
	return nil
}

// BulkInsertWorkloadWrapper wraps the existing bulk insert implementation
type BulkInsertWorkloadWrapper struct {
	generator *Generator
}

// Cleanup drops tables and reloads data (called only with --rebuild)
func (w *BulkInsertWorkloadWrapper) Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	if w.generator == nil {
		return fmt.Errorf("generator is nil")
	}
	return w.generator.Cleanup(ctx, db, cfg)
}

// Setup ensures schema exists (called with --setup or --rebuild)
func (w *BulkInsertWorkloadWrapper) Setup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	if w.generator == nil {
		return fmt.Errorf("generator is nil")
	}
	return w.generator.Setup(ctx, db, cfg)
}

// Run executes the load test
func (w *BulkInsertWorkloadWrapper) Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	if w.generator == nil {
		return fmt.Errorf("generator is nil")
	}
	if db == nil {
		return fmt.Errorf("database pool is nil")
	}
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if metrics == nil {
		return fmt.Errorf("metrics is nil")
	}
	return w.generator.Run(ctx, db, cfg, metrics)
}

// main function is required for Go plugins
func main() {
	// This function is required for Go plugins to build properly
	// It's not actually called when used as a plugin
}
