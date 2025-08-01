// Package plugin provides the core plugin system for StormDB workloads.
// It defines the common interface that all workload plugins must implement
// and provides the plugin discovery and loading functionality.
//
// This plugin system allows StormDB to load workloads dynamically at runtime
// without requiring recompilation. Workload plugins are compiled as shared
// libraries (.so files on Linux/macOS, .dll on Windows) and can be loaded
// from paths specified in configuration files or command line arguments.
//
// Plugin Interface:
// All workload plugins must implement the WorkloadPlugin interface which
// provides methods for plugin metadata, lifecycle management, and workload
// execution. The interface ensures compatibility and consistent behavior
// across all plugin implementations.
//
// Example Plugin Implementation:
//
//	type MyWorkloadPlugin struct{}
//
//	func (p *MyWorkloadPlugin) GetMetadata() *PluginMetadata {
//	    return &PluginMetadata{
//	        Name:           "my_workload",
//	        Version:        "1.0.0",
//	        APIVersion:     "1.0",
//	        Description:    "Custom workload for specific testing",
//	        Author:         "Your Name",
//	        WorkloadTypes:  []string{"my_workload", "my_workload_read"},
//	        Dependencies:   []string{},
//	        MinStormDB:     "0.2.0",
//	    }
//	}
//
//	func (p *MyWorkloadPlugin) CreateWorkload(workloadType string) (Workload, error) {
//	    switch workloadType {
//	    case "my_workload":
//	        return &MyWorkload{}, nil
//	    default:
//	        return nil, fmt.Errorf("unsupported workload type: %s", workloadType)
//	    }
//	}
//
// Plugin Loading:
// Plugins are loaded using Go's plugin package and must export a symbol
// named "WorkloadPlugin" that implements the WorkloadPlugin interface.
package plugin

import (
	"context"
	"time"

	"github.com/elchinoo/stormdb/pkg/types"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PluginMetadata contains information about a workload plugin including
// its name, version, supported workload types, and other descriptive information.
type PluginMetadata struct {
	// Name is the unique identifier for this plugin
	Name string `json:"name"`

	// Version is the semantic version of the plugin (e.g., "1.0.0")
	Version string `json:"version"`

	// APIVersion specifies the plugin API version this plugin was built for
	APIVersion string `json:"api_version"`

	// Description provides a human-readable description of the plugin's purpose
	Description string `json:"description"`

	// Author identifies the plugin creator or maintainer
	Author string `json:"author"`

	// WorkloadTypes lists all workload type strings this plugin supports
	WorkloadTypes []string `json:"workload_types"`

	// Dependencies lists other plugins this plugin depends on
	Dependencies []string `json:"dependencies,omitempty"`

	// MinStormDB specifies the minimum StormDB version required
	MinStormDB string `json:"min_stormdb,omitempty"`

	// RequiredExtensions lists PostgreSQL extensions required by this plugin
	RequiredExtensions []string `json:"required_extensions,omitempty"`

	// MinPostgreSQLVersion specifies the minimum PostgreSQL version required
	MinPostgreSQLVersion string `json:"min_postgresql_version,omitempty"`

	// Homepage provides a URL for the plugin's documentation or source code
	Homepage string `json:"homepage,omitempty"`
}

// Workload defines the contract for any database workload implementation.
// This interface must be implemented by all workload types, whether built-in
// or loaded from plugins.
type Workload interface {
	// Cleanup drops tables and reloads data (called only with --rebuild)
	Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error

	// Setup ensures schema exists (called with --setup or --rebuild)
	Setup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error

	// Run executes the load test
	Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error
}

// WorkloadPlugin is the main interface that all workload plugins must implement.
// It provides plugin metadata and factory methods for creating workload instances.
type WorkloadPlugin interface {
	// GetMetadata returns information about this plugin including name, version,
	// supported workload types, and other descriptive information
	GetMetadata() *PluginMetadata

	// CreateWorkload creates a new workload instance for the specified type.
	// The workloadType parameter must be one of the types listed in the plugin's metadata.
	CreateWorkload(workloadType string) (Workload, error)

	// Initialize is called once when the plugin is loaded, allowing for
	// any one-time setup or validation required by the plugin
	Initialize() error

	// Cleanup is called when the plugin is being unloaded, allowing for
	// any necessary cleanup of resources
	Cleanup() error
}

// PluginInfo contains runtime information about a loaded plugin
type PluginInfo struct {
	// Metadata contains the plugin's static information
	Metadata *PluginMetadata

	// FilePath is the absolute path to the plugin's shared library file
	FilePath string

	// Loaded indicates whether the plugin is currently loaded and available
	Loaded bool

	// Plugin is the actual plugin instance (nil if not loaded)
	Plugin WorkloadPlugin

	// LoadTime is when the plugin was successfully loaded
	LoadTime time.Time

	// LastHealthCheck is the last time the plugin passed a health check
	LastHealthCheck time.Time
}

// PluginRegistry manages plugin lifecycle with health checks and graceful degradation
type PluginRegistry interface {
	// Register adds a plugin to the registry
	Register(plugin WorkloadPlugin) error

	// Validate checks if plugin metadata is compatible with current StormDB version
	Validate(metadata *PluginMetadata) error

	// HealthCheck verifies if a plugin is still functional
	HealthCheck(pluginName string) error

	// SafeLoad loads a plugin with error recovery and isolation
	SafeLoad(pluginPath string) error

	// GetPlugin returns a loaded plugin by name
	GetPlugin(name string) (WorkloadPlugin, error)

	// ListPlugins returns all registered plugins
	ListPlugins() []*PluginInfo

	// UnloadPlugin safely unloads a plugin
	UnloadPlugin(name string) error
}
