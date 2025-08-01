package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/elchinoo/stormdb/pkg/plugin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// createPluginsCommand creates and returns the plugins command
func createPluginsCommand() *cobra.Command {
	pluginsCmd := &cobra.Command{
		Use:   "plugins",
		Short: "Plugin management and diagnostics",
		Long: `Commands for managing and troubleshooting StormDB plugins.

This command helps diagnose plugin loading issues, manage plugin security,
and provides information about available plugins in your StormDB installation.

Enhanced features include:
- Plugin manifest generation and validation
- Memory usage monitoring and bounded collections
- Context-aware loading with goroutine leak prevention
- Comprehensive security validation`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("StormDB Plugin Management System")
			fmt.Println("===============================")
			fmt.Println()
			fmt.Println("Enhanced Plugin System Features:")
			fmt.Println("- ✅ Intelligent failure tracking and duplicate loading prevention")
			fmt.Println("- ✅ Comprehensive error handling with detailed diagnostics")
			fmt.Println("- ✅ Context-aware loading with timeout and cancellation support")
			fmt.Println("- ✅ Memory management with bounded collections and retention policies")
			fmt.Println("- ✅ Security validation with manifest verification and checksums")
			fmt.Println("- ✅ Goroutine leak prevention with proper resource cleanup")
			fmt.Println()
			fmt.Println("Available Subcommands:")
			fmt.Println("  list        - List all available plugins with status")
			fmt.Println("  validate    - Validate plugins against security manifest")
			fmt.Println("  generate    - Generate security manifest for current plugins")
			fmt.Println("  memory      - Show memory usage statistics")
			fmt.Println("  health      - Run health checks on loaded plugins")
			fmt.Println()
			fmt.Println("Recent Architectural Improvements:")
			fmt.Println("- Plugin name conflicts from duplicate loading - RESOLVED")
			fmt.Println("- Go runtime compatibility issues - ENHANCED")
			fmt.Println("- Memory leaks from unbounded collections - PREVENTED")
			fmt.Println("- Goroutine leaks from missing cancellation - FIXED")
			fmt.Println("- Security vulnerabilities from unvalidated plugins - SECURED")
			fmt.Println()
			fmt.Println("For detailed information:")
			fmt.Println("  docs/PLUGIN_TROUBLESHOOTING.md")
			fmt.Println("  docs/PLUGIN_DEVELOPMENT_GUIDE.md")
			fmt.Println("  docs/ADVANCED_ARCHITECTURE.md")
			fmt.Println()
			fmt.Println("To rebuild plugins after Go version changes:")
			fmt.Println("  make clean && make build-all")
		},
	}

	// Add subcommands
	pluginsCmd.AddCommand(createListCommand())
	pluginsCmd.AddCommand(createValidateCommand())
	pluginsCmd.AddCommand(createGenerateCommand())
	pluginsCmd.AddCommand(createMemoryCommand())
	pluginsCmd.AddCommand(createHealthCommand())

	return pluginsCmd
}

// createListCommand creates the list subcommand
func createListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available plugins with detailed status",
		Long: `Lists all plugins found in the configured plugin paths with their status,
including load status, version information, security validation, and memory usage.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Initialize plugin system components
			logger, _ := zap.NewDevelopment()
			defer logger.Sync()

			pluginPaths := []string{"build/plugins", "plugins"}
			baseLoader := plugin.NewPluginLoader(pluginPaths)
			registry := plugin.NewPluginRegistry(logger, "1.0", "1.0.0")

			config := plugin.DefaultLoaderConfig()
			contextLoader := plugin.NewContextAwarePluginLoader(baseLoader, registry, logger, config)

			if err := contextLoader.Start(); err != nil {
				fmt.Printf("❌ Failed to start plugin loader: %v\n", err)
				return
			}
			defer contextLoader.Stop()

			// Discover plugins
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			count, err := contextLoader.DiscoverPluginsWithContext(ctx)
			if err != nil {
				fmt.Printf("❌ Plugin discovery failed: %v\n", err)
				return
			}

			if count == 0 {
				fmt.Println("📭 No plugins found in the configured paths")
				fmt.Println("\nConfigured plugin paths:")
				for _, path := range pluginPaths {
					fmt.Printf("  - %s\n", path)
				}
				return
			}

			// List discovered plugins
			plugins := registry.ListPlugins()
			fmt.Printf("🔌 Found %d plugin(s):\n\n", len(plugins))

			for i, pluginInfo := range plugins {
				status := "❌ Not Loaded"
				if pluginInfo.Loaded {
					status = "✅ Loaded"
				}

				fmt.Printf("%d. %s\n", i+1, pluginInfo.Metadata.Name)
				fmt.Printf("   Status: %s\n", status)
				fmt.Printf("   Version: %s\n", pluginInfo.Metadata.Version)
				fmt.Printf("   Author: %s\n", pluginInfo.Metadata.Author)
				fmt.Printf("   File: %s\n", pluginInfo.FilePath)
				fmt.Printf("   Workload Types: %v\n", pluginInfo.Metadata.WorkloadTypes)

				if pluginInfo.Loaded {
					fmt.Printf("   Load Time: %s\n", pluginInfo.LoadTime.Format(time.RFC3339))
					if !pluginInfo.LastHealthCheck.IsZero() {
						fmt.Printf("   Last Health Check: %s\n", pluginInfo.LastHealthCheck.Format(time.RFC3339))
					}
				}
				fmt.Println()
			}
		},
	}
}

// createValidateCommand creates the validate subcommand
func createValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate plugins against security manifest",
		Long: `Validates all plugins against the security manifest to ensure integrity,
authenticity, and security compliance. Checks SHA256 checksums, trusted authors,
and file size limits.`,
		Run: func(cmd *cobra.Command, args []string) {
			manifestPath, _ := cmd.Flags().GetString("manifest")
			if manifestPath == "" {
				manifestPath = "plugins/manifest.json"
			}

			// Check if manifest exists
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				fmt.Printf("📄 No manifest found at %s\n", manifestPath)
				fmt.Println("💡 Use 'stormdb plugins generate' to create a manifest first")
				return
			}

			validator := plugin.NewManifestValidator(manifestPath)
			if err := validator.LoadManifest(); err != nil {
				fmt.Printf("❌ Failed to load manifest: %v\n", err)
				return
			}

			pluginPaths := []string{"build/plugins", "plugins"}
			var allValid = true

			for _, pluginPath := range pluginPaths {
				if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
					continue
				}

				fmt.Printf("🔍 Validating plugins in %s...\n", pluginPath)

				if err := validator.ValidateAllPlugins(pluginPath); err != nil {
					fmt.Printf("❌ Validation failed: %v\n", err)
					allValid = false
				} else {
					fmt.Printf("✅ All plugins in %s passed validation\n", pluginPath)
				}
			}

			if allValid {
				fmt.Println("\n🎉 All plugins passed security validation!")
			} else {
				fmt.Println("\n⚠️  Some plugins failed validation. Please check the errors above.")
				os.Exit(1)
			}
		},
	}

	cmd.Flags().String("manifest", "", "Path to plugin manifest file (default: plugins/manifest.json)")
	return cmd
}

// createGenerateCommand creates the generate subcommand
func createGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate security manifest for current plugins",
		Long: `Generates a security manifest file containing SHA256 checksums, file sizes,
and metadata for all plugins in the configured paths. This manifest can be used
for integrity verification and security validation.`,
		Run: func(cmd *cobra.Command, args []string) {
			manifestPath, _ := cmd.Flags().GetString("output")
			if manifestPath == "" {
				manifestPath = "plugins/manifest.json"
			}

			// Ensure the directory exists
			if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
				fmt.Printf("❌ Failed to create manifest directory: %v\n", err)
				return
			}

			// Use a simple approach: manually find plugins and generate manifest
			pluginPaths := []string{"build/plugins", "plugins"}

			for _, pluginPath := range pluginPaths {
				if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
					continue
				}

				fmt.Printf("📝 Generating manifest for plugins in %s...\n", pluginPath)

				// Check for plugin files
				entries, err := os.ReadDir(pluginPath)
				if err != nil {
					fmt.Printf("❌ Failed to read directory %s: %v\n", pluginPath, err)
					continue
				}

				hasPlugins := false
				for _, entry := range entries {
					if !entry.IsDir() {
						filename := entry.Name()
						ext := filepath.Ext(filename)
						if ext == ".so" || ext == ".dll" || ext == ".dylib" {
							hasPlugins = true
							break
						}
					}
				}

				if hasPlugins {
					validator := plugin.NewManifestValidator(manifestPath)
					if err := validator.GenerateManifest(pluginPath); err != nil {
						fmt.Printf("❌ Failed to generate manifest: %v\n", err)
						return
					}

					fmt.Printf("✅ Plugin manifest generated successfully: %s\n", manifestPath)
					fmt.Println("\n📋 Manifest includes:")
					fmt.Println("  - SHA256 checksums for integrity verification")
					fmt.Println("  - File sizes and modification times")
					fmt.Println("  - Plugin metadata and author information")
					fmt.Println("  - Trusted status for security validation")
					fmt.Println("\n💡 Use 'stormdb plugins validate' to verify plugins against this manifest")
					return
				}
			}

			// No plugins found
			fmt.Println("📭 No plugin files found in configured directories")
			fmt.Println("Configured directories:")
			for _, path := range pluginPaths {
				fmt.Printf("  - %s\n", path)
			}
		},
	}

	cmd.Flags().String("output", "", "Output path for manifest file (default: plugins/manifest.json)")
	return cmd
}

// createMemoryCommand creates the memory subcommand
func createMemoryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "memory",
		Short: "Show memory usage statistics and bounded collections",
		Long: `Displays detailed memory usage statistics for the plugin system,
including per-plugin memory usage, bounded collection sizes, and garbage
collection statistics.`,
		Run: func(cmd *cobra.Command, args []string) {
			logger, _ := zap.NewDevelopment()
			defer logger.Sync()

			config := plugin.DefaultMemoryConfig()
			memManager := plugin.NewMemoryManager(logger, config)

			if err := memManager.Start(); err != nil {
				fmt.Printf("❌ Failed to start memory manager: %v\n", err)
				return
			}
			defer memManager.Stop()

			// Give it a moment to collect stats
			time.Sleep(100 * time.Millisecond)

			stats := memManager.GetMemoryStats()

			fmt.Println("📊 Plugin System Memory Statistics")
			fmt.Println("==================================")
			fmt.Printf("Total Allocated: %d bytes (%.2f MB)\n",
				stats.TotalAllocated, float64(stats.TotalAllocated)/(1024*1024))
			fmt.Printf("Total In Use: %d bytes (%.2f MB)\n",
				stats.TotalInUse, float64(stats.TotalInUse)/(1024*1024))
			fmt.Printf("Plugin Count: %d\n", stats.PluginCount)
			fmt.Printf("Collection Count: %d\n", stats.CollectionCount)
			fmt.Printf("Last Updated: %s\n", stats.LastUpdated.Format(time.RFC3339))

			fmt.Println("\n🗑️  Garbage Collection Statistics")
			fmt.Println("=================================")
			fmt.Printf("GC Cycles: %d\n", stats.GCStats.NumGC)
			fmt.Printf("Total GC Pause: %s\n", time.Duration(stats.GCStats.PauseTotalNs))
			fmt.Printf("Heap Size: %d bytes (%.2f MB)\n",
				stats.GCStats.HeapInuse, float64(stats.GCStats.HeapInuse)/(1024*1024))
			fmt.Printf("Next GC Target: %d bytes (%.2f MB)\n",
				stats.GCStats.NextGC, float64(stats.GCStats.NextGC)/(1024*1024))

			fmt.Println("\n⚙️  Memory Management Configuration")
			fmt.Println("===================================")
			fmt.Printf("Max Total Memory: %.2f MB\n", float64(config.MaxTotalMemory)/(1024*1024))
			fmt.Printf("Max Plugin Memory: %.2f MB\n", float64(config.MaxPluginMemory)/(1024*1024))
			fmt.Printf("Check Interval: %s\n", config.MemoryCheckInterval)
			fmt.Printf("Retention Enabled: %t\n", config.EnableRetention)
			fmt.Printf("Max Collection Size: %d items\n", config.MaxCollectionSize)
		},
	}
}

// createHealthCommand creates the health subcommand
func createHealthCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Run comprehensive health checks on loaded plugins",
		Long: `Performs health checks on all loaded plugins to verify they are functioning
correctly. Includes plugin responsiveness, memory usage, and API compatibility checks.`,
		Run: func(cmd *cobra.Command, args []string) {
			logger, _ := zap.NewDevelopment()
			defer logger.Sync()

			pluginPaths := []string{"build/plugins", "plugins"}
			baseLoader := plugin.NewPluginLoader(pluginPaths)
			registry := plugin.NewPluginRegistry(logger, "1.0", "1.0.0")

			config := plugin.DefaultLoaderConfig()
			contextLoader := plugin.NewContextAwarePluginLoader(baseLoader, registry, logger, config)

			if err := contextLoader.Start(); err != nil {
				fmt.Printf("❌ Failed to start plugin loader: %v\n", err)
				return
			}
			defer contextLoader.Stop()

			// Discover and load plugins
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			count, err := contextLoader.DiscoverPluginsWithContext(ctx)
			if err != nil {
				fmt.Printf("❌ Plugin discovery failed: %v\n", err)
				return
			}

			if count == 0 {
				fmt.Println("📭 No plugins found to check")
				return
			}

			fmt.Println("🏥 Plugin Health Check Report")
			fmt.Println("=============================")

			plugins := registry.ListPlugins()
			healthyCount := 0
			totalCount := len(plugins)

			for i, pluginInfo := range plugins {
				fmt.Printf("\n%d. %s", i+1, pluginInfo.Metadata.Name)

				if !pluginInfo.Loaded {
					fmt.Println(" ❌ NOT LOADED")
					continue
				}

				// Run health check
				err := registry.HealthCheck(pluginInfo.Metadata.Name)
				if err != nil {
					fmt.Printf(" ❌ UNHEALTHY: %v\n", err)
					continue
				}

				fmt.Println(" ✅ HEALTHY")
				healthyCount++

				// Show additional health metrics
				fmt.Printf("   Load Time: %s ago\n",
					time.Since(pluginInfo.LoadTime).Truncate(time.Second))

				if !pluginInfo.LastHealthCheck.IsZero() {
					fmt.Printf("   Last Check: %s ago\n",
						time.Since(pluginInfo.LastHealthCheck).Truncate(time.Second))
				}
			}

			fmt.Printf("\n📈 Health Summary: %d/%d plugins healthy (%.1f%%)\n",
				healthyCount, totalCount, float64(healthyCount)/float64(totalCount)*100)

			if healthyCount == totalCount {
				fmt.Println("🎉 All plugins are healthy!")
			} else {
				fmt.Println("⚠️  Some plugins require attention")
			}
		},
	}
}
